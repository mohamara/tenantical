#!/bin/bash

# Test script for Tenant Router
# این script را برای تست سریع استفاده کنید

BASE_URL="http://localhost:8080"
BACKEND_URL="${BACKEND_URL:-http://localhost:3000}"

echo "🧪 Testing Tenant Router"
echo "========================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Health Check
echo -e "${YELLOW}Test 1: Health Check${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed (HTTP $response)${NC}"
fi
echo ""

# Test 2: Add Tenant
echo -e "${YELLOW}Test 2: Add Tenant${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/admin/tenants" \
  -H "Content-Type: application/json" \
  -d '{"domain": "test.example.com", "tenant_id": "test-tenant-123"}')

if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Tenant added successfully${NC}"
else
    echo -e "${RED}✗ Failed to add tenant (HTTP $response)${NC}"
fi
echo ""

# Test 3: List Tenants
echo -e "${YELLOW}Test 3: List Tenants${NC}"
tenants=$(curl -s "$BASE_URL/admin/tenants")
echo "$tenants" | grep -q "test-tenant-123"
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Tenant listed successfully${NC}"
    echo "$tenants" | jq '.' 2>/dev/null || echo "$tenants"
else
    echo -e "${RED}✗ Tenant not found in list${NC}"
fi
echo ""

# Test 4: Proxy Request (with mock backend)
echo -e "${YELLOW}Test 4: Proxy Request${NC}"
echo "Note: This requires a backend server at $BACKEND_URL"
echo "Testing proxy forwarding..."
response=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: test.example.com" "$BASE_URL/test")
echo "Response code: $response"
echo ""

# Test 5: Wildcard Domain
echo -e "${YELLOW}Test 5: Add Wildcard Domain${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/admin/tenants" \
  -H "Content-Type: application/json" \
  -d '{"domain": "*.test.com", "tenant_id": "wildcard-tenant"}')

if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Wildcard domain added successfully${NC}"
else
    echo -e "${RED}✗ Failed to add wildcard domain (HTTP $response)${NC}"
fi
echo ""

# Test 6: Delete Tenant
echo -e "${YELLOW}Test 6: Delete Tenant${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/admin/tenants/test.example.com")
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓ Tenant deleted successfully${NC}"
else
    echo -e "${RED}✗ Failed to delete tenant (HTTP $response)${NC}"
fi
echo ""

echo "========================="
echo -e "${GREEN}Tests completed!${NC}"

