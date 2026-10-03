ALTER TABLE admission_applications DROP CONSTRAINT admission_applications_applying_for_grade_check;
ALTER TABLE admission_applications ADD CONSTRAINT admission_applications_applying_for_grade_check
    CHECK (applying_for_grade >= 1 AND applying_for_grade <= 13);
