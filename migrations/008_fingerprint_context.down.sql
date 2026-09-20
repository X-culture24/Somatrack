DROP INDEX IF EXISTS idx_fp_scans_class;
DROP INDEX IF EXISTS idx_fp_scans_subject;

ALTER TABLE fingerprint_scans
    DROP COLUMN IF EXISTS remarks,
    DROP COLUMN IF EXISTS class_id,
    DROP COLUMN IF EXISTS subject_id;

DROP INDEX IF EXISTS idx_fp_devices_subject;
DROP INDEX IF EXISTS idx_fp_devices_class;

ALTER TABLE fingerprint_devices
    DROP COLUMN IF EXISTS subject_id,
    DROP COLUMN IF EXISTS class_id;
