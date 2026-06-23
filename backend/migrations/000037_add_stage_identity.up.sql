-- INC 1: Stage identity end-to-end.
-- Each run_step carries its originating stage (the pipeline group it belongs to),
-- and each story tracks its current_stage. These columns are the durable record of
-- "which stage is this card in" — the source of truth the board projects from.
--
-- All additive and nullable: existing runs without stage info keep working; the
-- executor stamps stage info on new steps and advances stories.current_stage.

-- run_steps gain the stage they belong to. stage_id mirrors the PipelineGroup.ID,
-- stage_name mirrors PipelineGroup.Name (human-meaningful column label).
ALTER TABLE run_steps ADD COLUMN stage_id VARCHAR(255);
ALTER TABLE run_steps ADD COLUMN stage_name VARCHAR(255);

-- stories gain current_stage: the name of the stage the card currently sits in.
-- Nullable: NULL means "no stage" (backlog before first run, or after completion).
ALTER TABLE stories ADD COLUMN current_stage VARCHAR(255);

-- Backfill existing run_steps so they all carry a stage identity. Pre-INC-1 steps
-- were flattened without group identity; attribute them to the implicit "Default"
-- stage that ParsePipelineConfigYAML wraps legacy flat configs into.
UPDATE run_steps SET stage_id = 'default', stage_name = 'Default' WHERE stage_id IS NULL;
