-- Short codes carried in SMS links (see internal/api/links.go). Only a
-- SHA-256 of each code is stored, so a leaked database dump can't be turned
-- back into working links.
CREATE TABLE IF NOT EXISTS short_links (
    code_hash   BYTEA PRIMARY KEY,
    purpose     SMALLINT NOT NULL,
    record_id   UUID NOT NULL,
    student_id  UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    use_count   INTEGER NOT NULL DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_short_links_record ON short_links(purpose, record_id);
CREATE INDEX IF NOT EXISTS idx_short_links_expires ON short_links(expires_at);
