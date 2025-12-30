"""
SYNCTACLES Binary Signals API with Energy-Charts Fallback (ASYNC)
"""

from datetime import datetime, timezone
from typing import Dict, Any, Tuple, Optional
from fastapi import APIRouter, Depends, HTTPException, Header
from sqlalchemy.orm import Session
from sqlalchemy import text

from synctacles_db.api.dependencies import get_db
from synctacles_db import auth_service
from synctacles_db.fallback.fallback_manager import FallbackManager

router = APIRouter(prefix="/v1", tags=["signals"])


async def get_current_user_from_key(
    x_api_key: str = Header(None, alias="X-API-Key"),
    db: Session = Depends(get_db)
) -> dict:
    """Validate API key and return user info"""
    user = auth_service.validate_api_key(db, x_api_key) if x_api_key else None
    if x_api_key and not user:
        raise HTTPException(status_code=401, detail="Invalid API key")
    if not user:
        return {"user_id": 0, "email": "anonymous", "tier": "free"}

    return {
        "user_id": user.id,
        "email": user.email,
        "tier": user.tier
    }


@router.get("/signals")
async def get_signals(
    current_user: dict = Depends(get_current_user_from_key),
    db: Session = Depends(get_db)
) -> Dict[str, Any]:
    """
    Get binary signals for energy automation with fallback support

    Auth: X-API-Key header (automatic user extraction)

    Returns:
        - is_cheap: Current price below daily average
        - is_green: Renewable percentage above threshold
        - charge_now: Combined price + forecast recommendation
        - grid_stable: Grid balance within acceptable range
        - cheap_hour_coming: Price dip expected in next 3h
    """

    user_id = current_user["user_id"]

    # Get current metrics (with ASYNC fallback for renewable)
    current_price = get_current_price(db)
    daily_avg = get_daily_average_price(db)
    renewable_pct, renewable_age, renewable_quality, renewable_source = await get_renewable_with_fallback(db)
    balance_delta = get_balance_delta(db)
    next_3h_prices = get_next_3h_prices(db)

    # Thresholds (will be user-configurable later)
    RENEWABLE_THRESHOLD = 50  # %
    BALANCE_THRESHOLD = 500  # MW

    # Calculate signals with null safety
    try:
        is_cheap = current_price < daily_avg if (current_price and daily_avg) else False
        is_green = renewable_pct > RENEWABLE_THRESHOLD if renewable_pct is not None else False
        charge_now = (is_cheap and renewable_pct and renewable_pct > 40)
        grid_stable = abs(balance_delta) < BALANCE_THRESHOLD if balance_delta is not None else True
        
        if next_3h_prices and current_price:
            cheap_hour_coming = min(next_3h_prices) < (current_price * 0.9)
        else:
            cheap_hour_coming = False
    except (ValueError, TypeError):
        is_cheap = is_green = charge_now = cheap_hour_coming = False
        grid_stable = True

    return {
        "signals": {
            "is_cheap": is_cheap,
            "is_green": is_green,
            "charge_now": charge_now,
            "grid_stable": grid_stable,
            "cheap_hour_coming": cheap_hour_coming
        },
        "metadata": {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "user_id": str(user_id),
            "email": current_user["email"],
            "tier": current_user["tier"],
            "current_price": round(current_price, 4) if current_price else None,
            "daily_avg": round(daily_avg, 4) if daily_avg else None,
            "renewable_pct": round(renewable_pct, 1) if renewable_pct is not None else None,
            "balance_delta": round(balance_delta, 0) if balance_delta is not None else None,
            "confidence": "high",
            "data_quality": {
                "renewable_source": renewable_source,
                "renewable_quality": renewable_quality,
                "renewable_age_minutes": renewable_age
            }
        }
    }


async def get_renewable_with_fallback(db: Session) -> Tuple[Optional[float], int, str, str]:
    """
    Get renewable % with Energy-Charts fallback (ASYNC)
    Returns: (percentage, age_minutes, quality, source)
    """
    # Try database first
    result = db.execute(text("""
        SELECT 
            (COALESCE(b01_biomass_mw, 0) + 
             COALESCE(b16_solar_mw, 0) + 
             COALESCE(b18_wind_offshore_mw, 0) + 
             COALESCE(b19_wind_onshore_mw, 0)) * 100.0 / 
            NULLIF(total_mw, 0) as renewable_pct,
            EXTRACT(EPOCH FROM (NOW() - timestamp))/60 as age_minutes,
            b01_biomass_mw,
            b16_solar_mw,
            b18_wind_offshore_mw,
            b19_wind_onshore_mw,
            total_mw,
            timestamp
        FROM norm_entso_e_a75
        WHERE country = 'NL'
        ORDER BY timestamp DESC
        LIMIT 1
    """)).fetchone()
    
    db_data = None
    db_age = 999
    
    if result and result[0] is not None:
        db_data = {
            "renewable_pct": float(result[0]),
            "biomass_mw": float(result[2] or 0),
            "solar_mw": float(result[3] or 0),
            "wind_offshore_mw": float(result[4] or 0),
            "wind_onshore_mw": float(result[5] or 0),
            "total_mw": float(result[6] or 0),
            "timestamp": result[7].isoformat() if result[7] else None,
        }
        db_age = int(result[1])
    
    # Use ASYNC fallback manager
    data, source, quality = await FallbackManager.get_generation_with_fallback(
        db_result=db_data,
        db_age_minutes=db_age,
        country="nl"
    )
    
    if not data:
        return None, 999, "UNAVAILABLE", "None"
    
    # Calculate or extract renewable percentage
    if "renewable_pct" in data:
        pct = data["renewable_pct"]
    else:
        pct = FallbackManager.calculate_renewable_percentage(data)
    
    # Calculate age from timestamp if available
    if "timestamp" in data and isinstance(data["timestamp"], str):
        ts = datetime.fromisoformat(data["timestamp"].replace("Z", "+00:00"))
        age = int((datetime.now(timezone.utc) - ts).total_seconds() / 60)
    else:
        age = db_age
    
    return pct, age, quality, source


def get_current_price(db: Session) -> float:
    """Get most recent price from norm_entso_e_a44"""
    result = db.execute(text("""
        SELECT price_eur_mwh / 1000.0 as price_eur_kwh
        FROM norm_entso_e_a44
        WHERE country = 'NL'
        ORDER BY timestamp DESC
        LIMIT 1
    """)).fetchone()

    if not result:
        raise HTTPException(status_code=503, detail="No price data available")

    return float(result[0])


def get_daily_average_price(db: Session) -> float:
    """Get 24h rolling average price"""
    result = db.execute(text("""
        SELECT AVG(price_eur_mwh / 1000.0) as avg_price
        FROM norm_entso_e_a44
        WHERE country = 'NL'
          AND timestamp >= NOW() - INTERVAL '24 hours'
    """)).fetchone()

    return float(result[0]) if result and result[0] else 0.15


def get_balance_delta(db: Session) -> float:
    """Get current grid balance (positive = surplus, negative = deficit)"""
    result = db.execute(text("""
        SELECT delta_mw
        FROM norm_tennet_balance
        ORDER BY timestamp DESC
        LIMIT 1
    """)).fetchone()

    return float(result[0]) if result and result[0] else 0.0


def get_next_3h_prices(db: Session) -> list:
    """Get prices for next 3 hours"""
    result = db.execute(text("""
        SELECT price_eur_mwh / 1000.0 as price_eur_kwh
        FROM norm_entso_e_a44
        WHERE country = 'NL'
          AND timestamp > NOW()
          AND timestamp <= NOW() + INTERVAL '3 hours'
        ORDER BY timestamp
    """)).fetchall()

    return [float(r[0]) for r in result] if result else []
