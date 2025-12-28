# راهنمای Setup SSL با Certbot

این راهنما نحوه تنظیم SSL certificates با **certbot** برای Tenant Router را توضیح می‌دهد.

## معماری

```
[Client] HTTPS (443)
    ↓
[nginx] ← SSL Termination (certificates here!)
    ↓ HTTP (8080)
[Tenant Router]
    ↓ HTTP
[Backend Services]
```

**مهم:** SSL certificates در **nginx** (reverse proxy) ست می‌شوند، نه در Tenant Router.

---

## نصب Certbot

### Ubuntu/Debian
```bash
sudo apt update
sudo apt install certbot python3-certbot-nginx
```

### CentOS/RHEL
```bash
sudo yum install certbot python3-certbot-nginx
```

---

## Setup SSL برای Single Domain

### مرحله 1: تنظیم اولیه nginx (HTTP)

ابتدا nginx را با HTTP تنظیم کنید:

```nginx
# /etc/nginx/sites-available/tenant-router
server {
    listen 80;
    server_name tenant1.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

Enable و restart nginx:
```bash
sudo ln -s /etc/nginx/sites-available/tenant-router /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### مرحله 2: دریافت SSL Certificate با Certbot

```bash
sudo certbot --nginx -d tenant1.example.com
```

Certbot:
- Certificate را از Let's Encrypt دریافت می‌کند
- nginx config را به‌صورت خودکار update می‌کند
- Auto-renewal را setup می‌کند

---

## Setup SSL برای Wildcard Domain (Recommended)

برای اینکه تمام subdomains با یک certificate پوشش داده شوند، از **wildcard certificate** استفاده کنید.

### روش 1: DNS Challenge (برای Wildcard)

```bash
# Wildcard certificate برای *.example.com
sudo certbot certonly --manual --preferred-challenges dns \
  -d "*.example.com" \
  -d "example.com" \
  --email admin@example.com \
  --agree-tos \
  --manual-public-ip-logging-ok
```

Certbot یک TXT record برای DNS می‌خواهد. آن را در DNS provider خود اضافه کنید:

```
_acme-challenge.example.com TXT "xxxxx-xxxxx-xxxxx"
```

### روش 2: استفاده از DNS Plugin (Automatic)

اگر DNS provider شما plugin دارد (مثل Cloudflare, Route53):

```bash
# Cloudflare example
sudo certbot certonly \
  --dns-cloudflare \
  --dns-cloudflare-credentials ~/.secrets/cloudflare.ini \
  -d "*.example.com" \
  -d "example.com"
```

---

## تنظیم nginx با Wildcard Certificate

بعد از دریافت certificate:

```nginx
# /etc/nginx/sites-available/tenant-router
# HTTP Server - Redirect to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name *.example.com example.com;

    # Let's Encrypt challenge
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # Redirect all other traffic to HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
}

# HTTPS Server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name *.example.com example.com;

    # SSL Certificate (wildcard)
    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    # SSL Configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # HSTS (optional but recommended)
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Forward to Tenant Router
    location / {
        proxy_pass http://127.0.0.1:8080;
        
        # Preserve headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Original-Host $host;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check
    location /health {
        proxy_pass http://127.0.0.1:8080/health;
        access_log off;
    }
}
```

---

## Auto-Renewal (تمدید خودکار)

Certbot به‌صورت خودکار یک **systemd timer** یا **cron job** برای تمدید ایجاد می‌کند.

### بررسی Auto-Renewal

```bash
# Check certbot timer (systemd)
sudo systemctl status certbot.timer

# Test renewal (dry run)
sudo certbot renew --dry-run
```

### اگر نیاز به reload nginx بعد از renewal دارید

یک hook script ایجاد کنید:

```bash
# /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
#!/bin/bash
systemctl reload nginx
```

اجازه اجرا بدهید:
```bash
sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
```

### Manual Renewal

اگر می‌خواهید دستی تمدید کنید:

```bash
sudo certbot renew
sudo systemctl reload nginx
```

---

## مدیریت Multiple Domains

اگر چند domain دارید که نمی‌توانید wildcard استفاده کنید:

### Option 1: Multiple Certificates

```bash
# Certificate 1
sudo certbot --nginx -d domain1.com -d www.domain1.com

# Certificate 2
sudo certbot --nginx -d domain2.com -d www.domain2.com
```

### Option 2: SAN Certificate (Multiple Domains)

```bash
sudo certbot --nginx \
  -d domain1.com \
  -d domain2.com \
  -d domain3.com
```

### nginx Configuration برای Multiple Certificates

```nginx
# Domain 1
server {
    listen 443 ssl http2;
    server_name domain1.com www.domain1.com;
    
    ssl_certificate /etc/letsencrypt/live/domain1.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/domain1.com/privkey.pem;
    
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
    }
}

# Domain 2
server {
    listen 443 ssl http2;
    server_name domain2.com www.domain2.com;
    
    ssl_certificate /etc/letsencrypt/live/domain2.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/domain2.com/privkey.pem;
    
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
    }
}
```

---

## چک‌لیست Setup

- [ ] Certbot نصب شده
- [ ] nginx نصب و running است
- [ ] DNS records صحیح هستند (A/AAAA records)
- [ ] Port 80 و 443 باز هستند
- [ ] Certificate دریافت شده
- [ ] nginx config تست شده (`nginx -t`)
- [ ] Auto-renewal فعال است (`certbot renew --dry-run`)
- [ ] nginx reload می‌شود بعد از renewal

---

## Troubleshooting

### مشکل: Certificate دریافت نمی‌شود

```bash
# Check nginx is running
sudo systemctl status nginx

# Check port 80 is open
sudo netstat -tulpn | grep :80

# Check DNS
dig example.com
nslookup example.com

# Check certbot logs
sudo tail -f /var/log/letsencrypt/letsencrypt.log
```

### مشکل: Auto-renewal کار نمی‌کند

```bash
# Check certbot timer
sudo systemctl status certbot.timer
sudo systemctl enable certbot.timer

# Test renewal manually
sudo certbot renew --dry-run

# Check renewal logs
sudo journalctl -u certbot.timer
```

### مشکل: Certificate منقضی می‌شود

```bash
# Check certificate expiration
sudo certbot certificates

# Manual renewal
sudo certbot renew --force-renewal
sudo systemctl reload nginx
```

---

## Best Practices

1. **از Wildcard Certificate استفاده کنید** اگر تمام subdomains در یک domain هستند
2. **Auto-renewal را فعال کنید** - certbot این کار را خودکار انجام می‌دهد
3. **نظارت بر Expiration**: یک monitoring برای expiration date setup کنید
4. **Backup Certificates**: certificates را backup کنید

---

## مثال کامل: Setup با Wildcard

```bash
# 1. Install certbot
sudo apt install certbot python3-certbot-nginx

# 2. Get wildcard certificate (DNS challenge)
sudo certbot certonly --manual --preferred-challenges dns \
  -d "*.example.com" -d "example.com" \
  --email admin@example.com --agree-tos

# 3. Configure nginx (use example above)

# 4. Test nginx config
sudo nginx -t

# 5. Restart nginx
sudo systemctl restart nginx

# 6. Test SSL
curl -I https://tenant1.example.com

# 7. Verify auto-renewal
sudo certbot renew --dry-run
```

---

## مکان فایل‌های Certificate

Certificates در این مسیر ذخیره می‌شوند:

```
/etc/letsencrypt/live/{domain}/
├── fullchain.pem  ← SSL certificate (use this)
├── privkey.pem    ← Private key
├── cert.pem       ← Certificate only
└── chain.pem      ← Chain only
```

**نکته:** فقط `fullchain.pem` و `privkey.pem` را در nginx استفاده کنید.

---

## خلاصه

1. ✅ Certbot در **reverse proxy (nginx)** ست می‌شود
2. ✅ از **wildcard certificate** برای تمام subdomains استفاده کنید
3. ✅ **Auto-renewal** به‌صورت خودکار توسط certbot فعال می‌شود
4. ✅ Tenant Router نیازی به SSL ندارد (فقط HTTP داخلی)
5. ✅ nginx config را بعد از certificate setup کنید

