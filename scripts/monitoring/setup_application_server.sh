#!/bin/bash
# SYNCTACLES: Install node_exporter on application server
#
# Usage: sudo bash setup_application_server.sh <monitoring_server_ip>
#
# This script installs node_exporter (metrics collector) on an application server
# and configures firewall to allow only the monitoring server to scrape metrics.
#
# Run this on the APPLICATION SERVER, not the monitoring server.

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "════════════════════════════════════════════════════════════════"
echo "SYNCTACLES: Install node_exporter (Application Server)"
echo "════════════════════════════════════════════════════════════════"
echo ""

# ============================================================================
# PRE-FLIGHT CHECKS
# ============================================================================

echo "[1/5] Pre-flight checks..."

# Check root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}❌ ERROR: Must run as root${NC}"
    echo "Usage: sudo bash setup_application_server.sh <monitoring_server_ip>"
    exit 1
fi

# Check monitoring server IP provided
if [[ $# -lt 1 ]]; then
    echo -e "${RED}❌ ERROR: Monitoring server IP required${NC}"
    echo ""
    echo "Usage: sudo bash setup_application_server.sh <monitoring_server_ip>"
    echo ""
    echo "Example:"
    echo "  sudo bash setup_application_server.sh 10.0.1.100"
    echo "  sudo bash setup_application_server.sh 192.168.1.50"
    exit 1
fi

MONITORING_IP="$1"

# Validate IP format
if [[ ! "$MONITORING_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}❌ ERROR: Invalid IP format: $MONITORING_IP${NC}"
    echo ""
    echo "Expected format: X.X.X.X (e.g., 10.0.1.100)"
    exit 1
fi

# Check Ubuntu
if ! grep -q "Ubuntu" /etc/os-release; then
    echo -e "${RED}❌ ERROR: Ubuntu required${NC}"
    exit 1
fi

# Check internet
if ! ping -c 1 8.8.8.8 &>/dev/null; then
    echo -e "${RED}❌ ERROR: No internet connectivity${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Pre-flight checks passed${NC}"
echo "  Monitoring server IP: $MONITORING_IP"
echo ""

# ============================================================================
# INSTALL NODE_EXPORTER
# ============================================================================

echo "[2/5] Installing node_exporter..."

apt-get update > /dev/null 2>&1
apt-get install -y prometheus-node-exporter > /dev/null 2>&1

echo -e "${GREEN}✅ node_exporter installed${NC}"
echo ""

# ============================================================================
# START & ENABLE SERVICE
# ============================================================================

echo "[3/5] Starting node_exporter service..."

systemctl daemon-reload
systemctl enable prometheus-node-exporter > /dev/null 2>&1
systemctl start prometheus-node-exporter

# Wait for service to start
sleep 2

if systemctl is-active --quiet prometheus-node-exporter; then
    echo -e "${GREEN}✅ node_exporter service started${NC}"
else
    echo -e "${RED}❌ ERROR: node_exporter failed to start${NC}"
    journalctl -u prometheus-node-exporter -n 20
    exit 1
fi
echo ""

# ============================================================================
# CONFIGURE FIREWALL
# ============================================================================

echo "[4/5] Configuring firewall..."

# Check if UFW is enabled
if ufw status | grep -q "Status: active"; then
    # UFW is enabled - add rule for monitoring server
    ufw allow from "$MONITORING_IP" to any port 9100 comment "Prometheus scraping from monitoring server" > /dev/null 2>&1
    echo -e "${GREEN}✅ Firewall configured${NC}"
    echo "  Allowed: $MONITORING_IP:9100 (metrics scraping)"
else
    # UFW not enabled - suggest enabling
    echo -e "${YELLOW}⚠️  UFW not enabled${NC}"
    echo "  Recommended: sudo ufw enable"
    echo "  Then: sudo ufw allow from $MONITORING_IP to any port 9100"
fi
echo ""

# ============================================================================
# VALIDATION
# ============================================================================

echo "[5/5] Validation..."

# Check metrics are available
if curl -sf http://localhost:9100/metrics > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Metrics endpoint: http://localhost:9100/metrics${NC}"
else
    echo -e "${RED}❌ ERROR: Metrics endpoint not responding${NC}"
    exit 1
fi

# Test from monitoring server perspective (if we can reach it)
if timeout 2 bash -c "echo >/dev/tcp/$MONITORING_IP/9090" 2>/dev/null; then
    echo -e "${GREEN}✅ Network connectivity to monitoring server: OK${NC}"
else
    echo -e "${YELLOW}⚠️  Cannot reach monitoring server (may be network isolated)${NC}"
    echo "  If this is expected, ignore this warning"
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo -e "${GREEN}✅ NODE_EXPORTER INSTALLATION COMPLETE${NC}"
echo "════════════════════════════════════════════════════════════════"
echo ""

# Get server IP
SERVER_IP=$(hostname -I | awk '{print $1}')

echo "Summary:"
echo "  Server IP:        $SERVER_IP"
echo "  Metrics port:     9100"
echo "  Metrics endpoint: http://$SERVER_IP:9100/metrics"
echo "  Service status:   $(systemctl is-active prometheus-node-exporter)"
echo ""

echo "Next steps on MONITORING SERVER:"
echo "  1. Add this server to Prometheus:"
echo "     sudo ./scripts/monitoring/add_target.sh $SERVER_IP $(hostname)"
echo ""
echo "  2. Verify metrics appear (wait 15-30 seconds for first scrape):"
echo "     http://<monitoring_server_ip>:9090/targets"
echo ""
echo "  3. View in Grafana (wait 30 seconds):"
echo "     http://<monitoring_server_ip>:3000"
echo ""

exit 0
