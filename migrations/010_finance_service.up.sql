-- FINANCE SERVICE: payroll disbursement tracking + M-Pesa B2C payout log
ALTER TABLE payroll_entries
    ADD COLUMN IF NOT EXISTS disbursement_status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (disbursement_status IN ('pending','initiated','success','failed')),
    ADD COLUMN IF NOT EXISTS disbursement_ref TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS disbursed_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS b2c_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_entry_id UUID REFERENCES payroll_entries(id) ON DELETE SET NULL,
    staff_id UUID REFERENCES staff_profiles(id) ON DELETE SET NULL,
    phone_number VARCHAR(15) NOT NULL,
    amount NUMERIC(14,2) NOT NULL,
    remarks VARCHAR(100) NOT NULL DEFAULT '',
    occasion VARCHAR(100) NOT NULL DEFAULT '',
    command_id VARCHAR(50) NOT NULL DEFAULT 'SalaryPayment',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING','ACCEPTED','SUCCESS','FAILED','TIMEOUT','REJECTED')),
    originator_conversation_id VARCHAR(100) NOT NULL UNIQUE,
    conversation_id VARCHAR(100) NOT NULL DEFAULT '',
    response_code VARCHAR(10) NOT NULL DEFAULT '',
    response_description VARCHAR(255) NOT NULL DEFAULT '',
    result_code VARCHAR(10) NOT NULL DEFAULT '',
    result_desc VARCHAR(255) NOT NULL DEFAULT '',
    transaction_id VARCHAR(50) NOT NULL DEFAULT '',
    transaction_amount NUMERIC(14,2),
    receiver_public_name VARCHAR(255) NOT NULL DEFAULT '',
    raw_initiate_response JSONB,
    raw_callback JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_b2c_conversation ON b2c_transactions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_b2c_payroll_entry ON b2c_transactions(payroll_entry_id);
CREATE INDEX IF NOT EXISTS idx_b2c_staff ON b2c_transactions(staff_id, created_at DESC);
