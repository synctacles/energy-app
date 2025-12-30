"""
Tier Configuration - Rate Limit & Feature Definitions

Defines rate limits per subscription tier and tier-specific features.
"""

TIER_LIMITS = {
    "beta": 10_000,      # Beta users - high limit for testing
    "free": 1_000,       # Free tier - basic limit
    "paid": 100_000,     # Paid tier - high limit
    "unlimited": 100_000,  # Unlimited tier - enterprise
}

TIER_FEATURES = {
    "beta": {
        "name": "Beta",
        "rate_limit": 10_000,
        "api_access": True,
        "ha_integration": True,
        "support": "community",
    },
    "free": {
        "name": "Free",
        "rate_limit": 1_000,
        "api_access": True,
        "ha_integration": True,
        "support": "none",
    },
    "paid": {
        "name": "Paid",
        "rate_limit": 100_000,
        "api_access": True,
        "ha_integration": True,
        "support": "email",
    },
    "unlimited": {
        "name": "Unlimited",
        "rate_limit": 100_000,
        "api_access": True,
        "ha_integration": True,
        "support": "priority",
    },
}


def get_rate_limit(tier: str) -> int:
    """
    Get daily rate limit for a given tier.

    Falls back to 'free' tier limit if tier not found.
    """
    return TIER_LIMITS.get(tier, TIER_LIMITS["free"])


def get_tier_features(tier: str) -> dict:
    """
    Get feature set for a given tier.

    Falls back to 'free' tier features if tier not found.
    """
    return TIER_FEATURES.get(tier, TIER_FEATURES["free"])


def is_valid_tier(tier: str) -> bool:
    """Check if tier is valid."""
    return tier in TIER_LIMITS
