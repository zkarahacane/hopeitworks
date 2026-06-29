-- Planning write-back (one-way outbound: hopeitworks -> tracker). The existing
-- import is read-only; this adds the persisted connector config + an append-only
-- audit of every status push, plus two story columns.
--
-- planning_connectors: one row per project. status_field/done_options/epic_issue_type
-- consolidate the previously request-only import knobs so the connector is the single
-- source of truth; status_mapping (JSONB {backlog,running,done,failed -> option id})
-- + writeback_enabled/post_run_comment drive the outbound push.
CREATE TABLE planning_connectors (
    project_id        UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    source            VARCHAR(20)  NOT NULL,
    project_url       TEXT,
    status_field      VARCHAR(255) NOT NULL DEFAULT 'Status',
    done_options      JSONB        NOT NULL DEFAULT '[]',
    epic_issue_type   VARCHAR(255) NOT NULL DEFAULT 'Epic',
    status_mapping    JSONB        NOT NULL DEFAULT '{}',
    writeback_enabled BOOLEAN      NOT NULL DEFAULT false,
    post_run_comment  BOOLEAN      NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- planning_write_backs: append-only audit of every write-back attempt (one row per
-- transition push), success or failure. No FK on story_id/run_id so audit survives
-- a later story/run delete.
CREATE TABLE planning_write_backs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID         NOT NULL,
    story_id        UUID         NOT NULL,
    run_id          UUID,
    source          VARCHAR(20),
    external_id     TEXT,
    internal_status VARCHAR(50),
    remote_status   VARCHAR(255),
    success         BOOLEAN      NOT NULL,
    error_code      VARCHAR(64),
    error_message   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_planning_write_backs_story ON planning_write_backs (story_id, created_at DESC);

-- external_item_id is the ProjectV2Item id (distinct from the content node id stored
-- in external_id) -- it is the target of the field-value mutation, so it must be
-- captured at import. writeback_status mirrors the last outbound push state.
ALTER TABLE stories
  ADD COLUMN external_item_id TEXT,
  ADD COLUMN writeback_status VARCHAR(20);
