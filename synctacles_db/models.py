"""
Normalized data models (synctacles_db)
Includes both raw and normalized data models
"""
from sqlalchemy import (
    TIMESTAMP,
    Boolean,
    Column,
    DateTime,
    Float,
    Index,
    Integer,
    Numeric,
    String,
    UniqueConstraint,
    func,
    text,
)
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()


# === RAW DATA MODELS ===

# ARCHIVED MODEL - Phase 3: Hard Delete (2026-01-11)
# A75 (generation) data collection discontinued for Energy Action Focus.
# Model retained ONLY for alembic migration compatibility.
class RawEntsoeA75(Base):
    """ARCHIVED: Raw ENTSO-E A75 Generation per PSR-type"""
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


# ARCHIVED MODEL - Phase 3: Hard Delete (2026-01-11)
# A65 (load) data collection discontinued for Energy Action Focus.
# Model retained ONLY for alembic migration compatibility.
class RawEntsoeA65(Base):
    """ARCHIVED: Raw ENTSO-E A65 Load (actual + forecast)"""
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


# ARCHIVED MODEL - Phase 3: Hard Delete (2026-01-11)
# TenneT integration removed from SYNCTACLES. BYO-key only in HA.
# Model retained ONLY for alembic migration compatibility.
class RawTennetBalance(Base):
    """ARCHIVED: Raw TenneT Balance Delta per platform

    TenneT data has been moved to BYO-key (Bring Your Own) model.
    This model is kept for alembic migration compatibility only.
    Data table: archive_raw_tennet_balance

    Phase 3 (2026-01-11): Code removed, model retained for migrations.
    """
    __tablename__ = 'archive_raw_tennet_balance'

    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), nullable=False)
    platform = Column(String(20), nullable=False)
    delta_mw = Column(Float, nullable=False)
    price_eur_mwh = Column(Float, nullable=True)
    source_file = Column(String(255), nullable=True)
    imported_at = Column(DateTime(timezone=True), server_default=func.now())

    __table_args__ = (
        Index('ix_raw_tennet_balance_timestamp', 'timestamp'),
        Index('ix_raw_tennet_balance_platform', 'platform'),
        {'extend_existing': True}
    )


# === NORMALIZED DATA MODELS ===

# ARCHIVED MODEL - Phase 3: Hard Delete (2026-01-11)
# A75 (generation) data collection discontinued for Energy Action Focus.
# Model retained ONLY for alembic migration compatibility.
class NormEntsoeA75(Base):
    """ARCHIVED: Normalized ENTSO-E A75 Generation Mix (pivoted)"""
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


# ARCHIVED MODEL - Phase 3: Hard Delete (2026-01-11)
# A65 (load) data collection discontinued for Energy Action Focus.
# Model retained ONLY for alembic migration compatibility.
class NormEntsoeA65(Base):
    """ARCHIVED: Normalized ENTSO-E A65 Load"""
    __tablename__ = 'norm_entso_e_a65'

    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(DateTime(timezone=True), primary_key=True, nullable=False)
    country = Column(String(2), nullable=False)

    actual_mw = Column(Float)
    forecast_mw = Column(Float, nullable=True)

    quality_status = Column(String(20))
    last_updated = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

class NormTennetBalance(Base):
    """ARCHIVED: Normalized TenneT Balance (aggregated across platforms)

    TenneT data has been moved to BYO-key (Bring Your Own) model in Home Assistant.
    This model is kept for historical reference only.
    Data table has been renamed to: archive_norm_tennet_balance
    """
    __tablename__ = 'archive_norm_tennet_balance'

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


# === PRICE CACHE (Issue #61) ===

class PriceCache(Base):
    """24h price cache for Tier 4 fallback.

    Stores consumer prices from any source for 24h persistence.
    Used when all live sources are unavailable.
    """
    __tablename__ = 'price_cache'

    id = Column(Integer, primary_key=True, autoincrement=True)
    timestamp = Column(TIMESTAMP(timezone=True), nullable=False)
    country = Column(String(2), nullable=False, server_default='NL')
    price_eur_kwh = Column(Numeric(10, 6), nullable=False)
    source = Column(String(50), nullable=False)  # enever, entsoe+lookup, energy-charts+lookup
    quality = Column(String(20), nullable=False)  # live, estimated, cached
    created_at = Column(TIMESTAMP(timezone=True), server_default=text('NOW()'))

    __table_args__ = (
        Index('idx_price_cache_timestamp', 'timestamp'),
        Index('idx_price_cache_country_timestamp', 'country', text('timestamp DESC')),
    )


# === FRANK PRICES (Database-backed Fallback Chain) ===

class FrankPrices(Base):
    """Frank Energie direct prices (Tier 1 in fallback chain).

    Collected 2x daily (07:00, 15:00 UTC) from Frank GraphQL API.
    Contains full consumer prices including all taxes and markups.
    """
    __tablename__ = 'frank_prices'

    timestamp = Column(TIMESTAMP(timezone=True), primary_key=True)
    price_eur_kwh = Column(Numeric(10, 6), nullable=False)
    market_price = Column(Numeric(10, 6), nullable=True)
    market_price_tax = Column(Numeric(10, 6), nullable=True)
    sourcing_markup = Column(Numeric(10, 6), nullable=True)
    energy_tax = Column(Numeric(10, 6), nullable=True)
    created_at = Column(TIMESTAMP(timezone=True), server_default=text('NOW()'))

    __table_args__ = (
        Index('idx_frank_prices_timestamp', text('timestamp DESC')),
    )


# DEPRECATED: EneverFrankPrices removed in KISS Migration v2.0.0
# Table 'enever_frank_prices' no longer used - coefficient server discontinued
# Data remains in DB for historical reference but no new data collected
