package finance

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PayrollStore struct {
	pool *pgxpool.Pool
}

func NewPayrollStore(pool *pgxpool.Pool) *PayrollStore {
	return &PayrollStore{pool: pool}
}

var ErrPayrollRunExists = errors.New("a payroll run already exists for that month/year")

type PayrollRun struct {
	ID          uuid.UUID `json:"id"`
	Month       int       `json:"month"`
	Year        int       `json:"year"`
	RunDate     string    `json:"run_date"`
	Status      string    `json:"status"`
	StaffCount  int       `json:"staff_count"`
	TotalNetPay float64   `json:"total_net_pay"`
}

type PayrollEntry struct {
	ID                 uuid.UUID `json:"id"`
	PayrollRunID       uuid.UUID `json:"payroll_run_id"`
	StaffID            uuid.UUID `json:"staff_id"`
	StaffName          string    `json:"staff_name"`
	EmployeeNo         string    `json:"employee_no"`
	BasicSalary        float64   `json:"basic_salary"`
	Allowances         float64   `json:"allowances"`
	Deductions         float64   `json:"deductions"`
	GrossPay           float64   `json:"gross_pay"`
	NetPay             float64   `json:"net_pay"`
	PAYE               float64   `json:"paye"`
	NHIF               float64   `json:"nhif"`
	NSSF               float64   `json:"nssf"`
	DisbursementStatus string    `json:"disbursement_status"`
	DisbursementRef    string    `json:"disbursement_ref,omitempty"`
}

// GeneratePayrollRun computes basic gross/net pay for every active staff member:
// gross = basic_salary (no allowance schedule modeled yet); net = gross - the sum
// of their currently-active staff_deductions rows. There is no statutory PAYE/NHIF/NSSF
// tax-band calculation here — the paye/nhif/nssf columns only reflect amounts an admin
// has recorded in staff_deductions, not a computed statutory liability. Wire a real
// KRA tax-band calculator in here before relying on this for actual payslips.
func (s *PayrollStore) GeneratePayrollRun(ctx context.Context, month, year int, processedBy uuid.UUID) (*PayrollRun, []PayrollEntry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	runDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

	var runID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO payroll_runs (month, year, run_date, status, processed_by_id)
		VALUES ($1, $2, $3, 'draft', $4)
		RETURNING id`, month, year, runDate, processedBy).Scan(&runID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, nil, ErrPayrollRunExists
		}
		return nil, nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT sp.id, sp.basic_salary
		FROM staff_profiles sp
		WHERE sp.is_active = TRUE`)
	if err != nil {
		return nil, nil, err
	}
	type staffRow struct {
		ID     uuid.UUID
		Basic  float64
	}
	var staff []staffRow
	for rows.Next() {
		var r staffRow
		if err := rows.Scan(&r.ID, &r.Basic); err != nil {
			rows.Close()
			return nil, nil, err
		}
		staff = append(staff, r)
	}
	rows.Close()

	entries := []PayrollEntry{}
	for _, st := range staff {
		var paye, nhif, nssf, totalDeductions float64
		err := tx.QueryRow(ctx, `
			SELECT
				COALESCE(SUM(amount) FILTER (WHERE deduction_type = 'paye'), 0),
				COALESCE(SUM(amount) FILTER (WHERE deduction_type = 'nhif'), 0),
				COALESCE(SUM(amount) FILTER (WHERE deduction_type = 'nssf'), 0),
				COALESCE(SUM(amount), 0)
			FROM staff_deductions
			WHERE staff_id = $1 AND is_recurring = TRUE
			  AND (start_date IS NULL OR start_date <= $2)
			  AND (end_date IS NULL OR end_date >= $2)`,
			st.ID, runDate).Scan(&paye, &nhif, &nssf, &totalDeductions)
		if err != nil {
			return nil, nil, err
		}

		gross := st.Basic
		net := gross - totalDeductions

		var entryID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO payroll_entries
				(payroll_run_id, staff_id, basic_salary, allowances, deductions, gross_pay, net_pay, paye, nhif, nssf)
			VALUES ($1, $2, $3, 0, $4, $5, $6, $7, $8, $9)
			RETURNING id`,
			runID, st.ID, st.Basic, totalDeductions, gross, net, paye, nhif, nssf).Scan(&entryID)
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, PayrollEntry{
			ID: entryID, PayrollRunID: runID, StaffID: st.ID,
			BasicSalary: st.Basic, Deductions: totalDeductions,
			GrossPay: gross, NetPay: net, PAYE: paye, NHIF: nhif, NSSF: nssf,
			DisbursementStatus: "pending",
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	run := &PayrollRun{
		ID: runID, Month: month, Year: year,
		RunDate: runDate.Format("2006-01-02"), Status: "draft",
		StaffCount: len(entries),
	}
	for _, e := range entries {
		run.TotalNetPay += e.NetPay
	}
	return run, entries, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *PayrollStore) ListPayrollRuns(ctx context.Context) ([]PayrollRun, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT pr.id, pr.month, pr.year, pr.run_date::text, pr.status,
		       COUNT(pe.id), COALESCE(SUM(pe.net_pay), 0)
		FROM payroll_runs pr
		LEFT JOIN payroll_entries pe ON pe.payroll_run_id = pr.id
		GROUP BY pr.id
		ORDER BY pr.year DESC, pr.month DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PayrollRun{}
	for rows.Next() {
		var r PayrollRun
		if err := rows.Scan(&r.ID, &r.Month, &r.Year, &r.RunDate, &r.Status, &r.StaffCount, &r.TotalNetPay); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *PayrollStore) ListPayrollEntries(ctx context.Context, runID uuid.UUID) ([]PayrollEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT pe.id, pe.payroll_run_id, pe.staff_id,
		       COALESCE(u.first_name || ' ' || u.last_name, ''), sp.employee_no,
		       pe.basic_salary, pe.allowances, pe.deductions, pe.gross_pay, pe.net_pay,
		       pe.paye, pe.nhif, pe.nssf, pe.disbursement_status, pe.disbursement_ref
		FROM payroll_entries pe
		JOIN staff_profiles sp ON sp.id = pe.staff_id
		JOIN users u ON u.id = sp.user_id
		WHERE pe.payroll_run_id = $1
		ORDER BY u.first_name, u.last_name`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PayrollEntry{}
	for rows.Next() {
		var r PayrollEntry
		if err := rows.Scan(&r.ID, &r.PayrollRunID, &r.StaffID, &r.StaffName, &r.EmployeeNo,
			&r.BasicSalary, &r.Allowances, &r.Deductions, &r.GrossPay, &r.NetPay,
			&r.PAYE, &r.NHIF, &r.NSSF, &r.DisbursementStatus, &r.DisbursementRef); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

type PayrollEntryForDisbursement struct {
	ID          uuid.UUID
	StaffID     uuid.UUID
	NetPay      float64
	Phone       string
	StaffName   string
	AlreadyPaid bool
}

func (s *PayrollStore) GetEntryForDisbursement(ctx context.Context, entryID uuid.UUID) (*PayrollEntryForDisbursement, error) {
	var e PayrollEntryForDisbursement
	var status string
	err := s.pool.QueryRow(ctx, `
		SELECT pe.id, pe.staff_id, pe.net_pay, u.phone,
		       COALESCE(u.first_name || ' ' || u.last_name, ''), pe.disbursement_status
		FROM payroll_entries pe
		JOIN staff_profiles sp ON sp.id = pe.staff_id
		JOIN users u ON u.id = sp.user_id
		WHERE pe.id = $1`, entryID).Scan(&e.ID, &e.StaffID, &e.NetPay, &e.Phone, &e.StaffName, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntryNotFound
		}
		return nil, err
	}
	e.AlreadyPaid = status == "success" || status == "initiated"
	return &e, nil
}

var ErrEntryNotFound = errors.New("payroll entry not found")

func (s *PayrollStore) MarkDisbursement(ctx context.Context, entryID uuid.UUID, status, ref string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE payroll_entries
		SET disbursement_status = $1, disbursement_ref = $2,
		    disbursed_at = CASE WHEN $1 = 'success' THEN NOW() ELSE disbursed_at END
		WHERE id = $3`, status, ref, entryID)
	return err
}
