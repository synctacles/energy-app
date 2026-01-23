"""
Consumer Price Client.

DEPRECATED: This module is deprecated as of KISS Migration (2026-01-17).
The coefficient server (91.99.150.36) is being decommissioned.

Use instead:
- synctacles_db.clients.frank_energie_client.FrankEnergieClient (for consumer prices)
- synctacles_db.clients.easyenergy_client.EasyEnergyClient (for wholesale prices)
- synctacles_db.config.static_offsets (for wholesale → consumer conversion)

HTTP client for Coefficient Engine API (91.99.150.36:8080).
Provides consumer prices and coefficient data for 6-tier fallback chain.

Brand-free implementation - no external brand names in code.
"""
import asyncio
import json
import logging
import warnings
from datetime import UTC, datetime

import aiohttp

warnings.warn(
    "ConsumerPriceClient is deprecated. Use FrankEnergieClient, EasyEnergyClient, "
    "or static_offsets instead. Coefficient server will be decommissioned.",
    DeprecationWarning,
    stacklevel=2
)
from cachetools import TTLCache

_LOGGER = logging.getLogger(__name__)

# Coefficient Engine server (internal, IP whitelisted)
COEFFICIENT_SERVER = "http://91.99.150.36:8080"

# Default price model constants (Frank Energie calibrated, January 2026)
# Updated 2026-01-17: Frank uses wholesale passthrough pricing (slope=1.0, intercept=0.0)
# Validation: 99.9977% accuracy on Jan 10-16 data (23 hours tested)
DEFAULT_SLOPE = 1.0
DEFAULT_INTERCEPT = 0.0  # EUR/kWh fixed costs

# Bias correction factor (updated 2026-01-17 after fixing coefficient model)
# With correct slope/intercept (1.0/0.0), no bias correction needed
# Accuracy: 99.9977% without bias correction, 93% with 0.93 correction
BIAS_CORRECTION = 1.0

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

        now = datetime.now(UTC)
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
        _circuit_breaker["last_failure_time"] = datetime.now(UTC)
        _LOGGER.warning(f"Circuit breaker failure count: {_circuit_breaker['failure_count']}")

    @staticmethod
    def _record_success():
        """Reset circuit breaker on success."""
        if _circuit_breaker["failure_count"] > 0:
            _circuit_breaker["failure_count"] = 0
            _circuit_breaker["last_failure_time"] = None

    @staticmethod
    async def get_consumer_prices() -> dict | None:
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

        except aiohttp.ClientError as e:
            _LOGGER.error(f"Consumer prices network error: {e}")
            ConsumerPriceClient._record_failure()
            return None
        except asyncio.TimeoutError:
            _LOGGER.error("Consumer prices fetch timeout")
            ConsumerPriceClient._record_failure()
            return None
        except (json.JSONDecodeError, aiohttp.ContentTypeError) as e:
            _LOGGER.error(f"Consumer prices invalid response: {e}")
            ConsumerPriceClient._record_failure()
            return None
        except (KeyError, ValueError, TypeError) as e:
            _LOGGER.error(f"Consumer prices data error: {e}")
            ConsumerPriceClient._record_failure()
            return None

    @staticmethod
    async def get_frank_prices(date: str = "today") -> list[dict] | None:
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
    async def get_frank_price_for_hour(hour: int, date: str = "today") -> float | None:
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
    async def get_price_model(
        hour: int | None = None,
        day_type: str | None = None,
        month: int | None = None,
        country: str = "NL"
    ) -> dict | None:
        """
        Get slope + intercept model from Coefficient Engine.

        Linear regression model: consumer = wholesale × slope + intercept

        Args:
            hour: Hour 0-23 (default: current)
            day_type: 'weekday' or 'weekend' (default: current)
            month: Month 1-12 (default: current)
            country: Country code (default: NL)

        Returns:
            {
                "slope": 1.27,
                "intercept": 0.147,
                "confidence": 94,
                "sample_size": 75,
                "last_calibrated": "2026-01-12T15:09:42Z",
                "source": "lookup"
            }
        """
        now = datetime.now(UTC)

        if hour is None:
            hour = now.hour
        if day_type is None:
            day_type = "weekend" if now.weekday() >= 5 else "weekday"
        if month is None:
            month = now.month

        cache_key = f"model_{country}_{month}_{day_type}_{hour}"

        # Check cache first
        if cache_key in _coefficient_cache:
            _LOGGER.debug(f"Price model from cache: {cache_key}")
            return _coefficient_cache[cache_key]

        # Check circuit breaker
        if ConsumerPriceClient._check_circuit_breaker():
            return None

        try:
            params = {"country": country, "month": month, "day_type": day_type, "hour": hour}
            timeout = aiohttp.ClientTimeout(total=5)

            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(f"{COEFFICIENT_SERVER}/coefficient", params=params) as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        _coefficient_cache[cache_key] = data
                        ConsumerPriceClient._record_success()
                        _LOGGER.debug(f"Price model fetched: slope={data.get('slope')}, intercept={data.get('intercept')}")
                        return data
                    else:
                        _LOGGER.warning(f"Price model returned HTTP {resp.status}")
                        ConsumerPriceClient._record_failure()
                        return None

        except aiohttp.ClientError as e:
            _LOGGER.error(f"Price model network error: {e}")
            ConsumerPriceClient._record_failure()
            return None
        except asyncio.TimeoutError:
            _LOGGER.error("Price model fetch timeout")
            ConsumerPriceClient._record_failure()
            return None
        except (json.JSONDecodeError, aiohttp.ContentTypeError) as e:
            _LOGGER.error(f"Price model invalid response: {e}")
            ConsumerPriceClient._record_failure()
            return None
        except (KeyError, ValueError, TypeError) as e:
            _LOGGER.error(f"Price model data error: {e}")
            ConsumerPriceClient._record_failure()
            return None

    # Legacy method - keep for backward compatibility
    @staticmethod
    async def get_coefficient(
        hour: int | None = None,
        day_type: str | None = None,
        month: int | None = None
    ) -> dict | None:
        """
        Legacy method - wraps get_price_model for backward compatibility.

        Returns dict with slope, intercept, and legacy 'coefficient' field.
        """
        result = await ConsumerPriceClient.get_price_model(hour=hour, day_type=day_type, month=month)
        if result:
            # Add legacy 'coefficient' field for backward compatibility
            result["coefficient"] = result.get("slope", 1.0)
        return result

    @staticmethod
    async def get_coefficient_value(
        hour: int | None = None,
        day_type: str | None = None
    ) -> float | None:
        """
        Legacy method - get slope value only.

        Returns:
            Slope float, or None
        """
        result = await ConsumerPriceClient.get_price_model(hour=hour, day_type=day_type)
        return result.get("slope") if result else None

    @staticmethod
    def calculate_consumer_price(
        wholesale_eur_kwh: float,
        slope: float,
        intercept: float,
        apply_bias_correction: bool = True
    ) -> float:
        """
        Calculate consumer price from wholesale using linear model.

        Formula: consumer = (wholesale × slope + intercept) × bias_correction

        Args:
            wholesale_eur_kwh: ENTSO-E price in EUR/kWh
            slope: Multiplier (~1.27)
            intercept: Fixed costs (~€0.147)
            apply_bias_correction: Apply 0.93 correction factor (default: True)

        Returns:
            Consumer price in EUR/kWh
        """
        raw_price = wholesale_eur_kwh * slope + intercept
        if apply_bias_correction:
            return raw_price * BIAS_CORRECTION
        return raw_price

    @staticmethod
    async def get_all_coefficients() -> dict | None:
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

        except aiohttp.ClientError as e:
            _LOGGER.error(f"All coefficients network error: {e}")
            ConsumerPriceClient._record_failure()
            return None
        except asyncio.TimeoutError:
            _LOGGER.error("All coefficients fetch timeout")
            ConsumerPriceClient._record_failure()
            return None
        except (json.JSONDecodeError, aiohttp.ContentTypeError) as e:
            _LOGGER.error(f"All coefficients invalid response: {e}")
            ConsumerPriceClient._record_failure()
            return None
        except (KeyError, ValueError, TypeError) as e:
            _LOGGER.error(f"All coefficients data error: {e}")
            ConsumerPriceClient._record_failure()
            return None

    @staticmethod
    async def health_check() -> tuple[bool, str]:
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

        except (aiohttp.ClientError, asyncio.TimeoutError) as e:
            return (False, f"Network error: {e}")
        except (ValueError, KeyError, TypeError) as e:
            return (False, f"Data error: {e}")
