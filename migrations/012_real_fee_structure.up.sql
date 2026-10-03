-- Extends fee_structures with the two one-off Term-1 charges from the
-- school's real 2026 fee schedule (admission fee — new intake only — and
-- the diary & progress book, billed to everyone each year at Term 1), and
-- relaxes the grade range so Playgroup/PP1/PP2 can be represented alongside
-- Grade 1-13: Playgroup=-2, PP1=-1, PP2=0, Grade N=N.

ALTER TABLE fee_structures
    ADD COLUMN admission_fee numeric(14,2) NOT NULL DEFAULT 0,
    ADD COLUMN diary_book_fee numeric(14,2) NOT NULL DEFAULT 0;

ALTER TABLE fee_structures DROP CONSTRAINT fee_structures_grade_check;
ALTER TABLE fee_structures ADD CONSTRAINT fee_structures_grade_check CHECK (grade >= -2 AND grade <= 13);

ALTER TABLE class_groups DROP CONSTRAINT class_groups_grade_check;
ALTER TABLE class_groups ADD CONSTRAINT class_groups_grade_check CHECK (grade >= -2 AND grade <= 13);

ALTER TABLE grade_requirements DROP CONSTRAINT grade_requirements_grade_check;
ALTER TABLE grade_requirements ADD CONSTRAINT grade_requirements_grade_check CHECK (grade >= -2 AND grade <= 13);
