# SKILL 2 — SYSTEM ARCHITECTURE

Design Principles and Data Pipeline
**Version: 4.0 (2026-01-19) - Generated from source code**

> **KISS Migration v2.0.0:** System now focuses on Energy Action with 6-tier fallback stack.
> Dashboard endpoint bundles all data for HA efficiency.

---

## EXECUTIVE SUMMARY

SYNCTACLES provides energy price data and actionable recommendations via REST API.
The HA integration consumes this data and adds optional Enever BYO-key pricing.

**Key Architecture Changes (v4.0):**
- Dashboard endpoint (`/api/v1/dashboard`) bundles all data in single call
- 6-tier fallback stack (Frank DB → Frank Direct → EasyEnergy → ENTSO-E+Offset → Energy-Charts → Cache)
- P1/Live Cost sensors for real-time cost calculation
- 24 Enever providers (up from 19)

---

## API ENDPOINTS

### Primary Endpoints

| Endpoint | Method | Purpose | Cache TTL |
|----------|--------|---------|-----------|
| `/api/v1/dashboard` | GET | Bundled data (primary) | 2 min |
| `/api/v1/prices` | GET | Raw prices (fallback) | 5 min |
| `/api/v1/best-window` | GET | Best consumption window | 5 min |
| `/api/v1/tomorrow` | GET | Tomorrow preview | 5 min |
| `/api/v1/energy-action` | GET | GO/WAIT/AVOID recommendation | 2 min |
| `/health` | GET | System health | - |
| `/metrics` | GET | Prometheus metrics | - |

### Dashboard Response Structure

**File:** `synctacles_db/api/endpoints/windows.py:579-746`

```json
{
  "current": {
    "price_eur_kwh": 0.2509,
    "action": "WAIT",
    "action_reason": "Price +5% vs average",
    "cheapest_hour": "03:00",
    "cheapest_price_eur_kwh": 0.1982,
    "most_expensive_hour": "18:00",
    "most_expensive_price_eur_kwh": 0.3421,
    "average_price_eur_kwh": 0.2401,
    "hours_available": 24
  },
  "best_window": {
    "start": "2026-01-19T02:00:00+00:00",
    "end": "2026-01-19T05:00:00+00:00",
    "start_hour": "02:00",
    "end_hour": "05:00",
    "duration_hours": 3,
    "average_price_eur_kwh": 0.2012,
    "total_cost_estimate_eur": 0.6036
  },
  "runner_up": {
    "start_hour": "13:00",
    "end_hour": "16:00",
    "average_price_eur_kwh": 0.2156
  },
  "tomorrow": {
    "status": "FAVORABLE|NORMAL|EXPENSIVE|PENDING",
    "date": "2026-01-20",
    "cheapest_hour": "04:00",
    "cheapest_price_eur_kwh": 0.1845,
    "most_expensive_hour": "17:00",
    "most_expensive_price_eur_kwh": 0.3012,
    "average_price_eur_kwh": 0.2234,
    "best_window_3h": {...},
    "hours_available": 24
  },
  "meta": {
    "source": "Frank Direct",
    "quality": "FRESH",
    "country": "NL",
    "window_duration": 3,
    "hours_analyzed": 48
  },
  "timestamp": "2026-01-19T14:32:15.123456+00:00"
}
```

### Prices Response Structure

**File:** `synctacles_db/api/endpoints/prices.py:72-212`

```json
{
  "data": [
    {
      "timestamp": "2026-01-19T00:00:00+00:00",
      "price_eur_mwh": 85.50,
      "_reference": {
        "source": "Frank DB",
        "tier": 1,
        "expected_range": {"low": 0.20, "high": 0.35, "expected": 0.27}
      }
    }
  ],
  "meta": {
    "source": "Frank DB",
    "quality_status": "FRESH",
    "data_age_seconds": 120,
    "count": 48,
    "allow_go_action": true
  }
}
```

---

## DATA FLOW ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────────────┐
│                     6-TIER FALLBACK STACK                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Tier 1: Frank DB (PostgreSQL)        ← PRIMARY (100% consumer)    │
│    File: fallback_manager.py:591-600                                │
│    Table: frank_prices                                              │
│    Quality: FRESH/STALE                                             │
│    allow_go_action: TRUE                                            │
│         │                                                           │
│         ▼ (if unavailable)                                          │
│                                                                     │
│  Tier 2: Frank Direct API             ← LIVE consumer (100%)       │
│    File: frank_energie_client.py:82-198                             │
│    URL: https://graphql.frankenergie.nl                             │
│    Circuit breaker: 5 min cooldown after 3 failures                 │
│    allow_go_action: TRUE                                            │
│         │                                                           │
│         ▼ (if unavailable)                                          │
│                                                                     │
│  Tier 3: EasyEnergy Direct API        ← LIVE wholesale (100%)      │
│    File: easyenergy_client.py:83-184                                │
│    URL: https://mijn.easyenergy.com/nl/api/tariff/getapxtariffs    │
│    Returns: APX/EPEX spot prices                                    │
│    allow_go_action: TRUE                                            │
│         │                                                           │
│         ▼ (if unavailable)                                          │
│                                                                     │
│  Tier 4: ENTSO-E + Static Offset      ← CALCULATED (85-89%)        │
│    File: fallback_manager.py:653-668                                │
│    Table: norm_entso_e_a44                                          │
│    Offset: config/static_offsets.py                                 │
│    allow_go_action: TRUE                                            │
│         │                                                           │
│         ▼ (if unavailable)                                          │
│                                                                     │
│  Tier 5: Energy-Charts + Offset       ← FALLBACK (85-89%)          │
│    File: fallback_manager.py:673-697                                │
│    Circuit breaker: 2h cooldown after 404                           │
│    allow_go_action: FALSE (CRITICAL!)                               │
│         │                                                           │
│         ▼ (if unavailable)                                          │
│                                                                     │
│  Tier 6: Cache (Memory + PostgreSQL)  ← CACHED (stale)             │
│    File: fallback_manager.py:704-715                                │
│    In-memory: TTLCache (5 min TTL)                                  │
│    PostgreSQL: price_cache table (24h)                              │
│    allow_go_action: FALSE                                           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## SERVER CODE STRUCTURE

```
synctacles_db/
├── api/
│   ├── main.py                    # FastAPI app, middleware, routes (1-170)
│   ├── middleware.py              # Auth, rate limit, logging
│   ├── dependencies.py            # Shared dependencies
│   ├── models.py                  # Pydantic models
│   └── endpoints/
│       ├── windows.py             # Dashboard, Best Window, Tomorrow (1-747)
│       ├── prices.py              # Prices endpoint (1-213)
│       ├── energy_action.py       # GO/WAIT/AVOID endpoint
│       ├── auth.py                # API key management
│       ├── balance.py             # Balance endpoint
│       └── deprecated.py          # 410 Gone endpoints
│
├── clients/
│   ├── frank_energie_client.py    # Frank GraphQL client (1-279)
│   ├── easyenergy_client.py       # EasyEnergy REST client (1-267)
│   └── consumer_price_client.py   # Consumer price calculation
│
├── fallback/
│   ├── fallback_manager.py        # 6-tier fallback logic (1-943)
│   └── energy_charts_client.py    # Energy-Charts API client
│
├── config/
│   └── static_offsets.py          # Hourly price offsets (KISS)
│
├── cache.py                       # Central api_cache singleton
├── freshness_config.py            # Quality thresholds
└── services/
    └── price_cache.py             # PostgreSQL price cache
```

---

## HOME ASSISTANT COMPONENT STRUCTURE

**Repository:** `ha-energy-insights-nl`
**Version:** 2.2.3 (as of 2026-01-19)

```
custom_components/ha_energy_insights_nl/
├── manifest.json                  # Version, domain, requirements (1-14)
├── __init__.py                    # Entry point, coordinators (1-440)
│   ├── ServerDataCoordinator      # Lines 125-234
│   └── EneverDataCoordinator      # Lines 237-439
│
├── sensor.py                      # All sensors (1-1583)
│   ├── PriceCurrentSensor         # Lines 404-517
│   ├── CheapestHourSensor         # Lines 520-592
│   ├── ExpensiveHourSensor        # Lines 595-659
│   ├── EnergyActionSensor         # Lines 662-818
│   ├── BestWindowSensor           # Lines 1013-1068
│   ├── TomorrowPreviewSensor      # Lines 1071-1147
│   ├── LiveCostSensor             # Lines 1153-1319
│   ├── SavingsSensor              # Lines 1321-1583
│   ├── PricesTodaySensor          # Lines 821-910 (Enever BYO)
│   └── PricesTomorrowSensor       # Lines 913-1006 (Enever BYO)
│
├── config_flow.py                 # Setup wizard (1-373)
│   ├── ConfigFlow                 # Lines 117-263
│   └── OptionsFlow                # Lines 266-373
│
├── enever_client.py               # Enever API client (1-129)
│   └── LEVERANCIERS               # 24 providers (lines 12-37)
│
├── const.py                       # Constants (1-77)
└── translations/
    ├── en.json                    # English
    └── nl.json                    # Dutch
```

---

## COORDINATOR DETAILS

### ServerDataCoordinator

**File:** `__init__.py:125-234`

| Property | Value |
|----------|-------|
| Endpoint | `/api/v1/dashboard` |
| Polling interval | 15 minutes (`UPDATE_INTERVAL_SERVER`) |
| Cache fallback | 30 minutes (`CACHE_MAX_AGE_MINUTES`) |
| Fallback endpoint | `/api/v1/prices` |

**Data fetched:**
- `dashboard` (full response)
- `best_window`
- `runner_up`
- `tomorrow`
- `current`
- `meta`

### EneverDataCoordinator

**File:** `__init__.py:237-439`

| Property | Value |
|----------|-------|
| Endpoint | `https://enever.nl/apiv3/stroomprijs_*.php` |
| Polling interval | 1 hour |
| Smart caching | Fetch tomorrow @15:00, promote @midnight |
| API reduction | ~50% (31 calls/month vs 62) |

**Data fetched:**
- `prices_today`
- `prices_tomorrow`
- `leverancier`
- `resolution_minutes` (60 or 15)

---

## DATABASE SCHEMA

### Server Tables

```sql
-- Frank consumer prices (Tier 1)
CREATE TABLE frank_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    price_eur_kwh NUMERIC(10, 6) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ENTSO-E normalized prices (Tier 4)
CREATE TABLE norm_entso_e_a44 (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    country VARCHAR(2) NOT NULL,
    price_eur_mwh NUMERIC(10, 4) NOT NULL,
    data_source VARCHAR(50),
    data_quality VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Price cache (Tier 6)
CREATE TABLE price_cache (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    country VARCHAR(2) DEFAULT 'NL',
    price_eur_kwh NUMERIC(10, 6) NOT NULL,
    source VARCHAR(50) NOT NULL,
    quality VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## CACHING STRATEGY

| Cache Type | Location | TTL | Purpose |
|------------|----------|-----|---------|
| `api_cache` | In-memory (TTLCache) | 2-5 min | API response caching |
| `_frank_cache` | In-memory | 5 min | Frank API response |
| `_fallback_cache` | In-memory | 5 min | Fallback data |
| `price_cache` | PostgreSQL | 24h | Persistent fallback |

**Cache keys:**
- `dashboard:{country}:{window_duration}` → 2 min TTL
- `prices:{country}:{hours}` → 5 min TTL
- `best-window:{country}:{duration}` → 5 min TTL
- `tomorrow:{country}` → 5 min TTL

---

## CIRCUIT BREAKERS

### Frank Energie Circuit Breaker

**File:** `frank_energie_client.py:29-34`

```python
_circuit_breaker = {
    "last_failure_time": None,
    "cooldown_minutes": 5,
    "failure_count": 0,
    "max_failures": 3,
}
```

### EasyEnergy Circuit Breaker

**File:** `easyenergy_client.py:30-35`

Same pattern as Frank.

### Energy-Charts Circuit Breaker

**File:** `fallback_manager.py:39-43`

```python
_ec_circuit_breaker = {
    "last_404_time": None,
    "cooldown_minutes": 120,  # 2 hours
    "is_open": False,
}
```

---

## QUALITY & FRESHNESS

### Quality Status Values

| Status | Meaning | GO Action |
|--------|---------|-----------|
| `FRESH` | Data < 15 min old | YES |
| `STALE` | Data 15-60 min old | YES |
| `PARTIAL` | Hybrid merge | YES |
| `FALLBACK` | Energy-Charts | NO |
| `CACHED` | From cache | NO |
| `UNAVAILABLE` | No data | NO |

### Freshness Thresholds

**File:** `freshness_config.py`

| Source | Fresh | Stale |
|--------|-------|-------|
| ENTSO-E | < 15 min | 15-60 min |
| Frank | < 360 min (6h) | > 360 min |

---

## ENEVER PROVIDERS (24)

**File:** `enever_client.py:12-37`

| Provider | API Code |
|----------|----------|
| ANWB Energie | prijsANWB |
| Budget Energie | prijsBE |
| Coolblue Energie | prijsCB |
| EasyEnergy | prijsEE |
| Energiedirect | prijsED |
| Energie van Ons | prijsEVO |
| Energiek | prijsEG |
| EnergyZero | prijsEZ |
| Essent | prijsES |
| Frank Energie | prijsFR |
| Groenestroom Lokaal | prijsGSL |
| Hegg Energy | prijsHE |
| Innova Energie | prijsIN |
| Mijndomein Energie | prijsMDE |
| NextEnergy | prijsNE |
| Pure Energie | prijsPE |
| Quatt | prijsQU |
| SamSam | prijsSS |
| Tibber | prijsTI |
| Vandebron | prijsVDB |
| Vattenfall | prijsVF |
| Vrij op naam | prijsVON |
| Wout Energie | prijsWE |
| Zonneplan | prijsZP |

**Note:** Eneco (prijsEN) is NOT included - they only offer dynamic GAS, not electricity.

---

## DISCONTINUED FEATURES

| Feature | Removed | Reason |
|---------|---------|--------|
| TenneT integration | Phase 3 | BYO-key in HA only |
| A65 (load) endpoints | Phase 3 | Energy Action focus |
| A75 (generation) endpoints | Phase 3 | Energy Action focus |
| Grid stress sensors | Phase 3 | Simplified to GO/WAIT/AVOID |
| Balance sensors | Phase 3 | Not needed for price-based actions |
| Coefficient server | KISS v2.0.0 | Replaced by static offsets |

---

## RELATED SKILLS

- **SKILL 1**: Hard Rules
- **SKILL 3**: Coding Standards
- **SKILL 4**: Product Requirements
- **SKILL 6**: Data Sources
- **SKILL 10**: Deployment Workflow
- **SKILL 15**: Consumer Price Engine (partially superseded by static offsets)

---

*Generated from source code: 2026-01-19*
*Scanned files: windows.py, prices.py, fallback_manager.py, frank_energie_client.py, easyenergy_client.py, __init__.py, sensor.py, config_flow.py, enever_client.py, const.py*
