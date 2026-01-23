"""Fallback Manager - Orchestrates primary + fallback + cache"""

import logging
from datetime import UTC, datetime
from typing import Literal

from cachetools import TTLCache

from synctacles_db.fallback import energy_charts

logger = logging.getLogger(__name__)

QualityStatus = Literal["OK", "FALLBACK", "CACHED", "STALE", "NO_DATA"]

# In-memory cache (5 min TTL, 100 entries)
_generation_cache = TTLCache(maxsize=100, ttl=300)


async def get_generation_fallback(
    primary_data: list[dict] | None,
    country: str = "nl"
) -> tuple[list[dict] | None, QualityStatus]:
    """
    Returns (data, quality_status)
    
    Fallback chain:
    1. primary_data (if not None) → OK
    2. Energy-Charts → FALLBACK
    3. Cache (< 1h old) → CACHED/STALE
    4. None → NO_DATA
    """

    # Primary source OK
    if primary_data:
        _generation_cache[country] = (primary_data, datetime.now(UTC))
        return primary_data, "OK"

    # Try fallback source
    logger.warning("Primary source failed, trying Energy-Charts fallback")
    fallback_data = await energy_charts.fetch_generation_mix(country)

    if fallback_data:
        _generation_cache[country] = (fallback_data, datetime.now(UTC))
        return fallback_data, "FALLBACK"

    # Try cache
    cached = _generation_cache.get(country)
    if cached:
        data, cached_at = cached
        age = (datetime.now(UTC) - cached_at).total_seconds()

        if age < 3600:  # < 1h
            status = "CACHED" if age < 900 else "STALE"  # < 15 min = CACHED
            logger.warning(f"Serving cached data ({int(age)}s old, {status})")
            return data, status

    # No data available
    logger.error("No data from primary, fallback, or cache")
    return None, "NO_DATA"
