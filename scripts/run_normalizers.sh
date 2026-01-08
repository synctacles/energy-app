#!/bin/bash
# Run all data normalizers
# Normalizes raw data to normalized tables

set -e

INSTALL_PATH="${INSTALL_PATH:-/opt/energy-insights-nl}"
VENV_PATH="${VENV_PATH:-${INSTALL_PATH}/venv}"
APP_PATH="${APP_PATH:-${INSTALL_PATH}/app}"
LOG_PATH="${LOG_PATH:-/var/log/energy-insights}"

# Source environment variables
if [ -f "/opt/.env" ]; then
    set -a
    source /opt/.env
    set +a
fi

# Create log directory if needed
mkdir -p "${LOG_PATH}"

# Python path
PYTHON="${VENV_PATH}/bin/python3"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starting normalizer batch..."

# Run normalizers (they handle failures internally)
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Normalizer batch complete"
