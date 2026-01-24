#!/bin/bash

# Quick test script for carry-over functionality
# Make sure server is running on localhost:8070

echo "=== Testing Carry-Over Functionality ==="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="http://localhost:8070"
ADMIN_USERNAME="admin"
ADMIN_PASSWORD="password123"

echo -e "${BLUE}Step 1: Login as Admin${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/admin/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"$ADMIN_USERNAME\", \"password\": \"$ADMIN_PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ Login failed!"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ Login successful${NC}"
echo "Token: ${TOKEN:0:20}..."
echo ""

echo -e "${BLUE}Step 2: Get Leave Types${NC}"
LEAVE_TYPES=$(curl -s -X GET "$BASE_URL/api/leave-types" \
  -H "Authorization: Bearer $TOKEN")

echo "$LEAVE_TYPES" | jq '.'
ANNUAL_LEAVE_ID=$(echo "$LEAVE_TYPES" | jq -r '.[] | select(.name == "Annual") | .id')
echo -e "${GREEN}✓ Annual Leave ID: $ANNUAL_LEAVE_ID${NC}"
echo ""

echo -e "${BLUE}Step 3: Get Employee ID${NC}"
EMPLOYEES=$(curl -s -X GET "$BASE_URL/api/employees" \
  -H "Authorization: Bearer $TOKEN")

echo "$EMPLOYEES" | jq '.'
EMPLOYEE_ID=$(echo "$EMPLOYEES" | jq -r '.[0].id')
echo -e "${GREEN}✓ Employee ID: $EMPLOYEE_ID${NC}"
echo ""

echo -e "${BLUE}Step 4: Get Current Annual Leave Balance${NC}"
BALANCE=$(curl -s -X GET "$BASE_URL/api/hr/employees/$EMPLOYEE_ID/annual-leave-balance" \
  -H "Authorization: Bearer $TOKEN")

echo "$BALANCE" | jq '.'
echo ""

echo -e "${BLUE}Step 5: Process Year-End Carry-Over for 2024${NC}"
CARRYOVER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/hr/leaves/process-carryover" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"leave_type_id\": $ANNUAL_LEAVE_ID, \"from_year\": 2024}")

echo "$CARRYOVER_RESPONSE" | jq '.'
echo ""

echo -e "${BLUE}Step 6: Get Carry-Over History${NC}"
HISTORY=$(curl -s -X GET "$BASE_URL/api/hr/employees/$EMPLOYEE_ID/carryover-history?leave_type_id=$ANNUAL_LEAVE_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$HISTORY" | jq '.'
echo ""

echo -e "${BLUE}Step 7: Get Carry-Over Balance${NC}"
CARRYOVER_BALANCE=$(curl -s -X GET "$BASE_URL/api/hr/employees/$EMPLOYEE_ID/carryover-balance?leave_type_id=$ANNUAL_LEAVE_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$CARRYOVER_BALANCE" | jq '.'
echo ""

echo -e "${GREEN}=== Test Complete ===${NC}"
