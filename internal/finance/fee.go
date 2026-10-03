package finance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InvoiceStatus mirrors invoices.status's CHECK constraint values relevant to
// billing generation (overdue/cancelled/refunded are set elsewhere, not by
// this recompute).
type InvoiceStatus string

const (
	InvoiceUnpaid  InvoiceStatus = "unpaid"
	InvoicePartial InvoiceStatus = "partial"
	InvoicePaid    InvoiceStatus = "paid"
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
	ID             uuid.UUID `json:"id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	Grade          int       `json:"grade"`
	TermNumber     int       `json:"term_number"`
	AdmissionFee   float64   `json:"admission_fee"`
	TuitionFee     float64   `json:"tuition_fee"`
	BoardingFee    float64   `json:"boarding_fee"`
	TransportFee   float64   `json:"transport_fee"`
	MedicalFee     float64   `json:"medical_fee"`
	ActivityFee    float64   `json:"activity_fee"`
	ExamFee        float64   `json:"exam_fee"`
	LibraryFee     float64   `json:"library_fee"`
	ComputerFee    float64   `json:"computer_fee"`
	UniformFee     float64   `json:"uniform_fee"`
	DiaryBookFee   float64   `json:"diary_book_fee"`
	OtherFee       float64   `json:"other_fee"`
	Notes          string    `json:"notes"`
	TotalDay       float64   `json:"total_day"`
	TotalBoarding  float64   `json:"total_boarding"`
	IsActive       bool      `json:"is_active"`
}

const feeStructureCols = `id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee,
	transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee,
	uniform_fee, diary_book_fee, other_fee, notes, total_day, total_boarding, is_active`

func scanFeeStructure(row interface{ Scan(dest ...interface{}) error }) (FeeStructureRow, error) {
	var r FeeStructureRow
	err := row.Scan(&r.ID, &r.AcademicYearID, &r.Grade, &r.TermNumber, &r.AdmissionFee, &r.TuitionFee, &r.BoardingFee,
		&r.TransportFee, &r.MedicalFee, &r.ActivityFee, &r.ExamFee, &r.LibraryFee, &r.ComputerFee,
		&r.UniformFee, &r.DiaryBookFee, &r.OtherFee, &r.Notes, &r.TotalDay, &r.TotalBoarding, &r.IsActive)
	return r, err
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
	q := fmt.Sprintf(`SELECT %s FROM fee_structures WHERE %s ORDER BY grade, term_number`, feeStructureCols, joinAnd(where))
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FeeStructureRow{}
	for rows.Next() {
		r, err := scanFeeStructure(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *FeeStore) GetFeeStructure(ctx context.Context, id uuid.UUID) (*FeeStructureRow, error) {
	q := fmt.Sprintf(`SELECT %s FROM fee_structures WHERE id = $1`, feeStructureCols)
	r, err := scanFeeStructure(s.pool.QueryRow(ctx, q, id))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// FeeStructureInput is the writable subset of FeeStructureRow — total_day/
// total_boarding are always server-computed, never trusted from the client.
type FeeStructureInput struct {
	AcademicYearID uuid.UUID `json:"academic_year_id" validate:"required,uuid"`
	Grade          int       `json:"grade" validate:"min=-2,max=13"`
	TermNumber     int       `json:"term_number" validate:"required,min=1,max=3"`
	AdmissionFee   float64   `json:"admission_fee"`
	TuitionFee     float64   `json:"tuition_fee"`
	BoardingFee    float64   `json:"boarding_fee"`
	TransportFee   float64   `json:"transport_fee"`
	MedicalFee     float64   `json:"medical_fee"`
	ActivityFee    float64   `json:"activity_fee"`
	ExamFee        float64   `json:"exam_fee"`
	LibraryFee     float64   `json:"library_fee"`
	ComputerFee    float64   `json:"computer_fee"`
	UniformFee     float64   `json:"uniform_fee"`
	DiaryBookFee   float64   `json:"diary_book_fee"`
	OtherFee       float64   `json:"other_fee"`
	Notes          string    `json:"notes"`
	IsActive       *bool     `json:"is_active,omitempty"`
}

// totalDay matches the source fee schedule's own "Total per term" figure,
// which for Term 1 includes the one-off admission fee and diary/progress
// book charge alongside the recurring components.
func (in FeeStructureInput) totalDay() float64 {
	return in.AdmissionFee + in.TuitionFee + in.TransportFee + in.MedicalFee + in.ActivityFee +
		in.ExamFee + in.LibraryFee + in.ComputerFee + in.UniformFee + in.DiaryBookFee + in.OtherFee
}

func (s *FeeStore) CreateFeeStructure(ctx context.Context, in FeeStructureInput) (*FeeStructureRow, error) {
	totalDay := in.totalDay()
	totalBoarding := totalDay + in.BoardingFee
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}
	q := fmt.Sprintf(`
		INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee,
			transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee,
			uniform_fee, diary_book_fee, other_fee, notes, total_day, total_boarding, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), NOW())
		RETURNING %s`, feeStructureCols)
	r, err := scanFeeStructure(s.pool.QueryRow(ctx, q, uuid.New(), in.AcademicYearID, in.Grade, in.TermNumber,
		in.AdmissionFee, in.TuitionFee, in.BoardingFee, in.TransportFee, in.MedicalFee, in.ActivityFee, in.ExamFee,
		in.LibraryFee, in.ComputerFee, in.UniformFee, in.DiaryBookFee, in.OtherFee, in.Notes, totalDay, totalBoarding, isActive))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *FeeStore) UpdateFeeStructure(ctx context.Context, id uuid.UUID, in FeeStructureInput) (*FeeStructureRow, error) {
	totalDay := in.totalDay()
	totalBoarding := totalDay + in.BoardingFee
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}
	q := fmt.Sprintf(`
		UPDATE fee_structures SET academic_year_id=$1, grade=$2, term_number=$3, admission_fee=$4, tuition_fee=$5,
			boarding_fee=$6, transport_fee=$7, medical_fee=$8, activity_fee=$9, exam_fee=$10,
			library_fee=$11, computer_fee=$12, uniform_fee=$13, diary_book_fee=$14, other_fee=$15, notes=$16,
			total_day=$17, total_boarding=$18, is_active=$19, updated_at=NOW()
		WHERE id=$20
		RETURNING %s`, feeStructureCols)
	r, err := scanFeeStructure(s.pool.QueryRow(ctx, q, in.AcademicYearID, in.Grade, in.TermNumber, in.AdmissionFee,
		in.TuitionFee, in.BoardingFee, in.TransportFee, in.MedicalFee, in.ActivityFee, in.ExamFee,
		in.LibraryFee, in.ComputerFee, in.UniformFee, in.DiaryBookFee, in.OtherFee, in.Notes, totalDay, totalBoarding, isActive, id))
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *FeeStore) DeleteFeeStructure(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM fee_structures WHERE id = $1`, id)
	return err
}

type DiscountRow struct {
	ID            uuid.UUID `json:"id"`
	StudentID     uuid.UUID `json:"student_id"`
	TermID        uuid.UUID `json:"term_id"`
	WaiverType    string    `json:"waiver_type"`
	Amount        float64   `json:"amount"`
	Percentage    *float64  `json:"percentage,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	EffectiveDate string    `json:"effective_date"`
}

// DiscountInput is the writable subset of DiscountRow for creation.
type DiscountInput struct {
	StudentID     uuid.UUID `json:"student_id" validate:"required,uuid"`
	TermID        uuid.UUID `json:"term_id" validate:"required,uuid"`
	WaiverType    string    `json:"waiver_type" validate:"required,oneof=scholarship bursary discount staff_child orphan sibling_discount other"`
	Amount        float64   `json:"amount"`
	Percentage    *float64  `json:"percentage,omitempty"`
	Reason        string    `json:"reason"`
	EffectiveDate string    `json:"effective_date" validate:"required"`
	ExpiryDate    string    `json:"expiry_date,omitempty"`
	ApprovedByID  *uuid.UUID `json:"approved_by_id,omitempty"`
}

func (s *FeeStore) CreateDiscount(ctx context.Context, in DiscountInput) (*DiscountRow, error) {
	id := uuid.New()
	var expiry *string
	if in.ExpiryDate != "" {
		expiry = &in.ExpiryDate
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO discount_waivers (id, student_id, term_id, waiver_type, amount, percentage,
			reason, approved_by_id, effective_date, expiry_date, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
		id, in.StudentID, in.TermID, in.WaiverType, in.Amount, in.Percentage,
		in.Reason, in.ApprovedByID, in.EffectiveDate, expiry)
	if err != nil {
		return nil, err
	}
	return &DiscountRow{
		ID: id, StudentID: in.StudentID, TermID: in.TermID, WaiverType: in.WaiverType,
		Amount: in.Amount, Percentage: in.Percentage, Reason: in.Reason, EffectiveDate: in.EffectiveDate,
	}, nil
}

func (s *FeeStore) ListDiscounts(ctx context.Context, studentID *uuid.UUID) ([]DiscountRow, error) {
	where := []string{"true"}
	args := []interface{}{}
	if studentID != nil {
		where = append(where, "student_id = $1")
		args = append(args, *studentID)
	}
	q := fmt.Sprintf(`
		SELECT id, student_id, term_id, waiver_type, amount, percentage, reason, effective_date::text
		FROM discount_waivers WHERE %s ORDER BY effective_date DESC`, joinAnd(where))
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DiscountRow{}
	for rows.Next() {
		var r DiscountRow
		if err := rows.Scan(&r.ID, &r.StudentID, &r.TermID, &r.WaiverType, &r.Amount, &r.Percentage, &r.Reason, &r.EffectiveDate); err != nil {
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

type BillingSkip struct {
	StudentID uuid.UUID `json:"student_id"`
	Reason    string    `json:"reason"`
}

type BillingRunResult struct {
	TermID    uuid.UUID     `json:"term_id"`
	Generated int           `json:"generated"`
	Skipped   []BillingSkip `json:"skipped"`
}

// autoLineTypes are the invoice_lines.line_type values this generator owns —
// on regeneration only these are cleared and replaced. Manually-added lines
// (ad_hoc, uniform_item) are left untouched.
var autoLineTypes = []string{"tuition", "boarding", "transport", "medical", "activity", "exam", "library", "computer", "uniform", "other"}

// GenerateInvoicesForTerm creates/refreshes one invoice per active student for
// the given term, populated from the fee_structure matching (academic_year,
// grade, term_number) — boarding_fee only applies to boarding students,
// transport_fee only to students with uses_transport=true. A student whose
// grade has no matching fee_structure, or who has no current class assigned,
// is skipped and reported rather than silently billed 0.
// BillStudentForCurrentTerm generates/refreshes one student's invoice for
// whichever term is currently active, the moment they're enrolled — so a
// real fee balance exists immediately instead of waiting for someone to run
// a bulk "Generate Billing" pass. Mirrors GenerateInvoicesForTerm's per-
// student matching logic (fee structure by grade/term, mandatory ad-hoc
// fees, new-intake admission fee, route-based transport fare) for exactly
// one student instead of every active student.
func (s *FeeStore) BillStudentForCurrentTerm(ctx context.Context, studentID uuid.UUID) (*uuid.UUID, error) {
	var termID, academicYearID uuid.UUID
	var termNumber int
	var termEndDate, academicYearStart time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT t.id, t.term_number, t.academic_year_id, t.end_date, ay.start_date
		FROM terms t JOIN academic_years ay ON ay.id = t.academic_year_id
		WHERE ay.is_active = TRUE AND t.is_current = TRUE LIMIT 1`).
		Scan(&termID, &termNumber, &academicYearID, &termEndDate, &academicYearStart)
	if err != nil {
		return nil, fmt.Errorf("no current term configured: %w", err)
	}

	var admissionNo string
	var grade int
	var streamID *uuid.UUID
	var classID uuid.UUID
	var boardingStatus string
	var usesTransport bool
	var transportFare float64
	var admissionDate *time.Time
	err = s.pool.QueryRow(ctx, `
		SELECT s.admission_no, cg.grade, s.stream_id, s.current_class_id, s.boarding_status,
		       s.uses_transport, COALESCE(tr.fare, 0), s.admission_date
		FROM students s
		JOIN class_groups cg ON cg.id = s.current_class_id
		LEFT JOIN transport_routes tr ON tr.id = s.transport_route_id
		WHERE s.id = $1`, studentID).
		Scan(&admissionNo, &grade, &streamID, &classID, &boardingStatus, &usesTransport, &transportFare, &admissionDate)
	if err != nil {
		return nil, fmt.Errorf("student has no class assigned: %w", err)
	}

	fs, err := scanFeeStructure(s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM fee_structures
		WHERE academic_year_id = $1 AND grade = $2 AND term_number = $3 AND is_active = TRUE`, feeStructureCols),
		academicYearID, grade, termNumber))
	if err != nil {
		return nil, fmt.Errorf("no fee structure for grade %d term %d: %w", grade, termNumber, err)
	}

	mandatoryAll, err := s.ListAdhocFees(ctx, &termID)
	if err != nil {
		return nil, fmt.Errorf("load adhoc fees: %w", err)
	}
	isBoarding := boardingStatus == "boarding"
	var applicable []AdhocFee
	for _, f := range mandatoryAll {
		if !f.IsMandatory {
			continue
		}
		if f.Grade != nil && *f.Grade != grade {
			continue
		}
		if f.StreamID != nil && (streamID == nil || *f.StreamID != *streamID) {
			continue
		}
		if f.ClassID != nil && *f.ClassID != classID {
			continue
		}
		if f.AppliesToBoardingOnly && !isBoarding {
			continue
		}
		applicable = append(applicable, f)
	}
	isNewIntake := termNumber == 1 && admissionDate != nil && !admissionDate.Before(academicYearStart)

	if err := s.billStudent(ctx, studentID, admissionNo, termID, termEndDate, fs, isBoarding, usesTransport, transportFare, applicable, isNewIntake); err != nil {
		return nil, err
	}
	var invoiceID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT id FROM invoices WHERE student_id = $1 AND term_id = $2`, studentID, termID).
		Scan(&invoiceID); err != nil {
		return nil, err
	}
	return &invoiceID, nil
}

func (s *FeeStore) GenerateInvoicesForTerm(ctx context.Context, termID uuid.UUID) (*BillingRunResult, error) {
	var academicYearID uuid.UUID
	var termNumber int
	var termEndDate time.Time
	err := s.pool.QueryRow(ctx, `SELECT academic_year_id, term_number, end_date FROM terms WHERE id = $1`, termID).
		Scan(&academicYearID, &termNumber, &termEndDate)
	if err != nil {
		return nil, fmt.Errorf("term not found: %w", err)
	}
	var academicYearStart time.Time
	if err := s.pool.QueryRow(ctx, `SELECT start_date FROM academic_years WHERE id = $1`, academicYearID).
		Scan(&academicYearStart); err != nil {
		return nil, fmt.Errorf("academic year not found: %w", err)
	}

	type studentRow struct {
		id             uuid.UUID
		admissionNo    string
		grade          int
		streamID       *uuid.UUID
		classID        uuid.UUID
		boardingStatus string
		usesTransport  bool
		transportFare  float64
		admissionDate  *time.Time
	}
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.admission_no, cg.grade, s.stream_id, s.current_class_id, s.boarding_status,
		       s.uses_transport, COALESCE(tr.fare, 0), s.admission_date
		FROM students s
		JOIN class_groups cg ON cg.id = s.current_class_id
		LEFT JOIN transport_routes tr ON tr.id = s.transport_route_id
		WHERE s.is_active = TRUE`)
	if err != nil {
		return nil, err
	}
	var students []studentRow
	for rows.Next() {
		var sr studentRow
		if err := rows.Scan(&sr.id, &sr.admissionNo, &sr.grade, &sr.streamID, &sr.classID, &sr.boardingStatus,
			&sr.usesTransport, &sr.transportFare, &sr.admissionDate); err != nil {
			rows.Close()
			return nil, err
		}
		students = append(students, sr)
	}
	rows.Close()

	mandatoryAdhoc, err := s.ListAdhocFees(ctx, &termID)
	if err != nil {
		return nil, fmt.Errorf("load adhoc fees: %w", err)
	}
	var mandatory []AdhocFee
	for _, f := range mandatoryAdhoc {
		if f.IsMandatory {
			mandatory = append(mandatory, f)
		}
	}

	// Students with no current_class_id never matched the JOIN above — surface
	// them as skips too instead of silently omitting them from the run.
	unassigned, err := s.pool.Query(ctx, `SELECT id FROM students WHERE is_active = TRUE AND current_class_id IS NULL`)
	if err != nil {
		return nil, err
	}
	result := &BillingRunResult{TermID: termID}
	for unassigned.Next() {
		var id uuid.UUID
		if err := unassigned.Scan(&id); err == nil {
			result.Skipped = append(result.Skipped, BillingSkip{StudentID: id, Reason: "no current class assigned"})
		}
	}
	unassigned.Close()

	feeCache := map[int]*FeeStructureRow{}
	for _, sr := range students {
		fs, ok := feeCache[sr.grade]
		if !ok {
			row, err := scanFeeStructure(s.pool.QueryRow(ctx, fmt.Sprintf(`
				SELECT %s FROM fee_structures
				WHERE academic_year_id = $1 AND grade = $2 AND term_number = $3 AND is_active = TRUE`, feeStructureCols),
				academicYearID, sr.grade, termNumber))
			if err != nil {
				feeCache[sr.grade] = nil
			} else {
				feeCache[sr.grade] = &row
			}
			fs = feeCache[sr.grade]
		}
		if fs == nil {
			result.Skipped = append(result.Skipped, BillingSkip{
				StudentID: sr.id,
				Reason:    fmt.Sprintf("no fee structure for grade %d, term %d", sr.grade, termNumber),
			})
			continue
		}
		isBoarding := sr.boardingStatus == "boarding"
		var applicable []AdhocFee
		for _, f := range mandatory {
			if f.Grade != nil && *f.Grade != sr.grade {
				continue
			}
			if f.StreamID != nil && (sr.streamID == nil || *f.StreamID != *sr.streamID) {
				continue
			}
			if f.ClassID != nil && *f.ClassID != sr.classID {
				continue
			}
			if f.AppliesToBoardingOnly && !isBoarding {
				continue
			}
			applicable = append(applicable, f)
		}
		isNewIntake := termNumber == 1 && sr.admissionDate != nil && !sr.admissionDate.Before(academicYearStart)
		if err := s.billStudent(ctx, sr.id, sr.admissionNo, termID, termEndDate, *fs, isBoarding, sr.usesTransport, sr.transportFare, applicable, isNewIntake); err != nil {
			result.Skipped = append(result.Skipped, BillingSkip{StudentID: sr.id, Reason: err.Error()})
			continue
		}
		result.Generated++
	}
	return result, nil
}

func (s *FeeStore) billStudent(ctx context.Context, studentID uuid.UUID, admissionNo string, termID uuid.UUID, termEndDate time.Time,
	fs FeeStructureRow, isBoarding, usesTransport bool, transportFare float64, mandatoryAdhoc []AdhocFee, isNewIntake bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// admissionNo is unique per student (students.admission_no UNIQUE), and
	// combined with a term fragment this can't collide the way truncating a
	// UUID prefix can — sequential seed/demo UUIDs sharing the same first 8
	// hex characters (e.g. "aaaaaaaa-...-...aaaa" vs "...aaab") hit exactly
	// that collision when this used studentID.String()[:8].
	invoiceNo := fmt.Sprintf("INV-%s-%s", admissionNo, strings.ToUpper(termID.String()[:8]))
	var invoiceID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (student_id, term_id, invoice_no, invoice_date, due_date,
			total_amount, total_paid, balance, status, created_at)
		VALUES ($1, $2, $3, CURRENT_DATE, $4, 0, 0, 0, 'unpaid', NOW())
		ON CONFLICT (student_id, term_id) DO UPDATE SET updated_at = NOW()
		RETURNING id`, studentID, termID, invoiceNo, termEndDate).Scan(&invoiceID)
	if err != nil {
		return fmt.Errorf("upsert invoice: %w", err)
	}

	placeholders := make([]string, len(autoLineTypes))
	args := make([]interface{}, len(autoLineTypes)+1)
	args[0] = invoiceID
	for i, lt := range autoLineTypes {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = lt
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		`DELETE FROM invoice_lines WHERE invoice_id = $1 AND line_type IN (%s)`,
		strings.Join(placeholders, ",")), args...); err != nil {
		return fmt.Errorf("clear old lines: %w", err)
	}

	type line struct {
		lineType string
		desc     string
		amount   float64
	}
	lines := []line{
		{"tuition", "Tuition fee", fs.TuitionFee},
		{"medical", "Medical fee", fs.MedicalFee},
		{"activity", "Activity fee", fs.ActivityFee},
		{"exam", "Exam fee", fs.ExamFee},
		{"library", "Library fee", fs.LibraryFee},
		{"computer", "Computer fee", fs.ComputerFee},
		{"uniform", "Uniform fee", fs.UniformFee},
		{"other", "Other fee", fs.OtherFee},
	}
	// Both are one-off Term-1 charges from the real fee schedule: admission
	// fee only applies to a student newly admitted this academic year,
	// while the diary & progress book is billed to everyone each year.
	if fs.TermNumber == 1 {
		lines = append(lines, line{"other", "Diary & progress book", fs.DiaryBookFee})
		if isNewIntake {
			lines = append(lines, line{"other", "Admission fee", fs.AdmissionFee})
		}
	}
	if isBoarding {
		lines = append(lines, line{"boarding", "Boarding fee", fs.BoardingFee})
	}
	if usesTransport {
		lines = append(lines, line{"transport", "Transport fee", transportFare})
	}
	for _, l := range lines {
		if l.amount <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_lines (id, invoice_id, line_type, description, quantity, unit_price, amount, created_at)
			VALUES ($1, $2, $3, $4, 1, $5, $5, NOW())`,
			uuid.New(), invoiceID, l.lineType, l.desc, l.amount); err != nil {
			return fmt.Errorf("insert line %s: %w", l.lineType, err)
		}
	}

	// Mandatory ad-hoc fees matching this student (grade/stream/class/boarding)
	// are swept in per-fee (delete-then-insert keyed by adhoc_fee_id) so this
	// stays idempotent on rerun without touching manually-added elective
	// charges from AddInvoiceCharge, which live in the same table/line_type
	// but reference a different (or no) adhoc_fee_id.
	for _, f := range mandatoryAdhoc {
		if _, err := tx.Exec(ctx, `DELETE FROM invoice_lines WHERE invoice_id = $1 AND adhoc_fee_id = $2`,
			invoiceID, f.ID); err != nil {
			return fmt.Errorf("clear old adhoc line %s: %w", f.Name, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO invoice_lines (id, invoice_id, line_type, description, quantity, unit_price, amount, adhoc_fee_id, created_at)
			VALUES ($1, $2, 'ad_hoc', $3, 1, $4, $4, $5, NOW())`,
			uuid.New(), invoiceID, f.Name, f.Amount, f.ID); err != nil {
			return fmt.Errorf("insert adhoc line %s: %w", f.Name, err)
		}
	}

	if err := recomputeInvoiceTotals(ctx, tx, invoiceID); err != nil {
		return fmt.Errorf("recompute totals: %w", err)
	}
	return tx.Commit(ctx)
}

// recomputeInvoiceTotals recalculates total_amount (from invoice_lines),
// total_paid (from confirmed payments linked via invoice_payment_links) and
// total_waivers (from active discount_waivers for this student+term), then
// derives balance and status. Mirrors internal/payments' recomputeInvoiceTotalsInTx,
// duplicated here because finance-svc and payment-svc are separate services —
// keep both in sync if the formula changes.
func recomputeInvoiceTotals(ctx context.Context, tx pgx.Tx, invoiceID uuid.UUID) error {
	var studentID, termID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT student_id, term_id FROM invoices WHERE id = $1`, invoiceID).
		Scan(&studentID, &termID); err != nil {
		return err
	}

	var totalAmount float64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount), 0) FROM invoice_lines WHERE invoice_id = $1`, invoiceID).
		Scan(&totalAmount); err != nil {
		return err
	}

	var totalPaid float64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(p.amount), 0)
		FROM payments p
		JOIN invoice_payment_links ipl ON ipl.payment_id = p.id
		WHERE ipl.invoice_id = $1 AND p.status = 'confirmed'`, invoiceID).Scan(&totalPaid); err != nil {
		return err
	}

	var totalWaivers float64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount + (COALESCE(percentage, 0) / 100.0) * $3), 0)
		FROM discount_waivers
		WHERE student_id = $1 AND term_id = $2
		  AND effective_date <= CURRENT_DATE
		  AND (expiry_date IS NULL OR expiry_date >= CURRENT_DATE)`,
		studentID, termID, totalAmount).Scan(&totalWaivers); err != nil {
		return err
	}

	balance := totalAmount - totalPaid - totalWaivers
	var status InvoiceStatus
	switch {
	case balance <= 0 && totalAmount > 0:
		status = InvoicePaid
	case totalPaid > 0 || totalWaivers > 0:
		status = InvoicePartial
	default:
		status = InvoiceUnpaid
	}
	_, err := tx.Exec(ctx, `
		UPDATE invoices SET total_amount = $1, total_paid = $2, total_waivers = $3, balance = $4,
			status = $5, updated_at = NOW()
		WHERE id = $6`, totalAmount, totalPaid, totalWaivers, balance, status, invoiceID)
	return err
}

// AdhocFee is a one-off charge for something outside the recurring fee
// structure — a school trip, an excursion, a replacement item — scoped to a
// term and optionally narrowed to a grade/stream/class and/or boarding-only
// students. Mandatory ones (is_mandatory=true) are swept into every matching
// student's invoice automatically by GenerateInvoicesForTerm; optional ones
// (e.g. an elective trip) are meant to be added to individual invoices one
// at a time via AddInvoiceCharge instead, since not every student opts in.
type AdhocFee struct {
	ID                     uuid.UUID  `json:"id"`
	Name                   string     `json:"name"`
	Description            string     `json:"description"`
	Amount                 float64    `json:"amount"`
	Grade                  *int       `json:"grade,omitempty"`
	StreamID               *uuid.UUID `json:"stream_id,omitempty"`
	ClassID                *uuid.UUID `json:"class_id,omitempty"`
	TermID                 uuid.UUID  `json:"term_id"`
	IsMandatory            bool       `json:"is_mandatory"`
	AppliesToBoardingOnly  bool       `json:"applies_to_boarding_only"`
	Deadline               *string    `json:"deadline,omitempty"`
}

const adhocFeeCols = `id, name, description, amount, grade, stream_id, class_id, term_id,
	is_mandatory, applies_to_boarding_only, deadline::text`

func scanAdhocFee(row interface{ Scan(dest ...interface{}) error }) (AdhocFee, error) {
	var f AdhocFee
	err := row.Scan(&f.ID, &f.Name, &f.Description, &f.Amount, &f.Grade, &f.StreamID, &f.ClassID,
		&f.TermID, &f.IsMandatory, &f.AppliesToBoardingOnly, &f.Deadline)
	return f, err
}

type AdhocFeeInput struct {
	Name                  string     `json:"name" validate:"required"`
	Description           string     `json:"description"`
	Amount                float64    `json:"amount" validate:"required,gt=0"`
	Grade                 *int       `json:"grade,omitempty"`
	StreamID              *uuid.UUID `json:"stream_id,omitempty"`
	ClassID               *uuid.UUID `json:"class_id,omitempty"`
	TermID                uuid.UUID  `json:"term_id" validate:"required,uuid"`
	IsMandatory           bool       `json:"is_mandatory"`
	AppliesToBoardingOnly bool       `json:"applies_to_boarding_only"`
	Deadline              string     `json:"deadline,omitempty"`
}

func (s *FeeStore) ListAdhocFees(ctx context.Context, termID *uuid.UUID) ([]AdhocFee, error) {
	where := "true"
	args := []interface{}{}
	if termID != nil {
		where = "term_id = $1"
		args = append(args, *termID)
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`SELECT %s FROM adhoc_fees WHERE %s ORDER BY created_at DESC`, adhocFeeCols, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdhocFee{}
	for rows.Next() {
		f, err := scanAdhocFee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (s *FeeStore) CreateAdhocFee(ctx context.Context, in AdhocFeeInput) (*AdhocFee, error) {
	var deadline *string
	if in.Deadline != "" {
		deadline = &in.Deadline
	}
	q := fmt.Sprintf(`
		INSERT INTO adhoc_fees (id, name, description, amount, grade, stream_id, class_id, term_id,
			is_mandatory, applies_to_boarding_only, deadline, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING %s`, adhocFeeCols)
	f, err := scanAdhocFee(s.pool.QueryRow(ctx, q, uuid.New(), in.Name, in.Description, in.Amount,
		in.Grade, in.StreamID, in.ClassID, in.TermID, in.IsMandatory, in.AppliesToBoardingOnly, deadline))
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *FeeStore) DeleteAdhocFee(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM adhoc_fees WHERE id = $1`, id)
	return err
}

// AddInvoiceCharge adds one manual charge to an existing invoice — either a
// specific optional adhoc_fee (adhocFeeID set) or a freeform one-off amount
// (description/amount set directly) — then recomputes the invoice's totals.
// This is the "elective" counterpart to GenerateInvoicesForTerm's automatic
// mandatory adhoc-fee sweep: for charges not everyone should be billed for.
func (s *FeeStore) AddInvoiceCharge(ctx context.Context, invoiceID uuid.UUID, adhocFeeID *uuid.UUID, description string, amount float64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if adhocFeeID != nil {
		var fee AdhocFee
		row, err := scanAdhocFee(tx.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM adhoc_fees WHERE id = $1`, adhocFeeCols), *adhocFeeID))
		if err != nil {
			return fmt.Errorf("adhoc fee not found: %w", err)
		}
		fee = row
		description, amount = fee.Name, fee.Amount
	}
	if description == "" || amount <= 0 {
		return fmt.Errorf("description and a positive amount are required")
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_lines (id, invoice_id, line_type, description, quantity, unit_price, amount, adhoc_fee_id, created_at)
		VALUES ($1, $2, 'ad_hoc', $3, 1, $4, $4, $5, NOW())`,
		uuid.New(), invoiceID, description, amount, adhocFeeID); err != nil {
		return fmt.Errorf("insert charge: %w", err)
	}
	if err := recomputeInvoiceTotals(ctx, tx, invoiceID); err != nil {
		return fmt.Errorf("recompute totals: %w", err)
	}
	return tx.Commit(ctx)
}
