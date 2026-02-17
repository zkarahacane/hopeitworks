DROP TRIGGER IF EXISTS events_notify_trigger ON events;
DROP TRIGGER IF EXISTS events_no_delete ON events;
DROP TRIGGER IF EXISTS events_no_update ON events;
DROP FUNCTION IF EXISTS notify_event();
DROP FUNCTION IF EXISTS prevent_event_modification();
DROP TABLE IF EXISTS events;
