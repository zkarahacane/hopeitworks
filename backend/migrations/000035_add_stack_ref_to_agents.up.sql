-- P2a: let an agent reference a catalogued stack instead of a free-form image.
-- stack_id is a nullable FK into the stacks catalogue (000034). This is additive:
-- the `image` column is kept, and an agent with stack_id NULL resolves its image
-- exactly as before (from `image`). When stack_id is set, LaunchRun resolves the
-- effective image from stacks.image_ref instead. ON DELETE SET NULL so removing a
-- stack degrades the agent to its image fallback rather than cascading a delete.
ALTER TABLE agents ADD COLUMN stack_id UUID NULL REFERENCES stacks(id) ON DELETE SET NULL;

CREATE INDEX idx_agents_stack_id ON agents(stack_id);
