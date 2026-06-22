-- name: CreateCredential :one
INSERT INTO credentials (name, scope, project_id, encrypted_value)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCredential :one
SELECT * FROM credentials WHERE id = $1 LIMIT 1;

-- name: GetGlobalCredentialByName :one
SELECT * FROM credentials
WHERE name = $1 AND scope = 'global'
LIMIT 1;

-- name: GetProjectCredentialByName :one
SELECT * FROM credentials
WHERE name = $1 AND scope = 'project' AND project_id = $2
LIMIT 1;

-- name: ListCredentialsByScope :many
SELECT * FROM credentials
WHERE scope = 'global' OR ($1::uuid IS NOT NULL AND project_id = $1::uuid)
ORDER BY scope, name;

-- name: DeleteCredential :exec
DELETE FROM credentials WHERE id = $1;
