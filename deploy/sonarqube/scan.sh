#!/usr/bin/env bash
# sonar-scan.sh — Run SonarQube analysis for the hopeitworks monorepo
#
# Usage:
#   ./deploy/sonarqube/scan.sh              # Start SonarQube if needed, auto-setup, then scan
#   ./deploy/sonarqube/scan.sh --start      # Start SonarQube stack only (no scan)
#   ./deploy/sonarqube/scan.sh --stop       # Stop SonarQube stack
#   ./deploy/sonarqube/scan.sh --reset      # Full teardown (removes volumes + stored token)
#
# On first run, the script automatically:
#   1. Starts SonarQube
#   2. Changes the default admin password
#   3. Generates an analysis token
#   4. Stores it locally for future scans
#
# No manual steps needed — just run the script.

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SONAR_HOST="http://localhost:9000"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SONAR_COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SCANNER_IMAGE="sonarsource/sonar-scanner-cli:latest"
SONAR_NETWORK="sonarqube"

# Credentials stored locally, gitignored
TOKEN_FILE="${SCRIPT_DIR}/.sonar-token"
CREDENTIALS_FILE="${SCRIPT_DIR}/.sonar-credentials"

# Default admin credentials (SonarQube ships with these)
DEFAULT_PASSWORD="admin"
# Auto-generated password for the admin account after setup
# Must satisfy SonarQube password policy: uppercase + lowercase + digit + 12 chars
ADMIN_PASSWORD="Hpw-Sonar-$(date +%s | shasum | head -c 8)"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log()  { echo "[sonar] $*"; }
warn() { echo "[sonar] WARN: $*" >&2; }
die()  { echo "[sonar] ERROR: $*" >&2; exit 1; }

usage() {
  grep '^#' "$0" | grep -v '#!/' | sed 's/^# //' | sed 's/^#//'
  exit 0
}

# ---------------------------------------------------------------------------
# Parse flags
# ---------------------------------------------------------------------------
SCAN=true
for arg in "$@"; do
  case "$arg" in
    --start)  SCAN=false ;;
    --stop)
      log "Stopping SonarQube stack..."
      docker compose -f "$SONAR_COMPOSE_FILE" down
      log "SonarQube stopped."
      exit 0
      ;;
    --reset)
      log "Full teardown: removing containers, volumes, and stored token..."
      docker compose -f "$SONAR_COMPOSE_FILE" down -v
      rm -f "$TOKEN_FILE"
      log "Done. Next run will do a fresh setup."
      exit 0
      ;;
    --help|-h) usage ;;
    *)         die "Unknown argument: $arg" ;;
  esac
done

# ---------------------------------------------------------------------------
# Check SonarQube health
# ---------------------------------------------------------------------------
sonar_is_up() {
  local status
  status=$(curl -s --max-time 3 "${SONAR_HOST}/api/system/status" 2>/dev/null || echo "")
  echo "$status" | grep -q '"status":"UP"'
}

# ---------------------------------------------------------------------------
# Start SonarQube if not running
# ---------------------------------------------------------------------------
start_sonarqube() {
  log "Starting SonarQube stack..."
  docker compose -f "$SONAR_COMPOSE_FILE" up -d

  log "Waiting for SonarQube to be ready (this can take 60-90s on first start)..."
  local max_wait=180
  local waited=0
  local interval=5

  while ! sonar_is_up; do
    if [[ $waited -ge $max_wait ]]; then
      die "SonarQube did not become ready after ${max_wait}s. Check logs: docker logs sonar_server"
    fi
    printf "."
    sleep $interval
    waited=$((waited + interval))
  done
  echo ""
  log "SonarQube is ready at ${SONAR_HOST}"
}

# ---------------------------------------------------------------------------
# Auto-setup: change password + generate token on first run
# ---------------------------------------------------------------------------
setup_sonarqube() {
  log "First run detected — setting up SonarQube automatically..."

  # Step 1: Change default admin password
  log "Changing default admin password..."
  local pw_response
  pw_response=$(curl -s -o /dev/null -w "%{http_code}" \
    -u "admin:${DEFAULT_PASSWORD}" \
    -X POST "${SONAR_HOST}/api/users/change_password" \
    -d "login=admin&previousPassword=${DEFAULT_PASSWORD}&password=${ADMIN_PASSWORD}")

  if [[ "$pw_response" != "204" ]]; then
    die "Failed to change admin password (HTTP ${pw_response}). Is this a fresh install?"
  fi
  log "Admin password changed."

  # Step 2: Generate analysis token
  log "Generating analysis token..."
  local token_response
  token_response=$(curl -s \
    -u "admin:${ADMIN_PASSWORD}" \
    -X POST "${SONAR_HOST}/api/user_tokens/generate" \
    -d "name=cli-scan&type=GLOBAL_ANALYSIS_TOKEN")

  local token
  token=$(echo "$token_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

  if [[ -z "$token" ]]; then
    die "Failed to generate token. Response: ${token_response}"
  fi

  # Step 3: Store credentials locally
  mkdir -p "$(dirname "$TOKEN_FILE")"
  echo "$token" > "$TOKEN_FILE"
  cat > "$CREDENTIALS_FILE" <<CRED
url=${SONAR_HOST}
login=admin
password=${ADMIN_PASSWORD}
token=${token}
CRED
  chmod 600 "$CREDENTIALS_FILE"
  chmod 600 "$TOKEN_FILE"

  log "Credentials saved to deploy/sonarqube/.sonar-credentials"
  log "Setup complete."
}

# ---------------------------------------------------------------------------
# Load token
# ---------------------------------------------------------------------------
load_token() {
  if [[ -f "$TOKEN_FILE" ]]; then
    SONAR_TOKEN=$(cat "$TOKEN_FILE")
    return 0
  fi
  return 1
}

# Check if setup is needed (default admin:admin still works)
needs_setup() {
  local status_code
  status_code=$(curl -s -o /dev/null -w "%{http_code}" \
    -u "admin:${DEFAULT_PASSWORD}" \
    "${SONAR_HOST}/api/authentication/validate" 2>/dev/null || echo "000")
  # If we can authenticate with admin:admin, setup hasn't been done
  local body
  body=$(curl -s -u "admin:${DEFAULT_PASSWORD}" \
    "${SONAR_HOST}/api/authentication/validate" 2>/dev/null || echo "")
  echo "$body" | grep -q '"valid":true'
}

# ---------------------------------------------------------------------------
# Main flow
# ---------------------------------------------------------------------------

# Ensure SonarQube is running
if sonar_is_up; then
  log "SonarQube is already running at ${SONAR_HOST}"
else
  start_sonarqube
fi

# Auto-setup if needed (first run)
if ! load_token; then
  if needs_setup; then
    setup_sonarqube
    load_token || die "Token file not found after setup"
  else
    die "No token found and default credentials don't work. Run: ./deploy/sonarqube/scan.sh --reset"
  fi
fi

# If --start only, we're done
if [[ "$SCAN" == false ]]; then
  log "SonarQube stack started. UI: ${SONAR_HOST}"
  exit 0
fi

# ---------------------------------------------------------------------------
# Run the scanner
# ---------------------------------------------------------------------------
log "Running sonar-scanner via Docker..."
log "Project root: ${PROJECT_ROOT}"

docker run --rm \
  --network "${SONAR_NETWORK}" \
  -v "${PROJECT_ROOT}:/usr/src" \
  -w /usr/src \
  -e SONAR_HOST_URL="http://sonarqube:9000" \
  "${SCANNER_IMAGE}" \
  sonar-scanner \
  -Dsonar.token="${SONAR_TOKEN}" \
  -Dsonar.host.url="http://sonarqube:9000" \
  -Dsonar.projectBaseDir=/usr/src

log "Scan complete. Results: ${SONAR_HOST}/dashboard?id=hopeitworks"
