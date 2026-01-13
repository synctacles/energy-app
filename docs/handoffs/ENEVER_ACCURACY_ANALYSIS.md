# ANALYSE: Enever Prijsdata Nauwkeurigheid

**Datum:** 2026-01-13
**Dataset:** 26.543 uur-vergelijkingen (2022-12-31 tot 2026-01-13)
**Vergelijking:** Enever "Frank Energie" vs Echte Frank API prijzen

---

## EXECUTIVE SUMMARY

**Conclusie: Enever data is NIET geschikt als primaire bron voor Energy Actions.**

| Metric | Waarde | Beoordeling |
|--------|--------|-------------|
| Gemiddelde absolute fout | €23.25/MWh | HOOG |
| Uren met >10% afwijking | 30.7% | ONACCEPTABEL |
| Uren met >20% afwijking | 12.2% | KRITIEK |
| Energy Action accuracy | 76.4% | ONVOLDOENDE |
| Kritieke fouten (USE↔AVOID) | 0.4% | Acceptabel |

---

## 1. WAT BEREKENT ENEVER?

Enever publiceert "Frank Energie" prijzen, maar deze komen **NIET** van de Frank API.

**Enever's methode (gereconstrueerd):**
```
enever_frank_prijs = ENTSO-E_wholesale × factor + opslag
```

Dit is vergelijkbaar met ons coefficient model, maar met systematische fouten.

**Bewijs:** De afwijkingen correleren sterk met marktprijsveranderingen (r=0.65 in avonduren), wat wijst op vertraagde marktdata.

---

## 2. AFWIJKINGEN PER UUR

### Overzichtstabel

| Uur | Enever | Frank | Fout (€/MWh) | Fout % | >10% fout | Richting |
|-----|--------|-------|--------------|--------|-----------|----------|
| 00 | €0.261 | €0.250 | -11.5 | -4.9% | 10.9% | OVERSCHAT |
| 01 | €0.254 | €0.246 | -7.6 | -3.3% | 6.3% | OVERSCHAT |
| 02 | €0.250 | €0.245 | -5.2 | -2.3% | 4.1% | OVERSCHAT |
| 03 | €0.246 | €0.248 | +1.6 | +0.5% | 3.7% | GEMENGD |
| **04** | €0.246 | €0.260 | **+14.1** | **+5.0%** | **24.1%** | ONDERSCHAT |
| **05** | €0.252 | €0.275 | **+23.0** | **+7.4%** | **38.2%** | ONDERSCHAT |
| **06** | €0.269 | €0.281 | **+12.2** | **+3.2%** | **27.1%** | ONDERSCHAT |
| **07** | €0.286 | €0.274 | -12.6 | -6.7% | **37.2%** | OVERSCHAT |
| **08** | €0.289 | €0.257 | **-32.4** | **-15.6%** | **53.8%** | OVERSCHAT |
| **09** | €0.272 | €0.241 | **-31.4** | **-16.3%** | **51.5%** | OVERSCHAT |
| **10** | €0.253 | €0.229 | -24.3 | -15.8% | **40.5%** | OVERSCHAT |
| 11 | €0.239 | €0.221 | -18.0 | -15.2% | 31.9% | OVERSCHAT |
| 12 | €0.229 | €0.220 | -8.6 | -8.9% | 20.3% | OVERSCHAT |
| 13 | €0.223 | €0.229 | +6.1 | +1.9% | 15.2% | ONDERSCHAT |
| **14** | €0.225 | €0.244 | **+19.5** | **+8.7%** | **31.6%** | ONDERSCHAT |
| **15** | €0.235 | €0.268 | **+33.1** | **+13.3%** | **47.0%** | ONDERSCHAT |
| **16** | €0.251 | €0.297 | **+45.8** | **+15.9%** | **58.0%** | ONDERSCHAT |
| **17** | €0.279 | €0.317 | **+37.7** | **+11.8%** | **52.4%** | ONDERSCHAT |
| 18 | €0.301 | €0.315 | +13.9 | +3.9% | 40.4% | GEMENGD |
| 19 | €0.313 | €0.296 | -16.8 | -6.0% | 33.1% | OVERSCHAT |
| 20 | €0.308 | €0.281 | -27.4 | -9.9% | 35.0% | OVERSCHAT |
| 21 | €0.292 | €0.270 | -22.0 | -8.4% | 33.9% | OVERSCHAT |
| 22 | €0.280 | €0.262 | -18.1 | -7.4% | 23.2% | OVERSCHAT |
| 23 | €0.267 | €0.255 | -12.1 | -5.4% | 17.3% | OVERSCHAT |

### Kritieke uren (>40% kans op >10% fout)

| Uur | % >10% fout | Probleem |
|-----|-------------|----------|
| 08:00 | **53.8%** | Enever overschat (markt daalt na ochtendpiek) |
| 09:00 | **51.5%** | Enever overschat |
| 16:00 | **58.0%** | Enever onderschat (avondpiek start) |
| 17:00 | **52.4%** | Enever onderschat |

---

## 3. PATROON ANALYSE

### Waarom wijkt Enever af?

```
DAGPATROON VAN ENEVER'S FOUTEN:

  OVERSCHAT                              ONDERSCHAT
  (Enever te hoog)                       (Enever te laag)
      │                                      │
      │    ┌──────┐                          │      ┌──────┐
      │    │ 7-12 │                          │      │15-17 │
      │    │ uur  │                          │      │ uur  │
      ▼    └──────┘                          ▼      └──────┘

  00  02  04  06  08  10  12  14  16  18  20  22  24
  ─────────────────────────────────────────────────►

  ◄─────── NACHT ──────►◄──── DAG ────►◄── AVOND ──►

  Markt:  Daalt        Stijgt→Daalt    Stijgt→Daalt
  Enever: Volgt traag  Volgt traag     Volgt traag
```

**Root cause:** Enever gebruikt vertraagde of gesmoothde marktdata.

- Wanneer markt **stijgt** (ochtend 4-7u, middag 13-17u): Enever **onderschat**
- Wanneer markt **daalt** (ochtend 8-12u, avond 19-23u): Enever **overschat**

---

## 4. IMPACT OP ENERGY ACTIONS

### Definitie Energy Actions

| Actie | Prijsgrens | Betekenis |
|-------|------------|-----------|
| USE | < €0.20/kWh | Energie gebruiken (goedkoop) |
| WAIT | €0.20 - €0.28/kWh | Afwachten |
| AVOID | > €0.28/kWh | Vermijd gebruik (duur) |

### Accuracy per uur

| Uur | Accuracy | USE→AVOID fout | AVOID→USE fout | Totaal kritiek |
|-----|----------|----------------|----------------|----------------|
| 00 | 87.0% | 0.0% | 0.0% | 0.0% |
| 01 | 91.7% | 0.0% | 0.0% | 0.0% |
| 02 | 93.8% | 0.0% | 0.0% | 0.0% |
| 03 | 92.6% | 0.0% | 0.0% | 0.0% |
| 04 | 79.2% | 0.3% | 0.0% | 0.3% |
| 05 | 70.8% | 0.4% | 0.0% | 0.4% |
| 06 | 78.2% | 0.1% | 0.2% | 0.3% |
| 07 | 73.9% | 0.0% | 0.0% | 0.0% |
| **08** | **67.2%** | 0.0% | **1.2%** | **1.2%** |
| 09 | 69.7% | 0.0% | 0.7% | 0.7% |
| 10 | 75.5% | 0.0% | 0.1% | 0.1% |
| 11 | 81.8% | 0.0% | 0.1% | 0.1% |
| 12 | 89.6% | 0.0% | 0.0% | 0.0% |
| 13 | 88.4% | 0.1% | 0.0% | 0.1% |
| 14 | 80.1% | 0.2% | 0.0% | 0.2% |
| 15 | 66.1% | 0.7% | 0.0% | 0.7% |
| **16** | **59.2%** | **2.3%** | 0.0% | **2.3%** |
| **17** | **65.7%** | **1.2%** | 0.0% | **1.2%** |
| 18 | 71.0% | 0.3% | 0.0% | 0.3% |
| 19 | 74.6% | 0.1% | 0.1% | 0.2% |
| 20 | 74.2% | 0.0% | 0.0% | 0.0% |
| 21 | 72.6% | 0.0% | 0.1% | 0.1% |
| 22 | 76.6% | 0.0% | 0.2% | 0.2% |
| 23 | 84.5% | 0.0% | 0.2% | 0.2% |

### Kritieke fouten uitgelegd

**USE wanneer AVOID correct was (2.3% om 16:00):**
- Enever zegt: "Prijs is €0.19/kWh - GEBRUIK energie!"
- Werkelijk: "Prijs is €0.30/kWh - VERMIJD energie!"
- **Gevolg:** Gebruiker verbruikt energie op duurste moment

**AVOID wanneer USE correct was (1.2% om 08:00):**
- Enever zegt: "Prijs is €0.29/kWh - VERMIJD energie!"
- Werkelijk: "Prijs is €0.18/kWh - GEBRUIK energie!"
- **Gevolg:** Gebruiker mist goedkoop moment

---

## 5. KANS OP VERKEERDE BESLISSING

### Per uur: Hoeveel % meer kans op foute Energy Action?

Vergelijking met perfecte data (Frank API):

| Uur | Enever Accuracy | Extra fout-kans | Risico-niveau |
|-----|-----------------|-----------------|---------------|
| 00-03 | 87-94% | +6-13% | LAAG |
| 04-07 | 70-79% | +21-30% | MIDDEL |
| **08-09** | **67-70%** | **+30-33%** | **HOOG** |
| 10-13 | 76-90% | +10-24% | MIDDEL |
| 14-15 | 66-80% | +20-34% | HOOG |
| **16-17** | **59-66%** | **+34-41%** | **KRITIEK** |
| 18-23 | 71-85% | +15-29% | MIDDEL |

### Interpretatie

**Op basis van Enever data heeft een gebruiker:**

- **+41% meer kans** op verkeerde beslissing om **16:00** (avondpiek start)
- **+34% meer kans** op verkeerde beslissing om **15:00**
- **+33% meer kans** op verkeerde beslissing om **08:00** (na ochtendpiek)

**Dit zijn precies de momenten waarop Energy Actions het meest waardevol zijn!**

---

## 6. CONCLUSIE: BRUIKBAARHEID VOOR ENERGY ACTIONS

### Verdict: NIET GESCHIKT als primaire bron

| Criterium | Vereist | Enever | Status |
|-----------|---------|--------|--------|
| Gemiddelde accuracy | >90% | 76.4% | ❌ FAIL |
| Accuracy tijdens piekuren | >85% | 59-70% | ❌ FAIL |
| Kritieke fouten | <0.5% | 0.4% | ✅ PASS |
| >10% afwijking | <10% | 30.7% | ❌ FAIL |

### Aanbevelingen

1. **Tier 1 & 2**: Gebruik altijd Frank Direct API of Frank DB
2. **Tier 3 (Enever-Frank)**: Alleen als fallback MET correctie-coefficienten
3. **Tier 4+**: ENTSO-E + coefficient model is betrouwbaarder dan raw Enever

### Risico zonder correctie

Als Synctacles Enever data zou gebruiken zonder correctie:

- **30.7%** van de uren heeft >10% prijsfout
- **12.2%** van de uren heeft >20% prijsfout
- Tot **41% extra kans** op verkeerde Energy Action tijdens kritieke uren
- Gebruikers zouden **vertrouwen verliezen** in het systeem

### Waarom het Enever-Frank coefficient model WEL werkt

Het coefficient model corrigeert systematisch voor Enever's fouten:
- Per uur specifieke correctie
- Per dag-type (weekday/weekend)
- Per maand (seizoensgebonden)

Na correctie wordt de accuracy significant verbeterd.

---

## APPENDIX: Technische Details

### Dataset
- **Bron:** hist_enever_prices + hist_frank_prices
- **Periode:** 2022-12-31 tot 2026-01-13
- **Vergelijkingen:** 26.543 uur-paren
- **Filter:** leverancier = 'Frank Energie', prijs > 0

### Coefficient Model Hercalibratie
- **Oude bron:** hist_enever_prices (Enever's berekening)
- **Nieuwe bron:** hist_frank_prices (echte Frank API)
- **Resultaat:** 531 coefficienten (12 maanden × 2 dag_types × ~22 uren)
- **Gemiddelde slope:** 1.2785 (was 1.3023)
- **Gemiddelde intercept:** €0.148/kWh
- **Confidence:** 88.8% (was 84.4%)

---

*Rapport gegenereerd door Claude Code op 2026-01-13*
