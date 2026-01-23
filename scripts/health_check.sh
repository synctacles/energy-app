#!/usr/bin/env bash
# Brand-free version - uses environment variables
set -euo pipefail

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

BRAND_SLUG="${BRAND_SLUG:-synctacles}"
LOG_DIR="${LOG_PATH:-/var/log/${BRAND_SLUG}}/scheduler"
LOG_FILE="$LOG_DIR/health_$(date +%Y%m%d_%H%M%S).log"

mkdir -p "$LOG_DIR"

# Check API
if curl -sf http://localhost:8000/health > /dev/null; then
    echo "[$(date)] API OK" | tee -a "$LOG_FILE"
else
    echo "[$(date)] API FAILED" | tee -a "$LOG_FILE"
    exit 1
fi

# Check Database
if psql -U ${DB_USER:-synctacles} -d ${DB_NAME:-synctacles} -c "SELECT 1" > /dev/null 2>&1; then
    echo "[$(date)] DB OK" | tee -a "$LOG_FILE"
else
    echo "[$(date)] DB FAILED" | tee -a "$LOG_FILE"
    exit 1
fi

echo "[$(date)] Health check passed" | tee -a "$LOG_FILE"
