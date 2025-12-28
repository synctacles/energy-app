"""Constants for SYNCTACLES integration."""

DOMAIN = "synctacles"

# Config keys
CONF_API_URL = "api_url"

# Default values
DEFAULT_API_URL = "http://localhost:8000"

# Update intervals (seconds)
SCAN_INTERVAL_GENERATION = 900  # 15 min
SCAN_INTERVAL_LOAD = 900        # 15 min
SCAN_INTERVAL_BALANCE = 300     # 5 min

# Quality status thresholds (seconds)
QUALITY_OK_MAX = 900        # 15 min
QUALITY_STALE_MAX = 3600    # 1 hour
