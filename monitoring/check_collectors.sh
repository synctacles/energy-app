#!/bin/bash
LOG="/var/log/enin-collector-monitor.log"

check_service() {
    SERVICE=$1
    if ! systemctl is-active --quiet $SERVICE; then
        echo "[$(date)] ALERT: $SERVICE is not running" >> $LOG
        return 1
    fi
    return 0
}

# Check all timers
check_service enin-frank-collector.timer
check_service enin-enever-collector.timer
check_service energy-insights-nl-collector.timer

# Check data freshness
sudo -u postgres psql energy_insights_nl -t -c "
SELECT 
    CASE 
        WHEN MAX(timestamp) < NOW() - INTERVAL '36 hours' 
        THEN 'STALE' 
        ELSE 'OK' 
    END 
FROM frank_prices;" | grep -q "STALE" && {
    echo "[$(date)] ALERT: Frank prices stale on ENIN-NL" >> $LOG
}

echo "[$(date)] ENIN health check completed" >> $LOG
