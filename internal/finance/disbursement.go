package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type B2CTransaction struct {
	ID                       uuid.UUID  `json:"id"`
	PayrollEntryID           *uuid.UUID `json:"payroll_entry_id,omitempty"`
	StaffID                  *uuid.UUID `json:"staff_id,omitempty"`
	PhoneNumber              string     `json:"phone_number"`
	Amount                   float64    `json:"amount"`
	Status                   string     `json:"status"`
	OriginatorConversationID string     `json:"originator_conversation_id"`
	ConversationID           string     `json:"conversation_id,omitempty"`
	ResultDesc               string     `json:"result_desc,omitempty"`
	TransactionID            string     `json:"transaction_id,omitempty"`
}

var ErrAlreadyDisbursed = errors.New("this payroll entry has already been paid or has a payout in flight")

type DisbursementStore struct {
	pool     *pgxpool.Pool
	payroll  *PayrollStore
	client   *DarajaB2CClient
}

func NewDisbursementStore(pool *pgxpool.Pool, payroll *PayrollStore, client *DarajaB2CClient) *DisbursementStore {
	return &DisbursementStore{pool: pool, payroll: payroll, client: client}
}

// InitiateSalaryPayment triggers a Daraja B2C SalaryPayment for a payroll entry's
// net pay. Calling this without real DARAJA_* credentials configured will fail
// with a clear DarajaAPIError rather than silently doing nothing.
func (d *DisbursementStore) InitiateSalaryPayment(ctx context.Context, entryID uuid.UUID) (*B2CTransaction, error) {
	entry, err := d.payroll.GetEntryForDisbursement(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if entry.AlreadyPaid {
		return nil, ErrAlreadyDisbursed
	}
	if entry.Phone == "" {
		return nil, errors.New("staff member has no phone number on file")
	}

	originatorID := uuid.New().String()
	remarks := fmt.Sprintf("Salary payment - %s", entry.StaffName)

	res, sendErr := d.client.SendPayment(entry.Phone, entry.NetPay, remarks, "Payroll", "SalaryPayment", originatorID)

	status := "REJECTED"
	respCode, respDesc := "", ""
	var raw map[string]interface{}
	if sendErr == nil {
		respCode = res.ResponseCode
		respDesc = res.ResponseDescription
		raw = res.Raw
		if respCode == "0" {
			status = "ACCEPTED"
		}
	} else {
		respDesc = sendErr.Error()
	}

	rawJSON, _ := json.Marshal(raw)
	var txID uuid.UUID
	convID := ""
	if res != nil {
		convID = res.ConversationID
	}
	err = d.pool.QueryRow(ctx, `
		INSERT INTO b2c_transactions
			(payroll_entry_id, staff_id, phone_number, amount, remarks, occasion, command_id,
			 status, originator_conversation_id, conversation_id, response_code, response_description,
			 raw_initiate_response)
		VALUES ($1, $2, $3, $4, $5, $6, 'SalaryPayment', $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		entryID, entry.StaffID, entry.Phone, entry.NetPay, remarks, "Payroll",
		status, originatorID, convID, respCode, respDesc, rawJSON,
	).Scan(&txID)
	if err != nil {
		return nil, err
	}

	entryStatus := "failed"
	if status == "ACCEPTED" {
		entryStatus = "initiated"
	}
	if markErr := d.payroll.MarkDisbursement(ctx, entryID, entryStatus, originatorID); markErr != nil {
		return nil, markErr
	}

	if sendErr != nil {
		return &B2CTransaction{ID: txID, PayrollEntryID: &entryID, StaffID: &entry.StaffID,
			PhoneNumber: entry.Phone, Amount: entry.NetPay, Status: status,
			OriginatorConversationID: originatorID, ResultDesc: respDesc}, sendErr
	}
	return &B2CTransaction{ID: txID, PayrollEntryID: &entryID, StaffID: &entry.StaffID,
		PhoneNumber: entry.Phone, Amount: entry.NetPay, Status: status,
		OriginatorConversationID: originatorID, ConversationID: convID}, nil
}

func (d *DisbursementStore) GetTransaction(ctx context.Context, id uuid.UUID) (*B2CTransaction, error) {
	var t B2CTransaction
	err := d.pool.QueryRow(ctx, `
		SELECT id, payroll_entry_id, staff_id, phone_number, amount, status,
		       originator_conversation_id, conversation_id, result_desc, transaction_id
		FROM b2c_transactions WHERE id = $1`, id).Scan(
		&t.ID, &t.PayrollEntryID, &t.StaffID, &t.PhoneNumber, &t.Amount, &t.Status,
		&t.OriginatorConversationID, &t.ConversationID, &t.ResultDesc, &t.TransactionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("b2c transaction not found")
		}
		return nil, err
	}
	return &t, nil
}

// HandleResultCallback processes Safaricom's async ResultURL webhook payload.
func (d *DisbursementStore) HandleResultCallback(ctx context.Context, payload map[string]interface{}) error {
	result, _ := payload["Result"].(map[string]interface{})
	originatorID, _ := result["OriginatorConversationID"].(string)
	conversationID, _ := result["ConversationID"].(string)
	resultCodeRaw, hasCode := result["ResultCode"]
	resultDesc, _ := result["ResultDesc"].(string)

	var entryID *uuid.UUID
	var id uuid.UUID
	err := d.pool.QueryRow(ctx, `
		SELECT id, payroll_entry_id FROM b2c_transactions
		WHERE originator_conversation_id = $1 OR (conversation_id = $2 AND conversation_id <> '')
		LIMIT 1`, originatorID, conversationID).Scan(&id, &entryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // ack anyway — nothing to correlate against
	}
	if err != nil {
		return err
	}

	resultCode := ""
	if hasCode {
		resultCode = fmt.Sprintf("%v", resultCodeRaw)
	}
	status := "FAILED"
	if resultCode == "0" {
		status = "SUCCESS"
	}

	var transactionID, completedAt, receiverName string
	var amount *float64
	if params, ok := result["ResultParameters"].(map[string]interface{}); ok {
		if list, ok := params["ResultParameter"].([]interface{}); ok {
			for _, item := range list {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				key, _ := m["Key"].(string)
				switch key {
				case "TransactionAmount":
					if v, ok := m["Value"].(float64); ok {
						amount = &v
					}
				case "TransactionCompletedDateTime":
					completedAt, _ = m["Value"].(string)
				case "ReceiverPartyPublicName":
					receiverName, _ = m["Value"].(string)
				}
			}
		}
	}
	transactionID, _ = result["TransactionID"].(string)
	_ = completedAt

	rawJSON, _ := json.Marshal(payload)
	_, err = d.pool.Exec(ctx, `
		UPDATE b2c_transactions SET
			status = $1, result_code = $2, result_desc = $3, conversation_id = COALESCE(NULLIF($4,''), conversation_id),
			transaction_id = COALESCE(NULLIF($5,''), transaction_id),
			transaction_amount = COALESCE($6, transaction_amount),
			receiver_public_name = COALESCE(NULLIF($7,''), receiver_public_name),
			raw_callback = $8, updated_at = NOW()
		WHERE id = $9`,
		status, resultCode, resultDesc, conversationID, transactionID, amount, receiverName, rawJSON, id)
	if err != nil {
		return err
	}

	if entryID != nil {
		entryStatus := "failed"
		if status == "SUCCESS" {
			entryStatus = "success"
		}
		if err := d.payroll.MarkDisbursement(ctx, *entryID, entryStatus, transactionID); err != nil {
			return err
		}
	}
	return nil
}

func (d *DisbursementStore) HandleTimeoutCallback(ctx context.Context, payload map[string]interface{}) error {
	result, ok := payload["Result"].(map[string]interface{})
	if !ok {
		result = payload
	}
	originatorID, _ := result["OriginatorConversationID"].(string)
	conversationID, _ := result["ConversationID"].(string)
	resultDesc, _ := result["ResultDesc"].(string)
	if resultDesc == "" {
		resultDesc = "Request timed out"
	}

	var entryID *uuid.UUID
	var id uuid.UUID
	err := d.pool.QueryRow(ctx, `
		SELECT id, payroll_entry_id FROM b2c_transactions
		WHERE originator_conversation_id = $1 OR (conversation_id = $2 AND conversation_id <> '')
		LIMIT 1`, originatorID, conversationID).Scan(&id, &entryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	rawJSON, _ := json.Marshal(payload)
	if _, err := d.pool.Exec(ctx, `
		UPDATE b2c_transactions SET status = 'TIMEOUT', result_desc = $1, raw_callback = $2, updated_at = NOW()
		WHERE id = $3`, resultDesc, rawJSON, id); err != nil {
		return err
	}
	if entryID != nil {
		return d.payroll.MarkDisbursement(ctx, *entryID, "failed", "")
	}
	return nil
}
