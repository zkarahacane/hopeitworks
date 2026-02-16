#!/usr/bin/env bash
set -euo pipefail

# BMAD Dev Agent Launcher
#
# Story lifecycle: dev-story (Opus) → code-review (Sonnet) → merge-story (Sonnet)
# Branching: main → wave-X → feat/story-key
#
# Usage:
#   # Interactive (mount mode)
#   ./scripts/bmad-dev.sh
#
#   # Single story - one phase
#   ./scripts/bmad-dev.sh --story 1-1 --wave 1 --phase dev-story
#   ./scripts/bmad-dev.sh --story 1-1 --wave 1 --phase code-review
#   ./scripts/bmad-dev.sh --story 1-1 --wave 1 --phase merge-story
#
#   # Full wave - one phase on all stories
#   ./scripts/bmad-dev.sh --wave 1 --phase dev-story
#   ./scripts/bmad-dev.sh --wave 1 --phase code-review
#   ./scripts/bmad-dev.sh --wave 1 --phase merge-story
#
#   # Full pipeline per story (dev → review → merge) on entire wave
#   ./scripts/bmad-dev.sh --wave 1 --pipeline
#
#   # Setup wave branch
#   ./scripts/bmad-dev.sh --wave 1 --setup
#
#   # Monitoring
#   ./scripts/bmad-dev.sh --status
#
#   # Force rebuild
#   ./scripts/bmad-dev.sh --build
#
# Required env vars:
#   CLAUDE_CODE_OAUTH_TOKEN - OAuth token for Claude Code
#   GITHUB_TOKEN            - GitHub token for gh CLI

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="bmad-dev-agent"
SPRINT_STATUS="$PROJECT_ROOT/_bmad-output/implementation-artifacts/sprint-status.yaml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Parse args
BUILD=false
WAVE_NUM=""
STORY_NAME=""
PHASE=""
PIPELINE=false
SETUP=false
STATUS=false
CLAUDE_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)     BUILD=true; shift ;;
        --wave)      WAVE_NUM="$2"; shift 2 ;;
        --story)     STORY_NAME="$2"; shift 2 ;;
        --phase)     PHASE="$2"; shift 2 ;;
        --pipeline)  PIPELINE=true; shift ;;
        --setup)     SETUP=true; shift ;;
        --status)    STATUS=true; shift ;;
        -p|--prompt) CLAUDE_ARGS+=("-p" "$2"); shift 2 ;;
        --model)     CLAUDE_ARGS+=("--model" "$2"); shift 2 ;;
        *)           CLAUDE_ARGS+=("$1"); shift ;;
    esac
done

# Phase → BMAD workflow + model
get_workflow() {
    case "$1" in
        dev-story)    echo "/bmad-bmm-dev-story" ;;
        code-review)  echo "/bmad-bmm-code-review" ;;
        merge-story)  echo "/bmad-bmm-merge-story" ;;
        *)            echo "" ;;
    esac
}
get_model() {
    case "$1" in
        dev-story)    echo "opus" ;;
        code-review)  echo "sonnet" ;;
        merge-story)  echo "sonnet" ;;
        *)            echo "opus" ;;
    esac
}

# ============================================================
# HELPERS
# ============================================================

check_token() {
    if [[ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]]; then
        echo -e "${RED}Error: CLAUDE_CODE_OAUTH_TOKEN is not set${NC}"
        exit 1
    fi
}

get_repo_url() {
    cd "$PROJECT_ROOT"
    local url
    url=$(git remote get-url origin 2>/dev/null || echo "")
    if [[ "$url" == git@github.com:* ]]; then
        url="https://github.com/${url#git@github.com:}"
        url="${url%.git}"
    fi
    echo "$url"
}

build_image() {
    if $BUILD || ! docker image inspect "$IMAGE_NAME" &>/dev/null; then
        echo -e "${YELLOW}Building dev-agent image...${NC}"
        docker build -t "$IMAGE_NAME" -f "$SCRIPT_DIR/Dockerfile.dev-agent" "$SCRIPT_DIR"
        echo -e "${GREEN}Image built${NC}"
    fi
}

setup_ssh() {
    local tmpdir
    tmpdir=$(mktemp -d)
    if [[ -f "${HOME}/.ssh/known_hosts" ]]; then
        cp "${HOME}/.ssh/known_hosts" "${tmpdir}/known_hosts"
    else
        touch "${tmpdir}/known_hosts"
    fi
    cp "${HOME}/.ssh/id_"* "${tmpdir}/" 2>/dev/null || true
    cp "${HOME}/.ssh/config" "${tmpdir}/" 2>/dev/null || true
    echo "$tmpdir"
}

stop_container() {
    local name="$1"
    if docker ps -aq -f "name=^${name}$" | grep -q .; then
        docker stop "$name" >/dev/null 2>&1 || true
        docker rm "$name" >/dev/null 2>&1 || true
    fi
}

get_wave_stories() {
    local wave="$1"
    local in_wave=false
    local in_stories=false
    while IFS= read -r line; do
        # Detect wave start: "  wave-N:"
        if echo "$line" | grep -qE "^  wave-${wave}:"; then
            in_wave=true
            in_stories=false
            continue
        fi
        # Detect next wave or end of parallel_waves
        if $in_wave && echo "$line" | grep -qE "^  wave-[0-9]+:"; then
            break
        fi
        # Detect stories section
        if $in_wave && echo "$line" | grep -q "stories:"; then
            in_stories=true
            continue
        fi
        # Extract key values
        if $in_wave && $in_stories && echo "$line" | grep -qE "^\s+- key:"; then
            echo "$line" | sed 's/.*key: *//'
        fi
        # Stop stories section on non-indented or different section
        if $in_wave && $in_stories && echo "$line" | grep -qE "^    [a-z]" && ! echo "$line" | grep -qE "^\s+-"; then
            in_stories=false
        fi
    done < "$SPRINT_STATUS"
}

# Run container in CLONE mode (detached)
# Args: container_name base_branch story_key [claude_args...]
run_clone() {
    local container_name="$1"
    local base_branch="$2"
    local story_key="$3"
    shift 3

    local repo_url ssh_dir
    repo_url=$(get_repo_url)
    ssh_dir=$(setup_ssh)

    if [[ -z "$repo_url" ]]; then
        echo -e "${RED}Error: No git remote 'origin'${NC}"
        return 1
    fi

    local extra_env=()
    if [[ "${PIPELINE}" == "true" ]] || [[ "${_PIPELINE_MODE:-}" == "true" ]]; then
        extra_env+=(-e "PIPELINE=true" -e "STORY_KEY=${story_key}")
    fi

    docker run \
        --name "$container_name" \
        --rm -d \
        -v "${HOME}/.gitconfig:/root/.gitconfig:ro" \
        -v "${HOME}/.config/gh:/root/.config/gh:ro" \
        -v "${ssh_dir}:/root/.ssh" \
        -v "/var/run/docker.sock:/var/run/docker.sock" \
        -e "CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}" \
        -e "GITHUB_TOKEN=${GITHUB_TOKEN:-}" \
        -e "GH_TOKEN=${GITHUB_TOKEN:-}" \
        -e "REPO_URL=${repo_url}" \
        -e "BASE_BRANCH=${base_branch}" \
        -e "STORY_BRANCH=${story_key}" \
        "${extra_env[@]+"${extra_env[@]}"}" \
        "$IMAGE_NAME" \
        --dangerously-skip-permissions \
        "$@"
}

# Run container in MOUNT mode (interactive)
run_mount() {
    local container_name="$1"
    shift
    local ssh_dir
    ssh_dir=$(setup_ssh)

    docker run \
        --name "$container_name" \
        --rm -it \
        -v "$PROJECT_ROOT:/workspace" \
        -v "${HOME}/.gitconfig:/root/.gitconfig:ro" \
        -v "${HOME}/.config/gh:/root/.config/gh:ro" \
        -v "${ssh_dir}:/root/.ssh" \
        -v "/var/run/docker.sock:/var/run/docker.sock" \
        -e "CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}" \
        -e "GITHUB_TOKEN=${GITHUB_TOKEN:-}" \
        -e "GH_TOKEN=${GITHUB_TOKEN:-}" \
        "$IMAGE_NAME" \
        --dangerously-skip-permissions \
        "$@"

    rm -rf "$ssh_dir"
}

# ============================================================
# COMMANDS
# ============================================================

# --status: show running containers
if $STATUS; then
    echo -e "${GREEN}Running BMAD containers:${NC}"
    docker ps --filter "name=bmad-dev-" --format "table {{.Names}}\t{{.Status}}\t{{.RunningFor}}"
    exit 0
fi

check_token
build_image

# --wave --setup: create wave branch
if $SETUP && [[ -n "$WAVE_NUM" ]]; then
    WAVE_BRANCH="wave-${WAVE_NUM}"
    cd "$PROJECT_ROOT"

    if git show-ref --verify --quiet "refs/heads/$WAVE_BRANCH" 2>/dev/null; then
        echo -e "${YELLOW}Branch $WAVE_BRANCH already exists${NC}"
    else
        echo -e "${GREEN}Creating branch: $WAVE_BRANCH from main${NC}"
        git branch "$WAVE_BRANCH" main
        git push -u origin "$WAVE_BRANCH"
        echo -e "${GREEN}Done${NC}"
    fi
    exit 0
fi

# --wave --pipeline: full pipeline on all stories (dev → review → merge)
if $PIPELINE && [[ -n "$WAVE_NUM" ]]; then
    WAVE_BRANCH="wave-${WAVE_NUM}"

    echo -e "${GREEN}=== Wave $WAVE_NUM | FULL PIPELINE (dev → review → merge) ===${NC}"
    echo -e "  Base: $WAVE_BRANCH"
    echo ""

    STORIES=()
    while IFS= read -r line; do
        [[ -n "$line" ]] && STORIES+=("$line")
    done < <(get_wave_stories "$WAVE_NUM")

    if [[ ${#STORIES[@]} -eq 0 ]]; then
        echo -e "${RED}No stories for wave $WAVE_NUM${NC}"
        exit 1
    fi

    for s in "${STORIES[@]}"; do echo -e "  ${CYAN}$s${NC}"; done
    echo ""
    echo -e "${YELLOW}Launching ${#STORIES[@]} containers (each runs: dev-story → code-review → merge-story)${NC}"
    read -rp "Continue? [y/N] " confirm
    [[ "${confirm,,}" != "y" ]] && exit 0

    _PIPELINE_MODE=true
    for story in "${STORIES[@]}"; do
        cname="bmad-dev-${story}-pipeline"
        stop_container "$cname"
        run_clone "$cname" "$WAVE_BRANCH" "$story" \
            "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"
        echo -e "  ${GREEN}✓ $cname${NC}"
    done

    echo ""
    echo -e "${GREEN}=== ${#STORIES[@]} pipeline containers launched ===${NC}"
    echo ""
    echo "Monitor:"
    echo "  ./scripts/bmad-dev.sh --status"
    echo "  docker logs -f bmad-dev-<story>-pipeline"
    echo ""
    echo "Stop all:"
    echo "  docker ps --filter 'name=bmad-dev-' -q | xargs docker stop"
    exit 0
fi

# --wave --phase: single phase on all stories
if [[ -n "$WAVE_NUM" && -n "$PHASE" ]]; then
    WAVE_BRANCH="wave-${WAVE_NUM}"
    WORKFLOW="$(get_workflow "$PHASE")"
    MODEL="$(get_model "$PHASE")"

    if [[ -z "$WORKFLOW" ]]; then
        echo -e "${RED}Unknown phase: $PHASE${NC}"
        echo "Available: dev-story, code-review, merge-story"
        exit 1
    fi

    echo -e "${GREEN}=== Wave $WAVE_NUM | Phase: $PHASE ($MODEL) ===${NC}"
    echo -e "  Workflow:  $WORKFLOW"
    echo -e "  Base:      $WAVE_BRANCH"
    echo ""

    STORIES=()
    while IFS= read -r line; do
        [[ -n "$line" ]] && STORIES+=("$line")
    done < <(get_wave_stories "$WAVE_NUM")

    if [[ ${#STORIES[@]} -eq 0 ]]; then
        echo -e "${RED}No stories for wave $WAVE_NUM${NC}"
        exit 1
    fi

    for s in "${STORIES[@]}"; do echo -e "  ${CYAN}$s${NC}"; done
    echo ""
    echo -e "${YELLOW}Launching ${#STORIES[@]} containers ($MODEL)${NC}"
    read -rp "Continue? [y/N] " confirm
    [[ "${confirm,,}" != "y" ]] && exit 0

    for story in "${STORIES[@]}"; do
        cname="bmad-dev-${story}-${PHASE}"
        stop_container "$cname"
        run_clone "$cname" "$WAVE_BRANCH" "$story" \
            --model "$MODEL" \
            -p "$WORKFLOW" \
            "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"
        echo -e "  ${GREEN}✓ $cname${NC}"
    done

    echo ""
    echo -e "${GREEN}=== ${#STORIES[@]} containers launched ===${NC}"
    echo ""
    echo "Monitor:"
    echo "  ./scripts/bmad-dev.sh --status"
    echo "  docker logs -f bmad-dev-<story>-${PHASE}"
    echo ""
    echo "Stop all:"
    echo "  docker ps --filter 'name=bmad-dev-' -q | xargs docker stop"
    exit 0
fi

# --story --phase: single story, specific phase
if [[ -n "$STORY_NAME" && -n "$PHASE" ]]; then
    WAVE_BRANCH="${WAVE_NUM:+wave-${WAVE_NUM}}"
    WAVE_BRANCH="${WAVE_BRANCH:-main}"
    WORKFLOW="$(get_workflow "$PHASE")"
    MODEL="$(get_model "$PHASE")"

    if [[ -z "$WORKFLOW" ]]; then
        echo -e "${RED}Unknown phase: $PHASE${NC}"
        exit 1
    fi

    cname="bmad-dev-${STORY_NAME}-${PHASE}"
    stop_container "$cname"

    echo -e "${GREEN}Launching: $STORY_NAME | Phase: $PHASE ($MODEL)${NC}"
    echo -e "  Base: $WAVE_BRANCH → feat/$STORY_NAME"

    run_clone "$cname" "$WAVE_BRANCH" "$STORY_NAME" \
        --model "$MODEL" \
        -p "$WORKFLOW" \
        "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"

    echo -e "${GREEN}Container: $cname${NC}"
    echo "  docker logs -f $cname"
    exit 0
fi

# --story --pipeline: full pipeline on single story
if [[ -n "$STORY_NAME" ]] && $PIPELINE; then
    WAVE_BRANCH="${WAVE_NUM:+wave-${WAVE_NUM}}"
    WAVE_BRANCH="${WAVE_BRANCH:-main}"

    cname="bmad-dev-${STORY_NAME}-pipeline"
    stop_container "$cname"

    echo -e "${GREEN}Launching pipeline: $STORY_NAME (dev → review → merge)${NC}"
    echo -e "  Base: $WAVE_BRANCH → feat/$STORY_NAME"

    _PIPELINE_MODE=true
    run_clone "$cname" "$WAVE_BRANCH" "$STORY_NAME" \
        "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"

    echo -e "${GREEN}Container: $cname${NC}"
    echo "  docker logs -f $cname"
    exit 0
fi

# --- INTERACTIVE / CUSTOM MODE ---
CONTAINER_NAME="bmad-dev-${STORY_NAME:-${USER:-agent}-$$}"
stop_container "$CONTAINER_NAME"

echo -e "${GREEN}Launching BMAD Dev Agent (interactive)...${NC}"
echo -e "  Container: $CONTAINER_NAME"
echo ""

run_mount "$CONTAINER_NAME" "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"
