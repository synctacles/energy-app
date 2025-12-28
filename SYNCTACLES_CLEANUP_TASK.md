# SYNCTACLES String Cleanup - User-Facing Only

## OBJECTIVE
Fix ONLY user-facing "SYNCTACLES" strings that users will see in Home Assistant or external APIs. Leave technical/internal strings (docstrings, comments, logs) as-is.

## CRITICAL FIXES (Must Fix)

### 1. Home Assistant Sensor Names
**File:** `custom_components/synctacles/sensor.py`

**Lines to fix:**
```python
# Line 45 - BEFORE:
_attr_name = "SYNCTACLES Generation Total"
# Line 45 - AFTER:
_attr_name = f"{HA_COMPONENT_NAME} Generation Total"

# Line 124 - BEFORE:
_attr_name = "SYNCTACLES Load Actual"
# Line 124 - AFTER:
_attr_name = f"{HA_COMPONENT_NAME} Load Actual"

# Line 193 - BEFORE:
_attr_name = "SYNCTACLES Balance Delta"
# Line 193 - AFTER:
_attr_name = f"{HA_COMPONENT_NAME} Balance Delta"
```

**Import needed at top:**
```python
from .const import DOMAIN, HA_COMPONENT_NAME
```

---

### 2. Device Names
**File:** `custom_components/synctacles/sensor.py`

**Lines to fix:**
```python
# Lines 56, 135, 204 - BEFORE:
"name": "SYNCTACLES Energy Data"
# AFTER:
"name": f"{HA_COMPONENT_NAME} Energy Data"
```

---

### 3. User-Agent Header
**File:** `sparkcrawler_db/collectors/sparkcrawler_tennet_ingestor.py`

**Line 97 - BEFORE:**
```python
'User-Agent': 'SYNCTACLES-SparkCrawler/1.0',
```

**AFTER:**
```python
'User-Agent': f'{os.getenv("BRAND_SLUG", "energy-insights")}-collector/1.0',
```

**Import needed at top:**
```python
import os
```

---

### 4. Hardcoded Log Path
**File:** `sparkcrawler_db/collectors/sparkcrawler_tennet_ingestor.py`

**Line 34 - BEFORE:**
```python
LOG_DIR = Path(os.getenv("SYNCTACLES_LOG_DIR", "/opt/synctacles/logs"))
```

**AFTER:**
```python
LOG_DIR = Path(os.getenv("LOG_PATH", "/opt/energy-insights/logs"))
```

---

### 5. Update const.py for HA Component
**File:** `custom_components/synctacles/const.py`

**Add this constant:**
```python
# Branding (loaded from manifest for consistency)
import json
from pathlib import Path

_manifest_path = Path(__file__).parent / "manifest.json"
_manifest = json.loads(_manifest_path.read_text())
HA_COMPONENT_NAME = _manifest.get("name", "Energy Insights NL")
```

---

## DO NOT CHANGE (Leave As-Is)

### Comments & Docstrings
```python
"""Sensor platform for SYNCTACLES integration."""  # OK - internal doc
# This raw data is ready for SYNCTACLES normalizer  # OK - technical comment
```

### Log Messages
```python
logger.info("SYNCTACLES SparkCrawler — TenneT...")  # OK - technical log
```

### Schema Comments
```python
# Map Energy-Charts types to SYNCTACLES schema  # OK - internal reference
```

---

## VALIDATION CHECKLIST

After making changes, verify:

### 1. No User-Facing SYNCTACLES
```bash
# Should return 0:
grep -r "SYNCTACLES" custom_components/synctacles/sensor.py | grep -v "^[[:space:]]*#" | grep -v '"""' | wc -l
```

### 2. Sensor Names Dynamic
```bash
# Should show f-string usage:
grep "_attr_name" custom_components/synctacles/sensor.py
```

### 3. HA Component Const Exists
```bash
# Should exist:
grep "HA_COMPONENT_NAME" custom_components/synctacles/const.py
```

### 4. User-Agent Dynamic
```bash
# Should show os.getenv:
grep "User-Agent" sparkcrawler_db/collectors/sparkcrawler_tennet_ingestor.py
```

---

## FILES TO MODIFY

Only these 3 files need changes:
1. `custom_components/synctacles/sensor.py` (4 changes)
2. `custom_components/synctacles/const.py` (add HA_COMPONENT_NAME)
3. `sparkcrawler_db/collectors/sparkcrawler_tennet_ingestor.py` (2 changes)

**Total changes:** 7 lines across 3 files

---

## TESTING

After changes:
```python
# Test import works
from custom_components.synctacles.const import HA_COMPONENT_NAME
print(HA_COMPONENT_NAME)  # Should print: "Energy Insights NL"
```

---

## COMMIT MESSAGE

```
Fix: Remove user-facing SYNCTACLES branding

- HA sensor names now use HA_COMPONENT_NAME from manifest
- Device names dynamic from manifest
- User-Agent header uses BRAND_SLUG env var
- Log paths use LOG_PATH env var

Technical/internal strings (comments, logs) unchanged.

Files modified:
- custom_components/synctacles/sensor.py (sensor names)
- custom_components/synctacles/const.py (add HA_COMPONENT_NAME)
- sparkcrawler_db/collectors/sparkcrawler_tennet_ingestor.py (UA + path)
```
