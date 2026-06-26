-- name: CreateEpic :one
INSERT INTO epics (project_id, name, description, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetEpic :one
SELECT * FROM epics WHERE id = $1;

-- name: ListEpicsByProject :many
SELECT * FROM epics
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

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

-- name: GetEpicBySourceRef :one
-- Resolves an epic by its stable provenance identity (project, source, external_id).
SELECT * FROM epics
WHERE project_id = @project_id AND source = @source AND external_id = @external_id
LIMIT 1;

-- name: GetEpicByName :one
-- Name lookup backing source-guarded adoption: a markdown/github epic attaches to
-- an existing same-name epic instead of tripping epics_uq_project_name.
SELECT * FROM epics
WHERE project_id = @project_id AND name = @name
LIMIT 1;

-- name: CreateEpicFromImport :one
INSERT INTO epics (project_id, name, description, status, source, external_id, source_url, synced_at)
VALUES (@project_id, @name, @description, @status, @source, @external_id, @source_url, now())
RETURNING *;

-- name: UpdateEpicFromImport :one
-- The SERVICE has merged every value (preserve-on-absent description, promote-only
-- status, source-guarded provenance).
UPDATE epics SET
    name        = @name,
    description = @description,
    status      = @status,
    source      = @source,
    external_id = @external_id,
    source_url  = @source_url,
    synced_at   = now(),
    updated_at  = now()
WHERE id = @id
RETURNING *;
