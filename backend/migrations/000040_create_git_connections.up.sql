-- A project's live connection to its git host. PAT-only in v1 (token encrypted at
-- rest). The `kind` column is a 1-line forward hedge (CHECK pinned to 'pat'); a
-- later mechanism would widen it via an additive migration, not a rewrite.
-- encrypted_secret holds nonce+ciphertext+tag (pkg/crypto AES-256-GCM, the SAME
-- scheme and SAME ENCRYPTION_KEY as credentials/user_api_keys). The plaintext token
-- is produced only transiently at resolution time -- never logged, never returned.
CREATE TABLE git_connections (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider          VARCHAR(16)  NOT NULL DEFAULT 'github'
                        CHECK (provider IN ('github', 'gitea', 'gitlab', 'bitbucket')),
    kind              VARCHAR(16)  NOT NULL DEFAULT 'pat' CHECK (kind IN ('pat')),

    -- kind = 'pat'
    encrypted_secret  BYTEA        NOT NULL,        -- nonce+ciphertext+tag (AES-256-GCM)
    secret_last4      VARCHAR(8)   NULL,            -- display hint only ("cd12")
    token_type        VARCHAR(16)  NULL             -- classic | fine_grained | unknown (from prefix)
                        CHECK (token_type IS NULL OR token_type IN ('classic','fine_grained','unknown')),
    scopes            TEXT[]       NOT NULL DEFAULT '{}',  -- from X-OAuth-Scopes at validate time

    -- shared validation metadata (advisory / last-known; GitHub is source of truth)
    status            VARCHAR(24)  NOT NULL DEFAULT 'unconfigured'
                        CHECK (status IN ('unconfigured','connected','invalid','expired','insufficient_scope')),
    account_login     VARCHAR(255) NULL,            -- whoami login from the probe
    expires_at        TIMESTAMPTZ  NULL,            -- known for fine-grained PATs (header), null for classic
    last_validated_at TIMESTAMPTZ  NULL,
    validation_error  TEXT         NULL,            -- fixed code only, never raw provider text

    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),

    -- one connection per project
    CONSTRAINT git_connections_uq_project UNIQUE (project_id)
);

CREATE INDEX idx_git_connections_project_id ON git_connections(project_id);
