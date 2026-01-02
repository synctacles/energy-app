"""
synctacles_db/normalizers/normalize_tennet_balance.py

Aggregate TenneT balance data across 5 platforms.
"""

import logging
from datetime import datetime, timezone
from pathlib import Path
from sqlalchemy import create_engine, func
from sqlalchemy.dialects.postgresql import insert
from sqlalchemy.orm import sessionmaker

import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from synctacles_db.models import RawTennetBalance
from synctacles_db.models import NormTennetBalance
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


def normalize_tennet_balance():
    """Aggregate raw_tennet_balance (5 platforms) → norm_tennet_balance."""
    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()
    
    try:
        logger.info("Starting TenneT balance normalization...")
        
        latest_raw = session.query(func.max(RawTennetBalance.timestamp)).scalar()
        quality_status = calculate_quality_status(latest_raw)
        
        logger.info(f"Latest raw timestamp: {latest_raw}")
        logger.info(f"Quality status: {quality_status}")
        
        # Aggregate query: sum delta across platforms
        agg_query = session.query(
            RawTennetBalance.timestamp,
            func.sum(RawTennetBalance.delta_mw).label('delta_mw'),
            func.avg(RawTennetBalance.price_eur_mwh).label('price_eur_mwh')
        ).group_by(
            RawTennetBalance.timestamp
        ).order_by(
            RawTennetBalance.timestamp
        )
        
        records = []
        for row in agg_query:
            records.append({
                'timestamp': row.timestamp,
                'country': 'NL',
                'delta_mw': row.delta_mw,
                'price_eur_mwh': row.price_eur_mwh,
                'quality_status': quality_status
            })
        
        if not records:
            logger.warning("No records to normalize")
            return
        
        logger.info(f"Aggregated {len(records)} timestamp groups")
        
        # Upsert
        stmt = insert(NormTennetBalance).values(records)
        stmt = stmt.on_conflict_do_update(
            index_elements=['timestamp','country'],
            set_={
                'delta_mw': stmt.excluded.get('delta_mw'),
                'price_eur_mwh': stmt.excluded.get('price_eur_mwh'),
                'quality_status': stmt.excluded.get('quality_status')
            }
        )
        
        session.execute(stmt)
        session.commit()
        
        logger.info(f"✓ Normalized {len(records)} records to norm_tennet_balance")
        
        # Sample output
        sample = session.query(NormTennetBalance).order_by(NormTennetBalance.timestamp.desc()).first()
        if sample:
            logger.info(f"Sample: {sample.timestamp} | Delta: {sample.delta_mw} MW | Price: {sample.price_eur_mwh} EUR/MWh")
        
    except Exception as e:
        session.rollback()
        logger.error(f"Normalization failed: {e}", exc_info=True)
        raise
    finally:
        session.close()


if __name__ == '__main__':
    normalize_tennet_balance()
