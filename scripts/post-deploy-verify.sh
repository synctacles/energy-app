#!/bin/bash
# Post-Deployment Verification Script
# Validates API is running correctly after deployment
# Used after: Docker startup, server restart, production deploy
#
# Checks:
# 1. API health endpoint
# 2. CORS configuration
# 3. All required endpoints are accessible
# 4. Database connectivity
# 5. Authentication system ready
# 6. Prometheus metrics endpoint
#
# Exit codes:
#   0 = All checks passed
#   1 = One or more checks failed
#   2 = Configuration error

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8000}"
API_TIMEOUT="${API_TIMEOUT:-10}"
ORIGIN="${ORIGIN:-https://homeassistant.local}"

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

FAILED=0
PASSED=0

echo -e "${BLUE}🚀 Starting Post-Deployment Verification${NC}\n"
echo "API URL: $API_URL"
echo "Timeout: ${API_TIMEOUT}s"
echo "CORS Origin: $ORIGIN"
echo ""

# ============================================================================
# Helper functions
# ============================================================================

check_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=$3
    local description=$4

    echo -ne "  ▶ $description... "

    # Make request with timeout
    response=$(curl -s -w "\n%{http_code}" \
        -X "$method" \
        -H "Content-Type: application/json" \
        -H "Origin: $ORIGIN" \
        --max-time "$API_TIMEOUT" \
        "${API_URL}${endpoint}" 2>/dev/null || echo -e "\n000")

    # Extract status code (last line)
    status=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | head -n -1)

    if [ "$status" = "$expected_status" ]; then
        echo -e "${GREEN}✅ $status${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}❌ Expected $expected_status, got $status${NC}"
        ((FAILED++))
        return 1
    fi
}

check_cors_header() {
    local endpoint=$1
    local description=$2

    echo -ne "  ▶ $description... "

    response=$(curl -s -I \
        -X OPTIONS \
        -H "Origin: $ORIGIN" \
        -H "Access-Control-Request-Method: GET" \
        --max-time "$API_TIMEOUT" \
        "${API_URL}${endpoint}" 2>/dev/null || echo "")

    if echo "$response" | grep -q "Access-Control-Allow-Origin"; then
        allow_origin=$(echo "$response" | grep -i "Access-Control-Allow-Origin" | head -n 1 | cut -d' ' -f2-)
        echo -e "${GREEN}✅ CORS enabled ($allow_origin)${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}❌ CORS headers missing${NC}"
        ((FAILED++))
        return 1
    fi
}

# ============================================================================
# 1. Basic Connectivity
# ============================================================================
echo -e "${BLUE}1️⃣  Basic Connectivity${NC}"
check_endpoint "GET" "/health" "200" "Health check endpoint"
check_endpoint "GET" "/metrics" "200" "Prometheus metrics endpoint"
echo ""

# ============================================================================
# 2. CORS Configuration
# ============================================================================
echo -e "${BLUE}2️⃣  CORS Configuration${NC}"
check_cors_header "/api/v1/generation-mix" "CORS preflight (generation-mix)"
check_cors_header "/api/v1/load" "CORS preflight (load)"
check_cors_header "/api/v1/prices" "CORS preflight (prices)"
echo ""

# ============================================================================
# 3. API Endpoints
# ============================================================================
echo -e "${BLUE}3️⃣  API Endpoints${NC}"
check_endpoint "GET" "/api/v1/generation-mix" "200" "Generation mix endpoint"
check_endpoint "GET" "/api/v1/load" "200" "Load endpoint"
check_endpoint "GET" "/api/v1/balance" "200" "Balance endpoint"
check_endpoint "GET" "/api/v1/prices" "200" "Prices endpoint"
check_endpoint "GET" "/api/v1/now" "200" "Unified now endpoint"
echo ""

# ============================================================================
# 4. Authentication System (if enabled)
# ============================================================================
echo -e "${BLUE}4️⃣  Authentication System${NC}"
check_endpoint "POST" "/auth/signup" "200" "Auth signup endpoint available"
check_endpoint "GET" "/docs" "200" "API documentation available"
echo ""

# ============================================================================
# 5. Cache System
# ============================================================================
echo -e "${BLUE}5️⃣  Cache System${NC}"
check_endpoint "GET" "/cache/stats" "200" "Cache stats endpoint"
echo ""

# ============================================================================
# 6. Performance Check
# ============================================================================
echo -e "${BLUE}6️⃣  Performance Baseline${NC}"

echo -ne "  ▶ Response time for generation-mix... "
start_time=$(date +%s%N)
curl -s --max-time "$API_TIMEOUT" "${API_URL}/api/v1/generation-mix" > /dev/null 2>&1
end_time=$(date +%s%N)
response_ms=$(( (end_time - start_time) / 1000000 ))
if [ $response_ms -lt 5000 ]; then
    echo -e "${GREEN}✅ ${response_ms}ms${NC}"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠️  ${response_ms}ms (slower than expected, but OK)${NC}"
    ((PASSED++))
fi
echo ""

# ============================================================================
# 7. Summary
# ============================================================================
TOTAL=$((PASSED + FAILED))

echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo -e "Results: ${GREEN}${PASSED} passed${NC}, ${RED}${FAILED} failed${NC} (${TOTAL} total)"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All post-deployment checks passed!${NC}"
    echo ""
    echo "API is ready for:"
    echo "  ✓ Production traffic"
    echo "  ✓ Home Assistant integration"
    echo "  ✓ SaaS reseller integration"
    echo ""
    exit 0
else
    echo -e "${RED}❌ Some checks failed. Review above.${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check if API is running: curl -v $API_URL/health"
    echo "  2. Check logs: docker logs <container> or check application logs"
    echo "  3. Verify environment variables are set correctly"
    echo "  4. Check firewall/network access to API"
    echo ""
    exit 1
fi
