# SKILL 4 — PRODUCT REQUIREMENTS

Features, Capabilities, and Product Vision
Version: 1.0 (2025-12-30)

---

## PURPOSE

Define what SYNCTACLES does, what problems it solves, and what features it provides. This is the "what" and "why" from the user/customer perspective.

---

## EXECUTIVE SUMMARY

**SYNCTACLES** (SYNCTACLES = Synchronized Tactical Application for Collective Load Engagement and Sustainable Energy Logistics) is a Dutch energy data aggregation platform that provides real-time insights into electricity generation, load, and pricing to Home Assistant users and other consumers.

**Problem Solved:** Dutch households and businesses lack easy access to real-time grid data. SYNCTACLES fills this gap by aggregating data from ENTSO-E (European Grid Operator) and TenneT (Dutch System Operator) into a simple REST API, with automatic fallback to modeled data when primary sources fail.

---

## KEY CAPABILITIES

### 1. Real-Time Generation Mix

**What:** Current electricity generation by fuel type in the Dutch grid

**Data Source:** ENTSO-E A75 (updated every 15 minutes)

**Provides:**
- Nuclear generation (MW)
- Solar generation (MW)
- Wind power (onshore + offshore, MW)
- Fossil fuels (coal, gas, oil, MW)
- Hydro generation (MW)
- Biomass generation (MW)
- Waste generation (MW)
- Other sources (MW)

**Use Cases:**
- Home Assistant automation (e.g., run dishwasher when solar peaks)
- Display grid mix on dashboards
- Optimize electricity consumption based on renewable percentage

**Endpoint:** `GET /v1/generation/current`

---

### 2. Grid Load & Forecast

**What:** Current and forecasted electricity consumption

**Data Source:** ENTSO-E A65 (updated every 15 minutes)

**Provides:**
- Actual load (MW)
- Forecasted load (MW)
- Load difference (forecast vs actual)
- Quality score (0.0-1.0)

**Use Cases:**
- Predict grid stress (high load = expensive, polluting)
- Schedule power usage during low-load periods
- Automated load shifting

**Endpoint:** `GET /v1/load/current`

---

### 3. Electricity Prices (Day-Ahead)

**What:** Hourly electricity market prices for today and tomorrow

**Data Source:** ENTSO-E A44 (updated hourly)

**Provides:**
- Current hour price (€/MWh)
- Today's price profile
- Tomorrow's price forecast
- Min/max/average prices

**Use Cases:**
- Price-based automation (charge battery at cheap hours)
- User awareness (know when electricity is expensive)
- Load shifting (move consumption to cheap periods)

**Endpoint:** `GET /v1/prices/today`

---

### 4. Grid Balance Status

**What:** Grid frequency and reserve margin

**Data Source:** TenneT Ingestor (updated every 5 minutes)

**Provides:**
- Grid frequency (Hz, should be ~50 Hz)
- Reserve margin (MW)
- Activation status (normal, increased reserves, etc.)

**Use Cases:**
- Awareness of grid stress events
- Automation triggers (if frequency low, reduce consumption)
- Historical grid stability analysis

**Endpoint:** `GET /v1/balance/current`

---

### 5. Automation Signals

**What:** Pre-computed signals for common automation scenarios

**Calculated From:** Generation, load, and price data

**Provides Signals:**
- `renewable_percentage` - Percentage of renewable generation
- `grid_stress` - Indicator of grid stress (0-100)
- `price_level` - Relative price (cheap/normal/expensive)
- `carbon_intensity` - CO2 emissions per kWh

**Use Cases:**
- Trigger automation based on renewable energy
- Example: "If renewable > 80%, turn on EV charger"
- Example: "If grid stress > 80%, reduce HVAC"

**Endpoint:** `GET /v1/signals/automation`

---

### 6. System Health

**What:** Status of all data sources and system health

**Provides:**
- Overall system status (healthy/degraded/critical)
- Status of each data source (ENTSO-E, TenneT, Energy-Charts)
- Last successful data collection timestamp
- Uptime statistics
- API version

**Use Cases:**
- Alerting (notify if data sources down)
- Debugging (understand which source is failing)
- Health dashboards

**Endpoint:** `GET /health`

---

## HOME ASSISTANT INTEGRATION

### Custom Component

SYNCTACLES provides a Home Assistant custom component that:

1. **Auto-Discovery:** Detects SYNCTACLES API and auto-configures
2. **Entities:** Creates sensors for all data types
3. **Updates:** Polls API every 5 minutes
4. **Fallback:** Uses last-known-good value if API unavailable
5. **Quality:** Displays data quality scores

### Available Entities

```
sensor.generation_nuclear_mw
sensor.generation_solar_mw
sensor.generation_wind_onshore_mw
sensor.generation_wind_offshore_mw
sensor.generation_fossil_fuels_mw
sensor.generation_hydro_mw
sensor.generation_biomass_mw
sensor.generation_waste_mw
sensor.generation_renewable_percentage

sensor.load_current_mw
sensor.load_forecast_mw
sensor.load_quality

sensor.price_current_eur_per_mwh
sensor.price_min_today
sensor.price_max_today
sensor.price_quality

sensor.balance_frequency_hz
sensor.balance_reserve_margin_mw
sensor.balance_status

sensor.synctacles_health_status
sensor.synctacles_renewable_percentage
sensor.synctacles_grid_stress_level
sensor.synctacles_carbon_intensity
```

### Automation Examples

```yaml
# Automation: Charge EV when renewable percentage is high
- alias: "Charge EV when renewable energy high"
  trigger:
    platform: numeric_state
    entity_id: sensor.synctacles_renewable_percentage
    above: 80
  action:
    service: switch.turn_on
    target:
      entity_id: switch.ev_charger

# Automation: Reduce HVAC when grid stressed
- alias: "Reduce HVAC when grid stressed"
  trigger:
    platform: numeric_state
    entity_id: sensor.synctacles_grid_stress_level
    above: 80
  action:
    service: climate.set_temperature
    target:
      entity_id: climate.living_room
    data:
      temperature: 20  # Reduce to 20°C during stress

# Automation: Notify if renewable energy is very high
- alias: "Alert when renewable > 90%"
  trigger:
    platform: numeric_state
    entity_id: sensor.synctacles_renewable_percentage
    above: 90
  action:
    service: notify.telegram
    data:
      message: "Dutch grid is 90%+ renewable! Now is the time to use electricity."
```

---

## CURRENT FEATURE SET (MVP)

### Implemented (Phase 1-3)

- ✅ ENTSO-E A75 data collection (generation)
- ✅ ENTSO-E A65 data collection (load)
- ✅ ENTSO-E A44 data collection (prices)
- ✅ TenneT balance data collection
- ✅ 3-layer pipeline (Collectors → Importers → Normalizers)
- ✅ PostgreSQL storage with quality metadata
- ✅ REST API (FastAPI)
- ✅ Home Assistant custom component
- ✅ Automatic fallback to Energy-Charts
- ✅ Systemd integration (timers for scheduling)
- ✅ Multi-tenant deployment support
- ✅ Comprehensive API documentation

### In Development (Phase 4)

- 🔄 Complete documentation suite
- 🔄 Skills migration and consolidation
- 🔄 User guide and troubleshooting
- ✅ API key authentication
- ✅ Rate limiting per tier

### Planned (Phase 7-9)

- 📅 Advanced forecasting (ML-based generation/price prediction)
- 📅 Enhanced automation signals
- 📅 Price-triggered actions
- 📅 Battery scheduling optimization
- 📅 Data marketplace (sell aggregated data)

---

## NON-FUNCTIONAL REQUIREMENTS

### Performance

- API response time: < 100 ms (p95)
- Data collection: < 30 second per source
- Data freshness: < 15 minutes for generation/load
- Database queries: < 500 ms

### Availability

- Target uptime: 99.5% (production)
- Automatic failover to Energy-Charts if ENTSO-E/TenneT down
- Graceful degradation (serve last-known-good if all sources fail)

### Reliability

- Data quality scoring (0.0-1.0) for every data point
- Automatic fallback strategy
- Health checks every 5 minutes
- Alerting for data source failures

### Security

- No secrets in git repository
- Environment-based configuration
- Database credentials protected
- HTTPS/TLS for all external connections
- Per-tenant isolation (no data leakage)
- API key authentication with rate limiting

---

## SUBSCRIPTION TIERS

### Tier Structure

SYNCTACLES uses API key-based authentication with daily rate limits per tier:

| Tier | Rate Limit | Use Case | Notes |
|------|-----------|----------|-------|
| **Beta** | 10,000 req/day | Testing & early access | Default for new users |
| **Free** | 1,000 req/day | Home Assistant integration | Community tier |
| **Paid** | 100,000 req/day | Premium users | Commercial use |
| **Unlimited** | 100,000 req/day | Enterprise | Priority support |

### Authentication

1. **Signup:** `POST /auth/signup` with email
2. **Response:** Receive API key (shown only once!)
3. **Usage:** Include `X-API-Key` header in all API requests
4. **Rate Limits:** Daily counter resets at midnight UTC

### Rate Limit Headers

Responses include rate limit information:

```
X-RateLimit-Limit: 10000          # Daily limit for your tier
X-RateLimit-Remaining: 9876       # Requests remaining today
X-RateLimit-Reset: 1735689600     # Unix timestamp of reset (midnight UTC)
```

### Management Endpoints

- `GET /auth/stats` - View usage and rate limit info
- `POST /auth/regenerate-key` - Generate new key (invalidates old)
- `POST /auth/deactivate` - Deactivate account

### Scalability

- Support multiple concurrent tenants
- Each tenant completely independent
- Scale from 1 server to N servers
- Horizontal scaling via additional tenants

---

## CONSTRAINTS & LIMITATIONS

### Data Availability

- ENTSO-E A75/A65: Every 15 minutes (published ~15 min delayed)
- ENTSO-E A44: Hourly (published day-ahead + intraday updates)
- TenneT: Every 5 minutes
- Energy-Charts: Cached, updated daily

### Geographic Scope

- **Primary:** Dutch electricity grid (operated by TenneT)
- **Data from:** ENTSO-E (European data) filtered for Netherlands
- **Future:** Can be extended to other countries/regions

### Accuracy

- ENTSO-E data: Published as-is (no validation)
- Quality metadata indicates reliability
- Fallback data (Energy-Charts) is modeled/estimated

---

## PRODUCT ROADMAP

### Phase 5: Advanced Monitoring
- Detailed history graphs (7-day, 30-day views)
- Peak/low analysis per hour
- Trend detection (generation/price trends)
- Anomaly alerting

### Phase 6: Enhanced Automation
- Custom signal definitions (user-configurable)
- Price-based actions (buy electricity at X €/MWh)
- Load shifting recommendations
- Battery optimization (when to charge/discharge)

### Phase 7: Forecasting
- ML-based generation forecast (24-48 hours)
- Price forecast improvements
- Demand prediction
- Seasonal trend analysis

### Phase 8: Community Features
- Data sharing (public APIs for community apps)
- Leaderboards (renewable consumption %)
- Community forecasts (crowdsourced predictions)

### Phase 9: Data Marketplace
- Sell aggregated, anonymized data
- API for third-party developers
- Data quality guarantees
- Subscription models

---

## SUCCESS METRICS

### User Adoption

- Home Assistant installation count
- Active API users per month
- Automation rules created (from user bases)

### Data Quality

- Average data quality score (target: > 0.95)
- Fallback activation rate (target: < 1%)
- API uptime (target: > 99.5%)

### User Satisfaction

- Support request volume
- Bug report rate
- Feature request frequency

---

## COMPETITIVE ADVANTAGES

1. **Free & Open:** No subscription required
2. **Real-Time:** 15-minute granularity (vs daily competitors)
3. **Smart Integration:** Works directly with Home Assistant
4. **Reliable:** Automatic fallback prevents data gaps
5. **Transparent:** Shows data quality scores
6. **Extensible:** Custom automation signals

---

## RELATED SKILLS

- **SKILL 2**: Architecture (how features are implemented)
- **SKILL 9**: Installer (how to deploy features)
- **SKILL 10**: Deployment (how to release features)
- **SKILL 6**: Data Sources (where data comes from)
