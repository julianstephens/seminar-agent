-- Revert: restore original unique index without the status filter
DROP INDEX IF EXISTS unique_problem_set_per_session;

CREATE UNIQUE INDEX unique_problem_set_per_session
    ON problem_sets (assigned_from_session_id)
    WHERE assigned_from_session_id IS NOT NULL;
