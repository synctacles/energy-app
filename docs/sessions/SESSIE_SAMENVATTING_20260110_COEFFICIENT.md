# SESSIE SAMENVATTING - 10 januari 2026
## Coefficient Analyse & Business Decisions

---

## 1. COEFFICIENT ANALYSE RESULTATEN

### Data gebruikt
- **ANWB Energie**: 27.895 uren (nov 2022 - jan 2026)
- **ENTSO-E NL**: Volledige overlap periode
- **Analyse periode**: Focus op 2024-2026 (16.075 uren)

### Model vergelijking

| Model | Formule | Nauwkeurigheid |
|-------|---------|----------------|
| Multiplicatief | `consumer = wholesale × 3.4` | ❌ CV 60% - onbruikbaar |
| **Additief (uur)** | `consumer = wholesale + offset[uur]` | ✅ CV 20% - bruikbaar |

### Finale lookup table (24 waarden)

```python
HOURLY_OFFSET = {
    0: 0.1934, 1: 0.1903, 2: 0.1879, 3: 0.1819,
    4: 0.1705, 5: 0.1667, 6: 0.1789, 7: 0.1989,
    8: 0.2132, 9: 0.2099, 10: 0.2030, 11: 0.1968,
    12: 0.1899, 13: 0.1768, 14: 0.1669, 15: 0.1599,
    16: 0.1508, 17: 0.1571, 18: 0.1723, 19: 0.2009,
    20: 0.2085, 21: 0.2050, 22: 0.2006, 23: 0.1945
}

# Gebruik:
consumer_price = wholesale_price + HOURLY_OFFSET[hour]
```

### Prestaties simpel model

| Metric | Waarde |
|--------|--------|
| Prijs binnen 2 cent | 59% van de tijd |
| Prijs binnen 3 cent | 75% van de tijd |
| **Ranking 3+/4 goedkoopste uren** | **68% van de dagen** |
| EV nachtladen 3+/4 goed | 78% van de nachten |
| **Besparing behaald vs max** | **89%** |

### Extreme dagen model (NIET geïmplementeerd)
- Threshold: daily wholesale range > €0.15
- Verbeterde ranking op extreme dagen: 77% → 82%
- **Beslissing**: Niet implementeren, marginale winst (+5%) rechtvaardigt complexiteit niet

### Waarom afwijkingen?
- **Correlatie 0.71**: Hoge wholesale volatiliteit = slechte voorspelling
- ANWB dempt volatiliteit beide kanten op (geen "rip-off")
- 4% van dagen onbetrouwbaar (extreme volatiliteit)

---

## 2. BUSINESS INZICHTEN

### Besparing per gebruikersprofiel

| Profiel | Flexibel verbruik | Besparing/jaar | Gemist vs perfect |
|---------|-------------------|----------------|-------------------|
| Licht | 2 kWh/dag | €37 | €4 |
| Gemiddeld | 4 kWh/dag | €73 | €9 |
| **EV rijder** | 8 kWh/dag | **€146** | €17 |
| Zwaar | 12 kWh/dag | €219 | €26 |

### Lookup vs Real-time Enever

| Model | Besparing (EV) | Extra |
|-------|----------------|-------|
| Lookup (89%) | €146/jaar | - |
| Real-time (100%) | €164/jaar | +€17/jaar |

**Conclusie**: €17/jaar extra rechtvaardigt API complexiteit niet.

### Pricing implicatie

| Profiel | Max besparing | Bij €79/jaar product | Netto |
|---------|---------------|----------------------|-------|
| Licht | €37 | -€79 | **-€42 verlies** |
| Gemiddeld | €73 | -€79 | **-€6 verlies** |
| EV rijder | €146 | -€79 | +€67 winst |

**Conclusie**: €79/jaar werkt alleen voor EV rijders en hoger.

---

## 3. ARCHITECTUUR BESLISSINGEN

### Wat GESCHRAPT is

| Component | Reden |
|-----------|-------|
| Coefficient server | Niet nodig - lookup volstaat |
| VPN naar Enever | Niet nodig |
| Enever API integratie | Niet nodig voor V1 |
| SKILL_10 VPN doc | Verouderd |
| Aparte database | Niet nodig |

### Wat TOEGEVOEGD wordt

| Component | Omvang |
|-----------|--------|
| JSON config met 24 waarden | 1 bestand |
| ~10 regels Python in price service | Minimaal |

### Finale architectuur

```
ENTSO-E data (al aanwezig)
        ↓
+ HOURLY_OFFSET[hour]
        ↓
= consumer_price (89% accuraat)
        ↓
binary_sensor.energy_action_use
```

---

## 4. ENEVER PARTNERSHIP IDEE

### Architectuur met Enever optie

```
User heeft Enever HA integratie?
    JA  → Lees sensor.enever_* (100% accuraat)
    NEE → ENTSO-E + coefficient (89% accuraat)
         ↓
    SYNCTACLES voegt intelligence toe
         ↓
    binary_sensor.energy_action_use
```

### Business model

| SYNCTACLES tier | Databron | Accuraat | Enever ontvangt |
|-----------------|----------|----------|-----------------|
| Basis | Coefficient | 89% | Niets |
| Pro (met Enever) | User's Enever sensor | 100% | €12.50/maand van user |

### Voordelen
- **SYNCTACLES**: 100% legitiem, geen API kosten, geen rate limits
- **Enever**: Meer Supporters, recurring revenue
- **User**: Keuze, fallback werkt altijd

**Actie**: Partnership voorstel naar Enever opstellen

---

## 5. B2C → B2B STRATEGIE

### Pad naar B2B

| Timeline | B2C users | B2B status |
|----------|-----------|------------|
| 6 maanden | 200 | - |
| 12 maanden | 500 | Eerste gesprekken |
| 18 maanden | 1000 | 1-2 pilots |
| 24 maanden | 1500 | €2-5K MRR |

### B2B targets NL

| Partij | Waarde |
|--------|--------|
| Energieleverancier (white-label) | €500-2000/maand |
| Laadpaal operators | €1000-5000/maand |
| Installateurs | €200-500/maand |
| Woningcorporaties | €2000-10000/project |

### Tibber is geen concurrent
- Tibber = energieleverancier (je moet switchen)
- SYNCTACLES = werkt met elke leverancier
- **Tibber is potentiële B2B klant**

---

## 6. PRODUCT FILOSOFIE

### Kernboodschap
> "SYNCTACLES pakt 89% van de maximale besparing, zonder configuratie."

### Wat we NIET doen
- Sensor integratie (support nightmare)
- "Je bespaarde €X" claims (niet meetbaar)
- Verbruik uitlezen (privacy, complexiteit)
- Feedback loops op user gedrag

### Wat we WEL doen
- Eén simpel signaal: **USE / WAIT / AVOID**
- Install and forget
- Werkt met elke leverancier
- Fallback altijd aanwezig

---

## 7. OPEN ACTIES

### Voor CC (implementatie)
1. [ ] HOURLY_OFFSET lookup integreren in price service
2. [ ] Enever sensor detectie toevoegen (optioneel)
3. [ ] SKILL_10 VPN doc archiveren/verwijderen

### Voor Leo (business)
1. [ ] Pricing herzien (€29-49 range?)
2. [ ] Partnership voorstel Enever opstellen
3. [ ] Marketing boodschap: "89% van maximale besparing"

---

## 8. KEY QUOTES UIT SESSIE

> "68% ranking correct, maar 89% van de besparing behaald - de foute uren zijn meestal #5 of #6, niet de duurste."

> "€17/jaar extra voor real-time is €1.42/maand. De complexiteit niet waard."

> "Tibber is geen concurrent. Tibber is een potentiële B2B klant."

> "Stay in your lane: signaal geven. Punt."

---

*Gegenereerd: 10 januari 2026*
*Vorige sessie: Market validation (zie transcript)*
