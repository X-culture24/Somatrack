-- Ensure only one cat1/cat2/cat3 per (class, subject, term) combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_assessments_class_subject_term_type
    ON assessments (class_id, subject_id, term_id, assessment_type)
    WHERE assessment_type IN ('cat1','cat2','cat3');
