"""Energy Insights NL Configuration - ENV-driven branding and paths"""
import os
from pathlib import Path
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    # ============================================================================
    # BRANDING & IDENTITY (Controls all user-facing strings and URLs)
    # ============================================================================
    brand_name: str = os.getenv("BRAND_NAME", "Energy Insights NL")
    brand_slug: str = os.getenv("BRAND_SLUG", "energy-insights")
    brand_domain: str = os.getenv("BRAND_DOMAIN", "energy-insights.example.com")

    # Home Assistant specific branding
    ha_domain: str = os.getenv("HA_DOMAIN", "ha_energy_insights_nl")
    ha_component_name: str = os.getenv("HA_COMPONENT_NAME", "Energy Insights NL")

    # ============================================================================
    # API SETTINGS
    # ============================================================================
    api_host: str = os.getenv("API_HOST", "127.0.0.1")
    api_port: int = int(os.getenv("API_PORT", 8000))
    debug: bool = os.getenv("DEBUG", "False").lower() == "true"

    # ============================================================================
    # DATABASE CONFIGURATION
    # ============================================================================
    database_url: str = os.getenv("DATABASE_URL", "postgresql://synctacles@localhost:5432/synctacles")
    db_user: str = os.getenv("DB_USER", "synctacles")
    db_name: str = os.getenv("DB_NAME", "synctacles")
    db_host: str = os.getenv("DB_HOST", "localhost")
    db_port: int = int(os.getenv("DB_PORT", 5432))

    # ============================================================================
    # EXTERNAL API KEYS
    # ============================================================================
    entso_e_api_key: str = os.getenv("ENTSO_E_API_KEY", "")
    tennet_api_key: str = os.getenv("TENNET_API_KEY", "")
    tennet_api_base: str = os.getenv("TENNET_API_BASE", "https://api.tennet.eu")

    # ============================================================================
    # LOGGING & DEBUG
    # ============================================================================
    log_level: str = os.getenv("LOG_LEVEL", "INFO")
    log_path: str = os.getenv("LOG_PATH", "/var/log/energy-insights")

    # ============================================================================
    # INSTALLATION PATHS (Customizable via ENV)
    # ============================================================================
    install_path: str = os.getenv("INSTALL_PATH", "/opt/energy-insights")
    app_path: str = os.getenv("APP_PATH", "/opt/energy-insights/app")
    venv_path: str = os.getenv("VENV_PATH", "/opt/energy-insights/venv")
    data_path: str = os.getenv("DATA_PATH", "/var/lib/energy-insights")

    # Legacy path variable for backwards compatibility
    synctacles_log_dir: str = os.getenv("SYNCTACLES_LOG_DIR", "/opt/energy-insights/logs")

    # ============================================================================
    # SYSTEM USER CONFIGURATION
    # ============================================================================
    service_user: str = os.getenv("SERVICE_USER", "synctacles")
    service_group: str = os.getenv("SERVICE_GROUP", "synctacles")
    service_home: str = os.getenv("SERVICE_HOME", "/home/synctacles")

    # ============================================================================
    # GIT & DEPLOYMENT
    # ============================================================================
    github_repo: str = os.getenv("GITHUB_REPO", "git@github.com:DATADIO/ha-energy-insights-nl.git")
    github_repo_dev: str = os.getenv("GITHUB_REPO_DEV", "/opt/github/energy-insights-repo")
    git_user_name: str = os.getenv("GIT_USER_NAME", "DATADIO")
    git_user_email: str = os.getenv("GIT_USER_EMAIL", "admin@datadio.nl")

    class Config:
        env_file = os.getenv("ENV_FILE", "/opt/energy-insights/.env")
        extra = "ignore"

    @property
    def api_title(self) -> str:
        """API title derived from brand name."""
        return f"{self.brand_name} API"

    @property
    def api_description(self) -> str:
        """API description derived from brand name."""
        return f"{self.brand_name} Energy Data API"


# Global settings instance
settings = Settings()

# ============================================================================
# BACKWARDS COMPATIBILITY EXPORTS
# ============================================================================
# Export commonly used values for backwards compatibility
DATABASE_URL = settings.database_url
API_TITLE = settings.api_title
API_DESCRIPTION = settings.api_description
LOG_PATH = settings.log_path
INSTALL_PATH = settings.install_path
BRAND_NAME = settings.brand_name
BRAND_SLUG = settings.brand_slug