# Energy Insights NL User Guide

**Get real-time Dutch energy data in Home Assistant**

---

## Quick Start (5 minutes)

### 1. Create Account

Visit: Your API provider's website or see [Deployment Guide](deployment.md) to host your own instance.

Enter your email → Receive API key **immediately**

⚠️ **CRITICAL:** Copy your API key now - you cannot retrieve it later!

**Example response:**
```json
{
  "api_key": "a6d003acdcf9b1524546cdeca215e4db8e9c71a588f31e764f52bd5ac595909f",
  "license_key": "0dbccc43-f012-4604-8adc-e61ef13c366d"
}
```

Store `api_key` securely (password manager, .env file).

---

### 2. Install Home Assistant Integration

**Method 1: HACS (Recommended)**

1. HACS → Integrations → **⋮** → Custom repositories
2. Add: `https://github.com/DATADIO/ha-energy-insights-nl`
3. Category: Integration
4. Search: **Energy Insights NL** → Install
5. Restart Home Assistant

**Method 2: Manual**

```bash
# SSH into Home Assistant
cd /config/custom_components
git clone https://github.com/DATADIO/ha-energy-insights-nl.git energy_insights_nl

# Restart HA
ha core restart
```

---

### 3. Configure Integration

1. Settings → Devices & Services → **Add Integration**
2. Search: **Energy Insights NL**
3. Enter:
   - **API Endpoint:** Your API server URL (e.g., `http://192.168.1.100:8000`)
   - **API Key:** (paste from step 1)
   - **TenneT API Key (optional):** Your personal TenneT key for balance data
4. Submit → **Sensors created** ✓

---

### 4. Verify Installation

Navigate: **Settings → Devices & Services → Energy Insights NL**

**Expected sensors (without TenneT key):**
- `sensor.energy_insights_nl_generation_total` - Current electricity generation by source (MW)
- `sensor.energy_insights_nl_load_actual` - Current grid load (MW)

**Additional sensors (with TenneT BYO-key):**
- `sensor.energy_insights_nl_balance_delta` - Grid balance (MW, +surplus/-deficit)
- `sensor.energy_insights_nl_grid_stress` - Grid stress level (0-100)

**Check state:**
```yaml
# Developer Tools → States
sensor.energy_insights_nl_generation_total:
  state: 12345  # MW (total_mw)
  attributes:
    quality_status: FRESH  # ✅ Safe for automation
    source: ENTSO-E
    solar_mw: 0.0
    wind_onshore_mw: 2150.5
    wind_offshore_mw: 1234.5
    gas_mw: 3210.8
    nuclear_mw: 485.0
    biomass_mw: 375.0
    renewable_percentage: 42.3
    age_seconds: 245
    confidence_score: 92
```

---

## Dashboard Setup

### Energy Mix Card (ApexCharts)

**Install:** HACS → Frontend → ApexCharts Card

```yaml
type: custom:apexcharts-card
header:
  show: true
  title: Dutch Energy Mix (Live)
series:
  - entity: sensor.synctacles_generation_total
    attribute: solar_mw
    name: Solar
    color: yellow
  - entity: sensor.synctacles_generation_total
    attribute: wind_offshore_mw
    name: Wind Offshore
    color: blue
  - entity: sensor.synctacles_generation_total
    attribute: wind_onshore_mw
    name: Wind Onshore
    color: lightblue
  - entity: sensor.synctacles_generation_total
    attribute: gas_mw
    name: Gas
    color: orange
  - entity: sensor.synctacles_generation_total
    attribute: nuclear_mw
    name: Nuclear
    color: purple
```

### Simple Entities Card

```yaml
type: entities
title: NL Energy Status
entities:
  - entity: sensor.synctacles_generation_total
    name: Total Generation
    icon: mdi:lightning-bolt
  - entity: sensor.synctacles_load_actual
    name: Total Load
    icon: mdi:flash
  - entity: sensor.synctacles_balance_delta
    name: Grid Balance
    icon: mdi:scale-balance
```

---

## TenneT BYO-Key Setup (Optional)

Real-time grid balance data requires your personal TenneT API key.

### Why BYO-Key?

TenneT's API license prohibits server-side redistribution. Your personal key fetches data directly to your Home Assistant - it never passes through SYNCTACLES servers.

**Without TenneT key:** 2 sensors available (Generation + Load)
**With TenneT key:** 4 sensors available (+ Balance Delta + Grid Stress)

### Get Your TenneT Key

1. Visit: https://www.tennet.eu/developer-portal/
2. Create account (free)
3. Generate API key under "API Credentials"
4. Copy key securely (format: Bearer token)

### Configure in Home Assistant

1. Settings → Devices & Services → Energy Insights NL
2. Click **Configure** next to the integration
3. Enter **TenneT API Key** field
4. Submit → Restart integration

**Expected result:**
- `sensor.energy_insights_nl_balance_delta` appears ✓
- `sensor.energy_insights_nl_grid_stress` appears ✓

### Available Sensors (with BYO-key)

| Sensor | Description | Update Interval |
|--------|-------------|-----------------|
| `sensor.energy_insights_nl_balance_delta` | Grid balance MW (+surplus/-deficit) | 5 minutes |
| `sensor.energy_insights_nl_grid_stress` | Grid stress indicator (0-100) | 5 minutes |

**Balance Delta interpretation:**
- **Positive values** (+) = Surplus generation (Netherlands exports)
- **Negative values** (-) = Deficit (Netherlands imports)
- **±0-50 MW** = Balanced grid
- **±200+ MW** = Significant imbalance (grid stress)

### Troubleshooting TenneT

**Sensors show "unavailable":**
1. Verify TenneT key is correctly copied (no spaces)
2. Check HA logs: Settings → System → Logs → Search "tennet"
3. Confirm key hasn't expired at TenneT Developer Portal
4. TenneT has rate limit of 100 requests/minute (HA polls every 5 min = safe)

**Common TenneT errors:**
- `401 Unauthorized` → Invalid or expired key
- `429 Too Many Requests` → Rate limit hit (wait 1 min)
- `403 Forbidden` → Key lacks required permissions

---

## Automations

### Alert: High Renewable Energy

Trigger smart devices when renewables > 60%

```yaml
automation:
  - alias: "Charge EV during high renewables"
    trigger:
      - platform: template
        value_template: >
          {{ (state_attr('sensor.synctacles_generation_total', 'solar_mw') | float +
              state_attr('sensor.synctacles_generation_total', 'wind_offshore_mw') | float +
              state_attr('sensor.synctacles_generation_total', 'wind_onshore_mw') | float) /
             states('sensor.synctacles_generation_total') | float > 0.6 }}
    condition:
      - condition: state
        entity_id: sensor.synctacles_generation_total
        attribute: quality_status
        state: "OK"  # ⚠️ Only automate on OK quality
    action:
      - service: switch.turn_on
        target:
          entity_id: switch.ev_charger
      - service: notify.mobile_app
        data:
          message: "EV charging: 60%+ renewable energy"
```

### Alert: Grid Stress

Notify when balance delta indicates grid issues

```yaml
automation:
  - alias: "Alert: Grid imbalance"
    trigger:
      - platform: numeric_state
        entity_id: sensor.synctacles_balance_delta
        above: 300  # MW surplus
      - platform: numeric_state
        entity_id: sensor.synctacles_balance_delta
        below: -300  # MW deficit
    action:
      - service: notify.mobile_app
        data:
          message: >
            Grid imbalance detected: 
            {{ states('sensor.synctacles_balance_delta') }} MW
```

---

## Quality Status Guide

Every sensor has `quality_status` attribute:

| Status | Meaning | Automation Safe? |
|--------|---------|------------------|
| **OK** | Fresh data (< 15 min) | ✅ Yes |
| **FALLBACK** | Secondary source (Energy-Charts) | ⚠️ Caution |
| **STALE** | Old data (15 min - 1 hour) | ⚠️ Verify first |
| **NO_DATA** | No data available | ❌ Do NOT automate |

**Best Practice:**
```yaml
condition:
  - condition: state
    entity_id: sensor.synctacles_generation_total
    attribute: quality_status
    state: "OK"
```

---

## Rate Limits

**Free Tier:**
- 1000 requests/day
- Resets: 00:00 UTC daily
- Home Assistant: ~96 requests/day (15 min polling)
- Margin: **10x buffer** ✓

**Check usage:**
```bash
curl -H "X-API-Key: YOUR_KEY" https://synctacles.io/auth/stats
```

**Response:**
```json
{
  "usage_today": 48,
  "remaining_today": 952,
  "rate_limit_daily": 1000
}
```

---

## Troubleshooting

### Sensors Show "Unavailable"

**Check 1:** API key valid
```bash
curl -H "X-API-Key: YOUR_KEY" https://synctacles.io/health
```

**Expected:** `{"status":"ok"}`

**Check 2:** Integration logs
```
Settings → System → Logs
Search: "synctacles"
```

**Common errors:**
- `401 Unauthorized` → API key invalid (regenerate)
- `429 Rate Limit` → Exceeded 1000 requests/day (wait for reset)
- `503 Service Unavailable` → API maintenance (check status page)

---

### Sensors Show Old Data

**Check quality_status:**
```yaml
# Developer Tools → States
sensor.synctacles_generation_total:
  attributes:
    quality_status: STALE  # ⚠️ Old data
    data_age_seconds: 3600  # 1 hour old
```

**Resolution:**
- STALE (15 min - 1h): Wait for next API update (15 min cycle)
- NO_DATA (> 1h): Check [status page](https://synctacles.io/status)

---

### Integration Not Appearing

**Verify installation:**
```bash
# SSH into Home Assistant
ls /config/custom_components/synctacles/
```

**Expected files:**
```
__init__.py
sensor.py
config_flow.py
const.py
manifest.json
strings.json
```

**If missing:**
```bash
rm -rf /config/custom_components/synctacles
# Reinstall via HACS or manual method
```

---

## Advanced Configuration

### Polling Interval (Default: 15 min)

Reduce API calls by increasing interval:

```yaml
# configuration.yaml
synctacles:
  scan_interval: 1800  # 30 minutes (48 requests/day)
```

### Custom Attributes

Access raw API response:

```yaml
{{ state_attr('sensor.synctacles_generation_total', 'biomass_mw') }}
{{ state_attr('sensor.synctacles_load_actual', 'forecast_mw') }}
{{ state_attr('sensor.synctacles_balance_delta', 'price_eur_mwh') }}
```

---

## API Key Management

### View Stats
```bash
curl -H "X-API-Key: YOUR_KEY" https://synctacles.io/auth/stats
```

### Regenerate Key
```bash
curl -X POST -H "X-API-Key: YOUR_KEY" \
  https://synctacles.io/auth/regenerate-key
```

⚠️ Old key **immediately invalid** - update Home Assistant config!

### Deactivate Account
```bash
curl -X POST -H "X-API-Key: YOUR_KEY" \
  https://synctacles.io/auth/deactivate
```

---

## Support & Resources

**Documentation:**
- [Architecture Guide](ARCHITECTURE.md) - System design
- [API Reference](api-reference.md) - API endpoint docs
- [Troubleshooting Guide](troubleshooting.md) - Common issues
- [Deployment Guide](deployment.md) - Setup instructions

**Community:**
- GitHub Issues: https://github.com/DATADIO/ha-energy-insights-nl/issues
- Home Assistant Community: Check #integrations channel

---

## FAQ

**Q: What countries are supported?**
A: V1 = Netherlands only. Future versions will add Belgium, Germany, France.

**Q: Is data real-time?**
A: Near real-time (< 15 min lag from ENTSO-E/TenneT sources).

**Q: Can I use the data for trading?**
A: No - data is for monitoring only, not financial decisions.

**Q: What happens if I exceed rate limits?**
A: API returns 429 error, HA sensors show "unavailable" until reset (00:00 UTC).

**Q: How accurate is the generation mix?**
A: Primary source (ENTSO-E) = 95%+ accuracy. Fallback (Energy-Charts) = 90-93%.

**Q: Is my energy data stored?**
A: No - we only store API usage logs (30-day retention). Your personal data isn't tracked.

---

**Last Updated:** 2025-12-30
**Version:** 1.0.0
**Status:** Production Ready

**See Also:**
- [Architecture Guide](ARCHITECTURE.md) - How the system works
- [API Reference](api-reference.md) - Complete API documentation
- [Deployment Guide](deployment.md) - Setup your own server
- [Troubleshooting Guide](troubleshooting.md) - Common issues & fixes
