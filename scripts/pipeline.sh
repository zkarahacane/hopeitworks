#!/usr/bin/env bash
set -euo pipefail

# BMAD Story Pipeline - runs inside a container
# Chains: dev-story → code-review → merge-story
# Models: MODEL_DEV (default: opus), MODEL_REVIEW (default: sonnet), MODEL_MERGE (default: opus)
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
MODEL_DEV="${MODEL_DEV:-opus}"
MODEL_REVIEW="${MODEL_REVIEW:-sonnet}"
MODEL_MERGE="${MODEL_MERGE:-opus}"
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

# Phase 1: Dev Story
log "=== Phase 1/3: dev-story ($MODEL_DEV) ==="
DEV_PROMPT="${STORY_CONTEXT}
Execute /bmad-bmm-dev-story for story ${STORY_KEY}.
The story file is at _bmad-output/implementation-artifacts/${STORY_KEY}.md
Work on branch feat/${STORY_KEY}. Commit and push all code. Create a PR targeting ${BASE_BRANCH}."

if echo "$DEV_PROMPT" | claude --dangerously-skip-permissions --model "$MODEL_DEV" "$@"; then
    log "✅ dev-story complete"
else
    log "❌ dev-story failed (exit $?)"
    exit 1
fi

log ""

# Validation gate: lint + tests after dev-story, before code-review
log "=== Validation gate (post-dev-story) ==="
VALIDATION_ERRORS=""

if [[ -d "/workspace/backend" ]]; then
    log "Running backend lint..."
    BACKEND_LINT_OUT=$(cd /workspace/backend && golangci-lint run ./... 2>&1) && BACKEND_LINT_EXIT=0 || BACKEND_LINT_EXIT=$?
    if [[ "$BACKEND_LINT_EXIT" -ne 0 ]]; then
        log "⚠️  Backend lint FAILED"
        VALIDATION_ERRORS="${VALIDATION_ERRORS}
### Backend lint failures
\`\`\`
${BACKEND_LINT_OUT}
\`\`\`"
    else
        log "✅ Backend lint passed"
    fi

    log "Running backend tests (including integration, with race detector)..."
    BACKEND_TEST_OUT=$(cd /workspace/backend && go test ./... -race -count=1 2>&1) && BACKEND_TEST_EXIT=0 || BACKEND_TEST_EXIT=$?
    if [[ "$BACKEND_TEST_EXIT" -ne 0 ]]; then
        log "⚠️  Backend tests FAILED"
        VALIDATION_ERRORS="${VALIDATION_ERRORS}
### Backend test failures
\`\`\`
${BACKEND_TEST_OUT}
\`\`\`"
    else
        log "✅ Backend tests passed"
    fi
fi

if [[ -d "/workspace/frontend" ]]; then
    log "Running frontend lint..."
    FRONTEND_LINT_OUT=$(cd /workspace/frontend && npm run lint 2>&1) && FRONTEND_LINT_EXIT=0 || FRONTEND_LINT_EXIT=$?
    if [[ "$FRONTEND_LINT_EXIT" -ne 0 ]]; then
        log "⚠️  Frontend lint FAILED"
        VALIDATION_ERRORS="${VALIDATION_ERRORS}
### Frontend lint failures
\`\`\`
${FRONTEND_LINT_OUT}
\`\`\`"
    else
        log "✅ Frontend lint passed"
    fi

    log "Running frontend type-check..."
    FRONTEND_TC_OUT=$(cd /workspace/frontend && npm run type-check 2>&1) && FRONTEND_TC_EXIT=0 || FRONTEND_TC_EXIT=$?
    if [[ "$FRONTEND_TC_EXIT" -ne 0 ]]; then
        log "⚠️  Frontend type-check FAILED"
        VALIDATION_ERRORS="${VALIDATION_ERRORS}
### Frontend type-check failures
\`\`\`
${FRONTEND_TC_OUT}
\`\`\`"
    else
        log "✅ Frontend type-check passed"
    fi
fi

if [[ -n "$VALIDATION_ERRORS" ]]; then
    log "⚠️  Validation gate detected issues — passing to code-review agent as priority context"
else
    log "✅ Validation gate passed — no issues found"
fi

log ""

# Phase 2: Code Review
log "=== Phase 2/3: code-review ($MODEL_REVIEW) ==="

VALIDATION_PREAMBLE=""
if [[ -n "$VALIDATION_ERRORS" ]]; then
    VALIDATION_PREAMBLE="## PRIORITY: Validation gate failures detected

The following checks failed after dev-story completed. Address these FIRST before reviewing other aspects of the code:
${VALIDATION_ERRORS}

---

"
fi

REVIEW_PROMPT="${VALIDATION_PREAMBLE}${STORY_CONTEXT}
Execute /bmad-bmm-code-review for story ${STORY_KEY}.
The story file is at _bmad-output/implementation-artifacts/${STORY_KEY}.md
Review ALL code changes on branch feat/${STORY_KEY} vs ${BASE_BRANCH}. Fix any issues found. Push fixes and ensure CI is green."

if echo "$REVIEW_PROMPT" | claude --dangerously-skip-permissions --model "$MODEL_REVIEW" "$@"; then
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
    log "=== Phase 3/3: merge-story ($MODEL_MERGE) ==="
    MERGE_PROMPT="${STORY_CONTEXT}
Execute /bmad-bmm-merge-story for story ${STORY_KEY}.
Merge the PR for feat/${STORY_KEY} into ${BASE_BRANCH} via squash merge. Ensure CI is green before merging."

    if echo "$MERGE_PROMPT" | claude --dangerously-skip-permissions --model "$MODEL_MERGE" "$@"; then
        log "✅ merge-story complete"
    else
        log "❌ merge-story failed (exit $?)"
        exit 3
    fi
fi

log ""
log "=== Pipeline complete for $STORY_KEY ==="
