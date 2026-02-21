#!/usr/bin/env bash
# =============================================================================
# E2E tests for the Todo App API using curl
# =============================================================================
# Usage: bash __tests__/e2e.test.sh
# Requires: curl, a running todo-app instance on localhost:3000
# =============================================================================

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"
PASS=0
FAIL=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

assert_status() {
  local test_name="$1"
  local expected="$2"
  local actual="$3"

  if [ "$actual" = "$expected" ]; then
    echo -e "${GREEN}PASS${NC}: $test_name (status=$actual)"
    PASS=$((PASS + 1))
  else
    echo -e "${RED}FAIL${NC}: $test_name (expected=$expected, got=$actual)"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local test_name="$1"
  local expected="$2"
  local body="$3"

  if echo "$body" | grep -q "$expected"; then
    echo -e "${GREEN}PASS${NC}: $test_name (contains '$expected')"
    PASS=$((PASS + 1))
  else
    echo -e "${RED}FAIL${NC}: $test_name (missing '$expected' in response)"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Todo App E2E Tests ==="
echo "Base URL: $BASE_URL"
echo ""

# --- Health check ---
echo "--- Health Check ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
assert_status "GET /health returns 200" "200" "$STATUS"

BODY=$(curl -s "$BASE_URL/health")
assert_contains "Health response has status ok" '"ok"' "$BODY"

# --- CREATE ---
echo ""
echo "--- Create Todos ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/todos" \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy milk"}')
STATUS=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -1)
assert_status "POST /api/todos returns 201" "201" "$STATUS"
assert_contains "Created todo has title" '"Buy milk"' "$BODY"

TODO_ID=$(echo "$BODY" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/todos" \
  -H "Content-Type: application/json" \
  -d '{"title":"Walk the dog"}')
STATUS=$(echo "$RESPONSE" | tail -1)
assert_status "POST /api/todos second todo returns 201" "201" "$STATUS"

# --- CREATE with invalid input ---
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/todos" \
  -H "Content-Type: application/json" \
  -d '{"title":""}')
assert_status "POST /api/todos empty title returns 400" "400" "$STATUS"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/todos" \
  -H "Content-Type: application/json" \
  -d '{}')
assert_status "POST /api/todos missing title returns 400" "400" "$STATUS"

# --- READ ---
echo ""
echo "--- Read Todos ---"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/todos")
STATUS=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -1)
assert_status "GET /api/todos returns 200" "200" "$STATUS"
assert_contains "List contains Buy milk" '"Buy milk"' "$BODY"
assert_contains "List contains Walk the dog" '"Walk the dog"' "$BODY"

RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/todos/$TODO_ID")
STATUS=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -1)
assert_status "GET /api/todos/:id returns 200" "200" "$STATUS"
assert_contains "Single todo has correct title" '"Buy milk"' "$BODY"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/todos/99999")
assert_status "GET /api/todos/99999 returns 404" "404" "$STATUS"

# --- UPDATE ---
echo ""
echo "--- Update Todos ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/api/todos/$TODO_ID" \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy oat milk","completed":true}')
STATUS=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -1)
assert_status "PUT /api/todos/:id returns 200" "200" "$STATUS"
assert_contains "Updated title" '"Buy oat milk"' "$BODY"
assert_contains "Marked completed" '"completed":1' "$BODY"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/api/todos/99999" \
  -H "Content-Type: application/json" \
  -d '{"title":"Nope"}')
assert_status "PUT /api/todos/99999 returns 404" "404" "$STATUS"

# --- DELETE ---
echo ""
echo "--- Delete Todos ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/todos/$TODO_ID")
assert_status "DELETE /api/todos/:id returns 204" "204" "$STATUS"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/todos/$TODO_ID")
assert_status "GET deleted todo returns 404" "404" "$STATUS"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/todos/99999")
assert_status "DELETE /api/todos/99999 returns 404" "404" "$STATUS"

# --- Summary ---
echo ""
echo "=== Results ==="
echo -e "Passed: ${GREEN}${PASS}${NC}"
echo -e "Failed: ${RED}${FAIL}${NC}"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi

echo ""
echo "All E2E tests passed!"
