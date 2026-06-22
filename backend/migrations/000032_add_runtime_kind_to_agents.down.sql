ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_chk_runtime_kind;
ALTER TABLE agents DROP COLUMN IF EXISTS runtime_kind;
