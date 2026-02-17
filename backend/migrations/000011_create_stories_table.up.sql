CREATE TABLE stories (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    epic_id             UUID REFERENCES epics(id) ON DELETE SET NULL,
    key                 VARCHAR(50) NOT NULL,
    title               VARCHAR(255) NOT NULL,
    objective           TEXT,
    target_files        JSONB,
    depends_on          JSONB,
    scope               VARCHAR(50),
    status              VARCHAR(50) NOT NULL DEFAULT 'backlog',
    acceptance_criteria TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stories_uq_project_key UNIQUE (project_id, key)
);

CREATE INDEX idx_stories_project_id ON stories(project_id);
CREATE INDEX idx_stories_epic_id ON stories(epic_id);
CREATE INDEX idx_stories_status ON stories(status);
