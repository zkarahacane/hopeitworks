DROP INDEX IF EXISTS idx_run_steps_parent_step_id;
ALTER TABLE run_steps
    DROP COLUMN IF EXISTS parent_step_id,
    DROP COLUMN IF EXISTS retry_type,
    DROP COLUMN IF EXISTS retry_count;
