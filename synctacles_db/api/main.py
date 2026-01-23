"""
Energy Data API
FastAPI application entry point
Environment-driven branding and configuration
"""
import time
from datetime import UTC, datetime

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import CONTENT_TYPE_LATEST, Counter, Histogram, generate_latest
from starlette.responses import Response

from config.settings import settings
from synctacles_db.api.endpoints import auth, balance, energy_action, prices, windows
from synctacles_db.api.endpoints.deprecated import router as deprecated_router
from synctacles_db.api.endpoints.deprecated import (
    signals_router as deprecated_signals_router,
)
from synctacles_db.api.middleware import (
    auth_middleware,
    http_logging_middleware,
    rate_limit_middleware,
)
from synctacles_db.api.routes.pipeline import router as pipeline_router
from synctacles_db.cache import api_cache

# === LOGGING INITIALIZATION ===
from synctacles_db.core.logging import get_logger, setup_logging

setup_logging()
_LOGGER = get_logger(__name__)
_LOGGER.info("API initialization starting")
# === END LOGGING ===

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

# CORS Configuration (environment-driven for multi-deployment support)
# Development: CORS_ORIGINS defaults to ["*"]
# Production: Set CORS_ORIGINS env var to restrict (e.g., "https://homeassistant.local,https://example.com")
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    allow_headers=["Content-Type", "X-API-Key", "Authorization"],
    max_age=3600,  # Cache CORS preflight for 1 hour
)


# Middleware order matters: HTTP logging first, then auth, then rate limit
# 1. HTTP Logging: logs all requests/responses with timing
# 2. Auth: validates API keys
# 3. Rate Limit: enforces daily limits based on user context
app.middleware("http")(http_logging_middleware)
app.middleware("http")(auth_middleware)
app.middleware("http")(rate_limit_middleware)
# Health check
@app.get("/health")
async def health():
    """Health check endpoint for monitoring."""
    return {
        "status": "ok",
        "version": "1.0.0",
        "timestamp": datetime.now(UTC).isoformat(),
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

# V1 endpoints - Active
app.include_router(balance.router, prefix="/api/v1", tags=["balance"])
# Phase 3: /now endpoint moved to deprecated (2026-01-11)
app.include_router(prices.router, prefix="/api/v1", tags=["prices"])

# V1 endpoints - Deprecated (410 Gone)
# Fase 2: Soft Delete - Grid endpoints discontinued (2026-01-11)
app.include_router(deprecated_router, prefix="/api/v1", tags=["deprecated"])
app.include_router(deprecated_signals_router, prefix="/api", tags=["deprecated"])

# Energy Action (core endpoint with quality indicator)
app.include_router(energy_action.router, prefix="/api", tags=["energy-action"])

# Best Window Finder & Tomorrow Preview (wow features)
app.include_router(windows.router, prefix="/api", tags=["windows"])

# Pipeline monitoring
app.include_router(pipeline_router)

# Auth endpoints
app.include_router(auth.router, prefix="/auth", tags=["auth"])
