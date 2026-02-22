-- name: CreateRun :one
INSERT INTO runs (project_id, story_id, status, pipeline_config_snapshot, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRun :one
SELECT * FROM runs WHERE id = $1;

-- name: GetRunWithStoryKey :one
SELECT r.id, r.project_id, r.story_id, r.status, r.pipeline_config_snapshot,
       r.started_at, r.completed_at, r.error_message, r.created_at, r.updated_at,
       r.paused_at, r.metadata, COALESCE(s.key, '') AS story_key
FROM runs r
LEFT JOIN stories s ON s.id = r.story_id
WHERE r.id = $1;

-- name: ListRunsByProject :many
SELECT * FROM runs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRunsByProjectWithStoryKey :many
SELECT r.id, r.project_id, r.story_id, r.status, r.pipeline_config_snapshot,
       r.started_at, r.completed_at, r.error_message, r.created_at, r.updated_at,
       r.paused_at, r.metadata, COALESCE(s.key, '') AS story_key
FROM runs r
LEFT JOIN stories s ON s.id = r.story_id
WHERE r.project_id = $1
ORDER BY r.created_at DESC
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

-- name: GetActiveRunByStory :one
SELECT * FROM runs
WHERE story_id = $1 AND status IN ('pending', 'running', 'paused')
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateRunStatus :one
UPDATE runs
SET status = $2,
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    paused_at = COALESCE(sqlc.narg('paused_at'), paused_at),
    error_message = COALESCE(sqlc.narg('error_message'), error_message),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListChildRunsByParent :many
SELECT * FROM runs
WHERE project_id = $1 AND pipeline_config_snapshot @> sqlc.arg('parent_filter')::jsonb
ORDER BY created_at ASC;
