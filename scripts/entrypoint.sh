#!/usr/bin/env bash
set -euo pipefail

# BMAD Dev Agent Entrypoint
#
# Modes:
#   MOUNT (default): /workspace already populated via volume
#   CLONE: clones REPO_URL, checks out BASE_BRANCH, optionally creates STORY_BRANCH

if [[ -n "${REPO_URL:-}" ]]; then
    echo "=== CLONE mode ==="
    BASE_BRANCH="${BASE_BRANCH:-main}"

    echo "Cloning $REPO_URL (branch: $BASE_BRANCH)..."
    git clone --branch "$BASE_BRANCH" --single-branch "$REPO_URL" /workspace
    cd /workspace

    # Create or checkout story branch
    if [[ -n "${STORY_BRANCH:-}" ]]; then
        if git ls-remote --heads origin "$STORY_BRANCH" | grep -q "$STORY_BRANCH"; then
            echo "Checking out existing branch: $STORY_BRANCH"
            git fetch origin "$STORY_BRANCH"
            git checkout "$STORY_BRANCH"
        else
            echo "Creating new branch: $STORY_BRANCH"
            git checkout -b "$STORY_BRANCH"
        fi
    fi

    echo "=== Ready ==="
else
    echo "=== MOUNT mode ==="
    cd /workspace
fi

# All remaining args go to claude
echo "Starting Claude Code..."
exec claude "$@"
