-- Headteacher review of lesson plans: a comment back to the teacher and when
-- it was given, so the teacher can see why a plan was sent back and the
-- headteacher can see which plans changed since they last looked.
ALTER TABLE lesson_plans
    ADD COLUMN IF NOT EXISTS review_comment TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ;

-- 'returned' = headteacher sent it back for changes.
ALTER TABLE lesson_plans DROP CONSTRAINT IF EXISTS lesson_plans_status_check;
ALTER TABLE lesson_plans ADD CONSTRAINT lesson_plans_status_check
    CHECK (status IN ('draft','submitted','approved','reviewed','returned'));

CREATE INDEX IF NOT EXISTS idx_lessons_updated ON lesson_plans(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_lessons_class_subject ON lesson_plans(class_id, subject_id, term_id);
CREATE INDEX IF NOT EXISTS idx_class_subjects_teacher ON class_subjects(teacher_id);
