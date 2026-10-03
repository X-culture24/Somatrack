-- Equity EFT transactions: salary disbursements sent via Jenga Send Money
CREATE TABLE IF NOT EXISTS equity_eft_transactions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_entry_id        UUID REFERENCES payroll_entries(id) ON DELETE SET NULL,
    staff_id                UUID REFERENCES staff_profiles(id) ON DELETE SET NULL,
    dest_account_number     VARCHAR(50)  NOT NULL DEFAULT '',
    dest_bank_code          VARCHAR(20)  NOT NULL DEFAULT '',
    dest_name               VARCHAR(200) NOT NULL DEFAULT '',
    amount                  NUMERIC(12,2) NOT NULL,
    reference               VARCHAR(100) NOT NULL DEFAULT '',
    status                  VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    transaction_ref         VARCHAR(100) NOT NULL DEFAULT '',
    result_desc             VARCHAR(500) NOT NULL DEFAULT '',
    raw_initiate_response   JSONB,
    raw_callback            JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Equity RTB (Request to Pay / EazzyPay C2B): parent-initiated fee payments
CREATE TABLE IF NOT EXISTS equity_rtb_requests (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id              UUID REFERENCES students(id) ON DELETE SET NULL,
    initiated_by_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number            VARCHAR(30)  NOT NULL DEFAULT '',
    wallet_name             VARCHAR(50)  NOT NULL DEFAULT 'Equity',
    amount                  NUMERIC(12,2) NOT NULL,
    partner_ref             VARCHAR(100) NOT NULL DEFAULT '',
    request_ref             VARCHAR(100) NOT NULL DEFAULT '',
    status                  VARCHAR(20)  NOT NULL DEFAULT 'pending',
    result_desc             VARCHAR(500) NOT NULL DEFAULT '',
    payment_id              UUID REFERENCES payments(id) ON DELETE SET NULL,
    raw_initiate_response   JSONB,
    raw_callback            JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_equity_eft_payroll_entry ON equity_eft_transactions(payroll_entry_id);
CREATE INDEX IF NOT EXISTS idx_equity_rtb_student ON equity_rtb_requests(student_id);
CREATE INDEX IF NOT EXISTS idx_equity_rtb_partner_ref ON equity_rtb_requests(partner_ref);
