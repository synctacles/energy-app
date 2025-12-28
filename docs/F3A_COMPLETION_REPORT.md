# F3A Completion Report — Database Schema + ORM Models

**Date:** 2025-12-09  
**Duration:** 2 hours  
**Status:** ✅ COMPLETED

---

## Deliverables

### 1. Alembic Migration Infrastructure
- **Location:** `/opt/github/synctacles-repo/alembic/`
- **Migration:** `versions/304422564502_001_initial_schema.py`
- **Config:** `alembic.ini` + `env.py` (with ORM model imports)

### 2. Database Schema (PostgreSQL + TimescaleDB)
**Connection:** `postgresql://synctacles@localhost:5432/synctacles`

**Tables Created (8):**
```
raw_entso_e_a75         # ENTSO-E Generation per PSR-type
raw_entso_e_a65         # ENTSO-E Load (actual + forecast)
raw_tennet_balance      # TenneT Balance Delta per platform
norm_entso_e_a75        # Generation mix (pivoted, 9 PSR columns)
norm_entso_e_a65        # Load (actual_mw + forecast_mw)
norm_tennet_balance     # Balance delta (aggregated)
fetch_log               # Metadata audit trail
alembic_version         # Alembic tracking
```

**TimescaleDB Hypertables (3):**
- `norm_entso_e_a75` (partitioned on timestamp)
- `norm_entso_e_a65` (partitioned on timestamp)
- `norm_tennet_balance` (partitioned on timestamp)

**Indexes (8):**
- Raw tables: psr_type, type, platform (query optimization)
- Norm tables: timestamp (automatic via PK)
- Fetch log: source, fetch_time (audit queries)

### 3. ORM Models (SQLAlchemy)

**sparkcrawler_db/models.py:**
```python
RawEntsoeA75        # raw_entso_e_a75
RawEntsoeA65        # raw_entso_e_a65
RawTennetBalance    # raw_tennet_balance
```

**synctacles_db/models.py:**
```python
NormEntsoeA75       # norm_entso_e_a75 (pivoted)
NormEntsoeA65       # norm_entso_e_a65
NormTennetBalance   # norm_tennet_balance
FetchLog            # fetch_log
```

---

## Key Design Decisions

### 1. Composite Primary Keys (id, timestamp)
**Reason:** TimescaleDB hypertables require partitioning column in PK.

**Implementation:**
```python
sa.PrimaryKeyConstraint('id', 'timestamp')
```

**Impact:**
- Enables TimescaleDB time-series optimization
- Automatic index on timestamp (no explicit index needed)
- Supports efficient range queries

### 2. Two-Layer Architecture
**Raw Layer (sparkcrawler_db):**
- Immutable source data
- Preserves all original fields
- 30-day retention policy (future)

**Normalized Layer (synctacles_db):**
- Pivoted/aggregated for API consumption
- Uniform schema across sources
- Unlimited retention (product data)

### 3. Quality Status Field
**Values:** `OK` | `STALE` | `CACHED` | `NO_DATA`

**Purpose:**
- Track data freshness
- Enable graceful degradation (serve old data if fetch fails)
- Support fallback strategy (never expose raw data)

---

## Technical Challenges & Solutions

### Challenge 1: TimescaleDB Hypertable Requirements
**Problem:** `create_hypertable()` failed with "timestamp must be in primary key"

**Solution:**
- Changed PK from `(id)` to `(id, timestamp)` for all norm_* tables
- Removed explicit timestamp indexes (auto-created by PK)

**Code:**
```python
# Before (failed):
sa.PrimaryKeyConstraint('id'),
sa.UniqueConstraint('timestamp', 'country')

# After (success):
sa.PrimaryKeyConstraint('id', 'timestamp')
```

---

### Challenge 2: Index Name Mismatches
**Problem:** Alembic autogenerate detected "changes" due to index naming.

**Root Cause:**
- Migration used short names: `ix_raw_a75_timestamp`
- ORM auto-generated: `ix_raw_entso_e_a75_timestamp`

**Solution:**
- Updated migration script with full table-prefixed index names
- Reset database (no production data yet)
- Result: `alembic check` now clean

---

### Challenge 3: TimescaleDB Extension After Schema Drop
**Problem:** `DROP SCHEMA public CASCADE` removed TimescaleDB extension.

**Solution:**
```sql
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
```

**Lesson:** Always re-enable extensions after schema drops.

---

### Challenge 4: Compression Policy License
**Problem:** `add_compression_policy()` requires TimescaleDB Community license.

**Solution:**
- Commented out compression policies in migration
- Can be re-enabled later with proper license activation
- No impact on V1 functionality (optional optimization)

---

## Verification Tests

### 1. Alembic Check
```bash
alembic check
# Output: "No new upgrade operations detected" ✓
```

### 2. ORM Import Test
```python
from sparkcrawler_db.models import RawEntsoeA75
from synctacles_db.models import NormEntsoeA75
# Success: All models importable ✓
```

### 3. Database State
```sql
\dt  -- 8 tables
SELECT hypertable_name FROM timescaledb_information.hypertables;
-- 3 hypertables ✓
```

---

## File Structure

```
/opt/github/synctacles-repo/
├── alembic/
│   ├── env.py                          # ORM imports + target_metadata
│   ├── versions/
│   │   └── 304422564502_001_initial_schema.py  # Migration 001
│   └── alembic.ini                     # Config (DB URL)
├── sparkcrawler_db/
│   ├── __init__.py
│   └── models.py                       # Raw data ORM (3 models)
├── synctacles_db/
│   ├── __init__.py
│   └── models.py                       # Normalized ORM (4 models)
└── docs/
    └── F3A_COMPLETION_REPORT.md        # This file
```

---

## Next Phase: F3B (Importers)

### Scope
Build importers to read XML/JSON files from `logs/` and write to `raw_*` tables.

### Requirements
1. **ENTSO-E A75 Importer:**
   - Parse XML with namespaces
   - Extract PSR-type + quantity + timestamp
   - Handle timezone (UTC)
   - Upsert to `raw_entso_e_a75`

2. **ENTSO-E A65 Importer:**
   - Parse XML (actual + forecast)
   - Separate actual vs forecast records
   - Upsert to `raw_entso_e_a65`

3. **TenneT Balance Importer:**
   - Parse JSON (5 platforms)
   - Extract delta_mw + price (if available)
   - Upsert to `raw_tennet_balance`

### Dependencies
- ✅ Database schema live
- ✅ ORM models available
- ⏳ XML/JSON parsers (to be built)
- ⏳ Upsert logic (avoid duplicates)

### Estimated Time
3-4 hours (XML parsing + error handling + testing)

---

## Lessons Learned

1. **TimescaleDB Quirks:**
   - Hypertables require timestamp in PK
   - Compression requires license activation
   - Extension survives `DROP TABLE` but not `DROP SCHEMA`

2. **Index Naming:**
   - ORM auto-generates with table prefix
   - Match migration names to ORM for clean autogenerate
   - Explicit indexes on PK columns are redundant

3. **Alembic Best Practices:**
   - Use `alembic check` to verify model sync
   - Test autogenerate before production
   - Keep migration 001 clean (foundation for V2)

4. **Development Workflow:**
   - Empty database = safe to reset + re-run
   - Fix migration scripts before adding data
   - Backup files during debugging (`.bak` pattern)

---

## Git Commit History

```
bbf2376 - ADD: F3A - Database schema + ORM models + Alembic migration 001 (FINAL)
```

---

## Database Credentials

**Connection String:**
```
postgresql://synctacles@localhost:5432/synctacles
```

**Authentication:** Trust (no password, local connections only)

**Environment Variable:**
```bash
DATABASE_URL=postgresql://synctacles@localhost:5432/synctacles
```

---

## Status: ✅ READY FOR F3B

All F3A deliverables complete. Database schema is production-ready for importer development.

**Sign-off:** Leo Blom  
**Date:** 2025-12-09  
**Phase:** F3A Complete — Proceed to F3B
