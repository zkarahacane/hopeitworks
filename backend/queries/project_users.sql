-- name: AddUserToProject :one
INSERT INTO project_users (project_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveUserFromProject :exec
DELETE FROM project_users WHERE project_id = $1 AND user_id = $2;

-- name: ListProjectUsers :many
SELECT u.id, u.email, u.name, u.role AS user_role, pu.role AS project_role, pu.created_at AS assigned_at
FROM project_users pu
JOIN users u ON u.id = pu.user_id
WHERE pu.project_id = $1
ORDER BY pu.created_at ASC;

-- name: IsUserInProject :one
SELECT EXISTS(
    SELECT 1 FROM project_users WHERE project_id = $1 AND user_id = $2
) AS is_member;

-- name: ListUserProjectIDs :many
SELECT project_id FROM project_users WHERE user_id = $1;

-- name: ListProjectsByUser :many
SELECT p.* FROM projects p
INNER JOIN project_users pu ON pu.project_id = p.id
WHERE pu.user_id = $1
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountProjectsByUser :one
SELECT COUNT(*) FROM projects p
INNER JOIN project_users pu ON pu.project_id = p.id
WHERE pu.user_id = $1;
