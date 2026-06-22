-- Add runtime_kind: the pluggable agent runtime an agent runs on
-- (claude_code | opencode | cma). This replaces the implicit execution-mode
-- detection that used to key off the free-form `image` string
-- (strings.Contains(image, "hopeitworks/agent-")).
ALTER TABLE agents ADD COLUMN runtime_kind VARCHAR(20) NOT NULL DEFAULT 'claude_code';

ALTER TABLE agents ADD CONSTRAINT agents_chk_runtime_kind
    CHECK (runtime_kind IN ('claude_code', 'opencode', 'cma'));

-- One-time backfill from the legacy isCallbackMode heuristic, run once here so the
-- image substring is never consulted again. Every existing agent runs on the
-- agent-runtime images (hopeitworks/agent-*, i.e. callback mode); its runtime is the
-- provider's harness: opencode -> 'opencode', everything else -> 'claude_code'
-- (the prior default callback runtime). The `image` column is intentionally kept for
-- now (the stack catalogue + StackRef land in a later phase).
UPDATE agents
SET runtime_kind = CASE
    WHEN provider = 'opencode' AND image LIKE '%hopeitworks/agent-%' THEN 'opencode'
    ELSE 'claude_code'
END;
