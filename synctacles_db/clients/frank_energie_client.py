"""
Frank Energie Direct Client.

Direct HTTP client for Frank Energie GraphQL API (graphql.frankenergie.nl).
Primary data source for Tier 1 in the fallback chain.

Features:
- Direct API access (no VPN required since Jan 2026)
- Circuit breaker pattern for resilience
- TTL caching for performance
- Returns consumer prices including all taxes and markups
"""
import logging
from datetime import UTC, datetime, timedelta

import aiohttp
from cachetools import TTLCache

_LOGGER = logging.getLogger(__name__)

# Frank Energie GraphQL endpoint
FRANK_API_URL = "https://graphql.frankenergie.nl"

# Cache: 5 minute TTL (prices update hourly, 5 min is sufficient)
_frank_cache = TTLCache(maxsize=10, ttl=300)

# Circuit breaker for Frank API
_circuit_breaker = {
    "last_failure_time": None,
    "cooldown_minutes": 5,
    "failure_count": 0,
    "max_failures": 3,
}


class FrankEnergieClient:
    """Direct HTTP client for Frank Energie GraphQL API."""

    @staticmethod
    def _check_circuit_breaker() -> bool:
        """
        Check if circuit breaker is open.

        Returns True if requests should be skipped.
        """
        if _circuit_breaker["failure_count"] < _circuit_breaker["max_failures"]:
            return False

        if not _circuit_breaker["last_failure_time"]:
            return False

        now = datetime.now(UTC)
        last_failure = _circuit_breaker["last_failure_time"]
        minutes_since = (now - last_failure).total_seconds() / 60

        if minutes_since < _circuit_breaker["cooldown_minutes"]:
            _LOGGER.debug(f"Frank circuit breaker OPEN ({int(minutes_since)} min since failure)")
            return True

        # Reset after cooldown
        _circuit_breaker["failure_count"] = 0
        _circuit_breaker["last_failure_time"] = None
        _LOGGER.info("Frank circuit breaker CLOSED (cooldown expired)")
        return False

    @staticmethod
    def _record_failure():
        """Record a failure for circuit breaker."""
        _circuit_breaker["failure_count"] += 1
        _circuit_breaker["last_failure_time"] = datetime.now(UTC)
        _LOGGER.warning(f"Frank circuit breaker failure count: {_circuit_breaker['failure_count']}")

    @staticmethod
    def _record_success():
        """Reset circuit breaker on success."""
        if _circuit_breaker["failure_count"] > 0:
            _circuit_breaker["failure_count"] = 0
            _circuit_breaker["last_failure_time"] = None

    @staticmethod
    async def get_prices(
        start_date: str | None = None,
        end_date: str | None = None
    ) -> list[dict] | None:
        """
        Fetch electricity prices from Frank Energie GraphQL API.

        Args:
            start_date: Start date in YYYY-MM-DD format (default: today)
            end_date: End date in YYYY-MM-DD format (default: tomorrow)

        Returns:
            List of price dicts with:
            - timestamp: ISO timestamp
            - price_eur_kwh: Total consumer price including all taxes
            - market_price: Base market price
            - market_price_tax: Tax on market price
            - sourcing_markup: Supplier markup
            - energy_tax: Energy tax

            Returns None on failure.
        """
        # Default dates
        today = datetime.now(UTC).date()
        if start_date is None:
            start_date = today.isoformat()
        if end_date is None:
            end_date = (today + timedelta(days=1)).isoformat()

        cache_key = f"frank_{start_date}_{end_date}"

        # Check cache first
        if cache_key in _frank_cache:
            _LOGGER.debug("Frank prices from cache")
            return _frank_cache[cache_key]

        # Check circuit breaker
        if FrankEnergieClient._check_circuit_breaker():
            return None

        # GraphQL query
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
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.post(
                    FRANK_API_URL,
                    json={"query": query},
                    headers={"Content-Type": "application/json"}
                ) as resp:
                    if resp.status != 200:
                        _LOGGER.warning(f"Frank API returned HTTP {resp.status}")
                        FrankEnergieClient._record_failure()
                        return None

                    data = await resp.json()

                    if "errors" in data:
                        _LOGGER.error(f"Frank GraphQL errors: {data['errors']}")
                        FrankEnergieClient._record_failure()
                        return None

                    raw_prices = data.get("data", {}).get("marketPricesElectricity", [])

                    if not raw_prices:
                        _LOGGER.warning("Frank API returned empty prices")
                        FrankEnergieClient._record_failure()
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
                        total_price = (
                            float(p.get("marketPrice", 0) or 0) +
                            float(p.get("marketPriceTax", 0) or 0) +
                            float(p.get("sourcingMarkupPrice", 0) or 0) +
                            float(p.get("energyTaxPrice", 0) or 0)
                        )

                        prices.append({
                            "timestamp": ts_str,
                            "price_eur_kwh": total_price,
                            "market_price": p.get("marketPrice"),
                            "market_price_tax": p.get("marketPriceTax"),
                            "sourcing_markup": p.get("sourcingMarkupPrice"),
                            "energy_tax": p.get("energyTaxPrice"),
                        })

                    # Cache result
                    _frank_cache[cache_key] = prices
                    FrankEnergieClient._record_success()
                    _LOGGER.info(f"Frank Direct: fetched {len(prices)} prices")
                    return prices

        except Exception as e:
            _LOGGER.error(f"Frank API fetch failed: {e}")
            FrankEnergieClient._record_failure()
            return None

    @staticmethod
    async def get_prices_today() -> list[dict] | None:
        """
        Get today's prices.

        Returns:
            List of price dicts for today (24 hours), or None
        """
        today = datetime.now(UTC).date()
        return await FrankEnergieClient.get_prices(
            start_date=today.isoformat(),
            end_date=today.isoformat()
        )

    @staticmethod
    async def get_prices_tomorrow() -> list[dict] | None:
        """
        Get tomorrow's prices (available after ~14:00 CET).

        Returns:
            List of price dicts for tomorrow (24 hours), or None
        """
        tomorrow = (datetime.now(UTC).date() + timedelta(days=1))
        day_after = tomorrow + timedelta(days=1)
        return await FrankEnergieClient.get_prices(
            start_date=tomorrow.isoformat(),
            end_date=day_after.isoformat()
        )

    @staticmethod
    async def get_price_for_hour(hour: int, date: str = "today") -> float | None:
        """
        Get Frank price for specific hour.

        Args:
            hour: Hour 0-23
            date: "today" or "tomorrow"

        Returns:
            Price in EUR/kWh, or None
        """
        if date == "today":
            prices = await FrankEnergieClient.get_prices_today()
        else:
            prices = await FrankEnergieClient.get_prices_tomorrow()

        if not prices:
            return None

        for p in prices:
            ts = datetime.fromisoformat(p["timestamp"])
            if ts.hour == hour:
                return p.get("price_eur_kwh")

        return None

    @staticmethod
    async def health_check() -> tuple[bool, str]:
        """
        Check Frank API health.

        Returns:
            (is_healthy, message)
        """
        try:
            # Try to fetch a single day
            today = datetime.now(UTC).date()
            prices = await FrankEnergieClient.get_prices(
                start_date=today.isoformat(),
                end_date=today.isoformat()
            )

            if prices and len(prices) > 0:
                return (True, f"OK - {len(prices)} prices")
            else:
                return (False, "No prices returned")

        except Exception as e:
            return (False, str(e))
