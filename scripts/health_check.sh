#!/usr/bin/env bash
set -euo pipefail

LOG_DIR="${LOG_PATH:-/var/log/energy-insights-nl}/scheduler"
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
