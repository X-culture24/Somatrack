DROP TABLE IF EXISTS b2c_transactions;

ALTER TABLE payroll_entries
    DROP COLUMN IF EXISTS disbursed_at,
    DROP COLUMN IF EXISTS disbursement_ref,
    DROP COLUMN IF EXISTS disbursement_status;
