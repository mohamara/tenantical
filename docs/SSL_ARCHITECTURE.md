# SSL/TLS Architecture تصمیم‌گیری

این سند راهنمای تصمیم‌گیری برای معماری SSL/TLS termination است.

## 🎯 توصیه: SSL Termination در Reverse Proxy

**Best Practice:** استفاده از **nginx** یا **traefik** برای SSL termination

### چرا Reverse Proxy بهتر است؟

1. **Performance**: nginx/traefik برای SSL بهینه‌سازی شده‌اند (OpenSSL optimizations)
2. **مدیریت ساده**: گواهی‌ها در یک مکان متمرکز
3. **Auto-renewal**: cert-manager یا traefik می‌توانند خودکار تمدید کنند
4. **Wildcard Certificates**: مدیریت یک wildcard cert برای تمام subdomains
5. **Load Balancing**: اگر چند instance دارید، load balancing راحت‌تر است
6. **SSL Offloading**: کاهش بار CPU از application

### معماری پیشنهادی

```
[Client] 
   ↓ HTTPS (443)
[nginx/traefik] ← SSL Termination
   ↓ HTTP (8080)
[Tenant Router] ← Application
   ↓ HTTP
[Backend API]
```

### مثال: nginx با Let's Encrypt

```nginx
# همه subdomains با یک wildcard certificate
server {
    listen 443 ssl http2;
    server_name *.example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
    }
}
```

### مثال: Traefik (با Auto ACME)

```yaml
# docker-compose.yml
services:
  traefik:
    image: traefik:v2.10
    command:
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.letsencrypt.acme.email=admin@example.com
      - --certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json
      - --certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./letsencrypt:/letsencrypt

  tenant-router:
    image: tenant-router:latest
    labels:
      - "traefik.http.routers.tenant-router.rule=HostRegexp(`{subdomain:.+}.example.com`)"
      - "traefik.http.routers.tenant-router.entrypoints=websecure"
      - "traefik.http.routers.tenant-router.tls.certresolver=letsencrypt"
```

## 🔄 Alternative: SSL در Tenant Router (Self-contained)

اگر می‌خواهید **self-contained** باشید و بدون reverse proxy کار کنید، می‌توانید TLS را مستقیماً در Tenant Router پیاده‌سازی کنید.

### مزایا
- ✅ Self-contained: بدون نیاز به nginx/traefik
- ✅ Less dependencies
- ✅ Direct control

### معایب
- ❌ Performance کمتر نسبت به nginx
- ❌ مدیریت گواهی‌ها پیچیده‌تر
- ❌ نیاز به reload برای تغییر certs
- ❌ Auto-renewal پیچیده‌تر

### Implementation Options

#### Option 1: Single Wildcard Certificate

یک wildcard certificate برای تمام domains:

```go
// Simple: یک cert برای همه
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
}
```

#### Option 2: Multiple Certificates (SNI)

پشتیبانی از multiple certificates با SNI (Server Name Indication):

```go
// Advanced: SNI support
tlsConfig := &tls.Config{
    GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
        // Return certificate based on clientHello.ServerName
        return getCertForDomain(clientHello.ServerName)
    },
}
```

## 📊 مقایسه

| Feature | Reverse Proxy | Tenant Router TLS |
|---------|--------------|-------------------|
| Performance | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Management | ⭐⭐⭐⭐⭐ | ⭐⭐ |
| Auto-renewal | ⭐⭐⭐⭐⭐ | ⭐⭐ |
| Complexity | ⭐⭐⭐⭐ | ⭐⭐ |
| Self-contained | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| Production Ready | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

## 🎯 توصیه نهایی

### برای Production:

**استفاده از Reverse Proxy (nginx/traefik)** برای SSL termination:

1. Performance بهتر
2. مدیریت آسان‌تر
3. Auto-renewal با cert-manager
4. Industry standard

### برای Development/Simple Deployments:

می‌توانید از **Tenant Router با TLS** استفاده کنید اگر:
- نیاز به self-contained deployment دارید
- تعداد tenants کم است
- Performance critical نیست
- می‌خواهید setup ساده‌تری داشته باشید

## 📝 نتیجه‌گیری

**توصیه ما:** از **nginx/traefik** برای SSL termination استفاده کنید و Tenant Router را روی HTTP داخلی اجرا کنید. این approach:
- Best practice industry است
- Performance بهتری دارد
- Management آسان‌تر است
- Production-ready است

اگر واقعاً نیاز به self-contained دارید، می‌توانیم TLS support را به Tenant Router اضافه کنیم.

