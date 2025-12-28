#!/usr/bin/env bash
# rollback.sh
# Rollback SYNCTACLES deployment to previous backup
# Version: 1.0 (2025-12-21)
#
# Usage:
#   ./rollback.sh                    # Interactive: show backups, select one
#   ./rollback.sh 20251220-153045    # Restore specific backup
#   ./rollback.sh --list             # List available backups

set -euo pipefail

RED="\e[31m"; GREEN="\e[32m"; YELLOW="\e[33m"; BLUE="\e[34m"; NC="\e[0m"

BACKUP_DIR="/opt/synctacles/backups/deployment"
PROD_APP="/opt/synctacles/app"
VENV="/opt/synctacles/venv"

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
[[ $EUID -eq 0 ]] || fail "Must run as root (sudo ./rollback.sh)"

# List backups function
list_backups() {
    if [[ ! -d "$BACKUP_DIR" ]]; then
        echo "No backups found (directory doesn't exist)"
        return 1
    fi
    
    local backups=($(ls -1d "$BACKUP_DIR"/*/ 2>/dev/null | xargs -n1 basename | sort -r))
    
    if [[ ${#backups[@]} -eq 0 ]]; then
        echo "No backups found in $BACKUP_DIR"
        return 1
    fi
    
    echo "Available backups:"
    echo
    
    local idx=1
    for backup in "${backups[@]}"; do
        local backup_path="$BACKUP_DIR/$backup"
        local version="unknown"
        
        if [[ -f "$backup_path/VERSION" ]]; then
            version=$(cat "$backup_path/VERSION")
        fi
        
        local size=$(du -sh "$backup_path" 2>/dev/null | cut -f1)
        
        printf "%2d) %s  [v%s, %s]\n" "$idx" "$backup" "$version" "$size"
        ((idx++))
    done
    
    echo
}

# Handle --list
if [[ "${1:-}" == "--list" ]]; then
    list_backups
    exit 0
fi

header "SYNCTACLES Rollback"

# Determine backup to restore
BACKUP_ID="${1:-}"

if [[ -z "$BACKUP_ID" ]]; then
    # Interactive mode
    list_backups || exit 1
    
    read -rp "Select backup number (or timestamp): " SELECTION
    
    # Check if numeric selection
    if [[ "$SELECTION" =~ ^[0-9]+$ ]]; then
        local backups=($(ls -1d "$BACKUP_DIR"/*/ 2>/dev/null | xargs -n1 basename | sort -r))
        local idx=$((SELECTION - 1))
        
        if [[ $idx -ge 0 && $idx -lt ${#backups[@]} ]]; then
            BACKUP_ID="${backups[$idx]}"
        else
            fail "Invalid selection: $SELECTION"
        fi
    else
        BACKUP_ID="$SELECTION"
    fi
fi

BACKUP_PATH="$BACKUP_DIR/$BACKUP_ID"

# Verify backup exists
if [[ ! -d "$BACKUP_PATH" ]]; then
    fail "Backup not found: $BACKUP_PATH"
fi

info "Restoring from: $BACKUP_PATH"

# Show backup info
if [[ -f "$BACKUP_PATH/VERSION" ]]; then
    BACKUP_VERSION=$(cat "$BACKUP_PATH/VERSION")
    info "Backup version: $BACKUP_VERSION"
fi

echo
warn "This will OVERWRITE current production code!"
read -rp "Continue? (yes/NO): " CONFIRM

if [[ "$CONFIRM" != "yes" ]]; then
    info "Rollback cancelled"
    exit 0
fi

# Create backup of current state (before rollback)
header "Creating Pre-Rollback Backup"

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
PRE_ROLLBACK_BACKUP="$BACKUP_DIR/pre-rollback-$TIMESTAMP"

mkdir -p "$PRE_ROLLBACK_BACKUP"

if [[ -d "$PROD_APP" ]]; then
    info "Backing up current state to: $PRE_ROLLBACK_BACKUP"
    rsync -a "$PROD_APP/" "$PRE_ROLLBACK_BACKUP/" || warn "Pre-rollback backup incomplete"
    ok "Pre-rollback backup created"
fi

# Restore from backup
header "Restoring Backup"

info "Restoring files from $BACKUP_ID..."

rsync -a --delete "$BACKUP_PATH/" "$PROD_APP/" || fail "Restore failed"

ok "Files restored"

# Fix permissions
info "Setting ownership: synctacles:synctacles"
chown -R synctacles:synctacles "$PROD_APP" || warn "Ownership fix failed"
ok "Ownership set"

# Database migration (downgrade if needed)
header "Database State"

warn "Manual database migration may be required!"
echo
echo "If backup version < current version:"
echo "  1. Check current revision: alembic current"
echo "  2. Downgrade if needed:    alembic downgrade <target_revision>"
echo
read -rp "Skip database migration? (Y/n): " SKIP_DB

if [[ "${SKIP_DB,,}" != "n" ]]; then
    warn "Skipping database migration - verify manually if needed"
else
    info "Database migrations:"
    cd "$PROD_APP" || fail "Cannot cd to $PROD_APP"
    
    export PYTHONPATH="$PROD_APP:${PYTHONPATH:-}"
    
    sudo -u synctacles PYTHONPATH="$PYTHONPATH" "$VENV/bin/alembic" current || warn "Cannot check current migration"
    
    echo
    read -rp "Run 'alembic upgrade head'? (y/N): " RUN_MIGRATE
    
    if [[ "${RUN_MIGRATE,,}" == "y" ]]; then
        sudo -u synctacles PYTHONPATH="$PYTHONPATH" "$VENV/bin/alembic" upgrade head || warn "Migration failed"
    fi
fi

# Restart services
header "Restarting Services"

info "Reloading systemd..."
systemctl daemon-reload
ok "Systemd reloaded"

info "Restarting API service..."
systemctl restart synctacles-api.service
sleep 3
ok "API service restarted"

# Validation
header "Post-Rollback Validation"

info "Checking API health..."
sleep 5

if curl -sf http://localhost:8000/health >/dev/null 2>&1; then
    API_RESPONSE=$(curl -s http://localhost:8000/health 2>/dev/null)
    API_STATUS=$(echo "$API_RESPONSE" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
    API_VERSION=$(echo "$API_RESPONSE" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown")
    
    if [[ "$API_STATUS" == "ok" ]]; then
        ok "API health check passed (version: $API_VERSION)"
    else
        warn "API responding but status=$API_STATUS"
    fi
else
    fail "API NOT responding after rollback"
fi

# Database connectivity
if sudo -u synctacles psql synctacles -c "SELECT 1" >/dev/null 2>&1; then
    ok "Database connection OK"
else
    warn "Database connection FAILED"
fi

# Timer count
TIMER_COUNT=$(systemctl list-timers synctacles-* --no-legend 2>/dev/null | wc -l)
if [[ $TIMER_COUNT -ge 3 ]]; then
    ok "$TIMER_COUNT timers active"
else
    warn "Only $TIMER_COUNT timers active (expected ≥3)"
fi

# Summary
header "Rollback Summary"

echo "Restored from:          $BACKUP_ID"
echo "Pre-rollback backup:    $PRE_ROLLBACK_BACKUP"
echo "Rollback time:          $(date)"
echo

ok "Rollback complete!"
echo
echo "If issues persist, restore pre-rollback state:"
echo "  sudo ./rollback.sh pre-rollback-$TIMESTAMP"
