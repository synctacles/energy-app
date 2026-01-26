# SKILL ENERGY 00 — OVERVIEW

Product Overview and Features
**Version: 4.0 (2026-01-26)**

> **Purpose:** SYNCTACLES Energy provides real-time electricity price data and actionable recommendations for smart energy consumption.
> Energy is the acquisition funnel, Care is the kernproduct.

---

## EXECUTIVE SUMMARY

SYNCTACLES Energy delivers:
- **Price Data**: Real-time and day-ahead electricity prices (NL + EU)
- **Energy Action**: GO/WAIT/AVOID recommendations based on current prices
- **Best Windows**: Optimal consumption windows for flexible loads
- **BYO Pricing**: Bring-your-own-key Enever API integration (24 Dutch providers)
- **Live Cost**: Real-time energy cost calculation from P1 meters

**Repositories:**
- API Server: `synctacles-api` (this repo)
- HA Integration: `ha-integration`

**Runtime:**
- API: FastAPI + PostgreSQL on Hetzner VPS
- Integration: Python package in HA custom_components

---

## PRODUCT TIER MATRIX

| Feature | Gratis | Trial (14d NL) | Premium |
|---------|--------|----------------|---------|
| Prijzen NL | ✅ | ✅ | ✅ |
| Goedkoopste uren NL | ✅ | ✅ | ✅ |
| GO/WAIT/AVOID | ❌ | ✅ | ✅ |
| Best Window | ❌ | ✅ | ✅ |
| Live Cost | ❌ | ✅ | ✅ |
| Tomorrow preview | ❌ | ✅ | ✅ |
| **EU landen** | ❌ | ❌ | ✅ |
| Enever BYO-key | ✅ | ✅ | ✅ |

---

## KEY ARCHITECTURE COMPONENTS

### 1. API Server (`synctacles-api`)
**Location:** `/opt/github/synctacles-api`
**Tech Stack:** FastAPI, PostgreSQL, Gunicorn
**Purpose:** Price data aggregation & distribution

```
synctacles_db/
├── api/endpoints/
│   ├── windows.py          # Dashboard endpoint
│   ├── prices.py           # Price data
│   └── energy_action.py    # GO/WAIT/AVOID
├── fallback/
│   └── fallback_manager.py # 6-tier fallback stack
├── clients/
│   ├── frank_energie_client.py
│   └── easyenergy_client.py
└── services/
    └── price_cache.py      # PostgreSQL caching
```

### 2. HA Integration (`ha-integration`)
**Location:** `/opt/github/ha-integration`
**Tech Stack:** Python, aiohttp
**Purpose:** Home Assistant sensors & coordinators

```
custom_components/ha_energy_insights_nl/
├── __init__.py             # Coordinators
├── sensor.py               # 10+ sensors
├── config_flow.py          # Setup wizard
└── enever_client.py        # Enever BYO API
```

---

## 6-TIER FALLBACK STACK

| Tier | Source | Type | GO Action | Accuracy |
|------|--------|------|-----------|----------|
| 1 | Frank DB (PostgreSQL) | Consumer | ✅ | 100% |
| 2 | Frank Direct API | Consumer | ✅ | 100% |
| 3 | EasyEnergy Direct API | Wholesale | ✅ | 100% |
| 4 | ENTSO-E + Static Offset | Calculated | ✅ | 85-89% |
| 5 | Energy-Charts + Offset | Fallback | ❌ | 85-89% |
| 6 | Cache (Memory + PostgreSQL) | Stale | ❌ | - |

**Critical Rule:** GO actions only allowed on Tiers 1-4 (reliable consumer data).

---

## PRIMARY ENDPOINTS

| Endpoint | Method | Purpose | Cache TTL |
|----------|--------|---------|-----------|
| `/api/v1/dashboard` | GET | Bundled data (primary) | 2 min |
| `/api/v1/prices` | GET | Raw price data | 5 min |
| `/api/v1/best-window` | GET | Best consumption window | 5 min |
| `/api/v1/tomorrow` | GET | Tomorrow preview | 5 min |
| `/api/v1/energy-action` | GET | GO/WAIT/AVOID | 2 min |
| `/health` | GET | System health | - |

---

## HOME ASSISTANT SENSORS

### Server Data Sensors
| Sensor | Entity ID | Updates |
|--------|-----------|---------|
| Price Current | `sensor.energy_insights_nl_electricity_price` | 15 min |
| Cheapest Hour | `sensor.energy_insights_nl_cheapest_hour` | 15 min |
| Most Expensive Hour | `sensor.energy_insights_nl_most_expensive_hour` | 15 min |
| Energy Action | `sensor.energy_insights_nl_energy_action` | 15 min |
| Best Window | `sensor.energy_insights_nl_best_window_*` | 15 min |
| Tomorrow Preview | `sensor.energy_insights_nl_tomorrow_preview` | 15 min |
| Live Cost | `sensor.energy_insights_nl_live_cost` | Real-time |
| Savings | `sensor.energy_insights_nl_savings_*` | 15 min |

### Enever BYO Sensors
| Sensor | Entity ID | Updates |
|--------|-----------|---------|
| Prices Today | `sensor.energy_insights_nl_prices_today_*` | 1 hour |
| Prices Tomorrow | `sensor.energy_insights_nl_prices_tomorrow_*` | 1 hour |

**Note:** Enever sensors require user's own API key (leverancier dropdown, 24 providers).

---

## ENERGY ACTION LOGIC

### GO (price < average - 5%)
- **Icon:** mdi:play-circle (green)
- **Meaning:** Electricity is cheap, run flexible loads now
- **Example:** Start washing machine, charge EV, heat water boiler

### WAIT (price within ±5% of average)
- **Icon:** mdi:pause-circle (yellow)
- **Meaning:** Prices are average, wait for cheaper period
- **Example:** Delay non-urgent loads

### AVOID (price > average + 5%)
- **Icon:** mdi:stop-circle (red)
- **Meaning:** Electricity is expensive, avoid usage
- **Example:** Don't charge EV, postpone heavy loads

---

## ENEVER PROVIDERS (24)

| Provider | API Code | Resolution |
|----------|----------|------------|
| ANWB Energie | prijsANWB | 60 min |
| Budget Energie | prijsBE | 60 min |
| Coolblue Energie | prijsCB | 60 min |
| EasyEnergy | prijsEE | 60 min |
| Energiedirect | prijsED | 60 min |
| Energie van Ons | prijsEVO | 60 min |
| Energiek | prijsEG | 60 min |
| EnergyZero | prijsEZ | 15 min |
| Essent | prijsES | 60 min |
| Frank Energie | prijsFR | 60 min |
| Groenestroom Lokaal | prijsGSL | 60 min |
| Hegg Energy | prijsHE | 60 min |
| Innova Energie | prijsIN | 60 min |
| Mijndomein Energie | prijsMDE | 60 min |
| NextEnergy | prijsNE | 60 min |
| Pure Energie | prijsPE | 60 min |
| Quatt | prijsQU | 60 min |
| SamSam | prijsSS | 60 min |
| Tibber | prijsTI | 60 min |
| Vandebron | prijsVDB | 60 min |
| Vattenfall | prijsVF | 60 min |
| Vrij op naam | prijsVON | 60 min |
| Wout Energie | prijsWE | 60 min |
| Zonneplan | prijsZP | 60 min |

**Note:** Eneco (prijsEN) excluded - only offers dynamic GAS, not electricity.

---

## COORDINATORS

### ServerDataCoordinator
- **Endpoint:** `/api/v1/dashboard`
- **Interval:** 15 minutes
- **Fallback:** `/api/v1/prices` (if dashboard fails)
- **Cache:** 30 minutes max age

### EneverDataCoordinator
- **Endpoint:** `https://enever.nl/apiv3/stroomprijs_*.php`
- **Interval:** 1 hour
- **Smart Caching:**
  - Fetch tomorrow @15:00 (when available)
  - Promote tomorrow → today @midnight
  - Reduces API calls ~50% (31/month vs 62)

---

## LIVE COST SENSOR

Real-time energy cost calculation from P1 meter data:

```
Cost (€/h) = Power (kW) × Price (€/kWh)
```

**Requirements:**
- P1 meter integration in HA
- Entity with `device_class: power` and `unit_of_measurement: W`

**Updates:** Real-time (follows P1 sensor updates)

---

## DATABASE TABLES

### frank_prices
Consumer prices from Frank Energie (Tier 1):
```sql
CREATE TABLE frank_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    price_eur_kwh NUMERIC(10, 6) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### norm_entso_e_a44
ENTSO-E wholesale prices (Tier 4):
```sql
CREATE TABLE norm_entso_e_a44 (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    country VARCHAR(2) NOT NULL,
    price_eur_mwh NUMERIC(10, 4) NOT NULL,
    data_source VARCHAR(50),
    data_quality VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### price_cache
Persistent cache for fallback (Tier 6):
```sql
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

## DISCONTINUED FEATURES

| Feature | Removed | Reason |
|---------|---------|--------|
| TenneT integration | Phase 3 | BYO-key in HA only |
| A65 (load) endpoints | Phase 3 | Energy Action focus |
| A75 (generation) endpoints | Phase 3 | Energy Action focus |
| Grid stress sensors | Phase 3 | Simplified to GO/WAIT/AVOID |
| Balance sensors | Phase 3 | Not needed for price actions |
| Coefficient server | KISS v2.0.0 | Replaced by static offsets |

---

## RELATED SKILLS

| Skill | Description |
|-------|-------------|
| [SKILL_01_ARCHITECTURE.md](SKILL_01_ARCHITECTURE.md) | Technical architecture deep-dive |
| [SKILL_02_PRODUCT.md](SKILL_02_PRODUCT.md) | Product requirements & features |
| [SKILL_03_DATA_SOURCES.md](SKILL_03_DATA_SOURCES.md) | Data providers & APIs |
| [SKILL_04_PRICE_ENGINE.md](SKILL_04_PRICE_ENGINE.md) | Consumer price calculation |
| [../business/SKILL_00_GO_TO_MARKET.md](../business/SKILL_00_GO_TO_MARKET.md) | Business model & pricing |

---

*Generated: 2026-01-26*
