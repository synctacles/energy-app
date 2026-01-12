# SKILL 15 — CONSUMER PRICE ENGINE

Dual-Source Price Calculation with Frank & Enever Integration
Version: 2.1 (2026-01-12) — Bias Correction (×0.93)

---

## PURPOSE

Document the consumer price calculation engine: how wholesale prices are transformed into consumer prices, the dual-source validation strategy (Frank API + Enever), and the coefficient server architecture.

---

## OVERVIEW

### Problem

ENTSO-E provides wholesale electricity prices (€/MWh), but Home Assistant users need consumer prices (€/kWh) including:
- Energy tax (energiebelasting)
- VAT (BTW 21%)
- Provider markup/margin

### Solution

A coefficient engine that:
1. Uses hourly lookup tables (historical patterns from Enever)
2. Calibrates daily against Frank API (ground truth)
3. Validates with Enever as secondary source
4. Falls back gracefully when sources unavailable

---

## ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────────┐
│  Main API Server (135.181.255.83)                               │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │ Frank API       │    │ Enever Proxy    │                    │
│  │ (Direct, HTTPS) │    │ (via Coeff Srv) │                    │
│  └────────┬────────┘    └────────┬────────┘                    │
│           │                      │                              │
│           └──────────┬───────────┘                              │
│                      ▼                                          │
│           ┌─────────────────────┐                               │
│           │ Dual-Source         │                               │
│           │ Validation          │                               │
│           └─────────┬───────────┘                               │
│                     ▼                                           │
│           ┌─────────────────────┐                               │
│           │ Consumer Price      │                               │
│           │ Calculation         │                               │
│           └─────────────────────┘                               │
└─────────────────────────────────────────────────────────────────┘
                      │
                      │ Internal request
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│  Coefficient Server (91.99.150.36)                              │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │ Enever Proxy    │    │ VPN Split       │                    │
│  │ Endpoint        │───▶│ Tunnel (NL)     │───▶ Enever API     │
│  └─────────────────┘    └─────────────────┘                    │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │ Enever          │    │ PostgreSQL      │                    │
│  │ Collector       │───▶│ Historical Data │                    │
│  └─────────────────┘    └─────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## DATA SOURCES

### Primary: Frank API

**Endpoint:** `https://graphcdn.frankenergie.nl/`

**Method:** GraphQL POST

**Query:**
```graphql
query MarketPrices {
  marketPrices(startDate: "2026-01-11", endDate: "2026-01-11") {
    electricityPrices {
      from
      till
      marketPrice
      marketPriceTax
      sourcingMarkupPrice
      energyTaxPrice
    }
  }
}
```

**Response breakdown:**
```
Total Price = marketPrice + marketPriceTax + sourcingMarkupPrice + energyTaxPrice
            = wholesale   + BTW on WS     + Frank margin        + energy tax
```

**Characteristics:**
- ✅ Free, no auth required
- ✅ Real-time consumer prices
- ✅ Full breakdown available
- ✅ ~250ms response time
- ⚠️ Only Frank Energie prices (not other providers)

**Update schedule:** Daily at 15:05 via cron

---

### Secondary: Enever API (via Coefficient Proxy)

**Why proxy?** Enever may require Dutch IP. Direct access from German server = gray area.

**Coefficient proxy endpoint:** `GET http://91.99.150.36:8080/internal/enever/prices`

**Security:** IP whitelist (only 135.181.255.83)

**Enever endpoints called:**
- `https://enever.nl/api/stroomprijs_vandaag.php?token=XXX`
- `https://enever.nl/api/stroomprijs_morgen.php?token=XXX`

**Coverage:** 25 Dutch energy providers including:
- Frank Energie, Tibber, ANWB, Zonneplan
- EasyEnergie, Budget Energie, Coolblue
- Greenchoice, Eneco, Vattenfall, etc.

**Characteristics:**
- ✅ All major providers
- ✅ Tomorrow prices after 15:00
- ✅ 10,000 requests/month free
- ⚠️ Requires VPN for Dutch IP

**Update schedule:** 2x daily (00:30, 15:30) via systemd timer

---

### Tertiary: Hourly Lookup Table

**Source:** Historical Enever data (2025+)

**Structure:** 24 coefficient values (one per hour)

```python
HOURLY_LOOKUP = {
    0: 0.157, 1: 0.154, 2: 0.152, 3: 0.151,
    4: 0.152, 5: 0.155, 6: 0.162, 7: 0.168,
    8: 0.165, 9: 0.158, 10: 0.154, 11: 0.151,
    12: 0.149, 13: 0.148, 14: 0.150, 15: 0.153,
    16: 0.158, 17: 0.165, 18: 0.172, 19: 0.170,
    20: 0.166, 21: 0.163, 22: 0.161, 23: 0.159
}
```

**Patterns:**
- Evening peak (18-20h): Highest markup
- Solar dip (12-14h): Lowest markup
- Night (00-05h): Low, stable

**CV (Coefficient of Variation):** 6.61% — stable enough for lookup

---

## CALCULATION FLOW

### Daily Calibration (15:05)

```python
# 1. Fetch Frank live price
frank_live = fetch_frank_api()  # e.g., €0.183

# 2. Get lookup value for current hour
frank_lookup = HOURLY_LOOKUP[hour]  # e.g., €0.172

# 3. Calculate correction factor
correction_factor = frank_live / frank_lookup  # e.g., 1.064

# 4. Store for runtime use
save_correction_factor(correction_factor)
```

### Runtime Calculation (Legacy)

```python
def calculate_consumer_price(wholesale_price: float, hour: int) -> float:
    """Calculate consumer price from wholesale."""

    # Get stored correction factor
    correction = get_correction_factor()  # e.g., 1.064

    # Get hourly coefficient
    coefficient = HOURLY_LOOKUP[hour]  # e.g., €0.172

    # Calculate consumer price
    consumer_price = wholesale_price + (coefficient * correction)

    return consumer_price
```

---

## LINEAR REGRESSION MODEL (2026-01-12)

### Why Linear Regression?

The simple coefficient method (`consumer = wholesale × coefficient`) has a fundamental flaw:
it assumes consumer price is purely proportional to wholesale price.

**Reality:** Consumer price = wholesale + fixed costs

```
Consumer Price Components:
├── Variable: Wholesale price (ENTSO-E)
└── Fixed:
    ├── Energy tax (energiebelasting): ~€0.06/kWh
    ├── Grid costs (netbeheer):        ~€0.04/kWh
    ├── VAT (21%):                     ~€0.03/kWh
    └── Provider margin:               ~€0.02/kWh
    ────────────────────────────────────────────
    Total fixed:                       ~€0.15/kWh
```

### The Linear Model

**Formula:**
```
consumer = wholesale × slope + intercept
```

Where:
- **slope** (~1.27): How strongly consumer price responds to wholesale changes
- **intercept** (~€0.147): Fixed costs regardless of wholesale price

### Improvement

| Metric | Old (Coefficient) | New (Slope+Intercept) |
|--------|-------------------|----------------------|
| Average error | 10-12% | **5.1%** |
| Evening peak (17-21) | 24-39% | **5-11%** |
| R² confidence | n/a | 61-96% |

### Database Schema

```sql
CREATE TABLE coefficient_lookup (
    country     VARCHAR(2) NOT NULL DEFAULT 'NL',
    month       INTEGER NOT NULL,        -- 1-12
    day_type    VARCHAR(10) NOT NULL,    -- 'weekday' / 'weekend'
    hour        INTEGER NOT NULL,        -- 0-23
    slope       NUMERIC(10,4) NOT NULL,  -- ~1.0-2.0
    intercept   NUMERIC(10,6) DEFAULT 0, -- ~€0.15/kWh
    sample_size INTEGER DEFAULT 0,
    confidence  INTEGER DEFAULT 89,      -- R² as percentage
    last_calibrated TIMESTAMPTZ,
    PRIMARY KEY (country, month, day_type, hour)
);
```

### API Endpoint

**Request:**
```bash
curl "http://91.99.150.36:8080/coefficient?country=NL&hour=12&day_type=weekday"
```

**Response:**
```json
{
    "country": "NL",
    "slope": 1.1193,
    "intercept": 0.1464,
    "hour": 12,
    "day_type": "weekday",
    "month": 1,
    "confidence": 95,
    "sample_size": 64,
    "source": "lookup"
}
```

### Synctacles Fallback Integration

The fallback manager uses this model in Tier 3/4/5:

```python
# Default fallback (if Coefficient Engine unavailable)
DEFAULT_SLOPE = 1.27
DEFAULT_INTERCEPT = 0.147  # EUR/kWh

# Bias correction (calibrated 2026-01-12, reduces error from 7.58% to 2.11%)
BIAS_CORRECTION = 0.93

# Get model from Coefficient Engine
model = await ConsumerPriceClient.get_price_model(country="NL")
slope = model.get("slope", DEFAULT_SLOPE)
intercept = model.get("intercept", DEFAULT_INTERCEPT)

# Apply: consumer = (wholesale × slope + intercept) × bias_correction
consumer_kwh = (wholesale_kwh * slope + intercept) * BIAS_CORRECTION
```

### Calculation Example

**Input:**
- ENTSO-E price: €93.80/MWh = €0.0938/kWh
- Slope: 1.1193
- Intercept: €0.1464/kWh
- Bias correction: 0.93

**Calculation:**
```
raw      = €0.0938 × 1.1193 + €0.1464
         = €0.1050 + €0.1464
         = €0.2514/kWh

consumer = €0.2514 × 0.93
         = €0.2338/kWh
```

**Validation:**
- Calculated: €0.2338/kWh
- Frank actual: €0.2447/kWh
- Error: -4.5% (within acceptable range)

### Accuracy After Bias Correction

| Metric | Before (raw) | After (×0.93) |
|--------|--------------|---------------|
| Bias | +7.58% | **+0.05%** |
| Average error | 7.58% | **2.11%** |
| P95 error | ~15% | **4.08%** |
| Std deviation | 7.5% | **2.41%** |

### Comparison: SYNCTACLES vs Enever

Validated against 677 samples (30 days Frank API data):

| Source | Error vs Frank API | €/kWh |
|--------|-------------------|-------|
| **SYNCTACLES Model** | **2.11%** | €0.0051 |
| Enever "Frank Energie" | 2.99% | €0.0073 |

**Key finding:** Our model is more accurate than Enever's published Frank prices.

### Known Limitations (Post-Correction)

| Issue | Impact | Details |
|-------|--------|---------|
| Night hours (00-05h) | 2.5-3.0% error | Slightly higher than average |
| Morning volatility | <4% error | Improved from 15% pre-correction |
| Evening peak (17-21h) | <3% error | Improved from 7.4% pre-correction |

**All hours now within acceptable range (<5% error).**

### Improvement Roadmap

**✅ Completed:**
1. ~~**Bias correction** — Multiply result by 0.93~~ **(DONE 2026-01-12)**

**Future Improvements (Low Priority - Model Already Excellent):**
1. **Peak-hour specific models** — Separate coefficients for 17-21h (potential ~0.3% improvement)
2. **Rolling window calibration** — Auto-update bias correction weekly
3. **Multi-provider support** — Coefficients per energy provider (Tibber, Zonneplan, etc.)

### SQL Regression Query

De 576 coefficienten worden berekend met PostgreSQL's native regression functies:

```sql
WITH hourly_wholesale AS (
    -- Aggregeer ENTSO-E 15-min data naar uurlijks
    SELECT
        DATE_TRUNC('hour', timestamp) as hour,
        AVG(price_eur_mwh) / 1000 as wholesale_kwh
    FROM hist_entso_prices
    WHERE country_code = 'NL'
      AND timestamp > NOW() - INTERVAL '90 days'
    GROUP BY DATE_TRUNC('hour', timestamp)
),
paired_data AS (
    -- Koppel consumer prijzen aan wholesale
    SELECT
        e.price_eur_kwh as consumer,
        w.wholesale_kwh as wholesale
    FROM hist_enever_prices e
    JOIN hourly_wholesale w
        ON DATE_TRUNC('hour', e.timestamp) = w.hour
    WHERE e.leverancier = 'Frank Energie'
      AND EXTRACT(HOUR FROM e.timestamp) = {hour}       -- 0-23
      AND EXTRACT(DOW FROM e.timestamp) {dow_filter}    -- IN (0,6) or NOT IN (0,6)
)
SELECT
    REGR_SLOPE(consumer, wholesale) as slope,           -- ~1.0-2.0
    REGR_INTERCEPT(consumer, wholesale) as intercept,   -- ~€0.15/kWh
    POWER(CORR(wholesale, consumer), 2) as r_squared,   -- Confidence 0-1
    COUNT(*) as sample_size
FROM paired_data
```

**PostgreSQL Functies:**
- `REGR_SLOPE(y, x)`: Berekent de helling van de regressielijn
- `REGR_INTERCEPT(y, x)`: Berekent het snijpunt met de y-as
- `CORR(x, y)²`: R² als betrouwbaarheidsmaat (0-1, hoger = beter)

### Recalibratie Script

Om alle 576 coefficienten te hercalibreren:

```bash
ssh coefficient@91.99.150.36

cd /opt/github/coefficient-engine
python3 << 'EOF'
import psycopg2
from datetime import datetime, timezone
from dotenv import load_dotenv
import os

load_dotenv('.env')
conn = psycopg2.connect(os.getenv('DATABASE_URL'))
cur = conn.cursor()

now = datetime.now(timezone.utc)

for month in range(1, 13):
    for day_type in ['weekday', 'weekend']:
        dow = 'IN (0, 6)' if day_type == 'weekend' else 'NOT IN (0, 6)'
        for hour in range(24):
            cur.execute(f"""
                WITH paired AS (
                    SELECT e.price_eur_kwh as c, AVG(w.price_eur_mwh)/1000 as w
                    FROM hist_enever_prices e
                    JOIN hist_entso_prices w
                        ON DATE_TRUNC('hour', e.timestamp) = DATE_TRUNC('hour', w.timestamp)
                    WHERE e.leverancier = 'Frank Energie'
                      AND EXTRACT(HOUR FROM e.timestamp) = {hour}
                      AND EXTRACT(DOW FROM e.timestamp) {dow}
                      AND e.timestamp > NOW() - INTERVAL '90 days'
                      AND w.price_eur_mwh > 0
                    GROUP BY e.timestamp, e.price_eur_kwh
                )
                SELECT REGR_SLOPE(c, w), REGR_INTERCEPT(c, w),
                       POWER(CORR(w, c), 2), COUNT(*)
                FROM paired
            """)
            row = cur.fetchone()
            if row[0]:
                cur.execute("""
                    UPDATE coefficient_lookup
                    SET slope = %s, intercept = %s, confidence = %s,
                        sample_size = %s, last_calibrated = %s
                    WHERE country = 'NL' AND month = %s
                      AND day_type = %s AND hour = %s
                """, (row[0], row[1], int(row[2]*100), row[3], now,
                      month, day_type, hour))

conn.commit()
print("Recalibratie voltooid: 576 coefficienten bijgewerkt")
EOF
```

### Related Documentation

- [HANDOFF_CC_COEFFICIENT_LINEAR_REGRESSION.md](handoffs/2026-01-12-coefficient-model/HANDOFF_CC_COEFFICIENT_LINEAR_REGRESSION.md) - Model validation details
- [HANDOFF_CC_SYNCTACLES_SLOPE_INTERCEPT.md](handoffs/2026-01-12-coefficient-model/HANDOFF_CC_SYNCTACLES_SLOPE_INTERCEPT.md) - Integration implementation

---

## DUAL-SOURCE VALIDATION

### Logic

```python
async def get_validated_consumer_price(hour: int) -> dict:
    """Get consumer price with dual-source validation."""
    
    frank = await fetch_frank_price(hour)
    enever = await fetch_enever_price(hour)
    
    if frank and enever:
        delta = abs(frank - enever)
        if delta < 0.02:  # < €0.02 = agreement
            return {
                "price": frank,
                "source": "frank+enever",
                "confidence": "high"
            }
        else:
            log.warning(f"Price mismatch: Frank={frank}, Enever={enever}")
            return {
                "price": frank,  # Trust Frank as primary
                "source": "frank",
                "confidence": "medium"
            }
    
    elif frank:
        return {"price": frank, "source": "frank", "confidence": "medium"}
    
    elif enever:
        return {"price": enever, "source": "enever", "confidence": "medium"}
    
    else:
        return {
            "price": calculate_from_coefficient(hour),
            "source": "coefficient",
            "confidence": "low"
        }
```

### Confidence Levels

| Level | Meaning | Sources |
|-------|---------|---------|
| `high` | Both sources agree (< €0.02 delta) | Frank + Enever |
| `medium` | Single source available | Frank OR Enever |
| `low` | Calculated from historical patterns | Coefficient only |

---

## FALLBACK CASCADE

```
Priority    Source              Accuracy    Condition
────────────────────────────────────────────────────────
1           Frank + Enever      98%         Both available, agree
2           Frank only          95%         Frank available
3           Enever only         93%         Enever available
4           Cached correction   91%         < 48h old
5           No correction       89%         factor = 1.0
6           Daily average       85%         Emergency fallback
```

---

## ACCURACY

### Per Provider

Using Frank-calibrated coefficient (€0.149):

| Provider | Actual Markup | Our Estimate | Accuracy |
|----------|---------------|--------------|----------|
| Frank Energie | €0.149 | €0.149 | 100% |
| Tibber | €0.145 | €0.149 | 97% |
| NextEnergy | €0.147 | €0.149 | 99% |
| Zonneplan | €0.142 | €0.149 | 95% |
| ANWB | €0.158 | €0.149 | 94% |

**Average: 95% accuracy**

### For Energy Actions

| Use Case | Accuracy |
|----------|----------|
| Cheapest 4 hours correct | 89% |
| Most expensive 4 hours correct | 92% |
| Ranking ±1 position | 96% |
| Savings achieved vs optimal | 91% |

---

## FILES & LOCATIONS

### Main API Server (135.181.255.83)

```
/opt/energy-insights-nl/app/
├── config/
│   └── coefficients.py           # HOURLY_LOOKUP table
├── services/
│   ├── frank_calibration.py      # Frank API wrapper
│   ├── enever_client.py          # Enever proxy client
│   └── consumer_price.py         # Price calculation + validation
└── scripts/
    └── test_enever_integration.py

/etc/cron.d/
└── frank-calibration             # Daily 15:05 update
```

### Coefficient Server (91.99.150.36)

```
/opt/github/coefficient-engine/
├── api/
│   └── main.py                   # FastAPI app
├── routes/
│   └── enever.py                 # Proxy endpoint
├── services/
│   └── enever_client.py          # Enever API calls
└── collectors/
    └── enever_collector.py       # Scheduled data collection

/opt/coefficient/
├── .enever_token                 # API token (not in git)
└── .env                          # DB credentials

/etc/systemd/system/
├── coefficient-api.service       # API service (port 8080)
├── enever-collector.service      # Collector service
└── enever-collector.timer        # Timer (00:30, 15:30)
```

### Database (Coefficient Server)

```sql
-- Historical price storage
CREATE TABLE enever_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    provider VARCHAR(50) NOT NULL,
    hour INTEGER NOT NULL CHECK (hour >= 0 AND hour <= 23),
    price_total DECIMAL(8,5) NOT NULL,
    price_energy DECIMAL(8,5),
    price_tax DECIMAL(8,5),
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(timestamp, provider, hour)
);

CREATE INDEX idx_enever_timestamp ON enever_prices(timestamp);
CREATE INDEX idx_enever_provider ON enever_prices(provider);
```

---

## MONITORING

### Health Checks

```bash
# Coefficient API health
curl -s http://91.99.150.36:8080/health

# Enever proxy health
curl -s http://91.99.150.36:8080/internal/enever/health

# Dual-source validation test
cd /opt/energy-insights-nl
venv/bin/python3 app/scripts/test_enever_integration.py
```

### Alert Conditions

| Condition | Severity | Action |
|-----------|----------|--------|
| Frank API timeout | Warning | Use Enever/cached |
| Enever proxy timeout | Warning | Use Frank/cached |
| Both sources failed | High | Use coefficient fallback |
| Correction factor < 0.85 or > 1.15 | Warning | Investigate |
| Frank vs Enever delta > €0.02 | Info | Log, monitor |
| Frank vs Enever delta > €0.05 | Warning | Investigate data quality |

---

## VPN CONFIGURATION

Coefficient server uses WireGuard split tunnel for Enever access:

```
┌────────────────────────────────────────┐
│  Coefficient Server (Germany)          │
│                                        │
│  Traffic routing:                      │
│  • SSH, PostgreSQL, API → eth0 (direct)│
│  • Enever (84.46.252.107) → pia-split  │
│  • All other → eth0 (direct)           │
└────────────────────────────────────────┘
                │
                │ Only Enever traffic
                ▼
┌────────────────────────────────────────┐
│  PIA VPN Exit (Amsterdam, NL)          │
│  IP: 158.173.21.230                    │
└────────────────────────────────────────┘
                │
                ▼
┌────────────────────────────────────────┐
│  Enever API (sees Dutch IP)            │
│  84.46.252.107                         │
└────────────────────────────────────────┘
```

**Config:** `/etc/wireguard/pia-split.conf`

**Management:**
```bash
# Status
sudo wg show pia-split

# Restart
sudo systemctl restart wg-quick@pia-split

# Verify routing
ip route get 84.46.252.107
# Expected: dev pia-split
```

See SKILL_10_COEFFICIENT_VPN.md for full documentation.

---

## BUSINESS VALUE

### Historical Data Asset

The Enever collector builds a unique dataset:
- 25 Dutch providers
- Hourly granularity
- 3+ years historical (growing)

**Potential B2B applications:**
- EV fleet optimization
- Energy comparison analytics
- Academic research
- Provider margin analysis

### Competitive Moat

| Data Point | ENTSO-E | CBS | Comparison Sites | SYNCTACLES |
|------------|---------|-----|------------------|------------|
| Wholesale prices | ✅ | ❌ | ❌ | ✅ |
| Consumer prices | ❌ | Monthly avg | Current only | ✅ Hourly |
| Per provider | ❌ | ❌ | Current only | ✅ Historical |
| API access | ✅ | Limited | ❌ | ✅ |

---

## MAINTENANCE

### Daily (Automated)

- 15:05: Frank calibration update
- 00:30, 15:30: Enever data collection

### Weekly (Manual Check)

```bash
# Verify collector running
ssh coefficient@91.99.150.36 'sudo journalctl -u enever-collector.service -n 20'

# Check record count
ssh coefficient@91.99.150.36 'psql -U coefficient -d coefficient -c "SELECT COUNT(*), DATE(collected_at) FROM enever_prices GROUP BY DATE(collected_at) ORDER BY 2 DESC LIMIT 7;"'
```

### Monthly

- Review Frank vs Enever delta trends
- Check VPN tunnel stability
- Verify API rate limit usage (< 10,000/month)

---

## TROUBLESHOOTING

### Enever Proxy Timeout

```bash
# Check VPN
ssh coefficient@91.99.150.36 'sudo wg show pia-split'

# Test Enever direct
ssh coefficient@91.99.150.36 'curl -s "https://enever.nl/api/stroomprijs_vandaag.php?token=$(cat /opt/coefficient/.enever_token)" | jq ".data | length"'

# Restart VPN if needed
ssh coefficient@91.99.150.36 'sudo systemctl restart wg-quick@pia-split'
```

### High Price Delta

```bash
# Compare manually
cd /opt/energy-insights-nl
venv/bin/python3 << 'EOF'
import asyncio
from services.consumer_price import get_validated_consumer_price

async def check():
    result = await get_validated_consumer_price()
    print(f"Frank:  €{result['frank']['price']:.4f}")
    print(f"Enever: €{result['enever']['price']:.4f}")
    delta = abs(result['frank']['price'] - result['enever']['price'])
    print(f"Delta:  €{delta:.4f}")

asyncio.run(check())
EOF
```

---

## RELATED SKILLS

- **SKILL 2**: Architecture (system overview)
- **SKILL 6**: Data Sources (ENTSO-E, TenneT, Energy-Charts)
- **SKILL 10**: Coefficient VPN (WireGuard setup)
- **SKILL 14**: Coefficient Business Model (B2B potential)

---

## CHANGELOG

- **2026-01-12 v2.1**: Bias correction implementation
  - Added BIAS_CORRECTION = 0.93 constant
  - Formula now: `consumer = (wholesale × slope + intercept) × 0.93`
  - Bias reduced from +7.58% to +0.05%
  - Average error reduced from 7.58% to **2.11%**
  - P95 error reduced to 4.08%
  - Model now more accurate than Enever (2.11% vs 2.99%)

- **2026-01-12 v2.0**: Linear regression model
  - Added slope+intercept calculation (replaces simple coefficient)
  - Database schema with 576 hourly coefficients
  - Synctacles fallback integration (Tier 3/4/5)
  - Average error reduced from 10-12% to 5.1%
  - Related handoffs in `handoffs/2026-01-12-coefficient-model/`

- **2026-01-11 v1.0**: Initial skill created
  - Documented dual-source architecture
  - Frank API + Enever integration
  - Coefficient server proxy setup
  - Accuracy metrics and fallback cascade
