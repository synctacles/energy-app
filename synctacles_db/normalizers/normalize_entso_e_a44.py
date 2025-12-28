#!/usr/bin/env python3
"""
ENTSO-E A44 Normalizer: raw -> norm (with forward fill)
"""

import sys
from pathlib import Path
from datetime import datetime, timezone

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from synctacles_db.models import RawEntsoeA44, NormEntsoeA44
import logging

DB_URL = "postgresql://synctacles@localhost:5432/synctacles"

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

engine = create_engine(DB_URL)
Session = sessionmaker(bind=engine)

def get_previous_value(session, timestamp, country):
    """Get most recent price before given timestamp"""
    prev = session.query(NormEntsoeA44)\
        .filter(NormEntsoeA44.timestamp < timestamp)\
        .filter(NormEntsoeA44.country == country)\
        .order_by(NormEntsoeA44.timestamp.desc())\
        .first()
    
    if prev:
        return prev.price_eur_mwh
    return None

def normalize_prices():
    """Normalize raw prices to norm table with forward fill"""
    session = Session()
    normalized = 0
    forward_filled = 0
    
    try:
        # Get all raw records not yet normalized
        raw_records = session.query(RawEntsoeA44)\
            .filter(RawEntsoeA44.country == 'NL')\
            .order_by(RawEntsoeA44.timestamp)\
            .all()
        
        for raw in raw_records:
            # Check if already exists
            exists = session.query(NormEntsoeA44).filter(
                NormEntsoeA44.timestamp == raw.timestamp,
                NormEntsoeA44.country == raw.country
            ).first()
            
            if exists:
                continue
            
            # Determine quality
            if raw.price_eur_mwh is not None and raw.price_eur_mwh > 0:
                # Fresh ENTSO-E data
                norm = NormEntsoeA44(
                    timestamp=raw.timestamp,
                    country=raw.country,
                    price_eur_mwh=raw.price_eur_mwh,
                    data_source='ENTSO-E',
                    data_quality='OK',
                    needs_backfill=False
                )
                normalized += 1
            else:
                # Missing data - forward fill
                prev_price = get_previous_value(session, raw.timestamp, raw.country)
                
                if prev_price:
                    norm = NormEntsoeA44(
                        timestamp=raw.timestamp,
                        country=raw.country,
                        price_eur_mwh=prev_price,
                        data_source='ENTSO-E',
                        data_quality='FORWARD_FILL',
                        needs_backfill=True
                    )
                    forward_filled += 1
                else:
                    # No previous value - skip
                    logger.warning(f"No previous value for {raw.timestamp}, skipping")
                    continue
            
            session.add(norm)
        
        session.commit()
        logger.info(f"Normalized: {normalized} OK, {forward_filled} forward-filled")
        
    except Exception as e:
        session.rollback()
        logger.error(f"Normalization failed: {e}")
        raise
    finally:
        session.close()

def main():
    logger.info("=== A44 Normalizer ===")
    normalize_prices()
    logger.info("=== Complete ===")

if __name__ == '__main__':
    main()
