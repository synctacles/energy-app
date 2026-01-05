"""Base utilities for all normalizers."""
from sqlalchemy import create_engine, text
from config.settings import DATABASE_URL
import logging

_LOGGER = logging.getLogger(__name__)


def validate_db_connection():
    """
    Fail-fast database validation.
    Call at start of every normalizer.

    Returns:
        SQLAlchemy Engine instance if validation succeeds

    Raises:
        SystemExit(1) if validation fails
    """
    try:
        engine = create_engine(DATABASE_URL)
        with engine.connect() as conn:
            conn.execute(text("SELECT 1"))
        _LOGGER.info("✓ Database connectie gevalideerd")
        return engine
    except Exception as e:
        _LOGGER.critical(f"✗ Database connectie FAILED: {e}")
        _LOGGER.critical("  Check DATABASE_URL in /opt/.env")
        _LOGGER.critical("  Verwacht user: energy_insights_nl")
        raise SystemExit(1)
