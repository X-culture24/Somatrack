DROP TABLE IF EXISTS staff_attendance;

DROP INDEX IF EXISTS idx_fp_scans_staff_day;
DROP INDEX IF EXISTS idx_fp_scans_staff_ts;
ALTER TABLE fingerprint_scans
    DROP CONSTRAINT IF EXISTS chk_fp_scans_owner,
    DROP COLUMN IF EXISTS staff_id;

DROP INDEX IF EXISTS idx_fp_templates_staff_id;
DROP INDEX IF EXISTS idx_fp_templates_staff_active;
ALTER TABLE fingerprint_templates
    DROP CONSTRAINT IF EXISTS chk_fp_templates_owner,
    DROP COLUMN IF EXISTS staff_id,
    ALTER COLUMN student_id SET NOT NULL;
