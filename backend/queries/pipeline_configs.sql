-- name: GetPipelineConfig :one
SELECT * FROM pipeline_configs WHERE project_id = $1;

-- name: UpsertPipelineConfig :one
INSERT INTO pipeline_configs (project_id, config_yaml, version)
VALUES ($1, $2, 1)
ON CONFLICT (project_id) DO UPDATE
SET config_yaml = EXCLUDED.config_yaml,
    version = pipeline_configs.version + 1,
    updated_at = now()
RETURNING *;
