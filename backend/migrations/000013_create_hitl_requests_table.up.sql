CREATE TABLE hitl_requests (
    id              UUID PRIMARY KEY,
    run_step_id     UUID NOT NULL REFERENCES run_steps(id) ON DELETE CASCADE,
    gate_type       VARCHAR(50) NOT NULL DEFAULT 'approval',
    diff_content    TEXT,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    resolved_at     TIMESTAMPTZ,
    resolved_by     UUID REFERENCES users(id),
    rejection_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hitl_requests_run_step_id ON hitl_requests(run_step_id);
CREATE INDEX idx_hitl_requests_status ON hitl_requests(status);
