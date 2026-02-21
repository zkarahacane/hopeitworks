CREATE TYPE epic_run_status AS ENUM ('pending', 'running', 'completed', 'failed', 'paused');

CREATE TABLE epic_runs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    epic_id      UUID NOT NULL REFERENCES epics(id) ON DELETE CASCADE,
    status       epic_run_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_epic_runs_project_id ON epic_runs(project_id);
CREATE INDEX idx_epic_runs_epic_id ON epic_runs(epic_id);
CREATE INDEX idx_epic_runs_status ON epic_runs(status);
