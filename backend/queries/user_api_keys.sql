-- name: CreateUserAPIKey :one
INSERT INTO user_api_keys (id, user_id, provider, key_name, encrypted_key, key_hint, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
RETURNING *;

-- name: GetUserAPIKey :one
SELECT * FROM user_api_keys WHERE id = $1 LIMIT 1;

-- name: ListUserAPIKeys :many
SELECT * FROM user_api_keys WHERE user_id = $1 ORDER BY provider, key_name ASC;

-- name: GetUserAPIKeyByProvider :one
SELECT * FROM user_api_keys WHERE user_id = $1 AND provider = $2 LIMIT 1;

-- name: DeleteUserAPIKey :exec
DELETE FROM user_api_keys WHERE id = $1;
