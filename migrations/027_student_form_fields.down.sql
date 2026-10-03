ALTER TABLE students
    DROP COLUMN IF EXISTS photo_consent, DROP COLUMN IF EXISTS parental_status,
    DROP COLUMN IF EXISTS has_special_needs, DROP COLUMN IF EXISTS special_needs,
    DROP COLUMN IF EXISTS religion, DROP COLUMN IF EXISTS doctor_name, DROP COLUMN IF EXISTS doctor_phone,
    DROP COLUMN IF EXISTS preferred_hospital, DROP COLUMN IF EXISTS has_scholarship,
    DROP COLUMN IF EXISTS scholarship_type, DROP COLUMN IF EXISTS sponsor_name, DROP COLUMN IF EXISTS sponsor_phone,
    DROP COLUMN IF EXISTS sponsor_location, DROP COLUMN IF EXISTS sponsor_contact_person,
    DROP COLUMN IF EXISTS sponsor_contact_details, DROP COLUMN IF EXISTS citizenship,
    DROP COLUMN IF EXISTS home_county, DROP COLUMN IF EXISTS residence_county, DROP COLUMN IF EXISTS sub_county;
