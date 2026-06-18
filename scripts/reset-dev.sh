#!/usr/bin/env bash
# =============================================================================
# reset-dev.sh — Reset local dev environment and seed via API
# =============================================================================
# Usage:  ./scripts/reset-dev.sh
#
# What it does:
#   1. Drops and recreates the DB schema
#   2. Restarts the API (triggers migrations)
#   3. Creates all test data via API calls
#
# Credentials after reset:
#   admin@hopeitworks.dev  / admin1234  (admin)
#   sarah@hopeitworks.dev  / admin1234  (admin)
#   marc@hopeitworks.dev   / admin1234  (admin)
#   dev@hopeitworks.dev    / user1234   (user)
#   alice@hopeitworks.dev  / user1234   (user)
#   bob@hopeitworks.dev    / user1234   (user)
# =============================================================================

set -euo pipefail

# ─── Devcontainer guard ───────────────────────────────────────────────────────
# The devcontainer shares the host Docker socket. Running reset-dev.sh from
# inside the devcontainer would destroy the local stack's data. Use FORCE_RESET=1
# to override if you really know what you're doing.
if [[ "${HOPEITWORKS_ENV:-}" == "devcontainer" && "${FORCE_RESET:-0}" != "1" ]]; then
  echo -e "\033[0;31m[✗] reset-dev.sh is blocked inside the devcontainer.\033[0m"
  echo "    The devcontainer shares the host Docker socket — running this script"
  echo "    would destroy the local stack's persistent data."
  echo ""
  echo "    Run this script from your host machine instead, or set FORCE_RESET=1"
  echo "    to override: FORCE_RESET=1 ./scripts/reset-dev.sh"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deploy"

API="http://localhost:8080/api/v1"
COOKIE_JAR=$(mktemp)
trap 'rm -f "$COOKIE_JAR"' EXIT

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[+]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
fail() { echo -e "${RED}[✗]${NC} $1"; exit 1; }

# Helper: POST/PUT with cookie auth, return body
api_post() {
  local method="$1" path="$2" body="$3"
  local resp
  resp=$(curl -s -w "\n%{http_code}" -X "$method" "$API$path" \
    -H 'Content-Type: application/json' \
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    -d "$body")
  local code=$(echo "$resp" | tail -1)
  local data=$(echo "$resp" | sed '$d')
  if [[ "$code" -ge 400 ]]; then
    warn "$method $path → HTTP $code: $data"
    return 1
  fi
  echo "$data"
}

# ---------------------------------------------------------------------------
# 1. Reset database
# ---------------------------------------------------------------------------
log "Resetting database..."
docker exec hopeitworks-postgres psql -U hopeitworks -d hopeitworks_dev -q -c \
  "DROP SCHEMA public CASCADE; CREATE SCHEMA public; CREATE EXTENSION IF NOT EXISTS pgcrypto;" \
  2>/dev/null

# ---------------------------------------------------------------------------
# 2. Restart API (triggers migrations)
# ---------------------------------------------------------------------------
log "Restarting API (migrations)..."
docker compose -f "$COMPOSE_DIR/docker-compose.yml" restart api > /dev/null 2>&1

log "Waiting for API to be ready..."
for i in $(seq 1 30); do
  if curl -s -o /dev/null "$API/auth/login" 2>/dev/null; then
    break
  fi
  sleep 1
done

# Quick health check
curl -s -o /dev/null "$API/auth/login" 2>/dev/null || fail "API not responding after 30s"
log "API ready"

# ---------------------------------------------------------------------------
# 3. Register users
# ---------------------------------------------------------------------------
log "Registering users..."

register_user() {
  local email="$1" password="$2" name="$3"
  api_post POST "/auth/register" "{\"email\":\"$email\",\"password\":\"$password\",\"name\":\"$name\"}" > /dev/null
}

register_user "admin@hopeitworks.dev" "admin1234" "Admin User"
register_user "sarah@hopeitworks.dev" "admin1234" "Sarah Chen"
register_user "marc@hopeitworks.dev"  "admin1234"  "Marc Dupont"
register_user "dev@hopeitworks.dev"   "user1234"   "Dev User"
register_user "alice@hopeitworks.dev" "user1234" "Alice Martin"
register_user "bob@hopeitworks.dev"   "user1234"   "Bob Nguyen"

log "6 users registered"

# ---------------------------------------------------------------------------
# 4. Bootstrap admin (only SQL needed — chicken-and-egg)
# ---------------------------------------------------------------------------
log "Promoting admins..."
docker exec hopeitworks-postgres psql -U hopeitworks -d hopeitworks_dev -q -c \
  "UPDATE users SET role='admin' WHERE email IN ('admin@hopeitworks.dev','sarah@hopeitworks.dev','marc@hopeitworks.dev');" \
  2>/dev/null

# ---------------------------------------------------------------------------
# 5. Login as admin
# ---------------------------------------------------------------------------
log "Logging in as admin..."
api_post POST "/auth/login" '{"email":"admin@hopeitworks.dev","password":"admin1234"}' > /dev/null

# ---------------------------------------------------------------------------
# 6. Create project
# ---------------------------------------------------------------------------
log "Creating project..."
PROJECT=$(api_post POST "/projects" '{
  "name": "Todo App",
  "description": "Simple todo application — Go backend + Vue 3 frontend. Used as test project for pipeline validation.",
  "repo_url": "https://github.com/zkarahacane/todo-app",
  "git_provider": "github",
  "git_token_env": "GITHUB_TOKEN",
  "default_model": "claude-sonnet-4-6"
}')
PROJECT_ID=$(echo "$PROJECT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
log "Project created: $PROJECT_ID"

# ---------------------------------------------------------------------------
# 7. Add members
# ---------------------------------------------------------------------------
log "Adding project members..."
# Get user IDs
get_user_id() {
  local users
  users=$(curl -s -b "$COOKIE_JAR" "$API/users")
  echo "$users" | python3 -c "
import sys, json
data = json.load(sys.stdin)
users = data.get('data', data) if isinstance(data, dict) else data
for u in users:
    if u['email'] == '$1':
        print(u['id'])
        break
"
}

DEV_ID=$(get_user_id "dev@hopeitworks.dev")
ALICE_ID=$(get_user_id "alice@hopeitworks.dev")
BOB_ID=$(get_user_id "bob@hopeitworks.dev")
SARAH_ID=$(get_user_id "sarah@hopeitworks.dev")

for uid in "$DEV_ID" "$ALICE_ID" "$BOB_ID" "$SARAH_ID"; do
  if [[ -n "$uid" ]]; then
    api_post POST "/projects/$PROJECT_ID/users" "{\"user_id\":\"$uid\",\"role\":\"member\"}" > /dev/null 2>&1 || true
  fi
done
log "Members added"

# ---------------------------------------------------------------------------
# 8. Create epic
# ---------------------------------------------------------------------------
log "Creating epic..."
EPIC=$(api_post POST "/projects/$PROJECT_ID/epics" '{
  "name": "MVP",
  "description": "Minimum viable product — project scaffolding, CI, and basic infrastructure"
}')
EPIC_ID=$(echo "$EPIC" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
log "Epic created: $EPIC_ID"

# ---------------------------------------------------------------------------
# 9. Create stories
# ---------------------------------------------------------------------------
log "Creating stories..."

api_post POST "/projects/$PROJECT_ID/stories" "{
  \"key\": \"S-01\",
  \"title\": \"Scaffold Go backend + Vue frontend\",
  \"objective\": \"Create the initial project structure with a Go chi backend and a Vue 3 + PrimeVue frontend. Both should compile and serve a hello world page.\",
  \"epic_id\": \"$EPIC_ID\",
  \"scope\": \"shared\",
  \"acceptance_criteria\": \"Go backend starts on :8080 with GET /health returning 200. Vue frontend builds and serves on :5173 with a hello world page. Docker Compose brings up both services.\"
}" > /dev/null

api_post POST "/projects/$PROJECT_ID/stories" "{
  \"key\": \"S-02\",
  \"title\": \"Setup CI pipeline with GitHub Actions\",
  \"objective\": \"Configure GitHub Actions workflows for linting, testing, and building both backend and frontend on every push and PR.\",
  \"epic_id\": \"$EPIC_ID\",
  \"scope\": \"shared\",
  \"depends_on\": [\"S-01\"],
  \"acceptance_criteria\": \"CI runs on push to main and on PRs. Backend: golangci-lint + go test. Frontend: eslint + vitest + build. All jobs pass on a clean repo.\"
}" > /dev/null

api_post POST "/projects/$PROJECT_ID/stories" "{
  \"key\": \"S-03\",
  \"title\": \"Configure linting and code formatting\",
  \"objective\": \"Setup golangci-lint for the backend and ESLint + Prettier for the frontend with shared configs. Add pre-commit hooks via lefthook.\",
  \"epic_id\": \"$EPIC_ID\",
  \"scope\": \"shared\",
  \"depends_on\": [\"S-01\"],
  \"acceptance_criteria\": \"golangci-lint runs clean on backend. ESLint + Prettier run clean on frontend. Lefthook pre-commit hook runs both.\"
}" > /dev/null

log "3 stories created"

# ---------------------------------------------------------------------------
# 10. Create agents
# ---------------------------------------------------------------------------
log "Creating agents..."

# Global agents (scope=global via project endpoint)
api_post POST "/projects/$PROJECT_ID/agents" '{
  "name": "Opus Dev Agent",
  "model": "claude-opus-4-6",
  "image": "ghcr.io/hopeitworks/agent:latest",
  "scope": "global",
  "template_content": "Implement story {{story_key}}: {{story_title}}\n\n## Objective\n{{story_objective}}\n\n## Acceptance Criteria\n{{acceptance_criteria}}\n\n## Branch\n{{branch_name}}"
}' > /dev/null

api_post POST "/projects/$PROJECT_ID/agents" '{
  "name": "Sonnet Review Agent",
  "model": "claude-sonnet-4-6",
  "image": "ghcr.io/hopeitworks/agent:latest",
  "scope": "global",
  "template_content": "Review changes for {{story_key}}: {{story_title}}\n\n## Acceptance Criteria\n{{acceptance_criteria}}\n\n## Changes\n{{diff_content}}\n\n## Checklist\n- All acceptance criteria met\n- Linters pass\n- Tests added\n- No hardcoded secrets"
}' > /dev/null

# Project agents
api_post POST "/projects/$PROJECT_ID/agents" '{
  "name": "Todo Dev Agent",
  "model": "claude-sonnet-4-6",
  "image": "ghcr.io/hopeitworks/agent:latest",
  "scope": "project",
  "template_content": "Implement story {{story_key}}: {{story_title}}\n\n## Project\nTodo App — Go backend (chi, pgx, sqlc) + Vue 3 frontend (PrimeVue, Tailwind).\n\n## Objective\n{{story_objective}}\n\n## Acceptance Criteria\n{{acceptance_criteria}}\n\n## Branch\n{{branch_name}}"
}' > /dev/null

api_post POST "/projects/$PROJECT_ID/agents" '{
  "name": "Todo Merge Agent",
  "model": "claude-opus-4-6",
  "image": "ghcr.io/hopeitworks/agent:latest",
  "scope": "project",
  "template_content": "Merge story {{story_key}} (branch {{branch_name}}) into main.\n\n1. Verify CI green\n2. Rebase on main\n3. Create PR with squash merge\n4. Confirm CI green post-merge"
}' > /dev/null

log "4 agents created (2 global + 2 project)"

# ---------------------------------------------------------------------------
# 11. Configure pipeline
# ---------------------------------------------------------------------------
log "Configuring pipeline..."

api_post PUT "/projects/$PROJECT_ID/pipeline" '{
  "groups": [
    {
      "id": "setup",
      "name": "Setup",
      "steps": [
        {
          "id": "10000000-0000-0000-0000-000000000001",
          "name": "Create Branch",
          "action_type": "git_branch",
          "auto_approve": true,
          "retry_policy": {"max_retries": 0, "retry_type": "none"},
          "config": {"base_branch": "main"}
        }
      ]
    },
    {
      "id": "development",
      "name": "Development",
      "steps": [
        {
          "id": "10000000-0000-0000-0000-000000000002",
          "name": "Implement Story",
          "action_type": "agent_run",
          "model": "claude-sonnet-4-6",
          "auto_approve": false,
          "retry_policy": {"max_retries": 2, "retry_type": "on-failure"},
          "config": {"role": "dev", "phase": "dev-story"}
        }
      ]
    },
    {
      "id": "review-merge",
      "name": "Review & Merge",
      "steps": [
        {
          "id": "10000000-0000-0000-0000-000000000003",
          "name": "Code Review",
          "action_type": "agent_run",
          "model": "claude-sonnet-4-6",
          "auto_approve": true,
          "retry_policy": {"max_retries": 1, "retry_type": "on-failure"},
          "config": {"role": "review", "phase": "code-review"}
        },
        {
          "id": "10000000-0000-0000-0000-000000000004",
          "name": "Approval Gate",
          "action_type": "human",
          "auto_approve": false,
          "retry_policy": {"max_retries": 0, "retry_type": "none"},
          "config": {"message": "Review the changes and approve to proceed with merge"}
        },
        {
          "id": "10000000-0000-0000-0000-000000000005",
          "name": "Create PR",
          "action_type": "git_pr",
          "auto_approve": true,
          "retry_policy": {"max_retries": 1, "retry_type": "on-failure"},
          "config": {"target_branch": "main", "strategy": "squash"}
        }
      ]
    },
    {
      "id": "delivery",
      "name": "Delivery",
      "steps": [
        {
          "id": "10000000-0000-0000-0000-000000000006",
          "name": "Wait for CI",
          "action_type": "ci_poll",
          "auto_approve": true,
          "retry_policy": {"max_retries": 0, "retry_type": "none"},
          "config": {"timeout_minutes": "30"}
        },
        {
          "id": "10000000-0000-0000-0000-000000000007",
          "name": "Notify Completion",
          "action_type": "notification",
          "auto_approve": true,
          "retry_policy": {"max_retries": 0, "retry_type": "none"},
          "config": {"channel": "default", "message": "Story {story_key} pipeline completed successfully"}
        }
      ]
    }
  ]
}' > /dev/null

log "Pipeline configured (4 groups, 7 steps)"

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
echo ""
log "Dev environment ready!"
echo "  Frontend: http://localhost:5174"
echo "  API:      http://localhost:8080"
echo "  Login:    admin@hopeitworks.dev / admin1234"
