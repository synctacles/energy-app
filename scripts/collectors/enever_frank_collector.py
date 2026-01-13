#!/usr/bin/env python3
"""
Enever-Frank Price Collector.

Collects Frank-equivalent prices calculated from Enever data via the
Coefficient Server's /internal/enever/frank endpoint.

This provides TRUE redundancy for Tier 2 - prices are derived from Enever API
(independent source) with correction factors applied, NOT from Frank API.

Stores prices in the local Synctacles database (enever_frank_prices table).

Runs 2x daily via systemd timer:
- 07:05 UTC: Collect today's prices
- 15:05 UTC: Collect today + tomorrow prices

Usage:
    python enever_frank_collector.py [--tomorrow]

Exit codes:
    0: Success
    1: No prices collected
    2: Database error
"""
import asyncio
import logging
import os
import sys
from datetime import datetime, timezone, timedelta
from typing import List, Dict, Optional

import asyncpg
import aiohttp

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
_LOGGER = logging.getLogger(__name__)

# Coefficient Server API
COEFFICIENT_SERVER = os.getenv("COEFFICIENT_SERVER", "http://91.99.150.36:8080")

# Add project root to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
from config.settings import DATABASE_URL


async def fetch_enever_frank_prices(date: str = "today") -> Optional[List[Dict]]:
    """
    Fetch Frank-equivalent prices from Coefficient Server.

    Uses the /internal/enever/frank endpoint which:
    1. Reads Enever consumer prices (independent of Frank API)
    2. Applies correction factors from enever_frank_coefficient_lookup
    3. Returns Frank-equivalent prices

    This provides TRUE redundancy - Tier 2 is now independent of Frank API.

    Args:
        date: "today" or "tomorrow"

    Returns:
        List of price dicts with timestamp and price, or None on failure
    """
    try:
        timeout = aiohttp.ClientTimeout(total=30)
        async with aiohttp.ClientSession(timeout=timeout) as session:
            # Use the new Enever-Frank endpoint with coefficient model
            url = f"{COEFFICIENT_SERVER}/internal/enever/frank"
            if date == "tomorrow":
                url += "?date=tomorrow"

            _LOGGER.info(f"Fetching from: {url}")

            async with session.get(url) as resp:
                if resp.status != 200:
                    text = await resp.text()
                    _LOGGER.error(f"Coefficient API returned HTTP {resp.status}: {text}")
                    return None

                data = await resp.json()

                # Check for error response
                if "error" in data:
                    _LOGGER.warning(f"API returned error: {data['error']}")
                    return None

                # Extract prices from response
                prices_list = data.get("prices", [])
                if not prices_list:
                    _LOGGER.warning(f"No prices in response for {date}")
                    return None

                # Log metadata
                source = data.get("source", "unknown")
                model = data.get("model", "unknown")
                avg_correction = data.get("avg_correction", 1.0)
                _LOGGER.info(
                    f"Received {len(prices_list)} prices from {source} "
                    f"(model: {model}, avg_correction: {avg_correction:.4f})"
                )

                # Prices are already in correct format from endpoint
                prices = []
                for p in prices_list:
                    if "timestamp" in p and "price_eur_kwh" in p:
                        prices.append({
                            "timestamp": p["timestamp"],
                            "price_eur_kwh": float(p["price_eur_kwh"]),
                        })

                _LOGGER.info(f"Processed {len(prices)} prices for {date}")
                return prices

    except aiohttp.ClientError as e:
        _LOGGER.error(f"HTTP error connecting to Coefficient Server: {e}")
        return None
    except Exception as e:
        _LOGGER.error(f"Unexpected error fetching prices: {e}")
        return None


async def store_prices(prices: List[Dict]) -> int:
    """
    Store prices in enever_frank_prices table using UPSERT.

    Args:
        prices: List of price dicts

    Returns:
        Number of rows upserted
    """
    if not prices:
        return 0

    try:
        conn = await asyncpg.connect(DATABASE_URL)

        # Use UPSERT (INSERT ON CONFLICT UPDATE)
        upsert_sql = """
            INSERT INTO enever_frank_prices (timestamp, price_eur_kwh)
            VALUES ($1, $2)
            ON CONFLICT (timestamp) DO UPDATE SET
                price_eur_kwh = EXCLUDED.price_eur_kwh,
                created_at = NOW()
        """

        count = 0
        for p in prices:
            ts = datetime.fromisoformat(p["timestamp"])
            await conn.execute(
                upsert_sql,
                ts,
                p["price_eur_kwh"]
            )
            count += 1

        await conn.close()
        _LOGGER.info(f"Stored {count} prices in enever_frank_prices table")
        return count

    except Exception as e:
        _LOGGER.error(f"Database error: {e}")
        raise


async def main(include_tomorrow: bool = False) -> int:
    """
    Main collector function.

    Args:
        include_tomorrow: If True, also fetch tomorrow's prices

    Returns:
        Exit code (0=success, 1=no prices, 2=db error)
    """
    # Fetch today's prices
    _LOGGER.info("Collecting Enever-Frank prices for today")
    prices = await fetch_enever_frank_prices("today")

    if not prices:
        _LOGGER.error("No prices collected from Enever/Frank")
        return 1

    # Optionally fetch tomorrow's prices
    if include_tomorrow:
        _LOGGER.info("Also collecting Enever-Frank prices for tomorrow")
        tomorrow_prices = await fetch_enever_frank_prices("tomorrow")
        if tomorrow_prices:
            prices.extend(tomorrow_prices)
        else:
            _LOGGER.warning("Tomorrow's prices not yet available")

    # Store in database
    try:
        count = await store_prices(prices)
        _LOGGER.info(f"Enever-Frank collector completed: {count} prices stored")
        return 0
    except Exception:
        return 2


if __name__ == "__main__":
    include_tomorrow = "--tomorrow" in sys.argv
    exit_code = asyncio.run(main(include_tomorrow))
    sys.exit(exit_code)
