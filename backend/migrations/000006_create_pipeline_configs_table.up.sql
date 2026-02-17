CREATE TABLE pipeline_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID UNIQUE NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    config_yaml TEXT NOT NULL,
    version     INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipeline_configs_project_id ON pipeline_configs(project_id);
