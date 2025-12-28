#!/usr/bin/env python3
"""
ENTSO-E Backfill Test: Direct API calls (bypass entsoe-py bug)
"""
import requests
import xml.etree.ElementTree as ET
from datetime import datetime, timedelta
import os

API_KEY = os.getenv('ENTSOE_API_KEY')
BASE_URL = "https://web-api.tp.entsoe.eu/api"

# ENTSO-E document type codes
DOC_TYPE = "A75"  # Actual generation per type
PROCESS_TYPE = "A16"  # Realised

# PSR type code mapping
PSR_TYPES = {
    'B01': 'Biomass',
    'B04': 'Fossil Gas',
    'B05': 'Fossil Hard Coal',
    'B14': 'Nuclear',
    'B16': 'Solar',
    'B17': 'Waste',
    'B18': 'Wind Offshore',
    'B19': 'Wind Onshore',
    'B20': 'Other'
}

def fetch_generation(timestamp_str, psr_code='B01'):
    """Fetch generation for specific hour + PSR type"""
    ts = datetime.fromisoformat(timestamp_str.replace('+00:00', ''))
    
    # ENTSO-E tijdformat: YYYYMMDDHHmm
    period_start = ts.strftime('%Y%m%d%H%M')
    period_end = (ts + timedelta(hours=1)).strftime('%Y%m%d%H%M')
    
    params = {
        'securityToken': API_KEY,
        'documentType': DOC_TYPE,
        'processType': PROCESS_TYPE,
        'psrType': psr_code,
        'in_Domain': '10YNL----------L',  # Netherlands
        'periodStart': period_start,
        'periodEnd': period_end
    }
    
    response = requests.get(BASE_URL, params=params, timeout=30)
    
    if response.status_code != 200:
        return None, f"HTTP {response.status_code}"
    
    try:
        root = ET.fromstring(response.content)
        # Parse XML voor quantity waarde
        ns = {'ns': 'urn:iec62325.351:tc57wg16:451-6:generationloaddocument:3:0'}
        points = root.findall('.//ns:Point', ns)
        
        if points:
            quantity = float(points[0].find('ns:quantity', ns).text)
            return quantity, None
        else:
            return 0.0, "No data points"
            
    except Exception as e:
        return None, str(e)[:50]

# Test cases
test_cases = [
    '2025-12-22 00:00:00+00:00',
    '2025-12-21 17:00:00+00:00',
    '2025-12-21 14:00:00+00:00',
    '2025-12-21 11:00:00+00:00',
    '2025-12-21 10:00:00+00:00',
]

print("=" * 80)
print("ENTSO-E BACKFILL TEST (Raw API)")
print("=" * 80)

for ts in test_cases:
    print(f"\n[{ts}]")
    value, error = fetch_generation(ts, 'B01')  # Biomass
    
    if error:
        print(f"  ❌ Error: {error}")
    elif value > 0:
        print(f"  ✅ Biomass: {value:.1f} MW (RECONSTRUCTABLE)")
    else:
        print(f"  ❌ Biomass: 0.0 MW (permanent gap)")

print("\n" + "=" * 80)