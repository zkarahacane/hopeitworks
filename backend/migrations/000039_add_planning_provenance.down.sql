DROP INDEX IF EXISTS epics_uq_source_external;
DROP INDEX IF EXISTS stories_uq_source_external;

ALTER TABLE epics
  DROP COLUMN IF EXISTS synced_at,
  DROP COLUMN IF EXISTS source_url,
  DROP COLUMN IF EXISTS external_id,
  DROP COLUMN IF EXISTS source;

ALTER TABLE stories
  DROP COLUMN IF EXISTS last_import_hash,
  DROP COLUMN IF EXISTS synced_at,
  DROP COLUMN IF EXISTS source_url,
  DROP COLUMN IF EXISTS external_id,
  DROP COLUMN IF EXISTS source;
