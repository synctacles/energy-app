"""
Energy-Charts Day-Ahead Price Collector
Fetches NL electricity prices from Fraunhofer ISE API
"""
import json
import requests
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path

from synctacles_db.core.logging import get_logger
from config.settings import LOG_PATH

_LOGGER = get_logger(__name__)

BASE_URL = "https://api.energy-charts.info/price"
LOG_DIR = Path(LOG_PATH)
OUTPUT_DIR = LOG_DIR / "collectors" / "energy_charts_raw"

# Retry configuration
MAX_RETRIES = 3
RETRY_DELAYS = [30, 60, 120]  # seconds between retries (exponential backoff)


def _make_request_with_retry(url: str, params: dict) -> requests.Response:
    """Make HTTP request with retry logic for rate limiting."""
    last_error = None

    for attempt in range(MAX_RETRIES):
        try:
            response = requests.get(url, params=params, timeout=30)

            if response.status_code == 429:
                retry_delay = RETRY_DELAYS[min(attempt, len(RETRY_DELAYS) - 1)]
                retry_after = response.headers.get('Retry-After')
                if retry_after:
                    try:
                        retry_delay = int(retry_after)
                    except ValueError:
                        pass

                _LOGGER.warning(
                    f"Rate limited (429), attempt {attempt + 1}/{MAX_RETRIES}. "
                    f"Waiting {retry_delay}s before retry..."
                )
                time.sleep(retry_delay)
                continue

            response.raise_for_status()
            return response

        except requests.exceptions.HTTPError as e:
            if e.response is not None and e.response.status_code == 429:
                last_error = e
                continue
            raise
        except requests.exceptions.RequestException as e:
            last_error = e
            if attempt < MAX_RETRIES - 1:
                retry_delay = RETRY_DELAYS[min(attempt, len(RETRY_DELAYS) - 1)]
                _LOGGER.warning(
                    f"Request failed: {e}. Attempt {attempt + 1}/{MAX_RETRIES}. "
                    f"Retrying in {retry_delay}s..."
                )
                time.sleep(retry_delay)
            else:
                raise

    # If we exhausted retries due to rate limiting
    raise requests.exceptions.HTTPError(
        f"Rate limited after {MAX_RETRIES} attempts", response=last_error.response if last_error else None
    )


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

        response = _make_request_with_retry(BASE_URL, params)

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