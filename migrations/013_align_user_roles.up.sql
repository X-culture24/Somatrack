-- Aligns users.role's CHECK constraint with the Go application's actual
-- types.Role enum (internal/shared/types), which the frontend's UsersPage
-- role picker already assumes. The two had drifted: the DB allowed
-- deputy_head/bursar/nurse/storekeeper/transport_officer (never referenced
-- by the Go enum) while missing school_director/deputy_head_teacher/
-- transport_coordinator/admissions_officer/receptionist (which the frontend
-- already offers, so creating a user with any of those would have failed).
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role::text = ANY (ARRAY[
    'system_admin', 'school_director', 'finance_officer', 'head_teacher',
    'deputy_head_teacher', 'academic_coordinator', 'class_teacher', 'subject_teacher',
    'librarian', 'transport_coordinator', 'admissions_officer', 'receptionist',
    'student', 'parent'
]::text[]));
