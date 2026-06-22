-- name: CreateEnvironment :one
INSERT INTO environments (project_id, stacks, services, source, commands)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetEnvironment :one
SELECT * FROM environments WHERE id = $1 LIMIT 1;

-- name: GetEnvironmentByProjectID :one
SELECT * FROM environments WHERE project_id = $1 LIMIT 1;

-- name: UpdateEnvironment :one
UPDATE environments
SET stacks = $2, services = $3, source = $4, commands = $5
WHERE id = $1
RETURNING *;

-- name: DeleteEnvironment :exec
DELETE FROM environments WHERE id = $1;
