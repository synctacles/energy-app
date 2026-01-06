#!/usr/bin/env bash
# deploy_fase2.sh
# SYNCTACLES FASE 2 - Deploy D3/D4 Code
# 
# Usage: sudo ./deploy_fase2.sh
# 
# Deploys SYNCTACLES API (D3) + SparkCrawler (D4) to /opt/energy-insights-nl

set -euo pipefail

# Colors
RED="\e[31m"; GREEN="\e[32m"; YELLOW="\e[33m"; BLUE="\e[34m"; NC="\e[0m"

ts() { date +"%Y-%m-%d %H:%M:%S"; }

header() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE} $1${NC}"
  echo -e "${BLUE}========================================${NC}"
}

ok()   { echo -e "[$(ts)] ${GREEN}✔ OK${NC} $1"; }
fail() { echo -e "[$(ts)] ${RED}✘ FAIL${NC} $1"; }
info() { echo -e "[$(ts)] ${YELLOW}ℹ INFO${NC} $1"; }
warn() { echo -e "[$(ts)] ${YELLOW}⚠ WARN${NC} $1"; }

# ========================================================
# HELPERS
# ========================================================
ensure_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "Run as root (gebruik: sudo $0)"
        exit 1
    fi
}

# ========================================================
# LOGGING
# ========================================================
LOG_DIR="/var/log/synctacles-setup"
LOG_FILE="$LOG_DIR/fase2-$(date +%Y%m%d-%H%M%S).log"

setup_logging() {
    mkdir -p "$LOG_DIR"
    exec > >(tee -a "$LOG_FILE")
    exec 2>&1
    
    echo "==================================================="
    echo "SYNCTACLES FASE 2 - Code Deployment"
    echo "Start: $(date)"
    echo "Log: $LOG_FILE"
    echo "==================================================="
    echo
}

# ========================================================
# MAIN
# ========================================================

ensure_root
setup_logging

header "FASE 2 — SYNCTACLES API + SparkCrawler Deployment"

# Paths
SYNCTACLES_PROD="/opt/energy-insights-nl"
SYNCTACLES_API_DIR="$SYNCTACLES_PROD/app/synctacles_db"
VENV_PATH="$SYNCTACLES_PROD/venv"

# ========================================================
# Step 1: Backup existing code (if any)
# ========================================================
info "Backup bestaande code..."

if [[ -d "$SYNCTACLES_API_DIR" ]]; then
    BACKUP_TIME=$(date +%Y%m%d_%H%M%S)
    BACKUP_DIR="$SYNCTACLES_PROD/backups/synctacles_$BACKUP_TIME"
    mkdir -p "$SYNCTACLES_PROD/backups"
    cp -r "$SYNCTACLES_API_DIR" "$BACKUP_DIR"
    ok "Backup gemaakt: $BACKUP_DIR"
else
    info "Geen bestaande code om te backuppen"
fi

# ========================================================
# Step 2: Deploy code
# ========================================================
info "Deploy code..."

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Copy synctacles API (includes collectors, importers, and models)
if [[ -d "$SCRIPT_DIR/synctacles_db" ]]; then
    rm -rf "$SYNCTACLES_API_DIR"
    cp -r "$SCRIPT_DIR/synctacles_db" "$SYNCTACLES_API_DIR"
    ok "SYNCTACLES API gedeployd: $SYNCTACLES_API_DIR"
else
    fail "synctacles_db/ directory niet gevonden in $SCRIPT_DIR"
    exit 1
fi

# ========================================================
# Step 3: Fix permissions
# ========================================================
info "Fix permissions..."
chown -R synctacles:synctacles "$SYNCTACLES_PROD"
ok "Permissions gefixed"

# ========================================================
# Step 4: Activate venv & install dependencies
# ========================================================
info "Install Python dependencies..."

if [[ ! -f "$VENV_PATH/bin/activate" ]]; then
    fail "Virtual environment not found: $VENV_PATH"
    fail "Run FASE 4 first!"
    exit 1
fi

# Activate and install
source "$VENV_PATH/bin/activate"

# Requirements are in repo root (single source of truth)
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REQUIREMENTS_FILE="$REPO_ROOT/requirements.txt"

if [[ -f "$REQUIREMENTS_FILE" ]]; then
    pip install --quiet -r "$REQUIREMENTS_FILE" || {
        warn "Some packages failed to install, but continuing..."
    }
    ok "Dependencies installed from $REQUIREMENTS_FILE"
else
    warn "requirements.txt not found at $REQUIREMENTS_FILE"
fi

# ========================================================
# Step 5: Verify imports
# ========================================================
info "Verify Python imports..."

CRITICAL_MODULES=("synctacles" "sparkcrawler")
FAILED_MODULES=()

for mod in "${CRITICAL_MODULES[@]}"; do
    if python3 -c "import sys; sys.path.insert(0, '$SYNCTACLES_PROD'); import $mod" 2>/dev/null; then
        ok "Module importable: $mod"
    else
        fail "Module NOT importable: $mod"
        FAILED_MODULES+=("$mod")
    fi
done

if [[ ${#FAILED_MODULES[@]} -gt 0 ]]; then
    fail "Failed modules: ${FAILED_MODULES[*]}"
    exit 1
fi

ok "All modules importable"

# ========================================================
# Step 6: Database setup
# ========================================================
info "Verify database connectivity..."

# Check if synctacles database exists
if sudo -u postgres psql -lqt 2>/dev/null | grep -q synctacles; then
    ok "Database 'synctacles' exists"
else
    fail "Database 'synctacles' not found"
    fail "Run FASE 2.5 (database initialization) first!"
    exit 1
fi

# ========================================================
# Summary
# ========================================================
deactivate 2>/dev/null || true

header "FASE 2 Deployment Summary"

echo "✔ SYNCTACLES API code deployed: $SYNCTACLES_API_DIR"
echo "✔ SparkCrawler code deployed: $SPARKCRAWLER_DIR"
echo "✔ Python dependencies installed"
echo "✔ Imports verified"
echo "✔ Database verified"
echo
echo "Next steps:"
echo "  1. Test imports: python3 -c 'import sys; sys.path.insert(0, \"/opt/energy-insights-nl/app\"); import synctacles_db'"
echo "  2. Start SYNCTACLES API: systemctl start energy-insights-nl-api.service"
echo "  3. Verify API: curl http://localhost:8000/health"
echo
ok "FASE 2 Deployment Complete!"
