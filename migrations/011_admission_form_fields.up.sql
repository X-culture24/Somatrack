-- Fields from the real paper admission form that weren't modeled yet:
-- KEMIS/assessment numbers, transport use, document uploads (birth cert,
-- guardian ID, pupil passport photo), and co-curricular clubs.
ALTER TABLE students
    ADD COLUMN IF NOT EXISTS kemis_number VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS assessment_number VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS uses_transport BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS transport_route_id UUID REFERENCES transport_routes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS photo_path TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS birth_cert_document_path TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_students_transport_route ON students(transport_route_id);

ALTER TABLE guardians
    ADD COLUMN IF NOT EXISTS id_document_path TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS clubs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS student_clubs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
    joined_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, club_id)
);
CREATE INDEX IF NOT EXISTS idx_student_clubs_student ON student_clubs(student_id);
