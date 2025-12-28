"""
Balance Endpoint
"""
from fastapi import APIRouter, Depends, Query, Response
from sqlalchemy.orm import Session
from sqlalchemy import desc
from datetime import datetime, timedelta, timezone

from synctacles_db.api.dependencies import get_db
from synctacles_db.api.models import BalanceResponse, BalanceData, MetaData
from synctacles_db.models import NormTennetBalance
from synctacles_db.cache import api_cache

router = APIRouter()

@router.get("/balance", response_model=BalanceResponse)
async def get_balance(
    hours: int = Query(72, ge=1, le=168),
    db: Session = Depends(get_db)
):
    """
    Get balance delta for Netherlands (TenneT)
    """
    # Build cache key
    cache_key = f"balance:NL:{hours}"

    # Check cache first
    cached = api_cache.get(cache_key)
    if cached:
        response = Response(
            content=cached,
            media_type="application/json"
        )
        response.headers["X-Cache"] = "HIT"
        return response

    now = datetime.now(timezone.utc)
    start_time = now - timedelta(hours=hours)

    records = db.query(NormTennetBalance).filter(
        NormTennetBalance.country == 'NL',
        NormTennetBalance.timestamp >= start_time
    ).order_by(desc(NormTennetBalance.timestamp)).all()

    if not records:
        result = BalanceResponse(
            data=[],
            meta=MetaData(
                source="TenneT",
                quality_status="NO_DATA",
                timestamp_utc=now,
                data_age_seconds=0
            )
        )
        api_cache.set(cache_key, result.model_dump_json(), ttl=120)
        response = Response(
            content=result.model_dump_json(),
            media_type="application/json"
        )
        response.headers["X-Cache"] = "MISS"
        return response

    latest = records[0]
    data_age = int((now - latest.timestamp).total_seconds())

    if data_age < 900:
        quality = "OK"
    elif data_age < 3600:
        quality = "STALE"
    else:
        quality = "NO_DATA"

    data = [
        BalanceData(
            timestamp=r.timestamp,
            delta_mw=r.delta_mw or 0.0,
            price_eur_mwh=r.price_eur_mwh
        )
        for r in records
    ]

    result = BalanceResponse(
        data=data,
        meta=MetaData(
            source="TenneT",
            quality_status=quality,
            timestamp_utc=latest.timestamp,
            data_age_seconds=data_age,
            next_update_utc=latest.timestamp + timedelta(minutes=5)
        )
    )

    # Cache the result with shorter TTL (2 minutes for more frequent updates)
    api_cache.set(cache_key, result.model_dump_json(), ttl=120)

    response = Response(
        content=result.model_dump_json(),
        media_type="application/json"
    )
    response.headers["X-Cache"] = "MISS"
    return response