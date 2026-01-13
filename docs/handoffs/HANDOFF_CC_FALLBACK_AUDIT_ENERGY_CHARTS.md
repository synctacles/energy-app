# FALLBACK CHAIN AUDIT & ENERGY-CHARTS VALIDATIE

**Datum:** 2026-01-13 02:50 UTC  
**Server:** Synctacles (77.169.67.130) + Coefficient (91.99.150.36)  
**Auditor:** Claude Code (CC)

---

## EXECUTIVE SUMMARY

| Criterium | Resultaat | Status |
|-----------|-----------|--------|
| Energy-Charts vs ENTSO-E data | **0.00% verschil** | ✅ IDENTIEK |
| Coefficient accuracy (vs Frank) | **3.50% avg error** | ✅ ACCEPTABLE |
| API reliability | **100% (10/10)** | ✅ RELIABLE |
| Response time | **333ms avg** | ✅ FAST |

**AANBEVELING: ✅ IMPLEMENTEREN** - Energy-Charts is geschikt als Tier 5 fallback.

---

## DEEL 1: ARCHITECTUUR AUDIT

### 1. Services Inventarisatie

#### Synctacles Server (77.169.67.130)
```
Geen specifieke frank/enever/consumer systemd services
Collectors draaien via Python scripts in synctacles_db/collectors/
```

#### Coefficient Server (91.99.150.36)
```
SERVICES:
- coefficient-api.service         → RUNNING (FastAPI server)
- coefficient-calibration.service → via timer (6-hourly)
- consumer-collector.service      → via timer (daily 15:30)
- enever-collector.service        → via timer (daily 15:30)
- frank-live-collector.service    → via timer (daily 15:00)

TIMERS:
- coefficient-calibration.timer   → Next: 06:16 UTC
- frank-live-collector.timer      → Last: 01:00, Next: 15:00
- consumer-collector.timer        → Last: 00:30, Next: 15:30
- enever-collector.timer          → Last: 00:30, Next: 15:30
```

#### Collector Python Files (Coefficient Server)
```
/opt/github/coefficient-engine/collectors/
├── daily_consumer_collector.py
├── frank_live_collector.py
├── enever_collector.py
└── consumer_price_collector.py

/opt/github/coefficient-engine/routes/
└── consumer.py
```

### 2. Database Schema

#### Synctacles Database (energy_insights_nl)
| Tabel | Size | Beschrijving |
|-------|------|--------------|
| price_cache | 632 kB | Cached consumer prices (Frank) |
| norm_prices | 24 kB | Normalized prices |
| raw_prices | 80 kB | Raw import data |

**price_cache schema:**
```sql
id            SERIAL PRIMARY KEY
timestamp     TIMESTAMPTZ NOT NULL
country       VARCHAR(2) DEFAULT 'NL'
price_eur_kwh NUMERIC(10,6)
source        VARCHAR(50)      -- 'frank-energie', 'entsoe'
quality       VARCHAR(20)      -- 'live', 'estimated', 'cached'
created_at    TIMESTAMPTZ DEFAULT now()
```

**price_cache sources:**
| Source | Quality | Records | Latest |
|--------|---------|---------|--------|
| frank-energie | live | 2904 | 2026-01-13 02:37 UTC |
| entsoe | live | 1296 | 2026-01-12 00:14 UTC |

#### Coefficient Database (coefficient_db)
| Tabel | Size | Beschrijving |
|-------|------|--------------|
| hist_enever_prices | 65 MB | Historical Enever consumer prices |
| hist_entso_prices | 496 MB | Historical ENTSO-E wholesale prices |
| hist_frank_prices | 4.4 MB | Historical Frank API prices |
| hist_frank_live_prices | 56 kB | Live Frank prices via VPN |
| coefficient_history | 64 kB | Coefficient change history |

**Data freshness (last 7 days):**
| Tabel | Records | Latest |
|-------|---------|--------|
| hist_enever_prices | 2,985 | 2026-01-13 23:00 UTC |
| hist_frank_prices | 188 | 2026-01-13 22:00 UTC |

### 3. Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SYNCTACLES DATA ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐     ┌──────────────────┐     ┌─────────────────────────┐  │
│  │ Frank API   │────▶│ Coefficient Srv  │────▶│ SYNCTACLES API         │  │
│  │ (GraphQL)   │     │ (VPN proxy)      │     │ - ConsumerPriceClient  │  │
│  └─────────────┘     │ - consumer.py    │     │ - FallbackManager      │  │
│         │            │ - coefficient/   │     └─────────────────────────┘  │
│         │            └──────────────────┘              │                   │
│         │                    │                         │                   │
│         ▼                    ▼                         ▼                   │
│  ┌─────────────┐     ┌──────────────────┐     ┌─────────────────────────┐  │
│  │ Enever API  │────▶│ Coefficient DB   │     │ SYNCTACLES DB          │  │
│  │ (scraping)  │     │ - hist_enever    │────▶│ - price_cache          │  │
│  └─────────────┘     │ - hist_entso     │     │ - norm_entso_e_a44     │  │
│                      │ - coefficient_   │     └─────────────────────────┘  │
│                      │   lookup (576)   │                                  │
│                      └──────────────────┘                                  │
│                                                                             │
│  ┌─────────────┐                              ┌─────────────────────────┐  │
│  │ ENTSO-E API │─────────────────────────────▶│ SYNCTACLES DB          │  │
│  │ (direct)    │                              │ - norm_entso_e_a44     │  │
│  └─────────────┘                              └─────────────────────────┘  │
│                                                                             │
│  ┌─────────────┐                              ┌─────────────────────────┐  │
│  │ Energy-     │     *** TIER 5 FALLBACK ***  │ SYNCTACLES API         │  │
│  │ Charts API  │─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ▶│ (proposed)             │  │
│  └─────────────┘                              └─────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4. Frank Energie IP Blocking Test

| Test | Result | Details |
|------|--------|---------|
| Direct (Synctacles IP) | ✅ HTTP 200 | **NIET GEBLOKKEERD** |
| Via VPN (Coefficient) | ✅ HTTP 200 | Werkt zoals verwacht |

**Bevinding:** Frank Energie blokkeert momenteel NIET op IP. VPN is vooralsnog niet vereist, maar blijft beschikbaar als backup.

---

## DEEL 2: ENERGY-CHARTS VALIDATIE

### 1. API Beschikbaarheid

**Endpoint:** `https://api.energy-charts.info/price?bzn=NL`

| Metric | Waarde |
|--------|--------|
| Data points | 96 (24h × 4 quarters) |
| Unit | EUR/MWh |
| Update frequency | Real-time |
| First record | 2026-01-12 23:00 UTC |
| Last record | 2026-01-13 22:45 UTC |

### 2. Data Kwaliteit vs ENTSO-E

| Hour | Energy-Charts | ENTSO-E | Verschil |
|------|---------------|---------|----------|
| 00:00 | €80.28/MWh | €80.28/MWh | 0.00% |
| 01:00 | €80.50/MWh | €80.50/MWh | 0.00% |
| ... | ... | ... | ... |
| 23:00 | €82.41/MWh | €82.41/MWh | 0.00% |

**Samenvatting:**
| Metric | Waarde |
|--------|--------|
| Matching hours | 24 |
| Average difference | **0.00%** |
| Maximum difference | **0.00%** |

**Conclusie:** Energy-Charts en ENTSO-E data zijn **IDENTIEK** - beide halen van dezelfde ENTSO-E Transparency Platform.

### 3. Coefficient Accuracy Test

**Formule:** `consumer = (wholesale × 1.2725 + 0.1500) × 0.93`

| Hour | Wholesale | Calculated | Frank Actual | Error |
|------|-----------|------------|--------------|-------|
| 00:00 | €0.0803 | €0.2345 | €0.2248 | +4.32% |
| 07:00 | €0.1190 | €0.2803 | €0.2490 | +12.60% |
| 10:00 | €0.1211 | €0.2828 | €0.2845 | -0.59% |
| 17:00 | €0.0979 | €0.2553 | €0.2593 | -1.54% |
| 22:00 | €0.0824 | €0.2370 | €0.2371 | -0.05% |

**Samenvatting:**
| Metric | Waarde |
|--------|--------|
| Comparisons | 24 |
| Average error | **3.50%** |
| Maximum error | **12.60%** (07:00 - morning peak) |
| Minimum error | **0.03%** |

**Status:** ✅ ACCEPTABLE (avg < 5%)

### 4. ENTSO-E vs Energy-Charts Performance

| Source | Avg Error | Max Error | Min Error |
|--------|-----------|-----------|-----------|
| ENTSO-E + Model | 3.50% | 12.61% | 0.03% |
| Energy-Charts + Model | 3.50% | 12.61% | 0.03% |

**Conclusie:** **GELIJKWAARDIG** - identieke performance (zelfde brondata).

### 5. API Betrouwbaarheid

| Test | HTTP Code | Points | Response Time |
|------|-----------|--------|---------------|
| 1 | 200 | 96 | 322ms |
| 2 | 200 | 96 | 314ms |
| 3 | 200 | 96 | 369ms |
| 4 | 200 | 96 | 272ms |
| 5 | 200 | 96 | 348ms |
| 6 | 200 | 96 | 340ms |
| 7 | 200 | 96 | 308ms |
| 8 | 200 | 96 | 319ms |
| 9 | 200 | 96 | 312ms |
| 10 | 200 | 96 | 427ms |

**Samenvatting:**
| Metric | Waarde |
|--------|--------|
| Success rate | **100% (10/10)** |
| Average response time | **333ms** |
| Max response time | 427ms |

**Status:** ✅ RELIABLE

---

## CONCLUSIES

### Energy-Charts als Tier 5

| Criterium | Drempel | Resultaat | Status |
|-----------|---------|-----------|--------|
| Data kwaliteit vs ENTSO-E | <2% | **0.00%** | ✅ PASS |
| Coefficient accuracy | <5% avg | **3.50%** | ✅ PASS |
| API reliability | >95% | **100%** | ✅ PASS |

### ✅ AANBEVELING: IMPLEMENTEREN

Energy-Charts is geschikt als Tier 5 fallback vanwege:

1. **Identieke data** als ENTSO-E (0% verschil)
2. **Goede accuracy** met coefficient model (3.50% avg error)
3. **Betrouwbare API** (100% uptime in test, 333ms response)
4. **Geen authenticatie** vereist (publieke API)
5. **Alternatieve bron** bij ENTSO-E outage

### Aanbevolen Tier Structuur (Updated)

| Tier | Source | Data Type | Quality |
|------|--------|-----------|---------|
| 1 | Frank Live (via Coefficient VPN) | Consumer prices | live (100%) |
| 2 | Consumer Proxy (Tibber/Vattenfall) | Consumer prices | live (99%) |
| 3 | ENTSO-E Fresh + Coefficient | Calculated consumer | estimated (90%) |
| 4 | ENTSO-E Stale + Coefficient | Calculated consumer | estimated (85%) |
| **5** | **Energy-Charts + Coefficient** | **Calculated consumer** | **estimated (80%)** |
| 6 | PostgreSQL Cache | Cached prices | cached (50%) |

### Implementatie Notities

**Energy-Charts endpoint:**
```python
# API call
GET https://api.energy-charts.info/price?bzn=NL

# Response format
{
    "unix_seconds": [1736722800, 1736723700, ...],  # 15-min intervals
    "price": [80.28, 79.55, ...],                    # EUR/MWh
    "unit": "EUR / MWh"
}

# Convert to hourly
for i, ts in enumerate(data['unix_seconds']):
    hour = datetime.fromtimestamp(ts, tz=timezone.utc).replace(minute=0)
    wholesale_kwh = data['price'][i] / 1000
    consumer_kwh = (wholesale_kwh * slope + intercept) * 0.93
```

**Bestaande client:** `synctacles_db/fallback/energy_charts_client.py` moet uitgebreid worden met `fetch_prices()` method.

---

## BIJLAGEN

### A. Database Queries Used

```sql
-- Energy-Charts vs ENTSO-E vergelijking
SELECT DATE_TRUNC('hour', timestamp), AVG(price_eur_mwh)
FROM norm_entso_e_a44 GROUP BY 1;

-- Frank prices uit cache
SELECT timestamp, price_eur_kwh FROM price_cache
WHERE source = 'frank-energie';
```

### B. Coefficient Parameters

```
Hour 2 (weekday):
- slope: 1.2725
- intercept: 0.1500
- bias: 0.93

Formula: consumer = (wholesale × slope + intercept) × bias
```

### C. Test Environment

```
Synctacles: 77.169.67.130 (Ubuntu 24.04)
Coefficient: 91.99.150.36 (Ubuntu 24.04)
PostgreSQL: 16.x
Python: 3.12
```

---

**Rapport voltooid:** 2026-01-13 03:10 UTC  
**Auditor:** Claude Code (CC)  
**Status:** Alle tests succesvol, Energy-Charts aanbevolen voor Tier 5
