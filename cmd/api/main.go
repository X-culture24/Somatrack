package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	jwtutil "github.com/stmaryskabete/lms/internal/shared/jwt"

	"github.com/stmaryskabete/lms/internal/api"
	"github.com/stmaryskabete/lms/internal/config"
	"github.com/stmaryskabete/lms/internal/db"
	"github.com/stmaryskabete/lms/internal/jenga"
	"github.com/stmaryskabete/lms/internal/signedlink"
	lmsmiddleware "github.com/stmaryskabete/lms/internal/middleware"
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
	academicsSvc := api.NewAcademicsService(pool)
	guardiansSvc := api.NewGuardiansService(pool)
	studentsSvc := api.NewStudentsService(pool, financeClient)
	admissionsSvc := api.NewAdmissionsService(pool, financeClient)
	clubsSvc := api.NewClubsService(pool)
	transportRoutesSvc := api.NewTransportRoutesService(pool)
	uploadsSvc := api.NewUploadsService(cfg.MediaRoot, cfg.MediaURL)
	usersSvc := api.NewUsersService(pool)
	announcementsSvc := api.NewAnnouncementsService(pool)
	librarySvc := api.NewLibraryService(pool)
	staffSvc := api.NewStaffService(pool)
	departmentsSvc := api.NewDepartmentsService(pool)
	vehiclesSvc := api.NewVehiclesService(pool)
	transfersSvc := api.NewStudentTransfersService(pool, financeClient)
	feeAnalyticsSvc := api.NewFeeAnalyticsService(pool)
	cbcSvc := api.NewCBCService(pool)
	reportsSvc := api.NewReportsService(reportClient)
	curriculumSvc := api.NewCurriculumService(pool)
	lessonPlansSvc := api.NewLessonPlansService(pool)
	teacherSvc := api.NewTeacherService(pool)
	attendanceMarkSvc := api.NewAttendanceMarkService(pool)
	payFeesSvc := api.NewPayFeesService(pool, paymentClient)

	// Document signing is mandatory: receipts and result slips carry signed QR
	// codes, and unsigned documents could be forged undetectably.
	signingKeys, err := signedlink.KeyringFromEnv()
	if err != nil {
		if errors.Is(err, signedlink.ErrNoKey) {
			log.Fatalf("document signing key: SIGNING_KEY_PATH is not set — generate one with `go run ./cmd/signing-keygen -out secrets/signing_ed25519.pem`")
		}
		log.Fatalf("document signing key: %v", err)
	}
	linksSvc := api.NewLinksService(pool, signingKeys, cfg.PortalURL)

	// Declarative record-keeping forms (internal/api/resources.go).
	resources := api.NewResourceRegistry(pool)
	if err := resources.Verify(ctx); err != nil {
		log.Fatalf("%v", err)
	}
	documentsSvc := api.NewDocumentsService(pool, linksSvc)
	resultsSvc := api.NewResultsService(pool, linksSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(120 * time.Second))
	r.Use(lmsmiddleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(lmsmiddleware.Security)

	r.Get("/api/health/", api.HealthHandler(pool, map[string]string{
		"fingerprint-svc":  fpURL,
		"payment-svc":      paymentURL,
		"notification-svc": notifURL,
		"report-svc":       reportURL,
		"finance-svc":      financeURL,
	}))

	// Public (unauthenticated — Safaricom carries no JWT) STK Push callback,
	// relayed to payment-svc's internal endpoint. Lets ngrok's single free
	// tunnel (pointed at this gateway for the mobile app) also carry the
	// STK Push callback, instead of needing a second tunnel to payment-svc.
	r.Post("/api/webhooks/payments/stk-callback", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			httpx.ErrBadRequest(w, err.Error())
			return
		}
		if err := paymentClient.ForwardSTKCallback(r.Context(), payload); err != nil {
			log.Printf("[api-gateway] forward stk callback: %v", err)
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"ResultCode": "0", "ResultDesc": "Accepted"})
	})

	// Public Equity Jenga IPN endpoint (Jenga HQ → "IPN callback URL"). Jenga
	// sends Basic auth (or ?secret=), checked here before relaying the body
	// to payment-svc over the internal channel, same as the STK callback.
	jengaIPNAuth := jenga.IPNAuthFromEnv()
	r.Post("/api/webhooks/payments/jenga-ipn", func(w http.ResponseWriter, r *http.Request) {
		if !jengaIPNAuth.Check(r) {
			log.Printf("[api-gateway] jenga ipn rejected from %s", r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var payload json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			httpx.ErrBadRequest(w, err.Error())
			return
		}
		// Unlike the STK callback, a failed relay returns 5xx so Jenga retries
		// instead of the payment silently going missing.
		if err := paymentClient.ForwardJengaIPN(r.Context(), payload); err != nil {
			log.Printf("[api-gateway] forward jenga ipn: %v", err)
			http.Error(w, "temporarily unavailable", http.StatusBadGateway)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "received"})
	})

	linksSvc.RegisterPublicRoutes(r)

	r.Route("/api", func(r chi.Router) {
		authSvc.RegisterRoutes(r)

		r.Route("/", func(r chi.Router) {
			portalAuth := lmsmiddleware.Auth(jwtMgr, jwtutil.ScopePortal)
			r.Use(portalAuth)

			r.Get("/dashboard/", dashboard.Handle)

			fpProxy.RegisterRoutes(r)
			attendanceMarkSvc.RegisterRoutes(r)
			payFeesSvc.RegisterRoutes(r)
			financeProxy.RegisterRoutes(r)
			documentsSvc.RegisterRoutes(r)
			linksSvc.RegisterRoutes(r)
			resources.Mount(r)
			resultsSvc.RegisterRoutes(r)
			studentsSvc.RegisterRoutes(r)
			guardiansSvc.RegisterRoutes(r)
			admissionsSvc.RegisterRoutes(r)
			academicsSvc.RegisterRoutes(r)
			academicsSvc.RegisterExamRoutes(r)
			clubsSvc.RegisterRoutes(r)
			transportRoutesSvc.RegisterRoutes(r)
			vehiclesSvc.RegisterRoutes(r)
			transfersSvc.RegisterRoutes(r)
			feeAnalyticsSvc.RegisterRoutes(r)
			cbcSvc.RegisterRoutes(r)
			uploadsSvc.RegisterRoutes(r)

			usersSvc.RegisterRoutes(r)
			r.Route("/student-documents/", func(r chi.Router) { registerPlaceholderCRUD(r, "student-documents") })
			// student-transfers/ is a real endpoint now — see transfersSvc.RegisterRoutes(r) above.

			// academic-years/, terms/, streams/, subjects/, classes/, grade-requirements/
			// are real endpoints now — see academicsSvc.RegisterRoutes(r) above.
			r.Get("/school/profile/", placeholderHandler("school profile"))
			// school/curriculum/ and class-subjects/ are real now — see
			// curriculumSvc.RegisterRoutes(r) below.
			curriculumSvc.RegisterRoutes(r)

			departmentsSvc.RegisterRoutes(r)
			staffSvc.RegisterRoutes(r)
			// payroll-runs/, payroll-entries/, invoices/, payments/, fee-structures/, discounts/
			// are real endpoints now — see financeProxy.RegisterRoutes(r) above.
			r.Route("/staff-deductions/", func(r chi.Router) { registerPlaceholderCRUD(r, "staff-deductions") })

			r.Route("/unreconciled-payments/", func(r chi.Router) {
				r.Use(financeOnly())
				registerPlaceholderCRUD(r, "unreconciled-payments")
			})
			r.Route("/finance/", func(r chi.Router) {
				r.Use(financeOnly())
				// Cash, cheque, bank slip or card taken at the bursar's desk:
				// posted, receipted and confirmed by SMS like any M-Pesa payment.
				r.Post("/manual-payment/", func(w http.ResponseWriter, r *http.Request) {
					var body map[string]interface{}
					if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
						httpx.ErrBadRequest(w, "request body must be a JSON object")
						return
					}
					uid, _ := r.Context().Value("user_id").(uuid.UUID)
					body["received_by_id"] = uid.String()
					out, err := paymentClient.RecordManualPayment(r.Context(), body)
					var rpcErr *rpc.RPCError
					if errors.As(err, &rpcErr) && rpcErr.StatusCode < 500 {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(rpcErr.StatusCode)
						_, _ = w.Write([]byte(rpcErr.Body))
						return
					}
					if err != nil {
						httpx.Error(w, http.StatusBadGateway, err)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write(out)
				})
				r.Get("/reconciliation/", placeholderHandler("reconciliation"))
				r.Post("/send-reminders/", func(w http.ResponseWriter, r *http.Request) {
					sent, err := paymentClient.SendReminders(r.Context())
					if err != nil {
						httpx.Error(w, http.StatusBadGateway, err)
						return
					}
					httpx.JSON(w, http.StatusOK, map[string]int{"sent": sent})
				})
				// simulate-payment lets the dashboard fire a Safaricom-shaped
				// confirmation through the real fee-listener pipeline (match,
				// payment, invoice recompute) without needing Safaricom's
				// actual network — the local equivalent of the C2B webhook,
				// for demos and manual QA. Authenticated + finance-only, so
				// it doesn't need the public webhook's shared secret.
				r.Post("/simulate-payment/", func(w http.ResponseWriter, r *http.Request) {
					var body struct {
						Amount        float64 `json:"amount" validate:"required,gt=0"`
						BillRefNumber string  `json:"bill_ref_number" validate:"required"`
						Phone         string  `json:"phone"`
						PayerName     string  `json:"payer_name"`
						TransID       string  `json:"trans_id"`
					}
					if err := httpx.BindAndValidate(r, &body); err != nil {
						httpx.ErrBadRequest(w, err.Error())
						return
					}
					if body.Phone == "" {
						body.Phone = "254708374149"
					}
					if body.TransID == "" {
						body.TransID = "SIM" + uuid.New().String()[:8]
					}
					firstName, lastName := body.PayerName, ""
					if parts := strings.SplitN(body.PayerName, " ", 2); len(parts) == 2 {
						firstName, lastName = parts[0], parts[1]
					}
					req := &rpc.ProcessMpesaPaymentRequest{
						TransID:       body.TransID,
						TransAmount:   body.Amount,
						BillRefNumber: body.BillRefNumber,
						TransTime:     time.Now().In(loc).Format("20060102150405"),
						FirstName:     firstName,
						LastName:      lastName,
						MSISDN:        body.Phone,
						RawPayload:    "simulated from admin dashboard",
					}
					res, err := paymentClient.ProcessMpesaPayment(r.Context(), req)
					if err != nil {
						api.WriteUpstreamError(w, err)
						return
					}
					httpx.JSON(w, http.StatusOK, res)
				})
			})

			r.Route("/attendance/", func(r chi.Router) { registerPlaceholderCRUD(r, "attendance") })
			// lesson-plans/ is a real endpoint now — see lessonPlansSvc.RegisterRoutes(r) below.
			lessonPlansSvc.RegisterRoutes(r)
			teacherSvc.RegisterRoutes(r)

			announcementsSvc.RegisterRoutes(r)
			r.Route("/messages/", func(r chi.Router) { registerPlaceholderCRUD(r, "messages") })

			// transport-routes/ and vehicles/ are real endpoints now — see
			// transportRoutesSvc/vehiclesSvc.RegisterRoutes(r) above.
			r.Route("/transport-attendance/", func(r chi.Router) { registerPlaceholderCRUD(r, "transport-attendance") })

			librarySvc.RegisterRoutes(r)
			r.Route("/library/reservations/", func(r chi.Router) { registerPlaceholderCRUD(r, "library/reservations") })


			reportsSvc.RegisterRoutes(r)
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
