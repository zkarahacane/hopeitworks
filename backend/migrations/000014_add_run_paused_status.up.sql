-- Add 'paused' to the runs status CHECK constraint
ALTER TABLE runs DROP CONSTRAINT IF EXISTS runs_status_check;
ALTER TABLE runs ADD CONSTRAINT runs_status_check
    CHECK (status IN ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled'));

-- Add paused_at column to track when a run was last paused
ALTER TABLE runs ADD COLUMN paused_at TIMESTAMPTZ;
