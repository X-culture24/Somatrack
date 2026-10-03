-- The rest of the staff registration form, mirroring the student form:
-- personal and welfare details, medical contacts, home location, training
-- background, and scanned documents (photo, ID, certificates, appointment
-- letter, certificate of good conduct).
ALTER TABLE staff_profiles
    ADD COLUMN IF NOT EXISTS middle_name VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS marital_status VARCHAR(10) NOT NULL DEFAULT ''
        CHECK (marital_status IN ('', 'single', 'married', 'divorced', 'widowed', 'separated')),
    ADD COLUMN IF NOT EXISTS religion VARCHAR(30) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS has_disability BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS disability_details TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS alternate_phone VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS postal_address VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS citizenship VARCHAR(60) NOT NULL DEFAULT 'Kenya',
    ADD COLUMN IF NOT EXISTS home_county VARCHAR(40) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS residence_county VARCHAR(40) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sub_county VARCHAR(60) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS blood_group VARCHAR(5) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS allergies TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS doctor_name VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS doctor_phone VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS preferred_hospital VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS teaching_subjects VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS training_institution VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS year_of_graduation VARCHAR(4) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS previous_employer VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS contract_end_date DATE,
    ADD COLUMN IF NOT EXISTS photo_consent BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS photo_path VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS id_document_path VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS certificates_path VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS appointment_letter_path VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS good_conduct_path VARCHAR(255) NOT NULL DEFAULT '';
