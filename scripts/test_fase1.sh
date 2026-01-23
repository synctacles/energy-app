#!/bin/bash
# Fase 1 Test Script for Energy Action Focus
# Brand-free version - uses environment variables

set -e

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults
BRAND_NAME="${BRAND_NAME:-SYNCTACLES}"
BRAND_SLUG="${BRAND_SLUG:-synctacles}"
DB_NAME="${DB_NAME:-synctacles}"
DB_USER="${DB_USER:-synctacles}"
INSTALL_PATH="${INSTALL_PATH:-/opt/synctacles}"
APP_PATH="${APP_PATH:-/opt/github/synctacles-api}"
API_PORT="${API_PORT:-8000}"

echo "=== FASE 1 TEST SCRIPT ==="
echo "Date: $(date)"
echo "Brand: ${BRAND_SLUG}"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Step 1: Run migration
echo -e "${YELLOW}[1/5] Running database migration...${NC}"
cd "${APP_PATH}"
"${INSTALL_PATH}/venv/bin/alembic" upgrade head

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Migration successful${NC}"
else
    echo -e "${RED}Migration failed${NC}"
    exit 1
fi

# Step 2: Check table exists
echo -e "${YELLOW}[2/5] Checking price_cache table...${NC}"
TABLE_EXISTS=$(psql -U "${DB_USER}" -d "${DB_NAME}" -t -c "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'price_cache');")
if [[ $TABLE_EXISTS == *"t"* ]]; then
    echo -e "${GREEN}Table price_cache exists${NC}"
else
    echo -e "${RED}Table price_cache does NOT exist${NC}"
    exit 1
fi

# Step 3: Restart API
echo -e "${YELLOW}[3/5] Restarting API service...${NC}"
sudo systemctl restart "${BRAND_SLUG}-api"
sleep 3
if systemctl is-active --quiet "${BRAND_SLUG}-api"; then
    echo -e "${GREEN}API service running${NC}"
else
    echo -e "${RED}API service failed to start${NC}"
    journalctl -u "${BRAND_SLUG}-api" -n 20
    exit 1
fi

# Step 4: Test prices endpoint
echo -e "${YELLOW}[4/5] Testing /api/v1/prices endpoint...${NC}"
RESPONSE=$(curl -s "http://localhost:${API_PORT}/api/v1/prices")
echo "Response (truncated): $(echo "$RESPONSE" | head -c 200)..."

# Check required fields
if echo "$RESPONSE" | jq -e '.meta.source' > /dev/null 2>&1; then
    echo -e "${GREEN}source field present${NC}"
else
    echo -e "${RED}source field missing${NC}"
fi

if echo "$RESPONSE" | jq -e '.meta.quality_status' > /dev/null 2>&1; then
    echo -e "${GREEN}quality_status field present${NC}"
else
    echo -e "${RED}quality_status field missing${NC}"
fi

if echo "$RESPONSE" | jq -e '.data' > /dev/null 2>&1; then
    echo -e "${GREEN}data field present${NC}"
else
    echo -e "${RED}data field missing${NC}"
fi

# Step 5: Check logs for fallback info
echo -e "${YELLOW}[5/5] Checking API logs for fallback info...${NC}"
journalctl -u "${BRAND_SLUG}-api" -n 10 --no-pager | grep -E "(FRESH|STALE|FALLBACK|CACHED|Prices)" || echo "No fallback messages yet"

echo ""
echo "=== FASE 1 TEST COMPLETE ==="
echo ""
echo "Summary:"
echo "- price_cache table: created"
echo "- API service: running"
echo "- Source: $(echo "$RESPONSE" | jq -r '.meta.source // "N/A"')"
echo "- Quality: $(echo "$RESPONSE" | jq -r '.meta.quality_status // "N/A"')"
echo "- Count: $(echo "$RESPONSE" | jq -r '.meta.count // "N/A"') prices"
echo ""
echo -e "${GREEN}Ready for Fase 2 if all tests pass!${NC}"
