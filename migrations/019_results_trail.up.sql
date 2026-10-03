-- Results must outlive terms, classes and subjects: the results trail shows
-- every term a student has sat. Previously deleting a term, class, subject
-- or student silently CASCADE-deleted all their assessments and marks.
ALTER TABLE assessments DROP CONSTRAINT assessments_term_id_fkey,
    ADD CONSTRAINT assessments_term_id_fkey FOREIGN KEY (term_id) REFERENCES terms(id) ON DELETE RESTRICT;
ALTER TABLE assessments DROP CONSTRAINT assessments_class_id_fkey,
    ADD CONSTRAINT assessments_class_id_fkey FOREIGN KEY (class_id) REFERENCES class_groups(id) ON DELETE RESTRICT;
ALTER TABLE assessments DROP CONSTRAINT assessments_subject_id_fkey,
    ADD CONSTRAINT assessments_subject_id_fkey FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE RESTRICT;
ALTER TABLE marks DROP CONSTRAINT marks_assessment_id_fkey,
    ADD CONSTRAINT marks_assessment_id_fkey FOREIGN KEY (assessment_id) REFERENCES assessments(id) ON DELETE RESTRICT;
ALTER TABLE marks DROP CONSTRAINT marks_student_id_fkey,
    ADD CONSTRAINT marks_student_id_fkey FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE RESTRICT;

-- Exams belong to an exam thread (Term 3 2026 Assessments …), which carries
-- the CATs / main-exam weighting and the lock.
ALTER TABLE assessments ADD COLUMN IF NOT EXISTS exam_thread_id UUID REFERENCES exam_threads(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_assessments_class_term ON assessments(class_id, term_id);
CREATE INDEX IF NOT EXISTS idx_assessments_thread ON assessments(exam_thread_id);

-- CBC reports carry a competency level alongside (or instead of) a score.
ALTER TABLE marks ADD COLUMN IF NOT EXISTS competency_level VARCHAR(2) NOT NULL DEFAULT ''
    CHECK (competency_level IN ('', 'EE', 'ME', 'AE', 'BE'));
ALTER TABLE marks ADD CONSTRAINT marks_score_nonnegative CHECK (score >= 0);
