#!/bin/bash
# CC Session Start Verification Script
# Run at beginning of each CC session

echo "=== CC SESSION START VERIFICATION ==="
echo ""

# 1. GitHub CLI
echo "1. GitHub CLI Authentication:"
if sudo -u energy-insights-nl gh auth status 2>&1 | grep -q "Logged in"; then
    echo "   ✅ gh auth OK"
else
    echo "   ❌ gh auth FAILED - Run: sudo -u energy-insights-nl gh auth login"
fi
echo ""

# 2. Git status
echo "2. Git Repository Status:"
cd /opt/github/synctacles-api
if sudo -u energy-insights-nl git status --porcelain | grep -q .; then
    echo "   ⚠️ Uncommitted changes present"
    sudo -u energy-insights-nl git status --short
else
    echo "   ✅ Working directory clean"
fi
echo ""

# 3. Services
echo "3. Critical Services:"
systemctl is-active --quiet energy-insights-nl-api && echo "   ✅ API running" || echo "   ❌ API not running"
echo ""

# 4. Server reminder
echo "4. Infrastructure Reminder:"
echo "   - API Server: ENIN-NL (this server)"
echo "   - Monitoring: monitor.synctacles.com (NOT this server)"
echo ""

echo "=== VERIFICATION COMPLETE ==="
