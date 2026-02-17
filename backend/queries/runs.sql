-- name: CreateRun :one
INSERT INTO runs (project_id, story_id, status, pipeline_config_snapshot)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRun :one
SELECT * FROM runs WHERE id = $1;

-- name: ListRunsByProject :many
SELECT * FROM runs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRunsByStory :many
SELECT * FROM runs
WHERE story_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountRunsByProject :one
SELECT COUNT(*) FROM runs WHERE project_id = $1;

-- name: CountRunsByStory :one
SELECT COUNT(*) FROM runs WHERE story_id = $1;

-- name: UpdateRunStatus :one
UPDATE runs
SET status = $2,
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    error_message = COALESCE(sqlc.narg('error_message'), error_message),
    updated_at = now()
WHERE id = $1
RETURNING *;
