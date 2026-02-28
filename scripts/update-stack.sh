#!/usr/bin/env bash
# =============================================================================
# update-stack.sh — Rebuild and restart the local docker-compose stack
# =============================================================================
# Safe to run from the devcontainer: rebuilds images and restarts containers
# without touching the database volume (no data loss).
#
# Usage:
#   ./scripts/update-stack.sh              # rebuild + restart
#   ./scripts/update-stack.sh --reset      # rebuild + restart + reseed database
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/deploy/docker-compose.yml"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[+]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
fail() { echo -e "${RED}[✗]${NC} $1"; exit 1; }

DO_RESET=false
for arg in "$@"; do
  case "$arg" in
    --reset) DO_RESET=true ;;
    *) fail "Unknown flag: $arg. Usage: $0 [--reset]" ;;
  esac
done

# ─── Rebuild and restart ─────────────────────────────────────────────────────
log "Rebuilding and restarting docker-compose stack..."
docker compose -f "$COMPOSE_FILE" up --build -d

log "Waiting for API to be ready..."
for i in $(seq 1 60); do
  if curl -sf http://localhost:8080/api/v1/auth/login > /dev/null 2>&1; then
    break
  fi
  sleep 1
done
curl -sf http://localhost:8080/api/v1/auth/login > /dev/null 2>&1 || fail "API not responding after 60s"
log "API ready"

# ─── Optional reset ──────────────────────────────────────────────────────────
if $DO_RESET; then
  warn "Resetting database (--reset flag)..."
  FORCE_RESET=1 "$SCRIPT_DIR/reset-dev.sh"
fi

log "Stack updated."
docker compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
