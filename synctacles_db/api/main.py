"""
Energy Insights NL API
FastAPI application entry point
Environment-driven branding and configuration
"""
from datetime import datetime, timezone

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST
from starlette.responses import Response
import time

from synctacles_db.api.middleware import auth_middleware
from synctacles_db.api.endpoints import generation_mix, load, balance, now, prices, auth, signals
from synctacles_db.cache import api_cache
from config.settings import settings

# Create FastAPI app with branding from settings
app = FastAPI(
    title=settings.api_title,
    description=settings.api_description,
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc"
)


# Auth middleware (validates X-API-Key header)
# Prometheus metrics
http_requests_total = Counter(
    'http_requests_total',
    'Total HTTP requests',
    ['method', 'endpoint', 'status']
)

http_request_duration_seconds = Histogram(
    'http_request_duration_seconds',
    'HTTP request duration',
    ['method', 'endpoint']
)

@app.middleware("http")
async def metrics_middleware(request, call_next):
    start_time = time.time()
    response = await call_next(request)
    duration = time.time() - start_time
    
    http_requests_total.labels(
        method=request.method,
        endpoint=request.url.path,
        status=response.status_code
    ).inc()
    
    http_request_duration_seconds.labels(
        method=request.method,
        endpoint=request.url.path
    ).observe(duration)
    
    return response

# CORS (Home Assistant integration)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Dev mode - restrict in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Auth middleware - validates X-API-Key header
app.middleware("http")(auth_middleware)
# Health check
@app.get("/health")
async def health():
    """Health check endpoint for monitoring."""
    return {
        "status": "ok",
        "version": "1.0.0",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "service": settings.api_title,
        "brand": settings.brand_name
    }

# Prometheus metrics endpoint
@app.get("/metrics")
async def metrics():
    """Prometheus metrics endpoint"""
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)

# Cache management endpoints
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

# V1 endpoints
app.include_router(generation_mix.router, prefix="/api/v1", tags=["generation"])
app.include_router(load.router, prefix="/api/v1", tags=["load"])
app.include_router(balance.router, prefix="/api/v1", tags=["balance"])
app.include_router(now.router, prefix="/api/v1", tags=["Unified"])
app.include_router(prices.router, prefix="/api/v1", tags=["prices"])
app.include_router(signals.router, prefix="/api", tags=["signals"])

# Auth endpoints
app.include_router(auth.router, prefix="/auth", tags=["auth"])
