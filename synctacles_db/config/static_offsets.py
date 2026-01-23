"""
Wholesale → Consumer price conversion.

Hybrid model based on Frank Energie API data analysis (Jan 2026):
- BTW (VAT) is 21% of wholesale price (VARIABLE component)
- Sourcing markup: €0.018150/kWh (FIXED)
- Energiebelasting: €0.110850/kWh (FIXED, 2026 rate)

Formula:
    consumer_price = wholesale_price × 1.21 + €0.129

Accuracy: ~97% (vs 85% with pure static offset)

Note: The old HOURLY_OFFSET values are kept for backwards compatibility
but should not be used for new code.
"""


# =============================================================================
# HYBRID MODEL (RECOMMENDED) - Based on Frank API data analysis
# =============================================================================

# BTW (VAT) rate - 21% in Netherlands
BTW_RATE: float = 0.21

# Fixed markup components (EUR/kWh) - From Frank API Jan 2026
SOURCING_MARKUP: float = 0.018150  # Frank's sourcing/profit margin
ENERGY_TAX: float = 0.110850  # Energiebelasting 2026

# Total fixed markup (does NOT scale with wholesale price)
FIXED_MARKUP: float = SOURCING_MARKUP + ENERGY_TAX  # = €0.129/kWh


def apply_hybrid_conversion(wholesale_price_eur_kwh: float) -> float:
    """
    Convert wholesale price to estimated consumer price using hybrid model.

    Formula: consumer = wholesale × 1.21 + €0.129

    Components:
    - BTW (21%): Scales with wholesale price
    - Sourcing markup (€0.018): Fixed
    - Energiebelasting (€0.111): Fixed

    Args:
        wholesale_price_eur_kwh: Wholesale price in EUR/kWh (EPEX/APX)

    Returns:
        Estimated consumer price in EUR/kWh

    Example:
        >>> apply_hybrid_conversion(0.10)  # €0.10/kWh wholesale
        0.250  # = 0.10 × 1.21 + 0.129
    """
    btw_inclusive = wholesale_price_eur_kwh * (1 + BTW_RATE)
    return btw_inclusive + FIXED_MARKUP


def apply_hybrid_conversion_mwh(wholesale_price_eur_mwh: float) -> float:
    """
    Convert wholesale price to estimated consumer price (MWh version).

    Args:
        wholesale_price_eur_mwh: Wholesale price in EUR/MWh

    Returns:
        Estimated consumer price in EUR/MWh
    """
    wholesale_kwh = wholesale_price_eur_mwh / 1000.0
    consumer_kwh = apply_hybrid_conversion(wholesale_kwh)
    return consumer_kwh * 1000.0


# =============================================================================
# LEGACY STATIC OFFSETS (DEPRECATED - kept for backwards compatibility)
# =============================================================================

# EUR/kWh offset per hour (0-23) - DEPRECATED
# These bake BTW into a fixed offset, causing ~15% error at price extremes
# Use apply_hybrid_conversion() instead
HOURLY_OFFSET: dict[int, float] = {
    0: 0.1934,  # Night low
    1: 0.1903,
    2: 0.1879,
    3: 0.1819,
    4: 0.1705,
    5: 0.1667,  # Lowest offset (early morning)
    6: 0.1789,
    7: 0.1989,  # Morning rise
    8: 0.2132,  # Morning peak
    9: 0.2099,
    10: 0.2030,
    11: 0.1968,
    12: 0.1899,  # Afternoon drop
    13: 0.1768,
    14: 0.1669,
    15: 0.1599,
    16: 0.1508,  # Lowest afternoon
    17: 0.1571,
    18: 0.1723,  # Evening rise
    19: 0.2009,
    20: 0.2085,  # Evening peak
    21: 0.2050,
    22: 0.2006,
    23: 0.1945,
}

# Average offset across all hours (for quick estimates) - DEPRECATED
AVERAGE_OFFSET: float = sum(HOURLY_OFFSET.values()) / len(HOURLY_OFFSET)  # ~0.186


def apply_static_offset(wholesale_price_eur_kwh: float, hour: int = 0) -> float:
    """
    DEPRECATED: Use apply_hybrid_conversion() instead.

    This function now forwards to the hybrid model for better accuracy.
    The 'hour' parameter is kept for backwards compatibility but is ignored.

    Args:
        wholesale_price_eur_kwh: Wholesale price in EUR/kWh
        hour: (IGNORED) Hour of day - kept for backwards compatibility

    Returns:
        Estimated consumer price in EUR/kWh
    """
    # Forward to hybrid model (hour is ignored as BTW doesn't vary by hour)
    return apply_hybrid_conversion(wholesale_price_eur_kwh)


def apply_static_offset_mwh(wholesale_price_eur_mwh: float, hour: int = 0) -> float:
    """
    DEPRECATED: Use apply_hybrid_conversion_mwh() instead.

    Args:
        wholesale_price_eur_mwh: Wholesale price in EUR/MWh
        hour: (IGNORED) Hour of day - kept for backwards compatibility

    Returns:
        Estimated consumer price in EUR/MWh
    """
    return apply_hybrid_conversion_mwh(wholesale_price_eur_mwh)


def get_market_stats(wholesale_prices: list[float]) -> dict | None:
    """
    Calculate market statistics for reference data.

    Args:
        wholesale_prices: List of wholesale prices (EUR/kWh)

    Returns:
        {
            'average': float,
            'spread': float,
            'min': float,
            'max': float
        }
        or None if prices list is empty
    """
    if not wholesale_prices:
        return None

    return {
        "average": sum(wholesale_prices) / len(wholesale_prices),
        "spread": max(wholesale_prices) - min(wholesale_prices),
        "min": min(wholesale_prices),
        "max": max(wholesale_prices),
    }


def get_expected_range(
    market_average_eur_kwh: float, hour: int, tolerance_percent: float = 15.0
) -> dict[str, float]:
    """
    Calculate expected consumer price range for anomaly detection.

    Args:
        market_average_eur_kwh: Average wholesale price for the day (EUR/kWh)
        hour: Hour of day (0-23)
        tolerance_percent: Percentage tolerance for range (default 15%)

    Returns:
        {
            'expected': float,  # Expected consumer price
            'low': float,       # Lower bound
            'high': float       # Upper bound
        }
    """
    expected = apply_static_offset(market_average_eur_kwh, hour)
    tolerance = expected * (tolerance_percent / 100.0)

    return {
        "expected": round(expected, 4),
        "low": round(expected - tolerance, 4),
        "high": round(expected + tolerance, 4),
    }
