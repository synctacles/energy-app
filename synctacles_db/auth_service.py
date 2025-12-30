"""Authentication Service - User & API Key Management"""

import hashlib
import secrets
from typing import Optional, Tuple
from datetime import datetime, timedelta, timezone
from sqlalchemy.orm import Session
from sqlalchemy import func

from synctacles_db.auth_models import User, APIUsage
from config.settings import DEFAULT_TIER
from synctacles_db.auth.tiers import get_rate_limit


def hash_api_key(api_key: str) -> str:
    """SHA256 hash for API key storage"""
    return hashlib.sha256(api_key.encode()).hexdigest()


def generate_api_key() -> str:
    """Generate secure random API key (32 bytes = 64 hex chars)"""
    return secrets.token_hex(32)


def create_user(db: Session, email: str, tier: str = None) -> Tuple[User, str]:
    """
    Create new user with license key and API key

    Args:
        db: Database session
        email: User email address
        tier: User tier (defaults to DEFAULT_TIER from settings)

    Returns:
        (User object, plain API key)
    """
    email = email.lower().strip()

    # Check if email exists
    existing = db.query(User).filter(User.email == email).first()
    if existing:
        raise ValueError(f"Email already registered: {email}")

    # Generate API key (plain for return, hashed for storage)
    api_key_plain = generate_api_key()
    api_key_hash = hash_api_key(api_key_plain)

    # Use DEFAULT_TIER if not specified
    if tier is None:
        tier = DEFAULT_TIER

    # Get rate limit for tier
    rate_limit = get_rate_limit(tier)

    # Create user
    user = User(
        email=email,
        api_key_hash=api_key_hash,
        tier=tier,
        rate_limit_daily=rate_limit
    )

    db.add(user)
    db.commit()
    db.refresh(user)

    return user, api_key_plain


def validate_api_key(db: Session, api_key: str) -> Optional[User]:
    """
    Validate API key and return user if valid
    
    Returns:
        User object if valid, None otherwise
    """
    api_key_hash = hash_api_key(api_key)
    
    user = db.query(User).filter(
        User.api_key_hash == api_key_hash,
        User.is_active == True
    ).first()
    
    return user


def check_rate_limit(db: Session, user: User) -> bool:
    """
    Check if user is within daily rate limit
    
    Returns:
        True if allowed, False if rate limited
    """
    today_start = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)
    
    usage_count = db.query(func.count(APIUsage.id)).filter(
        APIUsage.user_id == user.id,
        APIUsage.timestamp >= today_start
    ).scalar()
    
    return usage_count < user.rate_limit_daily


def log_api_usage(db: Session, user: User, endpoint: str, status_code: int):
    """Log API request for rate limiting & analytics"""
    usage = APIUsage(
        user_id=user.id,
        endpoint=endpoint,
        status_code=status_code
    )
    
    db.add(usage)
    db.commit()


def get_user_stats(db: Session, user: User) -> dict:
    """Get user usage statistics"""
    today_start = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)
    
    today_count = db.query(func.count(APIUsage.id)).filter(
        APIUsage.user_id == user.id,
        APIUsage.timestamp >= today_start
    ).scalar()
    
    return {
        "user_id": str(user.id),
        "email": user.email,
        "tier": user.tier,
        "rate_limit_daily": user.rate_limit_daily,
        "usage_today": today_count,
        "remaining_today": user.rate_limit_daily - today_count
    }


def regenerate_api_key(db: Session, user: User) -> str:
    """
    Generate new API key for existing user
    
    Returns:
        New plain API key
    """
    new_api_key_plain = generate_api_key()
    new_api_key_hash = hash_api_key(new_api_key_plain)
    
    user.api_key_hash = new_api_key_hash
    db.commit()
    
    return new_api_key_plain


def deactivate_user(db: Session, user: User):
    """Deactivate user account"""
    user.is_active = False
    db.commit()


def reactivate_user(db: Session, user: User):
    """Reactivate user account"""
    user.is_active = True
    db.commit()


def get_user_by_email(db: Session, email: str) -> Optional[User]:
    """Find user by email"""
    email = email.lower().strip()
    return db.query(User).filter(User.email == email).first()


def get_user_by_license_key(db: Session, license_key: str) -> Optional[User]:
    """Find user by license key"""
    try:
        import uuid
        license_uuid = uuid.UUID(license_key)
        return db.query(User).filter(User.license_key == license_uuid).first()
    except ValueError:
        return None


def cleanup_old_usage_logs(db: Session, days: int = 30):
    """
    Delete API usage logs older than N days
    
    Call this daily via cron/systemd timer
    """
    from datetime import datetime, timedelta, timezone
    
    cutoff = datetime.now(timezone.utc) - timedelta(days=days)
    
    deleted = db.query(APIUsage).filter(APIUsage.timestamp < cutoff).delete()
    db.commit()
    
    return deleted
