# coding: utf-8
"""
ENTSO-E A75 Importer - Generation per PSR-type
Reads XML files from logs/entso_e_raw/ -> writes to raw_entso_e_a75
"""

import os
import sys
from pathlib import Path
from datetime import datetime, timedelta, UTC
from typing import Optional
import logging

from lxml import etree
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.dialects.postgresql import insert

# Add repo root to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from sparkcrawler_db.models import RawEntsoeA75

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://synctacles@localhost:5432/synctacles')

# Log directory configuration
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    force=True
)
logger = logging.getLogger(__name__)

# XML namespace
NS = {'ns': 'urn:iec62325.351:tc57wg16:451-6:generationloaddocument:3:0'}


def parse_resolution(resolution: str) -> int:
    """Parse PT15M ? 15 minutes"""
    if resolution == 'PT15M':
        return 15
    elif resolution == 'PT60M':
        return 60
    else:
        raise ValueError(f"Unsupported resolution: {resolution}")


def import_a75_file(filepath: Path, session) -> tuple[int, int]:
    """
    Import single A75 XML file
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
    
    # Extract PSR-type
    psr_elem = root.find('.//ns:MktPSRType/ns:psrType', NS)
    if psr_elem is None:
        print("ERROR: No PSR-type found")
        logger.error("No PSR-type found in document")
        return 0, 1
    
    psr_type = psr_elem.text
    
    # Extract TimeSeries ? Period
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
        
        # Calculate timestamp: start + (position - 1) * resolution
        timestamp = start_time + timedelta(minutes=(position - 1) * resolution_minutes)
        
        records.append({
            'timestamp': timestamp,
            'country': 'NL',
            'psr_type': psr_type,
            'quantity_mw': quantity,
            'source_file': filepath.name,
            'imported_at': datetime.now(UTC)
        })
    
    # Upsert to database
    if records:
        stmt = insert(RawEntsoeA75).values(records)
        stmt = stmt.on_conflict_do_update(
            index_elements=['timestamp', 'country', 'psr_type'],
            set_={
                'quantity_mw': stmt.excluded.quantity_mw,
                'source_file': stmt.excluded.source_file,
                'imported_at': stmt.excluded.imported_at
            }
        )
        session.execute(stmt)
        session.commit()
        
        print(f"? Imported {len(records)} records (PSR={psr_type})")
        logger.info(f"? Imported {len(records)} records (PSR={psr_type})")
        return len(records), 0
    
    return 0, 0


def main():
    """Import all A75 files from collectors/entso_e_raw/"""
    logs_dir = LOG_DIR / 'collectors' / 'entso_e_raw'
    
    print(f"Looking in: {logs_dir}")
    print(f"Exists: {logs_dir.exists()}")
    
    if not logs_dir.exists():
        print(f"ERROR: Directory not found")
        logger.error(f"Logs directory not found: {logs_dir}")
        return 1
    
    # Find all A75 files
    a75_files = sorted(logs_dir.glob('a75_NL_*.xml'))
    
    if not a75_files:
        print("WARNING: No files found")
        logger.warning("No A75 files found")
        return 0
    
    print(f"Found {len(a75_files)} A75 files")
    logger.info(f"Found {len(a75_files)} A75 files")
    
    # Database connection
    engine = create_engine(DATABASE_URL)
    Session = sessionmaker(bind=engine)
    session = Session()
    
    total_inserted = 0
    total_failed = 0
    
    try:
        for filepath in a75_files:
            inserted, failed = import_a75_file(filepath, session)
            total_inserted += inserted
            total_failed += failed
    
    finally:
        session.close()
    
    print(f"\n=== SUMMARY ===")
    print(f"Files processed: {len(a75_files)}")
    print(f"Records inserted: {total_inserted}")
    print(f"Files failed: {total_failed}")
    
    logger.info(f"=== SUMMARY ===")
    logger.info(f"Files processed: {len(a75_files)}")
    logger.info(f"Records inserted: {total_inserted}")
    logger.info(f"Files failed: {total_failed}")
    
    return 0 if total_failed == 0 else 1


if __name__ == '__main__':
    sys.exit(main())