CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns
CREATE INDEX idx_events_project_id_created_at ON events(project_id, created_at);
CREATE INDEX idx_events_entity_type_entity_id ON events(entity_type, entity_id);

-- Append-only: prevent UPDATE and DELETE on events table
CREATE OR REPLACE FUNCTION prevent_event_modification() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'events table is append-only: % operations are not allowed', TG_OP;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER events_no_update
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION prevent_event_modification();

CREATE TRIGGER events_no_delete
    BEFORE DELETE ON events
    FOR EACH ROW EXECUTE FUNCTION prevent_event_modification();

-- Trigger function to broadcast events via NOTIFY
CREATE OR REPLACE FUNCTION notify_event() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('events', json_build_object(
        'id', NEW.id,
        'project_id', NEW.project_id,
        'entity_type', NEW.entity_type,
        'entity_id', NEW.entity_id,
        'action', NEW.action
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to events table
CREATE TRIGGER events_notify_trigger
    AFTER INSERT ON events
    FOR EACH ROW EXECUTE FUNCTION notify_event();
