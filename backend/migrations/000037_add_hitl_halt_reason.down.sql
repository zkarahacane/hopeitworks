DROP INDEX IF EXISTS idx_hitl_requests_gate_type;
ALTER TABLE hitl_requests DROP COLUMN IF EXISTS resolution_action;
ALTER TABLE hitl_requests DROP COLUMN IF EXISTS halt_reason;
