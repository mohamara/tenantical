#!/bin/bash

# Tenantical Router - Automated Installation Script
# این اسکریپت تمام مراحل نصب و راه‌اندازی را به صورت خودکار انجام می‌دهد

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored messages
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then 
        print_error "Please run as root (use sudo)"
        exit 1
    fi
}

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        print_error "Cannot detect OS. Please run on Ubuntu/Debian or CentOS/RHEL"
        exit 1
    fi
    
    print_info "Detected OS: $OS $OS_VERSION"
}

# Install Docker
install_docker() {
    if command -v docker &> /dev/null; then
        print_success "Docker is already installed"
        return
    fi
    
    print_info "Installing Docker..."
    
    if [ "$OS" = "ubuntu" ] || [ "$OS" = "debian" ]; then
        # Update package index
        apt-get update
        
        # Install prerequisites
        apt-get install -y \
            ca-certificates \
            curl \
            gnupg \
            lsb-release
        
        # Add Docker's official GPG key
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/${OS}/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        chmod a+r /etc/apt/keyrings/docker.gpg
        
        # Set up repository
        echo \
          "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${OS} \
          $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        
        # Install Docker Engine
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
        
    elif [ "$OS" = "centos" ] || [ "$OS" = "rhel" ]; then
        # Install prerequisites
        yum install -y yum-utils
        
        # Add Docker repository
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        
        # Install Docker Engine
        yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    else
        print_error "Unsupported OS. Please install Docker manually."
        exit 1
    fi
    
    # Start and enable Docker
    systemctl start docker
    systemctl enable docker
    
    print_success "Docker installed successfully"
}

# Install Docker Compose (standalone if not included)
install_docker_compose() {
    if docker compose version &> /dev/null 2>&1; then
        print_success "Docker Compose is already installed"
        return
    fi
    
    print_info "Docker Compose plugin should be installed with Docker. Checking..."
    
    # If not available, install standalone version
    if ! command -v docker-compose &> /dev/null; then
        print_info "Installing Docker Compose standalone..."
        curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        chmod +x /usr/local/bin/docker-compose
        print_success "Docker Compose installed successfully"
    fi
}

# Install Nginx (if selected)
install_nginx() {
    if command -v nginx &> /dev/null; then
        print_success "Nginx is already installed"
        return
    fi
    
    print_info "Installing Nginx..."
    
    if [ "$OS" = "ubuntu" ] || [ "$OS" = "debian" ]; then
        apt-get update
        apt-get install -y nginx
    elif [ "$OS" = "centos" ] || [ "$OS" = "rhel" ]; then
        yum install -y nginx
    fi
    
    systemctl start nginx
    systemctl enable nginx
    
    print_success "Nginx installed successfully"
}

# Install Certbot (for SSL with Nginx)
install_certbot() {
    if command -v certbot &> /dev/null; then
        print_success "Certbot is already installed"
        return
    fi
    
    print_info "Installing Certbot..."
    
    if [ "$OS" = "ubuntu" ] || [ "$OS" = "debian" ]; then
        apt-get update
        apt-get install -y certbot python3-certbot-nginx
    elif [ "$OS" = "centos" ] || [ "$OS" = "rhel" ]; then
        yum install -y certbot python3-certbot-nginx
    fi
    
    print_success "Certbot installed successfully"
}

# Get user inputs
get_user_inputs() {
    echo ""
    print_info "Tenantical Router Installation"
    echo "===================================="
    echo ""
    
    # Choose reverse proxy
    echo "Choose reverse proxy:"
    echo "1) Nginx (with Certbot for SSL)"
    echo "2) Traefik (built-in SSL with Let's Encrypt)"
    read -p "Enter choice [1-2] (default: 1): " PROXY_CHOICE
    PROXY_CHOICE=${PROXY_CHOICE:-1}
    
    if [ "$PROXY_CHOICE" != "1" ] && [ "$PROXY_CHOICE" != "2" ]; then
        print_error "Invalid choice. Using default: Nginx"
        PROXY_CHOICE=1
    fi
    
    # Get admin domain
    read -p "Enter admin panel domain (e.g., tenantical.iranservat.com): " ADMIN_DOMAIN
    if [ -z "$ADMIN_DOMAIN" ]; then
        print_error "Admin domain is required"
        exit 1
    fi
    
    # Get base domain (for wildcard SSL)
    read -p "Enter base domain for wildcard SSL (e.g., iranservat.com) [optional]: " BASE_DOMAIN
    if [ -z "$BASE_DOMAIN" ]; then
        BASE_DOMAIN=$(echo $ADMIN_DOMAIN | sed 's/^[^.]*\.//')
        print_info "Using base domain: $BASE_DOMAIN"
    fi
    
    # Get email for Let's Encrypt
    read -p "Enter email for Let's Encrypt certificates: " SSL_EMAIL
    if [ -z "$SSL_EMAIL" ]; then
        SSL_EMAIL="admin@${BASE_DOMAIN}"
        print_info "Using email: $SSL_EMAIL"
    fi
    
    # Get backend URL
    read -p "Enter backend URL (default: http://backend:3000): " BACKEND_URL
    BACKEND_URL=${BACKEND_URL:-http://backend:3000}
    
    # Get installation directory
    read -p "Enter installation directory (default: /opt/tenantical): " INSTALL_DIR
    INSTALL_DIR=${INSTALL_DIR:-/opt/tenantical}
    
    # Get project source directory
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
    
    read -p "Enter project source directory (default: $PROJECT_DIR): " SOURCE_DIR
    SOURCE_DIR=${SOURCE_DIR:-$PROJECT_DIR}
}

# Create directory structure
create_directories() {
    print_info "Creating directory structure..."
    mkdir -p "$INSTALL_DIR"/{data,config,nginx,traefik,letsencrypt}
    print_success "Directories created"
}

# Copy project files
copy_project_files() {
    print_info "Copying project files..."
    
    if [ ! -f "$SOURCE_DIR/Dockerfile" ]; then
        print_error "Dockerfile not found in $SOURCE_DIR"
        print_error "Please run this script from the project directory or specify correct source directory"
        exit 1
    fi
    
    # Copy essential files
    cp "$SOURCE_DIR/Dockerfile" "$INSTALL_DIR/"
    cp "$SOURCE_DIR/go.mod" "$INSTALL_DIR/" 2>/dev/null || true
    cp "$SOURCE_DIR/go.sum" "$INSTALL_DIR/" 2>/dev/null || true
    
    # Copy source code
    mkdir -p "$INSTALL_DIR/cmd" "$INSTALL_DIR/internal"
    cp -r "$SOURCE_DIR/cmd"/* "$INSTALL_DIR/cmd/" 2>/dev/null || true
    cp -r "$SOURCE_DIR/internal"/* "$INSTALL_DIR/internal/" 2>/dev/null || true
    
    # Copy bin directory if exists
    if [ -d "$SOURCE_DIR/bin" ]; then
        cp -r "$SOURCE_DIR/bin" "$INSTALL_DIR/" 2>/dev/null || true
    fi
    
    print_success "Project files copied"
}

# Create docker-compose.yml for Nginx
create_docker_compose_nginx() {
    print_info "Creating docker-compose.yml for Nginx setup..."
    
    cat > "$INSTALL_DIR/docker-compose.yml" <<EOF
version: '3.8'

services:
  tenant-router:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: tenant-router
    restart: unless-stopped
    environment:
      - HOST=0.0.0.0
      - PORT=8080
      - BACKEND_URL=${BACKEND_URL}
      - DB_PATH=/data/tenants.db
      - ADMIN_DOMAIN=${ADMIN_DOMAIN}
    volumes:
      - ./data:/data
    ports:
      - "127.0.0.1:8080:8080"
    networks:
      - tenantical-network
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s

networks:
  tenantical-network:
    driver: bridge
EOF
    
    print_success "docker-compose.yml created"
}

# Create docker-compose.yml for Traefik
create_docker_compose_traefik() {
    print_info "Creating docker-compose.yml for Traefik setup..."
    
    cat > "$INSTALL_DIR/docker-compose.yml" <<EOF
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    container_name: traefik
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt
    command:
      # API
      - --api.dashboard=true
      - --api.insecure=false
      
      # Entrypoints
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      
      # Redirect HTTP to HTTPS
      - --entrypoints.web.http.redirections.entrypoint.to=websecure
      - --entrypoints.web.http.redirections.entrypoint.scheme=https
      
      # Let's Encrypt
      - --certificatesresolvers.letsencrypt.acme.email=${SSL_EMAIL}
      - --certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json
      - --certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web
      
      # Docker provider
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --providers.docker.network=tenantical-network
      
      # Logging
      - --log.level=INFO
      - --accesslog=true
    networks:
      - tenantical-network

  tenant-router:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: tenant-router
    restart: unless-stopped
    environment:
      - HOST=0.0.0.0
      - PORT=8080
      - BACKEND_URL=${BACKEND_URL}
      - DB_PATH=/data/tenants.db
      - ADMIN_DOMAIN=${ADMIN_DOMAIN}
    volumes:
      - ./data:/data
    labels:
      # Enable Traefik
      - "traefik.enable=true"
      
      # HTTP Router (redirect to HTTPS)
      - "traefik.http.routers.tenant-router-http.rule=HostRegexp(\`{subdomain:.+}.${BASE_DOMAIN}\`) || Host(\`${ADMIN_DOMAIN}\`)"
      - "traefik.http.routers.tenant-router-http.entrypoints=web"
      - "traefik.http.routers.tenant-router-http.middlewares=redirect-to-https"
      
      # HTTPS Router
      - "traefik.http.routers.tenant-router.rule=HostRegexp(\`{subdomain:.+}.${BASE_DOMAIN}\`) || Host(\`${ADMIN_DOMAIN}\`)"
      - "traefik.http.routers.tenant-router.entrypoints=websecure"
      - "traefik.http.routers.tenant-router.tls.certresolver=letsencrypt"
      
      # Service
      - "traefik.http.services.tenant-router.loadbalancer.server.port=8080"
      
      # Middleware
      - "traefik.http.middlewares.redirect-to-https.redirectscheme.scheme=https"
      - "traefik.http.middlewares.redirect-to-https.redirectscheme.permanent=true"
    networks:
      - tenantical-network
    depends_on:
      - traefik

networks:
  tenantical-network:
    driver: bridge
EOF
    
    # Create acme.json file with correct permissions
    touch "$INSTALL_DIR/letsencrypt/acme.json"
    chmod 600 "$INSTALL_DIR/letsencrypt/acme.json"
    
    print_success "docker-compose.yml created"
}

# Create nginx configuration
create_nginx_config() {
    print_info "Creating Nginx configuration..."
    
    cat > "$INSTALL_DIR/config/nginx.conf" <<EOF
# HTTP Server - Redirect to HTTPS + ACME Challenge
server {
    listen 80;
    listen [::]:80;
    server_name ${ADMIN_DOMAIN} *.${BASE_DOMAIN};

    # Let's Encrypt ACME Challenge
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
    server_name ${ADMIN_DOMAIN} *.${BASE_DOMAIN};

    # SSL Certificates
    ssl_certificate /etc/letsencrypt/live/${BASE_DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${BASE_DOMAIN}/privkey.pem;

    # SSL Protocols
    ssl_protocols TLSv1.2 TLSv1.3;
    
    # SSL Ciphers
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305';
    ssl_prefer_server_ciphers off;
    
    # SSL Session
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_session_tickets off;

    # OCSP Stapling
    ssl_stapling on;
    ssl_stapling_verify on;
    ssl_trusted_certificate /etc/letsencrypt/live/${BASE_DOMAIN}/chain.pem;
    resolver 8.8.8.8 8.8.4.4 valid=300s;
    resolver_timeout 5s;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Client body size limit
    client_max_body_size 10M;

    # Timeouts
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;

    # Forward to Tenant Router
    location / {
        proxy_pass http://127.0.0.1:8080;
        
        # Preserve original headers
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Original-Host \$host;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # Buffer settings
        proxy_buffering off;
        proxy_request_buffering off;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://127.0.0.1:8080/health;
        access_log off;
    }
}
EOF
    
    print_success "Nginx configuration created"
}

# Setup SSL with Certbot (for Nginx)
setup_ssl_certbot() {
    print_info "Setting up SSL with Certbot..."
    
    # Create temporary nginx config for certbot challenge
    mkdir -p /var/www/html
    cat > /etc/nginx/sites-available/tenantical-temp <<EOF
server {
    listen 80;
    server_name ${ADMIN_DOMAIN} *.${BASE_DOMAIN};

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
    
    # Enable temporary config
    ln -sf /etc/nginx/sites-available/tenantical-temp /etc/nginx/sites-enabled/tenantical-temp
    
    # Remove default nginx config if exists
    rm -f /etc/nginx/sites-enabled/default
    
    # Test and reload nginx
    nginx -t && systemctl reload nginx
    
    # Wait a bit for nginx to be ready
    sleep 3
    
    # Get certificate (non-wildcard for simplicity, can be extended)
    print_info "Getting SSL certificate for ${BASE_DOMAIN} and ${ADMIN_DOMAIN}..."
    
    certbot certonly --nginx \
        -d "${BASE_DOMAIN}" \
        -d "${ADMIN_DOMAIN}" \
        --email "${SSL_EMAIL}" \
        --agree-tos \
        --non-interactive \
        --redirect || {
        print_error "Failed to get SSL certificate. Please check DNS settings and try again."
        print_info "Make sure DNS A records are pointing to this server:"
        print_info "  ${BASE_DOMAIN} -> $(hostname -I | awk '{print $1}')"
        print_info "  ${ADMIN_DOMAIN} -> $(hostname -I | awk '{print $1}')"
        exit 1
    }
    
    # Copy nginx config to /etc/nginx/sites-available
    cp "$INSTALL_DIR/config/nginx.conf" /etc/nginx/sites-available/tenantical
    ln -sf /etc/nginx/sites-available/tenantical /etc/nginx/sites-enabled/tenantical
    rm -f /etc/nginx/sites-enabled/tenantical-temp
    
    # Test and reload nginx
    nginx -t && systemctl reload nginx
    
    # Setup auto-renewal
    print_info "Setting up SSL auto-renewal..."
    mkdir -p /etc/letsencrypt/renewal-hooks/deploy
    cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh <<'HOOK'
#!/bin/bash
systemctl reload nginx
HOOK
    chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
    
    # Test auto-renewal
    certbot renew --dry-run > /dev/null 2>&1
    
    print_success "SSL certificate installed and configured"
}

# Build and start services
build_and_start() {
    print_info "Building and starting services..."
    
    cd "$INSTALL_DIR"
    
    # Build Docker image
    docker compose build tenant-router
    
    # Start services
    docker compose up -d
    
    # Wait for services to be healthy
    print_info "Waiting for services to be ready..."
    sleep 10
    
    # Check status
    if docker compose ps | grep -q "Up"; then
        print_success "Services started successfully"
    else
        print_warning "Services started but may still be initializing"
    fi
}

# Create systemd service (optional, for auto-start)
create_systemd_service() {
    print_info "Creating systemd service..."
    
    cat > /etc/systemd/system/tenantical.service <<EOF
[Unit]
Description=Tenantical Router
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${INSTALL_DIR}
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable tenantical.service
    
    print_success "Systemd service created and enabled"
}

# Print summary
print_summary() {
    SERVER_IP=$(hostname -I | awk '{print $1}')
    
    echo ""
    echo "=========================================="
    print_success "Installation completed successfully!"
    echo "=========================================="
    echo ""
    echo "Installation directory: $INSTALL_DIR"
    echo "Admin panel domain: https://${ADMIN_DOMAIN}"
    echo ""
    
    if [ "$PROXY_CHOICE" = "1" ]; then
        echo "Reverse proxy: Nginx"
        echo "SSL: Let's Encrypt (auto-renewal configured)"
    else
        echo "Reverse proxy: Traefik"
        echo "SSL: Let's Encrypt (managed by Traefik)"
    fi
    
    echo ""
    echo "Next steps:"
    echo "1. Make sure DNS records are configured:"
    echo "   - ${ADMIN_DOMAIN} -> ${SERVER_IP}"
    if [ "$PROXY_CHOICE" = "2" ]; then
        echo "   - *.${BASE_DOMAIN} -> ${SERVER_IP}"
    fi
    echo ""
    echo "2. Access the admin panel at:"
    echo "   https://${ADMIN_DOMAIN}/admin"
    echo ""
    echo "3. Useful commands:"
    echo "   cd $INSTALL_DIR"
    echo "   docker compose logs -f          # View logs"
    echo "   docker compose ps               # Check status"
    echo "   docker compose restart          # Restart services"
    echo "   docker compose down             # Stop services"
    echo ""
    
    if [ "$PROXY_CHOICE" = "1" ]; then
        echo "   sudo nginx -t                # Test nginx config"
        echo "   sudo systemctl reload nginx  # Reload nginx"
        echo "   sudo certbot renew           # Renew SSL certificate"
    fi
    
    echo ""
}

# Main installation function
main() {
    echo ""
    print_info "Tenantical Router - Automated Installation"
    echo "================================================"
    echo ""
    
    check_root
    detect_os
    get_user_inputs
    
    print_info "Starting installation..."
    echo ""
    
    # Install dependencies
    install_docker
    install_docker_compose
    
    if [ "$PROXY_CHOICE" = "1" ]; then
        install_nginx
        install_certbot
    fi
    
    # Create directories
    create_directories
    
    # Copy project files
    copy_project_files
    
    # Create configurations
    if [ "$PROXY_CHOICE" = "1" ]; then
        create_docker_compose_nginx
        create_nginx_config
    else
        create_docker_compose_traefik
    fi
    
    # Build and start services (before SSL for Nginx, as certbot needs the service running)
    if [ "$PROXY_CHOICE" = "1" ]; then
        build_and_start
        setup_ssl_certbot
    else
        build_and_start
    fi
    
    # Create systemd service
    create_systemd_service
    
    # Print summary
    print_summary
}

# Run main function
main "$@"
