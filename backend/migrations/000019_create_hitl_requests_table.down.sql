DROP TABLE IF EXISTS hitl_requests;

-- Restore original run_steps status constraint
ALTER TABLE run_steps DROP CONSTRAINT IF EXISTS run_steps_status_check;
ALTER TABLE run_steps ADD CONSTRAINT run_steps_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));
