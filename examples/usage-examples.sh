#!/bin/bash

# Usage Examples for Tenant Router with Project Routing

BASE_URL="http://localhost:8080"

echo "🚀 Tenant Router - Project Routing Examples"
echo "==========================================="
echo ""

# Example 1: Add tenant with backend project route
echo "📝 Example 1: Add tenant for backend service"
curl -X POST "$BASE_URL/admin/tenants" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "api1.example.com",
    "tenant_id": "tenant-001",
    "project_route": "/projects/backend"
  }'
echo -e "\n"

# Example 2: Add tenant with frontend project route
echo "📝 Example 2: Add tenant for frontend service"
curl -X POST "$BASE_URL/admin/tenants" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "app1.example.com",
    "tenant_id": "tenant-002",
    "project_route": "/projects/frontend"
  }'
echo -e "\n"

# Example 3: Add tenant with admin project route
echo "📝 Example 3: Add tenant for admin service"
curl -X POST "$BASE_URL/admin/tenants" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "admin1.example.com",
    "tenant_id": "tenant-003",
    "project_route": "/projects/admin"
  }'
echo -e "\n"

# Example 4: Add tenant without project_route (uses default)
echo "📝 Example 4: Add tenant with default project route"
curl -X POST "$BASE_URL/admin/tenants" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "default.example.com",
    "tenant_id": "tenant-004"
  }'
echo -e "\n"

# Example 5: List all tenants
echo "📋 Example 5: List all tenants"
curl "$BASE_URL/admin/tenants" | jq '.'
echo -e "\n"

# Example 6: Test proxy routing
echo "🔀 Example 6: Test proxy routing"
echo "Request: GET api1.example.com/api/users"
echo "Will be forwarded to: {BACKEND_URL}/projects/backend/api/users"
echo "With header: X-Tenant-ID: tenant-001"
echo ""
curl -v -H "Host: api1.example.com" "$BASE_URL/api/users" 2>&1 | grep -E "(Host|X-Tenant-ID|HTTP|Location)"
echo -e "\n"

echo "✅ Examples completed!"

