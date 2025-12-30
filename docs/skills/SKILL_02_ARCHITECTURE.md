# SKILL 2 — SYSTEM ARCHITECTURE

Design Principles and 3-Layer Data Pipeline
Version: 2.0 (2025-12-30)

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
EXTERNAL SOURCES
├── ENTSO-E (Generation, Load, Prices)
├── TenneT (Grid Balance)
└── Energy-Charts (Fallback)
     │
     ▼
LAYER 1: COLLECTORS
├── entso_e_a75_generation.py  (15-min)
├── entso_e_a65_load.py        (15-min)
├── entso_e_a44_prices.py      (hourly)
├── tennet_ingestor.py         (5-min)
└── energy_charts_client.py    (fallback, cached)
     │
     ▼ (saves to /var/log/{{BRAND}}/collectors/raw/*.xml)
LAYER 2: IMPORTERS
├── import_entso_e_a75.py  → raw_entso_e_a75
├── import_entso_e_a65.py  → raw_entso_e_a65
├── import_entso_e_a44.py  → raw_entso_e_a44
└── import_tennet_balance.py → raw_tennet_balance
     │
     ▼ (PostgreSQL RAW tables)
LAYER 3: NORMALIZERS
├── normalize_entso_e_a75.py  → norm_generation
├── normalize_entso_e_a65.py  → norm_load
└── normalize_tennet_balance.py → norm_grid_balance
     │
     ▼ (with quality metadata)
LAYER 4: API
├── FastAPI application
├── /v1/generation/current
├── /v1/load/current
├── /v1/prices/today
├── /v1/balance/current
└── /health (system status)
     │
     ▼
HOME ASSISTANT
(custom component integration)
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

#### `/health`
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "services": {
    "database": "connected",
    "entso_e": "ok",
    "tennet": "ok",
    "energy_charts": "ok"
  }
}
```

---

## FALLBACK STRATEGY

When primary data sources fail, system automatically uses fallback.

### Primary → Fallback Cascade

```
Generation Data:
  1st choice: ENTSO-E A75 (real-time, 15-min)
  2nd choice: Energy-Charts (modeled, hourly)
  3rd choice: Last known good value

Load Data:
  1st choice: ENTSO-E A65 (real-time, 15-min)
  2nd choice: Energy-Charts (modeled, hourly)
  3rd choice: Forecast value

Prices:
  1st choice: ENTSO-E A44 (real-time, hourly)
  2nd choice: Energy-Charts (estimated)
```

### Quality Indicators

API always returns quality score so clients can decide whether to use data:
- `quality: 0.99` - Use normally
- `quality: 0.85` - Use, but note it's slightly stale
- `quality: 0.60` - Use cautiously, prefer other sources
- `quality: <0.5` - Use only if no alternative

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
