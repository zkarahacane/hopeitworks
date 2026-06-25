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
-- cost_usd is the run's total cost aggregated over every cost record of its
-- steps. SUM without COALESCE so it is NULL when the run has no cost record yet
-- (distinct from a real $0.00); the subquery keeps this a single, N+1-free
-- query whose ORDER BY / LIMIT / OFFSET are unaffected.
SELECT r.id, r.project_id, r.story_id, r.status, r.pipeline_config_snapshot,
       r.started_at, r.completed_at, r.error_message, r.created_at, r.updated_at,
       r.paused_at, r.metadata, COALESCE(s.key, '') AS story_key,
       (
           SELECT SUM(cr.cost_usd)::DECIMAL(10,6)
           FROM cost_records cr
           JOIN run_steps rs ON rs.id = cr.run_step_id
           WHERE rs.run_id = r.id
       ) AS cost_usd
FROM runs r
LEFT JOIN stories s ON s.id = r.story_id
WHERE r.project_id = $1
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRunsByStory :many
-- cost_usd mirrors ListRunsByProjectWithStoryKey: SUM without COALESCE → NULL
-- when no cost record exists for the run, numeric (incl. 0) otherwise.
SELECT r.id, r.project_id, r.story_id, r.status, r.pipeline_config_snapshot,
       r.started_at, r.completed_at, r.error_message, r.created_at, r.updated_at,
       r.paused_at, r.metadata,
       (
           SELECT SUM(cr.cost_usd)::DECIMAL(10,6)
           FROM cost_records cr
           JOIN run_steps rs ON rs.id = cr.run_step_id
           WHERE rs.run_id = r.id
       ) AS cost_usd
FROM runs r
WHERE r.story_id = $1
ORDER BY r.created_at DESC
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

-- name: ListRunsByStatus :many
SELECT * FROM runs
WHERE status = $1
ORDER BY created_at ASC;

-- name: MarkRunOrphanedIfRunning :execrows
-- Conditionally fail a run only while it is still running. The status='running'
-- guard makes orphan reconciliation TOCTOU-safe: a run that transitioned to a
-- terminal state between the reconciler's snapshot and this write is left intact
-- (0 rows affected).
UPDATE runs
SET status = 'failed',
    completed_at = $2,
    error_message = $3,
    updated_at = now()
WHERE id = $1 AND status = 'running';

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
-- id, name, action_type, status, index (step_order) and container_id.
SELECT
    r.id AS run_id,
    r.status AS run_status,
    (
        SELECT to_jsonb(cs)
        FROM (
            SELECT s.id, s.step_name AS name, s.action AS action_type, s.status, s.step_order AS index, s.container_id
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
            SELECT s.id, s.step_name AS name, s.action AS action_type, s.status, s.step_order AS index, s.container_id
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

-- name: GetDAGNodeRunInfoByStories :many
-- Per-story enrichment for the epic DAG: the latest run id/status, the most
-- relevant container id of that run (the active step's container, else the most
-- recent step that has one), and the total cost incurred by that latest run.
-- One row per story that has at least one run; stories with no run are absent.
-- No status filter on cost so a failed run reports its real total.
SELECT DISTINCT ON (r.story_id)
    r.story_id              AS story_id,
    r.id                   AS run_id,
    r.status               AS run_status,
    (
        SELECT s.container_id
        FROM run_steps s
        WHERE s.run_id = r.id AND s.container_id IS NOT NULL
        ORDER BY (s.status IN ('running', 'waiting_approval')) DESC, s.step_order DESC
        LIMIT 1
    )                      AS container_id,
    (
        SELECT COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6)
        FROM cost_records cr
        JOIN run_steps rs ON rs.id = cr.run_step_id
        WHERE rs.run_id = r.id
    )                      AS cost_usd
FROM runs r
WHERE r.story_id = ANY($1::uuid[])
ORDER BY r.story_id, r.created_at DESC;
