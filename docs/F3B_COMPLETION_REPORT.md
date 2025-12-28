# F3B Completion Report — Importers (XML/JSON → Database)

**Date:** 2025-12-10  
**Duration:** 1.5 hours  
**Status:** ✅ COMPLETED

---

## Deliverables

### 1. Three Functional Importers
**Location:** `/opt/github/synctacles-repo/sparkcrawler_db/importers/`

```
import_entso_e_a75.py    # Generation per PSR-type (XML → raw_entso_e_a75)
import_entso_e_a65.py    # Load actual/forecast (XML → raw_entso_e_a65)  
import_tennet_balance.py # Balance delta (JSON → raw_tennet_balance)
__init__.py              # Package marker
```

### 2. Import Results

**Database State:**
```sql
Source   | Records | Distinct Types | Timespan
---------|---------|----------------|------------------
A75      | 731     | 9 PSR-types    | 22 hours (Dec 8-9)
A65      | 239     | 2 types        | 2.5 days
TenneT   | 750     | 5 platforms    | ~25 hours

Total: 1,720 raw records imported
```

**PSR-Types (A75):**
- B01 (Biomass), B04 (Gas), B05 (Coal), B14 (Nuclear)
- B16 (Solar), B17 (Waste), B18 (Wind Offshore)
- B19 (Wind Onshore), B20 (Other)

**Load Types (A65):**
- actual (real-time consumption)
- forecast (day-ahead prediction)

**Balance Platforms (TenneT):**
- aFRR, IGCC, MARI, mFRRda, PICASSO

### 3. Database Schema Fixes

Applied to all `raw_*` tables:
```sql
-- Auto-increment sequences
CREATE SEQUENCE raw_{table}_id_seq;
ALTER TABLE raw_{table} ALTER COLUMN id SET DEFAULT nextval('raw_{table}_id_seq');

-- UNIQUE constraints for upsert
ALTER TABLE raw_entso_e_a75 ADD CONSTRAINT uq_raw_entso_e_a75_natural_key 
    UNIQUE (timestamp, country, psr_type);
    
ALTER TABLE raw_entso_e_a65 ADD CONSTRAINT uq_raw_entso_e_a65_natural_key 
    UNIQUE (timestamp, country, type);
    
ALTER TABLE raw_tennet_balance ADD CONSTRAINT uq_raw_tennet_balance_natural_key 
    UNIQUE (timestamp, platform);
```

---

## Key Design Decisions

### 1. Idempotent Upsert Strategy
**Implementation:**
```python
stmt = insert(Model).values(records)
stmt = stmt.on_conflict_do_update(
    index_elements=['timestamp', 'country', 'psr_type'],
    set_={
        'quantity_mw': stmt.excluded.quantity_mw,
        'source_file': stmt.excluded.source_file,
        'imported_at': stmt.excluded.imported_at
    }
)
```

**Result:**
- 3,059 insert operations → 731 unique A75 records
- Re-running importers = safe (no duplicates)
- Handles collector retries + API failures gracefully

**Why Critical:** Development workflow requires re-running imports on same files.

---

### 2. Batch Insert Pattern
**Per-file processing:**
1. Parse entire XML/JSON file
2. Build list of record dicts
3. Single bulk `INSERT ... ON CONFLICT` statement

**Performance:**
- A75: 39 files processed in 1.5 seconds
- TenneT: 750 records (2 files) in 0.4 seconds

**Trade-off:** Higher memory usage vs. database round-trips (acceptable for <1000 records/file).

---

### 3. Flexible File Detection
**Problem:** Collectors use inconsistent naming patterns.

**Solution:**
```python
# A75: Single pattern
a75_files = sorted(logs_dir.glob('a75_NL_*.xml'))

# A65: Multiple patterns (backward compatibility)
a65_files = sorted(
    list(logs_dir.glob('a65_NL_*.xml')) +
    list(logs_dir.glob('entso_e_a65_*.xml'))
)
```

**Lesson:** Real-world data = inconsistent naming → importers must adapt.

---

### 4. TenneT JSON Unpivoting
**Raw Format:**
```json
{
  "timestamp_start": "2025-12-08T00:41:24Z",
  "power_aFRR_in": 408.0,
  "power_aFRR_out": 0.0,
  "power_IGCC_in": 69.9,
  "power_IGCC_out": 0.0,
  ...  // 10 power columns (5 platforms × 2 directions)
  "metadata": {"mid_price": 50.74}
}
```

**Transformation:**
```python
for platform in ['aFRR', 'IGCC', 'MARI', 'mFRRda', 'PICASSO']:
    power_in = item.get(f'power_{platform.lower()}_in', 0.0)
    power_out = item.get(f'power_{platform.lower()}_out', 0.0)
    delta_mw = power_in - power_out
    
    records.append({
        'timestamp': timestamp,
        'platform': platform,
        'delta_mw': delta_mw,
        'price_eur_mwh': mid_price,
        ...
    })
```

**Result:** 150 JSON objects → 750 database records (5× expansion).

**Why:** Normalized schema (one platform per row) vs. wide JSON format.

---

## Technical Challenges & Solutions

### Challenge 1: Missing Auto-Increment Sequences
**Error:**
```
null value in column "id" violates not-null constraint
```

**Root Cause:** F3A migration created `id INTEGER PRIMARY KEY` but no DEFAULT value.

**Fix:**
```sql
CREATE SEQUENCE raw_entso_e_a75_id_seq;
ALTER TABLE raw_entso_e_a75 ALTER COLUMN id SET DEFAULT nextval('raw_entso_e_a75_id_seq');
ALTER SEQUENCE raw_entso_e_a75_id_seq OWNED BY raw_entso_e_a75.id;
```

**Impact:** Applied to all 3 raw tables (10 min per table).

**Lesson:** ORM models need `autoincrement=True` explicitly declared for composite PKs.

---

### Challenge 2: Missing UNIQUE Constraints
**Error:**
```
there is no unique or exclusion constraint matching the ON CONFLICT specification
```

**Root Cause:** F3A migration lacked UNIQUE constraints on natural keys.

**Fix:**
```sql
ALTER TABLE raw_entso_e_a75 
ADD CONSTRAINT uq_raw_entso_e_a75_natural_key 
UNIQUE (timestamp, country, psr_type);
```

**Why Needed:** `ON CONFLICT` requires explicit UNIQUE constraint (not just indexes).

**Lesson:** Upsert logic requires both index + constraint definition.

---

### Challenge 3: Schema Mismatch (TenneT Country Column)
**Error:**
```
Unconsumed column names: country
```

**Root Cause:** Code inserted `country: 'NL'` but `raw_tennet_balance` has no such column.

**Why:** TenneT is Netherlands-only → country is implicit (not stored).

**Fix:** Removed `country` from:
- Insert dict: `records.append({...})`
- Upsert conflict: `index_elements=['timestamp', 'platform']`

**Lesson:** Verify ORM model matches actual table schema before coding.

---

### Challenge 4: Silent Failures (No Output)
**Symptom:** Script exits with code 0 but no console output.

**Root Cause:** Python logging buffer not flushing to stdout.

**Fix:**
```python
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    force=True  # Override existing config
)

# Dual output for debugging
print(f"Processing: {filepath.name}")
logger.info(f"Processing: {filepath.name}")
```

**Result:** Immediate feedback during development runs.

---

## Verification Queries

### Data Completeness
```sql
-- Check all sources
SELECT 'A75' as source, COUNT(*) as records, COUNT(DISTINCT psr_type) as types 
FROM raw_entso_e_a75
UNION ALL
SELECT 'A65', COUNT(*), COUNT(DISTINCT type) FROM raw_entso_e_a65
UNION ALL
SELECT 'TenneT', COUNT(*), COUNT(DISTINCT platform) FROM raw_tennet_balance;
```

### Latest Data Points
```sql
-- Most recent generation mix
SELECT timestamp, psr_type, quantity_mw 
FROM raw_entso_e_a75 
WHERE timestamp = (SELECT MAX(timestamp) FROM raw_entso_e_a75)
ORDER BY psr_type;

-- Load actual vs forecast
SELECT type, COUNT(*), MIN(timestamp), MAX(timestamp)
FROM raw_entso_e_a65
GROUP BY type;

-- Balance delta by platform
SELECT platform, COUNT(*), 
       ROUND(AVG(delta_mw)::numeric, 2) as avg_delta,
       ROUND(AVG(price_eur_mwh)::numeric, 2) as avg_price
FROM raw_tennet_balance
WHERE price_eur_mwh IS NOT NULL
GROUP BY platform
ORDER BY platform;
```

---

## File Structure

```
/opt/github/synctacles-repo/
├── sparkcrawler_db/
│   ├── importers/
│   │   ├── __init__.py
│   │   ├── import_entso_e_a75.py       # 197 lines
│   │   ├── import_entso_e_a65.py       # 191 lines
│   │   └── import_tennet_balance.py    # 167 lines
│   └── models.py                        # ORM (updated with autoincrement)
├── logs/
│   ├── entso_e_raw/                    # 62 XML files (A75 + A65)
│   └── tennet_raw/                     # 2 JSON files
└── docs/
    └── F3B_COMPLETION_REPORT.md        # This file
```

---

## Lessons Learned

1. **Schema Validation First**
   - Always verify table schema before writing importers
   - `\d table_name` reveals missing sequences/constraints
   - Prevented 3× code rewrites

2. **Idempotency = Confidence**
   - Upsert logic eliminates fear of re-runs
   - Critical for development + production retries
   - 3,059 inserts → 731 records proves it works

3. **Flexible Parsers Win**
   - Real-world collectors use inconsistent naming
   - Support multiple glob patterns
   - Fallback logic for edge cases (A65 type detection)

4. **Dual Output = Faster Debug**
   - `print()` + `logger.info()` both
   - Immediate console feedback during dev
   - Log files for production analysis

5. **Error Isolation Prevents Cascades**
   - Per-file try/catch blocks
   - 1 bad file ≠ failed batch
   - Continue processing + report summary

6. **Batch > Loop for Performance**
   - Single INSERT with 88 values >> 88 INSERT calls
   - Acceptable memory trade-off (<1000 records)
   - 39 files in 1.5 seconds

---

## Next Phase: F3C (Normalizers)

### Scope
Transform `raw_*` → `norm_*` tables with business logic:

1. **A75 Normalizer**
   - Pivot 9 PSR-types into columns
   - Output: `norm_entso_e_a75` (B01_mw, B04_mw, ..., total_mw)

2. **A65 Normalizer**
   - Join actual + forecast into single row per timestamp
   - Output: `norm_entso_e_a65` (actual_mw, forecast_mw)

3. **TenneT Normalizer**
   - Aggregate 5 platforms → net delta + weighted avg price
   - Output: `norm_tennet_balance` (delta_mw, price_eur_mwh)

4. **Quality Status Logic**
   - Calculate freshness: OK / STALE / CACHED / NO_DATA
   - Basis for fallback strategy (never serve raw data)

### Dependencies
- ✅ Raw data in database (1,720 records)
- ✅ ORM models (synctacles_db.models)
- ⏳ Pivot/aggregation logic
- ⏳ Quality status calculation
- ⏳ TimescaleDB hypertable queries

### Estimated Time
**3-4 hours**

**Breakdown:**
- A75 pivot normalizer: 90 min
- A65 merge normalizer: 60 min
- TenneT aggregate normalizer: 60 min
- Testing + validation: 30 min

---

## Git Commit

```bash
git add sparkcrawler_db/importers/
git add sparkcrawler_db/models.py  # Updated with autoincrement
git commit -m "ADD: F3B - Importers for A75, A65, TenneT (XML/JSON → raw_* tables)"
```

---

## Status: ✅ F3B COMPLETE — Ready for F3C

**Sign-off:** Leo Blom  
**Date:** 2025-12-10  
**Phase:** F3B Complete → Proceed to F3C (Normalizers)
