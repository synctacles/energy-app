#!/usr/bin/env bash
# Brand-free log cleanup script
set -euo pipefail

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
INSTALL_PATH="${INSTALL_PATH:-/opt/${BRAND_SLUG}}"
LOGS_DIR="${LOG_PATH:-${INSTALL_PATH}/logs}"

echo "[$(date)] Cleaning up old log files..."

# ENTSO-E raw files
find "$LOGS_DIR/entso_e_raw" -type f -name "*.xml" -mtime +1 -delete 2>/dev/null || true

# TenneT raw files
find "$LOGS_DIR/tennet_raw" -type f -name "*.json" -mtime +1 -delete 2>/dev/null || true

echo "[$(date)] Cleanup complete"
