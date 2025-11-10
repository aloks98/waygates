#!/bin/bash

# Proxy API Test Script
# This script tests the complete workflow of proxy management

BASE_URL="http://localhost:8080/api"
PROXY_ID=""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Proxy Management API - Test Suite${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Function to print test name
print_test() {
    echo -e "\n${YELLOW}Test: $1${NC}"
    echo "-----------------------------------"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Wait for user
wait_for_user() {
    echo -e "\n${BLUE}Press Enter to continue...${NC}"
    read
}

# Test 1: Health Check
print_test "1. Health Check"
curl -s -X GET "$BASE_URL/health" | jq
print_success "Health endpoint is working"
wait_for_user

# Test 2: List Proxies (should be empty initially)
print_test "2. List Proxies (Initial)"
curl -s -X GET "$BASE_URL/proxies" | jq
print_success "List endpoint is working"
wait_for_user

# Test 3: Create a Reverse Proxy
print_test "3. Create Reverse Proxy"
RESPONSE=$(curl -s -X POST "$BASE_URL/proxies" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "reverse_proxy",
    "name": "Test Backend API",
    "hostname": "api-test.example.com",
    "upstreams": [
      {
        "host": "localhost",
        "port": 9999,
        "scheme": "http"
      }
    ],
    "block_exploits": true,
    "custom_headers": {
      "X-Test-Header": "test-value"
    }
  }')

echo "$RESPONSE" | jq

# Extract proxy ID
PROXY_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty')

if [ -n "$PROXY_ID" ]; then
    print_success "Proxy created with ID: $PROXY_ID"
else
    print_error "Failed to create proxy"
    exit 1
fi
wait_for_user

# Test 4: Get Proxy by ID
print_test "4. Get Proxy by ID ($PROXY_ID)"
curl -s -X GET "$BASE_URL/proxies/$PROXY_ID" | jq
print_success "Get proxy endpoint is working"
wait_for_user

# Test 5: List Proxies (should show our proxy)
print_test "5. List Proxies (Should show 1 proxy)"
curl -s -X GET "$BASE_URL/proxies" | jq
print_success "Proxy appears in list"
wait_for_user

# Test 6: Update Proxy
print_test "6. Update Proxy"
curl -s -X PUT "$BASE_URL/proxies/$PROXY_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "reverse_proxy",
    "name": "Test Backend API (Updated)",
    "hostname": "api-test-v2.example.com",
    "upstreams": [
      {
        "host": "localhost",
        "port": 8888,
        "scheme": "http"
      }
    ],
    "block_exploits": true,
    "custom_headers": {
      "X-Test-Header": "updated-value",
      "X-Version": "2.0"
    }
  }' | jq
print_success "Proxy updated successfully"
wait_for_user

# Test 7: Disable Proxy
print_test "7. Disable Proxy"
curl -s -X POST "$BASE_URL/proxies/$PROXY_ID/disable" | jq
print_success "Proxy disabled"
wait_for_user

# Test 8: Get Proxy (should show is_active: false)
print_test "8. Verify Proxy is Disabled"
curl -s -X GET "$BASE_URL/proxies/$PROXY_ID" | jq '.data.is_active'
wait_for_user

# Test 9: Enable Proxy
print_test "9. Enable Proxy"
curl -s -X POST "$BASE_URL/proxies/$PROXY_ID/enable" | jq
print_success "Proxy enabled"
wait_for_user

# Test 10: Get Proxy (should show is_active: true)
print_test "10. Verify Proxy is Enabled"
curl -s -X GET "$BASE_URL/proxies/$PROXY_ID" | jq '.data.is_active'
wait_for_user

# Test 11: Create a Redirect
print_test "11. Create Redirect Proxy"
REDIRECT_RESPONSE=$(curl -s -X POST "$BASE_URL/proxies" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "redirect",
    "name": "Test Redirect",
    "hostname": "old.example.com",
    "redirect_config": {
      "target": "https://new.example.com",
      "status_code": 301,
      "preserve_path": true,
      "preserve_query": true
    }
  }')

echo "$REDIRECT_RESPONSE" | jq

REDIRECT_ID=$(echo "$REDIRECT_RESPONSE" | jq -r '.data.id // empty')
if [ -n "$REDIRECT_ID" ]; then
    print_success "Redirect created with ID: $REDIRECT_ID"
else
    print_error "Failed to create redirect"
fi
wait_for_user

# Test 12: Create a Static Site
print_test "12. Create Static Site Proxy"
STATIC_RESPONSE=$(curl -s -X POST "$BASE_URL/proxies" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "static",
    "name": "Test Static Site",
    "hostname": "static.example.com",
    "static_config": {
      "root_path": "/var/www/html",
      "index_file": "index.html",
      "browse": false,
      "template_rendering": false,
      "try_files": ["index.html"]
    }
  }')

echo "$STATIC_RESPONSE" | jq

STATIC_ID=$(echo "$STATIC_RESPONSE" | jq -r '.data.id // empty')
if [ -n "$STATIC_ID" ]; then
    print_success "Static site created with ID: $STATIC_ID"
else
    print_error "Failed to create static site"
fi
wait_for_user

# Test 13: List All Proxies (should show 3)
print_test "13. List All Proxies (Should show 3)"
curl -s -X GET "$BASE_URL/proxies" | jq
wait_for_user

# Test 14: Filter by Type
print_test "14. Filter Proxies by Type (reverse_proxy)"
curl -s -X GET "$BASE_URL/proxies?type=reverse_proxy" | jq
wait_for_user

# Test 15: Search Proxies
print_test "15. Search Proxies (search=test)"
curl -s -X GET "$BASE_URL/proxies?search=test" | jq
wait_for_user

# Test 16: Delete Redirect Proxy
if [ -n "$REDIRECT_ID" ]; then
    print_test "16. Delete Redirect Proxy"
    curl -s -X DELETE "$BASE_URL/proxies/$REDIRECT_ID" | jq
    print_success "Redirect deleted"
    wait_for_user
fi

# Test 17: Delete Static Proxy
if [ -n "$STATIC_ID" ]; then
    print_test "17. Delete Static Proxy"
    curl -s -X DELETE "$BASE_URL/proxies/$STATIC_ID" | jq
    print_success "Static site deleted"
    wait_for_user
fi

# Test 18: Delete Original Proxy
print_test "18. Delete Original Reverse Proxy"
curl -s -X DELETE "$BASE_URL/proxies/$PROXY_ID" | jq
print_success "Reverse proxy deleted"
wait_for_user

# Test 19: Final List (should be empty)
print_test "19. Final List (Should be empty)"
curl -s -X GET "$BASE_URL/proxies" | jq
wait_for_user

# Test 20: Try to get deleted proxy (should fail)
print_test "20. Try to Get Deleted Proxy (Should fail with 404)"
curl -s -X GET "$BASE_URL/proxies/$PROXY_ID" | jq
wait_for_user

echo -e "\n${BLUE}========================================${NC}"
echo -e "${GREEN}All Tests Completed!${NC}"
echo -e "${BLUE}========================================${NC}\n"
