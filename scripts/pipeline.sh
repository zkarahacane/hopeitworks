#!/usr/bin/env bash
set -euo pipefail

# BMAD Story Pipeline - runs inside a container
# Chains: dev-story (opus) → code-review (sonnet) → merge-story (sonnet)
#
# Each phase runs Claude Code with the appropriate workflow.
# If a phase fails (non-zero exit), the pipeline stops.
#
# Usage (inside container):
#   /pipeline.sh [extra-claude-args...]

STORY_KEY="${STORY_KEY:-unknown}"
LOG_PREFIX="[pipeline:${STORY_KEY}]"

log() { echo "$LOG_PREFIX $1"; }

log "=== Starting pipeline ==="
log "Story: $STORY_KEY"
log "Branch: $(git branch --show-current 2>/dev/null || echo 'unknown')"
log ""

# Phase 1: Dev Story (Opus)
log "=== Phase 1/3: dev-story (opus) ==="
if claude --dangerously-skip-permissions --model opus -p "/bmad-bmm-dev-story" "$@"; then
    log "✅ dev-story complete"
else
    log "❌ dev-story failed (exit $?)"
    exit 1
fi

log ""

# Phase 2: Code Review (Sonnet)
log "=== Phase 2/3: code-review (sonnet) ==="
if claude --dangerously-skip-permissions --model sonnet -p "/bmad-bmm-code-review" "$@"; then
    log "✅ code-review complete"
else
    log "❌ code-review failed (exit $?)"
    exit 2
fi

log ""

# Phase 3: Merge Story (Sonnet)
log "=== Phase 3/3: merge-story (sonnet) ==="
if claude --dangerously-skip-permissions --model sonnet -p "/bmad-bmm-merge-story" "$@"; then
    log "✅ merge-story complete"
else
    log "❌ merge-story failed (exit $?)"
    exit 3
fi

log ""
log "=== Pipeline complete for $STORY_KEY ==="
