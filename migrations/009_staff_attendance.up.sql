-- STAFF FINGERPRINT ATTENDANCE
ALTER TABLE fingerprint_templates
    ALTER COLUMN student_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS staff_id UUID REFERENCES staff_profiles(id) ON DELETE CASCADE,
    ADD CONSTRAINT chk_fp_templates_owner CHECK (
        (student_id IS NOT NULL AND staff_id IS NULL) OR
        (student_id IS NULL AND staff_id IS NOT NULL)
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_fp_templates_staff_active ON fingerprint_templates(staff_id) WHERE is_active = TRUE AND staff_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_fp_templates_staff_id ON fingerprint_templates(staff_id);

ALTER TABLE fingerprint_scans
    ADD COLUMN IF NOT EXISTS staff_id UUID REFERENCES staff_profiles(id) ON DELETE SET NULL,
    ADD CONSTRAINT chk_fp_scans_owner CHECK (
        student_id IS NOT NULL OR staff_id IS NOT NULL
    );

CREATE INDEX IF NOT EXISTS idx_fp_scans_staff_ts ON fingerprint_scans(staff_id, scan_type, timestamp);
CREATE INDEX IF NOT EXISTS idx_fp_scans_staff_day ON fingerprint_scans(staff_id, scan_type, DATE(timestamp AT TIME ZONE 'Africa/Nairobi'));

CREATE TABLE IF NOT EXISTS staff_attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id UUID NOT NULL REFERENCES staff_profiles(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    scan_type TEXT NOT NULL CHECK (scan_type IN ('entry','exit')),
    timestamp TIMESTAMPTZ NOT NULL,
    device_id UUID NOT NULL REFERENCES fingerprint_devices(id),
    raw_score DOUBLE PRECISION,
    match_score DOUBLE PRECISION,
    scan_id UUID REFERENCES fingerprint_scans(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (staff_id, date, scan_type)
);
CREATE INDEX IF NOT EXISTS idx_staff_attendance_date ON staff_attendance(date, scan_type);
CREATE INDEX IF NOT EXISTS idx_staff_attendance_staff ON staff_attendance(staff_id, date DESC);
