#!/bin/bash
# Brand-free version - uses environment variables
set -euo pipefail

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
BRAND_NAME="${BRAND_NAME:-SYNCTACLES}"
INSTALL_PATH="${INSTALL_PATH:-/opt/${BRAND_SLUG}}"

echo "=== ${BRAND_NAME} Rollback ==="

BACKUP_BASE="${INSTALL_PATH}/backups"
APP_DIR="${INSTALL_PATH}/app"

# List available backups
echo "Available backups:"
ls -lt "$BACKUP_BASE" | head -10

# Select backup
if [[ -n "${1:-}" ]]; then
    BACKUP_DIR="$BACKUP_BASE/$1"
else
    LATEST=$(ls -t "$BACKUP_BASE" | head -1)
    BACKUP_DIR="$BACKUP_BASE/$LATEST"
    echo ""
    read -p "Rollback to $LATEST? [y/N] " -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi

if [[ ! -d "$BACKUP_DIR/app" ]]; then
    echo "❌ Backup not found: $BACKUP_DIR"
    exit 1
fi

# Restore
echo "Restoring from: $BACKUP_DIR"
rm -rf "$APP_DIR"
cp -r "$BACKUP_DIR/app" "$APP_DIR"

# Restart
systemctl restart "${BRAND_SLUG}-api"
sleep 3

# Verify
if curl -sf http://localhost:8000/health > /dev/null; then
    echo "✅ Rollback successful"
    curl -s http://localhost:8000/health | jq .
else
    echo "❌ Rollback FAILED - manual intervention required!"
    exit 1
fi
