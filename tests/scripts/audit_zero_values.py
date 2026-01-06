#!/usr/bin/env python3
"""
ENTSO-E Zero-Value Audit (met nachttijd filtering)
Detecteert abnormale 0-waarden, exclusief verwachte nullen (Solar 's nachts)
"""
import sys
import os
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import create_engine
from datetime import datetime
import pandas as pd

# Use environment variable for database URL
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://localhost:5432/synctacles")
engine = create_engine(DATABASE_URL)

PSR_COLUMNS = [
    'b01_biomass_mw', 'b04_gas_mw', 'b05_coal_mw', 'b14_nuclear_mw',
    'b16_solar_mw', 'b17_waste_mw', 'b18_wind_offshore_mw',
    'b19_wind_onshore_mw', 'b20_other_mw'
]

# Filtering regels per PSR-type
FILTERS = {
    'b16_solar_mw': lambda df: (df['timestamp'].dt.hour >= 8) & (df['timestamp'].dt.hour < 20),  # Alleen dag
    'b14_nuclear_mw': lambda df: df['timestamp'] == df['timestamp'],  # Altijd abnormaal
    'b04_gas_mw': lambda df: df['timestamp'] == df['timestamp'],      # Altijd abnormaal
}

print("=" * 80)
print("ENTSO-E ZERO-VALUE AUDIT (ABNORMALE NULLEN)")
print("=" * 80)

# A75 Generation
print("\n[A75 GENERATION]")
query_a75 = f"""
SELECT timestamp, country, {', '.join(PSR_COLUMNS)}
FROM norm_entso_e_a75
WHERE country = 'NL'
ORDER BY timestamp DESC
"""

df_a75 = pd.read_sql(query_a75, engine)
df_a75['timestamp'] = pd.to_datetime(df_a75['timestamp'])
print(f"Totaal records: {len(df_a75)}")

for col in PSR_COLUMNS:
    zeros = df_a75[df_a75[col] == 0.0].copy()
    
    # Pas filter toe indien gedefinieerd
    if col in FILTERS:
        zeros = zeros[FILTERS[col](zeros)]
    
    if len(zeros) > 0:
        psr_name = col.split('_')[1].title()
        print(f"\n{psr_name} (abnormaal):")
        print(f"  Nullen: {len(zeros)} / {len(df_a75)} ({len(zeros)/len(df_a75)*100:.1f}%)")
        print(f"  Laatste 5:")
        for _, row in zeros.head(5).iterrows():
            hour = row['timestamp'].hour
            print(f"    {row['timestamp']} (uur {hour:02d}) | {row[col]} MW")

# A65 Load
print("\n[A65 LOAD]")
query_a65 = """
SELECT timestamp, country, actual_mw, forecast_mw
FROM norm_entso_e_a65
WHERE country = 'NL' AND timestamp <= NOW()
ORDER BY timestamp DESC
"""

df_a65 = pd.read_sql(query_a65, engine)
df_a65['timestamp'] = pd.to_datetime(df_a65['timestamp'])
print(f"Totaal records: {len(df_a65)}")

for col in ['actual_mw', 'forecast_mw']:
    zeros = df_a65[df_a65[col] == 0.0]
    nulls = df_a65[df_a65[col].isna()]
    print(f"\n{col.replace('_', ' ').title()}:")
    print(f"  Nullen: {len(zeros)} ({len(zeros)/len(df_a65)*100:.1f}%)")
    print(f"  NULL: {len(nulls)} ({len(nulls)/len(df_a65)*100:.1f}%)")
    if len(zeros) > 0:
        print(f"  Laatste 5:")
        for _, row in zeros.head(5).iterrows():
            print(f"    {row['timestamp']} | {row[col]} MW")

# SAMENVATTING
print("\n" + "=" * 80)
print("SAMENVATTING")
print("=" * 80)
total_abnormal = sum(len(df_a75[(df_a75[col] == 0.0) & (FILTERS.get(col, lambda x: x==x)(df_a75))]) for col in PSR_COLUMNS)
print(f"Totaal abnormale nullen A75: {total_abnormal}")
print(f"Totaal load nullen: {len(df_a65[df_a65['actual_mw'] == 0.0])}")