# Energy Insights NL - API Reference

**Version:** 1.0.0
**Base URL:** `http://localhost:8000` (development) | `https://api.example.com` (production)
**Authentication:** API Key (X-API-Key header) - see [Deployment Guide](deployment.md) for setup

---

## Rate Limiting

All authenticated endpoints are subject to daily rate limits based on subscription tier:

### Tier Limits

- **Beta**: 10,000 requests/day (default for new users)
- **Free**: 1,000 requests/day
- **Paid**: 100,000 requests/day
- **Unlimited**: 100,000 requests/day (enterprise)

### Rate Limit Headers

Every authenticated response includes:

```
X-RateLimit-Limit: 10000          # Your daily limit
X-RateLimit-Remaining: 9876       # Requests remaining today
X-RateLimit-Reset: 1735689600     # Unix timestamp (midnight UTC)
```

### Exceeding Limits

When rate limit is exceeded, the API returns **HTTP 429** with:

```json
{
  "detail": "Rate limit exceeded. Daily limit reset at midnight UTC."
}
```

The reset time is always **00:00 UTC** (midnight).

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
**⚠️ DEPRECATED - Returns 501 Not Implemented**

Grid balance data is no longer available via the SYNCTACLES API due to TenneT license restrictions.

**Response:**
```json
{
  "error": "Not Implemented",
  "message": "Balance data available via BYO-key in HA component",
  "documentation": "https://github.com/DATADIO/ha-energy-insights-nl#tennet-byo-key",
  "reason": "TenneT API license prohibits server-side redistribution"
}
```

**HTTP Status:** 501 Not Implemented

**Alternative:** Configure your personal TenneT API key in the Home Assistant integration to enable real-time balance data locally.

**Update Interval:** N/A (endpoint deprecated)
**Data Source:** Available via BYO-key in HA component only

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

**Server-side (via API):**
- ENTSO-E Transparency Platform (Generation, Load, Prices)
- Energy-Charts (Fraunhofer ISE) - Fallback

**Client-side (HA Component with BYO-key):**
- TenneT TSO API (Grid Balance) - requires user's personal API key

**Fallback:**
- Energy-Charts (Fraunhofer ISE) - Generation only
- Cache (< 1 hour old) - All endpoints

**Attribution:**
- ENTSO-E data: [transparency.entsoe.eu](https://transparency.entsoe.eu)
- Energy-Charts: CC BY 4.0, Fraunhofer ISE
- TenneT: User's personal API key, local processing only

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
