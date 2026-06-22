DROP INDEX IF EXISTS idx_agents_stack_id;
ALTER TABLE agents DROP COLUMN IF EXISTS stack_id;
