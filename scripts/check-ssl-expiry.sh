#!/bin/bash

# Check SSL Certificate Expiration
# این script برای monitoring expiration date استفاده می‌شود

DOMAIN="${1}"

if [ -z "$DOMAIN" ]; then
    echo "Usage: $0 <domain>"
    echo "Example: $0 example.com"
    exit 1
fi

CERT_PATH="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"

if [ ! -f "$CERT_PATH" ]; then
    echo "❌ Certificate not found: $CERT_PATH"
    exit 1
fi

# Get expiration date
EXPIRY=$(openssl x509 -enddate -noout -in "$CERT_PATH" | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s)
CURRENT_EPOCH=$(date +%s)
DAYS_UNTIL_EXPIRY=$(( (EXPIRY_EPOCH - CURRENT_EPOCH) / 86400 ))

echo "📋 SSL Certificate Information"
echo "=============================="
echo "Domain: $DOMAIN"
echo "Certificate: $CERT_PATH"
echo "Expires on: $EXPIRY"
echo "Days until expiry: $DAYS_UNTIL_EXPIRY"

if [ $DAYS_UNTIL_EXPIRY -lt 30 ]; then
    echo "⚠️  WARNING: Certificate expires in less than 30 days!"
    echo "Run: sudo certbot renew"
    exit 1
elif [ $DAYS_UNTIL_EXPIRY -lt 60 ]; then
    echo "⚠️  Certificate expires in less than 60 days"
    exit 0
else
    echo "✅ Certificate is valid"
    exit 0
fi

