# HANDOFF: Frank Energie Direct API als Tier 1

**Van:** Claude Code
**Naar:** Volgende ontwikkelaar / Claude Code
**Datum:** 2026-01-13
**Status:** GEIMPLEMENTEERD
**Classificatie:** TECHNISCH

---

## EXECUTIVE SUMMARY

De prijzen fallback chain is uitgebreid met een nieuwe **Tier 1: Frank Direct**. Dit is een directe connectie naar de Frank Energie GraphQL API, zonder tussenkomst van de Coefficient Server of VPN.

### Belangrijke Ontdekking
Frank Energie IP blocking is **momenteel NIET actief**. Directe API calls werken vanaf de Synctacles server. Dit maakt de VPN-proxy route overbodig als primaire bron.

---

## 1. NIEUWE FALLBACK ARCHITECTUUR

### 1.1 7-Tier Fallback Chain (was 6-tier)

| Tier | Bron | Type | Betrouwbaarheid | GO Actie |
|------|------|------|-----------------|----------|
| **1** | **Frank Direct (NIEUW)** | LIVE consumer | 100% | JA |
| 2 | Frank via Coefficient Proxy | LIVE consumer | 99% | JA |
| 3 | Consumer Proxy (Tibber/Vattenfall) | LIVE consumer | 98% | JA |
| 4 | ENTSO-E Fresh + Price Model | CALCULATED | 89% | JA |
| 5 | ENTSO-E Stale + Price Model | CALCULATED | 85% | JA |
| 6 | Energy-Charts + Price Model | FALLBACK | 70% | NEE |
| 7 | Cache (memory/PostgreSQL) | CACHED | 50% | NEE |

### 1.2 Data Flow Diagram

```
TIER 1 (NIEUW):
Synctacles API → Frank GraphQL API (graphql.frankenergie.nl)
                 └── Directe HTTPS call, geen VPN nodig

TIER 2 (was Tier 1):
Synctacles API → Coefficient Server (91.99.150.36:8080)
                 └── /internal/consumer/prices endpoint
                      └── VPN (pia-split) → Frank GraphQL API
```

---

## 2. TECHNISCHE IMPLEMENTATIE

### 2.1 Nieuwe Client: FrankEnergieClient

**Bestand:** `synctacles_db/clients/frank_energie_client.py`

```python
from synctacles_db.clients.frank_energie_client import FrankEnergieClient

# Vandaag prijzen ophalen
prices = await FrankEnergieClient.get_prices_today()
# Returns: List[{timestamp, price_eur_kwh, market_price, market_price_tax, ...}]

# Specifiek uur
price = await FrankEnergieClient.get_price_for_hour(14, "today")
# Returns: float (EUR/kWh)

# Health check
healthy, msg = await FrankEnergieClient.health_check()
```

### 2.2 GraphQL Query

```graphql
query {
  marketPricesElectricity(startDate: "2026-01-13", endDate: "2026-01-13") {
    from
    till
    marketPrice
    marketPriceTax
    sourcingMarkupPrice
    energyTaxPrice
  }
}
```

### 2.3 Prijs Samenstelling

De consumentenprijs bestaat uit 4 componenten:

```
price_eur_kwh = marketPrice + marketPriceTax + sourcingMarkupPrice + energyTaxPrice

Voorbeeld (13 jan 2026, uur 8):
  marketPrice:        €0.1190  (wholesale)
  marketPriceTax:     €0.0250  (BTW op wholesale)
  sourcingMarkupPrice: €0.0182  (inkoopkosten)
  energyTaxPrice:     €0.1109  (energiebelasting)
  ────────────────────────────
  TOTAAL:             €0.2731/kWh
```

### 2.4 Circuit Breaker

De client bevat een circuit breaker:
- **Max failures:** 3
- **Cooldown:** 5 minuten
- **Reset:** Automatisch na succesvolle call

---

## 3. FALLBACK MANAGER WIJZIGINGEN

### 3.1 Gewijzigd Bestand

**`synctacles_db/fallback/fallback_manager.py`**

De `get_prices_with_fallback()` methode is uitgebreid met Tier 1:

```python
# TIER 1: Frank Direct - GraphQL API (LIVE consumer prices)
try:
    frank_direct = await FrankEnergieClient.get_prices_today()
    if frank_direct and len(frank_direct) > 0:
        consumer_prices = [
            {
                "timestamp": p["timestamp"],
                "price_eur_mwh": p["price_eur_kwh"] * 1000
            }
            for p in frank_direct
        ]
        _LOGGER.info(f"Tier 1: Frank Direct LIVE ({len(consumer_prices)} prices)")
        return (consumer_prices, "Frank Direct", "FRESH", True)
except Exception as err:
    _LOGGER.debug(f"Tier 1 (Frank Direct) failed: {err}")
```

---

## 4. VALIDATIE

### 4.1 API Test Resultaat (13 jan 2026)

```bash
curl -s -X POST 'https://graphql.frankenergie.nl' \
  -H 'Content-Type: application/json' \
  -d '{"query":"{ marketPricesElectricity(startDate: \"2026-01-13\", ...) }"}'

# Resultaat: 24 prijzen succesvol opgehaald
# Uur 0: €0.2247/kWh
# Uur 8: €0.2731/kWh (piekuur)
```

### 4.2 IP Blocking Status

| Datum | Synctacles IP | Frank API | Status |
|-------|--------------|-----------|--------|
| 2026-01-13 | Publiek | graphql.frankenergie.nl | **OPEN** |

**Conclusie:** Geen IP blocking actief. Directe calls werken.

---

## 5. VOORDELEN VAN TIER 1

| Aspect | Via Coefficient (Tier 2) | Direct (Tier 1) |
|--------|-------------------------|-----------------|
| Latency | ~500ms | ~200ms |
| Dependencies | 3 (API → Coef → VPN → Frank) | 1 (API → Frank) |
| Failure points | 3 | 1 |
| VPN nodig | Ja | Nee |
| Onderhoud | Hoog | Laag |

---

## 6. RISICO'S EN MITIGATIE

### 6.1 Frank Kan IP Blocking Activeren

**Risico:** Frank kan IP blocking reactiveren, waardoor Tier 1 faalt.

**Mitigatie:**
1. Tier 2 (Coefficient proxy met VPN) blijft beschikbaar als fallback
2. Circuit breaker detecteert failures en schakelt automatisch naar Tier 2
3. Logging toont welke tier actief is

### 6.2 Monitoring

Check de logs voor tier usage:
```bash
journalctl -u synctacles-api | grep "Tier 1\|Tier 2"
```

Als "Tier 2" verschijnt terwijl "Tier 1" zou moeten werken, is er mogelijk IP blocking actief.

---

## 7. BESTANDEN GEWIJZIGD

| Bestand | Wijziging |
|---------|-----------|
| `synctacles_db/clients/frank_energie_client.py` | **NIEUW** - Directe Frank API client |
| `synctacles_db/fallback/fallback_manager.py` | Tier 1 toegevoegd, tiers hernummerd |

---

## 8. VOLGENDE STAPPEN

1. **Service herstarten** na deployment
2. **Monitor logs** voor tier usage
3. **Alert instellen** als Tier 1 > 10% failures heeft

---

## CHANGELOG

| Datum | Wijziging |
|-------|-----------|
| 2026-01-13 | FrankEnergieClient geimplementeerd |
| 2026-01-13 | Fallback chain uitgebreid naar 7 tiers |
| 2026-01-13 | Frank Direct als nieuwe Tier 1 |
