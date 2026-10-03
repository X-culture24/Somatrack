ALTER TABLE staff_profiles
    DROP COLUMN IF EXISTS middle_name, DROP COLUMN IF EXISTS marital_status, DROP COLUMN IF EXISTS religion,
    DROP COLUMN IF EXISTS has_disability, DROP COLUMN IF EXISTS disability_details,
    DROP COLUMN IF EXISTS alternate_phone, DROP COLUMN IF EXISTS postal_address, DROP COLUMN IF EXISTS citizenship,
    DROP COLUMN IF EXISTS home_county, DROP COLUMN IF EXISTS residence_county, DROP COLUMN IF EXISTS sub_county,
    DROP COLUMN IF EXISTS blood_group, DROP COLUMN IF EXISTS allergies, DROP COLUMN IF EXISTS doctor_name,
    DROP COLUMN IF EXISTS doctor_phone, DROP COLUMN IF EXISTS preferred_hospital,
    DROP COLUMN IF EXISTS teaching_subjects, DROP COLUMN IF EXISTS training_institution,
    DROP COLUMN IF EXISTS year_of_graduation, DROP COLUMN IF EXISTS previous_employer,
    DROP COLUMN IF EXISTS contract_end_date, DROP COLUMN IF EXISTS photo_consent, DROP COLUMN IF EXISTS photo_path,
    DROP COLUMN IF EXISTS id_document_path, DROP COLUMN IF EXISTS certificates_path,
    DROP COLUMN IF EXISTS appointment_letter_path, DROP COLUMN IF EXISTS good_conduct_path;
