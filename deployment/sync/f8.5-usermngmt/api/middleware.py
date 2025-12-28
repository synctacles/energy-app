"""API Authentication & Rate Limiting Middleware"""

from fastapi import Request, HTTPException
from fastapi.responses import JSONResponse
from sqlalchemy.orm import Session
import logging

from synctacles_db import auth_service
from synctacles_db.api.dependencies import get_db

logger = logging.getLogger(__name__)


async def auth_middleware(request: Request, call_next):
    """
    Validate API key and enforce rate limits
    
    Exemptions:
    - /health
    - /metrics
    - /docs
    - /auth/signup
    - /auth/admin/* (uses X-Admin-Key instead)
    """
    
    # Exempt endpoints
    exempt_paths = [
        "/health", 
        "/metrics", 
        "/docs", 
        "/openapi.json", 
        "/auth/signup",
        "/auth/admin/"
    ]
    if any(request.url.path.startswith(path) for path in exempt_paths):
        return await call_next(request)
    
    # Get API key from header
    api_key = request.headers.get("X-API-Key")
    
    if not api_key:
        return JSONResponse(
            status_code=401,
            content={
                "detail": "Missing X-API-Key header",
                "hint": "Sign up at /auth/signup to get your API key"
            }
        )
    
    # Validate API key
    db = next(get_db())
    try:
        user = auth_service.validate_api_key(db, api_key)
        
        if not user:
            return JSONResponse(
                status_code=401,
                content={"detail": "Invalid API key"}
            )
        
        # Check rate limit
        if not auth_service.check_rate_limit(db, user):
            stats = auth_service.get_user_stats(db, user)
            return JSONResponse(
                status_code=429,
                content={
                    "detail": "Rate limit exceeded",
                    "limit": user.rate_limit_daily,
                    "used": stats["usage_today"],
                    "reset_at": "00:00 UTC"
                }
            )
        
        # Log usage (fire-and-forget)
        try:
            auth_service.log_api_usage(db, user, request.url.path, 200)
        except Exception as e:
            logger.error(f"Failed to log API usage: {e}")
        
        # Attach user to request state
        request.state.user = user
        
        response = await call_next(request)
        return response
        
    finally:
        db.close()
