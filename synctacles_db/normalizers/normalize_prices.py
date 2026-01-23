"""Normalize raw_prices to norm_prices with quality checks."""

import time
from datetime import UTC, datetime

from sqlalchemy import create_engine, text
from sqlalchemy.orm import sessionmaker

from config.settings import DATABASE_URL
from synctacles_db.core.logging import get_logger
from synctacles_db.normalizers.base import validate_db_connection

_LOGGER = get_logger(__name__)

# Validate database connection at startup
validate_db_connection()

engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)


def normalize_prices(country: str = "NL"):
    """Normalize prices with quality status."""
    _LOGGER.info(f"Price normalizer starting: country={country}")
    start_time = time.time()

    session = Session()
    now = datetime.now(UTC)

    try:
        # Get latest raw data
        raw_records = session.execute(
            text("""
            SELECT timestamp, price_eur_mwh, fetch_time
            FROM raw_prices
            WHERE country = :country
            AND timestamp >= NOW() - INTERVAL '48 hours'
            ORDER BY timestamp
        """),
            {"country": country},
        ).fetchall()

        _LOGGER.debug(f"Found {len(raw_records)} raw price records")

        normalized = 0
        stale_count = 0

        for record in raw_records:
            timestamp, price, fetch_time = record

            # Quality check
            age = (now - fetch_time).total_seconds() / 3600
            if age < 1:
                quality = "OK"
            elif age < 25:
                quality = "STALE"
                stale_count += 1
            else:
                quality = "NO_DATA"

            # Upsert
            session.execute(
                text("""
                INSERT INTO norm_prices (timestamp, country, price_eur_mwh, quality_status)
                VALUES (:ts, :country, :price, :quality)
                ON CONFLICT (timestamp, country) DO UPDATE
                SET price_eur_mwh = EXCLUDED.price_eur_mwh,
                    quality_status = EXCLUDED.quality_status,
                    normalized_at = NOW()
            """),
                {
                    "ts": timestamp,
                    "country": country,
                    "price": price,
                    "quality": quality,
                },
            )
            normalized += 1

        session.commit()

        elapsed = time.time() - start_time

        if stale_count > 0:
            _LOGGER.warning(f"Price data age: {stale_count} STALE records")

        _LOGGER.info(
            f"Price normalizer completed: {normalized} records normalized in {elapsed:.2f}s"
        )

    except Exception as err:
        session.rollback()
        elapsed = time.time() - start_time
        _LOGGER.error(
            f"Price normalizer failed after {elapsed:.2f}s: {type(err).__name__}: {err}"
        )
        raise
    finally:
        session.close()


if __name__ == "__main__":
    normalize_prices()
