-- name: CreateHITLRequest :one
INSERT INTO hitl_requests (id, run_step_id, gate_type, diff_content, message, status, halt_reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
RETURNING *;

-- name: GetHITLRequest :one
SELECT * FROM hitl_requests WHERE id = $1;

-- name: GetHITLRequestByRunStepID :one
SELECT * FROM hitl_requests WHERE run_step_id = $1 LIMIT 1;

-- name: UpdateHITLRequestStatus :one
UPDATE hitl_requests
SET status = $2,
    resolved_at = $3,
    resolved_by = $4,
    rejection_reason = $5,
    resolution_action = COALESCE(sqlc.narg('resolution_action'), resolution_action)
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

-- name: ListProbeHalts :many
-- Pending probe_halt gates across the platform (batch triage inbox). Enriched
-- with story/run context and the structured halt_reason so the UI can group by
-- reason and suggest a remedy. Ordered newest-first; project filter optional
-- (pass the nil UUID to list all projects).
SELECT
    hr.id,
    hr.run_step_id,
    hr.gate_type,
    hr.status,
    hr.halt_reason,
    hr.created_at,
    rs.run_id,
    rs.step_name,
    rs.stage_name,
    r.project_id,
    s.key   AS story_key,
    s.title AS story_title
FROM hitl_requests hr
JOIN run_steps rs ON rs.id = hr.run_step_id
JOIN runs r ON r.id = rs.run_id
JOIN stories s ON s.id = r.story_id
WHERE hr.gate_type = 'probe_halt'
  AND hr.status = 'pending'
  AND (sqlc.narg('project_id')::uuid IS NULL OR r.project_id = sqlc.narg('project_id')::uuid)
ORDER BY hr.created_at DESC;
