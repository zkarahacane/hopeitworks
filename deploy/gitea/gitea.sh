#!/usr/bin/env bash
# gitea.sh — Start Gitea + act_runner and auto-provision for hopeitworks
#
# Usage:
#   ./deploy/gitea/gitea.sh              # Start Gitea if needed, auto-setup, create todo-app repo
#   ./deploy/gitea/gitea.sh --start      # Start Gitea stack only (no setup)
#   ./deploy/gitea/gitea.sh --stop       # Stop Gitea stack
#   ./deploy/gitea/gitea.sh --reset      # Full teardown (removes volumes + stored credentials)
#
# On first run, the script automatically:
#   1. Starts Gitea
#   2. Creates the admin user (devops)
#   3. Generates an API token (all scopes)
#   4. Creates the todo-app repository
#   5. Generates a runner registration token
#   6. Writes .env + .gitea-credentials + .gitea-token
#   7. Restarts act_runner with the registration token
#
# No manual steps needed — just run the script.

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
GITEA_HOST="http://localhost:3030"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

ADMIN_USER="devops"
ADMIN_EMAIL="admin@local.host"
ADMIN_PASSWORD="admin123"
REPO_NAME="todo-app"
TOKEN_NAME="hopeitworks-api"

# Credentials stored locally, gitignored
TOKEN_FILE="${SCRIPT_DIR}/.gitea-token"
CREDENTIALS_FILE="${SCRIPT_DIR}/.gitea-credentials"
ENV_FILE="${SCRIPT_DIR}/.env"

HEALTH_TIMEOUT=120

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log()  { echo "[gitea] $*"; }
warn() { echo "[gitea] WARN: $*" >&2; }
die()  { echo "[gitea] ERROR: $*" >&2; exit 1; }

usage() {
  grep '^#' "$0" | grep -v '#!/' | sed 's/^# //' | sed 's/^#//'
  exit 0
}

gitea_api() {
  local method="$1" path="$2"; shift 2
  curl -sf -X "${method}" "${GITEA_HOST}/api/v1${path}" \
    -H "Content-Type: application/json" "$@"
}

gitea_api_auth() {
  local method="$1" path="$2"; shift 2
  local token; token="$(cat "${TOKEN_FILE}")"
  gitea_api "${method}" "${path}" -H "Authorization: token ${token}" "$@"
}

# ---------------------------------------------------------------------------
# Parse flags
# ---------------------------------------------------------------------------
for arg in "$@"; do
  case "$arg" in
    --start)
      log "Starting Gitea stack..."
      docker compose -f "$COMPOSE_FILE" up -d
      log "Gitea started. UI: ${GITEA_HOST}"
      exit 0
      ;;
    --stop)
      log "Stopping Gitea stack..."
      docker compose -f "$COMPOSE_FILE" down
      log "Gitea stopped."
      exit 0
      ;;
    --reset)
      log "Full teardown: removing containers, volumes, and stored credentials..."
      docker compose -f "$COMPOSE_FILE" down -v
      rm -f "$TOKEN_FILE" "$CREDENTIALS_FILE" "$ENV_FILE"
      log "Done. Next run will do a fresh setup."
      exit 0
      ;;
    --help|-h) usage ;;
    *)         die "Unknown argument: $arg" ;;
  esac
done

# ---------------------------------------------------------------------------
# Check Gitea health
# ---------------------------------------------------------------------------
gitea_is_up() {
  curl -sf --max-time 3 "${GITEA_HOST}/api/v1/version" > /dev/null 2>&1
}

# ---------------------------------------------------------------------------
# Start Gitea if not running
# ---------------------------------------------------------------------------
start_gitea() {
  log "Starting Gitea server..."
  docker compose -f "$COMPOSE_FILE" up -d gitea

  log "Waiting for Gitea to be ready (timeout: ${HEALTH_TIMEOUT}s)..."
  local waited=0 interval=5

  while ! gitea_is_up; do
    if [[ $waited -ge $HEALTH_TIMEOUT ]]; then
      die "Gitea did not become ready after ${HEALTH_TIMEOUT}s. Check logs: docker logs gitea_server"
    fi
    printf "."
    sleep $interval
    waited=$((waited + interval))
  done
  echo ""
  log "Gitea is ready at ${GITEA_HOST}"
}

# ---------------------------------------------------------------------------
# Check if setup is needed
# ---------------------------------------------------------------------------
needs_setup() {
  # If we can authenticate with devops:admin123, user already exists → no setup needed
  local body
  body=$(curl -sf -u "${ADMIN_USER}:${ADMIN_PASSWORD}" \
    "${GITEA_HOST}/api/v1/user" 2>/dev/null || echo "")
  if echo "$body" | grep -q '"login"'; then
    return 1  # user exists, no setup needed
  fi
  return 0
}

# ---------------------------------------------------------------------------
# Load token
# ---------------------------------------------------------------------------
load_token() {
  if [[ -f "$TOKEN_FILE" ]]; then
    GITEA_TOKEN=$(cat "$TOKEN_FILE")
    # Validate it still works
    if curl -sf -H "Authorization: token ${GITEA_TOKEN}" \
        "${GITEA_HOST}/api/v1/user" > /dev/null 2>&1; then
      return 0
    fi
    warn "Stored token is invalid."
    rm -f "$TOKEN_FILE"
  fi
  return 1
}

# ---------------------------------------------------------------------------
# Auto-setup on first run
# ---------------------------------------------------------------------------
setup_gitea() {
  log "First run detected — setting up Gitea automatically..."

  # Step 1: Create admin user via CLI inside container
  log "Creating admin user '${ADMIN_USER}'..."
  docker compose -f "$COMPOSE_FILE" exec -u git gitea \
    gitea admin user create \
      --admin \
      --username "${ADMIN_USER}" \
      --password "${ADMIN_PASSWORD}" \
      --email "${ADMIN_EMAIL}" \
      --must-change-password=false \
    2>&1 | grep -v "^$" || true

  # Verify user was created
  if ! curl -sf -u "${ADMIN_USER}:${ADMIN_PASSWORD}" \
      "${GITEA_HOST}/api/v1/user" > /dev/null 2>&1; then
    die "Failed to create admin user."
  fi
  log "Admin user created."

  # Step 2: Generate API token
  log "Generating API token '${TOKEN_NAME}'..."
  # Delete existing token with same name (idempotent)
  local existing
  existing=$(curl -sf -u "${ADMIN_USER}:${ADMIN_PASSWORD}" \
    "${GITEA_HOST}/api/v1/users/${ADMIN_USER}/tokens" 2>/dev/null || echo "[]")
  if echo "${existing}" | grep -q "\"${TOKEN_NAME}\""; then
    curl -sf -u "${ADMIN_USER}:${ADMIN_PASSWORD}" \
      -X DELETE "${GITEA_HOST}/api/v1/users/${ADMIN_USER}/tokens/${TOKEN_NAME}" \
      > /dev/null 2>&1 || true
  fi

  local token_response
  token_response=$(curl -sf \
    -u "${ADMIN_USER}:${ADMIN_PASSWORD}" \
    -X POST "${GITEA_HOST}/api/v1/users/${ADMIN_USER}/tokens" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"${TOKEN_NAME}\",\"scopes\":[\"all\"]}")

  local token
  token=$(echo "$token_response" | grep -o '"sha1":"[^"]*"' | cut -d'"' -f4)
  if [[ -z "$token" ]]; then
    die "Failed to extract token. Response: ${token_response}"
  fi

  echo "$token" > "$TOKEN_FILE"
  chmod 600 "$TOKEN_FILE"
  log "API token created: ${token:0:8}..."

  # Step 3: Create repository
  log "Creating repository '${REPO_NAME}'..."
  if gitea_api_auth GET "/repos/${ADMIN_USER}/${REPO_NAME}" > /dev/null 2>&1; then
    log "Repository '${REPO_NAME}' already exists — skipping."
  else
    gitea_api_auth POST "/user/repos" \
      -d "{
        \"name\": \"${REPO_NAME}\",
        \"description\": \"Reference todo app for pipeline validation\",
        \"private\": false,
        \"auto_init\": true,
        \"default_branch\": \"main\"
      }" > /dev/null || die "Failed to create repository '${REPO_NAME}'."
    log "Repository '${REPO_NAME}' created."
  fi

  # Step 4: Runner registration token
  log "Generating runner registration token..."
  local runner_response
  runner_response=$(gitea_api_auth GET "/admin/runners/registration-token") \
    || die "Failed to get runner registration token."

  local runner_token
  runner_token=$(echo "$runner_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
  [[ -n "$runner_token" ]] || die "Could not extract runner token."
  log "Runner token obtained: ${runner_token:0:8}..."

  # Step 5: Store credentials
  cat > "$CREDENTIALS_FILE" <<CRED
url=${GITEA_HOST}
login=${ADMIN_USER}
password=${ADMIN_PASSWORD}
token=${token}
CRED
  chmod 600 "$CREDENTIALS_FILE"

  cat > "$ENV_FILE" <<ENV
# Generated by gitea.sh — do NOT commit
GITEA_RUNNER_REGISTRATION_TOKEN=${runner_token}
ENV

  log "Credentials saved to deploy/gitea/.gitea-credentials"

  # Step 6: Restart runner with the new token
  log "Restarting act_runner..."
  docker compose -f "$COMPOSE_FILE" up -d act_runner --force-recreate \
    || warn "Could not restart runner. Start it manually: docker compose -f deploy/gitea/docker-compose.yml up -d"

  log "Setup complete."
}

# ---------------------------------------------------------------------------
# Main flow
# ---------------------------------------------------------------------------

# Ensure Gitea is running
if gitea_is_up; then
  log "Gitea is already running at ${GITEA_HOST}"
else
  start_gitea
fi

# Auto-setup if needed (first run)
if ! load_token; then
  if needs_setup; then
    setup_gitea
    load_token || die "Token file not found after setup"
  else
    # User exists but token is invalid — regenerate
    log "Regenerating API token..."
    setup_gitea
    load_token || die "Token file not found after setup"
  fi
fi

log "Gitea is ready."
log "  UI:          ${GITEA_HOST}"
log "  Admin:       ${ADMIN_USER} / ${ADMIN_PASSWORD}"
log "  Repository:  ${GITEA_HOST}/${ADMIN_USER}/${REPO_NAME}"
log "  Credentials: deploy/gitea/.gitea-credentials"
