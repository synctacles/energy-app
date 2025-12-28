# SYNCTACLES User Guide

**Get real-time Dutch energy data in Home Assistant**

---

## Quick Start (5 minutes)

### 1. Create Account

Visit: https://synctacles.io/docs  
Click: **Sign Up**

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
2. Add: `https://github.com/DATADIO/synctacles-ha`
3. Category: Integration
4. Search: **SYNCTACLES** → Install
5. Restart Home Assistant

**Method 2: Manual**

```bash
# SSH into Home Assistant
cd /config/custom_components
git clone https://github.com/DATADIO/synctacles-ha.git synctacles

# Restart HA
ha core restart
```

---

### 3. Configure Integration

1. Settings → Devices & Services → **Add Integration**
2. Search: **SYNCTACLES**
3. Enter:
   - **API Endpoint:** `https://synctacles.io`
   - **API Key:** (paste from step 1)
4. Submit → **3 sensors created** ✓

---

### 4. Verify Installation

Navigate: **Settings → Devices & Services → SYNCTACLES**

**Expected entities:**
- `sensor.synctacles_generation_total`
- `sensor.synctacles_load_actual`
- `sensor.synctacles_balance_delta`

**Check state:**
```yaml
# Developer Tools → States
sensor.synctacles_generation_total:
  state: 12345  # MW
  attributes:
    quality_status: OK  # ✅ Safe for automation
    source: ENTSO-E
    solar_mw: 0.0
    wind_offshore_mw: 1234.5
    gas_mw: 3210.8
    ...
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
- API Reference: https://synctacles.io/docs/api-reference
- Troubleshooting: https://synctacles.io/docs/troubleshooting
- Developer Guide: https://synctacles.io/docs/developer

**Community:**
- GitHub Issues: https://github.com/DATADIO/synctacles-ha/issues
- Home Assistant Community: [SYNCTACLES thread](https://community.home-assistant.io)

**Contact:**
- Email: support@synctacles.io
- Response time: < 24 hours

---

## FAQ

**Q: Does SYNCTACLES work outside Netherlands?**  
A: V1 = Netherlands only. V2 (2026) = Belgium, Germany, France.

**Q: Is data real-time?**  
A: Near real-time (< 15 min lag from ENTSO-E/TenneT sources).

**Q: Can I use SYNCTACLES for trading?**  
A: No - data is for monitoring only, not financial decisions.

**Q: What happens if I exceed rate limit?**  
A: API returns 429 error, HA sensors show "unavailable" until reset (00:00 UTC).

**Q: How accurate is generation mix data?**  
A: Primary source (ENTSO-E) = 95%+ accuracy. Fallback (Energy-Charts) = 90-93%.

**Q: Does SYNCTACLES store my data?**  
A: Only email + API usage logs (30-day retention). No personal energy data stored.

---

**Last Updated:** 2025-12-21  
**Version:** 1.0.0
