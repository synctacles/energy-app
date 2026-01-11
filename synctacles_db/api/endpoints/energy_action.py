"""
Energy Action API Endpoint

Issue #62 - Add quality indicator to Energy Action

Core endpoint for SYNCTACLES Energy Action functionality.
Returns USE/WAIT/SKIP recommendations with quality metadata.
"""

from datetime import datetime, timezone, timedelta
from typing import Optional
from fastapi import APIRouter, Depends, Query
from fastapi.responses import Response
from sqlalchemy.orm import Session
from sqlalchemy import text
import json
import logging

from synctacles_db.api.dependencies import get_db
from synctacles_db.cache import api_cache
from synctacles_db.fallback.fallback_manager import FallbackManager

_LOGGER = logging.getLogger(__name__)

router = APIRouter(prefix="/v1", tags=["energy-action"])


class PriceQuality:
    """Quality levels for price data."""
    LIVE = "live"           # 100% accurate (Enever/Frank)
    ESTIMATED = "estimated"  # 89% accurate (ENTSO-E + lookup)
    CACHED = "cached"        # Stale but functional
    UNAVAILABLE = "unavailable"


class EnergyAction:
    """Energy action recommendations."""
    USE = "USE"     # Good time to use energy
    WAIT = "WAIT"   # Wait for better pricing
    SKIP = "SKIP"   # Avoid energy use now


@router.get("/energy-action")
async def get_energy_action(
    country: str = Query("NL", description="Country code (NL, BE, DE)"),
    db: Session = Depends(get_db)
):
    """
    Get Energy Action recommendation with quality metadata.

    The core SYNCTACLES endpoint for home automation.

    Returns:
        - action: USE / WAIT / SKIP recommendation
        - price_eur_kwh: Current consumer price
        - quality: live / estimated / cached / unavailable
        - source: Data source (enever, entsoe+lookup, energy-charts+lookup, cache)
        - confidence: 0-100 percentage
        - cheapest_hour: Next cheapest hour today
        - most_expensive_hour: Next most expensive hour today

    Quality levels:
        - live (100%): Real consumer prices from Frank/Enever
        - estimated (89%): ENTSO-E + lookup table calculation
        - cached (50%): Stale data from cache
        - unavailable (0%): No data available
    """

    cache_key = f"energy-action:{country.upper()}"

    # Check cache (short TTL for this endpoint)
    cached_response = api_cache.get(cache_key)
    if cached_response:
        return Response(
            content=cached_response,
            media_type="application/json",
            headers={
                "X-Cache": "HIT",
                "X-Data-Source": "cache",
            }
        )

    now = datetime.now(timezone.utc)

    # Get price data with fallback chain
    prices_data = await get_prices_with_quality(db, country)

    if not prices_data["prices"]:
        result = {
            "action": None,
            "price_eur_kwh": None,
            "quality": PriceQuality.UNAVAILABLE,
            "source": None,
            "confidence": 0,
            "cheapest_hour": None,
            "most_expensive_hour": None,
            "timestamp": now.isoformat(),
            "message": "No price data available"
        }
    else:
        # Find current price
        current_price = get_current_price_from_list(prices_data["prices"], now)
        daily_avg = calculate_daily_average(prices_data["prices"])

        # Calculate action
        action = calculate_action(current_price, daily_avg, prices_data["prices"])

        # Find cheapest/expensive hours
        cheapest = find_cheapest_hour(prices_data["prices"], now)
        most_expensive = find_most_expensive_hour(prices_data["prices"], now)

        result = {
            "action": action,
            "price_eur_kwh": round(current_price, 4) if current_price else None,
            "quality": prices_data["quality"],
            "source": prices_data["source"],
            "confidence": prices_data["confidence"],
            "cheapest_hour": cheapest,
            "most_expensive_hour": most_expensive,
            "daily_average": round(daily_avg, 4) if daily_avg else None,
            "timestamp": now.isoformat(),
            "allow_automation": prices_data["allow_go_action"],
        }

    # Serialize and cache
    json_content = json.dumps(result, default=str)

    # Cache for 60 seconds (prices can change hourly)
    api_cache.set(cache_key, json_content, ttl=60)

    return Response(
        content=json_content,
        media_type="application/json",
        headers={
            "X-Cache": "MISS",
            "X-Data-Source": result.get("source", "none"),
            "X-Data-Quality": result.get("quality", "unavailable"),
            "X-Confidence": str(result.get("confidence", 0)),
        }
    )


async def get_prices_with_quality(db: Session, country: str) -> dict:
    """
    Get prices with quality metadata using fallback chain.

    Returns:
        dict with keys: prices, source, quality, confidence, allow_go_action
    """
    now = datetime.now(timezone.utc)

    # Query database for prices
    start_of_today = now.replace(hour=0, minute=0, second=0, microsecond=0)
    end_of_tomorrow = start_of_today + timedelta(days=2)

    result = db.execute(text("""
        SELECT
            timestamp,
            price_eur_mwh / 1000.0 as price_eur_kwh,
            data_source,
            data_quality
        FROM norm_entso_e_a44
        WHERE country = :country
          AND timestamp >= :start
          AND timestamp < :end
        ORDER BY timestamp DESC
    """), {
        "country": country.upper(),
        "start": start_of_today,
        "end": end_of_tomorrow
    }).fetchall()

    # Calculate database age
    db_age_minutes = 999
    db_results = None

    if result:
        db_results = [
            {
                "timestamp": row[0].isoformat(),
                "price_eur_mwh": float(row[1]) * 1000  # Convert back for FallbackManager
            }
            for row in result
        ]
        latest_timestamp = result[0][0]
        db_age_minutes = int((now - latest_timestamp).total_seconds() / 60)

    # Use FallbackManager for 5-tier fallback
    prices, source, quality, allow_go = await FallbackManager.get_prices_with_fallback(
        db_results=db_results,
        db_age_minutes=db_age_minutes,
        country=country.lower()
    )

    # Map quality to confidence
    confidence_map = {
        "FRESH": 100,
        "STALE": 85,
        "FALLBACK": 70,
        "CACHED": 50,
        "UNAVAILABLE": 0,
    }

    # Map quality to our enum
    quality_map = {
        "FRESH": PriceQuality.LIVE,
        "STALE": PriceQuality.ESTIMATED,
        "FALLBACK": PriceQuality.ESTIMATED,
        "CACHED": PriceQuality.CACHED,
        "UNAVAILABLE": PriceQuality.UNAVAILABLE,
    }

    return {
        "prices": prices,
        "source": source,
        "quality": quality_map.get(quality, PriceQuality.UNAVAILABLE),
        "confidence": confidence_map.get(quality, 0),
        "allow_go_action": allow_go,
    }


def get_current_price_from_list(prices: list, now: datetime) -> Optional[float]:
    """Get the price for the current hour."""
    current_hour = now.replace(minute=0, second=0, microsecond=0)

    for price in prices:
        ts_str = price.get("timestamp", "")
        if isinstance(ts_str, str):
            ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        else:
            ts = ts_str

        if ts.replace(minute=0, second=0, microsecond=0) == current_hour:
            # Convert from EUR/MWh to EUR/kWh if needed
            price_val = price.get("price_eur_mwh") or price.get("price_eur_kwh", 0)
            if price_val > 1:  # Likely EUR/MWh
                return float(price_val) / 1000.0
            return float(price_val)

    return None


def calculate_daily_average(prices: list) -> Optional[float]:
    """Calculate average price from price list."""
    if not prices:
        return None

    values = []
    for price in prices:
        price_val = price.get("price_eur_mwh") or price.get("price_eur_kwh", 0)
        if price_val > 1:  # Likely EUR/MWh
            values.append(float(price_val) / 1000.0)
        else:
            values.append(float(price_val))

    return sum(values) / len(values) if values else None


def calculate_action(current_price: Optional[float], daily_avg: Optional[float], prices: list) -> str:
    """
    Calculate energy action recommendation.

    USE: Current price is below average (good time)
    WAIT: Current price is near average
    SKIP: Current price is significantly above average
    """
    if current_price is None or daily_avg is None:
        return EnergyAction.WAIT

    # Calculate thresholds
    low_threshold = daily_avg * 0.85   # 15% below average
    high_threshold = daily_avg * 1.15  # 15% above average

    if current_price <= low_threshold:
        return EnergyAction.USE
    elif current_price >= high_threshold:
        return EnergyAction.SKIP
    else:
        return EnergyAction.WAIT


def find_cheapest_hour(prices: list, after: datetime) -> Optional[dict]:
    """Find the cheapest hour from now until end of day."""
    if not prices:
        return None

    future_prices = []
    for price in prices:
        ts_str = price.get("timestamp", "")
        if isinstance(ts_str, str):
            ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        else:
            ts = ts_str

        if ts > after:
            price_val = price.get("price_eur_mwh") or price.get("price_eur_kwh", 0)
            if price_val > 1:
                price_val = float(price_val) / 1000.0
            future_prices.append({
                "timestamp": ts.isoformat(),
                "hour": ts.hour,
                "price_eur_kwh": float(price_val)
            })

    if not future_prices:
        return None

    return min(future_prices, key=lambda x: x["price_eur_kwh"])


def find_most_expensive_hour(prices: list, after: datetime) -> Optional[dict]:
    """Find the most expensive hour from now until end of day."""
    if not prices:
        return None

    future_prices = []
    for price in prices:
        ts_str = price.get("timestamp", "")
        if isinstance(ts_str, str):
            ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        else:
            ts = ts_str

        if ts > after:
            price_val = price.get("price_eur_mwh") or price.get("price_eur_kwh", 0)
            if price_val > 1:
                price_val = float(price_val) / 1000.0
            future_prices.append({
                "timestamp": ts.isoformat(),
                "hour": ts.hour,
                "price_eur_kwh": float(price_val)
            })

    if not future_prices:
        return None

    return max(future_prices, key=lambda x: x["price_eur_kwh"])
