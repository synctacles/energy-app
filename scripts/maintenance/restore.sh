#!/bin/bash
set -euo pipefail

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults (fallback if no .env)
DB_NAME="${DB_NAME:-synctacles}"
DB_USER="${DB_USER:-synctacles}"
INSTALL_PATH="${INSTALL_PATH:-/opt/synctacles}"
BRAND_NAME="${BRAND_NAME:-SYNCTACLES}"
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
API_PORT="${API_PORT:-8000}"

echo "=== ${BRAND_NAME} Database Restore ==="

BACKUP_DIR="${INSTALL_PATH}/backups/db"

# List backups
echo "Available backups:"
ls -lh "$BACKUP_DIR"/*.sql.gz 2>/dev/null | tail -10

if [[ -z "${1:-}" ]]; then
    echo ""
    echo "Usage: $0 <backup_file>"
    echo "Example: $0 ${DB_NAME}_20251230_120000.sql.gz"
    exit 1
fi

BACKUP_FILE="$BACKUP_DIR/$1"
if [[ ! -f "$BACKUP_FILE" ]]; then
    echo "❌ Backup not found: $BACKUP_FILE"
    exit 1
fi

echo ""
echo "⚠️  WARNING: This will DROP and recreate the database!"
read -p "Continue with restore from $1? [y/N] " -n 1 -r
echo
[[ ! $REPLY =~ ^[Yy]$ ]] && exit 1

# Stop API
systemctl stop "${BRAND_SLUG}-api"

# Restore
echo "Restoring..."
sudo -u postgres dropdb --if-exists "$DB_NAME"
sudo -u postgres createdb "$DB_NAME" -O "$DB_USER"
gunzip -c "$BACKUP_FILE" | sudo -u postgres psql "$DB_NAME"

# Start API
systemctl start "${BRAND_SLUG}-api"
sleep 3

# Verify
if curl -sf "http://localhost:${API_PORT}/health" > /dev/null; then
    echo "✅ Restore successful"
else
    echo "❌ Restore FAILED - check logs!"
    exit 1
fi
