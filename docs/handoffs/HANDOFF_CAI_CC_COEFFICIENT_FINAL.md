# HANDOFF: Claude AI → Claude Code
## Coefficient Herberekening + Main API Integratie

**Datum:** 2026-01-10
**Van:** Claude AI
**Naar:** Claude Code
**Servers:** 91.99.150.36 (coefficient) → 135.181.255.83 (main API)

---

## CONTEXT

De huidige HOURLY_LOOKUP is berekend over oktober 2022 - januari 2026.
**Probleem:** 67% van die data komt uit crisisjaren met abnormaal lage energiebelasting.

Dit kan de uurpatronen vertroebelen, niet alleen het absolute niveau.

---

## OPDRACHT

### Deel 1: Herberekening (op coefficient server)

**Stap 1: Filter data op 2025+**

```bash
ssh coefficient@91.99.150.36
cd ~/data/enever_csv
```

Filter alle CSVs op datum >= 2025-01-01

**Stap 2: Bereken nieuwe HOURLY_LOOKUP**

Voor FrankEnergie (of gemiddelde over providers):

```python
# Pseudocode
for hour in range(24):
    data_2025_plus = filter(records, date >= "2025-01-01")
    markups = [record.consumer - record.wholesale for record in data_2025_plus if record.hour == hour]
    HOURLY_LOOKUP_NEW[hour] = mean(markups)
```

**Stap 3: Vergelijk oud vs nieuw**

```
Uur   Lookup OUD   Lookup NIEUW   Verschil   Significant?
         (2022-2026)  (2025-2026)
------------------------------------------------------------
00    €0.172        €0.???        €0.???     ?
01    €0.171        €0.???        €0.???     ?
...
19    €0.181        €0.???        €0.???     ?  ← Avondpiek
...
```

**Significantie:** Verschil > €0.01 (>5%) = significant

**Stap 4: Rapporteer bevindingen**

- Zijn de patronen (relatieve verhoudingen) stabiel?
- Is de avondpiek (17-20) nog steeds de duurste?
- Is de solar dip (12-15) nog steeds zichtbaar?
- Zijn nachturen (01-05) nog steeds goedkoopst?

---

### Deel 2: Bepaal finale HOURLY_LOOKUP

**Scenario A:** Patronen zijn stabiel (verschil <5%)
→ Gebruik oude lookup, correctiefactor volstaat

**Scenario B:** Patronen verschillen significant (verschil >5%)
→ Gebruik nieuwe lookup (2025+ data)

---

### Deel 3: Integratie Main API

**Locatie:** 135.181.255.83

**Bestand:** `/opt/synctacles/app/config/coefficients.py` (nieuw)

```python
"""
Consumer price coefficient lookup table.
Bron: Enever historische data (2025-2026)
Laatste update: 2026-01-10
"""

# Markup = consumer_price - wholesale_price
# Gebaseerd op FrankEnergie als referentie provider
HOURLY_MARKUP_LOOKUP = {
    0:  0.xxx,  # Vul in na herberekening
    1:  0.xxx,
    2:  0.xxx,
    3:  0.xxx,
    4:  0.xxx,
    5:  0.xxx,
    6:  0.xxx,
    7:  0.xxx,
    8:  0.xxx,
    9:  0.xxx,
    10: 0.xxx,
    11: 0.xxx,
    12: 0.xxx,
    13: 0.xxx,
    14: 0.xxx,
    15: 0.xxx,
    16: 0.xxx,
    17: 0.xxx,
    18: 0.xxx,
    19: 0.xxx,
    20: 0.xxx,
    21: 0.xxx,
    22: 0.xxx,
    23: 0.xxx
}

# Fallback als lookup niet beschikbaar
DEFAULT_MARKUP = 0.17
```

**Bestand:** `/opt/synctacles/app/services/frank_calibration.py` (nieuw)

```python
"""
Frank API calibratie service.
Haalt dagelijks correctiefactor op.
"""

import httpx
from datetime import datetime, timedelta
from typing import Optional
import logging

logger = logging.getLogger(__name__)

FRANK_API_URL = "https://graphql.frankenergie.nl"
FRANK_QUERY = """
{ marketPricesElectricity(startDate:"%s", endDate:"%s") {
    from
    marketPrice
    marketPriceTax
    sourcingMarkupPrice
    energyTaxPrice
}}
"""

# Cache
_correction_factor: float = 1.0
_correction_updated: Optional[datetime] = None


async def fetch_frank_prices(date: str) -> list[dict]:
    """Haal Frank prijzen op voor datum."""
    query = FRANK_QUERY % (date, date)
    
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.post(
            FRANK_API_URL,
            json={"query": query}
        )
        resp.raise_for_status()
        return resp.json()["data"]["marketPricesElectricity"]


def calculate_frank_markup(hour_data: dict) -> float:
    """Bereken markup uit Frank API response."""
    return (
        hour_data["marketPriceTax"] +
        hour_data["sourcingMarkupPrice"] +
        hour_data["energyTaxPrice"]
    )


async def update_correction_factor() -> float:
    """
    Update correctiefactor op basis van Frank live vs lookup.
    Aanroepen: 1x per dag om 15:05
    """
    global _correction_factor, _correction_updated
    
    from config.coefficients import HOURLY_MARKUP_LOOKUP
    
    try:
        today = datetime.now().strftime("%Y-%m-%d")
        frank_data = await fetch_frank_prices(today)
        
        # Bereken gemiddelde correctie over alle uren
        corrections = []
        for hour_data in frank_data:
            hour = int(hour_data["from"][11:13])
            frank_markup = calculate_frank_markup(hour_data)
            lookup_markup = HOURLY_MARKUP_LOOKUP.get(hour, 0.17)
            
            if lookup_markup > 0:
                corrections.append(frank_markup / lookup_markup)
        
        if corrections:
            _correction_factor = sum(corrections) / len(corrections)
            _correction_updated = datetime.now()
            logger.info(f"Correctiefactor updated: {_correction_factor:.4f}")
        
    except Exception as e:
        logger.error(f"Frank API error: {e}")
        # Behoud vorige waarde
    
    return _correction_factor


def get_correction_factor() -> tuple[float, str]:
    """
    Haal huidige correctiefactor op met fallback niveau.
    
    Returns:
        (factor, level) waar level = "live" | "cached" | "none"
    """
    global _correction_factor, _correction_updated
    
    if _correction_updated:
        age = datetime.now() - _correction_updated
        
        if age < timedelta(hours=24):
            return _correction_factor, "live"
        elif age < timedelta(hours=48):
            return _correction_factor, "cached"
    
    return 1.0, "none"
```

**Bestand:** `/opt/synctacles/app/services/consumer_price.py` (nieuw)

```python
"""
Consumer price berekening.
"""

from config.coefficients import HOURLY_MARKUP_LOOKUP, DEFAULT_MARKUP
from services.frank_calibration import get_correction_factor


def calculate_consumer_price(wholesale: float, hour: int) -> dict:
    """
    Bereken geschatte consumer price.
    
    Args:
        wholesale: ENTSO-E wholesale prijs (€/kWh)
        hour: Uur van de dag (0-23)
    
    Returns:
        {
            "consumer_price": float,
            "wholesale": float,
            "markup": float,
            "correction_factor": float,
            "fallback_level": str
        }
    """
    # Haal lookup markup
    base_markup = HOURLY_MARKUP_LOOKUP.get(hour, DEFAULT_MARKUP)
    
    # Haal correctiefactor
    correction, level = get_correction_factor()
    
    # Bereken
    adjusted_markup = base_markup * correction
    consumer_price = wholesale + adjusted_markup
    
    return {
        "consumer_price": round(consumer_price, 4),
        "wholesale": round(wholesale, 4),
        "markup": round(adjusted_markup, 4),
        "correction_factor": round(correction, 4),
        "fallback_level": level
    }
```

**Bestand:** Cron job voor dagelijkse update

```bash
# /etc/cron.d/frank-calibration

# Update correctiefactor na EPEX publicatie
5 15 * * * synctacles /opt/synctacles/venv/bin/python -c "
import asyncio
from services.frank_calibration import update_correction_factor
asyncio.run(update_correction_factor())
" >> /var/log/synctacles/calibration.log 2>&1
```

---

## DELIVERABLES

### Van coefficient server (91.99.150.36):

1. [ ] Herberekende HOURLY_LOOKUP (alleen 2025+ data)
2. [ ] Vergelijkingsrapport oud vs nieuw
3. [ ] Aanbeveling welke lookup te gebruiken

### Op main API (135.181.255.83):

4. [ ] `config/coefficients.py` met finale lookup
5. [ ] `services/frank_calibration.py` 
6. [ ] `services/consumer_price.py`
7. [ ] Cron job voor dagelijkse update
8. [ ] Test: bereken consumer price, vergelijk met Frank API

---

## VERIFICATIE

Na implementatie, test:

```bash
# Op main API server
curl -s "http://localhost:8000/debug/consumer-price?wholesale=0.045&hour=14"

# Verwacht:
{
    "consumer_price": 0.183,
    "wholesale": 0.045,
    "markup": 0.138,
    "correction_factor": 0.831,
    "fallback_level": "live"
}
```

Vergelijk met Frank API voor dezelfde dag/uur → verschil < €0.005 = OK

---

## TIJDLIJN

| Stap | Server | Geschatte tijd |
|------|--------|----------------|
| Herberekening lookup | coefficient | 30 min |
| Vergelijkingsrapport | coefficient | 15 min |
| Implementatie main API | main | 45 min |
| Testing | main | 15 min |
| **Totaal** | | **~2 uur** |

---

*Gegenereerd door Claude AI - 2026-01-10*
