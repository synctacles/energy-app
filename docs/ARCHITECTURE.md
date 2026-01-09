# Energy Insights NL - Architecture & Design Guide

**Version:** 1.0
**Date:** 2025-12-30
**Status:** Production Ready

---

## Executive Summary

Energy Insights NL is a Dutch energy data aggregation platform that collects real-time energy generation, load, and price data from ENTSO-E, normalizes it, and provides a REST API for Home Assistant integration. The system features automatic fallback to Energy-Charts when primary sources are unavailable. Grid balance data is available via BYO-key (Bring Your Own) in the Home Assistant component only, complying with TenneT API license restrictions.

**Key Capabilities:**
- Real-time Dutch generation mix (9 PSR types) updated every 15 minutes
- Grid load forecasts with actual values
- Day-ahead electricity prices
- Grid balance data via BYO-key in HA component (TenneT license restriction)
- Automatic failover to Energy-Charts (Fraunhofer ISE model)
- Home Assistant native integration via custom component
- Production-ready with monitoring, backups, and multi-tenant support

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Data Pipeline (3-Layer Architecture)](#data-pipeline-3-layer-architecture)
3. [Fallback Strategy](#fallback-strategy)
4. [Database Schema](#database-schema)
5. [API Specification](#api-specification)
6. [Architecture Decision Records (ADRs)](#architecture-decision-records-adrs)
7. [Deployment Model](#deployment-model)
8. [Security Model](#security-model)
9. [Monitoring & Operations](#monitoring--operations)
10. [Roadmap (F7-F9)](#roadmap-f7-f9)

---

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        EXTERNAL SOURCES                          │
├─────────────────┬───────────────────────────────────────────────┤
│    ENTSO-E      │      Energy-Charts      │    TenneT (BYO-key) │
│  (A75/A65/A44)  │       (Fallback)        │   (Client-side HA)  │
└────────┬────────┴─────────────┬───────────┴────────┬────────────┘
         │                      │                   │
         ▼                      ▼             (NOT fetched by server)
         │                      │                   │
         │                      │         ╔════════════════════╗
         │                      │         ║ Home Assistant     ║
         │                      │         ║ - User provides    ║
         │                      │         ║   TenneT key       ║
         │                      │         ║ - Fetches locally  ║
         │                      │         ║ - Creates sensors  ║
         │                      │         ║   (balance_delta)  ║
         │                      │         ╚════════════════════╝
┌─────────────────────────────────────────────────────────────────┐
│                   LAYER 1: COLLECTORS                            │
│  synctacles_db/collectors/                                       │
│  ├── entso_e_a75_generation.py   (15-min interval)              │
│  ├── entso_e_a65_load.py         (15-min interval)              │
│  ├── entso_e_a44_prices.py       (hourly)                       │
│  └── energy_charts_client.py     (fallback, cached)             │
│                                                                  │
│  Output: /var/log/{brand}/collectors/raw/*.xml                  │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                   LAYER 2: IMPORTERS                             │
│  synctacles_db/importers/                                        │
│  ├── import_entso_e_a75.py   → raw_entso_e_a75 table            │
│  ├── import_entso_e_a65.py   → raw_entso_e_a65 table            │
│  └── import_entso_e_a44.py   → raw_entso_e_a44 table            │
│                                                                  │
│  Parse XML/JSON → Insert to PostgreSQL RAW tables                │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                  LAYER 3: NORMALIZERS                            │
│  synctacles_db/normalizers/                                      │
│  ├── normalize_entso_e_a75.py  → norm_generation table          │
│  ├── normalize_entso_e_a65.py  → norm_load table                │
│  └── normalize_entso_e_a44.py  → norm_prices table              │
│                                                                  │
│  RAW tables → Normalized tables with quality metadata            │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                     LAYER 4: API                                 │
│  synctacles_db/api/                                              │
│  ├── main.py                 (FastAPI app)                       │
│  ├── routes/                 (endpoint definitions)              │
│  └── schemas/                (Pydantic models)                   │
│                                                                  │
│  Endpoints:                                                      │
│  ├── /v1/generation/current  → Current generation mix           │
│  ├── /v1/load/current        → Current grid load                │
│  ├── /v1/prices/today        → Today's electricity prices       │
│  ├── /v1/balance/current     → 501 Not Implemented (archived)    │
│  ├── /v1/signals/*           → Automation signals               │
│  └── /health                 → System health status             │
└─────────────────────────────────────────────────────────────────┘
         │                                              │
         │                                              ▼
         │                                      Home Assistant
         │                                   (custom component)
         │                                   ├─ Gen/Load/Prices
         │                                   ├─ Signals
         │                                   └─ Balance (BYO-key)
         │
         ▼
    Client Applications
```

### Data Flow

```
ENTSO-E API              PostgreSQL              REST API
    │                        │                       │
    │  XML Response          │                       │
    ▼                        │                       │
┌─────────┐                  │                       │
│Collector│ ─── saves ──────► /var/log/.../raw/     │
└─────────┘                  │                       │
    │                        │                       │
    │  trigger               │                       │
    ▼                        │                       │
┌─────────┐                  │                       │
│Importer │ ─── parses ─────► raw_entso_e_* tables  │
└─────────┘                  │                       │
    │                        │                       │
    │  trigger               │                       │
    ▼                        │                       │
┌──────────┐                 │                       │
│Normalizer│ ─── transform ─► norm_* tables          │
│  (with   │                 │                       │
│ fallback)│                 │  query                │
└──────────┘                 ▼                       │
    │                   ┌─────────┐                  │
    │                   │   API   │ ◄─── HTTP ──────┤
    │                   └─────────┘                  │
    │                        │                       │
    │                        │  JSON response        │
    │                        ▼                       │
    │                  Home Assistant               │
```

---

## Data Pipeline (3-Layer Architecture)

### Design Rationale

The 3-layer architecture (Collectors → Importers → Normalizers → API) provides:

1. **Separation of Concerns** - Each layer has a single responsibility
2. **Fault Isolation** - One layer failing doesn't cascade to others
3. **Debuggability** - Can inspect raw XML files to diagnose import issues
4. **Replayability** - Can re-run any layer without external API calls
5. **Audit Trail** - Raw data stored as backup before transformation

### Layer 1: Collectors

**Purpose:** Fetch data from external APIs, validate response, save raw files

**Components:**
- `entso_e_a75_generation.py` - Generation mix (9 PSR types), 15-min interval
- `entso_e_a65_load.py` - Load actual + forecast, 15-min interval
- `entso_e_a44_prices.py` - Electricity prices, hourly interval
- ~~`tennet_ingestor.py`~~ - **ARCHIVED** (BYO-key model)
- `energy_charts_client.py` - Fallback renewable data (cached, not polled)

**Note on TenneT:** Server no longer fetches TenneT data. Users configure their own TenneT API key in Home Assistant component for local processing (see ADR-008)

**Process:**
1. Load API credentials from environment
2. Call external API with retries
3. Validate XML/JSON response (check for errors)
4. Save raw file to disk: `/var/log/{brand}/collectors/raw/{source}_{timestamp}.xml`
5. Log success/failure to journalctl
6. Trigger importer (systemd service)

**Error Handling:**
- Network timeouts: Retry 3x with exponential backoff
- Invalid credentials: Fail fast with clear error message
- Malformed response: Save to disk for manual inspection
- Rate limits: Back off and retry after delay

### Layer 2: Importers

**Purpose:** Parse raw files, validate data, insert to PostgreSQL RAW tables

**Components:**
- `import_entso_e_a75.py` - Parse ENTSO-E A75 XML → `raw_entso_e_a75`
- `import_entso_e_a65.py` - Parse ENTSO-E A65 XML → `raw_entso_e_a65`
- `import_entso_e_a44.py` - Parse ENTSO-E A44 XML → `raw_entso_e_a44`
- `import_tennet_balance.py` - Parse TenneT JSON → `raw_tennet_balance`

**Process:**
1. Read latest raw file from disk
2. Parse XML/JSON format
3. Extract timestamped data points
4. Insert to RAW table (one row = one timestamp + one metric)
5. Update import_status in database
6. Log import metrics (rows inserted, duplicates, errors)
7. Trigger normalizer (systemd service)

**Error Handling:**
- Duplicate keys: Skip silently (expected if importer runs twice)
- Missing required fields: Log warning, skip record
- Data type mismatches: Convert to float safely or skip
- Constraint violations: Report and investigate

### Layer 3: Normalizers

**Purpose:** Transform RAW data into normalized tables with quality metadata and fallback handling

**Components:**
- `normalize_entso_e_a75.py` - Generation mix: pivot 9 PSR types → columns, add quality metadata
- `normalize_entso_e_a65.py` - Load: merge actual + forecast into single row, add quality
- `normalize_entso_e_a44.py` - Prices: denormalize and add quality metadata
- `normalize_prices.py` - Price post-processing and validation
- `normalize_tennet_balance.py` - Balance: aggregate 5-min data, add quality metadata

**Process:**
1. Query RAW table for new/updated data
2. Filter forecast data (`WHERE timestamp <= NOW()`) to exclude future timestamps
3. Pivot/aggregate/enrich data
4. **Apply Fallback Logic** (see Fallback Strategy section)
5. Add quality metadata (source, status, age_seconds)
6. Insert/update to Normalized table
7. Trigger API cache invalidation

**⚠️ CRITICAL: Forecast Data Filtering**

ENTSO-E includes forecast data (future timestamps) in API responses. ALL normalizers MUST filter these out:

```python
# WRONG - Includes future timestamps
SELECT * FROM raw_entso_e_a65;

# CORRECT - Historical data only
SELECT * FROM raw_entso_e_a65 WHERE timestamp <= NOW();
```

**Affected Sources:**
- **A65 (Load):** 24-hour forecast (~88 future records)
- **A44 (Prices):** Day-ahead prices (tomorrow's data)
- **A75 (Generation):** Minimal forecast data

**Consequence of not filtering:** Negative age values in health checks, incorrect data freshness calculations

**Quality Metadata Added:**
- `source` - Where data came from (ENTSO-E, Energy-Charts, FORWARD_FILL, CACHED)
- `status` - Data quality (FRESH, STALE, FALLBACK, UNAVAILABLE)
- `age_seconds` - How old the data is
- `confidence_score` - 0-100 quality indicator
- `needs_backfill` - Flag if ENTSO-E data is missing and needs retry

### Layer 4: API

**Purpose:** Serve normalized data via REST API with automatic fallback and caching

**Components:**
- `main.py` - FastAPI application setup
- `routes/*.py` - Endpoint definitions
- `schemas/*.py` - Pydantic models for validation
- `fallback_manager.py` - Fallback decision logic

**Endpoints:**
- `GET /health` - System health status
- `GET /v1/generation/current` - Current generation mix with quality
- `GET /v1/load/current` - Current grid load + forecast
- `GET /v1/prices/today` - Today's electricity prices
- `GET /v1/balance/current` - Current grid balance
- `GET /v1/signals/*` - Automation signals (is_green, should_charge, etc.)

**Caching Strategy:**
- In-memory cache (TTL varies by endpoint)
- `/generation` and `/load`: 5-min TTL (respects 15-min collection interval)
- `/prices`: 60-min TTL (hourly update)
- `/balance`: 1-min TTL (5-min collection)

---

## Fallback Strategy

### Overview

When primary data sources (ENTSO-E) are unavailable or stale, the system automatically falls back to Energy-Charts (Fraunhofer ISE model) to maintain service availability.

**Success Rate:**
- ENTSO-E alone: ~95% uptime
- ENTSO-E + Energy-Charts: ~99.9% uptime

### Fallback Hierarchy (4 Tiers)

#### Tier 1: Fresh Database Data (Primary)
```
Age < 30 min (generation) / 15 min (load)
Quality: FRESH
Use immediately, optimal data quality
```

#### Tier 2: Stale Database Data (Acceptable)
```
Age 30-150 min (generation) / 15-60 min (load)
Quality: STALE
Still usable (ENTSO-E A75 has 60-90 min structural delay)
```

#### Tier 3: Energy-Charts Fallback
```
Age > 150 min (generation) OR missing
Quality: FALLBACK
Fetch from Energy-Charts API, cache for 5 minutes
Includes solar validation (context-aware timing)
```

#### Tier 4: Forward Fill + Validation
```
All primary sources fail
Quality: FORWARD_FILL or VALIDATED
Use previous known value (conservative, realistic)
```

#### Tier 5: Cache (Last Resort)
```
All sources fail and cache expired
Quality: CACHED
Use cached fallback data (<5 min old)
```

#### Tier 6: Complete Failure
```
No data available
Quality: UNAVAILABLE
Return null with safe defaults for signals
```

### Realistic Data Validation

Before using Energy-Charts data, validate it's realistic:

**Minimum Values by PSR Type:**
```python
MIN_VALUES = {
    'biomass': 200,      # NL capacity ~500 MW
    'nuclear': 450,      # Borssele reactor = 485 MW
    'gas': 1000,         # Gas plants never turn off completely
    'coal': 0,           # Can be zero if shutdown
    'wind_offshore': 0,  # Can be zero if windstill
    'wind_onshore': 0,
    'solar': 0,          # Variable but validated separately
    'other': 0
}
```

**Solar Context-Aware Validation:**
```
Winter (Dec-Feb):   Sunrise 07:00, Sunset 15:00 UTC
Summer (May-Aug):   Sunrise 03:00, Sunset 20:00 UTC

Rules:
- Night (outside +/- 1h): solar=0 is OK
- Dawn/Dusk: solar=0 is suspicious but acceptable
- Daytime: solar=0 is highly suspicious (reject)
- Summer max: ~4000 MW
- Winter max: ~2000 MW
```

**Deviation Check:**
```
If Energy-Charts differs from forward-fill by >150%:
  Reject and use forward-fill instead
```

### API Metadata

All responses include `metadata` object indicating data source and quality:

```json
{
  "timestamp": "2025-12-30T14:30:00Z",
  "data": {
    "biomass_mw": 375.0,
    "wind_onshore_mw": 2150.0,
    "solar_mw": 0.0,
    ...
  },
  "metadata": {
    "source": "ENTSO-E",           // or "Energy-Charts", "FORWARD_FILL", "CACHED"
    "quality": "STALE",             // FRESH, STALE, FALLBACK, UNAVAILABLE
    "age_seconds": 2145,            // How old is this data
    "confidence_score": 92,         // 0-100 quality indicator
    "renewable_percentage": 68.5
  }
}
```

### Backfill Process (Daily 04:00 UTC)

When ENTSO-E data is missing for a timestamp, the system marks it for backfill and retries after 24 hours:

1. **Real-time:** Data missing → use Energy-Charts + mark `needs_backfill=TRUE`
2. **Daily (04:00 UTC):** Retry ENTSO-E API for all gaps in last 7 days
3. **Success:** Replace with ENTSO-E data, set `quality=BACKFILLED`
4. **Failure after 7 days:** Accept gap, set `needs_backfill=FALSE`

**Rationale:**
- ENTSO-E sometimes publishes data late (up to 24 hours)
- Backfill process recovers data automatically
- 7-day limit prevents accumulation of stale flags

### Monitoring Fallback Status

Check which sources are being used:

```bash
# Check last 24 hours
psql -d energy_insights_nl -c "
SELECT data_source, data_quality, COUNT(*)
FROM norm_generation
WHERE timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY data_source, data_quality
ORDER BY data_source;"
```

**Healthy pattern:**
```
ENTSO-E  | FRESH       | ~96 rows (most recent 24h)
ENTSO-E  | STALE       | ~1440 rows (older data)
ENTSO-E  | BACKFILLED  | ~0-5 rows (rare)
Energy-Charts | VALIDATED | ~0-10 rows (rare, only when ENTSO-E >150 min old)
```

**Unhealthy pattern:**
```
Energy-Charts | VALIDATED | >100 rows (ENTSO-E down too long)
```

---

## Database Schema

### RAW Tables (Layer 2 Output)

These tables store minimal transformation, directly from source APIs.

#### `raw_entso_e_a75` - Generation Mix (9 PSR Types)
```sql
CREATE TABLE raw_entso_e_a75 (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    psr_type VARCHAR(50) NOT NULL,  -- biomass, wind_onshore, solar, etc.
    value_mw FLOAT NOT NULL,        -- Megawatts
    source_file VARCHAR(255),        -- e.g., a75_2025-12-30_1430.xml
    inserted_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (timestamp, psr_type)
);
```

#### `raw_entso_e_a65` - Load (Actual + Forecast)
```sql
CREATE TABLE raw_entso_e_a65 (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    load_actual_mw FLOAT,
    load_forecast_mw FLOAT,
    source_file VARCHAR(255),
    inserted_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (timestamp)
);
```

#### `raw_entso_e_a44` - Prices
```sql
CREATE TABLE raw_entso_e_a44 (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    price_eur_per_mwh FLOAT NOT NULL,
    source_file VARCHAR(255),
    inserted_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (timestamp)
);
```

#### `archive_raw_tennet_balance` - Balance Delta (DEPRECATED)

**⚠️ DEPRECATED:** This table has been archived and renamed to `archive_raw_tennet_balance`. TenneT data is no longer collected server-side.

**Migration:** TenneT balance data is now available via BYO-key (Bring Your Own Key) in the Home Assistant integration. See [ADR-001: TenneT BYO-Key Model](decisions/ADR_001_TENNET_BYO_KEY.md).

```sql
CREATE TABLE archive_raw_tennet_balance (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    platform VARCHAR(20) NOT NULL,   -- aFRR, IGCC, MARI, mFRRda, PICASSO
    delta_mw FLOAT NOT NULL,         -- Positive = surplus, Negative = deficit
    price_eur_mwh FLOAT,             -- Price if imbalance persists
    source_file VARCHAR(255),
    imported_at TIMESTAMP DEFAULT NOW()
);
```

**Status:** Historical data preserved, collectors/importers/normalizers moved to `archive/` directories.

### Normalized Tables (Layer 3 Output)

These tables contain enriched, query-optimized data with quality metadata.

#### `norm_generation` - Generation Mix with Quality Metadata
```sql
CREATE TABLE norm_generation (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    biomass_mw FLOAT DEFAULT 0,
    wind_onshore_mw FLOAT DEFAULT 0,
    wind_offshore_mw FLOAT DEFAULT 0,
    solar_mw FLOAT DEFAULT 0,
    nuclear_mw FLOAT DEFAULT 0,
    gas_mw FLOAT DEFAULT 0,
    coal_mw FLOAT DEFAULT 0,
    waste_mw FLOAT DEFAULT 0,
    other_mw FLOAT DEFAULT 0,
    total_mw FLOAT DEFAULT 0,

    -- Quality Metadata
    data_source VARCHAR(50),         -- ENTSO-E, Energy-Charts, FORWARD_FILL
    data_quality VARCHAR(20),        -- FRESH, STALE, FALLBACK, UNAVAILABLE
    age_seconds INT,                 -- How old is this data
    confidence_score INT DEFAULT 100, -- 0-100
    renewable_percentage FLOAT,      -- % renewable energy

    -- Backfill Tracking
    needs_backfill BOOLEAN DEFAULT FALSE,

    inserted_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (timestamp)
);
```

#### `norm_load` - Grid Load with Quality Metadata
```sql
CREATE TABLE norm_load (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    load_actual_mw FLOAT,
    load_forecast_mw FLOAT,
    load_difference_mw FLOAT,        -- actual - forecast

    -- Quality Metadata
    data_source VARCHAR(50),
    data_quality VARCHAR(20),
    age_seconds INT,
    confidence_score INT DEFAULT 100,

    inserted_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (timestamp)
);
```

#### `norm_grid_balance` - Balance with Quality Metadata (DEPRECATED)

**⚠️ DEPRECATED:** This table is no longer actively populated. TenneT data collection has been discontinued.

**Migration:** Use BYO-key model in Home Assistant integration for real-time TenneT balance data. See [ADR-001](decisions/ADR_001_TENNET_BYO_KEY.md).

```sql
CREATE TABLE norm_grid_balance (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    balance_mw FLOAT,                -- Positive = surplus, Negative = deficit
    imbalance_price_eur FLOAT,

    -- Quality Metadata
    data_source VARCHAR(50),
    data_quality VARCHAR(20),
    age_seconds INT,

    inserted_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (timestamp)
);
```

**Status:** Table schema preserved for historical data access. No longer updated by normalizers.

### Indexes

```sql
-- For time-series queries (most common)
CREATE INDEX idx_norm_generation_timestamp ON norm_generation(timestamp DESC);
CREATE INDEX idx_norm_load_timestamp ON norm_load(timestamp DESC);
CREATE INDEX idx_norm_grid_balance_timestamp ON norm_grid_balance(timestamp DESC);  -- DEPRECATED (TenneT)

-- For quality/source queries
CREATE INDEX idx_norm_generation_quality ON norm_generation(data_quality);
CREATE INDEX idx_norm_generation_source ON norm_generation(data_source);

-- For dashboard/aggregations
CREATE INDEX idx_norm_generation_timestamp_quality
ON norm_generation(timestamp DESC, data_quality);
```

**Note:** Indexes on `norm_grid_balance` are preserved for historical data queries but no longer actively used in production pipelines.

---

## API Specification

### Authentication

**Current (Development):** None (localhost only)
**Production:** API Key header required

```bash
curl -H "X-API-Key: YOUR_API_KEY" https://api.example.com/v1/generation/current
```

### Base URL

**Development:** `http://localhost:8000`
**Production:** `https://api.example.com`

### Endpoints

#### Health Check

```
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-12-30T14:30:00Z",
  "components": {
    "database": "ok",
    "collectors": {
      "entso_e_a75": {"status": "ok", "last_run": "2025-12-30T14:30:00Z"},
      "entso_e_a65": {"status": "ok", "last_run": "2025-12-30T14:30:00Z"},
      "tennet": {"status": "ok", "last_run": "2025-12-30T14:25:00Z"}
    }
  }
}
```

#### Generation Mix (Current)

```
GET /v1/generation/current
```

**Response:**
```json
{
  "timestamp": "2025-12-30T14:30:00Z",
  "data": {
    "biomass_mw": 375.0,
    "wind_onshore_mw": 2150.5,
    "wind_offshore_mw": 680.0,
    "solar_mw": 0.0,
    "nuclear_mw": 485.0,
    "gas_mw": 1850.0,
    "coal_mw": 400.0,
    "waste_mw": 150.0,
    "other_mw": 280.5,
    "total_mw": 6370.0
  },
  "metadata": {
    "source": "ENTSO-E",
    "quality": "STALE",
    "age_seconds": 2145,
    "confidence_score": 92,
    "renewable_percentage": 42.3
  }
}
```

#### Load (Current)

```
GET /v1/load/current
```

**Response:**
```json
{
  "timestamp": "2025-12-30T14:30:00Z",
  "data": {
    "load_actual_mw": 5200.0,
    "load_forecast_mw": 5100.0,
    "load_difference_mw": 100.0
  },
  "metadata": {
    "source": "ENTSO-E",
    "quality": "STALE",
    "age_seconds": 1800,
    "confidence_score": 95
  }
}
```

#### Prices (Today)

```
GET /v1/prices/today
```

**Response:**
```json
{
  "date": "2025-12-30",
  "prices": [
    {"hour": 0, "price_eur_per_mwh": 45.50},
    {"hour": 1, "price_eur_per_mwh": 42.00},
    ...
    {"hour": 23, "price_eur_per_mwh": 55.25}
  ],
  "metadata": {
    "source": "ENTSO-E",
    "quality": "FRESH",
    "age_seconds": 300
  }
}
```

#### Balance (Current)

```
GET /v1/balance/current
```

**Response:**
```json
{
  "timestamp": "2025-12-30T14:30:00Z",
  "data": {
    "balance_mw": 125.0,
    "imbalance_price_eur": 2.50
  },
  "metadata": {
    "source": "TenneT",
    "quality": "FRESH",
    "age_seconds": 60
  }
}
```

#### Signals (Automation Recommendations)

```
GET /v1/signals/is-green
GET /v1/signals/should-charge
GET /v1/signals/charge-speed
```

**is-green Response:**
```json
{
  "is_green": true,
  "renewable_percentage": 68.5,
  "threshold": 50,
  "metadata": {
    "data_age_seconds": 2145,
    "primary_source": "ENTSO-E",
    "quality": "STALE"
  }
}
```

---

## Architecture Decision Records (ADRs)

### ADR-001: Three-Layer Data Pipeline

**Context:** Need reliable data processing with clear separation of concerns. Data comes from external APIs with different formats (XML, JSON), must be validated, stored, and transformed.

**Decision:** Separate data processing into three distinct layers:
1. **Collectors** - Fetch raw data from APIs
2. **Importers** - Parse and insert to RAW tables
3. **Normalizers** - Transform to normalized tables with quality metadata

**Consequences:**
- ✅ Each layer can fail independently
- ✅ Easy to debug (check RAW data first)
- ✅ Can replay/reprocess any layer without API calls
- ✅ Audit trail (raw XML preserved on disk)
- ❌ More moving parts to orchestrate
- ❌ Disk space for raw XML files (mitigated by log rotation)

**Alternatives Considered:**
- Monolithic: Single script does fetch + parse + insert (simpler but couples concerns)
- Direct API → API: No intermediate storage (no audit trail, can't replay)

---

### ADR-002: Fail-Fast Configuration

**Context:** Silent failures in early versions caused hours of debugging. Missing or incorrect config led to cryptic errors deep in the code.

**Decision:** All required configuration must be present and valid at startup. If any required variable is missing or invalid, fail immediately with a clear error message.

**Consequences:**
- ✅ Immediate feedback on misconfiguration
- ✅ "Works on my machine" issues are eliminated
- ✅ Clear error messages guide operators
- ❌ Requires complete setup before any testing
- ❌ No graceful degradation

**Implementation:**
```python
from config.settings import require_env

# Fails immediately if missing
DATABASE_URL = require_env("DATABASE_URL", "PostgreSQL connection string")
ENTSOE_API_KEY = require_env("ENTSOE_API_KEY", "ENTSO-E Transparency API key")
```

---

### ADR-003: Template-Based Deployment

**Context:** Same codebase needs to be deployed with different branding on multiple servers. Hardcoded brand strings cause git conflicts and require manual edits per deployment.

**Decision:** Use `{{PLACEHOLDER}}` templates in systemd services and scripts. Generate final versions at install time by substituting environment variables.

**Consequences:**
- ✅ Single codebase, multiple brands
- ✅ No git conflicts between deployments
- ✅ Audit trail (can see what was substituted)
- ❌ Extra generation step during install
- ❌ Placeholders must be carefully chosen to avoid false matches

**Placeholders:**
```
{{BRAND_NAME}} → "Energy Insights NL"
{{BRAND_SLUG}} → "energy-insights-nl"
{{INSTALL_PATH}} → "/opt/energy-insights-nl"
{{LOG_PATH}} → "/var/log/energy-insights-nl"
{{API_PORT}} → "8000"
{{SERVICE_USER}} → "energy-insights-nl"
{{DB_NAME}} → "energy_insights_nl"
```

---

### ADR-004: XML File Caching

**Context:** ENTSO-E has rate limits. Need ability to replay imports and debug data issues without making expensive API calls.

**Decision:** Collectors save raw XML to disk before processing. Importers read from disk, not from collector output.

**Consequences:**
- ✅ Can replay imports without API calls
- ✅ Debug data issues by inspecting raw files
- ✅ Natural backup of source data
- ✅ Rate limit protection (disk storage, not API calls)
- ❌ Disk space usage (managed with log rotation)
- ❌ File I/O adds latency

**Storage:**
```
/var/log/{brand}/collectors/raw/
├── a75_2025-12-30_1430.xml
├── a75_2025-12-30_1415.xml
└── ...
```

---

### ADR-005: Quality Metadata on All Data

**Context:** Users need to know if data is fresh from ENTSO-E, stale, from fallback, or missing. Without this, automations might make decisions on bad data.

**Decision:** All normalized data includes quality metadata:
- `source` - Where data came from
- `quality` - Data quality status
- `age_seconds` - How old the data is
- `confidence_score` - 0-100 quality indicator

**Consequences:**
- ✅ Home Assistant can show data quality to users
- ✅ Automations can use fallback logic based on quality
- ✅ Monitoring can alert on data quality degradation
- ❌ Slightly more complex API responses
- ❌ Must maintain metadata through all transformations

---

### ADR-006: Automatic Fallback to Energy-Charts

**Context:** ENTSO-E is the authoritative source but sometimes unavailable or delayed. Energy-Charts (Fraunhofer ISE model) can provide backup data with realistic values.

**Decision:** When ENTSO-E data is stale (>150 min), automatically fetch from Energy-Charts and validate against realistic thresholds before serving.

**Consequences:**
- ✅ Service uptime: 95% (ENTSO-E only) → 99.9% (with fallback)
- ✅ Transparent to clients (metadata indicates source)
- ✅ Data validation prevents obviously wrong values
- ❌ Adds complexity to normalizer logic
- ❌ Energy-Charts data requires attribution in API responses

---

### ADR-007: Systemd for Scheduling

**Context:** Need reliable, production-grade scheduling for collectors/importers/normalizers. Cron is too fragile, manual scripts are unmaintainable.

**Decision:** Use systemd timers for all scheduling. Each task has a .service and .timer unit.

**Consequences:**
- ✅ Built-in to Ubuntu, no external dependencies
- ✅ Automatic retries on failure
- ✅ Integration with journalctl for logging
- ✅ Easy to manage: `systemctl status`, `systemctl start`
- ❌ Requires templates for multi-tenant deployments
- ❌ Learning curve for new operators

**Services:**
```
energy-insights-nl-collector.timer     (15 min)
energy-insights-nl-importer.timer      (15 min)
energy-insights-nl-normalizer.timer    (15 min)
energy-insights-nl-api.service         (always running)
```

---

### ADR-008: TenneT BYO-Key Architecture

**Context:** TenneT API Gateway General Terms prohibit "distributing, selling, or sharing data obtained through the APIs with third parties". Server-side redistribution of TenneT balance data violates these terms.

**Decision:** Move TenneT data fetching from server-side to client-side (Home Assistant component). Users provide their own TenneT API key (BYO = Bring Your Own).

**Implementation:**
1. Remove server-side collectors, importers, normalizers for TenneT
2. Archive database tables: `raw_tennet_balance`, `norm_tennet_balance`
3. Return 501 Not Implemented for `/api/v1/balance` API endpoint
4. Implement TenneT client in Home Assistant custom component
5. Balance sensors (`balance_delta`, `grid_stress`) created only when user provides personal API key
6. Data fetched locally in Home Assistant, never passes through SYNCTACLES servers

**API Impact:**
```
Before (old endpoint):
GET /v1/balance/current → 200 OK + TenneT data

After (new endpoint):
GET /api/v1/balance → 501 Not Implemented

Why the change:
- TenneT license prohibits server-side data redistribution
- User's personal key = user owns the data access
- Home Assistant = local processing (HA owns the hardware)
- No data leaves user's local network
```

**Data Flow (BYO-Key Model):**
```
User's Home Assistant
  ↓
  ├─ HA Integration (ha-energy-insights-nl)
  │  ├─ User provides: TenneT API key
  │  └─ Local TenneT client fetches: balance_delta, grid_stress
  │     (Data never leaves HA → No SYNCTACLES involvement)
  │
  ├─ SYNCTACLES API (via X-API-Key)
  │  ├─ Fetches: generation-mix, load (ENTSO-E only)
  │  └─ No TenneT data involved
```

**Sensor Availability:**
```
Scenario 1: Without TenneT key
  ✓ sensor.energy_insights_nl_generation_total (from server)
  ✓ sensor.energy_insights_nl_load_actual (from server)
  ✗ sensor.energy_insights_nl_balance_delta
  ✗ sensor.energy_insights_nl_grid_stress

Scenario 2: With TenneT key (configured in HA)
  ✓ sensor.energy_insights_nl_generation_total (from server)
  ✓ sensor.energy_insights_nl_load_actual (from server)
  ✓ sensor.energy_insights_nl_balance_delta (from TenneT, local)
  ✓ sensor.energy_insights_nl_grid_stress (from TenneT, local)
```

**Consequences:**
- ✅ Legally compliant (user's personal key, local processing only)
- ✅ No server-side TenneT infrastructure to maintain (simpler ops)
- ✅ Users control their own rate limits and data storage
- ✅ Demonstrates thoughtful legal architecture
- ✅ Reduced server load (no TenneT polling)
- ✅ GDPR compliant (no personal data on SYNCTACLES servers)
- ❌ Requires user to obtain TenneT key separately (one-time 5-min effort)
- ❌ Balance data optional (not all users will configure)
- ❌ HA component slightly more complex (local TenneT client needed)

**Alternatives Considered:**
1. **Server-side redistribution** - REJECTED (illegal, violates TenneT terms)
2. **ENTSO-E proxy for balance** - REJECTED (60-90 min delay, inferior quality)
3. **Remove balance feature entirely** - REJECTED (loses competitive advantage)
4. **BYO-key in HA component** - SELECTED (legal, maintains feature, user control)

**Migration Status:**
- ✅ 2026-01-02: Archive server-side TenneT infrastructure (Fase 1)
- ✅ 2026-01-02: Update documentation with BYO-key instructions (Fase 2)
- 🔄 2026-01-02: Implement HA component TenneT client (Fase 3)
- ⏳ 2026-01-02: Verify compliance and user feedback (Fase 4)

---

## Deployment Model

### Installation Architecture

```
/opt/
├── .env                              # Shared environment config
├── github/
│   └── ha-energy-insights-nl/        # Git repository (read-only)
└── energy-insights-nl/               # Deployment instance
    ├── app/                          # Application code
    │   ├── synctacles_db/            # Python modules
    │   ├── alembic/                  # Database migrations
    │   ├── config/                   # Configuration
    │   └── systemd/                  # Service templates
    ├── venv/                         # Python virtual environment
    └── logs/                         # Symlink to /var/log/...

/var/log/energy-insights-nl/
├── collectors/
│   └── raw/                          # Raw XML files
├── scheduler/                        # Systemd service logs
└── api/                              # FastAPI logs

/etc/systemd/system/
└── energy-insights-nl-*.service|timer # Systemd units
```

### Multi-Tenant Support

Same codebase can be deployed multiple times with different branding:

**Server 1: NL Instance**
```
BRAND_NAME="Energy Insights NL"
BRAND_SLUG="energy-insights-nl"
INSTALL_PATH="/opt/energy-insights-nl"
DB_NAME="energy_insights_nl"
SERVICE_USER="energy-insights-nl"
API_PORT="8000"
```

**Server 2: DE Instance (same server or different)**
```
BRAND_NAME="Energy Insights DE"
BRAND_SLUG="energy-insights-de"
INSTALL_PATH="/opt/energy-insights-de"
DB_NAME="energy_insights_de"
SERVICE_USER="energy-insights-de"
API_PORT="8001"
```

Each deployment is completely isolated:
- Separate databases
- Separate systemd units
- Separate Python venvs
- Separate logs

### High Availability (Future)

For production:
1. Database replication (Postgres streaming replication)
2. API load balancer (Nginx/HAProxy)
3. Collector failover (primary + secondary servers)
4. Automated health checks and failover

---

## Security Model

### ADR-001: Network Security via Hetzner Cloud Firewall

**Status:** Accepted
**Date:** 2026-01-09

**Context:**
Servers draaien op Hetzner Cloud. Keuze tussen UFW (OS-level) of Hetzner Cloud Firewall (netwerkniveau).

**Decision:**
Hetzner Cloud Firewall als primaire netwerkbeveiliging. Geen UFW op servers.

**Rationale:**
- KISS: één firewall-laag, centraal beheerd
- Traffic geblokkeerd vóór server (minder load)
- Eenvoudiger auditing via Hetzner console
- Minder OS-configuratie drift tussen servers

**Consequences:**
- Firewall rules alleen via Hetzner console/API
- Security audit checkt Hetzner Firewall, niet UFW

---

### ADR-002: PostgreSQL Trust Authentication (No Password)

**Status:** Accepted
**Date:** 2026-01-09

**Context:**
Security audit adviseerde scram-sha-256 (password auth) voor PostgreSQL. Standaard best practice is "altijd wachtwoord".

**Decision:**
Behoud `trust` authentication voor localhost connections. Geen password.

**Rationale:**

Threat model analyse:

| Aanvaller shell als... | trust | password in .env |
|------------------------|-------|------------------|
| service user | DB access | Leest .env → DB access |
| andere user | DB access | .env onleesbaar (600) → geblokkeerd |
| root | DB access | DB access |

Password beschermt alleen tegen niet-service-user én niet-root shell access. Op deze server:
- Geen andere users
- Aanvaller is service user (app exploit) of root
- Password in plaintext .env = security theater

Echte bescherming zit in perimeter:
1. Hetzner FW — voorkomt toegang
2. SSH keys-only — voorkomt toegang
3. localhost binding — DB niet extern bereikbaar

Binnen = binnen.

**Consequences:**
- Eenvoudiger setup (KISS)
- Geen secrets rotation nodig voor DB
- Geen .env credential sprawl
- Bewuste keuze, geen oversight

---

### Current State (Development)

⚠️ **Not production ready**

| Aspect | Current | Production Required |
|--------|---------|---------------------|
| API Auth | None | API key / JWT |
| DB Auth | Peer (Unix socket) | Password + SSL |
| Secrets | Plain text .env | Encrypted at rest |
| Network | Hetzner Cloud Firewall | Firewall + reverse proxy |
| TLS | None | Full HTTPS |

### Production Hardening

#### 1. Database Security

```bash
# Use password authentication instead of peer auth
ALTER USER energy_insights_nl PASSWORD 'strong_password';

# Update pg_hba.conf
local all all scram-sha-256

# Require SSL connections
ALTER SYSTEM SET ssl = on;
```

#### 2. API Authentication

Options:
- **API Key** - Simple, suitable for fixed client list
- **JWT** - Stateless, suitable for distributed clients
- **Reverse Proxy Auth** - Nginx/Caddy handles auth, API trusts proxies

#### 3. Secrets Management

Options:
- **Encrypted .env** - Simple, good for small deployments
- **HashiCorp Vault** - Enterprise, good for large deployments
- **systemd Credentials** - Linux native, good integration

#### 4. File Permissions

```bash
# .env readable only by owner and service user
chmod 600 /opt/.env
chown root:energy-insights-nl /opt/.env

# Python venv and app code owned by service user
chown -R energy-insights-nl:energy-insights-nl /opt/energy-insights-nl/app
chmod 750 /opt/energy-insights-nl/app
```

#### 5. Network Security

- Reverse proxy (Nginx) in front of API
- TLS termination at proxy
- Firewall rules (inbound: only HTTPS, SSH)
- Database on private network only

---

## Monitoring & Operations

### Health Checks

**Application Level:**
```bash
curl http://localhost:8000/health                    # Basic health check
curl http://localhost:8000/v1/pipeline/health        # Detailed pipeline status (JSON)
curl http://localhost:8000/v1/pipeline/metrics       # Pipeline metrics (Prometheus)
curl http://localhost:8000/metrics                   # General app metrics (Prometheus)
```

**System Level:**
```bash
# Check timers
systemctl list-timers "energy-insights-nl-*"

# Check service status
systemctl status energy-insights-nl-api

# Check logs
journalctl -u energy-insights-nl-* --since "1 hour ago"
```

**Systemd Timers:**

| Timer | Interval | Purpose | Depends On |
|-------|----------|---------|------------|
| `energy-insights-nl-collector.timer` | 15 min | Fetch data from ENTSO-E/Energy-Charts APIs | - |
| `energy-insights-nl-importer.timer` | 15 min | Import collected data to raw_* tables | collector |
| `energy-insights-nl-normalizer.timer` | 15 min | Normalize data to norm_* tables (A44, A65, A75, prices) | importer |
| `energy-insights-nl-health.timer` | 5 min | System health checks and diagnostics | - |

**Timer Configuration:**
- **OnBootSec**: Delay after system boot (1-2 min)
- **OnUnitActiveSec**: Interval between runs
- **Dependencies**: Timers run independently, but data flows collector → importer → normalizer

### Metrics & Observability

**✅ Implemented:**
- Prometheus metrics endpoint ([/v1/pipeline/metrics](../synctacles_db/api/routes/pipeline.py:168-209))
- General app metrics endpoint (`/metrics` - FastAPI default)
- Grafana dashboard: [Pipeline Health](https://monitor.synctacles.com/d/5fd1f7f9-e2bb-4a81-a04e-50f9fbbf0ec0/pipeline-health)
- Alert rules for pipeline health (7 rules configured)

**Pipeline-Specific Metrics** (`/v1/pipeline/metrics`):

| Metric | Description | Labels |
|--------|-------------|--------|
| `pipeline_timer_status` | Timer status (1=active, 0=stopped) | timer={collector\|importer\|normalizer\|health} |
| `pipeline_timer_last_trigger_minutes` | Minutes since last trigger | timer=... |
| `pipeline_data_status` | Data status code (0-3) | source={a44\|a65\|a75} |
| `pipeline_data_freshness_minutes` | Data age in minutes | source=... |
| `pipeline_raw_norm_gap_minutes` | Gap between raw and normalized data | source=... |

**General App Metrics** (`/metrics`):

| Metric | Description |
|--------|-------------|
| `python_gc_*` | Python garbage collection stats |
| `python_info` | Python platform information |
| `process_*` | Process memory, CPU usage |
| `http_*` | HTTP request metrics (if enabled) |

**Data Status Codes:**
- **0 (FRESH):** Data <90 min old
- **1 (STALE):** Data 90-180 min old (**NORMAL for A75** due to ENTSO-E delay)
- **2 (UNAVAILABLE):** Data >180 min old
- **3 (NO_DATA):** No data in database

**Key Metrics:**
- Data freshness (age of latest record) ✅ Implemented
- Raw vs Normalized gap (indicates normalizer issues) ✅ Implemented (2026-01-08)
- Collector success rate (% of runs that completed) ⏳ Future
- API latency (p50, p95, p99) ⏳ Future
- Error rate (5xx errors per minute) ⏳ Future
- Fallback rate (% of requests using Energy-Charts) ⏳ Future

### Cache Management

**Endpoints:**

| Endpoint | Method | Purpose | Response |
|----------|--------|---------|----------|
| `/cache/stats` | GET | View cache statistics | `{"size": 0, "maxsize": 100, "hits": 0, "misses": 1, "hit_rate_pct": 0.0}` |
| `/cache/clear` | POST | Clear entire cache | `{"message": "Cache cleared", "items_removed": N}` |
| `/cache/invalidate/{pattern}` | POST | Invalidate specific cache entries | `{"message": "Pattern invalidated", "items_removed": N}` |

**Use Cases:**
- **stats**: Monitor cache performance, hit rates, memory usage
- **clear**: Force refresh after data quality issues or manual corrections
- **invalidate**: Selective cache invalidation for specific endpoints (e.g., `/cache/invalidate/generation`)

**Authentication:** Requires admin API key or internal access

**Example Usage:**
```bash
# View cache statistics
curl https://enin.xteleo.nl/cache/stats

# Clear entire cache (requires auth)
curl -X POST https://enin.xteleo.nl/cache/clear \
  -H "Authorization: Bearer $ADMIN_API_KEY"

# Invalidate specific pattern
curl -X POST https://enin.xteleo.nl/cache/invalidate/prices \
  -H "Authorization: Bearer $ADMIN_API_KEY"
```

### Pipeline Health Troubleshooting

**Common Issue: Normalizer Silent Failure**

**Symptom:** Timers show "active" but normalized data becomes stale while raw data is fresh.

**Diagnosis:**
```bash
# Check gap between raw and normalized
psql -d energy_insights_nl -c "
SELECT
    'a75' as source,
    EXTRACT(EPOCH FROM (
        (SELECT MAX(timestamp) FROM raw_entso_e_a75 WHERE timestamp <= NOW()) -
        (SELECT MAX(timestamp) FROM norm_entso_e_a75 WHERE timestamp <= NOW())
    ))/60 as gap_minutes;
"

# Gap >30 minutes indicates normalizer not processing data
```

**Root Cause:** Check [scripts/run_normalizers.sh](../scripts/run_normalizers.sh) - verify ALL sources are included:
```bash
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44  # Prices
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65  # Load ⚠️ May be missing
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75  # Generation ⚠️ May be missing
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices       # Price post-processing
```

**If normalizers are missing:** This is a CRITICAL issue requiring CAI review before fixes (see [HANDOFF_CC_CAI_NORMALIZER_MISSING_CRITICAL.md](handoffs/HANDOFF_CC_CAI_NORMALIZER_MISSING_CRITICAL.md))

**ENTSO-E A75 Normal Behavior:**

A75 showing "STALE" status is **EXPECTED** due to ENTSO-E publishing delay:
- **Normal delay:** 2-4 hours after actual generation
- **Update schedule:** 13:01 UTC daily (large batch), 03:54 UTC (smaller update)
- **Only investigate if:** A75 shows UNAVAILABLE (>180 min) OR raw-to-normalized gap >30 min

### Log Locations

| Component | Log Location |
|-----------|-------------|
| Collectors | `/var/log/energy-insights-nl/scheduler/collectors_*.log` |
| Importers | `/var/log/energy-insights-nl/scheduler/importers_*.log` |
| Normalizers | `/var/log/energy-insights-nl/scheduler/normalizers_*.log` |
| API | `journalctl -u energy-insights-nl-api` |
| Systemd | `journalctl -u energy-insights-nl-*` |

### Backup Strategy

**Daily automated backups:**
```bash
# Database backup (compressed)
pg_dump energy_insights_nl | gzip > /backup/energy_insights_nl_$(date +%Y%m%d).sql.gz

# Configuration backup
cp /opt/.env /backup/env_$(date +%Y%m%d).backup
```

**Retention:** 30 days (automatic cleanup)

### Disaster Recovery

| Scenario | RTO | Process |
|----------|-----|---------|
| Code rollback | 5 min | `git checkout <commit>`; restart services |
| Database corruption | 15 min | Stop services, restore backup, verify |
| Complete server loss | 1 hour | Provision new server, run installer, restore backup |

---

## Roadmap (F7-F9)

### F7: Technical Hardening (4-5 days)

**Goal:** Stable, reproducible, measurable, loadable system

| Item | Priority | Exit Criteria |
|------|----------|---------------|
| F7.1 | DNS + SSL | `curl https://domain.com` → 200, valid cert |
| F7.2 | Logging/Monitoring | Grafana dashboard live, alerts fire |
| F7.3 | Install Script | Fresh Ubuntu 24.04 → running, no manual steps |
| F7.4 | Uptime Monitoring | 48h+ uptime logs, external monitoring active |
| F7.5 | Load Testing | p95 <40ms, 5k RPS, no errors |
| F7.6 | Backup/Restore | Restore in <30min, data integrity verified |
| F7.7 | Automated Tests | pytest, coverage >80%, all pass |
| F7.8 | Release Discipline | Git tags, changelog, rollback procedure |
| F7.9 | Security Baseline | Firewall, SSH hardened, no secrets in git |
| F7.10 | SLO Definition | "API up" = all endpoints <100ms p95, <0.1% error |

### F8: Pre-Production (2-3 days)

**Goal:** New user: install → key → data → 15 minutes

| Item | Priority | Exit Criteria |
|------|----------|---------------|
| F8.1 | Home Assistant Intro | Understand HA architecture (entities, domains) |
| F8.2 | HA Integration | HACS installation, custom component, entities created |
| F8.3 | Fallback | Primary down → fallback active, transparent to user |
| F8.4 | License System | License key flow end-to-end, zero manual DB edits |
| F8.5 | User Management | Email signup → API key generation → rate limiting |
| F8.6 | Documentation | ReadTheDocs live, 90% coverage, searchable |

### F9: Production Launch (2-3 days)

**Goal:** Accept payments, support users, grow sustainably

| Item | Priority | Exit Criteria |
|------|----------|---------------|
| F9.1 | Payment Gateway | Mollie/Stripe integration, automatic license generation |
| F9.2 | Soft Launch | 5-10 beta users, paid, monitored, support active |
| F9.3 | Branding | Logo ready, website branded, brand guide exists |
| F9.4 | Support | Helpdesk system, FAQ, monitoring alerts to ops |
| F9.5 | Analytics | User behavior tracking, feature usage metrics |

---

## Glossary

- **ADR** - Architecture Decision Record
- **ENTSO-E** - European Network of Transmission System Operators (electricity)
- **PSR Type** - Power Generating Module Type (biomass, wind, solar, etc.)
- **A75** - ENTSO-E dataset: Generation per PSR type
- **A65** - ENTSO-E dataset: Load actual + forecast
- **A44** - ENTSO-E dataset: Electricity prices
- **TenneT** - Dutch Transmission System Operator
- **Fallback** - Automatic switch to backup data source
- **Forward Fill** - Use previous known value when current unavailable
- **Raw Data** - Unprocessed, as-received from API
- **Normalized Data** - Cleaned, validated, enriched with metadata

---

## Performance Benchmarks

### Cache Migration Results (2026-01-08) ✅ COMPLETED

After migrating all endpoints to unified api_cache pattern with 91.7% hit rate.

**Server:** Hetzner CX33 (4 vCPU, 8GB RAM) with 8 Gunicorn workers

| Concurrent Users | Req/sec | Success Rate | Avg Latency | Before Cache Fix |
|------------------|---------|--------------|-------------|-------------------|
| 10 | **1,621** | 100% | 5ms | 66 req/sec |
| 50 | **2,370** | 100% | 19ms | 284 req/sec |
| 100 | **2,596** | 100% | 34ms | 152 req/sec (90% success) |
| 200 | **2,354** | 100% | 68ms | N/A |
| 500 | **880** | 100% | 224ms | N/A |
| 1,000 | **1,623** | 99.97% | 451ms | N/A |
| 2,000 | **1,362** | 99.97% | 1,402ms | N/A |

**Key Achievements:**
- ✅ **24.5x throughput** improvement at 10 concurrent (66 → 1,621 req/sec)
- ✅ **17x throughput** improvement at 100 concurrent (152 → 2,596 req/sec)
- ✅ **20x capacity** increase (100 → 2,000 concurrent users)
- ✅ **100% success rate** up to 500 concurrent users
- ✅ **Cache hit rate: 91.7%** across all endpoints

**Production Capacity:**
- **Sweet spot:** 100-200 concurrent (2,350+ req/sec, <100ms latency)
- **Conservative limit:** 500 concurrent (100% success, 200ms latency)
- **Maximum capacity:** 2,000 concurrent (99.97% success)

**See:**
- [Load Test Report 2026-01-08](reports/LOAD_TEST_REPORT_2026-01-08.md) (initial)
- [Capacity Test Report 2026-01-08](reports/CAPACITY_TEST_2026-01-08.md) (post-fix)

---

## Further Reading

- [API Reference](api-reference.md)
- [Deployment Guide](deployment.md)
- [Troubleshooting Guide](troubleshooting.md)
- [Environment Variables](env-reference.md)

---

## Critical Known Issues

### 1. Missing Normalizers in run_normalizers.sh

**Status:** ✅ RESOLVED (2026-01-08)

**Description:** `scripts/run_normalizers.sh` was missing A65 (load) and A75 (generation) normalizers.

**Impact (Historical):**
- A65/A75 normalized data became stale (2+ hours old)
- Potential ENTSO-E license compliance issue
- Inconsistent data quality across sources

**Resolution:**
- Added A65 and A75 normalizers to `scripts/run_normalizers.sh`
- Added `pipeline_raw_norm_gap_minutes` metric for early detection
- Created `scripts/validate_pipeline.sh` to prevent regression
- Deployed 2026-01-08, verified all sources FRESH

**See:**
- [HANDOFF_CAI_CC_NORMALIZER_FIX_AND_DOCUMENTATION.md](handoffs/HANDOFF_CAI_CC_NORMALIZER_FIX_AND_DOCUMENTATION.md)
- Git commit: `ce92159`

---

### 2. DNS Resolution Broken on monitor.synctacles.com

**Status:** ⚠️ KNOWN LIMITATION - Workaround in Place

**Description:** Monitoring server cannot resolve external domains (e.g., api.synctacles.com)

**Impact:**
- Grafana Infinity plugin does NOT work (shows "No data")
- JSON-based datasources unusable
- Limits dashboard design options

**Workaround:** Use Prometheus datasource with IP address + SNI header configuration

**See:** [HANDOFF_CC_CAI_GRAFANA_DASHBOARD_COMPLETE.md](handoffs/HANDOFF_CC_CAI_GRAFANA_DASHBOARD_COMPLETE.md)

**Resolution:** Low priority - Prometheus approach works reliably

---

**Last Updated:** 2026-01-08
**Status:** Production Ready (with known issues documented)
**Version:** 1.1
