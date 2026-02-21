#!/usr/bin/env bash
set -euo pipefail

# BMAD Dev Agent Entrypoint
#
# Modes:
#   MOUNT (default): /workspace already populated via volume
#   CLONE: clones REPO_URL, checks out CLONE_BRANCH, creates feat/ branch
#
# Env vars:
#   CLONE_BRANCH - branch to clone from (default: develop)
#   BASE_BRANCH  - PR merge target (default: CLONE_BRANCH) — passed to pipeline
#   STORY_BRANCH - story key → creates feat/{STORY_BRANCH}
#
# Pipeline mode:
#   If PIPELINE=true, runs /pipeline.sh instead of claude directly
#   Chains: dev-story → code-review → merge-story

if [[ -n "${REPO_URL:-}" ]]; then
    echo "=== CLONE mode ==="
    CLONE_BRANCH="${CLONE_BRANCH:-develop}"
    BASE_BRANCH="${BASE_BRANCH:-$CLONE_BRANCH}"

    # Inject token for HTTPS auth on private repos
    CLONE_URL="$REPO_URL"
    if [[ -n "${GITHUB_TOKEN:-}${GH_TOKEN:-}" ]] && [[ "$CLONE_URL" == https://github.com/* ]]; then
        TOKEN="${GITHUB_TOKEN:-$GH_TOKEN}"
        CLONE_URL="${CLONE_URL/https:\/\/github.com/https://${TOKEN}@github.com}"
    fi

    echo "Cloning $REPO_URL (branch: $CLONE_BRANCH)..."
    git clone --branch "$CLONE_BRANCH" --single-branch "$CLONE_URL" /workspace
    cd /workspace

    # Create or checkout feature branch (feat/ prefix for BMAD compatibility)
    if [[ -n "${STORY_BRANCH:-}" ]]; then
        FEAT_BRANCH="feat/${STORY_BRANCH}"
        if git ls-remote --heads origin "$FEAT_BRANCH" | grep -q "$FEAT_BRANCH"; then
            echo "Checking out existing branch: $FEAT_BRANCH"
            git fetch origin "$FEAT_BRANCH:$FEAT_BRANCH"
            git checkout "$FEAT_BRANCH"
        else
            echo "Creating new branch: $FEAT_BRANCH (from $CLONE_BRANCH)"
            git checkout -b "$FEAT_BRANCH"
        fi
    fi

    # Configure git to use token for push operations too
    if [[ -n "${GITHUB_TOKEN:-}${GH_TOKEN:-}" ]]; then
        TOKEN="${GITHUB_TOKEN:-$GH_TOKEN}"
        git remote set-url origin "${REPO_URL/https:\/\/github.com/https://${TOKEN}@github.com}"
    fi

    echo "Clone: $CLONE_BRANCH | Merge target: $BASE_BRANCH"
    echo "=== Ready ==="
else
    echo "=== MOUNT mode ==="
    cd /workspace
fi

# Install pre-push git hook to enforce lint + unit tests before every push
install_pre_push_hook() {
    local hooks_dir="/workspace/.git/hooks"
    mkdir -p "$hooks_dir"

    cat > "$hooks_dir/pre-push" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail

# Pre-push hook: detect changed directories and run lint + unit tests.
# Receives stdin lines: <local ref> <local sha> <remote ref> <remote sha>

ERRORS=0

# Determine the commit range being pushed
read -r LOCAL_REF LOCAL_SHA REMOTE_REF REMOTE_SHA || true

if [[ -z "${LOCAL_SHA:-}" ]]; then
    echo "[pre-push] No commits to push, skipping validation."
    exit 0
fi

# If remote sha is all zeros this is a first push — compare against empty tree
EMPTY_TREE="4b825dc642cb6eb9a060e54bf8d69288fbee4904"
if [[ "${REMOTE_SHA:-}" == "0000000000000000000000000000000000000000" ]] || [[ -z "${REMOTE_SHA:-}" ]]; then
    RANGE="${EMPTY_TREE}..${LOCAL_SHA}"
else
    RANGE="${REMOTE_SHA}..${LOCAL_SHA}"
fi

echo "[pre-push] Validating range: $RANGE"

# Detect which top-level directories have changed
CHANGED_DIRS=$(git diff --name-only "$RANGE" 2>/dev/null | cut -d/ -f1 | sort -u || true)

BACKEND_CHANGED=false
FRONTEND_CHANGED=false

if echo "$CHANGED_DIRS" | grep -q "^backend$"; then
    BACKEND_CHANGED=true
fi
if echo "$CHANGED_DIRS" | grep -q "^frontend$"; then
    FRONTEND_CHANGED=true
fi

if [[ "$BACKEND_CHANGED" == "false" && "$FRONTEND_CHANGED" == "false" ]]; then
    echo "[pre-push] No backend or frontend changes detected — skipping checks."
    exit 0
fi

# Backend checks
if [[ "$BACKEND_CHANGED" == "true" ]] && [[ -d "/workspace/backend" ]]; then
    echo "[pre-push] Backend changed — running lint..."
    if (cd /workspace/backend && golangci-lint run ./... 2>&1); then
        echo "[pre-push] Backend lint: PASSED"
    else
        echo "[pre-push] Backend lint: FAILED"
        ERRORS=$((ERRORS + 1))
    fi

    echo "[pre-push] Backend changed — running unit tests..."
    if (cd /workspace/backend && go test ./... -short 2>&1); then
        echo "[pre-push] Backend unit tests: PASSED"
    else
        echo "[pre-push] Backend unit tests: FAILED"
        ERRORS=$((ERRORS + 1))
    fi
fi

# Frontend checks
if [[ "$FRONTEND_CHANGED" == "true" ]] && [[ -d "/workspace/frontend" ]]; then
    echo "[pre-push] Frontend changed — running lint..."
    if (cd /workspace/frontend && npm run lint 2>&1); then
        echo "[pre-push] Frontend lint: PASSED"
    else
        echo "[pre-push] Frontend lint: FAILED"
        ERRORS=$((ERRORS + 1))
    fi

    echo "[pre-push] Frontend changed — running type-check..."
    if (cd /workspace/frontend && npm run type-check 2>&1); then
        echo "[pre-push] Frontend type-check: PASSED"
    else
        echo "[pre-push] Frontend type-check: FAILED"
        ERRORS=$((ERRORS + 1))
    fi
fi

if [[ "$ERRORS" -gt 0 ]]; then
    echo ""
    echo "[pre-push] PUSH BLOCKED: $ERRORS check(s) failed. Fix the issues above before pushing."
    exit 1
fi

echo "[pre-push] All checks passed."
exit 0
HOOK

    chmod +x "$hooks_dir/pre-push"
    echo "=== Pre-push hook installed ==="
}

install_pre_push_hook

# Pipeline or single-phase mode
if [[ "${PIPELINE:-false}" == "true" ]]; then
    echo "=== Pipeline mode ==="
    exec /pipeline.sh "$@"
else
    echo "Starting Claude Code..."
    exec claude "$@"
fi
