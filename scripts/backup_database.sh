#!/usr/bin/env bash
# Brand-free Database Backup
set -euo pipefail

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
DB_NAME="${DB_NAME:-${BRAND_SLUG}}"
DB_USER="${DB_USER:-${BRAND_SLUG}}"
INSTALL_PATH="${INSTALL_PATH:-/opt/${BRAND_SLUG}}"

BACKUP_DIR="${INSTALL_PATH}/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_$TIMESTAMP.sql.gz"

mkdir -p "$BACKUP_DIR"

# Dump database
pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$BACKUP_FILE"

echo "[$(date)] Backup created: $BACKUP_FILE"

# Cleanup old backups (7 days retention)
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +7 -delete

echo "[$(date)] Old backups cleaned"
