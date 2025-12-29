#!/usr/bin/env python3
"""
ENTSO-E A44 Day-Ahead Prices Collector
"""

import os
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from entsoe import EntsoePandasClient
import pandas as pd
import logging

SCRIPT_DIR = Path(__file__).parent
LOG_DIR = Path(os.getenv('LOG_PATH', '/var/log/energy-insights'))
OUTPUT_DIR = LOG_DIR / 'collectors' / 'entso_e_raw'

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

API_KEY = os.getenv('ENTSOE_API_KEY')
if not API_KEY:
    logger.error("ENTSOE_API_KEY not set")
    sys.exit(1)

client = EntsoePandasClient(api_key=API_KEY)
COUNTRY_CODE = 'NL'

def fetch_day_ahead_prices(start_date, end_date):
    """Fetch prices using pandas Timestamp"""
    try:
        prices = client.query_day_ahead_prices(
            country_code=COUNTRY_CODE,
            start=start_date,
            end=end_date
        )
        logger.info(f"Fetched {len(prices)} price records")
        return prices
    except Exception as e:
        logger.error(f"ENTSO-E API error: {e}")
        return None

def save_to_file(prices, date_str):
    """Save to CSV"""
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    filename = OUTPUT_DIR / f"a44_NL_prices_{date_str}.csv"
    
    with open(filename, 'w') as f:
        f.write("timestamp,price_eur_mwh\n")
        for timestamp, price in prices.items():
            ts_utc = timestamp.tz_convert('UTC').isoformat()
            f.write(f"{ts_utc},{price:.2f}\n")
    
    logger.info(f"Saved to {filename}")
    return filename

def main():
    logger.info("=== ENTSO-E A44 Price Collector ===")
    
    # Use pandas Timestamp with timezone
    now = pd.Timestamp.now(tz='UTC')
    
    # Today
    start_today = now.normalize()  # 00:00:00 UTC today
    end_today = start_today + pd.Timedelta(days=1)
    
    logger.info(f"Fetching TODAY: {start_today} to {end_today}")
    prices_today = fetch_day_ahead_prices(start_today, end_today)
    
    if prices_today is not None and len(prices_today) > 0:
        save_to_file(prices_today, now.strftime('%Y%m%d'))
    else:
        logger.warning("No prices for today")
    
    # Tomorrow
    start_tomorrow = end_today
    end_tomorrow = start_tomorrow + pd.Timedelta(days=1)
    
    logger.info(f"Fetching TOMORROW: {start_tomorrow} to {end_tomorrow}")
    prices_tomorrow = fetch_day_ahead_prices(start_tomorrow, end_tomorrow)
    
    if prices_tomorrow is not None and len(prices_tomorrow) > 0:
        tomorrow_date = (now + pd.Timedelta(days=1)).strftime('%Y%m%d')
        save_to_file(prices_tomorrow, tomorrow_date)
    else:
        logger.warning("No prices for tomorrow (not published yet)")
    
    logger.info("=== Collector Complete ===")

if __name__ == '__main__':
    main()
