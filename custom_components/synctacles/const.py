"""Constants for Energy Insights NL integration."""
import json
from pathlib import Path

DOMAIN = "ha_energy_insights_nl"

# Load component name from manifest for consistency
_manifest_path = Path(__file__).parent / "manifest.json"
try:
    _manifest = json.loads(_manifest_path.read_text())
    HA_COMPONENT_NAME = _manifest.get("name", "Energy Insights NL")
except Exception:
    HA_COMPONENT_NAME = "Energy Insights NL"

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
