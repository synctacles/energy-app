#!/bin/bash
# Run all data collectors
# Collects raw data from external sources (ENTSO-E, Energy-Charts, etc.)

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

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starting collector batch..."

# Run collectors (they handle failures internally)
# A44 prices (primary ENTSO-E source)
"${PYTHON}" -m synctacles_db.collectors.entso_e_a44_prices

# A65 load (ENTSO-E)
"${PYTHON}" -m synctacles_db.collectors.entso_e_a65_load

# A75 generation (ENTSO-E)
"${PYTHON}" -m synctacles_db.collectors.entso_e_a75_generation

# Energy-Charts prices (fallback source)
"${PYTHON}" -m synctacles_db.collectors.energy_charts_prices

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Collector batch complete"
