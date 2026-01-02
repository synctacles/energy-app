# coding: utf-8
"""
TenneT Balance Delta Importer
Reads JSON files from logs/tennet_raw/ -> writes to raw_tennet_balance
"""
import os
import sys
import json
from pathlib import Path
from datetime import datetime, UTC
import logging

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.dialects.postgresql import insert

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from synctacles_db.models import RawTennetBalance

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://synctacles@localhost:5432/synctacles')

# Log directory configuration
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    force=True
)
logger = logging.getLogger(__name__)

PLATFORMS = ['aFRR', 'IGCC', 'MARI', 'mFRRda', 'PICASSO']


def import_tennet_file(filepath: Path, session) -> tuple[int, int]:
    """
    Import single TenneT JSON file
    Returns: (records_inserted, records_failed)
    """
    print(f"Processing: {filepath.name}")
    logger.info(f"Processing: {filepath.name}")
    
    try:
        with open(filepath, 'r') as f:
            data = json.load(f)
    except Exception as e:
        print(f"ERROR: JSON parse failed: {e}")
        logger.error(f"JSON parse failed: {e}")
        return 0, 1
    
    if not isinstance(data, list):
        print("ERROR: Expected JSON array")
        logger.error("Expected JSON array")
        return 0, 1
    
    records = []
    
    for item in data:
        # Parse timestamp
        timestamp_str = item.get('timestamp_start')
        if not timestamp_str:
            continue
        
        timestamp = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
        
        # Get mid_price from metadata (single value for all platforms)
        mid_price = None
        if 'metadata' in item and item['metadata']:
            mid_price = item['metadata'].get('mid_price')
        
        # Unpivot: 5 platforms × 2 directions = 10 columns -> 5 records
        for platform in PLATFORMS:
            power_in = item.get(f'power_{platform.lower()}_in', 0.0)
            power_out = item.get(f'power_{platform.lower()}_out', 0.0)
            
            # Convert to float (JSON may have strings)
            try:
                power_in = float(power_in) if power_in else 0.0
                power_out = float(power_out) if power_out else 0.0
            except (ValueError, TypeError):
                power_in = 0.0
                power_out = 0.0
            
            # Calculate delta: in - out (positive = import, negative = export)
            delta_mw = power_in - power_out
            
            records.append({
                'timestamp': timestamp,
                'platform': platform,
                'delta_mw': delta_mw,
                'price_eur_mwh': mid_price,
                'source_file': filepath.name,
                'imported_at': datetime.now(UTC)
            })
    
    # Upsert
    if records:
        stmt = insert(RawTennetBalance).values(records)
        stmt = stmt.on_conflict_do_update(
            index_elements=['timestamp', 'platform'],
            set_={
                'delta_mw': stmt.excluded.delta_mw,
                'price_eur_mwh': stmt.excluded.price_eur_mwh,
                'source_file': stmt.excluded.source_file,
                'imported_at': stmt.excluded.imported_at
            }
        )
        session.execute(stmt)
        session.commit()
        
        print(f"✓ Imported {len(records)} records (5 platforms)")
        logger.info(f"✓ Imported {len(records)} records")
        return len(records), 0
    
    return 0, 0


def main():
    """Import all TenneT files from collectors/tennet_raw/"""
    logs_dir = LOG_DIR / 'collectors' / 'tennet_raw'
    
    print(f"Looking in: {logs_dir}")
    
    if not logs_dir.exists():
        print("ERROR: Directory not found")
        logger.error(f"Logs directory not found: {logs_dir}")
        return 1
    
    # Find TenneT JSON files
    tennet_files = sorted(logs_dir.glob('tennet_balance_*.json'))
    
    if not tennet_files:
        print("WARNING: No files found")
        logger.warning("No TenneT files found")
        return 0
    
    print(f"Found {len(tennet_files)} TenneT files")
    logger.info(f"Found {len(tennet_files)} TenneT files")
    
    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()
    
    total_inserted = 0
    total_failed = 0
    
    try:
        for filepath in tennet_files:
            inserted, failed = import_tennet_file(filepath, session)
            total_inserted += inserted
            total_failed += failed
    finally:
        session.close()
    
    print(f"\n=== SUMMARY ===")
    print(f"Files processed: {len(tennet_files)}")
    print(f"Records inserted: {total_inserted}")
    print(f"Files failed: {total_failed}")
    
    logger.info("=== SUMMARY ===")
    logger.info(f"Files processed: {len(tennet_files)}")
    logger.info(f"Records inserted: {total_inserted}")
    logger.info(f"Files failed: {total_failed}")
    
    return 0 if total_failed == 0 else 1


if __name__ == '__main__':
    sys.exit(main())
