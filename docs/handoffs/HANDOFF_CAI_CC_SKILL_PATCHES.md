# HANDOFF: CAI → CC | SKILL Documentation Patches

**Datum:** 2026-01-11
**Van:** Claude AI (Anthropic)
**Naar:** Claude Code
**Status:** READY FOR IMPLEMENTATION

---

## OPDRACHT

Pas de volgende documentatie updates toe op de SYNCTACLES repo SKILL bestanden.

**Repo locatie:** `/opt/github/synctacles-repo/` (of waar SKILL bestanden staan)

---

## 1. NIEUW BESTAND: SKILL_15_CONSUMER_PRICE_ENGINE.md

**Actie:** Maak nieuw bestand aan

**Bron:** Zie bijgevoegd `SKILL_15_CONSUMER_PRICE_ENGINE.md`

**Locatie:** Zelfde directory als andere SKILL bestanden

---

## 2. UPDATE: SKILL_06_DATA_SOURCES.md

### 2A. Voeg toe NA regel ~244 (na "### 4. Enever.nl" sectie)

```markdown
### 5. Frank Energie API (Server-Side)

**Website:** https://frankenergie.nl/

**What They Provide:**
- Real-time consumer electricity prices for Frank Energie customers
- Full price breakdown (wholesale, tax, margin)
- Day-ahead prices

**Access Method:** GraphQL API (no authentication required)

**SYNCTACLES Integration:**
- ✅ **Available via SYNCTACLES API** (server-side, free)
- Used for daily coefficient calibration
- Primary source for consumer price validation
- Ground truth for dual-source verification

**API Endpoint:**
```
POST https://graphcdn.frankenergie.nl/
Content-Type: application/json

{
  "query": "query MarketPrices { marketPrices(startDate: \"2026-01-11\", endDate: \"2026-01-11\") { electricityPrices { from till marketPrice marketPriceTax sourcingMarkupPrice energyTaxPrice } } }"
}
```

**Response Breakdown:**
```json
{
  "electricityPrices": [{
    "from": "2026-01-11T00:00:00+01:00",
    "till": "2026-01-11T01:00:00+01:00",
    "marketPrice": 0.04532,
    "marketPriceTax": 0.00952,
    "sourcingMarkupPrice": 0.0389,
    "energyTaxPrice": 0.09854
  }]
}
```

**Total Price Calculation:**
```
consumer_price = marketPrice + marketPriceTax + sourcingMarkupPrice + energyTaxPrice
```

**Frank API Details:**
- **Base URL:** https://graphcdn.frankenergie.nl/
- **Authentication:** None required
- **Rate Limit:** Not documented, appears unlimited
- **Response Format:** JSON (GraphQL)
- **Response Time:** ~250ms
- **Timeout:** 10 seconds recommended

**Reliability:**
- Highly reliable (public endpoint)
- No auth failures possible
- Used as primary calibration source

**Cost:** Free (public API)

**Update Schedule:**
- SYNCTACLES fetches daily at 15:05
- Tomorrow prices available after ~15:00

**Use in SYNCTACLES:**
- Daily coefficient calibration
- Dual-source validation with Enever
- Consumer price calculation baseline

See SKILL_15_CONSUMER_PRICE_ENGINE.md for full integration details.
```

### 2B. Vervang "### Price Data Fallback" sectie (rond regel 418)

**Zoek:** `### Price Data Fallback`

**Vervang hele sectie met:**

```markdown
### Price Data Fallback

```
Try in order:
  1. Frank API + Enever (dual-source validation)
     - Both agree (< €0.02 delta)
     - Quality: HIGH (98% accuracy)

  2. Frank API only
     - Primary source available
     - Quality: MEDIUM (95% accuracy)

  3. Enever API only (via coefficient proxy)
     - Secondary source available
     - Quality: MEDIUM (93% accuracy)

  4. Cached correction factor (< 48h)
     - Last known calibration
     - Quality: LOW (91% accuracy)

  5. Coefficient lookup only
     - Historical hourly patterns
     - Quality: LOW (89% accuracy)

  6. Daily average fallback
     - Emergency: €0.17/kWh coefficient
     - Quality: DEGRADED (85% accuracy)
```
```

### 2C. Voeg toe in "## FUTURE DATA SOURCES" sectie

**Zoek:** `### Planned (Phase 7-9)`

**Voeg toe VOOR die sectie:**

```markdown
### Implemented (2026-01)

- **Frank Energie API** - Server-side consumer price calibration
- **Enever Proxy** - Dual-source validation via coefficient server
- **Historical Enever Data** - 25 providers, hourly granularity, growing dataset

```

---

## 3. UPDATE: SKILL_02_ARCHITECTURE.md

### 3A. Voeg toe NA "### Layer 4: API" sectie (rond regel 360)

```markdown
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
```

### 3B. Update "### Component Overview" diagram (rond regel 98)

**Zoek:** Het bestaande diagram dat begint met `EXTERNAL SOURCES`

**Voeg toe aan EXTERNAL SOURCES lijst:**
```
├── Frank Energie API (Consumer prices)
├── Enever API (via Coefficient Proxy)
```

**Voeg toe NA `LAYER 1-4` in diagram:**
```
     │
     ▼
LAYER 5: CONSUMER PRICE ENGINE
├── Frank calibration (daily 15:05)
├── Enever validation (via proxy)
├── Dual-source verification
└── Coefficient fallback
```

### 3C. Voeg toe in "## DATABASE SCHEMA" sectie (rond regel 680)

```markdown
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
```

---

## 4. UPDATE: SKILL_10_COEFFICIENT_VPN.md

### 4A. Vervang PURPOSE sectie (regel 8-15)

**Zoek:** `## PURPOSE`

**Vervang met:**

```markdown
## PURPOSE

This skill documents the VPN configuration and Enever proxy functionality for the coefficient engine server. The server requires a Dutch IP address to access Enever consumer price data while maintaining SSH accessibility and normal service operation.

**Server Roles:**
1. WireGuard VPN split tunnel for Dutch IP
2. Enever API proxy endpoint for main API server
3. Historical Enever data collection and storage

**Related:**
- SKILL_15: Consumer Price Engine (how prices are calculated)
- ADR-002: VPN Split Tunneling decision rationale
- HANDOFF_CC_CAI_ENEVER_IMPLEMENTATION_COMPLETE.md: Implementation details
```

### 4B. Voeg toe NA "## OVERVIEW" sectie (rond regel 70)

```markdown
## ENEVER PROXY FUNCTIONALITY

### Architecture

```
Main API (135.181.255.83)
│
└── GET /internal/enever/prices
            │
            ▼
    Coefficient Server (91.99.150.36)
    ├── FastAPI Proxy (port 8080)
    │   └── IP whitelist: 135.181.255.83
    │
    ├── VPN Split Tunnel
    │   └── Routes 84.46.252.107 → pia-split
    │
    └── Enever API
        └── https://enever.nl/api/
```

### Proxy Endpoint

**URL:** `http://91.99.150.36:8080/internal/enever/prices`

**Security:** IP whitelist (only main API server)

**Response:**
```json
{
  "timestamp": "2026-01-11T15:30:00Z",
  "source": "enever",
  "prices_today": {
    "Frank Energie": [
      {"hour": 0, "price": 0.2463},
      {"hour": 1, "price": 0.2412}
    ],
    "Tibber": [...]
  },
  "prices_tomorrow": null
}
```

**Health Endpoint:** `GET /internal/enever/health`

### Data Collection

**Schedule:** 2x daily via systemd timer
- 00:30 UTC — Fetch today's final prices
- 15:30 UTC — Fetch today + tomorrow

**Storage:** PostgreSQL on coefficient server

```sql
CREATE TABLE enever_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    provider VARCHAR(50) NOT NULL,
    hour INTEGER NOT NULL,
    price_total DECIMAL(8,5) NOT NULL,
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(timestamp, provider, hour)
);
```

**Volume:** ~1200 records/day (25 providers × 24 hours × 2 collections)

### SystemD Services

**coefficient-api.service** — FastAPI proxy
```bash
sudo systemctl status coefficient-api.service
# Port 8080, auto-restart
```

**enever-collector.timer** — Data collection
```bash
sudo systemctl status enever-collector.timer
# Runs at 00:30 and 15:30 UTC
```

### Files

```
/opt/github/coefficient-engine/
├── api/main.py
├── routes/enever.py
├── services/enever_client.py
└── collectors/enever_collector.py

/opt/coefficient/
├── .enever_token
└── .env

/etc/systemd/system/
├── coefficient-api.service
├── enever-collector.service
└── enever-collector.timer
```

### Verification

```bash
# Test proxy from main API
ssh root@135.181.255.83 'curl -s http://91.99.150.36:8080/internal/enever/prices | jq ".prices_today | keys | length"'
# Expected: 25

# Check collector logs
ssh coefficient@91.99.150.36 'sudo journalctl -u enever-collector.service -n 20'

# Verify database records
ssh coefficient@91.99.150.36 'psql -U coefficient -d coefficient -c "SELECT COUNT(*) FROM enever_prices WHERE collected_at > NOW() - INTERVAL '\''1 day'\'';"'
# Expected: 1200+
```
```

### 4C. Voeg toe aan CHANGELOG (einde document)

```markdown
- **2026-01-11 v1.1**: Added Enever proxy functionality (Claude Code)
  - Documented proxy endpoint architecture
  - Added data collection schedule
  - Included SystemD services documentation
  - Added verification commands
```

---

## 5. NIEUW BESTAND: SESSIE_SAMENVATTING_20260111_CONSUMER_PRICE_COMPLETE.md

**Actie:** Maak nieuw bestand aan

**Bron:** Zie bijgevoegd bestand

---

## VERIFICATIE

Na toepassen, bevestig:

1. SKILL_15 bestaat en is compleet
2. SKILL_06 bevat Frank API sectie
3. SKILL_02 bevat Layer 5 sectie
4. SKILL_10 bevat Enever proxy sectie
5. Sessie samenvatting is toegevoegd

---

## RAPPORTAGE

Lever na afronding:
- Lijst van gewijzigde bestanden
- Git commit hash
- Eventuele afwijkingen

---

**Ready for implementation.**
