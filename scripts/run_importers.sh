#!/bin/bash
# Run all data importers
# Imports raw data from external sources into raw tables

set -e

INSTALL_PATH="${INSTALL_PATH:-/opt/energy-insights-nl}"
VENV_PATH="${VENV_PATH:-${INSTALL_PATH}/venv}"
APP_PATH="${APP_PATH:-${INSTALL_PATH}/app}"
LOG_PATH="${LOG_PATH:-/var/log/energy-insights}"
ENV_FILE="${ENV_FILE:-/opt/.env}"

# Create log directory if needed
mkdir -p "${LOG_PATH}"

# Python path
PYTHON="${VENV_PATH}/bin/python3"

# Source environment variables
if [ -f "$ENV_FILE" ]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

cd "$APP_PATH"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starting importers (Energy Action Focus mode)..."

# Run importers (they handle failures internally)
# NOTE: TenneT importer intentionally excluded (off-limits, BYO-KEY model per SKILL_02)
#
# Phase 2: Energy Action Focus (2026-01-11)
# A65 (load) and A75 (generation) importers are SKIPPED
# Only A44 (prices) is needed for Energy Action

# SKIPPED: "${PYTHON}" -m synctacles_db.importers.import_entso_e_a75  # Generation - DISCONTINUED
# SKIPPED: "${PYTHON}" -m synctacles_db.importers.import_entso_e_a65  # Load - DISCONTINUED
"${PYTHON}" -m synctacles_db.importers.import_entso_e_a44  # Prices - ACTIVE

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Importers complete (Energy Action Focus mode)"
