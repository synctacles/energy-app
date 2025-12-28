#!/usr/bin/env bash
# infra-deploy.sh
# Infrastructure deployment for SYNCTACLES
# Version: 1.0 (2025-12-21)
#
# Deploys:
#   - Nginx (reverse proxy + SSL)
#   - Prometheus target config
#   - UptimeRobot monitors
#   - Grafana datasources

set -euo pipefail

RED="\e[31m"; GREEN="\e[32m"; YELLOW="\e[33m"; BLUE="\e[34m"; NC="\e[0m"

DOMAIN="${DOMAIN:-synctacles.io}"
API_PORT="8000"
EMAIL="${EMAIL:-lblom@smartkit.nl}"

header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

info() { echo -e "${BLUE}ℹ${NC} $1"; }
ok() { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; exit 1; }

# Ensure root
[[ $EUID -eq 0 ]] || fail "Must run as root (sudo ./infra-deploy.sh)"

header "SYNCTACLES Infrastructure Deployment"

# NGINX
header "1. Nginx Configuration"

if ! command -v nginx >/dev/null 2>&1; then
    info "Installing Nginx..."
    apt-get update -qq
    apt-get install -y nginx >/dev/null 2>&1
    ok "Nginx installed"
else
    ok "Nginx already installed"
fi

# Create Nginx config
info "Creating Nginx reverse proxy config..."

cat > /etc/nginx/sites-available/synctacles <<EOF
# SYNCTACLES API Reverse Proxy
# Generated: $(date)

upstream synctacles_api {
    server 127.0.0.1:${API_PORT} fail_timeout=0;
}

# HTTP → HTTPS redirect
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN} www.${DOMAIN};
    
    # Let's Encrypt ACME challenge
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

# HTTPS
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ${DOMAIN} www.${DOMAIN};
    
    # SSL certificates (Let's Encrypt)
    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;
    
    # SSL security settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # API proxy
    location /api/ {
        proxy_pass http://synctacles_api;
        proxy_http_version 1.1;
        
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }
    
    # Health endpoint (no auth)
    location /health {
        proxy_pass http://synctacles_api;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
    }
    
    # Metrics endpoint (Prometheus)
    location /metrics {
        proxy_pass http://synctacles_api;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        
        # Restrict to Prometheus server
        allow 127.0.0.1;
        deny all;
    }
    
    # Docs endpoint
    location /docs {
        proxy_pass http://synctacles_api;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
    }
    
    # Root redirect
    location = / {
        return 302 /docs;
    }
    
    # Logging
    access_log /var/log/nginx/synctacles_access.log;
    error_log /var/log/nginx/synctacles_error.log;
}
EOF

ok "Nginx config created"

# Enable site
if [[ ! -L /etc/nginx/sites-enabled/synctacles ]]; then
    ln -sf /etc/nginx/sites-available/synctacles /etc/nginx/sites-enabled/
    ok "Site enabled"
fi

# Test config
if nginx -t >/dev/null 2>&1; then
    ok "Nginx config valid"
else
    fail "Nginx config invalid - check manually: nginx -t"
fi

# SSL Certificate
header "2. SSL Certificate (Let's Encrypt)"

if ! command -v certbot >/dev/null 2>&1; then
    info "Installing certbot..."
    apt-get install -y certbot python3-certbot-nginx >/dev/null 2>&1
    ok "Certbot installed"
fi

if [[ ! -d "/etc/letsencrypt/live/${DOMAIN}" ]]; then
    warn "SSL certificate not found - manual setup required:"
    echo
    echo "  sudo certbot --nginx -d ${DOMAIN} -d www.${DOMAIN} --email ${EMAIL} --agree-tos --non-interactive"
    echo
    read -rp "Run certbot now? (y/N): " RUN_CERTBOT
    
    if [[ "${RUN_CERTBOT,,}" == "y" ]]; then
        certbot --nginx -d "${DOMAIN}" -d "www.${DOMAIN}" --email "${EMAIL}" --agree-tos --non-interactive || warn "Certbot failed - configure manually"
    else
        warn "SSL setup deferred - Nginx won't start without certificates"
    fi
else
    ok "SSL certificate exists: ${DOMAIN}"
fi

# Reload Nginx (if certificates exist)
if [[ -f "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ]]; then
    info "Reloading Nginx..."
    systemctl reload nginx || systemctl restart nginx
    ok "Nginx reloaded"
fi

# PROMETHEUS
header "3. Prometheus Configuration"

PROM_CONFIG="/etc/prometheus/prometheus.yml"

if [[ -f "$PROM_CONFIG" ]]; then
    info "Adding SYNCTACLES target to Prometheus..."
    
    # Check if target already exists
    if grep -q "synctacles-api" "$PROM_CONFIG" 2>/dev/null; then
        ok "SYNCTACLES target already configured"
    else
        # Backup existing config
        cp "$PROM_CONFIG" "${PROM_CONFIG}.bak-$(date +%Y%m%d-%H%M%S)"
        
        # Add SYNCTACLES job
        cat >> "$PROM_CONFIG" <<EOF

  # SYNCTACLES API
  - job_name: 'synctacles-api'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:${API_PORT}']
        labels:
          service: 'synctacles'
          env: 'production'
EOF
        
        ok "SYNCTACLES target added to Prometheus"
        
        # Reload Prometheus
        if systemctl is-active --quiet prometheus 2>/dev/null; then
            systemctl reload prometheus
            ok "Prometheus reloaded"
        fi
    fi
else
    warn "Prometheus config not found - skip target setup"
fi

# UPTIMEROBOT
header "4. UptimeRobot Monitors"

warn "UptimeRobot monitors must be configured manually via web UI:"
echo
echo "  https://uptimerobot.com/dashboard"
echo
echo "Monitors to create:"
echo "  1. API Health:       https://${DOMAIN}/health (HTTP 200, 5min interval)"
echo "  2. Generation Mix:   https://${DOMAIN}/api/v1/generation-mix (HTTP 200, 15min)"
echo "  3. Load:             https://${DOMAIN}/api/v1/load (HTTP 200, 15min)"
echo "  4. Balance:          https://${DOMAIN}/api/v1/balance (HTTP 200, 15min)"
echo "  5. SSL Certificate:  https://${DOMAIN} (SSL expiry alert)"
echo
echo "Alert contacts: ${EMAIL}"
echo

# GRAFANA
header "5. Grafana Datasource"

if systemctl is-active --quiet grafana-server 2>/dev/null; then
    info "Grafana datasource configuration:"
    echo
    echo "  Manual setup required via Grafana UI:"
    echo "  1. Navigate to: http://localhost:3000/datasources"
    echo "  2. Add Prometheus datasource"
    echo "  3. URL: http://localhost:9090"
    echo "  4. Save & Test"
    echo
    
    GRAFANA_IP=$(hostname -I | awk '{print $1}')
    ok "Grafana accessible at: http://${GRAFANA_IP}:3000"
else
    warn "Grafana not running - skip datasource setup"
fi

# FIREWALL
header "6. Firewall Status"

info "Hetzner Cloud Firewall (external):"
echo
echo "  Required rules:"
echo "    - SSH (22)     → Your IP"
echo "    - HTTP (80)    → 0.0.0.0/0"
echo "    - HTTPS (443)  → 0.0.0.0/0"
echo
echo "  Configure at: https://console.hetzner.cloud → Firewalls"
echo

# Summary
header "Infrastructure Deployment Summary"

echo "Domain:             ${DOMAIN}"
echo "API Port:           ${API_PORT}"
echo "Nginx Config:       /etc/nginx/sites-available/synctacles"
echo "SSL Certificate:    /etc/letsencrypt/live/${DOMAIN}/"
echo "Prometheus Config:  ${PROM_CONFIG}"
echo

ok "Infrastructure deployment complete!"
echo
echo "Manual steps remaining:"
echo "  1. Configure UptimeRobot monitors (see above)"
echo "  2. Add Prometheus datasource to Grafana"
echo "  3. Verify Hetzner firewall rules"
echo
echo "Validation commands:"
echo "  curl -I https://${DOMAIN}/health"
echo "  curl -I https://${DOMAIN}/api/v1/generation-mix"
echo "  systemctl status nginx prometheus grafana-server"
