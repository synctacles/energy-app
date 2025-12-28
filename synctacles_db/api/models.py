"""
Pydantic response models
"""
from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, List

class MetaData(BaseModel):
    """Quality metadata for all responses"""
    source: str = Field(..., description="Data source (ENTSO-E, TenneT, etc)")
    quality_status: str = Field(..., description="OK, STALE, NO_DATA")
    timestamp_utc: datetime = Field(..., description="Data timestamp")
    data_age_seconds: int = Field(..., description="Age in seconds")
    next_update_utc: Optional[datetime] = Field(None, description="Expected next update")

class GenerationMixData(BaseModel):
    """Single generation mix record"""
    timestamp: datetime
    biomass_mw: float = 0.0
    gas_mw: float = 0.0
    coal_mw: float = 0.0
    nuclear_mw: float = 0.0
    solar_mw: float = 0.0
    waste_mw: float = 0.0
    wind_offshore_mw: float = 0.0
    wind_onshore_mw: float = 0.0
    other_mw: float = 0.0
    total_mw: float

    class Config:
        populate_by_name = True

class GenerationMixResponse(BaseModel):
    """Generation mix endpoint response"""
    data: List[GenerationMixData]
    meta: MetaData

class LoadData(BaseModel):
    """Single load record"""
    timestamp: datetime
    actual_mw: float
    forecast_mw: Optional[float] = None

class LoadResponse(BaseModel):
    """Load endpoint response"""
    data: List[LoadData]
    meta: MetaData

class BalanceData(BaseModel):
    """Single balance record"""
    timestamp: datetime
    delta_mw: float
    price_eur_mwh: Optional[float] = None

class BalanceResponse(BaseModel):
    """Balance endpoint response"""
    data: List[BalanceData]
    meta: MetaData