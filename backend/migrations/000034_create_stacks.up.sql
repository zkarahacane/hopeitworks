-- P2a stack catalogue: a stack is a catalogued, multi-arch runtime image carrying a
-- toolchain, the runtime CLI (claude/opencode) and (eventually) the per-language LSP.
-- Agents will reference a stack by FK (stack_id, added in a later migration) instead of
-- the free-form `agents.image` string. This change is purely additive: the `image`
-- column stays, and an agent with only `image` and no stack resolves exactly as before.
--
-- image_ref holds a digest-pinned ref when one is known, otherwise a tag. The platform
-- owns these refs so pulls are deterministic.
CREATE TABLE stacks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key        VARCHAR(32) NOT NULL UNIQUE CHECK (key IN ('go', 'node', 'python', 'go-node')),
    image_ref  TEXT        NOT NULL,
    toolchain  JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the four catalogued stacks with the current ghcr image refs. The toolchain jsonb
-- mirrors what the agent-images actually carry today (Go 1.23 / Node 22 / Python 3.12 +
-- the claude/opencode CLIs + dev tools). LSP entries are intentionally omitted until the
-- images ship them (P2b). Idempotent on key so a re-run — or a later digest pin via
-- image_ref — is safe. Agents are NOT repointed here: existing agents keep their
-- free-form image and behave exactly as before.
INSERT INTO stacks (key, image_ref, toolchain) VALUES
    ('go',      'ghcr.io/zkarahacane/hopeitworks/agent-go:latest',
        '{"go": "1.23", "node": "22", "cli": ["claude", "opencode"], "tools": ["sqlc", "oapi-codegen", "wire", "golangci-lint", "gh"]}'::jsonb),
    ('node',    'ghcr.io/zkarahacane/hopeitworks/agent-node:latest',
        '{"node": "22", "cli": ["claude", "opencode"], "tools": ["typescript", "prettier", "eslint", "gh"]}'::jsonb),
    ('python',  'ghcr.io/zkarahacane/hopeitworks/agent-python:latest',
        '{"python": "3.12", "node": "22", "cli": ["claude", "opencode"], "tools": ["gh"]}'::jsonb),
    ('go-node', 'ghcr.io/zkarahacane/hopeitworks/agent-go-node:latest',
        '{"go": "1.23", "node": "22", "cli": ["claude", "opencode"], "tools": ["sqlc", "oapi-codegen", "wire", "golangci-lint", "gh"]}'::jsonb)
ON CONFLICT (key) DO NOTHING;
