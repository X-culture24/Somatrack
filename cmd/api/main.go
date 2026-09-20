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

	jwtutil "github.com/stmaryskabete/lms/internal/shared/jwt"

	"github.com/stmaryskabete/lms/internal/api"
	lmsmiddleware "github.com/stmaryskabete/lms/internal/middleware"
	"github.com/stmaryskabete/lms/internal/config"
	"github.com/stmaryskabete/lms/internal/db"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
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

	jwtMgr := jwtutil.NewManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, cfg.JWTRotateRefresh)
	loc := config.NairobiLocation()

	svcKey := cfg.DjangoSecretKey

	fpURL := getEnv("FINGERPRINT_SERVICE_URL", "http://fingerprint-svc:8004")
	fpKey := getEnv("FINGERPRINT_SERVICE_KEY", svcKey+":fingerprint")
	fpClient := rpc.NewFingerprintClient(fpURL, fpKey)

	paymentURL := getEnv("PAYMENT_SERVICE_URL", "http://payment-svc:8002")
	paymentKey := getEnv("PAYMENT_SERVICE_KEY", svcKey+":payment")
	paymentClient := rpc.NewPaymentClient(paymentURL, paymentKey)

	notifURL := getEnv("NOTIFICATION_SERVICE_URL", "http://notification-svc:8003")
	notifKey := getEnv("NOTIFICATION_SERVICE_KEY", svcKey+":notification")
	notifClient := rpc.NewNotificationClient(notifURL, notifKey)

	reportURL := getEnv("REPORT_SERVICE_URL", "http://report-svc:8005")
	reportKey := getEnv("REPORT_SERVICE_KEY", svcKey+":report")
	reportClient := rpc.NewReportClient(reportURL, reportKey)

	financeURL := getEnv("FINANCE_SERVICE_URL", "http://finance-svc:8006")
	financeKey := getEnv("FINANCE_SERVICE_KEY", svcKey+":finance")
	financeClient := rpc.NewFinanceClient(financeURL, financeKey)

	authSvc := api.NewAuthService(pool, jwtMgr, cfg.NovaURL, loc, notifClient)
	dashboard := api.NewDashboardService(pool, paymentClient, reportClient)
	fpProxy := api.NewFingerprintProxy(fpClient, pool)
	financeProxy := api.NewFinanceProxy(financeClient, pool)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(120 * time.Second))
	r.Use(lmsmiddleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(lmsmiddleware.Security)

	r.Get("/api/health/", api.HealthHandler(pool))

	r.Route("/api", func(r chi.Router) {
		authSvc.RegisterRoutes(r)

		r.Route("/", func(r chi.Router) {
			portalAuth := lmsmiddleware.Auth(jwtMgr, jwtutil.ScopePortal)
			r.Use(portalAuth)

			r.Get("/dashboard/", dashboard.Handle)

			fpProxy.RegisterRoutes(r)
			financeProxy.RegisterRoutes(r)

			r.Route("/users/", func(r chi.Router) {
				r.Use(adminOnly())
				registerPlaceholderCRUD(r, "users")
			})
			r.Route("/students/", func(r chi.Router) { registerPlaceholderCRUD(r, "students") })
			r.Route("/guardians/", func(r chi.Router) { registerPlaceholderCRUD(r, "guardians") })
			r.Route("/admissions/", func(r chi.Router) { registerPlaceholderCRUD(r, "admissions") })
			r.Route("/student-documents/", func(r chi.Router) { registerPlaceholderCRUD(r, "student-documents") })
			r.Route("/medical-records/", func(r chi.Router) { registerPlaceholderCRUD(r, "medical-records") })
			r.Route("/student-transfers/", func(r chi.Router) { registerPlaceholderCRUD(r, "student-transfers") })
			r.Route("/student-withdrawals/", func(r chi.Router) { registerPlaceholderCRUD(r, "student-withdrawals") })

			r.Route("/academic-years/", func(r chi.Router) { registerPlaceholderCRUD(r, "academic-years") })
			r.Route("/terms/", func(r chi.Router) { registerPlaceholderCRUD(r, "terms") })
			r.Route("/streams/", func(r chi.Router) { registerPlaceholderCRUD(r, "streams") })
			r.Route("/subjects/", func(r chi.Router) { registerPlaceholderCRUD(r, "subjects") })
			r.Route("/classes/", func(r chi.Router) { registerPlaceholderCRUD(r, "classes") })
			r.Route("/grade-requirements/", func(r chi.Router) { registerPlaceholderCRUD(r, "grade-requirements") })
			r.Get("/school/profile/", placeholderHandler("school profile"))
			r.Get("/school/curriculum/", placeholderHandler("school curriculum"))

			r.Route("/departments/", func(r chi.Router) { registerPlaceholderCRUD(r, "departments") })
			r.Route("/staff/", func(r chi.Router) { registerPlaceholderCRUD(r, "staff") })
			// payroll-runs/, payroll-entries/, invoices/, payments/, fee-structures/, discounts/
			// are real endpoints now — see financeProxy.RegisterRoutes(r) above.
			r.Route("/staff-deductions/", func(r chi.Router) { registerPlaceholderCRUD(r, "staff-deductions") })

			r.Route("/uniforms/", func(r chi.Router) { registerPlaceholderCRUD(r, "uniforms") })
			r.Route("/unreconciled-payments/", func(r chi.Router) {
				r.Use(financeOnly())
				registerPlaceholderCRUD(r, "unreconciled-payments")
			})
			r.Route("/finance/", func(r chi.Router) {
				r.Use(financeOnly())
				r.Post("/manual-payment/", placeholderHandler("manual payment"))
				r.Get("/reconciliation/", placeholderHandler("reconciliation"))
				r.Post("/send-reminders/", func(w http.ResponseWriter, r *http.Request) {
					sent, err := paymentClient.SendReminders(r.Context())
					if err != nil {
						httpx.Error(w, http.StatusBadGateway, err)
						return
					}
					httpx.JSON(w, http.StatusOK, map[string]int{"sent": sent})
				})
			})

			r.Route("/attendance/", func(r chi.Router) { registerPlaceholderCRUD(r, "attendance") })
			r.Route("/assessments/", func(r chi.Router) { registerPlaceholderCRUD(r, "assessments") })
			r.Route("/marks/", func(r chi.Router) { registerPlaceholderCRUD(r, "marks") })
			r.Route("/timetable/", func(r chi.Router) { registerPlaceholderCRUD(r, "timetable") })
			r.Route("/lesson-plans/", func(r chi.Router) { registerPlaceholderCRUD(r, "lesson-plans") })

			r.Route("/announcements/", func(r chi.Router) { registerPlaceholderCRUD(r, "announcements") })
			r.Route("/messages/", func(r chi.Router) { registerPlaceholderCRUD(r, "messages") })

			r.Route("/transport-routes/", func(r chi.Router) { registerPlaceholderCRUD(r, "transport-routes") })
			r.Route("/vehicles/", func(r chi.Router) { registerPlaceholderCRUD(r, "vehicles") })
			r.Route("/transport-attendance/", func(r chi.Router) { registerPlaceholderCRUD(r, "transport-attendance") })

			r.Route("/library/books/", func(r chi.Router) { registerPlaceholderCRUD(r, "library/books") })
			r.Route("/library/loans/", func(r chi.Router) { registerPlaceholderCRUD(r, "library/loans") })
			r.Route("/library/reservations/", func(r chi.Router) { registerPlaceholderCRUD(r, "library/reservations") })

			r.Route("/inventory/", func(r chi.Router) { registerPlaceholderCRUD(r, "inventory") })

			r.Route("/reports/", func(r chi.Router) {
				r.Get("/attendance/", func(w http.ResponseWriter, r *http.Request) {
					_ = reportClient
					placeholderHandler("reports/attendance")(w, r)
				})
				r.Get("/grades/", placeholderHandler("reports/grades"))
				r.Route("/fee-collection/", func(r chi.Router) {
					r.Use(financeOnly())
					r.Get("/", placeholderHandler("reports/fee-collection"))
				})
				r.Get("/class-performance/", placeholderHandler("reports/class-performance"))
			})
		})

		r.Route("/nova/", func(r chi.Router) {
			r.Use(lmsmiddleware.Auth(jwtMgr, jwtutil.ScopeNova))
			registerPlaceholderCRUD(r, "nova/courses")
			registerPlaceholderCRUD(r, "nova/materials")
			registerPlaceholderCRUD(r, "nova/assignments")
			registerPlaceholderCRUD(r, "nova/submissions")
			registerPlaceholderCRUD(r, "nova/quizzes")
			registerPlaceholderCRUD(r, "nova/questions")
			registerPlaceholderCRUD(r, "nova/attempts")
			registerPlaceholderCRUD(r, "nova/discussions")
		})
	})

	port := getEnv("API_PORT", cfg.ServerPort)
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		fmt.Fprintf(os.Stderr, "[api-gateway] listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Fprintf(os.Stderr, "[api-gateway] shutting down\n")
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

func adminOnly() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("role").(types.Role)
			ok := false
			for _, a := range types.AdminRoles {
				if a == role {
					ok = true
					break
				}
			}
			if role == types.RoleSystemAdmin {
				ok = true
			}
			if !ok {
				httpx.ErrForbidden(w, "admin role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func financeOnly() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("role").(types.Role)
			ok := false
			for _, a := range types.FinanceRoles {
				if a == role {
					ok = true
					break
				}
			}
			if !ok {
				httpx.ErrForbidden(w, "finance role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func placeholderHandler(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]interface{}{
			"resource": name,
			"method":   r.Method,
			"status":   "placeholder_stub",
			"message":  "route registered — handler to be ported from Django views",
		})
	}
}

func registerPlaceholderCRUD(r chi.Router, base string) {
	r.Get("/", placeholderHandler(base+" list"))
	r.Post("/", placeholderHandler(base+" create"))
	r.Get("/{id}/", placeholderHandler(base+" detail"))
	r.Put("/{id}/", placeholderHandler(base+" update"))
	r.Patch("/{id}/", placeholderHandler(base+" partial update"))
	r.Delete("/{id}/", placeholderHandler(base+" delete"))
}
