-- name: CreateEpic :one
INSERT INTO epics (project_id, name, description, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetEpic :one
SELECT * FROM epics WHERE id = $1;

-- name: ListEpicsByProject :many
SELECT * FROM epics WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountEpicsByProject :one
SELECT COUNT(*) FROM epics WHERE project_id = $1;

-- name: UpdateEpic :one
UPDATE epics
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteEpic :exec
DELETE FROM epics WHERE id = $1;
