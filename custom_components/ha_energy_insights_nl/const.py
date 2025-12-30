"""Constants for Energy Insights NL integration."""
import json
import os
from pathlib import Path

DOMAIN = "ha_energy_insights_nl"

# Load component name from manifest (generated at install time)
_manifest_path = Path(__file__).parent / "manifest.json"
if not _manifest_path.exists():
    raise FileNotFoundError(
        f"manifest.json not found at {_manifest_path}\n"
        f"Run: sudo ./scripts/setup/setup_synctacles_server_v2.3.4.sh fase0\n"
        f"This generates manifest.json from manifest.json.template"
    )
_manifest = json.loads(_manifest_path.read_text())
HA_COMPONENT_NAME = _manifest["name"]  # No fallback - must exist

# GitHub account (from environment, no hardcoded defaults)
GITHUB_ACCOUNT = os.getenv("GITHUB_ACCOUNT", "Unknown")

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
