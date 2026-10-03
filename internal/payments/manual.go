package payments

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// ManualPayment is money received at the bursar's desk — cash, cheque, bank
// slip or card — rather than through a webhook. It gets the same treatment
// as M-Pesa: applied to the current invoice, receipted, and confirmed to the
// family by SMS.
type ManualPayment struct {
	StudentID   uuid.UUID  `json:"student_id"`
	Amount      float64    `json:"amount"`
	Method      string     `json:"payment_method"`
	PaymentDate string     `json:"payment_date"` // YYYY-MM-DD, default today
	Reference   string     `json:"reference"`    // bank slip / transaction no.
	BankName    string     `json:"bank_name"`
	ChequeNo    string     `json:"cheque_no"`
	PayerName   string     `json:"payer_name"`
	PayerPhone  string     `json:"payer_phone"`
	Notes       string     `json:"notes"`
	ReceivedBy  *uuid.UUID `json:"received_by_id"`
	// IdempotencyKey is generated once per form, so a double-click or a
	// retried request records the payment once.
	IdempotencyKey string `json:"idempotency_key"`
}

var manualMethods = map[string]bool{"cash": true, "cheque": true, "bank_transfer": true, "card": true, "other": true}

func (m *ManualPayment) validate() map[string]string {
	e := map[string]string{}
	if m.StudentID == uuid.Nil {
		e["student_id"] = "is required"
	}
	if m.Amount <= 0 || m.Amount > 10_000_000 {
		e["amount"] = "must be more than 0"
	}
	if !manualMethods[m.Method] {
		e["payment_method"] = "must be cash, cheque, bank_transfer, card or other"
	}
	if m.Method == "cheque" && strings.TrimSpace(m.ChequeNo) == "" {
		e["cheque_no"] = "is required for a cheque"
	}
	if m.Method == "bank_transfer" && strings.TrimSpace(m.Reference) == "" {
		e["reference"] = "the bank slip or transaction number is required"
	}
	if m.PaymentDate != "" {
		d, err := time.Parse("2006-01-02", m.PaymentDate)
		if err != nil {
			e["payment_date"] = "must be a date (YYYY-MM-DD)"
		} else if d.After(time.Now().Add(24 * time.Hour)) {
			e["payment_date"] = "can't be in the future"
		}
	}
	return e
}

type ManualPaymentResult struct {
	PaymentID uuid.UUID `json:"payment_id"`
	ReceiptNo string    `json:"receipt_no"`
	InvoiceID uuid.UUID `json:"invoice_id"`
	Duplicate bool      `json:"duplicate"`
}

// ErrValidation carries per-field messages back to the form.
type ErrValidation map[string]string

func (e ErrValidation) Error() string { return "validation failed" }

// RecordManualPayment posts a counter payment and queues its confirmation.
func (r *Reconciler) RecordManualPayment(ctx context.Context, m ManualPayment) (*ManualPaymentResult, error) {
	if e := m.validate(); len(e) > 0 {
		return nil, ErrValidation(e)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var txnKey *string
	if k := strings.TrimSpace(m.IdempotencyKey); k != "" {
		key := "MANUAL-" + k
		txnKey = &key
		var existing ManualPaymentResult
		err := tx.QueryRow(ctx, `
			SELECT p.id, p.receipt_no, COALESCE(l.invoice_id, '00000000-0000-0000-0000-000000000000')
			FROM payments p LEFT JOIN invoice_payment_links l ON l.payment_id = p.id
			WHERE p.transaction_id = $1 LIMIT 1`, key).Scan(&existing.PaymentID, &existing.ReceiptNo, &existing.InvoiceID)
		if err == nil {
			existing.Duplicate = true
			return &existing, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	// Former students can still clear arrears, so is_active isn't checked.
	var one int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM students WHERE id = $1`, m.StudentID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrValidation{"student_id": "student not found"}
		}
		return nil, err
	}

	paidOn := time.Now().In(r.nairobiLoc)
	if m.PaymentDate != "" {
		paidOn, _ = time.ParseInLocation("2006-01-02", m.PaymentDate, r.nairobiLoc)
	}
	receiptNo, err := newReceiptNo(paidOn)
	if err != nil {
		return nil, err
	}
	paymentID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO payments (id, student_id, receipt_no, amount, payment_method, payment_date, transaction_id,
			bank_name, cheque_no, payer_name, payer_phone, status, allocated_amount, unallocated_amount,
			reference, received_by_id, notes, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$4,0,$13,$14,$15, now())`,
		paymentID, m.StudentID, receiptNo, m.Amount, m.Method, paidOn, txnKey,
		strings.TrimSpace(m.BankName), strings.TrimSpace(m.ChequeNo), strings.TrimSpace(m.PayerName),
		strings.TrimSpace(m.PayerPhone), dbPaymentStatusConfirmed, strings.TrimSpace(m.Reference), m.ReceivedBy,
		strings.TrimSpace(m.Notes))
	if err != nil {
		return nil, fmt.Errorf("insert payment: %w", err)
	}
	invID, err := r.attachInvoiceForStudent(ctx, tx, m.StudentID, paidOn)
	if err != nil {
		return nil, fmt.Errorf("attach invoice: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO invoice_payment_links (id, invoice_id, payment_id, amount_applied, created_at)
		VALUES ($1, $2, $3, $4, now())`, uuid.New(), *invID, paymentID, m.Amount); err != nil {
		return nil, fmt.Errorf("link payment to invoice: %w", err)
	}
	if err := r.recomputeInvoiceTotalsInTx(ctx, tx, *invID); err != nil {
		return nil, fmt.Errorf("recompute invoice: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	go r.enqueueNotifications(paymentID, m.StudentID, *invID, m.Amount)
	return &ManualPaymentResult{PaymentID: paymentID, ReceiptNo: receiptNo, InvoiceID: *invID}, nil
}

// newReceiptNo: RCT-20260925-7K3QX9 — dated for the bursar, random so two
// desks can't collide (receipt_no is UNIQUE).
func newReceiptNo(on time.Time) (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)[:6]
	return "RCT-" + on.Format("20060102") + "-" + code, nil
}

func (s *Service) handleManualPayment(w http.ResponseWriter, r *http.Request) {
	var m ManualPayment
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&m); err != nil {
		httpx.ErrBadRequest(w, "request body must be a JSON object")
		return
	}
	res, err := s.reconciler.RecordManualPayment(r.Context(), m)
	var ve ErrValidation
	if errors.As(err, &ve) {
		httpx.JSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "validation", "message": "Please correct the highlighted fields.", "fields": map[string]string(ve),
		})
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	status := http.StatusCreated
	if res.Duplicate {
		status = http.StatusOK
	}
	httpx.JSON(w, status, res)
}
