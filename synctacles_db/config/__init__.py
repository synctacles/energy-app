"""Configuration modules for Synctacles."""
from synctacles_db.config.static_offsets import (
    # Hybrid model (recommended)
    BTW_RATE,
    FIXED_MARKUP,
    SOURCING_MARKUP,
    ENERGY_TAX,
    apply_hybrid_conversion,
    apply_hybrid_conversion_mwh,
    # Legacy (deprecated, forwards to hybrid)
    HOURLY_OFFSET,
    AVERAGE_OFFSET,
    apply_static_offset,
    apply_static_offset_mwh,
    # Utilities
    get_market_stats,
    get_expected_range,
)

__all__ = [
    # Hybrid model (recommended)
    "BTW_RATE",
    "FIXED_MARKUP",
    "SOURCING_MARKUP",
    "ENERGY_TAX",
    "apply_hybrid_conversion",
    "apply_hybrid_conversion_mwh",
    # Legacy (deprecated)
    "HOURLY_OFFSET",
    "AVERAGE_OFFSET",
    "apply_static_offset",
    "apply_static_offset_mwh",
    # Utilities
    "get_market_stats",
    "get_expected_range",
]
