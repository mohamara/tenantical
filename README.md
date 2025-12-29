# Tenantical Router

یک Tenant Manager و Router مستقل برای Edge Server که درخواست‌ها را بر اساس دامنه به backend ارسال می‌کند و Tenant ID را در header اضافه می‌کند.

## ویژگی‌ها

- ✅ **Domain-based Routing**: تشخیص tenant بر اساس دامنه درخواست
- ✅ **Project Routing**: Forward کردن درخواست به project/service route مناسب
- ✅ **Custom Port Support**: پشتیبانی از پورت‌های مختلف برای هر tenant
- ✅ **SQLite Storage**: ذخیره‌سازی mapping دامنه به tenant ID + project route + project port
- ✅ **Header Injection**: افزودن خودکار `X-Tenant-ID` به درخواست‌های backend
- ✅ **Wildcard Support**: پشتیبانی از wildcard domains (مثل `*.example.com`)
- ✅ **In-memory Cache**: کش کردن نتایج برای performance بهتر
- ✅ **Admin API**: API برای مدیریت tenants
- ✅ **Zero Dependencies**: مستقل از backend و هیچ وابستگی خارجی ندارد
- ✅ **Production Ready**: Error handling، logging، graceful shutdown

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Chi Router
- **Database**: SQLite (embedded)
- **HTTP Client**: net/http (standard library)

## نصب و راه‌اندازی

### پیش‌نیازها

- Go 1.21 یا بالاتر
- SQLite (برای build با CGO)

### نصب

```bash
# Clone repository
git clone <repository-url>
cd tenantical

# Download dependencies
go mod download

# Build
make build

# یا
go build -o bin/tenant-router ./cmd/server
```

### اجرا

```bash
# با make
make run

# یا مستقیم
./bin/tenant-router

# با environment variables
BACKEND_URL=http://api.example.com DB_PATH=./tenants.db ./bin/tenant-router
```

## Configuration

تنظیمات از طریق Environment Variables انجام می‌شود:

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | آدرس host برای server |
| `PORT` | `8080` | پورت server |
| `DB_PATH` | `./tenants.db` | مسیر فایل SQLite database |
| `BACKEND_URL` | `http://localhost:3000` | URL backend API |
| `READ_TIMEOUT` | `10` | Timeout برای read (ثانیه) |
| `WRITE_TIMEOUT` | `10` | Timeout برای write (ثانیه) |
| `IDLE_TIMEOUT` | `120` | Timeout برای idle connections (ثانیه) |
| `PROXY_TIMEOUT` | `30` | Timeout برای proxy requests (ثانیه) |
| `PROXY_MAX_IDLE_CONNS` | `100` | حداکثر idle connections در pool |
| `PROXY_IDLE_CONN_TIMEOUT` | `90` | Timeout برای idle connections در pool (ثانیه) |

## استفاده

### 1. اضافه کردن Tenant

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "tenant1.example.com",
    "tenant_id": "tenant-123",
    "project_route": "/projects/backend"
  }'
```

**project_route** (اختیاری، پیش‌فرض: `/projects/backend`): مسیر پروژه در reverse proxy که درخواست به آن forward می‌شود.

**project_port** (اختیاری): پورت اختصاصی برای پروژه. اگر مشخص نشود، از پورت پیش‌فرض در `BACKEND_URL` استفاده می‌شود.

**مثال با پورت اختصاصی:**
```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "api.localhost",
    "tenant_id": "tenant-123",
    "project_route": "/projects/backend",
    "project_port": 85
  }'
```

### 2. اضافه کردن Wildcard Domain

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "*.saas.com",
    "tenant_id": "tenant-456"
  }'
```

### 3. لیست تمام Tenants

```bash
curl http://localhost:8080/admin/tenants
```

### 4. حذف Tenant

```bash
curl -X DELETE http://localhost:8080/admin/tenants/tenant1.example.com
```

### 5. ارسال درخواست از طریق Proxy

```bash
# درخواست به tenant1.example.com
curl -H "Host: tenant1.example.com" http://localhost:8080/api/users

# Router به backend ارسال می‌کند با header:
# X-Tenant-ID: tenant-123
```

## Architecture

```
┌─────────────┐
│   Client    │
│ tenant1.example.com/api/users
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│     Tenant Router (Edge)        │
│  ┌───────────────────────────┐  │
│  │  Domain Resolution        │  │
│  │  tenant1.example.com      │  │
│  └───────────────────────────┘  │
│  ┌───────────────────────────┐  │
│  │  Database Lookup          │  │
│  │  → tenant_id: tenant-123  │  │
│  │  → project_route:         │  │
│  │    /projects/backend      │  │
│  │  → project_port: 85       │  │
│  │    (optional)              │  │
│  └───────────────────────────┘  │
│  ┌───────────────────────────┐  │
│  │  Header Injection         │  │
│  │  X-Tenant-ID: tenant-123  │  │
│  └───────────────────────────┘  │
│  ┌───────────────────────────┐  │
│  │  Route Construction       │  │
│  │  /projects/backend/       │  │
│  │    + /api/users           │  │
│  │  URL: http://localhost:85 │  │
│  │    (if project_port set)  │  │
│  └───────────────────────────┘  │
└──────────────┬──────────────────┘
               │
               │ Forwarded Request
               │ URL: http://localhost:85/projects/backend/api/users
               │      (or {BACKEND_URL} if no project_port)
               │ Header: X-Tenant-ID: tenant-123
               ▼
┌─────────────────────────────────┐
│    Reverse Proxy (nginx/traefik)│
│  Routes:                        │
│    /projects/backend   → backend│
│    /projects/frontend  → frontend│
│    /projects/admin     → admin  │
└──────────────┬──────────────────┘
               │
               ▼
┌─────────────────────────────────┐
│      Project Services           │
│  (backend/frontend/admin/...)   │
└─────────────────────────────────┘
```

## Docker

### Build

```bash
docker build -t tenant-router .
```

### Run

```bash
docker run -p 8080:8080 \
  -e BACKEND_URL=http://api.example.com \
  -e DB_PATH=/data/tenants.db \
  -v $(pwd)/data:/data \
  tenant-router
```

### Docker Compose

```bash
docker-compose up -d
```

## API Endpoints

### Health Check

```http
GET /health
```

Response:
```json
{
  "status": "ok",
  "service": "tenant-router"
}
```

### Admin API

#### Add Tenant
```http
POST /admin/tenants
Content-Type: application/json

{
  "domain": "tenant1.example.com",
  "tenant_id": "tenant-123",
  "project_route": "/projects/backend",
  "project_port": 85
}
```

**Fields:**
- `domain` (required): Domain یا subdomain tenant
- `tenant_id` (required): شناسه یکتا tenant
- `project_route` (optional): مسیر پروژه در reverse proxy (default: `/projects/backend`)
- `project_port` (optional): پورت اختصاصی برای پروژه (default: استفاده از پورت در `BACKEND_URL`)

**Examples:**

**با project_route:**
- `/projects/backend` → برای backend API
- `/projects/frontend` → برای frontend application
- `/projects/admin` → برای admin panel
- `/projects/api` → برای API service

**با project_port (برای پروژه‌های روی پورت‌های مختلف):**
```json
{
  "domain": "api.localhost",
  "tenant_id": "tenant-123",
  "project_route": "/projects/backend",
  "project_port": 85
}
```
درخواست به `http://localhost:85/projects/backend/...` forward می‌شود.

#### List Tenants
```http
GET /admin/tenants
```

#### Delete Tenant
```http
DELETE /admin/tenants/{domain}
```

### Proxy (Catch-all)

```http
ANY /*
```

تمام درخواست‌های دیگر به backend forward می‌شوند با header `X-Tenant-ID`.

## Wildcard Domain Matching

پشتیبانی از wildcard domains:

- `*.example.com` → matches `subdomain.example.com`, `app.example.com`, etc.
- `tenant.*` → matches `tenant.com`, `tenant.org`, etc.

## Performance

- **Latency**: < 1ms برای domain resolution (با cache)
- **Throughput**: > 10k requests/sec (بستگی به hardware دارد)
- **Memory**: ~20-50MB در حالت idle
- **CPU**: کم (I/O bound)

## Security Considerations

1. **Admin API**: در production، admin endpoints را با authentication محافظت کنید
2. **Rate Limiting**: برای جلوگیری از abuse، rate limiting اضافه کنید
3. **HTTPS**: همیشه از HTTPS استفاده کنید (SSL در reverse proxy)
4. **Input Validation**: domain validation در admin API
5. **SSL Certificates**: از certbot برای دریافت و تمدید certificates استفاده کنید

## SSL/HTTPS Setup

برای setup SSL certificates:

1. **استفاده از Reverse Proxy** (nginx/traefik) برای SSL termination
2. **Certbot** برای دریافت و تمدید خودکار certificates
3. راهنمای کامل: [docs/SSL_SETUP_CERTBOT.md](docs/SSL_SETUP_CERTBOT.md)

Quick setup:
```bash
# با script
sudo ./scripts/setup-ssl.sh example.com admin@example.com

# یا دستی
sudo certbot --nginx -d example.com -d *.example.com
```

**نکته مهم:** SSL certificates در **reverse proxy (nginx)** ست می‌شوند، نه در Tenant Router.

## Development

```bash
# Run tests
make test

# Run with hot reload (با air یا similar)
air

# Check code
go vet ./...
golangci-lint run
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
