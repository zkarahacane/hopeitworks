-- name: CreateAgent :one
INSERT INTO agents (id, name, model, image, template_content, type, scope, provider, project_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
RETURNING *;

-- name: GetAgent :one
SELECT * FROM agents WHERE id = $1 LIMIT 1;

-- name: ListAgentsByProject :many
SELECT * FROM agents WHERE project_id = $1 ORDER BY name ASC;

-- name: ListGlobalAgents :many
SELECT * FROM agents WHERE scope = 'global' ORDER BY name ASC;

-- name: ListAgentsByProjectMerged :many
SELECT * FROM agents
WHERE project_id = $1 OR scope = 'global'
ORDER BY scope DESC, name ASC;

-- name: UpdateAgent :one
UPDATE agents
SET name = $2, model = $3, image = $4, template_content = $5, provider = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAgent :exec
DELETE FROM agents WHERE id = $1;
