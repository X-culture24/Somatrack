-- Reverse FK dependency order
DROP TABLE IF EXISTS audit_logs;

-- Nova
DROP TABLE IF EXISTS nova_xp_transactions;
DROP TABLE IF EXISTS nova_student_xp;
DROP TABLE IF EXISTS nova_login_tickets;
DROP TABLE IF EXISTS nova_discussion_posts;
DROP TABLE IF EXISTS nova_discussions;
DROP TABLE IF EXISTS nova_question_attempts;
DROP TABLE IF EXISTS nova_quiz_attempts;
DROP TABLE IF EXISTS nova_questions;
DROP TABLE IF EXISTS nova_quizzes;
DROP TABLE IF EXISTS nova_submission_files;
DROP TABLE IF EXISTS nova_submissions;
DROP TABLE IF EXISTS nova_assignments;
DROP TABLE IF EXISTS nova_materials;
DROP TABLE IF EXISTS nova_course_enrollments;
DROP TABLE IF EXISTS nova_courses;

-- Communication
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS message_templates;
DROP TABLE IF EXISTS contact_group_members;
DROP TABLE IF EXISTS contact_groups;
DROP TABLE IF EXISTS announcements;

-- Inventory
DROP TABLE IF EXISTS inventory_transactions;
DROP TABLE IF EXISTS inventory_items;

-- Library
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS loans;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS book_categories;

-- Transport
DROP TABLE IF EXISTS transport_attendance;
DROP TABLE IF EXISTS vehicles;
DROP TABLE IF EXISTS transport_routes;

-- Academics
DROP TABLE IF EXISTS lesson_plans;
DROP TABLE IF EXISTS timetable_slots;
DROP TABLE IF EXISTS marks;
DROP TABLE IF EXISTS assessments;
DROP TABLE IF EXISTS attendance_records;

-- Finance
DROP TABLE IF EXISTS finance_webhook_events;
DROP TABLE IF EXISTS discount_waivers;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS invoice_lines;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS uniform_items;
DROP TABLE IF EXISTS adhoc_fees;
DROP TABLE IF EXISTS fee_structures;

-- Students
DROP TABLE IF EXISTS student_withdrawals;
DROP TABLE IF EXISTS student_transfers;
DROP TABLE IF EXISTS admission_applications;
DROP TABLE IF EXISTS student_documents;
DROP TABLE IF EXISTS medical_records;
DROP TABLE IF EXISTS student_guardians;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS guardians;

-- Staff
DROP TABLE IF EXISTS staff_deductions;
DROP TABLE IF EXISTS payroll_entries;
DROP TABLE IF EXISTS payroll_runs;
DROP TABLE IF EXISTS staff_profiles;
DROP TABLE IF EXISTS departments;

-- School
DROP TABLE IF EXISTS school_profiles;
DROP TABLE IF EXISTS grade_requirements;
DROP TABLE IF EXISTS class_subjects;
DROP TABLE IF EXISTS class_groups;
DROP TABLE IF EXISTS subject_components;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS streams;
DROP TABLE IF EXISTS terms;
DROP TABLE IF EXISTS academic_years;

-- Auth
DROP TABLE IF EXISTS user_user_permissions;
DROP TABLE IF EXISTS user_groups;
DROP TABLE IF EXISTS auth_group_permissions;
DROP TABLE IF EXISTS auth_group;
DROP TABLE IF EXISTS auth_permission;
DROP TABLE IF EXISTS django_content_type;
DROP TABLE IF EXISTS users;

-- Extension
DROP EXTENSION IF EXISTS "pgcrypto";
