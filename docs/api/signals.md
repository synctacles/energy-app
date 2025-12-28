# Signals API Documentation

## Overview
The Signals API provides binary decision signals for home energy automation, designed for Home Assistant integration.

**Base URL:** `https://api.synctacles.io`  
**Version:** v1  
**Authentication:** API Key (X-API-Key header)

---

## Endpoint: Get Binary Signals

### Request

**Method:** `GET`  
**Path:** `/api/v1/signals`  
**Auth:** Required

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| user_id | UUID | Yes | User identifier |

**Headers:**
```http
X-API-Key: your_api_key_here
```

**Example Request:**
```bash
curl -H "X-API-Key: your_api_key" \
  "https://api.synctacles.io/api/v1/signals?user_id=123e4567-e89b-12d3-a456-426614174000"
```

---

### Response

**Success (200 OK):**
```json
{
  "signals": {
    "is_cheap": true,
    "is_green": false,
    "charge_now": false,
    "grid_stable": true,
    "cheap_hour_coming": false
  },
  "metadata": {
    "timestamp": "2025-12-25T01:19:13.743713+00:00",
    "user_id": "00000000-0000-0000-0000-000000000000",
    "current_price": 0.0526,
    "daily_avg": 0.0673,
    "renewable_pct": 0.0,
    "balance_delta": 58.0,
    "confidence": "high"
  }
}
```

**Error (401 Unauthorized):**
```json
{
  "detail": "Invalid API key"
}
```

**Error (503 Service Unavailable):**
```json
{
  "detail": "No price data available"
}
```

---

## Signal Definitions

### is_cheap
**Definition:** Current electricity price is below the 24-hour rolling average

**Logic:**
```python
current_price < daily_average_price
```

**Use Case:** Trigger energy-intensive tasks (charging, heating, dishwasher) during low-price periods

**Example:**
- Current: €0.0526/kWh
- 24h avg: €0.0673/kWh
- Result: `true` (26% cheaper than average)

---

### is_green
**Definition:** Renewable energy percentage exceeds threshold (50%)

**Logic:**
```python
renewable_percentage > 50%

# Renewable sources:
# - Biomass (B01)
# - Solar (B16)
# - Wind Offshore (B18)
# - Wind Onshore (B19)
```

**Use Case:** Prioritize energy consumption during high renewable generation periods

**Example:**
- Current renewable: 0%
- Threshold: 50%
- Result: `false`

---

### charge_now
**Definition:** Combined signal for optimal charging (cheap AND moderately green)

**Logic:**
```python
(current_price < daily_average) AND (renewable_percentage > 40%)
```

**Use Case:** Smart EV charging, battery storage optimization

**Example:**
- Price check: ✓ (cheap)
- Green check: ✗ (0% < 40%)
- Result: `false`

---

### grid_stable
**Definition:** Grid balance within acceptable range (±500 MW)

**Logic:**
```python
|grid_balance_delta| < 500 MW
```

**Use Case:** Avoid adding load during grid stress, participate in demand response

**Example:**
- Grid delta: 58 MW
- Threshold: 500 MW
- Result: `true` (grid stable)

**Notes:**
- Positive delta = surplus (more generation than consumption)
- Negative delta = deficit (more consumption than generation)

---

### cheap_hour_coming
**Definition:** Price dip expected in next 3 hours (>10% cheaper)

**Logic:**
```python
min(next_3h_prices) < (current_price * 0.9)
```

**Use Case:** Delay non-urgent tasks if significant price drop is imminent

**Example:**
- Current price: €0.0526/kWh
- Next 3h min: €0.0530/kWh
- Result: `false` (no significant dip)

---

## Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| timestamp | ISO 8601 | Response generation time (UTC) |
| user_id | UUID | User identifier from request |
| current_price | float | Current electricity price (EUR/kWh) |
| daily_avg | float | 24h rolling average price (EUR/kWh) |
| renewable_pct | float | Current renewable generation percentage |
| balance_delta | float | Grid balance delta (MW) |
| confidence | string | Data quality indicator ("high", "medium", "low") |

---

## Thresholds (Default Values)

| Signal | Threshold | Configurable |
|--------|-----------|--------------|
| is_cheap | price < 24h avg | No* |
| is_green | renewable > 50% | Future |
| charge_now | cheap + renewable > 40% | Future |
| grid_stable | \|delta\| < 500 MW | Future |
| cheap_hour_coming | min(3h) < current * 0.9 | No* |

*Dynamic thresholds based on market data

---

## Data Sources

**Pricing:**
- Table: `norm_entso_e_a44`
- Source: ENTSO-E Day-Ahead Market
- Update: Every 15 minutes
- Retention: 72 hours

**Generation Mix:**
- Table: `norm_entso_e_a75`
- Source: ENTSO-E Actual Generation
- Update: Every 15 minutes
- Resolution: Per production type

**Grid Balance:**
- Table: `norm_tennet_balance`
- Source: TenneT System Balance
- Update: Every 5 minutes
- Coverage: Netherlands only

---

## Rate Limits

- **Requests:** 100 per hour per API key
- **Burst:** 10 requests per minute
- **Response Time:** < 100ms (p95)
- **Cache TTL:** 5 minutes

---

## Error Handling

**Common Errors:**

| Code | Reason | Solution |
|------|--------|----------|
| 401 | Missing/invalid API key | Include valid X-API-Key header |
| 422 | Invalid user_id format | Use valid UUID format |
| 503 | No data available | Retry after 5 minutes |
| 429 | Rate limit exceeded | Reduce request frequency |

**Retry Strategy:**
- 503: Exponential backoff (5s, 10s, 20s)
- 429: Wait until rate limit reset
- 5xx: Max 3 retries

---

## Integration Examples

### Home Assistant Automation

```yaml
automation:
  - alias: "Charge EV when optimal"
    trigger:
      - platform: state
        entity_id: binary_sensor.synctacles_charge_now
        to: "on"
    condition:
      - condition: numeric_state
        entity_id: sensor.ev_battery_level
        below: 80
    action:
      - service: switch.turn_on
        target:
          entity_id: switch.ev_charger
```

### Python Client

```python
import requests

API_KEY = "your_api_key"
USER_ID = "123e4567-e89b-12d3-a456-426614174000"

response = requests.get(
    f"https://api.synctacles.io/api/v1/signals",
    params={"user_id": USER_ID},
    headers={"X-API-Key": API_KEY}
)

signals = response.json()["signals"]

if signals["charge_now"]:
    print("🔋 Optimal time to charge!")
```

---

## Changelog

### v1.0.0 (2025-12-25)
- Initial release
- 5 binary signals implemented
- Real-time data integration
- Netherlands coverage only

---

## Support

**Documentation:** https://docs.synctacles.io  
**API Status:** https://status.synctacles.io  
**Contact:** support@synctacles.io

---

*Last Updated: 2025-12-25*
