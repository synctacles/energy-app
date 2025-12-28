"""SYNCTACLES Configuration"""
import os
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    # API Settings
    api_host: str = os.getenv("API_HOST", "127.0.0.1")
    api_port: int = int(os.getenv("API_PORT", 8000))
    debug: bool = os.getenv("DEBUG", "False").lower() == "true"
    
    # Database
    database_url: str = os.getenv("DATABASE_URL", "postgresql://synctacles@localhost:5432/synctacles")
    
    # ENTSO-E API
    entso_e_api_key: str = os.getenv("ENTSO_E_API_KEY", "")
    
    # Logging
    log_level: str = os.getenv("LOG_LEVEL", "INFO")
    
    class Config:
        env_file = "/opt/synctacles/.env"
        extra = "ignore"

settings = Settings()

# Export voor imports
DATABASE_URL = settings.database_url