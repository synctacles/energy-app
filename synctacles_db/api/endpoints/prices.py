"""
ENTSO-E A44: Day-ahead electricity prices with 4-tier fallback

Implements 4-tier fallback strategy:
1. Fresh ENTSO-E data (< 15 min) → allow_go_action=True
2. Stale ENTSO-E data (< 60 min) → allow_go_action=True
3. Energy-Charts fallback → allow_go_action=False (CRITICAL!)
4. Cache fallback → allow_go_action=False

CRITICAL RULE: Energy-Charts prices MUST NEVER trigger GO actions!
"""

import logging

# Local imports AFTER external deps
import sys
from datetime import UTC, datetime, timedelta
from pathlib import Path

from fastapi import APIRouter, Query
from pydantic import BaseModel, Field
from sqlalchemy import create_engine, desc
from sqlalchemy.orm import sessionmaker
from starlette.responses import Response

sys.path.insert(0, str(Path(__file__).parent.parent.parent.parent))

from config.settings import DATABASE_URL
from synctacles_db.cache import api_cache
from synctacles_db.fallback.fallback_manager import FallbackManager
from synctacles_db.models import NormEntsoeA44

_LOGGER = logging.getLogger(__name__)

router = APIRouter()

DB_URL = DATABASE_URL
engine = create_engine(DB_URL)
Session = sessionmaker(bind=engine)


class ExpectedRange(BaseModel):
    """Expected price range for anomaly detection."""

    low: float
    high: float
    expected: float


class ReferenceData(BaseModel):
    """Reference data for HA anomaly detection (KISS Migration)."""

    source: str
    tier: int
    expected_range: ExpectedRange
    timestamp: str | None = None
    market: dict | None = None


class PriceRecord(BaseModel):
    timestamp: datetime
    price_eur_mwh: float
    # Pydantic v2: use Field with serialization_alias for underscore-prefixed fields
    reference: ReferenceData | None = Field(
        default=None, serialization_alias="_reference"
    )


class MetaData(BaseModel):
    source: str
    quality_status: str
    data_age_seconds: int
    count: int
    allow_go_action: bool  # Flag for GO action safety


class PricesResponse(BaseModel):
    data: list[PriceRecord]
    meta: MetaData


@router.get("/prices", response_model=PricesResponse)
async def get_prices(country: str = Query("nl"), hours: int = Query(48, ge=1, le=168)):
    """
    Day-ahead electricity prices with 4-tier fallback.

    Response includes 'allow_go_action' flag:
    - True: Safe for automation (ENTSO-E only)
    - False: DO NOT automate (Energy-Charts, cache, or fallback)
    """

    cache_key = f"prices:{country.upper()}:{hours}"
    cached_response = api_cache.get(cache_key)

    if cached_response:
        return Response(
            content=cached_response,
            media_type="application/json",
            headers={
                "X-Cache": "HIT",
                "X-Data-Source": "Cache",
            },
        )

    session = Session()
    now = datetime.now(UTC)

    # Get today 00:00 + tomorrow 23:59 (48h day-ahead data)
    start_of_today = now.replace(hour=0, minute=0, second=0, microsecond=0)
    end_of_tomorrow = start_of_today + timedelta(days=2)

    # Query ENTSO-E database
    try:
        records = (
            session.query(NormEntsoeA44)
            .filter(NormEntsoeA44.country == country.upper())
            .filter(NormEntsoeA44.timestamp >= start_of_today)
            .filter(NormEntsoeA44.timestamp < end_of_tomorrow)
            .order_by(desc(NormEntsoeA44.timestamp))
            .all()
        )
    finally:
        session.close()

    # Calculate database age
    db_age_minutes = 999
    if records:
        latest_timestamp = records[0].timestamp
        db_age_minutes = int((now - latest_timestamp).total_seconds() / 60)

    # Use FallbackManager for 4-tier fallback with GO action control
    try:
        db_data = (
            [
                {"timestamp": r.timestamp.isoformat(), "price_eur_mwh": r.price_eur_mwh}
                for r in records
            ]
            if records
            else None
        )

        (
            data,
            source,
            quality,
            allow_go_action,
        ) = await FallbackManager.get_prices_with_fallback(
            db_results=db_data, db_age_minutes=db_age_minutes, country=country.lower()
        )
    except Exception as err:
        _LOGGER.error(f"FallbackManager error: {err}")
        # Fallback to database if available
        if records:
            data = [
                {"timestamp": r.timestamp.isoformat(), "price_eur_mwh": r.price_eur_mwh}
                for r in records
            ]
            source = "ENTSO-E"
            quality = "STALE"
            allow_go_action = True
        else:
            data = None
            source = "None"
            quality = "UNAVAILABLE"
            allow_go_action = False

    # Build response
    if data:
        price_records = []
        for i, d in enumerate(data):
            record_kwargs = {
                "timestamp": datetime.fromisoformat(
                    d["timestamp"].replace("Z", "+00:00")
                ),
                "price_eur_mwh": float(d["price_eur_mwh"]),
            }
            # Include reference only on first record (KISS Migration anomaly detection)
            if i == 0 and "_reference" in d:
                ref = d["_reference"]
                record_kwargs["reference"] = ReferenceData(
                    source=ref.get("source", "unknown"),
                    tier=ref.get("tier", 0),
                    expected_range=ExpectedRange(
                        low=ref["expected_range"]["low"],
                        high=ref["expected_range"]["high"],
                        expected=ref["expected_range"].get("expected", 0),
                    ),
                    timestamp=ref.get("timestamp"),
                    market=ref.get("market"),
                )
            price_records.append(PriceRecord(**record_kwargs))

        result = PricesResponse(
            data=price_records,
            meta=MetaData(
                source=source,
                quality_status=quality,
                data_age_seconds=db_age_minutes * 60
                if db_age_minutes < 999
                else 999999,
                count=len(price_records),
                allow_go_action=allow_go_action,  # Safety flag
            ),
        )
    else:
        result = PricesResponse(
            data=[],
            meta=MetaData(
                source=source,
                quality_status=quality,
                data_age_seconds=999999,
                count=0,
                allow_go_action=False,  # Never automate when NO_DATA
            ),
        )

    # Serialize and cache (by_alias=True for '_reference', exclude_none to omit null fields)
    json_content = result.model_dump_json(by_alias=True, exclude_none=True)
    cache_ttl = 60 if quality == "UNAVAILABLE" else 300
    api_cache.set(cache_key, json_content, ttl=cache_ttl)

    # Return with informative headers
    return Response(
        content=json_content,
        media_type="application/json",
        headers={
            "X-Cache": "MISS",
            "X-Data-Source": source,
            "X-Data-Quality": quality,
            "X-Allow-GO": "true" if allow_go_action else "false",
        },
    )
