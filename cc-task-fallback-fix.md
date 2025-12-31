# CC Task: Fallback Procedure Onderzoek & Fix

## Context

Server: 135.181.255.83 (SSH as root)
Project: /opt/github/ha-energy-insights-nl
Database: energy_insights_nl
Venv: /opt/energy-insights-nl/venv

## Doel

1. Onderzoek huidige fallback implementatie
2. Bepaal correcte freshness thresholds per bron
3. Fix quality_status logic
4. Test fallback cascade werkt

---

## FASE 1: ONDERZOEK (30 min)

### 1.1 Code Analyse

Zoek en documenteer de huidige implementatie:

```bash
# Vind fallback gerelateerde code
grep -r "fallback" /opt/github/ha-energy-insights-nl/synctacles_db/ --include="*.py"
grep -r "quality_status\|FRESH\|STALE" /opt/github/ha-energy-insights-nl/synctacles_db/ --include="*.py"
grep -r "energy.charts\|energycharts\|ENERGY_CHARTS" /opt/github/ha-energy-insights-nl/synctacles_db/ --include="*.py"
```

Documenteer:
- Waar wordt quality_status gezet?
- Waar wordt fallback getriggerd?
- Hoe werkt Energy-Charts client?
- Waar zijn thresholds gedefinieerd?

### 1.2 Energy-Charts API Onderzoek

Test Energy-Charts response tijd en data freshness:

```bash
# Activeer venv
source /opt/energy-insights-nl/venv/bin/activate
cd /opt/github/ha-energy-insights-nl

# Test Energy-Charts API (als client bestaat)
python3 << 'EOF'
import time
import requests
from datetime import datetime, timezone

# Energy-Charts API test
# Documentatie: https://api.energy-charts.info/

BASE_URL = "https://api.energy-charts.info"

# Test 1: Power data Nederland
start = time.time()
try:
    response = requests.get(
        f"{BASE_URL}/public_power",
        params={
            "country": "nl",
            "start": "2025-12-30T00:00+01:00",
            "end": "2025-12-30T23:59+01:00"
        },
        timeout=30
    )
    elapsed = time.time() - start
    
    print(f"=== Energy-Charts API Test ===")
    print(f"Status: {response.status_code}")
    print(f"Response time: {elapsed:.2f}s")
    
    if response.ok:
        data = response.json()
        print(f"Data keys: {list(data.keys())}")
        
        # Check data freshness
        if 'unix_seconds' in data:
            timestamps = data['unix_seconds']
            if timestamps:
                latest = max(timestamps)
                latest_dt = datetime.fromtimestamp(latest, tz=timezone.utc)
                age_minutes = (datetime.now(timezone.utc) - latest_dt).total_seconds() / 60
                print(f"Latest data point: {latest_dt}")
                print(f"Data age: {age_minutes:.0f} minutes")
        
        # Show available generation types
        for key in data.keys():
            if key not in ['unix_seconds', 'datetime']:
                vals = data[key]
                if vals and any(v is not None for v in vals):
                    print(f"  {key}: {len([v for v in vals if v is not None])} values")
    else:
        print(f"Error: {response.text[:500]}")
        
except Exception as e:
    print(f"Error: {e}")
EOF
```

### 1.3 ENTSO-E Data Delay Meting

```bash
# Check werkelijke ENTSO-E delay in database
psql -U energy_insights_nl -d energy_insights_nl << 'EOF'
-- Verschil tussen data timestamp en wanneer we het opsloegen
SELECT 
    timestamp as data_timestamp,
    last_updated as opgeslagen_om,
    EXTRACT(EPOCH FROM (last_updated - timestamp))/60 as delay_minuten
FROM norm_entso_e_a75
WHERE last_updated IS NOT NULL
ORDER BY timestamp DESC
LIMIT 20;

-- Gemiddelde delay
SELECT 
    AVG(EXTRACT(EPOCH FROM (last_updated - timestamp))/60) as avg_delay_min,
    MIN(EXTRACT(EPOCH FROM (last_updated - timestamp))/60) as min_delay_min,
    MAX(EXTRACT(EPOCH FROM (last_updated - timestamp))/60) as max_delay_min
FROM norm_entso_e_a75
WHERE last_updated IS NOT NULL;
EOF
```

---

## FASE 2: FRESHNESS THRESHOLDS DEFINITIE (15 min)

Op basis van onderzoek, definieer thresholds:

### Verwachte Uitkomst

```python
# synctacles_db/config/freshness.py

FRESHNESS_THRESHOLDS = {
    "ENTSO-E": {
        "fresh": 120,      # Data < 2 uur = FRESH (structurele 60-90 min delay)
        "stale": 180,      # Data 2-3 uur = STALE
        # > 180 min = trigger fallback
    },
    "TenneT": {
        "fresh": 15,       # Data < 15 min = FRESH (near real-time)
        "stale": 30,       # Data 15-30 min = STALE
    },
    "Energy-Charts": {
        "fresh": ???,      # Bepaal uit onderzoek
        "stale": ???,      # Bepaal uit onderzoek
    },
    "Cache": {
        "fresh": 60,       # Cache < 1 uur = acceptabel
        "stale": 120,      # Cache 1-2 uur = STALE
    },
}

def get_quality_status(source: str, age_minutes: float) -> str:
    """Bepaal quality status op basis van bron en data leeftijd."""
    thresholds = FRESHNESS_THRESHOLDS.get(source, FRESHNESS_THRESHOLDS["ENTSO-E"])
    
    if age_minutes < thresholds["fresh"]:
        return "FRESH"
    elif age_minutes < thresholds["stale"]:
        return "STALE"
    else:
        return "NEEDS_FALLBACK"
```

---

## FASE 3: FALLBACK CASCADE IMPLEMENTATIE (45 min)

### 3.1 Fallback Manager

Maak of update `synctacles_db/services/fallback_manager.py`:

```python
"""
Fallback cascade voor data retrieval.

Volgorde:
1. ENTSO-E (primair, authoritative)
2. Energy-Charts (fallback, modeled data)
3. Cache (laatste bekende waarde)
4. None (UNAVAILABLE)
"""

from datetime import datetime, timezone
from typing import Optional, Dict, Any
import logging

from synctacles_db.config.freshness import FRESHNESS_THRESHOLDS, get_quality_status

logger = logging.getLogger(__name__)


class FallbackManager:
    """Manages data retrieval with automatic fallback."""
    
    def __init__(self, db_session, entso_e_client, energy_charts_client, cache):
        self.db = db_session
        self.entso_e = entso_e_client
        self.energy_charts = energy_charts_client
        self.cache = cache
    
    def get_generation_data(self) -> Dict[str, Any]:
        """
        Get generation data met fallback cascade.
        
        Returns dict met:
        - data: generation values
        - source: waar data vandaan komt
        - quality_status: FRESH/STALE/FALLBACK/CACHED/UNAVAILABLE
        - age_minutes: hoe oud is de data
        """
        
        # Tier 1: Check database (ENTSO-E data)
        db_data = self._get_from_database()
        if db_data:
            age = self._calculate_age(db_data['timestamp'])
            status = get_quality_status("ENTSO-E", age)
            
            if status in ["FRESH", "STALE"]:
                logger.info(f"Using ENTSO-E data (age: {age:.0f} min, status: {status})")
                return {
                    "data": db_data,
                    "source": "ENTSO-E",
                    "quality_status": status,
                    "age_minutes": age
                }
        
        # Tier 2: Energy-Charts fallback
        logger.warning("ENTSO-E data too old or missing, trying Energy-Charts")
        ec_data = self._get_from_energy_charts()
        if ec_data:
            age = self._calculate_age(ec_data['timestamp'])
            status = get_quality_status("Energy-Charts", age)
            
            # Mark database record for backfill
            self._mark_needs_backfill(db_data['timestamp'] if db_data else None)
            
            logger.info(f"Using Energy-Charts fallback (age: {age:.0f} min)")
            return {
                "data": ec_data,
                "source": "Energy-Charts",
                "quality_status": "FALLBACK",
                "age_minutes": age
            }
        
        # Tier 3: Cache fallback
        logger.warning("Energy-Charts failed, trying cache")
        cached_data = self._get_from_cache()
        if cached_data:
            age = self._calculate_age(cached_data['timestamp'])
            
            logger.info(f"Using cached data (age: {age:.0f} min)")
            return {
                "data": cached_data,
                "source": "Cache",
                "quality_status": "CACHED",
                "age_minutes": age
            }
        
        # Tier 4: No data available
        logger.error("All data sources failed - UNAVAILABLE")
        return {
            "data": None,
            "source": None,
            "quality_status": "UNAVAILABLE",
            "age_minutes": None
        }
    
    def _get_from_database(self) -> Optional[Dict]:
        """Get latest ENTSO-E data from database."""
        # TODO: Implement actual query
        pass
    
    def _get_from_energy_charts(self) -> Optional[Dict]:
        """Get data from Energy-Charts API."""
        # TODO: Implement actual API call
        pass
    
    def _get_from_cache(self) -> Optional[Dict]:
        """Get data from local cache."""
        # TODO: Implement cache retrieval
        pass
    
    def _calculate_age(self, timestamp: datetime) -> float:
        """Calculate age in minutes."""
        now = datetime.now(timezone.utc)
        if timestamp.tzinfo is None:
            timestamp = timestamp.replace(tzinfo=timezone.utc)
        return (now - timestamp).total_seconds() / 60
    
    def _mark_needs_backfill(self, timestamp: Optional[datetime]):
        """Mark timestamp for later ENTSO-E backfill."""
        if timestamp:
            # TODO: Update needs_backfill flag in database
            logger.info(f"Marked {timestamp} for backfill")
```

### 3.2 Update API Endpoint

Update generation endpoint om FallbackManager te gebruiken:

```python
# In synctacles_db/api/routes/generation.py

@router.get("/v1/generation-mix")
async def get_generation_mix():
    """Get current generation mix met automatische fallback."""
    
    fallback_mgr = FallbackManager(...)
    result = fallback_mgr.get_generation_data()
    
    return {
        "timestamp": result["data"]["timestamp"] if result["data"] else None,
        "data": result["data"],
        "meta": {
            "source": result["source"],
            "quality_status": result["quality_status"],
            "age_minutes": result["age_minutes"],
            "fallback_used": result["source"] != "ENTSO-E"
        }
    }
```

### 3.3 Energy-Charts Client

Maak of update `synctacles_db/collectors/energy_charts_client.py`:

```python
"""
Energy-Charts API client voor fallback data.
API docs: https://api.energy-charts.info/
"""

import requests
from datetime import datetime, timezone
from typing import Optional, Dict
import logging

logger = logging.getLogger(__name__)


class EnergyChartsClient:
    """Client for Energy-Charts public API."""
    
    BASE_URL = "https://api.energy-charts.info"
    TIMEOUT = 30
    
    # Mapping Energy-Charts keys -> onze keys
    FIELD_MAPPING = {
        "nuclear": "b14_nuclear_mw",
        "hydro": "b12_hydro_mw",
        "biomass": "b01_biomass_mw",
        "gas": "b04_gas_mw",
        "coal": "b05_coal_mw",
        "solar": "b16_solar_mw",
        "wind_onshore": "b19_wind_onshore_mw",
        "wind_offshore": "b18_wind_offshore_mw",
        # etc.
    }
    
    def get_current_generation(self, country: str = "nl") -> Optional[Dict]:
        """
        Get current generation mix from Energy-Charts.
        
        Args:
            country: Country code (nl, de, etc.)
            
        Returns:
            Dict with generation data or None if failed
        """
        try:
            response = requests.get(
                f"{self.BASE_URL}/public_power",
                params={"country": country},
                timeout=self.TIMEOUT
            )
            response.raise_for_status()
            
            data = response.json()
            return self._parse_response(data)
            
        except requests.RequestException as e:
            logger.error(f"Energy-Charts API error: {e}")
            return None
    
    def _parse_response(self, data: Dict) -> Optional[Dict]:
        """Parse Energy-Charts response to our format."""
        try:
            # Get latest timestamp
            timestamps = data.get("unix_seconds", [])
            if not timestamps:
                return None
            
            latest_idx = -1  # Last entry
            latest_ts = timestamps[latest_idx]
            
            # Build generation dict
            generation = {
                "timestamp": datetime.fromtimestamp(latest_ts, tz=timezone.utc),
                "source": "Energy-Charts",
            }
            
            total = 0
            for ec_key, our_key in self.FIELD_MAPPING.items():
                if ec_key in data:
                    values = data[ec_key]
                    if values and values[latest_idx] is not None:
                        generation[our_key] = values[latest_idx]
                        total += values[latest_idx]
                    else:
                        generation[our_key] = 0
            
            generation["total_mw"] = total
            return generation
            
        except (KeyError, IndexError) as e:
            logger.error(f"Error parsing Energy-Charts response: {e}")
            return None
```

---

## FASE 4: DATABASE UPDATES (15 min)

### 4.1 Add Missing Columns (indien nodig)

```sql
-- Check of needs_backfill bestaat
SELECT column_name 
FROM information_schema.columns 
WHERE table_name = 'norm_entso_e_a75' 
AND column_name = 'needs_backfill';

-- Voeg toe indien niet bestaat
ALTER TABLE norm_entso_e_a75 
ADD COLUMN IF NOT EXISTS needs_backfill BOOLEAN DEFAULT false;

ALTER TABLE norm_entso_e_a75 
ADD COLUMN IF NOT EXISTS data_source VARCHAR(50) DEFAULT 'ENTSO-E';

-- Index voor backfill queries
CREATE INDEX IF NOT EXISTS idx_needs_backfill 
ON norm_entso_e_a75(needs_backfill) 
WHERE needs_backfill = true;
```

### 4.2 Update Existing Records

```sql
-- Update quality_status voor bestaande data (met nieuwe thresholds)
-- ENTSO-E: FRESH < 120 min, STALE < 180 min
UPDATE norm_entso_e_a75
SET quality_status = CASE
    WHEN EXTRACT(EPOCH FROM (NOW() - timestamp))/60 < 120 THEN 'FRESH'
    WHEN EXTRACT(EPOCH FROM (NOW() - timestamp))/60 < 180 THEN 'STALE'
    ELSE 'STALE'  -- Historical data stays STALE
END
WHERE quality_status IS NOT NULL;

-- Set data_source voor bestaande records
UPDATE norm_entso_e_a75
SET data_source = 'ENTSO-E'
WHERE data_source IS NULL;
```

---

## FASE 5: TESTING (30 min)

### 5.1 Unit Test Freshness Logic

```python
# Test freshness thresholds
python3 << 'EOF'
from datetime import datetime, timezone, timedelta

# Simulate freshness calculation
def get_quality_status(source: str, age_minutes: float) -> str:
    thresholds = {
        "ENTSO-E": {"fresh": 120, "stale": 180},
        "TenneT": {"fresh": 15, "stale": 30},
        "Energy-Charts": {"fresh": 60, "stale": 120},
    }
    t = thresholds.get(source, thresholds["ENTSO-E"])
    
    if age_minutes < t["fresh"]:
        return "FRESH"
    elif age_minutes < t["stale"]:
        return "STALE"
    else:
        return "NEEDS_FALLBACK"

# Tests
print("=== Freshness Logic Tests ===")
print(f"ENTSO-E 60 min: {get_quality_status('ENTSO-E', 60)}")   # FRESH
print(f"ENTSO-E 90 min: {get_quality_status('ENTSO-E', 90)}")   # FRESH
print(f"ENTSO-E 130 min: {get_quality_status('ENTSO-E', 130)}") # STALE
print(f"ENTSO-E 200 min: {get_quality_status('ENTSO-E', 200)}") # NEEDS_FALLBACK
print(f"TenneT 10 min: {get_quality_status('TenneT', 10)}")     # FRESH
print(f"TenneT 25 min: {get_quality_status('TenneT', 25)}")     # STALE
print(f"Energy-Charts 30 min: {get_quality_status('Energy-Charts', 30)}") # FRESH
EOF
```

### 5.2 Integration Test Fallback

```bash
# Test fallback door ENTSO-E data te "verouderen"
psql -U energy_insights_nl -d energy_insights_nl << 'EOF'
-- Tijdelijk: maak nieuwste record heel oud om fallback te triggeren
-- (bewaar origineel timestamp eerst!)
SELECT timestamp, quality_status 
FROM norm_entso_e_a75 
ORDER BY timestamp DESC LIMIT 1;
EOF

# Test API response
curl -s https://enin.xteleo.nl/api/v1/generation-mix | jq '.meta'
```

### 5.3 Test Energy-Charts Direct

```bash
# Direct Energy-Charts call testen
curl -s "https://api.energy-charts.info/public_power?country=nl" | jq 'keys'
```

---

## FASE 6: DOCUMENTATIE (15 min)

Update de volgende documenten:

### 6.1 SKILL_02_ARCHITECTURE.md

Voeg sectie toe:

```markdown
## Fallback Strategy

### Freshness Thresholds (per bron)

| Bron | FRESH | STALE | Fallback Trigger |
|------|-------|-------|------------------|
| ENTSO-E | < 120 min | 120-180 min | > 180 min |
| TenneT | < 15 min | 15-30 min | > 30 min |
| Energy-Charts | < 60 min | 60-120 min | > 120 min |
| Cache | < 60 min | 60-120 min | > 120 min |

### Fallback Cascade

1. ENTSO-E (primair) → FRESH/STALE
2. Energy-Charts (fallback) → FALLBACK status
3. Cache (nood) → CACHED status
4. None → UNAVAILABLE status
```

### 6.2 SKILL_06_DATA_SOURCES.md

Update Energy-Charts sectie met werkelijke response times.

---

## DELIVERABLES

Na afronding:

1. ✅ Freshness thresholds config file
2. ✅ FallbackManager class
3. ✅ Energy-Charts client (werkend)
4. ✅ Updated API endpoint met fallback
5. ✅ Database schema updates
6. ✅ Tests (unit + integration)
7. ✅ Documentatie updates
8. ✅ Git commit + push

---

## VALIDATIE CHECKLIST

- [ ] Energy-Charts API response tijd gemeten
- [ ] ENTSO-E gemiddelde delay berekend
- [ ] Freshness thresholds gedefinieerd per bron
- [ ] quality_status correct gezet (niet meer 100% STALE)
- [ ] Fallback naar Energy-Charts werkt
- [ ] Cache fallback werkt
- [ ] needs_backfill flag werkt
- [ ] API response bevat juiste meta info
- [ ] Documentatie bijgewerkt
- [ ] Alle code in git

---

## GESCHATTE TIJD

| Fase | Tijd |
|------|------|
| Onderzoek | 30 min |
| Thresholds definitie | 15 min |
| Fallback implementatie | 45 min |
| Database updates | 15 min |
| Testing | 30 min |
| Documentatie | 15 min |
| **Totaal** | **~2.5 uur** |
