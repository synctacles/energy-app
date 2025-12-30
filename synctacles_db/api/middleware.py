from fastapi import Request, status
from fastapi.responses import JSONResponse
from synctacles_db import auth_service
from synctacles_db.api.dependencies import get_db

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
    MVP: /v1/* endpoints are free (no auth required)
    """
    path = request.url.path
    
    # Skip auth for exempt paths
    if path in EXEMPT_PATHS:
        return await call_next(request)
    
    # Skip auth for exempt prefixes (MVP free tier)
    if path.startswith(EXEMPT_PREFIXES):
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
