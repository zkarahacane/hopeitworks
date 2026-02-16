CREATE TABLE epics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    status      VARCHAR(50) NOT NULL DEFAULT 'backlog' CHECK (status IN ('backlog', 'in_progress', 'done')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_epics_project_id ON epics(project_id);
CREATE INDEX idx_epics_status ON epics(status);

CREATE TRIGGER set_epics_updated_at
    BEFORE UPDATE ON epics
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
