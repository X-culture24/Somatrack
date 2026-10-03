package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"golang.org/x/time/rate"

	lmsmiddleware "github.com/stmaryskabete/lms/internal/middleware"
	"github.com/stmaryskabete/lms/internal/shared/crypto"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
	jwtutil "github.com/stmaryskabete/lms/internal/shared/jwt"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

type AuthService struct {
	pool         *pgxpool.Pool
	jwt          *jwtutil.Manager
	novaRoles    []types.Role
	novaURL      string
	nairobiLoc   *time.Location
	notification *rpc.NotificationClient
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
		r.Post("/auth/change-password/", a.handleChangePassword)
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

// handleChangePassword is real self-service password change — the "Change
// Password" menu item on both the teacher and parent mobile profile screens
// (and the web equivalent) previously did nothing at all.
func (a *AuthService) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		httpx.ErrUnauthorized(w, "user missing")
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	u, err := a.getUserByID(r.Context(), uid)
	if err != nil {
		httpx.ErrNotFound(w, "user not found")
		return
	}
	if _, err := crypto.VerifyPassword(body.CurrentPassword, u.PasswordHash); err != nil {
		httpx.ErrBadRequest(w, "current password is incorrect")
		return
	}
	hash, err := crypto.HashPassword(body.NewPassword)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := a.pool.Exec(r.Context(), `
		UPDATE users SET password = $1, must_change_password = false, updated_at = NOW() WHERE id = $2`,
		hash, uid); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "password changed"})
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
		ID        uuid.UUID
		UserID    uuid.UUID
		UsedAt    *time.Time
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

// HealthResponse/HealthHandler moved to health.go (now aggregates downstream
// service health too).

type DashboardService struct {
	pool    *pgxpool.Pool
	payment *rpc.PaymentClient
	reports *rpc.ReportClient
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
	type summary struct {
		StudentCount             int     `json:"student_count"`
		ClassCount               int     `json:"class_count"`
		OpenAdmissionsCount      int     `json:"open_admissions_count"`
		OutstandingFeesTotal     float64 `json:"outstanding_fees_total"`
		UnreconciledPaymentCount int     `json:"unreconciled_payment_count"`
		TeacherCount             int     `json:"teacher_count"`
		ParentCount              int     `json:"parent_count"`
		TotalCollected           float64 `json:"total_collected"`
	}
	var res summary
	_ = d.pool.QueryRow(r.Context(), `
		SELECT
			(SELECT COUNT(*) FROM students WHERE is_active = TRUE)::int,
			(SELECT COUNT(*) FROM class_groups)::int,
			(SELECT COUNT(*) FROM admission_applications WHERE status = 'pending')::int,
			COALESCE((SELECT SUM(balance) FROM invoices WHERE balance > 0),0),
			(SELECT COUNT(*) FROM mpesa_webhook_events WHERE status = 'failed')::int,
			(SELECT COUNT(*) FROM users WHERE role IN ('class_teacher','subject_teacher','head_teacher','deputy_head'))::int,
			(SELECT COUNT(*) FROM users WHERE role = 'parent')::int,
			COALESCE((SELECT SUM(amount) FROM payments WHERE status = 'confirmed'),0)
		`).Scan(
		&res.StudentCount, &res.ClassCount, &res.OpenAdmissionsCount,
		&res.OutstandingFeesTotal, &res.UnreconciledPaymentCount,
		&res.TeacherCount, &res.ParentCount, &res.TotalCollected,
	)

	type recentStudent struct {
		FullName     string `json:"full_name"`
		AdmissionNo  string `json:"admission_no"`
		Gender       string `json:"gender"`
		AdmissionDate string `json:"admission_date"`
		CreatedAt    string `json:"created_at"`
	}
	recentStudents := []recentStudent{}
	sRows, _ := d.pool.Query(r.Context(), `
		SELECT full_name, admission_no, gender,
		       COALESCE(admission_date::text, ''),
		       to_char(created_at, 'DD/MM/YYYY')
		FROM students WHERE is_active = TRUE ORDER BY created_at DESC LIMIT 8`)
	if sRows != nil {
		defer sRows.Close()
		for sRows.Next() {
			var s recentStudent
			if sRows.Scan(&s.FullName, &s.AdmissionNo, &s.Gender, &s.AdmissionDate, &s.CreatedAt) == nil {
				recentStudents = append(recentStudents, s)
			}
		}
	}

	type recentPayment struct {
		StudentName   string  `json:"student_name"`
		Amount        float64 `json:"amount"`
		PaymentDate   string  `json:"payment_date"`
		PayerName     string  `json:"payer_name"`
		PaymentMethod string  `json:"payment_method"`
	}
	recentPayments := []recentPayment{}
	pRows, _ := d.pool.Query(r.Context(), `
		SELECT s.full_name, p.amount::float8,
		       to_char(p.payment_date, 'DD/MM/YYYY'), p.payer_name, p.payment_method
		FROM payments p
		JOIN students s ON s.id = p.student_id
		WHERE p.status = 'confirmed'
		ORDER BY p.created_at DESC LIMIT 8`)
	if pRows != nil {
		defer pRows.Close()
		for pRows.Next() {
			var p recentPayment
			if pRows.Scan(&p.StudentName, &p.Amount, &p.PaymentDate, &p.PayerName, &p.PaymentMethod) == nil {
				recentPayments = append(recentPayments, p)
			}
		}
	}

	type activity struct {
		ID        uuid.UUID `json:"id"`
		Type      string    `json:"type"`
		Message   string    `json:"message"`
		CreatedAt time.Time `json:"created_at"`
	}
	activities := []activity{}
	aRows, _ := d.pool.Query(r.Context(), `
		SELECT id, 'payment' AS type,
		       format('Payment of KES %s received from %s', amount, payer_name) AS message,
		       created_at
		FROM payments ORDER BY created_at DESC LIMIT 10`)
	if aRows != nil {
		defer aRows.Close()
		for aRows.Next() {
			var a activity
			if aRows.Scan(&a.ID, &a.Type, &a.Message, &a.CreatedAt) == nil {
				activities = append(activities, a)
			}
		}
	}

	out, _ := json.Marshal(map[string]interface{}{
		"student_count":              res.StudentCount,
		"class_count":                res.ClassCount,
		"open_admissions_count":      res.OpenAdmissionsCount,
		"outstanding_fees_total":     res.OutstandingFeesTotal,
		"unreconciled_payment_count": res.UnreconciledPaymentCount,
		"teacher_count":              res.TeacherCount,
		"parent_count":               res.ParentCount,
		"total_collected":            res.TotalCollected,
		"recent_students":            recentStudents,
		"recent_payments":            recentPayments,
		"recent_activity":            activities,
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func (d *DashboardService) teacher(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	classes, err := myClassesFor(r.Context(), d.pool, uid)
	if err != nil {
		log.Printf("[dashboard] teacher classes for %s: %v", uid, err)
		classes = []TeacherClass{}
	}
	learnerCount := 0
	for _, c := range classes {
		learnerCount += c.StudentCount
	}

	recentPlans := []map[string]interface{}{}
	rows, err := d.pool.Query(r.Context(), `
		SELECT lp.topic, lp.lesson_date::text, COALESCE(cg.name,''), COALESCE(subj.name,'')
		FROM lesson_plans lp
		LEFT JOIN class_groups cg ON cg.id = lp.class_id
		LEFT JOIN subjects subj ON subj.id = lp.subject_id
		WHERE lp.teacher_id = $1 ORDER BY lp.lesson_date DESC LIMIT 5`, uid)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var topic, date, class, subj string
			if rows.Scan(&topic, &date, &class, &subj) == nil {
				recentPlans = append(recentPlans, map[string]interface{}{
					"topic": topic, "lesson_date": date, "class_name": class, "subject_name": subj,
				})
			}
		}
	}

	// Today's timetable slots for this teacher (day_of_week: 1=Mon…7=Sun).
	type slot struct {
		Period        int    `json:"period"`
		StartTime     string `json:"start_time"`
		EndTime       string `json:"end_time"`
		SubjectName   string `json:"subject_name"`
		ClassName     string `json:"class_name"`
		Venue         string `json:"venue,omitempty"`
		AttendanceDone bool  `json:"attendance_done"`
	}
	todaySlots := []slot{}
	srows, _ := d.pool.Query(r.Context(), `
		SELECT ts.period_number, ts.start_time::text, ts.end_time::text,
		       COALESCE(subj.name,''), COALESCE(cg.name,''), ts.venue,
		       EXISTS(
		           SELECT 1 FROM attendance_records ar
		           WHERE ar.class_id = ts.class_id
		             AND ar.subject_id = ts.subject_id
		             AND ar.date = CURRENT_DATE
		       ) AS attendance_done
		FROM timetable_slots ts
		JOIN terms t ON t.id = ts.term_id AND CURRENT_DATE BETWEEN t.start_date AND t.end_date
		LEFT JOIN subjects subj ON subj.id = ts.subject_id
		LEFT JOIN class_groups cg ON cg.id = ts.class_id
		WHERE ts.teacher_id = $1
		  AND ts.day_of_week = EXTRACT(ISODOW FROM CURRENT_DATE)::int
		ORDER BY ts.period_number`, uid)
	if srows != nil {
		defer srows.Close()
		for srows.Next() {
			var s slot
			if srows.Scan(&s.Period, &s.StartTime, &s.EndTime,
				&s.SubjectName, &s.ClassName, &s.Venue, &s.AttendanceDone) == nil {
				todaySlots = append(todaySlots, s)
			}
		}
	}

	// How many students have attendance already marked today in homeroom classes.
	attendanceMarkedToday := 0
	for _, c := range classes {
		if !c.IsHomeroom {
			continue
		}
		var n int
		_ = d.pool.QueryRow(r.Context(), `
			SELECT COUNT(*)::int FROM attendance_records
			WHERE class_id = $1 AND date = CURRENT_DATE AND subject_id IS NULL`,
			c.ID).Scan(&n)
		attendanceMarkedToday += n
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"homeroom_classes":       classes,
		"learner_count":          learnerCount,
		"nova_course_count":      0,
		"open_assignment_count":  0,
		"recent_lesson_plans":    recentPlans,
		"recent_marks":           []interface{}{},
		"recent_attendance":      []interface{}{},
		"today_slots":            todaySlots,
		"attendance_marked_today": attendanceMarkedToday,
	})
}

func (d *DashboardService) parent(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	type attendanceSummary struct {
		Present int `json:"present"`
		Absent  int `json:"absent"`
		Late    int `json:"late"`
		Days    int `json:"days"` // school days in the last 30 days
	}
	type child struct {
		StudentID   uuid.UUID         `json:"student_id"`
		FullName    string            `json:"full_name"`
		AdmissionNo string            `json:"admission_no"`
		ClassName   string            `json:"class_name,omitempty"`
		FeeBalance  float64           `json:"fee_balance"`
		Attendance  attendanceSummary `json:"attendance"`
	}
	children := []child{}
	rows, err := d.pool.Query(r.Context(), `
		SELECT st.id, st.full_name, st.admission_no, COALESCE(cg.name, ''),
		       COALESCE((SELECT SUM(i.balance) FROM invoices i WHERE i.student_id = st.id), 0)
		FROM guardians g
		JOIN student_guardians sg ON sg.guardian_id = g.id
		JOIN students st ON st.id = sg.student_id
		LEFT JOIN class_groups cg ON cg.id = st.current_class_id
		WHERE g.user_id = $1
		ORDER BY st.full_name`, uid)
	if err != nil {
		log.Printf("[dashboard] parent children for %s: %v", uid, err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var c child
			if rows.Scan(&c.StudentID, &c.FullName, &c.AdmissionNo, &c.ClassName, &c.FeeBalance) == nil {
				children = append(children, c)
			}
		}
	}
	// Attendance last 30 days per child.
	for i, c := range children {
		var sum attendanceSummary
		_ = d.pool.QueryRow(r.Context(), `
			SELECT
				COUNT(*) FILTER (WHERE status = 'present')::int,
				COUNT(*) FILTER (WHERE status = 'absent')::int,
				COUNT(*) FILTER (WHERE status = 'late')::int,
				COUNT(*)::int
			FROM attendance_records
			WHERE student_id = $1 AND subject_id IS NULL AND date >= CURRENT_DATE - 30`,
			c.StudentID).Scan(&sum.Present, &sum.Absent, &sum.Late, &sum.Days)
		children[i].Attendance = sum
	}
	// Announcements aimed at parents.
	type announcement struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
	}
	announcements := []announcement{}
	arows, _ := d.pool.Query(r.Context(), `
		SELECT id::text, title, COALESCE(body,''), created_at::text
		FROM announcements
		WHERE audience IN ('all','parent') AND is_published = TRUE
		ORDER BY created_at DESC LIMIT 5`)
	if arows != nil {
		defer arows.Close()
		for arows.Next() {
			var a announcement
			if arows.Scan(&a.ID, &a.Title, &a.Body, &a.CreatedAt) == nil {
				announcements = append(announcements, a)
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"children":      children,
		"announcements": announcements,
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
