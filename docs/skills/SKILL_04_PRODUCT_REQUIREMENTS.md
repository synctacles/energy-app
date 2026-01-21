# SKILL 4 — PRODUCT REQUIREMENTS

Features, Sensors, and Product Vision
**Version: 3.0 (2026-01-19) - Generated from source code**

> **KISS Migration v2.0.0:** Focus on actionable price-based recommendations.
> P1 Live Cost and Savings sensors added in v2.2.0.

---

## EXECUTIVE SUMMARY

**SYNCTACLES** provides Dutch energy price data and actionable recommendations:
- **GO/WAIT/AVOID** - Simple action recommendation
- **Best Window Finder** - Optimal consumption window
- **Tomorrow Preview** - Next day outlook
- **Live Cost** - Real-time electricity cost calculation
- **Savings** - Track savings vs peak price

---

## HOME ASSISTANT SENSORS

### Core Sensors (Server Data)

| Sensor | Entity ID | State | Type |
|--------|-----------|-------|------|
| Electricity Price | `sensor.energy_insights_nl_electricity_price` | €/kWh | float |
| Cheapest Hour | `sensor.energy_insights_nl_cheapest_hour` | HH:00 | string |
| Most Expensive Hour | `sensor.energy_insights_nl_most_expensive_hour` | HH:00 | string |
| Energy Action | `sensor.energy_insights_nl_energy_action` | GO/WAIT/AVOID | string |
| Best Window | `sensor.energy_insights_nl_best_window` | HH:00 (start) | string |
| Tomorrow Preview | `sensor.energy_insights_nl_tomorrow_preview` | FAVORABLE/NORMAL/EXPENSIVE/PENDING | string |

### P1/Power Sensors (Requires Power Sensor Config)

| Sensor | Entity ID | State | Type |
|--------|-----------|-------|------|
| Live Cost | `sensor.energy_insights_nl_live_cost` | €/h | float |
| Savings | `sensor.energy_insights_nl_savings` | € | float |

### BYO Sensors (Requires Enever API Key)

| Sensor | Entity ID | State | Type |
|--------|-----------|-------|------|
| Prices Today (BYO) | `sensor.energy_insights_nl_prices_today_byo` | "Today" | string |
| Prices Tomorrow (BYO) | `sensor.energy_insights_nl_prices_tomorrow_byo` | "Tomorrow" / "Not Available" | string |

---

## SENSOR DETAILS

### 1. Electricity Price Sensor

**File:** `sensor.py:404-517`

| Property | Value |
|----------|-------|
| State | Current price in €/kWh (e.g., `0.2509`) |
| Device class | `monetary` |
| State class | `measurement` |
| Unit | €/kWh |

**Attributes:**
```yaml
source: "Frank Direct"         # Data source
quality: "FRESH"               # Data quality
daily_average_kwh: 0.2401      # Today's average
daily_min_kwh: 0.1982          # Today's minimum
daily_max_kwh: 0.3421          # Today's maximum
best_hours: "02:00, 03:00, 04:00, 05:00"  # 4 cheapest hours
leverancier: "Zonneplan"       # Only if Enever BYO configured
```

---

### 2. Cheapest Hour Sensor

**File:** `sensor.py:520-592`

| Property | Value |
|----------|-------|
| State | Hour as string (e.g., `"03:00"`) |
| Icon | `mdi:clock-time-three` |

**Attributes:**
```yaml
price_eur_mwh: 198.20          # Price in EUR/MWh
timestamp: "2026-01-19T03:00:00+00:00"
source: "Frank Direct"
```

---

### 3. Most Expensive Hour Sensor

**File:** `sensor.py:595-659`

| Property | Value |
|----------|-------|
| State | Hour as string (e.g., `"18:00"`) |
| Icon | `mdi:clock-time-six` |

**Attributes:**
```yaml
price_eur_mwh: 342.10          # Price in EUR/MWh
timestamp: "2026-01-19T18:00:00+00:00"
source: "Frank Direct"
```

---

### 4. Energy Action Sensor

**File:** `sensor.py:662-818`

| Property | Value |
|----------|-------|
| State | `GO` / `WAIT` / `AVOID` |
| Icon | Dynamic based on action |

**Action Thresholds (configurable):**
- **GO**: Price < daily average - 15% (default)
- **WAIT**: Price within ±15-20% of average
- **AVOID**: Price > daily average + 20% (default)

**Attributes:**
```yaml
reason: "Price -18% vs average"
action_description: "Good time to use electricity"
go_threshold: -15              # User-configurable
avoid_threshold: 20            # User-configurable
next_go_time: "03:00"          # Next GO period
next_go_price: 0.1982          # Price at next GO
price_level_percent: -18.5     # Current vs average
best_hours_today: ["02:00", "03:00", "04:00", "05:00"]
```

---

### 5. Best Window Sensor

**File:** `sensor.py:1013-1068`

| Property | Value |
|----------|-------|
| State | Start hour (e.g., `"02:00"`) |
| Icon | `mdi:clock-star` |

**Attributes:**
```yaml
start_time: "02:00"
end_time: "05:00"
start_iso: "2026-01-19T02:00:00+00:00"
end_iso: "2026-01-19T05:00:00+00:00"
duration_hours: 3
average_price_eur_kwh: 0.2012
total_cost_estimate_eur: 0.6036  # For 1kW load over duration
runner_up_start: "13:00"
runner_up_end: "16:00"
runner_up_average_price: 0.2156
```

---

### 6. Tomorrow Preview Sensor

**File:** `sensor.py:1071-1147`

| Property | Value |
|----------|-------|
| State | `FAVORABLE` / `NORMAL` / `EXPENSIVE` / `PENDING` |
| Icon | `mdi:calendar-arrow-right` |

**State Meanings:**
- **FAVORABLE**: Tomorrow cheaper than today (good to wait)
- **NORMAL**: Similar prices
- **EXPENSIVE**: Tomorrow more expensive
- **PENDING**: Prices not yet available (usually before 13:00 CET)

**Attributes:**
```yaml
date: "2026-01-20"
cheapest_hour: "04:00"
cheapest_price_eur_kwh: 0.1845
most_expensive_hour: "17:00"
most_expensive_price_eur_kwh: 0.3012
average_price_eur_kwh: 0.2234
best_3h_start: "03:00"
best_3h_end: "06:00"
best_3h_average: 0.1901
hours_available: 24
message: "Tomorrow's prices not yet available"  # Only when PENDING
```

---

### 7. Live Cost Sensor

**File:** `sensor.py:1153-1319`

**Requires:** Power sensor configuration in Options Flow

| Property | Value |
|----------|-------|
| State | Current cost in €/h (e.g., `0.1234`) |
| Device class | `monetary` |
| Unit | €/h |

**Calculation (line 1240-1242):**
```python
cost_per_hour = (power_w / 1000) * price_kwh
```

**Attributes:**
```yaml
power_sensor_entity: "sensor.p1_meter_power"
quality: "FRESH"
current_power_w: 1250.0
current_price_eur_kwh: 0.2509
cost_today_so_far: 1.23        # Running total
projected_cost_today: 4.56      # Estimated end-of-day
status: "consuming"             # "consuming" or "exporting"
```

**Special Behavior:**
- Shows negative values when exporting power
- Updates every 5 seconds (synced with power sensor)
- Falls back to server price if Enever unavailable

---

### 8. Savings Sensor

**File:** `sensor.py:1321-1583`

**Requires:** Power sensor configuration

| Property | Value |
|----------|-------|
| State | Today's savings in € (e.g., `0.45`) |
| Device class | `monetary` |
| State class | `total_increasing` |
| Unit | € |

**Calculation (line 1499-1504):**
```python
kwh_estimate = (power_w / 1000) * (5 / 3600)  # 5-sec interval
actual_cost = kwh_estimate * current_price
peak_cost = kwh_estimate * peak_price
savings = peak_cost - actual_cost
```

**Attributes:**
```yaml
savings_today: 0.45
cost_actual_today: 2.34         # What you actually paid
cost_if_peak_today: 2.79        # What you would have paid at peak
smart_usage_score_today: 84     # Percentage (0-100)
savings_month: 12.50            # Month-to-date
tracking_since: "2026-01-01"
peak_price_source: "Enever BYO" # or "Server dashboard"
```

**Peak Price Resolution:**
1. Enever BYO `prices_today` → `max(price_eur_kwh)`
2. Server dashboard → `current.most_expensive_price_eur_kwh`

**RestoreEntity:** Savings persist across HA restarts.

---

### 9. Prices Today Sensor (BYO)

**File:** `sensor.py:821-910`

**Requires:** Enever API key

| Property | Value |
|----------|-------|
| State | "Today" |
| Icon | `mdi:calendar-today` |

**Attributes:**
```yaml
hours:
  "00": {price_eur_kwh: 0.22, color: "green", status: "GO"}
  "01": {price_eur_kwh: 0.21, color: "green", status: "GO"}
  ...
  "18": {price_eur_kwh: 0.34, color: "red", status: "AVOID"}
daily_average: 0.2401
daily_min: 0.1982
daily_max: 0.3421
leverancier: "Zonneplan"
resolution_minutes: 60         # or 15 for supporters
```

---

### 10. Prices Tomorrow Sensor (BYO)

**File:** `sensor.py:913-1006`

**Requires:** Enever API key

| Property | Value |
|----------|-------|
| State | "Tomorrow" or "Not Available" |
| Icon | `mdi:calendar-arrow-right` |

**Attributes:** Same as Prices Today + `note` when unavailable.

**Availability:** Usually after 13:00-15:00 CET.

---

## CONFIG FLOW

### Step 1: User Credentials

**File:** `config_flow.py:117-190`

| Field | Type | Required | Default |
|-------|------|----------|---------|
| `api_url` | string | Yes | `https://energy.synctacles.com/api` |
| `api_key` | string | Yes | - |
| `enever_token` | string | No | `""` |
| `enever_leverancier` | dropdown | No | Zonneplan |
| `enever_supporter` | bool | No | False |

### Step 2: Power Sensor (Optional)

**File:** `config_flow.py:193-263`

| Field | Type | Required | Default |
|-------|------|----------|---------|
| `power_sensor` | entity selector | No | Auto-detected |
| `skip_power_sensor` | bool | No | False |

**Auto-detection priority:**
1. HomeWizard P1 meter (`sensor.*p1*power*`)
2. Any power sensor (`device_class: power`)

### Options Flow (Reconfigure)

**File:** `config_flow.py:266-373`

All fields from Step 1 + 2, plus:

| Field | Type | Default | Range |
|-------|------|---------|-------|
| `go_threshold` | int | -15 | -50 to 0 |
| `avoid_threshold` | int | 20 | 0 to 100 |

---

## POLLING INTERVALS

| Coordinator | Interval | Endpoint |
|-------------|----------|----------|
| ServerDataCoordinator | 15 min | `/api/v1/dashboard` |
| EneverDataCoordinator | 1 hour | Enever API |
| Live Cost updates | 5 sec | Tracks power sensor |
| Savings updates | 5 sec | Tracks power sensor |

### Hourly Scheduling (v2.4.0)

Naast de normale polling interval, scheduled de `ServerDataCoordinator` ook callbacks op elk uur-transitiepunt. Dit zorgt ervoor dat de Energy Action (GO/WAIT/AVOID) **instant** update wanneer de elektriciteitsprijs verandert, in plaats van tot 15 minuten te wachten op de volgende poll.

```
Normale flow (vóór v2.4.0):
  13:50 poll → prijs = WAIT
  14:00 prijs verandert → GO (maar HA weet het nog niet)
  14:05 poll → GO (5+ min vertraging)

Met hourly scheduling (v2.4.0):
  13:50 poll → prijs = WAIT, schedules timer voor 14:00
  14:00:00 timer triggert → instant refresh → GO
```

**Implementatie:** `_schedule_hourly_updates()` zet timers voor de komende 24 uur. Elke timer triggert `async_request_refresh()` op de coordinator.

---

## FEATURE MATRIX

| Feature | Required Config | Sensors Created |
|---------|-----------------|-----------------|
| Basic pricing | API key only | 6 core sensors |
| P1/Live Cost | + Power sensor | + 2 sensors (Live Cost, Savings) |
| Enever BYO | + Enever token | + 2 sensors (Today, Tomorrow) |
| 15-min resolution | + Supporter tier | Enhanced Enever sensors |

---

## VERSIONING

| Component | Current | Released |
|-----------|---------|----------|
| HA Integration | **2.2.3** | 2026-01-19 |
| Server API | v1 (no explicit version) | - |

**Changelog Highlights:**
- v2.2.0: P1 Live Cost + Savings sensors
- v2.2.1: Enever BYO priority + anomaly detection
- v2.2.2: Options flow UI fixes
- v2.2.3: 5 new Enever providers + translations

---

## ENEVER PROVIDERS (24)

| Provider | 15-min Support |
|----------|----------------|
| Tibber | Yes (supporter) |
| Zonneplan | Yes (supporter) |
| Frank Energie | Yes (supporter) |
| All others | 60-min only |

**Note:** Eneco does NOT offer dynamic electricity, only gas.

---

## AUTOMATION EXAMPLES

### Basic GO/WAIT/AVOID

```yaml
automation:
  - alias: "Start dishwasher on GO"
    trigger:
      - platform: state
        entity_id: sensor.energy_insights_nl_energy_action
        to: "GO"
    condition:
      - condition: time
        after: "08:00:00"
        before: "22:00:00"
    action:
      - service: switch.turn_on
        target:
          entity_id: switch.dishwasher
```

### Best Window Charging

```yaml
automation:
  - alias: "Schedule EV charging in best window"
    trigger:
      - platform: time
        at: "00:05:00"
    action:
      - service: input_datetime.set_datetime
        target:
          entity_id: input_datetime.ev_charge_start
        data:
          time: "{{ state_attr('sensor.energy_insights_nl_best_window', 'start_time') }}"
```

### Tomorrow-Aware Decisions

```yaml
automation:
  - alias: "Notify if tomorrow is favorable"
    trigger:
      - platform: state
        entity_id: sensor.energy_insights_nl_tomorrow_preview
        to: "FAVORABLE"
    action:
      - service: notify.mobile_app
        data:
          message: "Tomorrow's electricity is cheaper - consider waiting for big appliances!"
```

### Live Cost Alerts

```yaml
automation:
  - alias: "Alert when spending fast"
    trigger:
      - platform: numeric_state
        entity_id: sensor.energy_insights_nl_live_cost
        above: 0.50  # €0.50/hour
    action:
      - service: notify.mobile_app
        data:
          message: "High electricity usage: €{{ states('sensor.energy_insights_nl_live_cost') }}/hour"
```

---

## RELATED SKILLS

- **SKILL 2**: Architecture (how features are implemented)
- **SKILL 3**: Coding Standards
- **SKILL 6**: Data Sources (where data comes from)
- **SKILL 13**: Logging & Diagnostics

---

*Generated from source code: 2026-01-19*
*Scanned files: sensor.py, config_flow.py, const.py, enever_client.py, __init__.py*
