ALTER TABLE marks DROP CONSTRAINT IF EXISTS marks_score_nonnegative;
ALTER TABLE marks DROP COLUMN IF EXISTS competency_level;
DROP INDEX IF EXISTS idx_assessments_thread;
DROP INDEX IF EXISTS idx_assessments_class_term;
ALTER TABLE assessments DROP COLUMN IF EXISTS exam_thread_id;
ALTER TABLE marks DROP CONSTRAINT marks_student_id_fkey,
    ADD CONSTRAINT marks_student_id_fkey FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;
ALTER TABLE marks DROP CONSTRAINT marks_assessment_id_fkey,
    ADD CONSTRAINT marks_assessment_id_fkey FOREIGN KEY (assessment_id) REFERENCES assessments(id) ON DELETE CASCADE;
ALTER TABLE assessments DROP CONSTRAINT assessments_subject_id_fkey,
    ADD CONSTRAINT assessments_subject_id_fkey FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE CASCADE;
ALTER TABLE assessments DROP CONSTRAINT assessments_class_id_fkey,
    ADD CONSTRAINT assessments_class_id_fkey FOREIGN KEY (class_id) REFERENCES class_groups(id) ON DELETE CASCADE;
ALTER TABLE assessments DROP CONSTRAINT assessments_term_id_fkey,
    ADD CONSTRAINT assessments_term_id_fkey FOREIGN KEY (term_id) REFERENCES terms(id) ON DELETE CASCADE;
