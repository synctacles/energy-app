# SKILL 6 — DATA SOURCES

ENTSO-E, TenneT, Energy-Charts, and Fallback Strategy
Version: 1.0 (2025-12-30)

---

## PURPOSE

Document the external data sources that feed SYNCTACLES: what data they provide, how to access them, rate limits, reliability expectations, and fallback strategies when they fail.

---

## PRIMARY SOURCES

### 1. ENTSO-E (European Network of Transmission System Operators)

**Website:** https://www.entsoe.eu/

**What They Provide:**
- Pan-European electricity data
- Real-time generation, load, prices
- Published as XML via REST API

**Access Method:** OAuth2 API with security token

**Key Document Types:**

#### A75: Actual Generation per Type
- **Update Frequency:** Every 15 minutes
- **Delay:** Published ~15 minutes after data time
- **Data Points:**
  - Nuclear generation
  - Fossil fuels (coal, gas, oil)
  - Hydro generation
  - Wind power (onshore + offshore)
  - Solar generation
  - Biomass
  - Waste
  - Other

**Example Request:**
```
GET https://web-api.tp.entsoe.eu/api
  ?securityToken={{ENTSO_E_TOKEN}}
  &documentType=A75
  &processType=A16
  &In_Domain=10YNL----------L  # Netherlands
  &periodStart=202512301000Z
  &periodEnd=202512301015Z
```

**Example Response (XML):**
```xml
<GL_MarketDocument>
  <TimeSeries>
    <Period>
      <timeInterval>
        <start>2025-12-30T10:00Z</start>
        <end>2025-12-30T10:15Z</end>
      </timeInterval>
      <Point>
        <position>1</position>
        <quantity>450</quantity>  <!-- Solar: 450 MW -->
      </Point>
    </Period>
  </TimeSeries>
</GL_MarketDocument>
```

#### A65: Actual Demand
- **Update Frequency:** Every 15 minutes
- **Delay:** Published ~15 minutes after
- **Data Points:**
  - Actual load (MW)
  - Load forecast (MW)

#### A44: Day-Ahead Prices
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

### 2. TenneT (Dutch Transmission System Operator)

**Website:** https://www.tennet.eu/

**What They Provide:**
- Dutch grid-specific data
- Frequency, reserve margins
- Grid stress events

**Access Method:** HTTP API (no authentication required)

**Key Data Points:**

#### Grid Frequency
- Current frequency in Hz (should be ~50 Hz)
- Part of grid stability
- Updated every 5 seconds

#### Reserve Margin
- Spinning reserve (MW)
- Available capacity
- Used to assess grid stress

#### Activation Status
- Normal operation
- Increased reserves activated
- Emergency measures (rare)

**API Endpoint:**
```
GET https://www.tennet.nl/api/grid-data/frequency
GET https://www.tennet.nl/api/grid-data/reserve-margin
```

**Example Response (JSON):**
```json
{
  "timestamp": "2025-12-30T10:15:32Z",
  "frequency_hz": 50.02,
  "reserve_margin_mw": 1500,
  "status": "normal"
}
```

**TenneT API Details:**

- **Base URL:** https://www.tennet.nl/api/
- **Authentication:** None (public)
- **Rate Limit:** 100 requests per minute
- **Response Format:** JSON
- **Timeout:** 10 seconds

**Reliability:**
- Very reliable (99.9%+ availability)
- Updates every 5 minutes
- Minimal delay

**Cost:** Free (public API)

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
| TenneT | < 15 min | 15-30 min | > 30 min | Real-time |
| Energy-Charts | < 240 min | 240-480 min | > 480 min | ~187 min (3h+) |
| Cache | < 120 min | 120-360 min | > 360 min | Variable |

**Note:** ENTSO-E has structural ~60 minute delay due to upstream data processing. Thresholds account for this plus 30-minute buffer.

### Generation Data Fallback

```
Try in order:
  1. ENTSO-E A75 (measured, 15-min)
     - Tier: FRESH (< 90 min) or STALE (90-180 min)
     - Quality: FRESH | STALE
     - Data age: Typically 55-60 minutes

  2. Energy-Charts (modeled, fills NULLs or fallback)
     - Hybrid merge: Fills ENTSO-E NULL values from Energy-Charts
     - Complete fallback: If ENTSO-E > 180 min old
     - Quality: PARTIAL (hybrid) | FALLBACK
     - Data age: ~3 hours

  3. Known Capacity (pragmatic estimates)
     - Nuclear: 485 MW (Borssele)
     - Biomass: 350 MW (avg)
     - Solar: Estimated based on time/season
     - Quality: PARTIAL (estimates)

  4. Cache (in-memory, 5-min TTL)
     - Last successful response
     - Quality: CACHED

  5. None (report UNAVAILABLE)
     - Quality: UNAVAILABLE
```

### Load Data Fallback

```
Try in order:
  1. ENTSO-E A65 (measured, 15-min)
     - Tier: FRESH (< 90 min) or STALE (90-180 min)
     - Quality: FRESH | STALE

  2. Energy-Charts generation total (proxy for load)
     - If ENTSO-E > 180 min old
     - Quality: FALLBACK

  3. Cache (in-memory, 5-min TTL)
     - Quality: CACHED

  4. None (report UNAVAILABLE)
```

### Price Data Fallback

```
Try in order:
  1. ENTSO-E A44 (measured, hourly)
     - Updated daily at 12:42 CET
     - Quality: FRESH

  2. ENTSO-E A46 (intraday, continuous)
     - Intraday auction prices
     - Quality: FRESH | STALE

  3. Energy-Charts estimate
     - Modeled estimate
     - Quality: FALLBACK

  4. None (report UNAVAILABLE)
```

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

### When TenneT Fails

Similar pattern: log warning, return None, let normalizer use fallback.

### When Energy-Charts Fails

TenneT data is usually most critical (grid stability signals).
Energy-Charts fallback is best-effort.

---

## DATA QUALITY SCORING

### ENTSO-E A75 Quality Score

```python
def score_entso_e_a75(raw_data):
    """Score quality of generation data."""
    age_minutes = (now - raw_data.source_timestamp).total_seconds() / 60

    if age_minutes < 20:
        quality = 0.99  # Fresh data
    elif age_minutes < 40:
        quality = 0.90  # Slightly delayed
    elif age_minutes < 60:
        quality = 0.80  # Older
    elif age_minutes < 240:
        quality = 0.50  # Very old, use fallback
    else:
        quality = 0.0   # Too old, unusable

    # Penalty if data is incomplete (some PSR types missing)
    if missing_psr_types:
        quality *= 0.9

    return quality
```

### Energy-Charts Quality Score

```python
def score_energy_charts():
    """Score quality of modeled data."""
    # Always lower than real-time ENTSO-E
    return 0.65  # Modeled, not measured
```

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
        "tennet": {
            "status": "ok",
            "last_update": "2025-12-30T10:15:32Z",
            "age_minutes": 0,
            "quality": 0.99
        },
        "energy_charts": {
            "status": "ok",
            "last_update": "2025-12-30T00:00:00Z",
            "age_hours": 10,
            "quality": 0.65  # Modeled data
        }
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
