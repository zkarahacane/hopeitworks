#!/usr/bin/env bash
set -euo pipefail

# BMAD Dev Agent Entrypoint
#
# Modes:
#   MOUNT (default): /workspace already populated via volume
#   CLONE: clones REPO_URL, checks out BASE_BRANCH, creates feat/ branch
#
# Pipeline mode:
#   If PIPELINE=true, runs /pipeline.sh instead of claude directly
#   Chains: dev-story → code-review → merge-story

if [[ -n "${REPO_URL:-}" ]]; then
    echo "=== CLONE mode ==="
    BASE_BRANCH="${BASE_BRANCH:-main}"

    # Inject token for HTTPS auth on private repos
    CLONE_URL="$REPO_URL"
    if [[ -n "${GITHUB_TOKEN:-}${GH_TOKEN:-}" ]] && [[ "$CLONE_URL" == https://github.com/* ]]; then
        TOKEN="${GITHUB_TOKEN:-$GH_TOKEN}"
        CLONE_URL="${CLONE_URL/https:\/\/github.com/https://${TOKEN}@github.com}"
    fi

    echo "Cloning $REPO_URL (branch: $BASE_BRANCH)..."
    git clone --branch "$BASE_BRANCH" --single-branch "$CLONE_URL" /workspace
    cd /workspace

    # Create or checkout feature branch (feat/ prefix for BMAD compatibility)
    if [[ -n "${STORY_BRANCH:-}" ]]; then
        FEAT_BRANCH="feat/${STORY_BRANCH}"
        if git ls-remote --heads origin "$FEAT_BRANCH" | grep -q "$FEAT_BRANCH"; then
            echo "Checking out existing branch: $FEAT_BRANCH"
            git fetch origin "$FEAT_BRANCH"
            git checkout "$FEAT_BRANCH"
        else
            echo "Creating new branch: $FEAT_BRANCH (from $BASE_BRANCH)"
            git checkout -b "$FEAT_BRANCH"
        fi
    fi

    # Configure git to use token for push operations too
    if [[ -n "${GITHUB_TOKEN:-}${GH_TOKEN:-}" ]]; then
        TOKEN="${GITHUB_TOKEN:-$GH_TOKEN}"
        git remote set-url origin "${REPO_URL/https:\/\/github.com/https://${TOKEN}@github.com}"
    fi

    echo "=== Ready ==="
else
    echo "=== MOUNT mode ==="
    cd /workspace
fi

# Pipeline or single-phase mode
if [[ "${PIPELINE:-false}" == "true" ]]; then
    echo "=== Pipeline mode ==="
    exec /pipeline.sh "$@"
else
    echo "Starting Claude Code..."
    exec claude "$@"
fi
