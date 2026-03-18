#!/usr/bin/env bash
# Test process-accruals manually via API (for automation scripts).
# Usage:
#   ./scripts/test-process-accruals.sh [MONTH]
#   MONTH in YYYY-MM (default: previous month). Example: 2025-02

set -e
BASE="${BASE_URL:-http://localhost:8070}"
MONTH="${1:-}"
USERNAME="${ADMIN_USER:-admin}"
PASSWORD="${ADMIN_PASSWORD:-password123}"

if [ -z "$MONTH" ]; then
  # Previous month in YYYY-MM
  if date -v-1m &>/dev/null; then
    MONTH=$(date -v-1m +%Y-%m)
  else
    MONTH=$(date -d "last month" +%Y-%m)
  fi
fi

echo "Base URL: $BASE"
echo "Month:    $MONTH"
echo "Login:    $USERNAME"
echo ""

# 1) Get JWT
echo "1. Logging in..."
RESP=$(curl -s -X POST "$BASE/auth/admin/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

if echo "$RESP" | grep -q '"token"'; then
  if command -v jq &>/dev/null; then
    TOKEN=$(echo "$RESP" | jq -r '.token')
  else
    TOKEN=$(echo "$RESP" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
  fi
  echo "   Token obtained."
else
  echo "   Login failed. Response: $RESP"
  exit 1
fi

# 2) Process accruals
echo "2. Processing accruals for $MONTH..."
RESULT=$(curl -s -X POST "$BASE/api/hr/leaves/process-accruals" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"month\":\"$MONTH\"}")

echo "$RESULT" | head -c 500
echo ""
if echo "$RESULT" | grep -q '"processed"'; then
  echo ""
  echo "Done. Check 'processed' and 'errors' in the response above."
else
  echo "Request may have failed. Full response above."
  exit 1
fi
