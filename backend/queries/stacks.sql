-- name: ListStacks :many
SELECT * FROM stacks ORDER BY key ASC;

-- name: GetStack :one
SELECT * FROM stacks WHERE id = $1 LIMIT 1;

-- name: GetStackByKey :one
SELECT * FROM stacks WHERE key = $1 LIMIT 1;

-- name: UpsertStack :one
INSERT INTO stacks (key, image_ref, toolchain) VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET image_ref = EXCLUDED.image_ref, toolchain = EXCLUDED.toolchain
RETURNING *;
