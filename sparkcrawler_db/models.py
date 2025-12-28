
"""
Raw data models for SparkCrawler database layer.
All raw_* tables store unprocessed API responses.
"""

from sqlalchemy import Column, Integer, String, Float, DateTime, Index
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.sql import func

Base = declarative_base()


class RawEntsoeA75(Base):
    """Raw ENTSO-E A75 Generation per PSR-type"""
    __tablename__ = 'raw_entso_e_a75'
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), nullable=False)
    country = Column(String(2), nullable=False)
    psr_type = Column(String(3), nullable=False)  # B01, B04, etc.
    quantity_mw = Column(Float, nullable=False)
    source_file = Column(String(255), nullable=True)
    imported_at = Column(DateTime(timezone=True), server_default=func.now())
    
    __table_args__ = (
        Index('ix_raw_entso_e_a75_timestamp', 'timestamp'),
        Index('ix_raw_entso_e_a75_psr_type', 'psr_type'),
        {'extend_existing': True}
    )


class RawEntsoeA65(Base):
    """Raw ENTSO-E A65 Load (actual + forecast)"""
    __tablename__ = 'raw_entso_e_a65'
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), nullable=False)
    country = Column(String(2), nullable=False)
    type = Column(String(20), nullable=False)  # 'actual' or 'forecast'
    quantity_mw = Column(Float, nullable=False)
    source_file = Column(String(255), nullable=True)
    imported_at = Column(DateTime(timezone=True), server_default=func.now())
    
    __table_args__ = (
        Index('ix_raw_entso_e_a65_timestamp', 'timestamp'),
        Index('ix_raw_entso_e_a65_type', 'type'),
        {'extend_existing': True}
    )


class RawTennetBalance(Base):
    """Raw TenneT Balance Delta per platform"""
    __tablename__ = 'raw_tennet_balance'
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), nullable=False)
    platform = Column(String(20), nullable=False)  # aFRR, IGCC, MARI, mFRRda, PICASSO
    delta_mw = Column(Float, nullable=False)
    price_eur_mwh = Column(Float, nullable=True)
    source_file = Column(String(255), nullable=True)
    imported_at = Column(DateTime(timezone=True), server_default=func.now())
    
    __table_args__ = (
        Index('ix_raw_tennet_balance_timestamp', 'timestamp'),
        Index('ix_raw_tennet_balance_platform', 'platform'),
        {'extend_existing': True}
    )
