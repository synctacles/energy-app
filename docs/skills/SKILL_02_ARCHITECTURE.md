# SKILL 2 — SYSTEM ARCHITECTURE

Design Principles and 3-Layer Data Pipeline
Version: 3.0 (2026-01-11) - Energy Action Focus

> **Phase 3 Update:** A65 (load), A75 (generation), and TenneT integration have been
> discontinued. SYNCTACLES now focuses exclusively on Energy Action (price-based
> recommendations). See FASE_3_COMPLETION_REPORT.md for details.

---

## PURPOSE

Explain the architectural foundation of SYNCTACLES: how data flows from external sources through transformation layers, why the 3-layer design was chosen, and how configuration is managed for multi-tenant deployments.

---

## DESIGN PRINCIPLES

### 1. KISS (Keep It Simple, Stupid)

Every architectural decision prioritizes simplicity over cleverness.

**Manifestations:**
- Modular components with clear responsibilities
- Direct data flow (no unnecessary queues or caching)
- Standard PostgreSQL schema (no NoSQL complexity)
- Systemd for scheduling (industry standard, not custom scheduler)

**Example:** Why not use Kafka for data ingestion?
- Extra complexity
- Overkill for 15-minute data collection
- KISS: Direct HTTP collection → PostgreSQL INSERT

---

### 2. Fail-Fast Configuration

The system should refuse to run if misconfigured, never silently degrading.

**Manifestations:**
- Missing env var → ValueError at startup
- Invalid config → SystemExit before service runs
- No fallback defaults for critical values
- Clear error messages guide to solution

**Example:**
```python
# At startup (const.py)
BRAND_NAME = os.getenv("BRAND_NAME")
if not BRAND_NAME:
    raise ValueError(
        "BRAND_NAME required. Set in /opt/.env"
    )
# System won't start if misconfigured
```

---

### 3. Brand-Free Repository

Single codebase, multiple deployments. No tenant-specific code.

**Manifestations:**
- Templates with `{{PLACEHOLDER}}` values
- All config via environment variables
- No "Energy Insights NL" strings in code
- Multi-tenant isolation per instance

**Example:**
```
Same repository deployed on:
- Server A: BRAND_NAME="Energy Insights NL" (Dutch)
- Server B: BRAND_NAME="Energie Einblicke" (German)
- No code changes between deployments
```

---

### 4. Three-Layer Data Pipeline

Data transformation separated into distinct, testable layers.

**Manifestations:**
- Layer 1: Collectors (fetch raw data)
- Layer 2: Importers (parse into PostgreSQL)
- Layer 3: Normalizers (transform with quality metadata)
- Layer 4: API (serve via REST)

**Why this structure?**
- Each layer has single responsibility
- Easy to test each layer independently
- Easy to add new data sources (create new collector/importer/normalizer)
- Enables fallback at any layer

---

## SYSTEM ARCHITECTURE

### Component Overview

```
EXTERNAL SOURCES (Energy Action Focus - Phase 3)
├── ENTSO-E A44 (Day-Ahead Prices only)
├── Energy-Charts (Fallback prices)
├── Frank Energie API (Consumer prices)
└── Enever API (via Coefficient Proxy)
     │
     ▼
LAYER 1: COLLECTORS
└── entso_e_a44_prices.py      (hourly)
     │
     ▼ (saves to /var/log/{{BRAND}}/collectors/raw/*.xml)
LAYER 2: IMPORTERS
└── import_entso_e_a44.py  → raw_entso_e_a44
     │
     ▼ (PostgreSQL RAW tables)
LAYER 3: NORMALIZERS
├── normalize_entso_e_a44.py  → norm_entso_e_a44
└── normalize_prices.py       → price aggregation
     │
     ▼ (with quality metadata)
LAYER 4: API
├── FastAPI application
├── /v1/energy-action (GO/WAIT/AVOID)
├── /v1/prices/today
├── /v1/prices/tomorrow
└── /health (system status)
     │
     ▼
LAYER 5: CONSUMER PRICE ENGINE
├── Frank calibration (daily 15:05)
├── Enever validation (via proxy)
├── Dual-source verification
└── Coefficient fallback
     │
     ▼
HOME ASSISTANT (6 sensors)
├── ServerDataCoordinator (15-min) → Server API
├── EneverDataCoordinator (1hr) → Enever.nl API (BYO-key)
├── PriceCurrentSensor
├── CheapestHourSensor
├── ExpensiveHourSensor
├── EnergyActionSensor
├── PricesTodaySensor (Enever BYO-key)
└── PricesTomorrowSensor (Enever BYO-key)

DISCONTINUED (Phase 3):
├── A65 (load) collectors/importers/normalizers → archive/
├── A75 (generation) collectors/importers/normalizers → archive/
├── TenneT integration → BYO-key in HA only (not server)
├── /v1/generation-mix, /v1/load, /v1/signals, /v1/now → 410 Gone
└── Generation/Load/Balance/GridStress sensors → removed
```

---

## DATA PIPELINE (3-LAYER ARCHITECTURE)

### Layer 1: Collectors

**Purpose:** Fetch raw data from external sources

**Files:** `synctacles_db/collectors/*.py`

**Characteristics:**
- HTTP requests to external APIs
- Error handling and retries
- Saves raw XML/JSON to filesystem
- Rate-limited (respect API quotas)

**Example: ENTSO-E Collector**
```python
# entso_e_a75_generation.py
class EntsoECollector:
    def __init__(self, config):
        self.api_url = "https://web-api.tp.entsoe.eu/api"
        self.security_token = config['ENTSO_E_TOKEN']

    def collect(self):
        # Request generation mix (A75 document type)
        response = self._request_a75()
        # Save raw XML
        self._save_raw(response)
        # Let importer handle parsing
```

**Schedule:**
- Generation/Load: Every 15 minutes
- Prices: Hourly
- Balance: Every 5 minutes
- All via systemd timers

---

### Layer 2: Importers

**Purpose:** Parse raw data and insert into PostgreSQL RAW tables

**Files:** `synctacles_db/importers/*.py`

**Characteristics:**
- Reads files from Layer 1 output
- XML/JSON parsing
- Database INSERT into raw tables
- Minimal transformation (preserves raw data)
- Tracks source timestamp and import time

**Example: ENTSO-E A75 Importer**
```python
# import_entso_e_a75.py
class EntsoEA75Importer:
    def import_data(self, xml_file):
        # Parse XML
        tree = ET.parse(xml_file)
        # Extract PSR types (fossil fuels, renewables, nuclear, etc.)
        for psr_type, value in self._extract_values(tree):
            # Insert into raw_entso_e_a75
            db.insert({
                'psr_type': psr_type,
                'value_mw': value,
                'source_timestamp': tree.find('.//createdDateTime').text,
                'import_timestamp': datetime.now(),
                'file_source': xml_file
            })
```

**Raw Table Schema:**
```sql
CREATE TABLE raw_entso_e_a75 (
    id SERIAL PRIMARY KEY,
    psr_type VARCHAR(50),
    value_mw FLOAT,
    source_timestamp TIMESTAMP,
    import_timestamp TIMESTAMP,
    file_source VARCHAR(255)
);
```

---

### Layer 3: Normalizers

**Purpose:** Transform raw data into normalized, quality-tagged data

**Files:** `synctacles_db/normalizers/*.py`

**Characteristics:**
- Reads from Layer 2 RAW tables
- Applies transformations (unit conversions, aggregations, quality rules)
- Adds quality metadata (source reliability, age, completeness)
- Outputs to normalized tables
- Implements fallback strategy

**Example: Generation Normalizer**
```python
# normalize_entso_e_a75.py
class GenerationNormalizer:
    def normalize(self):
        # Read raw data
        raw = db.query(raw_entso_e_a75)

        # Group by PSR type, sum values
        for psr_type in PSR_TYPES:
            total_mw = sum(r.value_mw for r in raw if r.psr_type == psr_type)

            # Calculate quality score
            quality = self._calculate_quality(raw)
            age = self._calculate_age(raw)

            # Insert to normalized table WITH metadata
            db.insert('norm_generation', {
                'psr_type': psr_type,
                'generation_mw': total_mw,
                'source': 'entso_e_a75',
                'quality': quality,
                'age_minutes': age,
                'source_timestamp': max(r.source_timestamp for r in raw),
                'normalized_timestamp': datetime.now()
            })

            # If quality too low, try fallback
            if quality < 0.8:
                self._apply_fallback_strategy('generation')
```

**Normalized Table Schema:**
```sql
CREATE TABLE norm_generation (
    id SERIAL PRIMARY KEY,
    psr_type VARCHAR(50),
    generation_mw FLOAT,
    source VARCHAR(100),
    quality FLOAT,           -- 0.0 to 1.0
    age_minutes INT,
    source_timestamp TIMESTAMP,
    normalized_timestamp TIMESTAMP
);
```

**Quality Scoring:**
- 1.0 = Fresh from primary source, complete
- 0.8-0.99 = Slightly delayed or partial
- 0.5-0.79 = Stale or from fallback source
- <0.5 = Unreliable, use with caution

---

### Layer 4: API

**Purpose:** Serve normalized data via REST API

**Framework:** FastAPI (async, modern, well-documented)

**Endpoints:**

#### `/v1/generation/current`
```json
{
  "timestamp": "2025-12-30T10:15:00Z",
  "mix": {
    "nuclear": 3200,
    "solar": 450,
    "wind_onshore": 1200,
    "wind_offshore": 600,
    "fossil_fuels": 4500,
    "hydro": 200,
    "biomass": 150,
    "waste": 100,
    "other": 50
  },
  "total_mw": 10450,
  "quality": 0.98,
  "age_minutes": 2,
  "source": "entso_e_a75"
}
```

#### `/v1/load/current`
```json
{
  "timestamp": "2025-12-30T10:15:00Z",
  "load_mw": 12500,
  "forecast_mw": 12400,
  "quality": 0.95,
  "age_minutes": 5,
  "source": "entso_e_a65"
}
```

#### `/v1/prices/today`
```json
{
  "period": "2025-12-30T00:00:00Z to 2025-12-31T23:59:59Z",
  "data": [
    {"timestamp": "2025-12-30T00:00:00Z", "price_eur_mwh": 45.50},
    {"timestamp": "2025-12-30T01:00:00Z", "price_eur_mwh": 42.30},
    ...
  ],
  "meta": {
    "source": "ENTSO-E",
    "quality_status": "FRESH",
    "data_age_seconds": -112800,
    "count": 48,
    "allow_go_action": true
  }
}
```

#### `/health`
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "services": {
    "database": "connected",
    "entso_e": "ok",
    "energy_charts": "ok"
  }
}
```

---

### Layer 5: Consumer Price Engine

**Purpose:** Transform wholesale prices into consumer prices with dual-source validation

**Location:** Main API + Coefficient Server

**Architecture:**
```
                    Main API Server
                    ┌─────────────────────────────────┐
                    │                                 │
 ENTSO-E Wholesale ─┼──▶ Consumer Price Service      │
                    │         │                       │
                    │         ▼                       │
                    │    ┌─────────────┐              │
                    │    │ Frank API   │──────────────┼──▶ GraphQL (direct)
                    │    │ (primary)   │              │
                    │    └─────────────┘              │
                    │         │                       │
                    │         ▼                       │
                    │    ┌─────────────┐              │
                    │    │ Enever Proxy│──────────────┼──▶ Coefficient Server
                    │    │ (secondary) │              │         │
                    │    └─────────────┘              │         ▼
                    │         │                       │    VPN ──▶ Enever API
                    │         ▼                       │
                    │    ┌─────────────┐              │
                    │    │ Dual-Source │              │
                    │    │ Validation  │              │
                    │    └─────────────┘              │
                    │         │                       │
                    │         ▼                       │
                    │    Consumer Price               │
                    │    (€/kWh incl BTW)             │
                    └─────────────────────────────────┘
```

**Components:**

1. **Frank Calibration** (`services/frank_calibration.py`)
   - Daily update at 15:05
   - Fetches live Frank prices via GraphQL
   - Calculates correction factor

2. **Enever Client** (`services/enever_client.py`)
   - Proxied via coefficient server
   - 25 Dutch providers
   - Secondary validation source

3. **Consumer Price Service** (`services/consumer_price.py`)
   - Combines wholesale + coefficient
   - Dual-source validation
   - Fallback cascade

**Calculation:**
```python
consumer_price = wholesale + (HOURLY_LOOKUP[hour] × correction_factor)
```

**Validation Flow:**
```
Frank + Enever agree   → confidence: HIGH
Frank only             → confidence: MEDIUM
Enever only            → confidence: MEDIUM
Neither available      → confidence: LOW (coefficient fallback)
```

**Files:**
```
Main API:
├── config/coefficients.py        # 24-hour lookup table
├── services/frank_calibration.py # Frank API wrapper
├── services/enever_client.py     # Enever proxy client
└── services/consumer_price.py    # Price calculation

Coefficient Server:
├── routes/enever.py              # Proxy endpoint
├── services/enever_client.py     # Enever API calls
└── collectors/enever_collector.py # Historical data
```

**See:** SKILL_15_CONSUMER_PRICE_ENGINE.md for full documentation

---

## PRICE DATA ARCHITECTURE

### Two-Source Price System: ENTSO-E A44 + Energy-Charts Fallback

The system maintains **two independent price pipelines** for resilience:

#### Primary Source: ENTSO-E A44 (Official Day-Ahead Prices)
```
ENTSO-E API (Official EU Prices)
    ↓
entso_e_a44_prices.py (Collector)
    ↓ (saves raw XML every 15 min)
raw_entso_e_a44 (PostgreSQL)
    ↓
normalize_entso_e_a44.py (Normalizer)
    ↓
norm_entso_e_a44 (PostgreSQL - NORMALIZED)
    ↓
FastAPI /v1/prices (Primary source)
```

**Characteristics:**
- Official ENTSO-E day-ahead market prices
- Updated once daily at 13:00 CET (12:00 UTC)
- Covers today + tomorrow (48 hourly values)
- Quality: HIGH (official source)
- `allow_go_action=true` (safe for automation)

#### Fallback Source: Energy-Charts (Fraunhofer ISE Data)
```
Energy-Charts API (Estimated/Historical)
    ↓
energy_charts_prices.py (Collector)
    ↓
raw_prices (PostgreSQL)
    ↓
normalize_prices.py (Normalizer)
    ↓
norm_prices (PostgreSQL - NORMALIZED)
    ↓
FastAPI /v1/prices (Fallback if A44 unavailable)
```

**Characteristics:**
- Estimated day-ahead prices from Fraunhofer ISE
- Modeled data, not official
- Updated less frequently
- Quality: MEDIUM (fallback only)
- `allow_go_action=false` (not for automation)

### API Response Strategy

The API follows a **priority-based fallback chain:**

```python
# /v1/prices endpoint logic:

# 1. Try to get fresh ENTSO-E A44 data (primary)
prices = query(NormEntsoeA44)
if prices and is_fresh(prices):
    return {
        "data": prices,
        "meta": {
            "source": "ENTSO-E",
            "quality_status": "FRESH",
            "allow_go_action": True  # ← Safe for automation
        }
    }

# 2. Fallback to Energy-Charts (secondary)
prices = query(NormPrices)
if prices:
    return {
        "data": prices,
        "meta": {
            "source": "Energy-Charts",
            "quality_status": "FALLBACK",
            "allow_go_action": False  # ← Not for automation
        }
    }

# 3. No data available
return {"data": [], "meta": {"status": "UNAVAILABLE"}}
```

### Quality Scoring

| Source | Status | A44 Age | Allow Go | Notes |
|--------|--------|---------|----------|-------|
| ENTSO-E A44 | FRESH | < 24h | ✅ YES | Official, safe for automation |
| ENTSO-E A44 | STALE | 24-48h | ✅ YES | Yesterday's prices, still official |
| Energy-Charts | FALLBACK | > 48h | ❌ NO | Fallback only, estimated data |
| None | UNAVAILABLE | N/A | ❌ NO | No data from any source |

### Data Tables Reference

**Primary Pipeline:**
```sql
-- Raw ENTSO-E data
raw_entso_e_a44 (timestamp, country, price_eur_mwh, fetch_time)

-- Normalized ENTSO-E data (with quality metadata)
norm_entso_e_a44 (
    timestamp, country, price_eur_mwh,
    data_source='ENTSO-E',
    data_quality='OK',
    needs_backfill=false
)
```

**Fallback Pipeline:**
```sql
-- Raw Energy-Charts data
raw_prices (timestamp, price_eur_mwh, fetch_time)

-- Normalized Energy-Charts data
norm_prices (timestamp, price_eur_mwh, quality, age_minutes)
```

### Why Two Sources Matter

1. **Resilience:** If ENTSO-E unavailable, API still returns data (from Energy-Charts)
2. **Automation Safety:** `allow_go_action` flag ensures automation only uses official data
3. **Data Integrity:** No silent failures or stale data returned without metadata
4. **Architecture Learning:** Two sources demonstrate fallback pattern for other data types

**Note:** This is NOT an automatic merge. The API returns one source at a time, with clear quality metadata about which source and whether it's safe for automation.

---

## FALLBACK STRATEGY

When primary data sources fail, system automatically uses fallback cascade.

### Freshness Thresholds (per data source)

Based on real-world measurements and structural delays:

| Source | FRESH | STALE | Fallback Trigger | Notes |
|--------|-------|-------|------------------|-------|
| ENTSO-E | < 15 min | 15-60 min | > 60 min | Primary source |
| Energy-Charts | N/A | N/A | Tier 3 | Fallback only |
| PostgreSQL Cache | N/A | N/A | Tier 4b | 24h persistence |
| In-Memory Cache | N/A | N/A | Tier 4a | 5-min TTL |

**Note:** TenneT is no longer available via SYNCTACLES API (BYO-key in HA component only).

### 5-Tier Price Fallback Chain (Fase 1)

```
┌─────────────────────────────────────────────────────────────┐
│                    PRICE FALLBACK CHAIN                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Tier 1: ENTSO-E (Fresh)          ← < 15 min old           │
│     ↓ fail                           allow_go = TRUE        │
│     │                                quality = "live"       │
│     │                                confidence = 100%      │
│                                                             │
│  Tier 2: ENTSO-E (Stale)          ← 15-60 min old          │
│     ↓ fail                           allow_go = TRUE        │
│     │                                quality = "estimated"  │
│     │                                confidence = 85%       │
│                                                             │
│  Tier 3: Energy-Charts            ← Live API call          │
│     ↓ fail                           allow_go = FALSE       │
│     │                                quality = "estimated"  │
│     │                                confidence = 70%       │
│                                                             │
│  Tier 4a: In-Memory Cache         ← TTLCache (5 min)       │
│     ↓ fail                           allow_go = FALSE       │
│     │                                quality = "cached"     │
│     │                                confidence = 50%       │
│                                                             │
│  Tier 4b: PostgreSQL Cache        ← 24h persistence        │
│     ↓ fail                           allow_go = FALSE       │
│     │                                quality = "cached"     │
│     │                                confidence = 50%       │
│                                                             │
│  Tier 5: UNAVAILABLE              ← Return null            │
│                                      allow_go = FALSE       │
│                                      quality = "unavailable"│
│                                      confidence = 0%        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Critical Rule:** Energy-Charts and Cache tiers NEVER allow `allow_go_action = true`.
This prevents automated actions based on potentially inaccurate fallback data.

### Primary → Fallback Cascade (Legacy)

```
Generation Data:
  1st choice: ENTSO-E A75 (FRESH < 90min, STALE 90-180min)
  2nd choice: Energy-Charts (FRESH < 240min, hybrid merge for NULLs)
  3rd choice: Known Capacity (pragmatic estimates)
  4th choice: Cache (in-memory, 5-min TTL)
  5th choice: UNAVAILABLE

Load Data:
  1st choice: ENTSO-E A65 (FRESH < 90min, STALE 90-180min)
  2nd choice: Energy-Charts (use total_mw as load proxy)
  3rd choice: Cache
  4th choice: UNAVAILABLE

Prices:
  1st choice: ENTSO-E A44 (hourly)
  2nd choice: Energy-Charts (estimated)
  3rd choice: PostgreSQL Cache (24h)
  4th choice: In-Memory Cache (5 min)
```

### Quality Status Values

`quality_status` field returned in API responses:

- `FRESH` - Data within fresh threshold (authoritative, use normally)
- `STALE` - Data within stale threshold (slightly delayed, usable)
- `PARTIAL` - Hybrid merge (ENTSO-E + Energy-Charts NULL filling)
- `FALLBACK` - Using fallback source due to primary unavailable
- `CACHED` - Using in-memory cache (data age unknown)
- `UNAVAILABLE` - No data available from any source

### Fallback Features

**Hybrid Merge for Generation:**
- Always attempts Energy-Charts to fill ENTSO-E NULL values
- Tracks which fields come from which source (`_field_sources` metadata)
- Preserves data integrity by not inventing values

**Circuit Breaker:**
- Skips Energy-Charts for 2 hours after HTTP 404 error
- Prevents cascade failures and respects API limits
- Automatically retries after cooldown

**Data Transparency:**
- Every API response includes source and quality metadata
- Clients can decide whether to use based on freshness
- `_field_sources` shows which generator types are from which source

---

## BYO-KEY ARCHITECTURES

### TenneT BYO-Key Architecture (DISCONTINUED)

> **Phase 3 (2026-01-11):** TenneT integration has been discontinued from SYNCTACLES.
> Users requiring TenneT data should use the official TenneT API directly or alternative
> Home Assistant integrations.

**Historical Note:** TenneT provided real-time grid balance data (Dutch grid only).
The TennetDataCoordinator and associated sensors (balance_delta, grid_stress) have
been removed from the HA component.

---

### Enever.nl BYO-Key Architecture

**Purpose:** Leverancier-specific pricing (consumer prices, not wholesale)

**Flow:**
```
User's Enever Token
        ↓
HA Component (EneverDataCoordinator)
        ↓
https://api.enever.nl/
        ↓
Prices Today + Tomorrow sensors
```

**Coordinator Details:**
- **Update interval:** 1 hour
- **Smart caching:** Fetches tomorrow after 15:00, promotes at midnight
- **API reduction:** ~50% fewer calls than naive polling
- **Fallback:** ENTSO-E server prices if Enever unavailable

**Sensors Created (conditional):**
- `sensor.energy_insights_nl_prices_today` - Hourly prices today
- `sensor.energy_insights_nl_prices_tomorrow` - Hourly prices tomorrow

**Resolution Tiers:**
- Default: 60-min (24 points/day)
- Supporter + compatible supplier: 15-min (96 points/day)

**Smart Caching Strategy:**
```
Daily cycle:
00:00 - 14:59: Use cached "today" prices (fetched yesterday at 15:00)
15:00 - 15:59: Fetch tomorrow's prices (becomes today at midnight)
16:00 - 23:59: Use cached prices, no new fetches

API optimization:
- Traditional: ~62 calls/month (2/day for today+tomorrow)
- Smart caching: ~31 calls/month (1/day for tomorrow only)
- Reduction: 50%
```

**Leverancier Support:**
19 Dutch energy suppliers including Tibber, Zonneplan, Frank Energie, Greenchoice, Essent, Budget Energie, and 13 others.

---

## DATABASE SCHEMA

### Core Tables

```sql
-- Raw data tables (Layer 2 output)
CREATE TABLE raw_entso_e_a75 (
    id SERIAL PRIMARY KEY,
    psr_type VARCHAR(50),
    value_mw FLOAT,
    source_timestamp TIMESTAMP,
    import_timestamp TIMESTAMP,
    file_source VARCHAR(255)
);

-- Normalized data tables (Layer 3 output)
CREATE TABLE norm_generation (
    id SERIAL PRIMARY KEY,
    psr_type VARCHAR(50),
    generation_mw FLOAT,
    source VARCHAR(100),
    quality FLOAT,
    age_minutes INT,
    source_timestamp TIMESTAMP,
    normalized_timestamp TIMESTAMP
);

-- API should always query normalized tables
```

### Consumer Price Tables (Coefficient Server)

```sql
-- Historical consumer prices from Enever
-- Located on Coefficient Server (91.99.150.36)
CREATE TABLE enever_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    provider VARCHAR(50) NOT NULL,
    hour INTEGER NOT NULL CHECK (hour >= 0 AND hour <= 23),
    price_total DECIMAL(8,5) NOT NULL,
    price_energy DECIMAL(8,5),
    price_tax DECIMAL(8,5),
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(timestamp, provider, hour)
);

-- 25 providers × 24 hours × 2 collections/day = ~1200 records/day
-- Retention: Indefinite (B2B data asset)
```

### Price Cache Table (Fase 1)

```sql
-- 24h rolling price cache for Tier 4b fallback
-- Located on Main API Server (135.181.255.83)
CREATE TABLE price_cache (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    country VARCHAR(2) NOT NULL DEFAULT 'NL',
    price_eur_kwh NUMERIC(10, 6) NOT NULL,
    source VARCHAR(50) NOT NULL,      -- entsoe, energy-charts, enever
    quality VARCHAR(20) NOT NULL,      -- live, estimated, cached
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_price_cache_timestamp ON price_cache(timestamp);
CREATE INDEX idx_price_cache_country_timestamp ON price_cache(country, timestamp DESC);

-- Purpose: Persistent fallback when all live sources fail
-- Retention: Rolling 24h window (automatic cleanup)
-- Usage: Tier 4b in fallback chain
```

**Important:** Every normalized table includes:
- `source` - Where the data came from
- `quality` - Reliability score (0.0-1.0)
- `age_minutes` - How old is the data
- `source_timestamp` - When source recorded it
- `normalized_timestamp` - When we normalized it

---

## MULTI-TENANT DEPLOYMENT

### Complete Isolation Per Tenant

Each deployment gets its own:
- Database ({{DB_NAME}})
- Service user ({{SERVICE_USER}})
- Installation directory ({{INSTALL_PATH}})
- Systemd units (named with {{BRAND_SLUG}})
- Virtual environment (in {{INSTALL_PATH}}/venv)
- Log directory (/var/log/{{BRAND_SLUG}})

### Example: Two Tenants

**Tenant A (Energy Insights NL):**
```
.env: BRAND_SLUG=energy-insights-nl
DB: synctacles_energy_insights_nl
Service: energy-insights-nl-api
User: energy-insights-nl
Path: /opt/energy-insights-nl
```

**Tenant B (Energie Einblicke DE):**
```
.env: BRAND_SLUG=energie-einblicke-de
DB: synctacles_energie_einblicke_de
Service: energie-einblicke-de-api
User: energie-einblicke-de
Path: /opt/energie-einblicke-de
```

**Same repository, completely isolated instances.**

---

## ENV-DRIVEN CONFIGURATION

### Critical Configuration Variables

```bash
# Brand/Tenant
BRAND_NAME="Energy Insights NL"
BRAND_SLUG="energy-insights-nl"
BRAND_DOMAIN="energy-insights.nl"
HA_DOMAIN="energy_insights_nl"

# Database
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="synctacles_energy_insights_nl"
DB_USER="energy_insights_nl"
DB_PASSWORD="secret"

# Paths
INSTALL_PATH="/opt/energy-insights-nl"
APP_PATH="/opt/energy-insights-nl/app"
LOG_PATH="/var/log/energy-insights-nl"

# Service
SERVICE_USER="energy-insights-nl"
SERVICE_GROUP="energy-insights-nl"

# API
API_HOST="0.0.0.0"
API_PORT="8000"
API_KEY="secret"

# External APIs
ENTSO_E_TOKEN="secret"
TENNET_API_KEY="secret"
```

### Fail-Fast Pattern

**At startup (main.py):**
```python
import os

# Validate all required vars before anything else
required_vars = [
    'BRAND_NAME', 'DB_HOST', 'DB_NAME', 'API_KEY'
]

missing = [v for v in required_vars if not os.getenv(v)]
if missing:
    raise ValueError(
        f"Missing required environment variables: {missing}\n"
        f"Run FASE 0 setup or check /opt/.env"
    )

# System doesn't start if misconfigured
```

---

## SCHEDULED TASKS (Systemd Timers)

Each layer has systemd timers:

```
energy-insights-nl-collect.timer      → Runs entso_e collector every 15 min
energy-insights-nl-import.timer       → Runs importers every 5 min
energy-insights-nl-normalize.timer    → Runs normalizers every 5 min
energy-insights-nl-api.service        → Runs API continuously
```

Why systemd timers?
- KISS principle: standard tool, everyone knows it
- Logging: journalctl shows everything
- Monitoring: systemctl list-timers shows status
- Restart policy: automatic recovery

---

## OBSERVABILITY

### Logging

Each layer writes to:
```
/var/log/{{BRAND_SLUG}}/collectors/
/var/log/{{BRAND_SLUG}}/importers/
/var/log/{{BRAND_SLUG}}/normalizers/
/var/log/{{BRAND_SLUG}}/api/
```

Accessible via:
```bash
journalctl -u energy-insights-nl-api -n 100 -f
journalctl -u energy-insights-nl-collect.timer --all
```

### Health Endpoint

`GET /health` always returns:
```json
{
  "status": "healthy|degraded|critical",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "services": {
    "database": "connected|error",
    "entso_e": "ok|slow|error",
    "tennet": "ok|slow|error",
    "energy_charts": "ok|slow|error"
  },
  "last_collection": "2025-12-30T10:15:00Z",
  "last_normalization": "2025-12-30T10:20:00Z"
}
```

---

## DEPLOYMENT MODEL

### Installation (FASE 0-6)

```
FASE 0: Interactive brand configuration
  └─ Generates /opt/.env with all variables

FASE 1-6: Standard installation
  ├─ FASE 1: System updates
  ├─ FASE 2: Software stack (PostgreSQL, Python)
  ├─ FASE 3: Security (firewall, permissions)
  ├─ FASE 4: Python environment (venv, dependencies)
  ├─ FASE 5: Production services (systemd units)
  └─ FASE 6: Development tools (optional)
```

### Deployment (DEV → PROD)

See SKILL 10: Deployment Workflow for 6-phase process:
1. Pre-deploy validation
2. Backup current production
3. Sync files
4. Post-sync actions
5. Validation
6. Rollback (if needed)

---

## SECURITY MODEL

### Data Protection

- Database credentials in .env (never git)
- API keys in .env (never git)
- .gitignore prevents accidental commits
- Systemd EnvironmentFile protects .env permissions

### Service Isolation

- Each tenant runs as separate system user
- Each tenant has separate database
- No cross-tenant data leakage possible
- Logs per tenant (separate directories)

### HTTPS/TLS

- Reverse proxy (nginx) handles HTTPS
- API runs on localhost:8000
- Only nginx communicates with API
- Certificates managed externally

---

## RELATED SKILLS

- **SKILL 1**: Hard Rules (principles enforced here)
- **SKILL 3**: Coding Standards (implementation details)
- **SKILL 9**: Installer Specs (how to deploy)
- **SKILL 10**: Deployment Workflow (how to update)
- **SKILL 12**: Brand-Free Architecture (why templates)

---

## ROADMAP (F7-F9)

### F7: Advanced Forecasting
- Machine learning model for generation forecast
- Improved price prediction
- Anomaly detection

### F8: Enhanced Automation
- Price-triggered automation signals
- Demand response integration
- Battery scheduling

### F9: Data Marketplace
- Sell processed energy data
- API for third-party integrations
- Data quality guarantees

---

## DIAGRAM SUMMARY

```
External APIs
     │
     ▼
COLLECTORS (fetch raw)
     │
     ▼
IMPORTERS (parse → RAW tables)
     │
     ▼
NORMALIZERS (transform + metadata)
     │
     ▼
API (serve + quality scores)
     │
     ▼
Home Assistant / Client Apps
```

**Each layer:**
- Has single responsibility
- Is independently testable
- Can be easily extended
- Includes error handling
- Tracks data quality
