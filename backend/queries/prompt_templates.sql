-- name: CreatePromptTemplate :one
INSERT INTO prompt_templates (project_id, name, template_content, type)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPromptTemplate :one
SELECT * FROM prompt_templates WHERE id = $1;

-- name: GetPromptTemplateByProjectAndName :one
SELECT * FROM prompt_templates
WHERE project_id = $1 AND name = $2;

-- name: ListPromptTemplatesByProject :many
SELECT * FROM prompt_templates
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPromptTemplatesByProject :one
SELECT COUNT(*) FROM prompt_templates WHERE project_id = $1;

-- name: UpdatePromptTemplate :one
UPDATE prompt_templates
SET name = COALESCE(sqlc.narg('name'), name),
    template_content = COALESCE(sqlc.narg('template_content'), template_content),
    type = COALESCE(sqlc.narg('type'), type),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeletePromptTemplate :exec
DELETE FROM prompt_templates WHERE id = $1;
