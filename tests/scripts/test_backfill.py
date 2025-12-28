#!/usr/bin/env python3
"""
ENTSO-E Backfill Test: Probeer nullen van 21-22 dec te reconstrueren
"""
from entsoe import EntsoePandasClient
import pandas as pd
from datetime import datetime, timezone
import os

# API key uit .env
API_KEY = os.getenv('ENTSOE_API_KEY', 'c3eca61e-37a9-4727-bf60-83e213b22a9eRE')
client = EntsoePandasClient(api_key=API_KEY)

# Test timestamps van jouw audit (biomass nullen)
test_cases = [
    '2025-12-22 00:00:00+00:00',
    '2025-12-21 17:00:00+00:00',
    '2025-12-21 14:00:00+00:00',
    '2025-12-21 11:00:00+00:00',
    '2025-12-21 10:00:00+00:00',
]

print("=" * 80)
print("ENTSO-E BACKFILL TEST")
print("=" * 80)

for ts_str in test_cases:
    ts = pd.Timestamp(ts_str)
    start = ts
    end = ts + pd.Timedelta(hours=1)
    
    print(f"\n[{ts}]")
    
    try:
        # Query generation data
        gen = client.query_generation(
            country_code='NL',
            start=start,
            end=end,
            psr_type=None
        )
        
        # Check Biomass waarde
        if 'Biomass' in gen.columns:
            biomass_value = gen['Biomass'].iloc[0] if len(gen) > 0 else 0
            print(f"  Biomass: {biomass_value:.1f} MW")
            
            if biomass_value > 0:
                print(f"  ✅ RECONSTRUCTABLE (was 0 in database)")
            else:
                print(f"  ❌ STILL ZERO (permanent gap)")
        else:
            print(f"  ❌ No Biomass data in response")
            
        # Toon alle beschikbare types
        if len(gen.columns) > 0:
            print(f"  Available: {', '.join(gen.columns)}")
        
    except Exception as e:
        print(f"  ❌ API Error: {str(e)[:100]}")

print("\n" + "=" * 80)
print("CONCLUSIE")
print("=" * 80)
print("Als Biomass > 0 bij timestamps waar database 0 heeft:")
print("  → Backfill script zou gaps hebben gefixt")
print("Als Biomass = 0 ook in API:")
print("  → Permanent gap, ENTSO-E heeft nooit data gepubliceerd")