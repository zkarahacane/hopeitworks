-- Remove rows with NULL project_id (global agents) before restoring NOT NULL
DELETE FROM agents WHERE project_id IS NULL;

-- Restore project_id NOT NULL constraint
ALTER TABLE agents ALTER COLUMN project_id SET NOT NULL;

-- Drop new columns
ALTER TABLE agents
    DROP COLUMN IF EXISTS scope,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS image;

-- Rename back
ALTER TABLE agents RENAME TO prompt_templates;

-- Restore index/constraint names
ALTER INDEX IF EXISTS idx_agents_project_id RENAME TO idx_prompt_templates_project_id;
ALTER TABLE prompt_templates RENAME CONSTRAINT agents_pkey TO prompt_templates_pkey;
ALTER TABLE prompt_templates RENAME CONSTRAINT agents_uq_project_name TO prompt_templates_uq_project_name;
ALTER TABLE prompt_templates RENAME CONSTRAINT agents_project_id_fkey TO prompt_templates_project_id_fkey;
