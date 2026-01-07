#!/bin/bash
# Fix __pycache__ ownership issues
# Issue #31: Python __pycache__ owned by root blocks service account
#
# Problem: Claude Code runs as root, creating __pycache__ with root:root ownership.
# This prevents the energy-insights-nl service account from writing new .pyc files.
#
# Solution: Remove all __pycache__ directories and fix ownership.
# Python will recreate them with correct ownership when services restart.

set -euo pipefail

APP_DIR="/opt/energy-insights-nl/app"
REPO_DIR="/opt/github/synctacles-api"
SERVICE_USER="energy-insights-nl"
SERVICE_GROUP="energy-insights-nl"

echo "=== Fix __pycache__ Ownership ==="
echo "Date: $(date)"

# 1. Find and remove all __pycache__ directories in app
echo ""
echo "--- Cleaning __pycache__ in $APP_DIR ---"
PYCACHE_COUNT=$(find "$APP_DIR" -name "__pycache__" -type d 2>/dev/null | wc -l)
if [[ $PYCACHE_COUNT -gt 0 ]]; then
    find "$APP_DIR" -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
    echo "✅ Removed $PYCACHE_COUNT __pycache__ directories from app"
else
    echo "✅ No __pycache__ directories found in app"
fi

# 2. Find and remove all __pycache__ directories in repo (if exists)
if [[ -d "$REPO_DIR" ]]; then
    echo ""
    echo "--- Cleaning __pycache__ in $REPO_DIR ---"
    PYCACHE_COUNT=$(find "$REPO_DIR" -name "__pycache__" -type d 2>/dev/null | wc -l)
    if [[ $PYCACHE_COUNT -gt 0 ]]; then
        find "$REPO_DIR" -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
        echo "✅ Removed $PYCACHE_COUNT __pycache__ directories from repo"
    else
        echo "✅ No __pycache__ directories found in repo"
    fi
fi

# 3. Remove .pyc files that might be scattered around
echo ""
echo "--- Cleaning stray .pyc files ---"
PYC_COUNT=$(find "$APP_DIR" -name "*.pyc" -type f 2>/dev/null | wc -l)
if [[ $PYC_COUNT -gt 0 ]]; then
    find "$APP_DIR" -name "*.pyc" -type f -delete 2>/dev/null || true
    echo "✅ Removed $PYC_COUNT .pyc files from app"
else
    echo "✅ No stray .pyc files found"
fi

# 4. Fix ownership of entire app directory
echo ""
echo "--- Fixing ownership of $APP_DIR ---"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$APP_DIR"
echo "✅ Ownership set to $SERVICE_USER:$SERVICE_GROUP"

# 5. Verify
echo ""
echo "--- Verification ---"
ROOT_OWNED=$(find "$APP_DIR" -user root 2>/dev/null | wc -l)
if [[ $ROOT_OWNED -eq 0 ]]; then
    echo "✅ No root-owned files in $APP_DIR"
else
    echo "⚠️  Found $ROOT_OWNED root-owned files"
    find "$APP_DIR" -user root 2>/dev/null | head -5
fi

echo ""
echo "=== Fix Complete ==="
echo "Services can be restarted safely now"
echo "Python will regenerate __pycache__ with correct ownership on startup"
