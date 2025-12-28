# DAG 2 — In-Memory Caching Implementation

**Project:** SYNCTACLES V1.0  
**Task:** Add TTLCache to API endpoints  
**Goal:** Reduce p95 latency from ~1.1s to <500ms  
**Estimated Time:** 3-4 hours

---

## Context

SYNCTACLES is an energy data aggregation platform for Home Assistant users. Currently all API endpoints query PostgreSQL directly, causing ~1.1s response times. We need in-memory caching to improve performance.

---

## Architecture

```
synctacles_db/
├── api/
│   ├── main.py              # FastAPI app
│   ├── dependencies.py      # DB session factory
│   └── endpoints/
│       ├── generation_mix.py
│       ├── load.py
│       ├── balance.py
│       └── prices.py
├── models.py                # SQLAlchemy ORM
└── cache.py                 # ✅ ALREADY CREATED (see below)
```

---

## Cache Module (Already Implemented)

**Location:** `synctacles_db/cache.py`

This file is **already created** and ready to use. It provides:

```python
"""API Response Cache Manager"""
from cachetools import TTLCache
from typing import Optional, Dict, Any
import logging

logger = logging.getLogger(__name__)

class APICache:
    """
    In-memory cache with TTL expiration.
    
    Features:
    - Configurable TTL per entry
    - Pattern-based invalidation (prefix matching)
    - Hit/miss statistics tracking
    - Graceful degradation on errors
    """
    
    def __init__(self, maxsize: int = 100, default_ttl: int = 300):
        self._cache = TTLCache(maxsize=maxsize, ttl=default_ttl)
        self._default_ttl = default_ttl
        self._hits = 0
        self._misses = 0
        
    def get(self, key: str) -> Optional[Dict[str, Any]]:
        """Get from cache, returns None on miss"""
        try:
            value = self._cache[key]
            self._hits += 1
            logger.debug(f"Cache HIT: {key}")
            return value
        except KeyError:
            self._misses += 1
            logger.debug(f"Cache MISS: {key}")
            return None
            
    def set(self, key: str, value: Dict[str, Any], ttl: Optional[int] = None):
        """Set value with optional custom TTL"""
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
        """Invalidate all keys matching prefix pattern"""
        to_delete = [k for k in list(self._cache.keys()) if k.startswith(pattern)]
        for key in to_delete:
            del self._cache[key]
        if to_delete:
            logger.info(f"Invalidated {len(to_delete)} keys: '{pattern}'")
        return len(to_delete)
            
    def clear(self):
        """Clear entire cache and reset stats"""
        self._cache.clear()
        self._hits = 0
        self._misses = 0
        
    def stats(self) -> Dict[str, Any]:
        """Get cache statistics"""
        total = self._hits + self._misses
        hit_rate = (self._hits / total * 100) if total > 0 else 0
        return {
            "size": len(self._cache),
            "maxsize": self._cache.maxsize,
            "hits": self._hits,
            "misses": self._misses,
            "hit_rate_pct": round(hit_rate, 2)
        }

# Global singleton instance
api_cache = APICache(maxsize=100, default_ttl=300)
```

**Usage:**
```python
from synctacles_db.cache import api_cache

# Check cache
cached = api_cache.get("my-key")
if cached:
    return cached

# On miss: query DB, then cache result
result = fetch_from_db()
api_cache.set("my-key", result, ttl=300)
return result
```

---

## Task 1: Integrate Cache in Endpoints (90 min)

### TTL Strategy

| Endpoint | Update Frequency | TTL | Rationale |
|----------|------------------|-----|-----------|
| `/api/v1/generation-mix` | 15 min | 300s (5 min) | Balance freshness vs load |
| `/api/v1/load` | 15 min | 300s (5 min) | Same as generation |
| `/api/v1/balance` | 5 min | 120s (2 min) | Most frequent updates |
| `/api/v1/prices` | Daily 13:00 UTC | 1800s (30 min) | Stable daily data |

---

### 1.1 Generation Mix Endpoint

**File:** `synctacles_db/api/endpoints/generation_mix.py`

**Changes:**

```python
from synctacles_db.cache import api_cache

@router.get("/generation-mix")
async def get_generation_mix(
    hours: int = 72,
    country: str = "NL",
    db: Session = Depends(get_db)
):
    # Build cache key
    cache_key = f"generation-mix:{country}:{hours}"
    
    # Check cache first
    cached = api_cache.get(cache_key)
    if cached:
        return cached
    
    # MISS: Execute existing DB query logic
    now = datetime.now(timezone.utc)
    start_time = now - timedelta(hours=hours)
    
    records = db.query(NormEntsoeA75).filter(
        NormEntsoeA75.country == country,
        NormEntsoeA75.timestamp >= start_time
    ).order_by(desc(NormEntsoeA75.timestamp)).all()
    
    # ... (keep existing response building logic)
    
    response = {
        "data": [...],
        "meta": {...}
    }
    
    # Cache the result
    api_cache.set(cache_key, response, ttl=300)  # 5 minutes
    
    return response
```

**Key Points:**
- Cache key includes `country` and `hours` parameters
- TTL = 300 seconds (5 minutes)
- Graceful: If cache fails, DB query still works

---

### 1.2 Load Endpoint

**File:** `synctacles_db/api/endpoints/load.py`

**Apply same pattern:**
- Cache key: `f"load:{country}:{hours}"`
- TTL: 300 seconds (5 minutes)
- Import: `from synctacles_db.cache import api_cache`

---

### 1.3 Balance Endpoint

**File:** `synctacles_db/api/endpoints/balance.py`

**Apply same pattern:**
- Cache key: `f"balance:{country}:{hours}"`
- TTL: 120 seconds (2 minutes, more frequent updates)

---

### 1.4 Prices Endpoint

**File:** `synctacles_db/api/endpoints/prices.py`

**Apply same pattern:**
- Cache key: `f"prices:{country}"`
- TTL: 1800 seconds (30 minutes, daily updates)

---

## Task 2: Add Cache Management Endpoints (30 min)

**File:** `synctacles_db/api/main.py`

**Add these endpoints:**

```python
from synctacles_db.cache import api_cache

@app.get("/cache/stats")
async def cache_stats():
    """
    Get cache statistics.
    
    Returns hit/miss counts, hit rate, and cache size.
    Note: In production, this should be admin-only.
    """
    return api_cache.stats()

@app.post("/cache/clear")
async def cache_clear():
    """
    Clear entire cache.
    
    Note: In production, this should be admin-only.
    """
    api_cache.clear()
    return {"message": "Cache cleared", "status": "success"}

@app.post("/cache/invalidate/{pattern}")
async def cache_invalidate(pattern: str):
    """
    Invalidate cache entries matching pattern (prefix).
    
    Examples:
    - /cache/invalidate/generation-mix (clears all generation-mix keys)
    - /cache/invalidate/load:NL (clears Dutch load data)
    
    Note: In production, this should be admin-only.
    """
    count = api_cache.invalidate_pattern(pattern)
    return {
        "invalidated": count,
        "pattern": pattern,
        "status": "success"
    }
```

---

## Task 3: Add Cache Headers (30 min)

Add `X-Cache: HIT|MISS` header to all cached endpoints for debugging.

**Pattern (apply to all 4 endpoints):**

```python
from fastapi import Response
import json

@router.get("/generation-mix")
async def get_generation_mix(...):
    cache_key = f"generation-mix:{country}:{hours}"
    
    # Check cache
    cached = api_cache.get(cache_key)
    if cached:
        response = Response(
            content=json.dumps(cached),
            media_type="application/json"
        )
        response.headers["X-Cache"] = "HIT"
        return response
    
    # DB query (existing logic)
    result = {...}
    
    # Cache result
    api_cache.set(cache_key, result, ttl=300)
    
    # Return with MISS header
    response = Response(
        content=json.dumps(result),
        media_type="application/json"
    )
    response.headers["X-Cache"] = "MISS"
    return response
```

**Alternative (simpler):**

If the Response wrapping is too complex, just add the header to metadata:

```python
response = {
    "data": [...],
    "meta": {
        ...,
        "cached": True/False  # Add this field
    }
}
```

---

## Testing Checklist

### 1. Cold Start Test
```bash
# First request (cache miss)
curl -i http://localhost:8000/api/v1/generation-mix

# Expected:
# - X-Cache: MISS (or meta.cached: false)
# - Response time: ~1100ms
```

### 2. Warm Cache Test
```bash
# Second request (cache hit)
curl -i http://localhost:8000/api/v1/generation-mix

# Expected:
# - X-Cache: HIT (or meta.cached: true)
# - Response time: <100ms
```

### 3. Cache Stats
```bash
curl http://localhost:8000/cache/stats

# Expected output:
{
  "size": 4,
  "maxsize": 100,
  "hits": 5,
  "misses": 4,
  "hit_rate_pct": 55.56
}
```

### 4. Cache Invalidation
```bash
# Invalidate generation-mix cache
curl -X POST http://localhost:8000/cache/invalidate/generation-mix

# Expected:
{
  "invalidated": 1,
  "pattern": "generation-mix",
  "status": "success"
}

# Next request should be MISS
curl -i http://localhost:8000/api/v1/generation-mix
# Expected: X-Cache: MISS
```

### 5. Cache Clear
```bash
# Clear all cache
curl -X POST http://localhost:8000/cache/clear

# Expected:
{
  "message": "Cache cleared",
  "status": "success"
}

# Stats should show 0 hits/misses
curl http://localhost:8000/cache/stats
```

### 6. TTL Expiration Test
```bash
# Request prices endpoint (30 min TTL)
curl http://localhost:8000/api/v1/prices

# Wait 31 minutes (or change TTL to 10s for testing)
sleep 1810

# Request again - should be MISS (expired)
curl -i http://localhost:8000/api/v1/prices
# Expected: X-Cache: MISS
```

---

## Performance Baseline

### Before Caching (Current State)
- Generation mix: p95 ~1100ms
- Load: p95 ~1200ms
- Balance: p95 ~1000ms

### After Caching (Target)
- Cold start: Same as before (~1100ms)
- Warm cache: p95 <100ms
- Overall p95: <500ms (mix of cold/warm)

---

## Exit Criteria

- [ ] All 4 endpoints have cache integration
- [ ] Cache stats endpoint returns valid data
- [ ] Cache invalidation works (pattern + clear)
- [ ] `X-Cache` header (or `meta.cached` field) present
- [ ] Cold start: ~1100ms (unchanged)
- [ ] Warm cache: <100ms (10x improvement)
- [ ] No errors in application logs
- [ ] Cache survives API restarts (in-memory, so starts empty - expected)

---

## Important Notes

### Graceful Degradation
If cache operations fail (unlikely), the DB query still executes. Users never see cache errors.

```python
cached = api_cache.get(cache_key)
if cached:
    return cached  # Fast path

# Slow path (cache miss or error) - always works
result = query_database()
api_cache.set(cache_key, result)  # Best effort
return result
```

### Cache Key Design
- Include all parameters that affect response: `{endpoint}:{country}:{hours}`
- Use consistent naming: lowercase, colon-separated
- Examples:
  - `generation-mix:NL:72`
  - `load:NL:48`
  - `balance:NL:24`
  - `prices:NL`

### TTL Selection
- Short TTL (2-5 min): Data updates frequently, balance freshness vs load
- Long TTL (30 min): Data updates once daily, reduce DB load
- Never cache longer than data update frequency

### Admin Endpoints
In production, `/cache/*` endpoints should require admin authentication. For V1, we leave them open for debugging.

---

## Dependencies

**Already installed:**
- `cachetools==5.3.2` (from F8-A fallback work)

**Imports needed:**
```python
from synctacles_db.cache import api_cache
from fastapi import Response
import json
```

---

## File Locations

**Create:**
- ✅ `synctacles_db/cache.py` (ALREADY CREATED)

**Modify:**
- `synctacles_db/api/endpoints/generation_mix.py`
- `synctacles_db/api/endpoints/load.py`
- `synctacles_db/api/endpoints/balance.py`
- `synctacles_db/api/endpoints/prices.py`
- `synctacles_db/api/main.py`

---

## Troubleshooting

### Cache not working
- Check import: `from synctacles_db.cache import api_cache`
- Check logs: `logger.debug` messages show HIT/MISS
- Verify TTL hasn't expired

### Performance not improved
- Confirm `X-Cache: HIT` header (or `meta.cached: true`)
- Check cache stats: hit rate should be >50% after warmup
- Measure with `time curl ...`

### Cache growing too large
- Default maxsize=100 should be sufficient
- Oldest entries auto-evicted (LRU)
- Manual clear: `curl -X POST http://localhost:8000/cache/clear`

---

## Next Steps (After Completion)

1. **Deploy to production** (deploy.sh)
2. **Monitor cache stats** for 24h
3. **Tune TTL values** based on hit rate
4. **DAG 3: Unified `/api/v1/now` endpoint** (combines all data in one call)

---

**Questions?** Check `synctacles_db/cache.py` source code for implementation details.

**Start with Task 1, test each endpoint, then Task 2+3.**
