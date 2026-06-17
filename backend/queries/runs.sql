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

-- name: UpdateRunMetadata :exec
UPDATE runs SET metadata = $2, updated_at = now() WHERE id = $1;

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

-- name: GetLatestRunByStory :one
-- Returns the most recent run for a story along with its current in-progress step
-- (running or waiting_approval, lowest step_order) and the total step count.
-- current_step is a JSON object (NULL when no step is in progress) carrying
-- id, name, action_type, status and index (step_order).
SELECT
    r.id AS run_id,
    r.status AS run_status,
    (
        SELECT to_jsonb(cs)
        FROM (
            SELECT s.id, s.step_name AS name, s.action AS action_type, s.status, s.step_order AS index
            FROM run_steps s
            WHERE s.run_id = r.id AND s.status IN ('running', 'waiting_approval')
            ORDER BY s.step_order ASC
            LIMIT 1
        ) cs
    ) AS current_step,
    (SELECT COUNT(*) FROM run_steps s WHERE s.run_id = r.id)::int AS total_steps
FROM runs r
WHERE r.story_id = $1
ORDER BY r.created_at DESC
LIMIT 1;

-- name: GetLatestRunsByStories :many
-- Batch version of GetLatestRunByStory: returns the most recent run per story
-- for the given story IDs, with current step (JSON) and total step count.
-- Avoids N+1 when listing stories of an epic.
SELECT DISTINCT ON (r.story_id)
    r.story_id AS story_id,
    r.id AS run_id,
    r.status AS run_status,
    (
        SELECT to_jsonb(cs)
        FROM (
            SELECT s.id, s.step_name AS name, s.action AS action_type, s.status, s.step_order AS index
            FROM run_steps s
            WHERE s.run_id = r.id AND s.status IN ('running', 'waiting_approval')
            ORDER BY s.step_order ASC
            LIMIT 1
        ) cs
    ) AS current_step,
    (SELECT COUNT(*) FROM run_steps s WHERE s.run_id = r.id)::int AS total_steps
FROM runs r
WHERE r.story_id = ANY($1::uuid[])
ORDER BY r.story_id, r.created_at DESC;
