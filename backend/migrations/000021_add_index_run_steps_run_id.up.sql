-- Add index on run_steps.run_id for HITL pending list query performance
-- The ListPendingHITLRequestsByProject query joins hitl_requests -> run_steps -> runs
-- Without this index, the join performs a sequential scan on run_steps
CREATE INDEX IF NOT EXISTS idx_run_steps_run_id ON run_steps(run_id);
