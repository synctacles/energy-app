"""
Static hourly offsets for wholesale → consumer price conversion.

Based on 27,895 hours of ANWB data (2022-2026).
These offsets represent the average markup from wholesale to consumer price
for each hour of the day.

Usage:
    consumer_price = wholesale_price + HOURLY_OFFSET[hour]

Accuracy: 85-89% for ranking (sufficient for fallback tiers 4-5).

Note: This is a KISS (Keep It Simple, Stupid) approach that replaces
the complex coefficient server with a static lookup table.
For direct consumer price data, use Frank Energie or EasyEnergy APIs instead.
"""

from typing import Dict, List, Optional

# EUR/kWh offset per hour (0-23)
# These represent the average markup from wholesale (EPEX) to consumer price
HOURLY_OFFSET: Dict[int, float] = {
    0: 0.1934,   # Night low
    1: 0.1903,
    2: 0.1879,
    3: 0.1819,
    4: 0.1705,
    5: 0.1667,   # Lowest offset (early morning)
    6: 0.1789,
    7: 0.1989,   # Morning rise
    8: 0.2132,   # Morning peak
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

# Average offset across all hours (for quick estimates)
AVERAGE_OFFSET: float = sum(HOURLY_OFFSET.values()) / len(HOURLY_OFFSET)  # ~0.186


def apply_static_offset(wholesale_price_eur_kwh: float, hour: int) -> float:
    """
    Apply static hourly offset to wholesale price.

    Args:
        wholesale_price_eur_kwh: Wholesale price in EUR/kWh
        hour: Hour of day (0-23)

    Returns:
        Estimated consumer price in EUR/kWh

    Raises:
        ValueError: If hour is not in range 0-23
    """
    if hour not in range(24):
        raise ValueError(f"Invalid hour: {hour}. Must be 0-23.")

    offset = HOURLY_OFFSET[hour]
    return wholesale_price_eur_kwh + offset


def apply_static_offset_mwh(wholesale_price_eur_mwh: float, hour: int) -> float:
    """
    Apply static hourly offset to wholesale price (MWh version).

    Args:
        wholesale_price_eur_mwh: Wholesale price in EUR/MWh
        hour: Hour of day (0-23)

    Returns:
        Estimated consumer price in EUR/MWh
    """
    # Convert to kWh, apply offset, convert back
    wholesale_kwh = wholesale_price_eur_mwh / 1000.0
    consumer_kwh = apply_static_offset(wholesale_kwh, hour)
    return consumer_kwh * 1000.0


def get_market_stats(wholesale_prices: List[float]) -> Optional[Dict]:
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
        'average': sum(wholesale_prices) / len(wholesale_prices),
        'spread': max(wholesale_prices) - min(wholesale_prices),
        'min': min(wholesale_prices),
        'max': max(wholesale_prices)
    }


def get_expected_range(
    market_average_eur_kwh: float,
    hour: int,
    tolerance_percent: float = 15.0
) -> Dict[str, float]:
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
        'expected': round(expected, 4),
        'low': round(expected - tolerance, 4),
        'high': round(expected + tolerance, 4)
    }
