"""
Energy-Charts Day-Ahead Price Collector
Fetches NL electricity prices from Fraunhofer ISE API
"""
import os
import json
import requests
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path

from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)

BASE_URL = "https://api.energy-charts.info/price"
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))
OUTPUT_DIR = LOG_DIR / "collectors" / "energy_charts_raw"

def fetch_prices(country: str = "NL", days: int = 2) -> dict:
    """Fetch day-ahead prices for country."""
    _LOGGER.info(f"Energy-Charts collector starting: country={country}, days={days}")
    start_time = time.time()

    try:
        OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

        today = datetime.now(timezone.utc).date()
        start = today.isoformat()
        end = (today + timedelta(days=days)).isoformat()

        params = {"bzn": country, "start": start, "end": end}

        _LOGGER.debug(f"Request URL: {BASE_URL}, params: {params}")

        response = requests.get(BASE_URL, params=params, timeout=30)
        response.raise_for_status()

        _LOGGER.debug(f"Response status: {response.status_code}, size: {len(response.content)} bytes")

        data = response.json()
        price_points = len(data.get('price', []))

        # Save raw response
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%d_%H%M%S")
        output_file = OUTPUT_DIR / f"prices_{country}_{timestamp}.json"

        with open(output_file, "w") as f:
            json.dump(data, f, indent=2)

        elapsed = time.time() - start_time
        _LOGGER.info(f"Energy-Charts collector completed: {price_points} records in {elapsed:.2f}s")

        return data

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"Energy-Charts collector failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise

if __name__ == "__main__":
    fetch_prices()