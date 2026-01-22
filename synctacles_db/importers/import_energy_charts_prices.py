"""Import Energy-Charts price JSON to raw_prices table."""
import json
import time
from datetime import datetime, timezone
from pathlib import Path
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from synctacles_db.core.logging import get_logger
from config.settings import DATABASE_URL, LOG_PATH

_LOGGER = get_logger(__name__)
LOG_DIR = Path(LOG_PATH)
INPUT_DIR = LOG_DIR / "collectors" / "energy_charts_raw"

engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)

def import_prices(file_path: Path, country: str = "NL"):
    """Import prices from JSON file to database."""
    _LOGGER.info(f"Energy-Charts importer starting: {file_path.name}")
    start_time = time.time()

    try:
        _LOGGER.debug(f"Parsing JSON file: {file_path}")
        with open(file_path) as f:
            data = json.load(f)

        unix_seconds = data.get("unix_seconds", [])
        prices = data.get("price", [])

        if len(unix_seconds) != len(prices):
            raise ValueError(f"Mismatch: {len(unix_seconds)} unix_seconds vs {len(prices)} prices")

        _LOGGER.debug(f"Found {len(unix_seconds)} price records to import")

        session = Session()
        imported = 0

        for ts_unix, price in zip(unix_seconds, prices):
            timestamp = datetime.fromtimestamp(ts_unix, tz=timezone.utc)

            # Upsert
            session.execute("""
                INSERT INTO raw_prices (timestamp, country, price_eur_mwh, source, source_file)
                VALUES (:ts, :country, :price, 'energy-charts', :file)
                ON CONFLICT (timestamp, country, source) DO UPDATE
                SET price_eur_mwh = EXCLUDED.price_eur_mwh
            """, {
                "ts": timestamp,
                "country": country,
                "price": price,
                "file": str(file_path)
            })
            imported += 1

        session.commit()
        session.close()

        elapsed = time.time() - start_time
        _LOGGER.info(f"Energy-Charts importer completed: {imported} records imported in {elapsed:.2f}s")

        return imported

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"Energy-Charts importer failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise

def import_all():
    """Import all unprocessed JSON files."""
    _LOGGER.info("Energy-Charts importer batch starting")
    start_time = time.time()

    try:
        if not INPUT_DIR.exists():
            _LOGGER.error(f"Directory not found: {INPUT_DIR}")
            return

        files = sorted(INPUT_DIR.glob("prices_NL_*.json"))

        if not files:
            _LOGGER.warning("No Energy-Charts price files found to import")
            return

        _LOGGER.info(f"Found {len(files)} Energy-Charts files to process")

        total_imported = 0
        failed_files = []

        for file_path in files:
            try:
                imported = import_prices(file_path)
                total_imported += imported
            except Exception as e:
                _LOGGER.debug(f"Failed to import {file_path.name}: {type(e).__name__}: {e}")
                failed_files.append(file_path.name)

        elapsed = time.time() - start_time

        if failed_files:
            _LOGGER.warning(f"Failed to import {len(failed_files)} files")
            _LOGGER.debug(f"Failed files: {failed_files}")

        _LOGGER.info(f"Energy-Charts importer batch completed: {total_imported} total records in {elapsed:.2f}s")

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"Energy-Charts batch importer failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise

if __name__ == "__main__":
    import_all()