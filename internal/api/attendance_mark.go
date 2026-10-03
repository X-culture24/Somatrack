package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// AttendanceMarkService is the write side that internal/api/fingerprint_proxy.go's
// daily-summary/roster/range read endpoints never had a counterpart for — the
// mobile "Take Attendance" screen let a teacher tap through a whole class,
// mark everyone present/late/absent, tap Save, and see "Attendance saved
// successfully", but no request was ever sent (a hardcoded 1-second delay
// then a fake success SnackBar). Writes directly into attendance_records
// (the same table fingerprint-svc's scan pipeline and the read endpoints
// use), since there's no manual-entry counterpart on fingerprint-svc either
// and this table lives in the same shared Postgres database.
type AttendanceMarkService struct {
	pool *pgxpool.Pool
}

func NewAttendanceMarkService(pool *pgxpool.Pool) *AttendanceMarkService {
	return &AttendanceMarkService{pool: pool}
}

func (s *AttendanceMarkService) RegisterRoutes(r chi.Router) {
	r.Post("/attendance/mark/", adminOrTeacherOnly()(http.HandlerFunc(s.mark)).ServeHTTP)
}

func adminOrTeacherOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value("role").(types.Role)
			ok := isAdminRole(role)
			if !ok {
				for _, t := range types.TeacherRoles {
					if t == role {
						ok = true
						break
					}
				}
			}
			if !ok {
				httpx.ErrForbidden(w, "admin or teacher role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type attendanceMarkEntry struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
	Status    string    `json:"status" validate:"required,oneof=present absent late sick permission holiday suspended"`
}

type attendanceMarkBody struct {
	ClassID uuid.UUID             `json:"class_id" validate:"required"`
	Date    string                `json:"date" validate:"required"`
	Entries []attendanceMarkEntry `json:"entries" validate:"required,min=1,dive"`
}

func (s *AttendanceMarkService) mark(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	var body attendanceMarkBody
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())

	for _, e := range body.Entries {
		// Targets the same partial unique index the fingerprint scan
		// pipeline relies on for a whole-day (not per-subject) record —
		// re-marking a student for the same day overwrites, not duplicates.
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO attendance_records (student_id, class_id, date, status, marked_by_id)
			VALUES ($1, $2, $3::date, $4, $5)
			ON CONFLICT (student_id, date) WHERE subject_id IS NULL
			DO UPDATE SET status = $4, class_id = $2, marked_by_id = $5`,
			e.StudentID, body.ClassID, body.Date, e.Status, userID); err != nil {
			httpx.Error(w, http.StatusBadRequest, err)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "marked": len(body.Entries)})
}
