# Energy-Charts API Integration Guide

This guide explains how to integrate the Energy-Charts price API response format into the `energy_charts_client.py` module.

## Quick Reference

**API Endpoint:** `https://api.energy-charts.info/price`

**Response Format:**
```json
{
  "license_info": "...",
  "unix_seconds": [1767135600, 1767136500, ...],
  "price": [94.87, 90.31, ...],
  "unit": "EUR / MWh",
  "deprecated": false
}
```

**Key Constraint:** `len(unix_seconds)` MUST equal `len(price)`

---

## Current Implementation Status

### Files Involved

1. **Collector:** `/opt/github/synctacles-api/synctacles_db/collectors/energy_charts_prices.py`
   - Fetches raw JSON from API
   - Saves to `/var/log/energy-insights/collectors/energy_charts_raw/`

2. **Importer:** `/opt/github/synctacles-api/synctacles_db/importers/import_energy_charts_prices.py`
   - Reads saved JSON files
   - Parses unix_seconds and price arrays
   - Inserts into `raw_prices` table

3. **Client (Fallback):** `/opt/github/synctacles-api/synctacles_db/fallback/energy_charts_client.py`
   - Currently handles generation mix data (public_power endpoint)
   - Should be extended for price data

### Current Collector Code

```python
def fetch_prices(country: str = "NL", days: int = 2) -> dict:
    """Fetch day-ahead prices for country."""
    BASE_URL = "https://api.energy-charts.info/price"
    params = {"bzn": country, "start": start, "end": end}

    response = requests.get(BASE_URL, params=params, timeout=30)
    response.raise_for_status()

    data = response.json()  # Returns the response format documented above

    # Save raw response
    output_file = OUTPUT_DIR / f"prices_{country}_{timestamp}.json"
    with open(output_file, "w") as f:
        json.dump(data, f, indent=2)

    return data
```

---

## Response Parsing Workflow

### Step 1: Fetch Raw Data

```python
import requests
from datetime import datetime, timedelta, timezone

def fetch_prices(country: str = "NL", days: int = 2) -> dict:
    """Fetch from Energy-Charts price endpoint."""
    BASE_URL = "https://api.energy-charts.info/price"

    today = datetime.now(timezone.utc).date()
    start = today.isoformat()
    end = (today + timedelta(days=days)).isoformat()

    params = {"bzn": country, "start": start, "end": end}

    response = requests.get(BASE_URL, params=params, timeout=30)
    response.raise_for_status()  # Raises HTTPError on 4xx/5xx

    return response.json()
```

**Returns:** Dict with structure shown above

---

### Step 2: Validate Response Structure

```python
def validate_response(data: dict) -> bool:
    """Validate Energy-Charts price response format."""
    required_fields = ["license_info", "unix_seconds", "price", "unit", "deprecated"]

    # Check all required fields exist
    for field in required_fields:
        if field not in data:
            raise ValueError(f"Missing required field: {field}")

    # Validate field types
    if not isinstance(data["unix_seconds"], list):
        raise TypeError("unix_seconds must be array")

    if not isinstance(data["price"], list):
        raise TypeError("price must be array")

    # Critical: Arrays must have same length
    if len(data["unix_seconds"]) != len(data["price"]):
        raise ValueError(
            f"Array length mismatch: "
            f"unix_seconds={len(data['unix_seconds'])}, "
            f"price={len(data['price'])}"
        )

    # Validate data types in arrays
    for ts in data["unix_seconds"]:
        if not isinstance(ts, int):
            raise TypeError(f"unix_seconds elements must be integers, got {type(ts)}")

    for price in data["price"]:
        if not isinstance(price, (int, float)):
            raise TypeError(f"price elements must be numeric, got {type(price)}")

    return True
```

---

### Step 3: Parse and Transform Data

```python
from datetime import datetime, timezone

def parse_price_response(data: dict, country: str = "NL") -> list:
    """Convert Energy-Charts response to normalized records."""

    # Validate first
    validate_response(data)

    unix_seconds = data["unix_seconds"]
    prices = data["price"]

    records = []

    for unix_ts, price_value in zip(unix_seconds, prices):
        # Convert Unix timestamp to ISO 8601 UTC datetime
        timestamp = datetime.fromtimestamp(unix_ts, tz=timezone.utc)

        # Handle null prices (rare)
        price_eur_mwh = float(price_value) if price_value is not None else None

        record = {
            "timestamp": timestamp,          # datetime object
            "timestamp_iso": timestamp.isoformat(),  # ISO 8601 string
            "country": country,              # "NL"
            "price_eur_mwh": price_eur_mwh,
            "source": "energy-charts",       # Data source identifier
            "unit": data["unit"],            # "EUR / MWh"
        }

        records.append(record)

    return records
```

---

### Step 4: Store in Database

```python
from sqlalchemy.orm import Session
from datetime import datetime

def import_prices_to_db(data: dict, session: Session, country: str = "NL"):
    """Import parsed prices into raw_prices table."""

    records = parse_price_response(data, country)
    imported_count = 0

    for record in records:
        # Prepare database insert
        sql = """
            INSERT INTO raw_prices
            (timestamp, country, price_eur_mwh, source, source_file)
            VALUES (:timestamp, :country, :price_eur_mwh, :source, :source_file)
            ON CONFLICT (timestamp, country, source)
            DO UPDATE SET
                price_eur_mwh = EXCLUDED.price_eur_mwh
        """

        params = {
            "timestamp": record["timestamp"],
            "country": record["country"],
            "price_eur_mwh": record["price_eur_mwh"],
            "source": record["source"],
            "source_file": "energy_charts_prices.json"
        }

        session.execute(sql, params)
        imported_count += 1

    session.commit()

    print(f"Imported {imported_count} price records from Energy-Charts")
    return imported_count
```

---

## Complete Example: Fetch and Process

```python
import json
import logging
from datetime import datetime, timedelta, timezone
from pathlib import Path
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

_LOGGER = logging.getLogger(__name__)
LOG_DIR = Path("/var/log/energy-insights")
OUTPUT_DIR = LOG_DIR / "collectors" / "energy_charts_raw"

def fetch_and_process_prices(country: str = "NL", days: int = 2,
                             db_url: str = None) -> int:
    """Complete pipeline: fetch -> validate -> parse -> store."""

    # Step 1: Fetch raw data
    _LOGGER.info(f"Fetching {country} electricity prices for {days} days...")
    data = fetch_prices(country, days)

    # Step 2: Save raw response
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now(timezone.utc).strftime("%Y%m%d_%H%M%S")
    output_file = OUTPUT_DIR / f"prices_{country}_{timestamp}.json"

    with open(output_file, "w") as f:
        json.dump(data, f, indent=2)

    _LOGGER.info(f"Saved raw response: {output_file}")

    # Step 3: Validate
    try:
        validate_response(data)
        _LOGGER.info(f"Response validation passed. {len(data['price'])} data points.")
    except Exception as e:
        _LOGGER.error(f"Response validation failed: {e}")
        raise

    # Step 4: Parse
    records = parse_price_response(data, country)
    _LOGGER.info(f"Parsed {len(records)} price records")

    # Step 5: Store in database
    if db_url:
        engine = create_engine(db_url)
        Session = sessionmaker(bind=engine)
        session = Session()

        try:
            imported = import_prices_to_db(data, session, country)
            _LOGGER.info(f"Successfully imported {imported} records to database")
            return imported
        finally:
            session.close()

    return len(records)
```

---

## Timestamp Handling

### Convert Unix Timestamp to ISO 8601

```python
from datetime import datetime, timezone

unix_timestamp = 1767135600
iso_timestamp = datetime.fromtimestamp(unix_timestamp, tz=timezone.utc).isoformat()
print(iso_timestamp)  # Output: 2025-12-31T10:00:00+00:00
```

### Database Column Definition

```sql
CREATE TABLE raw_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,  -- ISO 8601 format
    country VARCHAR(2) NOT NULL,                   -- "NL"
    price_eur_mwh DECIMAL(10, 2),                  -- Supports up to 99999.99
    source VARCHAR(50),                            -- "energy-charts"
    source_file VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(timestamp, country, source)
);
```

---

## Error Handling

### Common Issues and Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| `KeyError: 'unix_seconds'` | Missing field in response | Validate response structure first |
| Array length mismatch | Corrupted data | Check `len(unix_seconds) == len(price)` |
| `TypeError: float() argument must be a string or a number` | Null in price array | Handle None: `float(x) if x is not None else 0` |
| Invalid ISO 8601 timestamp | Timezone issue | Always use `tz=timezone.utc` in conversion |
| HTTP 429 (Rate Limited) | Too many requests | Implement circuit breaker (skip 2 hours) |
| HTTP 404 | Invalid date range | Verify start < end and both are valid dates |

### Robust Error Handling Pattern

```python
def fetch_and_process_with_fallback(country: str = "NL", days: int = 2):
    """Fetch with comprehensive error handling."""
    try:
        data = fetch_prices(country, days)
    except requests.Timeout:
        _LOGGER.warning("Energy-Charts timeout, will retry next cycle")
        return None
    except requests.HTTPError as e:
        if e.response.status_code == 429:
            _LOGGER.warning("Energy-Charts rate limited, skipping for 2 hours")
            # Implement circuit breaker
            return None
        elif e.response.status_code == 404:
            _LOGGER.error(f"No data found for {country} in date range")
            return None
        else:
            _LOGGER.error(f"HTTP error: {e}")
            raise
    except Exception as e:
        _LOGGER.error(f"Unexpected error: {e}")
        raise

    try:
        validate_response(data)
        return parse_price_response(data, country)
    except ValueError as e:
        _LOGGER.error(f"Response validation error: {e}")
        return None
```

---

## Testing

### Unit Test Template

```python
import pytest
from datetime import datetime, timezone

@pytest.fixture
def valid_price_response():
    return {
        "license_info": "CC BY 4.0 ...",
        "unix_seconds": [1767135600, 1767136500, 1767137400],
        "price": [94.87, 90.31, 86.19],
        "unit": "EUR / MWh",
        "deprecated": False
    }

def test_validate_response_valid(valid_price_response):
    assert validate_response(valid_price_response) is True

def test_parse_price_response(valid_price_response):
    records = parse_price_response(valid_price_response, "NL")

    assert len(records) == 3
    assert records[0]["price_eur_mwh"] == 94.87
    assert records[0]["country"] == "NL"
    assert records[0]["source"] == "energy-charts"
    assert isinstance(records[0]["timestamp"], datetime)

def test_array_length_mismatch():
    invalid_data = {
        "license_info": "...",
        "unix_seconds": [1767135600, 1767136500],
        "price": [94.87, 90.31, 86.19],  # Length mismatch!
        "unit": "EUR / MWh",
        "deprecated": False
    }

    with pytest.raises(ValueError, match="Array length mismatch"):
        validate_response(invalid_data)
```

---

## Performance Considerations

### Request Optimization

- **Rate Limit:** 10 requests per minute (free tier)
- **Timeout:** Set 30 seconds (longer than typical 330ms response)
- **Batch Requests:** Fetch multiple days in single request if possible
- **Caching:** Cache results (valid for 24 hours for day-ahead data)

### Database Optimization

```python
# Use batch insert for better performance
def import_prices_batch(data: dict, session: Session, country: str = "NL"):
    """Batch insert for performance."""
    records = parse_price_response(data, country)

    # Convert to bulk insert format
    values = [
        {
            "timestamp": r["timestamp"],
            "country": r["country"],
            "price_eur_mwh": r["price_eur_mwh"],
            "source": r["source"],
        }
        for r in records
    ]

    # Bulk insert
    session.execute(
        "INSERT INTO raw_prices (timestamp, country, price_eur_mwh, source) "
        "VALUES (:timestamp, :country, :price_eur_mwh, :source) "
        "ON CONFLICT DO NOTHING",
        values
    )

    session.commit()
    return len(records)
```

---

## Monitoring and Alerting

### Health Check

```python
def check_energy_charts_health():
    """Monitor Energy-Charts availability and data freshness."""
    try:
        # Fetch recent data
        data = fetch_prices("NL", days=1)
        validate_response(data)

        # Check data freshness
        latest_ts = max(data["unix_seconds"])
        latest_dt = datetime.fromtimestamp(latest_ts, tz=timezone.utc)
        age_hours = (datetime.now(timezone.utc) - latest_dt).total_seconds() / 3600

        return {
            "status": "healthy" if age_hours < 24 else "stale",
            "data_points": len(data["price"]),
            "age_hours": age_hours,
            "latest_timestamp": latest_dt.isoformat()
        }
    except Exception as e:
        return {
            "status": "error",
            "error": str(e)
        }
```

---

## See Also

- **API Documentation:** `/docs/ENERGY_CHARTS_API_RESPONSE_FORMAT.md`
- **JSON Schema:** `/docs/ENERGY_CHARTS_JSON_SCHEMA.json`
- **Response Example:** `/docs/ENERGY_CHARTS_RESPONSE_EXAMPLE.json`
- **Data Sources Guide:** `/docs/skills/SKILL_06_DATA_SOURCES.md`
- **Current Collector:** `/synctacles_db/collectors/energy_charts_prices.py`
- **Current Importer:** `/synctacles_db/importers/import_energy_charts_prices.py`

---

## Next Steps

1. **Add price endpoint to EnergyChartsClient class** in `energy_charts_client.py`
2. **Implement proper error handling** with circuit breaker for rate limits
3. **Add unit tests** for response validation and parsing
4. **Integrate with fallback system** for when ENTSO-E prices unavailable
5. **Add monitoring** to track API health and data freshness
