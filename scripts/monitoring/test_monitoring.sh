#!/bin/bash
# SYNCTACLES: Test monitoring infrastructure
#
# Validates that Prometheus, Grafana, and node_exporter are working correctly.
# Run on the monitoring server.

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ERRORS=0
WARNINGS=0

echo "════════════════════════════════════════════════════════════════"
echo "SYNCTACLES Monitoring Infrastructure Test"
echo "════════════════════════════════════════════════════════════════"
echo ""

# ============================================================================
# TEST 1: Docker containers running
# ============================================================================

echo "[1/8] Checking Docker containers..."

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker not installed${NC}"
    ((ERRORS++))
else
    RUNNING=0
    TOTAL=0

    for container in prometheus grafana alertmanager; do
        ((TOTAL++))
        if docker ps --format "table {{.Names}}" | grep -q "^${container}$"; then
            echo -e "  ${GREEN}✅${NC} $container: running"
            ((RUNNING++))
        else
            echo -e "  ${RED}❌${NC} $container: not running"
            ((ERRORS++))
        fi
    done

    if [[ $RUNNING -eq $TOTAL ]]; then
        echo -e "${GREEN}✅ All containers running${NC}"
    fi
fi

echo ""

# ============================================================================
# TEST 2: Prometheus API
# ============================================================================

echo "[2/8] Testing Prometheus API..."

if curl -sf http://localhost:9090/-/ready > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Prometheus API: responsive${NC}"

    # Check alert rules loaded
    RULES=$(curl -s http://localhost:9090/api/v1/rules 2>/dev/null | grep -o '"name":"' | wc -l)
    if [[ $RULES -gt 0 ]]; then
        echo -e "  ${GREEN}✅${NC} Alert rules: $RULES groups loaded"
    else
        echo -e "  ${YELLOW}⚠️${NC} No alert rules loaded"
        ((WARNINGS++))
    fi
else
    echo -e "${RED}❌ Prometheus API: unreachable${NC}"
    ((ERRORS++))
fi

echo ""

# ============================================================================
# TEST 3: Grafana API
# ============================================================================

echo "[3/8] Testing Grafana API..."

if curl -sf http://localhost:3000/api/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Grafana API: responsive${NC}"

    # Check datasource (Prometheus)
    DS=$(curl -s http://localhost:3000/api/datasources 2>/dev/null | grep -o '"name":"' | wc -l)
    if [[ $DS -gt 0 ]]; then
        echo -e "  ${GREEN}✅${NC} Datasources: $DS configured"
    else
        echo -e "  ${YELLOW}⚠️${NC} No datasources configured${NC}"
        ((WARNINGS++))
    fi
else
    echo -e "${RED}❌ Grafana API: unreachable${NC}"
    ((ERRORS++))
fi

echo ""

# ============================================================================
# TEST 4: node_exporter (localhost)
# ============================================================================

echo "[4/8] Testing node_exporter (monitoring server)..."

if curl -sf http://localhost:9100/metrics | grep -q "node_cpu_seconds_total"; then
    echo -e "${GREEN}✅ node_exporter: publishing metrics${NC}"
else
    echo -e "${RED}❌ node_exporter: not responding${NC}"
    ((ERRORS++))
fi

echo ""

# ============================================================================
# TEST 5: Prometheus targets
# ============================================================================

echo "[5/8] Checking Prometheus targets..."

if command -v jq &> /dev/null; then
    TARGET_COUNT=$(curl -s http://localhost:9090/api/v1/targets 2>/dev/null | jq '.data.activeTargets | length' 2>/dev/null || echo "0")

    if [[ "$TARGET_COUNT" -gt 0 ]]; then
        echo -e "${GREEN}✅ Prometheus targets: $TARGET_COUNT active${NC}"

        # Show targets
        curl -s http://localhost:9090/api/v1/targets 2>/dev/null | jq -r '.data.activeTargets[] | "  - \(.labels.instance) (\(.labels.job))"' 2>/dev/null || true
    else
        echo -e "${YELLOW}⚠️  No targets configured (yet)${NC}"
        echo "    Add with: ./scripts/monitoring/add_target.sh <ip> <name>"
        ((WARNINGS++))
    fi
else
    echo -e "${YELLOW}⚠️  jq not installed - skipping detailed target info${NC}"
    ((WARNINGS++))
fi

echo ""

# ============================================================================
# TEST 6: Firewall
# ============================================================================

echo "[6/8] Checking firewall..."

if command -v ufw &> /dev/null && ufw status | grep -q "Status: active"; then
    PORTS_OK=0

    if ufw status | grep -qE "9090.*ALLOW"; then
        echo -e "  ${GREEN}✅${NC} Port 9090 (Prometheus): allowed"
        ((PORTS_OK++))
    else
        echo -e "  ${YELLOW}⚠️${NC} Port 9090 (Prometheus): check firewall"
    fi

    if ufw status | grep -qE "3000.*ALLOW"; then
        echo -e "  ${GREEN}✅${NC} Port 3000 (Grafana): allowed"
        ((PORTS_OK++))
    else
        echo -e "  ${YELLOW}⚠️${NC} Port 3000 (Grafana): check firewall"
    fi

    if [[ $PORTS_OK -eq 2 ]]; then
        echo -e "${GREEN}✅ Firewall: monitoring ports open${NC}"
    else
        ((WARNINGS++))
    fi
else
    echo -e "${YELLOW}⚠️  UFW not enabled or not installed${NC}"
    ((WARNINGS++))
fi

echo ""

# ============================================================================
# TEST 7: Disk space
# ============================================================================

echo "[7/8] Checking disk space..."

DISK_USAGE=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
DISK_AVAILABLE=$(df -h / | awk 'NR==2 {print $4}')

if [[ "$DISK_USAGE" -lt 85 ]]; then
    echo -e "${GREEN}✅ Disk usage: ${DISK_USAGE}% (${DISK_AVAILABLE} available)${NC}"
else
    echo -e "${YELLOW}⚠️  Disk usage: ${DISK_USAGE}% (${DISK_AVAILABLE} available)${NC}"
    ((WARNINGS++))
fi

echo ""

# ============================================================================
# TEST 8: Docker stats
# ============================================================================

echo "[8/8] Checking resource usage..."

if docker ps > /dev/null 2>&1; then
    # Get memory usage of containers
    echo "  Container memory usage:"
    docker stats --no-stream --format "table {{.Container}}\t{{.MemUsage}}" 2>/dev/null | tail -n +2 | head -3

    # Check if any container using >1GB
    HIGH_MEM=$(docker stats --no-stream --format "{{.MemUsage}}" 2>/dev/null | grep -o "[0-9]*GiB" | sed 's/GiB//' || true)
    if [[ -n "$HIGH_MEM" && $(echo "$HIGH_MEM" | head -1) -gt 1 ]]; then
        echo -e "${YELLOW}⚠️  Some containers using > 1GB RAM${NC}"
        ((WARNINGS++))
    else
        echo -e "${GREEN}✅ Memory usage: normal${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Could not check Docker stats${NC}"
fi

echo ""

# ============================================================================
# SUMMARY
# ============================================================================

echo "════════════════════════════════════════════════════════════════"

if [[ $ERRORS -eq 0 && $WARNINGS -eq 0 ]]; then
    echo -e "${GREEN}✅ ALL TESTS PASSED${NC}"
    echo ""
    echo "Monitoring infrastructure is operational!"
    echo ""
    echo "Access URLs:"
    echo "  Prometheus: http://$(hostname -I | awk '{print $1}'):9090"
    echo "  Grafana:    http://$(hostname -I | awk '{print $1}'):3000"
    echo "  Alertmanager: http://$(hostname -I | awk '{print $1}'):9093"
    echo ""
    echo "Next steps:"
    echo "  1. Add application servers:"
    echo "     ./scripts/monitoring/add_target.sh <server_ip> <server_name>"
    echo ""
    echo "  2. Configure Grafana dashboards"
    echo ""
    echo "  3. Test alerts via Prometheus: http://localhost:9090/alerts"
    echo ""
    exit 0
elif [[ $ERRORS -eq 0 ]]; then
    echo -e "${YELLOW}⚠️  TESTS PASSED WITH $WARNINGS WARNING(S)${NC}"
    echo ""
    echo "Check the warnings above and resolve as needed."
    echo ""
    exit 0
else
    echo -e "${RED}❌ $ERRORS TEST(S) FAILED${NC}"
    echo ""
    echo "Troubleshooting:"
    echo "  - Check Docker: docker ps -a"
    echo "  - Check logs: docker-compose logs"
    echo "  - Restart: docker-compose restart"
    echo "  - Rebuild: docker-compose down && docker-compose up -d"
    echo ""
    exit 1
fi
