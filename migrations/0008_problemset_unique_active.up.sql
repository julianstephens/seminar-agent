-- Fix the unique index on problem_sets.assigned_from_session_id to exclude deleted records,
-- allowing a new problem set to be created for a session after the previous one is deleted.
DROP INDEX IF EXISTS unique_problem_set_per_session;

CREATE UNIQUE INDEX unique_problem_set_per_session
    ON problem_sets (assigned_from_session_id)
    WHERE assigned_from_session_id IS NOT NULL AND status != 'deleted';
