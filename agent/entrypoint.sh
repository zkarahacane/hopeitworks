#!/usr/bin/env bash
set -euo pipefail

# hopeitworks Runtime Agent Entrypoint
#
# Clones a repository, checks out the target branch, injects CLAUDE.md,
# runs Claude Code with the rendered prompt, and emits NDJSON logs.
#
# Required env vars:
#   REPO_URL             - HTTPS URL of the git repository
#   BRANCH_NAME          - Branch to checkout (created if absent)
#   CLAUDE_MD_CONTENT    - Full CLAUDE.md content to inject
#   PROMPT_CONTENT       - Rendered agent prompt
#   GIT_TOKEN            - Token for git authentication (or GITHUB_TOKEN for backward compat)
#   CLAUDE_CODE_OAUTH_TOKEN - OAuth token for Claude Code
#
# Optional env vars:
#   GIT_PROVIDER         - Git provider (default: github). Set to "gitea" for Gitea.
#   STORY_KEY            - Story key for git commit context
#   GIT_AUTHOR_NAME      - Git author name (default: hopeitworks-agent)
#   GIT_AUTHOR_EMAIL     - Git author email (default: agent@hopeitworks.local)

# --- Helper: emit a structured NDJSON log line ---
emit_log() {
    local message="$1"
    local level="${2:-info}"
    local ts
    ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    printf '{"type":"log","level":"%s","message":"%s","timestamp":"%s"}\n' \
        "$level" \
        "$(echo "$message" | sed 's/"/\\"/g' | tr -d '\n')" \
        "$ts"
}

# --- Resolve token (GIT_TOKEN preferred, GITHUB_TOKEN as fallback) ---
GIT_TOKEN="${GIT_TOKEN:-${GITHUB_TOKEN:-}}"
GIT_PROVIDER="${GIT_PROVIDER:-github}"

# --- Validate required env vars ---
for var in REPO_URL BRANCH_NAME CLAUDE_MD_CONTENT PROMPT_CONTENT GIT_TOKEN CLAUDE_CODE_OAUTH_TOKEN; do
    if [[ -z "${!var:-}" ]]; then
        emit_log "Missing required env var: $var" "error"
        exit 1
    fi
done

# --- Configure git identity ---
export GIT_AUTHOR_NAME="${GIT_AUTHOR_NAME:-hopeitworks-agent}"
export GIT_AUTHOR_EMAIL="${GIT_AUTHOR_EMAIL:-agent@hopeitworks.local}"
export GIT_COMMITTER_NAME="$GIT_AUTHOR_NAME"
export GIT_COMMITTER_EMAIL="$GIT_AUTHOR_EMAIL"
git config --global user.name "$GIT_AUTHOR_NAME"
git config --global user.email "$GIT_AUTHOR_EMAIL"

# --- Configure GitHub CLI auth (only for GitHub) ---
if [[ "$GIT_PROVIDER" == "github" ]]; then
    export GH_TOKEN="${GIT_TOKEN}"
fi

# --- Configure Claude Code authentication ---
# CLAUDE_CODE_OAUTH_TOKEN is natively recognized by Claude Code (authMethod: oauth_token).
# Do NOT remap it to ANTHROPIC_API_KEY — that expects a direct API key, not an OAuth token.

# --- Clone repository ---
emit_log "Cloning repository: $REPO_URL (branch: $BRANCH_NAME)"

CLONE_URL="$REPO_URL"
if [[ -n "${GIT_TOKEN:-}" ]]; then
    # Inject token into any HTTPS URL: https://token@host/owner/repo
    CLONE_URL=$(echo "$CLONE_URL" | sed -E "s|^(https?://)|\1${GIT_TOKEN}@|")
fi

# Clone default branch first, then handle target branch
if ! git clone "$CLONE_URL" /workspace 2>&1; then
    emit_log "Failed to clone repository: $REPO_URL" "error"
    exit 1
fi
cd /workspace

# --- Checkout target branch ---
if git ls-remote --heads origin "$BRANCH_NAME" | grep -q "$BRANCH_NAME"; then
    emit_log "Checking out existing branch: $BRANCH_NAME"
    if ! git fetch origin "$BRANCH_NAME:$BRANCH_NAME"; then
        emit_log "Failed to fetch branch: $BRANCH_NAME" "error"
        exit 1
    fi
    if ! git checkout "$BRANCH_NAME"; then
        emit_log "Failed to checkout branch: $BRANCH_NAME" "error"
        exit 1
    fi
else
    emit_log "Creating new branch: $BRANCH_NAME"
    if ! git checkout -b "$BRANCH_NAME"; then
        emit_log "Failed to create branch: $BRANCH_NAME" "error"
        exit 1
    fi
fi

# Configure remote to use token for push operations
if [[ -n "${GIT_TOKEN:-}" ]]; then
    # Inject token into any HTTPS URL: https://token@host/owner/repo
    PUSH_URL=$(echo "$REPO_URL" | sed -E "s|^(https?://)|\1${GIT_TOKEN}@|")
    git remote set-url origin "$PUSH_URL"
fi

emit_log "Branch ready: $BRANCH_NAME"

# --- Inject CLAUDE.md ---
echo "$CLAUDE_MD_CONTENT" > /workspace/CLAUDE.md
emit_log "CLAUDE.md injected"

# --- Write prompt to file ---
echo "$PROMPT_CONTENT" > /tmp/prompt.md
emit_log "Prompt written to /tmp/prompt.md"

# --- Run Claude Code ---
emit_log "Starting Claude Code agent"

EXIT_CODE=0
claude --dangerously-skip-permissions --print --verbose --output-format stream-json < /tmp/prompt.md || EXIT_CODE=$?

# --- Propagate exit code ---
if [[ "$EXIT_CODE" -ne 0 ]]; then
    emit_log "Agent exited with code $EXIT_CODE" "error"
else
    emit_log "Agent completed successfully"
fi

exit "$EXIT_CODE"
