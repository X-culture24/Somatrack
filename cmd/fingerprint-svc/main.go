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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/config"
	"github.com/stmaryskabete/lms/internal/db"
	"github.com/stmaryskabete/lms/internal/fingerprint"
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

	crypto, err := fingerprint.NewCrypto(cfg.FingerprintEncKey, cfg.FingerprintEncSalt)
	if err != nil {
		log.Fatalf("init fingerprint crypto: %v", err)
	}
	devices := fingerprint.NewDeviceStore(pool)
	templates := fingerprint.NewTemplateStore(pool, crypto)
	loc := config.NairobiLocation()
	scans := fingerprint.NewScannerStore(pool, loc)

	var notifCli *rpc.NotificationClient
	if notifURL := os.Getenv("NOTIFICATION_SERVICE_URL"); notifURL != "" {
		key := os.Getenv("NOTIFICATION_SERVICE_KEY")
		if key == "" {
			key = cfg.DjangoSecretKey + ":notification"
		}
		notifCli = rpc.NewNotificationClient(notifURL, key)
	}

	svc := fingerprint.NewService(devices, templates, scans, loc, pool, notifCli)

	go runDailyRollup(pool, loc, notifCli)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{
			"service": "fingerprint-svc",
			"status":  "ok",
		})
	})

	serviceKey := cfg.FingerprintEncKey + cfg.DjangoSecretKey
	_ = serviceKey

	r.Route("/api/fingerprint", func(r chi.Router) {
		svc.RegisterPublicRoutes(r)
	})

	r.Route("/internal", func(r chi.Router) {
		intKey := getEnv("FINGERPRINT_SERVICE_KEY", cfg.DjangoSecretKey+":fingerprint")
		r.Use(httpx.ServiceKeyMiddleware(intKey))
		svc.RegisterInternalRoutes(r)
	})

	port := getEnv("FINGERPRINT_PORT", "8004")
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		fmt.Fprintf(os.Stderr, "[fingerprint-svc] listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Fprintf(os.Stderr, "[fingerprint-svc] shutting down\n")
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx2)
}

func getEnv(k, fb string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fb
}

func runDailyRollup(pool *pgxpool.Pool, loc *time.Location, notifCli *rpc.NotificationClient) {
	for {
		now := time.Now().In(loc)
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 9, 15, 0, 0, loc)
		if !now.Before(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}
		wait := nextRun.Sub(now)
		log.Printf("[rollup] next daily attendance rollup at %s (in %s)", nextRun.Format(time.RFC3339), wait)
		time.Sleep(wait)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		today := time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), 0, 0, 0, 0, loc)

		rows, err := pool.Query(ctx, `
			SELECT s.id, s.first_name, s.last_name, s.current_class_id
			FROM students s
			WHERE s.status = 'active' AND s.is_active = TRUE
			  AND s.id NOT IN (
				SELECT fa.student_id FROM fingerprint_attendance fa
				WHERE fa.date = $1 AND fa.scan_type = 'entry'
			  )`, today)
		if err != nil {
			log.Printf("[rollup] query absent students: %v", err)
			cancel()
			continue
		}
		type absent struct {
			ID         uuid.UUID
			First, Last string
			ClassID    *uuid.UUID
		}
		absentList := make([]absent, 0, 64)
		for rows.Next() {
			var a absent
			if err := rows.Scan(&a.ID, &a.First, &a.Last, &a.ClassID); err == nil {
				absentList = append(absentList, a)
			}
		}
		rows.Close()

		for _, a := range absentList {
			_, err := pool.Exec(ctx, `
				INSERT INTO attendance_records (student_id, class_id, date, status, remarks)
				VALUES ($1, $2, $3, 'absent', 'daily rollup — no entry scan')
				ON CONFLICT DO NOTHING`, a.ID, a.ClassID, today)
			if err != nil {
				log.Printf("[rollup] insert absent row student=%s: %v", a.ID, err)
				continue
			}
			if notifCli != nil {
				grows, err := pool.Query(ctx, `
					SELECT sg.guardian_id
					FROM student_guardians sg
					WHERE sg.student_id = $1
					ORDER BY sg.is_primary DESC, sg.contact_priority ASC
					LIMIT 1`, a.ID)
				if err == nil {
					var gid uuid.UUID
					if grows.Next() {
						if grows.Scan(&gid) == nil {
							req := &rpc.GuardianNotification{
								GuardianID:  gid,
								StudentID:   a.ID,
								TemplateKey: "attendance_alert",
								TemplateData: map[string]string{
									"student_first": a.First,
									"student_last":  a.Last,
									"status":        "absent today",
									"time":          "09:15 AM",
								},
								SMS:   true,
								Email: false,
							}
							_ = notifCli.DispatchGuardianNotification(ctx, req)
						}
					}
					grows.Close()
				}
			}
		}
		log.Printf("[rollup] marked %d students absent for %s", len(absentList), today.Format("2006-01-02"))
		cancel()
	}
}
