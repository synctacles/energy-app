#!/usr/bin/env bash
# Delete raw XML/JSON files older than 24 hours

set -euo pipefail

LOGS_DIR="/opt/synctacles/logs"

echo "[$(date)] Cleaning up old log files..."

# ENTSO-E raw files
find "$LOGS_DIR/entso_e_raw" -type f -name "*.xml" -mtime +1 -delete 2>/dev/null || true

# TenneT raw files
find "$LOGS_DIR/tennet_raw" -type f -name "*.json" -mtime +1 -delete 2>/dev/null || true

echo "[$(date)] Cleanup complete"
