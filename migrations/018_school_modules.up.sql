-- Tables for the record-keeping modules in the SmartShule reference screens
-- (mock-up/). They are served by the declarative resource engine
-- (internal/resource, specs in internal/api/resources.go), so each needs an
-- id UUID primary key and created_at. Students are referenced with
-- ON DELETE RESTRICT: these are permanent school records.

-- ---------------------------------------------------------------- Students
CREATE TABLE IF NOT EXISTS admission_enquiries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_name VARCHAR(200) NOT NULL,
    date_of_birth DATE,
    grade_sought VARCHAR(50) NOT NULL DEFAULT '',
    parent_name VARCHAR(200) NOT NULL,
    parent_phone VARCHAR(20) NOT NULL,
    parent_email VARCHAR(254) NOT NULL DEFAULT '',
    enquiry_date DATE NOT NULL DEFAULT CURRENT_DATE,
    source VARCHAR(20) NOT NULL DEFAULT 'walk_in'
        CHECK (source IN ('walk_in','phone','website','referral','social_media','other')),
    status VARCHAR(20) NOT NULL DEFAULT 'new'
        CHECK (status IN ('new','contacted','visited','applied','admitted','declined')),
    follow_up_date DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS pocket_money_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    txn_type VARCHAR(12) NOT NULL CHECK (txn_type IN ('deposit','withdrawal')),
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    txn_date DATE NOT NULL DEFAULT CURRENT_DATE,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pocket_money_student ON pocket_money_transactions(student_id);

CREATE TABLE IF NOT EXISTS discipline_cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    incident_date DATE NOT NULL,
    severity VARCHAR(10) NOT NULL DEFAULT 'minor' CHECK (severity IN ('minor','major','severe')),
    offence VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    action_taken TEXT NOT NULL DEFAULT '',
    reported_by VARCHAR(200) NOT NULL DEFAULT '',
    parent_notified BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved')),
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_discipline_student ON discipline_cases(student_id);

CREATE TABLE IF NOT EXISTS student_hobbies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    hobby VARCHAR(120) NOT NULL,
    category VARCHAR(12) NOT NULL DEFAULT 'other'
        CHECK (category IN ('sport','art','music','reading','technology','outdoors','other')),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS leadership_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    position VARCHAR(120) NOT NULL,
    academic_year_id UUID REFERENCES academic_years(id) ON DELETE SET NULL,
    start_date DATE,
    end_date DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS leaving_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    certificate_no VARCHAR(50) NOT NULL UNIQUE,
    leaving_date DATE NOT NULL,
    reason VARCHAR(12) NOT NULL DEFAULT 'completed' CHECK (reason IN ('completed','transfer','other')),
    last_grade VARCHAR(50) NOT NULL DEFAULT '',
    conduct VARCHAR(10) NOT NULL DEFAULT 'good' CHECK (conduct IN ('excellent','good','fair','poor')),
    remarks TEXT NOT NULL DEFAULT '',
    issued_by VARCHAR(200) NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS emergency_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    name VARCHAR(200) NOT NULL,
    relationship VARCHAR(60) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    alt_phone VARCHAR(20) NOT NULL DEFAULT '',
    address VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    cert_type VARCHAR(10) NOT NULL DEFAULT 'other' CHECK (cert_type IN ('national','other')),
    title VARCHAR(200) NOT NULL,
    certificate_no VARCHAR(80) NOT NULL DEFAULT '',
    issuer VARCHAR(200) NOT NULL DEFAULT '',
    issue_date DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------- Boarding
CREATE TABLE IF NOT EXISTS hostels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL UNIQUE,
    gender VARCHAR(6) NOT NULL DEFAULT 'mixed' CHECK (gender IN ('boys','girls','mixed')),
    capacity INTEGER NOT NULL DEFAULT 0 CHECK (capacity >= 0),
    warden_name VARCHAR(200) NOT NULL DEFAULT '',
    warden_phone VARCHAR(20) NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS hostel_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostel_id UUID NOT NULL REFERENCES hostels(id) ON DELETE RESTRICT,
    room_no VARCHAR(30) NOT NULL,
    floor VARCHAR(30) NOT NULL DEFAULT '',
    capacity INTEGER NOT NULL DEFAULT 0 CHECK (capacity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hostel_id, room_no)
);

CREATE TABLE IF NOT EXISTS hostel_beds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES hostel_rooms(id) ON DELETE RESTRICT,
    bed_no VARCHAR(30) NOT NULL,
    status VARCHAR(12) NOT NULL DEFAULT 'available' CHECK (status IN ('available','occupied','maintenance')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (room_id, bed_no)
);

CREATE TABLE IF NOT EXISTS bed_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bed_id UUID NOT NULL REFERENCES hostel_beds(id) ON DELETE RESTRICT,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    term_id UUID REFERENCES terms(id) ON DELETE SET NULL,
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE,
    status VARCHAR(8) NOT NULL DEFAULT 'active' CHECK (status IN ('active','ended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- One active bed per student, one active student per bed.
CREATE UNIQUE INDEX IF NOT EXISTS uq_bed_alloc_active_bed ON bed_allocations(bed_id) WHERE status = 'active';
CREATE UNIQUE INDEX IF NOT EXISTS uq_bed_alloc_active_student ON bed_allocations(student_id) WHERE status = 'active';

-- ---------------------------------------------------------------- Front office
CREATE TABLE IF NOT EXISTS visitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_name VARCHAR(200) NOT NULL,
    phone VARCHAR(20) NOT NULL DEFAULT '',
    id_number VARCHAR(30) NOT NULL DEFAULT '',
    purpose VARCHAR(255) NOT NULL,
    person_to_see VARCHAR(200) NOT NULL DEFAULT '',
    vehicle_reg VARCHAR(20) NOT NULL DEFAULT '',
    time_in TIMESTAMPTZ NOT NULL DEFAULT now(),
    time_out TIMESTAMPTZ,
    notes TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------- Procurement
CREATE TABLE IF NOT EXISTS suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL UNIQUE,
    contact_person VARCHAR(200) NOT NULL DEFAULT '',
    phone VARCHAR(20) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL DEFAULT '',
    kra_pin VARCHAR(20) NOT NULL DEFAULT '',
    address VARCHAR(255) NOT NULL DEFAULT '',
    payment_terms VARCHAR(100) NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS expense_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES expense_categories(id) ON DELETE RESTRICT,
    supplier_id UUID REFERENCES suppliers(id) ON DELETE SET NULL,
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    payee VARCHAR(200) NOT NULL,
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(8) NOT NULL DEFAULT 'cash' CHECK (payment_method IN ('cash','mpesa','bank','cheque')),
    reference VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(expense_date);

CREATE TABLE IF NOT EXISTS supplier_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    invoice_no VARCHAR(60) NOT NULL,
    invoice_date DATE NOT NULL,
    due_date DATE,
    amount NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    amount_paid NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (amount_paid >= 0),
    status VARCHAR(8) NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid','partial','paid')),
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (supplier_id, invoice_no)
);

CREATE TABLE IF NOT EXISTS purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    po_number VARCHAR(40) NOT NULL UNIQUE,
    supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expected_date DATE,
    items TEXT NOT NULL,
    total_amount NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    status VARCHAR(10) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','sent','received','cancelled')),
    notes TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS requisitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    requested_by VARCHAR(200) NOT NULL,
    request_date DATE NOT NULL DEFAULT CURRENT_DATE,
    items TEXT NOT NULL,
    estimated_cost NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (estimated_cost >= 0),
    status VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','fulfilled')),
    approved_by VARCHAR(200) NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS petty_cash_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    txn_date DATE NOT NULL DEFAULT CURRENT_DATE,
    txn_type VARCHAR(8) NOT NULL CHECK (txn_type IN ('top_up','expense')),
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    description VARCHAR(255) NOT NULL,
    received_by VARCHAR(200) NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS other_revenue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(200) NOT NULL,
    amount NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    received_date DATE NOT NULL DEFAULT CURRENT_DATE,
    payment_method VARCHAR(8) NOT NULL DEFAULT 'cash' CHECK (payment_method IN ('cash','mpesa','bank','cheque')),
    reference VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sales_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL UNIQUE,
    unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    stock_qty INTEGER NOT NULL DEFAULT 0,
    reorder_level INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL REFERENCES sales_items(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    sale_date DATE NOT NULL DEFAULT CURRENT_DATE,
    buyer_name VARCHAR(200) NOT NULL DEFAULT '',
    student_id UUID REFERENCES students(id) ON DELETE RESTRICT,
    payment_method VARCHAR(8) NOT NULL DEFAULT 'cash' CHECK (payment_method IN ('cash','mpesa','bank','invoice')),
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------- Accounts
CREATE TABLE IF NOT EXISTS vote_heads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL DEFAULT '',
    priority INTEGER NOT NULL DEFAULT 100,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fee_pledges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    pledge_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL,
    pledged_by VARCHAR(200) NOT NULL DEFAULT '',
    phone VARCHAR(20) NOT NULL DEFAULT '',
    status VARCHAR(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open','fulfilled','defaulted')),
    notes TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS balances_brought_forward (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    amount NUMERIC(12,2) NOT NULL,
    as_of_date DATE NOT NULL,
    term_id UUID REFERENCES terms(id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------- Academics
CREATE TABLE IF NOT EXISTS exam_threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE RESTRICT,
    curriculum VARCHAR(10) NOT NULL DEFAULT 'cbc' CHECK (curriculum IN ('cbc','844_igcse')),
    cats_weight NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (cats_weight BETWEEN 0 AND 100),
    main_exam_weight NUMERIC(5,2) NOT NULL DEFAULT 100 CHECK (main_exam_weight BETWEEN 0 AND 100),
    description TEXT NOT NULL DEFAULT '',
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (cats_weight + main_exam_weight = 100)
);

CREATE TABLE IF NOT EXISTS exam_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_by VARCHAR(14) NOT NULL DEFAULT 'class_teacher' CHECK (comment_by IN ('class_teacher','head_teacher')),
    min_score NUMERIC(5,2) NOT NULL CHECK (min_score BETWEEN 0 AND 100),
    max_score NUMERIC(5,2) NOT NULL CHECK (max_score BETWEEN 0 AND 100),
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (min_score <= max_score)
);

-- ---------------------------------------------------------------- People
CREATE TABLE IF NOT EXISTS board_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name VARCHAR(200) NOT NULL,
    position VARCHAR(120) NOT NULL,
    phone VARCHAR(20) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL DEFAULT '',
    term_start DATE,
    term_end DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS staff_appraisals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id UUID NOT NULL REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    appraisal_date DATE NOT NULL,
    period VARCHAR(60) NOT NULL DEFAULT '',
    score NUMERIC(5,2) CHECK (score BETWEEN 0 AND 100),
    rating VARCHAR(12) NOT NULL DEFAULT 'meets'
        CHECK (rating IN ('outstanding','exceeds','meets','below','poor')),
    appraiser VARCHAR(200) NOT NULL DEFAULT '',
    strengths TEXT NOT NULL DEFAULT '',
    improvements TEXT NOT NULL DEFAULT '',
    comments TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS clearances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_type VARCHAR(7) NOT NULL CHECK (subject_type IN ('student','staff')),
    student_id UUID REFERENCES students(id) ON DELETE RESTRICT,
    staff_id UUID REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    department VARCHAR(20) NOT NULL
        CHECK (department IN ('library','finance','boarding','sports','laboratory','stores','other')),
    status VARCHAR(8) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','cleared','blocked')),
    cleared_by VARCHAR(200) NOT NULL DEFAULT '',
    cleared_on DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((subject_type = 'student' AND student_id IS NOT NULL AND staff_id IS NULL)
        OR (subject_type = 'staff' AND staff_id IS NOT NULL AND student_id IS NULL))
);
