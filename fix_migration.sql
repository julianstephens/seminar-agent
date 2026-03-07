-- Manual migration to fix the failed 0006 migration
-- Run this if the migration hasn't been applied yet

-- Check if problem_set_id column exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='artifacts' AND column_name='problem_set_id'
    ) THEN
        -- Add unique constraint on assigned_from_session_id
        CREATE UNIQUE INDEX IF NOT EXISTS unique_problem_set_per_session 
            ON problem_sets (assigned_from_session_id) 
            WHERE assigned_from_session_id IS NOT NULL;

        -- Update status check constraint to include 'deleted' status
        ALTER TABLE problem_sets
            DROP CONSTRAINT IF EXISTS problem_sets_status_check;

        ALTER TABLE problem_sets
            ADD CONSTRAINT problem_sets_status_check
            CHECK (status IN ('assigned', 'submitted', 'reviewed', 'deleted'));

        -- Add problem_set_id column to artifacts table
        ALTER TABLE artifacts
            ADD COLUMN problem_set_id UUID REFERENCES problem_sets(id) ON DELETE SET NULL;

        -- Create index on problem_set_id
        CREATE INDEX idx_artifacts_problem_set ON artifacts(problem_set_id);

        -- Update artifacts kind check constraint
        ALTER TABLE artifacts
            DROP CONSTRAINT IF EXISTS artifacts_kind_check;

        ALTER TABLE artifacts
            ADD CONSTRAINT artifacts_kind_check
            CHECK (kind IN ('summary', 'notes', 'problem_set', 'diagnostic', 'problem_set_response'));

        RAISE NOTICE 'Migration applied successfully';
    ELSE
        RAISE NOTICE 'Migration already applied, skipping';
    END IF;
END $$;
