#!/bin/bash
set -euo pipefail

echo "=== SYNCTACLES Deploy ==="
echo "Started: $(date)"

# Variables
REPO_DIR="/opt/github/ha-energy-insights-nl"
APP_DIR="/opt/energy-insights-nl/app"
BACKUP_BASE="/opt/energy-insights-nl/backups"

# 1. Pre-checks
echo ""
echo "--- Pre-checks ---"
cd "$REPO_DIR"

if [[ -n $(git status --porcelain) ]]; then
    echo "⚠️  Uncommitted changes detected"
    git status --short
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi
echo "✅ Git status OK"

# 2. Backup current
echo ""
echo "--- Creating backup ---"
BACKUP_DIR="$BACKUP_BASE/deploy-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp -r "$APP_DIR" "$BACKUP_DIR/"
echo "✅ Backup created: $BACKUP_DIR"

# 3. Pull latest
echo ""
echo "--- Pulling latest ---"
git pull origin main
echo "✅ Git pulled"

# 4. Sync files
echo ""
echo "--- Syncing files ---"
rsync -av --delete \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    --exclude='.git' \
    "$REPO_DIR/synctacles_db/" "$APP_DIR/synctacles_db/"

rsync -av \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    "$REPO_DIR/config/" "$APP_DIR/config/"

rsync -av \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    "$REPO_DIR/alembic/" "$APP_DIR/alembic/"

echo "✅ Files synced"

# 5. Run migrations (if any)
echo ""
echo "--- Database migrations ---"
cd "$APP_DIR"
source /opt/energy-insights-nl/venv/bin/activate
export PYTHONPATH="$APP_DIR"
if alembic current 2>/dev/null; then
    alembic upgrade head
    echo "✅ Migrations complete"
else
    echo "⚠️  Alembic not configured, skipping migrations"
fi

# 6. Fix ownership (Claude Code runs as root, creates __pycache__ with root ownership)
echo ""
echo "--- Fixing file ownership ---"
chown -R energy-insights-nl:energy-insights-nl "$APP_DIR"
find "$APP_DIR" -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
echo "✅ Ownership fixed, pycache cleaned"

# 7. Restart services
echo ""
echo "--- Restarting services ---"
systemctl restart energy-insights-nl-api
sleep 3
echo "✅ API restarted"

# 8. Health check
echo ""
echo "--- Health check ---"
if curl -sf http://localhost:8000/health > /dev/null; then
    echo "✅ Health check passed"
    curl -s http://localhost:8000/health | jq .
else
    echo "❌ Health check FAILED!"
    echo "Rolling back..."
    cp -r "$BACKUP_DIR/app/"* "$APP_DIR/"
    systemctl restart energy-insights-nl-api
    exit 1
fi

echo ""
echo "=== Deploy Complete ==="
echo "Finished: $(date)"
