package finance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeeStore struct {
	pool *pgxpool.Pool
}

func NewFeeStore(pool *pgxpool.Pool) *FeeStore {
	return &FeeStore{pool: pool}
}

type InvoiceSummary struct {
	ID         uuid.UUID `json:"id"`
	StudentID  uuid.UUID `json:"student_id"`
	InvoiceNo  string    `json:"invoice_no"`
	TermID     uuid.UUID `json:"term_id"`
	TermName   string    `json:"term_name,omitempty"`
	InvoiceDate string   `json:"invoice_date"`
	DueDate    string    `json:"due_date"`
	Status     string    `json:"status"`
	TotalDue   float64   `json:"total_due"`
	TotalPaid  float64   `json:"total_paid"`
	Balance    float64   `json:"balance"`
}

type InvoiceLine struct {
	ID          uuid.UUID `json:"id"`
	LineType    string    `json:"line_type"`
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Amount      float64   `json:"amount"`
}

type InvoiceDetail struct {
	InvoiceSummary
	Lines []InvoiceLine `json:"lines"`
}

func (s *FeeStore) ListInvoices(ctx context.Context, studentID *uuid.UUID, status *string, page, pageSize int) ([]InvoiceSummary, int64, error) {
	where := []string{"true"}
	args := []interface{}{}
	argIdx := 1
	if studentID != nil {
		where = append(where, fmt.Sprintf("i.student_id = $%d", argIdx))
		args = append(args, *studentID)
		argIdx++
	}
	if status != nil && *status != "" {
		where = append(where, fmt.Sprintf("i.status = $%d", argIdx))
		args = append(args, *status)
		argIdx++
	}
	whereClause := joinAnd(where)

	var total int64
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM invoices i WHERE %s`, whereClause)
	if err := s.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	offset := (page - 1) * pageSize

	q := fmt.Sprintf(`
		SELECT i.id, i.student_id, i.invoice_no, i.term_id, COALESCE(t.name, ''),
		       i.invoice_date::text, i.due_date::text, i.status,
		       i.total_amount, i.total_paid, i.balance
		FROM invoices i
		LEFT JOIN terms t ON t.id = i.term_id
		WHERE %s
		ORDER BY i.invoice_date DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []InvoiceSummary{}
	for rows.Next() {
		var r InvoiceSummary
		if err := rows.Scan(&r.ID, &r.StudentID, &r.InvoiceNo, &r.TermID, &r.TermName,
			&r.InvoiceDate, &r.DueDate, &r.Status, &r.TotalDue, &r.TotalPaid, &r.Balance); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, nil
}

func (s *FeeStore) GetInvoice(ctx context.Context, id uuid.UUID) (*InvoiceDetail, error) {
	var d InvoiceDetail
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, i.student_id, i.invoice_no, i.term_id, COALESCE(t.name, ''),
		       i.invoice_date::text, i.due_date::text, i.status,
		       i.total_amount, i.total_paid, i.balance
		FROM invoices i
		LEFT JOIN terms t ON t.id = i.term_id
		WHERE i.id = $1`, id).Scan(
		&d.ID, &d.StudentID, &d.InvoiceNo, &d.TermID, &d.TermName,
		&d.InvoiceDate, &d.DueDate, &d.Status, &d.TotalDue, &d.TotalPaid, &d.Balance)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, line_type, description, quantity, unit_price, amount
		FROM invoice_lines WHERE invoice_id = $1 ORDER BY created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d.Lines = []InvoiceLine{}
	for rows.Next() {
		var l InvoiceLine
		if err := rows.Scan(&l.ID, &l.LineType, &l.Description, &l.Quantity, &l.UnitPrice, &l.Amount); err != nil {
			return nil, err
		}
		d.Lines = append(d.Lines, l)
	}
	return &d, nil
}

type PaymentSummary struct {
	ID            uuid.UUID `json:"id"`
	StudentID     uuid.UUID `json:"student_id"`
	ReceiptNo     string    `json:"receipt_no"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	PaymentDate   string    `json:"payment_date"`
	TransactionID string    `json:"transaction_id,omitempty"`
	Status        string    `json:"status"`
	PayerName     string    `json:"payer_name,omitempty"`
}

func (s *FeeStore) ListPayments(ctx context.Context, studentID *uuid.UUID, page, pageSize int) ([]PaymentSummary, int64, error) {
	where := []string{"true"}
	args := []interface{}{}
	argIdx := 1
	if studentID != nil {
		where = append(where, fmt.Sprintf("p.student_id = $%d", argIdx))
		args = append(args, *studentID)
		argIdx++
	}
	whereClause := joinAnd(where)

	var total int64
	if err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM payments p WHERE %s`, whereClause), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	offset := (page - 1) * pageSize

	q := fmt.Sprintf(`
		SELECT p.id, p.student_id, p.receipt_no, p.amount, p.payment_method,
		       p.payment_date::text, COALESCE(p.transaction_id, ''), p.status, p.payer_name
		FROM payments p
		WHERE %s
		ORDER BY p.payment_date DESC, p.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []PaymentSummary{}
	for rows.Next() {
		var r PaymentSummary
		if err := rows.Scan(&r.ID, &r.StudentID, &r.ReceiptNo, &r.Amount, &r.PaymentMethod,
			&r.PaymentDate, &r.TransactionID, &r.Status, &r.PayerName); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, nil
}

type FeeStructureRow struct {
	ID           uuid.UUID `json:"id"`
	Grade        int       `json:"grade"`
	TermNumber   int       `json:"term_number"`
	TotalDay     float64   `json:"total_day"`
	TotalBoarding float64  `json:"total_boarding"`
	IsActive     bool      `json:"is_active"`
}

func (s *FeeStore) ListFeeStructures(ctx context.Context, academicYearID *uuid.UUID, grade *int) ([]FeeStructureRow, error) {
	where := []string{"is_active = TRUE"}
	args := []interface{}{}
	argIdx := 1
	if academicYearID != nil {
		where = append(where, fmt.Sprintf("academic_year_id = $%d", argIdx))
		args = append(args, *academicYearID)
		argIdx++
	}
	if grade != nil {
		where = append(where, fmt.Sprintf("grade = $%d", argIdx))
		args = append(args, *grade)
		argIdx++
	}
	q := fmt.Sprintf(`
		SELECT id, grade, term_number, total_day, total_boarding, is_active
		FROM fee_structures WHERE %s ORDER BY grade, term_number`, joinAnd(where))
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FeeStructureRow{}
	for rows.Next() {
		var r FeeStructureRow
		if err := rows.Scan(&r.ID, &r.Grade, &r.TermNumber, &r.TotalDay, &r.TotalBoarding, &r.IsActive); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

type DiscountRow struct {
	ID             uuid.UUID `json:"id"`
	StudentID      uuid.UUID `json:"student_id"`
	WaiverType     string    `json:"waiver_type"`
	Amount         float64   `json:"amount"`
	Percentage     *float64  `json:"percentage,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	EffectiveDate  string    `json:"effective_date"`
}

func (s *FeeStore) ListDiscounts(ctx context.Context, studentID *uuid.UUID) ([]DiscountRow, error) {
	where := []string{"true"}
	args := []interface{}{}
	if studentID != nil {
		where = append(where, "student_id = $1")
		args = append(args, *studentID)
	}
	q := fmt.Sprintf(`
		SELECT id, student_id, waiver_type, amount, percentage, reason, effective_date::text
		FROM discount_waivers WHERE %s ORDER BY effective_date DESC`, joinAnd(where))
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DiscountRow{}
	for rows.Next() {
		var r DiscountRow
		if err := rows.Scan(&r.ID, &r.StudentID, &r.WaiverType, &r.Amount, &r.Percentage, &r.Reason, &r.EffectiveDate); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}
