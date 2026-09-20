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

type PaymentStatus string

const (
	PaymentPosted        PaymentStatus = "posted"
	PaymentNeedsReview   PaymentStatus = "needs_review"
	PaymentReversed      PaymentStatus = "reversed"
)

type InvoiceStatus string

const (
	InvoiceOpen    InvoiceStatus = "open"
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
}

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
		e := tx.QueryRow(ctx, `SELECT id FROM payments WHERE trans_id = $1 LIMIT 1`, p.TransID).Scan(&pid)
		if e == nil {
			return &pid, nil, nil, string(PaymentPosted), nil
		}
		return nil, nil, nil, string(PaymentPosted), nil
	}
	var pidCheck uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM payments WHERE trans_id = $1 LIMIT 1`, p.TransID).Scan(&pidCheck)
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

	paymentIDOut := uuid.New()
	var reconStatus PaymentStatus
	var studentID *uuid.UUID
	var invID *uuid.UUID
	matchedName := p.FirstName + " " + p.LastName

	if match.Matched && !match.Ambiguous {
		reconStatus = PaymentPosted
		studentID = match.StudentID
		_, err = tx.Exec(ctx, `
			INSERT INTO payments (id, student_id, amount, payment_method, reference, trans_id,
				msisdn, payer_name, reconciliation_status, match_score, match_mode,
				transaction_date, created_at)
			VALUES ($1, $2, $3, 'mpesa', $4, $5, $6, $7, $8, $9, $10, $11, NOW())`,
			paymentIDOut, studentID, p.TransAmount, cleaned, p.TransID,
			p.MSISDN, matchedName, string(reconStatus), match.Score, match.MatchMode, transTime)
		if err != nil {
			return nil, nil, nil, "", fmt.Errorf("insert payment: %w", err)
		}
		iid, err := r.attachInvoiceForStudent(ctx, tx, *studentID, transTime)
		if err == nil {
			invID = iid
		}
	} else {
		reconStatus = PaymentNeedsReview
		_, err = tx.Exec(ctx, `
			INSERT INTO payments (id, amount, payment_method, reference, trans_id,
				msisdn, payer_name, reconciliation_status, transaction_date, created_at)
			VALUES ($1, $2, 'mpesa', $3, $4, $5, $6, $7, $8, NOW())`,
			paymentIDOut, p.TransAmount, cleaned, p.TransID,
			p.MSISDN, matchedName, string(reconStatus), transTime)
		if err != nil {
			return nil, nil, nil, "", fmt.Errorf("insert nr payment: %w", err)
		}
	}

	_, _ = tx.Exec(ctx, `UPDATE mpesa_webhook_events SET status = 'processed', processed_at = NOW() WHERE id = $1`, eventID)

	if invID != nil {
		if err := r.recomputeInvoiceTotalsInTx(ctx, tx, *invID); err != nil {
			return nil, nil, nil, "", fmt.Errorf("recompute invoice: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, nil, "", err
	}

	if reconStatus == PaymentPosted && studentID != nil && invID != nil {
		go r.enqueueNotifications(paymentIDOut, *studentID, *invID, p.TransAmount)
	}

	return &paymentIDOut, studentID, invID, string(reconStatus), nil
}

func (r *Reconciler) attachInvoiceForStudent(ctx context.Context, tx pgx.Tx, studentID uuid.UUID, transTime time.Time) (*uuid.UUID, error) {
	var termID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT t.id FROM terms t
		JOIN academic_years ay ON ay.id = t.academic_year_id
		WHERE ay.is_current = TRUE AND t.is_current = TRUE
		LIMIT 1`).Scan(&termID)
	if err != nil {
		return nil, err
	}
	var invID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (student_id, term_id, total_due, total_paid, balance, status, created_at)
		VALUES ($1, $2, 0, 0, 0, 'open', NOW())
		ON CONFLICT (student_id, term_id) DO UPDATE SET updated_at = NOW()
		RETURNING id`, studentID, termID).Scan(&invID)
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
		WHERE reconciliation_status = 'posted'
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
		status = InvoiceOpen
	}
	_, err = tx.Exec(ctx, `
		UPDATE invoices
		SET total_due = $1, total_paid = $2, balance = $3, status = $4, updated_at = NOW()
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
		WHERE ipl.invoice_id = $1 AND p.reconciliation_status = 'posted'`, invoiceID).Scan(&totalPaid)
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
		status = InvoiceOpen
	}
	_, err = tx.Exec(ctx, `UPDATE invoices SET total_due=$1,total_paid=$2,balance=$3,status=$4,updated_at=NOW() WHERE id=$5`,
		totalDue, totalPaid, balance, status, invoiceID)
	if err != nil {
		return nil, 0, 0, 0, "", err
	}
	return nil, totalDue, totalPaid, balance, string(status), tx.Commit(ctx)
}

func (r *Reconciler) enqueueNotifications(paymentID, studentID, invoiceID uuid.UUID, amount float64) {
	_ = paymentID
	_ = invoiceID
	_ = amount
	log.Printf("[payments] TODO: enqueue notifications for student %s", studentID)
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
