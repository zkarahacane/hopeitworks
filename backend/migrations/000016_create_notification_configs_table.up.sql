CREATE TABLE notification_configs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    channel_type  VARCHAR NOT NULL CHECK (channel_type IN ('discord', 'webhook')),
    config        JSONB NOT NULL DEFAULT '{}',
    events_filter JSONB NOT NULL DEFAULT '[]',
    enabled       BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_configs_project_enabled
    ON notification_configs(project_id, enabled);
