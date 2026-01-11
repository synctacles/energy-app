# PLAN VAN AANPAK: Energy Action Focus

**Voor:** Claude Code  
**Van:** Claude (Opus) + Leo  
**Datum:** 2026-01-11  
**Doel:** SYNCTACLES strippen naar Energy Action only + robuuste fallback chain bouwen

---

## CONTEXT

### Beslissing
SYNCTACLES focust exclusief op **Energy Action** als product. Alle "nerd features" (grid data, generation, load) worden verwijderd om:
- Support footprint te minimaliseren
- Codebase te simplificeren
- Focus te houden op killer feature

### Kritieke voorwaarde
**EERST fallbacks bouwen, DAN strippen.**

Een kapotte fallback chain bij uitval van Enever/ENTSO-E = onacceptabel voor €30/jaar product.

### Gewenste eindsituatie

```
┌─────────────────────────────────────────────────────────────┐
│                     FALLBACK CHAIN                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Tier 1: Live Consumer Price (Enever)                       │
│     │    └── 100% accuraat                                  │
│     │                                                       │
│     ▼ (als Enever faalt)                                    │
│                                                             │
│  Tier 2: ENTSO-E + Lookup Table                             │
│     │    └── 89% accuraat                                   │
│     │                                                       │
│     ▼ (als ENTSO-E faalt)                                   │
│                                                             │
│  Tier 3: Energy-Charts + Lookup Table                       │
│     │    └── 89% accuraat (zelfde data, andere bron)        │
│     │                                                       │
│     ▼ (als Energy-Charts faalt)                             │
│                                                             │
│  Tier 4: Cached Data (laatste 24h)                          │
│     │    └── Degraded, maar functioneel                     │
│     │                                                       │
│     ▼ (als cache leeg/stale)                                │
│                                                             │
│  Tier 5: UNKNOWN status                                     │
│          └── User ziet "geen data beschikbaar"              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## FASERING

```
FASE 1: Fallback Infrastructure (MOET EERST)
   │
   ├── 1.1 Energy-Charts collector
   ├── 1.2 Fallback chain in price service
   ├── 1.3 Cache layer
   └── 1.4 Quality indicator
   │
   ▼
FASE 2: Soft Delete (Disable, niet verwijderen)
   │
   ├── 2.1 Grid endpoints → 410 Gone
   ├── 2.2 HA entities conditioneel disabled
   └── 2.3 Importers/normalizers skippen
   │
   ▼
FASE 3: Hard Delete (Code cleanup)
   │
   ├── 3.1 TenneT code verwijderen
   ├── 3.2 A65/A75 code verwijderen
   ├── 3.3 Grid stress logica verwijderen
   └── 3.4 Unused dependencies verwijderen
   │
   ▼
FASE 4: Documentatie & GitHub Cleanup
   │
   ├── 4.1 SKILL files updaten
   ├── 4.2 API docs updaten
   ├── 4.3 GitHub issues sluiten/archiveren
   └── 4.4 Architecture docs updaten
```

---

## FASE 1: FALLBACK INFRASTRUCTURE

**Prioriteit:** CRITICAL  
**Geschatte tijd:** 8-10 uur  
**Exit criteria:** Alle tiers werken, getest, quality indicator zichtbaar

### 1.1 Energy-Charts Collector

**GitHub Issue:** `#60 - Add Energy-Charts as ENTSO-E fallback`  
**Priority:** CRITICAL  
**Labels:** `fallback`, `reliability`, `fase-1`

**Beschrijving:**
Energy-Charts (Fraunhofer ISE) levert dezelfde ENTSO-E data via alternatieve API. Dient als fallback wanneer ENTSO-E direct niet bereikbaar is.

**Acceptatiecriteria:**
- [ ] `app/collectors/energy_charts_collector.py` aangemaakt
- [ ] Endpoint: `https://api.energy-charts.info/price?country=nl`
- [ ] Response parsing identiek aan A44 normalizer output
- [ ] Error handling met logging
- [ ] Unit tests

**Technische details:**

```python
# app/collectors/energy_charts_collector.py

import httpx
from datetime import datetime, timezone
from typing import Optional, List, Dict

ENERGY_CHARTS_URL = "https://api.energy-charts.info/price"

async def fetch_prices(country: str = "nl") -> Optional[List[Dict]]:
    """
    Fetch day-ahead prices from Energy-Charts API.
    
    Returns list of:
    {
        "timestamp": datetime,
        "price_eur_mwh": float,
        "source": "energy-charts"
    }
    """
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            response = await client.get(
                ENERGY_CHARTS_URL,
                params={"country": country}
            )
            response.raise_for_status()
            data = response.json()
            
            prices = []
            for i, unix_ts in enumerate(data.get("unix_seconds", [])):
                prices.append({
                    "timestamp": datetime.fromtimestamp(unix_ts, tz=timezone.utc),
                    "price_eur_mwh": data["price"][i],
                    "source": "energy-charts"
                })
            
            return prices
            
    except Exception as e:
        logger.error(f"Energy-Charts fetch failed: {e}")
        return None
```

**Geschatte tijd:** 2 uur

---

### 1.2 Fallback Chain in Price Service

**GitHub Issue:** `#61 - Implement price fallback chain`  
**Priority:** CRITICAL  
**Labels:** `fallback`, `reliability`, `fase-1`

**Beschrijving:**
Price service moet automatisch door fallback tiers gaan wanneer primaire bron faalt.

**Acceptatiecriteria:**
- [ ] Tier 1: Enever (live consumer price)
- [ ] Tier 2: ENTSO-E + lookup table
- [ ] Tier 3: Energy-Charts + lookup table
- [ ] Tier 4: Cache (laatste 24h)
- [ ] Tier 5: Return UNKNOWN
- [ ] Elk tier logt welke bron gebruikt wordt
- [ ] Quality/source metadata in response

**Technische details:**

```python
# app/services/price_service.py

from app.collectors import enever, entsoe, energy_charts
from app.cache import price_cache
from app.config import HOURLY_OFFSET

class PriceQuality:
    LIVE = "live"           # 100% accurate
    ESTIMATED = "estimated" # 89% accurate (lookup)
    CACHED = "cached"       # Stale maar functioneel
    UNAVAILABLE = "unavailable"

async def get_current_price() -> dict:
    """
    Get current consumer price with fallback chain.
    """
    hour = datetime.now().hour
    
    # Tier 1: Live Enever
    price = await enever.get_current_price()
    if price is not None:
        return {
            "price_eur_kwh": price,
            "quality": PriceQuality.LIVE,
            "source": "enever",
            "confidence": 100
        }
    
    # Tier 2: ENTSO-E + lookup
    wholesale = await entsoe.get_current_price()
    if wholesale is not None:
        consumer_price = (wholesale / 1000) + HOURLY_OFFSET[hour]
        return {
            "price_eur_kwh": consumer_price,
            "quality": PriceQuality.ESTIMATED,
            "source": "entsoe+lookup",
            "confidence": 89
        }
    
    # Tier 3: Energy-Charts + lookup
    wholesale = await energy_charts.get_current_price()
    if wholesale is not None:
        consumer_price = (wholesale / 1000) + HOURLY_OFFSET[hour]
        return {
            "price_eur_kwh": consumer_price,
            "quality": PriceQuality.ESTIMATED,
            "source": "energy-charts+lookup",
            "confidence": 89
        }
    
    # Tier 4: Cache
    cached = await price_cache.get_last_known()
    if cached is not None:
        return {
            "price_eur_kwh": cached["price"],
            "quality": PriceQuality.CACHED,
            "source": "cache",
            "confidence": 50,
            "cached_at": cached["timestamp"]
        }
    
    # Tier 5: Unavailable
    return {
        "price_eur_kwh": None,
        "quality": PriceQuality.UNAVAILABLE,
        "source": None,
        "confidence": 0
    }
```

**Geschatte tijd:** 3 uur

---

### 1.3 Cache Layer

**GitHub Issue:** `#62 - Add 24h price cache for fallback`  
**Priority:** HIGH  
**Labels:** `fallback`, `reliability`, `fase-1`

**Beschrijving:**
Cache laag die laatste 24 uur aan prijzen bewaart voor Tier 4 fallback.

**Acceptatiecriteria:**
- [ ] Cache in PostgreSQL (niet Redis, KISS)
- [ ] Rolling window van 24 uur
- [ ] Automatische cleanup van oude entries
- [ ] `get_last_known()` retourneert meest recente prijs

**Technische details:**

```sql
-- Nieuwe tabel (of hergebruik bestaande)
CREATE TABLE IF NOT EXISTS price_cache (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    price_eur_kwh DECIMAL(10,6) NOT NULL,
    source TEXT NOT NULL,
    quality TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_price_cache_timestamp ON price_cache(timestamp DESC);

-- Cleanup oude entries (via cron of app)
DELETE FROM price_cache WHERE timestamp < NOW() - INTERVAL '24 hours';
```

```python
# app/cache/price_cache.py

async def store_price(price: float, source: str, quality: str):
    """Store price in cache."""
    await db.execute("""
        INSERT INTO price_cache (timestamp, price_eur_kwh, source, quality)
        VALUES (NOW(), $1, $2, $3)
    """, price, source, quality)

async def get_last_known() -> Optional[dict]:
    """Get most recent cached price."""
    row = await db.fetchrow("""
        SELECT price_eur_kwh, source, timestamp
        FROM price_cache
        ORDER BY timestamp DESC
        LIMIT 1
    """)
    if row:
        return {
            "price": float(row["price_eur_kwh"]),
            "source": row["source"],
            "timestamp": row["timestamp"]
        }
    return None
```

**Geschatte tijd:** 2 uur

---

### 1.4 Quality Indicator op API/Sensor

**GitHub Issue:** `#63 - Add quality indicator to Energy Action`  
**Priority:** HIGH  
**Labels:** `api`, `ha-component`, `fase-1`

**Beschrijving:**
Energy Action response moet quality metadata bevatten zodat:
1. API consumers weten hoe betrouwbaar de data is
2. HA component dit als attribute toont
3. Users automations kunnen bouwen met confidence check

**Acceptatiecriteria:**
- [ ] API response bevat `quality`, `source`, `confidence`
- [ ] HA sensor heeft attributes: `quality`, `source`, `confidence`, `last_update`
- [ ] Documentatie bijgewerkt

**API Response (nieuw):**

```json
{
  "action": "USE",
  "price_eur_kwh": 0.1845,
  "quality": "live",
  "source": "enever",
  "confidence": 100,
  "timestamp": "2026-01-11T14:30:00Z"
}
```

**HA Sensor (nieuw):**

```yaml
sensor.energy_action:
  state: "USE"
  attributes:
    price_eur_kwh: 0.1845
    quality: "live"
    source: "enever"
    confidence: 100
    last_update: "2026-01-11T14:30:00Z"
```

**Geschatte tijd:** 2 uur (1 uur API, 1 uur HA component)

---

## FASE 2: SOFT DELETE

**Prioriteit:** HIGH  
**Geschatte tijd:** 4-6 uur  
**Exit criteria:** Alle grid endpoints return 410, HA toont alleen 6 entities

**Prerequisite:** Fase 1 MOET compleet en getest zijn.

---

### 2.1 Grid Endpoints → 410 Gone

**GitHub Issue:** `#64 - Disable grid/generation endpoints`  
**Priority:** HIGH  
**Labels:** `cleanup`, `soft-delete`, `fase-2`

**Beschrijving:**
Alle niet-prijs endpoints returnen HTTP 410 Gone met duidelijke message.

**Acceptatiecriteria:**
- [ ] `/api/v1/generation` → 410
- [ ] `/api/v1/load` → 410
- [ ] `/api/v1/balance` → 410
- [ ] `/api/v1/grid-stress` → 410
- [ ] Response body bevat uitleg

**Technische details:**

```python
# app/api/v1/deprecated.py

from fastapi import APIRouter, HTTPException

router = APIRouter()

DISCONTINUED_MESSAGE = {
    "error": "Endpoint discontinued",
    "message": "This endpoint has been removed. SYNCTACLES now focuses exclusively on Energy Action.",
    "documentation": "https://docs.synctacles.io/migration"
}

@router.get("/generation")
@router.get("/load")
@router.get("/balance")
@router.get("/grid-stress")
async def discontinued_endpoint():
    raise HTTPException(
        status_code=410,
        detail=DISCONTINUED_MESSAGE
    )
```

**Geschatte tijd:** 1 uur

---

### 2.2 HA Entities Conditioneel Disabled

**GitHub Issue:** `#65 - Reduce HA component to 6 entities`  
**Priority:** HIGH  
**Labels:** `ha-component`, `soft-delete`, `fase-2`

**Beschrijving:**
HA component maakt alleen nog de 6 core entities aan.

**Entities die BLIJVEN (6):**
```
sensor.energy_action
sensor.electricity_price
sensor.cheapest_hour
sensor.most_expensive_hour
sensor.prices_today
sensor.prices_tomorrow
```

**Entities die VERDWIJNEN (6):**
```
sensor.generation_total      # ❌
sensor.load_actual           # ❌
sensor.balance_delta         # ❌
sensor.grid_stress           # ❌
sensor.price_level           # ❌
sensor.price_status          # ❌
```

**Acceptatiecriteria:**
- [ ] Coordinator fetcht geen grid data meer
- [ ] Entity registry bevat alleen 6 entities
- [ ] Bestaande users krijgen geen errors (graceful removal)
- [ ] CHANGELOG bijgewerkt

**Technische details:**

```python
# custom_components/energy_insights_nl/sensor.py

# VERWIJDER deze entity descriptions:
# - EnergyInsightsGenerationSensor
# - EnergyInsightsLoadSensor
# - EnergyInsightsBalanceSensor
# - EnergyInsightsGridStressSensor
# - EnergyInsightsPriceLevelSensor
# - EnergyInsightsPriceStatusSensor

SENSOR_DESCRIPTIONS = [
    EnergyInsightsEnergyActionSensor,      # ✅ Blijft
    EnergyInsightsElectricityPriceSensor,  # ✅ Blijft
    EnergyInsightsCheapestHourSensor,      # ✅ Blijft
    EnergyInsightsMostExpensiveHourSensor, # ✅ Blijft
    EnergyInsightsPricesTodaySensor,       # ✅ Blijft
    EnergyInsightsPricesTomorrowSensor,    # ✅ Blijft
]
```

**Geschatte tijd:** 2 uur

---

### 2.3 Importers/Normalizers Skippen

**GitHub Issue:** `#66 - Skip TenneT and A65/A75 processing`  
**Priority:** MEDIUM  
**Labels:** `cleanup`, `soft-delete`, `fase-2`

**Beschrijving:**
Scripts skippen TenneT, A65, A75 maar code blijft (nog) intact.

**Acceptatiecriteria:**
- [ ] `run_importers.sh` voert alleen A44 uit
- [ ] `run_normalizers.sh` voert alleen A44 uit
- [ ] Systemd timers ongewijzigd (minder werk, zelfde schedule)
- [ ] Logs tonen "skipped" voor disabled importers

**Technische details:**

```bash
#!/bin/bash
# scripts/run_importers.sh

echo "Running importers..."

# Active
python -m app.importers.a44_importer

# Disabled (soft delete)
# python -m app.importers.a65_importer  # SKIPPED
# python -m app.importers.a75_importer  # SKIPPED
# python -m app.importers.tennet_importer  # SKIPPED

echo "Importers complete"
```

**Geschatte tijd:** 0.5 uur

---

## FASE 3: HARD DELETE

**Prioriteit:** MEDIUM  
**Geschatte tijd:** 4-6 uur  
**Exit criteria:** Geen unused code meer, dependencies opgeschoond

**Prerequisite:** Fase 2 stabiel gedraaid (minimaal 1 week)

---

### 3.1 TenneT Code Verwijderen

**GitHub Issue:** `#67 - Remove TenneT integration code`  
**Priority:** MEDIUM  
**Labels:** `cleanup`, `hard-delete`, `fase-3`

**Te verwijderen:**
- [ ] `app/importers/tennet_importer.py`
- [ ] `app/collectors/tennet_collector.py`
- [ ] `app/services/tennet_service.py`
- [ ] Gerelateerde tests
- [ ] Config entries voor TenneT

**Geschatte tijd:** 1.5 uur

---

### 3.2 A65/A75 Code Verwijderen

**GitHub Issue:** `#68 - Remove A65/A75 (load/generation) code`  
**Priority:** MEDIUM  
**Labels:** `cleanup`, `hard-delete`, `fase-3`

**Te verwijderen:**
- [ ] `app/importers/a65_importer.py`
- [ ] `app/importers/a75_importer.py`
- [ ] `app/normalizers/a65_normalizer.py`
- [ ] `app/normalizers/a75_normalizer.py`
- [ ] Gerelateerde modellen en schemas
- [ ] Gerelateerde tests

**Geschatte tijd:** 2 uur

---

### 3.3 Grid Stress Logica Verwijderen

**GitHub Issue:** `#69 - Remove grid stress calculation`  
**Priority:** LOW  
**Labels:** `cleanup`, `hard-delete`, `fase-3`

**Te verwijderen:**
- [ ] `app/services/grid_stress_service.py`
- [ ] Grid stress endpoint definitie
- [ ] Gerelateerde utilities

**Geschatte tijd:** 1 uur

---

### 3.4 Unused Dependencies Verwijderen

**GitHub Issue:** `#70 - Clean up unused dependencies`  
**Priority:** LOW  
**Labels:** `cleanup`, `hard-delete`, `fase-3`

**Acceptatiecriteria:**
- [ ] `pip-autoremove` of handmatige check
- [ ] `requirements.txt` opgeschoond
- [ ] Geen import errors na cleanup

**Geschatte tijd:** 1 uur

---

## FASE 4: DOCUMENTATIE & GITHUB

**Prioriteit:** HIGH  
**Geschatte tijd:** 3-4 uur  
**Exit criteria:** Alle docs actueel, GitHub issues opgeruimd

---

### 4.1 SKILL Files Updaten

**GitHub Issue:** `#71 - Update SKILL documentation for Energy Action focus`  
**Priority:** HIGH  
**Labels:** `documentation`, `fase-4`

**Te updaten:**
- [ ] SKILL_02_ARCHITECTURE.md - TenneT/A65/A75 sectie verwijderen
- [ ] SKILL_04_PRODUCT_REQUIREMENTS.md - Focus op 6 entities
- [ ] SKILL_06_DATA_SOURCES.md - TenneT verwijderen, fallback chain toevoegen
- [ ] SKILL_13_LOGGING.md - Fallback logging documenteren
- [ ] SKILL_14_COEFFICIENT.md - Fallback chain beschrijven

**Geschatte tijd:** 2 uur

---

### 4.2 API Docs Updaten

**GitHub Issue:** `#72 - Update API reference for discontinued endpoints`  
**Priority:** HIGH  
**Labels:** `documentation`, `fase-4`

**Te updaten:**
- [ ] `api-reference.md` - Discontinued endpoints markeren
- [ ] OpenAPI spec - 410 responses documenteren
- [ ] Migration guide schrijven

**Geschatte tijd:** 1 uur

---

### 4.3 GitHub Issues Cleanup

**GitHub Issue:** `#73 - Archive obsolete GitHub issues`  
**Priority:** LOW  
**Labels:** `housekeeping`, `fase-4`

**Acceptatiecriteria:**
- [ ] Issues gerelateerd aan TenneT → Close met "Won't fix - discontinued"
- [ ] Issues gerelateerd aan grid data → Close met "Won't fix - discontinued"
- [ ] Labels opschonen
- [ ] Milestones bijwerken

**Geschatte tijd:** 0.5 uur

---

### 4.4 Architecture Docs Updaten

**GitHub Issue:** `#74 - Update ARCHITECTURE.md`  
**Priority:** MEDIUM  
**Labels:** `documentation`, `fase-4`

**Te updaten:**
- [ ] Data flow diagram (zonder TenneT/grid)
- [ ] Fallback chain diagram toevoegen
- [ ] Component overzicht versimpelen

**Geschatte tijd:** 1 uur

---

## GITHUB ISSUES AANMAKEN

### Commando's voor CC

```bash
cd /opt/github/synctacles-api

# Labels aanmaken
gh label create "fallback" --color "0E8A16" --description "Fallback chain related"
gh label create "soft-delete" --color "FEF2C0" --description "Disabled but not removed"
gh label create "hard-delete" --color "D93F0B" --description "Code removal"
gh label create "fase-1" --color "C5DEF5" --description "Phase 1: Fallback Infrastructure"
gh label create "fase-2" --color "BFD4F2" --description "Phase 2: Soft Delete"
gh label create "fase-3" --color "D4C5F9" --description "Phase 3: Hard Delete"
gh label create "fase-4" --color "F9D0C4" --description "Phase 4: Documentation"

# Milestone aanmaken
gh api repos/{owner}/{repo}/milestones -f title="Energy Action Focus" -f description="Strip to Energy Action only with robust fallback chain" -f due_on="2026-02-01T00:00:00Z"

# FASE 1 Issues
gh issue create --title "#60 Add Energy-Charts as ENTSO-E fallback" \
  --body "See PvA section 1.1" \
  --label "critical,fallback,fase-1" \
  --milestone "Energy Action Focus"

gh issue create --title "#61 Implement price fallback chain" \
  --body "See PvA section 1.2" \
  --label "critical,fallback,fase-1" \
  --milestone "Energy Action Focus"

gh issue create --title "#62 Add 24h price cache for fallback" \
  --body "See PvA section 1.3" \
  --label "high,fallback,fase-1" \
  --milestone "Energy Action Focus"

gh issue create --title "#63 Add quality indicator to Energy Action" \
  --body "See PvA section 1.4" \
  --label "high,api,ha-component,fase-1" \
  --milestone "Energy Action Focus"

# FASE 2 Issues
gh issue create --title "#64 Disable grid/generation endpoints (410)" \
  --body "See PvA section 2.1" \
  --label "high,soft-delete,fase-2" \
  --milestone "Energy Action Focus"

gh issue create --title "#65 Reduce HA component to 6 entities" \
  --body "See PvA section 2.2" \
  --label "high,ha-component,soft-delete,fase-2" \
  --milestone "Energy Action Focus"

gh issue create --title "#66 Skip TenneT and A65/A75 processing" \
  --body "See PvA section 2.3" \
  --label "medium,soft-delete,fase-2" \
  --milestone "Energy Action Focus"

# FASE 3 Issues
gh issue create --title "#67 Remove TenneT integration code" \
  --body "See PvA section 3.1" \
  --label "medium,hard-delete,fase-3" \
  --milestone "Energy Action Focus"

gh issue create --title "#68 Remove A65/A75 (load/generation) code" \
  --body "See PvA section 3.2" \
  --label "medium,hard-delete,fase-3" \
  --milestone "Energy Action Focus"

gh issue create --title "#69 Remove grid stress calculation" \
  --body "See PvA section 3.3" \
  --label "low,hard-delete,fase-3" \
  --milestone "Energy Action Focus"

gh issue create --title "#70 Clean up unused dependencies" \
  --body "See PvA section 3.4" \
  --label "low,hard-delete,fase-3" \
  --milestone "Energy Action Focus"

# FASE 4 Issues
gh issue create --title "#71 Update SKILL documentation" \
  --body "See PvA section 4.1" \
  --label "high,documentation,fase-4" \
  --milestone "Energy Action Focus"

gh issue create --title "#72 Update API reference" \
  --body "See PvA section 4.2" \
  --label "high,documentation,fase-4" \
  --milestone "Energy Action Focus"

gh issue create --title "#73 Archive obsolete GitHub issues" \
  --body "See PvA section 4.3" \
  --label "low,housekeeping,fase-4" \
  --milestone "Energy Action Focus"

gh issue create --title "#74 Update ARCHITECTURE.md" \
  --body "See PvA section 4.4" \
  --label "medium,documentation,fase-4" \
  --milestone "Energy Action Focus"
```

---

## PRIORITEITEN OVERZICHT

| # | Issue | Prioriteit | Fase | Uren |
|---|-------|------------|------|------|
| 60 | Energy-Charts fallback | CRITICAL | 1 | 2 |
| 61 | Fallback chain | CRITICAL | 1 | 3 |
| 62 | Cache layer | HIGH | 1 | 2 |
| 63 | Quality indicator | HIGH | 1 | 2 |
| 64 | Endpoints → 410 | HIGH | 2 | 1 |
| 65 | HA → 6 entities | HIGH | 2 | 2 |
| 66 | Skip importers | MEDIUM | 2 | 0.5 |
| 67 | Remove TenneT | MEDIUM | 3 | 1.5 |
| 68 | Remove A65/A75 | MEDIUM | 3 | 2 |
| 69 | Remove grid stress | LOW | 3 | 1 |
| 70 | Clean dependencies | LOW | 3 | 1 |
| 71 | Update SKILLs | HIGH | 4 | 2 |
| 72 | Update API docs | HIGH | 4 | 1 |
| 73 | Archive issues | LOW | 4 | 0.5 |
| 74 | Update ARCHITECTURE | MEDIUM | 4 | 1 |
| | **TOTAAL** | | | **22.5** |

---

## EXIT CRITERIA PER FASE

### Fase 1 Complete
- [ ] Energy-Charts collector werkt
- [ ] Fallback chain test: disable Enever → automatic fallback
- [ ] Fallback chain test: disable ENTSO-E → Energy-Charts kicks in
- [ ] Cache bevat data
- [ ] Quality indicator zichtbaar in API response
- [ ] Alle unit tests groen

### Fase 2 Complete
- [ ] `/api/v1/generation` returns 410
- [ ] `/api/v1/load` returns 410
- [ ] `/api/v1/balance` returns 410
- [ ] `/api/v1/grid-stress` returns 410
- [ ] HA component toont exact 6 entities
- [ ] Geen errors in HA logs bij bestaande users
- [ ] 1 week stabiel gedraaid

### Fase 3 Complete
- [ ] Geen TenneT code meer in repo
- [ ] Geen A65/A75 code meer in repo
- [ ] `pip check` toont geen issues
- [ ] Alle tests groen
- [ ] Applicatie start zonder errors

### Fase 4 Complete
- [ ] Alle SKILL files actueel
- [ ] API docs actueel
- [ ] Obsolete GitHub issues gesloten
- [ ] ARCHITECTURE.md actueel
- [ ] README.md bijgewerkt

---

## ROLLBACK PLAN

**Als Fase 1 faalt:**
- Revert commits
- Oude price service blijft werken
- Geen impact op users

**Als Fase 2 faalt:**
- Re-enable endpoints (verwijder 410 handlers)
- Re-enable entities in HA component
- Geen data verlies (soft delete)

**Als Fase 3 faalt:**
- Git revert naar pre-Fase 3
- Code was al disabled, dus working state hersteld

---

## COMMUNICATIE

### Na Fase 2 (user-facing changes)

**GitHub Release Notes:**
```markdown
## v2.0.0 - Energy Action Focus

### Breaking Changes
- Removed: `sensor.generation_total`
- Removed: `sensor.load_actual`
- Removed: `sensor.balance_delta`
- Removed: `sensor.grid_stress`
- Removed: `sensor.price_level`
- Removed: `sensor.price_status`

### Why
SYNCTACLES now focuses exclusively on Energy Action - the one feature that saves you money.

### Migration
Remove any automations using the discontinued sensors.

### New
- Added: Quality indicator on Energy Action sensor
- Added: Confidence attribute (100 = live, 89 = estimated)
- Improved: Automatic fallback when data sources are unavailable
```

---

## START COMMANDO

```
Begin met Fase 1, Issue #60 (Energy-Charts collector).
Rapporteer na completion van elk issue.
Wacht met Fase 2 tot Fase 1 volledig getest is.
```

---

*Plan van Aanpak versie 1.0 - 2026-01-11*
