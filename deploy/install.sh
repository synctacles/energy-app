#!/bin/bash
# Energy API Go - Deployment Script
# Usage: sudo ./deploy/install.sh

set -e

REPO_DIR="/opt/github/energy-go"
BINARY_NAME="energy-api"
SERVICE_NAME="energy-api.service"
LOG_DIR="/var/log/energy-api"
USER="synctacles-dev"

echo "=== Energy API Go Deployment ==="
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: Please run as root (sudo)"
  exit 1
fi

# Check if we're in the right directory
if [ ! -f "$REPO_DIR/cmd/energy-api/main.go" ]; then
  echo "Error: $REPO_DIR/cmd/energy-api/main.go not found"
  exit 1
fi

# Step 1: Build binary
echo "1. Building binary..."
cd "$REPO_DIR"
sudo -u $USER /usr/local/go/bin/go build -o $BINARY_NAME ./cmd/energy-api/
chmod +x $BINARY_NAME
echo "   ✓ Binary built: $REPO_DIR/$BINARY_NAME"
echo

# Step 2: Create log directory
echo "2. Creating log directory..."
mkdir -p $LOG_DIR
chown $USER:$USER $LOG_DIR
chmod 755 $LOG_DIR
echo "   ✓ Log directory created: $LOG_DIR"
echo

# Step 3: Install systemd service
echo "3. Installing systemd service..."
cp deploy/systemd/$SERVICE_NAME /etc/systemd/system/
systemctl daemon-reload
echo "   ✓ Service installed: /etc/systemd/system/$SERVICE_NAME"
echo

# Step 4: Enable and start service
echo "4. Starting service..."
systemctl enable $SERVICE_NAME
systemctl restart $SERVICE_NAME
sleep 2
echo

# Step 5: Check status
echo "5. Checking service status..."
systemctl status $SERVICE_NAME --no-pager -l
echo

# Step 6: Test health endpoint
echo "6. Testing health endpoint..."
sleep 1
curl -s http://localhost:8002/health | jq . || echo "Health check failed (jq not installed?)"
echo

echo "=== Deployment Complete ==="
echo
echo "Service: systemctl status $SERVICE_NAME"
echo "Logs:    journalctl -u $SERVICE_NAME -f"
echo "         tail -f $LOG_DIR/energy-api.log"
echo "Health:  curl http://localhost:8002/health"
echo
