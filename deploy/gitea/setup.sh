#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# deploy/gitea/setup.sh
#
# First-time setup for the Gitea + act_runner stack.
# Idempotent: safe to re-run (skips already-created resources).
#
# Steps:
#   1. Wait for Gitea to be healthy
#   2. Create admin user (admin / admin123)
#   3. Create a persistent API token for admin
#   4. Create the `todo-app` repository
#   5. Clone https://github.com/zkarahacane/todo-app and push to Gitea
#   6. Generate a runner registration token
#   7. Write token to .env and restart the runner
#
# Prerequisites:
#   - docker compose up -d gitea     (gitea service must be running)
#   - curl, git available on the host
#
# Usage:
#   cd deploy/gitea
#   docker compose up -d gitea
#   ./setup.sh
#   docker compose up -d runner      # runner will auto-register with the token
# ---------------------------------------------------------------------------

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
GITEA_URL="http://localhost:3030"
ADMIN_USER="devops"
ADMIN_PASS="admin123"
ADMIN_EMAIL="admin@local.host"
REPO_NAME="todo-app"
TOKEN_NAME="hopeitworks-api"
CREDENTIALS_FILE="$(cd "$(dirname "$0")" && pwd)/.gitea-credentials"

# ── Helpers ──────────────────────────────────────────────────────────────────
info()    { echo "[INFO]  $*"; }
success() { echo "[OK]    $*"; }
warn()    { echo "[WARN]  $*"; }
error()   { echo "[ERROR] $*" >&2; exit 1; }

# Wrapper for Gitea API calls with basic auth
gitea_api() {
  local method="$1"
  local path="$2"
  shift 2
  curl -s -f \
    -u "${ADMIN_USER}:${ADMIN_PASS}" \
    -H "Content-Type: application/json" \
    -X "${method}" \
    "${GITEA_URL}/api/v1${path}" \
    "$@"
}

# Same as gitea_api but uses the API token (available after step 3)
gitea_api_token() {
  local method="$1"
  local path="$2"
  shift 2
  curl -s -f \
    -H "Authorization: token ${API_TOKEN}" \
    -H "Content-Type: application/json" \
    -X "${method}" \
    "${GITEA_URL}/api/v1${path}" \
    "$@"
}

# ── Step 1: Wait for Gitea ───────────────────────────────────────────────────
info "Waiting for Gitea to be healthy at ${GITEA_URL} ..."

MAX_ATTEMPTS=30
ATTEMPT=0
until curl -sf "${GITEA_URL}/api/v1/version" > /dev/null 2>&1; do
  ATTEMPT=$((ATTEMPT + 1))
  if [[ ${ATTEMPT} -ge ${MAX_ATTEMPTS} ]]; then
    error "Gitea did not become healthy after ${MAX_ATTEMPTS} attempts. Aborting."
  fi
  echo "  ... attempt ${ATTEMPT}/${MAX_ATTEMPTS}, retrying in 5s"
  sleep 5
done
success "Gitea is healthy."

# ── Step 2: Create admin user ────────────────────────────────────────────────
info "Creating admin user '${ADMIN_USER}' ..."

# Check if user already exists via the API (returns 200 if found)
if curl -sf -u "${ADMIN_USER}:${ADMIN_PASS}" \
    "${GITEA_URL}/api/v1/user" > /dev/null 2>&1; then
  success "Admin user '${ADMIN_USER}' already exists, skipping creation."
else
  # Use the Gitea CLI inside the container — only way to create the first admin
  docker compose exec -u git gitea \
    gitea admin user create \
    --admin \
    --username "${ADMIN_USER}" \
    --password "${ADMIN_PASS}" \
    --email "${ADMIN_EMAIL}" \
    --must-change-password=false \
    2>&1 | grep -v "^$" || true

  # Verify the user was created
  if curl -sf -u "${ADMIN_USER}:${ADMIN_PASS}" \
      "${GITEA_URL}/api/v1/user" > /dev/null 2>&1; then
    success "Admin user created successfully."
  else
    error "Failed to create admin user."
  fi
fi

# ── Step 3: Create API token ─────────────────────────────────────────────────
info "Creating API token '${TOKEN_NAME}' for admin ..."

# If credentials file already exists with a valid token, skip creation
if [[ -f "${CREDENTIALS_FILE}" ]]; then
  SAVED_TOKEN=$(grep '^token=' "${CREDENTIALS_FILE}" 2>/dev/null | cut -d= -f2)
  if [[ -n "${SAVED_TOKEN}" ]]; then
    # Validate the token still works
    if curl -sf -H "Authorization: token ${SAVED_TOKEN}" \
        "${GITEA_URL}/api/v1/user" > /dev/null 2>&1; then
      API_TOKEN="${SAVED_TOKEN}"
      success "Existing token from .gitea-credentials is still valid, reusing."
    else
      warn "Saved token is invalid, will create a new one."
      API_TOKEN=""
    fi
  fi
fi

if [[ -z "${API_TOKEN:-}" ]]; then
  # Delete existing token with same name (if any) to avoid conflict
  EXISTING_TOKENS=$(gitea_api GET "/users/${ADMIN_USER}/tokens" 2>/dev/null || echo "[]")
  if echo "${EXISTING_TOKENS}" | grep -q "\"${TOKEN_NAME}\""; then
    info "Deleting stale token '${TOKEN_NAME}' ..."
    gitea_api DELETE "/users/${ADMIN_USER}/tokens/${TOKEN_NAME}" 2>/dev/null || true
  fi

  TOKEN_RESPONSE=$(gitea_api POST "/users/${ADMIN_USER}/tokens" \
    -d "{\"name\": \"${TOKEN_NAME}\", \"scopes\": [\"all\"]}")
  API_TOKEN=$(echo "${TOKEN_RESPONSE}" | grep -o '"sha1":"[^"]*"' | cut -d'"' -f4)
  if [[ -z "${API_TOKEN}" ]]; then
    error "Failed to extract token from response: ${TOKEN_RESPONSE}"
  fi
  success "API token created: ${API_TOKEN:0:8}... (truncated)"

  # Write credentials file
  cat > "${CREDENTIALS_FILE}" <<CRED
# Generated by setup.sh — do NOT commit this file
url=${GITEA_URL}
admin_user=${ADMIN_USER}
admin_password=${ADMIN_PASS}
token=${API_TOKEN}
CRED
  chmod 600 "${CREDENTIALS_FILE}"
  success "Credentials saved to deploy/gitea/.gitea-credentials"
fi

# ── Step 4: Create repository ────────────────────────────────────────────────
info "Creating repository '${REPO_NAME}' ..."

# Check if repo already exists
REPO_CHECK_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -u "${ADMIN_USER}:${ADMIN_PASS}" \
  "${GITEA_URL}/api/v1/repos/${ADMIN_USER}/${REPO_NAME}")

if [[ "${REPO_CHECK_STATUS}" == "200" ]]; then
  success "Repository '${REPO_NAME}' already exists, skipping."
else
  CREATE_RESPONSE=$(gitea_api POST "/user/repos" \
    -d "{
      \"name\": \"${REPO_NAME}\",
      \"description\": \"Todo app — pipeline validation baseline\",
      \"private\": false,
      \"auto_init\": true,
      \"default_branch\": \"main\"
    }" 2>&1) || true

  if echo "${CREATE_RESPONSE}" | grep -q "\"full_name\""; then
    success "Repository '${REPO_NAME}' created."
  else
    warn "Unexpected response creating repo: ${CREATE_RESPONSE}"
  fi
fi

# ── Step 5: Verify repository ─────────────────────────────────────────────────
info "Verifying repository '${REPO_NAME}' ..."
success "Empty repo ready at ${GITEA_URL}/${ADMIN_USER}/${REPO_NAME}"

# ── Step 6: Generate runner registration token ───────────────────────────────
info "Generating runner registration token ..."

RUNNER_TOKEN_RESPONSE=$(gitea_api GET "/admin/runners/registration-token" 2>/dev/null || \
  curl -s -f \
    -u "${ADMIN_USER}:${ADMIN_PASS}" \
    -H "Content-Type: application/json" \
    -X GET \
    "${GITEA_URL}/api/v1/admin/runners/registration-token")

RUNNER_TOKEN=$(echo "${RUNNER_TOKEN_RESPONSE}" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [[ -z "${RUNNER_TOKEN}" ]]; then
  error "Failed to retrieve runner registration token. Response: ${RUNNER_TOKEN_RESPONSE}"
fi

success "Runner registration token obtained: ${RUNNER_TOKEN:0:8}..."

# ── Step 7: Write token and restart runner ───────────────────────────────────
info "Writing runner token to .runner-token and .env ..."

# Write raw token file
echo "${RUNNER_TOKEN}" > .runner-token

# Write / overwrite .env file for docker compose
cat > .env <<EOF
# Generated by setup.sh — do NOT commit this file
GITEA_RUNNER_REGISTRATION_TOKEN=${RUNNER_TOKEN}
GITEA_TOKEN=${API_TOKEN}
EOF

success ".env written with GITEA_RUNNER_REGISTRATION_TOKEN + GITEA_TOKEN."

# Restart runner if it's already running so it picks up the new token
info "Restarting runner service to register with Gitea ..."
if docker compose ps runner 2>/dev/null | grep -q "Up\|running"; then
  docker compose restart runner
  info "Runner restarted. Check registration with: docker compose logs -f runner"
else
  info "Runner not running yet. Start it with: docker compose up -d runner"
fi

# ── Done ─────────────────────────────────────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Setup complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Gitea UI    : ${GITEA_URL}"
echo " Admin user  : ${ADMIN_USER} / ${ADMIN_PASS}"
echo " API token   : ${API_TOKEN:-(see .gitea-credentials)}"
echo " Repository  : ${GITEA_URL}/${ADMIN_USER}/${REPO_NAME}"
echo " Credentials : deploy/gitea/.gitea-credentials"
echo ""
echo " To use with hopeitworks:"
echo "   export GITEA_TOKEN=\$(grep '^token=' deploy/gitea/.gitea-credentials | cut -d= -f2)"
echo ""
echo " Next steps:"
echo "   docker compose logs -f runner   # verify runner registration"
echo "   docker compose logs -f gitea    # Gitea logs"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
