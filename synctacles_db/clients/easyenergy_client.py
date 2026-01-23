"""
EasyEnergy API Client.

Direct HTTP client for EasyEnergy price API.
Returns APX wholesale prices (same as EPEX spot prices).

API: https://mijn.easyenergy.com/nl/api/tariff/getapxtariffs

Features:
- Direct API access (no authentication required)
- Circuit breaker pattern for resilience
- Uses central api_cache for consistency
- Returns wholesale prices in EUR/kWh and EUR/MWh
"""
import asyncio
import json
import logging
from datetime import UTC, datetime, timedelta

import aiohttp

# Use central cache singleton for consistency
from synctacles_db.cache import api_cache

_LOGGER = logging.getLogger(__name__)

# EasyEnergy API endpoint
EASYENERGY_API_URL = "https://mijn.easyenergy.com/nl/api/tariff"

# Circuit breaker for EasyEnergy API
_circuit_breaker = {
    "last_failure_time": None,
    "cooldown_minutes": 5,
    "failure_count": 0,
    "max_failures": 3,
}


class EasyEnergyClient:
    """Direct HTTP client for EasyEnergy price API."""

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
            _LOGGER.debug(f"EasyEnergy circuit breaker OPEN ({int(minutes_since)} min since failure)")
            return True

        # Reset after cooldown
        _circuit_breaker["failure_count"] = 0
        _circuit_breaker["last_failure_time"] = None
        _LOGGER.info("EasyEnergy circuit breaker CLOSED (cooldown expired)")
        return False

    @staticmethod
    def _record_failure():
        """Record a failure for circuit breaker."""
        _circuit_breaker["failure_count"] += 1
        _circuit_breaker["last_failure_time"] = datetime.now(UTC)
        _LOGGER.warning(f"EasyEnergy circuit breaker failure count: {_circuit_breaker['failure_count']}")

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
        Fetch electricity prices from EasyEnergy API.

        Args:
            start_date: Start date in YYYY-MM-DD format (default: today)
            end_date: End date in YYYY-MM-DD format (default: tomorrow)

        Returns:
            List of price dicts with:
            - timestamp: ISO timestamp
            - price_eur_kwh: Wholesale price in EUR/kWh
            - price_eur_mwh: Wholesale price in EUR/MWh

            Returns None on failure.
        """
        # Default dates
        today = datetime.now(UTC).date()
        if start_date is None:
            start_date = today.isoformat()
        if end_date is None:
            end_date = (today + timedelta(days=1)).isoformat()

        cache_key = f"easyenergy_{start_date}_{end_date}"

        # Check cache first (using central api_cache)
        cached = api_cache.get(cache_key)
        if cached is not None:
            _LOGGER.debug("EasyEnergy prices from cache")
            return cached

        # Check circuit breaker
        if EasyEnergyClient._check_circuit_breaker():
            return None

        try:
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                # EasyEnergy API expects startTimestamp and endTimestamp
                # The API returns data for the period [start, end)
                url = f"{EASYENERGY_API_URL}/getapxtariffs"
                params = {
                    "startTimestamp": f"{start_date}T00:00:00.000Z",
                    "endTimestamp": f"{end_date}T00:00:00.000Z"
                }

                async with session.get(
                    url,
                    params=params,
                    headers={
                        "Accept": "application/json",
                        "User-Agent": "Synctacles/1.0"
                    }
                ) as resp:
                    if resp.status != 200:
                        _LOGGER.warning(f"EasyEnergy API returned HTTP {resp.status}")
                        EasyEnergyClient._record_failure()
                        return None

                    data = await resp.json()

                    if not data or len(data) == 0:
                        _LOGGER.warning("EasyEnergy API returned empty data")
                        EasyEnergyClient._record_failure()
                        return None

                    # Transform to standard format
                    prices = []
                    for record in data:
                        # EasyEnergy returns TariffReturn in EUR/MWh
                        price_eur_mwh = float(record.get("TariffReturn", 0))
                        price_eur_kwh = price_eur_mwh / 1000.0

                        # Parse timestamp
                        ts_str = record.get("Timestamp", "")
                        # Normalize to ISO format with timezone
                        if ts_str:
                            # Handle various timestamp formats
                            if "+" not in ts_str and "Z" not in ts_str:
                                ts_str = ts_str + "+00:00"
                            elif ts_str.endswith("Z"):
                                ts_str = ts_str[:-1] + "+00:00"

                        prices.append({
                            "timestamp": ts_str,
                            "price_eur_kwh": price_eur_kwh,
                            "price_eur_mwh": price_eur_mwh,
                        })

                    # Cache result (using central api_cache, 5 min TTL)
                    api_cache.set(cache_key, prices, ttl=300)
                    EasyEnergyClient._record_success()
                    _LOGGER.info(f"EasyEnergy Direct: fetched {len(prices)} prices")
                    return prices

        except aiohttp.ClientError as e:
            _LOGGER.error(f"EasyEnergy API network error: {e}")
            EasyEnergyClient._record_failure()
            return None
        except asyncio.TimeoutError:
            _LOGGER.error("EasyEnergy API timeout")
            EasyEnergyClient._record_failure()
            return None
        except (json.JSONDecodeError, aiohttp.ContentTypeError) as e:
            _LOGGER.error(f"EasyEnergy API invalid response: {e}")
            EasyEnergyClient._record_failure()
            return None
        except (KeyError, ValueError, TypeError) as e:
            _LOGGER.error(f"EasyEnergy API data error: {e}")
            EasyEnergyClient._record_failure()
            return None

    @staticmethod
    async def get_prices_today() -> list[dict] | None:
        """
        Get today's prices.

        Returns:
            List of price dicts for today (24 hours), or None
        """
        today = datetime.now(UTC).date()
        tomorrow = today + timedelta(days=1)
        return await EasyEnergyClient.get_prices(
            start_date=today.isoformat(),
            end_date=tomorrow.isoformat()
        )

    @staticmethod
    async def get_prices_tomorrow() -> list[dict] | None:
        """
        Get tomorrow's prices (available after ~15:00 CET).

        Returns:
            List of price dicts for tomorrow (24 hours), or None
        """
        tomorrow = datetime.now(UTC).date() + timedelta(days=1)
        day_after = tomorrow + timedelta(days=1)
        return await EasyEnergyClient.get_prices(
            start_date=tomorrow.isoformat(),
            end_date=day_after.isoformat()
        )

    @staticmethod
    async def get_price_for_hour(hour: int, date: str = "today") -> float | None:
        """
        Get EasyEnergy price for specific hour.

        Args:
            hour: Hour 0-23
            date: "today" or "tomorrow"

        Returns:
            Price in EUR/kWh, or None
        """
        if date == "today":
            prices = await EasyEnergyClient.get_prices_today()
        else:
            prices = await EasyEnergyClient.get_prices_tomorrow()

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
        Check EasyEnergy API health.

        Returns:
            (is_healthy, message)
        """
        try:
            # Try to fetch a single day
            today = datetime.now(UTC).date()
            tomorrow = today + timedelta(days=1)
            prices = await EasyEnergyClient.get_prices(
                start_date=today.isoformat(),
                end_date=tomorrow.isoformat()
            )

            if prices and len(prices) > 0:
                return (True, f"OK - {len(prices)} prices")
            else:
                return (False, "No prices returned")

        except (aiohttp.ClientError, asyncio.TimeoutError) as e:
            return (False, f"Network error: {e}")
        except (ValueError, KeyError, TypeError) as e:
            return (False, f"Data error: {e}")
