-- admission_applications.applying_for_grade must use the same grade numbering
-- as fee_structures/class_groups (Playgroup=-2, PP1=-1, PP2=0, Grade N=N) so
-- an admitted application's grade directly matches the fee structure it
-- should be billed against — the frontend previously worked around the old
-- 1-13-only range with its own incompatible offset mapping.
ALTER TABLE admission_applications DROP CONSTRAINT admission_applications_applying_for_grade_check;
ALTER TABLE admission_applications ADD CONSTRAINT admission_applications_applying_for_grade_check
    CHECK (applying_for_grade >= -2 AND applying_for_grade <= 13);
