ALTER TABLE fingerprint_devices
    ADD COLUMN IF NOT EXISTS class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fp_devices_class ON fingerprint_devices(class_id);
CREATE INDEX IF NOT EXISTS idx_fp_devices_subject ON fingerprint_devices(subject_id);

ALTER TABLE fingerprint_scans
    ADD COLUMN IF NOT EXISTS subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS remarks TEXT;

CREATE INDEX IF NOT EXISTS idx_fp_scans_subject ON fingerprint_scans(subject_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_fp_scans_class ON fingerprint_scans(class_id, timestamp);
