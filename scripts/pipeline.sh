#!/usr/bin/env bash
set -euo pipefail

# BMAD Story Pipeline - runs inside a container
# Chains: dev-story (opus) → code-review (sonnet) → merge-story (opus)
#
# Each phase runs Claude Code with the appropriate workflow.
# If a phase fails (non-zero exit), the pipeline stops.
#
# Env vars (set by entrypoint):
#   STORY_KEY   - story key (e.g. 1-2-openapi-spec-code-gen-pipeline)
#   BASE_BRANCH - target branch for merge (e.g. wave-1)
#
# Usage (inside container):
#   /pipeline.sh [extra-claude-args...]

STORY_KEY="${STORY_KEY:-unknown}"
BASE_BRANCH="${BASE_BRANCH:-main}"
SKIP_MERGE="${SKIP_MERGE:-false}"
LOG_PREFIX="[pipeline:${STORY_KEY}]"

log() { echo "$LOG_PREFIX $1"; }

log "=== Starting pipeline ==="
log "Story: $STORY_KEY"
log "Base: $BASE_BRANCH"
log "Branch: $(git branch --show-current 2>/dev/null || echo 'unknown')"
log ""

# Build context prompt that tells each agent exactly what to do
STORY_CONTEXT="You are running in automated pipeline mode inside a Docker container.
Story key: ${STORY_KEY}
Feature branch: feat/${STORY_KEY}
Base branch: ${BASE_BRANCH}
IMPORTANT: Do NOT ask questions. Act autonomously. Pick story ${STORY_KEY} automatically."

# Phase 1: Dev Story (Opus)
log "=== Phase 1/3: dev-story (opus) ==="
DEV_PROMPT="${STORY_CONTEXT}
Execute /bmad-bmm-dev-story for story ${STORY_KEY}.
The story file is at _bmad-output/implementation-artifacts/${STORY_KEY}.md
Work on branch feat/${STORY_KEY}. Commit and push all code. Create a PR targeting ${BASE_BRANCH}."

if echo "$DEV_PROMPT" | claude --dangerously-skip-permissions --model opus "$@"; then
    log "✅ dev-story complete"
else
    log "❌ dev-story failed (exit $?)"
    exit 1
fi

log ""

# Phase 2: Code Review (Sonnet)
log "=== Phase 2/3: code-review (sonnet) ==="
REVIEW_PROMPT="${STORY_CONTEXT}
Execute /bmad-bmm-code-review for story ${STORY_KEY}.
The story file is at _bmad-output/implementation-artifacts/${STORY_KEY}.md
Review ALL code changes on branch feat/${STORY_KEY} vs ${BASE_BRANCH}. Fix any issues found. Push fixes and ensure CI is green."

if echo "$REVIEW_PROMPT" | claude --dangerously-skip-permissions --model sonnet "$@"; then
    log "✅ code-review complete"
else
    log "❌ code-review failed (exit $?)"
    exit 2
fi

log ""

# Phase 3: Merge Story (Sonnet)
if [ "$SKIP_MERGE" = "true" ]; then
    log "=== Phase 3/3: merge-story SKIPPED (SKIP_MERGE=true) ==="
else
    log "=== Phase 3/3: merge-story (opus) ==="
    MERGE_PROMPT="${STORY_CONTEXT}
Execute /bmad-bmm-merge-story for story ${STORY_KEY}.
Merge the PR for feat/${STORY_KEY} into ${BASE_BRANCH} via squash merge. Ensure CI is green before merging."

    if echo "$MERGE_PROMPT" | claude --dangerously-skip-permissions --model opus "$@"; then
        log "✅ merge-story complete"
    else
        log "❌ merge-story failed (exit $?)"
        exit 3
    fi
fi

log ""
log "=== Pipeline complete for $STORY_KEY ==="
