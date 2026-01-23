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
        msg += "\n  Ensure /opt/.env is sourced with 'set -a && source /opt/.env && set +a'"
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

# Auth & Rate Limiting - Feature flags (disabled by default)
AUTH_REQUIRED = optional_env("AUTH_REQUIRED", "false").lower() == "true"
RATE_LIMIT_ENABLED = optional_env("RATE_LIMIT_ENABLED", "false").lower() == "true"
DEFAULT_TIER = optional_env("DEFAULT_TIER", "beta")

# API Settings
API_PORT = int(optional_env("API_PORT", "8000"))
API_HOST = optional_env("API_HOST", "0.0.0.0")

# Brand (for multi-tenant) - REQUIRED for systemd service naming
BRAND_NAME = require_env("BRAND_NAME", "Display name for the brand")
BRAND_SLUG = require_env(
    "BRAND_SLUG", "URL-safe slug used for systemd services (e.g., 'energy-insights-nl')"
)

# GitHub (for documentation links) - REQUIRED for API responses
GITHUB_ACCOUNT = require_env("GITHUB_ACCOUNT", "GitHub account name (e.g., 'DATADIO')")
HA_REPO_NAME = require_env("REPO_NAME", "Home Assistant integration repo name")

# Log Level
LOG_LEVEL = optional_env("LOG_LEVEL", "INFO")

# CORS Configuration - Restrict origins in production
# Format: "https://homeassistant.local,https://example.com" (comma-separated)
# Development default: allows all origins
CORS_ORIGINS = (
    optional_env("CORS_ORIGINS", "").split(",")
    if optional_env("CORS_ORIGINS", "")
    else ["*"]
)
# Strip whitespace from each origin
CORS_ORIGINS = [origin.strip() for origin in CORS_ORIGINS if origin.strip()]


# =============================================================================
# Settings object for backward compatibility
# =============================================================================
class Settings:
    """Settings wrapper for backward compatibility with API code."""

    def __init__(self):
        self.database_url = DATABASE_URL
        self.db_host = DB_HOST
        self.db_port = DB_PORT
        self.db_name = DB_NAME
        self.db_user = DB_USER
        self.install_path = INSTALL_PATH
        self.log_path = LOG_PATH
        self.entsoe_api_key = ENTSOE_API_KEY
        self.admin_api_key = ADMIN_API_KEY
        self.api_port = API_PORT
        self.api_host = API_HOST
        self.brand_name = BRAND_NAME
        self.brand_slug = BRAND_SLUG
        self.github_account = GITHUB_ACCOUNT
        self.ha_repo_name = HA_REPO_NAME
        self.auth_required = AUTH_REQUIRED
        self.rate_limit_enabled = RATE_LIMIT_ENABLED
        self.default_tier = DEFAULT_TIER
        self.cors_origins = CORS_ORIGINS


settings = Settings()

# Fix: add missing API attributes
Settings.api_title = property(lambda self: f"{self.brand_name} API")
Settings.api_description = property(
    lambda self: f"Energy data API for {self.brand_name}"
)
