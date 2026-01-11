#!/bin/bash
# Fase 1 Test Script for Energy Action Focus
# Run this on the production server (135.181.255.83)

set -e

echo "=== FASE 1 TEST SCRIPT ==="
echo "Date: $(date)"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Step 1: Run migration
echo -e "${YELLOW}[1/5] Running database migration...${NC}"
cd /opt/github/synctacles-api
source /opt/energy-insights-nl/.env
/opt/energy-insights-nl/venv/bin/alembic upgrade head

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Migration successful${NC}"
else
    echo -e "${RED}Migration failed${NC}"
    exit 1
fi

# Step 2: Check table exists
echo -e "${YELLOW}[2/5] Checking price_cache table...${NC}"
TABLE_EXISTS=$(psql -U energy-insights-nl -d energy_insights_nl -t -c "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'price_cache');")
if [[ $TABLE_EXISTS == *"t"* ]]; then
    echo -e "${GREEN}Table price_cache exists${NC}"
else
    echo -e "${RED}Table price_cache does NOT exist${NC}"
    exit 1
fi

# Step 3: Restart API
echo -e "${YELLOW}[3/5] Restarting API service...${NC}"
sudo systemctl restart synctacles-api
sleep 3
if systemctl is-active --quiet synctacles-api; then
    echo -e "${GREEN}API service running${NC}"
else
    echo -e "${RED}API service failed to start${NC}"
    journalctl -u synctacles-api -n 20
    exit 1
fi

# Step 4: Test energy-action endpoint
echo -e "${YELLOW}[4/5] Testing /api/v1/energy-action endpoint...${NC}"
RESPONSE=$(curl -s http://localhost:8000/api/v1/energy-action)
echo "Response: $RESPONSE"

# Check required fields
if echo "$RESPONSE" | jq -e '.action' > /dev/null 2>&1; then
    echo -e "${GREEN}action field present${NC}"
else
    echo -e "${RED}action field missing${NC}"
fi

if echo "$RESPONSE" | jq -e '.quality' > /dev/null 2>&1; then
    echo -e "${GREEN}quality field present${NC}"
else
    echo -e "${RED}quality field missing${NC}"
fi

if echo "$RESPONSE" | jq -e '.confidence' > /dev/null 2>&1; then
    echo -e "${GREEN}confidence field present${NC}"
else
    echo -e "${RED}confidence field missing${NC}"
fi

if echo "$RESPONSE" | jq -e '.source' > /dev/null 2>&1; then
    echo -e "${GREEN}source field present${NC}"
else
    echo -e "${RED}source field missing${NC}"
fi

# Step 5: Check logs for fallback info
echo -e "${YELLOW}[5/5] Checking API logs for fallback info...${NC}"
journalctl -u synctacles-api -n 10 --no-pager | grep -E "(FRESH|STALE|FALLBACK|CACHED|Prices)" || echo "No fallback messages yet"

echo ""
echo "=== FASE 1 TEST COMPLETE ==="
echo ""
echo "Summary:"
echo "- price_cache table: created"
echo "- API service: running"
echo "- /api/v1/energy-action: $(echo $RESPONSE | jq -r '.action // "N/A"')"
echo "- Quality: $(echo $RESPONSE | jq -r '.quality // "N/A"')"
echo "- Confidence: $(echo $RESPONSE | jq -r '.confidence // "N/A"')%"
echo "- Source: $(echo $RESPONSE | jq -r '.source // "N/A"')"
echo ""
echo -e "${GREEN}Ready for Fase 2 if all tests pass!${NC}"
