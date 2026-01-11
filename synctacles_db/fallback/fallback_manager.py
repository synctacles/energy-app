"""Fallback manager with hybrid merge for Energy-Charts data when ENTSO-E has NULLs.

Implements 5-tier fallback strategy with NULL filling:
1. Primary: Database (ENTSO-E normalized data)
2. Hybrid: Energy-Charts fills NULL values from ENTSO-E
3. Known Capacity: Pragmatic estimates when EC unavailable (circuit breaker)
4. Fallback: Energy-Charts complete (if ENTSO-E unavailable)
5. Cache: In-memory cache (stale but better than nothing)

Key features:
- Track which fields come from which source for transparency
- Circuit breaker: skip EC for 2h after HTTP 404 to respect API limits
- Known capacity modeling for pragmatic NULL filling
"""

from datetime import datetime, timezone, timedelta
from typing import Dict, List, Optional, Tuple, Literal
from cachetools import TTLCache
import logging
import math

from synctacles_db.fallback.energy_charts_client import EnergyChartsClient
from synctacles_db.freshness_config import FRESHNESS_THRESHOLDS, QualityStatus, get_quality_status
from synctacles_db.services.price_cache import price_cache_service

_LOGGER = logging.getLogger(__name__)

# Cache: 100 entries, 5 min TTL
_fallback_cache = TTLCache(maxsize=100, ttl=300)

# Circuit breaker for Energy-Charts API
_ec_circuit_breaker = {
    "last_404_time": None,      # Timestamp of last 404
    "cooldown_minutes": 120,    # 2 hours cooldown after 404
    "is_open": False,           # True = skip EC calls
}

# Known capacity for NL (pragmatic estimates)
KNOWN_CAPACITY_NL = {
    "nuclear_mw": 485.0,        # Borssele nuclear plant (assume always on)
    "biomass_mw": 350.0,        # Average biomass capacity
    "waste_mw": 80.0,           # Waste-to-energy plants
    "wind_onshore_ratio": 0.3,  # Onshore ~30% of offshore
}


class FallbackManager:
    """Manages fallback data sources with hybrid merge capability."""

    # Component-specific thresholds (minutes) - maps components to freshness_config sources
    THRESHOLDS = {
        "generation_mix": FRESHNESS_THRESHOLDS["ENTSO-E"],  # Use ENTSO-E thresholds
        "load": FRESHNESS_THRESHOLDS["ENTSO-E"],            # Use ENTSO-E thresholds
        "prices": FRESHNESS_THRESHOLDS["ENTSO-E"],          # Use ENTSO-E thresholds
    }
    
    @staticmethod
    def _check_circuit_breaker() -> bool:
        """
        Check if circuit breaker is open (EC unavailable).
        
        Returns True if EC should be skipped (within cooldown period).
        """
        if not _ec_circuit_breaker["last_404_time"]:
            return False  # Never failed, proceed
        
        # Calculate time since last 404
        now = datetime.now(timezone.utc)
        last_404 = _ec_circuit_breaker["last_404_time"]
        minutes_since = (now - last_404).total_seconds() / 60
        
        if minutes_since < _ec_circuit_breaker["cooldown_minutes"]:
            _LOGGER.info(f"EC circuit breaker OPEN ({int(minutes_since)} min since 404, cooling down)")
            return True  # Still cooling down
        
        # Cooldown expired, reset and allow retry
        _LOGGER.info(f"EC circuit breaker CLOSED (cooldown expired, retrying)")
        _ec_circuit_breaker["last_404_time"] = None
        _ec_circuit_breaker["is_open"] = False
        return False
    
    @staticmethod
    def _open_circuit_breaker():
        """Open circuit breaker after EC 404 error."""
        _ec_circuit_breaker["last_404_time"] = datetime.now(timezone.utc)
        _ec_circuit_breaker["is_open"] = True
        _LOGGER.warning("EC circuit breaker OPENED (404 error, cooldown for 2h)")
    
    @staticmethod
    def _estimate_solar_nl(timestamp: str) -> float:
        """
        Estimate solar production for NL based on time of day.
        
        Simple model: 0 at night, peaks at 13:00 UTC.
        """
        try:
            dt = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
            hour_utc = dt.hour
            
            # Night time (no solar)
            if hour_utc < 6 or hour_utc > 20:
                return 0.0
            
            # Daytime: sine curve peaking at 13:00
            # Peak capacity ~2000 MW for NL in summer, ~500 in winter
            month = dt.month
            if 5 <= month <= 8:  # Summer
                peak_capacity = 2000.0
            elif 11 <= month or month <= 2:  # Winter
                peak_capacity = 500.0
            else:  # Spring/Fall
                peak_capacity = 1200.0
            
            # Sine curve from 6:00 to 20:00, peak at 13:00
            hours_from_sunrise = hour_utc - 6
            solar_mw = peak_capacity * math.sin((hours_from_sunrise / 14.0) * math.pi)
            
            return max(0.0, solar_mw)
        except Exception as err:
            _LOGGER.error(f"Solar estimate error: {err}")
            return 0.0
    
    @staticmethod
    def _fill_with_known_capacity(data: Dict) -> Tuple[Dict, Dict[str, str]]:
        """
        Fill NULL fields with known capacity estimates.
        
        Args:
            data: ENTSO-E data with potential NULLs
            
        Returns:
            (filled_data, field_sources)
        """
        filled = data.copy()
        field_sources = {}
        
        timestamp = data.get("timestamp", datetime.now(timezone.utc).isoformat())
        
        # Track what we fill
        psr_fields = [
            'biomass_mw', 'gas_mw', 'coal_mw', 'nuclear_mw',
            'solar_mw', 'waste_mw', 'wind_offshore_mw',
            'wind_onshore_mw', 'other_mw'
        ]
        
        for field in psr_fields:
            if filled.get(field) is not None:
                field_sources[field] = "ENTSO-E"
            else:
                # Fill with known capacity
                if field == "nuclear_mw":
                    filled[field] = KNOWN_CAPACITY_NL["nuclear_mw"]
                    field_sources[field] = "Known Capacity"
                elif field == "biomass_mw":
                    filled[field] = KNOWN_CAPACITY_NL["biomass_mw"]
                    field_sources[field] = "Known Capacity"
                elif field == "waste_mw":
                    filled[field] = KNOWN_CAPACITY_NL["waste_mw"]
                    field_sources[field] = "Known Capacity"
                elif field == "solar_mw":
                    filled[field] = FallbackManager._estimate_solar_nl(timestamp)
                    field_sources[field] = "Estimated"
                elif field == "wind_onshore_mw":
                    # Correlate with offshore if available
                    offshore = filled.get("wind_offshore_mw", 0)
                    if offshore and offshore > 0:
                        filled[field] = offshore * KNOWN_CAPACITY_NL["wind_onshore_ratio"]
                        field_sources[field] = "Estimated (from offshore)"
                    else:
                        filled[field] = None  # Can't estimate without offshore
                        field_sources[field] = "Missing"
                else:
                    field_sources[field] = "Missing"
        
        return filled, field_sources
    
    @staticmethod
    async def get_component_with_fallback(
        component: Literal["generation_mix", "load"],
        db_result: Optional[Dict],
        db_age_minutes: int,
        country: str = "nl"
    ) -> Tuple[Optional[Dict], str, str]:
        """
        Get component data with hybrid fallback logic.
        
        For generation_mix: ALWAYS fetches Energy-Charts to fill ENTSO-E NULLs.
        For load: Standard fallback only.
        
        Args:
            component: "generation_mix" or "load"
            db_result: Database query result (may contain NULLs)
            db_age_minutes: Age of database data in minutes
            country: Country code
            
        Returns:
            Tuple of (data, source, quality)
            - data: Component dict (with _field_sources for generation_mix)
            - source: "ENTSO-E" | "Energy-Charts" | "Hybrid" | "Cache"
            - quality: "FRESH" | "STALE" | "PARTIAL" | "FALLBACK" | "CACHED" | "UNAVAILABLE"
        """
        
        thresholds = FallbackManager.THRESHOLDS.get(component, {"fresh": 15, "stale": 60})
        fresh_threshold = thresholds["fresh"]
        stale_threshold = thresholds["stale"]
        
        # === GENERATION MIX: HYBRID MERGE STRATEGY ===
        if component == "generation_mix":
            return await FallbackManager._handle_generation_mix_with_hybrid(
                db_result, db_age_minutes, country, fresh_threshold, stale_threshold
            )
        
        # === LOAD: STANDARD FALLBACK ===
        elif component == "load":
            return await FallbackManager._handle_standard_fallback(
                component, db_result, db_age_minutes, country, fresh_threshold, stale_threshold
            )
        
        # Unknown component
        return (None, "None", "UNAVAILABLE")
    
    @staticmethod
    async def _handle_generation_mix_with_hybrid(
        db_result: Optional[Dict],
        db_age_minutes: int,
        country: str,
        fresh_threshold: int,
        stale_threshold: int
    ) -> Tuple[Optional[Dict], str, str]:
        """
        Handle generation_mix with hybrid merge.
        
        Uses circuit breaker pattern: skips EC for 2h after 404.
        Falls back to Known Capacity modeling when EC unavailable.
        """
        
        # Check circuit breaker before attempting EC fetch
        ec_data = None
        skip_ec = FallbackManager._check_circuit_breaker()
        
        if not skip_ec:
            # Circuit closed - try Energy-Charts
            try:
                ec_data_list = await EnergyChartsClient.fetch_generation_mix(country=country, limit=1)
                if ec_data_list and len(ec_data_list) > 0:
                    ec_data = ec_data_list[0]
                    _LOGGER.info("Energy-Charts fetch successful")
            except Exception as err:
                error_msg = str(err)
                _LOGGER.error(f"Energy-Charts fetch failed: {error_msg}")
                
                # Check if 404 error - open circuit breaker
                if "404" in error_msg or "HTTP 404" in error_msg:
                    FallbackManager._open_circuit_breaker()
        else:
            _LOGGER.info("EC circuit breaker open - using Known Capacity instead")
        
        # Tier 1: Database data exists (may have NULLs)
        if db_result:
            # Check for NULL values
            null_fields = FallbackManager._find_null_fields(db_result)
            
            # If NULLs exist - try EC first, then Known Capacity
            if null_fields:
                if ec_data:
                    # Tier 1a: HYBRID MERGE with Energy-Charts
                    merged_data, field_sources, ec_timestamp = FallbackManager._hybrid_merge_generation(
                        db_result, ec_data
                    )
                    
                    filled_count = sum(1 for src in field_sources.values() if src == "Energy-Charts")
                    
                    _LOGGER.info(f"Hybrid merge (EC): filled {filled_count} NULL fields from Energy-Charts")
                    
                    # Add metadata
                    merged_data["_field_sources"] = field_sources
                    merged_data["_ec_timestamp"] = ec_timestamp
                    
                    quality = "PARTIAL"
                    source = f"Hybrid (ENTSO-E + {filled_count} from Energy-Charts)"
                    
                    # Cache hybrid result
                    cache_key = f"generation_{country}"
                    _fallback_cache[cache_key] = merged_data
                    
                    return (merged_data, source, quality)
                
                else:
                    # Tier 1b: KNOWN CAPACITY FALLBACK (EC unavailable)
                    filled_data, field_sources = FallbackManager._fill_with_known_capacity(db_result)
                    
                    filled_count = sum(1 for src in field_sources.values() 
                                     if src in ["Known Capacity", "Estimated", "Estimated (from offshore)"])
                    
                    _LOGGER.info(f"Known Capacity fill: {filled_count} fields estimated")
                    
                    # Add metadata
                    filled_data["_field_sources"] = field_sources
                    
                    quality = "PARTIAL"
                    source = f"Hybrid (ENTSO-E + {filled_count} from Known Capacity)"
                    
                    # Cache
                    cache_key = f"generation_{country}"
                    _fallback_cache[cache_key] = filled_data
                    
                    return (filled_data, source, quality)
            
            # No NULLs - Use ENTSO-E as-is
            else:
                # Add field sources (all ENTSO-E)
                field_sources = FallbackManager._all_entso_sources(db_result)
                db_result["_field_sources"] = field_sources
                
                if db_age_minutes < fresh_threshold:
                    return (db_result, "ENTSO-E", "FRESH")
                elif db_age_minutes < stale_threshold:
                    return (db_result, "ENTSO-E", "STALE")
                else:
                    # Too stale, try full Energy-Charts fallback below
                    pass
        
        # Tier 2: Database missing or too stale ? Full Energy-Charts fallback
        if ec_data:
            _LOGGER.info(f"Using full Energy-Charts fallback (DB age: {db_age_minutes} min)")
            
            # Add field sources (all Energy-Charts)
            field_sources = FallbackManager._all_energy_charts_sources(ec_data)
            ec_data["_field_sources"] = field_sources
            
            # Cache
            cache_key = f"generation_{country}"
            _fallback_cache[cache_key] = ec_data
            
            return (ec_data, "Energy-Charts", "FALLBACK")
        
        # Tier 3: Check cache
        cache_key = f"generation_{country}"
        if cache_key in _fallback_cache:
            cached_data = _fallback_cache[cache_key]
            _LOGGER.warning(f"Using cached fallback data")
            return (cached_data, "Cache", "CACHED")
        
        # Tier 4: All failed ? Return stale DB if available
        if db_result:
            _LOGGER.warning(f"All fallback failed, returning stale ENTSO-E ({db_age_minutes} min)")
            field_sources = FallbackManager._all_entso_sources(db_result)
            db_result["_field_sources"] = field_sources
            return (db_result, "ENTSO-E", "STALE")
        
        # Complete failure
        _LOGGER.error("All data sources failed")
        return (None, "None", "UNAVAILABLE")
    
    @staticmethod
    async def _handle_standard_fallback(
        component: str,
        db_result: Optional[Dict],
        db_age_minutes: int,
        country: str,
        fresh_threshold: int,
        stale_threshold: int
    ) -> Tuple[Optional[Dict], str, str]:
        """Standard fallback for components without hybrid merge."""
        
        # Tier 1: Fresh database data
        if db_result and db_age_minutes < fresh_threshold:
            return (db_result, "ENTSO-E", "FRESH")
        
        # Tier 1.5: Acceptable database data
        if db_result and db_age_minutes < stale_threshold:
            return (db_result, "ENTSO-E", "STALE")
        
        # Tier 2: Database too stale ? Try Energy-Charts (for load)
        if component == "load":
            try:
                # Energy-Charts doesn't have dedicated load endpoint
                # Use total_mw from generation as proxy
                ec_data_list = await EnergyChartsClient.fetch_generation_mix(country=country, limit=1)
                if ec_data_list and len(ec_data_list) > 0:
                    ec_gen = ec_data_list[0]
                    load_data = {"load_mw": ec_gen.get("total_mw", 0)}
                    
                    _LOGGER.info(f"Energy-Charts load fallback: {load_data['load_mw']:.1f} MW")
                    
                    # Cache
                    cache_key = f"load_{country}"
                    _fallback_cache[cache_key] = load_data
                    
                    return (load_data, "Energy-Charts", "FALLBACK")
            except Exception as err:
                _LOGGER.error(f"Energy-Charts load fallback failed: {err}")
        
        # Tier 3: Check cache
        cache_key = f"{component}_{country}"
        if cache_key in _fallback_cache:
            cached_data = _fallback_cache[cache_key]
            _LOGGER.warning(f"Using cached {component} data")
            return (cached_data, "Cache", "CACHED")
        
        # Tier 4: Return stale DB if available
        if db_result:
            _LOGGER.warning(f"Returning stale {component} data ({db_age_minutes} min)")
            return (db_result, "ENTSO-E", "STALE")
        
        # Complete failure
        return (None, "None", "UNAVAILABLE")
    
    @staticmethod
    def _find_null_fields(data: Dict) -> List[str]:
        """Find fields with NULL/None values in generation data."""
        psr_fields = [
            'biomass_mw', 'gas_mw', 'coal_mw', 'nuclear_mw',
            'solar_mw', 'waste_mw', 'wind_offshore_mw',
            'wind_onshore_mw', 'other_mw'
        ]
        
        null_fields = []
        for field in psr_fields:
            if data.get(field) is None:
                null_fields.append(field)
        
        return null_fields
    
    @staticmethod
    def _hybrid_merge_generation(
        entso_data: Dict,
        ec_data: Dict
    ) -> Tuple[Dict, Dict[str, str], Optional[str]]:
        """
        Merge ENTSO-E and Energy-Charts generation data.
        
        ENTSO-E is primary, Energy-Charts fills NULLs.
        Accepts timestamp mismatch up to 90 minutes (tolerance for EC lag).
        
        Args:
            entso_data: ENTSO-E data (may have NULLs)
            ec_data: Energy-Charts data (complete, may be older)
        
        Returns:
            (merged_data, field_sources, ec_timestamp)
            - merged_data: Combined dict
            - field_sources: Dict mapping field -> source name
            - ec_timestamp: Energy-Charts timestamp if used, for transparency
        """
        merged = entso_data.copy()
        field_sources = {}
        ec_timestamp = ec_data.get('timestamp')  # Track EC timestamp for transparency
        
        psr_fields = [
            'biomass_mw', 'gas_mw', 'coal_mw', 'nuclear_mw',
            'solar_mw', 'waste_mw', 'wind_offshore_mw',
            'wind_onshore_mw', 'other_mw'
        ]
        
        for field in psr_fields:
            entso_value = merged.get(field)
            ec_value = ec_data.get(field)
            
            if entso_value is None and ec_value is not None:
                # Fill NULL with Energy-Charts
                merged[field] = ec_value
                field_sources[field] = "Energy-Charts"
            elif entso_value is not None:
                # Keep ENTSO-E value
                field_sources[field] = "ENTSO-E"
            else:
                # Both NULL
                field_sources[field] = "Missing"
        
        # Total and timestamp always from ENTSO-E if available
        if merged.get('total_mw') is not None:
            field_sources['total_mw'] = "ENTSO-E"
        elif ec_data.get('total_mw') is not None:
            merged['total_mw'] = ec_data['total_mw']
            field_sources['total_mw'] = "Energy-Charts"
        
        return merged, field_sources, ec_timestamp
    
    @staticmethod
    def _all_entso_sources(data: Dict) -> Dict[str, str]:
        """Create field_sources dict with all fields marked as ENTSO-E."""
        psr_fields = [
            'biomass_mw', 'gas_mw', 'coal_mw', 'nuclear_mw',
            'solar_mw', 'waste_mw', 'wind_offshore_mw',
            'wind_onshore_mw', 'other_mw', 'total_mw'
        ]
        
        return {field: "ENTSO-E" for field in psr_fields if field in data}
    
    @staticmethod
    def _all_energy_charts_sources(data: Dict) -> Dict[str, str]:
        """Create field_sources dict with all fields marked as Energy-Charts."""
        psr_fields = [
            'biomass_mw', 'gas_mw', 'coal_mw', 'nuclear_mw',
            'solar_mw', 'waste_mw', 'wind_offshore_mw',
            'wind_onshore_mw', 'other_mw', 'total_mw'
        ]
        
        return {field: "Energy-Charts" for field in psr_fields if field in data}
    
    # === LEGACY WRAPPER FOR BACKWARD COMPATIBILITY ===
    
    @staticmethod
    async def get_generation_with_fallback(
        db_result: Optional[Dict],
        db_age_minutes: int,
        country: str = "nl"
    ) -> Tuple[Optional[Dict], str, str]:
        """Legacy wrapper for get_component_with_fallback."""
        return await FallbackManager.get_component_with_fallback(
            component="generation_mix",
            db_result=db_result,
            db_age_minutes=db_age_minutes,
            country=country
        )
    
    @staticmethod
    def calculate_renewable_percentage(data: Dict) -> Optional[float]:
        """Calculate renewable percentage from generation mix data."""
        try:
            renewable_mw = (
                (data.get("biomass_mw") or 0) +
                (data.get("solar_mw") or 0) +
                (data.get("wind_offshore_mw") or 0) +
                (data.get("wind_onshore_mw") or 0) +
                (data.get("hydro_mw") or 0)
            )

            total_mw = data.get("total_mw") or 0

            if total_mw <= 0:
                return None

            return (renewable_mw / total_mw * 100.0)

        except Exception as err:
            _LOGGER.error(f"Error calculating renewable percentage: {err}")
            return None

    @staticmethod
    async def get_prices_with_fallback(
        db_results: Optional[List[Dict]],
        db_age_minutes: int,
        country: str = "nl"
    ) -> Tuple[Optional[List[Dict]], str, str, bool]:
        """
        Get electricity prices with 4-tier fallback strategy.

        CRITICAL RULE: Energy-Charts prices MUST NOT trigger GO actions!

        4-Tier Fallback:
        1. Fresh ENTSO-E data → allow_go_action=True
        2. Stale ENTSO-E data → allow_go_action=True
        3. Energy-Charts fallback → allow_go_action=False (CRITICAL!)
        4. Cache fallback → allow_go_action=False

        Args:
            db_results: List of price records from database
            db_age_minutes: Age of database data
            country: Country code

        Returns:
            Tuple of (data, source, quality, allow_go_action)
            - data: List of price dicts or None
            - source: "ENTSO-E" | "Energy-Charts" | "Cache"
            - quality: "FRESH" | "STALE" | "FALLBACK" | "CACHED" | "UNAVAILABLE"
            - allow_go_action: bool (False for Energy-Charts!)
        """
        thresholds = FallbackManager.THRESHOLDS.get("prices", {"fresh": 15, "stale": 60})
        fresh_threshold = thresholds["fresh"]
        stale_threshold = thresholds["stale"]

        # Tier 1: Fresh ENTSO-E data
        if db_results and db_age_minutes < fresh_threshold:
            _LOGGER.info(f"Prices FRESH from ENTSO-E ({db_age_minutes} min)")
            # Store in PostgreSQL cache for persistence
            FallbackManager._cache_prices_to_db(db_results, "entsoe", "live", country)
            return (db_results, "ENTSO-E", "FRESH", True)  # GO actions allowed

        # Tier 2: Stale ENTSO-E data (acceptable)
        if db_results and db_age_minutes < stale_threshold:
            _LOGGER.info(f"Prices STALE from ENTSO-E ({db_age_minutes} min)")
            return (db_results, "ENTSO-E", "STALE", True)  # GO actions allowed

        # Tier 3: Energy-Charts fallback
        if FallbackManager._check_circuit_breaker():
            _LOGGER.warning("EC circuit breaker open - skipping Energy-Charts fallback")
        else:
            try:
                ec_prices = await EnergyChartsClient.fetch_prices(country=country, hours=48)
                if ec_prices and len(ec_prices) > 0:
                    _LOGGER.warning(f"Using Energy-Charts fallback ({len(ec_prices)} prices)")

                    # Cache in memory for quick access
                    cache_key = f"prices_{country}"
                    _fallback_cache[cache_key] = ec_prices

                    # Store in PostgreSQL cache for persistence
                    FallbackManager._cache_prices_to_db(ec_prices, "energy-charts", "estimated", country)

                    # CRITICAL: allow_go_action=False for Energy-Charts!
                    return (ec_prices, "Energy-Charts", "FALLBACK", False)  # NO GO actions
            except Exception as err:
                _LOGGER.error(f"Energy-Charts fallback failed: {err}")
                if "404" in str(err):
                    FallbackManager._open_circuit_breaker()

        # Tier 4: In-memory cache fallback (quick check)
        cache_key = f"prices_{country}"
        if cache_key in _fallback_cache:
            cached_prices = _fallback_cache[cache_key]
            _LOGGER.warning(f"Using in-memory cached prices ({len(cached_prices)} records)")
            return (cached_prices, "Cache", "CACHED", False)  # NO GO actions on cache

        # Tier 4b: PostgreSQL cache fallback (persistent, 24h)
        db_cached = price_cache_service.get_cached_prices(country=country, hours=24)
        if db_cached:
            _LOGGER.warning(f"Using PostgreSQL cached prices ({len(db_cached)} records)")
            return (db_cached, "Cache (DB)", "CACHED", False)  # NO GO actions on cache

        # Complete failure - return stale DB if available (but still no GO)
        if db_results:
            _LOGGER.warning(f"All fallback failed, using stale ENTSO-E ({db_age_minutes} min)")
            return (db_results, "ENTSO-E", "STALE", False)  # NO GO on very stale data

        # No data at all
        _LOGGER.error("All price data sources failed")
        return (None, "None", "UNAVAILABLE", False)

    @staticmethod
    def _cache_prices_to_db(prices: List[Dict], source: str, quality: str, country: str):
        """Store prices in PostgreSQL cache for 24h persistence."""
        try:
            from datetime import datetime, timezone

            for price_record in prices[:24]:  # Only cache first 24 hours
                timestamp_str = price_record.get("timestamp")
                price_value = price_record.get("price_eur_mwh")

                if timestamp_str and price_value is not None:
                    # Convert MWh to kWh
                    price_kwh = float(price_value) / 1000.0

                    # Parse timestamp
                    if isinstance(timestamp_str, str):
                        ts = datetime.fromisoformat(timestamp_str.replace("Z", "+00:00"))
                    else:
                        ts = timestamp_str

                    price_cache_service.store_price(
                        price=price_kwh,
                        source=source,
                        quality=quality,
                        country=country.upper(),
                        timestamp=ts
                    )
        except Exception as e:
            _LOGGER.error(f"Failed to cache prices to DB: {e}")