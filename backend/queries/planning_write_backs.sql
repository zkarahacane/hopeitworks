-- name: CreatePlanningWriteBack :one
-- Append-only audit row for one write-back attempt (success or failure).
INSERT INTO planning_write_backs (
    project_id, story_id, run_id, source, external_id,
    internal_status, remote_status, success, error_code, error_message
) VALUES (
    @project_id, @story_id, @run_id, @source, @external_id,
    @internal_status, @remote_status, @success, @error_code, @error_message
)
RETURNING *;

-- name: ListPlanningWriteBacksByStory :many
SELECT * FROM planning_write_backs
WHERE story_id = $1
ORDER BY created_at DESC
LIMIT $2;
