CREATE TABLE IF NOT EXISTS stk_push_requests (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id                UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    initiated_by_id           UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number              VARCHAR(20) NOT NULL,
    amount                    NUMERIC(14,2) NOT NULL,
    account_reference         VARCHAR(50) NOT NULL,
    merchant_request_id       VARCHAR(100),
    checkout_request_id       VARCHAR(100) UNIQUE,
    status                    VARCHAR(20) NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending', 'accepted', 'success', 'failed', 'cancelled')),
    result_code               VARCHAR(10),
    result_desc               TEXT NOT NULL DEFAULT '',
    mpesa_receipt_no          VARCHAR(50),
    payment_id                UUID REFERENCES payments(id) ON DELETE SET NULL,
    raw_initiate_response     JSONB,
    raw_callback              JSONB,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stk_push_student ON stk_push_requests(student_id);
CREATE INDEX IF NOT EXISTS idx_stk_push_status ON stk_push_requests(status);
