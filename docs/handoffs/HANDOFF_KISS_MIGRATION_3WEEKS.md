# HANDOFF: KISS STACK IMPLEMENTATIE + DECOMMISSION COEFFICIENT SERVER
**Date:** 2026-01-17
**From:** Claude (CAI)
**To:** Claude Code (CC)
**Decision:** GO - Decommission coefficient server

---

## STATUS UPDATE: 2026-01-17

### ✅ Week 1: COMPLETED
All 6 backend tasks implemented:
- **#74** EasyEnergy Client → `synctacles_db/clients/easyenergy_client.py`
- **#75** Static Offset Configuration → `synctacles_db/config/static_offsets.py`
- **#76** Fallback Manager Refactor → 7-tier → 6-tier, no coefficient calls
- **#77** Reference Data → `_add_reference_data()` in fallback manager
- **#78** Unit Tests → `tests/test_kiss_stack.py`
- **#79** Code Cleanup → ConsumerPriceClient marked deprecated

### 🔜 Week 2: HA Component (issues 80-82)
### 📅 Week 3: Decommission (issues 83-88)

---

## MISSION

Implementeer KISS stack voor NL in 3 weken:
1. **Week 1:** Backend aanpassingen (EasyEnergy, fallback split, reference data) ✅
2. **Week 2:** HA component updates (anomalie detectie)
3. **Week 3:** Decommission coefficient server (backup + shutdown)

**Expected outcome:** 100% accuracy, €150/jaar bespaard, minder complexity

---

# WEEK 1: BACKEND AANPASSINGEN ✅ COMPLETED

**Server:** 135.181.255.83 (enin-nl)  
**Repository:** `/opt/github/synctacles-api/`  
**Time:** 20-30 uur

---

## 1.1 CREATE EASYENERGY CLIENT

### Action 1.1.1: Implementeer EasyEnergyClient

```bash
ssh root@135.181.255.83
cd /opt/github/synctacles-api/

# Create EasyEnergy client
cat > synctacles_db/clients/easyenergy_client.py << 'EOF'
"""
EasyEnergy API Client
Official API: https://mijn.easyenergy.com/nl/api/tariff/getapxtariffs
"""

import requests
from datetime import datetime, timedelta
from typing import List, Dict, Optional
import logging

logger = logging.getLogger(__name__)

class EasyEnergyClient:
    """Client for EasyEnergy price API"""
    
    BASE_URL = "https://mijn.easyenergy.com/nl/api/tariff"
    
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Synctacles/1.0',
            'Accept': 'application/json'
        })
    
    def get_prices_today(self) -> List[Dict]:
        """
        Get today's prices (00:00 - 23:00)
        
        Returns:
            List[Dict]: [
                {
                    'timestamp': '2026-01-17T00:00:00+01:00',
                    'price_eur_kwh': 0.125
                },
                ...
            ]
        """
        today = datetime.now().date()
        return self.get_prices_range(today, today)
    
    def get_prices_tomorrow(self) -> List[Dict]:
        """Get tomorrow's prices (available after 15:00 CET)"""
        tomorrow = datetime.now().date() + timedelta(days=1)
        return self.get_prices_range(tomorrow, tomorrow)
    
    def get_prices_range(self, start_date: datetime.date, 
                        end_date: datetime.date) -> List[Dict]:
        """
        Get prices for date range
        
        Args:
            start_date: Start date (inclusive)
            end_date: End date (inclusive)
            
        Returns:
            List of price records
        """
        try:
            # API endpoint
            url = f"{self.BASE_URL}/getapxtariffs"
            params = {
                'startTimestamp': start_date.strftime('%Y-%m-%d'),
                'endTimestamp': (end_date + timedelta(days=1)).strftime('%Y-%m-%d')
            }
            
            logger.info(f"Fetching EasyEnergy prices: {start_date} to {end_date}")
            response = self.session.get(url, params=params, timeout=10)
            response.raise_for_status()
            
            data = response.json()
            
            # Transform to our format
            prices = []
            for record in data:
                prices.append({
                    'timestamp': record['Timestamp'],
                    'price_eur_kwh': record['TariffReturn'] / 1000,  # Convert to EUR/kWh
                    'price_eur_mwh': record['TariffReturn']
                })
            
            logger.info(f"Retrieved {len(prices)} EasyEnergy prices")
            return prices
            
        except requests.RequestException as e:
            logger.error(f"EasyEnergy API error: {e}")
            return []
        except (KeyError, ValueError) as e:
            logger.error(f"EasyEnergy response parsing error: {e}")
            return []
    
    def health_check(self) -> bool:
        """Check if API is responsive"""
        try:
            today = datetime.now().date()
            prices = self.get_prices_range(today, today)
            return len(prices) > 0
        except Exception:
            return False

# Module-level instance
easy_client = EasyEnergyClient()
EOF

chmod 644 synctacles_db/clients/easyenergy_client.py
```

---

### Action 1.1.2: Test EasyEnergy Client

```bash
cd /opt/github/synctacles-api/

# Test client
python3 -c "
import sys
sys.path.insert(0, '.')
from synctacles_db.clients.easyenergy_client import easy_client
from datetime import datetime

# Test today
prices = easy_client.get_prices_today()
print(f'Today prices: {len(prices)} records')
if prices:
    print(f'First price: {prices[0]}')
    print(f'Last price: {prices[-1]}')

# Test health
health = easy_client.health_check()
print(f'Health check: {health}')
"
```

**Expected output:**
```
Today prices: 24 records
First price: {'timestamp': '2026-01-17T00:00:00+01:00', 'price_eur_kwh': 0.125, ...}
Last price: {'timestamp': '2026-01-17T23:00:00+01:00', 'price_eur_kwh': 0.148, ...}
Health check: True
```

---

## 1.2 STATIC OFFSET TABLE

### Action 1.2.1: Create Offset Configuration

```bash
cd /opt/github/synctacles-api/

# Create static offset module
cat > synctacles_db/config/static_offsets.py << 'EOF'
"""
Static hourly offsets for wholesale → consumer price conversion
Based on 27,895 hours ANWB data (2022-2026)

Usage: consumer_price = wholesale_price + HOURLY_OFFSET[hour]
Accuracy: 85-89% for ranking
"""

# EUR/kWh offset per hour (0-23)
HOURLY_OFFSET = {
    0: 0.1934,   # Night low
    1: 0.1903,
    2: 0.1879,
    3: 0.1819,
    4: 0.1705,
    5: 0.1667,   # Lowest offset
    6: 0.1789,
    7: 0.1989,   # Morning rise
    8: 0.2132,   # Morning peak
    9: 0.2099,
    10: 0.2030,
    11: 0.1968,
    12: 0.1899,  # Afternoon drop
    13: 0.1768,
    14: 0.1669,
    15: 0.1599,
    16: 0.1508,  # Lowest afternoon
    17: 0.1571,
    18: 0.1723,  # Evening rise
    19: 0.2009,
    20: 0.2085,  # Evening peak
    21: 0.2050,
    22: 0.2006,
    23: 0.1945
}

def apply_static_offset(wholesale_price_eur_kwh: float, hour: int) -> float:
    """
    Apply static hourly offset to wholesale price
    
    Args:
        wholesale_price_eur_kwh: Wholesale price in EUR/kWh
        hour: Hour of day (0-23)
        
    Returns:
        Estimated consumer price in EUR/kWh
    """
    if hour not in range(24):
        raise ValueError(f"Invalid hour: {hour}. Must be 0-23.")
    
    offset = HOURLY_OFFSET[hour]
    return wholesale_price_eur_kwh + offset

def get_market_stats(wholesale_prices: list) -> dict:
    """
    Calculate market statistics for reference data
    
    Args:
        wholesale_prices: List of wholesale prices (EUR/kWh)
        
    Returns:
        {
            'average': float,
            'spread': float,
            'min': float,
            'max': float
        }
    """
    if not wholesale_prices:
        return None
    
    return {
        'average': sum(wholesale_prices) / len(wholesale_prices),
        'spread': max(wholesale_prices) - min(wholesale_prices),
        'min': min(wholesale_prices),
        'max': max(wholesale_prices)
    }
EOF

chmod 644 synctacles_db/config/static_offsets.py
```

---

### Action 1.2.2: Test Static Offset

```bash
cd /opt/github/synctacles-api/

python3 -c "
import sys
sys.path.insert(0, '.')
from synctacles_db.config.static_offsets import apply_static_offset, get_market_stats

# Test offset application
wholesale = 0.05  # EUR/kWh
for hour in [0, 8, 16, 20]:
    consumer = apply_static_offset(wholesale, hour)
    print(f'Hour {hour:02d}: €{wholesale:.3f} + offset = €{consumer:.3f}')

# Test market stats
prices = [0.04, 0.05, 0.06, 0.08, 0.10]
stats = get_market_stats(prices)
print(f'Market stats: {stats}')
"
```

**Expected output:**
```
Hour 00: €0.050 + offset = €0.243
Hour 08: €0.050 + offset = €0.263
Hour 16: €0.050 + offset = €0.201
Hour 20: €0.050 + offset = €0.259
Market stats: {'average': 0.066, 'spread': 0.06, 'min': 0.04, 'max': 0.10}
```

---

## 1.3 UPDATE FALLBACK MANAGER

### Action 1.3.1: Refactor Fallback Tiers

```bash
cd /opt/github/synctacles-api/

# Backup current fallback manager
cp synctacles_db/fallback/fallback_manager.py synctacles_db/fallback/fallback_manager.py.backup

# Find current tier structure
grep -n "Tier" synctacles_db/fallback/fallback_manager.py | head -20

# Update fallback manager (manual edit required)
nano synctacles_db/fallback/fallback_manager.py
```

**Changes to make:**

```python
# OLD TIERS (7):
# Tier 1: Frank DB
# Tier 2: Enever-Frank DB (VIA COEFFICIENT SERVER - REMOVE)
# Tier 3: Frank Direct API
# Tier 4: ENTSO-E + Model (VIA COEFFICIENT SERVER - CHANGE)
# Tier 5: ENTSO-E Stale + Model (VIA COEFFICIENT SERVER - CHANGE)
# Tier 6: Energy-Charts + Model (VIA COEFFICIENT SERVER - CHANGE)
# Tier 7: Cache

# NEW TIERS (6):
# Tier 1: Frank DB (local cached)
# Tier 2: Frank Direct API (GraphQL)
# Tier 3: EasyEnergy Direct API (for Easy customers OR fallback for signals)
# Tier 4: ENTSO-E + Static Offset
# Tier 5: Energy-Charts + Static Offset
# Tier 6: Cache

# Add imports at top:
from synctacles_db.clients.easyenergy_client import easy_client
from synctacles_db.config.static_offsets import apply_static_offset, get_market_stats

# REMOVE old Tier 2 (Enever-Frank DB):
# def _try_tier2_enever_frank_db(self, ...):
#     # DELETE THIS ENTIRE METHOD

# UPDATE Tier 4-6 to use static offset:
# OLD:
# def _apply_price_model(self, wholesale_price, timestamp):
#     # Call coefficient server for slope/intercept
#     ...

# NEW:
def _apply_static_offset(self, wholesale_price_eur_kwh, timestamp):
    """Apply static hourly offset to wholesale price"""
    hour = timestamp.hour
    return apply_static_offset(wholesale_price_eur_kwh, hour)

# ADD new Tier 3 (EasyEnergy):
def _try_tier3_easyenergy_api(self, timestamp, provider):
    """
    Tier 3: EasyEnergy Direct API
    Use for EasyEnergy customers OR as fallback for signal ranking
    """
    try:
        logger.info(f"Tier 3: Trying EasyEnergy Direct API for {timestamp}")
        
        # Get prices for the date
        date = timestamp.date()
        prices = easy_client.get_prices_range(date, date)
        
        if not prices:
            logger.warning("EasyEnergy API returned no data")
            return None
        
        # Find matching timestamp
        for price_record in prices:
            if price_record['timestamp'] == timestamp.isoformat():
                logger.info(f"Tier 3 SUCCESS: EasyEnergy price found")
                return {
                    'price_eur_kwh': price_record['price_eur_kwh'],
                    'source': 'EasyEnergy Direct API',
                    'tier': 3,
                    'quality': 'FRESH',
                    'allow_go': True
                }
        
        logger.warning(f"EasyEnergy: No price for {timestamp}")
        return None
        
    except Exception as e:
        logger.error(f"Tier 3 failed: {e}")
        return None

# UPDATE main fallback logic:
def get_price(self, timestamp, provider, allow_fallback=True):
    """Get price with fallback chain"""
    
    # Tier 1: Frank DB
    result = self._try_tier1_frank_db(timestamp)
    if result:
        return self._add_reference_data(result, timestamp)
    
    # Tier 2: Frank Direct API
    result = self._try_tier2_frank_direct_api(timestamp)
    if result:
        return self._add_reference_data(result, timestamp)
    
    # Tier 3: EasyEnergy Direct API
    if allow_fallback:
        result = self._try_tier3_easyenergy_api(timestamp, provider)
        if result:
            return self._add_reference_data(result, timestamp)
    
    # Tier 4: ENTSO-E + Static Offset
    if allow_fallback:
        result = self._try_tier4_entso_e_static(timestamp)
        if result:
            return self._add_reference_data(result, timestamp)
    
    # Tier 5: Energy-Charts + Static Offset
    if allow_fallback:
        result = self._try_tier5_energy_charts_static(timestamp)
        if result:
            return self._add_reference_data(result, timestamp)
    
    # Tier 6: Cache
    result = self._try_tier6_cache(timestamp)
    if result:
        return self._add_reference_data(result, timestamp)
    
    # All tiers failed
    logger.error(f"ALL TIERS FAILED for {timestamp}")
    return None
```

---

### Action 1.3.2: Add Reference Data to Response

```python
# Add new method to fallback_manager.py:

def _add_reference_data(self, result: dict, timestamp: datetime) -> dict:
    """
    Add reference data for HA client anomalie detectie
    
    Args:
        result: Price result from tier
        timestamp: Requested timestamp
        
    Returns:
        Enhanced result with reference data
    """
    try:
        # Get Frank live price for reference (if available)
        frank_live = self._get_frank_live_price(timestamp)
        
        # Get market stats from ENTSO-E
        market_stats = self._get_market_stats(timestamp.date())
        
        # Calculate expected range (±15% from market average)
        if market_stats:
            avg_consumer = apply_static_offset(market_stats['average'], timestamp.hour)
            expected_low = avg_consumer * 0.85
            expected_high = avg_consumer * 1.15
        else:
            # Fallback to typical range
            expected_low = 0.15
            expected_high = 0.35
        
        # Add reference to result
        result['reference'] = {
            'frank_live': frank_live,
            'market': market_stats,
            'expected_range': {
                'low': round(expected_low, 3),
                'high': round(expected_high, 3)
            }
        }
        
        return result
        
    except Exception as e:
        logger.warning(f"Could not add reference data: {e}")
        # Return result without reference (graceful degradation)
        return result

def _get_frank_live_price(self, timestamp: datetime) -> dict:
    """Get Frank live price for reference"""
    try:
        # Try Frank DB first
        from synctacles_db.database import get_db_connection
        conn = get_db_connection()
        cursor = conn.cursor()
        
        cursor.execute("""
            SELECT price_eur_kwh 
            FROM frank_prices 
            WHERE timestamp = %s
        """, (timestamp,))
        
        row = cursor.fetchone()
        if row:
            return {
                'timestamp': timestamp.isoformat(),
                'price_eur_kwh': float(row[0])
            }
        
        return None
        
    except Exception as e:
        logger.debug(f"Could not get Frank live price: {e}")
        return None

def _get_market_stats(self, date: datetime.date) -> dict:
    """Get wholesale market statistics for the day"""
    try:
        from synctacles_db.database import get_db_connection
        conn = get_db_connection()
        cursor = conn.cursor()
        
        # Get ENTSO-E prices for the day
        cursor.execute("""
            SELECT price_eur_mwh 
            FROM norm_entso_e_a44
            WHERE DATE(timestamp) = %s
            ORDER BY timestamp
        """, (date,))
        
        rows = cursor.fetchall()
        if not rows:
            return None
        
        prices_kwh = [float(row[0]) / 1000 for row in rows]  # Convert MWh to kWh
        
        return get_market_stats(prices_kwh)
        
    except Exception as e:
        logger.debug(f"Could not get market stats: {e}")
        return None
```

---

## 1.4 UPDATE API ENDPOINTS

### Action 1.4.1: Enhance Price Endpoint Response

```bash
cd /opt/github/synctacles-api/

# Find price endpoint
grep -rn "def get.*price" api/endpoints/ | grep -v __pycache__

# Update endpoint to include reference data
nano api/endpoints/prices.py
```

**Changes:**

```python
# In price endpoint (example):
@app.get("/api/v1/prices/{provider}")
async def get_current_price(
    provider: str,
    timestamp: Optional[str] = None
):
    """
    Get current price for provider
    
    Response now includes reference data for HA anomalie detectie
    """
    # ... existing code ...
    
    # Get price with fallback
    result = fallback_manager.get_price(ts, provider)
    
    if not result:
        raise HTTPException(status_code=503, detail="No price data available")
    
    # Response now includes reference data automatically
    return result
    # Example response:
    # {
    #   "price_eur_kwh": 0.247,
    #   "source": "Frank DB",
    #   "tier": 1,
    #   "quality": "FRESH",
    #   "allow_go": true,
    #   "reference": {
    #     "frank_live": {...},
    #     "market": {...},
    #     "expected_range": {"low": 0.21, "high": 0.28}
    #   }
    # }
```

---

## 1.5 TESTING

### Action 1.5.1: Unit Tests

```bash
cd /opt/github/synctacles-api/

# Create test file
cat > tests/test_kiss_stack.py << 'EOF'
"""
Unit tests for KISS stack implementation
"""

import unittest
from datetime import datetime
import sys
sys.path.insert(0, '/opt/github/synctacles-api')

from synctacles_db.clients.easyenergy_client import easy_client
from synctacles_db.config.static_offsets import apply_static_offset, get_market_stats
from synctacles_db.fallback.fallback_manager import fallback_manager

class TestKISSStack(unittest.TestCase):
    
    def test_easyenergy_client(self):
        """Test EasyEnergy client can fetch today's prices"""
        prices = easy_client.get_prices_today()
        self.assertGreater(len(prices), 0, "Should get today's prices")
        self.assertIn('timestamp', prices[0])
        self.assertIn('price_eur_kwh', prices[0])
    
    def test_static_offset(self):
        """Test static offset calculation"""
        wholesale = 0.05
        
        # Test different hours
        morning_peak = apply_static_offset(wholesale, 8)
        self.assertGreater(morning_peak, wholesale)
        
        night_low = apply_static_offset(wholesale, 5)
        self.assertGreater(night_low, wholesale)
        
        # Morning peak should have higher offset than night
        self.assertGreater(morning_peak, night_low)
    
    def test_market_stats(self):
        """Test market statistics calculation"""
        prices = [0.04, 0.05, 0.06, 0.08, 0.10]
        stats = get_market_stats(prices)
        
        self.assertEqual(stats['average'], 0.066)
        self.assertEqual(stats['spread'], 0.06)
        self.assertEqual(stats['min'], 0.04)
        self.assertEqual(stats['max'], 0.10)
    
    def test_fallback_chain(self):
        """Test fallback manager returns price"""
        timestamp = datetime.now()
        result = fallback_manager.get_price(timestamp, 'Frank Energie')
        
        self.assertIsNotNone(result, "Should get price from some tier")
        self.assertIn('price_eur_kwh', result)
        self.assertIn('tier', result)
        self.assertIn('reference', result)
    
    def test_reference_data_structure(self):
        """Test reference data has expected structure"""
        timestamp = datetime.now()
        result = fallback_manager.get_price(timestamp, 'Frank Energie')
        
        if 'reference' in result:
            ref = result['reference']
            self.assertIn('expected_range', ref)
            self.assertIn('low', ref['expected_range'])
            self.assertIn('high', ref['expected_range'])

if __name__ == '__main__':
    unittest.main()
EOF

# Run tests
python3 tests/test_kiss_stack.py
```

**Expected:** All tests pass

---

### Action 1.5.2: Integration Test

```bash
cd /opt/github/synctacles-api/

# Test full fallback chain
python3 -c "
import sys
sys.path.insert(0, '.')
from synctacles_db.fallback.fallback_manager import fallback_manager
from datetime import datetime, timedelta

# Test current timestamp
now = datetime.now()
print(f'Testing timestamp: {now}')

result = fallback_manager.get_price(now, 'Frank Energie')

print(f'\\nResult:')
print(f'  Price: €{result[\"price_eur_kwh\"]:.3f}/kWh')
print(f'  Source: {result[\"source\"]}')
print(f'  Tier: {result[\"tier\"]}')
print(f'  Quality: {result[\"quality\"]}')
print(f'  Allow GO: {result[\"allow_go\"]}')

if 'reference' in result:
    print(f'\\nReference data:')
    ref = result['reference']
    if 'expected_range' in ref:
        print(f'  Expected range: €{ref[\"expected_range\"][\"low\"]:.3f} - €{ref[\"expected_range\"][\"high\"]:.3f}')
    if 'market' in ref and ref['market']:
        print(f'  Market average: €{ref[\"market\"][\"average\"]:.3f}/kWh')
"
```

**Expected output:**
```
Testing timestamp: 2026-01-17 14:23:45

Result:
  Price: €0.247/kWh
  Source: Frank DB
  Tier: 1
  Quality: FRESH
  Allow GO: true

Reference data:
  Expected range: €0.210 - €0.280
  Market average: €0.245/kWh
```

---

## 1.6 CODE CLEANUP

### Action 1.6.1: Remove Old Components

```bash
ssh root@135.181.255.83
cd /opt/github/synctacles-api/

# 1. Remove old client files
echo "=== Removing old coefficient client ==="
rm -f synctacles_db/clients/consumer_price_client.py
rm -f synctacles_db/clients/coefficient_client.py

# 2. Remove old tier 2 code
echo "=== Removing Tier 2 Enever-Frank code ==="
rm -f synctacles_db/fallback/tier2_enever_frank.py

# 3. Remove VPN-related code (if exists)
echo "=== Removing VPN code ==="
rm -rf synctacles_db/vpn/ 2>/dev/null || echo "No VPN directory found"

# 4. List files to verify removal
ls -la synctacles_db/clients/
ls -la synctacles_db/fallback/

# Should NOT see:
# - consumer_price_client.py
# - coefficient_client.py
# - tier2_enever_frank.py
```

---

### Action 1.6.2: Clean Imports and References

```bash
cd /opt/github/synctacles-api/

# Find all references to old code
echo "=== Searching for old imports ==="
grep -r "ConsumerPriceClient" --include="*.py" . || echo "✓ No ConsumerPriceClient references"
grep -r "CoefficientClient" --include="*.py" . || echo "✓ No CoefficientClient references"
grep -r "tier2_enever" --include="*.py" . || echo "✓ No tier2_enever references"

# If found, manually edit each file to remove imports
# Example:
# nano synctacles_db/fallback/fallback_manager.py
# Remove: from synctacles_db.clients.consumer_price_client import ConsumerPriceClient

# Verify no broken imports
python3 -c "
import sys
sys.path.insert(0, '.')
from synctacles_db.fallback.fallback_manager import fallback_manager
print('✓ Imports OK - no broken references')
"
```

---

### Action 1.6.3: Database Cleanup

```bash
# Remove old tables
sudo -u postgres psql energy_insights_nl << 'EOF'
-- Drop Tier 2 table (Enever-Frank via coefficient server)
DROP TABLE IF EXISTS enever_frank_prices CASCADE;

-- Drop coefficient cache (if exists)
DROP TABLE IF EXISTS coefficient_cache CASCADE;

-- Drop price model metadata (if exists)
DROP TABLE IF EXISTS price_model_metadata CASCADE;

-- Verify tables removed
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public'
AND (table_name LIKE '%coef%' OR table_name LIKE '%enever_frank%');
-- Should return 0 rows

-- Check remaining tables and sizes
SELECT 
    table_name,
    pg_size_pretty(pg_total_relation_size(table_name::regclass)) as size
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY pg_total_relation_size(table_name::regclass) DESC;
EOF
```

---

### Action 1.6.4: Config Cleanup

```bash
cd /opt/github/synctacles-api/

# Backup current .env
cp .env .env.backup

# Remove coefficient server variables
echo "=== Cleaning .env config ==="
sed -i '/COEFFICIENT_SERVER/d' .env
sed -i '/PRICE_MODEL/d' .env
sed -i '/VPN_/d' .env

# Show what was removed
echo "Removed from .env:"
diff .env.backup .env || echo "No changes"

# Verify API still starts
python3 -c "
from dotenv import load_dotenv
load_dotenv()
print('✓ .env loads successfully')
"
```

---

### Action 1.6.5: Dependency Cleanup

```bash
cd /opt/github/synctacles-api/

# Check for VPN-related dependencies
echo "=== Checking for VPN dependencies ==="
pip list | grep -i vpn || echo "✓ No VPN packages"
pip list | grep -i openvpn || echo "✓ No OpenVPN packages"

# If found, uninstall:
# pip uninstall -y openvpn-api
# pip uninstall -y pia-python-client
# pip uninstall -y nordvpn

# Update requirements.txt (remove VPN dependencies)
nano requirements.txt
# Remove lines containing: vpn, openvpn, pia, etc.

# Reinstall from clean requirements
pip install -r requirements.txt
```

---

### Action 1.6.6: Verify Cleanup

```bash
cd /opt/github/synctacles-api/

# Run all tests
echo "=== Running tests after cleanup ==="
python3 -m pytest tests/ -v

# Verify no coefficient references in code
echo "=== Checking for remaining coefficient references ==="
grep -r "coefficient" --include="*.py" . | grep -v "^Binary" | grep -v ".pyc" | grep -v "test_" | grep -v "#.*coefficient"

# Expected: Only find references in:
# - synctacles_db/config/static_offsets.py (comments about replacing coefficients)
# - docs/ (archived documentation)

# Start API and test
systemctl restart synctacles-api
sleep 5
curl http://localhost:8000/api/v1/health

# Should return: {"status": "healthy"}
```

---

## 1.7 DELIVERABLE: WEEK 1 REPORT

**Create:** `/tmp/WEEK1_BACKEND_REPORT.md`

```markdown
# Week 1: Backend Aanpassingen - Status Report

## Completed Tasks

### 1. EasyEnergy Client ✅
- [x] Created `easyenergy_client.py`
- [x] Implemented `get_prices_today()`
- [x] Implemented `get_prices_tomorrow()`
- [x] Implemented `get_prices_range()`
- [x] Health check functional
- [x] Unit tests passing

**Test Results:**
- API responsive: [YES/NO]
- Today prices fetched: [COUNT] records
- Data quality: [GOOD/ISSUES]

### 2. Static Offset ✅
- [x] Created `static_offsets.py`
- [x] 24-hour offset table implemented
- [x] `apply_static_offset()` working
- [x] `get_market_stats()` working
- [x] Unit tests passing

**Test Results:**
- Offset calculation: [PASS/FAIL]
- Market stats: [PASS/FAIL]

### 3. Fallback Manager ✅
- [x] Removed old Tier 2 (Enever-Frank DB)
- [x] Added new Tier 3 (EasyEnergy Direct)
- [x] Updated Tiers 4-6 to use static offset
- [x] Added reference data to responses
- [x] `_add_reference_data()` implemented
- [x] Integration tests passing

**New Tier Structure:**
1. Frank DB → [WORKING/BROKEN]
2. Frank Direct API → [WORKING/BROKEN]
3. EasyEnergy Direct API → [WORKING/BROKEN]
4. ENTSO-E + Static → [WORKING/BROKEN]
5. Energy-Charts + Static → [WORKING/BROKEN]
6. Cache → [WORKING/BROKEN]

### 4. API Endpoints ✅
- [x] Updated price endpoints
- [x] Reference data in responses
- [x] API tests passing

**Sample Response:**
```json
[paste actual API response]
```

### 5. Testing ✅
- [x] Unit tests created
- [x] Integration tests run
- [x] All tests passing: [YES/NO]

**Test Summary:**
- Tests run: [COUNT]
- Tests passed: [COUNT]
- Tests failed: [COUNT]
- Coverage: [%]

### 6. Code Cleanup ✅
- [x] Removed `consumer_price_client.py`
- [x] Removed `coefficient_client.py`
- [x] Removed `tier2_enever_frank.py`
- [x] Removed VPN code directory
- [x] Cleaned old imports: [COUNT] files updated
- [x] Dropped database tables: [enever_frank_prices, coefficient_cache, etc.]
- [x] Cleaned .env config: [COUNT] variables removed
- [x] Uninstalled dependencies: [LIST or "None"]
- [x] Verified no broken references

**Cleanup Summary:**
- Files removed: [COUNT]
- Database tables dropped: [COUNT]
- Config variables removed: [COUNT]
- Dependencies uninstalled: [LIST]

**Verification:**
- ✓ All tests still passing
- ✓ API starts successfully
- ✓ No coefficient references in active code
- ✓ No broken imports

## Issues Found

[List any issues encountered]

## Ready for Week 2

- [x] Backend changes complete
- [x] API serving reference data
- [x] All tests passing
- [x] **Old code cleaned up**
- [x] **No orphaned components**
- [x] Ready for HA component updates

**Estimated accuracy improvement:** 95% → 100% (Frank/Easy direct)
**Code reduction:** [XX] files removed, [XX] KB saved
```

---

# WEEK 2: HA COMPONENT UPDATES

**Repository:** HA Custom Component  
**Time:** 10-15 uur

---

## 2.1 ADD ANOMALIE DETECTIE

### Action 2.1.1: Update Sensor Code

```python
# In HA custom component sensor.py (pseudo-code):

class SynctaclesPriceSensor(CoordinatorDataUpdateCoordinator):
    """Synctacles price sensor with anomalie detectie"""
    
    async def async_update(self):
        """Update sensor data"""
        
        # 1. Fetch API data (includes reference)
        api_data = await self._fetch_api_data()
        
        # 2. Check if user has Enever BYO sensor
        byo_sensor = self.hass.states.get('sensor.enever_byo_price')
        
        if not byo_sensor:
            # No BYO sensor, use API data directly
            self._attr_native_value = api_data['price_eur_kwh']
            self._attr_extra_state_attributes = {
                'source': api_data['source'],
                'tier': api_data['tier'],
                'quality': api_data['quality']
            }
            return
        
        # 3. Anomalie detectie
        byo_price = float(byo_sensor.state)
        api_price = api_data['price_eur_kwh']
        
        # Get expected range from reference data
        if 'reference' in api_data and 'expected_range' in api_data['reference']:
            expected = api_data['reference']['expected_range']
            low_threshold = expected['low'] - 0.03
            high_threshold = expected['high'] + 0.03
            
            # Check for anomalie
            if byo_price < low_threshold:
                _LOGGER.warning(
                    f"Anomalie detected: BYO price €{byo_price:.3f} is too low "
                    f"(expected €{expected['low']:.3f} - €{expected['high']:.3f}). "
                    f"Using Synctacles price instead."
                )
                # Override with API price
                self._attr_native_value = api_price
                self._attr_extra_state_attributes = {
                    'source': f"Synctacles (BYO anomalie: too low)",
                    'byo_price': byo_price,
                    'override_reason': 'anomalie_low'
                }
                
            elif byo_price > high_threshold:
                _LOGGER.warning(
                    f"Anomalie detected: BYO price €{byo_price:.3f} is too high "
                    f"(expected €{expected['low']:.3f} - €{expected['high']:.3f}). "
                    f"Using Synctacles price instead."
                )
                # Override with API price
                self._attr_native_value = api_price
                self._attr_extra_state_attributes = {
                    'source': f"Synctacles (BYO anomalie: too high)",
                    'byo_price': byo_price,
                    'override_reason': 'anomalie_high'
                }
                
            else:
                # BYO is valid, use it
                _LOGGER.debug(f"BYO price €{byo_price:.3f} is valid")
                self._attr_native_value = byo_price
                self._attr_extra_state_attributes = {
                    'source': 'Enever BYO (validated)',
                    'synctacles_price': api_price
                }
        
        else:
            # No reference data, use BYO without validation
            _LOGGER.warning("No reference data available, using BYO without validation")
            self._attr_native_value = byo_price
            self._attr_extra_state_attributes = {
                'source': 'Enever BYO (unvalidated)',
                'synctacles_price': api_price
            }
```

---

### Action 2.1.2: Test Anomalie Detectie

```python
# Create test scenarios

# Test 1: Normal BYO (within range)
byo_price = 0.245
expected_range = {'low': 0.21, 'high': 0.28}
# Expected: Use BYO, no override

# Test 2: BYO too low (anomalie)
byo_price = 0.15  # €0.15 vs expected €0.21-0.28
# Expected: Override with API price

# Test 3: BYO too high (anomalie)
byo_price = 0.35  # €0.35 vs expected €0.21-0.28
# Expected: Override with API price

# Test 4: No reference data
byo_price = 0.245
expected_range = None
# Expected: Use BYO without validation (with warning)

# Test 5: No BYO sensor
byo_sensor = None
# Expected: Use API price directly
```

---

## 2.3 HA COMPONENT CLEANUP

### Action 2.3.1: Remove Old Sensor Code

```python
# In HA custom component repository

# 1. Find old sensor types
grep -r "CoefficientPriceSensor\|OldPriceSensor" --include="*.py" .

# 2. Remove old sensor classes (if found)
# Edit sensor.py:
nano custom_components/synctacles/sensor.py

# DELETE old code like:
# class CoefficientPriceSensor(Entity):
#     """Old sensor that used coefficient server"""
#     ...

# DELETE old imports:
# from .coefficient_client import CoefficientClient

# 3. Verify only new sensor exists
grep "class.*Sensor" custom_components/synctacles/sensor.py

# Should only see:
# class SynctaclesPriceSensor(CoordinatorDataUpdateCoordinator):
```

---

### Action 2.3.2: Update Manifest

```bash
# Update version number (indicates breaking change)
nano custom_components/synctacles/manifest.json

# OLD:
# "version": "1.5.0"

# NEW:
# "version": "2.0.0"  # Major version = breaking changes

# Add changelog note:
# "version": "2.0.0",
# "changelog": "Migrated to KISS stack - removed coefficient server dependency"
```

---

### Action 2.3.3: Remove Old Config Options

```yaml
# If old configuration.yaml options existed:

# Edit custom_components/synctacles/config_flow.py
# Remove old schema fields like:

# OLD:
# CONF_USE_COEFFICIENT_MODEL: bool
# CONF_COEFFICIENT_SERVER_URL: str

# NEW:
# (Remove these entirely - no longer needed)

# Update default config:
# Remove coefficient-related settings
```

---

### Action 2.3.4: Clean Old Helper Functions

```python
# In custom_components/synctacles/

# Find old helper files
find . -name "*coefficient*" -o -name "*model*" | grep -v __pycache__

# Delete old helpers:
rm -f helpers/coefficient_helper.py
rm -f helpers/price_model.py

# Verify no broken imports:
python3 -c "
import sys
sys.path.insert(0, 'custom_components/synctacles')
from sensor import SynctaclesPriceSensor
print('✓ Imports OK')
"
```

---

### Action 2.3.5: Update Documentation

```markdown
# Edit README.md

# Remove sections about:
# - Coefficient server configuration
# - Price model settings
# - VPN requirements

# Add new section:
## v2.0 Migration Notes

**Breaking changes:**
- Removed coefficient server dependency
- Removed price model configuration options
- Added automatic anomalie detectie for Enever BYO

**Migration:**
- Update to v2.0.0
- Remove `use_coefficient_model` from configuration
- No manual steps needed - works automatically

**Improvements:**
- 100% accuracy for Frank/EasyEnergy (was 95%)
- Client-side BYO validation (privacy-friendly)
- Simpler architecture (no external server dependency)
```

---

### Action 2.3.6: Test Clean Install

```bash
# Simulate fresh installation

# 1. Remove current installation
rm -rf custom_components/synctacles/

# 2. Re-install from updated repo
# (as user would via HACS)
git clone https://github.com/[repo]/synctacles.git
cp -r synctacles/custom_components/synctacles custom_components/

# 3. Restart Home Assistant
# Home Assistant UI → Settings → System → Restart

# 4. Verify integration loads
# Home Assistant logs should show:
# "Synctacles integration v2.0.0 loaded successfully"

# 5. Test sensor
# Developer Tools → States → sensor.synctacles_price
# Should show current price with reference data
```

---

## 2.4 UPDATE DOCUMENTATION

### Action 2.4.1: User Guide

```markdown
# Synctacles HA Integration - Anomalie Detectie

## Wat is Anomalie Detectie?

Synctacles valideert automatisch Enever BYO prijzen tegen onze referentie data.
Als Enever een onrealistische prijs rapporteert, schakelen we over naar onze data.

## Hoe Werkt Het?

1. Synctacles haalt prijzen op (inclusief referentie range)
2. Als je Enever BYO sensor hebt, vergelijken we de prijs
3. Bij anomalie (prijs >±€0.03 buiten verwachte range) → override
4. Anders → gebruik je eigen Enever prijs

## Voorbeeld

**Normale situatie:**
- Enever BYO: €0.245/kWh
- Verwachte range: €0.21 - €0.28
- ✅ Binnen range → Gebruik Enever BYO

**Anomalie situatie:**
- Enever BYO: €0.35/kWh (te hoog!)
- Verwachte range: €0.21 - €0.28
- ❌ Buiten range → Override met Synctacles (€0.247)

## Privacy

Je Enever prijs wordt NIET naar onze servers gestuurd.
Validatie gebeurt lokaal in jouw Home Assistant.
```

---

## 2.5 DELIVERABLE: WEEK 2 REPORT

**Create:** `/tmp/WEEK2_HA_COMPONENT_REPORT.md`

```markdown
# Week 2: HA Component Updates - Status Report

## Completed Tasks

### 1. Anomalie Detectie ✅
- [x] Added reference data parsing
- [x] Implemented BYO validation logic
- [x] Override mechanism working
- [x] Logging added

**Test Results:**
- Normal BYO (within range): [PASS/FAIL]
- BYO too low: [OVERRIDE/FAILED]
- BYO too high: [OVERRIDE/FAILED]
- No reference data: [FALLBACK/FAILED]
- No BYO sensor: [API DIRECT/FAILED]

### 2. Component Cleanup ✅
- [x] Removed old sensor classes: [COUNT]
- [x] Updated manifest to v2.0.0
- [x] Removed old config options: [LIST]
- [x] Deleted old helper files: [COUNT] files
- [x] Updated imports: [COUNT] files
- [x] Tested clean install: [PASS/FAIL]

**Cleanup Summary:**
- Old sensors removed: [COUNT]
- Helper files deleted: [COUNT]
- Config options removed: [COUNT]
- Version bump: 1.x.x → 2.0.0

### 3. Documentation ✅
- [x] User guide updated
- [x] Migration notes added
- [x] Code comments added
- [x] Examples documented
- [x] README updated with v2.0 changes

### 4. Testing ✅
- [x] Unit tests created
- [x] Integration tests run
- [x] All scenarios tested
- [x] Clean install verified

**Test Summary:**
- Scenarios tested: [COUNT]
- Passed: [COUNT]
- Failed: [COUNT]

## Issues Found

[List any issues]

## Ready for Week 3

- [x] HA component updated
- [x] Anomalie detectie working
- [x] **Old code removed**
- [x] **Clean v2.0.0 release**
- [x] Tests passing
- [x] Documentation complete
- [x] Ready for decommissioning

**Privacy-friendly:** BYO data stays local ✅
**Clean architecture:** No orphaned code ✅
```

---

# WEEK 3: DECOMMISSION COEFFICIENT SERVER

**Server:** 91.99.150.36 (coefficient)  
**Time:** 6-10 uur

---

## 3.1 PREPARATION

### Action 3.1.1: Final Backup

```bash
ssh root@91.99.150.36

# Create comprehensive backup
BACKUP_DIR="/tmp/coefficient_backup_$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

# 1. Backup database
sudo -u postgres pg_dump coefficient_db > $BACKUP_DIR/coefficient_db.sql
gzip $BACKUP_DIR/coefficient_db.sql

# 2. Backup repository
cd /opt/github/
tar -czf $BACKUP_DIR/coefficient-engine.tar.gz coefficient-engine/

# 3. Backup configs
cp -r /etc/systemd/system/*coefficient* $BACKUP_DIR/
cp -r /etc/systemd/system/*frank* $BACKUP_DIR/
cp -r /etc/systemd/system/*enever* $BACKUP_DIR/

# 4. Export key data
sudo -u postgres psql coefficient_db -c "
COPY (
    SELECT * FROM coefficient_lookup_v2
) TO '/tmp/coefficient_lookup_v2.csv' CSV HEADER;
"
cp /tmp/coefficient_lookup_v2.csv $BACKUP_DIR/

# 5. Copy to safe location
scp -r $BACKUP_DIR root@135.181.255.83:/backup/coefficient_server/

# Verify backup
ls -lh $BACKUP_DIR/
```

**Expected files:**
```
coefficient_db.sql.gz          ~50MB
coefficient-engine.tar.gz      ~10MB
coefficient_lookup_v2.csv      ~50KB
systemd configs                ~20KB
```

---

### Action 3.1.2: Monitor Without Coefficient Calls

```bash
ssh root@135.181.255.83

# Check API logs for coefficient calls
journalctl -u synctacles-api --since "24 hours ago" | grep -i coefficient

# Should show NO coefficient calls after Week 1 deploy

# Check fallback tier usage
grep "Tier [0-9]" /var/log/synctacles-api.log | tail -100

# Should show:
# - Tier 1: Frank DB (most common)
# - Tier 2: Frank Direct API
# - Tier 3: EasyEnergy Direct
# - Tier 4-5: Rarely (only when APIs down)
# - NO coefficient model calls
```

**Monitor for 7 days. If stable → proceed.**

---

## 3.2 DECOMMISSION STEPS

### Action 3.2.1: Disable Collectors

```bash
ssh root@91.99.150.36

# Stop all collectors
systemctl stop enever-collector.timer
systemctl stop frank-live-collector.timer
systemctl stop consumer-collector.timer

# Disable timers (prevent restart)
systemctl disable enever-collector.timer
systemctl disable frank-live-collector.timer
systemctl disable consumer-collector.timer

# Verify stopped
systemctl list-timers | grep -E "enever|frank|consumer"
# Should return nothing

# Check database writes stopped
sudo -u postgres psql coefficient_db -c "
SELECT 
    table_name,
    MAX(imported_at) as last_update
FROM (
    SELECT 'hist_frank_prices' as table_name, MAX(imported_at) as imported_at 
    FROM hist_frank_prices
    UNION ALL
    SELECT 'hist_enever_prices', MAX(timestamp) 
    FROM hist_enever_prices
) t
GROUP BY table_name;
"
# Timestamps should be frozen (no new data)
```

---

### Action 3.2.2: Update DNS/Firewall

```bash
# On main server (135.181.255.83)
ssh root@135.181.255.83

# Remove coefficient server from DNS/config
nano /opt/github/synctacles-api/config/servers.json

# OLD:
# {
#   "coefficient_server": "91.99.150.36",
#   ...
# }

# NEW:
# {
#   // coefficient_server removed
#   ...
# }

# Update firewall rules (if any)
iptables -L | grep 91.99.150.36
# Remove any rules for coefficient server

# Test API still works
curl http://localhost:8000/api/v1/health
curl http://localhost:8000/api/v1/prices/frank-energie
```

---

### Action 3.2.3: Shutdown Server

```bash
ssh root@91.99.150.36

# Final checks
echo "=== FINAL PRE-SHUTDOWN CHECKS ==="
echo "Backups created: $(ls -d /tmp/coefficient_backup_* | wc -l)"
echo "Last DB update: $(sudo -u postgres psql coefficient_db -t -c 'SELECT MAX(imported_at) FROM hist_frank_prices')"
echo "Collectors stopped: $(systemctl list-units --type=timer | grep -c coefficient)"

# Shutdown database
sudo -u postgres pg_ctl stop -D /var/lib/postgresql/data

# Shutdown server (graceful)
shutdown -h now

# On Hetzner control panel:
# - Mark server as "archived"
# - Do NOT delete yet (keep 30 days for rollback)
```

---

### Action 3.2.4: Disconnect VPN

```bash
# Disconnect PIA VPN (was used for Enever collection)
# This happens automatically when server shuts down

# Verify VPN not needed anymore:
# - Enever data comes from user's BYO sensor
# - Frank data comes from direct API
# - EasyEnergy data comes from direct API
# - No scraping = no VPN needed
```

---

## 3.3 POST-DECOMMISSION CLEANUP

### Action 3.3.1: Monitoring Cleanup

```bash
ssh root@135.181.255.83

# 1. Prometheus cleanup
echo "=== Cleaning Prometheus config ==="
nano /etc/prometheus/prometheus.yml

# REMOVE job for coefficient server:
# - job_name: 'coefficient-server'
#   static_configs:
#     - targets: ['91.99.150.36:9090']

# Reload Prometheus
systemctl reload prometheus

# Verify removed
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job == "coefficient-server")'
# Should return empty

# 2. Grafana cleanup
echo "=== Removing Grafana dashboards ==="
# Via Grafana UI or API:
curl -X DELETE http://admin:password@localhost:3000/api/dashboards/uid/coefficient-server-health
curl -X DELETE http://localhost:3000/api/dashboards/uid/price-model-accuracy
curl -X DELETE http://localhost:3000/api/dashboards/uid/vpn-status

# Keep dashboards:
# - API Performance
# - Fallback Tier Usage
# - System Health

# 3. Alertmanager cleanup
echo "=== Cleaning Alertmanager rules ==="
nano /etc/alertmanager/alertmanager.yml

# REMOVE alerts:
# - coefficient_server_down
# - vpn_connection_lost
# - calibration_failed
# - price_model_stale

# Keep alerts:
# - api_error_rate_high
# - fallback_tier_degradation
# - database_connection_lost

# Reload Alertmanager
systemctl reload alertmanager

# 4. Verify monitoring still works
curl http://localhost:9090/api/v1/query?query=up
# Should show synctacles-api as up
```

---

### Action 3.3.2: DNS & Firewall Cleanup

```bash
# 1. DNS cleanup (if coefficient server had DNS entry)
echo "=== Cleaning DNS ==="

# If using external DNS provider:
# Remove A record: coefficient.synctacles.com → 91.99.150.36

# If using /etc/hosts:
nano /etc/hosts
# Remove lines:
# 91.99.150.36 coefficient coefficient-server coef

# Verify removed
ping coefficient.synctacles.com
# Should fail or resolve to nothing

# 2. Firewall cleanup
echo "=== Cleaning firewall rules ==="

# List current rules for coefficient server
iptables -L -n -v | grep 91.99.150.36

# Remove rules (if any):
iptables -D INPUT -s 91.99.150.36 -j ACCEPT
iptables -D OUTPUT -d 91.99.150.36 -j ACCEPT
iptables -D FORWARD -d 91.99.150.36 -j ACCEPT

# Save rules
iptables-save > /etc/iptables/rules.v4

# 3. SSH config cleanup
nano ~/.ssh/config
# Remove:
# Host coefficient
#   HostName 91.99.150.36
#   User root
#   IdentityFile ~/.ssh/id_rsa

# 4. Known hosts cleanup
ssh-keygen -R 91.99.150.36
ssh-keygen -R coefficient.synctacles.com
```

---

### Action 3.3.3: Documentation Cleanup

```bash
cd /opt/github/synctacles-api/

# 1. Archive old documentation
echo "=== Archiving old docs ==="
mkdir -p docs/archive/

# Move deprecated docs to archive
mv docs/coefficient-server.md docs/archive/
mv docs/vpn-setup.md docs/archive/
mv docs/calibration.md docs/archive/
mv docs/price-model.md docs/archive/

# Add archive note
cat > docs/archive/README.md << 'EOF'
# Archived Documentation

These documents are archived and no longer applicable after the KISS migration.

**Migration date:** 2026-01-17

**Archived components:**
- Coefficient Server (91.99.150.36)
- VPN Setup (PIA Netherlands)
- Calibration Process
- Price Model Training

**Reason:** Migrated to KISS stack with direct APIs.

**Replacement docs:**
- [KISS Stack Architecture](../kiss-architecture.md)
- [EasyEnergy Client](../easyenergy-client.md)
- [Static Offset Table](../static-offset.md)
EOF

# 2. Update architecture documentation
nano docs/architecture.md

# UPDATE:
# - Change server diagram: 2 servers → 1 server
# - Update fallback tiers: 7 → 6
# - Remove coefficient server references
# - Add EasyEnergy client
# - Document static offset approach

# 3. Update README
nano README.md

# REMOVE:
# - Coefficient server setup instructions
# - VPN configuration steps
# - Price model training guide

# ADD:
# - KISS stack explanation
# - EasyEnergy integration
# - Simplified architecture diagram

# 4. Git commit archive
git add docs/archive/
git commit -m "Archive coefficient server documentation (KISS migration)"
git push
```

---

### Action 3.3.4: Dependency Final Cleanup

```bash
cd /opt/github/synctacles-api/

# 1. Review remaining dependencies
echo "=== Checking for unused dependencies ==="
pip list --format=freeze > /tmp/current_deps.txt

# Check for VPN-related packages
cat /tmp/current_deps.txt | grep -i "vpn\|openvpn\|pia\|nord"

# If found, uninstall:
pip uninstall -y openvpn-api pia-python nordvpn-cli

# 2. Review requirements.txt
nano requirements.txt

# Remove any:
# - openvpn-*
# - pia-*
# - nordvpn-*
# - Any coefficient-specific dependencies

# 3. Reinstall clean dependencies
pip install -r requirements.txt --force-reinstall

# 4. Verify API still works
systemctl restart synctacles-api
sleep 5
curl http://localhost:8000/api/v1/health

# Should return: {"status": "healthy"}
```

---

### Action 3.3.5: Verify Complete Independence

```bash
# Final verification that system works WITHOUT coefficient server

echo "=== INDEPENDENCE VERIFICATION ==="

# 1. Check no references to coefficient server
grep -r "91.99.150.36" /opt/github/synctacles-api/ --include="*.py" --include="*.json" --include="*.yml"
# Should return ZERO results (or only in archive/)

# 2. Check no coefficient server imports
grep -r "coefficient.*server\|consumer_price_client\|price_model" /opt/github/synctacles-api/ --include="*.py"
# Should return ZERO results (or only in archive/)

# 3. Test all API endpoints
echo "Testing Frank prices..."
curl http://localhost:8000/api/v1/prices/frank-energie | jq '.price_eur_kwh'

echo "Testing EasyEnergy prices..."
curl http://localhost:8000/api/v1/prices/easyenergy | jq '.price_eur_kwh'

echo "Testing fallback with simulated failure..."
# Temporarily block Frank API
# (test that EasyEnergy fallback works)

# 4. Monitor logs for 1 hour
echo "Monitoring logs for coefficient server references..."
tail -f /var/log/synctacles-api.log | grep -i "coefficient\|91.99.150.36" &
MONITOR_PID=$!

sleep 3600  # Monitor for 1 hour

kill $MONITOR_PID
# Should see ZERO references to coefficient server

echo "✓ System fully independent of coefficient server"
```

---

## 3.4 POST-DECOMMISSION VALIDATION

### Action 3.4.1: Verify System Working

```bash
ssh root@135.181.255.83

# Test fallback chain (all tiers)
python3 -c "
import sys
sys.path.insert(0, '/opt/github/synctacles-api')
from synctacles_db.fallback.fallback_manager import fallback_manager
from datetime import datetime

ts = datetime.now()
result = fallback_manager.get_price(ts, 'Frank Energie')

print(f'Price: €{result[\"price_eur_kwh\"]:.3f}')
print(f'Source: {result[\"source\"]}')
print(f'Tier: {result[\"tier\"]}')
print(f'Success: {result is not None}')
"

# Should work WITHOUT coefficient server

# Check API endpoints
curl http://localhost:8000/api/v1/prices/frank-energie | jq
curl http://localhost:8000/api/v1/prices/easyenergy | jq

# Should return valid prices

# Monitor logs for errors
journalctl -u synctacles-api --since "1 hour ago" | grep -i error
# Should show NO coefficient-related errors
```

---

### Action 3.4.2: Monitor for 7 Days

```bash
# Check daily:

# 1. API uptime
curl http://localhost:8000/api/v1/health

# 2. Fallback tier distribution
grep "Tier" /var/log/synctacles-api.log | tail -1000 | sort | uniq -c

# Should show:
# - Tier 1 (Frank DB): ~80%
# - Tier 2 (Frank Direct): ~15%
# - Tier 3 (EasyEnergy): ~3%
# - Tier 4-5: ~2%

# 3. Error rate
journalctl -u synctacles-api --since "24 hours ago" | grep -c ERROR

# Should be LOW (<10/day)

# 4. User reports
# Monitor HA community forum, GitHub issues
# Should see NO complaints about missing prices
```

---

## 3.5 DELIVERABLE: WEEK 3 REPORT

**Create:** `/tmp/WEEK3_DECOMMISSION_REPORT.md`

```markdown
# Week 3: Decommission Coefficient Server - Status Report

## Pre-Decommission

### Backups Created ✅
- [x] Database dump: `coefficient_db.sql.gz` ([SIZE])
- [x] Repository backup: `coefficient-engine.tar.gz` ([SIZE])
- [x] Config files: Systemd units backed up
- [x] Key data: `coefficient_lookup_v2.csv` exported
- [x] Backup location: `135.181.255.83:/backup/coefficient_server/`

**Backup verification:** [PASS/FAIL]

### Monitoring Period ✅
- [x] 7 days without coefficient calls
- [x] API stable: [YES/NO]
- [x] No errors: [YES/NO]
- [x] Fallback tiers working: [YES/NO]

## Decommission Steps

### 1. Collectors Disabled ✅
- [x] enever-collector: Stopped & disabled
- [x] frank-live-collector: Stopped & disabled
- [x] consumer-collector: Stopped & disabled
- [x] Last DB update: [TIMESTAMP]

### 2. Server Shutdown ✅
- [x] Database stopped gracefully
- [x] Server shut down: [TIMESTAMP]
- [x] Hetzner marked as archived
- [x] VPN disconnected

### 3. Infrastructure Cleanup ✅
- [x] DNS entries removed: [COUNT]
- [x] Firewall rules removed: [COUNT]
- [x] SSH config cleaned
- [x] Known hosts updated
- [x] Config files updated: [COUNT]

### 4. Monitoring Cleanup ✅
- [x] Prometheus: Job removed
- [x] Grafana: [COUNT] dashboards deleted
- [x] Alertmanager: [COUNT] alerts removed
- [x] Monitoring verified working

**Monitoring cleanup:**
- Prometheus jobs removed: [COUNT]
- Grafana dashboards archived: [COUNT]
- Alertmanager rules removed: [COUNT]
- Active monitors remaining: [COUNT]

### 5. Documentation Cleanup ✅
- [x] Archived old docs: [COUNT] files
- [x] Updated architecture.md
- [x] Updated README.md
- [x] Created archive/README.md
- [x] Git committed changes

**Documentation:**
- Files archived: [LIST]
- Architecture diagram updated: [YES/NO]
- README updated: [YES/NO]

### 6. Dependency Cleanup ✅
- [x] VPN packages uninstalled: [LIST or "None"]
- [x] requirements.txt updated
- [x] Clean reinstall completed
- [x] API tested after cleanup

**Dependencies removed:** [COUNT]

## Post-Decommission Validation

### API Health ✅
- [x] Health endpoint: [RESPONDING/DOWN]
- [x] Price endpoints: [WORKING/BROKEN]
- [x] Fallback chain: [WORKING/BROKEN]

**Test results:**
```
Frank price: €[X.XXX]/kWh from Tier [N]
EasyEnergy price: €[X.XXX]/kWh from Tier [N]
```

### Independence Verification ✅
- [x] No coefficient server references in code: [VERIFIED]
- [x] No coefficient server in configs: [VERIFIED]
- [x] No coefficient server imports: [VERIFIED]
- [x] 1-hour log monitoring clean: [VERIFIED]

**Grep results:**
- Code references to 91.99.150.36: [0]
- Imports of coefficient client: [0]
- Config references: [0]

### Tier Usage (7 days) ✅
- Tier 1 (Frank DB): [XX]%
- Tier 2 (Frank Direct): [XX]%
- Tier 3 (EasyEnergy): [XX]%
- Tier 4-5 (ENTSO-E + Static): [XX]%
- Tier 6 (Cache): [XX]%

**Expected:** Tier 1-3 = ~98%, Tier 4-6 = ~2%

### Error Rate ✅
- Errors/day: [COUNT]
- Coefficient errors: [0] (should be zero)
- API errors: [COUNT] (should be <10)

### User Feedback ✅
- GitHub issues: [COUNT]
- Forum complaints: [COUNT]
- Support requests: [COUNT]

**Expected:** Zero complaints about missing prices

## Financial Impact

### Costs Eliminated
- Hetzner CX23: €150/year
- VPN subscription: €[XX]/year (if cancelled)
- Monitoring overhead: (time saved)

**Total savings:** €[XXX]/year

### Operational Improvements
- Servers to manage: 2 → 1 (-50%)
- External dependencies: Coef server → Direct APIs
- Failure modes: 4 → 1 (-75%)
- Complexity: High → Low
- Code files: [XX] removed
- Database tables: [XX] dropped
- Config variables: [XX] removed

## Cleanup Summary

**Total cleanup:**
- Files removed: [XX]
- Database tables dropped: [XX]
- Config variables removed: [XX]
- Dependencies uninstalled: [XX]
- Monitoring configs removed: [XX]
- Documentation archived: [XX]
- DNS entries removed: [XX]
- Firewall rules removed: [XX]

**Code quality:**
- No orphaned code: ✅
- No broken imports: ✅
- No dead configs: ✅
- No unused dependencies: ✅

## Rollback Plan

**Server kept for 30 days** (until [DATE])

If issues arise:
1. Re-enable coefficient server in Hetzner
2. Start server (takes 2 min)
3. Re-enable collectors
4. Update synctacles-api config
5. Rollback code changes (git revert)
6. Monitor

**Rollback time:** ~1 hour

## Conclusion

**Decommission Status:** [SUCCESS/FAILED/PARTIAL]

**System Health:** [EXCELLENT/GOOD/ISSUES]

**Cleanup Complete:** [YES/NO]

**Independence Verified:** [YES/NO]

**Ready for Production:** [YES/NO]

**Recommendation:**
- [Keep server archived for 30 days]
- [After 30 days: Delete permanently]
- [Cancel VPN subscription: YES/NO]

---

**Decommission Date:** [TIMESTAMP]
**Cleanup Completed:** [TIMESTAMP]
**Report Generated:** [TIMESTAMP]
**Final Sign-Off:** [Leo/Pending]
```

---

# WEEK 4+: MULTI-COUNTRY PREP (OPTIONAL)

**Time:** 15-20 uur

---

## 4.1 COUNTRY CONFIG SCHEMA

### Action 4.1.1: Design Configuration

```bash
cd /opt/github/synctacles-api/

# Create country config
cat > synctacles_db/config/countries.json << 'EOF'
{
  "NL": {
    "name": "Netherlands",
    "mode": "direct_api",
    "currency": "EUR",
    "timezone": "Europe/Amsterdam",
    
    "sources": {
      "primary": [
        {
          "name": "Frank Energie",
          "type": "graphql",
          "endpoint": "https://frank-graphql-prod.graphcdn.app/",
          "accuracy": "100%"
        },
        {
          "name": "EasyEnergy",
          "type": "rest",
          "endpoint": "https://mijn.easyenergy.com/nl/api/tariff",
          "accuracy": "100%"
        }
      ],
      "fallback": {
        "wholesale": "ENTSO-E",
        "method": "static_offset",
        "offset_table": "HOURLY_OFFSET"
      }
    },
    
    "market": {
      "bidding_zone": "NL",
      "entso_e_code": "10YNL----------L"
    }
  },
  
  "DE": {
    "name": "Germany",
    "mode": "coefficient",
    "currency": "EUR",
    "timezone": "Europe/Berlin",
    
    "sources": {
      "primary": [
        {
          "name": "aWATTar",
          "type": "rest",
          "endpoint": "https://api.awattar.de/",
          "status": "RESEARCH_NEEDED",
          "accuracy": "TBD"
        }
      ],
      "fallback": {
        "wholesale": "ENTSO-E",
        "method": "coefficient",
        "slope": 1.25,
        "intercept": 0.18,
        "source": "manual_calibration_2026",
        "accuracy": "85-90%"
      }
    },
    
    "market": {
      "bidding_zone": "DE-LU",
      "entso_e_code": "10Y1001A1001A82H"
    }
  }
}
EOF
```

---

### Action 4.1.2: Country-Aware Fallback

```python
# Add to fallback_manager.py:

import json

class FallbackManager:
    """Country-aware fallback manager"""
    
    def __init__(self, country_code='NL'):
        self.country = self._load_country_config(country_code)
        # ... rest of init
    
    def _load_country_config(self, country_code):
        """Load country-specific configuration"""
        with open('synctacles_db/config/countries.json') as f:
            countries = json.load(f)
        
        if country_code not in countries:
            raise ValueError(f"Country {country_code} not supported")
        
        return countries[country_code]
    
    def _get_fallback_method(self):
        """Get fallback method for current country"""
        return self.country['sources']['fallback']['method']
    
    def _apply_fallback_conversion(self, wholesale_price, timestamp):
        """Apply country-specific wholesale → consumer conversion"""
        
        method = self._get_fallback_method()
        
        if method == 'static_offset':
            # NL: Use static hourly offset
            return apply_static_offset(wholesale_price, timestamp.hour)
        
        elif method == 'coefficient':
            # DE/BE/AT: Use slope + intercept
            slope = self.country['sources']['fallback']['slope']
            intercept = self.country['sources']['fallback']['intercept']
            return wholesale_price * slope + intercept
        
        else:
            raise ValueError(f"Unknown fallback method: {method}")
```

---

## 4.2 RESEARCH: GERMANY (aWATTar)

### Action 4.2.1: API Investigation

```bash
# Research aWATTar API
# Documentation: https://www.awattar.de/services/api

# Test endpoint
curl "https://api.awattar.de/v1/marketdata" | jq

# Expected response:
# {
#   "data": [
#     {
#       "start_timestamp": 1642377600000,
#       "end_timestamp": 1642381200000,
#       "marketprice": 150.25,
#       "unit": "Eur/MWh"
#     },
#     ...
#   ]
# }

# Questions to answer:
# 1. Is API publicly available? YES/NO
# 2. Rate limits? [DETAILS]
# 3. Historical data? [HOW FAR BACK]
# 4. Accuracy vs ENTSO-E? [TEST NEEDED]
# 5. Coverage? [REGIONS]
```

---

## 4.3 DELIVERABLE: MULTI-COUNTRY REPORT

**Create:** `/tmp/MULTICOUNTRY_PREP_REPORT.md`

```markdown
# Multi-Country Preparation Report

## Country Config Schema ✅

### Implemented
- [x] `countries.json` schema designed
- [x] NL configuration complete
- [x] DE template created
- [x] Country-aware fallback manager

### Configuration Structure
```json
{
  "mode": "direct_api" | "coefficient",
  "sources": {
    "primary": [...],
    "fallback": {
      "method": "static_offset" | "coefficient"
    }
  }
}
```

## Germany Research

### aWATTar API
- Public availability: [YES/NO/UNKNOWN]
- Rate limits: [DETAILS]
- Historical data: [RANGE]
- Accuracy: [TBD - needs testing]

### Next Steps
1. [Create aWATTar test account]
2. [Compare with ENTSO-E for 30 days]
3. [Calculate coefficient if needed]
4. [Pilot with 5-10 German users]

## Rollout Timeline

| Country | Priority | Ready Date | Blockers |
|---------|----------|------------|----------|
| NL | Live | 2026-01-XX | None |
| DE | High | 2026-Q1 | aWATTar research |
| BE | Medium | 2026-Q2 | No API found yet |
| AT | Low | 2026-Q3 | aWATTar coverage? |

## Conclusion

**Country config:** Ready for expansion ✅
**NL: KISS stack** → No coefficient needed
**DE/BE/AT:** Coefficient config ready, awaits research
```

---

# FINAL DELIVERABLE: COMPLETE KISS MIGRATION REPORT

**Create:** `/tmp/KISS_MIGRATION_COMPLETE.md`

```markdown
# KISS Migration Complete - Final Report

**Migration Period:** Week 1 - Week 3
**Status:** [SUCCESS/PARTIAL/FAILED]
**Date:** [TIMESTAMP]

---

## Executive Summary

### Objective
Migrate from coefficient server architecture to KISS stack (Direct APIs + Static Offset).

### Results

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Servers | 2 | 1 | -50% |
| Accuracy (Frank) | 95% | 100% | +5pp |
| Accuracy (EasyEnergy) | 95% | 100% | +5pp |
| Annual cost | €150 | €0 | -€150 |
| Fallback tiers | 7 | 6 | Simplified |
| External dependencies | 3 | 2 | -33% |
| VPN needed | Yes | No | Eliminated |

### Deliverables

✅ Week 1: Backend (EasyEnergy client, static offset, fallback refactor)
✅ Week 2: HA Component (anomalie detectie)
✅ Week 3: Decommission (backup, shutdown, validation)
✅ Week 4: Multi-country prep (country config schema)

---

## Technical Changes

### New Components
1. `easyenergy_client.py` - Direct API integration
2. `static_offsets.py` - 24-hour offset table
3. Reference data in API responses
4. Client-side anomalie detectie (HA)

### Removed Components
1. Coefficient server (91.99.150.36)
2. ConsumerPriceClient.get_price_model()
3. VPN configuration (PIA Netherlands)
4. Calibration jobs (slope/intercept training)

### Modified Components
1. `fallback_manager.py` - 7 tiers → 6 tiers
2. API endpoints - Added reference data
3. HA sensor - Added BYO validation

---

## Validation Results

### API Health (7 days post-decommission)
- Uptime: [XX.XX]%
- Avg response time: [XX]ms
- Error rate: [X.XX]%

### Tier Distribution
- Tier 1 (Frank DB): [XX]%
- Tier 2 (Frank Direct): [XX]%
- Tier 3 (EasyEnergy): [XX]%
- Tier 4-5 (Fallback): [XX]%
- Tier 6 (Cache): [XX]%

### User Impact
- Support tickets: [COUNT]
- Price complaints: [COUNT]
- Anomalie overrides: [COUNT/day]

**Expected:** Zero complaints, <5 overrides/day

---

## Financial Impact

### Cost Savings
- Server: €150/year
- VPN: €[XX]/year (if cancelled)
- Time saved: ~[XX] hours/month (no calibration)

**Total:** €[XXX]/year + [XX] hours/month

### ROI
- Migration effort: ~50 hours
- Payback period: ~[X] months

---

## Lessons Learned

### What Went Well
- [List successes]
- [e.g., EasyEnergy API very reliable]
- [e.g., Static offset surprisingly accurate]

### What Could Improve
- [List challenges]
- [e.g., Documentation could be better]

### Unexpected Findings
- [List surprises]
- [e.g., Frank/EasyEnergy perfect correlation]

---

## Rollback Status

**Coefficient server:** Archived (kept 30 days until [DATE])
**Rollback readiness:** READY (can restore in ~1 hour)
**Rollback triggered:** NO

**Recommendation:** Delete server after [DATE] if no issues

---

## Next Steps

### Short Term (Week 5-8)
1. Monitor anomalie detection accuracy
2. Optimize static offset based on real data
3. Cancel VPN subscription (if confirmed not needed)

### Medium Term (Month 2-3)
1. Germany pilot (aWATTar research)
2. Country config testing
3. Multi-country rollout prep

### Long Term (Q2+)
1. Expand to DE, BE, AT
2. Evaluate other EU markets
3. Partnership opportunities (Enever?)

---

## Sign-Off

**Technical Lead:** Claude Code (CC)
**Product Owner:** Leo
**Status:** [APPROVED/NEEDS REVIEW]
**Date:** [TIMESTAMP]

---

**Migration Status:** ✅ COMPLETE
**Production Ready:** ✅ YES
**Coefficient Server:** 🗑️ DECOMMISSIONED
```

---

# ROLLBACK PLAN (IF NEEDED)

**Trigger conditions:**
- API uptime <99% for 3+ days
- Error rate >5% for 2+ days
- Multiple user complaints about prices
- Fallback tiers failing >10% of time

**Rollback steps:**

```bash
# 1. Re-enable coefficient server (Hetzner)
# Boot server from Hetzner control panel

# 2. Start collectors
ssh root@91.99.150.36
systemctl start enever-collector.timer
systemctl start frank-live-collector.timer
systemctl start consumer-collector.timer

# 3. Rollback code
ssh root@135.181.255.83
cd /opt/github/synctacles-api/
git revert [COMMIT_HASH_WEEK1]
systemctl restart synctacles-api

# 4. Verify
curl http://localhost:8000/api/v1/prices/frank-energie

# 5. Monitor
# Watch logs for 24h to ensure stability
```

**Rollback time:** ~1 hour  
**Data loss:** None (backups available)

---

# SUCCESS METRICS

Track these for 30 days post-migration:

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| API uptime | >99.9% | [XX.X]% | [✅/❌] |
| Tier 1-3 usage | >95% | [XX]% | [✅/❌] |
| Error rate | <1% | [X.X]% | [✅/❌] |
| User complaints | 0 | [X] | [✅/❌] |
| Anomalie overrides | <1% | [X.X]% | [✅/❌] |
| Response time | <200ms | [XX]ms | [✅/❌] |

**Overall Success:** [PASS/FAIL]

---

# COMPLETE CLEANUP CHECKLIST

Use this checklist to verify ALL cleanup tasks completed:

## Week 1: Backend Cleanup

### Code Cleanup
- [ ] Removed `consumer_price_client.py`
- [ ] Removed `coefficient_client.py`
- [ ] Removed `tier2_enever_frank.py`
- [ ] Removed VPN code directory
- [ ] Updated all imports (no broken references)
- [ ] Verified with: `grep -r "ConsumerPriceClient" --include="*.py" .`
- [ ] Verified with: `grep -r "coefficient.*client" --include="*.py" .`

### Database Cleanup
- [ ] Dropped `enever_frank_prices` table
- [ ] Dropped `coefficient_cache` table
- [ ] Dropped `price_model_metadata` table
- [ ] Verified with: `SELECT * FROM information_schema.tables WHERE table_name LIKE '%coef%'`

### Config Cleanup
- [ ] Removed `COEFFICIENT_SERVER_URL` from .env
- [ ] Removed `COEFFICIENT_SERVER_API_KEY` from .env
- [ ] Removed `VPN_*` variables from .env
- [ ] Backed up old .env as .env.backup

### Dependency Cleanup
- [ ] Uninstalled VPN packages (if any)
- [ ] Updated requirements.txt
- [ ] Reinstalled from clean requirements
- [ ] Verified API starts: `systemctl status synctacles-api`

### Testing
- [ ] All unit tests passing
- [ ] Integration tests passing
- [ ] API health check: `curl http://localhost:8000/api/v1/health`
- [ ] Price endpoints working

---

## Week 2: HA Component Cleanup

### Component Cleanup
- [ ] Removed old sensor classes
- [ ] Removed `coefficient_helper.py` (if exists)
- [ ] Removed `price_model.py` (if exists)
- [ ] Updated manifest.json to v2.0.0
- [ ] Removed old config options
- [ ] Updated all imports

### Documentation Cleanup
- [ ] Updated README.md
- [ ] Added v2.0 migration notes
- [ ] Documented breaking changes
- [ ] Added anomalie detectie documentation

### Testing
- [ ] Clean install tested
- [ ] All anomalie scenarios tested
- [ ] Version bump verified (2.0.0)
- [ ] No broken imports

---

## Week 3: Infrastructure Cleanup

### Monitoring Cleanup
- [ ] Removed Prometheus job for coefficient server
- [ ] Deleted Grafana dashboards: [LIST]
- [ ] Removed Alertmanager rules: [LIST]
- [ ] Reloaded Prometheus: `systemctl reload prometheus`
- [ ] Reloaded Alertmanager: `systemctl reload alertmanager`
- [ ] Verified monitoring still works

### DNS & Network Cleanup
- [ ] Removed DNS entry: coefficient.synctacles.com
- [ ] Cleaned /etc/hosts
- [ ] Removed firewall rules for 91.99.150.36
- [ ] Saved iptables: `iptables-save`
- [ ] Cleaned ~/.ssh/config
- [ ] Cleaned known_hosts: `ssh-keygen -R 91.99.150.36`

### Documentation Cleanup
- [ ] Archived `coefficient-server.md`
- [ ] Archived `vpn-setup.md`
- [ ] Archived `calibration.md`
- [ ] Archived `price-model.md`
- [ ] Created `docs/archive/README.md`
- [ ] Updated `docs/architecture.md`
- [ ] Updated main README.md
- [ ] Git committed archive changes

### Dependency Final Cleanup
- [ ] Verified no VPN packages: `pip list | grep vpn`
- [ ] Checked requirements.txt clean
- [ ] Reinstalled dependencies
- [ ] API tested after cleanup

### Independence Verification
- [ ] No code references: `grep -r "91.99.150.36" . --include="*.py"`
- [ ] No config references: `grep -r "coefficient.*server" . --include="*.json"`
- [ ] No import references: `grep -r "consumer_price_client" . --include="*.py"`
- [ ] 1-hour log monitoring (no coefficient mentions)
- [ ] All API endpoints working
- [ ] Fallback chain tested

---

## Post-Migration (After 30 Days)

### Final Cleanup
- [ ] Verified 30 days of stable operation
- [ ] No rollback needed
- [ ] Delete coefficient server in Hetzner
- [ ] Delete old backups (keep archive)
- [ ] Cancel VPN subscription (if not needed elsewhere)
- [ ] Update billing (verify €150/year savings)

### Archive
- [ ] Final backup created: `coefficient_backup_archive_[DATE].tar.gz`
- [ ] Moved to long-term storage: `/backup/archive/`
- [ ] Delete original backup: `/backup/coefficient_server/`

---

## Cleanup Verification Commands

Run these to verify cleanup complete:

```bash
# 1. No coefficient code references
cd /opt/github/synctacles-api/
grep -r "coefficient" --include="*.py" . | grep -v "^Binary" | grep -v "archive/"
# Should return 0 results (except comments)

# 2. No old imports
grep -r "ConsumerPriceClient\|CoefficientClient" --include="*.py" .
# Should return 0 results

# 3. No coefficient server in configs
grep -r "91.99.150.36" . --include="*.json" --include="*.yml" --include="*.env"
# Should return 0 results

# 4. No VPN dependencies
pip list | grep -i "vpn\|openvpn"
# Should return 0 results

# 5. Database clean
sudo -u postgres psql energy_insights_nl -c "
SELECT table_name FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name LIKE '%coef%' OR table_name LIKE '%enever_frank%';
"
# Should return 0 rows

# 6. Monitoring clean
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job == "coefficient-server")'
# Should return empty

# 7. DNS clean
ping -c 1 coefficient.synctacles.com
# Should fail or timeout

# 8. Firewall clean
iptables -L -n | grep 91.99.150.36
# Should return 0 results

# 9. API working
curl http://localhost:8000/api/v1/health
# Should return: {"status": "healthy"}

# 10. Fallback working
curl http://localhost:8000/api/v1/prices/frank-energie | jq '.tier'
# Should return: 1, 2, or 3 (NOT 4-6 unless APIs down)
```

---

## Final Sign-Off

After ALL cleanup tasks completed:

- [ ] Week 1 cleanup: COMPLETE
- [ ] Week 2 cleanup: COMPLETE
- [ ] Week 3 cleanup: COMPLETE
- [ ] All verification commands: PASS
- [ ] 30-day monitoring: STABLE
- [ ] Final cleanup: COMPLETE

**Migration Status:** ✅ COMPLETE AND CLEAN

**Signed off by:** [Leo] on [DATE]

---

**HANDOFF COMPLETE**

CC: Begin with Week 1 backend changes. Report progress after each week. Do NOT proceed to next week without Leo's approval.

**Estimated timeline:** 3-4 weeks for full migration.
