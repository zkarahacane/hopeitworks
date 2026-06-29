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

-- name: CountStoriesByEpicGroupedByStatus :many
SELECT status, COUNT(*) AS count
FROM stories
WHERE epic_id = $1
GROUP BY status;

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

-- name: UpdateStoryCurrentStage :one
UPDATE stories
SET current_stage = sqlc.narg('current_stage'),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteStory :exec
DELETE FROM stories WHERE id = $1;

-- name: GetStoryBySourceRef :one
-- Resolves a remote-sourced story by its stable provenance identity. Used by the
-- planning importer for non-author-owned keys (e.g. github_projects); markdown
-- resolution stays on GetStoryByKey.
SELECT * FROM stories
WHERE project_id = @project_id AND source = @source AND external_id = @external_id
LIMIT 1;

-- name: CreateStoryFromImport :one
-- Import-create: the SERVICE has computed every value (status projection, epic
-- resolution, provenance). target_files is deliberately ABSENT (never written by
-- import — DB default applies); current_stage is executor-owned. external_item_id
-- is the ProjectV2Item id (write-back target), distinct from the content node id.
INSERT INTO stories (
    project_id, epic_id, key, title, objective, acceptance_criteria,
    scope, depends_on, status, source, external_id, external_item_id, source_url,
    synced_at, last_import_hash
) VALUES (
    @project_id, @epic_id, @key, @title, @objective, @acceptance_criteria,
    @scope, @depends_on, @status, @source, @external_id, @external_item_id, @source_url,
    now(), @last_import_hash
)
RETURNING *;

-- name: UpdateStoryFromImport :one
-- Import-update for an UNLOCKED row: the SERVICE has already merged every value
-- (preserve-on-absent for spec fields, set-once epic_id, promote-only status).
-- current_stage / target_files are deliberately ABSENT (executor-owned).
UPDATE stories SET
    title               = @title,
    objective           = @objective,
    acceptance_criteria = @acceptance_criteria,
    scope               = @scope,
    depends_on          = @depends_on,
    status              = @status,
    epic_id             = @epic_id,
    source              = @source,
    external_id         = @external_id,
    external_item_id    = @external_item_id,
    source_url          = @source_url,
    synced_at           = now(),
    last_import_hash    = @last_import_hash,
    updated_at          = now()
WHERE id = @id
RETURNING *;

-- name: UpdateStoryProvenanceOnly :one
-- Locked rows (running/failed/in-stage): cosmetic title + provenance refresh only.
-- Crucially does NOT touch last_import_hash, so the deferred spec change is
-- re-applied on the FIRST re-import after the run terminates.
UPDATE stories SET
    title            = @title,
    source           = @source,
    external_id      = @external_id,
    external_item_id = @external_item_id,
    source_url       = @source_url,
    synced_at        = now(),
    updated_at       = now()
WHERE id = @id
RETURNING *;

-- name: SetStoryWritebackStatus :exec
-- Sets the last write-back state (disabled|pending|synced|failed). Managed solely by
-- the write-back path; the importer and run engine never touch this column.
UPDATE stories
SET writeback_status = $2, updated_at = now()
WHERE id = $1;
