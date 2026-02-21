-- name: InsertNotificationConfig :one
INSERT INTO notification_configs (project_id, channel_type, config, events_filter, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetNotificationConfig :one
SELECT * FROM notification_configs WHERE id = $1;

-- name: ListNotificationConfigsByProject :many
SELECT * FROM notification_configs WHERE project_id = $1 ORDER BY created_at DESC;

-- name: UpdateNotificationConfig :one
UPDATE notification_configs
SET channel_type  = $2,
    config        = $3,
    events_filter = $4,
    enabled       = $5,
    updated_at    = now()
WHERE id = $1
RETURNING *;

-- name: DeleteNotificationConfig :exec
DELETE FROM notification_configs WHERE id = $1;

-- name: ListEnabledConfigsByProject :many
SELECT * FROM notification_configs WHERE project_id = $1 AND enabled = true ORDER BY created_at DESC;
