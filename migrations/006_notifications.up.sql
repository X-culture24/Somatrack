CREATE TABLE IF NOT EXISTS notification_logs (
    id UUID PRIMARY KEY,
    channel TEXT NOT NULL CHECK (channel IN ('sms','email','in_app','push')),
    recipient TEXT NOT NULL,
    subject TEXT,
    content TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','sending','delivered','failed','read')),
    error TEXT,
    meta JSONB,
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notif_logs_channel_created ON notification_logs(channel, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_logs_status ON notification_logs(status);
