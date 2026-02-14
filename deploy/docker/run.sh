#!/usr/bin/with-contenv bashio
# Run script for HA addon — reads options from /data/options.json via bashio.

# Read addon options and export as env vars
export ENERGY_ZONE="$(bashio::config 'zone' 'NL')"
export ENERGY_GO_THRESHOLD="$(bashio::config 'go_threshold' '-15')"
export ENERGY_AVOID_THRESHOLD="$(bashio::config 'avoid_threshold' '20')"
export ENEVER_ENABLED="$(bashio::config 'enever_enabled' 'false')"
export ENEVER_TOKEN="$(bashio::config 'enever_token' '')"
export ENEVER_LEVERANCIER="$(bashio::config 'enever_leverancier' '')"
export ENERGY_COEFFICIENT="$(bashio::config 'coefficient' '0')"
export BEST_WINDOW_HOURS="$(bashio::config 'best_window_hours' '3')"
export POWER_SENSOR_ENTITY="$(bashio::config 'power_sensor' '')"
# Hidden: override Synctacles server URL (empty = use production default)
_synctacles_url="$(bashio::config 'synctacles_url' '')"
if [ -n "${_synctacles_url}" ]; then
    export SYNCTACLES_URL="${_synctacles_url}"
    bashio::log.info "Using custom Synctacles server: ${_synctacles_url}"
fi
export DEBUG_MODE="$(bashio::config 'debug_mode' 'false')"

# HA provides these automatically
export SUPERVISOR_TOKEN="${SUPERVISOR_TOKEN}"
export INGRESS_PORT="${INGRESS_PORT:-8098}"

bashio::log.info "Starting Synctacles Energy addon (zone=${ENERGY_ZONE})"

exec /usr/bin/energy-addon
