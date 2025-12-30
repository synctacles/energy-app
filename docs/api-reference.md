# Energy Insights NL - API Reference

**Version:** 1.0.0
**Base URL:** `http://localhost:8000` (development) | `https://api.example.com` (production)
**Authentication:** API Key (X-API-Key header) - see [Deployment Guide](deployment.md) for setup

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

### GET /v1/generation/current
**Current electricity generation by source** (Netherlands)

**Headers:**
```
X-API-Key: YOUR_API_KEY
```

**Response:**
```json
{
  "timestamp": "2025-12-30T14:30:00Z",
  "data": {
    "biomass_mw": 375.0,
    "wind_onshore_mw": 2150.5,
    "wind_offshore_mw": 680.0,
    "solar_mw": 0.0,
    "nuclear_mw": 485.0,
    "gas_mw": 1850.0,
    "coal_mw": 400.0,
    "waste_mw": 150.0,
    "other_mw": 280.5,
    "total_mw": 6370.0
  },
  "metadata": {
    "source": "ENTSO-E",
    "quality": "STALE",
    "age_seconds": 2145,
    "confidence_score": 92,
    "renewable_percentage": 42.3
  }
}
```

**Quality Values:**
- `FRESH` - Data < 30 min old, use immediately
- `STALE` - Data 30-150 min old (normal for ENTSO-E A75)
- `FALLBACK` - Using Energy-Charts (ENTSO-E unavailable)
- `UNAVAILABLE` - No data available

**Update Interval:** Every 15 minutes
**Data Source:** ENTSO-E A75 (authoritative) → Energy-Charts (fallback)

---

### GET /v1/load/current
**Current grid load (actual + forecast)** (Netherlands)

**Response:**
```json
{
  "timestamp": "2025-12-30T14:30:00Z",
  "data": {
    "load_actual_mw": 5200.0,
    "load_forecast_mw": 5100.0,
    "load_difference_mw": 100.0
  },
  "metadata": {
    "source": "ENTSO-E",
    "quality": "STALE",
    "age_seconds": 1800,
    "confidence_score": 95
  }
}
```

**Update Interval:** Every 15 minutes
**Data Source:** ENTSO-E A65 (authoritative) → Energy-Charts (fallback)

---

### GET /v1/balance/current
**Current grid balance delta** (Netherlands)

**Response:**
```json
{
  "timestamp": "2025-12-30T14:30:00Z",
  "data": {
    "balance_mw": 125.0,
    "imbalance_price_eur": 2.50
  },
  "metadata": {
    "source": "TenneT",
    "quality": "FRESH",
    "age_seconds": 60
  }
}
```

**Balance Delta Interpretation:**
- **Positive (>0):** Generation > Load = surplus energy (good for charging)
- **Negative (<0):** Load > Generation = deficit (critical periods)
- **Zero:** Perfectly balanced grid (rare)

**Update Interval:** Every 5 minutes (rate-limited from TenneT)
**Data Source:** TenneT API (no fallback available)

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

## See Also

- [Architecture Guide](ARCHITECTURE.md) - System design & data flow
- [Deployment Guide](deployment.md) - Installation & operations
- [Troubleshooting Guide](troubleshooting.md) - Common issues & fixes
- [Home Assistant Integration](api/signals.md) - HA sensor setup

---

**Last Updated:** 2025-12-30
**Status:** Production Ready
**Repository:** [DATADIO/ha-energy-insights-nl](https://github.com/DATADIO/ha-energy-insights-nl)
