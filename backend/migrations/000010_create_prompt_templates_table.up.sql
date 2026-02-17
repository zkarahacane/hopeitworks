CREATE TABLE prompt_templates (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    template_content TEXT NOT NULL,
    type             VARCHAR(50) NOT NULL CHECK (type IN ('implement', 'retry', 'review', 'merge', 'custom')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT prompt_templates_uq_project_name UNIQUE (project_id, name)
);

CREATE INDEX idx_prompt_templates_project_id ON prompt_templates(project_id);
