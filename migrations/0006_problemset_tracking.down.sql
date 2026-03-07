BEGIN;

-- Remove problem_set_response from artifacts kind constraint
ALTER TABLE artifacts
    DROP CONSTRAINT artifacts_kind_check;

ALTER TABLE artifacts
    ADD CONSTRAINT artifacts_kind_check
    CHECK (kind IN ('summary', 'notes', 'problem_set', 'diagnostic'));

-- Drop index on artifacts.problem_set_id
DROP INDEX IF EXISTS idx_artifacts_problem_set;

-- Drop problem_set_id column from artifacts
ALTER TABLE artifacts
    DROP COLUMN IF EXISTS problem_set_id;

-- Remove deleted status from problem_sets status constraint
ALTER TABLE problem_sets
    DROP CONSTRAINT problem_sets_status_check;

ALTER TABLE problem_sets
    ADD CONSTRAINT problem_sets_status_check
    CHECK (status IN ('assigned', 'submitted', 'reviewed'));

-- Remove unique constraint on assigned_from_session_id
DROP INDEX IF EXISTS unique_problem_set_per_session;

COMMIT;
