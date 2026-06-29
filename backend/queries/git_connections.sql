-- name: GetGitConnectionByProject :one
SELECT * FROM git_connections WHERE project_id = $1 LIMIT 1;

-- name: UpsertGitConnectionPAT :one
INSERT INTO git_connections
  (project_id, provider, kind, encrypted_secret, secret_last4, token_type, scopes,
   status, account_login, expires_at, last_validated_at, validation_error)
VALUES ($1, $2, 'pat', $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (project_id) DO UPDATE SET
  provider=$2, kind='pat', encrypted_secret=$3, secret_last4=$4, token_type=$5,
  scopes=$6, status=$7, account_login=$8, expires_at=$9, last_validated_at=$10,
  validation_error=$11, updated_at=now()
RETURNING *;

-- name: SetGitConnectionValidation :exec
UPDATE git_connections
SET status=$2, account_login=$3, scopes=$4, expires_at=$5,
    last_validated_at=now(), validation_error=$6, updated_at=now()
WHERE project_id=$1;

-- name: MarkGitConnectionStatus :exec
UPDATE git_connections
SET status=$2, validation_error=$3, last_validated_at=now(), updated_at=now()
WHERE project_id=$1;

-- name: DeleteGitConnectionByProject :exec
DELETE FROM git_connections WHERE project_id = $1;
