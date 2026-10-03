-- Fields from the staff registration form that weren't modeled yet.
-- employment_type matters for payroll and reporting: TSC-employed teachers
-- vs. those employed by the Board of Management (BOM).
ALTER TABLE staff_profiles
    ADD COLUMN IF NOT EXISTS gender VARCHAR(10) NOT NULL DEFAULT ''
        CHECK (gender IN ('', 'male', 'female', 'other')),
    ADD COLUMN IF NOT EXISTS employment_type VARCHAR(20) NOT NULL DEFAULT ''
        CHECK (employment_type IN ('', 'tsc', 'bom', 'contract', 'intern', 'volunteer'));
