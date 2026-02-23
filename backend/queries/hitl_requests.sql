-- name: CreateHITLRequest :one
INSERT INTO hitl_requests (id, run_step_id, gate_type, diff_content, message, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
RETURNING *;

-- name: GetHITLRequest :one
SELECT * FROM hitl_requests WHERE id = $1;

-- name: GetHITLRequestByRunStepID :one
SELECT * FROM hitl_requests WHERE run_step_id = $1 LIMIT 1;

-- name: UpdateHITLRequestStatus :one
UPDATE hitl_requests
SET status = $2, resolved_at = $3, resolved_by = $4, rejection_reason = $5
WHERE id = $1
RETURNING *;

-- name: ListPendingHITLRequestsByProject :many
SELECT
    hr.id,
    rs.run_id,
    rs.id AS step_id,
    s.key AS story_key,
    hr.diff_url,
    hr.created_at
FROM hitl_requests hr
JOIN run_steps rs ON rs.id = hr.run_step_id
JOIN runs r ON r.id = rs.run_id
JOIN stories s ON s.id = r.story_id
WHERE r.project_id = $1
  AND hr.status = 'pending'
ORDER BY hr.created_at DESC;

-- name: CountPendingHITLRequestsByProject :one
SELECT COUNT(*)
FROM hitl_requests hr
JOIN run_steps rs ON rs.id = hr.run_step_id
JOIN runs r ON r.id = rs.run_id
WHERE r.project_id = $1
  AND hr.status = 'pending';

-- name: ListHITLRequestsFiltered :many
SELECT * FROM hitl_requests
WHERE ($1::text = '' OR status = $1::text)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountHITLRequestsFiltered :one
SELECT COUNT(*) FROM hitl_requests
WHERE ($1::text = '' OR status = $1::text);
