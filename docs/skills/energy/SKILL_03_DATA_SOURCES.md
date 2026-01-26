# SKILL ENERGY 03 — DATA SOURCES

ENTSO-E A44 Prices, Energy-Charts Fallback, and Consumer Price Engine
Version: 2.1 (2026-01-22) - Hybrid Conversion + EasyEnergy Fallback

> **Phase 3 Update:** SYNCTACLES now focuses exclusively on Energy Action (price-based
> recommendations). A65 (load), A75 (generation), and TenneT integration have been
> discontinued. Only A44 (day-ahead prices) data is collected.

---

## PURPOSE

Document the external data sources that feed SYNCTACLES: price data sources, how to access them, rate limits, reliability expectations, and fallback strategies when they fail.

---

## PRIMARY SOURCES

2 Primary Sources for Dutch Energy Prices (Energy Action Focus)

### 1. ENTSO-E (European Network of Transmission System Operators)

**Website:** https://www.entsoe.eu/

**What They Provide:**
- Pan-European electricity data
- Day-ahead prices (A44 document type)
- Published as XML via REST API

**Access Method:** OAuth2 API with security token

> **Phase 3 Note:** Only A44 (prices) is now collected. A65 (load) and A75 (generation)
> document types have been discontinued for Energy Action Focus.

**Key Document Type:**

#### A44: Day-Ahead Prices (ACTIVE)
- **Update Frequency:** Hourly (updated ~12:42 CET for next day)
- **Delay:** Published day-ahead
- **Data Points:**
  - Market price per hour (€/MWh)
  - Also published intraday for same-day updates

#### A46: Intraday Prices
- **Update Frequency:** Continuous
- **Delay:** Real-time
- **Data Points:**
  - Intraday auction prices

**ENTSO-E API Details:**

- **Base URL:** https://web-api.tp.entsoe.eu/api
- **Authentication:** Security token in query parameter
- **Rate Limit:** 400 requests per minute
- **Response Format:** XML
- **Timeout:** 30 seconds (connection + response)

**Reliability:**
- Generally reliable (99%+ availability)
- Occasional 5-10 minute data delays during peak hours
- Planned maintenance ~monthly (announced)

**Cost:** Free (public API)

**Netherlands Code:**
```
In_Domain: 10YNL----------L (consumption)
Out_Domain: 10YNL----------L (generation)
```

---

### 2. TenneT (Dutch Transmission System Operator) - DISCONTINUED

> **Phase 3 (2026-01-11):** TenneT integration has been fully discontinued from SYNCTACLES.
> This includes both server-side and BYO-key integrations in the HA component.

**Historical Note:** TenneT provided Dutch grid-specific data including grid balance delta,
frequency, and reserve margins. The integration was removed as part of the Energy Action
Focus initiative to simplify the system to price-based recommendations only.

**For users requiring TenneT data:** Use the official TenneT API directly or alternative
Home Assistant integrations such as `tennet-balance` or similar community integrations.

---

### 4. Enever.nl (Dutch Energy Pricing - BYO-KEY)

⚠️ **LICENSE NOTICE:** Enever.nl data via BYO-key in Home Assistant component only.

**Website:** https://enever.nl/

**What They Provide:**
- Leverancier-specific electricity prices (not just ENTSO-E wholesale)
- Real consumer prices including taxes, markup, delivery costs
- Day-ahead prices (today + tomorrow)
- Support for 24 Dutch energy suppliers

**Supported Leveranciers (24):**
ANWB Energie, Budget Energie, Coolblue Energie, EasyEnergy, Energiedirect,
Energie van Ons, Energiek, EnergyZero, Essent, Frank Energie, Groenestroom Lokaal,
Hegg Energy, Innova Energie, Mijndomein Energie, NextEnergy, Pure Energie, Quatt,
SamSam, Tibber, Vandebron, Vattenfall, Vrij op naam, Wout Energie, Zonneplan

**Note:** Eneco is NOT included — they only offer dynamic gas, not electricity.

**Access Method:** API token (user registers at enever.nl)

**SYNCTACLES Integration:**
- ❌ **NOT available via SYNCTACLES API** (BYO-key only)
- ✅ **Available via Home Assistant component** with user's Enever token
- User registers at https://enever.nl/
- User enters token + selects leverancier in HA config
- Data fetched locally in Home Assistant

**API Details:**
- **Update interval:** 1 hour
- **Smart caching:** ~31 API calls/month (vs ~62 without caching)
- **Resolution:** 60-min default, 15-min for supporters + compatible suppliers
- **Tomorrow prices:** Available after 15:00

**Data Points:**
- Hourly prices today (24 values)
- Hourly prices tomorrow (24 values, after 15:00)
- Price includes all leverancier-specific costs

**Reliability:**
- Dependent on enever.nl uptime
- Fallback to ENTSO-E server prices if unavailable

**Cost:** Free tier available, supporter tier for 15-min resolution

---

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

See SKILL_04_PRICE_ENGINE.md for full integration details.

---

### 3. Energy-Charts (Fraunhofer ISE Model)

**Website:** https://energy-charts.info/

**What They Provide:**
- Modeled (not measured) generation data
- Covers all European countries
- Used as fallback when real-time data unavailable

**Access Method:** HTTP API (no authentication)

**Key Data Points:**

#### Estimated Generation
- Solar estimate (MW)
- Wind estimate (MW)
- Hydro estimate (MW)
- By country (including Netherlands)

**API Endpoint:**
```
GET https://api.energy-charts.info/public_power
  ?country=nl  # Country code (de=Germany, nl=Netherlands)
```

**Example Response (JSON):**
```json
{
  "unix_seconds": [1704024000, 1704024900, ...],
  "production_types": [
    {"name": "solar", "data": [420, 425, ...]},
    {"name": "wind_onshore", "data": [1200, 1210, ...]},
    {"name": "wind_offshore", "data": [600, 610, ...]},
    ...
  ]
}
```

**Energy-Charts Details (Measured):**

- **Base URL:** https://api.energy-charts.info/
- **Authentication:** None (public, rate-limited)
- **Rate Limit:** 10 requests per minute (for free tier)
- **Response Format:** JSON
- **Response Time:** ~0.33 seconds (fast)
- **Timeout:** 30 seconds
- **Data Freshness:** ~3+ hours behind current time (modeled estimate)
- **Update Frequency:** Updated regularly but with significant delay
- **Accuracy:** ±5-10% for generation estimation

**Real-World Measurements (2025-12-30):**
- Response time: 330ms
- Data age: ~187 minutes old
- Status: Reliable, but data is significantly delayed

**Reliability:**
- Highly reliable (model doesn't fail)
- Used as fallback specifically for reliability
- Data is modeled, not measured (lower accuracy than ENTSO-E)
- Circuit breaker: Skips for 2h after HTTP 404 to respect rate limits

**Cost:** Free (public API)

---

## FALLBACK STRATEGY

Automated fallback cascade when primary sources fail or are too stale.

### Freshness Thresholds

| Source | FRESH | STALE | Fallback Trigger | Structural Delay |
|--------|-------|-------|------------------|------------------|
| ENTSO-E | < 90 min | 90-180 min | > 180 min | ~60 min avg |
| Energy-Charts | < 240 min | 240-480 min | > 480 min | ~187 min (3h+) |
| Cache | < 120 min | 120-360 min | > 360 min | Variable |

**Note:** TenneT is no longer available via SYNCTACLES API (BYO-key in HA component). ENTSO-E has structural ~60 minute delay due to upstream data processing. Thresholds account for this plus 30-minute buffer.

> **Phase 3 Note:** Generation (A75) and Load (A65) fallback chains have been discontinued.
> SYNCTACLES now focuses exclusively on Price Data (A44) for Energy Action recommendations.

### Price Data Fallback (6-Tier Chain - 2026-01-22)

```
┌─────────────────────────────────────────────────────────────┐
│              PRICE FALLBACK CHAIN (Consumer Prices)         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Tier 1: Frank DB                 ← Pre-collected           │
│     ↓ fail                           price_type = consumer  │
│     │                                accuracy = 100%        │
│     │                                quality = FRESH/STALE  │
│                                                             │
│  Tier 2: Frank Direct API         ← Live GraphQL call       │
│     ↓ fail                           price_type = consumer  │
│     │                                accuracy = 100%        │
│     │                                quality = FRESH        │
│                                                             │
│  Tier 3: EasyEnergy + Hybrid      ← Wholesale + conversion  │
│     ↓ fail                           price_type = estimate  │
│     │                                accuracy = 100%*       │
│     │                                quality = FRESH        │
│                                                             │
│  Tier 4: ENTSO-E + Hybrid         ← Wholesale + conversion  │
│     ↓ fail                           price_type = estimate  │
│     │                                accuracy = 100%*       │
│     │                                quality = FRESH/STALE  │
│                                                             │
│  Tier 5: Energy-Charts + Hybrid   ← Wholesale + conversion  │
│     ↓ fail                           price_type = estimate  │
│     │                                accuracy = 100%*       │
│     │                                quality = FRESH        │
│                                                             │
│  Tier 6: Cache                    ← PostgreSQL 24h          │
│     ↓ fail                           price_type = varies    │
│     │                                accuracy = varies      │
│     │                                quality = CACHED       │
│                                                             │
│  Tier 7: UNAVAILABLE              ← Return null             │
│                                      quality = UNAVAILABLE  │
│                                                             │
└─────────────────────────────────────────────────────────────┘

* Hybrid conversion: consumer = wholesale × 1.21 + €0.129
  (BTW 21% + energiebelasting €0.111 + sourcing €0.018)
```

**Key Changes (2026-01-22):**
- Frank DB added as Tier 1 (requires `--tomorrow` flag on collector)
- EasyEnergy added as Tier 3 (was missing)
- Hybrid conversion replaces static offset (100% vs 85% accuracy)
- See SKILL_04_PRICE_ENGINE for hybrid formula details

### Energy-Charts Price API (Tier 3)

```
GET https://api.energy-charts.info/price?country=nl
```

**Response:**
```json
{
  "unix_seconds": [1704024000, 1704027600, ...],
  "price": [45.50, 42.30, ...]
}
```

**Characteristics:**
- Returns €/MWh (wholesale)
- 48 hours of data
- Updates ~every hour
- Used as Tier 3 fallback when ENTSO-E unavailable

### PostgreSQL Price Cache (Tier 4b)

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

**Automatic Caching:**
- Prices cached on every successful Tier 1-3 fetch
- Rolling 24h window
- Serves as persistent fallback
- Auto-cleanup via service method

### Fallback Features

**Circuit Breaker:**
- Skips Energy-Charts for 2 hours after HTTP 404 error
- Prevents cascade failures when EC API is down
- Automatically retries after cooldown period

**Field Source Tracking:**
- Generation responses include `_field_sources` showing source of each field
- Allows clients to assess confidence in specific generator types
- Example: `{solar_mw: {source: "Energy-Charts"}, wind_onshore_mw: {source: "ENTSO-E"}}`

**Quality Status in Response:**
```json
{
  "data": {...},
  "meta": {
    "source": "ENTSO-E",
    "quality_status": "FRESH",
    "age_minutes": 58,
    "fallback_used": false,
    "field_sources": {...}
  }
}
```

---

## RATE LIMITS & THROTTLING

### ENTSO-E

- 400 requests/minute → ~1 request per 150ms
- Solution: Batch requests, cache results
- SYNCTACLES strategy: Request all data types per collection cycle

### TenneT

- 100 requests/minute → ~1 request per 600ms
- Solution: Batch requests, cache results

### Energy-Charts

- 10 requests/minute (free tier) → ~1 request per 6 seconds
- Solution: Cache results (we cache daily, acceptable delay)
- Pro tier available (100 req/min for €50/month)

---

## AUTHENTICATION & CREDENTIALS

### ENTSO-E Token

Required environment variable:
```bash
ENTSO_E_TOKEN="your-security-token"
```

Get token from: https://www.entsoe.eu/

**Security:**
- Store in /opt/.env (never git)
- Treat as secret
- Rotate annually

### TenneT & Energy-Charts

No authentication required (public APIs).

---

## ERROR HANDLING

### When ENTSO-E Fails

**Common Causes:**
1. API timeout (server slow)
2. Invalid security token (expired/wrong)
3. Rate limit exceeded (too many requests)
4. Maintenance (announced downtime)

**Collector Response:**
```python
try:
    data = fetch_entso_e_a75()
except requests.Timeout:
    logger.warning("ENTSO-E timeout, will retry in 5 min")
    return None  # Normalizer will use fallback
except UnauthorizedError:
    logger.error("ENTSO-E token invalid. Check ENTSO_E_TOKEN in .env")
    raise  # Critical error, should alert
except RateLimitError:
    logger.warning("ENTSO-E rate limit hit, backing off")
    time.sleep(10)
    return None
```

### When TenneT Fails (HA Component)

TenneT errors are handled locally in the Home Assistant component.
Server-side has no TenneT dependency.

**Common Causes (user's HA):**
1. Invalid/expired personal API key
2. TenneT rate limit exceeded
3. Network issues

**HA Component Response:**
- Sensor shows "unavailable"
- Logs error to HA system log
- Does not affect other SYNCTACLES sensors

### When Energy-Charts Fails

Energy-Charts is used as Tier 5 fallback for price data.
See SKILL_02 for current 6-tier fallback stack.

---

## TESTING & MOCKING

### Mock ENTSO-E Response

```python
@patch('synctacles_db.collectors.entso_e.requests.get')
def test_collector_handles_entso_e_failure(mock_get):
    """Test fallback when ENTSO-E unavailable."""
    mock_get.side_effect = requests.Timeout()

    result = collect_entso_e_data()

    assert result is None  # Collector returns None
    # Normalizer will use fallback
```

### Sample Data Files

```
tests/fixtures/
├── entso_e_a75_valid.xml
├── entso_e_a75_incomplete.xml  # Missing some PSR types
├── entso_e_a75_old.xml          # Timestamp very old
├── tennet_valid.json
├── energy_charts_valid.json
```

---

## MONITORING DATA SOURCES

### Health Check Endpoints

```python
@app.get("/health/sources")
async def source_health():
    """Report status of each data source."""
    return {
        "entso_e_a75": {
            "status": "ok",  # ok, slow, error
            "last_update": "2025-12-30T10:15:00Z",
            "age_minutes": 2,
            "quality": 0.98
        },
        "entso_e_a65": {
            "status": "ok",
            "last_update": "2025-12-30T10:15:00Z",
            "age_minutes": 2,
            "quality": 0.98
        },
        "entso_e_a44": {
            "status": "ok",
            "last_update": "2025-12-30T12:42:00Z",
            "age_hours": 1,
            "quality": 0.99  # Day-ahead prices
        },
        "energy_charts": {
            "status": "ok",
            "last_update": "2025-12-30T00:00:00Z",
            "age_hours": 10,
            "quality": 0.65  # Modeled data
        }
        # TenneT removed - BYO-key only in HA component
    }
```

### Alerting Rules

Set alerts for:
- ENTSO-E unavailable > 30 minutes
- TenneT unavailable > 1 hour
- All sources failed (critical)
- Data quality < 0.5 for > 2 hours

---

## FUTURE DATA SOURCES

### Implemented (2026-01)

- **Frank Energie API** - Server-side consumer price calibration
- **Enever Proxy** - Dual-source validation via coefficient server
- **Historical Enever Data** - 25 providers, hourly granularity, growing dataset

### Planned (Phase 7-9)

- Weather data (for generation forecasting)
- Market data (more granular pricing)
- Weather forecasts (for ML prediction models)
- Regional data (expand beyond Netherlands)

---

## RELATED SKILLS

- **SKILL 2**: Architecture (how data sources integrate)
- **SKILL 3**: Coding Standards (how to fetch data safely)
- **SKILL 4**: Product Requirements (what data users need)
- **SKILL 9**: Installer Specs (environment variable setup)
