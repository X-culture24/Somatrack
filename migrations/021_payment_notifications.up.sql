-- One confirmation SMS per payment, however many times a payment is reported
-- (Jenga IPN and M-Pesa C2B can both report the same money). The notifier
-- claims the row before sending; a conflict means someone already did.
CREATE TABLE IF NOT EXISTS payment_notifications (
    payment_id UUID PRIMARY KEY REFERENCES payments(id) ON DELETE CASCADE,
    recipient VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
