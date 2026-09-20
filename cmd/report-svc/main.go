package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/stmaryskabete/lms/internal/config"
	"github.com/stmaryskabete/lms/internal/db"
	"github.com/stmaryskabete/lms/internal/reports"
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

	loc := config.NairobiLocation()
	svc := reports.NewService(pool, loc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(300 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{
			"service": "report-svc",
			"status":  "ok",
		})
	})

	r.Route("/internal", func(r chi.Router) {
		intKey := getEnvStr("REPORT_SERVICE_KEY", cfg.DjangoSecretKey+":report")
		r.Use(httpx.ServiceKeyMiddleware(intKey))
		svc.RegisterInternalRoutes(r)
	})

	port := getEnvStr("REPORT_PORT", "8005")
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		fmt.Fprintf(os.Stderr, "[report-svc] listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Fprintf(os.Stderr, "[report-svc] shutting down\n")
	ctx2, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx2)
}

func getEnvStr(k, fb string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fb
}
