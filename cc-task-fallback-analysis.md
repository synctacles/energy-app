# CC Task: Fallback & Data Quality Analyse

## Context

Server: 135.181.255.83 (SSH as root)
Database: energy_insights_nl (PostgreSQL)
User: energy_insights_nl

## Doel

Genereer een compleet overzicht van:
1. Data sources gebruikt (ENTSO-E vs Energy-Charts vs Cache)
2. Data quality verdeling (FRESH/STALE/FALLBACK/UNAVAILABLE)
3. Nul-waarden in ENTSO-E data
4. Fallback events (wanneer, hoe vaak)
5. Backfill status (needs_backfill flags)
6. Achteraf herstelde data (BACKFILLED records)

---

## Stap 1: Schema Analyse

```sql
-- Check welke columns bestaan in norm_generation
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'norm_generation'
ORDER BY ordinal_position;

-- Check ook raw tables
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'raw_entso_e_a75'
ORDER BY ordinal_position;
```

---

## Stap 2: Algemene Statistieken

```sql
-- Totaal records en tijdspan
SELECT 
    COUNT(*) as total_records,
    MIN(timestamp) as eerste_record,
    MAX(timestamp) as laatste_record,
    MAX(timestamp) - MIN(timestamp) as tijdspan
FROM norm_generation;
```

---

## Stap 3: Data Source Verdeling

```sql
-- Per bron: hoeveel records
SELECT 
    data_source,
    COUNT(*) as records,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage,
    MIN(timestamp) as eerste,
    MAX(timestamp) as laatste
FROM norm_generation
GROUP BY data_source
ORDER BY records DESC;
```

---

## Stap 4: Data Quality Verdeling

```sql
-- Per quality status
SELECT 
    data_quality,
    COUNT(*) as records,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM norm_generation
GROUP BY data_quality
ORDER BY records DESC;
```

---

## Stap 5: Nul-Waarden Analyse

```sql
-- Records met nul-waarden per PSR type
SELECT 
    'solar_mw = 0' as type,
    COUNT(*) as count,
    MIN(timestamp) as eerste,
    MAX(timestamp) as laatste
FROM norm_generation WHERE solar_mw = 0
UNION ALL
SELECT 'wind_offshore_mw = 0', COUNT(*), MIN(timestamp), MAX(timestamp)
FROM norm_generation WHERE wind_offshore_mw = 0
UNION ALL
SELECT 'wind_onshore_mw = 0', COUNT(*), MIN(timestamp), MAX(timestamp)
FROM norm_generation WHERE wind_onshore_mw = 0
UNION ALL
SELECT 'nuclear_mw = 0', COUNT(*), MIN(timestamp), MAX(timestamp)
FROM norm_generation WHERE nuclear_mw = 0
UNION ALL
SELECT 'gas_mw = 0', COUNT(*), MIN(timestamp), MAX(timestamp)
FROM norm_generation WHERE gas_mw = 0
UNION ALL
SELECT 'biomass_mw = 0', COUNT(*), MIN(timestamp), MAX(timestamp)
FROM norm_generation WHERE biomass_mw = 0;

-- Verdachte nul-waarden (bv nuclear of gas = 0 is onwaarschijnlijk)
SELECT 
    timestamp,
    nuclear_mw,
    gas_mw,
    data_source,
    data_quality
FROM norm_generation
WHERE nuclear_mw = 0 OR gas_mw = 0
ORDER BY timestamp DESC
LIMIT 20;
```

---

## Stap 6: Fallback Events

```sql
-- Alle fallback events (niet ENTSO-E)
SELECT 
    timestamp,
    data_source,
    data_quality,
    total_mw,
    renewable_percentage
FROM norm_generation
WHERE data_source != 'ENTSO-E' OR data_quality IN ('FALLBACK', 'CACHED', 'FORWARD_FILL')
ORDER BY timestamp DESC
LIMIT 50;

-- Fallback events per dag
SELECT 
    DATE(timestamp) as dag,
    data_source,
    data_quality,
    COUNT(*) as events
FROM norm_generation
WHERE data_source != 'ENTSO-E' OR data_quality NOT IN ('FRESH', 'STALE')
GROUP BY DATE(timestamp), data_source, data_quality
ORDER BY dag DESC;
```

---

## Stap 7: Backfill Status

```sql
-- Records die backfill nodig hebben
SELECT 
    COUNT(*) as needs_backfill_count
FROM norm_generation
WHERE needs_backfill = true;

-- Details van backfill candidates
SELECT 
    timestamp,
    data_source,
    data_quality,
    needs_backfill
FROM norm_generation
WHERE needs_backfill = true
ORDER BY timestamp DESC
LIMIT 20;

-- Reeds ge-backfillde records
SELECT 
    COUNT(*) as backfilled_count
FROM norm_generation
WHERE data_quality = 'BACKFILLED';

-- Details backfilled
SELECT 
    timestamp,
    data_source,
    data_quality,
    updated_at
FROM norm_generation
WHERE data_quality = 'BACKFILLED'
ORDER BY updated_at DESC
LIMIT 20;
```

---

## Stap 8: Tijdlijn Analyse

```sql
-- Laatste 24 uur per uur
SELECT 
    DATE_TRUNC('hour', timestamp) as uur,
    data_source,
    data_quality,
    COUNT(*) as records,
    ROUND(AVG(total_mw), 0) as avg_mw
FROM norm_generation
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', timestamp), data_source, data_quality
ORDER BY uur DESC;
```

---

## Stap 9: Load & Balance Tables (indien aanwezig)

```sql
-- Check load table
SELECT 
    data_source,
    data_quality,
    COUNT(*) as records
FROM norm_load
GROUP BY data_source, data_quality;

-- Check balance table
SELECT 
    data_source,
    data_quality,
    COUNT(*) as records
FROM norm_grid_balance
GROUP BY data_source, data_quality;
```

---

## Stap 10: Genereer Rapport

Maak een samenvattend rapport met:

```markdown
# Fallback & Data Quality Rapport

## Periode
- Eerste record: [datum]
- Laatste record: [datum]
- Totaal records: [aantal]

## Data Sources
| Bron | Records | Percentage |
|------|---------|------------|
| ENTSO-E | x | y% |
| Energy-Charts | x | y% |
| Cache | x | y% |

## Data Quality
| Status | Records | Percentage |
|--------|---------|------------|
| FRESH | x | y% |
| STALE | x | y% |
| FALLBACK | x | y% |

## Nul-Waarden (Verdacht)
| Type | Count | Opmerking |
|------|-------|-----------|
| nuclear = 0 | x | Onwaarschijnlijk (Borssele draait altijd) |
| gas = 0 | x | Onwaarschijnlijk |

## Fallback Events
- Totaal fallback events: x
- Energy-Charts gebruikt: x keer
- Cache gebruikt: x keer

## Backfill Status
- Needs backfill: x records
- Reeds ge-backfilled: x records

## Conclusie
[Betrouwbaarheid percentage]
[Aanbevelingen]
```

Sla rapport op als: `/opt/energy-insights-nl/reports/fallback-report-YYYYMMDD.md`

---

## Output

1. Alle query resultaten tonen
2. Markdown rapport genereren
3. Rapport opslaan op server
4. Samenvatting teruggeven

