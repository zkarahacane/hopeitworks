-- name: CreateProject :one
INSERT INTO projects (name, description, owner_id, repo_url, git_provider, git_token_env, agent_runtime, default_model, max_budget)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountProjects :one
SELECT COUNT(*) FROM projects;

-- name: UpdateProject :one
UPDATE projects
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    owner_id = COALESCE(sqlc.narg('owner_id'), owner_id),
    repo_url = COALESCE(sqlc.narg('repo_url'), repo_url),
    git_provider = COALESCE(sqlc.narg('git_provider'), git_provider),
    git_token_env = COALESCE(sqlc.narg('git_token_env'), git_token_env),
    agent_runtime = COALESCE(sqlc.narg('agent_runtime'), agent_runtime),
    default_model = COALESCE(sqlc.narg('default_model'), default_model),
    max_budget = COALESCE(sqlc.narg('max_budget'), max_budget),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;

-- name: IncrementCircuitBreakerCount :one
UPDATE projects
SET circuit_breaker_count = circuit_breaker_count + 1,
    circuit_breaker_active = CASE
        WHEN circuit_breaker_count + 1 >= circuit_breaker_max THEN true
        ELSE circuit_breaker_active
    END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ResetCircuitBreaker :one
UPDATE projects
SET circuit_breaker_count = 0,
    circuit_breaker_active = false,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetCircuitBreakerState :one
SELECT id, circuit_breaker_count, circuit_breaker_active, circuit_breaker_max
FROM projects
WHERE id = $1;
