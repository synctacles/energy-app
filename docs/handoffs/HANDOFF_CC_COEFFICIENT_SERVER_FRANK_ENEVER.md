# HANDOFF: Frank Energie API + Enever Data Processing

**Server:** Coefficient Engine  
**IP:** 91.99.150.36  
**SSH:** `ssh coefficient@91.99.150.36`  
**Prioriteit:** MEDIUM  
**Geschatte tijd:** 1.5-2 uur

---

## CONTEXT

Dit is de **coefficient server** (apart van main API). Hier draait:
- VPN split tunnel naar Enever (NL IP via PIA)
- Coefficient engine (private repo)
- PostgreSQL voor coefficient data

---

## TAAK 1: FRANK ENERGIE API TEST (45 min)

### Doel
Bepaal of Frank Energie GraphQL bruikbaar is als primaire gratis prijzen-API.

### Endpoints

```bash
# Primary
FRANK_PRIMARY="https://graphql.frankenergie.nl"

# CDN fallback
FRANK_CDN="https://graphcdn.frankenergie.nl"
```

### Test 1: Connectivity

```bash
# Basic connectivity
curl -s -o /dev/null -w "%{http_code}" $FRANK_PRIMARY

# GraphQL introspection (geen auth)
curl -s -X POST $FRANK_PRIMARY \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __schema { types { name } } }"}' | head -100
```

### Test 2: Prijzen Query (GEEN AUTH NODIG)

```bash
# Vandaag + morgen prijzen
curl -s -X POST $FRANK_PRIMARY \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { marketPricesElectricity(startDate: \"2026-01-10\", endDate: \"2026-01-11\") { from till marketPrice priceIncludingMarkup } }"
  }' | python3 -m json.tool
```

**Verwachte response:**
```json
{
  "data": {
    "marketPricesElectricity": [
      {
        "from": "2026-01-10T00:00:00+01:00",
        "till": "2026-01-10T01:00:00+01:00",
        "marketPrice": 0.08234,
        "priceIncludingMarkup": 0.2847
      }
    ]
  }
}
```

### Test 3: Rate Limit Discovery

```bash
# Burst test (10 requests snel)
for i in {1..10}; do
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $FRANK_PRIMARY \
    -H "Content-Type: application/json" \
    -d '{"query": "{ marketPricesElectricity(startDate: \"2026-01-10\", endDate: \"2026-01-10\") { from } }"}')
  echo "Request $i: $CODE"
  sleep 0.5
done

# Check response headers
curl -s -I -X POST $FRANK_PRIMARY \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __typename }"}' | grep -i "rate\|limit\|retry\|x-"
```

### Test 4: CDN Fallback

```bash
# Test CDN endpoint
curl -s -X POST $FRANK_CDN \
  -H "Content-Type: application/json" \
  -d '{"query": "{ marketPricesElectricity(startDate: \"2026-01-10\", endDate: \"2026-01-10\") { from marketPrice } }"}' | python3 -m json.tool
```

### Deliverable

Maak: `~/FRANK_ENERGIE_API_REPORT.md`

```markdown
# Frank Energie API Test Report
Date: 2026-01-10

## Connectivity
- Primary (graphql.frankenergie.nl): [OK/FAIL]
- CDN (graphcdn.frankenergie.nl): [OK/FAIL]

## Price Data
- Market prices available: [YES/NO]
- Consumer prices available: [YES/NO]
- Auth required: [YES/NO]

## Rate Limits
- Burst test (10 req): [results]
- Headers found: [list]
- Recommended interval: [X seconds]

## Recommendation
[GO/NO-GO voor gebruik als primaire API]

## Sample Response
[JSON snippet]
```

---

## TAAK 2: ENEVER HISTORISCHE DATA (45 min)

### Prerequisites

Leo heeft CSV's geüpload. Check locatie:

```bash
ls -la ~/data/enever/
# Of
ls -la /opt/coefficient/data/
```

**Verwachte bestanden:**
- `Beurprijs_zonder_toeslagen_belastingen.csv` (wholesale)
- `download.csv` (consumer)

### Stap 1: Validatie Script

```python
#!/usr/bin/env python3
# ~/validate_enever.py

import pandas as pd
from datetime import datetime

# Load wholesale
wholesale = pd.read_csv('Beurprijs_zonder_toeslagen_belastingen.csv', 
                        parse_dates=['timestamp'])
print(f"Wholesale: {len(wholesale)} records")
print(f"Period: {wholesale['timestamp'].min()} - {wholesale['timestamp'].max()}")
print(f"Nulls: {wholesale['price'].isna().sum()}")

# Load consumer  
consumer = pd.read_csv('download.csv', parse_dates=['timestamp'])
print(f"\nConsumer: {len(consumer)} records")
print(f"Period: {consumer['timestamp'].min()} - {consumer['timestamp'].max()}")
print(f"Nulls: {consumer['price'].isna().sum()}")

# Gap detection
def find_gaps(df):
    df = df.sort_values('timestamp')
    df['gap'] = df['timestamp'].diff()
    gaps = df[df['gap'] > pd.Timedelta(hours=1.1)]
    return gaps[['timestamp', 'gap']]

print("\nWholesale gaps:")
print(find_gaps(wholesale))

print("\nConsumer gaps:")
print(find_gaps(consumer))
```

### Stap 2: Cross-Check met ENTSO-E

Sample timestamps voor validatie:

| Timestamp | Enever €/kWh | ENTSO-E €/MWh (verwacht) |
|-----------|--------------|--------------------------|
| 2023-01-15 14:00 | 0.080150 | 80.15 |
| 2023-07-20 10:00 | 0.086300 | 86.30 |
| 2024-03-10 18:00 | 0.084310 | 84.31 |

```bash
# Query ENTSO-E data (als beschikbaar in coefficient DB)
psql -d coefficient -c "
SELECT timestamp, price_eur_mwh 
FROM entso_day_ahead
WHERE timestamp IN (
  '2023-01-15 14:00+01',
  '2023-07-20 10:00+02',
  '2024-03-10 18:00+01'
);
"
```

### Stap 3: Coefficient Berekening

```python
#!/usr/bin/env python3
# ~/calculate_coefficients.py

import pandas as pd
import json

# Load data
wholesale = pd.read_csv('wholesale.csv', parse_dates=['timestamp'])
consumer = pd.read_csv('consumer.csv', parse_dates=['timestamp'])

# Merge (inner join - alleen waar beide data hebben)
merged = pd.merge(wholesale, consumer, on='timestamp', 
                  suffixes=('_wholesale', '_consumer'))
merged = merged.dropna()

# Calculate markup
merged['markup'] = merged['price_consumer'] - merged['price_wholesale']
merged['hour'] = merged['timestamp'].dt.hour
merged['month'] = merged['timestamp'].dt.month
merged['is_weekend'] = merged['timestamp'].dt.dayofweek >= 5

# Per-hour coefficients
coefficients = {}
for hour in range(24):
    hour_data = merged[merged['hour'] == hour]['markup']
    if len(hour_data) >= 100:
        coefficients[hour] = {
            'mean': round(hour_data.mean(), 6),
            'std': round(hour_data.std(), 6),
            'min': round(hour_data.min(), 6),
            'max': round(hour_data.max(), 6),
            'count': len(hour_data)
        }

# Save
with open('coefficients_enever_historical.json', 'w') as f:
    json.dump(coefficients, f, indent=2)

print("Coefficients saved!")
print(json.dumps(coefficients, indent=2))
```

### Deliverable

1. **Validation report:** `~/ENEVER_VALIDATION_REPORT.md`
2. **Coefficients:** `~/coefficients_enever_historical.json`
3. **Scripts:** `~/validate_enever.py`, `~/calculate_coefficients.py`

---

## SUCCESS CRITERIA

### Frank API
- [ ] Connectivity getest (beide endpoints)
- [ ] Prijzen query werkt (geen auth)
- [ ] Rate limits gedocumenteerd
- [ ] GO/NO-GO beslissing

### Enever Data
- [ ] CSV's gevalideerd (record counts, gaps)
- [ ] Cross-check met ENTSO-E uitgevoerd
- [ ] Coefficients berekend per uur
- [ ] JSON output gegenereerd

---

## NOTES

- VPN zou al actief moeten zijn voor Enever access
- Check met `wg show pia-split` als Enever calls falen
- Frank API zou GEEN VPN nodig moeten hebben
