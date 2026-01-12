#!/bin/bash
# Run all data collectors
# Collects raw data from external sources (ENTSO-E, Energy-Charts, etc.)
#
# Error handling: Each collector runs independently. A failure in one
# collector does not stop the others. Exit code reflects overall status.

# Don't use set -e - we handle errors per collector
set -u
set -o pipefail

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

# Track failures
FAILED_COLLECTORS=()
SUCCESSFUL_COLLECTORS=()

# Function to run a collector with error handling
run_collector() {
    local name="$1"
    local module="$2"

    echo "[$(date +'%Y-%m-%d %H:%M:%S')] Running: ${name}..."

    if "${PYTHON}" -m "${module}" 2>&1; then
        SUCCESSFUL_COLLECTORS+=("${name}")
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] ${name}: SUCCESS"
    else
        FAILED_COLLECTORS+=("${name}")
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] ${name}: FAILED (continuing with next collector)"
    fi
}

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starting collector batch (Energy Action Focus mode)..."

# Phase 3: Energy Action Focus (2026-01-11)
# Only A44 (prices) and Energy-Charts (fallback) collectors are active
# A65 (load) and A75 (generation) collectors are DISCONTINUED

# Run collectors - each one independently (failures don't stop others)
run_collector "ENTSO-E A44 Prices" "synctacles_db.collectors.entso_e_a44_prices"
# SKIPPED: run_collector "ENTSO-E A65 Load" "synctacles_db.collectors.entso_e_a65_load"  # DISCONTINUED
# SKIPPED: run_collector "ENTSO-E A75 Generation" "synctacles_db.collectors.entso_e_a75_generation"  # DISCONTINUED
run_collector "Energy-Charts Prices" "synctacles_db.collectors.energy_charts_prices"

# Summary
echo ""
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Collector batch complete"
echo "  Successful: ${#SUCCESSFUL_COLLECTORS[@]} (${SUCCESSFUL_COLLECTORS[*]:-none})"
echo "  Failed: ${#FAILED_COLLECTORS[@]} (${FAILED_COLLECTORS[*]:-none})"

# Exit with error only if ALL collectors failed
if [ ${#SUCCESSFUL_COLLECTORS[@]} -eq 0 ] && [ ${#FAILED_COLLECTORS[@]} -gt 0 ]; then
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] CRITICAL: All collectors failed!"
    exit 1
fi

# Exit with warning code if some collectors failed
if [ ${#FAILED_COLLECTORS[@]} -gt 0 ]; then
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: Some collectors failed, but batch continues"
    exit 0  # Still exit 0 so systemd doesn't mark service as failed
fi

exit 0
