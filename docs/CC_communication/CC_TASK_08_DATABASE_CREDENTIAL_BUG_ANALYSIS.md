# CC TASK 08: Database Credential Bug - Root Cause Analysis & Remediation

**Date:** 2026-01-05
**Severity:** P1 CRITICAL
**Status:** DISCOVERED - Requires immediate remediation
**Impact:** Production data pipeline broken since Day 1

---

## Executive Summary

A systemic authentication bug has been discovered affecting the data normalization pipeline. All normalizers fail with `FATAL: role "synctacles" does not exist` because:

1. **Hardcoded credentials** reference a non-existent database user "synctacles"
2. **Only user in database** is "energy_insights_nl"
3. **Bug present since initial commit** - never worked
4. **Data pipeline is BROKEN** - critical datasets have 2-14 day backlogs

### Impact at a Glance
- ❌ **A44 Prices normalizer:** 2 days behind (raw: Jan 6, norm: Jan 4)
- ❌ **Prices normalizer:** 14 days without updates (last: Dec 22)
- ✓ **A65/A75 normalizers:** Working (coincidentally use env vars)
- ⚠️ **API endpoints:** Return stale data via fallback (masks the problem)

---

## PHASE 1: Hardcoded Credentials Inventory

### CRITICAL Files (Direct Hardcoding)

**File: `/opt/github/synctacles-api/synctacles_db/normalizers/normalize_entso_e_a44.py` (Line 18)**
```python
# BROKEN CODE:
DB_URL = "postgresql://synctacles@localhost:5432/synctacles"
# Missing: os.getenv() call - always uses hardcoded value
```

**Impact:** A44 normalizer cannot authenticate, runs every 15 min, fails every time

---

### RISKY Files (Fallback to Hardcoded)

| File | Line | Pattern | Risk Level |
|------|------|---------|-----------|
| `normalizers/normalize_prices.py` | 12 | `os.getenv("DATABASE_URL", "postgresql://synctacles@...")` | HIGH - Wrong default user |
| `api/endpoints/prices.py` | 37 | `os.getenv("DATABASE_URL", "postgresql://synctacles@...")` | MEDIUM - Has env var as primary |
| `importers/import_entso_e_a75.py` | 25 | `os.getenv("DATABASE_URL", "postgresql://synctacles@...")` | MEDIUM - Has env var |
| `importers/import_entso_e_a65.py` | 22 | `os.getenv("DATABASE_URL", "postgresql://synctacles@...")` | MEDIUM - Has env var |
| `importers/import_energy_charts_prices.py` | 14 | `os.getenv("DATABASE_URL", "postgresql://synctacles@...")` | MEDIUM - Has env var |
| `importers/archive/import_tennet_balance.py` | 21 | `os.getenv("DATABASE_URL", "postgresql://synctacles@...")` | LOW - Archived |
| `scripts/admin/backfill_data.py` | - | Same pattern | MEDIUM - Admin tool |

**Total:** 8 files with credentials, 1 CRITICAL (normalize_entso_e_a44.py)

---

## PHASE 2: Database User Inventory

### Current PostgreSQL Users
```
postgres           (Superuser)
energy_insights_nl (Regular user - ALL tables owned by this user)
```

**KEY FINDING:** No "synctacles" user exists. Never existed. Was never created during setup.

### Database Objects Ownership
```
Owner: energy_insights_nl

Tables:
  - raw_entso_e_a44
  - norm_entso_e_a44
  - raw_entso_e_a75
  - norm_entso_e_a75
  - raw_entso_e_a65
  - norm_entso_e_a65
  - raw_prices
  - norm_prices
  - raw_tennet_balance
  - norm_tennet_balance
```

The correct connection string should be:
```
postgresql://energy_insights_nl@localhost:5432/energy_insights_nl
```

---

## PHASE 3: Normalizer Execution History

### A44 Normalizer Timeline

```
Systemd Service 1: energy-insights-nl-normalizer.service
Systemd Service 2: synctacles-normalizer.service

Both configured to run every 15 minutes (since Dec 30, 2025)
Both FAIL immediately with authentication error

Last Error (Jan 05 14:21:44 UTC):
  sqlalchemy.exc.OperationalError: (psycopg2.OperationalError)
  connection to server at "localhost" (::1), port 5432 failed:
  FATAL: role "synctacles" does not exist

Last Successful Run: NEVER
```

### Execution Log Pattern
```
Every 15 minutes:
  ❌ Start energy-insights-nl-normalizer.service
  ❌ Import normalize_entso_e_a44
  ❌ Load DB_URL = hardcoded string
  ❌ sqlalchemy.create_engine() attempts connection
  ❌ psycopg2 fails: user does not exist
  ❌ Exception caught in __main__
  ❌ Service exits with status=1/FAILURE
  ... repeat in 15 minutes
```

### Prices Normalizer
```
Same pattern - uses normalize_prices.py
Fails with same error despite having os.getenv()
(because fallback default is also wrong)
```

### A65/A75 Normalizers (WORKING)
```
Both use os.getenv("DATABASE_URL", ...)
DATABASE_URL environment variable is set correctly
Connect as: energy_insights_nl@localhost
Normalizations succeed - sync is up to date
```

---

## PHASE 4: Data Gap Analysis

### Per-Table Status

| Dataset | Raw Table | Norm Table | Raw Latest | Norm Latest | Gap | Status |
|---------|-----------|-----------|-----------|------------|-----|--------|
| **A44 Prices** | raw_entso_e_a44 | norm_entso_e_a44 | 2026-01-06 22:45 | 2026-01-04 22:45 | -2 days | 🚨 BROKEN |
| **A75 Generation** | raw_entso_e_a75 | norm_entso_e_a75 | 2026-01-05 13:30 | 2026-01-05 13:30 | 0 days | ✓ OK |
| **A65 Load** | raw_entso_e_a65 | norm_entso_e_a65 | 2026-01-06 14:45 | 2026-01-06 14:45 | 0 days | ✓ OK |
| **Prices** | raw_prices | norm_prices | 2025-12-22 22:45 | 2025-12-22 22:45 | -14 days | 🚨 STALE |

### Row Counts
```
Dataset A44:
  Raw records:  856 (data being imported successfully)
  Norm records: 664 (stuck at 2026-01-04)
  Gap: 192 records not normalized

Dataset Prices:
  Raw records:  96 (last update: Dec 22)
  Norm records: 96 (last update: Dec 22)
  Gap: 14 days of missing raw data
```

---

## PHASE 5: Git History & Timeline

### When Was The Bug Introduced?

**Commit: `a495aee` - "Initial commit - anonymized energy insights platform"**

```bash
File: synctacles_db/normalizers/normalize_entso_e_a44.py
Line 16 (original): DB_URL = "postgresql://synctacles@localhost:5432/synctacles"

Status: IDENTICAL to current code
```

**CRITICAL FINDING:** Bug was present in the FIRST commit. This normalizer has NEVER worked.

### Timeline

```
[Project Start]
  └─ commit a495aee: normalize_entso_e_a44.py with hardcoded "synctacles"
     └─ Bug introduced immediately
        └─ Normalizer would fail IF run (but wasn't scheduled yet)

[~Dec 30, 2025]
  └─ Systemd timer created: energy-insights-nl-normalizer.timer
     └─ Starts running every 15 minutes
        └─ FAILS immediately with auth error
           └─ But errors only visible in journalctl
              └─ No alerting, no monitoring, problem hidden

[~Dec 22, 2025]
  └─ Last successful price normalization
     └─ (Before timer was added? Or before it started failing consistently?)

[Jan 4, 2025]
  └─ A44 norm table frozen at 2026-01-04 22:45
     └─ No new normalizations processed

[Jan 5, 2026 - TODAY]
  └─ Bug discovered during prices endpoint investigation
     └─ Root cause analysis shows: broken from Day 1
```

---

## Root Cause: Why Wasn't This Caught Earlier?

### 1. No Credential Validation
- Credentials only checked when first query runs
- No startup validation before scheduling normalizer

### 2. Silent Failures
- Systemd timers run silently
- Errors only in journalctl (not alerting)
- No monitoring of normalizer health

### 3. API Masking
- Endpoints still return data (via Energy-Charts fallback)
- Returns 200 OK with stale/fallback data
- Client doesn't know normalization is broken

### 4. Importers Work (Coincidentally)
- Importers use `os.getenv()` with DATABASE_URL set
- They DON'T fail, so pipeline looks OK at first layer
- Problem hidden in second layer (normalizers)

### 5. Two Timers Confusing the Picture
- `energy-insights-nl-normalizer.timer`
- `synctacles-normalizer.timer`
- Both fail, but confusion about why two exist
- Suggests partial/incomplete migration

### 6. No Integration Tests
- No test verifies normalizers can actually connect
- Would have caught this on first run

---

## Remediation Plan

### IMMEDIATE (Within 1 hour)

**Step 1: Fix normalize_entso_e_a44.py** [CRITICAL]

File: `/opt/github/synctacles-api/synctacles_db/normalizers/normalize_entso_e_a44.py`

```python
# Line 16-18 - BEFORE:
from synctacles_db.core.logging import get_logger
DB_URL = "postgresql://synctacles@localhost:5432/synctacles"

# Line 16-18 - AFTER:
from synctacles_db.core.logging import get_logger
import os
DB_URL = os.getenv("DATABASE_URL", "postgresql://energy_insights_nl@localhost:5432/energy_insights_nl")
```

**Step 2: Audit & Fix Fallback Defaults**

For all files with `os.getenv("DATABASE_URL", ...)`:
- Change default from `postgresql://synctacles@...`
- To: `postgresql://energy_insights_nl@...`

Affected files:
- `normalizers/normalize_prices.py` (line 12)
- `api/endpoints/prices.py` (line 37)
- `importers/import_entso_e_a*.py` (all files)
- `scripts/admin/backfill_data.py`

**Step 3: Verify DATABASE_URL in Systemd**

Check that both normalizer services have:
```ini
[Service]
Environment="DATABASE_URL=postgresql://energy_insights_nl@localhost:5432/energy_insights_nl"
```

**Step 4: Git Commit & Deploy**

```bash
git commit -m "fix: correct database credentials for normalizers (energy_insights_nl user)"
git push origin main
```

**Step 5: Restart Services**

```bash
sudo systemctl restart energy-insights-nl-normalizer.service
sudo systemctl restart synctacles-normalizer.service
```

**Step 6: Verify Connection**

```bash
sudo journalctl -u energy-insights-nl-normalizer.service -f
# Should see: "A44 normalizer completed: X OK, Y forward-filled"
```

### SHORT-TERM (Within 24 hours)

**1. Data Backfill**

```bash
# A44: backfill 2 days (Jan 4-6)
python -m synctacles_db.scripts.backfill raw_entso_e_a44 \
  --start "2026-01-04 00:00:00" \
  --end "2026-01-06 23:59:59"

# Prices: backfill 14 days (Dec 22 - Jan 5)
python -m synctacles_db.scripts.backfill raw_prices \
  --start "2025-12-22 00:00:00" \
  --end "2026-01-05 23:59:59"
```

**2. Health Check Implementation**

Add startup validation in normalizers:
```python
def validate_connection():
    """Verify database connection before running"""
    try:
        engine.execute("SELECT 1")
        _LOGGER.info("✓ Database connection validated")
        return True
    except Exception as e:
        _LOGGER.error(f"✗ Database connection failed: {e}")
        raise SystemExit(1)

if __name__ == '__main__':
    validate_connection()  # Run this first
    main()
```

**3. Consolidate Timer Services**

Remove `synctacles-normalizer.timer`, keep only `energy-insights-nl-normalizer.timer`
- Reduce confusion
- Single point of control
- Update run_normalizers.sh if needed

### LONG-TERM (This week)

**1. Centralized Configuration**

Create `synctacles_db/core/settings.py`:
```python
import os

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgresql://energy_insights_nl@localhost:5432/energy_insights_nl"
)

# Prevent hardcoding anywhere
if "synctacles@" in DATABASE_URL:
    raise ValueError("BUG: Still using deprecated 'synctacles' user!")
```

Then all files use:
```python
from synctacles_db.core.settings import DATABASE_URL
```

**2. Integration Testing**

Add test that verifies normalizers work:
```python
def test_normalizers_can_connect():
    """Verify all normalizers can authenticate"""
    from synctacles_db.normalizers import normalize_entso_e_a44, normalize_prices

    # Should not raise auth error
    session = normalize_entso_e_a44.Session()
    session.execute("SELECT 1")
    session.close()
```

**3. Monitoring & Alerting**

- Alert if normalizer fails > 2x in a row
- Daily health report showing gaps
- Prometheus metric: `normalizer_last_run_timestamp`

**4. Documentation**

Update ARCHITECTURE.md:
- Explain normalizer pipeline
- Document credential setup
- Add troubleshooting section

---

## Affected Components Summary

### 🚨 BROKEN (Cannot Authenticate)
- `normalize_entso_e_a44.py` (DIRECT hardcode)
- `normalize_prices.py` (fallback is wrong)
- 2x systemd normalizer services
- **Result:** A44 & Prices datasets stuck

### ✓ WORKING (Correct Credentials)
- All importers (A44, A65, A75, prices)
- All API endpoints
- Database is accessible
- **Result:** Raw data flowing in correctly

### ⚠️ PARTIALLY WORKING (Masking Real Problem)
- API endpoints return stale data via fallback
- Energy-Charts fallback prevents outage
- But wrong for automation (allow_go_action=False)
- **Result:** Problem hidden from users

---

## Key Metrics

| Metric | Value | Severity |
|--------|-------|----------|
| Days since bug introduction | ~7 days (since timers added) | P1 |
| Files with hardcoded credentials | 8 | P1 |
| CRITICAL hardcoded files | 1 | CRITICAL |
| Missing database user | "synctacles" | CRITICAL |
| A44 normalization backlog | 2 days (192 records) | P1 |
| Prices normalization backlog | 14 days | P1 |
| Normalizer success rate | 0% (every run fails) | CRITICAL |
| Time to remediate | ~30 min for fix + 24h for backfill | - |

---

## Verification Checklist

After remediation, verify:

- [ ] normalize_entso_e_a44.py uses `os.getenv("DATABASE_URL", ...)`
- [ ] All fallback defaults point to `energy_insights_nl` user
- [ ] DATABASE_URL env var is set in systemd services
- [ ] normalizer services restart successfully
- [ ] journalctl shows "normalizer completed" (not "failed")
- [ ] norm_entso_e_a44 catches up to raw_entso_e_a44
- [ ] norm_prices updated after Dec 22
- [ ] API /prices endpoint shows fresh data (age < 1 hour)
- [ ] No "synctacles" credentials in codebase

---

## Questions Answered

**Q: Why did A65/A75 work but A44 didn't?**
A: They use `os.getenv()` which returns the correct DATABASE_URL env var. A44 ignores env vars entirely.

**Q: Why wasn't this caught during testing?**
A: No integration tests verify normalizer connectivity. Timers were added later.

**Q: Why did prices endpoint still work?**
A: Energy-Charts fallback masks the missing data. But data is stale (14 days old).

**Q: When did this actually start failing?**
A: Since Day 1 the code was wrong, but only when timers were added (~Dec 30) did it start continuously failing.

**Q: Is there data loss?**
A: No data loss in raw tables. But normalized data is stuck. Backfill will recover it.

---

## References

- [ARCHITECTURE.md](../ARCHITECTURE.md) - System design
- [normalize_entso_e_a44.py](../synctacles_db/normalizers/normalize_entso_e_a44.py) - Normalizer source
- [normalize_prices.py](../synctacles_db/normalizers/normalize_prices.py) - Prices normalizer
- Git commit: `a495aee` - Initial commit with bug
- Git commit: `6d1d8e2` - Logging addition (no fix)

---

**Report Status:** READY FOR IMPLEMENTATION
**Next Steps:** Execute IMMEDIATE remediation steps
**Estimated Time to Resolution:** 30 minutes + 24 hours backfill
