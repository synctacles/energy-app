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
GET https://api.energy-charts.info/v1/power
  ?country=de  # Country code (de=Germany, nl=Netherlands)
  &resolution=15m
```

**Example Response (JSON):**
```json
{
  "data": [
    {
      "time": "2025-12-30T10:00Z",
      "solar_mw": 420,
      "wind_onshore_mw": 1200,
      "wind_offshore_mw": 600,
      "hydro_mw": 150
    }
  ]
}
```

**Energy-Charts Details:**

- **Base URL:** https://api.energy-charts.info/v1/
- **Authentication:** None (public, rate-limited)
- **Rate Limit:** 10 requests per minute (for free tier)
- **Response Format:** JSON
- **Timeout:** 30 seconds
- **Data Freshness:** Updated daily (not real-time)
- **Accuracy:** ±5-10% for generation estimation

**Reliability:**
- Highly reliable (model doesn't fail)
- Used as fallback specifically for reliability
- Data is modeled, not measured (lower accuracy)

**Cost:** Free (public API)

---

## FALLBACK STRATEGY

When to use fallback:

### Generation Data Fallback

```
Try in order:
  1. ENTSO-E A75 (real-time, 15-min)
     - Latest within 20 minutes
     - Quality > 0.85

  2. Energy-Charts model
     - Last successful update
     - Quality: 0.6-0.7

  3. Last known good value
     - From previous collection
     - Quality: 0.3-0.4

  4. None (report error)
     - Quality: 0.0
```

### Load Data Fallback

```
Try in order:
  1. ENTSO-E A65 (real-time, 15-min)
     - Latest within 20 minutes
     - Quality > 0.85

  2. ENTSO-E A65 forecast
     - Previously published forecast
     - Quality: 0.7

  3. Last known good value
     - Quality: 0.3
```

### Price Data Fallback

```
Try in order:
  1. ENTSO-E A44 (real-time)
     - Updated daily at 12:42 CET
     - Quality > 0.9

  2. ENTSO-E A46 (intraday)
     - Updated continuously
     - Quality: 0.8

  3. Energy-Charts estimate
     - Quality: 0.5
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
