#!/bin/bash
# Brand-free health check script - uses environment variables

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults (fallback if no .env)
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
DB_NAME="${DB_NAME:-synctacles}"
DB_USER="${DB_USER:-synctacles}"
API_PORT="${API_PORT:-8000}"

echo "=== ${BRAND_NAME:-SYNCTACLES} Health Check ==="
echo "Time: $(date)"
echo "Brand: ${BRAND_SLUG}"
echo ""

ERRORS=0

# Check API service
echo "--- Services ---"
if systemctl is-active --quiet "${BRAND_SLUG}-api"; then
    echo "✅ API service: running"
else
    echo "❌ API service: NOT running"
    ((ERRORS++))
fi

# Check timers
TIMER_COUNT=$(systemctl list-timers "${BRAND_SLUG}-*" --no-pager 2>/dev/null | grep -c "${BRAND_SLUG}" || echo 0)
if [[ $TIMER_COUNT -ge 3 ]]; then
    echo "✅ Timers active: $TIMER_COUNT"
else
    echo "⚠️  Timers active: $TIMER_COUNT (expected >= 3)"
fi

# Check nginx
if systemctl is-active --quiet nginx; then
    echo "✅ Nginx: running"
else
    echo "⚠️  Nginx: NOT running"
fi

# Check PostgreSQL
if systemctl is-active --quiet postgresql; then
    echo "✅ PostgreSQL: running"
else
    echo "❌ PostgreSQL: NOT running"
    ((ERRORS++))
fi

# Check API endpoints
echo ""
echo "--- API Endpoints ---"
for endpoint in health "api/v1/prices"; do
    if curl -sf "http://localhost:${API_PORT}/$endpoint" > /dev/null 2>&1; then
        echo "✅ /$endpoint"
    else
        echo "❌ /$endpoint"
        ((ERRORS++))
    fi
done

# Check database
echo ""
echo "--- Database ---"
if sudo -u postgres psql -d "${DB_NAME}" -c "SELECT 1" > /dev/null 2>&1; then
    echo "✅ Database connection OK"

    # Row counts
    for table in frank_prices norm_entso_e_a44 price_cache; do
        COUNT=$(sudo -u postgres psql -d "${DB_NAME}" -t -c "SELECT COUNT(*) FROM $table" 2>/dev/null | tr -d ' ')
        echo "   $table: $COUNT rows"
    done
else
    echo "❌ Database connection FAILED"
    ((ERRORS++))
fi

# Check disk space
echo ""
echo "--- Disk Space ---"
DISK_USAGE=$(df -h /opt | tail -1 | awk '{print $5}' | tr -d '%')
if [[ $DISK_USAGE -lt 80 ]]; then
    echo "✅ Disk usage: ${DISK_USAGE}%"
else
    echo "⚠️  Disk usage: ${DISK_USAGE}% (warning: >80%)"
fi

# Summary
echo ""
echo "=== Summary ==="
if [[ $ERRORS -eq 0 ]]; then
    echo "✅ All critical checks passed"
    exit 0
else
    echo "❌ $ERRORS critical error(s) found"
    exit 1
fi
