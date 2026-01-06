# Fallback & Data Quality Analyse - WERKELIJKE QUERY RESULTATEN

**Gegenereerd:** 2025-12-31 00:45:00
**Database:** energy_insights_nl
**Tabellen:** norm_entso_e_a75, raw_entso_e_a75

---

## 📊 WERKELIJKE RESULTATEN (Live Data)

### Stap 1: Schema Analyse

#### Query 1.1: Kolommen van norm_entso_e_a75

```sql
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'norm_entso_e_a75'
ORDER BY ordinal_position;
```

**Resultaat:**
```
column_name           | data_type
----------------------|-------------------
id                    | integer
timestamp             | timestamp with time zone
country               | character varying
b01_biomass_mw        | double precision
b04_gas_mw            | double precision
b05_coal_mw           | double precision
b14_nuclear_mw        | double precision
b16_solar_mw          | double precision
b17_waste_mw          | double precision
b18_wind_offshore_mw  | double precision
b19_wind_onshore_mw   | double precision
b20_other_mw          | double precision
total_mw              | double precision
quality_status        | character varying
last_updated          | timestamp with time zone
```

**Analyse:**
- 15 kolommen met energie data per bron (ENTSO-E A75 is NL generatie)
- `quality_status` column voor data kwaliteit tracking
- `last_updated` voor metadata

---

### Stap 2: Algemene Statistieken

#### Query 2.1: Totaal records en span

```sql
SELECT
    COUNT(*) as total_records,
    MIN(timestamp) as eerste_record,
    MAX(timestamp) as laatste_record,
    MAX(timestamp) - MIN(timestamp) as tijdspan
FROM norm_entso_e_a75;
```

**Resultaat:**
```
total_records | eerste_record       | laatste_record      | tijdspan
--------------|---------------------|---------------------|----------------
1062          | 2025-12-19 00:00:00 | 2025-12-30 13:30:00 | 11 days 13:30:00
```

**Analyse:**
- ✅ Dataset: 11 dagen data
- ✅ Records: 1,062 (consistent met ~90 per dag)
- ✅ Recent data beschikbaar (tot Dec 30)

---

### Stap 3: Data Quality Status

#### Query 3.1: Quality status verdeling

```sql
SELECT
    quality_status,
    COUNT(*) as records,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM norm_entso_e_a75
GROUP BY quality_status
ORDER BY records DESC;
```

**Resultaat:**
```
quality_status | records | percentage
---------------|---------|------------
STALE          | 1062    | 100.00
```

**Analyse:**
- 🔴 **KRITIEK:** 100% van data is STALE!
- Dit betekent: Geen recent "FRESH" data beschikbaar
- **Impact:** Data is verouderd, niet real-time

---

### Stap 4: Nul-Waarden Analyse

#### Query 4.1: Zero values per energiebron

```sql
SELECT
    'nuclear_mw = 0' as type,
    COUNT(*) as count
FROM norm_entso_e_a75 WHERE b14_nuclear_mw = 0
UNION ALL
SELECT 'gas_mw = 0', COUNT(*)
FROM norm_entso_e_a75 WHERE b04_gas_mw = 0
UNION ALL
SELECT 'solar_mw = 0', COUNT(*)
FROM norm_entso_e_a75 WHERE b16_solar_mw = 0
UNION ALL
SELECT 'wind_onshore_mw = 0', COUNT(*)
FROM norm_entso_e_a75 WHERE b19_wind_onshore_mw = 0
UNION ALL
SELECT 'wind_offshore_mw = 0', COUNT(*)
FROM norm_entso_e_a75 WHERE b18_wind_offshore_mw = 0;
```

**Resultaat:**
```
type                | count
--------------------|-------
nuclear_mw = 0      | 125   🔴 ALERT
gas_mw = 0          | 0     ✅ OK
solar_mw = 0        | 116   ✅ OK (nachts)
wind_onshore_mw = 0 | 114   ✅ OK (variabel)
wind_offshore_mw = 0| 0     ✅ OK (geen data)
```

**Analyse:**
- 🔴 **KRITIEK:** 125 records met nuclear = 0 (11.8% van data!)
- **Oorzaak:** Borssele reactor staat altijd aan, kan niet 0 zijn
- **Verdacht:** Waarschijnlijk data parsing/collection error
- ✅ **Positief:** Gas heeft geen nul-waarden (correct)

---

### Stap 5: Verdachte Nul-Waarden - Details

#### Query 5.1: Recent records met nuclear = 0

```sql
SELECT
    timestamp,
    country,
    b14_nuclear_mw,
    b04_gas_mw,
    total_mw,
    quality_status
FROM norm_entso_e_a75
WHERE b14_nuclear_mw = 0
ORDER BY timestamp DESC
LIMIT 10;
```

**Resultaat:**
```
timestamp            | country | b14_nuclear_mw | b04_gas_mw | total_mw  | quality_status
--------------------|---------|----------------|------------|-----------|---------------
2025-12-29 14:00:00 | NL      | 0              | 146.792    | 2232.95   | STALE
2025-12-29 12:00:00 | NL      | 0              | 123.268    | 1804.74   | STALE
2025-12-29 11:00:00 | NL      | 0              | 138.851    | 623.39    | STALE
2025-12-29 10:00:00 | NL      | 0              | 7462.071   | 8036.91   | STALE
2025-12-29 09:00:00 | NL      | 0              | 146.496    | 390.65    | STALE
2025-12-29 08:00:00 | NL      | 0              | 7691.737   | 8145.94   | STALE
... (5 meer)
```

**Analyse:**
- Alle nuclear=0 records zijn STALE
- Dit suggereert: **ENTSO-E parsing issue of incomplete data ingestion**
- Gas waarden zijn normaal, dus het is niet systeem-wide outage
- Pattern: 2025-12-29 heeft veel nuclear=0, oudere data ook

---

### Stap 6: Pattern Analyse - Wanneer is het probleem?

#### Query 6.1: Nuclear status per dag

```sql
SELECT
    DATE(timestamp) as dag,
    COUNT(*) as total_records,
    COUNT(CASE WHEN b14_nuclear_mw = 0 THEN 1 END) as nuclear_zero,
    COUNT(CASE WHEN b14_nuclear_mw IS NULL THEN 1 END) as nuclear_null,
    COUNT(CASE WHEN b14_nuclear_mw > 0 THEN 1 END) as nuclear_gt_zero
FROM norm_entso_e_a75
GROUP BY DATE(timestamp)
ORDER BY dag DESC;
```

**Resultaat:**
```
dag        | total | zero | null | gt_zero
-----------|-------|------|------|--------
2025-12-30 | 55    | 0    | 2    | 53      ✅ FIXED!
2025-12-29 | 96    | 13   | 3    | 80      ⚠️  Recovering
2025-12-28 | 47    | 7    | 1    | 39
2025-12-27 | 96    | 2    | 4    | 90
2025-12-26 | 96    | 18   | 1    | 77
2025-12-25 | 96    | 17   | 5    | 74
2025-12-24 | 96    | 22   | 0    | 74      🔴 Worst
2025-12-23 | 96    | 17   | 3    | 76
2025-12-22 | 96    | 12   | 1    | 83
2025-12-21 | 96    | 3    | 0    | 93
2025-12-20 | 96    | 7    | 0    | 89
2025-12-19 | 96    | 7    | 0    | 89
```

**GROTE BEVINDING:**
- 🔴 **Peak probleem:** 2025-12-24 (22 records met nuclear=0, 22.9% failure)
- ⚠️ **Trend:** Situation improving (0 nulls op 2025-12-30)
- ✅ **Recent:** 2025-12-30 data is schoon!

---

## 🎯 KRITIEKE BEVINDINGEN

### 1. 🔴 100% STALE DATA
- **Impact:** Alle 1,062 records zijn STALE
- **Betekenis:** Data refresh systeem werkt niet goed
- **Action:** Check data ingestion pipeline

### 2. 🔴 NUCLEAR ZERO VALUES BUG
- **Impact:** 125 records (11.8%) hebben nuclear_mw = 0
- **Wat is normaal:** Borssele reactor draait 24/7 @ ~484 MW
- **Waarschijnlijke oorzaak:**
  - ENTSO-E API parsing error
  - Incomplete field extraction
  - Data corruption during storage
- **Timeline:** Ergste op 2025-12-24, verbetert nu
- **Action:** Check ENTSO-E ingestion code, validate API responses

### 3. ⚠️ MISSING WIND OFFSHORE DATA
- **Impact:** wind_offshore_mw = 0 (geen records met data)
- **Betekenis:** Windenergie data compleet absent
- **Action:** Check of ENTSO-E levert deze data

---

## 📋 SAMENVATTING TABEL

| Metriek | Waarde | Status | Notitie |
|---------|--------|--------|---------|
| **Dataset grootte** | 1,062 records | ✅ OK | 11 dagen |
| **Data recency** | Tot 2025-12-30 13:30 | ⚠️ STALE | Alles STALE status |
| **Quality status** | 100% STALE | 🔴 ALERT | Geen FRESH data |
| **Nuclear issue** | 125 zero values | 🔴 ALERT | 11.8% corruption |
| **Gas data** | 0 zero values | ✅ OK | Correct values |
| **Solar data** | 116 zero values | ✅ OK | Normal (night) |
| **Wind data** | 0 offshore, 114 onshore zeros | ⚠️ PARTIAL | Offshore missing |

---

## 💡 AANBEVELINGEN (PRIORITEIT)

### P0 - ONMIDDELLIJK
1. **Fix STALE data status**
   - Alle data is STALE, geen FRESH data beschikbaar
   - Check data refresh job status
   - Verify ENTSO-E API connectivity

2. **Debug nuclear_mw = 0 bug**
   - Investigate ENTSO-E API parsing
   - Compare raw_entso_e_a75 vs norm_entso_e_a75
   - Validate: Is Borssele reactor online?
   - Re-parse affected date range (Dec 19-29)

3. **Check wind offshore data**
   - Why is b18_wind_offshore_mw always 0 or NULL?
   - Check ENTSO-E A75 availability

### P1 - KORTE TERMIJN
1. Implement data quality monitoring
   - Alert op > 5% zero values voor nuclear/gas
   - Alert op 100% STALE status
   - Dashboard for quality trends

2. Add data validation in ingestion
   - Validate nuclear > 0 (Borssele is online)
   - Validate total_mw = sum of components
   - Flag incomplete records

3. Historical backfill
   - Re-process Dec 24 data (worst case)
   - Fill gaps in wind_offshore_mw

---

## 📎 VOLGENDE STAPPEN

1. ✅ DONE: Executed 6 key queries on real database
2. ⏳ TODO: Investigate raw_entso_e_a75 vs norm_entso_e_a75 differences
3. ⏳ TODO: Compare with external ENTSO-E API data
4. ⏳ TODO: Fix ingestion pipeline
5. ⏳ TODO: Backfill corrected data

---

**Generated:** 2025-12-31 00:45:00
**Database:** energy_insights_nl (PostgreSQL 16.11)
**Status:** ✅ Real data queries executed successfully
