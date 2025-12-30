#!/bin/bash
set -euo pipefail

echo "=== SYNCTACLES Database Backup ==="

BACKUP_DIR="/opt/energy-insights-nl/backups/db"
DB_NAME="energy_insights_nl"
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
