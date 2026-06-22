-- P1 runtime/capabilities rework: capabilities are runtime-agnostic platform data
-- (skills / MCP servers / tool policies), composed onto agents and assembled into a
-- fetch-at-startup bundle. Credentials are encrypted secrets resolved at runtime —
-- never baked, never stored in clear, never passed through the container env.

-- capabilities: a versioned, scope-aware capability (skill | mcp_server | tool_policy).
-- spec is the agnostic jsonb (skill files / url+auth / allow-deny) an adapter translates.
CREATE TABLE capabilities (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind        VARCHAR(32)  NOT NULL CHECK (kind IN ('skill', 'mcp_server', 'tool_policy')),
    name        VARCHAR(255) NOT NULL,
    version     INT          NOT NULL DEFAULT 1,
    scope       VARCHAR(16)  NOT NULL CHECK (scope IN ('global', 'project')),
    project_id  UUID         NULL REFERENCES projects(id) ON DELETE CASCADE,
    spec        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    -- A project capability must carry a project_id; a global one must not.
    CONSTRAINT capabilities_scope_project_chk CHECK (
        (scope = 'global'  AND project_id IS NULL) OR
        (scope = 'project' AND project_id IS NOT NULL)
    )
);

CREATE INDEX idx_capabilities_project_id ON capabilities(project_id);
CREATE INDEX idx_capabilities_kind ON capabilities(kind);

-- A global capability is uniquely identified by (name, version); a project one by
-- (project_id, name, version). Partial unique indexes keep the two scopes independent.
CREATE UNIQUE INDEX capabilities_uq_global_name_version
    ON capabilities(name, version) WHERE scope = 'global';
CREATE UNIQUE INDEX capabilities_uq_project_name_version
    ON capabilities(project_id, name, version) WHERE scope = 'project';

-- agent_capabilities: the composition join binding a capability onto an agent.
CREATE TABLE agent_capabilities (
    agent_id      UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    capability_id UUID NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, capability_id)
);

CREATE INDEX idx_agent_capabilities_capability_id ON agent_capabilities(capability_id);

-- credentials: named secrets, encrypted at rest with AES-256-GCM (pkg/crypto, the same
-- scheme used for user API keys). encrypted_value holds nonce+ciphertext+tag as raw bytes.
-- Referenced by capability specs (mcp_server.credential_ref); resolved + decrypted only
-- when assembling a bundle for an authenticated container fetch.
CREATE TABLE credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    scope           VARCHAR(16)  NOT NULL CHECK (scope IN ('global', 'project')),
    project_id      UUID         NULL REFERENCES projects(id) ON DELETE CASCADE,
    encrypted_value BYTEA        NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT credentials_scope_project_chk CHECK (
        (scope = 'global'  AND project_id IS NULL) OR
        (scope = 'project' AND project_id IS NOT NULL)
    )
);

CREATE INDEX idx_credentials_project_id ON credentials(project_id);
CREATE UNIQUE INDEX credentials_uq_global_name
    ON credentials(name) WHERE scope = 'global';
CREATE UNIQUE INDEX credentials_uq_project_name
    ON credentials(project_id, name) WHERE scope = 'project';
