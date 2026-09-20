package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// FinanceProxy exposes fee management (invoices/payments/fee-structures/discounts)
// and payroll (runs/entries/M-Pesa B2C disbursement) through the api-gateway,
// backed by finance-svc. It replaces the placeholder stubs previously registered
// for these resources in cmd/api/main.go.
type FinanceProxy struct {
	client *rpc.FinanceClient
	pool   *pgxpool.Pool
}

func NewFinanceProxy(c *rpc.FinanceClient, pool *pgxpool.Pool) *FinanceProxy {
	return &FinanceProxy{client: c, pool: pool}
}

func (p *FinanceProxy) RegisterRoutes(r chi.Router) {
	r.Get("/invoices/", p.requireAuthenticated(p.handleListInvoices))
	r.Get("/invoices/{id}/", p.requireAuthenticated(p.handleGetInvoice))
	r.Get("/payments/", p.requireAuthenticated(p.handleListPayments))
	r.Get("/fee-structures/", p.requireAuthenticated(p.handleListFeeStructures))
	r.Get("/discounts/", p.requireAuthenticated(p.handleListDiscounts))

	r.Post("/payroll-runs/", p.requireFinance(p.handleGeneratePayrollRun))
	r.Get("/payroll-runs/", p.requireFinance(p.handleListPayrollRuns))
	r.Get("/payroll-runs/{id}/entries/", p.requireFinance(p.handleListPayrollEntries))
	r.Post("/payroll-entries/{id}/disburse/", p.requireFinance(p.handleDisburseEntry))
}

func (p *FinanceProxy) requireAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value("user_id").(uuid.UUID); !ok {
			httpx.ErrUnauthorized(w, "auth required")
			return
		}
		next(w, r)
	}
}

func (p *FinanceProxy) requireFinance(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		next(w, r)
	}
}

// resolveStudentScope returns the set of student IDs the caller may view finance
// data for. A nil slice means "no restriction" (finance/admin roles). An empty,
// non-nil slice means "no students visible" (e.g. a parent with no wards).
func (p *FinanceProxy) resolveStudentScope(ctx context.Context, role types.Role, userID uuid.UUID) ([]uuid.UUID, error) {
	for _, a := range types.FinanceRoles {
		if a == role {
			return nil, nil
		}
	}
	if role == types.RoleSystemAdmin {
		return nil, nil
	}
	if p.pool == nil {
		return []uuid.UUID{}, nil
	}
	switch role {
	case types.RoleStudent:
		var id uuid.UUID
		err := p.pool.QueryRow(ctx, `SELECT id FROM students WHERE user_id = $1`, userID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return []uuid.UUID{}, nil
		}
		if err != nil {
			return nil, err
		}
		return []uuid.UUID{id}, nil
	case types.RoleParent:
		rows, err := p.pool.Query(ctx, `
			SELECT sg.student_id FROM guardians g
			JOIN student_guardians sg ON sg.guardian_id = g.id
			WHERE g.user_id = $1`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []uuid.UUID{}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				continue
			}
			out = append(out, id)
		}
		return out, nil
	default:
		// Teachers and other non-finance staff have no fee-data visibility.
		return []uuid.UUID{}, nil
	}
}

func inScope(scope []uuid.UUID, id uuid.UUID) bool {
	for _, s := range scope {
		if s == id {
			return true
		}
	}
	return false
}

func (p *FinanceProxy) handleListInvoices(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	scope, err := p.resolveStudentScope(r.Context(), role, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	var studentID *uuid.UUID
	if s := r.URL.Query().Get("student_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		studentID = &id
	}

	if scope != nil {
		if len(scope) == 0 {
			httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}, "total_count": 0})
			return
		}
		if studentID == nil {
			studentID = &scope[0]
		} else if !inScope(scope, *studentID) {
			httpx.ErrForbidden(w, "cannot view invoices for this student")
			return
		}
	}

	var status *string
	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	res, err := p.client.ListInvoices(r.Context(), studentID, status, page, pageSize)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FinanceProxy) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid invoice id")
		return
	}
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	scope, err := p.resolveStudentScope(r.Context(), role, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	inv, err := p.client.GetInvoice(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "invoice not found")
		return
	}
	if scope != nil && !inScope(scope, inv.StudentID) {
		httpx.ErrForbidden(w, "cannot view this invoice")
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (p *FinanceProxy) handleListPayments(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	scope, err := p.resolveStudentScope(r.Context(), role, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	var studentID *uuid.UUID
	if s := r.URL.Query().Get("student_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		studentID = &id
	}
	if scope != nil {
		if len(scope) == 0 {
			httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}, "total_count": 0})
			return
		}
		if studentID == nil {
			studentID = &scope[0]
		} else if !inScope(scope, *studentID) {
			httpx.ErrForbidden(w, "cannot view payments for this student")
			return
		}
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	res, err := p.client.ListPayments(r.Context(), studentID, page, pageSize)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (p *FinanceProxy) handleListFeeStructures(w http.ResponseWriter, r *http.Request) {
	var ayID *uuid.UUID
	if s := r.URL.Query().Get("academic_year_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid academic_year_id")
			return
		}
		ayID = &id
	}
	var grade *int
	if s := r.URL.Query().Get("grade"); s != "" {
		g, err := strconv.Atoi(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid grade")
			return
		}
		grade = &g
	}
	rows, err := p.client.ListFeeStructures(r.Context(), ayID, grade)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (p *FinanceProxy) handleListDiscounts(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	scope, err := p.resolveStudentScope(r.Context(), role, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	var studentID *uuid.UUID
	if s := r.URL.Query().Get("student_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid student_id")
			return
		}
		studentID = &id
	}
	if scope != nil {
		if len(scope) == 0 {
			httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}})
			return
		}
		if studentID == nil {
			studentID = &scope[0]
		} else if !inScope(scope, *studentID) {
			httpx.ErrForbidden(w, "cannot view discounts for this student")
			return
		}
	}

	rows, err := p.client.ListDiscounts(r.Context(), studentID)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (p *FinanceProxy) handleGeneratePayrollRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Month int `json:"month" validate:"required,min=1,max=12"`
		Year  int `json:"year" validate:"required"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	run, entries, err := p.client.GeneratePayrollRun(r.Context(), body.Month, body.Year, userID)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"run": run, "entries": entries})
}

func (p *FinanceProxy) handleListPayrollRuns(w http.ResponseWriter, r *http.Request) {
	rows, err := p.client.ListPayrollRuns(r.Context())
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (p *FinanceProxy) handleListPayrollEntries(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid run id")
		return
	}
	rows, err := p.client.ListPayrollEntries(r.Context(), runID)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"data": rows})
}

func (p *FinanceProxy) handleDisburseEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid entry id")
		return
	}
	tx, err := p.client.DisburseEntry(r.Context(), entryID)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, tx)
}
