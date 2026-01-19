"""
Best Window Finder & Tomorrow Preview Endpoints

Two "wow" features for SYNCTACLES:
1. Best Window Finder - Find optimal time window for energy consumption
2. Tomorrow Preview - Summary of tomorrow's price outlook

These features differentiate SYNCTACLES from Nordpool integration.
"""

from datetime import datetime, timezone, timedelta
from typing import Optional, List, Dict, Tuple
from fastapi import APIRouter, Query
from fastapi.responses import Response
import json
import logging

from synctacles_db.cache import api_cache
from synctacles_db.fallback.fallback_manager import FallbackManager

_LOGGER = logging.getLogger(__name__)

router = APIRouter(prefix="/v1", tags=["windows"])


# =============================================================================
# BEST WINDOW FINDER
# =============================================================================

@router.get("/best-window")
async def get_best_window(
    duration: int = Query(3, ge=1, le=8, description="Window duration in hours (1-8)"),
    country: str = Query("NL", description="Country code"),
):
    """
    Find the cheapest consecutive time window for energy consumption.

    This solves a real problem: Nordpool only gives "cheapest hour", but users
    need to know "when should I run my 3-hour EV charge?" or "best time for
    2-hour dishwasher cycle?".

    Algorithm: Sliding window over all available prices (today + tomorrow).

    Args:
        duration: Window length in hours (1-8)
        country: Country code (NL, BE, DE)

    Returns:
        - window: Best window with start/end times and average price
        - runner_up: Second best option (useful if best window is inconvenient)
        - comparison: Savings vs current price
        - meta: Data source and quality info
    """
    cache_key = f"best-window:{country.upper()}:{duration}"

    cached = api_cache.get(cache_key)
    if cached:
        return Response(
            content=cached,
            media_type="application/json",
            headers={"X-Cache": "HIT"}
        )

    now = datetime.now(timezone.utc)

    # Get prices (today + tomorrow if available)
    prices = await _get_all_prices(country)

    if not prices or len(prices) < duration:
        result = {
            "window": None,
            "runner_up": None,
            "comparison": None,
            "meta": {
                "source": "none",
                "quality": "UNAVAILABLE",
                "hours_analyzed": 0,
                "error": f"Insufficient price data (need {duration}h, have {len(prices) if prices else 0}h)"
            },
            "timestamp": now.isoformat()
        }
        return Response(
            content=json.dumps(result),
            media_type="application/json"
        )

    # Find best windows
    best, runner_up = _find_best_windows(prices, duration, now)

    # Get current price for comparison
    current_price = _get_price_at_time(prices, now)

    # Build comparison
    comparison = None
    if best and current_price:
        savings = current_price - best["average_price_eur_kwh"]
        comparison = {
            "current_price_eur_kwh": round(current_price, 4),
            "savings_vs_now_eur_kwh": round(savings, 4),
            "savings_percent": round((savings / current_price) * 100, 1) if current_price > 0 else 0
        }

    result = {
        "window": best,
        "runner_up": runner_up,
        "comparison": comparison,
        "meta": {
            "source": prices[0].get("_source", "unknown") if prices else "none",
            "quality": prices[0].get("_quality", "unknown") if prices else "UNAVAILABLE",
            "hours_analyzed": len(prices),
            "duration_requested": duration
        },
        "timestamp": now.isoformat()
    }

    json_content = json.dumps(result, default=str)
    api_cache.set(cache_key, json_content, ttl=300)  # 5 min cache

    return Response(
        content=json_content,
        media_type="application/json",
        headers={"X-Cache": "MISS"}
    )


def _find_best_windows(
    prices: List[Dict],
    duration: int,
    now: datetime
) -> Tuple[Optional[Dict], Optional[Dict]]:
    """
    Find the two best consecutive windows using sliding window algorithm.

    Only considers windows starting from now or later.
    Handles overnight windows (e.g., 23:00 - 02:00).
    """
    if len(prices) < duration:
        return None, None

    # Sort prices by timestamp
    sorted_prices = sorted(prices, key=lambda x: x["timestamp"])

    # Filter to only future hours
    future_prices = []
    for p in sorted_prices:
        ts = _parse_timestamp(p["timestamp"])
        if ts >= now.replace(minute=0, second=0, microsecond=0):
            future_prices.append(p)

    if len(future_prices) < duration:
        return None, None

    # Sliding window
    windows = []
    for i in range(len(future_prices) - duration + 1):
        window_prices = future_prices[i:i + duration]

        # Calculate average
        values = [_extract_price_kwh(p) for p in window_prices]
        if None in values:
            continue

        avg_price = sum(values) / len(values)

        start_ts = _parse_timestamp(window_prices[0]["timestamp"])
        end_ts = _parse_timestamp(window_prices[-1]["timestamp"]) + timedelta(hours=1)

        windows.append({
            "start": start_ts.isoformat(),
            "end": end_ts.isoformat(),
            "start_hour": start_ts.strftime("%H:%M"),
            "end_hour": end_ts.strftime("%H:%M"),
            "duration_hours": duration,
            "average_price_eur_kwh": round(avg_price, 4),
            "total_cost_estimate_eur": round(avg_price * duration, 4),
            "_avg_for_sort": avg_price
        })

    if not windows:
        return None, None

    # Sort by average price
    windows.sort(key=lambda x: x["_avg_for_sort"])

    # Remove sort key from output
    for w in windows:
        del w["_avg_for_sort"]

    best = windows[0]
    runner_up = windows[1] if len(windows) > 1 else None

    return best, runner_up


# =============================================================================
# TOMORROW PREVIEW
# =============================================================================

@router.get("/tomorrow")
async def get_tomorrow_preview(
    country: str = Query("NL", description="Country code"),
):
    """
    Get a summary of tomorrow's electricity prices.

    Tomorrow's prices are typically available after 13:00 CET.
    This endpoint provides a quick overview without needing to parse
    the full price list.

    Returns:
        - status: FAVORABLE / NORMAL / EXPENSIVE (quick assessment)
        - available: Whether tomorrow's data exists
        - summary: Cheapest/most expensive hours, averages
        - best_window_3h: Pre-calculated best 3-hour window
        - comparison_vs_today: How tomorrow compares to today
    """
    cache_key = f"tomorrow:{country.upper()}"

    cached = api_cache.get(cache_key)
    if cached:
        return Response(
            content=cached,
            media_type="application/json",
            headers={"X-Cache": "HIT"}
        )

    now = datetime.now(timezone.utc)
    tomorrow = (now + timedelta(days=1)).date()
    today = now.date()

    # Get all prices
    all_prices = await _get_all_prices(country)

    if not all_prices:
        result = {
            "status": "UNKNOWN",
            "available": False,
            "summary": None,
            "best_window_3h": None,
            "comparison_vs_today": None,
            "meta": {
                "source": "none",
                "quality": "UNAVAILABLE",
                "message": "No price data available"
            },
            "timestamp": now.isoformat()
        }
        return Response(
            content=json.dumps(result),
            media_type="application/json"
        )

    # Split into today and tomorrow
    today_prices = []
    tomorrow_prices = []

    for p in all_prices:
        ts = _parse_timestamp(p["timestamp"])
        if ts.date() == today:
            today_prices.append(p)
        elif ts.date() == tomorrow:
            tomorrow_prices.append(p)

    # Check if tomorrow data is available
    if len(tomorrow_prices) < 12:  # Need at least half a day
        result = {
            "status": "PENDING",
            "available": False,
            "summary": None,
            "best_window_3h": None,
            "comparison_vs_today": None,
            "meta": {
                "source": all_prices[0].get("_source", "unknown") if all_prices else "none",
                "quality": "PENDING",
                "message": "Tomorrow's prices not yet available (usually after 13:00 CET)",
                "hours_available": len(tomorrow_prices)
            },
            "timestamp": now.isoformat()
        }
        json_content = json.dumps(result)
        api_cache.set(cache_key, json_content, ttl=300)
        return Response(
            content=json_content,
            media_type="application/json",
            headers={"X-Cache": "MISS"}
        )

    # Calculate tomorrow summary
    tomorrow_values = [_extract_price_kwh(p) for p in tomorrow_prices]
    tomorrow_values = [v for v in tomorrow_values if v is not None]

    if not tomorrow_values:
        result = {
            "status": "UNKNOWN",
            "available": False,
            "summary": None,
            "meta": {"error": "Could not parse tomorrow prices"},
            "timestamp": now.isoformat()
        }
        return Response(content=json.dumps(result), media_type="application/json")

    tomorrow_avg = sum(tomorrow_values) / len(tomorrow_values)
    tomorrow_min = min(tomorrow_values)
    tomorrow_max = max(tomorrow_values)

    # Find cheapest and most expensive hours
    sorted_tomorrow = sorted(tomorrow_prices, key=lambda x: _extract_price_kwh(x) or 999)
    cheapest = sorted_tomorrow[0]
    most_expensive = sorted_tomorrow[-1]

    cheapest_ts = _parse_timestamp(cheapest["timestamp"])
    expensive_ts = _parse_timestamp(most_expensive["timestamp"])

    # Calculate best 3h window for tomorrow
    best_3h, _ = _find_best_windows(tomorrow_prices, 3,
                                     datetime.combine(tomorrow, datetime.min.time()).replace(tzinfo=timezone.utc))

    # Compare to today
    comparison = None
    if today_prices:
        today_values = [_extract_price_kwh(p) for p in today_prices]
        today_values = [v for v in today_values if v is not None]
        if today_values:
            today_avg = sum(today_values) / len(today_values)
            diff = tomorrow_avg - today_avg
            diff_percent = (diff / today_avg) * 100 if today_avg > 0 else 0
            comparison = {
                "today_average_eur_kwh": round(today_avg, 4),
                "tomorrow_average_eur_kwh": round(tomorrow_avg, 4),
                "difference_eur_kwh": round(diff, 4),
                "difference_percent": round(diff_percent, 1)
            }

    # Determine status
    status = _determine_status(tomorrow_avg, today_prices)

    result = {
        "status": status,
        "available": True,
        "date": tomorrow.isoformat(),
        "summary": {
            "cheapest_hour": cheapest_ts.strftime("%H:%M"),
            "cheapest_price_eur_kwh": round(tomorrow_min, 4),
            "most_expensive_hour": expensive_ts.strftime("%H:%M"),
            "most_expensive_price_eur_kwh": round(tomorrow_max, 4),
            "average_price_eur_kwh": round(tomorrow_avg, 4),
            "price_spread_eur_kwh": round(tomorrow_max - tomorrow_min, 4)
        },
        "best_window_3h": best_3h,
        "comparison_vs_today": comparison,
        "meta": {
            "source": all_prices[0].get("_source", "unknown") if all_prices else "none",
            "quality": all_prices[0].get("_quality", "FRESH") if all_prices else "UNAVAILABLE",
            "hours_available": len(tomorrow_prices)
        },
        "timestamp": now.isoformat()
    }

    json_content = json.dumps(result, default=str)
    api_cache.set(cache_key, json_content, ttl=300)

    return Response(
        content=json_content,
        media_type="application/json",
        headers={"X-Cache": "MISS"}
    )


def _determine_status(tomorrow_avg: float, today_prices: List[Dict]) -> str:
    """
    Determine tomorrow's status: FAVORABLE / NORMAL / EXPENSIVE

    Based on comparison with today and absolute thresholds.
    """
    # Absolute thresholds (EUR/kWh)
    CHEAP_THRESHOLD = 0.20
    EXPENSIVE_THRESHOLD = 0.30

    # Check absolute first
    if tomorrow_avg < CHEAP_THRESHOLD:
        return "FAVORABLE"
    elif tomorrow_avg > EXPENSIVE_THRESHOLD:
        return "EXPENSIVE"

    # Compare to today if available
    if today_prices:
        today_values = [_extract_price_kwh(p) for p in today_prices]
        today_values = [v for v in today_values if v is not None]
        if today_values:
            today_avg = sum(today_values) / len(today_values)

            # 10% better than today = favorable
            if tomorrow_avg < today_avg * 0.90:
                return "FAVORABLE"
            # 10% worse than today = expensive
            elif tomorrow_avg > today_avg * 1.10:
                return "EXPENSIVE"

    return "NORMAL"


# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

async def _get_all_prices(country: str) -> List[Dict]:
    """
    Get all available prices (today + tomorrow).

    Strategy:
    - FallbackManager for TODAY (Frank DB/Direct = consumer prices)
    - Frank Direct for TOMORROW (consumer prices, available after ~14:00 CET)
    - ENTSO-E as fallback for TOMORROW (wholesale prices)
    - Merge both for complete 48h coverage
    """
    from sqlalchemy import create_engine, text
    from sqlalchemy.orm import sessionmaker
    from config.settings import DATABASE_URL
    from synctacles_db.clients.frank_energie_client import FrankEnergieClient

    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()

    now = datetime.now(timezone.utc)
    today = now.date()
    tomorrow = (now + timedelta(days=1)).date()
    start_of_today = now.replace(hour=0, minute=0, second=0, microsecond=0)
    end_of_tomorrow = start_of_today + timedelta(days=2)

    # Get ENTSO-E data directly (has both today AND tomorrow) - as fallback
    try:
        result = session.execute(text("""
            SELECT timestamp, price_eur_mwh
            FROM norm_entso_e_a44
            WHERE country = :country
              AND timestamp >= :start
              AND timestamp < :end
            ORDER BY timestamp ASC
        """), {
            "country": country.upper(),
            "start": start_of_today,
            "end": end_of_tomorrow
        }).fetchall()
    finally:
        session.close()

    entsoe_prices = []
    if result:
        entsoe_prices = [
            {"timestamp": row[0].isoformat(), "price_eur_mwh": float(row[1])}
            for row in result
        ]

    # Get FallbackManager data for TODAY (Frank = consumer prices)
    db_age_minutes = 999
    if entsoe_prices:
        from datetime import datetime as dt
        latest = max(dt.fromisoformat(p["timestamp"].replace("Z", "+00:00")) for p in entsoe_prices)
        db_age_minutes = int((now - latest).total_seconds() / 60)

    fallback_prices, source, quality, _ = await FallbackManager.get_prices_with_fallback(
        db_results=entsoe_prices,
        db_age_minutes=db_age_minutes,
        country=country.lower()
    )

    # Split ENTSO-E data into today/tomorrow (fallback only)
    entsoe_tomorrow = []
    for p in entsoe_prices:
        ts_str = p["timestamp"]
        ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        if ts.date() == tomorrow:
            entsoe_tomorrow.append(p)

    # Try Frank Direct for TOMORROW (consumer prices!)
    frank_tomorrow = []
    tomorrow_source = "ENTSO-E"  # default fallback
    try:
        frank_data = await FrankEnergieClient.get_prices_tomorrow()
        if frank_data and len(frank_data) > 0:
            _LOGGER.info(f"Frank Direct: got {len(frank_data)} tomorrow prices (consumer)")
            for p in frank_data:
                frank_tomorrow.append({
                    "timestamp": p["timestamp"],
                    "price_eur_kwh": p["price_eur_kwh"],
                    "_source": "Frank Direct",
                    "_quality": "FRESH",
                    "_price_type": "consumer"  # Includes taxes + markup
                })
            tomorrow_source = "Frank Direct"
    except Exception as e:
        _LOGGER.warning(f"Frank Direct tomorrow failed, using ENTSO-E: {e}")

    # Merge strategy:
    # - Use FallbackManager data for today (Frank prices if available)
    # - Use Frank Direct for tomorrow (consumer prices) or ENTSO-E (wholesale)
    all_prices = []

    if fallback_prices:
        # FallbackManager returned data (likely Frank for today)
        for p in fallback_prices:
            p["_source"] = source
            p["_quality"] = quality
        all_prices.extend(fallback_prices)

        # Check if fallback already has tomorrow data
        has_tomorrow = False
        for p in fallback_prices:
            ts_str = p.get("timestamp", "")
            if ts_str:
                ts = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
                if ts.date() == tomorrow:
                    has_tomorrow = True
                    break

        # Add tomorrow data if not already included
        if not has_tomorrow:
            if frank_tomorrow:
                # Prefer Frank consumer prices for tomorrow
                _LOGGER.info(f"Adding {len(frank_tomorrow)} Frank consumer prices for tomorrow")
                all_prices.extend(frank_tomorrow)
            elif entsoe_tomorrow:
                # Fallback to ENTSO-E wholesale
                _LOGGER.info(f"Adding {len(entsoe_tomorrow)} ENTSO-E wholesale prices for tomorrow")
                for p in entsoe_tomorrow:
                    p["_source"] = "ENTSO-E"
                    p["_quality"] = "FRESH"
                    p["_price_type"] = "wholesale"  # Excludes taxes
                all_prices.extend(entsoe_tomorrow)
    else:
        # No fallback data, use ENTSO-E directly
        for p in entsoe_prices:
            p["_source"] = "ENTSO-E"
            p["_quality"] = "FRESH"
            p["_price_type"] = "wholesale"
        all_prices = entsoe_prices

    # Sort by timestamp
    all_prices.sort(key=lambda x: x.get("timestamp", ""))

    _LOGGER.debug(f"_get_all_prices: {len(all_prices)} total ({source} today + {tomorrow_source} tomorrow)")
    return all_prices


def _parse_timestamp(ts_str: str) -> datetime:
    """Parse ISO timestamp string to datetime."""
    if isinstance(ts_str, datetime):
        return ts_str
    return datetime.fromisoformat(ts_str.replace("Z", "+00:00"))


def _extract_price_kwh(price: Dict) -> Optional[float]:
    """Extract price in EUR/kWh from price dict."""
    # Try different field names
    if "price_eur_kwh" in price:
        return float(price["price_eur_kwh"])
    elif "price_eur_mwh" in price:
        return float(price["price_eur_mwh"]) / 1000.0
    return None


def _get_price_at_time(prices: List[Dict], target: datetime) -> Optional[float]:
    """Get price for a specific hour."""
    target_hour = target.replace(minute=0, second=0, microsecond=0)

    for p in prices:
        ts = _parse_timestamp(p["timestamp"])
        if ts.replace(minute=0, second=0, microsecond=0) == target_hour:
            return _extract_price_kwh(p)

    return None


# =============================================================================
# DASHBOARD (BUNDLED ENDPOINT)
# =============================================================================

@router.get("/dashboard")
async def get_dashboard(
    country: str = Query("NL", description="Country code"),
    window_duration: int = Query(3, ge=1, le=8, description="Best window duration in hours"),
):
    """
    Bundled endpoint returning all energy insights in a single call.

    Designed for Home Assistant integration to minimize API calls.
    Returns prices, best window, tomorrow preview, and energy action in one response.

    Benefits:
    - Single API call instead of 3-4 separate calls
    - Atomic data: all values from same timestamp
    - Simpler rate limiting
    - Lower latency for HA coordinator

    Args:
        country: Country code (NL, BE, DE)
        window_duration: Duration for best window calculation (1-8 hours)
    """
    cache_key = f"dashboard:{country.upper()}:{window_duration}"

    cached = api_cache.get(cache_key)
    if cached:
        return Response(
            content=cached,
            media_type="application/json",
            headers={"X-Cache": "HIT"}
        )

    now = datetime.now(timezone.utc)

    # Get all prices (shared data source)
    all_prices = await _get_all_prices(country)

    # === CURRENT PRICE & ENERGY ACTION ===
    current_price = None
    energy_action = "WAIT"
    action_reason = "No data"

    today = now.date()
    today_prices = []

    if all_prices:
        current_price = _get_price_at_time(all_prices, now)
        today_prices = [p for p in all_prices if _parse_timestamp(p["timestamp"]).date() == today]

    # Today's stats (for cheapest/expensive hour sensors)
    today_cheapest_hour = None
    today_cheapest_price = None
    today_expensive_hour = None
    today_expensive_price = None
    today_average = None

    if today_prices:
        values = [_extract_price_kwh(p) for p in today_prices if _extract_price_kwh(p)]
        if values:
            today_average = sum(values) / len(values)

            # Find cheapest and most expensive hours
            sorted_today = sorted(today_prices, key=lambda x: _extract_price_kwh(x) or 999)
            if sorted_today:
                cheapest = sorted_today[0]
                expensive = sorted_today[-1]
                today_cheapest_hour = _parse_timestamp(cheapest["timestamp"]).strftime("%H:%M")
                today_cheapest_price = _extract_price_kwh(cheapest)
                today_expensive_hour = _parse_timestamp(expensive["timestamp"]).strftime("%H:%M")
                today_expensive_price = _extract_price_kwh(expensive)

            if current_price:
                deviation = ((current_price - today_average) / today_average) * 100 if today_average > 0 else 0

                if deviation <= -15:
                    energy_action = "GO"
                    action_reason = f"Price {abs(deviation):.0f}% below average"
                elif deviation >= 20:
                    energy_action = "AVOID"
                    action_reason = f"Price {deviation:.0f}% above average"
                else:
                    energy_action = "WAIT"
                    action_reason = f"Price {deviation:+.0f}% vs average"

    # === BEST WINDOW ===
    best_window = None
    runner_up = None

    if all_prices and len(all_prices) >= window_duration:
        best_window, runner_up = _find_best_windows(all_prices, window_duration, now)

    # === TOMORROW PREVIEW ===
    tomorrow_data = None
    tomorrow_status = "PENDING"
    tomorrow_date = (now + timedelta(days=1)).date()

    tomorrow_prices = [
        p for p in all_prices
        if _parse_timestamp(p["timestamp"]).date() == tomorrow_date
    ] if all_prices else []

    if len(tomorrow_prices) >= 12:
        tomorrow_values = [_extract_price_kwh(p) for p in tomorrow_prices]
        tomorrow_values = [v for v in tomorrow_values if v is not None]

        if tomorrow_values:
            tomorrow_avg = sum(tomorrow_values) / len(tomorrow_values)
            tomorrow_status = _determine_status(tomorrow_avg,
                [p for p in all_prices if _parse_timestamp(p["timestamp"]).date() == now.date()])

            sorted_tomorrow = sorted(tomorrow_prices, key=lambda x: _extract_price_kwh(x) or 999)
            cheapest = sorted_tomorrow[0]
            expensive = sorted_tomorrow[-1]

            # Best 3h window for tomorrow
            best_3h, _ = _find_best_windows(
                tomorrow_prices, 3,
                datetime.combine(tomorrow_date, datetime.min.time()).replace(tzinfo=timezone.utc)
            )

            tomorrow_data = {
                "date": tomorrow_date.isoformat(),
                "status": tomorrow_status,
                "cheapest_hour": _parse_timestamp(cheapest["timestamp"]).strftime("%H:%M"),
                "cheapest_price_eur_kwh": round(min(tomorrow_values), 4),
                "most_expensive_hour": _parse_timestamp(expensive["timestamp"]).strftime("%H:%M"),
                "most_expensive_price_eur_kwh": round(max(tomorrow_values), 4),
                "average_price_eur_kwh": round(tomorrow_avg, 4),
                "best_window_3h": best_3h,
                "hours_available": len(tomorrow_prices),
            }

    # === BUILD RESPONSE ===
    result = {
        "current": {
            "price_eur_kwh": round(current_price, 4) if current_price else None,
            "action": energy_action,
            "action_reason": action_reason,
            "cheapest_hour": today_cheapest_hour,
            "cheapest_price_eur_kwh": round(today_cheapest_price, 4) if today_cheapest_price else None,
            "most_expensive_hour": today_expensive_hour,
            "most_expensive_price_eur_kwh": round(today_expensive_price, 4) if today_expensive_price else None,
            "average_price_eur_kwh": round(today_average, 4) if today_average else None,
            "hours_available": len(today_prices) if today_prices else 0,
        },
        "best_window": best_window,
        "runner_up": runner_up,
        "tomorrow": tomorrow_data if tomorrow_data else {
            "status": "PENDING",
            "message": "Tomorrow's prices not yet available (usually after 13:00 CET)",
        },
        "meta": {
            "source": all_prices[0].get("_source", "unknown") if all_prices else "none",
            "quality": all_prices[0].get("_quality", "unknown") if all_prices else "UNAVAILABLE",
            "country": country.upper(),
            "window_duration": window_duration,
            "hours_analyzed": len(all_prices) if all_prices else 0,
        },
        "timestamp": now.isoformat(),
    }

    json_content = json.dumps(result, default=str)
    api_cache.set(cache_key, json_content, ttl=120)  # 2 min cache

    return Response(
        content=json_content,
        media_type="application/json",
        headers={"X-Cache": "MISS"}
    )
