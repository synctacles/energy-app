"""
Normalized data models (synctacles_db)
"""
import sqlalchemy as sa
from sqlalchemy import Column, Integer, String, Float, DateTime, func
from sqlalchemy import Column, Integer, String, Float, DateTime, Numeric, Boolean, TIMESTAMP, Index, UniqueConstraint, func, text
from sqlalchemy.ext.declarative import declarative_base
Base = declarative_base()

class NormEntsoeA75(Base):
    """Normalized ENTSO-E A75 Generation Mix (pivoted)"""
    __tablename__ = 'norm_entso_e_a75'
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), primary_key=True, nullable=False)
    country = Column(String(2), nullable=False)
    
    # Generation by PSR type (MW)
    b01_biomass_mw = Column(Float)
    b04_gas_mw = Column(Float)
    b05_coal_mw = Column(Float)
    b14_nuclear_mw = Column(Float)
    b16_solar_mw = Column(Float)
    b17_waste_mw = Column(Float)
    b18_wind_offshore_mw = Column(Float)
    b19_wind_onshore_mw = Column(Float)
    b20_other_mw = Column(Float)
    total_mw = Column(Float)
    
    # Metadata
    quality_status = Column(String(20))
    last_updated = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

class NormEntsoeA65(Base):
    """Normalized ENTSO-E A65 Load"""
    __tablename__ = 'norm_entso_e_a65'
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), primary_key=True, nullable=False)
    country = Column(String(2), nullable=False)
    
    actual_mw = Column(Float)
    forecast_mw = Column(Float, nullable=True)
    
    quality_status = Column(String(20))
    last_updated = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

class NormTennetBalance(Base):
    """Normalized TenneT Balance (aggregated across platforms)"""
    __tablename__ = 'norm_tennet_balance'
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), primary_key=True, nullable=False)
    country = Column(String(2), nullable=False, default='NL')
    
    # Aggregated balance
    delta_mw = Column(Float, nullable=False)
    price_eur_mwh = Column(Float, nullable=True)
    
    # Metadata
    quality_status = Column(String(20), nullable=True)
    last_updated = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

class FetchLog(Base):
    """Audit log for API fetches"""
    __tablename__ = 'fetch_log'
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    source = Column(String(50), nullable=False)
    fetch_time = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    status = Column(String(20))
    records_fetched = Column(Integer)
    error_message = Column(String(500), nullable=True)


# === ENTSO-E A44: Day-Ahead Prices ===

class RawEntsoeA44(Base):
    __tablename__ = 'raw_entso_e_a44'
    
    id = Column(Integer, primary_key=True)
    timestamp = Column(TIMESTAMP(timezone=True), nullable=False)
    country = Column(String(2), nullable=False)
    price_eur_mwh = Column(Numeric(10, 2))
    xml_file = Column(String(255))
    created_at = Column(TIMESTAMP(timezone=True), server_default=text('NOW()'))


class NormEntsoeA44(Base):
    __tablename__ = 'norm_entso_e_a44'
    
    id = Column(Integer, primary_key=True)
    timestamp = Column(TIMESTAMP(timezone=True), nullable=False)
    country = Column(String(2), nullable=False)
    price_eur_mwh = Column(Numeric(10, 2))
    
    # Fallback support
    data_source = Column(String(20), server_default='ENTSO-E')
    data_quality = Column(String(20), server_default='OK')
    needs_backfill = Column(Boolean, server_default='false')
    
    created_at = Column(TIMESTAMP(timezone=True), server_default=text('NOW()'))
    
    __table_args__ = (
        UniqueConstraint('timestamp', 'country', name='uq_prices_time_country'),
        Index('idx_prices_country_time', 'country', text('timestamp DESC'))
    )
