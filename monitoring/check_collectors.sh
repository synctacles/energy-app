#!/bin/bash
# Brand-free collector health check script

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
DB_NAME="${DB_NAME:-synctacles}"
DB_USER="${DB_USER:-synctacles}"
LOG_PATH="${LOG_PATH:-/var/log/${BRAND_SLUG}}"
LOG="${LOG_PATH}/collector-monitor.log"

check_service() {
    SERVICE=$1
    if ! systemctl is-active --quiet $SERVICE; then
        echo "[$(date)] ALERT: $SERVICE is not running" >> $LOG
        return 1
    fi
    return 0
}

# Check all timers
check_service "${BRAND_SLUG}-frank-collector.timer"
check_service "${BRAND_SLUG}-collector.timer"

# Check data freshness
sudo -u postgres psql "${DB_NAME}" -t -c "
SELECT
    CASE
        WHEN MAX(timestamp) < NOW() - INTERVAL '36 hours'
        THEN 'STALE'
        ELSE 'OK'
    END
FROM frank_prices;" | grep -q "STALE" && {
    echo "[$(date)] ALERT: Frank prices stale" >> $LOG
}

echo "[$(date)] Health check completed" >> $LOG
