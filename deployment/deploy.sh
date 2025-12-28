#!/usr/bin/env bash
# deploy.sh
# Master deployment script for SYNCTACLES
# Version: 1.0 (2025-12-21)
#
# Usage:
#   ./deploy.sh              # Deploy current version
#   ./deploy.sh v1.0.1       # Deploy specific tag
#   ./deploy.sh --dry-run    # Show what would be synced
#   ./deploy.sh --skip-checks # Skip pre-deploy validation (dangerous)

set -euo pipefail

RED="\e[31m"; GREEN="\e[32m"; YELLOW="\e[33m"; BLUE="\e[34m"; NC="\e[0m"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST="$REPO_ROOT/deployment/sync-manifest.txt"
BACKUP_DIR="/opt/synctacles/backups/deployment"
PROD_APP="/opt/synctacles/app"
VENV="/opt/synctacles/venv"

DRY_RUN=false
SKIP_CHECKS=false
VERSION_TAG=""

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
        --skip-checks) SKIP_CHECKS=true ;;
        v*) VERSION_TAG="$arg" ;;
        *) echo "Unknown argument: $arg"; exit 1 ;;
    esac
done

header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

info() { echo -e "${BLUE}ℹ${NC} $1"; }
ok() { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; exit 1; }

# Ensure root
[[ $EUID -eq 0 ]] || fail "Must run as root (sudo ./deploy.sh)"

cd "$REPO_ROOT" || fail "Cannot cd to $REPO_ROOT"

header "SYNCTACLES Deployment"

# FASE 1: Pre-Deploy Checks
if ! $SKIP_CHECKS; then
    header "FASE 1: Pre-Deploy Validation"
    
    if [[ -x "$REPO_ROOT/deployment/pre-deploy-checks.sh" ]]; then
        "$REPO_ROOT/deployment/pre-deploy-checks.sh" || fail "Pre-deploy checks failed"
    else
        warn "pre-deploy-checks.sh not found or not executable"
    fi
else
    warn "Skipping pre-deploy checks (--skip-checks)"
fi

# Checkout version tag if specified
if [[ -n "$VERSION_TAG" ]]; then
    info "Checking out version: $VERSION_TAG"
    git fetch --tags >/dev/null 2>&1 || true
    git checkout "$VERSION_TAG" || fail "Cannot checkout $VERSION_TAG"
    ok "Checked out $VERSION_TAG"
fi

# Read VERSION file
if [[ -f VERSION ]]; then
    APP_VERSION=$(cat VERSION)
    info "Deploying version: $APP_VERSION"
else
    warn "VERSION file not found - proceeding without version tracking"
    APP_VERSION="unknown"
fi

# FASE 2: Backup Current Production
header "FASE 2: Backup Production"

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_PATH="$BACKUP_DIR/$TIMESTAMP"

mkdir -p "$BACKUP_PATH"

if [[ -d "$PROD_APP" ]]; then
    info "Backing up $PROD_APP to $BACKUP_PATH"
    rsync -a "$PROD_APP/" "$BACKUP_PATH/" || warn "Backup incomplete"
    ok "Backup created: $BACKUP_PATH"
else
    warn "Production app directory not found (first deploy?)"
fi

# FASE 3: Sync Files
header "FASE 3: Sync Files"

if [[ ! -f "$MANIFEST" ]]; then
    fail "Sync manifest not found: $MANIFEST"
fi

mkdir -p "$PROD_APP"

# Parse manifest and sync
while IFS= read -r line; do
    # Skip comments and empty lines
    [[ "$line" =~ ^# ]] && continue
    [[ -z "$line" ]] && continue
    
    # Parse SOURCE → DESTINATION
    if [[ "$line" =~ ^(.+)\ →\ (.+)$ ]]; then
        SRC="${BASH_REMATCH[1]}"
        DST="${BASH_REMATCH[2]}"
        
        # Expand wildcards in source
        SRC_FULL="$REPO_ROOT/$SRC"
        
        if [[ "$SRC" == *"*"* ]]; then
            # Wildcard sync (e.g., systemd/*.service)
            SRC_PATTERN=$(basename "$SRC")
            SRC_DIR=$(dirname "$SRC_FULL")
            
            if [[ -d "$SRC_DIR" ]]; then
                for file in "$SRC_DIR"/$SRC_PATTERN; do
                    [[ -e "$file" ]] || continue
                    
                    if $DRY_RUN; then
                        echo "  WOULD COPY: $file → $DST"
                    else
                        cp "$file" "$DST/" 2>/dev/null || warn "Failed: $file → $DST"
                        ok "Synced: $(basename "$file") → $DST"
                    fi
                done
            fi
        elif [[ -d "$SRC_FULL" ]]; then
            # Directory sync
            if $DRY_RUN; then
                echo "  WOULD SYNC: $SRC → $DST"
            else
                rsync -a --delete "$SRC_FULL/" "$DST/" || warn "Failed: $SRC → $DST"
                ok "Synced: $SRC → $DST"
            fi
        elif [[ -f "$SRC_FULL" ]]; then
            # File sync
            DST_DIR=$(dirname "$DST")
            mkdir -p "$DST_DIR"
            
            if $DRY_RUN; then
                echo "  WOULD COPY: $SRC → $DST"
            else
                cp "$SRC_FULL" "$DST" || warn "Failed: $SRC → $DST"
                ok "Synced: $SRC → $DST"
            fi
        else
            warn "Source not found: $SRC"
        fi
    fi
done < "$MANIFEST"

if $DRY_RUN; then
    info "Dry-run complete - no changes made"
    exit 0
fi

# FASE 4: Post-Sync Actions
header "FASE 4: Post-Sync Actions"

# Permissions
info "Setting ownership: synctacles:synctacles"
chown -R synctacles:synctacles "$PROD_APP" || warn "Ownership fix failed"
ok "Ownership set"

# Database migrations
if [[ -d "$PROD_APP/alembic" ]]; then
    info "Running database migrations..."
    
    cd "$PROD_APP" || fail "Cannot cd to $PROD_APP"
    
    export PYTHONPATH="$PROD_APP:${PYTHONPATH:-}"
    
    if sudo -u synctacles PYTHONPATH="$PYTHONPATH" "$VENV/bin/alembic" upgrade head 2>&1; then
        ok "Database migrations applied"
    else
        fail "Database migrations FAILED - rollback recommended"
    fi
    
    cd "$REPO_ROOT" || true
fi

# Systemd reload
if systemctl --version >/dev/null 2>&1; then
    info "Reloading systemd..."
    systemctl daemon-reload
    ok "Systemd reloaded"
fi

# Restart API service
if systemctl is-active --quiet synctacles-api.service 2>/dev/null; then
    info "Restarting API service..."
    systemctl restart synctacles-api.service
    sleep 3
    ok "API service restarted"
fi

# FASE 5: Validation
header "FASE 5: Post-Deploy Validation"

# API health check
info "Checking API health..."
sleep 5  # Allow service to stabilize

if curl -sf http://localhost:8000/health >/dev/null 2>&1; then
    API_RESPONSE=$(curl -s http://localhost:8000/health 2>/dev/null)
    API_STATUS=$(echo "$API_RESPONSE" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
    API_VERSION=$(echo "$API_RESPONSE" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown")
    
    if [[ "$API_STATUS" == "ok" ]]; then
        ok "API health check passed (version: $API_VERSION)"
        
        # Version mismatch warning
        if [[ "$API_VERSION" != "$APP_VERSION" ]]; then
            warn "Version mismatch: API=$API_VERSION, Deployed=$APP_VERSION"
            warn "API may need manual restart"
        fi
    else
        warn "API responding but status=$API_STATUS"
    fi
else
    fail "API NOT responding - rollback recommended"
fi

# Timer count
TIMER_COUNT=$(systemctl list-timers synctacles-* --no-legend 2>/dev/null | wc -l)
if [[ $TIMER_COUNT -ge 3 ]]; then
    ok "$TIMER_COUNT timers active"
else
    warn "Only $TIMER_COUNT timers active (expected ≥3)"
fi

# Database connectivity
if sudo -u synctacles psql synctacles -c "SELECT 1" >/dev/null 2>&1; then
    ok "Database connection OK"
else
    warn "Database connection FAILED"
fi

# VERSION file verification
if [[ -f "$PROD_APP/VERSION" ]]; then
    DEPLOYED_VER=$(cat "$PROD_APP/VERSION")
    ok "VERSION file deployed: $DEPLOYED_VER"
else
    warn "VERSION file missing in production"
fi

# Summary
header "Deployment Summary"

echo "Deployed version: $APP_VERSION"
echo "Backup location:  $BACKUP_PATH"
echo "Deployment time:  $(date)"
echo

if [[ -n "$VERSION_TAG" ]]; then
    info "Returning to main branch..."
    git checkout main >/dev/null 2>&1 || warn "Could not checkout main"
fi

ok "Deployment complete!"
echo
echo "Rollback command (if needed):"
echo "  sudo $REPO_ROOT/deployment/rollback.sh $TIMESTAMP"
