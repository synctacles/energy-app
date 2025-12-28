# SYNCTACLES API Reference

**Version:** 1.0.0  
**Base URL:** `https://synctacles.io`  
**Authentication:** API Key (X-API-Key header)

---

## Authentication

### POST /auth/signup
**Public endpoint** - Create new user account

**Request:**
```json
{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "user_id": "d20bedae-6253-41dd-b52b-74d24baab6d5",
  "email": "user@example.com",
  "license_key": "0dbccc43-f012-4604-8adc-e61ef13c366d",
  "api_key": "d4ab518e76817c4e5a89841b1dcafbea888adba63ce9dc14040b965403422916",
  "message": "Account created. Save your API key - it won't be shown again!"
}
```

**⚠️ IMPORTANT:** Store API key securely - it cannot be retrieved later.

---

### GET /auth/stats
**Requires authentication** - View account usage

**Headers:**
```
X-API-Key: YOUR_API_KEY
```

**Response:**
```json
{
  "user_id": "d20bedae-6253-41dd-b52b-74d24baab6d5",
  "email": "user@example.com",
  "tier": "free",
  "rate_limit_daily": 1000,
  "usage_today": 18,
  "remaining_today": 982
}
```

---

### POST /auth/regenerate-key
**Requires authentication** - Generate new API key (invalidates old)

**Headers:**
```
X-API-Key: YOUR_CURRENT_API_KEY
```

**Response:**
```json
{
  "new_api_key": "a6d003acdcf9b1524546cdeca215e4db8e9c71a588f31e764f52bd5ac595909f",
  "message": "API key regenerated. Old key is now invalid."
}
```

---

## Energy Data Endpoints

All endpoints require `X-API-Key` header.

### GET /api/v1/generation-mix
**Electricity generation by source** (Netherlands)

**Query Parameters:**
- `hours` (optional, default: 72) - Historical data window (max: 168)
- `include_forecast` (optional, default: false) - Include future projections

**Response:**
```json
{
  "data": [
    {
      "timestamp": "2025-12-21T02:00:00Z",
      "solar_mw": 0.0,
      "wind_offshore_mw": 1234.5,
      "wind_onshore_mw": 543.2,
      "gas_mw": 3210.8,
      "coal_mw": 0.0,
      "nuclear_mw": 482.0,
      "biomass_mw": 127.4,
      "waste_mw": 56.3,
      "other_mw": 23.1,
      "total_mw": 5677.3
    }
  ],
  "meta": {
    "source": "ENTSO-E",
    "quality_status": "OK",
    "timestamp_utc": "2025-12-21T02:15:00Z",
    "data_age_seconds": 900,
    "next_update_utc": "2025-12-21T02:30:00Z"
  }
}
```

**Quality Status:**
- `OK` - Fresh data (< 15 min old), safe for automation
- `FALLBACK` - Secondary source (Energy-Charts), use with caution
- `STALE` - Old data (15 min - 1 hour), verify before automation
- `NO_DATA` - No data available, do not automate

**Rate Limit:** 1000 requests/day (free tier)

---

### GET /api/v1/load
**Electricity consumption** (Netherlands)

**Query Parameters:**
- `hours` (optional, default: 72) - Historical data window
- `forecast_hours` (optional, default: 24) - Future forecast window

**Response:**
```json
{
  "data": [
    {
      "timestamp": "2025-12-21T02:00:00Z",
      "actual_mw": 12345.6,
      "forecast_mw": 12778.3
    }
  ],
  "meta": {
    "source": "ENTSO-E",
    "quality_status": "OK",
    "data_age_seconds": 1200
  }
}
```

---

### GET /api/v1/balance
**Grid balance delta** (Netherlands)

**Query Parameters:**
- `hours` (optional, default: 72) - Historical data window

**Response:**
```json
{
  "data": [
    {
      "timestamp": "2025-12-21T02:00:00Z",
      "delta_mw": 219.9,
      "price_eur_mwh": 61.56,
      "platforms": {
        "aFRR": 120.3,
        "IGCC": 45.2,
        "MARI": 32.1,
        "mFRRda": 15.8,
        "PICASSO": 6.5
      }
    }
  ],
  "meta": {
    "source": "TenneT",
    "quality_status": "OK",
    "data_age_seconds": 300
  }
}
```

**Balance Delta:**
- Positive: Generation > Load (surplus)
- Negative: Load > Generation (deficit)

---

## Error Responses

### 401 Unauthorized
```json
{
  "detail": "Invalid API key"
}
```

### 429 Rate Limit Exceeded
```json
{
  "detail": "Rate limit exceeded. Daily limit: 1000 requests."
}
```

### 500 Internal Server Error
```json
{
  "detail": "Internal server error. Contact support."
}
```

---

## Rate Limits

**Free Tier:**
- 1000 requests/day
- Resets daily at 00:00 UTC
- Applies to all `/api/v1/*` endpoints

**Premium Tier (V1.1):**
- 10,000 requests/day
- €4.99/month

---

## Data Sources

**Primary:**
- ENTSO-E Transparency Platform (Generation, Load)
- TenneT TSO API (Grid Balance)

**Fallback:**
- Energy-Charts (Fraunhofer ISE) - Generation only
- Cache (< 1 hour old) - All endpoints

**Attribution:**
- ENTSO-E data: [transparency.entsoe.eu](https://transparency.entsoe.eu)
- TenneT data: [api.tennet.eu](https://api.tennet.eu)
- Energy-Charts: CC BY 4.0, Fraunhofer ISE

---

## Integration Examples

### Python (requests)
```python
import requests

API_KEY = "your_api_key_here"
BASE_URL = "https://synctacles.io"

headers = {"X-API-Key": API_KEY}

# Get generation mix
response = requests.get(f"{BASE_URL}/api/v1/generation-mix", headers=headers)
data = response.json()

print(f"Total generation: {data['data'][0]['total_mw']} MW")
print(f"Quality: {data['meta']['quality_status']}")
```

### Home Assistant (YAML)
```yaml
sensor:
  - platform: rest
    name: "NL Generation Total"
    resource: https://synctacles.io/api/v1/generation-mix
    headers:
      X-API-Key: YOUR_API_KEY
    value_template: "{{ value_json.data[0].total_mw }}"
    unit_of_measurement: "MW"
    scan_interval: 900  # 15 minutes
```

### cURL (Bash)
```bash
curl -H "X-API-Key: YOUR_API_KEY" \
  https://synctacles.io/api/v1/generation-mix | jq '.data[0]'
```

---

## Support

**Email:** support@synctacles.io  
**GitHub:** [DATADIO/synctacles-repo](https://github.com/DATADIO/synctacles-repo) (issues)  
**Documentation:** https://synctacles.io/docs

---

**Last Updated:** 2025-12-21
