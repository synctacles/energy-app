#!/bin/bash
# Deploy Energy API Go to ENERGY-PROD
# Run from DEV server: ./deploy/deploy-to-prod.sh

set -e

PROD_SERVER="energy-prod"
PROD_DIR="/opt/github/energy-go"
PROD_DB="energy_prod"

echo "=== Energy API Go → PROD Deployment ==="
echo "Target: $PROD_SERVER (energy.synctacles.com)"
echo

# Step 1: Create deployment package
echo "1. Creating deployment package..."
cd /opt/github/energy-go
tar czf /tmp/energy-go-prod.tar.gz \
  cmd/ \
  internal/ \
  deploy/ \
  go.mod \
  go.sum \
  --exclude='*.log' \
  --exclude='.git' \
  --exclude='energy-api'

echo "   ✓ Package created: $(du -h /tmp/energy-go-prod.tar.gz | cut -f1)"
echo

# Step 2: Transfer to PROD via cc-hub
echo "2. Transferring to PROD..."
scp /tmp/energy-go-prod.tar.gz cc-hub:/tmp/
ssh cc-hub "scp /tmp/energy-go-prod.tar.gz $PROD_SERVER:/tmp/"
echo "   ✓ Files transferred"
echo

# Step 3: Setup and deploy on PROD
echo "3. Deploying on PROD..."
ssh cc-hub "ssh $PROD_SERVER 'bash -s'" << 'ENDSSH'
set -e

# Create directory
echo "   - Creating directory..."
sudo mkdir -p /opt/github/energy-go
sudo chown synctacles-dev:synctacles-dev /opt/github/energy-go
cd /opt/github/energy-go

# Extract
echo "   - Extracting files..."
tar xzf /tmp/energy-go-prod.tar.gz
rm /tmp/energy-go-prod.tar.gz

# Create production .env
echo "   - Creating .env..."
cat > .env << 'EOF'
PORT=8002
DATABASE_URL=postgres://synctacles_dev@localhost:5432/energy_prod?sslmode=disable
AUTH_SERVICE_URL=http://localhost:8000
LOG_LEVEL=info
LOG_FILE=/var/log/energy-api/energy-api.log
EOF
chmod 600 .env

# Grant database permissions
echo "   - Granting database permissions..."
sudo -u postgres psql -d energy_prod << 'EOSQL'
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO synctacles_dev;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO synctacles_dev;
EOSQL

# Run deployment
echo "   - Running install.sh..."
sudo ./deploy/install.sh

echo
echo "   ✓ Deployment complete on PROD"
ENDSSH

echo

# Step 4: Verify deployment
echo "4. Verifying deployment..."
ssh cc-hub "ssh $PROD_SERVER 'systemctl status energy-api --no-pager | head -20'"
echo

echo "5. Testing health endpoint..."
ssh cc-hub "ssh $PROD_SERVER 'curl -s http://localhost:8002/health | jq'"
echo

# Cleanup
rm /tmp/energy-go-prod.tar.gz

echo "=== Deployment Complete ==="
echo
echo "Next steps:"
echo "  1. Monitor logs: ssh cc-hub \"ssh $PROD_SERVER 'journalctl -u energy-api -f'\""
echo "  2. Test endpoint: curl http://energy.synctacles.com:8002/health"
echo "  3. Check metrics: ssh cc-hub \"ssh $PROD_SERVER 'curl http://localhost:8002/metrics'\""
echo
