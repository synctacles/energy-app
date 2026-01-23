"""
Pytest configuration and fixtures for SYNCTACLES tests.

Sets up environment variables and common fixtures.
"""

import os

# Set ALL required test environment variables BEFORE importing app modules
# These must be set at module level to prevent import errors
# Test database URL - uses local test DB without real credentials
os.environ.setdefault("DATABASE_URL", "postgresql://localhost:5432/test_db")
os.environ.setdefault("DB_HOST", "localhost")
os.environ.setdefault("DB_PORT", "5432")
os.environ.setdefault("DB_NAME", "test_db")
os.environ.setdefault("DB_USER", "test")
os.environ.setdefault("INSTALL_PATH", "/opt/synctacles-dev")
os.environ.setdefault("LOG_PATH", "/tmp/synctacles-test-logs")
os.environ.setdefault("ENVIRONMENT", "test")
os.environ.setdefault("BRAND_NAME", "SYNCTACLES TEST")
os.environ.setdefault("BRAND_SLUG", "synctacles-test")
os.environ.setdefault("GITHUB_ACCOUNT", "synctacles")
os.environ.setdefault("REPO_NAME", "ha-integration")
os.environ.setdefault("ENTSOE_API_KEY", "test-key-not-real")
os.environ.setdefault("SECRET_KEY", "test-secret-key")

from unittest.mock import MagicMock, patch

import pytest


@pytest.fixture(scope="session", autouse=True)
def setup_test_environment():
    """Set up test environment variables."""
    env_vars = {
        "DATABASE_URL": "postgresql://localhost:5432/test_db",
        "DB_HOST": "localhost",
        "DB_PORT": "5432",
        "DB_NAME": "test_db",
        "DB_USER": "test",
        "INSTALL_PATH": "/opt/synctacles-dev",
        "LOG_PATH": "/tmp/synctacles-test-logs",
        "ENVIRONMENT": "test",
        "BRAND_NAME": "SYNCTACLES TEST",
        "BRAND_SLUG": "synctacles-test",
        "GITHUB_ACCOUNT": "synctacles",
        "REPO_NAME": "ha-integration",
        "ENTSOE_API_KEY": "test-key-not-real",
        "SECRET_KEY": "test-secret-key",
        "LOG_LEVEL": "WARNING",
    }

    with patch.dict(os.environ, env_vars):
        yield


@pytest.fixture
def mock_db_session():
    """Provide a mock database session."""
    session = MagicMock()
    session.execute = MagicMock(return_value=MagicMock())
    session.commit = MagicMock()
    session.rollback = MagicMock()
    session.close = MagicMock()
    return session


@pytest.fixture
def sample_price_data():
    """Provide sample price data for tests."""
    return [
        {
            "timestamp": "2026-01-23T12:00:00Z",
            "price_eur_mwh": 50.0,
            "price_eur_kwh": 0.05,
        },
        {
            "timestamp": "2026-01-23T13:00:00Z",
            "price_eur_mwh": 55.0,
            "price_eur_kwh": 0.055,
        },
        {
            "timestamp": "2026-01-23T14:00:00Z",
            "price_eur_mwh": 48.0,
            "price_eur_kwh": 0.048,
        },
    ]
