"""Load endpoint with Energy-Charts fallback."""

from datetime import datetime, timezone
from typing import Dict, Any
from fastapi import APIRouter, Depends
from fastapi.responses import Response
from sqlalchemy.orm import Session
from sqlalchemy import text
import json

from synctacles_db.api.dependencies import get_db
from synctacles_db.cache import api_cache
from synctacles_db.fallback.fallback_manager import FallbackManager

router = APIRouter(prefix="", tags=["load"])


@router.get("/load")
async def get_load(
    db: Session = Depends(get_db)
):
    """
    Get actual electricity load with Energy-Charts fallback.

    Returns latest load data from ENTSO-E, falls back to Energy-Charts if needed.
    """

    # Cache key
    cache_key = "load:latest"

    # Check cache
    cached_response = api_cache.get(cache_key)
    if cached_response:
        return Response(
            content=cached_response,
            media_type="application/json",
            headers={"X-Cache": "HIT"}
        )

    # Try database first
    result = db.execute(text("""
        SELECT
            actual_mw as load_mw,
            timestamp,
            EXTRACT(EPOCH FROM (NOW() - timestamp))/60 as age_minutes
        FROM norm_entso_e_a65
        WHERE country = 'NL'
          AND actual_mw IS NOT NULL
          AND timestamp <= NOW()
        ORDER BY timestamp DESC
        LIMIT 1
    """)).fetchone()
    
    db_data = None
    db_age = 999
    
    if result:
        db_data = {
            "load_mw": float(result[0] or 0),
            "timestamp": result[1].isoformat() if result[1] else None,
        }
        db_age = int(result[2])
    
    # Use component-based fallback
    data, source, quality = await FallbackManager.get_component_with_fallback(
        component="load",
        db_result=db_data,
        db_age_minutes=db_age,
        country="nl"
    )
    
    if not data:
        return {
            "load_mw": None,
            "metadata": {
                "source": "None",
                "quality": "UNAVAILABLE",
                "timestamp": datetime.now(timezone.utc).isoformat(),
            }
        }
    
    # Calculate age from timestamp
    if "timestamp" in data and data["timestamp"]:
        ts = datetime.fromisoformat(data["timestamp"].replace("Z", "+00:00"))
        age = int((datetime.now(timezone.utc) - ts).total_seconds() / 60)
    else:
        age = db_age
    
    # Build response
    result_dict = {
        "load_mw": round(data.get("load_mw", 0), 2),
        "metadata": {
            "source": source,
            "quality": quality,
            "age_minutes": age,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }
    }

    # Serialize to JSON
    json_content = json.dumps(result_dict, default=str)

    # Cache for 5 minutes (300s)
    api_cache.set(cache_key, json_content, ttl=300)

    return Response(
        content=json_content,
        media_type="application/json",
        headers={"X-Cache": "MISS"}
    )
