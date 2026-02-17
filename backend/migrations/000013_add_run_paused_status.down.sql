-- Remove paused_at column
ALTER TABLE runs DROP COLUMN IF EXISTS paused_at;

-- Restore original status CHECK constraint without 'paused'
ALTER TABLE runs DROP CONSTRAINT IF EXISTS runs_status_check;
ALTER TABLE runs ADD CONSTRAINT runs_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));
