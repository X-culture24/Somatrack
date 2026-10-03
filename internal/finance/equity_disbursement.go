package finance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/jenga"
)

// EquityDisbursementStore sends teacher salaries via Equity EFT (Jenga Send Money).
// It is used when the staff profile has a bank_account_no, offering a direct
// bank credit rather than an M-Pesa push.
type EquityDisbursementStore struct {
	pool    *pgxpool.Pool
	payroll *PayrollStore
	client  *jenga.Client
}

func NewEquityDisbursementStore(pool *pgxpool.Pool, payroll *PayrollStore, client *jenga.Client) *EquityDisbursementStore {
	return &EquityDisbursementStore{pool: pool, payroll: payroll, client: client}
}

type EquityEFTTransaction struct {
	ID              uuid.UUID `json:"id"`
	PayrollEntryID  uuid.UUID `json:"payroll_entry_id"`
	Amount          float64   `json:"amount"`
	DestAccount     string    `json:"dest_account_number"`
	DestName        string    `json:"dest_name"`
	Reference       string    `json:"reference"`
	Status          string    `json:"status"`
	TransactionRef  string    `json:"transaction_ref,omitempty"`
	ResultDesc      string    `json:"result_desc,omitempty"`
}

// staffBankDetails holds what we need from staff_profiles + users.
type staffBankDetails struct {
	StaffID        uuid.UUID
	StaffName      string
	BankAccountNo  string
	BankAccountName string
	NetPay         float64
	DisburseStatus string
}

func (s *EquityDisbursementStore) getStaffBankDetails(ctx context.Context, entryID uuid.UUID) (*staffBankDetails, error) {
	var d staffBankDetails
	err := s.pool.QueryRow(ctx, `
		SELECT pe.staff_id, COALESCE(u.first_name || ' ' || u.last_name, ''),
		       sp.bank_account_no, sp.bank_account_name, pe.net_pay, pe.disbursement_status
		FROM payroll_entries pe
		JOIN staff_profiles sp ON sp.id = pe.staff_id
		JOIN users u ON u.id = sp.user_id
		WHERE pe.id = $1`, entryID).Scan(
		&d.StaffID, &d.StaffName, &d.BankAccountNo, &d.BankAccountName, &d.NetPay, &d.DisburseStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEntryNotFound
	}
	return &d, err
}

// InitiateEFT sends the staff member's net pay via Equity EFT. Returns
// ErrAlreadyDisbursed if the entry is already paid or in flight, and
// ErrNoBankAccount if no account number is recorded for the staff member.
func (s *EquityDisbursementStore) InitiateEFT(ctx context.Context, entryID uuid.UUID) (*EquityEFTTransaction, error) {
	d, err := s.getStaffBankDetails(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if d.DisburseStatus == "success" || d.DisburseStatus == "initiated" {
		return nil, ErrAlreadyDisbursed
	}
	if d.BankAccountNo == "" {
		return nil, ErrNoBankAccount
	}

	destName := d.BankAccountName
	if destName == "" {
		destName = d.StaffName
	}
	orderDate := time.Now().Format("2006-01-02")
	reference := fmt.Sprintf("SAL-%s-%s", entryID.String()[:8], orderDate)

	// "63" is Equity Bank's local bank sort code in Kenya.
	res, sendErr := s.client.SendMoney(
		s.client.AccountNumber(),
		"63",
		d.BankAccountNo,
		destName,
		d.NetPay,
		reference,
		"Salary payment — "+d.StaffName,
		orderDate,
	)

	status := "FAILED"
	transRef := ""
	resultDesc := ""
	if sendErr == nil {
		transRef = res.TransactionRef
		resultDesc = res.Status
		if res.Status != "" {
			status = "ACCEPTED"
		}
	} else {
		resultDesc = sendErr.Error()
	}

	var txID uuid.UUID
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO equity_eft_transactions
			(payroll_entry_id, staff_id, dest_account_number, dest_bank_code, dest_name,
			 amount, reference, status, transaction_ref, result_desc)
		VALUES ($1::uuid, $2::uuid, $3, '63', $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		entryID, d.StaffID, d.BankAccountNo, destName,
		d.NetPay, reference, status, transRef, resultDesc,
	).Scan(&txID); err != nil {
		return nil, err
	}

	entryStatus := "failed"
	if status == "ACCEPTED" {
		entryStatus = "initiated"
	}
	if err := s.payroll.MarkDisbursement(ctx, entryID, entryStatus, transRef); err != nil {
		return nil, err
	}

	if sendErr != nil {
		return &EquityEFTTransaction{
			ID: txID, PayrollEntryID: entryID, Amount: d.NetPay,
			DestAccount: d.BankAccountNo, DestName: destName, Reference: reference,
			Status: status, ResultDesc: resultDesc,
		}, sendErr
	}
	return &EquityEFTTransaction{
		ID: txID, PayrollEntryID: entryID, Amount: d.NetPay,
		DestAccount: d.BankAccountNo, DestName: destName, Reference: reference,
		Status: status, TransactionRef: transRef,
	}, nil
}

var ErrNoBankAccount = errors.New("staff member has no bank account number on file — add one on the Staff page to enable Equity EFT")
