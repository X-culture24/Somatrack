ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role::text = ANY (ARRAY[
    'system_admin', 'deputy_head', 'head_teacher', 'academic_coordinator',
    'finance_officer', 'bursar', 'class_teacher', 'subject_teacher', 'librarian',
    'nurse', 'storekeeper', 'transport_officer', 'student', 'parent'
]::text[]));
