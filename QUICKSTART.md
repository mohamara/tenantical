# Quick Start Guide

راهنمای سریع برای شروع کار با Tenant Router.

## نصب سریع

```bash
# 1. Build
make build

# 2. Run
./bin/tenant-router
```

## راه‌اندازی اولیه

### 1. شروع Server

```bash
# با default settings
./bin/tenant-router

# یا با environment variables
BACKEND_URL=http://api.example.com PORT=8080 ./bin/tenant-router
```

### 2. اضافه کردن Tenant ها

```bash
# Tenant 1
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "tenant1.example.com", "tenant_id": "tenant-123"}'

# Tenant 2
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "tenant2.example.com", "tenant_id": "tenant-456"}'

# Wildcard domain
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "*.saas.com", "tenant_id": "tenant-789"}'
```

### 3. تست Proxy

```bash
# درخواست به tenant1
curl -H "Host: tenant1.example.com" http://localhost:8080/api/users

# این درخواست به backend ارسال میشه با header:
# X-Tenant-ID: tenant-123
```

### 4. مشاهده لیست Tenants

```bash
curl http://localhost:8080/admin/tenants
```

## Docker Quick Start

```bash
# Build و Run با docker-compose
docker-compose up -d

# یا مستقیم
docker build -t tenant-router .
docker run -p 8080:8080 \
  -e BACKEND_URL=http://api.example.com \
  tenant-router
```

## مثال کامل

```bash
# 1. Start server
./bin/tenant-router &

# 2. Wait for server to start
sleep 2

# 3. Add tenants
curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "app1.example.com", "tenant_id": "t1"}'

curl -X POST http://localhost:8080/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{"domain": "app2.example.com", "tenant_id": "t2"}'

# 4. Test routing
curl -v -H "Host: app1.example.com" http://localhost:8080/health
# باید header X-Tenant-ID: t1 را در request به backend ببیند

curl -v -H "Host: app2.example.com" http://localhost:8080/health
# باید header X-Tenant-ID: t2 را در request به backend ببیند
```

## تنظیمات مهم

### Environment Variables

```bash
# Backend URL (الزامی)
export BACKEND_URL=http://your-backend-api.com

# Database path
export DB_PATH=./tenants.db

# Server port
export PORT=8080
```

### DNS/Proxy Configuration

برای استفاده در production، باید:

1. **DNS Configuration**: تمام subdomain ها را به IP edge server اشاره دهید
2. **Reverse Proxy**: از nginx یا cloudflare استفاده کنید که Host header را حفظ کند
3. **HTTPS**: حتماً از HTTPS استفاده کنید

مثال nginx configuration:

```nginx
server {
    listen 80;
    server_name *.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Troubleshooting

### مشکل: "Invalid tenant domain"

- مطمئن شوید tenant را اضافه کرده‌اید
- بررسی کنید Host header صحیح است
- از wildcard matching استفاده کنید اگر نیاز دارید

### مشکل: Connection refused

- بررسی کنید backend URL صحیح است
- مطمئن شوید backend server در دسترس است
- بررسی firewall rules

### مشکل: Database locked

- فقط یک instance از برنامه را اجرا کنید
- یا از SQLite WAL mode استفاده کنید (پیش‌فرض فعال است)

## Next Steps

- مطالعه [README.md](README.md) برای جزئیات بیشتر
- بررسی API endpoints
- تنظیم authentication برای admin API
- اضافه کردن monitoring و logging

