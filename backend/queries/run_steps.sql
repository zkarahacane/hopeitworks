-- name: CreateRunStep :one
INSERT INTO run_steps (run_id, step_name, step_order, action, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRunStep :one
SELECT * FROM run_steps WHERE id = $1;

-- name: ListRunStepsByRun :many
SELECT * FROM run_steps
WHERE run_id = $1
ORDER BY step_order ASC;

-- name: UpdateRunStepStatus :one
UPDATE run_steps
SET status = $2,
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    error_message = COALESCE(sqlc.narg('error_message'), error_message),
    container_id = COALESCE(sqlc.narg('container_id'), container_id),
    log_tail = COALESCE(sqlc.narg('log_tail'), log_tail)
WHERE id = $1
RETURNING *;

-- name: UpdateRunStepContainerInfo :one
UPDATE run_steps
SET container_id = COALESCE(sqlc.narg('container_id'), container_id),
    log_tail = COALESCE(sqlc.narg('log_tail'), log_tail)
WHERE id = $1
RETURNING *;

-- name: CreateRetryRunStep :one
INSERT INTO run_steps (
    id, run_id, step_name, step_order, action, status,
    retry_count, retry_type, parent_step_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: ListRetryStepsByParent :many
SELECT * FROM run_steps
WHERE parent_step_id = $1
ORDER BY retry_count ASC;

-- name: AppendRunStepLogTail :exec
UPDATE run_steps
SET log_tail = right(coalesce(log_tail, '') || $2, 16000)
WHERE id = $1;
