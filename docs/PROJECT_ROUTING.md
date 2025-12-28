# Project Routing Guide

## Overview

Tenant Router علاوه بر تشخیص Tenant ID از دامنه، می‌تواند درخواست‌ها را به project/service route مناسب در reverse proxy forward کند.

## معماری

```
Client Request: tenant1.example.com/api/users
    ↓
Tenant Router:
    ├─ Domain Lookup: tenant1.example.com
    ├─ Database Query:
    │   ├─ tenant_id: "tenant-123"
    │   └─ project_route: "/projects/backend"
    ├─ Header Injection: X-Tenant-ID: tenant-123
    └─ URL Construction: {BACKEND_URL}/projects/backend/api/users
    ↓
Reverse Proxy (nginx/traefik):
    ├─ Route: /projects/backend → backend service
    ├─ Route: /projects/frontend → frontend service
    └─ Route: /projects/admin → admin service
    ↓
Project Services
```

## Database Schema

```sql
CREATE TABLE tenants (
    domain TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_route TEXT NOT NULL DEFAULT '/projects/backend',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Project Route Examples

### Backend Service
```json
{
  "domain": "api.example.com",
  "tenant_id": "tenant-123",
  "project_route": "/projects/backend"
}
```
**Result:** Request به `{BACKEND_URL}/projects/backend/{path}` forward می‌شود

### Frontend Service
```json
{
  "domain": "app.example.com",
  "tenant_id": "tenant-123",
  "project_route": "/projects/frontend"
}
```
**Result:** Request به `{BACKEND_URL}/projects/frontend/{path}` forward می‌شود

### Admin Service
```json
{
  "domain": "admin.example.com",
  "tenant_id": "tenant-123",
  "project_route": "/projects/admin"
}
```
**Result:** Request به `{BACKEND_URL}/projects/admin/{path}` forward می‌شود

### Custom Route
```json
{
  "domain": "custom.example.com",
  "tenant_id": "tenant-123",
  "project_route": "/custom/path"
}
```
**Result:** Request به `{BACKEND_URL}/custom/path/{path}` forward می‌شود

## API Usage

### Add Tenant with Project Route

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "api.example.com",
    "tenant_id": "tenant-123",
    "project_route": "/projects/backend"
  }'
```

### Add Tenant without Project Route (uses default)

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "default.example.com",
    "tenant_id": "tenant-456"
  }'
```
**Note:** اگر `project_route` مشخص نشود، پیش‌فرض `/projects/backend` استفاده می‌شود.

## Reverse Proxy Configuration

برای اینکه reverse proxy بتواند project routes را handle کند:

### nginx Example

```nginx
# Backend service
location /projects/backend {
    proxy_pass http://backend-service:3000;
    proxy_set_header Host $host;
    proxy_set_header X-Tenant-ID $http_x_tenant_id;
}

# Frontend service
location /projects/frontend {
    proxy_pass http://frontend-service:3001;
    proxy_set_header Host $host;
    proxy_set_header X-Tenant-ID $http_x_tenant_id;
}

# Admin service
location /projects/admin {
    proxy_pass http://admin-service:3002;
    proxy_set_header Host $host;
    proxy_set_header X-Tenant-ID $http_x_tenant_id;
}
```

### Traefik Example

```yaml
services:
  tenant-router:
    labels:
      - "traefik.http.routers.tenant-router.rule=HostRegexp(`{subdomain:.+}.example.com`)"
      
  backend:
    labels:
      - "traefik.http.routers.backend.rule=PathPrefix(`/projects/backend`)"
      - "traefik.http.services.backend.loadbalancer.server.port=3000"
      
  frontend:
    labels:
      - "traefik.http.routers.frontend.rule=PathPrefix(`/projects/frontend`)"
      - "traefik.http.services.frontend.loadbalancer.server.port=3001"
```

## Use Cases

### 1. Multi-Service Architecture

اگر هر tenant می‌تواند به services مختلف دسترسی داشته باشد:

```json
// Tenant A - فقط backend
{
  "domain": "tenant-a.example.com",
  "tenant_id": "tenant-a",
  "project_route": "/projects/backend"
}

// Tenant B - backend + frontend
{
  "domain": "tenant-b-frontend.example.com",
  "tenant_id": "tenant-b",
  "project_route": "/projects/frontend"
}

{
  "domain": "tenant-b-api.example.com",
  "tenant_id": "tenant-b",
  "project_route": "/projects/backend"
}
```

### 2. Environment-based Routing

```json
// Production
{
  "domain": "api.prod.example.com",
  "tenant_id": "tenant-prod",
  "project_route": "/projects/backend-prod"
}

// Staging
{
  "domain": "api.staging.example.com",
  "tenant_id": "tenant-staging",
  "project_route": "/projects/backend-staging"
}
```

### 3. Feature-based Routing

```json
// Standard plan
{
  "domain": "tenant1.example.com",
  "tenant_id": "tenant-1",
  "project_route": "/projects/backend-standard"
}

// Premium plan
{
  "domain": "tenant2.example.com",
  "tenant_id": "tenant-2",
  "project_route": "/projects/backend-premium"
}
```

## Best Practices

1. **Consistent Naming**: از یک naming convention برای project routes استفاده کنید
   - `/projects/{service-name}`
   - `/services/{service-name}`
   - `/apps/{app-name}`

2. **Default Route**: همیشه یک default route تعریف کنید (`/projects/backend`)

3. **Route Validation**: در reverse proxy، route validation کنید تا route های نامعتبر reject شوند

4. **Monitoring**: log کنید که هر tenant به کدام project route forward می‌شود

5. **Caching**: project routes تغییر نمی‌کنند، بنابراین cache کردن نتیجه مفید است (فعلاً implemented است)

## Migration

اگر از نسخه قدیمی استفاده می‌کنید که project_route نداشت:

1. Database migration به صورت خودکار انجام می‌شود (default: `/projects/backend`)
2. می‌توانید بعداً project_route را برای هر tenant به‌روزرسانی کنید:

```sql
UPDATE tenants SET project_route = '/projects/frontend' WHERE domain = 'frontend.example.com';
```

