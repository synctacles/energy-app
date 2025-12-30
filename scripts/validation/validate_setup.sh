#!/bin/bash
set -e

echo "========================================"
echo "  SYNCTACLES Setup Validation"
echo "  $(date)"
echo "========================================"
echo ""

ERRORS=0
WARNINGS=0

check_pass() { echo "✅ $1"; }
check_fail() { echo "❌ $1"; ((ERRORS++)); }
check_warn() { echo "⚠️  $1"; ((WARNINGS++)); }

# 1. Environment
echo "--- Environment ---"
[[ -f /opt/.env ]] && check_pass ".env exists" || check_fail ".env missing"

source /opt/.env 2>/dev/null || true
[[ -n "${BRAND_NAME:-}" ]] && check_pass "BRAND_NAME set: $BRAND_NAME" || check_fail "BRAND_NAME not set"
[[ -n "${DATABASE_URL:-}" ]] && check_pass "DATABASE_URL set" || check_fail "DATABASE_URL not set"

# 2. Services
echo ""
echo "--- Services ---"
systemctl is-active --quiet energy-insights-nl-api && check_pass "API service running" || check_fail "API service not running"
systemctl is-active --quiet postgresql && check_pass "PostgreSQL running" || check_fail "PostgreSQL not running"
systemctl is-active --quiet nginx && check_pass "Nginx running" || check_warn "Nginx not running"

# 3. Timers
echo ""
echo "--- Scheduled Tasks ---"
for timer in collector importer normalizer; do
    if systemctl list-timers | grep -q "energy-insights-nl-${timer}"; then
        check_pass "Timer: $timer"
    else
        check_warn "Timer missing: $timer"
    fi
done

# 4. API Endpoints
echo ""
echo "--- API Endpoints ---"
API_BASE="http://localhost:8000"

for endpoint in /health /api/v1/generation-mix /api/v1/load /api/v1/balance /api/v1/prices /api/v1/signals; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE$endpoint" 2>/dev/null || echo "000")
    if [[ "$STATUS" == "200" ]]; then
        check_pass "$endpoint (HTTP $STATUS)"
    else
        check_fail "$endpoint (HTTP $STATUS)"
    fi
done

# 5. Database
echo ""
echo "--- Database ---"
if sudo -u postgres psql -d energy_insights_nl -c "SELECT 1" > /dev/null 2>&1; then
    check_pass "Database connection"

    # Check tables
    for table in norm_entso_e_a75 norm_entso_e_a65 norm_entso_e_a44 norm_tennet_balance; do
        if sudo -u postgres psql -d energy_insights_nl -c "SELECT 1 FROM $table LIMIT 1" > /dev/null 2>&1; then
            check_pass "Table exists: $table"
        else
            check_warn "Table missing or empty: $table"
        fi
    done
else
    check_fail "Database connection failed"
fi

# 6. File System
echo ""
echo "--- File System ---"
[[ -d /opt/energy-insights-nl/app ]] && check_pass "App directory exists" || check_fail "App directory missing"
[[ -d /opt/energy-insights-nl/venv ]] && check_pass "Venv exists" || check_fail "Venv missing"
[[ -d /opt/github/ha-energy-insights-nl ]] && check_pass "Git repo exists" || check_fail "Git repo missing"

# 7. Permissions
echo ""
echo "--- Permissions ---"
if [[ $(stat -c %U /opt/energy-insights-nl/app) == "energy-insights-nl" ]]; then
    check_pass "App owned by service user"
else
    check_warn "App ownership incorrect"
fi

# Summary
echo ""
echo "========================================"
echo "  Validation Complete"
echo "========================================"
echo "  Errors:   $ERRORS"
echo "  Warnings: $WARNINGS"
echo ""

if [[ $ERRORS -eq 0 ]]; then
    echo "✅ Setup is VALID"
    exit 0
else
    echo "❌ Setup has ERRORS - fix before production"
    exit 1
fi
