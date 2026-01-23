#!/bin/bash
# CC Session Start Verification Script
# Run at beginning of each CC session
# Brand-free version - uses environment variables

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
SERVICE_USER="${SERVICE_USER:-synctacles}"
APP_PATH="${APP_PATH:-/opt/github/synctacles-api}"

echo "=== CC SESSION START VERIFICATION ==="
echo ""

# 1. GitHub CLI
echo "1. GitHub CLI Authentication:"
if sudo -u "${SERVICE_USER}" gh auth status 2>&1 | grep -q "Logged in"; then
    echo "   ✅ gh auth OK"
else
    echo "   ❌ gh auth FAILED - Run: sudo -u ${SERVICE_USER} gh auth login"
fi
echo ""

# 2. Git status
echo "2. Git Repository Status:"
cd "${APP_PATH}"
if sudo -u "${SERVICE_USER}" git status --porcelain | grep -q .; then
    echo "   ⚠️ Uncommitted changes present"
    sudo -u "${SERVICE_USER}" git status --short
else
    echo "   ✅ Working directory clean"
fi
echo ""

# 3. Services
echo "3. Critical Services:"
systemctl is-active --quiet "${BRAND_SLUG}-api" && echo "   ✅ API running" || echo "   ❌ API not running"
echo ""

# 4. Server reminder
echo "4. Infrastructure Reminder:"
echo "   - Brand: ${BRAND_SLUG}"
echo "   - Service User: ${SERVICE_USER}"
echo ""

echo "=== VERIFICATION COMPLETE ==="
