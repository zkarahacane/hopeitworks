-- name: ListStacks :many
SELECT * FROM stacks ORDER BY key ASC;

-- name: GetStack :one
SELECT * FROM stacks WHERE id = $1 LIMIT 1;

-- name: GetStackByKey :one
SELECT * FROM stacks WHERE key = $1 LIMIT 1;
