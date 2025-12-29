# coding: utf-8
"""
ENTSO-E A65 Importer - Total Load (actual + forecast)
Reads XML files from logs/entso_e_raw/ -> writes to raw_entso_e_a65
"""
import os
import sys
from pathlib import Path
from datetime import datetime, timedelta, UTC
import logging

from lxml import etree
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.dialects.postgresql import insert

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from synctacles_db.models import RawEntsoeA65

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://synctacles@localhost:5432/synctacles')

# Log directory configuration
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    force=True
)
logger = logging.getLogger(__name__)

NS = {'ns': 'urn:iec62325.351:tc57wg16:451-6:generationloaddocument:3:0'}


def parse_resolution(resolution: str) -> int:
    if resolution == 'PT15M':
        return 15
    elif resolution == 'PT60M':
        return 60
    else:
        raise ValueError(f"Unsupported resolution: {resolution}")


def import_a65_file(filepath: Path, session) -> tuple[int, int]:
    """
    Import single A65 XML file
    Returns: (records_inserted, records_failed)
    """
    print(f"Processing: {filepath.name}")
    logger.info(f"Processing: {filepath.name}")
    
    try:
        tree = etree.parse(str(filepath))
        root = tree.getroot()
    except Exception as e:
        print(f"ERROR: XML parse failed: {e}")
        logger.error(f"XML parse failed: {e}")
        return 0, 1
    
    # Detect type from filename (a65_NL_actual_* or a65_NL_forecast_*)
    if 'actual' in filepath.name.lower():
        load_type = 'actual'
    elif 'forecast' in filepath.name.lower():
        load_type = 'forecast'
    else:
        # Fallback: check businessType in XML
        business_type = root.find('.//ns:TimeSeries/ns:businessType', NS)
        if business_type is not None and business_type.text == 'A04':
            load_type = 'actual'
        else:
            load_type = 'forecast'
    
    # Extract TimeSeries -> Period
    period = root.find('.//ns:TimeSeries/ns:Period', NS)
    if period is None:
        print("ERROR: No Period found")
        logger.error("No Period found")
        return 0, 1
    
    # Start time
    start_elem = period.find('ns:timeInterval/ns:start', NS)
    if start_elem is None:
        print("ERROR: No start time")
        logger.error("No start time found")
        return 0, 1
    
    start_time = datetime.fromisoformat(start_elem.text.replace('Z', '+00:00'))
    
    # Resolution
    resolution_elem = period.find('ns:resolution', NS)
    if resolution_elem is None:
        print("ERROR: No resolution")
        logger.error("No resolution found")
        return 0, 1
    
    resolution_minutes = parse_resolution(resolution_elem.text)
    
    # Points
    points = period.findall('ns:Point', NS)
    if not points:
        print("WARNING: No points")
        logger.warning("No data points found")
        return 0, 0
    
    records = []
    for point in points:
        position = int(point.find('ns:position', NS).text)
        quantity = float(point.find('ns:quantity', NS).text)
        
        timestamp = start_time + timedelta(minutes=(position - 1) * resolution_minutes)
        
        records.append({
            'timestamp': timestamp,
            'country': 'NL',
            'type': load_type,
            'quantity_mw': quantity,
            'source_file': filepath.name,
            'imported_at': datetime.now(UTC)
        })
    
    # Upsert
    if records:
        stmt = insert(RawEntsoeA65).values(records)
        stmt = stmt.on_conflict_do_update(
            index_elements=['timestamp', 'country', 'type'],
            set_={
                'quantity_mw': stmt.excluded.quantity_mw,
                'source_file': stmt.excluded.source_file,
                'imported_at': stmt.excluded.imported_at
            }
        )
        session.execute(stmt)
        session.commit()
        
        print(f"✓ Imported {len(records)} records (type={load_type})")
        logger.info(f"✓ Imported {len(records)} records (type={load_type})")
        return len(records), 0
    
    return 0, 0


def main():
    """Import all A65 files from collectors/entso_e_raw/"""
    logs_dir = LOG_DIR / 'collectors' / 'entso_e_raw'
    
    print(f"Looking in: {logs_dir}")
    
    if not logs_dir.exists():
        print("ERROR: Directory not found")
        logger.error(f"Logs directory not found: {logs_dir}")
        return 1
    
    # Find A65 files (both naming patterns)
    a65_files = sorted(
        list(logs_dir.glob('a65_NL_*.xml')) +
        list(logs_dir.glob('entso_e_a65_*.xml'))
    )
    
    if not a65_files:
        print("WARNING: No files found")
        logger.warning("No A65 files found")
        return 0
    
    print(f"Found {len(a65_files)} A65 files")
    logger.info(f"Found {len(a65_files)} A65 files")
    
    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()
    
    total_inserted = 0
    total_failed = 0
    
    try:
        for filepath in a65_files:
            inserted, failed = import_a65_file(filepath, session)
            total_inserted += inserted
            total_failed += failed
    finally:
        session.close()
    
    print(f"\n=== SUMMARY ===")
    print(f"Files processed: {len(a65_files)}")
    print(f"Records inserted: {total_inserted}")
    print(f"Files failed: {total_failed}")
    
    logger.info("=== SUMMARY ===")
    logger.info(f"Files processed: {len(a65_files)}")
    logger.info(f"Records inserted: {total_inserted}")
    logger.info(f"Files failed: {total_failed}")
    
    return 0 if total_failed == 0 else 1


if __name__ == '__main__':
    sys.exit(main())
