-- Rename table
ALTER TABLE prompt_templates RENAME TO agents;

-- Add new columns
ALTER TABLE agents
    ADD COLUMN scope  VARCHAR(10)  NOT NULL DEFAULT 'project',
    ADD COLUMN model  VARCHAR(100),
    ADD COLUMN image  VARCHAR(255);

-- Make project_id nullable (global agents have no project)
ALTER TABLE agents ALTER COLUMN project_id DROP NOT NULL;

-- Rename constraints and indexes to match new table name
ALTER INDEX IF EXISTS idx_prompt_templates_project_id RENAME TO idx_agents_project_id;
ALTER TABLE agents RENAME CONSTRAINT prompt_templates_pkey TO agents_pkey;
ALTER TABLE agents RENAME CONSTRAINT prompt_templates_uq_project_name TO agents_uq_project_name;
ALTER TABLE agents RENAME CONSTRAINT prompt_templates_project_id_fkey TO agents_project_id_fkey;
