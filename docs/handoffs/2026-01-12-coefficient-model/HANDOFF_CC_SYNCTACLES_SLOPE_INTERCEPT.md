# HANDOFF: Synctacles API - Slope+Intercept Model Integratie

**Van:** Claude Code
**Naar:** Volgende ontwikkelaar / Claude Code
**Datum:** 2026-01-12
**Status:** VOLTOOID
**Classificatie:** VERTROUWELIJK

---

## SAMENVATTING

De Synctacles API fallback chain is geüpgraded van een simpele coefficient multiplier naar een **lineair regressie model met slope en intercept**. Dit verbetert de prijsvoorspelling significant.

| Aspect | Oud | Nieuw |
|--------|-----|-------|
| Formule | `consumer = wholesale × coefficient` | `consumer = wholesale × slope + intercept` |
| Nauwkeurigheid | ~10% afwijking | **~5% afwijking** |
| Avondpiek fout | 24-39% | **5-11%** |

---

## GEWIJZIGDE BESTANDEN

### 1. consumer_price_client.py

**Locatie:** `synctacles_db/clients/consumer_price_client.py`

**Nieuwe methodes:**

```python
@staticmethod
async def get_price_model(
    hour: Optional[int] = None,
    day_type: Optional[str] = None,
    month: Optional[int] = None,
    country: str = "NL"
) -> Optional[Dict]:
    """
    Get slope + intercept model from Coefficient Engine.

    Returns:
        {
            "slope": 1.27,
            "intercept": 0.147,
            "confidence": 94,
            "sample_size": 75,
            "last_calibrated": "2026-01-12T15:09:42Z",
            "source": "lookup"
        }
    """

@staticmethod
def calculate_consumer_price(
    wholesale_eur_kwh: float,
    slope: float,
    intercept: float
) -> float:
    """
    Calculate consumer price from wholesale using linear model.

    Formula: consumer = wholesale × slope + intercept
    """
    return wholesale_eur_kwh * slope + intercept
```

**Legacy wrapper voor backward compatibility:**

```python
@staticmethod
async def get_coefficient(...) -> Optional[Dict]:
    """Legacy method - wraps get_price_model."""
    result = await ConsumerPriceClient.get_price_model(...)
    if result:
        result["coefficient"] = result.get("slope", 1.0)  # Legacy field
    return result
```

### 2. fallback_manager.py

**Locatie:** `synctacles_db/fallback/fallback_manager.py`

**Nieuwe methode (lijn 722-749):**

```python
@staticmethod
def _apply_price_model(
    prices: List[Dict],
    slope: float,
    intercept: float
) -> List[Dict]:
    """
    Apply linear regression model to wholesale prices.

    Formula: consumer = wholesale × slope + intercept
    """
    adjusted = []
    for p in prices:
        new_p = p.copy()
        if "price_eur_mwh" in new_p and new_p["price_eur_mwh"] is not None:
            # Convert MWh to kWh, apply model, convert back to MWh
            wholesale_kwh = float(new_p["price_eur_mwh"]) / 1000.0
            consumer_kwh = wholesale_kwh * slope + intercept
            new_p["price_eur_mwh"] = consumer_kwh * 1000.0
        adjusted.append(new_p)
    return adjusted
```

**Default fallback waarden (lijn 622-624):**

```python
DEFAULT_SLOPE = 1.27
DEFAULT_INTERCEPT = 0.147  # EUR/kWh fixed costs
```

**Geüpdatete Tiers:**

| Tier | Oude Source | Nieuwe Source |
|------|-------------|---------------|
| 3 | `ENTSO-E+Coef` | `ENTSO-E+Model` |
| 4 | `ENTSO-E+Coef (stale)` | `ENTSO-E+Model (stale)` |
| 5 | `Energy-Charts+Coef` | `Energy-Charts+Model` |

---

## FALLBACK CHAIN OVERZICHT

```
┌─────────────────────────────────────────────────────────────┐
│  TIER 1: Database Cache (Consumer prices)                   │
│  └── Directe cached consumentenprijzen                      │
├─────────────────────────────────────────────────────────────┤
│  TIER 2: Coefficient Engine Consumer Proxy                  │
│  └── Frank Energie live prijzen via proxy                   │
│      Source: "Frank Energie"                                │
├─────────────────────────────────────────────────────────────┤
│  TIER 3: ENTSO-E Fresh + Price Model                        │
│  └── ENTSO-E wholesale × slope + intercept                  │
│      Source: "ENTSO-E+Model (95%)"                          │
├─────────────────────────────────────────────────────────────┤
│  TIER 4: ENTSO-E Stale + Price Model                        │
│  └── Oudere ENTSO-E data met price model                    │
│      Source: "ENTSO-E+Model (stale)"                        │
├─────────────────────────────────────────────────────────────┤
│  TIER 5: Energy-Charts + Price Model                        │
│  └── Energy-Charts als fallback met price model             │
│      Source: "Energy-Charts+Model"                          │
├─────────────────────────────────────────────────────────────┤
│  TIER 6: Cache Fallback                                     │
│  └── In-memory of PostgreSQL cache                          │
└─────────────────────────────────────────────────────────────┘
```

---

## COEFFICIENT ENGINE API

**Endpoint:** `GET http://91.99.150.36:8080/coefficient`

**Parameters:**
- `country`: Land code (default: NL)
- `month`: Maand 1-12 (default: huidige)
- `day_type`: 'weekday' of 'weekend' (default: huidige)
- `hour`: Uur 0-23 (default: huidige)

**Response:**
```json
{
    "country": "NL",
    "slope": 1.1193,
    "intercept": 0.1464,
    "month": 1,
    "day_type": "weekday",
    "hour": 12,
    "confidence": 95,
    "sample_size": 64,
    "source": "lookup",
    "timestamp": "2026-01-12T16:39:00Z"
}
```

---

## BEREKENING VOORBEELD

**Input:**
- ENTSO-E prijs: €93.80/MWh = €0.0938/kWh
- Slope: 1.1193
- Intercept: €0.1464/kWh

**Berekening:**
```
consumer = wholesale × slope + intercept
         = €0.0938 × 1.1193 + €0.1464
         = €0.1050 + €0.1464
         = €0.2514/kWh
         = €251.4/MWh
```

**Vergelijking:**
- Berekend: €0.2514/kWh
- Frank Energie werkelijk: €0.2447/kWh
- Afwijking: +2.7%

---

## VALIDATIE RESULTATEN

Gebaseerd op validatie van 9 januari 2026:

| Metric | Waarde |
|--------|--------|
| Gemiddelde afwijking | 5.1% |
| Beste uur | 15:00 (0.3%) |
| Slechtste uur | 09:00 (15.8%) |
| Middag uren (12-17) | 3.6% |
| Avond uren (18-23) | 7.4% |

---

## SERVICE RESTART

Na wijzigingen:

```bash
sudo systemctl restart energy-insights-nl-api.service
```

Verificatie:

```bash
# Check status
sudo systemctl status energy-insights-nl-api.service

# Check logs
sudo journalctl -u energy-insights-nl-api.service --since "5 minutes ago" | grep -E "(Tier|Model)"

# Test API
curl -s "http://localhost:8000/api/v1/prices?country=NL" | python3 -m json.tool | head -20
```

---

## GERELATEERDE DOCUMENTATIE

- [HANDOFF_CC_COEFFICIENT_LINEAR_REGRESSION.md](HANDOFF_CC_COEFFICIENT_LINEAR_REGRESSION.md) - Technische details lineair regressie model
- [HANDOFF_CC_COEFFICIENT_ENGINE.md](../HANDOFF_CC_COEFFICIENT_ENGINE.md) - Coefficient Engine setup

---

## CHANGELOG

| Datum | Wijziging |
|-------|-----------|
| 2026-01-12 | consumer_price_client.py: get_price_model() toegevoegd |
| 2026-01-12 | fallback_manager.py: _apply_price_model() methode |
| 2026-01-12 | Tier 3/4/5 geüpgraded naar slope+intercept model |
| 2026-01-12 | Default fallback: slope=1.27, intercept=0.147 |
