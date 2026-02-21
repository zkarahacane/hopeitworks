-- name: CreateEvent :one
INSERT INTO events (id, project_id, entity_type, entity_id, action, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListEventsByProject :many
SELECT * FROM events
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetEventsByEntityID :many
SELECT * FROM events
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at ASC;

-- name: GetEventsSince :many
SELECT e.*
FROM events e
WHERE e.project_id = $1
  AND e.created_at > (
      SELECT anchor.created_at FROM events anchor WHERE anchor.id = $2
  )
ORDER BY e.created_at ASC;
