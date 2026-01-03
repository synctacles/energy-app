# coding: utf-8
"""
ENTSO-E A75 Importer - Generation per PSR-type
Reads XML files from logs/entso_e_raw/ -> writes to raw_entso_e_a75
"""

import os
import sys
import time
from pathlib import Path
from datetime import datetime, timedelta, UTC
from typing import Optional

from lxml import etree
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.dialects.postgresql import insert

# Add repo root to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from synctacles_db.models import RawEntsoeA75
from synctacles_db.core.logging import get_logger

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://synctacles@localhost:5432/synctacles')

# Log directory configuration
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))

_LOGGER = get_logger(__name__)

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
    _LOGGER.info(f"A75 XML importer starting: {filepath.name}")
    start_time = time.time()

    try:
        _LOGGER.debug(f"Parsing XML file: {filepath}")
        tree = etree.parse(str(filepath))
        root = tree.getroot()
    except Exception as e:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75 XML parse failed after {elapsed:.2f}s: {type(e).__name__}: {e}")
        return 0, 1
    
    # Extract PSR-type
    psr_elem = root.find('.//ns:MktPSRType/ns:psrType', NS)
    if psr_elem is None:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75: No PSR-type found after {elapsed:.2f}s")
        return 0, 1

    psr_type = psr_elem.text

    # Extract TimeSeries -> Period
    period = root.find('.//ns:TimeSeries/ns:Period', NS)
    if period is None:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75: No Period found after {elapsed:.2f}s")
        return 0, 1

    # Start time
    start_elem = period.find('ns:timeInterval/ns:start', NS)
    if start_elem is None:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75: No start time found after {elapsed:.2f}s")
        return 0, 1

    start_ts = datetime.fromisoformat(start_elem.text.replace('Z', '+00:00'))

    # Resolution
    resolution_elem = period.find('ns:resolution', NS)
    if resolution_elem is None:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75: No resolution found after {elapsed:.2f}s")
        return 0, 1

    resolution_minutes = parse_resolution(resolution_elem.text)

    # Points
    points = period.findall('ns:Point', NS)
    if not points:
        elapsed = time.time() - start_time
        _LOGGER.debug(f"A75: No data points found after {elapsed:.2f}s")
        return 0, 0
    
    records = []
    for point in points:
        position = int(point.find('ns:position', NS).text)
        quantity = float(point.find('ns:quantity', NS).text)

        # Calculate timestamp: start + (position - 1) * resolution
        timestamp = start_ts + timedelta(minutes=(position - 1) * resolution_minutes)
        
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

        elapsed = time.time() - start_time
        _LOGGER.info(f"A75 XML importer completed: {len(records)} records (PSR={psr_type}) in {elapsed:.2f}s")
        return len(records), 0

    return 0, 0


def main():
    """Import all A75 files from collectors/entso_e_raw/"""
    _LOGGER.info("A75 XML importer batch starting")
    start_time = time.time()

    try:
        logs_dir = LOG_DIR / 'collectors' / 'entso_e_raw'

        if not logs_dir.exists():
            _LOGGER.error(f"Logs directory not found: {logs_dir}")
            return 1

        # Find all A75 files
        a75_files = sorted(logs_dir.glob('a75_NL_*.xml'))

        if not a75_files:
            _LOGGER.warning("No A75 files found to import")
            return 0

        _LOGGER.info(f"Found {len(a75_files)} A75 files to process")

        # Database connection
        engine = create_engine(DATABASE_URL)
        Session = sessionmaker(bind=engine)
        session = Session()

        total_inserted = 0
        total_failed = 0
        failed_files = []

        try:
            for filepath in a75_files:
                try:
                    inserted, failed = import_a75_file(filepath, session)
                    total_inserted += inserted
                    total_failed += failed
                except Exception as e:
                    _LOGGER.debug(f"Failed to import {filepath.name}: {type(e).__name__}")
                    failed_files.append(filepath.name)

        finally:
            session.close()

        elapsed = time.time() - start_time

        if failed_files:
            _LOGGER.warning(f"Failed to import {len(failed_files)} files")
            _LOGGER.debug(f"Failed files: {failed_files}")

        _LOGGER.info(f"A75 XML importer batch completed: {total_inserted} records, {total_failed} failures in {elapsed:.2f}s")

        return 0 if total_failed == 0 else 1

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75 batch importer failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise


if __name__ == '__main__':
    sys.exit(main())