-- Revert INC 1 stage identity columns.
ALTER TABLE stories DROP COLUMN current_stage;
ALTER TABLE run_steps DROP COLUMN stage_name;
ALTER TABLE run_steps DROP COLUMN stage_id;
