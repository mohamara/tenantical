# معماری Tenant Router

## معماری پیشنهادی بر اساس نیاز شما

```
┌─────────────────────────────────────────────────────────┐
│                    Client Request                        │
│         tenant1.example.com/api/users                   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Tenant Router (Edge Layer)                  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ 1. Domain Resolution                              │  │
│  │    tenant1.example.com                            │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ 2. Database Lookup                               │  │
│  │    Domain → Tenant ID + Project Route            │  │
│  │    tenant-123 → /projects/backend                │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ 3. Header Injection                              │  │
│  │    X-Tenant-ID: tenant-123                       │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ 4. Forward to Project Route                      │  │
│  │    /projects/backend/api/users                   │  │
│  └───────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            Reverse Proxy (nginx/traefik)                │
│  Routes:                                                │
│    /projects/backend   → backend service               │
│    /projects/frontend  → frontend service              │
│    /projects/admin     → admin service                 │
│    /projects/api       → api service                   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Project Services                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Backend    │  │   Frontend   │  │    Admin     │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Schema Database

```sql
CREATE TABLE tenants (
    domain TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_route TEXT NOT NULL,  -- مثال: /projects/backend
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Flow

1. **Client** درخواست می‌دهد به `tenant1.example.com/api/users`
2. **Tenant Router**:
   - Domain را می‌خواند: `tenant1.example.com`
   - از دیتابیس می‌خواند: `tenant_id` و `project_route`
   - Header `X-Tenant-ID` را اضافه می‌کند
   - درخواست را به `/projects/{project_route}/api/users` forward می‌کند
3. **Reverse Proxy**:
   - Route `/projects/backend` را می‌بیند
   - درخواست را به backend service forward می‌کند
4. **Backend Service**:
   - Header `X-Tenant-ID` را می‌خواند
   - Request را با tenant context پردازش می‌کند

