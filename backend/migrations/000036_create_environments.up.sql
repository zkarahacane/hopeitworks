-- P2c1 environments: a project's execution composition — which stacks, which
-- sidecar services, where config is derived from, and the commands to run. This is
-- the persistence layer only (additive): no API endpoint and no run-path wiring yet
-- (that lands in P2c2). An Environment is distinct from a Stack image: a Stack is a
-- base image, an Environment is how a project actually runs.
--
-- Product decision: exactly one Environment per project (UNIQUE project_id). The repo
-- exposes GetByProjectID returning a single row (NotFound when absent).
--
-- The `update_updated_at_column()` trigger function is shared (defined in migration
-- 000001) and is reused here, never recreated.
CREATE TABLE environments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stacks     TEXT[]      NOT NULL DEFAULT '{}',
    services   JSONB       NOT NULL DEFAULT '[]'::jsonb,
    source     VARCHAR(32) NOT NULL DEFAULT 'declared' CHECK (source IN ('devcontainer', 'compose', 'makefile', 'declared')),
    commands   JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT environments_uq_project UNIQUE (project_id)
);

CREATE INDEX idx_environments_project_id ON environments(project_id);

CREATE TRIGGER set_environments_updated_at
    BEFORE UPDATE ON environments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
