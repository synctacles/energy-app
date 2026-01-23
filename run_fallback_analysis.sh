#!/bin/bash

# Fallback & Data Quality Analyse Script
# Executes all queries on remote PostgreSQL server
# Brand-free version - uses environment variables

# Load environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
fi

# Defaults
SSH_HOST="${SSH_HOST:-135.181.255.83}"
SSH_USER="${SSH_USER:-root}"
DB_HOST="${DB_HOST:-localhost}"
DB_NAME="${DB_NAME:-synctacles}"
DB_USER="${DB_USER:-synctacles}"
BRAND_SLUG="${BRAND_SLUG:-synctacles}"

# Output files
REPORT_DIR="${REPORT_DIR:-/opt/${BRAND_SLUG}/reports}"
REPORT_FILE="$REPORT_DIR/fallback-report-$(date +%Y%m%d).md"
RESULTS_FILE="/tmp/fallback_results_$(date +%Y%m%d_%H%M%S).txt"

echo "=========================================="
echo "FALLBACK & DATA QUALITY ANALYSE"
echo "Start: $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="
echo ""

# Create reports directory
mkdir -p "$REPORT_DIR"

# Function to run query on remote server
run_query() {
    local query_name=$1
    local query=$2

    echo "📊 Query: $query_name"

    # SSH naar server en voer psql query uit
    ssh -l "$SSH_USER" "$SSH_HOST" << EOF
psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -P pager=off << 'QUERY'
$query
QUERY
EOF

    echo "---"
}

# ====== STAP 1: Schema Analyse ======
echo "📋 STAP 1: Schema Analyse" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Schema norm_generation" "
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'norm_generation'
ORDER BY ordinal_position;" | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"

run_query "Schema raw_entso_e_a75" "
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'raw_entso_e_a75'
ORDER BY ordinal_position;" | tee -a "$RESULTS_FILE"

# ====== STAP 2: Algemene Statistieken ======
echo "📋 STAP 2: Algemene Statistieken" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Totaal records en tijdspan" "
SELECT
    COUNT(*) as total_records,
    MIN(timestamp) as eerste_record,
    MAX(timestamp) as laatste_record,
    MAX(timestamp) - MIN(timestamp) as tijdspan
FROM norm_generation;" | tee -a "$RESULTS_FILE"

# ====== STAP 3: Data Source Verdeling ======
echo "📋 STAP 3: Data Source Verdeling" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Data source verdeling" "
SELECT
    data_source,
    COUNT(*) as records,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage,
    MIN(timestamp) as eerste,
    MAX(timestamp) as laatste
FROM norm_generation
GROUP BY data_source
ORDER BY records DESC;" | tee -a "$RESULTS_FILE"

# ====== STAP 4: Data Quality Verdeling ======
echo "📋 STAP 4: Data Quality Verdeling" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Data quality verdeling" "
SELECT
    data_quality,
    COUNT(*) as records,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM norm_generation
GROUP BY data_quality
ORDER BY records DESC;" | tee -a "$RESULTS_FILE"

# ====== STAP 5: Nul-Waarden Analyse ======
echo "📋 STAP 5: Nul-Waarden Analyse" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Nul-waarden per type" "
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
FROM norm_generation WHERE biomass_mw = 0;" | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"

run_query "Verdachte nul-waarden (nuclear/gas=0)" "
SELECT
    timestamp,
    nuclear_mw,
    gas_mw,
    data_source,
    data_quality
FROM norm_generation
WHERE nuclear_mw = 0 OR gas_mw = 0
ORDER BY timestamp DESC
LIMIT 20;" | tee -a "$RESULTS_FILE"

# ====== STAP 6: Fallback Events ======
echo "📋 STAP 6: Fallback Events" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Recente fallback events" "
SELECT
    timestamp,
    data_source,
    data_quality,
    total_mw,
    renewable_percentage
FROM norm_generation
WHERE data_source != 'ENTSO-E' OR data_quality IN ('FALLBACK', 'CACHED', 'FORWARD_FILL')
ORDER BY timestamp DESC
LIMIT 50;" | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"

run_query "Fallback events per dag" "
SELECT
    DATE(timestamp) as dag,
    data_source,
    data_quality,
    COUNT(*) as events
FROM norm_generation
WHERE data_source != 'ENTSO-E' OR data_quality NOT IN ('FRESH', 'STALE')
GROUP BY DATE(timestamp), data_source, data_quality
ORDER BY dag DESC;" | tee -a "$RESULTS_FILE"

# ====== STAP 7: Backfill Status ======
echo "📋 STAP 7: Backfill Status" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Backfill nodig count" "
SELECT
    COUNT(*) as needs_backfill_count
FROM norm_generation
WHERE needs_backfill = true;" | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"

run_query "Backfill nodig details" "
SELECT
    timestamp,
    data_source,
    data_quality,
    needs_backfill
FROM norm_generation
WHERE needs_backfill = true
ORDER BY timestamp DESC
LIMIT 20;" | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"

run_query "Reeds ge-backfilled count" "
SELECT
    COUNT(*) as backfilled_count
FROM norm_generation
WHERE data_quality = 'BACKFILLED';" | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"

run_query "Backfilled details" "
SELECT
    timestamp,
    data_source,
    data_quality,
    updated_at
FROM norm_generation
WHERE data_quality = 'BACKFILLED'
ORDER BY updated_at DESC
LIMIT 20;" | tee -a "$RESULTS_FILE"

# ====== STAP 8: Tijdlijn Analyse ======
echo "📋 STAP 8: Tijdlijn Analyse (24h)" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "Laatste 24 uur per uur" "
SELECT
    DATE_TRUNC('hour', timestamp) as uur,
    data_source,
    data_quality,
    COUNT(*) as records,
    ROUND(AVG(total_mw), 0) as avg_mw
FROM norm_generation
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', timestamp), data_source, data_quality
ORDER BY uur DESC;" | tee -a "$RESULTS_FILE"

# ====== STAP 9: Load & Balance Tables ======
echo "📋 STAP 9: Load & Balance Tables" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

run_query "norm_load statistieken" "
SELECT
    data_source,
    data_quality,
    COUNT(*) as records
FROM norm_load
GROUP BY data_source, data_quality;" | tee -a "$RESULTS_FILE"

echo "" | tee -a "$RESULTS_FILE"

run_query "norm_grid_balance statistieken" "
SELECT
    data_source,
    data_quality,
    COUNT(*) as records
FROM norm_grid_balance
GROUP BY data_source, data_quality;" | tee -a "$RESULTS_FILE"

echo ""
echo "=========================================="
echo "✅ Alle queries voltooid"
echo "Resultaten opgeslagen in: $RESULTS_FILE"
echo "=========================================="
