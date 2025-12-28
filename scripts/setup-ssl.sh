#!/bin/bash

# Setup SSL with Certbot for Tenant Router
# این script برای setup اولیه SSL certificates استفاده می‌شود

set -e

DOMAIN="${1:-example.com}"
EMAIL="${2:-admin@${DOMAIN}}"
WILDCARD="${3:-true}"

echo "🔒 SSL Setup Script for Tenant Router"
echo "======================================"
echo "Domain: $DOMAIN"
echo "Email: $EMAIL"
echo "Wildcard: $WILDCARD"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Please run as root (use sudo)"
    exit 1
fi

# Check if certbot is installed
if ! command -v certbot &> /dev/null; then
    echo "📦 Installing certbot..."
    if command -v apt-get &> /dev/null; then
        apt-get update
        apt-get install -y certbot python3-certbot-nginx
    elif command -v yum &> /dev/null; then
        yum install -y certbot python3-certbot-nginx
    else
        echo "❌ Cannot detect package manager. Please install certbot manually."
        exit 1
    fi
fi

# Check if nginx is installed
if ! command -v nginx &> /dev/null; then
    echo "❌ nginx is not installed. Please install nginx first."
    exit 1
fi

# Create nginx config directory if it doesn't exist
NGINX_CONF="/etc/nginx/sites-available/tenant-router"
if [ ! -f "$NGINX_CONF" ]; then
    echo "📝 Creating nginx configuration..."
    cat > "$NGINX_CONF" <<EOF
# HTTP Server - Temporary config for certbot
server {
    listen 80;
    server_name ${DOMAIN} *.$DOMAIN;

    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
}
EOF
    ln -sf "$NGINX_CONF" /etc/nginx/sites-enabled/tenant-router
    nginx -t && systemctl reload nginx
fi

# Get certificate
if [ "$WILDCARD" = "true" ]; then
    echo "🔐 Getting wildcard certificate for *.$DOMAIN..."
    echo "⚠️  You will need to add a DNS TXT record when prompted"
    certbot certonly --manual --preferred-challenges dns \
        -d "*.$DOMAIN" \
        -d "$DOMAIN" \
        --email "$EMAIL" \
        --agree-tos \
        --manual-public-ip-logging-ok
else
    echo "🔐 Getting certificate for $DOMAIN..."
    certbot --nginx -d "$DOMAIN" --email "$EMAIL" --agree-tos --non-interactive
fi

# Update nginx config with SSL
echo "📝 Updating nginx configuration with SSL..."
cat > "$NGINX_CONF" <<EOF
# HTTP Server - Redirect to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN} *.$DOMAIN;

    # Let's Encrypt challenge
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # Redirect all other traffic to HTTPS
    location / {
        return 301 https://\$host\$request_uri;
    }
}

# HTTPS Server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ${DOMAIN} *.$DOMAIN;

    # SSL Certificate
    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

    # SSL Configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Forward to Tenant Router
    location / {
        proxy_pass http://127.0.0.1:8080;
        
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Original-Host \$host;
        
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        
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
EOF

# Test and reload nginx
echo "✅ Testing nginx configuration..."
nginx -t

echo "🔄 Reloading nginx..."
systemctl reload nginx

# Setup auto-renewal hook
echo "📋 Setting up auto-renewal hook..."
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh <<'HOOK'
#!/bin/bash
systemctl reload nginx
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh

# Test auto-renewal
echo "🧪 Testing auto-renewal..."
certbot renew --dry-run

echo ""
echo "✅ SSL setup completed!"
echo ""
echo "Certificate location: /etc/letsencrypt/live/${DOMAIN}/"
echo "Nginx config: $NGINX_CONF"
echo ""
echo "To check certificate expiration:"
echo "  sudo certbot certificates"
echo ""
echo "To manually renew:"
echo "  sudo certbot renew"
echo "  sudo systemctl reload nginx"

