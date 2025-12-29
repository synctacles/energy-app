"""
Synctacles Configuration - Fail-Fast Design

All configuration MUST come from environment variables.
No fallback defaults - missing config = immediate failure.
"""
import os
import sys


class ConfigurationError(Exception):
    """Raised when required configuration is missing."""
    pass


def require_env(key: str, description: str = "") -> str:
    """Get required environment variable or fail immediately."""
    value = os.getenv(key)
    if value is None or value == "":
        msg = f"FATAL: Required environment variable '{key}' is not set."
        if description:
            msg += f"\n  Description: {description}"
        msg += f"\n  Ensure /opt/.env is sourced with 'set -a && source /opt/.env && set +a'"
        print(msg, file=sys.stderr)
        raise ConfigurationError(msg)
    return value


def optional_env(key: str, default: str = "") -> str:
    """Get optional environment variable with explicit default."""
    return os.getenv(key, default)


# =============================================================================
# Required Configuration (fail-fast if missing)
# =============================================================================

# Database - REQUIRED
DATABASE_URL = require_env("DATABASE_URL", "PostgreSQL connection string")
DB_HOST = require_env("DB_HOST", "Database hostname")
DB_PORT = require_env("DB_PORT", "Database port")
DB_NAME = require_env("DB_NAME", "Database name")
DB_USER = require_env("DB_USER", "Database user")

# Paths - REQUIRED
INSTALL_PATH = require_env("INSTALL_PATH", "Base installation directory")
LOG_PATH = require_env("LOG_PATH", "Log directory")

# =============================================================================
# Optional Configuration (explicit defaults)
# =============================================================================

# API Keys - Optional (will fail at runtime if needed but not set)
ENTSOE_API_KEY = optional_env("ENTSOE_API_KEY", "")
ADMIN_API_KEY = optional_env("ADMIN_API_KEY", "")

# API Settings
API_PORT = int(optional_env("API_PORT", "8000"))
API_HOST = optional_env("API_HOST", "0.0.0.0")

# Brand (for multi-tenant)
BRAND_NAME = optional_env("BRAND_NAME", "Synctacles")
BRAND_SLUG = optional_env("BRAND_SLUG", "synctacles")

# Log Level
LOG_LEVEL = optional_env("LOG_LEVEL", "INFO")
