#!/usr/bin/env bash
set -euo pipefail

# BMAD Dev Agent Launcher
# Runs Claude Code in a Docker sandbox with full project tooling
#
# Usage:
#   ./scripts/bmad-dev.sh                              # Interactive mode
#   ./scripts/bmad-dev.sh -p "/bmad-bmm-dev-story"     # Run dev-story workflow
#   ./scripts/bmad-dev.sh -p "/bmad-bmm-code-review"   # Run code-review
#   ./scripts/bmad-dev.sh --build                       # Force rebuild image
#   ./scripts/bmad-dev.sh --story 1-1-go-scaffolding    # Name container after story
#
# Required env vars:
#   CLAUDE_CODE_OAUTH_TOKEN - OAuth token for Claude Code
#   GITHUB_TOKEN            - GitHub token for gh CLI (or gh auth login inside container)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="bmad-dev-agent"
STORY_NAME=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Parse args
BUILD=false
CLAUDE_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD=true
            shift
            ;;
        --story)
            STORY_NAME="$2"
            shift 2
            ;;
        -p|--prompt)
            CLAUDE_ARGS+=("-p" "$2")
            shift 2
            ;;
        --model)
            CLAUDE_ARGS+=("--model" "$2")
            shift 2
            ;;
        *)
            CLAUDE_ARGS+=("$1")
            shift
            ;;
    esac
done

# Container name: unique per story (allows parallel containers)
if [[ -n "$STORY_NAME" ]]; then
    CONTAINER_NAME="bmad-dev-${STORY_NAME}"
else
    CONTAINER_NAME="bmad-dev-${USER:-agent}-$$"
fi

# Check required env vars
if [[ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]]; then
    echo -e "${RED}Error: CLAUDE_CODE_OAUTH_TOKEN is not set${NC}"
    echo "Export it: export CLAUDE_CODE_OAUTH_TOKEN=<your-oauth-token>"
    exit 1
fi

# Build image if needed
if $BUILD || ! docker image inspect "$IMAGE_NAME" &>/dev/null; then
    echo -e "${YELLOW}Building dev-agent image...${NC}"
    docker build \
        -t "$IMAGE_NAME" \
        -f "$SCRIPT_DIR/Dockerfile.dev-agent" \
        "$SCRIPT_DIR"
    echo -e "${GREEN}Image built: $IMAGE_NAME${NC}"
fi

# Stop existing container with same name if running
if docker ps -q -f "name=^${CONTAINER_NAME}$" | grep -q .; then
    echo -e "${YELLOW}Stopping existing container: $CONTAINER_NAME${NC}"
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
fi

# Ensure SSH known_hosts exists (writable copy for container)
SSH_TMPDIR=$(mktemp -d)
if [[ -f "${HOME}/.ssh/known_hosts" ]]; then
    cp "${HOME}/.ssh/known_hosts" "${SSH_TMPDIR}/known_hosts"
else
    touch "${SSH_TMPDIR}/known_hosts"
fi
# Copy keys (read-only in container via permissions, but known_hosts writable)
cp "${HOME}/.ssh/id_"* "${SSH_TMPDIR}/" 2>/dev/null || true
cp "${HOME}/.ssh/config" "${SSH_TMPDIR}/" 2>/dev/null || true
trap "rm -rf ${SSH_TMPDIR}" EXIT

echo -e "${GREEN}Launching BMAD Dev Agent in sandbox...${NC}"
echo -e "  Project:   $PROJECT_ROOT"
echo -e "  Image:     $IMAGE_NAME"
echo -e "  Container: $CONTAINER_NAME"
if [[ ${#CLAUDE_ARGS[@]} -gt 0 ]]; then
    echo -e "  Args:      ${CLAUDE_ARGS[*]}"
fi
echo ""

# Run container
# - Mount project directory (read-write)
# - Mount git config for commits (read-only)
# - Mount gh config for GitHub CLI (read-only)
# - Mount SSH with writable known_hosts (temp copy)
# - Pass OAuth token + GitHub token
# - Isolated network (no port conflicts between parallel containers)
# - Docker socket for stories that need container management (Epic 3+)
docker run \
    --name "$CONTAINER_NAME" \
    --rm \
    -it \
    -v "$PROJECT_ROOT:/workspace" \
    -v "${HOME}/.gitconfig:/root/.gitconfig:ro" \
    -v "${HOME}/.config/gh:/root/.config/gh:ro" \
    -v "${SSH_TMPDIR}:/root/.ssh" \
    -v "/var/run/docker.sock:/var/run/docker.sock" \
    -e "CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}" \
    -e "GITHUB_TOKEN=${GITHUB_TOKEN:-}" \
    -e "GH_TOKEN=${GITHUB_TOKEN:-}" \
    "$IMAGE_NAME" \
    --dangerously-skip-permissions \
    "${CLAUDE_ARGS[@]}"
