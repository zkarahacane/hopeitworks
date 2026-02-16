CREATE TABLE projects (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(255) NOT NULL UNIQUE,
    description    TEXT,
    owner_id       UUID,
    repo_url       TEXT,
    git_provider   VARCHAR(50) NOT NULL DEFAULT 'github',
    git_token_env  VARCHAR(255),
    agent_runtime  VARCHAR(50) NOT NULL DEFAULT 'docker',
    default_model  VARCHAR(100),
    max_budget     NUMERIC(10,2),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_projects_created_at ON projects(created_at DESC);
