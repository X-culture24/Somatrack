DROP TABLE IF EXISTS student_clubs;
DROP TABLE IF EXISTS clubs;

ALTER TABLE guardians
    DROP COLUMN IF EXISTS id_document_path;

DROP INDEX IF EXISTS idx_students_transport_route;
ALTER TABLE students
    DROP COLUMN IF EXISTS birth_cert_document_path,
    DROP COLUMN IF EXISTS photo_path,
    DROP COLUMN IF EXISTS transport_route_id,
    DROP COLUMN IF EXISTS uses_transport,
    DROP COLUMN IF EXISTS assessment_number,
    DROP COLUMN IF EXISTS kemis_number;
