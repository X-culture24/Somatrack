CREATE TABLE IF NOT EXISTS mpesa_webhook_events (
    id UUID PRIMARY KEY,
    trans_id TEXT NOT NULL UNIQUE,
    raw_payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','processed','failed')),
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_mpesa_events_status ON mpesa_webhook_events(status);
CREATE INDEX IF NOT EXISTS idx_mpesa_events_created ON mpesa_webhook_events(created_at DESC);

CREATE TABLE IF NOT EXISTS invoice_payment_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    amount_applied NUMERIC(14,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(invoice_id, payment_id)
);
