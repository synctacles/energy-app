#!/bin/bash
set -euo pipefail

echo "=== SYNCTACLES Database Restore ==="

BACKUP_DIR="/opt/energy-insights-nl/backups/db"
DB_NAME="energy_insights_nl"

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
systemctl stop energy-insights-nl-api

# Restore
echo "Restoring..."
sudo -u postgres dropdb --if-exists "$DB_NAME"
sudo -u postgres createdb "$DB_NAME" -O energy_insights_nl
gunzip -c "$BACKUP_FILE" | sudo -u postgres psql "$DB_NAME"

# Start API
systemctl start energy-insights-nl-api
sleep 3

# Verify
if curl -sf http://localhost:8000/health > /dev/null; then
    echo "✅ Restore successful"
else
    echo "❌ Restore FAILED - check logs!"
    exit 1
fi
