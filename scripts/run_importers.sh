#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/opt/synctacles/app"
VENV_DIR="/opt/synctacles/venv"
LOG_DIR="/opt/synctacles/logs/scheduler"

mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/importers_$(date +%Y%m%d_%H%M%S).log"

# Activate venv
source "$VENV_DIR/bin/activate"

# Set PYTHONPATH
export PYTHONPATH="$APP_DIR:${PYTHONPATH:-}"

# Load environment variables
if [[ -f /opt/synctacles/.env ]]; then
    set -a
    source /opt/synctacles/.env
    set +a
fi

cd "$APP_DIR"

echo "[$(date)] Starting importers..." | tee -a "$LOG_FILE"
python3 -m synctacles_db.importers.import_entso_e_a75 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.importers.import_entso_e_a65 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.importers.import_tennet_balance 2>&1 | tee -a "$LOG_FILE"
echo "[$(date)] Importers complete" | tee -a "$LOG_FILE"
