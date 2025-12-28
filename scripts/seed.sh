#!/bin/bash

# Seed script for initializing tenant database

DB_PATH="${DB_PATH:-./tenants.db}"

echo "Seeding tenant database at $DB_PATH"

# Add sample tenants
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "tenant1.example.com", "tenant_id": "tenant-123"}' || true

curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "tenant2.example.com", "tenant_id": "tenant-456"}' || true

curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "*.saas.com", "tenant_id": "tenant-789"}' || true

echo "Seeding complete!"

