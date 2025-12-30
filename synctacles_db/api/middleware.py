from fastapi import Request, status
from fastapi.responses import JSONResponse
from datetime import datetime, timezone, timedelta
import time

from synctacles_db import auth_service
from synctacles_db.api.dependencies import get_db
from config.settings import AUTH_REQUIRED, RATE_LIMIT_ENABLED

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


async def auth_middleware(request: Request, call_next):
    """
    Validate API key on all requests except exempt paths.
    Support feature flag: AUTH_REQUIRED (default: false)
    """
    path = request.url.path

    # Skip auth for exempt paths
    if path in EXEMPT_PATHS:
        return await call_next(request)

    # Skip auth for exempt prefixes (MVP free tier)
    if path.startswith(EXEMPT_PREFIXES):
        return await call_next(request)

    # If AUTH_REQUIRED is disabled, allow all requests
    if not AUTH_REQUIRED:
        return await call_next(request)

    # Get API key
    api_key = request.headers.get("X-API-Key")
    if not api_key:
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
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Invalid API key"}
            )
        # Store user in request for downstream use
        request.state.user = user
        db.close()
    except Exception:
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
    """
    path = request.url.path

    # Skip rate limiting for exempt paths
    if path in EXEMPT_PATHS or path.startswith(EXEMPT_PREFIXES):
        return await call_next(request)

    # If RATE_LIMIT_ENABLED is disabled, skip rate limiting
    if not RATE_LIMIT_ENABLED:
        return await call_next(request)

    # Check if user is in request state (set by auth middleware)
    user = getattr(request.state, "user", None)
    if not user:
        # No user context, allow request (will fail auth if AUTH_REQUIRED)
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
            return JSONResponse(
                status_code=429,
                content={"detail": "Rate limit exceeded. Daily limit reset at midnight UTC."},
                headers={
                    "X-RateLimit-Limit": str(user.rate_limit_daily),
                    "X-RateLimit-Remaining": "0",
                    "X-RateLimit-Reset": str(reset_time),
                }
            )

        db.close()
    except Exception:
        # On error, continue (don't break the API)
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

        db.close()
    except Exception:
        # On error, still return response (don't break the API)
        pass

    return response
