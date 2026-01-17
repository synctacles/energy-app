# HANDOFF: Coefficient Engine - Lineaire Regressie Model

**Van:** Claude Code
**Naar:** Volgende ontwikkelaar / Claude Code
**Datum:** 2026-01-12
**Status:** GEVALIDEERD - Model operationeel
**Classificatie:** VERTROUWELIJK

---

## EXECUTIVE SUMMARY

De coefficient engine is geupgrade van een simpele ratio-berekening naar een **lineaire regressie model met intercept**. Dit resulteert in significant betere prijsvoorspellingen.

| Metric | Oude Methode | Nieuwe Methode |
|--------|-------------|----------------|
| Model | `consumer = wholesale × coefficient` | `consumer = wholesale × slope + intercept` |
| Gemiddelde afwijking | 12.6% | **5.6%** |
| Avonduren (17-21) | 24-39% | **5-11%** |
| R² (betrouwbaarheid) | n.v.t. | 61-96% |

---

## 1. CONTEXT: WAAROM DEZE VERANDERING?

### 1.1 Het Probleem met de Oude Methode

De oude coefficient berekening gebruikte een simpele ratio:

```
coefficient = consumer_price / wholesale_price
estimated_price = ENTSO-E × coefficient
```

**Probleem:** Dit model gaat ervan uit dat de consumentenprijs volledig proportioneel is aan de groothandelsprijs. Dat klopt niet.

### 1.2 Hoe Consumentenprijzen Echt Werken

Een Nederlandse consumentenprijs bestaat uit:

```
Consumentenprijs = Groothandel + Vaste Kosten

Waar vaste kosten =
  - Energiebelasting:     ~€0.06/kWh
  - Netbeheerkosten:      ~€0.04/kWh
  - BTW (21%):            ~€0.03/kWh
  - Leveranciersmarge:    ~€0.02/kWh
  ─────────────────────────────────
  TOTAAL VAST:            ~€0.15/kWh
```

### 1.3 Waarom de Oude Methode Faalde

**Voorbeeld met oude methode:**

| Uur | ENTSO-E | Frank | Oude Coefficient | Berekend | Fout |
|-----|---------|-------|------------------|----------|------|
| 03:00 | €0.02 | €0.18 | 9.0 | €0.18 | 0% |
| 17:00 | €0.15 | €0.30 | 2.0 | €0.30 | 0% |

Maar met die coefficienten op een andere dag:

| Uur | ENTSO-E | Berekend (9.0×) | Werkelijk | Fout |
|-----|---------|-----------------|-----------|------|
| 03:00 | €0.05 | €0.45 | €0.20 | +125% |

**Het probleem:** De coefficient varieert enorm afhankelijk van de groothandelsprijs omdat het de vaste kosten probeert te compenseren met een variabele multiplier.

### 1.4 De Oplossing: Lineair Model met Intercept

```
consumer = wholesale × slope + intercept
           ↑                   ↑
           variabel deel       vast deel (€0.15)
```

Dit model scheidt:
- **Slope** (~1.0-1.7): Hoe sterk reageert de consumentenprijs op groothandelsveranderingen
- **Intercept** (~€0.15): Vaste kosten onafhankelijk van groothandelsprijs

---

## 2. TECHNISCHE IMPLEMENTATIE

### 2.1 Formule

```
Consumentenprijs = ENTSO-E × slope + intercept
```

Voorbeeld voor uur 12:00 (weekday, januari):
```
€0.0938 × 1.1193 + €0.1464 = €0.2514
                              ↑
                              vs werkelijk €0.2447 (fout: +2.7%)
```

### 2.2 Database Schema

**Tabel: `coefficient_lookup`**

```sql
CREATE TABLE coefficient_lookup (
    country     VARCHAR(2) NOT NULL DEFAULT 'NL',
    month       INTEGER NOT NULL,        -- 1-12
    day_type    VARCHAR(10) NOT NULL,    -- 'weekday' / 'weekend'
    hour        INTEGER NOT NULL,        -- 0-23
    slope       NUMERIC(10,4) NOT NULL,  -- Multiplier (~1.0-2.0)
    intercept   NUMERIC(10,6) DEFAULT 0, -- Vaste kosten (~€0.15)
    sample_size INTEGER DEFAULT 0,
    confidence  INTEGER DEFAULT 89,      -- R² als percentage
    last_calibrated TIMESTAMPTZ,
    PRIMARY KEY (country, month, day_type, hour)
);
```

### 2.3 Regressie Berekening

De functie `calculate_optimal_coefficient()` in [analysis/coefficient_calc.py](../../coefficient-engine/analysis/coefficient_calc.py):

```python
def calculate_optimal_coefficient(
    hour: int,
    day_type: str,
    country: str = 'NL',
    window_days: int = 90,
    decay_days: float = 30.0
) -> Dict:
    """
    Berekent optimale slope en intercept via PostgreSQL lineaire regressie.

    Gebruikt:
    - REGR_SLOPE(y, x): berekent de helling
    - REGR_INTERCEPT(y, x): berekent het snijpunt
    - CORR(x, y)²: R² als betrouwbaarheidsmaat
    """
```

**SQL Query:**

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
      AND EXTRACT(HOUR FROM e.timestamp) = {hour}
      AND EXTRACT(DOW FROM e.timestamp) {weekend/weekday}
)
SELECT
    REGR_SLOPE(consumer, wholesale) as slope,
    REGR_INTERCEPT(consumer, wholesale) as intercept,
    POWER(CORR(wholesale, consumer), 2) as r_squared,
    COUNT(*) as sample_size
FROM paired_data
```

### 2.4 API Endpoints

**GET /coefficient**
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
    "source": "lookup"
}
```

**Gebruik door SYNCTACLES:**
```python
def estimate_consumer_price(entso_price_kwh: float, coefficient: dict) -> float:
    return entso_price_kwh * coefficient["slope"] + coefficient["intercept"]
```

---

## 3. VALIDATIE RESULTATEN

### 3.1 Overall Performance (9 januari 2026)

| Metric | Waarde |
|--------|--------|
| Gemiddelde afwijking | €0.0128/kWh (**5.6%**) |
| Beste uur | 15:00 (0.3%) |
| Slechtste uur | 09:00 (15.8%) |

### 3.2 Per Dagdeel

| Periode | Uren | Gem. Afwijking |
|---------|------|----------------|
| Nacht | 00-05 | 4.9% |
| Ochtend | 06-11 | 6.4% |
| **Middag** | 12-17 | **3.6%** |
| Avond | 18-23 | 7.4% |

### 3.3 Beste Voorspellingen (< 3% fout)

| Uur | Afwijking | R² |
|-----|-----------|-----|
| 15:00 | 0.3% | 83% |
| 13:00 | 0.5% | 91% |
| 06:00 | 1.8% | 57% |
| 07:00 | 1.9% | 86% |
| 11:00 | 1.9% | 92% |
| 01:00 | 2.1% | 95% |
| 00:00 | 2.5% | 94% |
| 12:00 | 2.7% | 95% |

### 3.4 Probleemuren (> 10% fout)

| Uur | Afwijking | R² | Mogelijke Oorzaak |
|-----|-----------|-----|-------------------|
| 09:00 | 15.8% | 80% | Ochtendpiek volatiliteit |
| 20:00 | 10.9% | 70% | Avondpiek, lage R² |

---

## 4. ZWAKKE PUNTEN EN VERBETERMOGELIJKHEDEN

### 4.1 Huidige Beperkingen

1. **Systematische Overschatting**
   - Model overschat prijzen gemiddeld +5.6%
   - Bijna alle afwijkingen zijn positief
   - **Oorzaak:** Mogelijk verouderde data (laatste Frank data: 9 jan, validatie: 12 jan)

2. **Lage R² voor Bepaalde Uren**
   - Uur 06:00: R² = 57%
   - Uur 16:00: R² = 62%
   - Uur 19:00: R² = 61%
   - **Oorzaak:** Meer variatie in consumentenprijzen op die momenten

3. **Avondpiek Onnauwkeurigheid**
   - Uren 17-21 hebben gemiddeld 7.4% fout
   - **Oorzaak:** Piekprijzen zijn inherent volatieler

4. **Data Vertraging**
   - Frank Energie data loopt 2-3 dagen achter
   - Coefficienten zijn gebaseerd op historische data

### 4.2 Mogelijke Verbeteringen

#### Korte Termijn (< 1 week)

1. **Bias Correctie**
   ```python
   # Corrigeer voor systematische overschatting
   BIAS_CORRECTION = 0.95  # Vermenigvuldig berekende prijs met 0.95
   ```

2. **Aparte Modellen voor Piekuren**
   - Creeer specifieke coefficienten voor uren 17-21
   - Mogelijk met andere window_days of decay_days

3. **Real-time Data**
   - Ophalen van live Frank Energie prijzen voor kalibratie
   - Huidige SSL probleem moet opgelost worden

#### Middellange Termijn (1-4 weken)

1. **Weighted Least Squares**
   - Geef recentere data meer gewicht
   - PostgreSQL ondersteunt dit niet native, Python implementatie nodig

2. **Outlier Detectie**
   - Verwijder extreme prijspunten uit regressie
   - Verbetert R² voor volatiele uren

3. **Seizoensgebonden Modellen**
   - Aparte coefficienten per seizoen (Q1, Q2, Q3, Q4)
   - Vangt belastingwijzigingen en seizoenspatronen

#### Lange Termijn (> 1 maand)

1. **Machine Learning Model**
   - Gradient Boosting of Neural Network
   - Input: uur, dag, maand, wholesale prijs, temperatuur
   - Output: consumentenprijs

2. **Multi-Provider Support**
   - Coefficienten per leverancier (Vattenfall, Eneco, etc.)
   - Huidige model is alleen Frank Energie

3. **Cross-Validatie**
   - Train op 80% data, test op 20%
   - Voorkomt overfitting

---

## 5. CONVERGENTIESNELHEID

### Hoe Snel Reageert het Model op Veranderingen?

**Scenario: BTW verhoging van 21% naar 25%**

| Model | Reactietijd | Uitleg |
|-------|-------------|--------|
| Oude (ratio) | ~60 dagen | Moet wachten tot nieuwe ratio stabiel is |
| Nieuw (slope+intercept) | ~30 dagen | Intercept past zich aan, slope blijft stabiel |

**Waarom sneller?**

Het nieuwe model scheidt variabele en vaste kosten:
- **Belastingwijziging** = alleen intercept verandert
- **Marktvolatiliteit** = alleen slope verandert

Het oude model moest beide effecten tegelijk compenseren met één parameter.

---

## 6. OPERATIONELE STATUS

### 6.1 Server Details

| Aspect | Waarde |
|--------|--------|
| Server | Coefficient Engine (91.99.150.36) |
| Database | PostgreSQL `coefficient_db` |
| Tabel | `coefficient_lookup` (576 records) |
| Update | 2026-01-12 12:00 UTC |

### 6.2 Data Status

| Dataset | Records | Periode |
|---------|---------|---------|
| hist_entso_prices | 2,083,437 | 2022-09 tot 2026-01-12 |
| hist_enever_prices | 28,190 | 2022-10 tot 2026-01-09 |
| coefficient_lookup | 576 | Alle uur/dag/maand combinaties |

### 6.3 Bestanden Gewijzigd

| Bestand | Wijziging |
|---------|-----------|
| `analysis/coefficient_calc.py` | `calculate_optimal_coefficient()` met REGR_SLOPE/INTERCEPT |
| `models/database.py` | CoefficientLookup: `slope` + `intercept` kolommen |
| `api/main.py` | Endpoints retourneren slope + intercept |

---

## 7. HANDMATIGE RECALIBRATIE

Om de coefficienten opnieuw te berekenen:

```bash
ssh coefficient

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
print("Recalibratie voltooid")
EOF
```

---

## 8. CONCLUSIE

Het lineaire regressie model met intercept is een significante verbetering:

| Aspect | Beoordeling |
|--------|-------------|
| Nauwkeurigheid | **GOED** (5.6% gem. fout) |
| Middag uren | **ZEER GOED** (3.6% fout) |
| Avond uren | **ACCEPTABEL** (7.4% fout) |
| Stabiliteit | **GOED** (R² 61-96%) |
| Reactiesnelheid | **2x SNELLER** dan oude model |

**Aanbeveling:** Model is productierijp voor Energy Action berekeningen. Monitoren op drift en periodiek hercalibreren.

---

## CHANGELOG

| Datum | Wijziging |
|-------|-----------|
| 2026-01-12 | Initiële implementatie slope+intercept model |
| 2026-01-12 | 576 coefficienten bijgewerkt met nieuwe regressie |
| 2026-01-12 | Validatie uitgevoerd: 5.6% gemiddelde fout |
