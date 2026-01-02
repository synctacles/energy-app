# CC Task: Fallback Validation Queries

**Status**: Ready for execution
**Database**: energy_insights_nl
**Focus**: Validate fallback logic, data sources, and quality metrics

---

## 6 Validation Queries

### Query 1: Quality Status Verdeling (FRESH vs STALE)

**Doel**: Check hoeveel records zijn FRESH vs STALE per tabel

```sql
-- A44 Prices Quality Distribution
SELECT
    data_quality as quality_status,
    COUNT(*) as record_count,
    ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM norm_entso_e_a44), 2) as percentage,
    MIN(timestamp) as oldest,
    MAX(timestamp) as newest
FROM norm_entso_e_a44
GROUP BY data_quality
ORDER BY record_count DESC;

-- A75 Generation Quality Distribution
SELECT
    quality_status,
    COUNT(*) as record_count,
    ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM norm_entso_e_a75), 2) as percentage,
    MIN(timestamp) as oldest,
    MAX(timestamp) as newest
FROM norm_entso_e_a75
WHERE country = 'NL'
GROUP BY quality_status
ORDER BY record_count DESC;

-- A65 Load Quality Distribution
SELECT
    quality_status,
    COUNT(*) as record_count,
    ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM norm_entso_e_a65), 2) as percentage,
    MIN(timestamp) as oldest,
    MAX(timestamp) as newest
FROM norm_entso_e_a65
WHERE country = 'NL'
GROUP BY quality_status
ORDER BY record_count DESC;
```

---

### Query 2: Data Source Verdeling (ENTSO-E vs Energy-Charts vs Cache)

**Doel**: Check which data sources zijn gebruikt voor A44 prices

```sql
SELECT
    data_source,
    COUNT(*) as record_count,
    ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM norm_entso_e_a44), 2) as percentage,
    MIN(timestamp) as oldest_record,
    MAX(timestamp) as newest_record,
    ROUND(AVG(CAST(price_eur_mwh AS DECIMAL)), 2) as avg_price,
    MIN(CAST(price_eur_mwh AS DECIMAL)) as min_price,
    MAX(CAST(price_eur_mwh AS DECIMAL)) as max_price
FROM norm_entso_e_a44
GROUP BY data_source
ORDER BY record_count DESC;
```

---

### Query 3: Backfill Status (needs_backfill + backfilled counts)

**Doel**: Check backfill status per bron

```sql
-- Backfill Status Overview
SELECT
    data_source,
    needs_backfill,
    COUNT(*) as record_count,
    ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM norm_entso_e_a44 WHERE data_source = n.data_source), 2) as pct_of_source
FROM norm_entso_e_a44 n
GROUP BY data_source, needs_backfill
ORDER BY data_source, needs_backfill DESC;

-- Summary: How many records need backfill?
SELECT
    COUNT(*) as total_records,
    SUM(CASE WHEN needs_backfill = true THEN 1 ELSE 0 END) as needs_backfill_count,
    SUM(CASE WHEN needs_backfill = false THEN 1 ELSE 0 END) as backfilled_count,
    ROUND(100.0 * SUM(CASE WHEN needs_backfill = true THEN 1 ELSE 0 END) / COUNT(*), 2) as needs_backfill_pct
FROM norm_entso_e_a44;

-- Records currently needing backfill
SELECT
    COUNT(*) as pending_backfill,
    MIN(timestamp) as earliest_gap,
    MAX(timestamp) as latest_gap,
    DATEDIFF(day, MIN(timestamp), MAX(timestamp)) as day_range
FROM norm_entso_e_a44
WHERE needs_backfill = true;
```

---

### Query 4: Laatste 24h Breakdown

**Doel**: Detailed breakdown van afgelopen 24 uur per data source en quality

```sql
-- Last 24h: A44 Prices by source and quality
SELECT
    DATE(timestamp) as date,
    data_source,
    data_quality,
    COUNT(*) as record_count,
    ROUND(AVG(CAST(price_eur_mwh AS DECIMAL)), 2) as avg_price,
    MIN(CAST(price_eur_mwh AS DECIMAL)) as min_price,
    MAX(CAST(price_eur_mwh AS DECIMAL)) as max_price,
    ROUND(STDDEV(CAST(price_eur_mwh AS DECIMAL)), 2) as stddev_price
FROM norm_entso_e_a44
WHERE timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY DATE(timestamp), data_source, data_quality
ORDER BY date DESC, data_source, data_quality;

-- Last 24h: A75 Generation by source (if available)
SELECT
    DATE(timestamp) as date,
    quality_status,
    COUNT(*) as record_count,
    ROUND(AVG(b14_nuclear_mw), 2) as avg_nuclear,
    ROUND(AVG(b19_wind_onshore_mw), 2) as avg_wind_onshore,
    ROUND(AVG(b16_solar_mw), 2) as avg_solar,
    ROUND(AVG(b04_gas_mw), 2) as avg_gas,
    ROUND(AVG(total_mw), 2) as avg_total
FROM norm_entso_e_a75
WHERE country = 'NL' AND timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY DATE(timestamp), quality_status
ORDER BY date DESC, quality_status;

-- Last 24h: Hourly distribution
SELECT
    EXTRACT(HOUR FROM timestamp) as hour,
    COUNT(*) as record_count,
    COUNT(DISTINCT DATE(timestamp)) as distinct_days,
    COUNT(DISTINCT data_source) as distinct_sources
FROM norm_entso_e_a44
WHERE timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY EXTRACT(HOUR FROM timestamp)
ORDER BY hour;
```

---

### Query 5: Nuclear = 0 Trend (7 dagen)

**Doel**: Analyse van nuclear generation = 0 pattern

```sql
-- Daily nuclear generation stats (last 7 days)
SELECT
    DATE(timestamp) as date,
    ROUND(AVG(b14_nuclear_mw), 2) as avg_nuclear,
    ROUND(MIN(b14_nuclear_mw), 2) as min_nuclear,
    ROUND(MAX(b14_nuclear_mw), 2) as max_nuclear,
    COUNT(CASE WHEN b14_nuclear_mw = 0 THEN 1 END) as zero_count,
    COUNT(*) as total_count,
    ROUND(100.0 * COUNT(CASE WHEN b14_nuclear_mw = 0 THEN 1 END) / COUNT(*), 2) as zero_pct
FROM norm_entso_e_a75
WHERE country = 'NL'
  AND timestamp >= NOW() - INTERVAL '7 days'
GROUP BY DATE(timestamp)
ORDER BY date DESC;

-- Breakdown of zero events (last 7 days)
SELECT
    DATE(timestamp) as date,
    EXTRACT(HOUR FROM timestamp) as hour,
    COUNT(*) as zero_count,
    COUNT(DISTINCT quality_status) as distinct_quality
FROM norm_entso_e_a75
WHERE country = 'NL'
  AND b14_nuclear_mw = 0
  AND timestamp >= NOW() - INTERVAL '7 days'
GROUP BY DATE(timestamp), EXTRACT(HOUR FROM timestamp)
ORDER BY date DESC, hour;

-- Nuclear generation time series (last 7 days, 12 hourly)
SELECT
    timestamp,
    b14_nuclear_mw as nuclear_mw,
    quality_status,
    CASE WHEN b14_nuclear_mw = 0 THEN 'ZERO' ELSE 'NORMAL' END as status
FROM norm_entso_e_a75
WHERE country = 'NL'
  AND timestamp >= NOW() - INTERVAL '7 days'
ORDER BY timestamp DESC
LIMIT 100;
```

---

### Query 6: Meest Recente Record Details

**Doel**: Laatste data point analyse per tabel

```sql
-- Most recent A44 record (with full context)
SELECT
    *,
    NOW() as query_time,
    EXTRACT(EPOCH FROM (NOW() - timestamp)) / 3600 as hours_ago
FROM norm_entso_e_a44
ORDER BY timestamp DESC
LIMIT 1;

-- Most recent A75 record (with full context)
SELECT
    timestamp,
    country,
    b01_biomass_mw,
    b04_gas_mw,
    b05_coal_mw,
    b14_nuclear_mw,
    b16_solar_mw,
    b17_waste_mw,
    b18_wind_offshore_mw,
    b19_wind_onshore_mw,
    b20_other_mw,
    total_mw,
    quality_status,
    last_updated,
    NOW() as query_time,
    EXTRACT(EPOCH FROM (NOW() - timestamp)) / 3600 as hours_ago
FROM norm_entso_e_a75
WHERE country = 'NL'
ORDER BY timestamp DESC
LIMIT 1;

-- Most recent A65 record (with full context)
SELECT
    timestamp,
    country,
    actual_mw,
    forecast_mw,
    quality_status,
    last_updated,
    NOW() as query_time,
    EXTRACT(EPOCH FROM (NOW() - timestamp)) / 3600 as hours_ago
FROM norm_entso_e_a65
WHERE country = 'NL'
ORDER BY timestamp DESC
LIMIT 1;

-- Recent fetch log summary (last 10 entries)
SELECT
    source,
    fetch_time,
    status,
    records_fetched,
    error_message,
    EXTRACT(EPOCH FROM (NOW() - fetch_time)) / 3600 as hours_ago
FROM fetch_log
ORDER BY fetch_time DESC
LIMIT 10;
```

---

## Execution Instructions

### 1. Connect to Database

```bash
ssh root@135.181.255.83
cd /opt/github/ha-energy-insights-nl
source /opt/energy-insights-nl/venv/bin/activate
psql -U energy_insights_nl -d energy_insights_nl -h localhost
```

### 2. Run Individual Queries

Copy each query block and execute in psql:
- Q1: Quality Status Distribution
- Q2: Data Source Distribution
- Q3: Backfill Status
- Q4: Last 24h Breakdown
- Q5: Nuclear = 0 Trend
- Q6: Most Recent Records

### 3. Export Results

```bash
# Run all queries and export to JSON/CSV
psql -U energy_insights_nl -d energy_insights_nl -h localhost << 'EOF'
\pset format json
-- Run all queries above
EOF
```

---

## Expected Outputs

### Q1 - Quality Status
- FRESH/STALE ratio per table
- Age of oldest/newest records
- Percentage distribution

### Q2 - Data Source
- ENTSO-E vs Energy-Charts vs Cache breakdown
- Price statistics per source
- Data recency

### Q3 - Backfill
- Count of records needing backfill
- Percentage by source
- Date range of gaps

### Q4 - Last 24h
- Hourly distribution of records
- Average/min/max prices
- Generation mix changes

### Q5 - Nuclear Zeros
- Days with nuclear = 0
- Percentage of hours with zero production
- Pattern analysis (if weekend/maintenance)

### Q6 - Latest Records
- Most recent timestamp per table
- Current quality status
- Age in hours
- Last 10 fetch log entries

---

## Analysis Notes

**Key Metrics to Monitor**:
1. % of STALE vs FRESH (target: >95% FRESH)
2. Data source reliability (ENTSO-E should be primary)
3. Backfill lag (target: <0 records needing backfill)
4. Nuclear zeros pattern (expected behavior?)
5. Record freshness (should be <1h old)
6. Fetch success rate (target: 100%)

---

## Related Tasks

- cc-task-fallback-analysis.md - Fallback procedure analysis
- cc-task-fallback-fix.md - Implementation details
- cc-task-auth-ratelimit.md - API auth & rate limiting
