"""
Unified data endpoint - /api/v1/now
"""

from fastapi import APIRouter, Depends
from fastapi.responses import Response
from sqlalchemy.orm import Session

from synctacles_db.api.dependencies import get_db
from synctacles_db.unified_service import get_unified_snapshot
from synctacles_db.cache import api_cache

router = APIRouter()


@router.get("/now")
def get_now(
    country: str = "NL",
    db: Session = Depends(get_db)
):
    """
    Get unified data snapshot (generation + load + balance).
    
    Returns most recent available data for all components with quality metadata.
    
    Args:
        country: ISO country code (default: NL)
    
    Response includes:
        - generation: Total MW + renewable percentage (strict policy: waste excluded)
        - load: Actual + forecast consumption
        - balance: Grid delta + settlement price
        - overall_status: Worst status across all components
        - per-component: availability, freshness, timestamps
    """
    
    # Cache key (country-specific)
    cache_key = f"now:{country}:latest"
    
    # Check cache
    cached = api_cache.get(cache_key)
    if cached:
        return Response(
            content=cached,
            media_type="application/json",
            headers={"X-Cache": "HIT"}
        )
    
    # Fetch fresh data
    result = get_unified_snapshot(db, country=country)
    
    # Serialize with Pydantic-style JSON (datetime handling)
    import json
    json_content = json.dumps(result, default=str)
    
    # Cache for 5 minutes
    api_cache.set(cache_key, json_content, ttl=300)
    
    return Response(
        content=json_content,
        media_type="application/json",
        headers={"X-Cache": "MISS"}
    )
