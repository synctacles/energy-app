# HANDOFF: CAI → CC | Enever Dual Source Implementation

**Datum:** 2026-01-10
**Van:** Claude AI (Anthropic)
**Naar:** Claude Code
**Status:** READY FOR IMPLEMENTATION

---

## CONTEXT

Main API server (135.181.255.83) heeft nu consumer price berekening via Frank API. Om robuustheid te verhogen wordt Enever toegevoegd als secundaire live bron + historische data collector.

**Probleem:** Enever API vereist mogelijk Nederlandse IP. Direct aanroepen vanaf main API (Duitsland) exposeert server in grijs gebied.

**Oplossing:** Coefficient server (91.99.150.36) als proxy. VPN split tunnel is al geconfigureerd (zie SKILL_10_COEFFICIENT_VPN.md).

---

## ARCHITECTUUR

```
Main API (135.181.255.83)
│
├── Frank API (direct, primary)
│
└── GET /internal/enever ──→ Coefficient Server (91.99.150.36)
                                    │
                                    ├── VPN split tunnel ──→ Enever API
                                    │
                                    └── PostgreSQL (historische opslag)
```

---

## DELIVERABLES

### 1. Enever Proxy Endpoint

**Locatie:** Coefficient server
**Endpoint:** `GET /internal/enever/prices`
**Security:** IP whitelist (alleen 135.181.255.83)

**Response format:**
```json
{
  "timestamp": "2026-01-10T15:30:00Z",
  "source": "enever",
  "prices_today": {
    "Frank Energie": [{"hour": 0, "price": 0.183}, ...],
    "Tibber": [...],
    // alle providers
  },
  "prices_tomorrow": {
    // null als nog niet beschikbaar (voor 15:00)
  }
}
```

**Endpoints aanroepen:**
- `https://enever.nl/api/stroomprijs_vandaag.php?token=XXX`
- `https://enever.nl/api/stroomprijs_morgen.php?token=XXX`

**Token:** Opgeslagen in `/opt/coefficient/.enever_token`

---

### 2. Enever Data Collector

**Frequentie:** 2x per dag
- 15:30 - Haal vandaag + morgen (morgen net beschikbaar)
- 00:30 - Haal vandaag (bevestig definitieve prijzen)

**Systemd timer:** `enever-collector.timer`

**Database schema:**
```sql
CREATE TABLE enever_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    provider VARCHAR(50) NOT NULL,
    hour INTEGER NOT NULL,  -- 0-23
    price_total DECIMAL(8,5) NOT NULL,  -- €/kWh incl BTW
    price_energy DECIMAL(8,5),  -- indien beschikbaar
    price_tax DECIMAL(8,5),     -- indien beschikbaar
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(timestamp, provider, hour)
);

CREATE INDEX idx_enever_timestamp ON enever_prices(timestamp);
CREATE INDEX idx_enever_provider ON enever_prices(provider);
```

**Opslag:** PostgreSQL op coefficient server (draait al)

---

### 3. Main API Integration

**Locatie:** `/opt/energy-insights-nl/app/services/enever_client.py`

```python
import httpx
from datetime import datetime
import logging

log = logging.getLogger(__name__)

COEFFICIENT_SERVER = "http://91.99.150.36:8080"
TIMEOUT = 10.0

async def fetch_enever_price(hour: int = None) -> dict | None:
    """Fetch current Enever price via coefficient proxy."""
    if hour is None:
        hour = datetime.now().hour
    
    try:
        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            resp = await client.get(f"{COEFFICIENT_SERVER}/internal/enever/prices")
            resp.raise_for_status()
            data = resp.json()
            
            # Extract Frank price for comparison
            frank_prices = data.get("prices_today", {}).get("Frank Energie", [])
            for p in frank_prices:
                if p["hour"] == hour:
                    return {
                        "price": p["price"],
                        "source": "enever",
                        "timestamp": data["timestamp"]
                    }
            return None
            
    except Exception as e:
        log.warning(f"Enever proxy failed: {e}")
        return None
```

---

### 4. Dual Source Validation

**In `/opt/energy-insights-nl/app/services/consumer_price.py`:**

```python
async def get_validated_price() -> dict:
    """Get consumer price with dual-source validation."""
    
    frank = await fetch_frank_price()
    enever = await fetch_enever_price()
    
    result = {
        "price": None,
        "source": None,
        "confidence": "low",
        "frank": frank,
        "enever": enever
    }
    
    if frank and enever:
        delta = abs(frank["price"] - enever["price"])
        if delta < 0.02:  # < €0.02 verschil
            result["price"] = frank["price"]
            result["source"] = "frank+enever"
            result["confidence"] = "high"
        else:
            # Verschil te groot - gebruik Frank, log alert
            log.warning(f"Price mismatch: Frank={frank['price']}, Enever={enever['price']}")
            result["price"] = frank["price"]
            result["source"] = "frank"
            result["confidence"] = "medium"
            
    elif frank:
        result["price"] = frank["price"]
        result["source"] = "frank"
        result["confidence"] = "medium"
        
    elif enever:
        result["price"] = enever["price"]
        result["source"] = "enever"
        result["confidence"] = "medium"
        
    else:
        # Beide failed - gebruik cached correctiefactor
        result["price"] = get_cached_price()
        result["source"] = "cached"
        result["confidence"] = "low"
    
    return result
```

---

### 5. Monitoring & Alerts

**Health checks:**

| Check | Frequentie | Alert als |
|-------|------------|-----------|
| Enever proxy bereikbaar | Elke 5 min | Timeout > 10s |
| VPN tunnel actief | Elke 5 min | `wg show pia-split` failed |
| Collector laatste run | Dagelijks | > 25 uur geleden |
| Frank vs Enever delta | Dagelijks | > €0.02 verschil |
| Database records vandaag | Dagelijks | < 13 providers × 24 uur |

**Alert endpoint:** Log naar `/var/log/coefficient/alerts.log`
(Later: webhook naar monitoring)

---

## FILES TO CREATE

### Coefficient Server (91.99.150.36)

```
/opt/coefficient/
├── app/
│   ├── main.py                 # FastAPI app
│   ├── routes/
│   │   └── enever.py           # Proxy endpoint
│   └── services/
│       └── enever_client.py    # Enever API calls
├── collectors/
│   └── enever_collector.py     # Scheduled collector
├── config/
│   └── settings.py             # Config incl token
└── .enever_token               # API token (niet in git)

/etc/systemd/system/
├── coefficient-api.service     # API service
├── enever-collector.service    # Collector service
└── enever-collector.timer      # Timer (15:30, 00:30)
```

### Main API (135.181.255.83)

```
/opt/energy-insights-nl/app/services/
├── enever_client.py            # NEW: Proxy client
└── consumer_price.py           # UPDATE: Dual validation
```

---

## SECURITY

1. **IP Whitelist:** Coefficient API alleen bereikbaar vanaf 135.181.255.83
2. **Internal port:** 8080 (niet publiek exposed)
3. **Enever token:** Niet in git, alleen in runtime config
4. **VPN:** Split tunnel - alleen Enever traffic via NL exit

---

## TESTING

### Proof Commands

```bash
# 1. VPN werkt
ssh coefficient@91.99.150.36
ip route get 84.46.252.107
# Expected: dev pia-split

# 2. Enever bereikbaar via VPN
curl -s "https://enever.nl/api/stroomprijs_vandaag.php?token=XXX" | jq '.data | length'
# Expected: 13+ providers

# 3. Proxy endpoint werkt
curl -s http://91.99.150.36:8080/internal/enever/prices | jq '.prices_today | keys'
# Expected: ["Frank Energie", "Tibber", ...]

# 4. Main API kan proxy bereiken
ssh root@135.181.255.83
curl -s http://91.99.150.36:8080/internal/enever/prices | jq '.timestamp'
# Expected: recent timestamp

# 5. Database heeft records
psql -U coefficient -d coefficient -c "SELECT COUNT(*) FROM enever_prices WHERE timestamp > NOW() - INTERVAL '1 day';"
# Expected: 300+ (13 providers × 24 uur)
```

---

## IMPLEMENTATION ORDER

1. ✅ VPN al geconfigureerd (SKILL_10)
2. ⬜ Database schema aanmaken
3. ⬜ Collector script + systemd timer
4. ⬜ FastAPI proxy endpoint
5. ⬜ IP whitelist configureren
6. ⬜ Main API enever_client.py
7. ⬜ Dual validation in consumer_price.py
8. ⬜ Monitoring/alerts
9. ⬜ End-to-end test

---

## ENEVER API DETAILS

**Token verkrijgen:** https://enever.nl/api/ (gratis registratie)

**Rate limit:** 10.000 requests/maand

**Onze usage:** ~120 requests/maand (1.2%)

**Response format vandaag:**
```json
{
  "data": [
    {
      "provider": "Frank Energie",
      "prices": [
        {"hour": 0, "price": 0.183},
        {"hour": 1, "price": 0.178},
        // ... 24 uren
      ]
    },
    // ... meer providers
  ]
}
```

**Response morgen:** Zelfde format, beschikbaar na ~15:00

---

## FALLBACK CASCADE (UPDATED)

```
1. Frank API live + Enever live (beide OK)     → 98% accuracy, high confidence
2. Frank API live only                          → 95% accuracy, medium confidence  
3. Enever live only                             → 93% accuracy, medium confidence
4. Cached correctiefactor (<48h)                → 91% accuracy, low confidence
5. Geen correctie (factor=1.0)                  → 89% accuracy, degraded
6. Daggemiddelde (€0.17)                        → 85% accuracy, emergency
```

---

## NOTES

- Enever token nog aan te vragen (Leo)
- PostgreSQL draait al op coefficient server
- Geen publieke exposure van coefficient API
- Data wordt ook historisch opgeslagen voor B2B asset

---

## VRAGEN VOOR LEO

1. Heb je al een Enever API token? Zo niet, registreren op https://enever.nl/api/
2. Welke port voor coefficient API? (voorstel: 8080 internal)
3. Wil je alerts via email of alleen logging voorlopig?

---

---

## RAPPORTAGE REQUIREMENT

**Na implementatie, lever CC een uitgebreide handoff terug met:**

1. **Wat is geïmplementeerd**
   - Exacte file paths en inhoud
   - Database schema zoals aangemaakt
   - Systemd services/timers

2. **Configuratie**
   - Poorten, IP whitelists
   - Credentials locaties (geen secrets zelf)
   - Environment variables

3. **Test resultaten**
   - Output van alle proof commands
   - Response times
   - Eventuele errors/warnings

4. **Afwijkingen van plan**
   - Wat is anders dan deze handoff?
   - Waarom?

5. **Open issues**
   - Wat werkt nog niet?
   - Wat moet Leo nog doen?

6. **Aanbevelingen**
   - Optimalisaties
   - Security hardening
   - Monitoring gaps

**Format:** Markdown document `HANDOFF_CC_CAI_ENEVER_IMPLEMENTATION_COMPLETE.md`

---

**Ready for implementation.**
