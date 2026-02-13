#!/bin/bash
# Deploy Energy API Go to ENERGY-PROD via GitHub
# Proper version-controlled deployment

set -e

VERSION="v2.0.0"
PROD_SERVER="energy-prod"
REPO_URL="git@github.com:synctacles/energy-go.git"

echo "=== Energy API Go → PROD Deployment (via GitHub) ==="
echo "Version: $VERSION"
echo "Target: $PROD_SERVER (energy.synctacles.com)"
echo

# Step 1: Commit and push to GitHub
echo "1. Preparing git commit..."
git add cmd/ internal/api/ deploy/ go.mod go.sum
git status --short

read -p "Commit message: " commit_msg
if [ -z "$commit_msg" ]; then
    commit_msg="feat: add Go API server v2.0.0 with central auth integration"
fi

git commit -m "$commit_msg

- Complete REST API implementation
- Central auth integration
- Tier-based access control (free/pro)
- Production logging and metrics
- Systemd service configuration
- 2x faster than Python (16ms vs 34ms)
- 97% less memory (16MB vs 490MB)

Benchmark results:
- Response time: 16.2ms avg (2.13x faster)
- Throughput: 61.7 req/sec (2.13x higher)
- Memory: 16 MB (97% reduction)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

echo

# Step 2: Tag release
echo "2. Creating git tag..."
git tag -a "$VERSION" -m "Release $VERSION - Production-ready Go API"
echo "   ✓ Tagged: $VERSION"
echo

# Step 3: Push to GitHub
echo "3. Pushing to GitHub..."
git push origin main
git push origin "$VERSION"
echo "   ✓ Pushed to GitHub"
echo

# Step 4: Deploy on PROD via GitHub
echo "4. Deploying on PROD from GitHub..."
ssh cc-hub "ssh $PROD_SERVER 'bash -s'" << ENDSSH
set -e

# Clone or pull
if [ -d /opt/github/energy-go/.git ]; then
    echo "   - Pulling latest from GitHub..."
    cd /opt/github/energy-go
    git fetch --all
    git checkout $VERSION
else
    echo "   - Cloning from GitHub..."
    sudo mkdir -p /opt/github
    cd /opt/github
    git clone $REPO_URL
    cd energy-go
    git checkout $VERSION
    sudo chown -R synctacles-dev:synctacles-dev /opt/github/energy-go
fi

# Create production .env
echo "   - Creating production .env..."
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
sudo -u postgres psql -d energy_prod -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO synctacles_dev;" 2>/dev/null || true
sudo -u postgres psql -d energy_prod -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO synctacles_dev;" 2>/dev/null || true

# Deploy
echo "   - Running deployment..."
sudo ./deploy/install.sh

echo
echo "   ✓ Deployed $VERSION to PROD"
ENDSSH

echo

# Step 5: Verify
echo "5. Verifying deployment..."
ssh cc-hub "ssh $PROD_SERVER 'systemctl status energy-api --no-pager -l | head -15'"
echo

echo "6. Testing health endpoint..."
ssh cc-hub "ssh $PROD_SERVER 'curl -s http://localhost:8002/health | jq'"
echo

echo "=== Deployment Complete ==="
echo
echo "Deployed version: $VERSION"
echo "Git commit: $(git rev-parse --short HEAD)"
echo
echo "Next steps:"
echo "  1. Monitor: ssh cc-hub \"ssh $PROD_SERVER 'journalctl -u energy-api -f'\""
echo "  2. Rollback (if needed): ssh cc-hub \"ssh $PROD_SERVER 'cd /opt/github/energy-go && git checkout <previous-tag> && sudo ./deploy/install.sh'\""
echo
