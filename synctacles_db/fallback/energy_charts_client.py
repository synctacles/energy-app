"""Energy-Charts API client for fallback data source.

Provides generation mix data from Fraunhofer ISE when ENTSO-E is unavailable.
API: https://api.energy-charts.info/
"""

from datetime import datetime, timezone
from typing import List, Dict, Optional
import logging
import aiohttp

_LOGGER = logging.getLogger(__name__)


class EnergyChartsClient:
    """Client for Energy-Charts API (Fraunhofer ISE)."""
    
    BASE_URL = "https://api.energy-charts.info"
    
    # Map Energy-Charts types to SYNCTACLES schema
    TYPE_MAPPING = {
        "Solar": "solar_mw",
        "Wind offshore": "wind_offshore_mw",
        "Wind onshore": "wind_onshore_mw",
        "Fossil gas": "gas_mw",
        "Fossil hard coal": "coal_mw",
        "Nuclear": "nuclear_mw",
        "Biomass": "biomass_mw",
        "Waste": "waste_mw",
        "Hydro Run-of-river": "hydro_mw",
        "Hydro pumped storage": "pumped_storage_mw",
        "Others": "other_mw",
    }
    
    @staticmethod
    async def fetch_generation_mix(
        country: str = "nl",
        limit: int = 1
    ) -> List[Dict]:
        """
        Fetch latest generation mix data from Energy-Charts.
        
        Args:
            country: Country code (nl, de, fr, be)
            limit: Number of latest records to return
            
        Returns:
            List of generation mix records with SYNCTACLES schema
        """
        try:
            url = f"{EnergyChartsClient.BASE_URL}/public_power"
            params = {
                "country": country,
            }
            
            async with aiohttp.ClientSession() as session:
                async with session.get(url, params=params, timeout=10) as response:
                    if response.status != 200:
                        _LOGGER.error(f"Energy-Charts API error: HTTP {response.status}")
                        return []
                    
                    data = await response.json()
                    
                    # Parse nested structure
                    # Response format: {unix_seconds: [...], production_types: [{name, data: [...]}]}
                    return EnergyChartsClient._parse_response(data, limit)
        
        except aiohttp.ClientError as err:
            _LOGGER.error(f"Energy-Charts connection error: {err}")
            return []
        except Exception as err:
            _LOGGER.error(f"Energy-Charts unexpected error: {err}")
            return []
    
    @staticmethod
    def _parse_response(data: Dict, limit: int) -> List[Dict]:
        """Parse Energy-Charts response to SYNCTACLES format."""
        try:
            timestamps = data.get("unix_seconds", [])
            production_types = data.get("production_types", [])
            
            if not timestamps or not production_types:
                _LOGGER.warning("Energy-Charts returned empty data")
                return []
            
            # Get latest N timestamps
            latest_timestamps = timestamps[-limit:] if len(timestamps) >= limit else timestamps
            
            results = []
            
            for idx, ts in enumerate(latest_timestamps):
                record = {
                    "timestamp": datetime.fromtimestamp(ts, tz=timezone.utc).isoformat(),
                    "source": "Energy-Charts",
                }
                
                total_mw = 0.0
                
                # Extract values for each production type
                for prod_type in production_types:
                    type_name = prod_type.get("name")
                    values = prod_type.get("data", [])
                    
                    if type_name not in EnergyChartsClient.TYPE_MAPPING:
                        continue
                    
                    # Get value at this timestamp index (from end)
                    value_idx = len(values) - limit + idx
                    if value_idx < 0 or value_idx >= len(values):
                        continue
                    
                    value = values[value_idx]
                    if value is None:
                        value = 0.0
                    
                    # Map to SYNCTACLES field
                    field_name = EnergyChartsClient.TYPE_MAPPING[type_name]
                    record[field_name] = float(value)
                    total_mw += float(value)
                
                # Set defaults for missing fields
                for field in EnergyChartsClient.TYPE_MAPPING.values():
                    if field not in record:
                        record[field] = 0.0
                
                record["total_mw"] = total_mw
                results.append(record)
            
            return results
        
        except Exception as err:
            _LOGGER.error(f"Energy-Charts parse error: {err}")
            return []


# Synchronous wrapper for use in non-async contexts
def fetch_generation_mix_sync(country: str = "nl", limit: int = 1) -> List[Dict]:
    """Synchronous wrapper for Energy-Charts fetch."""
    import asyncio
    
    try:
        loop = asyncio.get_event_loop()
    except RuntimeError:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
    
    return loop.run_until_complete(
        EnergyChartsClient.fetch_generation_mix(country, limit)
    )