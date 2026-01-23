"""
Deprecated Endpoints - HTTP 410 Gone

As of 2026-01-11, SYNCTACLES focuses exclusively on Energy Action.
Grid data endpoints (generation, load, signals) have been discontinued.

See: PVA_ENERGY_ACTION_FOCUS.md - Phase 2: Soft Delete
"""
from fastapi import APIRouter
from starlette.responses import JSONResponse

from config.settings import GITHUB_ACCOUNT

router = APIRouter(prefix="", tags=["deprecated"])

# Build documentation URL from ENV (brand-free)
_DOC_URL = f"https://github.com/{GITHUB_ACCOUNT}/synctacles-api#energy-action-focus"

DISCONTINUED_MESSAGE = {
    "error": "Gone",
    "message": "This endpoint has been discontinued. SYNCTACLES now focuses exclusively on Energy Action.",
    "documentation": _DOC_URL,
    "migration": {
        "energy_action": "/api/v1/energy-action",
        "prices": "/api/v1/prices/today",
        "rationale": "Energy Action provides the actionable insights you need without the complexity of raw grid data."
    }
}


@router.get("/generation-mix")
async def deprecated_generation_mix():
    """
    Generation mix endpoint - DISCONTINUED

    This endpoint has been removed as part of the Energy Action Focus initiative.
    Use /api/v1/energy-action for actionable energy recommendations.
    """
    return JSONResponse(
        status_code=410,
        content={
            **DISCONTINUED_MESSAGE,
            "endpoint": "/api/v1/generation-mix",
            "replacement": "/api/v1/energy-action"
        }
    )


@router.get("/load")
async def deprecated_load():
    """
    Load endpoint - DISCONTINUED

    This endpoint has been removed as part of the Energy Action Focus initiative.
    Use /api/v1/energy-action for actionable energy recommendations.
    """
    return JSONResponse(
        status_code=410,
        content={
            **DISCONTINUED_MESSAGE,
            "endpoint": "/api/v1/load",
            "replacement": "/api/v1/energy-action"
        }
    )


# Signals router for /api/v1/signals path
signals_router = APIRouter(prefix="/v1", tags=["deprecated"])


@signals_router.get("/signals")
async def deprecated_signals():
    """
    Signals endpoint - DISCONTINUED

    This endpoint has been removed as part of the Energy Action Focus initiative.
    Use /api/v1/energy-action for actionable energy recommendations.
    """
    return JSONResponse(
        status_code=410,
        content={
            **DISCONTINUED_MESSAGE,
            "endpoint": "/api/v1/signals",
            "replacement": "/api/v1/energy-action",
            "note": "Energy Action provides is_cheap, cheapest_hour, and allow_automation signals."
        }
    )


@router.get("/now")
async def deprecated_now():
    """
    Unified data endpoint - DISCONTINUED (Phase 3: 2026-01-11)

    This endpoint combined generation, load, and balance data.
    Use /api/v1/energy-action for actionable energy recommendations.
    Use /api/v1/prices/today for price data.
    """
    return JSONResponse(
        status_code=410,
        content={
            **DISCONTINUED_MESSAGE,
            "endpoint": "/api/v1/now",
            "replacement": "/api/v1/energy-action",
            "note": "The /now endpoint combined generation/load/balance which are no longer collected."
        }
    )
