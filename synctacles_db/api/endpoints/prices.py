import os
"""
ENTSO-E A44: Day-ahead electricity prices
"""

from fastapi import APIRouter, Query
from pydantic import BaseModel
from typing import List
from datetime import datetime, timezone, timedelta
from sqlalchemy import create_engine, desc
from sqlalchemy.orm import sessionmaker
from starlette.responses import Response

# Local imports AFTER external deps
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent.parent.parent))

from synctacles_db.models import NormEntsoeA44
from synctacles_db.cache import api_cache
from synctacles_db.api.cache import cached, prices_cache

router = APIRouter()

DB_URL = os.getenv("DATABASE_URL", "postgresql://synctacles@localhost:5432/synctacles")
engine = create_engine(DB_URL)
Session = sessionmaker(bind=engine)

class PriceRecord(BaseModel):
    timestamp: datetime
    price_eur_mwh: float
    data_quality: str

class MetaData(BaseModel):
    source: str
    quality_status: str
    data_age_seconds: int
    next_update_utc: datetime
    count: int

class PricesResponse(BaseModel):
    data: List[PriceRecord]
    meta: MetaData

@router.get("/prices", response_model=PricesResponse)
def get_prices(
    country: str = Query("NL"),
    hours: int = Query(48, ge=1, le=168)
):
    """Day-ahead electricity prices"""
    
    cache_key = f"prices:{country}:{hours}"
    cached = api_cache.get(cache_key)
    
    if cached:
        return Response(content=cached, media_type="application/json", headers={"X-Cache": "HIT"})
    
    session = Session()
    now = datetime.now(timezone.utc)
    start_time = now - timedelta(hours=hours)
    
    records = session.query(NormEntsoeA44)\
        .filter(NormEntsoeA44.country == country)\
        .filter(NormEntsoeA44.timestamp >= start_time)\
        .filter(NormEntsoeA44.timestamp <= now).\
        order_by(desc(NormEntsoeA44.timestamp))\
        .all()
    
    session.close()
    
    if not records:
        result = PricesResponse(
            data=[],
            meta=MetaData(
                source="ENTSO-E",
                quality_status="NO_DATA",
                data_age_seconds=999999,
                next_update_utc=now,
                count=0
            )
        )
        json_content = result.model_dump_json()
        api_cache.set(cache_key, json_content, ttl=60)
        return Response(content=json_content, media_type="application/json", headers={"X-Cache": "MISS"})
    
    latest = records[0]
    data_age = int((now - latest.timestamp).total_seconds())
    
    if data_age < 900:
        quality_status = "OK"
    elif data_age < 3600:
        quality_status = "STALE"
    else:
        quality_status = "NO_DATA"
    
    result = PricesResponse(
        data=[PriceRecord(
            timestamp=r.timestamp,
            price_eur_mwh=r.price_eur_mwh,
            data_quality=r.data_quality
        ) for r in records],
        meta=MetaData(
            source="ENTSO-E",
            quality_status=quality_status,
            data_age_seconds=data_age,
            next_update_utc=now + timedelta(minutes=15),
            count=len(records)
        )
    )
    
    json_content = result.model_dump_json()
    api_cache.set(cache_key, json_content, ttl=300)
    
    return Response(content=json_content, media_type="application/json", headers={"X-Cache": "MISS"})
