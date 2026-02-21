-- Stores revoked JWT IDs (JTI) to prevent reuse after logout.
-- Entries are cleaned up once expires_at has passed.
CREATE TABLE revoked_tokens (
    jti        TEXT        NOT NULL PRIMARY KEY,  -- JWT ID claim (uuid string)
    expires_at TIMESTAMPTZ NOT NULL               -- copied from JWT exp claim
);

-- Index for fast lookup in auth middleware hot path
CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens (expires_at);
