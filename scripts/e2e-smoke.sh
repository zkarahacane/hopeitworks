#!/usr/bin/env bash
set -euo pipefail

# ─── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ─── Project root detection ─────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

COMPOSE_FILE="${PROJECT_ROOT}/deploy/docker-compose.yml"
E2E_STACK="${SCRIPT_DIR}/e2e-stack.sh"
RESULTS_DIR="${PROJECT_ROOT}/frontend/e2e/real-results"
BACKEND_LOGS="${RESULTS_DIR}/backend-logs.txt"

LOG_CAPTURE_PID=""

# ─── Helpers ───────────────────────────────────────────────────────────────────
log_info()    { echo -e "${BLUE}[e2e-smoke]${NC} $*"; }
log_success() { echo -e "${GREEN}[e2e-smoke]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[e2e-smoke]${NC} $*"; }
log_error()   { echo -e "${RED}[e2e-smoke]${NC} $*" >&2; }

# ─── Cleanup on exit ───────────────────────────────────────────────────────────
cleanup() {
  if [ -n "${LOG_CAPTURE_PID}" ] && kill -0 "${LOG_CAPTURE_PID}" 2>/dev/null; then
    log_info "Stopping backend log capture (PID: ${LOG_CAPTURE_PID})..."
    kill "${LOG_CAPTURE_PID}" || true
    wait "${LOG_CAPTURE_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# ─── Step 1: Verify stack is running ───────────────────────────────────────────
log_info "Verifying E2E stack health..."
if ! "${E2E_STACK}" status; then
  log_error "Stack is not fully healthy. Run './scripts/e2e-stack.sh up' first."
  exit 1
fi
log_success "Stack is healthy."

# ─── Step 2: Reset DB ──────────────────────────────────────────────────────────
log_info "Resetting database..."
"${E2E_STACK}" reset
log_success "Database reset complete."

# ─── Step 3: Create output directory + start capturing backend logs ────────────
mkdir -p "${RESULTS_DIR}"
log_info "Starting backend log capture -> ${BACKEND_LOGS}"
docker compose -f "${COMPOSE_FILE}" logs -f api > "${BACKEND_LOGS}" 2>&1 &
LOG_CAPTURE_PID=$!
log_info "Log capture started (PID: ${LOG_CAPTURE_PID})."

# ─── Step 4: Run Playwright ────────────────────────────────────────────────────
log_info "Running Playwright E2E tests..."
PLAYWRIGHT_EXIT=0
(cd "${PROJECT_ROOT}/frontend" && npx playwright test --config playwright.e2e-real.config.ts) || PLAYWRIGHT_EXIT=$?

if [ "${PLAYWRIGHT_EXIT}" -eq 0 ]; then
  log_success "Playwright tests passed."
else
  log_warn "Playwright tests exited with code ${PLAYWRIGHT_EXIT}."
fi

# ─── Step 5: Stop log capture ──────────────────────────────────────────────────
log_info "Stopping backend log capture..."
if kill -0 "${LOG_CAPTURE_PID}" 2>/dev/null; then
  kill "${LOG_CAPTURE_PID}" || true
  wait "${LOG_CAPTURE_PID}" 2>/dev/null || true
fi
LOG_CAPTURE_PID=""
log_info "Backend logs saved to: ${BACKEND_LOGS}"

# ─── Step 6: Analyze backend logs ──────────────────────────────────────────────
PANIC_COUNT=0
ERROR_COUNT=0
WARN_COUNT=0

if [ -f "${BACKEND_LOGS}" ]; then
  PANIC_COUNT=$(grep -c "panic" "${BACKEND_LOGS}" 2>/dev/null || true)
  ERROR_COUNT=$(grep -c "ERROR" "${BACKEND_LOGS}" 2>/dev/null || true)
  WARN_COUNT=$(grep -c  "WARN"  "${BACKEND_LOGS}" 2>/dev/null || true)
fi

# ─── Step 7: Print summary report ──────────────────────────────────────────────
echo ""
echo -e "${CYAN}${BOLD}─── E2E Smoke Test Report ──────────────────────────────────${NC}"
echo ""

echo -e "  ${BOLD}Playwright results:${NC}"
if [ "${PLAYWRIGHT_EXIT}" -eq 0 ]; then
  echo -e "    Status : ${GREEN}PASSED${NC}"
else
  echo -e "    Status : ${RED}FAILED${NC} (exit code: ${PLAYWRIGHT_EXIT})"
fi

echo ""
echo -e "  ${BOLD}Backend log analysis (${BACKEND_LOGS}):${NC}"

if [ "${PANIC_COUNT}" -gt 0 ]; then
  echo -e "    Panics : ${RED}${PANIC_COUNT}${NC}"
else
  echo -e "    Panics : ${GREEN}${PANIC_COUNT}${NC}"
fi

if [ "${ERROR_COUNT}" -gt 0 ]; then
  echo -e "    ERRORs : ${RED}${ERROR_COUNT}${NC}"
else
  echo -e "    ERRORs : ${GREEN}${ERROR_COUNT}${NC}"
fi

if [ "${WARN_COUNT}" -gt 0 ]; then
  echo -e "    WARNs  : ${YELLOW}${WARN_COUNT}${NC}"
else
  echo -e "    WARNs  : ${GREEN}${WARN_COUNT}${NC}"
fi

echo ""
echo -e "${CYAN}${BOLD}────────────────────────────────────────────────────────────${NC}"
echo ""

# ─── Step 8: Exit with appropriate code ────────────────────────────────────────
FINAL_EXIT=0

if [ "${PLAYWRIGHT_EXIT}" -ne 0 ]; then
  FINAL_EXIT="${PLAYWRIGHT_EXIT}"
fi

if [ "${PANIC_COUNT}" -gt 0 ]; then
  log_error "Backend panics detected — treating as failure."
  FINAL_EXIT=1
fi

if [ "${FINAL_EXIT}" -eq 0 ]; then
  log_success "Smoke tests completed successfully."
else
  log_error "Smoke tests completed with failures."
fi

exit "${FINAL_EXIT}"
