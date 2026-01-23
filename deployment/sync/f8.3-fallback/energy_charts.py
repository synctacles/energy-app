"""Energy-Charts API Client - Complete mapping"""

import logging
from datetime import UTC, datetime

import httpx

logger = logging.getLogger(__name__)
BASE_URL = "https://api.energy-charts.info"


async def fetch_generation_mix(country: str = "nl") -> list[dict] | None:
    """Fetch and normalize Energy-Charts generation data"""
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(
                f"{BASE_URL}/public_power", params={"country": country}, timeout=10.0
            )

            if response.status_code != 200:
                return None

            raw = response.json()
            normalized = _normalize_generation(raw)
            logger.info(f"Energy-Charts: {len(normalized)} datapoints")
            return normalized

    except Exception as e:
        logger.error(f"Energy-Charts error: {e}")
        return None


def _normalize_generation(raw: dict) -> list[dict]:
    """Convert Energy-Charts to SYNCTACLES canonical format"""
    if not raw or "unix_seconds" not in raw:
        return []

    timestamps = raw["unix_seconds"]
    production_types = raw.get("production_types", [])

    # Exact mapping based on Energy-Charts response
    type_mapping = {
        "Solar": "solar_mw",
        "Wind offshore": "wind_offshore_mw",
        "Wind onshore": "wind_onshore_mw",
        "Fossil gas": "gas_mw",
        "Fossil hard coal": "coal_mw",
        "Nuclear": "nuclear_mw",
        "Biomass": "biomass_mw",
        "Waste": "waste_mw",
        "Others": "other_mw",
    }

    # Index production types by mapped name
    type_data = {}
    for ptype in production_types:
        name = ptype.get("name", "")
        if name in type_mapping:
            type_data[type_mapping[name]] = ptype.get("data", [])

    normalized = []

    for i, unix_ts in enumerate(timestamps):
        dt = datetime.fromtimestamp(unix_ts, tz=UTC)

        record = {"timestamp": dt.isoformat()}
        total = 0.0

        # All SYNCTACLES fields
        for field in [
            "solar_mw",
            "wind_offshore_mw",
            "wind_onshore_mw",
            "gas_mw",
            "coal_mw",
            "nuclear_mw",
            "biomass_mw",
            "waste_mw",
            "other_mw",
        ]:
            values = type_data.get(field, [])
            value = values[i] if i < len(values) else 0.0
            # Only positive values (generation, not consumption/import)
            record[field] = round(value, 2) if value > 0 else 0.0
            total += record[field]

        record["total_mw"] = round(total, 2)
        normalized.append(record)

    return normalized
