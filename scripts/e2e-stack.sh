#!/usr/bin/env bash
set -euo pipefail

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

COMPOSE_FILE="${PROJECT_ROOT}/deploy/docker-compose.yml"
FRONTEND_PID_FILE="${PROJECT_ROOT}/.e2e-frontend.pid"

BACKEND_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:5173"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5432"

HEALTH_TIMEOUT=120

# ─── Helpers ───────────────────────────────────────────────────────────────────
log_info()    { echo -e "${BLUE}[e2e-stack]${NC} $*"; }
log_success() { echo -e "${GREEN}[e2e-stack]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[e2e-stack]${NC} $*"; }
log_error()   { echo -e "${RED}[e2e-stack]${NC} $*" >&2; }

check_backend() {
  curl -sf "${BACKEND_URL}/healthz" > /dev/null 2>&1 || \
  curl -sf "${BACKEND_URL}/health" > /dev/null 2>&1
}

check_frontend() {
  curl -sf "${FRONTEND_URL}" > /dev/null 2>&1
}

check_postgres() {
  nc -z "${POSTGRES_HOST}" "${POSTGRES_PORT}" > /dev/null 2>&1
}

# ─── Subcommands ───────────────────────────────────────────────────────────────

cmd_wait() {
  local timeout="${HEALTH_TIMEOUT}"
  local elapsed=0
  local interval=2

  log_info "Waiting for all services to be healthy (timeout: ${timeout}s)..."

  while [ "${elapsed}" -lt "${timeout}" ]; do
    local backend_ok=false
    local frontend_ok=false
    local postgres_ok=false

    check_backend  && backend_ok=true
    check_frontend && frontend_ok=true
    check_postgres && postgres_ok=true

    if "${backend_ok}" && "${frontend_ok}" && "${postgres_ok}"; then
      log_success "All services are healthy."
      return 0
    fi

    local status=""
    "${backend_ok}"  || status+=" backend"
    "${frontend_ok}" || status+=" frontend"
    "${postgres_ok}" || status+=" postgres"

    log_info "Waiting for:${status} (${elapsed}s elapsed)..."
    sleep "${interval}"
    elapsed=$((elapsed + interval))
  done

  log_error "Timeout reached (${timeout}s). Some services are still not healthy."
  return 1
}

cmd_status() {
  echo -e "${CYAN}─── E2E Stack Status ───────────────────────────────────${NC}"

  if check_backend; then
    echo -e "  Backend   ${BACKEND_URL}   ${GREEN}UP${NC}"
  else
    echo -e "  Backend   ${BACKEND_URL}   ${RED}DOWN${NC}"
  fi

  if check_frontend; then
    echo -e "  Frontend  ${FRONTEND_URL}  ${GREEN}UP${NC}"
  else
    echo -e "  Frontend  ${FRONTEND_URL}  ${RED}DOWN${NC}"
  fi

  if check_postgres; then
    echo -e "  Postgres  ${POSTGRES_HOST}:${POSTGRES_PORT}           ${GREEN}UP${NC}"
  else
    echo -e "  Postgres  ${POSTGRES_HOST}:${POSTGRES_PORT}           ${RED}DOWN${NC}"
  fi

  echo -e "${CYAN}────────────────────────────────────────────────────────${NC}"

  check_backend && check_frontend && check_postgres
}

cmd_reset() {
  log_info "Resetting database..."
  (cd "${PROJECT_ROOT}/backend" && make reset-db)
  log_success "Database reset complete."
}

cmd_up() {
  log_info "Starting docker-compose stack..."
  docker compose -f "${COMPOSE_FILE}" up -d --build

  log_info "Waiting for backend to be healthy..."
  local elapsed=0
  local interval=2
  while [ "${elapsed}" -lt "${HEALTH_TIMEOUT}" ]; do
    if check_backend; then
      log_success "Backend is healthy."
      break
    fi
    log_info "Backend not ready yet (${elapsed}s)..."
    sleep "${interval}"
    elapsed=$((elapsed + interval))
    if [ "${elapsed}" -ge "${HEALTH_TIMEOUT}" ]; then
      log_error "Backend health timeout reached (${HEALTH_TIMEOUT}s)."
      exit 1
    fi
  done

  log_info "Resetting database..."
  cmd_reset

  log_info "Starting frontend dev server in background..."
  (cd "${PROJECT_ROOT}/frontend" && exec npm run dev) &
  echo $! > "${FRONTEND_PID_FILE}"
  log_success "Frontend started (PID: $(cat "${FRONTEND_PID_FILE}"))."

  log_info "Waiting for frontend to be ready..."
  local fe_elapsed=0
  while [ "${fe_elapsed}" -lt "${HEALTH_TIMEOUT}" ]; do
    if check_frontend; then
      log_success "Frontend is healthy."
      break
    fi
    sleep "${interval}"
    fe_elapsed=$((fe_elapsed + interval))
    if [ "${fe_elapsed}" -ge "${HEALTH_TIMEOUT}" ]; then
      log_error "Frontend health timeout reached (${HEALTH_TIMEOUT}s)."
      exit 1
    fi
  done

  log_success "E2E stack is up and ready."
  cmd_status
}

cmd_down() {
  log_info "Stopping E2E stack..."

  if [ -f "${FRONTEND_PID_FILE}" ]; then
    local pid
    pid="$(cat "${FRONTEND_PID_FILE}")"
    if kill -0 "${pid}" 2>/dev/null; then
      log_info "Killing frontend dev server (PID: ${pid})..."
      kill "${pid}" || true
    else
      log_warn "Frontend PID ${pid} not running."
    fi
    rm -f "${FRONTEND_PID_FILE}"
  else
    log_warn "No frontend PID file found at ${FRONTEND_PID_FILE}."
  fi

  log_info "Bringing down docker-compose stack..."
  docker compose -f "${COMPOSE_FILE}" down

  log_success "E2E stack is down."
}

# ─── Entrypoint ────────────────────────────────────────────────────────────────
usage() {
  echo -e "${CYAN}Usage:${NC} $(basename "$0") <subcommand>"
  echo ""
  echo -e "  ${GREEN}up${NC}      Start docker-compose stack + reset DB + start frontend dev server"
  echo -e "  ${GREEN}down${NC}    Stop frontend dev server + docker-compose stack"
  echo -e "  ${GREEN}reset${NC}   Reset the database (drop, recreate, migrate, seed)"
  echo -e "  ${GREEN}status${NC}  Check health of backend, frontend, and postgres"
  echo -e "  ${GREEN}wait${NC}    Block until all services are healthy (timeout: ${HEALTH_TIMEOUT}s)"
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
