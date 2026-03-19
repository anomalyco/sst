#!/usr/bin/env bash
set -euo pipefail

# Test the shared ALB example endpoints after deployment.
#
# Usage:
#   ./test.sh [alb-url]
#
# If no URL is provided, attempts to read it from sst output.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Get ALB URL from argument or sst output
if [ -n "${1:-}" ]; then
  ALB_URL="$1"
else
  echo "Fetching ALB URL from sst output..."
  ALB_URL=$(npx sst output url 2>/dev/null || true)
  if [ -z "$ALB_URL" ]; then
    echo "ERROR: Could not determine ALB URL."
    echo "Usage: ./test.sh <alb-url>"
    echo "  e.g. ./test.sh http://SharedAlb-123456.us-east-1.elb.amazonaws.com"
    exit 1
  fi
fi

# Strip trailing slash
ALB_URL="${ALB_URL%/}"

echo "=== Testing Shared ALB Example ==="
echo "ALB URL: $ALB_URL"
echo ""

PASS=0
FAIL=0

run_test() {
  local name="$1"
  local url="$2"
  local expected="$3"

  echo -n "TEST: $name ... "
  HTTP_CODE=$(curl -s -o /tmp/alb_test_response -w "%{http_code}" --max-time 10 "$url" 2>/dev/null || echo "000")
  BODY=$(cat /tmp/alb_test_response 2>/dev/null || echo "")

  if [ "$HTTP_CODE" = "000" ]; then
    echo "FAIL (connection error)"
    FAIL=$((FAIL + 1))
    return
  fi

  if echo "$BODY" | grep -q "$expected"; then
    echo "OK (HTTP $HTTP_CODE) → $BODY"
    PASS=$((PASS + 1))
  else
    echo "FAIL (HTTP $HTTP_CODE, expected '$expected')"
    echo "  Got: $BODY"
    FAIL=$((FAIL + 1))
  fi
}

# Note: ALB path pattern "/api/*" matches "/api/<anything>" but NOT "/api" itself.
# Tests use paths with a trailing segment to match the wildcard.

# Test 1: API service — /api/health returns { "status": "ok" }
run_test "API health (/api/health)" "$ALB_URL/api/health" '"status":"ok"'

# Test 2: API service — /api/users returns { "service": "api", "path": "/api/users" }
run_test "API routing (/api/users)" "$ALB_URL/api/users" '"service":"api"'

# Test 3: API service — /api/greeting returns path info
run_test "API greeting (/api/greeting)" "$ALB_URL/api/greeting" "/api/greeting"

# Test 4: Web service — /app/health returns { "status": "ok" }
run_test "Web health (/app/health)" "$ALB_URL/app/health" '"status":"ok"'

# Test 5: Web service — /app/dashboard returns HTML with path
run_test "Web routing (/app/dashboard)" "$ALB_URL/app/dashboard" "/app/dashboard"

# Test 6: Web service — /app/greeting returns path info
run_test "Web greeting (/app/greeting)" "$ALB_URL/app/greeting" "/app/greeting"

# Test 7: Default action — / should return 404 (ALB default)
echo -n "TEST: Default action (/) ... "
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$ALB_URL/" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "403" ]; then
  echo "PASS (HTTP 403 as expected)"
  PASS=$((PASS + 1))
else
  echo "FAIL (expected 403, got HTTP $HTTP_CODE)"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
