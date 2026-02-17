-- name: CreateStory :one
INSERT INTO stories (project_id, epic_id, key, title, objective, target_files, depends_on, scope, status, acceptance_criteria)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetStory :one
SELECT * FROM stories WHERE id = $1;

-- name: GetStoryByKey :one
SELECT * FROM stories WHERE project_id = $1 AND key = $2;

-- name: ListStoriesByProject :many
SELECT * FROM stories
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListStoriesByStatus :many
SELECT * FROM stories
WHERE project_id = $1 AND status = ANY($2::text[])
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListStoriesByEpic :many
SELECT * FROM stories
WHERE epic_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoriesByProject :one
SELECT COUNT(*) FROM stories WHERE project_id = $1;

-- name: CountStoriesByStatus :one
SELECT COUNT(*) FROM stories WHERE project_id = $1 AND status = ANY($2::text[]);

-- name: CountStoriesByEpic :one
SELECT COUNT(*) FROM stories WHERE epic_id = $1;

-- name: UpdateStory :one
UPDATE stories
SET title = COALESCE(sqlc.narg('title'), title),
    objective = COALESCE(sqlc.narg('objective'), objective),
    target_files = COALESCE(sqlc.narg('target_files'), target_files),
    depends_on = COALESCE(sqlc.narg('depends_on'), depends_on),
    scope = COALESCE(sqlc.narg('scope'), scope),
    status = COALESCE(sqlc.narg('status'), status),
    acceptance_criteria = COALESCE(sqlc.narg('acceptance_criteria'), acceptance_criteria),
    epic_id = COALESCE(sqlc.narg('epic_id'), epic_id),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteStory :exec
DELETE FROM stories WHERE id = $1;
