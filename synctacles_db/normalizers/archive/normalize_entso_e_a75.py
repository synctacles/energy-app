"""
synctacles_db/normalizers/normalize_entso_e_a75.py

Transform raw generation data into pivoted normalized table.
"""

import sys
import time
from datetime import UTC, datetime
from pathlib import Path

from sqlalchemy import case, create_engine, func
from sqlalchemy.dialects.postgresql import insert
from sqlalchemy.orm import sessionmaker

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from config.settings import DATABASE_URL
from synctacles_db.core.logging import get_logger
from synctacles_db.models import NormEntsoeA75, RawEntsoeA75
from synctacles_db.normalizers.base import validate_db_connection

_LOGGER = get_logger(__name__)

# Validate database connection at startup
validate_db_connection()


def calculate_quality_status(latest_timestamp: datetime) -> str:
    """Calculate data quality based on age."""
    if latest_timestamp is None:
        return 'NO_DATA'

    now = datetime.now(UTC)
    age_minutes = (now - latest_timestamp).total_seconds() / 60

    if age_minutes < 15:
        return 'OK'
    elif age_minutes < 1440:
        return 'STALE'
    else:
        return 'CACHED'


def normalize_a75_generation():
    """Pivot raw_entso_e_a75 → norm_entso_e_a75."""
    _LOGGER.info("A75 normalizer starting")
    start_time = time.time()

    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()

    try:
        latest_raw = session.query(func.max(RawEntsoeA75.timestamp)).scalar()
        quality_status = calculate_quality_status(latest_raw)

        _LOGGER.debug(f"Latest raw timestamp: {latest_raw}")
        _LOGGER.debug(f"Quality status: {quality_status}")

        pivot_query = session.query(
            RawEntsoeA75.timestamp,
            RawEntsoeA75.country,
            func.max(case((RawEntsoeA75.psr_type == 'B01', RawEntsoeA75.quantity_mw))).label('b01_biomass_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B04', RawEntsoeA75.quantity_mw))).label('b04_gas_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B05', RawEntsoeA75.quantity_mw))).label('b05_coal_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B14', RawEntsoeA75.quantity_mw))).label('b14_nuclear_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B16', RawEntsoeA75.quantity_mw))).label('b16_solar_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B17', RawEntsoeA75.quantity_mw))).label('b17_waste_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B18', RawEntsoeA75.quantity_mw))).label('b18_wind_offshore_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B19', RawEntsoeA75.quantity_mw))).label('b19_wind_onshore_mw'),
            func.max(case((RawEntsoeA75.psr_type == 'B20', RawEntsoeA75.quantity_mw))).label('b20_other_mw'),
            func.sum(RawEntsoeA75.quantity_mw).label('total_mw')
        ).group_by(
            RawEntsoeA75.timestamp,
            RawEntsoeA75.country
        ).order_by(
            RawEntsoeA75.timestamp
        )

        records = []
        for row in pivot_query:
            records.append({
                'timestamp': row.timestamp,
                'country': row.country,
                'b01_biomass_mw': row.b01_biomass_mw,
                'b04_gas_mw': row.b04_gas_mw,
                'b05_coal_mw': row.b05_coal_mw,
                'b14_nuclear_mw': row.b14_nuclear_mw,
                'b16_solar_mw': row.b16_solar_mw,
                'b17_waste_mw': row.b17_waste_mw,
                'b18_wind_offshore_mw': row.b18_wind_offshore_mw,
                'b19_wind_onshore_mw': row.b19_wind_onshore_mw,
                'b20_other_mw': row.b20_other_mw,
                'total_mw': row.total_mw,
                'quality_status': quality_status
            })

        if not records:
            _LOGGER.warning("No records to normalize")
            return

        _LOGGER.debug(f"Pivoted {len(records)} timestamp groups")

        stmt = insert(NormEntsoeA75).values(records)
        stmt = stmt.on_conflict_do_update(
            index_elements=['timestamp', 'country'],
            set_={
                'b01_biomass_mw': stmt.excluded.get('b01_biomass_mw'),
                'b04_gas_mw': stmt.excluded.get('b04_gas_mw'),
                'b05_coal_mw': stmt.excluded.get('b05_coal_mw'),
                'b14_nuclear_mw': stmt.excluded.get('b14_nuclear_mw'),
                'b16_solar_mw': stmt.excluded.get('b16_solar_mw'),
                'b17_waste_mw': stmt.excluded.get('b17_waste_mw'),
                'b18_wind_offshore_mw': stmt.excluded.get('b18_wind_offshore_mw'),
                'b19_wind_onshore_mw': stmt.excluded.get('b19_wind_onshore_mw'),
                'b20_other_mw': stmt.excluded.get('b20_other_mw'),
                'total_mw': stmt.excluded.get('total_mw'),
                'quality_status': stmt.excluded.get('quality_status')
            }
        )

        session.execute(stmt)
        session.commit()

        elapsed = time.time() - start_time
        _LOGGER.info(f"A75 normalizer completed: {len(records)} records normalized in {elapsed:.2f}s")

        sample = session.query(NormEntsoeA75).order_by(NormEntsoeA75.timestamp.desc()).first()
        if sample:
            _LOGGER.debug(f"Sample: {sample.timestamp} | Total: {sample.total_mw} MW | Solar: {sample.b16_solar_mw} MW")

    except Exception as e:
        session.rollback()
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75 normalizer failed after {elapsed:.2f}s: {type(e).__name__}: {e}")
        raise
    finally:
        session.close()


if __name__ == '__main__':
    normalize_a75_generation()
