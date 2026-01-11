"""
Consumer Price Client.

HTTP client for Coefficient Engine API (91.99.150.36:8080).
Provides consumer prices and coefficient data for 6-tier fallback chain.

Brand-free implementation - no external brand names in code.
"""
import logging
from datetime import datetime, timezone
from typing import Optional, Dict, List, Tuple
from cachetools import TTLCache

import aiohttp

_LOGGER = logging.getLogger(__name__)

# Coefficient Engine server (internal, IP whitelisted)
COEFFICIENT_SERVER = "http://91.99.150.36:8080"

# Cache: consumer prices (5 min TTL), coefficients (1 hour TTL)
_consumer_cache = TTLCache(maxsize=10, ttl=300)
_coefficient_cache = TTLCache(maxsize=100, ttl=3600)

# Circuit breaker for coefficient server
_circuit_breaker = {
    "last_failure_time": None,
    "cooldown_minutes": 5,
    "failure_count": 0,
    "max_failures": 3,
}


class ConsumerPriceClient:
    """HTTP client for Coefficient Engine API."""

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

        now = datetime.now(timezone.utc)
        last_failure = _circuit_breaker["last_failure_time"]
        minutes_since = (now - last_failure).total_seconds() / 60

        if minutes_since < _circuit_breaker["cooldown_minutes"]:
            _LOGGER.debug(f"Circuit breaker OPEN ({int(minutes_since)} min since failure)")
            return True

        # Reset after cooldown
        _circuit_breaker["failure_count"] = 0
        _circuit_breaker["last_failure_time"] = None
        _LOGGER.info("Circuit breaker CLOSED (cooldown expired)")
        return False

    @staticmethod
    def _record_failure():
        """Record a failure for circuit breaker."""
        _circuit_breaker["failure_count"] += 1
        _circuit_breaker["last_failure_time"] = datetime.now(timezone.utc)
        _LOGGER.warning(f"Circuit breaker failure count: {_circuit_breaker['failure_count']}")

    @staticmethod
    def _record_success():
        """Reset circuit breaker on success."""
        if _circuit_breaker["failure_count"] > 0:
            _circuit_breaker["failure_count"] = 0
            _circuit_breaker["last_failure_time"] = None

    @staticmethod
    async def get_consumer_prices() -> Optional[Dict]:
        """
        Fetch consumer prices from Coefficient Engine proxy.

        Returns:
            {
                "timestamp": "2026-01-11T21:00:00Z",
                "source": "consumer-proxy",
                "prices_today": {
                    "Frank Energie": [{"hour": 0, "price": 0.183}, ...],
                    ...
                },
                "prices_tomorrow": {...} | None
            }
        """
        cache_key = "consumer_prices"

        # Check cache first
        if cache_key in _consumer_cache:
            _LOGGER.debug("Consumer prices from cache")
            return _consumer_cache[cache_key]

        # Check circuit breaker
        if ConsumerPriceClient._check_circuit_breaker():
            return None

        try:
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(f"{COEFFICIENT_SERVER}/internal/consumer/prices") as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        _consumer_cache[cache_key] = data
                        ConsumerPriceClient._record_success()
                        _LOGGER.info(f"Consumer prices fetched: {len(data.get('prices_today', {}))} providers")
                        return data
                    else:
                        _LOGGER.warning(f"Consumer prices returned HTTP {resp.status}")
                        ConsumerPriceClient._record_failure()
                        return None

        except Exception as e:
            _LOGGER.error(f"Consumer prices fetch failed: {e}")
            ConsumerPriceClient._record_failure()
            return None

    @staticmethod
    async def get_frank_prices(date: str = "today") -> Optional[List[Dict]]:
        """
        Get Frank Energie prices specifically.

        Args:
            date: "today" or "tomorrow"

        Returns:
            List of {"hour": 0, "price": 0.183} dicts, or None
        """
        data = await ConsumerPriceClient.get_consumer_prices()
        if not data:
            return None

        prices_key = "prices_today" if date == "today" else "prices_tomorrow"
        prices = data.get(prices_key)
        if not prices:
            return None

        return prices.get("Frank Energie")

    @staticmethod
    async def get_frank_price_for_hour(hour: int, date: str = "today") -> Optional[float]:
        """
        Get Frank Energie price for specific hour.

        Args:
            hour: Hour 0-23
            date: "today" or "tomorrow"

        Returns:
            Price in EUR/kWh, or None
        """
        frank_prices = await ConsumerPriceClient.get_frank_prices(date)
        if not frank_prices:
            return None

        for p in frank_prices:
            if p.get("hour") == hour:
                return p.get("price")

        return None

    @staticmethod
    async def get_coefficient(
        hour: Optional[int] = None,
        day_type: Optional[str] = None,
        month: Optional[int] = None
    ) -> Optional[Dict]:
        """
        Get coefficient from Coefficient Engine.

        Args:
            hour: Hour 0-23 (default: current)
            day_type: 'weekday' or 'weekend' (default: current)
            month: Month 1-12 (default: current)

        Returns:
            {
                "coefficient": 1.847,
                "confidence": 95,
                "sample_size": 45,
                "last_calibrated": "2026-01-11T12:00:00Z",
                "source": "lookup"
            }
        """
        now = datetime.now(timezone.utc)

        if hour is None:
            hour = now.hour
        if day_type is None:
            day_type = "weekend" if now.weekday() >= 5 else "weekday"
        if month is None:
            month = now.month

        cache_key = f"coef_{month}_{day_type}_{hour}"

        # Check cache first
        if cache_key in _coefficient_cache:
            _LOGGER.debug(f"Coefficient from cache: {cache_key}")
            return _coefficient_cache[cache_key]

        # Check circuit breaker
        if ConsumerPriceClient._check_circuit_breaker():
            return None

        try:
            params = {"month": month, "day_type": day_type, "hour": hour}
            timeout = aiohttp.ClientTimeout(total=5)

            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(f"{COEFFICIENT_SERVER}/coefficient", params=params) as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        _coefficient_cache[cache_key] = data
                        ConsumerPriceClient._record_success()
                        _LOGGER.debug(f"Coefficient fetched: {data.get('coefficient')}")
                        return data
                    else:
                        _LOGGER.warning(f"Coefficient returned HTTP {resp.status}")
                        ConsumerPriceClient._record_failure()
                        return None

        except Exception as e:
            _LOGGER.error(f"Coefficient fetch failed: {e}")
            ConsumerPriceClient._record_failure()
            return None

    @staticmethod
    async def get_coefficient_value(
        hour: Optional[int] = None,
        day_type: Optional[str] = None
    ) -> Optional[float]:
        """
        Get coefficient value only.

        Returns:
            Coefficient float, or None
        """
        result = await ConsumerPriceClient.get_coefficient(hour=hour, day_type=day_type)
        return result.get("coefficient") if result else None

    @staticmethod
    async def get_all_coefficients() -> Optional[Dict]:
        """
        Get all coefficients from lookup table.

        Useful for caching entire table locally.

        Returns:
            {
                "count": 576,
                "coefficients": {
                    "1_weekday_0": {"coefficient": 1.85, ...},
                    ...
                }
            }
        """
        cache_key = "all_coefficients"

        if cache_key in _coefficient_cache:
            return _coefficient_cache[cache_key]

        if ConsumerPriceClient._check_circuit_breaker():
            return None

        try:
            timeout = aiohttp.ClientTimeout(total=15)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(f"{COEFFICIENT_SERVER}/coefficient/all") as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        _coefficient_cache[cache_key] = data
                        ConsumerPriceClient._record_success()
                        _LOGGER.info(f"All coefficients fetched: {data.get('count')} entries")
                        return data
                    else:
                        ConsumerPriceClient._record_failure()
                        return None

        except Exception as e:
            _LOGGER.error(f"All coefficients fetch failed: {e}")
            ConsumerPriceClient._record_failure()
            return None

    @staticmethod
    async def health_check() -> Tuple[bool, str]:
        """
        Check Coefficient Engine health.

        Returns:
            (is_healthy, message)
        """
        try:
            timeout = aiohttp.ClientTimeout(total=5)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(f"{COEFFICIENT_SERVER}/health") as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        return (True, f"OK - {data.get('service')} v{data.get('version', '?')}")
                    else:
                        return (False, f"HTTP {resp.status}")

        except Exception as e:
            return (False, str(e))
