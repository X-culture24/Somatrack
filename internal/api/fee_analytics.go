package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// FeeAnalyticsService feeds the admin dashboard's fee graphs (collections by
// term, outstanding by grade, invoice status mix, payment method mix). It
// reads invoices/payments directly off the gateway's own pool — the same
// tables dashboard.go's adminFinance summary already queries — rather than
// round-tripping through finance-svc, since this is read-only aggregate
// reporting, not a write path that needs finance-svc's business rules.
type FeeAnalyticsService struct {
	pool *pgxpool.Pool
}

func NewFeeAnalyticsService(pool *pgxpool.Pool) *FeeAnalyticsService {
	return &FeeAnalyticsService{pool: pool}
}

func (s *FeeAnalyticsService) RegisterRoutes(r chi.Router) {
	r.Get("/fee-analytics/", s.handle)
}

type termFeeStat struct {
	TermID      string  `json:"term_id"`
	Label       string  `json:"label"`
	Billed      float64 `json:"billed"`
	Collected   float64 `json:"collected"`
	Outstanding float64 `json:"outstanding"`
}

type gradeFeeStat struct {
	Grade       int     `json:"grade"`
	Billed      float64 `json:"billed"`
	Collected   float64 `json:"collected"`
	Outstanding float64 `json:"outstanding"`
}

type statusFeeStat struct {
	Status string  `json:"status"`
	Count  int     `json:"count"`
	Total  float64 `json:"total"`
}

type methodFeeStat struct {
	Method string  `json:"method"`
	Total  float64 `json:"total"`
}

func (s *FeeAnalyticsService) handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	byTerm := []termFeeStat{}
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.name || ' ' || ay.name AS label,
		       COALESCE(SUM(i.total_amount), 0), COALESCE(SUM(i.total_paid), 0), COALESCE(SUM(i.balance), 0)
		FROM terms t
		JOIN academic_years ay ON ay.id = t.academic_year_id
		LEFT JOIN invoices i ON i.term_id = t.id
		GROUP BY t.id, t.name, ay.name, ay.start_date, t.term_number
		ORDER BY ay.start_date, t.term_number
		LIMIT 12`)
	if err == nil {
		for rows.Next() {
			var v termFeeStat
			if rows.Scan(&v.TermID, &v.Label, &v.Billed, &v.Collected, &v.Outstanding) == nil {
				byTerm = append(byTerm, v)
			}
		}
		rows.Close()
	}

	byGrade := []gradeFeeStat{}
	rows, err = s.pool.Query(ctx, `
		SELECT cg.grade,
		       COALESCE(SUM(i.total_amount), 0), COALESCE(SUM(i.total_paid), 0), COALESCE(SUM(i.balance), 0)
		FROM invoices i
		JOIN students st ON st.id = i.student_id
		JOIN class_groups cg ON cg.id = st.current_class_id
		GROUP BY cg.grade
		ORDER BY cg.grade`)
	if err == nil {
		for rows.Next() {
			var v gradeFeeStat
			if rows.Scan(&v.Grade, &v.Billed, &v.Collected, &v.Outstanding) == nil {
				byGrade = append(byGrade, v)
			}
		}
		rows.Close()
	}

	byStatus := []statusFeeStat{}
	rows, err = s.pool.Query(ctx, `
		SELECT status, COUNT(*), COALESCE(SUM(total_amount), 0)
		FROM invoices GROUP BY status ORDER BY status`)
	if err == nil {
		for rows.Next() {
			var v statusFeeStat
			if rows.Scan(&v.Status, &v.Count, &v.Total) == nil {
				byStatus = append(byStatus, v)
			}
		}
		rows.Close()
	}

	byMethod := []methodFeeStat{}
	rows, err = s.pool.Query(ctx, `
		SELECT payment_method, COALESCE(SUM(amount), 0)
		FROM payments WHERE status = 'confirmed' GROUP BY payment_method ORDER BY SUM(amount) DESC`)
	if err == nil {
		for rows.Next() {
			var v methodFeeStat
			if rows.Scan(&v.Method, &v.Total) == nil {
				byMethod = append(byMethod, v)
			}
		}
		rows.Close()
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"by_term":         byTerm,
		"by_grade":        byGrade,
		"by_status":       byStatus,
		"payment_methods": byMethod,
	})
}
