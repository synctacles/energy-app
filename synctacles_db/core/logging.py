"""
SYNCTACLES centralized logging configuration.

Single log file, rotation, configurable level via .env

Usage:
    from synctacles_db.core.logging import setup_logging, get_logger

    # Once at startup:
    setup_logging()

    # In any module:
    _LOGGER = get_logger(__name__)
    _LOGGER.info("Message")
"""

import logging
import os
from logging.handlers import RotatingFileHandler
from pathlib import Path

LOG_LEVELS = {
    "off": 100,
    "error": logging.ERROR,
    "warning": logging.WARNING,
    "info": logging.INFO,
    "debug": logging.DEBUG,
}

DEFAULT_LOG_PATH = "/var/log/synctacles/synctacles.log"
DEFAULT_LOG_LEVEL = "warning"
MAX_BYTES = 10_000_000  # 10MB per file
BACKUP_COUNT = 3  # 4 files total = max 40MB

_initialized = False


def setup_logging(
    level: str | None = None,
    log_path: str | None = None,
) -> logging.Logger:
    """
    Initialize logging for all synctacles_db modules.

    Call once at application startup. Idempotent.

    Args:
        level: off|error|warning|info|debug
               (default: LOG_LEVEL env or 'warning')
        log_path: Override log file path
                  (default: LOG_PATH_FILE env or standard location)

    Returns:
        Root logger for synctacles_db namespace
    """
    global _initialized
    if _initialized:
        return logging.getLogger("synctacles_db")

    # Resolve config from env or defaults
    level = level or os.getenv("LOG_LEVEL", DEFAULT_LOG_LEVEL)
    log_path = log_path or os.getenv("LOG_PATH_FILE", DEFAULT_LOG_PATH)
    level_num = LOG_LEVELS.get(level.lower(), logging.WARNING)

    # Handle "off" level
    if level.lower() == "off":
        _initialized = True
        root_logger = logging.getLogger("synctacles_db")
        root_logger.addHandler(logging.NullHandler())
        return root_logger

    # Ensure log directory exists
    log_dir = Path(log_path).parent
    log_dir.mkdir(parents=True, exist_ok=True)

    # Configure rotating file handler
    handler = RotatingFileHandler(
        log_path,
        maxBytes=MAX_BYTES,
        backupCount=BACKUP_COUNT,
        encoding="utf-8",
    )
    handler.setFormatter(
        logging.Formatter(
            "%(asctime)s [%(levelname)-7s] %(name)s: %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
    )

    # Configure root logger for synctacles_db namespace
    root_logger = logging.getLogger("synctacles_db")
    root_logger.setLevel(level_num)
    root_logger.addHandler(handler)

    # Prevent propagation to root (avoid duplicate logs)
    root_logger.propagate = False

    _initialized = True
    root_logger.info(f"Logging initialized: level={level}, path={log_path}")

    return root_logger


def get_logger(name: str) -> logging.Logger:
    """
    Get logger for a module.

    Args:
        name: Usually __name__ of the calling module

    Returns:
        Logger instance under synctacles_db namespace

    Example:
        from synctacles_db.core.logging import get_logger
        _LOGGER = get_logger(__name__)
        _LOGGER.info("Processing started")
    """
    # Auto-initialize if not done yet
    if not _initialized:
        setup_logging()

    return logging.getLogger(name)
