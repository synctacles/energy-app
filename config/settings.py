"""Energy Insights NL Configuration - Brand-free template system"""
import os
from pathlib import Path
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    # ============================================================================
    # BRANDING & IDENTITY (Controls all user-facing strings and URLs)
    # REQUIRED: All brand values must be provided via environment variables
    # ============================================================================
    brand_name: str = os.getenv("BRAND_NAME")
    brand_slug: str = os.getenv("BRAND_SLUG")
    brand_domain: str = os.getenv("BRAND_DOMAIN")
    github_account: str = os.getenv("GITHUB_ACCOUNT")
    repo_name: str = os.getenv("REPO_NAME")

    # Home Assistant specific branding (derived from BRAND_SLUG)
    ha_domain: str = os.getenv("HA_DOMAIN")
    ha_component_name: str = os.getenv("HA_COMPONENT_NAME")

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

    # ============================================================================
    # SYSTEM USER CONFIGURATION
    # ============================================================================
    service_user: str = os.getenv("SERVICE_USER", "synctacles")
    service_group: str = os.getenv("SERVICE_GROUP", "synctacles")
    service_home: str = os.getenv("SERVICE_HOME", "/home/synctacles")

    # ============================================================================
    # GIT & DEPLOYMENT
    # ============================================================================
    github_repo: str = os.getenv("GITHUB_REPO")
    github_repo_dev: str = os.getenv("GITHUB_REPO_DEV")
    git_user_name: str = os.getenv("GIT_USER_NAME")
    git_user_email: str = os.getenv("GIT_USER_EMAIL")

    def __init__(self, **data):
        super().__init__(**data)
        # Validate required branding ENV variables
        _required = {
            "BRAND_NAME": self.brand_name,
            "BRAND_SLUG": self.brand_slug,
            "BRAND_DOMAIN": self.brand_domain,
            "GITHUB_ACCOUNT": self.github_account,
            "REPO_NAME": self.repo_name,
            "HA_DOMAIN": self.ha_domain,
            "HA_COMPONENT_NAME": self.ha_component_name,
            "GITHUB_REPO": self.github_repo,
            "GITHUB_REPO_DEV": self.github_repo_dev,
            "GIT_USER_NAME": self.git_user_name,
            "GIT_USER_EMAIL": self.git_user_email,
        }
        _missing = [k for k, v in _required.items() if not v]
        if _missing:
            raise ValueError(
                f"Missing required environment variables: {', '.join(_missing)}\n"
                f"Please run: sudo ./scripts/setup/setup_synctacles_server_v2.3.4.sh fase0\n"
                f"Or create .env from .env.example"
            )

    class Config:
        env_file = os.getenv("ENV_FILE", "/opt/.env")
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