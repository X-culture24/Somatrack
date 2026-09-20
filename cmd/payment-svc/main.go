package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"

	"github.com/stmaryskabete/lms/internal/config"
	"github.com/stmaryskabete/lms/internal/db"
	"github.com/stmaryskabete/lms/internal/payments"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	pool, err := db.Init(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer db.Close()

	loc := config.NairobiLocation()
	matcher := payments.NewMatcher(pool, cfg.MpesaAccountPrefix, loc)
	reconciler := payments.NewReconciler(pool, loc)

	useAsync := cfg.UseRedis
	var processor *payments.AsynqProcessor
	if useAsync {
		p, closeFn, err := payments.NewAsynqProcessor(reconciler, matcher, cfg.RedisURL)
		if err != nil {
			log.Printf("[payment-svc] asynq disabled: %v", err)
			useAsync = false
		} else {
			processor = p
			defer closeFn()
			go runWorker(cfg.RedisURL, reconciler, matcher)
		}
	}

	var notifCli *rpc.NotificationClient
	if notifURL := os.Getenv("NOTIFICATION_SERVICE_URL"); notifURL != "" {
		key := os.Getenv("NOTIFICATION_SERVICE_KEY")
		if key == "" {
			key = cfg.DjangoSecretKey + ":notification"
		}
		notifCli = rpc.NewNotificationClient(notifURL, key)
	}

	svc := payments.NewService(matcher, reconciler, processor, cfg.MpesaSharedSecret, cfg.MpesaAllowedIPs, notifCli, loc, useAsync)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(120 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{
			"service": "payment-svc",
			"status":  "ok",
		})
	})

	r.Route("/api", func(r chi.Router) {
		svc.RegisterPublicRoutes(r)
	})

	r.Route("/internal", func(r chi.Router) {
		intKey := getEnv("PAYMENT_SERVICE_KEY", cfg.DjangoSecretKey+":payment")
		r.Use(httpx.ServiceKeyMiddleware(intKey))
		svc.RegisterInternalRoutes(r)
	})

	port := getEnv("PAYMENT_PORT", "8002")
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		fmt.Fprintf(os.Stderr, "[payment-svc] listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Fprintf(os.Stderr, "[payment-svc] shutting down\n")
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx2)
}

func runWorker(redisURL string, rec *payments.Reconciler, m *payments.Matcher) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		log.Printf("[payment-svc] worker parse redis: %v", err)
		return
	}
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: 5,
		Queues: map[string]int{
			"payments": 6,
			"default":  3,
		},
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc(payments.TaskProcessMpesa, func(ctx context.Context, t *asynq.Task) error {
		var p payments.ProcessPaymentParams
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		_, _, _, _, err := rec.ProcessPayment(ctx, p, m)
		return err
	})
	if err := srv.Run(mux); err != nil {
		log.Printf("[payment-svc] worker exited: %v", err)
	}
}

func getEnv(k, fb string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fb
}
