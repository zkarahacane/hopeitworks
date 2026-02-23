DROP INDEX IF EXISTS idx_cost_records_agent_id;
ALTER TABLE cost_records DROP COLUMN IF EXISTS agent_id;
