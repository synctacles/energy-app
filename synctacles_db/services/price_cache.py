"""Price Cache Service for 24h fallback persistence.

Issue #61 - Add 24h price cache for fallback

Provides PostgreSQL-backed price caching with automatic 24h cleanup.
Used as Tier 4 fallback when all live sources are unavailable.
"""

from datetime import datetime, timezone, timedelta
from typing import Optional, Dict, List
from decimal import Decimal
import logging

from sqlalchemy import create_engine, desc, delete
from sqlalchemy.orm import sessionmaker

from synctacles_db.models import PriceCache
from config.settings import DATABASE_URL

_LOGGER = logging.getLogger(__name__)

engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)


class PriceCacheService:
    """Service for managing 24h price cache in PostgreSQL."""

    CACHE_WINDOW_HOURS = 24

    @staticmethod
    def store_price(
        price: float,
        source: str,
        quality: str,
        country: str = "NL",
        timestamp: Optional[datetime] = None
    ) -> bool:
        """
        Store a price in the cache.

        Args:
            price: Consumer price in EUR/kWh
            source: Data source (frank, entsoe+offset, energy-charts+offset, easyenergy)
            quality: Quality level (live, estimated, cached)
            country: Country code (default: NL)
            timestamp: Price timestamp (default: now)

        Returns:
            True if stored successfully, False otherwise
        """
        session = Session()
        try:
            cache_entry = PriceCache(
                timestamp=timestamp or datetime.now(timezone.utc),
                country=country.upper(),
                price_eur_kwh=Decimal(str(price)),
                source=source,
                quality=quality,
            )
            session.add(cache_entry)
            session.commit()
            _LOGGER.debug(f"Price cached: {price:.4f} EUR/kWh from {source}")
            return True

        except Exception as e:
            session.rollback()
            _LOGGER.error(f"Failed to cache price: {e}")
            return False

        finally:
            session.close()

    @staticmethod
    def get_last_known(country: str = "NL") -> Optional[Dict]:
        """
        Get most recent cached price.

        Args:
            country: Country code (default: NL)

        Returns:
            Dict with price, source, quality, timestamp or None
        """
        session = Session()
        try:
            row = session.query(PriceCache)\
                .filter(PriceCache.country == country.upper())\
                .order_by(desc(PriceCache.timestamp))\
                .first()

            if row:
                return {
                    "price": float(row.price_eur_kwh),
                    "source": row.source,
                    "quality": row.quality,
                    "timestamp": row.timestamp.isoformat(),
                }
            return None

        except Exception as e:
            _LOGGER.error(f"Failed to get cached price: {e}")
            return None

        finally:
            session.close()

    @staticmethod
    def get_cached_prices(country: str = "NL", hours: int = 24) -> List[Dict]:
        """
        Get cached prices for the last N hours.

        Args:
            country: Country code (default: NL)
            hours: Number of hours to retrieve (default: 24)

        Returns:
            List of price dicts with timestamp, price, source, quality
        """
        session = Session()
        try:
            cutoff = datetime.now(timezone.utc) - timedelta(hours=hours)

            rows = session.query(PriceCache)\
                .filter(PriceCache.country == country.upper())\
                .filter(PriceCache.timestamp >= cutoff)\
                .order_by(desc(PriceCache.timestamp))\
                .all()

            return [
                {
                    "timestamp": row.timestamp.isoformat(),
                    "price_eur_kwh": float(row.price_eur_kwh),
                    "source": row.source,
                    "quality": row.quality,
                }
                for row in rows
            ]

        except Exception as e:
            _LOGGER.error(f"Failed to get cached prices: {e}")
            return []

        finally:
            session.close()

    @staticmethod
    def cleanup_old_entries(hours: int = 24) -> int:
        """
        Remove cache entries older than specified hours.

        Args:
            hours: Maximum age in hours (default: 24)

        Returns:
            Number of deleted entries
        """
        session = Session()
        try:
            cutoff = datetime.now(timezone.utc) - timedelta(hours=hours)

            result = session.execute(
                delete(PriceCache).where(PriceCache.timestamp < cutoff)
            )
            deleted_count = result.rowcount
            session.commit()

            if deleted_count > 0:
                _LOGGER.info(f"Cleaned up {deleted_count} old cache entries")

            return deleted_count

        except Exception as e:
            session.rollback()
            _LOGGER.error(f"Failed to cleanup cache: {e}")
            return 0

        finally:
            session.close()

    @staticmethod
    def get_cache_stats(country: str = "NL") -> Dict:
        """
        Get cache statistics.

        Args:
            country: Country code (default: NL)

        Returns:
            Dict with count, oldest, newest, sources
        """
        session = Session()
        try:
            from sqlalchemy import func

            # Count entries
            count = session.query(func.count(PriceCache.id))\
                .filter(PriceCache.country == country.upper())\
                .scalar()

            # Get oldest and newest
            oldest = session.query(func.min(PriceCache.timestamp))\
                .filter(PriceCache.country == country.upper())\
                .scalar()

            newest = session.query(func.max(PriceCache.timestamp))\
                .filter(PriceCache.country == country.upper())\
                .scalar()

            # Count by source
            sources = session.query(
                PriceCache.source,
                func.count(PriceCache.id)
            ).filter(PriceCache.country == country.upper())\
                .group_by(PriceCache.source)\
                .all()

            return {
                "count": count or 0,
                "oldest": oldest.isoformat() if oldest else None,
                "newest": newest.isoformat() if newest else None,
                "sources": {src: cnt for src, cnt in sources},
            }

        except Exception as e:
            _LOGGER.error(f"Failed to get cache stats: {e}")
            return {"count": 0, "oldest": None, "newest": None, "sources": {}}

        finally:
            session.close()


# Singleton instance
price_cache_service = PriceCacheService()
