#!/usr/bin/env python3
"""
ENTSO-E A44 Importer: CSV -> raw_entso_e_a44
"""

import sys
import time
from pathlib import Path
from datetime import datetime, timezone
import csv

# Add parent to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from synctacles_db.models import RawEntsoeA44
from synctacles_db.core.logging import get_logger
from config.settings import DATABASE_URL, LOG_PATH

LOG_DIR = Path(LOG_PATH) / 'collectors' / 'entso_e_raw'

_LOGGER = get_logger(__name__)

engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)

def import_csv_file(filepath):
    """Import single CSV file to raw table"""
    _LOGGER.info(f"A44 CSV importer starting: {filepath.name}")
    start_time = time.time()

    session = Session()
    imported = 0
    skipped = 0

    try:
        _LOGGER.debug(f"Parsing CSV file: {filepath}")
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

        elapsed = time.time() - start_time

        if skipped > 0:
            _LOGGER.debug(f"Skipped {skipped} duplicate records")

        _LOGGER.info(f"A44 CSV importer completed: {imported} inserted, {skipped} skipped in {elapsed:.2f}s")

    except Exception as e:
        session.rollback()
        elapsed = time.time() - start_time
        _LOGGER.error(f"A44 CSV import failed after {elapsed:.2f}s: {type(e).__name__}: {e}")
        raise
    finally:
        session.close()

    return imported, skipped

def main():
    _LOGGER.info("A44 CSV importer batch starting")
    start_time = time.time()

    try:
        # Find all CSV files
        csv_files = sorted(LOG_DIR.glob('a44_NL_prices_*.csv'))

        if not csv_files:
            _LOGGER.warning("No A44 CSV files found to import")
            return

        _LOGGER.info(f"Found {len(csv_files)} A44 CSV files to process")

        total_imported = 0
        total_skipped = 0
        failed_files = []

        for csv_file in csv_files:
            try:
                imported, skipped = import_csv_file(csv_file)
                total_imported += imported
                total_skipped += skipped
            except Exception as e:
                _LOGGER.debug(f"Failed to import {csv_file.name}: {type(e).__name__}")
                failed_files.append(csv_file.name)

        elapsed = time.time() - start_time

        if failed_files:
            _LOGGER.warning(f"Failed to import {len(failed_files)} files")
            _LOGGER.debug(f"Failed files: {failed_files}")

        _LOGGER.info(f"A44 CSV importer batch completed: {total_imported} imported, {total_skipped} skipped in {elapsed:.2f}s")

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A44 batch importer failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise

if __name__ == '__main__':
    main()
