-- name: CreateHITLRequest :one
INSERT INTO hitl_requests (id, run_step_id, gate_type, diff_content, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetHITLRequestByRunStepID :one
SELECT * FROM hitl_requests WHERE run_step_id = $1 LIMIT 1;

-- name: GetPendingHITLRequestByRunID :one
SELECT hr.*
FROM hitl_requests hr
JOIN run_steps rs ON rs.id = hr.run_step_id
WHERE rs.run_id = $1
  AND hr.status = 'pending'
LIMIT 1;

-- name: UpdateHITLRequestStatus :one
UPDATE hitl_requests
SET status = $2, resolved_at = $3, resolved_by = $4, rejection_reason = $5
WHERE id = $1
RETURNING *;
