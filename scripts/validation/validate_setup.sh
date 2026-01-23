#!/bin/bash
set -e

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults (fallback if no .env)
BRAND_NAME="${BRAND_NAME:-SYNCTACLES}"
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
DB_NAME="${DB_NAME:-synctacles}"
SERVICE_USER="${SERVICE_USER:-synctacles}"
INSTALL_PATH="${INSTALL_PATH:-/opt/synctacles}"
APP_PATH="${APP_PATH:-/opt/github/synctacles-api}"
API_PORT="${API_PORT:-8000}"

echo "========================================"
echo "  ${BRAND_NAME} Setup Validation"
echo "  $(date)"
echo "  Brand: ${BRAND_SLUG}"
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
[[ -n "${BRAND_NAME:-}" ]] && check_pass "BRAND_NAME set: $BRAND_NAME" || check_fail "BRAND_NAME not set"
[[ -n "${DATABASE_URL:-}" ]] && check_pass "DATABASE_URL set" || check_fail "DATABASE_URL not set"

# 2. Services
echo ""
echo "--- Services ---"
systemctl is-active --quiet "${BRAND_SLUG}-api" && check_pass "API service running" || check_fail "API service not running"
systemctl is-active --quiet postgresql && check_pass "PostgreSQL running" || check_fail "PostgreSQL not running"
systemctl is-active --quiet nginx && check_pass "Nginx running" || check_warn "Nginx not running"

# 3. Timers
echo ""
echo "--- Scheduled Tasks ---"
for timer in collector importer normalizer health frank-collector; do
    if systemctl list-timers | grep -q "${BRAND_SLUG}-${timer}"; then
        check_pass "Timer: $timer"
    else
        check_warn "Timer missing: $timer"
    fi
done

# 4. API Endpoints
echo ""
echo "--- API Endpoints ---"
API_BASE="http://localhost:${API_PORT}"

for endpoint in /health /api/v1/prices; do
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
if sudo -u postgres psql -d "${DB_NAME}" -c "SELECT 1" > /dev/null 2>&1; then
    check_pass "Database connection"

    # Check tables
    for table in frank_prices norm_entso_e_a44 price_cache; do
        if sudo -u postgres psql -d "${DB_NAME}" -c "SELECT 1 FROM $table LIMIT 1" > /dev/null 2>&1; then
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
[[ -d "${APP_PATH}" ]] && check_pass "App directory exists" || check_fail "App directory missing"
[[ -d "${INSTALL_PATH}/venv" ]] && check_pass "Venv exists" || check_fail "Venv missing"

# 7. Permissions
echo ""
echo "--- Permissions ---"
if [[ -d "${APP_PATH}" ]]; then
    OWNER=$(stat -c %U "${APP_PATH}" 2>/dev/null || echo "unknown")
    if [[ "$OWNER" == "${SERVICE_USER}" ]]; then
        check_pass "App owned by service user (${SERVICE_USER})"
    else
        check_warn "App ownership incorrect (owner: ${OWNER}, expected: ${SERVICE_USER})"
    fi
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
