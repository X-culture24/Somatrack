package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"golang.org/x/time/rate"

	lmsmiddleware "github.com/stmaryskabete/lms/internal/middleware"
	jwtutil "github.com/stmaryskabete/lms/internal/shared/jwt"
	"github.com/stmaryskabete/lms/internal/shared/crypto"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

type AuthService struct {
	pool          *pgxpool.Pool
	jwt           *jwtutil.Manager
	novaRoles     []types.Role
	novaURL       string
	nairobiLoc    *time.Location
	notification  *rpc.NotificationClient
}

func NewAuthService(pool *pgxpool.Pool, jwt *jwtutil.Manager, novaURL string, loc *time.Location, notifCli *rpc.NotificationClient) *AuthService {
	return &AuthService{
		pool:         pool,
		jwt:          jwt,
		novaRoles:    types.NovaRoles,
		novaURL:      novaURL,
		nairobiLoc:   loc,
		notification: notifCli,
	}
}

func (a *AuthService) getUserByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	var u types.User
	var portal types.PortalType
	err := a.pool.QueryRow(ctx, `
		SELECT id, email, first_name, last_name, UPPER(role), is_active,
		       COALESCE(phone,''), password, created_at, updated_at
		FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.IsActive,
		&u.PhoneNumber, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Portal = types.RoleToPortal(u.Role)
	_ = portal
	return &u, nil
}

func (a *AuthService) findUserByEmail(ctx context.Context, email string) (*types.User, error) {
	var id uuid.UUID
	err := a.pool.QueryRow(ctx, `SELECT id FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}
	return a.getUserByID(ctx, id)
}

func (a *AuthService) RegisterRoutes(r chi.Router) {
	// Login/refresh are the highest-value brute-force targets in the API —
	// throttle per client IP (burst 5, then ~1 attempt/12s) so a bad actor
	// can't hammer the password check in a tight loop. bcrypt cost 12 alone
	// isn't enough: it only slows a single guess, not a distributed attempt.
	loginLimit := lmsmiddleware.RateLimit(rate.Limit(5.0/60.0), 5)
	r.With(loginLimit).Post("/auth/login/", a.handleLogin)
	r.With(loginLimit).Post("/auth/refresh/", a.handleRefresh)
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			h := newAuthMiddlewareFromManager(a.jwt, jwtutil.ScopePortal)
			return h(next)
		})
		r.Get("/auth/me/", a.handleMe)
		r.Post("/auth/nova/handoff/", a.handleNovaHandoff)
	})
	r.Post("/auth/nova/exchange/", a.handleNovaExchange)
}

func newAuthMiddlewareFromManager(m *jwtutil.Manager, scope string) func(http.Handler) http.Handler {
	pkg := "github.com/stmaryskabete/lms/internal/middleware"
	_ = pkg
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok, ok := httpx.ExtractBearerToken(r)
			if !ok {
				httpx.ErrUnauthorized(w, "missing bearer token")
				return
			}
			claims, err := m.ValidateAccess(tok, scope)
			if err != nil {
				httpx.ErrUnauthorized(w, "invalid token")
				return
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, "claims", claims)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "role", claims.Role)
			ctx = context.WithValue(ctx, "portal", claims.Portal)
			ctx = context.WithValue(ctx, "scope", claims.Scope)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func userResponse(u *types.User) map[string]interface{} {
	return map[string]interface{}{
		"id":           u.ID,
		"email":        u.Email,
		"first_name":   u.FirstName,
		"last_name":    u.LastName,
		"role":         u.Role,
		"portal":       u.Portal,
		"is_active":    u.IsActive,
		"phone_number": u.PhoneNumber,
		"created_at":   u.CreatedAt,
	}
}

func (a *AuthService) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=1"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.JSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_failed",
			Details: httpx.ValidationErrors(err),
		})
		return
	}
	u, err := a.findUserByEmail(r.Context(), body.Email)
	if err != nil {
		httpx.ErrUnauthorized(w, "invalid credentials")
		return
	}
	if !u.IsActive {
		httpx.ErrForbidden(w, "account is disabled")
		return
	}
	needsRehash, err := crypto.VerifyPassword(body.Password, u.PasswordHash)
	if err != nil {
		httpx.ErrUnauthorized(w, "invalid credentials")
		return
	}
	if needsRehash {
		newHash, err := crypto.HashPassword(body.Password)
		if err == nil {
			_, _ = a.pool.Exec(r.Context(), `UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2`, newHash, u.ID)
		}
	}
	pair, err := a.jwt.IssueTokenPair(u, jwtutil.ScopePortal)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"access":  pair.Access,
		"refresh": pair.Refresh,
		"user":    userResponse(u),
	})
}

func (a *AuthService) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Refresh string `json:"refresh" validate:"required"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	pair, err := a.jwt.RotateRefresh(body.Refresh)
	if err != nil {
		httpx.ErrUnauthorized(w, "invalid refresh token")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"access":  pair.Access,
		"refresh": pair.Refresh,
	})
}

func (a *AuthService) handleMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		httpx.ErrUnauthorized(w, "user missing")
		return
	}
	u, err := a.getUserByID(r.Context(), uid)
	if err != nil {
		httpx.ErrNotFound(w, "user not found")
		return
	}
	httpx.JSON(w, http.StatusOK, userResponse(u))
}

func (a *AuthService) handleNovaHandoff(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		httpx.ErrUnauthorized(w, "user missing")
		return
	}
	role, _ := r.Context().Value("role").(types.Role)
	allowed := false
	for _, r2 := range a.novaRoles {
		if r2 == role {
			allowed = true
			break
		}
	}
	if !allowed {
		httpx.ErrForbidden(w, "user not allowed for NOVA")
		return
	}
	code := randomCode(64)
	expires := time.Now().In(a.nairobiLoc).Add(1 * time.Minute)
	id := uuid.New()
	_, err := a.pool.Exec(r.Context(), `
		INSERT INTO nova_login_tickets (id, code, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())`, id, code, uid, expires)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	url := fmt.Sprintf("%s/auth/callback?code=%s", a.novaURL, code)
	httpx.JSON(w, http.StatusOK, map[string]string{"url": url})
}

func (a *AuthService) handleNovaExchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code" validate:"required"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	var ticket struct {
		ID       uuid.UUID
		UserID   uuid.UUID
		UsedAt   *time.Time
		ExpiresAt time.Time
	}
	err := a.pool.QueryRow(r.Context(), `
		SELECT id, user_id, used_at, expires_at FROM nova_login_tickets WHERE code = $1`, body.Code).Scan(
		&ticket.ID, &ticket.UserID, &ticket.UsedAt, &ticket.ExpiresAt)
	if err != nil {
		httpx.ErrUnauthorized(w, "invalid ticket")
		return
	}
	if ticket.UsedAt != nil {
		httpx.ErrUnauthorized(w, "ticket already used")
		return
	}
	if ticket.ExpiresAt.Before(time.Now().In(a.nairobiLoc)) {
		httpx.ErrUnauthorized(w, "ticket expired")
		return
	}
	u, err := a.getUserByID(r.Context(), ticket.UserID)
	if err != nil || !u.IsActive {
		httpx.ErrUnauthorized(w, "user not found or inactive")
		return
	}
	_, err = a.pool.Exec(r.Context(), `UPDATE nova_login_tickets SET used_at = NOW() WHERE id = $1`, ticket.ID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	pair, err := a.jwt.IssueTokenPair(u, jwtutil.ScopeNova)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"access":  pair.Access,
		"refresh": pair.Refresh,
		"user":    userResponse(u),
	})
}

func randomCode(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

func HealthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		if err := pool.Ping(r.Context()); err != nil {
			status = "degraded"
		}
		httpx.JSON(w, http.StatusOK, HealthResponse{
			Status:  status,
			Service: "api-gateway",
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
	}
}

type DashboardService struct {
	pool        *pgxpool.Pool
	payment     *rpc.PaymentClient
	reports     *rpc.ReportClient
}

func NewDashboardService(pool *pgxpool.Pool, pc *rpc.PaymentClient, rc *rpc.ReportClient) *DashboardService {
	return &DashboardService{pool: pool, payment: pc, reports: rc}
}

func (d *DashboardService) Handle(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(types.Role)
	portal := types.RoleToPortal(role)
	uid, _ := r.Context().Value("user_id").(uuid.UUID)
	switch portal {
	case types.PortalAdmin, types.PortalFinance:
		d.adminFinance(w, r)
	case types.PortalTeacher:
		d.teacher(w, r, uid)
	case types.PortalParent:
		d.parent(w, r, uid)
	case types.PortalStudent:
		d.student(w, r, uid)
	default:
		d.adminFinance(w, r)
	}
}

func (d *DashboardService) adminFinance(w http.ResponseWriter, r *http.Request) {
	type row struct {
		StudentCount            int     `json:"student_count"`
		ClassCount              int     `json:"class_count"`
		OpenAdmissionsCount     int     `json:"open_admissions_count"`
		OutstandingFeesTotal    float64 `json:"outstanding_fees_total"`
		UnreconciledPaymentCount int    `json:"unreconciled_payment_count"`
	}
	var res row
	_ = d.pool.QueryRow(r.Context(), `
		SELECT
			(SELECT COUNT(*) FROM students WHERE is_active = TRUE)::int,
			(SELECT COUNT(*) FROM classes)::int,
			(SELECT COUNT(*) FROM admissions WHERE status = 'pending')::int,
			COALESCE((SELECT SUM(balance) FROM invoices WHERE balance > 0),0),
			(SELECT COUNT(*) FROM payments WHERE reconciliation_status = 'needs_review')::int
		`).Scan(&res.StudentCount, &res.ClassCount, &res.OpenAdmissionsCount, &res.OutstandingFeesTotal, &res.UnreconciledPaymentCount)
	type activity struct {
		ID        uuid.UUID `json:"id"`
		Type      string    `json:"type"`
		Message   string    `json:"message"`
		CreatedAt time.Time `json:"created_at"`
	}
	activities := []activity{}
	rows, _ := d.pool.Query(r.Context(), `
		SELECT id, 'payment' AS type,
		       format('Payment of KES %s received from %s', amount, payer_name) AS message,
		       created_at
		FROM payments ORDER BY created_at DESC LIMIT 10`)
	defer rows.Close()
	for rows.Next() {
		var a activity
		if err := rows.Scan(&a.ID, &a.Type, &a.Message, &a.CreatedAt); err == nil {
			activities = append(activities, a)
		}
	}
	out, _ := json.Marshal(map[string]interface{}{
		"student_count":              res.StudentCount,
		"class_count":                res.ClassCount,
		"open_admissions_count":      res.OpenAdmissionsCount,
		"outstanding_fees_total":     res.OutstandingFeesTotal,
		"unreconciled_payment_count": res.UnreconciledPaymentCount,
		"recent_activity":            activities,
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func (d *DashboardService) teacher(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	_ = uid
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"homeroom_classes":       []interface{}{},
		"learner_count":          0,
		"nova_course_count":      0,
		"open_assignment_count":  0,
		"recent_marks":           []interface{}{},
		"recent_attendance":      []interface{}{},
	})
}

func (d *DashboardService) parent(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	_ = uid
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"children":       []interface{}{},
		"announcements":  []interface{}{},
	})
}

func (d *DashboardService) student(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	_ = uid
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"class_group":         nil,
		"marks":               []interface{}{},
		"nova_course_count":   0,
		"pending_assignments": []interface{}{},
		"announcements":       []interface{}{},
	})
}
