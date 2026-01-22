#!/bin/bash
# Run all data normalizers
# Normalizes raw data to normalized tables

set -e

# Source environment variables FIRST (required for paths)
if [ -f "/opt/.env" ]; then
    set -a
    source /opt/.env
    set +a
else
    echo "FATAL: /opt/.env not found. Run setup FASE 0 first." >&2
    exit 1
fi

# Paths from ENV (fail if not set)
: "${INSTALL_PATH:?INSTALL_PATH not set in /opt/.env}"
: "${LOG_PATH:?LOG_PATH not set in /opt/.env}"
VENV_PATH="${VENV_PATH:-${INSTALL_PATH}/venv}"
APP_PATH="${APP_PATH:-${INSTALL_PATH}/app}"

# Create log directory if needed
mkdir -p "${LOG_PATH}"

# Python path
PYTHON="${VENV_PATH}/bin/python3"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starting normalizer batch (Energy Action Focus mode)..."

# Phase 2: Energy Action Focus (2026-01-11)
# Only A44 (prices) normalizer is needed for Energy Action
# A65 (load) and A75 (generation) normalizers are SKIPPED

# ENTSO-E normalizers - Only A44 (prices)
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A44 (prices)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44

# SKIPPED: A65 (load) - DISCONTINUED
# echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A65 (load)..."
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65

# SKIPPED: A75 (generation) - DISCONTINUED
# echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A75 (generation)..."
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75

# Price post-processing
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing price aggregation..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Normalizer batch complete (Energy Action Focus mode)"
