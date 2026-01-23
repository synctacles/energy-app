#!/bin/bash
# SYNCTACLES Performance Setup
# Run after base installation (FASE 1-3)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "========================================"
echo "  SYNCTACLES Performance Setup"
echo "========================================"

# Source environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
else
    echo "❌ /opt/.env not found. Run FASE 0 first."
    exit 1
fi

# 1. TCP Tuning
echo ""
echo "--- Step 1: TCP Kernel Tuning ---"
"$SCRIPT_DIR/optimize_tcp.sh"

# 2. Gunicorn Service (if exists, update it)
echo ""
echo "--- Step 2: Gunicorn Service ---"
if [[ -f "$REPO_DIR/systemd/api.service.template" ]]; then
    sed -e "s|{{BRAND_NAME}}|$BRAND_NAME|g" \
        -e "s|{{SERVICE_USER}}|$SERVICE_USER|g" \
        -e "s|{{SERVICE_GROUP}}|$SERVICE_GROUP|g" \
        -e "s|{{APP_PATH}}|$APP_PATH|g" \
        -e "s|{{INSTALL_PATH}}|$INSTALL_PATH|g" \
        -e "s|{{API_PORT}}|${API_PORT:-8000}|g" \
        -e "s|{{ENV_FILE}}|/opt/.env|g" \
        "$REPO_DIR/systemd/api.service.template" \
        > "/etc/systemd/system/${BRAND_SLUG}-api.service"

    systemctl daemon-reload
    echo "✅ Gunicorn service updated"
else
    echo "⚠️ Service template not found, skipping"
fi

# 3. Nginx Config (if nginx installed)
echo ""
echo "--- Step 3: Nginx Configuration ---"
if command -v nginx &> /dev/null; then
    if [[ -f "$REPO_DIR/config/nginx/api.conf.template" ]]; then
        sed -e "s|{{BRAND_NAME}}|$BRAND_NAME|g" \
            -e "s|{{BRAND_DOMAIN}}|${BRAND_DOMAIN:-_}|g" \
            -e "s|{{API_PORT}}|${API_PORT:-8000}|g" \
            "$REPO_DIR/config/nginx/api.conf.template" \
            > "/etc/nginx/sites-available/${BRAND_SLUG}"

        ln -sf "/etc/nginx/sites-available/${BRAND_SLUG}" /etc/nginx/sites-enabled/

        if nginx -t; then
            systemctl reload nginx
            echo "✅ Nginx config updated"
        else
            echo "❌ Nginx config invalid"
            exit 1
        fi
    else
        echo "⚠️ Nginx template not found, skipping"
    fi
else
    echo "⚠️ Nginx not installed, skipping"
fi

# 4. Restart API
echo ""
echo "--- Step 4: Restart Services ---"
if systemctl is-active --quiet "${BRAND_SLUG}-api"; then
    systemctl restart "${BRAND_SLUG}-api"
    sleep 3
    echo "✅ API restarted"
fi

# 5. Verify
echo ""
echo "--- Verification ---"
echo "TCP settings:"
sysctl net.ipv4.tcp_tw_reuse net.ipv4.tcp_fin_timeout net.core.somaxconn

echo ""
echo "Gunicorn workers:"
ps aux | grep "[g]unicorn" | wc -l

echo ""
echo "========================================"
echo "  Performance Setup Complete"
echo "========================================"
