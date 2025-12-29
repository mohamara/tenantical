# راهنمای نصب خودکار Tenantical Router

این اسکریپت تمام مراحل نصب و راه‌اندازی Tenantical Router را به صورت خودکار انجام می‌دهد.

## پیش‌نیازها

- سرور خام (Ubuntu/Debian یا CentOS/RHEL)
- دسترسی root یا sudo
- اتصال به اینترنت
- DNS records باید از قبل تنظیم شوند (یا بعد از نصب)

## استفاده

```bash
# Clone یا دانلود پروژه
git clone <repository-url>
cd tenantical

# اجرای اسکریپت نصب
sudo bash scripts/install.sh
```

## مراحل نصب

اسکریپت به صورت تعاملی از شما سوال می‌پرسد:

1. **انتخاب Reverse Proxy**:
   - `1` برای Nginx (با Certbot برای SSL)
   - `2` برای Traefik (SSL خودکار با Let's Encrypt)

2. **Domain پنل مدیریت**: 
   - مثال: `tenantical.iranservat.com`

3. **Base Domain** (برای wildcard SSL):
   - مثال: `iranservat.com`
   - اگر خالی بماند، به صورت خودکار از admin domain استخراج می‌شود

4. **Email برای Let's Encrypt**:
   - برای دریافت گواهی SSL

5. **Backend URL**:
   - URL سرویس backend
   - پیش‌فرض: `http://backend:3000`

6. **مسیر نصب**:
   - پیش‌فرض: `/opt/tenantical`

## چه چیزهایی نصب می‌شود؟

### با انتخاب Nginx:
- Docker & Docker Compose
- Nginx
- Certbot (برای SSL)
- Tenantical Router (در Docker)
- SSL Certificate (از Let's Encrypt)
- Auto-renewal برای SSL
- Systemd service (برای auto-start)

### با انتخاب Traefik:
- Docker & Docker Compose
- Traefik (در Docker)
- Tenantical Router (در Docker)
- SSL Certificate (خودکار با Traefik)
- Systemd service (برای auto-start)

## تنظیمات DNS

قبل یا بعد از نصب، باید DNS records زیر را تنظیم کنید:

```
A     tenantical.iranservat.com    ->    <SERVER_IP>
A     *.iranservat.com             ->    <SERVER_IP>  (اگر از Traefik استفاده می‌کنید)
```

## دسترسی به پنل مدیریت

بعد از نصب و تنظیم DNS:

```
https://tenantical.iranservat.com/admin
```

## دستورات مفید

```bash
cd /opt/tenantical

# مشاهده لاگ‌ها
docker compose logs -f

# بررسی وضعیت
docker compose ps

# راه‌اندازی مجدد
docker compose restart

# توقف سرویس
docker compose down

# شروع مجدد
docker compose up -d
```

### برای Nginx:

```bash
# تست تنظیمات
sudo nginx -t

# بارگذاری مجدد
sudo systemctl reload nginx

# تمدید SSL (دستورات خودکار هم تنظیم شده)
sudo certbot renew
```

## عیب‌یابی

### مشکل در دریافت SSL Certificate:

1. مطمئن شوید DNS records تنظیم شده‌اند
2. مطمئن شوید port 80 باز است
3. بررسی کنید که سرویس tenant-router در حال اجرا است:

```bash
docker compose ps
docker compose logs tenant-router
```

### مشکل در دسترسی به پنل:

1. بررسی کنید که سرویس‌ها در حال اجرا هستند
2. بررسی لاگ‌ها برای خطاها
3. بررسی تنظیمات firewall (ports 80, 443 باید باز باشند)

## حذف نصب

```bash
cd /opt/tenantical
docker compose down
sudo systemctl stop tenantical
sudo systemctl disable tenantical
sudo rm -rf /opt/tenantical
sudo rm /etc/systemd/system/tenantical.service
sudo systemctl daemon-reload
```

برای Nginx:
```bash
sudo rm /etc/nginx/sites-enabled/tenantical
sudo rm /etc/nginx/sites-available/tenantical
sudo systemctl reload nginx
```

