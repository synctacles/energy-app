# Energy-Charts API Response Format Documentation

## Overview

This document describes the response format from the Energy-Charts API endpoint used to fetch day-ahead electricity prices for the Netherlands.

**API Endpoint:** `https://api.energy-charts.info/price`

**Authentication:** None (public API)

**Rate Limit:** 10 requests per minute (free tier)

---

## Request Parameters

### Query Parameters

```
GET https://api.energy-charts.info/price?bzn=NL&start=2025-12-31&end=2026-01-02
```

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `bzn`     | string | Yes      | Bidding Zone code (NL = Netherlands) |
| `start`   | string | Yes      | Start date in ISO 8601 format (YYYY-MM-DD) |
| `end`     | string | Yes      | End date in ISO 8601 format (YYYY-MM-DD) |

### Example Request

```bash
curl "https://api.energy-charts.info/price?bzn=NL&start=2025-12-31&end=2026-01-02"
```

---

## Response Structure

### Top-Level Fields

```json
{
  "license_info": "string",      // Licensing and data attribution
  "unix_seconds": [integer, ...],// Unix timestamps (seconds since epoch)
  "price": [number, ...],        // Price values corresponding to each timestamp
  "unit": "string",              // Unit of measurement (EUR / MWh)
  "deprecated": boolean          // API deprecation status flag
}
```

### Field Descriptions

#### `license_info` (string)
- **Type:** String
- **Format:** License attribution notice
- **Example:** `"CC BY 4.0 (creativecommons.org/licenses/by/4.0) from Bundesnetzagentur | SMARD.de"`
- **Purpose:** Indicates data licensing and attribution requirements
- **Note:** Data sourced from SMARD.de (Strombörse Market Data)

#### `unix_seconds` (array of integers)
- **Type:** Array of integers
- **Element Type:** Integer (Unix timestamp in seconds)
- **Format:** Seconds since Unix epoch (January 1, 1970, 00:00:00 UTC)
- **Length:** Typically 360 elements for 2-day period (15-minute intervals)
- **Example Values:** `[1767135600, 1767136500, 1767137400, ...]`
- **Purpose:** Provides timestamps for each price data point
- **Notes:**
  - Each timestamp represents a 15-minute interval
  - Timestamps increment by 900 seconds (15 minutes × 60 seconds)
  - Time zone: UTC
  - **CRITICAL:** `unix_seconds` and `price` arrays MUST have the same length

#### `price` (array of numbers)
- **Type:** Array of floating-point numbers
- **Element Type:** Float (decimal number)
- **Format:** Day-ahead electricity price per hour
- **Unit:** EUR / MWh (Euro per Megawatt-hour)
- **Length:** Must match length of `unix_seconds`
- **Range:** Typically 0-200+ EUR/MWh (can vary)
- **Example Values:** `[94.87, 90.31, 86.19, 81.04, ...]`
- **Purpose:** Market electricity price at each timestamp
- **Data Source:** Day-ahead auction prices from SMARD.de
- **Notes:**
  - Values can be negative (rare, indicates surplus)
  - Values may include decimals (e.g., 94.87)
  - Each price corresponds to the timestamp at the same array index

#### `unit` (string)
- **Type:** String
- **Value:** `"EUR / MWh"`
- **Purpose:** Specifies the unit of measurement for price values
- **Note:** Always "EUR / MWh" for price endpoint

#### `deprecated` (boolean)
- **Type:** Boolean
- **Value:** `true` or `false`
- **Purpose:** Indicates if the API endpoint or response format is deprecated
- **Note:** Should be monitored for API migration planning

---

## Complete Example Response

### Live Response (2025-12-31 to 2026-01-02)

```json
{
  "license_info": "CC BY 4.0 (creativecommons.org/licenses/by/4.0) from Bundesnetzagentur | SMARD.de",
  "unix_seconds": [
    1767135600,
    1767136500,
    1767137400,
    1767138300,
    1767139200,
    1767140100,
    1767141000,
    1767141900,
    1767142800,
    ...
  ],
  "price": [
    94.87,
    90.31,
    86.19,
    81.04,
    90.11,
    83.48,
    82.69,
    81.15,
    83.54,
    ...
  ],
  "unit": "EUR / MWh",
  "deprecated": false
}
```

### Sample Data Mapping

For a 2-day request (start: 2025-12-31, end: 2026-01-02), expect approximately 288-360 data points:
- 2 days × 24 hours × 4 data points per hour = 192 minimum
- Additional days beyond end date may be included

### Data Point Alignment

Each price value at index `i` corresponds to the timestamp at the same index:

```
Index 0:
  Timestamp: 1767135600 (2025-12-31 10:00:00 UTC)
  Price: 94.87 EUR/MWh

Index 1:
  Timestamp: 1767136500 (2025-12-31 10:15:00 UTC)
  Price: 90.31 EUR/MWh

Index 2:
  Timestamp: 1767137400 (2025-12-31 10:30:00 UTC)
  Price: 86.19 EUR/MWh
```

---

## Data Characteristics

### Time Intervals

- **Frequency:** 15-minute intervals (900 seconds between consecutive timestamps)
- **Duration:** Typically 2+ days of historical data
- **Time Zone:** UTC/Zulu time
- **Start Time:** Depends on request start date parameter

### Price Characteristics

- **Data Type:** Floating-point decimal
- **Typical Range:** 0 - 200+ EUR/MWh
- **Precision:** Up to 2 decimal places (cents)
- **Missing Data:** Represented as `null` (rare)
- **Negative Prices:** Possible but rare (indicates oversupply)

### Array Lengths

- Both `unix_seconds` and `price` arrays **MUST** have identical lengths
- Typical length for 2-day request: 280-360 elements
- **Validation:** Check that `len(unix_seconds) == len(price)`

---

## Error Scenarios

### HTTP Status Codes

| Status | Meaning | Handling |
|--------|---------|----------|
| 200 | Success | Parse response normally |
| 400 | Bad request | Check query parameters (date format, bzn code) |
| 404 | Not found | No data available for date range |
| 429 | Rate limited | Wait and retry (10 req/min limit) |
| 500 | Server error | Retry with exponential backoff |
| 503 | Service unavailable | Service down, use fallback |

### Error Response Example

```json
{
  "error": "Invalid date range",
  "message": "Start date must be before end date"
}
```

---

## Integration with SYNCTACLES

### Parsing Function Requirements

The `fetch_prices()` function in `energy_charts_client.py` should:

1. **Fetch the API response**
   ```python
   response = requests.get(
       "https://api.energy-charts.info/price",
       params={"bzn": country, "start": start, "end": end},
       timeout=30
   )
   ```

2. **Validate response structure**
   ```python
   data = response.json()
   assert "unix_seconds" in data
   assert "price" in data
   assert len(data["unix_seconds"]) == len(data["price"])
   ```

3. **Convert timestamps to ISO 8601**
   ```python
   from datetime import datetime, timezone
   for unix_ts in data["unix_seconds"]:
       iso_ts = datetime.fromtimestamp(unix_ts, tz=timezone.utc).isoformat()
   ```

4. **Pair timestamps with prices**
   ```python
   for unix_ts, price_eur_mwh in zip(data["unix_seconds"], data["price"]):
       # Create price record for database
   ```

### Database Schema

Prices should be stored in the `raw_prices` table:

| Column | Type | Value |
|--------|------|-------|
| `timestamp` | TIMESTAMP | Converted from `unix_seconds` to ISO 8601 |
| `country` | VARCHAR | "NL" (or from `bzn` param) |
| `price_eur_mwh` | DECIMAL | Value from `price` array |
| `source` | VARCHAR | "energy-charts" |
| `source_file` | VARCHAR | Original JSON file name |

### Example Processing

```python
def process_energy_charts_response(data: dict, country: str = "NL"):
    """Convert Energy-Charts response to database records."""
    unix_seconds = data.get("unix_seconds", [])
    prices = data.get("price", [])

    if len(unix_seconds) != len(prices):
        raise ValueError("Mismatch: unix_seconds vs prices length")

    records = []
    for unix_ts, price in zip(unix_seconds, prices):
        timestamp = datetime.fromtimestamp(unix_ts, tz=timezone.utc)
        records.append({
            "timestamp": timestamp,
            "country": country,
            "price_eur_mwh": float(price),
            "source": "energy-charts"
        })

    return records
```

---

## Response Validation Checklist

Before processing prices, verify:

- [ ] HTTP status is 200
- [ ] Response is valid JSON
- [ ] `unix_seconds` key exists and contains array of integers
- [ ] `price` key exists and contains array of numbers
- [ ] Array lengths match: `len(unix_seconds) == len(price)`
- [ ] Timestamps are in ascending order
- [ ] No null or missing values (unless intentional)
- [ ] `unit` is "EUR / MWh"
- [ ] `deprecated` is false (or false is acceptable)

---

## API Behavior Notes

### Data Freshness

- Data is updated daily with day-ahead prices
- Prices published approximately 12:42 CET for next day
- Historical data available up to several years back
- Data sourced from SMARD.de (German electricity exchange)

### Netherlands Specifics

- **Bidding Zone Code:** `NL` (or `10YNL----------L` for ENTSO-E)
- **Time Zone:** CET/CEST (Central European Time)
- **Data Source:** APX ENDEX market (Dutch power exchange)
- **Pricing:** Day-ahead auction prices

### Rate Limiting

- **Free Tier:** 10 requests per minute
- **Recommended Strategy:** Cache results, batch requests
- **Circuit Breaker:** Skip for 2 hours after HTTP 404

---

## Comparison with Generation Mix API

Energy-Charts also provides generation mix data via a different endpoint:

```bash
GET https://api.energy-charts.info/public_power?country=nl
```

This endpoint returns:
```json
{
  "unix_seconds": [...],
  "production_types": [
    {"name": "Solar", "data": [...]},
    {"name": "Wind onshore", "data": [...]},
    ...
  ]
}
```

**Note:** The price endpoint documented here is specifically for electricity prices, not generation mix.

---

## See Also

- **Energy-Charts Website:** https://energy-charts.info/
- **SMARD.de:** https://www.smard.de/
- **Data Source Documentation:** `/docs/skills/SKILL_06_DATA_SOURCES.md`
- **Collector Code:** `/synctacles_db/collectors/energy_charts_prices.py`
- **Import Code:** `/synctacles_db/importers/import_energy_charts_prices.py`

---

## Document History

| Date | Version | Notes |
|------|---------|-------|
| 2026-01-02 | 1.0 | Initial documentation with live API verification |
