#!/bin/bash
# SYNCTACLES: Add application server to Prometheus monitoring
#
# Usage: ./add_target.sh <server_ip> [server_name]
#
# This script adds a new application server to the Prometheus configuration
# so metrics are scraped every 15 seconds.
#
# Run this on the MONITORING SERVER.

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ============================================================================
# ARGUMENT VALIDATION
# ============================================================================

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <server_ip> [server_name]"
    echo ""
    echo "Examples:"
    echo "  $0 10.0.1.5 production-api"
    echo "  $0 192.168.1.100 staging-collector"
    echo "  $0 10.0.1.10"
    exit 1
fi

SERVER_IP="$1"
SERVER_NAME="${2:-server-$SERVER_IP}"

# Validate IP format
if [[ ! "$SERVER_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}❌ Invalid IP format: $SERVER_IP${NC}"
    echo "Expected: X.X.X.X (e.g., 10.0.1.5)"
    exit 1
fi

echo "════════════════════════════════════════════════════════════════"
echo "SYNCTACLES: Add Target to Prometheus"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "Server IP:   $SERVER_IP"
echo "Server name: $SERVER_NAME"
echo ""

# ============================================================================
# CHECK PROMETHEUS CONFIG
# ============================================================================

echo "[1/3] Checking Prometheus configuration..."

PROM_CONFIG="/opt/monitoring/prometheus.yml"

# Try Docker first
if docker ps 2>/dev/null | grep -q prometheus; then
    # Running in Docker - config is in volume or container
    PROM_CONFIG="/opt/monitoring/prometheus.yml"
    echo -e "${GREEN}✅ Prometheus running in Docker${NC}"
elif systemctl is-active --quiet prometheus; then
    # Running as systemd service
    PROM_CONFIG="/etc/prometheus/prometheus.yml"
    echo -e "${GREEN}✅ Prometheus running as systemd service${NC}"
else
    echo -e "${RED}❌ Prometheus not running${NC}"
    exit 1
fi

if [[ ! -f "$PROM_CONFIG" ]]; then
    echo -e "${RED}❌ Prometheus config not found: $PROM_CONFIG${NC}"
    exit 1
fi

echo ""

# ============================================================================
# CHECK IF ALREADY EXISTS
# ============================================================================

echo "[2/3] Checking if already configured..."

if grep -q "$SERVER_IP:9100" "$PROM_CONFIG"; then
    echo -e "${YELLOW}⚠️  Server $SERVER_IP already in Prometheus targets${NC}"
    exit 0
fi

echo -e "${GREEN}✅ Server not yet configured${NC}"
echo ""

# ============================================================================
# ADD TO PROMETHEUS CONFIG
# ============================================================================

echo "[3/3] Adding to Prometheus..."

# Backup original
cp "$PROM_CONFIG" "$PROM_CONFIG.backup.$(date +%Y%m%d-%H%M%S)"
echo "  Backup created: $PROM_CONFIG.backup.*"

# Add target
# Format: - targets: ['10.0.1.5:9100']  # server-name
sed -i "/# AUTO-GENERATED/a \          - targets: ['$SERVER_IP:9100']  # $SERVER_NAME" "$PROM_CONFIG"

echo "  Added: $SERVER_IP:9100 ($SERVER_NAME)"

# Reload Prometheus
if docker ps 2>/dev/null | grep -q prometheus; then
    docker-compose -f /opt/monitoring/docker-compose.yml exec prometheus kill -HUP 1 2>/dev/null || true
    sleep 2
    echo "  Reloading Prometheus configuration..."
elif systemctl is-active --quiet prometheus; then
    systemctl reload prometheus
    sleep 2
    echo "  Reloading Prometheus configuration..."
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo -e "${GREEN}✅ TARGET ADDED SUCCESSFULLY${NC}"
echo "════════════════════════════════════════════════════════════════"
echo ""

echo "Next steps:"
echo "  1. Wait 15-30 seconds for first scrape"
echo "  2. Check target health:"
echo "     http://<monitoring_server_ip>:9090/targets"
echo ""
echo "  3. Verify metrics in Grafana:"
echo "     http://<monitoring_server_ip>:3000"
echo ""
echo "  4. If metrics don't appear:"
echo "     - Check node_exporter on app server: curl http://$SERVER_IP:9100/metrics"
echo "     - Check firewall: sudo ufw status | grep 9100"
echo "     - Check Prometheus logs: docker-compose logs prometheus (or journalctl -u prometheus)"
echo ""

exit 0
