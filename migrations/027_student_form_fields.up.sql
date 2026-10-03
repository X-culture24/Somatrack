-- Remaining fields from the school's student admission form: consent,
-- family/welfare status, medical contacts, sponsorship and home location.
ALTER TABLE students
    ADD COLUMN IF NOT EXISTS photo_consent BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS parental_status VARCHAR(20) NOT NULL DEFAULT ''
        CHECK (parental_status IN ('', 'both_alive', 'father_deceased', 'mother_deceased', 'both_deceased')),
    ADD COLUMN IF NOT EXISTS has_special_needs BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS special_needs TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS religion VARCHAR(30) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS doctor_name VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS doctor_phone VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS preferred_hospital VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS has_scholarship BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS scholarship_type VARCHAR(20) NOT NULL DEFAULT ''
        CHECK (scholarship_type IN ('', 'full', 'partial', 'bursary', 'sponsorship', 'other')),
    ADD COLUMN IF NOT EXISTS sponsor_name VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sponsor_phone VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sponsor_location VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sponsor_contact_person VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sponsor_contact_details VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS citizenship VARCHAR(60) NOT NULL DEFAULT 'Kenya',
    ADD COLUMN IF NOT EXISTS home_county VARCHAR(40) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS residence_county VARCHAR(40) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sub_county VARCHAR(60) NOT NULL DEFAULT '';
