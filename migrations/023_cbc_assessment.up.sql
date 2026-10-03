-- CBC assessment extensions:
-- 1. Add reflection text on each mark (per-subject comment by teacher)
-- 2. Create cbc_signoffs for the headteacher sign-off section of the report

ALTER TABLE marks ADD COLUMN IF NOT EXISTS reflection TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS cbc_signoffs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id              UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    term_id                 UUID NOT NULL REFERENCES terms(id)    ON DELETE CASCADE,
    class_id                UUID REFERENCES class_groups(id)      ON DELETE SET NULL,
    teacher_comment         TEXT NOT NULL DEFAULT '',
    general_performance_level TEXT NOT NULL DEFAULT '',
    headteacher_comment     TEXT NOT NULL DEFAULT '',
    headteacher_signed      BOOLEAN NOT NULL DEFAULT FALSE,
    headteacher_signed_at   TIMESTAMPTZ,
    headteacher_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    opening_date            TEXT NOT NULL DEFAULT '',
    closing_date            TEXT NOT NULL DEFAULT '',
    created_by              UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, term_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_signoffs_class_term ON cbc_signoffs (class_id, term_id);
