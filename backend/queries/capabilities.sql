-- name: CreateCapability :one
INSERT INTO capabilities (kind, name, version, scope, project_id, spec)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCapability :one
SELECT * FROM capabilities WHERE id = $1 LIMIT 1;

-- name: ListCapabilitiesByScope :many
SELECT * FROM capabilities
WHERE scope = 'global' OR ($1::uuid IS NOT NULL AND project_id = $1::uuid)
ORDER BY kind, name, version DESC;

-- name: DeleteCapability :exec
DELETE FROM capabilities WHERE id = $1;

-- name: AttachCapabilityToAgent :exec
INSERT INTO agent_capabilities (agent_id, capability_id)
VALUES ($1, $2)
ON CONFLICT (agent_id, capability_id) DO NOTHING;

-- name: DetachCapabilityFromAgent :exec
DELETE FROM agent_capabilities WHERE agent_id = $1 AND capability_id = $2;

-- name: ListCapabilitiesForAgent :many
SELECT c.*
FROM capabilities c
JOIN agent_capabilities ac ON ac.capability_id = c.id
WHERE ac.agent_id = $1
ORDER BY c.kind, c.name;
