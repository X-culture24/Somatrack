ALTER TABLE grade_requirements DROP CONSTRAINT grade_requirements_grade_check;
ALTER TABLE grade_requirements ADD CONSTRAINT grade_requirements_grade_check CHECK (grade >= 1 AND grade <= 13);

ALTER TABLE class_groups DROP CONSTRAINT class_groups_grade_check;
ALTER TABLE class_groups ADD CONSTRAINT class_groups_grade_check CHECK (grade >= 1 AND grade <= 13);

ALTER TABLE fee_structures DROP CONSTRAINT fee_structures_grade_check;
ALTER TABLE fee_structures ADD CONSTRAINT fee_structures_grade_check CHECK (grade >= 1 AND grade <= 13);

ALTER TABLE fee_structures
    DROP COLUMN admission_fee,
    DROP COLUMN diary_book_fee;
