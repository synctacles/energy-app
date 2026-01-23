#!/bin/bash
set -euo pipefail

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults (fallback if no .env)
DB_NAME="${DB_NAME:-synctacles}"
INSTALL_PATH="${INSTALL_PATH:-/opt/synctacles}"
BRAND_NAME="${BRAND_NAME:-SYNCTACLES}"

echo "=== ${BRAND_NAME} Database Backup ==="

BACKUP_DIR="${INSTALL_PATH}/backups/db"
RETENTION_DAYS=30

mkdir -p "$BACKUP_DIR"

# Create backup
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_$(date +%Y%m%d_%H%M%S).sql.gz"
sudo -u postgres pg_dump "$DB_NAME" | gzip > "$BACKUP_FILE"

echo "✅ Backup created: $BACKUP_FILE"
echo "   Size: $(du -h "$BACKUP_FILE" | cut -f1)"

# Cleanup old backups
echo ""
echo "Cleaning backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete -print

echo ""
echo "Current backups:"
ls -lh "$BACKUP_DIR"/*.sql.gz 2>/dev/null | tail -5

echo ""
echo "=== Backup Complete ==="
