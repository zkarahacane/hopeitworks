#!/usr/bin/env bash
set -euo pipefail

# BMAD Dev Agent Launcher
#
# Pipeline per story:
#   write-story (Sonnet) → dev-story (Opus) → code-review (Sonnet) → merge (Opus)
#   End of sprint: release agent
#
# Branching: main → wave-X → story/1-1-xxx
#
# Usage:
#   # Interactive (mount mode)
#   ./scripts/bmad-dev.sh
#
#   # Single story - each phase
#   ./scripts/bmad-dev.sh --story 1-1 --phase dev-story
#   ./scripts/bmad-dev.sh --story 1-1 --phase code-review
#   ./scripts/bmad-dev.sh --story 1-1 --phase merge
#
#   # Full wave - launches all stories in parallel for a phase
#   ./scripts/bmad-dev.sh --wave 1 --phase dev-story
#   ./scripts/bmad-dev.sh --wave 1 --phase code-review
#   ./scripts/bmad-dev.sh --wave 1 --phase merge
#
#   # Setup wave branch (do this first)
#   ./scripts/bmad-dev.sh --wave 1 --setup
#
#   # Sprint release
#   ./scripts/bmad-dev.sh --release
#
#   # Other
#   ./scripts/bmad-dev.sh --build             # Force rebuild image
#   ./scripts/bmad-dev.sh --status            # Show running containers
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
SETUP=false
RELEASE=false
STATUS=false
CLAUDE_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)   BUILD=true; shift ;;
        --wave)    WAVE_NUM="$2"; shift 2 ;;
        --story)   STORY_NAME="$2"; shift 2 ;;
        --phase)   PHASE="$2"; shift 2 ;;
        --setup)   SETUP=true; shift ;;
        --release) RELEASE=true; shift ;;
        --status)  STATUS=true; shift ;;
        -p|--prompt) CLAUDE_ARGS+=("-p" "$2"); shift 2 ;;
        --model)   CLAUDE_ARGS+=("--model" "$2"); shift 2 ;;
        *)         CLAUDE_ARGS+=("$1"); shift ;;
    esac
done

# Phase → BMAD workflow mapping + model
# merge has no BMAD workflow - uses a direct prompt
declare -A PHASE_WORKFLOW=(
    [write-story]="/bmad-bmm-create-story"
    [dev-story]="/bmad-bmm-dev-story"
    [code-review]="/bmad-bmm-code-review"
    [merge]="__merge_prompt__"
)
declare -A PHASE_MODEL=(
    [write-story]="sonnet"
    [dev-story]="opus"
    [code-review]="sonnet"
    [merge]="sonnet"
)

# Direct prompt for merge phase (no BMAD workflow)
MERGE_PROMPT='You are a merge agent. For the current story branch:
1. Verify all code-review findings are addressed
2. Run tests (make test or equivalent)
3. If tests pass, push the branch and create a PR to the base branch
4. Use "gh pr create" with a clear title and summary
Do NOT merge the PR - just create it for human review.'

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
    python3 -c "
import yaml
with open('$SPRINT_STATUS') as f:
    data = yaml.safe_load(f)
for w in data.get('parallel_waves', []):
    if str(w.get('wave')) == '$wave':
        for s in w.get('stories', []):
            print(s['key'])
        break
" 2>/dev/null || {
        awk "/^- wave: $wave\$/,/^- wave:/{/key:/{print \$NF}}" "$SPRINT_STATUS"
    }
}

# Run container in CLONE mode (detached)
run_clone() {
    local container_name="$1"
    local base_branch="$2"
    local story_branch="$3"
    shift 3

    local repo_url ssh_dir
    repo_url=$(get_repo_url)
    ssh_dir=$(setup_ssh)

    if [[ -z "$repo_url" ]]; then
        echo -e "${RED}Error: No git remote 'origin'${NC}"
        return 1
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
        -e "STORY_BRANCH=${story_branch}" \
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

# --release: merge wave branches to main
if $RELEASE; then
    echo -e "${GREEN}TODO: Release agent - merge wave branches to main${NC}"
    echo "Not yet implemented. Run manually or create a release workflow."
    exit 0
fi

# --wave --phase: launch full wave for a specific phase
if [[ -n "$WAVE_NUM" && -n "$PHASE" ]]; then
    WAVE_BRANCH="wave-${WAVE_NUM}"
    WORKFLOW="${PHASE_WORKFLOW[$PHASE]:-}"
    MODEL="${PHASE_MODEL[$PHASE]:-opus}"

    if [[ -z "$WORKFLOW" ]]; then
        echo -e "${RED}Unknown phase: $PHASE${NC}"
        echo "Available: write-story, dev-story, code-review, merge"
        exit 1
    fi

    echo -e "${GREEN}=== Wave $WAVE_NUM | Phase: $PHASE ($MODEL) ===${NC}"
    echo -e "  Workflow:  $WORKFLOW"
    echo -e "  Base:      $WAVE_BRANCH"
    echo ""

    mapfile -t STORIES < <(get_wave_stories "$WAVE_NUM")

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
        story_branch="story/${story}"
        stop_container "$cname"

        if [[ "$WORKFLOW" == "__merge_prompt__" ]]; then
            run_clone "$cname" "$WAVE_BRANCH" "$story_branch" \
                --model "$MODEL" \
                -p "$MERGE_PROMPT" \
                "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"
        else
            run_clone "$cname" "$WAVE_BRANCH" "$story_branch" \
                --model "$MODEL" \
                -p "$WORKFLOW" \
                "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"
        fi
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
    WORKFLOW="${PHASE_WORKFLOW[$PHASE]:-}"
    MODEL="${PHASE_MODEL[$PHASE]:-opus}"

    if [[ -z "$WORKFLOW" ]]; then
        echo -e "${RED}Unknown phase: $PHASE${NC}"
        exit 1
    fi

    cname="bmad-dev-${STORY_NAME}-${PHASE}"
    story_branch="story/${STORY_NAME}"
    stop_container "$cname"

    echo -e "${GREEN}Launching: $STORY_NAME | Phase: $PHASE ($MODEL)${NC}"
    echo -e "  Base: $WAVE_BRANCH → $story_branch"

    if [[ "$WORKFLOW" == "__merge_prompt__" ]]; then
        run_clone "$cname" "$WAVE_BRANCH" "$story_branch" \
            --model "$MODEL" \
            -p "$MERGE_PROMPT" \
            "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"
    else
        run_clone "$cname" "$WAVE_BRANCH" "$story_branch" \
            --model "$MODEL" \
            -p "$WORKFLOW" \
            "${CLAUDE_ARGS[@]+"${CLAUDE_ARGS[@]}"}"
    fi

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
