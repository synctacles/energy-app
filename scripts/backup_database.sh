#!/usr/bin/env bash
# SYNCTACLES Database Backup

set -euo pipefail

BACKUP_DIR="/opt/synctacles/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/synctacles_$TIMESTAMP.sql.gz"

mkdir -p "$BACKUP_DIR"

# Dump database
pg_dump -U synctacles synctacles | gzip > "$BACKUP_FILE"

echo "[$(date)] Backup created: $BACKUP_FILE"

# Cleanup old backups (7 days retention)
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +7 -delete

echo "[$(date)] Old backups cleaned"
