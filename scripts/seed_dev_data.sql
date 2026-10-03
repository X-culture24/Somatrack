-- Dev/demo seed data with FIXED ids, so any endpoint can be tested
-- standalone (GET/PUT/PATCH/DELETE by id) without first running a chain of
-- "Create" requests. Not a migration — safe to re-run (ON CONFLICT DO
-- NOTHING/UPDATE throughout) and safe to skip in production.
-- These same ids are the Postman collection's default variable values.
--
-- Every column on every seeded row is populated with a realistic value
-- (not left at its blank/empty-string default) so admin UI screens like
-- StudentDetailPage render a fully-filled-in record instead of empty
-- fields — the only exceptions are class_teacher_id (staff module is
-- still a placeholder stub) and admission_applications.reviewed_by_id /
-- admission_date (genuinely unset until someone reviews the application).

INSERT INTO academic_years (id, name, start_date, end_date, is_active)
VALUES ('11111111-1111-1111-1111-111111111111', '2026', '2026-01-01', '2026-12-31', true)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date, is_active = EXCLUDED.is_active;

INSERT INTO terms (id, academic_year_id, name, term_number, start_date, end_date, is_current)
VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111',
        'Term 1', 1, '2026-01-05', '2026-04-03', true)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, term_number = EXCLUDED.term_number,
    start_date = EXCLUDED.start_date, end_date = EXCLUDED.end_date, is_current = EXCLUDED.is_current;

INSERT INTO streams (id, name, code) VALUES
    ('33333333-3333-3333-3333-333333333333', 'East', 'E'),
    ('33333333-3333-3333-3333-333333333334', 'West', 'W')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, code = EXCLUDED.code;

INSERT INTO subjects (id, name, code, description, learning_area, is_active) VALUES
    ('44444444-4444-4444-4444-444444444444', 'Mathematics', 'MATH', 'CBC Mathematics', 'STEM', true),
    ('44444444-4444-4444-4444-444444444445', 'English', 'ENG', 'CBC English', 'Languages', true),
    ('44444444-4444-4444-4444-444444444446', 'Kiswahili', 'KISW', 'CBC Kiswahili', 'Languages', true),
    ('44444444-4444-4444-4444-444444444447', 'Science & Technology', 'SCI', 'CBC Science & Technology', 'STEM', true),
    ('44444444-4444-4444-4444-444444444448', 'Social Studies', 'SST', 'CBC Social Studies', 'Humanities', true),
    ('44444444-4444-4444-4444-444444444449', 'Christian Religious Education', 'CRE', 'CBC CRE', 'Humanities', true),
    ('4444444d-4444-4444-4444-44444444444a', 'Creative Arts', 'ARTS', 'CBC Creative Arts', 'Arts', true),
    ('4444444d-4444-4444-4444-44444444444b', 'Physical & Health Education', 'PHE', 'CBC PE', 'Wellbeing', true)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, code = EXCLUDED.code,
    description = EXCLUDED.description, learning_area = EXCLUDED.learning_area, is_active = EXCLUDED.is_active;

INSERT INTO class_groups (id, name, grade, stream_id, academic_year_id, capacity) VALUES
    ('55555555-5555-5555-5555-555555555555', 'Grade 4 East', 4,
     '33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 35),
    ('55555555-5555-5555-5555-555555555556', 'Grade 4 West', 4,
     '33333333-3333-3333-3333-333333333334', '11111111-1111-1111-1111-111111111111', 35)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, grade = EXCLUDED.grade,
    stream_id = EXCLUDED.stream_id, academic_year_id = EXCLUDED.academic_year_id,
    capacity = EXCLUDED.capacity;

-- Full CBC curriculum (all 8 learning-area subjects) assigned to both
-- Grade 4 streams — powers the "CBC Curriculum" admin page. teacher_id is
-- left NULL (no teacher-subject assignment data exists yet); the page
-- shows "No teacher assigned" for those.
INSERT INTO class_subjects (class_id, subject_id, is_elective)
SELECT cg.id, subj.id, false
FROM class_groups cg
CROSS JOIN subjects subj
WHERE cg.id IN ('55555555-5555-5555-5555-555555555555', '55555555-5555-5555-5555-555555555556')
  AND subj.id IN (
    '44444444-4444-4444-4444-444444444444', '44444444-4444-4444-4444-444444444445',
    '44444444-4444-4444-4444-444444444446', '44444444-4444-4444-4444-444444444447',
    '44444444-4444-4444-4444-444444444448', '44444444-4444-4444-4444-444444444449',
    '4444444d-4444-4444-4444-44444444444a', '4444444d-4444-4444-4444-44444444444b'
  )
ON CONFLICT (class_id, subject_id) DO NOTHING;

INSERT INTO grade_requirements (id, grade, min_score, max_score, grade_letter, grade_point, remarks)
VALUES ('66666666-6666-6666-6666-666666666666', 4, 80, 100, 'EE', 4, 'Exceeds Expectation')
ON CONFLICT (id) DO UPDATE SET min_score = EXCLUDED.min_score, max_score = EXCLUDED.max_score,
    grade_point = EXCLUDED.grade_point, remarks = EXCLUDED.remarks;

INSERT INTO clubs (id, name, description, is_active) VALUES
    ('77777777-7777-7777-7777-777777777777', 'Chess Club', 'Weekly chess practice', true),
    ('77777777-7777-7777-7777-777777777778', 'Drama Club', 'Termly drama productions', true)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description,
    is_active = EXCLUDED.is_active;

-- Real 2026 transport routes/fares from ACK St. Mary's School Kabete
-- (LAWRENCE.pdf) — flat per-route monthly fare, same across every grade.
INSERT INTO transport_routes (id, route_name, route_code, description, start_location, end_location,
    stops, fare, driver_name, driver_phone, is_active) VALUES
    ('88888888-8888-8888-8888-888888888888', 'Ndumbu-ini, Vet Labs, Jerico', 'R1',
     'Route 1', 'Ndumbu-ini', 'Jerico', '["Ndumbu-ini", "Vet Labs", "Jerico"]'::jsonb,
     1300, 'James Mwangi', '0711000111', true),
    ('88888888-8888-8888-8888-888888888889', 'Ndumbu-ini, Newhope Road, Uthiru Centre, Fortsmith Rd, Reli, Gichagi', 'R2',
     'Route 2', 'Ndumbu-ini', 'Gichagi', '["Ndumbu-ini", "Newhope Road", "Uthiru Centre", "Fortsmith Rd", "Reli", "Gichagi"]'::jsonb,
     1600, 'Peter Kamau', '0711000112', true),
    ('8888888a-8888-8888-8888-888888888888', 'Gichagi, Reli, Muthua, Co-operation', 'R3',
     'Route 3', 'Gichagi', 'Co-operation', '["Gichagi", "Reli", "Muthua", "Co-operation"]'::jsonb,
     2200, 'Samuel Njoroge', '0711000113', true),
    ('8888888b-8888-8888-8888-888888888888', '87, Gichagi Kwa Chief, Loresho, KSG', 'R4',
     'Route 4', '87', 'KSG', '["87", "Gichagi Kwa Chief", "Loresho", "KSG"]'::jsonb,
     2600, 'David Otieno', '0711000114', true),
    ('8888888c-8888-8888-8888-888888888888', 'Kinoo, Kangemi, Gathondeki, Kabete', 'R5',
     'Route 5', 'Kinoo', 'Kabete', '["Kinoo", "Kangemi", "Gathondeki", "Kabete"]'::jsonb,
     3000, 'Francis Muriuki', '0711000115', true),
    ('8888888d-8888-8888-8888-888888888888', 'Mwimuto, Kabete, Kangemi, Kinoo Interior', 'R6',
     'Route 6', 'Mwimuto', 'Kinoo Interior', '["Mwimuto", "Kabete", "Kangemi", "Kinoo Interior"]'::jsonb,
     3300, 'Joseph Kariuki', '0711000116', true)
ON CONFLICT (id) DO UPDATE SET route_name = EXCLUDED.route_name, route_code = EXCLUDED.route_code,
    description = EXCLUDED.description, start_location = EXCLUDED.start_location,
    end_location = EXCLUDED.end_location, stops = EXCLUDED.stops, fare = EXCLUDED.fare,
    driver_name = EXCLUDED.driver_name, driver_phone = EXCLUDED.driver_phone,
    is_active = EXCLUDED.is_active;

INSERT INTO guardians (id, first_name, last_name, email, phone, alternate_phone, national_id, relation,
    occupation, employer, residential_address, postal_address, is_active, id_document_path) VALUES
    ('99999999-9999-9999-9999-999999999999', 'John', 'Otieno', 'john.otieno@example.com', '0712345678',
     '0733112233', '23456789', 'father', 'Accountant', 'Kabete Sacco',
     'Kabete, Kiambu County', 'P.O. Box 100-00100, Nairobi', true, '/media/documents/seed-john-id.pdf'),
    ('99999999-9999-9999-9999-999999999998', 'Alice', 'Kamau', 'alice.kamau@example.com', '0723456789',
     '0744223311', '24567890', 'mother', 'Teacher', 'Uthiru Primary School',
     'Uthiru, Kiambu County', 'P.O. Box 200-00100, Nairobi', true, '/media/documents/seed-alice-id.pdf')
ON CONFLICT (id) DO UPDATE SET first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name,
    email = EXCLUDED.email, phone = EXCLUDED.phone, alternate_phone = EXCLUDED.alternate_phone,
    national_id = EXCLUDED.national_id, relation = EXCLUDED.relation, occupation = EXCLUDED.occupation,
    employer = EXCLUDED.employer, residential_address = EXCLUDED.residential_address,
    postal_address = EXCLUDED.postal_address, is_active = EXCLUDED.is_active,
    id_document_path = EXCLUDED.id_document_path;

INSERT INTO students (id, admission_no, first_name, middle_name, last_name, full_name, search_name, gender,
    date_of_birth, national_id, birth_certificate_no, phone, email, class_id, current_class_id, stream_id,
    academic_year_id, admission_date, boarding_status, blood_group, allergies, chronic_conditions,
    residential_address, former_school, kcpe_marks, kcpe_grade, upi, status, is_active,
    kemis_number, assessment_number, uses_transport, transport_route_id, photo_path, birth_cert_document_path)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'ADM-SEED-001', 'Faith', 'Wanjiru', 'Otieno', 'Faith Wanjiru Otieno',
     'Faith Wanjiru Otieno', 'female', '2016-03-14', '', 'BC-2016-00123', '0712345671',
     'faith.otieno@stmaryskabete.ac.ke', '55555555-5555-5555-5555-555555555555',
     '55555555-5555-5555-5555-555555555555', '33333333-3333-3333-3333-333333333333',
     '11111111-1111-1111-1111-111111111111', '2026-01-10', 'day', 'O+', 'None', 'None',
     'Kabete, Kiambu County', 'Kabete Nursery School', NULL, '', 'UPI-2026-0001', 'active', true,
     'KEMIS-000123456', 'ASMT-000456', true, '88888888-8888-8888-8888-888888888888',
     '/media/photos/seed-faith.jpg', '/media/documents/seed-faith-birthcert.pdf'),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab', 'ADM-SEED-002', 'Brian', 'Njoroge', 'Kamau', 'Brian Njoroge Kamau',
     'Brian Njoroge Kamau', 'male', '2016-07-22', '', 'BC-2016-00456', '0723456780',
     'brian.kamau@stmaryskabete.ac.ke', '55555555-5555-5555-5555-555555555556',
     '55555555-5555-5555-5555-555555555556', '33333333-3333-3333-3333-333333333334',
     '11111111-1111-1111-1111-111111111111', '2026-01-10', 'day', 'A+', 'Peanuts', 'None',
     'Uthiru, Kiambu County', 'Uthiru Kindergarten', NULL, '', 'UPI-2026-0002', 'active', true,
     'KEMIS-000123457', 'ASMT-000457', false, NULL,
     '/media/photos/seed-brian.jpg', '/media/documents/seed-brian-birthcert.pdf')
ON CONFLICT (id) DO UPDATE SET
    admission_no = EXCLUDED.admission_no, first_name = EXCLUDED.first_name,
    middle_name = EXCLUDED.middle_name, last_name = EXCLUDED.last_name, full_name = EXCLUDED.full_name,
    search_name = EXCLUDED.search_name, gender = EXCLUDED.gender, date_of_birth = EXCLUDED.date_of_birth,
    national_id = EXCLUDED.national_id, birth_certificate_no = EXCLUDED.birth_certificate_no,
    phone = EXCLUDED.phone, email = EXCLUDED.email, class_id = EXCLUDED.class_id,
    current_class_id = EXCLUDED.current_class_id, stream_id = EXCLUDED.stream_id,
    academic_year_id = EXCLUDED.academic_year_id, admission_date = EXCLUDED.admission_date,
    boarding_status = EXCLUDED.boarding_status, blood_group = EXCLUDED.blood_group,
    allergies = EXCLUDED.allergies, chronic_conditions = EXCLUDED.chronic_conditions,
    residential_address = EXCLUDED.residential_address, former_school = EXCLUDED.former_school,
    kcpe_marks = EXCLUDED.kcpe_marks, kcpe_grade = EXCLUDED.kcpe_grade, upi = EXCLUDED.upi,
    status = EXCLUDED.status, is_active = EXCLUDED.is_active, kemis_number = EXCLUDED.kemis_number,
    assessment_number = EXCLUDED.assessment_number, uses_transport = EXCLUDED.uses_transport,
    transport_route_id = EXCLUDED.transport_route_id, photo_path = EXCLUDED.photo_path,
    birth_cert_document_path = EXCLUDED.birth_cert_document_path;

INSERT INTO student_guardians (student_id, guardian_id, relation, is_primary, has_custody, contact_priority) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '99999999-9999-9999-9999-999999999999', 'father', true, true, 1),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab', '99999999-9999-9999-9999-999999999998', 'mother', true, true, 1)
ON CONFLICT (student_id, guardian_id) DO NOTHING;

INSERT INTO student_clubs (student_id, club_id) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '77777777-7777-7777-7777-777777777777')
ON CONFLICT (student_id, club_id) DO NOTHING;

INSERT INTO admission_applications (id, first_name, middle_name, last_name, gender, date_of_birth,
    applying_for_grade, applying_for_year, applying_for_term, former_school, kcpe_marks, guardian_name,
    guardian_phone, guardian_email, guardian_relation, residential_address, status, notes)
VALUES ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Grace', 'Wambui', 'Mwangi', 'female', '2019-11-02',
        2, 2026, 1, 'Kawangware Nursery School', NULL, 'Peter Mwangi', '0700111222',
        'peter.mwangi@example.com', 'parent', 'Kawangware, Nairobi', 'pending',
        'Application submitted online, awaiting document review.')
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name, middle_name = EXCLUDED.middle_name, last_name = EXCLUDED.last_name,
    gender = EXCLUDED.gender, date_of_birth = EXCLUDED.date_of_birth,
    applying_for_grade = EXCLUDED.applying_for_grade, applying_for_year = EXCLUDED.applying_for_year,
    applying_for_term = EXCLUDED.applying_for_term, former_school = EXCLUDED.former_school,
    kcpe_marks = EXCLUDED.kcpe_marks, guardian_name = EXCLUDED.guardian_name,
    guardian_phone = EXCLUDED.guardian_phone, guardian_email = EXCLUDED.guardian_email,
    guardian_relation = EXCLUDED.guardian_relation, residential_address = EXCLUDED.residential_address,
    status = EXCLUDED.status, notes = EXCLUDED.notes;

-- Real 2026 fee schedule from ACK St. Mary's School Kabete (LAWRENCE.pdf),
-- replacing the earlier placeholder numbers. The real sheet only has
-- Admission + School fees (tuition) + Diary & progress book as recurring
-- line items (boarding/medical/activity/exam/library/computer/uniform/other
-- are zeroed — the school doesn't charge those as flat per-term fees; uniform
-- and transport are priced separately, see uniform_items / transport_routes).
-- grade: Playgroup=-2, PP1=-1, PP2=0, Grade N=N.
DELETE FROM fee_structures WHERE academic_year_id = '11111111-1111-1111-1111-111111111111';

INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('33f8e6df-c713-573e-a08e-130fd45f0d08', '11111111-1111-1111-1111-111111111111', -2, 1, 0, 11500, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 12500, 12500, 'Playgroup Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('2c182ad8-5cf6-5f67-9bf8-f9d0fe0df724', '11111111-1111-1111-1111-111111111111', -2, 2, 0, 11500, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11500, 11500, 'Playgroup Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('ca3e9741-0250-58af-a4b4-04553b4124f8', '11111111-1111-1111-1111-111111111111', -2, 3, 0, 10500, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10500, 10500, 'Playgroup Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('7b538084-c880-5036-be9e-6d669ed70974', '11111111-1111-1111-1111-111111111111', -1, 1, 1000, 12000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 14000, 14000, 'PP1 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('61a99fa8-e2d8-51cc-b2f1-d9f0e18c1254', '11111111-1111-1111-1111-111111111111', -1, 2, 0, 11500, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11500, 11500, 'PP1 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('9a51b88a-6acd-545a-8b45-8222725b61d0', '11111111-1111-1111-1111-111111111111', -1, 3, 0, 11000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11000, 11000, 'PP1 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('7dd07a9b-2952-59b1-996f-058398119310', '11111111-1111-1111-1111-111111111111', 0, 1, 1000, 12000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 14000, 14000, 'PP2 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('913e281a-108e-5360-b512-be57ab2a6a55', '11111111-1111-1111-1111-111111111111', 0, 2, 0, 11500, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11500, 11500, 'PP2 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('1d1aad42-d3e1-50fc-8cdd-fc604e31d0f1', '11111111-1111-1111-1111-111111111111', 0, 3, 0, 11000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11000, 11000, 'PP2 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('892d165f-50dc-5286-94b4-30806681135b', '11111111-1111-1111-1111-111111111111', 1, 1, 1000, 13000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 15000, 15000, 'Grade1 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('cf67955c-94a9-5e01-b82a-eaab12bd99cc', '11111111-1111-1111-1111-111111111111', 1, 2, 0, 13000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13000, 13000, 'Grade1 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('a3527a6e-77ac-5414-8820-fe1d5e1f6d40', '11111111-1111-1111-1111-111111111111', 1, 3, 0, 11000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11000, 11000, 'Grade1 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('1fa34181-d722-5432-bb31-429f5f5a448d', '11111111-1111-1111-1111-111111111111', 2, 1, 1000, 13000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 15000, 15000, 'Grade2 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('a9cf922d-6a1e-5908-843d-c65d1b9d886a', '11111111-1111-1111-1111-111111111111', 2, 2, 0, 13000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13000, 13000, 'Grade2 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('2273e64c-e697-5f73-8dda-2a2661fbcfde', '11111111-1111-1111-1111-111111111111', 2, 3, 0, 11000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11000, 11000, 'Grade2 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('7be1eb71-3763-50ac-8f98-453ea2cd75b9', '11111111-1111-1111-1111-111111111111', 3, 1, 1000, 13000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 15000, 15000, 'Grade3 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('9ca476ce-c8f3-5a46-8073-bb34f54ff4c1', '11111111-1111-1111-1111-111111111111', 3, 2, 0, 13000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13000, 13000, 'Grade3 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('b61bec97-a989-5198-ad08-cb700a52fc70', '11111111-1111-1111-1111-111111111111', 3, 3, 0, 11000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11000, 11000, 'Grade3 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111', 4, 1, 1000, 14000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 16000, 16000, 'Grade4 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('48cbbd48-6d92-5171-8c8a-df296ab6e32b', '11111111-1111-1111-1111-111111111111', 4, 2, 0, 14000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 14000, 14000, 'Grade4 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('b74cb585-b36d-5247-b91f-afc6a83b3542', '11111111-1111-1111-1111-111111111111', 4, 3, 0, 12000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12000, 12000, 'Grade4 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('b8954eff-2b46-59ab-924a-488852621c97', '11111111-1111-1111-1111-111111111111', 5, 1, 1000, 14000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 16000, 16000, 'Grade5 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('24d0c6cf-8e3c-5a4e-bf05-d97c0f1d28a1', '11111111-1111-1111-1111-111111111111', 5, 2, 0, 14000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 14000, 14000, 'Grade5 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('27ddf12d-15eb-5d2b-960b-66469bf9e13f', '11111111-1111-1111-1111-111111111111', 5, 3, 0, 12000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12000, 12000, 'Grade5 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('f4823177-8ea8-55e6-863d-8fac0dd1528a', '11111111-1111-1111-1111-111111111111', 6, 1, 1000, 14000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 16000, 16000, 'Grade6 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('2f666c64-5607-5cf6-bc8b-4740d17fd81b', '11111111-1111-1111-1111-111111111111', 6, 2, 0, 14000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 14000, 14000, 'Grade6 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('fdeb4c42-b9a0-5bf7-b8bb-b82a3d9de3f3', '11111111-1111-1111-1111-111111111111', 6, 3, 0, 12000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12000, 12000, 'Grade6 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('dde75f4b-f109-541e-bdb8-ab79dca92293', '11111111-1111-1111-1111-111111111111', 7, 1, 1000, 17000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 19000, 19000, 'Grade7 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('18a6c300-95cb-51ba-801d-6ed5d9fbef4b', '11111111-1111-1111-1111-111111111111', 7, 2, 0, 17000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17000, 17000, 'Grade7 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('06d19cae-a0a7-5e70-9a57-f7dbd88bfd4d', '11111111-1111-1111-1111-111111111111', 7, 3, 0, 15000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15000, 15000, 'Grade7 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('d34d7a9e-b147-5293-a23f-05f50f7d04df', '11111111-1111-1111-1111-111111111111', 8, 1, 1000, 17000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 19000, 19000, 'Grade8 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('99450a7b-3c76-5d20-9a0d-2c3bcdc441d2', '11111111-1111-1111-1111-111111111111', 8, 2, 0, 17000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17000, 17000, 'Grade8 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('06db2aea-efec-54b3-a582-b8ee3941f866', '11111111-1111-1111-1111-111111111111', 8, 3, 0, 15000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15000, 15000, 'Grade8 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('52bd5f65-88ef-5298-abff-8d2610490ee0', '11111111-1111-1111-1111-111111111111', 9, 1, 1000, 17000, 0, 0, 0, 0, 0, 0, 0, 0, 1000, 0, 19000, 19000, 'Grade9 Term 1 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('9dfe7455-5dd8-5667-a66c-54282acb16eb', '11111111-1111-1111-1111-111111111111', 9, 2, 0, 17000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17000, 17000, 'Grade9 Term 2 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;
INSERT INTO fee_structures (id, academic_year_id, grade, term_number, admission_fee, tuition_fee, boarding_fee, transport_fee, medical_fee, activity_fee, exam_fee, library_fee, computer_fee, uniform_fee, diary_book_fee, other_fee, total_day, total_boarding, notes, is_active) VALUES ('dd8e239a-7292-5017-88ca-dc4603676155', '11111111-1111-1111-1111-111111111111', 9, 3, 0, 15000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15000, 15000, 'Grade9 Term 3 2026 (real fee schedule)', true) ON CONFLICT (academic_year_id, grade, term_number) DO UPDATE SET admission_fee=EXCLUDED.admission_fee, tuition_fee=EXCLUDED.tuition_fee, diary_book_fee=EXCLUDED.diary_book_fee, boarding_fee=0, transport_fee=0, medical_fee=0, activity_fee=0, exam_fee=0, library_fee=0, computer_fee=0, uniform_fee=0, other_fee=0, total_day=EXCLUDED.total_day, total_boarding=EXCLUDED.total_boarding, notes=EXCLUDED.notes;

INSERT INTO discount_waivers (id, student_id, term_id, waiver_type, amount, reason, effective_date)
VALUES ('dddddddd-dddd-dddd-dddd-dddddddddddd', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab',
    '22222222-2222-2222-2222-222222222222', 'sibling_discount', 3000, 'Sibling discount', '2026-01-05')
ON CONFLICT (id) DO UPDATE SET amount = EXCLUDED.amount;

-- Invoices/invoice_lines are intentionally NOT seeded directly here — they're
-- produced by POST /billing/generate/ (term_id = the seed term above), which
-- reads the fee_structure + discount_waivers rows just inserted. Run that
-- once (the Postman collection's "Generate Billing" request does this) to
-- populate real invoices for the two seed students.
