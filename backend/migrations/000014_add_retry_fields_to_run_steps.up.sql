ALTER TABLE run_steps
    ADD COLUMN retry_count      INT          NOT NULL DEFAULT 0,
    ADD COLUMN retry_type       VARCHAR(50)  NULL
        CHECK (retry_type IN ('incremental', 'full')),
    ADD COLUMN parent_step_id   UUID         NULL
        REFERENCES run_steps(id) ON DELETE RESTRICT;

CREATE INDEX idx_run_steps_parent_step_id ON run_steps(parent_step_id);
