-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- ACCOUNTS / USERS
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    password VARCHAR(128) NOT NULL,
    last_login TIMESTAMPTZ,
    is_superuser BOOLEAN NOT NULL DEFAULT FALSE,
    first_name VARCHAR(150) NOT NULL DEFAULT '',
    last_name VARCHAR(150) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL UNIQUE,
    is_staff BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    date_joined TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    role VARCHAR(32) NOT NULL DEFAULT 'student' CHECK (role IN (
        'system_admin','deputy_head','head_teacher','academic_coordinator',
        'finance_officer','bursar','class_teacher','subject_teacher',
        'librarian','nurse','storekeeper','transport_officer',
        'student','parent'
    )),
    phone VARCHAR(32) NOT NULL DEFAULT '',
    must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
    is_active_staff_account BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- Django groups / permissions — minimal stubs so token-claim lookups don't need ORM tables
CREATE TABLE IF NOT EXISTS django_content_type (
    id SERIAL PRIMARY KEY,
    app_label VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    UNIQUE (app_label, model)
);
CREATE TABLE IF NOT EXISTS auth_permission (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    content_type_id INTEGER NOT NULL REFERENCES django_content_type(id) ON DELETE CASCADE,
    codename VARCHAR(100) NOT NULL,
    UNIQUE (content_type_id, codename)
);
CREATE TABLE IF NOT EXISTS auth_group (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS auth_group_permissions (
    id BIGSERIAL PRIMARY KEY,
    group_id INTEGER NOT NULL REFERENCES auth_group(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES auth_permission(id) ON DELETE CASCADE,
    UNIQUE (group_id, permission_id)
);
CREATE TABLE IF NOT EXISTS user_groups (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id INTEGER NOT NULL REFERENCES auth_group(id) ON DELETE CASCADE,
    UNIQUE (user_id, group_id)
);
CREATE TABLE IF NOT EXISTS user_user_permissions (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES auth_permission(id) ON DELETE CASCADE,
    UNIQUE (user_id, permission_id)
);

-- ============================================================
-- SCHOOL MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS academic_years (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    term_number INTEGER NOT NULL CHECK (term_number IN (1,2,3)),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (academic_year_id, term_number)
);
CREATE INDEX IF NOT EXISTS idx_terms_current ON terms(is_current);

CREATE TABLE IF NOT EXISTS streams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    code VARCHAR(10) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subject_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL,
    weight INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subject_id, code)
);

CREATE TABLE IF NOT EXISTS class_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL,
    grade INTEGER NOT NULL CHECK (grade BETWEEN 1 AND 13),
    stream_id UUID REFERENCES streams(id) ON DELETE SET NULL,
    class_teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    capacity INTEGER NOT NULL DEFAULT 40,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (academic_year_id, grade, stream_id)
);
CREATE INDEX IF NOT EXISTS idx_classgroups_grade ON class_groups(grade);
CREATE INDEX IF NOT EXISTS idx_classgroups_teacher ON class_groups(class_teacher_id);

CREATE TABLE IF NOT EXISTS class_subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_elective BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (class_id, subject_id)
);

CREATE TABLE IF NOT EXISTS grade_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grade INTEGER NOT NULL CHECK (grade BETWEEN 1 AND 13),
    min_score DOUBLE PRECISION NOT NULL,
    max_score DOUBLE PRECISION NOT NULL,
    grade_letter VARCHAR(4) NOT NULL,
    grade_point DOUBLE PRECISION NOT NULL DEFAULT 0,
    remarks VARCHAR(200) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (grade, grade_letter)
);

CREATE TABLE IF NOT EXISTS school_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_name VARCHAR(200) NOT NULL DEFAULT 'ACK St. Mary''s School Kabete',
    address TEXT NOT NULL DEFAULT '',
    phone VARCHAR(32) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL DEFAULT '',
    website VARCHAR(200) NOT NULL DEFAULT '',
    motto VARCHAR(200) NOT NULL DEFAULT '',
    vision TEXT NOT NULL DEFAULT '',
    mission TEXT NOT NULL DEFAULT '',
    bank_account_name VARCHAR(200) NOT NULL DEFAULT '',
    bank_name VARCHAR(200) NOT NULL DEFAULT '',
    bank_account_no VARCHAR(50) NOT NULL DEFAULT '',
    mpesa_paybill VARCHAR(20) NOT NULL DEFAULT '',
    mpesa_account_prefix VARCHAR(20) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- STAFF MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE,
    head_of_department_id UUID REFERENCES users(id) ON DELETE SET NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS staff_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    employee_no VARCHAR(50) NOT NULL UNIQUE,
    national_id VARCHAR(20) NOT NULL DEFAULT '',
    tsc_no VARCHAR(50) NOT NULL DEFAULT '',
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    designation VARCHAR(100) NOT NULL DEFAULT '',
    date_of_birth DATE,
    date_of_joining DATE,
    basic_salary NUMERIC(14,2) NOT NULL DEFAULT 0,
    kra_pin VARCHAR(20) NOT NULL DEFAULT '',
    nhif_no VARCHAR(20) NOT NULL DEFAULT '',
    nssf_no VARCHAR(20) NOT NULL DEFAULT '',
    bank_name VARCHAR(100) NOT NULL DEFAULT '',
    bank_branch VARCHAR(100) NOT NULL DEFAULT '',
    bank_account_no VARCHAR(50) NOT NULL DEFAULT '',
    emergency_contact_name VARCHAR(200) NOT NULL DEFAULT '',
    emergency_contact_phone VARCHAR(32) NOT NULL DEFAULT '',
    emergency_contact_relation VARCHAR(50) NOT NULL DEFAULT '',
    highest_qualification VARCHAR(100) NOT NULL DEFAULT '',
    subjects_taught TEXT NOT NULL DEFAULT '',
    residential_address TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_staff_dept ON staff_profiles(department_id);
CREATE INDEX IF NOT EXISTS idx_staff_active ON staff_profiles(is_active);

CREATE TABLE IF NOT EXISTS payroll_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
    year INTEGER NOT NULL,
    run_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','approved','paid','reconciled')),
    processed_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (year, month)
);

CREATE TABLE IF NOT EXISTS payroll_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_run_id UUID NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES staff_profiles(id) ON DELETE CASCADE,
    basic_salary NUMERIC(14,2) NOT NULL DEFAULT 0,
    allowances NUMERIC(14,2) NOT NULL DEFAULT 0,
    deductions NUMERIC(14,2) NOT NULL DEFAULT 0,
    gross_pay NUMERIC(14,2) NOT NULL DEFAULT 0,
    net_pay NUMERIC(14,2) NOT NULL DEFAULT 0,
    paye NUMERIC(14,2) NOT NULL DEFAULT 0,
    nhif NUMERIC(14,2) NOT NULL DEFAULT 0,
    nssf NUMERIC(14,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (payroll_run_id, staff_id)
);
CREATE INDEX IF NOT EXISTS idx_payroll_entries_staff ON payroll_entries(staff_id);

CREATE TABLE IF NOT EXISTS staff_deductions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id UUID NOT NULL REFERENCES staff_profiles(id) ON DELETE CASCADE,
    deduction_type VARCHAR(50) NOT NULL CHECK (deduction_type IN ('paye','nhif','nssf','sacco','loan','insurance','other')),
    description VARCHAR(200) NOT NULL DEFAULT '',
    amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    is_recurring BOOLEAN NOT NULL DEFAULT TRUE,
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_staff_deductions_staff ON staff_deductions(staff_id);

-- ============================================================
-- STUDENTS MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS guardians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    first_name VARCHAR(150) NOT NULL,
    last_name VARCHAR(150) NOT NULL,
    email VARCHAR(254),
    phone VARCHAR(32) NOT NULL,
    alternate_phone VARCHAR(32) NOT NULL DEFAULT '',
    national_id VARCHAR(20) NOT NULL DEFAULT '',
    relation VARCHAR(50) NOT NULL DEFAULT 'parent' CHECK (relation IN ('father','mother','guardian','sibling','other','parent')),
    occupation VARCHAR(100) NOT NULL DEFAULT '',
    employer VARCHAR(200) NOT NULL DEFAULT '',
    residential_address TEXT NOT NULL DEFAULT '',
    postal_address VARCHAR(200) NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_guardians_phone ON guardians(phone);
CREATE INDEX IF NOT EXISTS idx_guardians_name ON guardians(last_name, first_name);

CREATE TABLE IF NOT EXISTS students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    admission_no VARCHAR(50) NOT NULL UNIQUE,
    first_name VARCHAR(150) NOT NULL,
    middle_name VARCHAR(150) NOT NULL DEFAULT '',
    last_name VARCHAR(150) NOT NULL,
    full_name VARCHAR(450) NOT NULL,
    search_name VARCHAR(450) NOT NULL,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('male','female','other')),
    date_of_birth DATE,
    national_id VARCHAR(20) NOT NULL DEFAULT '',
    birth_certificate_no VARCHAR(50) NOT NULL DEFAULT '',
    phone VARCHAR(32) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL DEFAULT '',
    class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    current_class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    stream_id UUID REFERENCES streams(id) ON DELETE SET NULL,
    academic_year_id UUID REFERENCES academic_years(id) ON DELETE SET NULL,
    admission_date DATE,
    boarding_status VARCHAR(20) NOT NULL DEFAULT 'day' CHECK (boarding_status IN ('day','boarding')),
    blood_group VARCHAR(5) NOT NULL DEFAULT '',
    allergies TEXT NOT NULL DEFAULT '',
    chronic_conditions TEXT NOT NULL DEFAULT '',
    residential_address TEXT NOT NULL DEFAULT '',
    former_school VARCHAR(200) NOT NULL DEFAULT '',
    kcpe_marks INTEGER,
    kcpe_grade VARCHAR(4) NOT NULL DEFAULT '',
    upi VARCHAR(30) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','transferred','withdrawn','suspended','graduated','admitted')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_students_admission ON students(admission_no);
CREATE INDEX IF NOT EXISTS idx_students_name ON students(full_name);
CREATE INDEX IF NOT EXISTS idx_students_search ON students(search_name);
CREATE INDEX IF NOT EXISTS idx_students_class ON students(class_id);
CREATE INDEX IF NOT EXISTS idx_students_status ON students(status);

CREATE TABLE IF NOT EXISTS student_guardians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    guardian_id UUID NOT NULL REFERENCES guardians(id) ON DELETE CASCADE,
    relation VARCHAR(50) NOT NULL DEFAULT 'parent' CHECK (relation IN ('father','mother','guardian','sibling','other','parent')),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    has_custody BOOLEAN NOT NULL DEFAULT TRUE,
    contact_priority INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, guardian_id)
);

CREATE TABLE IF NOT EXISTS medical_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    visit_date DATE NOT NULL,
    symptoms TEXT NOT NULL DEFAULT '',
    diagnosis TEXT NOT NULL DEFAULT '',
    treatment_given TEXT NOT NULL DEFAULT '',
    medication_prescribed TEXT NOT NULL DEFAULT '',
    referred_to_hospital BOOLEAN NOT NULL DEFAULT FALSE,
    hospital_name VARCHAR(200) NOT NULL DEFAULT '',
    administered_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    follow_up_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_medical_student ON medical_records(student_id, visit_date DESC);

CREATE TABLE IF NOT EXISTS student_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL CHECK (document_type IN ('birth_certificate','kcpe_certificate','national_id','transfer_letter','medical_report','other')),
    document_name VARCHAR(200) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    uploaded_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_student_docs_student ON student_documents(student_id);

CREATE TABLE IF NOT EXISTS admission_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(150) NOT NULL,
    middle_name VARCHAR(150) NOT NULL DEFAULT '',
    last_name VARCHAR(150) NOT NULL,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('male','female','other')),
    date_of_birth DATE,
    applying_for_grade INTEGER NOT NULL CHECK (applying_for_grade BETWEEN 1 AND 13),
    applying_for_year INTEGER NOT NULL,
    applying_for_term INTEGER NOT NULL CHECK (applying_for_term IN (1,2,3)),
    former_school VARCHAR(200) NOT NULL DEFAULT '',
    kcpe_marks INTEGER,
    guardian_name VARCHAR(200) NOT NULL,
    guardian_phone VARCHAR(32) NOT NULL,
    guardian_email VARCHAR(254) NOT NULL DEFAULT '',
    guardian_relation VARCHAR(50) NOT NULL DEFAULT 'parent',
    residential_address TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','reviewing','shortlisted','interview','accepted','rejected','deferred','admitted')),
    reviewed_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    admission_date DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_admissions_status ON admission_applications(status);

CREATE TABLE IF NOT EXISTS student_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    transfer_date DATE NOT NULL,
    from_class_id UUID REFERENCES class_groups(id),
    to_class_id UUID REFERENCES class_groups(id),
    reason TEXT NOT NULL DEFAULT '',
    approved_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS student_withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    withdrawal_date DATE NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    reason_code VARCHAR(50) NOT NULL DEFAULT 'other' CHECK (reason_code IN ('financial','academic','disciplinary','health','family_relocation','transfer','other')),
    refund_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    approved_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- FINANCE MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS fee_structures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade INTEGER NOT NULL CHECK (grade BETWEEN 1 AND 13),
    term_number INTEGER NOT NULL CHECK (term_number IN (1,2,3)),
    tuition_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    boarding_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    transport_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    medical_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    activity_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    exam_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    library_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    computer_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    uniform_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    other_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_day NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_boarding NUMERIC(14,2) NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (academic_year_id, grade, term_number)
);

CREATE TABLE IF NOT EXISTS adhoc_fees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    grade INTEGER,
    stream_id UUID REFERENCES streams(id) ON DELETE SET NULL,
    class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    term_id UUID REFERENCES terms(id) ON DELETE SET NULL,
    is_mandatory BOOLEAN NOT NULL DEFAULT TRUE,
    applies_to_boarding_only BOOLEAN NOT NULL DEFAULT FALSE,
    deadline DATE,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS uniform_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'general' CHECK (category IN ('general','boys','girls','sports','winter','house')),
    gender VARCHAR(10) NOT NULL DEFAULT 'unisex' CHECK (gender IN ('male','female','unisex')),
    size VARCHAR(20) NOT NULL DEFAULT 'M',
    unit_price NUMERIC(14,2) NOT NULL DEFAULT 0,
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    reorder_level INTEGER NOT NULL DEFAULT 10,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    invoice_no VARCHAR(50) NOT NULL UNIQUE,
    invoice_date DATE NOT NULL,
    due_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid','partial','paid','overdue','cancelled','refunded')),
    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_paid NUMERIC(14,2) NOT NULL DEFAULT 0,
    total_waivers NUMERIC(14,2) NOT NULL DEFAULT 0,
    balance NUMERIC(14,2) NOT NULL DEFAULT 0,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, term_id)
);
CREATE INDEX IF NOT EXISTS idx_invoices_student ON invoices(student_id);
CREATE INDEX IF NOT EXISTS idx_invoices_term ON invoices(term_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_due ON invoices(due_date);

CREATE TABLE IF NOT EXISTS invoice_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    line_type VARCHAR(50) NOT NULL CHECK (line_type IN ('tuition','boarding','transport','medical','activity','exam','library','computer','uniform','other','ad_hoc','uniform_item')),
    description TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price NUMERIC(14,2) NOT NULL DEFAULT 0,
    amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    uniform_item_id UUID REFERENCES uniform_items(id) ON DELETE SET NULL,
    adhoc_fee_id UUID REFERENCES adhoc_fees(id) ON DELETE SET NULL,
    is_optional BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invoice_lines_invoice ON invoice_lines(invoice_id);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    receipt_no VARCHAR(50) NOT NULL UNIQUE,
    amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    payment_method VARCHAR(30) NOT NULL DEFAULT 'mpesa' CHECK (payment_method IN ('mpesa','cash','cheque','bank_transfer','card','mobile_money','other')),
    payment_date DATE NOT NULL,
    transaction_id VARCHAR(100),
    bank_name VARCHAR(200) NOT NULL DEFAULT '',
    cheque_no VARCHAR(50) NOT NULL DEFAULT '',
    cheque_date DATE,
    payer_name VARCHAR(200) NOT NULL DEFAULT '',
    payer_phone VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'confirmed' CHECK (status IN ('pending','confirmed','bounced','reversed','failed','refunded')),
    allocated_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    unallocated_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    reference TEXT NOT NULL DEFAULT '',
    received_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payments_student ON payments(student_id);
CREATE INDEX IF NOT EXISTS idx_payments_date ON payments(payment_date);
CREATE INDEX IF NOT EXISTS idx_payments_transaction ON payments(transaction_id);
CREATE INDEX IF NOT EXISTS idx_payments_method ON payments(payment_method, status);

CREATE TABLE IF NOT EXISTS discount_waivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    waiver_type VARCHAR(50) NOT NULL CHECK (waiver_type IN ('scholarship','bursary','discount','staff_child','orphan','sibling_discount','other')),
    code VARCHAR(50) NOT NULL DEFAULT '',
    amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    percentage NUMERIC(5,2),
    reason TEXT NOT NULL DEFAULT '',
    approved_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    effective_date DATE NOT NULL,
    expiry_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_waivers_student ON discount_waivers(student_id, term_id);

CREATE TABLE IF NOT EXISTS finance_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'mpesa',
    external_id VARCHAR(200),
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processed','failed','ignored')),
    error TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webhook_external ON finance_webhook_events(source, external_id);
CREATE INDEX IF NOT EXISTS idx_webhook_status ON finance_webhook_events(status);

-- ============================================================
-- ACADEMICS MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS attendance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'present' CHECK (status IN ('present','absent','late','sick','permission','holiday','suspended')),
    marked_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    remarks VARCHAR(200) NOT NULL DEFAULT '',
    time_in TIME,
    time_out TIME,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, date, subject_id)
);
CREATE INDEX IF NOT EXISTS idx_attendance_date ON attendance_records(date);
CREATE INDEX IF NOT EXISTS idx_attendance_student_date ON attendance_records(student_id, date);
CREATE INDEX IF NOT EXISTS idx_attendance_class_date ON attendance_records(class_id, date);

CREATE TABLE IF NOT EXISTS assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    component_id UUID REFERENCES subject_components(id) ON DELETE SET NULL,
    name VARCHAR(200) NOT NULL,
    assessment_type VARCHAR(50) NOT NULL CHECK (assessment_type IN ('cat1','cat2','cat3','midterm','endterm','mock','assignment','project','practical','homework','other')),
    total_marks DOUBLE PRECISION NOT NULL DEFAULT 100,
    weight DOUBLE PRECISION NOT NULL DEFAULT 100,
    assessment_date DATE,
    description TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_graded BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_assessments_term ON assessments(term_id);

CREATE TABLE IF NOT EXISTS marks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    weighted_score DOUBLE PRECISION,
    grade_letter VARCHAR(4),
    grade_point DOUBLE PRECISION,
    comments TEXT NOT NULL DEFAULT '',
    marked_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    marked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assessment_id, student_id)
);
CREATE INDEX IF NOT EXISTS idx_marks_student ON marks(student_id);

CREATE TABLE IF NOT EXISTS timetable_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,
    day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    period_number INTEGER NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    venue VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (term_id, class_id, day_of_week, period_number)
);

CREATE TABLE IF NOT EXISTS lesson_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL,
    lesson_date DATE NOT NULL,
    topic VARCHAR(300) NOT NULL,
    sub_topic VARCHAR(300) NOT NULL DEFAULT '',
    learning_objectives TEXT NOT NULL DEFAULT '',
    learning_outcomes TEXT NOT NULL DEFAULT '',
    teaching_methods TEXT NOT NULL DEFAULT '',
    materials TEXT NOT NULL DEFAULT '',
    introduction TEXT NOT NULL DEFAULT '',
    lesson_development TEXT NOT NULL DEFAULT '',
    conclusion TEXT NOT NULL DEFAULT '',
    assessment TEXT NOT NULL DEFAULT '',
    homework TEXT NOT NULL DEFAULT '',
    reflection TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','submitted','approved','reviewed')),
    submitted_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_lessons_teacher ON lesson_plans(teacher_id, lesson_date);

-- ============================================================
-- TRANSPORT MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS transport_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_name VARCHAR(200) NOT NULL UNIQUE,
    route_code VARCHAR(20) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    start_location VARCHAR(200) NOT NULL,
    end_location VARCHAR(200) NOT NULL,
    stops JSONB NOT NULL DEFAULT '[]'::jsonb,
    fare NUMERIC(14,2) NOT NULL DEFAULT 0,
    driver_name VARCHAR(200) NOT NULL DEFAULT '',
    driver_phone VARCHAR(32) NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plate_no VARCHAR(20) NOT NULL UNIQUE,
    vehicle_type VARCHAR(50) NOT NULL DEFAULT 'bus' CHECK (vehicle_type IN ('bus','van','car','other')),
    capacity INTEGER NOT NULL DEFAULT 40,
    route_id UUID REFERENCES transport_routes(id) ON DELETE SET NULL,
    driver_name VARCHAR(200) NOT NULL DEFAULT '',
    driver_phone VARCHAR(32) NOT NULL DEFAULT '',
    last_service_date DATE,
    next_service_date DATE,
    insurance_expiry DATE,
    road_license_expiry DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','maintenance','grounded','decommissioned')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transport_attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    route_id UUID NOT NULL REFERENCES transport_routes(id) ON DELETE CASCADE,
    vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    trip_type VARCHAR(10) NOT NULL CHECK (trip_type IN ('morning','evening')),
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'onboard' CHECK (status IN ('onboard','not_onboard','absent','permission')),
    boarded_at TIME,
    alighted_at TIME,
    recorded_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, route_id, trip_type, date)
);

-- ============================================================
-- LIBRARY MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS book_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    fine_per_day NUMERIC(10,2) NOT NULL DEFAULT 5,
    loan_period_days INTEGER NOT NULL DEFAULT 14,
    max_renewals INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL,
    author VARCHAR(300) NOT NULL DEFAULT '',
    isbn VARCHAR(50) NOT NULL DEFAULT '',
    category_id UUID REFERENCES book_categories(id) ON DELETE SET NULL,
    publisher VARCHAR(200) NOT NULL DEFAULT '',
    publication_year INTEGER,
    edition VARCHAR(50) NOT NULL DEFAULT '',
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    shelf_location VARCHAR(100) NOT NULL DEFAULT '',
    total_copies INTEGER NOT NULL DEFAULT 1,
    available_copies INTEGER NOT NULL DEFAULT 1,
    reference_only BOOLEAN NOT NULL DEFAULT FALSE,
    condition VARCHAR(20) NOT NULL DEFAULT 'good' CHECK (condition IN ('new','good','fair','poor','damaged','lost')),
    price NUMERIC(14,2) NOT NULL DEFAULT 0,
    date_acquired DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_author ON books(author);
CREATE INDEX IF NOT EXISTS idx_books_isbn ON books(isbn);
CREATE INDEX IF NOT EXISTS idx_books_category ON books(category_id);

CREATE TABLE IF NOT EXISTS loans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    borrowed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at TIMESTAMPTZ NOT NULL,
    returned_at TIMESTAMPTZ,
    renew_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','returned','overdue','lost')),
    fine_days INTEGER NOT NULL DEFAULT 0,
    fine_amount NUMERIC(10,2) NOT NULL DEFAULT 0,
    fine_paid BOOLEAN NOT NULL DEFAULT FALSE,
    issued_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    received_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_loans_student ON loans(student_id);
CREATE INDEX IF NOT EXISTS idx_loans_due ON loans(due_at, status);
CREATE INDEX IF NOT EXISTS idx_loans_book ON loans(book_id, status);

CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    reserved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','fulfilled','expired','cancelled')),
    fulfilled_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- INVENTORY MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    sku VARCHAR(50) NOT NULL UNIQUE,
    category VARCHAR(50) NOT NULL DEFAULT 'general' CHECK (category IN ('stationery','uniform','books','cleaning','kitchen','sports','furniture','electronics','medical','other')),
    unit_of_measure VARCHAR(20) NOT NULL DEFAULT 'pcs',
    unit_price NUMERIC(14,2) NOT NULL DEFAULT 0,
    quantity_in_stock INTEGER NOT NULL DEFAULT 0,
    reorder_level INTEGER NOT NULL DEFAULT 10,
    location VARCHAR(100) NOT NULL DEFAULT '',
    supplier VARCHAR(200) NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_inventory_category ON inventory_items(category);
CREATE INDEX IF NOT EXISTS idx_inventory_sku ON inventory_items(sku);

CREATE TABLE IF NOT EXISTS inventory_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('purchase','issue','return','adjustment','damage','loss','transfer')),
    quantity INTEGER NOT NULL,
    unit_price NUMERIC(14,2),
    total_amount NUMERIC(14,2),
    recipient VARCHAR(200) NOT NULL DEFAULT '',
    supplier VARCHAR(200) NOT NULL DEFAULT '',
    reference_no VARCHAR(50) NOT NULL DEFAULT '',
    processed_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    transaction_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_inv_trans_item ON inventory_transactions(item_id, transaction_date);

-- ============================================================
-- COMMUNICATION MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(300) NOT NULL,
    content TEXT NOT NULL,
    summary VARCHAR(500) NOT NULL DEFAULT '',
    audience VARCHAR(20) NOT NULL DEFAULT 'all' CHECK (audience IN ('all','staff','parents','students','teachers','finance','administration','class')),
    target_class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'normal' CHECK (priority IN ('low','normal','high','urgent')),
    publish_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_published BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_announcements_publish ON announcements(publish_date DESC);
CREATE INDEX IF NOT EXISTS idx_announcements_audience ON announcements(audience);

CREATE TABLE IF NOT EXISTS contact_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    group_type VARCHAR(20) NOT NULL DEFAULT 'general' CHECK (group_type IN ('general','class','staff','parents','students','department','club')),
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contact_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES contact_groups(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    phone VARCHAR(32) NOT NULL,
    email VARCHAR(254) NOT NULL DEFAULT '',
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    guardian_id UUID REFERENCES guardians(id) ON DELETE SET NULL,
    UNIQUE (group_id, phone)
);

CREATE TABLE IF NOT EXISTS message_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    channel VARCHAR(10) NOT NULL DEFAULT 'sms' CHECK (channel IN ('sms','email','in_app','push')),
    subject VARCHAR(300) NOT NULL DEFAULT '',
    body_template TEXT NOT NULL,
    variables JSONB NOT NULL DEFAULT '[]'::jsonb,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel VARCHAR(10) NOT NULL CHECK (channel IN ('sms','email','in_app','push')),
    sender_id UUID REFERENCES users(id) ON DELETE SET NULL,
    recipient_phone VARCHAR(32),
    recipient_email VARCHAR(254),
    recipient_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    group_id UUID REFERENCES contact_groups(id) ON DELETE SET NULL,
    template_id UUID REFERENCES message_templates(id) ON DELETE SET NULL,
    subject VARCHAR(300) NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','queued','sending','sent','delivered','failed','read')),
    external_reference VARCHAR(200),
    error TEXT,
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_recipient_user ON messages(recipient_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel, created_at DESC);

-- ============================================================
-- NOVA (LEARNING MANAGEMENT) MODULE
-- ============================================================
CREATE TABLE IF NOT EXISTS nova_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(300) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    code VARCHAR(50) NOT NULL UNIQUE,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    class_id UUID REFERENCES class_groups(id) ON DELETE SET NULL,
    term_id UUID REFERENCES terms(id) ON DELETE SET NULL,
    teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,
    cover_image_url VARCHAR(500) NOT NULL DEFAULT '',
    syllabus_outline TEXT NOT NULL DEFAULT '',
    grading_criteria TEXT NOT NULL DEFAULT '',
    prerequisites TEXT NOT NULL DEFAULT '',
    learning_outcomes TEXT NOT NULL DEFAULT '',
    expected_hours INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    published_at TIMESTAMPTZ,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nova_courses_status ON nova_courses(status);
CREATE INDEX IF NOT EXISTS idx_nova_courses_class ON nova_courses(class_id);

CREATE TABLE IF NOT EXISTS nova_course_enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES nova_courses(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    progress_pct INTEGER NOT NULL DEFAULT 0,
    final_grade VARCHAR(4),
    final_score DOUBLE PRECISION,
    status VARCHAR(20) NOT NULL DEFAULT 'enrolled' CHECK (status IN ('enrolled','in_progress','completed','dropped')),
    UNIQUE (course_id, student_id)
);

CREATE TABLE IF NOT EXISTS nova_materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES nova_courses(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES nova_materials(id) ON DELETE SET NULL,
    title VARCHAR(300) NOT NULL,
    material_type VARCHAR(30) NOT NULL CHECK (material_type IN ('folder','pdf','video','document','link','audio','image','text','quiz','assignment')),
    content_url VARCHAR(500) NOT NULL DEFAULT '',
    content_text TEXT NOT NULL DEFAULT '',
    file_path VARCHAR(500) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    file_name VARCHAR(300) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_visible BOOLEAN NOT NULL DEFAULT TRUE,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    unlock_at TIMESTAMPTZ,
    lock_at TIMESTAMPTZ,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nova_materials_course ON nova_materials(course_id, sort_order);

CREATE TABLE IF NOT EXISTS nova_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES nova_courses(id) ON DELETE CASCADE,
    title VARCHAR(300) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    material_id UUID REFERENCES nova_materials(id) ON DELETE SET NULL,
    due_at TIMESTAMPTZ,
    lock_at TIMESTAMPTZ,
    unlock_at TIMESTAMPTZ,
    total_marks DOUBLE PRECISION NOT NULL DEFAULT 100,
    assignment_type VARCHAR(30) NOT NULL DEFAULT 'upload' CHECK (assignment_type IN ('upload','text','url','file','quiz')),
    allowed_submission_types JSONB NOT NULL DEFAULT '["pdf","doc","docx"]'::jsonb,
    max_attempts INTEGER NOT NULL DEFAULT 1,
    max_file_size_bytes BIGINT NOT NULL DEFAULT 10485760,
    allow_late BOOLEAN NOT NULL DEFAULT FALSE,
    late_penalty_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
    group_assignment BOOLEAN NOT NULL DEFAULT FALSE,
    published BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nova_assign_course ON nova_assignments(course_id);

CREATE TABLE IF NOT EXISTS nova_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES nova_assignments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    submission_text TEXT NOT NULL DEFAULT '',
    submission_url VARCHAR(500) NOT NULL DEFAULT '',
    attempt_number INTEGER NOT NULL DEFAULT 1,
    submitted_at TIMESTAMPTZ,
    graded_at TIMESTAMPTZ,
    graded_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    score DOUBLE PRECISION,
    grade_letter VARCHAR(4),
    grade_point DOUBLE PRECISION,
    feedback TEXT NOT NULL DEFAULT '',
    is_late BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','submitted','unsubmitted','graded','returned','needs_review')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assignment_id, student_id, attempt_number)
);
CREATE INDEX IF NOT EXISTS idx_nova_sub_student ON nova_submissions(student_id, assignment_id);

CREATE TABLE IF NOT EXISTS nova_submission_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES nova_submissions(id) ON DELETE CASCADE,
    file_name VARCHAR(300) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nova_quizzes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES nova_courses(id) ON DELETE CASCADE,
    material_id UUID REFERENCES nova_materials(id) ON DELETE SET NULL,
    assignment_id UUID REFERENCES nova_assignments(id) ON DELETE SET NULL,
    title VARCHAR(300) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    time_limit_minutes INTEGER,
    max_attempts INTEGER NOT NULL DEFAULT 1,
    shuffle_questions BOOLEAN NOT NULL DEFAULT TRUE,
    shuffle_answers BOOLEAN NOT NULL DEFAULT TRUE,
    show_correct_answers BOOLEAN NOT NULL DEFAULT TRUE,
    show_correct_answers_at TIMESTAMPTZ,
    total_marks DOUBLE PRECISION NOT NULL DEFAULT 0,
    unlock_at TIMESTAMPTZ,
    lock_at TIMESTAMPTZ,
    due_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','closed','archived')),
    published_at TIMESTAMPTZ,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nova_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id UUID NOT NULL REFERENCES nova_quizzes(id) ON DELETE CASCADE,
    question_type VARCHAR(20) NOT NULL CHECK (question_type IN ('mcq','multiselect','short','long','truefalse','fill','numeric','match','file')),
    sort_order INTEGER NOT NULL DEFAULT 0,
    question_text TEXT NOT NULL,
    marks DOUBLE PRECISION NOT NULL DEFAULT 1,
    correct_answer TEXT,
    explanation TEXT NOT NULL DEFAULT '',
    options JSONB NOT NULL DEFAULT '[]'::jsonb,
    matching_pairs JSONB NOT NULL DEFAULT '[]'::jsonb,
    accepted_answers JSONB NOT NULL DEFAULT '[]'::jsonb,
    tolerance DOUBLE PRECISION,
    image_url VARCHAR(500) NOT NULL DEFAULT '',
    difficulty VARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (difficulty IN ('easy','medium','hard')),
    needs_manual_grade BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nova_questions_quiz ON nova_questions(quiz_id, sort_order);

CREATE TABLE IF NOT EXISTS nova_quiz_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id UUID NOT NULL REFERENCES nova_quizzes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    submission_id UUID REFERENCES nova_submissions(id) ON DELETE SET NULL,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    time_spent_seconds INTEGER,
    total_marks DOUBLE PRECISION NOT NULL DEFAULT 0,
    scored_marks DOUBLE PRECISION,
    score_pct DOUBLE PRECISION,
    grade_letter VARCHAR(4),
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress','completed','submitted','needs_review','abandoned')),
    manually_graded BOOLEAN NOT NULL DEFAULT FALSE,
    graded_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    graded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (quiz_id, student_id, attempt_number)
);
CREATE INDEX IF NOT EXISTS idx_nova_attempt_student ON nova_quiz_attempts(student_id, quiz_id);

CREATE TABLE IF NOT EXISTS nova_question_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL REFERENCES nova_quiz_attempts(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES nova_questions(id) ON DELETE CASCADE,
    student_answer TEXT,
    student_options JSONB,
    student_match JSONB,
    student_file_url VARCHAR(500),
    is_correct BOOLEAN,
    awarded_marks DOUBLE PRECISION,
    manual_feedback TEXT NOT NULL DEFAULT '',
    manually_graded BOOLEAN NOT NULL DEFAULT FALSE,
    graded_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    time_spent_seconds INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (attempt_id, question_id)
);

CREATE TABLE IF NOT EXISTS nova_discussions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES nova_courses(id) ON DELETE CASCADE,
    title VARCHAR(300) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nova_disc_course ON nova_discussions(course_id, created_at DESC);

CREATE TABLE IF NOT EXISTS nova_discussion_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discussion_id UUID NOT NULL REFERENCES nova_discussions(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES nova_discussion_posts(id) ON DELETE SET NULL,
    author_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    author_student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    attachments JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    is_answer BOOLEAN NOT NULL DEFAULT FALSE,
    likes_count INTEGER NOT NULL DEFAULT 0,
    edited_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nova_posts_disc ON nova_discussion_posts(discussion_id, created_at);

CREATE TABLE IF NOT EXISTS nova_login_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(128) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL,
    scopes VARCHAR(32)[] NOT NULL DEFAULT ARRAY['portal','nova']::VARCHAR(32)[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    consumed BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_nova_tickets_code ON nova_login_tickets(code);
CREATE INDEX IF NOT EXISTS idx_nova_tickets_user ON nova_login_tickets(user_id, expires_at);

CREATE TABLE IF NOT EXISTS nova_student_xp (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL UNIQUE REFERENCES students(id) ON DELETE CASCADE,
    total_xp INTEGER NOT NULL DEFAULT 0,
    current_level INTEGER NOT NULL DEFAULT 1,
    current_level_xp INTEGER NOT NULL DEFAULT 0,
    next_level_xp INTEGER NOT NULL DEFAULT 100,
    badges JSONB NOT NULL DEFAULT '[]'::jsonb,
    streak_days INTEGER NOT NULL DEFAULT 0,
    last_activity_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nova_xp_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    xp_delta INTEGER NOT NULL,
    reason VARCHAR(100) NOT NULL,
    reference_type VARCHAR(50),
    reference_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nova_xp_student ON nova_xp_transactions(student_id, created_at DESC);

-- ============================================================
-- AUDIT TRAIL (immutable)
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_role VARCHAR(32),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    payload_before JSONB,
    payload_after JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_type, resource_id);
