"""
synctacles_db/normalizers/normalize_entso_e_a65.py

Transform raw load data (actual + forecast) into merged normalized table.
"""

import logging
from datetime import datetime, timezone
from pathlib import Path
from sqlalchemy import create_engine, func, case
from sqlalchemy.dialects.postgresql import insert
from sqlalchemy.orm import sessionmaker

import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from synctacles_db.models import RawEntsoeA65
from synctacles_db.models import NormEntsoeA65
from config.settings import DATABASE_URL

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    force=True
)
logger = logging.getLogger(__name__)


def calculate_quality_status(latest_timestamp: datetime) -> str:
    """Calculate data quality based on age."""
    if latest_timestamp is None:
        return 'NO_DATA'
    
    now = datetime.now(timezone.utc)
    age_minutes = (now - latest_timestamp).total_seconds() / 60
    
    if age_minutes < 15:
        return 'OK'
    elif age_minutes < 1440:
        return 'STALE'
    else:
        return 'CACHED'


def normalize_a65_load():
    """Merge raw_entso_e_a65 actual + forecast → norm_entso_e_a65."""
    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()
    
    try:
        logger.info("Starting A65 normalization...")
        
        now = datetime.now(timezone.utc)
        latest_raw = session.query(func.max(RawEntsoeA65.timestamp)).filter(
            RawEntsoeA65.timestamp <= now
        ).scalar()

        quality_status = calculate_quality_status(latest_raw)        
        logger.info(f"Latest raw timestamp: {latest_raw}")
        logger.info(f"Quality status: {quality_status}")
        
        merge_query = session.query(
            RawEntsoeA65.timestamp,
            RawEntsoeA65.country,
            func.max(case((RawEntsoeA65.type == 'actual', RawEntsoeA65.quantity_mw))).label('actual_mw'),
            func.max(case((RawEntsoeA65.type == 'forecast', RawEntsoeA65.quantity_mw))).label('forecast_mw')
        ).group_by(
            RawEntsoeA65.timestamp,
            RawEntsoeA65.country
        ).order_by(
            RawEntsoeA65.timestamp
        )
        
        records = []
        for row in merge_query:
            records.append({
                'timestamp': row.timestamp,
                'country': row.country,
                'actual_mw': row.actual_mw,
                'forecast_mw': row.forecast_mw,
                'quality_status': quality_status
            })
        
        if not records:
            logger.warning("No records to normalize")
            return
        
        logger.info(f"Merged {len(records)} timestamp groups")
        
        stmt = insert(NormEntsoeA65).values(records)
        stmt = stmt.on_conflict_do_update(
            index_elements=['timestamp', 'country'],
            set_={
                'actual_mw': stmt.excluded.get('actual_mw'),
                'forecast_mw': stmt.excluded.get('forecast_mw'),
                'quality_status': stmt.excluded.get('quality_status')
            }
        )
        
        session.execute(stmt)
        session.commit()
        
        logger.info(f"✓ Normalized {len(records)} records to norm_entso_e_a65")
        
        sample = session.query(NormEntsoeA65).order_by(NormEntsoeA65.timestamp.desc()).first()
        if sample:
            logger.info(f"Sample: {sample.timestamp} | Actual: {sample.actual_mw} MW | Forecast: {sample.forecast_mw} MW")
        
    except Exception as e:
        session.rollback()
        logger.error(f"Normalization failed: {e}", exc_info=True)
        raise
    finally:
        session.close()


if __name__ == '__main__':
    normalize_a65_load()
