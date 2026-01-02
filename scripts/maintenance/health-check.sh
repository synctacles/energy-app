#!/bin/bash

echo "=== SYNCTACLES Health Check ==="
echo "Time: $(date)"
echo ""

ERRORS=0

# Check API service
echo "--- Services ---"
if systemctl is-active --quiet energy-insights-nl-api; then
    echo "✅ API service: running"
else
    echo "❌ API service: NOT running"
    ((ERRORS++))
fi

# Check timers
TIMER_COUNT=$(systemctl list-timers energy-insights-nl-* --no-pager 2>/dev/null | grep -c energy-insights || echo 0)
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
for endpoint in health "api/v1/generation-mix" "api/v1/load" "api/v1/signals"; do
    if curl -sf "http://localhost:8000/$endpoint" > /dev/null 2>&1; then
        echo "✅ /$endpoint"
    else
        echo "❌ /$endpoint"
        ((ERRORS++))
    fi
done

# Check database
echo ""
echo "--- Database ---"
if sudo -u postgres psql -d energy_insights_nl -c "SELECT 1" > /dev/null 2>&1; then
    echo "✅ Database connection OK"

    # Row counts
    for table in norm_entso_e_a75 norm_entso_e_a65 norm_entso_e_a44; do
        COUNT=$(sudo -u postgres psql -d energy_insights_nl -t -c "SELECT COUNT(*) FROM $table" 2>/dev/null | tr -d ' ')
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
