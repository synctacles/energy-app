#!/usr/bin/env python3
"""
ENTSO-E A44 Day-Ahead Prices Collector
"""

import os
import sys
import time
from pathlib import Path

import pandas as pd
from entsoe import EntsoePandasClient

from config.settings import LOG_PATH
from synctacles_db.core.logging import get_logger

SCRIPT_DIR = Path(__file__).parent
LOG_DIR = Path(LOG_PATH)
OUTPUT_DIR = LOG_DIR / "collectors" / "entso_e_raw"

_LOGGER = get_logger(__name__)

API_KEY = os.getenv("ENTSOE_API_KEY")
if not API_KEY:
    _LOGGER.error("ENTSOE_API_KEY not set")
    sys.exit(1)

client = EntsoePandasClient(api_key=API_KEY)
COUNTRY_CODE = "NL"


def fetch_day_ahead_prices(start_date, end_date):
    """Fetch prices using pandas Timestamp"""
    try:
        _LOGGER.debug(f"Request: A44 prices from {start_date} to {end_date}")
        prices = client.query_day_ahead_prices(
            country_code=COUNTRY_CODE, start=start_date, end=end_date
        )
        _LOGGER.debug(f"Response: {len(prices)} price records received")
        return prices
    except Exception as e:
        _LOGGER.error(f"ENTSO-E A44 API error: {type(e).__name__}: {e}")
        return None


def save_to_file(prices, date_str):
    """Save to CSV"""
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    filename = OUTPUT_DIR / f"a44_NL_prices_{date_str}.csv"

    with open(filename, "w") as f:
        f.write("timestamp,price_eur_mwh\n")
        for timestamp, price in prices.items():
            ts_utc = timestamp.tz_convert("UTC").isoformat()
            f.write(f"{ts_utc},{price:.2f}\n")

    _LOGGER.debug(f"Saved A44 prices to {filename}")
    return filename


def main():
    _LOGGER.info("ENTSO-E A44 Price Collector starting")
    start_time = time.time()

    try:
        # Use pandas Timestamp with timezone
        now = pd.Timestamp.now(tz="UTC")

        # Today
        start_today = now.normalize()  # 00:00:00 UTC today
        end_today = start_today + pd.Timedelta(days=1)

        _LOGGER.info(f"Fetching A44 prices for TODAY: {start_today} to {end_today}")
        prices_today = fetch_day_ahead_prices(start_today, end_today)

        if prices_today is not None and len(prices_today) > 0:
            save_to_file(prices_today, now.strftime("%Y%m%d"))
        else:
            _LOGGER.warning("No A44 prices available for today")

        # Tomorrow
        start_tomorrow = end_today
        end_tomorrow = start_tomorrow + pd.Timedelta(days=1)

        _LOGGER.info(
            f"Fetching A44 prices for TOMORROW: {start_tomorrow} to {end_tomorrow}"
        )
        prices_tomorrow = fetch_day_ahead_prices(start_tomorrow, end_tomorrow)

        if prices_tomorrow is not None and len(prices_tomorrow) > 0:
            tomorrow_date = (now + pd.Timedelta(days=1)).strftime("%Y%m%d")
            save_to_file(prices_tomorrow, tomorrow_date)
        else:
            _LOGGER.warning("No A44 prices for tomorrow (not published yet)")

        elapsed = time.time() - start_time
        _LOGGER.info(f"ENTSO-E A44 Price Collector completed in {elapsed:.2f}s")

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(
            f"A44 collector failed after {elapsed:.2f}s: {type(err).__name__}: {err}"
        )
        raise


if __name__ == "__main__":
    main()
