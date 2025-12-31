#!/usr/bin/env python3
"""
Fallback & Data Quality Analyse Script
Verbindt met PostgreSQL en voert alle queries uit
"""

import psycopg2
import json
from datetime import datetime
from pathlib import Path
import subprocess
import sys

# Configuratie
SSH_HOST = "135.181.255.83"
SSH_USER = "root"
DB_HOST = "localhost"
DB_NAME = "energy_insights_nl"
DB_USER = "energy_insights_nl"
DB_PORT = 5432

# SSH tunnel cmd voor remote exec
def run_psql_query(query):
    """Voer SQL query uit via SSH op remote server"""
    try:
        cmd = f"""ssh -l {SSH_USER} {SSH_HOST} << 'EOF'
psql -h {DB_HOST} -U {DB_USER} -d {DB_NAME} -P pager=off -c "{query.replace('"', '\\"')}"
EOF"""

        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=30)

        if result.returncode != 0:
            print(f"❌ Query fout: {result.stderr}")
            return None
        return result.stdout
    except Exception as e:
        print(f"❌ SSH/Query fout: {e}")
        return None

# Alle queries
queries = {
    "1_schema_norm_generation": """
        SELECT column_name, data_type
        FROM information_schema.columns
        WHERE table_name = 'norm_generation'
        ORDER BY ordinal_position;
    """,

    "1_schema_raw_entso_e": """
        SELECT column_name, data_type
        FROM information_schema.columns
        WHERE table_name = 'raw_entso_e_a75'
        ORDER BY ordinal_position;
    """,

    "2_general_stats": """
        SELECT
            COUNT(*) as total_records,
            MIN(timestamp) as eerste_record,
            MAX(timestamp) as laatste_record,
            MAX(timestamp) - MIN(timestamp) as tijdspan
        FROM norm_generation;
    """,

    "3_data_source_verdeling": """
        SELECT
            data_source,
            COUNT(*) as records,
            ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage,
            MIN(timestamp) as eerste,
            MAX(timestamp) as laatste
        FROM norm_generation
        GROUP BY data_source
        ORDER BY records DESC;
    """,

    "4_data_quality_verdeling": """
        SELECT
            data_quality,
            COUNT(*) as records,
            ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
        FROM norm_generation
        GROUP BY data_quality
        ORDER BY records DESC;
    """,

    "5_nul_waarden_analyse": """
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
    """,

    "5_verdachte_nul_waarden": """
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
    """,

    "6_fallback_events_recent": """
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
    """,

    "6_fallback_events_per_dag": """
        SELECT
            DATE(timestamp) as dag,
            data_source,
            data_quality,
            COUNT(*) as events
        FROM norm_generation
        WHERE data_source != 'ENTSO-E' OR data_quality NOT IN ('FRESH', 'STALE')
        GROUP BY DATE(timestamp), data_source, data_quality
        ORDER BY dag DESC;
    """,

    "7_backfill_needs_count": """
        SELECT
            COUNT(*) as needs_backfill_count
        FROM norm_generation
        WHERE needs_backfill = true;
    """,

    "7_backfill_needs_details": """
        SELECT
            timestamp,
            data_source,
            data_quality,
            needs_backfill
        FROM norm_generation
        WHERE needs_backfill = true
        ORDER BY timestamp DESC
        LIMIT 20;
    """,

    "7_backfilled_count": """
        SELECT
            COUNT(*) as backfilled_count
        FROM norm_generation
        WHERE data_quality = 'BACKFILLED';
    """,

    "7_backfilled_details": """
        SELECT
            timestamp,
            data_source,
            data_quality,
            updated_at
        FROM norm_generation
        WHERE data_quality = 'BACKFILLED'
        ORDER BY updated_at DESC
        LIMIT 20;
    """,

    "8_timeline_24h": """
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
    """,

    "9_norm_load": """
        SELECT
            data_source,
            data_quality,
            COUNT(*) as records
        FROM norm_load
        GROUP BY data_source, data_quality;
    """,

    "9_grid_balance": """
        SELECT
            data_source,
            data_quality,
            COUNT(*) as records
        FROM norm_grid_balance
        GROUP BY data_source, data_quality;
    """
}

def main():
    print("=" * 80)
    print("FALLBACK & DATA QUALITY ANALYSE")
    print(f"Start: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 80)

    results = {}

    # Voer alle queries uit
    for query_name, query in queries.items():
        print(f"\n📊 Executing: {query_name}...")
        result = run_psql_query(query)

        if result:
            results[query_name] = result
            print(f"✅ Success")
            print(result[:500])  # Toon eerste 500 chars
        else:
            print(f"❌ Failed")
            results[query_name] = None

    # Sla raw resultaten op
    with open('/tmp/fallback_analysis_raw.json', 'w') as f:
        json.dump(results, f, indent=2, default=str)

    print("\n" + "=" * 80)
    print("✅ Alle queries voltooid")
    print("=" * 80)

    return results

if __name__ == "__main__":
    results = main()
