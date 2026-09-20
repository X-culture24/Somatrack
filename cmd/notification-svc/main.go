package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/config"
	"github.com/stmaryskabete/lms/internal/db"
	"github.com/stmaryskabete/lms/internal/notifications"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
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

	smsBackend := getEnvStr("SMS_BACKEND", cfg.EmailBackend)
	smsSender := &notifications.SMSSender{
		Backend:   smsBackend,
		APIKey:    os.Getenv("SMS_API_KEY"),
		APISender: getEnvStr("SMS_SENDER", "STMARYS"),
		APIURL:    os.Getenv("SMS_API_URL"),
	}
	smtpPort, _ := strconv.Atoi(getEnvStr("SMTP_PORT", "587"))
	emailSender := &notifications.EmailSender{
		Backend:      cfg.EmailBackend,
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     smtpPort,
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    cfg.DefaultFromEmail,
		FromName:     getEnvStr("EMAIL_FROM_NAME", cfg.SchoolName),
	}

	useAsync := cfg.UseRedis
	var queue *notifications.Queue
	var closeFn func()
	if useAsync {
		queue, closeFn, err = notifications.NewQueue(pool, cfg.RedisURL)
		if err != nil {
			log.Printf("[notification-svc] asynq disabled: %v", err)
			useAsync = false
		} else {
			defer closeFn()
			go runWorker(cfg.RedisURL, smsSender, emailSender, pool)
		}
	}

	svc := notifications.NewService(queue, smsSender, emailSender, useAsync, pool, cfg.DefaultFromEmail)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{
			"service": "notification-svc",
			"status":  "ok",
		})
	})

	r.Route("/internal", func(r chi.Router) {
		intKey := getEnvStr("NOTIFICATION_SERVICE_KEY", cfg.DjangoSecretKey+":notification")
		r.Use(httpx.ServiceKeyMiddleware(intKey))
		svc.RegisterInternalRoutes(r)
	})

	port := getEnvStr("NOTIFICATION_PORT", "8003")
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		fmt.Fprintf(os.Stderr, "[notification-svc] listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Fprintf(os.Stderr, "[notification-svc] shutting down\n")
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx2)
}

func runWorker(redisURL string, sms *notifications.SMSSender, email *notifications.EmailSender, pool *pgxpool.Pool) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		log.Printf("[notification-svc] worker parse redis: %v", err)
		return
	}
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"notifications": 6,
			"default":       3,
		},
	})
	h := notifications.WorkerHandler(sms, email, pool)
	if err := srv.Run(h); err != nil {
		log.Printf("[notification-svc] worker exited: %v", err)
	}
}

func getEnvStr(k, fb string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fb
}
