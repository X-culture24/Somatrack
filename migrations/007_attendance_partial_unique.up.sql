CREATE UNIQUE INDEX IF NOT EXISTS idx_attendance_records_daily_unique
ON attendance_records(student_id, date) WHERE subject_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_attendance_records_class_date
ON attendance_records(class_id, date) WHERE subject_id IS NULL;
