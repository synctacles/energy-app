#!/usr/bin/env python3
"""
ENTSO-E A44 Importer: CSV -> raw_entso_e_a44
"""

import sys
from pathlib import Path
from datetime import datetime, timezone
import csv

# Add parent to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from synctacles_db.models import RawEntsoeA44
import logging

LOG_DIR = Path('/opt/synctacles/logs/collectors/entso_e_raw')
DB_URL = "postgresql://synctacles@localhost:5432/synctacles"

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

engine = create_engine(DB_URL)
Session = sessionmaker(bind=engine)

def import_csv_file(filepath):
    """Import single CSV file to raw table"""
    session = Session()
    imported = 0
    skipped = 0
    
    try:
        with open(filepath, 'r') as f:
            reader = csv.DictReader(f)
            
            for row in reader:
                timestamp = datetime.fromisoformat(row['timestamp'])
                price = float(row['price_eur_mwh'])
                
                # Check if exists
                exists = session.query(RawEntsoeA44).filter(
                    RawEntsoeA44.timestamp == timestamp,
                    RawEntsoeA44.country == 'NL'
                ).first()
                
                if not exists:
                    record = RawEntsoeA44(
                        timestamp=timestamp,
                        country='NL',
                        price_eur_mwh=price,
                        xml_file=filepath.name
                    )
                    session.add(record)
                    imported += 1
                else:
                    skipped += 1
        
        session.commit()
        logger.info(f"{filepath.name}: Imported {imported}, Skipped {skipped}")
        
    except Exception as e:
        session.rollback()
        logger.error(f"Import failed: {e}")
        raise
    finally:
        session.close()
    
    return imported, skipped

def main():
    logger.info("=== A44 Importer ===")
    
    # Find all CSV files
    csv_files = sorted(LOG_DIR.glob('a44_NL_prices_*.csv'))
    
    if not csv_files:
        logger.warning("No CSV files found")
        return
    
    total_imported = 0
    total_skipped = 0
    
    for csv_file in csv_files:
        imported, skipped = import_csv_file(csv_file)
        total_imported += imported
        total_skipped += skipped
    
    logger.info(f"=== Total: {total_imported} imported, {total_skipped} skipped ===")

if __name__ == '__main__':
    main()
