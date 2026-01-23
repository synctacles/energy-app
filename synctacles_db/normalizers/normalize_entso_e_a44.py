#!/usr/bin/env python3
"""
ENTSO-E A44 Normalizer: raw -> norm (with forward fill)
"""

import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from sqlalchemy import create_engine
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy.orm import sessionmaker

from config.settings import DATABASE_URL
from synctacles_db.core.logging import get_logger
from synctacles_db.models import NormEntsoeA44, RawEntsoeA44
from synctacles_db.normalizers.base import validate_db_connection

DB_URL = DATABASE_URL

_LOGGER = get_logger(__name__)

# Validate database connection at startup
validate_db_connection()

engine = create_engine(DB_URL)
Session = sessionmaker(bind=engine)


def get_previous_value(session, timestamp, country):
    """Get most recent price before given timestamp"""
    prev = (
        session.query(NormEntsoeA44)
        .filter(NormEntsoeA44.timestamp < timestamp)
        .filter(NormEntsoeA44.country == country)
        .order_by(NormEntsoeA44.timestamp.desc())
        .first()
    )

    if prev:
        return prev.price_eur_mwh
    return None


def normalize_prices():
    """Normalize raw prices to norm table with forward fill"""
    _LOGGER.info("A44 normalizer starting")
    start_time = time.time()

    session = Session()
    normalized = 0
    forward_filled = 0

    try:
        # Get all raw records not yet normalized
        raw_records = (
            session.query(RawEntsoeA44)
            .filter(RawEntsoeA44.country == "NL")
            .order_by(RawEntsoeA44.timestamp)
            .all()
        )

        _LOGGER.debug(f"Found {len(raw_records)} raw A44 records")

        for raw in raw_records:
            # Check if already exists
            exists = (
                session.query(NormEntsoeA44)
                .filter(
                    NormEntsoeA44.timestamp == raw.timestamp,
                    NormEntsoeA44.country == raw.country,
                )
                .first()
            )

            if exists:
                continue

            # Determine quality
            if raw.price_eur_mwh is not None and raw.price_eur_mwh > 0:
                # Fresh ENTSO-E data
                norm = NormEntsoeA44(
                    timestamp=raw.timestamp,
                    country=raw.country,
                    price_eur_mwh=raw.price_eur_mwh,
                    data_source="ENTSO-E",
                    data_quality="OK",
                    needs_backfill=False,
                )
                normalized += 1
            else:
                # Missing data - forward fill (fallback)
                _LOGGER.debug(f"Data gap at {raw.timestamp}, activating forward fill")
                prev_price = get_previous_value(session, raw.timestamp, raw.country)

                if prev_price:
                    norm = NormEntsoeA44(
                        timestamp=raw.timestamp,
                        country=raw.country,
                        price_eur_mwh=prev_price,
                        data_source="ENTSO-E",
                        data_quality="FORWARD_FILL",
                        needs_backfill=True,
                    )
                    forward_filled += 1
                else:
                    # No previous value - skip
                    _LOGGER.warning(
                        f"No previous value for {raw.timestamp}, cannot forward fill"
                    )
                    continue

            session.add(norm)

        session.commit()

        elapsed = time.time() - start_time
        _LOGGER.info(
            f"A44 normalizer completed: {normalized} OK, {forward_filled} forward-filled in {elapsed:.2f}s"
        )

    except SQLAlchemyError as e:
        session.rollback()
        elapsed = time.time() - start_time
        _LOGGER.error(
            f"A44 normalizer database error after {elapsed:.2f}s: {type(e).__name__}: {e}"
        )
        raise
    finally:
        session.close()


def main():
    _LOGGER.info("A44 Normalizer batch starting")
    normalize_prices()
    _LOGGER.info("A44 Normalizer batch complete")


if __name__ == "__main__":
    main()
