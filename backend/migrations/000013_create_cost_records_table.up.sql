CREATE TABLE cost_records (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_step_id   UUID NOT NULL REFERENCES run_steps(id) ON DELETE CASCADE,
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tokens_input  BIGINT NOT NULL DEFAULT 0,
    tokens_output BIGINT NOT NULL DEFAULT 0,
    cost_usd      DECIMAL(10,6) NOT NULL DEFAULT 0,
    model         VARCHAR(100) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cost_records_run_step_id ON cost_records(run_step_id);
CREATE INDEX idx_cost_records_project_created ON cost_records(project_id, created_at DESC);
