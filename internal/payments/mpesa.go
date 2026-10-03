package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/fuzzy"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

type MatchResult struct {
	Matched      bool
	StudentID    *uuid.UUID
	Ambiguous    bool
	Score        int
	MatchMode    string
}

type StudentCandidate struct {
	ID          uuid.UUID
	FullName    string
	SearchName  string
	IsActive    bool
}

type Matcher struct {
	pool             *pgxpool.Pool
	accountPrefix    string
	threshold        int
	nairobiLoc       *time.Location
}

func NewMatcher(pool *pgxpool.Pool, accountPrefix string, nairobiLoc *time.Location) *Matcher {
	t := 80
	return &Matcher{pool: pool, accountPrefix: accountPrefix, threshold: t, nairobiLoc: nairobiLoc}
}

var nonAlphaNum = regexp.MustCompile(`[^A-Z0-9]+`)

func (m *Matcher) ParseBillRef(raw string) string {
	ref := strings.TrimSpace(raw)
	if m.accountPrefix != "" && strings.HasPrefix(ref, m.accountPrefix) {
		ref = strings.TrimPrefix(ref, m.accountPrefix)
	}
	ref = strings.ToUpper(ref)
	ref = nonAlphaNum.ReplaceAllString(ref, " ")
	ref = strings.TrimSpace(ref)
	ref = regexp.MustCompile(`\s+`).ReplaceAllString(ref, " ")
	return ref
}

func (m *Matcher) ParseTransTime(raw string) (time.Time, error) {
	layout := "20060102150405"
	t, err := time.ParseInLocation(layout, raw, m.nairobiLoc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func (m *Matcher) MatchStudent(ctx context.Context, cleanedRef string) (*MatchResult, error) {
	if cleanedRef == "" {
		return &MatchResult{Matched: false}, nil
	}
	// Admission number first — it's the reference reminders and receipts tell
	// parents to use, and unlike names it's unique. Separators are ignored on
	// both sides so "ADM-SEED-001", "adm seed 001" and "ADMSEED001" all match.
	var admID uuid.UUID
	err := m.pool.QueryRow(ctx, `
		SELECT id FROM students
		WHERE is_active = TRUE
		  AND regexp_replace(upper(admission_no), '[^A-Z0-9]', '', 'g') = $1`,
		strings.ReplaceAll(cleanedRef, " ", "")).Scan(&admID)
	if err == nil {
		return &MatchResult{Matched: true, StudentID: &admID, Score: 100, MatchMode: "admission_no"}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	rows, err := m.pool.Query(ctx, `
		SELECT id,
		       COALESCE(first_name,'') || ' ' || COALESCE(middle_name,'') || ' ' || COALESCE(last_name,'') AS full_name,
		       COALESCE(first_name,'') || ' ' || COALESCE(last_name,'') AS search_name,
		       is_active
		FROM students
		WHERE is_active = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exactMatches []StudentCandidate
	var top StudentCandidate
	var topScore int
	for rows.Next() {
		var c StudentCandidate
		if err := rows.Scan(&c.ID, &c.FullName, &c.SearchName, &c.IsActive); err != nil {
			return nil, err
		}
		exactFn := strings.EqualFold(strings.TrimSpace(c.FullName), cleanedRef)
		exactSn := strings.EqualFold(strings.TrimSpace(c.SearchName), cleanedRef)
		if exactFn || exactSn {
			exactMatches = append(exactMatches, c)
			continue
		}
		s1 := fuzzy.TokenSetRatioInt(cleanedRef, c.FullName)
		s2 := fuzzy.TokenSetRatioInt(cleanedRef, c.SearchName)
		best := s1
		if s2 > best {
			best = s2
		}
		if best > topScore {
			topScore = best
			top = c
		}
	}
	if len(exactMatches) > 1 {
		return &MatchResult{Matched: false, Ambiguous: true, Score: 100, MatchMode: "ambiguous_exact"}, nil
	}
	if len(exactMatches) == 1 {
		sid := exactMatches[0].ID
		return &MatchResult{Matched: true, StudentID: &sid, Score: 100, MatchMode: "exact"}, nil
	}
	if topScore >= m.threshold && top.ID != uuid.Nil {
		sid := top.ID
		return &MatchResult{Matched: true, StudentID: &sid, Score: topScore, MatchMode: "fuzzy"}, nil
	}
	return &MatchResult{Matched: false, Score: topScore}, nil
}

// PaymentStatus is this package's internal reconciliation-outcome label —
// distinct from payments.status, whose CHECK constraint only allows
// pending/confirmed/bounced/reversed/failed/refunded. PaymentPosted maps to
// the DB's "confirmed"; PaymentNeedsReview never gets written to `payments`
// at all (student_id there is NOT NULL, so an unmatched payment can't be a
// payments row — it stays recorded only on mpesa_webhook_events).
type PaymentStatus string

const (
	PaymentPosted      PaymentStatus = "posted"
	PaymentNeedsReview PaymentStatus = "needs_review"
	PaymentReversed    PaymentStatus = "reversed"
)

// dbPaymentStatus maps PaymentPosted to the payments.status CHECK constraint's
// actual vocabulary.
const dbPaymentStatusConfirmed = "confirmed"

type InvoiceStatus string

// Must match the invoices.status CHECK constraint exactly (no "open").
const (
	InvoiceUnpaid  InvoiceStatus = "unpaid"
	InvoicePartial InvoiceStatus = "partial"
	InvoicePaid    InvoiceStatus = "paid"
)

type Invoice struct {
	ID         uuid.UUID
	StudentID  uuid.UUID
	TermID     uuid.UUID
	TotalDue   float64
	TotalPaid  float64
	Balance    float64
	Status     InvoiceStatus
}

type Reconciler struct {
	pool       *pgxpool.Pool
	nairobiLoc *time.Location
	notifier   *Notifier // nil = no confirmation SMS (notification-svc not configured)
}

// SetNotifier turns on the payment confirmation SMS.
func (r *Reconciler) SetNotifier(n *Notifier) { r.notifier = n }

func NewReconciler(pool *pgxpool.Pool, nairobiLoc *time.Location) *Reconciler {
	return &Reconciler{pool: pool, nairobiLoc: nairobiLoc}
}

func (r *Reconciler) EnsureWebhookEvent(ctx context.Context, transID, rawPayload string) (eventID uuid.UUID, skip bool, err error) {
	var existingID *uuid.UUID
	err = r.pool.QueryRow(ctx, `SELECT id FROM mpesa_webhook_events WHERE trans_id = $1`, transID).Scan(&existingID)
	if err == nil && existingID != nil {
		return *existingID, true, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}
	id := uuid.New()
	_, err = r.pool.Exec(ctx, `
		INSERT INTO mpesa_webhook_events (id, trans_id, raw_payload, status, created_at)
		VALUES ($1, $2, $3, 'pending', NOW())
		ON CONFLICT (trans_id) DO NOTHING`, id, transID, rawPayload)
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, false, nil
}

type ProcessPaymentParams struct {
	TransID       string
	TransAmount   float64
	BillRefNumber string
	TransTime     string
	FirstName     string
	MiddleName    string
	LastName      string
	MSISDN        string
	RawPayload    string
}

func (r *Reconciler) ProcessPayment(ctx context.Context, p ProcessPaymentParams, matcher *Matcher) (paymentID *uuid.UUID, matchedStudentID *uuid.UUID, invoiceID *uuid.UUID, recStatus string, finalErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, nil, "", err
	}
	defer tx.Rollback(ctx)

	var eventID uuid.UUID
	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT id, status FROM mpesa_webhook_events WHERE trans_id = $1 FOR UPDATE`,
		p.TransID).Scan(&eventID, &currentStatus)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("lock event: %w", err)
	}
	if currentStatus == "processed" {
		var pid uuid.UUID
		e := tx.QueryRow(ctx, `SELECT id FROM payments WHERE transaction_id = $1 LIMIT 1`, p.TransID).Scan(&pid)
		if e == nil {
			return &pid, nil, nil, string(PaymentPosted), nil
		}
		return nil, nil, nil, string(PaymentPosted), nil
	}
	var pidCheck uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM payments WHERE transaction_id = $1 LIMIT 1`, p.TransID).Scan(&pidCheck)
	if err == nil {
		_, _ = tx.Exec(ctx, `UPDATE mpesa_webhook_events SET status = 'processed' WHERE id = $1`, eventID)
		return &pidCheck, nil, nil, string(PaymentPosted), tx.Commit(ctx)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil, "", err
	}

	cleaned := matcher.ParseBillRef(p.BillRefNumber)
	match, err := matcher.MatchStudent(ctx, cleaned)
	if err != nil {
		return nil, nil, nil, "", err
	}

	transTime, _ := matcher.ParseTransTime(p.TransTime)
	if transTime.IsZero() {
		transTime = time.Now().In(matcher.nairobiLoc)
	}
	matchedName := p.FirstName + " " + p.LastName

	// Unmatched payments can't become a `payments` row at all —
	// payments.student_id is NOT NULL. Record the failure on the webhook
	// event itself (status/error columns exist for exactly this) instead
	// of trying to force an orphaned row into a table that requires a
	// valid student.
	if !match.Matched || match.Ambiguous {
		reason := fmt.Sprintf("no confident match for bill ref %q (amount %.2f, msisdn %s)", cleaned, p.TransAmount, p.MSISDN)
		if match.Ambiguous {
			reason = fmt.Sprintf("ambiguous match for bill ref %q — multiple students matched", cleaned)
		}
		_, _ = tx.Exec(ctx, `UPDATE mpesa_webhook_events SET status = 'failed', error = $1, processed_at = NOW() WHERE id = $2`,
			reason, eventID)
		if err := tx.Commit(ctx); err != nil {
			return nil, nil, nil, "", err
		}
		return nil, nil, nil, string(PaymentNeedsReview), nil
	}

	paymentIDOut := uuid.New()
	studentID := match.StudentID
	receiptNo := "MPESA-" + p.TransID
	_, err = tx.Exec(ctx, `
		INSERT INTO payments (id, student_id, receipt_no, amount, payment_method, payment_date,
			transaction_id, payer_name, payer_phone, status, allocated_amount, unallocated_amount,
			reference, created_at)
		VALUES ($1, $2, $3, $4, 'mpesa', $5, $6, $7, $8, $9, $10, $11, $12, NOW())`,
		paymentIDOut, studentID, receiptNo, p.TransAmount, transTime,
		p.TransID, matchedName, p.MSISDN, dbPaymentStatusConfirmed, p.TransAmount, 0, cleaned)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("insert payment: %w", err)
	}

	invID, err := r.attachInvoiceForStudent(ctx, tx, *studentID, transTime)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("attach invoice: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO invoice_payment_links (id, invoice_id, payment_id, amount_applied, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (invoice_id, payment_id) DO NOTHING`,
		uuid.New(), *invID, paymentIDOut, p.TransAmount)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("link payment to invoice: %w", err)
	}

	_, _ = tx.Exec(ctx, `UPDATE mpesa_webhook_events SET status = 'processed', processed_at = NOW() WHERE id = $1`, eventID)

	if err := r.recomputeInvoiceTotalsInTx(ctx, tx, *invID); err != nil {
		return nil, nil, nil, "", fmt.Errorf("recompute invoice: %w", err)
	}

	reconStatus := PaymentPosted
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, nil, "", err
	}

	if reconStatus == PaymentPosted && studentID != nil && invID != nil {
		go r.enqueueNotifications(paymentIDOut, *studentID, *invID, p.TransAmount)
	}

	return &paymentIDOut, studentID, invID, string(reconStatus), nil
}

// ProcessKnownStudentPayment is ProcessPayment's counterpart for a payment
// where the student is already certain — an STK Push the parent triggered
// themselves for a specific child in their account, not a C2B/Paybill
// payment that has to be fuzzy-matched from a free-text bill reference. Uses
// the same idempotency-by-trans-id, invoice-attach, and totals-recompute
// logic, just without the Matcher step (there's nothing to disambiguate).
func (r *Reconciler) ProcessKnownStudentPayment(ctx context.Context, studentID uuid.UUID, p ProcessPaymentParams) (paymentID *uuid.UUID, invoiceID *uuid.UUID, finalErr error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var eventID uuid.UUID
	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT id, status FROM mpesa_webhook_events WHERE trans_id = $1 FOR UPDATE`,
		p.TransID).Scan(&eventID, &currentStatus)
	if err != nil {
		return nil, nil, fmt.Errorf("lock event: %w", err)
	}
	var pidCheck uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM payments WHERE transaction_id = $1 LIMIT 1`, p.TransID).Scan(&pidCheck)
	if err == nil {
		_, _ = tx.Exec(ctx, `UPDATE mpesa_webhook_events SET status = 'processed' WHERE id = $1`, eventID)
		return &pidCheck, nil, tx.Commit(ctx)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}

	transTime, _ := r.parseTransTimeOrNow(p.TransTime)
	matchedName := strings.TrimSpace(p.FirstName + " " + p.LastName)
	receiptNo := "MPESA-" + p.TransID

	paymentIDOut := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO payments (id, student_id, receipt_no, amount, payment_method, payment_date,
			transaction_id, payer_name, payer_phone, status, allocated_amount, unallocated_amount,
			reference, created_at)
		VALUES ($1, $2, $3, $4, 'mpesa', $5, $6, $7, $8, $9, $10, $11, $12, NOW())`,
		paymentIDOut, studentID, receiptNo, p.TransAmount, transTime,
		p.TransID, matchedName, p.MSISDN, dbPaymentStatusConfirmed, p.TransAmount, 0, p.BillRefNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("insert payment: %w", err)
	}

	invID, err := r.attachInvoiceForStudent(ctx, tx, studentID, transTime)
	if err != nil {
		return nil, nil, fmt.Errorf("attach invoice: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO invoice_payment_links (id, invoice_id, payment_id, amount_applied, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (invoice_id, payment_id) DO NOTHING`,
		uuid.New(), *invID, paymentIDOut, p.TransAmount)
	if err != nil {
		return nil, nil, fmt.Errorf("link payment to invoice: %w", err)
	}

	_, _ = tx.Exec(ctx, `UPDATE mpesa_webhook_events SET status = 'processed', processed_at = NOW() WHERE id = $1`, eventID)

	if err := r.recomputeInvoiceTotalsInTx(ctx, tx, *invID); err != nil {
		return nil, nil, fmt.Errorf("recompute invoice: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	go r.enqueueNotifications(paymentIDOut, studentID, *invID, p.TransAmount)
	return &paymentIDOut, invID, nil
}

func (r *Reconciler) parseTransTimeOrNow(raw string) (time.Time, error) {
	layout := "20060102150405"
	t, err := time.ParseInLocation(layout, raw, r.nairobiLoc)
	if err != nil || t.IsZero() {
		return time.Now().In(r.nairobiLoc), err
	}
	return t, nil
}

func (r *Reconciler) attachInvoiceForStudent(ctx context.Context, tx pgx.Tx, studentID uuid.UUID, transTime time.Time) (*uuid.UUID, error) {
	var termID uuid.UUID
	var termEndDate time.Time
	err := tx.QueryRow(ctx, `
		SELECT t.id, t.end_date FROM terms t
		JOIN academic_years ay ON ay.id = t.academic_year_id
		WHERE ay.is_active = TRUE AND t.is_current = TRUE
		LIMIT 1`).Scan(&termID, &termEndDate)
	if err != nil {
		return nil, fmt.Errorf("find current term: %w", err)
	}
	var admissionNo string
	if err := tx.QueryRow(ctx, `SELECT admission_no FROM students WHERE id = $1`, studentID).Scan(&admissionNo); err != nil {
		return nil, fmt.Errorf("find student: %w", err)
	}
	var invID uuid.UUID
	// admissionNo (students.admission_no UNIQUE) + a term fragment avoids the
	// collision a truncated studentID prefix can hit on sequential/patterned
	// UUIDs (e.g. seed data ids that share the same first 8 hex characters).
	invoiceNo := fmt.Sprintf("INV-%s-%s", admissionNo, strings.ToUpper(termID.String()[:8]))
	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (student_id, term_id, invoice_no, invoice_date, due_date,
			total_amount, total_paid, balance, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, 0, 0, 'unpaid', NOW())
		ON CONFLICT (student_id, term_id) DO UPDATE SET updated_at = NOW()
		RETURNING id`, studentID, termID, invoiceNo, transTime, termEndDate).Scan(&invID)
	if err != nil {
		return nil, err
	}
	return &invID, nil
}

func (r *Reconciler) recomputeInvoiceTotalsInTx(ctx context.Context, tx pgx.Tx, invoiceID uuid.UUID) error {
	var totalDue, totalPaid float64
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM invoice_lines WHERE invoice_id = $1`, invoiceID).Scan(&totalDue)
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM payments
		WHERE status = 'confirmed'
		  AND EXISTS (
			SELECT 1 FROM invoice_payment_links ipl
			WHERE ipl.payment_id = payments.id AND ipl.invoice_id = $1
		  )`, invoiceID).Scan(&totalPaid)
	if err != nil {
		return err
	}
	balance := totalDue - totalPaid
	var status InvoiceStatus
	switch {
	case balance <= 0 && totalDue > 0:
		status = InvoicePaid
	case totalPaid > 0:
		status = InvoicePartial
	default:
		status = InvoiceUnpaid
	}
	_, err = tx.Exec(ctx, `
		UPDATE invoices
		SET total_amount = $1, total_paid = $2, balance = $3, status = $4, updated_at = NOW()
		WHERE id = $5`, totalDue, totalPaid, balance, status, invoiceID)
	return err
}

func (r *Reconciler) RecomputeInvoiceTotals(ctx context.Context, invoiceID uuid.UUID) (*types.PaginatedResponse[string], float64, float64, float64, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, 0, 0, 0, "", err
	}
	defer tx.Rollback(ctx)
	var totalDue, totalPaid float64
	err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM invoice_lines WHERE invoice_id = $1`, invoiceID).Scan(&totalDue)
	if err != nil {
		return nil, 0, 0, 0, "", err
	}
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(p.amount),0)
		FROM payments p
		JOIN invoice_payment_links ipl ON ipl.payment_id = p.id
		WHERE ipl.invoice_id = $1 AND p.status = 'confirmed'`, invoiceID).Scan(&totalPaid)
	if err != nil {
		return nil, 0, 0, 0, "", err
	}
	balance := totalDue - totalPaid
	var status InvoiceStatus
	switch {
	case balance <= 0 && totalDue > 0:
		status = InvoicePaid
	case totalPaid > 0:
		status = InvoicePartial
	default:
		status = InvoiceUnpaid
	}
	_, err = tx.Exec(ctx, `UPDATE invoices SET total_amount=$1,total_paid=$2,balance=$3,status=$4,updated_at=NOW() WHERE id=$5`,
		totalDue, totalPaid, balance, status, invoiceID)
	if err != nil {
		return nil, 0, 0, 0, "", err
	}
	return nil, totalDue, totalPaid, balance, string(status), tx.Commit(ctx)
}

// enqueueNotifications runs (in its own goroutine) after a payment commits,
// so a slow or failing SMS gateway never holds up posting the money.
func (r *Reconciler) enqueueNotifications(paymentID, studentID, invoiceID uuid.UUID, amount float64) {
	if r.notifier == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()
	if err := r.notifier.PaymentReceived(ctx, paymentID); err != nil {
		log.Printf("[payments] confirmation SMS for payment %s (student %s, invoice %s, KES %.2f): %v",
			paymentID, studentID, invoiceID, amount, err)
	}
}

type AsynqProcessor struct {
	reconciler *Reconciler
	matcher    *Matcher
	client     *asynq.Client
}

func NewAsynqProcessor(r *Reconciler, m *Matcher, redisAddr string) (*AsynqProcessor, func(), error) {
	opt, err := asynq.ParseRedisURI(redisAddr)
	if err != nil {
		return nil, nil, err
	}
	client := asynq.NewClient(opt)
	return &AsynqProcessor{reconciler: r, matcher: m, client: client}, func() { client.Close() }, nil
}

const TaskProcessMpesa = "mpesa:process"

func (p *AsynqProcessor) EnqueueProcess(ctx context.Context, payload ProcessPaymentParams) (*asynq.TaskInfo, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	task := asynq.NewTask(TaskProcessMpesa, b, asynq.MaxRetry(5), asynq.Queue("payments"))
	return p.client.EnqueueContext(ctx, task)
}

func (p *AsynqProcessor) HandleProcess(ctx context.Context, t *asynq.Task) error {
	var payload ProcessPaymentParams
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
	_, _, _, _, err := p.reconciler.ProcessPayment(ctx, payload, p.matcher)
	return err
}
