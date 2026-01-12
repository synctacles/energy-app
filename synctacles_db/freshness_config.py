"""
Freshness thresholds for different data sources.

Based on real-world measurements:
- ENTSO-E A75: ~58-60 minutes structural delay
- Energy-Charts: ~3+ hours old data
- Cache: Stale but available

Phase 3: TenneT thresholds removed (2026-01-11) - TenneT is BYO-key only
"""

from enum import Enum
from typing import Dict, Literal

class QualityStatus(str, Enum):
    """Quality status values for data freshness."""
    FRESH = "FRESH"           # Data within fresh threshold
    STALE = "STALE"           # Data within stale threshold but older than fresh
    PARTIAL = "PARTIAL"       # Hybrid merge (mixed sources)
    FALLBACK = "FALLBACK"     # Using fallback source (Energy-Charts)
    CACHED = "CACHED"         # Using cached data
    UNAVAILABLE = "UNAVAILABLE"  # No data available


# Freshness thresholds (in minutes) per data source
# Phase 3: TenneT removed (BYO-key only, not server-side)
FRESHNESS_THRESHOLDS: Dict[str, Dict[Literal["fresh", "stale"], int]] = {
    "ENTSO-E": {
        "fresh": 90,        # < 90 min = FRESH (accounts for ~60min structural delay + 30min buffer)
        "stale": 180,       # 90-180 min = STALE (beyond acceptable, trigger fallback)
        # > 180 min triggers fallback to Energy-Charts
    },
    "Energy-Charts": {
        "fresh": 240,       # < 240 min (4h) = FRESH (EC is typically 3+ hours behind)
        "stale": 480,       # 240-480 min (4-8h) = STALE
    },
    "Cache": {
        "fresh": 120,       # < 120 min (2h) = FRESH
        "stale": 360,       # 120-360 min (2-6h) = STALE
    },
}


def get_quality_status(source: str, age_minutes: float) -> QualityStatus:
    """
    Determine quality status based on data source and age.

    Args:
        source: Data source name (e.g., "ENTSO-E", "Energy-Charts")
        age_minutes: Age of data in minutes

    Returns:
        QualityStatus value indicating data freshness
    """
    thresholds = FRESHNESS_THRESHOLDS.get(
        source,
        FRESHNESS_THRESHOLDS["ENTSO-E"]  # Default to ENTSO-E thresholds
    )

    if age_minutes < thresholds["fresh"]:
        return QualityStatus.FRESH
    elif age_minutes < thresholds["stale"]:
        return QualityStatus.STALE
    else:
        return QualityStatus.UNAVAILABLE


def should_trigger_fallback(source: str, age_minutes: float) -> bool:
    """
    Check if fallback should be triggered for this source.

    Fallback is triggered when data is older than the stale threshold.

    Args:
        source: Data source name
        age_minutes: Age of data in minutes

    Returns:
        True if fallback should be triggered
    """
    thresholds = FRESHNESS_THRESHOLDS.get(
        source,
        FRESHNESS_THRESHOLDS["ENTSO-E"]
    )

    return age_minutes >= thresholds["stale"]
