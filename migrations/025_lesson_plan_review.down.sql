DROP INDEX IF EXISTS idx_class_subjects_teacher;
DROP INDEX IF EXISTS idx_lessons_class_subject;
DROP INDEX IF EXISTS idx_lessons_updated;
UPDATE lesson_plans SET status = 'submitted' WHERE status = 'returned';
ALTER TABLE lesson_plans DROP CONSTRAINT IF EXISTS lesson_plans_status_check;
ALTER TABLE lesson_plans ADD CONSTRAINT lesson_plans_status_check
    CHECK (status IN ('draft','submitted','approved','reviewed'));
ALTER TABLE lesson_plans
    DROP COLUMN IF EXISTS submitted_at,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS review_comment;
