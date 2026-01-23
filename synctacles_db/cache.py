"""API Response Cache Manager"""
import logging
from typing import Any

from cachetools import TTLCache

logger = logging.getLogger(__name__)

class APICache:
    def __init__(self, maxsize: int = 100, default_ttl: int = 300):
        self._cache = TTLCache(maxsize=maxsize, ttl=default_ttl)
        self._default_ttl = default_ttl
        self._hits = 0
        self._misses = 0

    def get(self, key: str) -> dict[str, Any] | None:
        try:
            value = self._cache[key]
            self._hits += 1
            logger.debug(f"Cache HIT: {key}")
            return value
        except KeyError:
            self._misses += 1
            logger.debug(f"Cache MISS: {key}")
            return None

    def set(self, key: str, value: dict[str, Any], ttl: int | None = None):
        try:
            if ttl and ttl != self._default_ttl:
                temp_cache = TTLCache(maxsize=1, ttl=ttl)
                temp_cache[key] = value
                self._cache[key] = temp_cache[key]
            else:
                self._cache[key] = value
            logger.debug(f"Cache SET: {key} (TTL: {ttl or self._default_ttl}s)")
        except Exception as e:
            logger.error(f"Cache SET failed: {e}")

    def invalidate_pattern(self, pattern: str) -> int:
        to_delete = [k for k in list(self._cache.keys()) if k.startswith(pattern)]
        for key in to_delete:
            del self._cache[key]
        if to_delete:
            logger.info(f"Invalidated {len(to_delete)} keys: '{pattern}'")
        return len(to_delete)

    def clear(self):
        self._cache.clear()
        self._hits = 0
        self._misses = 0

    def stats(self) -> dict[str, Any]:
        total = self._hits + self._misses
        hit_rate = (self._hits / total * 100) if total > 0 else 0
        return {
            "size": len(self._cache),
            "maxsize": self._cache.maxsize,
            "hits": self._hits,
            "misses": self._misses,
            "hit_rate_pct": round(hit_rate, 2)
        }

# Singleton
api_cache = APICache(maxsize=100, default_ttl=300)
