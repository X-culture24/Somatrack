-- FINGERPRINT MODULE TABLES
CREATE TABLE IF NOT EXISTS fingerprint_devices (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    device_type TEXT NOT NULL,
    secret_hash TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_fp_devices_active ON fingerprint_devices(is_active);

CREATE TABLE IF NOT EXISTS fingerprint_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    template_data BYTEA NOT NULL,
    template_version INTEGER NOT NULL DEFAULT 1,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enrolled_by UUID NOT NULL REFERENCES users(id),
    device_id UUID NOT NULL REFERENCES fingerprint_devices(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_fp_templates_student_active ON fingerprint_templates(student_id) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_fp_templates_student_id ON fingerprint_templates(student_id);

CREATE TABLE IF NOT EXISTS fingerprint_scans (
    id UUID PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES fingerprint_devices(id),
    student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    template_id UUID REFERENCES fingerprint_templates(id) ON DELETE SET NULL,
    fingerprint_match_score DOUBLE PRECISION,
    scan_type TEXT NOT NULL CHECK (scan_type IN ('entry','exit','class_checkin')),
    timestamp TIMESTAMPTZ NOT NULL,
    raw_score DOUBLE PRECISION,
    is_duplicate BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fp_scans_student_ts ON fingerprint_scans(student_id, scan_type, timestamp);
CREATE INDEX IF NOT EXISTS idx_fp_scans_device_ts ON fingerprint_scans(device_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_fp_scans_student_day ON fingerprint_scans(student_id, scan_type, DATE(timestamp AT TIME ZONE 'Africa/Nairobi'));

CREATE TABLE IF NOT EXISTS fingerprint_attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    scan_type TEXT NOT NULL CHECK (scan_type IN ('entry','exit','class_checkin')),
    timestamp TIMESTAMPTZ NOT NULL,
    device_id UUID NOT NULL REFERENCES fingerprint_devices(id),
    raw_score DOUBLE PRECISION,
    match_score DOUBLE PRECISION,
    scan_id UUID REFERENCES fingerprint_scans(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, date, scan_type)
);
CREATE INDEX IF NOT EXISTS idx_fp_attendance_date ON fingerprint_attendance(date, scan_type);
CREATE INDEX IF NOT EXISTS idx_fp_attendance_student ON fingerprint_attendance(student_id, date DESC);
