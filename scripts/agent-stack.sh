#!/usr/bin/env bash
set -euo pipefail

# ─── Agent Test Stack ─────────────────────────────────────────────────────────
# Manages the isolated agent test stack (hopeitworks-test project).
# Designed to run from INSIDE the devcontainer — no devcontainer guard.
#
# Ports: API=8081, Postgres=5433, MailHog SMTP=1026 / UI=8026
# ──────────────────────────────────────────────────────────────────────────────

# ─── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ─── Project root detection ─────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/docker-compose.agent-test.yml"
COMPOSE_PROJECT="hopeitworks-test"

BACKEND_URL="http://localhost:8081"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5433"
MAILHOG_UI_URL="http://localhost:8026"

DB_CONTAINER="hopeitworks-test-postgres"
DB_USER="${POSTGRES_USER:-hopeitworks}"
DB_NAME="hopeitworks_test"
API_CONTAINER="hopeitworks-test-api"

HEALTH_TIMEOUT=120

# ─── Helpers ───────────────────────────────────────────────────────────────────
log_info()    { echo -e "${BLUE}[agent-stack]${NC} $*"; }
log_success() { echo -e "${GREEN}[agent-stack]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[agent-stack]${NC} $*"; }
log_error()   { echo -e "${RED}[agent-stack]${NC} $*" >&2; }

check_backend() {
  curl -sf "${BACKEND_URL}/healthz" > /dev/null 2>&1 || \
  curl -sf "${BACKEND_URL}/health" > /dev/null 2>&1
}

check_postgres() {
  nc -z "${POSTGRES_HOST}" "${POSTGRES_PORT}" > /dev/null 2>&1
}

check_mailhog() {
  curl -sf "${MAILHOG_UI_URL}" > /dev/null 2>&1
}

# ─── Subcommands ───────────────────────────────────────────────────────────────

cmd_wait() {
  local timeout="${HEALTH_TIMEOUT}"
  local elapsed=0
  local interval=2

  log_info "Waiting for all services to be healthy (timeout: ${timeout}s)..."

  while [ "${elapsed}" -lt "${timeout}" ]; do
    local backend_ok=false
    local postgres_ok=false
    local mailhog_ok=false

    check_backend  && backend_ok=true
    check_postgres && postgres_ok=true
    check_mailhog  && mailhog_ok=true

    if "${backend_ok}" && "${postgres_ok}" && "${mailhog_ok}"; then
      log_success "All services are healthy."
      return 0
    fi

    local status=""
    "${backend_ok}"  || status+=" api"
    "${postgres_ok}" || status+=" postgres"
    "${mailhog_ok}"  || status+=" mailhog"

    log_info "Waiting for:${status} (${elapsed}s elapsed)..."
    sleep "${interval}"
    elapsed=$((elapsed + interval))
  done

  log_error "Timeout reached (${timeout}s). Some services are still not healthy."
  return 1
}

cmd_status() {
  echo -e "${CYAN}─── Agent Test Stack Status ────────────────────────────${NC}"

  if check_backend; then
    echo -e "  API       ${BACKEND_URL}   ${GREEN}UP${NC}"
  else
    echo -e "  API       ${BACKEND_URL}   ${RED}DOWN${NC}"
  fi

  if check_postgres; then
    echo -e "  Postgres  ${POSTGRES_HOST}:${POSTGRES_PORT}              ${GREEN}UP${NC}"
  else
    echo -e "  Postgres  ${POSTGRES_HOST}:${POSTGRES_PORT}              ${RED}DOWN${NC}"
  fi

  if check_mailhog; then
    echo -e "  MailHog   ${MAILHOG_UI_URL}    ${GREEN}UP${NC}"
  else
    echo -e "  MailHog   ${MAILHOG_UI_URL}    ${RED}DOWN${NC}"
  fi

  echo -e "${CYAN}────────────────────────────────────────────────────────${NC}"

  check_backend && check_postgres && check_mailhog
}

cmd_reset() {
  log_info "Resetting database via docker exec (no local psql tools required)..."

  # Verify the postgres container is running
  if ! docker exec "${DB_CONTAINER}" pg_isready -U "${DB_USER}" > /dev/null 2>&1; then
    log_error "Postgres container '${DB_CONTAINER}' is not running or not ready."
    return 1
  fi

  # Terminate active connections and drop/recreate database
  log_info "Dropping database ${DB_NAME}..."
  docker exec "${DB_CONTAINER}" psql -U "${DB_USER}" -d postgres \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();" \
    > /dev/null 2>&1 || true
  docker exec "${DB_CONTAINER}" psql -U "${DB_USER}" -d postgres \
    -c "DROP DATABASE IF EXISTS ${DB_NAME};"

  log_info "Creating database ${DB_NAME}..."
  docker exec "${DB_CONTAINER}" psql -U "${DB_USER}" -d postgres \
    -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};"

  # Run migrations (sorted order)
  log_info "Running migrations..."
  local migration_dir="${PROJECT_ROOT}/backend/migrations"
  local migration_count=0
  for f in "${migration_dir}"/*.up.sql; do
    [ -f "$f" ] || continue
    docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" --set ON_ERROR_STOP=1 < "$f"
    migration_count=$((migration_count + 1))
  done
  log_info "Applied ${migration_count} migrations."

  # Seed test data
  log_info "Seeding test data..."
  docker exec -i "${DB_CONTAINER}" psql -U "${DB_USER}" -d "${DB_NAME}" --set ON_ERROR_STOP=1 \
    < "${PROJECT_ROOT}/backend/testdata/seed.sql"

  log_success "Database reset complete."
  log_info "Credentials: admin@hopeitworks.dev/admin1234"

  # Restart API to pick up fresh DB state
  log_info "Restarting API container..."
  docker restart "${API_CONTAINER}"

  log_info "Waiting for API to be healthy after restart..."
  local elapsed=0
  local interval=2
  while [ "${elapsed}" -lt "${HEALTH_TIMEOUT}" ]; do
    if check_backend; then
      log_success "API is healthy."
      return 0
    fi
    sleep "${interval}"
    elapsed=$((elapsed + interval))
  done

  log_error "API did not recover within ${HEALTH_TIMEOUT}s after restart."
  return 1
}

cmd_up() {
  log_info "Starting agent test stack (project: ${COMPOSE_PROJECT})..."
  docker compose -p "${COMPOSE_PROJECT}" -f "${COMPOSE_FILE}" up -d --build

  log_info "Waiting for API to be healthy..."
  local elapsed=0
  local interval=2
  while [ "${elapsed}" -lt "${HEALTH_TIMEOUT}" ]; do
    if check_backend; then
      log_success "API is healthy."
      break
    fi
    log_info "API not ready yet (${elapsed}s)..."
    sleep "${interval}"
    elapsed=$((elapsed + interval))
    if [ "${elapsed}" -ge "${HEALTH_TIMEOUT}" ]; then
      log_error "API health timeout reached (${HEALTH_TIMEOUT}s)."
      exit 1
    fi
  done

  log_info "Seeding database..."
  cmd_reset

  log_success "Agent test stack is up and ready."
  cmd_status
}

cmd_down() {
  log_info "Stopping agent test stack (project: ${COMPOSE_PROJECT})..."
  docker compose -p "${COMPOSE_PROJECT}" -f "${COMPOSE_FILE}" down
  log_success "Agent test stack is down."
}

# ─── Entrypoint ────────────────────────────────────────────────────────────────
usage() {
  echo -e "${CYAN}Usage:${NC} $(basename "$0") <subcommand>"
  echo ""
  echo -e "  ${GREEN}up${NC}      Start agent test stack + reset DB + seed"
  echo -e "  ${GREEN}down${NC}    Stop agent test stack"
  echo -e "  ${GREEN}reset${NC}   Reset the database (drop, recreate, migrate, seed) + restart API"
  echo -e "  ${GREEN}status${NC}  Check health of API, postgres, and mailhog"
  echo -e "  ${GREEN}wait${NC}    Block until all services are healthy (timeout: ${HEALTH_TIMEOUT}s)"
  echo ""
  echo -e "${CYAN}Ports:${NC}"
  echo -e "  API       → 8081"
  echo -e "  Postgres  → 5433"
  echo -e "  MailHog   → 1026 (SMTP) / 8026 (UI)"
  echo ""
}

if [ $# -lt 1 ]; then
  usage
  exit 1
fi

case "$1" in
  up)     cmd_up     ;;
  down)   cmd_down   ;;
  reset)  cmd_reset  ;;
  status) cmd_status ;;
  wait)   cmd_wait   ;;
  *)
    log_error "Unknown subcommand: $1"
    usage
    exit 1
    ;;
esac
