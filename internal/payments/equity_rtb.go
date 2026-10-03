package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/jenga"
)

// EquityRTBStore manages Equity "Request to Pay" (EazzyPay C2B) requests.
// When a parent taps "Pay via Equity" in the portal, this sends a payment
// prompt to their Equity Mobile app or linked mobile wallet, which they then
// approve on their phone — the equivalent of M-Pesa STK Push but on Equity's
// network.
type EquityRTBStore struct {
	pool       *pgxpool.Pool
	reconciler *Reconciler
	client     *jenga.Client
}

func NewEquityRTBStore(pool *pgxpool.Pool, reconciler *Reconciler, client *jenga.Client) *EquityRTBStore {
	return &EquityRTBStore{pool: pool, reconciler: reconciler, client: client}
}

type EquityRTBRequest struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	PhoneNumber string    `json:"phone_number"`
	Amount      float64   `json:"amount"`
	WalletName  string    `json:"wallet_name"`
	PartnerRef  string    `json:"partner_ref"`
	RequestRef  string    `json:"request_ref,omitempty"`
	Status      string    `json:"status"`
	ResultDesc  string    `json:"result_desc,omitempty"`
}

// Initiate looks up the student's admission number for the customer reference,
// fires the Jenga Request to Pay, and records the attempt regardless of outcome.
func (s *EquityRTBStore) Initiate(ctx context.Context, studentID, initiatedBy uuid.UUID, phone, walletName string, amount float64) (*EquityRTBRequest, error) {
	var admissionNo, fullName string
	if err := s.pool.QueryRow(ctx, `SELECT admission_no, full_name FROM students WHERE id = $1`, studentID).
		Scan(&admissionNo, &fullName); err != nil {
		return nil, fmt.Errorf("student not found: %w", err)
	}

	if walletName == "" {
		walletName = "Equity"
	}
	id := uuid.New()
	partnerRef := fmt.Sprintf("FEE-%s", id.String()[:8])
	orderDate := time.Now().Format("2006-01-02")
	narrative := fmt.Sprintf("School fees - %s", fullName)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO equity_rtb_requests
			(id, student_id, initiated_by_id, phone_number, wallet_name, amount, partner_ref, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')`,
		id, studentID, initiatedBy, phone, walletName, amount, partnerRef)
	if err != nil {
		return nil, fmt.Errorf("record equity rtb request: %w", err)
	}

	res, sendErr := s.client.RequestToPay(
		admissionNo, partnerRef, phone, walletName,
		amount, narrative, orderDate,
	)

	status := "failed"
	requestRef := ""
	resultDesc := ""
	var rawJSON []byte
	if sendErr == nil {
		requestRef = res.RequestRef
		resultDesc = res.Description
		rawJSON, _ = json.Marshal(res.Raw)
		if res.Status != "" {
			status = "accepted"
		}
	} else {
		resultDesc = sendErr.Error()
	}

	_, updErr := s.pool.Exec(ctx, `
		UPDATE equity_rtb_requests SET
			status = $1, request_ref = $2, result_desc = $3, raw_initiate_response = $4, updated_at = NOW()
		WHERE id = $5`,
		status, requestRef, resultDesc, rawJSON, id)
	if updErr != nil {
		return nil, updErr
	}

	if sendErr != nil {
		return &EquityRTBRequest{
			ID: id, StudentID: studentID, PhoneNumber: phone,
			Amount: amount, WalletName: walletName, PartnerRef: partnerRef,
			Status: status, ResultDesc: resultDesc,
		}, sendErr
	}
	return &EquityRTBRequest{
		ID: id, StudentID: studentID, PhoneNumber: phone,
		Amount: amount, WalletName: walletName, PartnerRef: partnerRef,
		RequestRef: requestRef, Status: status,
	}, nil
}

// HandleCallback processes the Equity IPN/callback for a completed RTB request.
// Equity notifies via the IPN webhook; this correlates by partnerRef and
// creates a payment record if the transaction succeeded.
func (s *EquityRTBStore) HandleCallback(ctx context.Context, partnerRef, transactionRef, status string, amount float64, rawPayload string) error {
	var id, studentID uuid.UUID
	var phone string
	var storedAmount float64
	err := s.pool.QueryRow(ctx, `
		SELECT id, student_id, phone_number, amount FROM equity_rtb_requests WHERE partner_ref = $1`,
		partnerRef).Scan(&id, &studentID, &phone, &storedAmount)
	if err != nil {
		return nil // nothing to correlate — ack anyway
	}

	finalStatus := "failed"
	if status == "SUCCESS" || status == "success" {
		finalStatus = "success"
	}

	var paymentID *uuid.UUID
	if finalStatus == "success" && transactionRef != "" {
		if _, _, err := s.reconciler.EnsureWebhookEvent(ctx, transactionRef, rawPayload); err == nil {
			pid, _, err := s.reconciler.ProcessKnownStudentPayment(ctx, studentID, ProcessPaymentParams{
				TransID:       transactionRef,
				TransAmount:   amount,
				BillRefNumber: "EQ-" + partnerRef,
				MSISDN:        phone,
				RawPayload:    rawPayload,
			})
			if err == nil {
				paymentID = pid
			}
		}
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE equity_rtb_requests SET
			status = $1, request_ref = COALESCE(NULLIF($2,''), request_ref),
			result_desc = $3, payment_id = $4, raw_callback = $5, updated_at = NOW()
		WHERE id = $6`,
		finalStatus, transactionRef, status, paymentID, rawPayload, id)
	return err
}

// GetStatus lets the app poll for the async result.
func (s *EquityRTBStore) GetStatus(ctx context.Context, id uuid.UUID) (*EquityRTBRequest, error) {
	var r EquityRTBRequest
	var requestRef, resultDesc *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, student_id, phone_number, amount, wallet_name, partner_ref, request_ref, status, result_desc
		FROM equity_rtb_requests WHERE id = $1`, id).Scan(
		&r.ID, &r.StudentID, &r.PhoneNumber, &r.Amount, &r.WalletName,
		&r.PartnerRef, &requestRef, &r.Status, &resultDesc)
	if err != nil {
		return nil, err
	}
	if requestRef != nil {
		r.RequestRef = *requestRef
	}
	if resultDesc != nil {
		r.ResultDesc = *resultDesc
	}
	return &r, nil
}
