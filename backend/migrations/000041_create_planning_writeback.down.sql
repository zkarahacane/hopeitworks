ALTER TABLE stories
  DROP COLUMN IF EXISTS writeback_status,
  DROP COLUMN IF EXISTS external_item_id;

DROP TABLE IF EXISTS planning_write_backs;
DROP TABLE IF EXISTS planning_connectors;
