from fastapi import Request, status
from fastapi.responses import JSONResponse
from datetime import datetime, timezone, timedelta
import time
import json

from synctacles_db import auth_service
from synctacles_db.api.dependencies import get_db
from synctacles_db.core.logging import get_logger
from config.settings import AUTH_REQUIRED, RATE_LIMIT_ENABLED

_LOGGER = get_logger(__name__)

# Paths that don't require authentication
EXEMPT_PATHS = {
    "/health",
    "/metrics",
    "/docs",
    "/redoc",
    "/openapi.json",
    "/auth/signup",
}

# Prefixes that don't require authentication (MVP free tier)
EXEMPT_PREFIXES = (
    "/api/v1/",
)


async def http_logging_middleware(request: Request, call_next):
    """
    Log HTTP requests and responses with timing.

    INFO level: method, path, status, duration
    DEBUG level: headers, query params, status (for non-2xx/3xx)
    """
    start_time = time.time()
    path = request.url.path
    method = request.method
    query_string = request.url.query if request.url.query else ""

    # Log request
    _LOGGER.debug(
        f"HTTP request: {method} {path}",
        extra={
            "method": method,
            "path": path,
            "query": query_string,
            "client": request.client.host if request.client else "unknown",
            "user_agent": request.headers.get("user-agent", "unknown"),
        }
    )

    try:
        response = await call_next(request)
    except Exception as e:
        elapsed = time.time() - start_time
        _LOGGER.error(
            f"HTTP request failed: {method} {path} - {type(e).__name__}: {e}",
            extra={
                "method": method,
                "path": path,
                "duration_ms": elapsed * 1000,
                "error": type(e).__name__,
            }
        )
        raise

    elapsed = time.time() - start_time
    status_code = response.status_code

    # Log response with appropriate level
    if 200 <= status_code < 400:
        # Success/redirect - INFO level
        _LOGGER.info(
            f"HTTP response: {method} {path} {status_code}",
            extra={
                "method": method,
                "path": path,
                "status": status_code,
                "duration_ms": elapsed * 1000,
            }
        )
    else:
        # Client/server error - WARNING level
        _LOGGER.warning(
            f"HTTP error: {method} {path} {status_code}",
            extra={
                "method": method,
                "path": path,
                "status": status_code,
                "duration_ms": elapsed * 1000,
            }
        )

    # Add timing header to response
    response.headers["X-Response-Time"] = f"{elapsed:.3f}"

    return response


async def auth_middleware(request: Request, call_next):
    """
    Validate API key on all requests except exempt paths.
    Support feature flag: AUTH_REQUIRED (default: false)
    Logs auth successes/failures at DEBUG/WARNING levels.
    """
    path = request.url.path

    # Skip auth for exempt paths
    if path in EXEMPT_PATHS:
        _LOGGER.debug(f"Auth exempt path: {path}")
        return await call_next(request)

    # Skip auth for exempt prefixes (MVP free tier)
    if path.startswith(EXEMPT_PREFIXES):
        _LOGGER.debug(f"Auth exempt prefix: {path}")
        return await call_next(request)

    # If AUTH_REQUIRED is disabled, allow all requests
    if not AUTH_REQUIRED:
        _LOGGER.debug("AUTH_REQUIRED disabled, allowing all requests")
        return await call_next(request)

    # Get API key
    api_key = request.headers.get("X-API-Key")
    if not api_key:
        _LOGGER.warning(f"Auth failed: missing X-API-Key header for {path}")
        return JSONResponse(
            status_code=status.HTTP_401_UNAUTHORIZED,
            content={"detail": "API key required. Include X-API-Key header."}
        )

    # Validate key
    try:
        db = next(get_db())
        user = auth_service.validate_api_key(db, api_key)
        if not user:
            db.close()
            _LOGGER.warning(f"Auth failed: invalid API key for {path}")
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Invalid API key"}
            )
        # Store user in request for downstream use
        request.state.user = user
        _LOGGER.debug(f"Auth success: user {user.id} for {path}")
        db.close()
    except Exception as e:
        _LOGGER.error(f"Auth error: {type(e).__name__}: {e} for {path}")
        return JSONResponse(
            status_code=status.HTTP_401_UNAUTHORIZED,
            content={"detail": "Authentication failed"}
        )

    # Continue to endpoint
    response = await call_next(request)
    return response


async def rate_limit_middleware(request: Request, call_next):
    """
    Rate limit users based on tier-specific daily limits.
    Support feature flag: RATE_LIMIT_ENABLED (default: false)

    Returns 429 if limit exceeded with rate limit headers.
    Logs limit checks and overages at WARNING level.
    """
    path = request.url.path

    # Skip rate limiting for exempt paths
    if path in EXEMPT_PATHS or path.startswith(EXEMPT_PREFIXES):
        _LOGGER.debug(f"Rate limiting exempt: {path}")
        return await call_next(request)

    # If RATE_LIMIT_ENABLED is disabled, skip rate limiting
    if not RATE_LIMIT_ENABLED:
        _LOGGER.debug("RATE_LIMIT_ENABLED disabled")
        return await call_next(request)

    # Check if user is in request state (set by auth middleware)
    user = getattr(request.state, "user", None)
    if not user:
        # No user context, allow request (will fail auth if AUTH_REQUIRED)
        _LOGGER.debug("No user context for rate limiting")
        return await call_next(request)

    # Check rate limit
    try:
        db = next(get_db())

        # Calculate remaining requests
        today_start = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)
        from sqlalchemy import func
        usage_count = db.query(func.count(auth_service.APIUsage.id)).filter(
            auth_service.APIUsage.user_id == user.id,
            auth_service.APIUsage.timestamp >= today_start
        ).scalar()

        remaining = max(0, user.rate_limit_daily - usage_count)
        reset_time = int((today_start + timedelta(days=1)).timestamp())

        # Check if limit exceeded
        if usage_count >= user.rate_limit_daily:
            db.close()
            _LOGGER.warning(
                f"Rate limit exceeded: user {user.id}, daily limit {user.rate_limit_daily}",
                extra={
                    "user_id": user.id,
                    "limit": user.rate_limit_daily,
                    "usage": usage_count,
                    "path": path,
                }
            )
            return JSONResponse(
                status_code=429,
                content={"detail": "Rate limit exceeded. Daily limit reset at midnight UTC."},
                headers={
                    "X-RateLimit-Limit": str(user.rate_limit_daily),
                    "X-RateLimit-Remaining": "0",
                    "X-RateLimit-Reset": str(reset_time),
                }
            )

        _LOGGER.debug(
            f"Rate limit check: user {user.id}, usage {usage_count}/{user.rate_limit_daily}",
            extra={
                "user_id": user.id,
                "usage": usage_count,
                "limit": user.rate_limit_daily,
                "remaining": remaining,
            }
        )
        db.close()
    except Exception as e:
        # On error, continue (don't break the API)
        _LOGGER.error(f"Rate limit check error: {type(e).__name__}: {e}")
        pass

    # Get response
    start_time = time.time()
    response = await call_next(request)

    # Log usage and add rate limit headers
    try:
        db = next(get_db())
        auth_service.log_api_usage(db, user, path, response.status_code)

        # Recalculate remaining after logging
        today_start = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)
        from sqlalchemy import func
        usage_count = db.query(func.count(auth_service.APIUsage.id)).filter(
            auth_service.APIUsage.user_id == user.id,
            auth_service.APIUsage.timestamp >= today_start
        ).scalar()

        remaining = max(0, user.rate_limit_daily - usage_count)
        reset_time = int((today_start + timedelta(days=1)).timestamp())

        response.headers["X-RateLimit-Limit"] = str(user.rate_limit_daily)
        response.headers["X-RateLimit-Remaining"] = str(remaining)
        response.headers["X-RateLimit-Reset"] = str(reset_time)

        _LOGGER.debug(
            f"API usage logged: user {user.id}, path {path}, status {response.status_code}",
            extra={
                "user_id": user.id,
                "path": path,
                "status": response.status_code,
                "remaining": remaining,
            }
        )

        db.close()
    except Exception as e:
        # On error, still return response (don't break the API)
        _LOGGER.error(f"Rate limit logging error: {type(e).__name__}: {e}")
        pass

    return response
