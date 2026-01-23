#!/usr/bin/env python3
"""
Frank Energie Price Collector.

Collects consumer prices from Frank Energie GraphQL API and stores them
in the local Synctacles database (frank_prices table).

Runs 2x daily via systemd timer:
- 07:00 UTC: Collect today's prices
- 15:00 UTC: Collect today + tomorrow prices (tomorrow available after 14:00 CET)

Usage:
    python frank_collector.py [--tomorrow]

Exit codes:
    0: Success
    1: No prices collected
    2: Database error
"""

import asyncio
import logging
import os
import sys
from datetime import UTC, datetime, timedelta

import aiohttp
import asyncpg

# Setup logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
_LOGGER = logging.getLogger(__name__)

# Frank Energie GraphQL endpoint
FRANK_API_URL = "https://graphql.frankenergie.nl"

# Add project root to path for imports
sys.path.insert(
    0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
)
from config.settings import DATABASE_URL


async def fetch_frank_prices(start_date: str, end_date: str) -> list[dict] | None:
    """
    Fetch electricity prices from Frank Energie GraphQL API.

    Args:
        start_date: Start date in YYYY-MM-DD format
        end_date: End date in YYYY-MM-DD format

    Returns:
        List of price dicts with timestamp and price components, or None on failure
    """
    query = """
    query {
      marketPricesElectricity(startDate: "%s", endDate: "%s") {
        from
        till
        marketPrice
        marketPriceTax
        sourcingMarkupPrice
        energyTaxPrice
      }
    }
    """ % (start_date, end_date)

    try:
        timeout = aiohttp.ClientTimeout(total=30)
        async with aiohttp.ClientSession(timeout=timeout) as session:
            async with session.post(
                FRANK_API_URL,
                json={"query": query},
                headers={"Content-Type": "application/json"},
            ) as resp:
                if resp.status != 200:
                    _LOGGER.error(f"Frank API returned HTTP {resp.status}")
                    return None

                data = await resp.json()

                if "errors" in data:
                    _LOGGER.error(f"Frank GraphQL errors: {data['errors']}")
                    return None

                raw_prices = data.get("data", {}).get("marketPricesElectricity", [])

                if not raw_prices:
                    _LOGGER.warning("Frank API returned empty prices")
                    return None

                # Transform to standard format
                prices = []
                for p in raw_prices:
                    # Parse timestamp
                    ts_str = p["from"]
                    if ts_str.endswith("Z"):
                        ts_str = ts_str[:-1] + "+00:00"
                    ts_str = ts_str.replace(".000Z", "+00:00")

                    # Calculate total consumer price (all components)
                    market_price = float(p.get("marketPrice", 0) or 0)
                    market_price_tax = float(p.get("marketPriceTax", 0) or 0)
                    sourcing_markup = float(p.get("sourcingMarkupPrice", 0) or 0)
                    energy_tax = float(p.get("energyTaxPrice", 0) or 0)

                    total_price = (
                        market_price + market_price_tax + sourcing_markup + energy_tax
                    )

                    prices.append(
                        {
                            "timestamp": ts_str,
                            "price_eur_kwh": total_price,
                            "market_price": market_price,
                            "market_price_tax": market_price_tax,
                            "sourcing_markup": sourcing_markup,
                            "energy_tax": energy_tax,
                        }
                    )

                _LOGGER.info(f"Fetched {len(prices)} prices from Frank API")
                return prices

    except Exception as e:
        _LOGGER.error(f"Frank API fetch failed: {e}")
        return None


async def store_prices(prices: list[dict]) -> int:
    """
    Store prices in frank_prices table using UPSERT.

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
            INSERT INTO frank_prices (timestamp, price_eur_kwh, market_price, market_price_tax, sourcing_markup, energy_tax)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (timestamp) DO UPDATE SET
                price_eur_kwh = EXCLUDED.price_eur_kwh,
                market_price = EXCLUDED.market_price,
                market_price_tax = EXCLUDED.market_price_tax,
                sourcing_markup = EXCLUDED.sourcing_markup,
                energy_tax = EXCLUDED.energy_tax,
                created_at = NOW()
        """

        count = 0
        for p in prices:
            ts = datetime.fromisoformat(p["timestamp"])
            await conn.execute(
                upsert_sql,
                ts,
                p["price_eur_kwh"],
                p["market_price"],
                p["market_price_tax"],
                p["sourcing_markup"],
                p["energy_tax"],
            )
            count += 1

        await conn.close()
        _LOGGER.info(f"Stored {count} prices in frank_prices table")
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
    today = datetime.now(UTC).date()

    # Fetch today's prices
    _LOGGER.info(f"Collecting Frank prices for {today}")
    prices = await fetch_frank_prices(today.isoformat(), today.isoformat())

    if not prices:
        _LOGGER.error("No prices collected from Frank API")
        return 1

    # Optionally fetch tomorrow's prices (available after 14:00 CET)
    if include_tomorrow:
        tomorrow = today + timedelta(days=1)
        _LOGGER.info(f"Also collecting Frank prices for {tomorrow}")
        tomorrow_prices = await fetch_frank_prices(
            tomorrow.isoformat(), tomorrow.isoformat()
        )
        if tomorrow_prices:
            prices.extend(tomorrow_prices)
        else:
            _LOGGER.warning("Tomorrow's prices not yet available")

    # Store in database
    try:
        count = await store_prices(prices)
        _LOGGER.info(f"Frank collector completed: {count} prices stored")
        return 0
    except Exception:
        return 2


if __name__ == "__main__":
    include_tomorrow = "--tomorrow" in sys.argv
    exit_code = asyncio.run(main(include_tomorrow))
    sys.exit(exit_code)
