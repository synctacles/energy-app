"""In-memory caching for API responses."""
from cachetools import TTLCache
from functools import wraps
import hashlib
import json

# Cache instances with TTL (time-to-live in seconds)
generation_cache = TTLCache(maxsize=100, ttl=300)   # 5 min
load_cache = TTLCache(maxsize=100, ttl=300)         # 5 min
prices_cache = TTLCache(maxsize=100, ttl=3600)      # 1 hour
balance_cache = TTLCache(maxsize=100, ttl=60)       # 1 min
signals_cache = TTLCache(maxsize=100, ttl=60)       # 1 min


def cache_key(*args, **kwargs):
    """Generate cache key from arguments."""
    key_data = json.dumps({"args": args, "kwargs": kwargs}, sort_keys=True, default=str)
    return hashlib.md5(key_data.encode()).hexdigest()


def cached(cache_instance):
    """Decorator for caching function results."""
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            key = cache_key(func.__name__, *args, **kwargs)
            
            if key in cache_instance:
                return cache_instance[key]
            
            result = await func(*args, **kwargs)
            cache_instance[key] = result
            return result
        return wrapper
    return decorator


def clear_all_caches():
    """Clear all caches (useful after data import)."""
    generation_cache.clear()
    load_cache.clear()
    prices_cache.clear()
    balance_cache.clear()
    signals_cache.clear()
