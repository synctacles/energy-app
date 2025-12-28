# ENV Variable Consistency Fix - LOG_PATH

## OBJECTIVE
Replace inconsistent `SYNCTACLES_LOG_DIR` with standardized `LOG_PATH` across all collectors and importers.

## SCOPE
9 files need updates:
- config/settings.py
- sparkcrawler_db/collectors/ (4 files)
- sparkcrawler_db/importers/ (4 files)

## CHANGES NEEDED

### Pattern to Replace

**BEFORE:**
```python
LOG_DIR = Path(os.getenv("SYNCTACLES_LOG_DIR", "/opt/synctacles/logs"))
```

**AFTER:**
```python
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))
```

**Rationale:**
- `LOG_PATH` matches .env.example standard
- Default `/var/log/energy-insights` follows Linux FHS (Filesystem Hierarchy Standard)
- Consistent with other ENV vars (no brand-specific naming)

---

## FILES TO UPDATE

### 1. config/settings.py
**Find:**
```python
SYNCTACLES_LOG_DIR
```

**Replace with:**
```python
LOG_PATH
```

---

### 2. sparkcrawler_db/collectors/energy_charts_prices.py
**Find:**
```python
LOG_DIR = Path(os.getenv("SYNCTACLES_LOG_DIR", "/opt/synctacles/logs"))
```

**Replace with:**
```python
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))
```

---

### 3. sparkcrawler_db/collectors/sparkcrawler_entso_e_a44_prices.py
Same as #2

---

### 4. sparkcrawler_db/collectors/sparkcrawler_entso_e_a65_load.py
Same as #2

---

### 5. sparkcrawler_db/collectors/sparkcrawler_entso_e_a75_generation.py
Same as #2

---

### 6. sparkcrawler_db/importers/import_energy_charts_prices.py
Same as #2

---

### 7. sparkcrawler_db/importers/import_entso_e_a65.py
Same as #2

---

### 8. sparkcrawler_db/importers/import_entso_e_a75.py
Same as #2

---

### 9. sparkcrawler_db/importers/import_tennet_balance.py
Same as #2

---

## VALIDATION

After changes:

### 1. No more SYNCTACLES_LOG_DIR
```bash
# Should return 0:
grep -r "SYNCTACLES_LOG_DIR" --include="*.py" . | wc -l
```

### 2. All use LOG_PATH
```bash
# Should return 9+:
grep -r "LOG_PATH" --include="*.py" . | wc -l
```

### 3. Check .env.example alignment
```bash
# Should exist:
grep "LOG_PATH" .env.example
```

---

## .env.example Verification

Make sure .env.example has:
```bash
LOG_PATH="/var/log/energy-insights"
```

If not present, add it under the "## PATH CONFIGURATION" section.

---

## COMMIT MESSAGE

```
Refactor: Standardize log directory ENV variable

- Replace SYNCTACLES_LOG_DIR with LOG_PATH (9 files)
- Update default path to /var/log/energy-insights
- Align with .env.example standards
- Follow Linux FHS for log locations

Affected:
- config/settings.py
- All collectors (4 files)
- All importers (4 files)
```

---

## NOTES

- This is purely a naming consistency fix
- No functional changes to logging behavior
- Follows .env.example pattern established earlier
- Makes ENV variables brand-agnostic
