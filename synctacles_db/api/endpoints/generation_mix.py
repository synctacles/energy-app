"""Generation Mix endpoint with Energy-Charts fallback."""

from datetime import datetime, timezone
from typing import Dict, Any
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from sqlalchemy import text

from synctacles_db.api.dependencies import get_db
from synctacles_db.api.cache import cached, generation_cache
from synctacles_db.fallback.fallback_manager import FallbackManager

router = APIRouter(prefix="", tags=["generation"])


@cached(generation_cache)
@router.get("/generation-mix")
async def get_generation_mix(
    limit: int = Query(default=10, ge=1, le=100),
    db: Session = Depends(get_db)
) -> Dict[str, Any]:
    """
    Get power generation mix by source with Energy-Charts fallback.
    
    Returns latest generation data from ENTSO-E, falls back to Energy-Charts if needed.
    """
    
    # Try database first
    result = db.execute(text("""
        SELECT
            b01_biomass_mw,
            b04_gas_mw,
            b05_coal_mw,
            b14_nuclear_mw as nuclear_mw,
            b20_other_mw,
            b16_solar_mw,
            b17_waste_mw,
            b18_wind_offshore_mw,
            b19_wind_onshore_mw,
            total_mw,
            timestamp,
            EXTRACT(EPOCH FROM (NOW() - timestamp))/60 as age_minutes
        FROM norm_entso_e_a75
        WHERE country = 'NL'
        ORDER BY timestamp DESC
        LIMIT 1
    """)).fetchone()
    
    db_data = None
    db_age = 999
    
    if result:
        db_data = {
            "biomass_mw": float(result[0]) if result[0] is not None else None,
            "gas_mw": float(result[1]) if result[1] is not None else None,
            "coal_mw": float(result[2]) if result[2] is not None else None,
            "nuclear_mw": float(result[3]) if result[3] is not None else None,
            "other_mw": float(result[4]) if result[4] is not None else None,
            "solar_mw": float(result[5]) if result[5] is not None else None,
            "waste_mw": float(result[6]) if result[6] is not None else None,
            "wind_offshore_mw": float(result[7]) if result[7] is not None else None,
            "wind_onshore_mw": float(result[8]) if result[8] is not None else None,
            "total_mw": float(result[9]) if result[9] is not None else None,
            "timestamp": result[10].isoformat() if result[10] else None,
        }
        db_age = int(result[11])
    
    # Use component-based fallback
    data, source, quality = await FallbackManager.get_component_with_fallback(
        component="generation_mix",
        db_result=db_data,
        db_age_minutes=db_age,
        country="nl"
    )
    
    if not data:
        return {
            "data": [],
            "metadata": {
                "count": 0,
                "source": "None",
                "quality": "UNAVAILABLE",
            }
        }
    
    # Extract field sources and EC timestamp (internal metadata)
    field_sources = data.pop("_field_sources", {})
    ec_timestamp = data.pop("_ec_timestamp", None)
    
    # Calculate age from timestamp
    if "timestamp" in data and data["timestamp"]:
        ts = datetime.fromisoformat(data["timestamp"].replace("Z", "+00:00"))
        age = int((datetime.now(timezone.utc) - ts).total_seconds() / 60)
    else:
        age = db_age
    
    # Calculate renewable percentage
    renewable_pct = FallbackManager.calculate_renewable_percentage(data)
    
    # Build metadata
    metadata = {
        "count": 1,
        "source": source,
        "quality": quality,
        "age_minutes": age,
        "renewable_percentage": round(renewable_pct, 1) if renewable_pct else None,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "field_sources": field_sources,
    }
    
    # Add EC timestamp if hybrid merge was used
    if ec_timestamp:
        metadata["ec_timestamp"] = ec_timestamp
        # Calculate EC data age for transparency
        ec_ts = datetime.fromisoformat(ec_timestamp.replace("Z", "+00:00"))
        ec_age = int((datetime.now(timezone.utc) - ec_ts).total_seconds() / 60)
        metadata["ec_age_minutes"] = ec_age
    
    return {
        "data": [data],
        "metadata": metadata
    }