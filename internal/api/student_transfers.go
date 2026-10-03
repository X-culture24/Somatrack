package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
)

// StudentTransfersService moves a student between classes (an internal
// promotion/stream change/mid-year move) or out of the school entirely
// (to_class_id omitted — the student is marked "transferred" and
// deactivated), recording every move as an immutable history row. This
// replaced a registerPlaceholderCRUD stub that accepted nothing real.
type StudentTransfersService struct {
	pool    *pgxpool.Pool
	finance *rpc.FinanceClient
}

func NewStudentTransfersService(pool *pgxpool.Pool, finance *rpc.FinanceClient) *StudentTransfersService {
	return &StudentTransfersService{pool: pool, finance: finance}
}

func (s *StudentTransfersService) RegisterRoutes(r chi.Router) {
	r.Route("/student-transfers/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", s.create)
		r.Get("/{id}/", s.get)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.delete)).ServeHTTP)
	})
}

type StudentTransfer struct {
	ID            uuid.UUID  `json:"id"`
	StudentID     uuid.UUID  `json:"student_id" validate:"required"`
	TransferDate  string     `json:"transfer_date,omitempty"`
	FromClassID   *uuid.UUID `json:"from_class_id,omitempty"`
	ToClassID     *uuid.UUID `json:"to_class_id,omitempty"`
	Reason        string     `json:"reason"`
	ApprovedByID  *uuid.UUID `json:"approved_by_id,omitempty"`
	StudentName   string     `json:"student_name,omitempty"`
	AdmissionNo   string     `json:"admission_no,omitempty"`
	FromClassName string     `json:"from_class_name,omitempty"`
	ToClassName   string     `json:"to_class_name,omitempty"`
	ApprovedBy    string     `json:"approved_by,omitempty"`
}

const transferCols = `t.id, t.student_id, t.transfer_date::text, t.from_class_id, t.to_class_id, t.reason,
	t.approved_by_id, st.full_name, st.admission_no, COALESCE(fc.name, ''), COALESCE(tc.name, ''),
	COALESCE(u.first_name || ' ' || u.last_name, '')`

const transferJoin = `student_transfers t
	JOIN students st ON st.id = t.student_id
	LEFT JOIN class_groups fc ON fc.id = t.from_class_id
	LEFT JOIN class_groups tc ON tc.id = t.to_class_id
	LEFT JOIN users u ON u.id = t.approved_by_id`

func scanTransfer(row interface {
	Scan(dest ...interface{}) error
}) (StudentTransfer, error) {
	var t StudentTransfer
	err := row.Scan(&t.ID, &t.StudentID, &t.TransferDate, &t.FromClassID, &t.ToClassID, &t.Reason,
		&t.ApprovedByID, &t.StudentName, &t.AdmissionNo, &t.FromClassName, &t.ToClassName, &t.ApprovedBy)
	return t, err
}

func (s *StudentTransfersService) list(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("student_id")
	var rows pgx.Rows
	var err error
	if studentID != "" {
		rows, err = s.pool.Query(r.Context(), `SELECT `+transferCols+` FROM `+transferJoin+`
			WHERE t.student_id = $1 ORDER BY t.transfer_date DESC, t.created_at DESC`, studentID)
	} else {
		rows, err = s.pool.Query(r.Context(), `SELECT `+transferCols+` FROM `+transferJoin+`
			ORDER BY t.transfer_date DESC, t.created_at DESC LIMIT 200`)
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []StudentTransfer{}
	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *StudentTransfersService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	t, err := scanTransfer(s.pool.QueryRow(r.Context(), `SELECT `+transferCols+` FROM `+transferJoin+` WHERE t.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "transfer not found")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

// create is the whole feature: it records the move AND actually re-points
// the student at their new class (or, if to_class_id is omitted, marks them
// transferred out and deactivates them) in one transaction, then re-runs
// auto-billing for an internal move so the new class's fee structure takes
// effect immediately — exactly like a promotion via the Students edit form,
// except this also leaves a permanent paper trail of the move.
func (s *StudentTransfersService) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID    uuid.UUID  `json:"student_id" validate:"required"`
		ToClassID    *uuid.UUID `json:"to_class_id,omitempty"`
		Reason       string     `json:"reason"`
		TransferDate string     `json:"transfer_date,omitempty"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.TransferDate == "" {
		body.TransferDate = time.Now().Format("2006-01-02")
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())

	var fromClassID *uuid.UUID
	if err := tx.QueryRow(r.Context(), `SELECT current_class_id FROM students WHERE id = $1`, body.StudentID).
		Scan(&fromClassID); err != nil {
		if err == pgx.ErrNoRows {
			httpx.ErrNotFound(w, "student not found")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}

	var approvedBy *uuid.UUID
	if uid, ok := r.Context().Value("user_id").(uuid.UUID); ok {
		approvedBy = &uid
	}

	var transferID uuid.UUID
	err = tx.QueryRow(r.Context(), `
		INSERT INTO student_transfers (student_id, transfer_date, from_class_id, to_class_id, reason, approved_by_id)
		VALUES ($1, $2::date, $3, $4, $5, $6) RETURNING id`,
		body.StudentID, body.TransferDate, fromClassID, body.ToClassID, body.Reason, approvedBy,
	).Scan(&transferID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}

	leavingSchool := body.ToClassID == nil
	_, err = tx.Exec(r.Context(), `
		UPDATE students SET
			current_class_id = $1,
			status = CASE WHEN $2::boolean THEN 'transferred' ELSE status END,
			is_active = CASE WHEN $2::boolean THEN false ELSE is_active END,
			updated_at = NOW()
		WHERE id = $3`,
		body.ToClassID, leavingSchool, body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	if !leavingSchool && s.finance != nil {
		_, _ = s.finance.BillStudent(r.Context(), body.StudentID)
	}

	t, err := scanTransfer(s.pool.QueryRow(r.Context(), `SELECT `+transferCols+` FROM `+transferJoin+` WHERE t.id = $1`, transferID))
	if err != nil {
		httpx.ErrNotFound(w, "transfer not found")
		return
	}
	httpx.JSON(w, http.StatusCreated, t)
}

func (s *StudentTransfersService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM student_transfers WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}
