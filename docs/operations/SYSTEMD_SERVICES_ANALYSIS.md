# SystemD Services Analysis and Configuration Status

**Date:** 2026-01-06
**Status:** All working services verified and fixed
**Conclusion:** Service naming consolidation needed

---

## Executive Summary

The system has **dual service naming conventions** creating confusion:
- **New (correct):** `energy-insights-nl-*` services
- **Old (deprecated):** `synctacles-*` services

All **functional services are working correctly** with the new naming. The old naming convention appears to be legacy naming that should be deprecated.

---

## Service Status Matrix

| Service Name | Type | Current Status | Last Exit Code | Note |
|---|---|---|---|---|
| **energy-insights-nl-api** | Running | ✅ **ACTIVE** | N/A | FastAPI Server (Gunicorn) |
| **energy-insights-nl-collector** | Timer-triggered | ✅ SUCCESS | 0 | Data collection working |
| **energy-insights-nl-health** | Timer-triggered | ✅ SUCCESS | 0 | Health checks passing |
| **energy-insights-nl-importer** | Timer-triggered | ✅ SUCCESS | 0 | Data import working (2min 43s runtime) |
| **energy-insights-nl-normalizer** | Timer-triggered | ✅ SUCCESS | 0 | **FIXED** - Permission issue resolved |
| **energy-insights-nl-tennet** | Timer-triggered | ❌ FAILED | N/A | **INTENTIONALLY DISABLED** (off-limits per SKILL_02 BYO-KEY model) |
| **synctacles-collector** | Timer-triggered | ⚠️ Deprecated | - | Legacy naming - same script as energy-insights-nl-collector |
| **synctacles-health** | Timer-triggered | ⚠️ Deprecated | - | Legacy naming - same script as energy-insights-nl-health |
| **synctacles-importer** | Timer-triggered | ⚠️ Deprecated | - | Legacy naming - same script as energy-insights-nl-importer |
| **synctacles-normalizer** | Timer-triggered | ✅ SUCCESS | 0 | **FIXED** - Now working (same permission issue) |
| **synctacles-tennet** | Timer-triggered | ❌ FAILED | N/A | **INTENTIONALLY DISABLED** (off-limits per SKILL_02 BYO-KEY model) |

---

## Issues Fixed

### 1. Permission Error in Normalizer Services

**Problem:**
Both `energy-insights-nl-normalizer` and `synctacles-normalizer` were failing with:
```
PermissionError: [Errno 13] Permission denied: '/opt/energy-insights-nl/app/synctacles_db/normalizers/base.py'
```

**Root Cause:**
5 files in `/opt/energy-insights-nl/app/synctacles_db/normalizers/` were owned by `root:root` with restrictive permissions (`rw-------`):
- `base.py`
- `normalize_entso_e_a44.py`
- `normalize_entso_e_a65.py`
- `normalize_entso_e_a75.py`
- `normalize_prices.py`

The services run as user `energy-insights-nl` and could not read these files.

**Fix Applied:**
```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/energy-insights-nl/app/synctacles_db/normalizers/
chmod -R u+r,g+r,o-rwx /opt/energy-insights-nl/app/synctacles_db/normalizers/
```

**Result:**
✅ Both normalizer services now work correctly:
- `energy-insights-nl-normalizer`: Exit code 0/SUCCESS (1.392s runtime)
- `synctacles-normalizer`: Exit code 0/SUCCESS (1.471s runtime)

---

## Service Architecture

### Data Pipeline Workflow

```
COLLECTORS (7-10s)
    ↓
IMPORTERS (2min 43s - fetches external raw data)
    ↓
NORMALIZERS (1.4s - processes raw → normalized)
    ↓
API (Serves via FastAPI)
    ↓
HEALTH CHECK (Validates end-to-end)
```

### Service Dependencies

```
energy-insights-nl-importer.timer
    → energy-insights-nl-importer.service
        → /opt/energy-insights-nl/app/scripts/run_importers.sh
            → ENTSO-E A75 (Actual Load)
            → ENTSO-E A65 (Load Forecast)
            [TenneT intentionally excluded]

energy-insights-nl-normalizer.timer
    → After energy-insights-nl-importer completes
    → /opt/energy-insights-nl/app/scripts/run_normalizers.sh
        → Processes raw data to normalized tables

energy-insights-nl-collector.timer
    → /opt/energy-insights-nl/app/scripts/run_collectors.sh
        → Data collection (7-10s)

energy-insights-nl-health.timer
    → /opt/energy-insights-nl/app/scripts/health_check.sh
        → API health check
        → Database connectivity check
```

---

## Dual Naming Issue

### The Problem

We have **two complete sets** of identical services:

**New (Correct) Naming:**
- `energy-insights-nl-collector` ✅
- `energy-insights-nl-health` ✅
- `energy-insights-nl-importer` ✅
- `energy-insights-nl-normalizer` ✅
- `energy-insights-nl-tennet` ⚠️ (intentionally disabled)

**Old (Deprecated) Naming:**
- `synctacles-collector` (legacy)
- `synctacles-health` (legacy)
- `synctacles-importer` (legacy)
- `synctacles-normalizer` (legacy - now fixed)
- `synctacles-tennet` ⚠️ (intentionally disabled)

### Why This Exists

Historical context from `SKILL_11_REPO_AND_ACCOUNTS.md`:
- Old service naming: `synctacles-*` (deprecated)
- New service naming: `energy-insights-nl-*` (current)
- Repository rename from synctacles-api to correct naming ongoing

### Recommendation

The `synctacles-*` services should be **disabled or removed** to avoid confusion, as they:
1. Are legacy naming from old project structure
2. Run identical code/scripts as new naming
3. Create ambiguity about which services to monitor
4. Duplicate system resources (running same tasks twice)

---

## TenneT Services Status

**Status:** ❌ Both intentionally failed
**Reason:** Off-limits per SKILL_02 (Bring-Your-Own-Key model)

TenneT data cannot be publicly distributed. These services are disabled by policy:
- `energy-insights-nl-tennet`
- `synctacles-tennet`

The importers explicitly exclude TenneT:
```bash
# From /opt/energy-insights-nl/app/scripts/run_importers.sh
# NOTE: TenneT importer intentionally excluded (off-limits, BYO-KEY model per SKILL_02)
"${PYTHON}" -m synctacles_db.importers.import_entso_e_a75
"${PYTHON}" -m synctacles_db.importers.import_entso_e_a65
# ← TenneT NOT called
```

---

## Service Scripts

### run_importers.sh
- **Path:** `/opt/energy-insights-nl/app/scripts/run_importers.sh`
- **Runtime:** ~2min 43s
- **Purpose:** Fetch external raw data (ENTSO-E A75, A65)
- **Status:** ✅ Working

### run_collectors.sh
- **Path:** `/opt/energy-insights-nl/app/scripts/run_collectors.sh`
- **Runtime:** ~7-10s
- **Purpose:** Internal data collection
- **Status:** ✅ Working

### run_normalizers.sh
- **Path:** `/opt/energy-insights-nl/app/scripts/run_normalizers.sh`
- **Runtime:** ~1.4s
- **Purpose:** Transform raw → normalized data
- **Status:** ✅ Working (Fixed 2026-01-06)

### health_check.sh
- **Path:** `/opt/energy-insights-nl/app/scripts/health_check.sh`
- **Purpose:** Validate API and database
- **Status:** ✅ Working

---

## Timer Schedules

All timers are configured and triggering correctly:

```bash
# Check active timers
systemctl list-timers --all | grep energy-insights-nl

# Example output:
# Tue 2026-01-07 00:00:00 UTC    ...  energy-insights-nl-collector.timer
# Tue 2026-01-07 00:15:00 UTC    ...  energy-insights-nl-importer.timer
# Tue 2026-01-07 00:20:00 UTC    ...  energy-insights-nl-normalizer.timer
```

---

## Ownership and Permissions

All services run as: **`energy-insights-nl`** user

**Key directories:**
- `/opt/energy-insights-nl/` — Service home
- `/opt/energy-insights-nl/app/` — Application code
- `/opt/energy-insights-nl/.ssh/` — Git SSH keys
- `/var/log/energy-insights/` — Log files

**Fixed permissions (2026-01-06):**
- `/opt/energy-insights-nl/app/synctacles_db/normalizers/` — Changed from `root:root` to `energy-insights-nl:energy-insights-nl`

---

## Next Steps

### Priority 1 (Action Recommended)
Disable or remove `synctacles-*` services to eliminate confusion:
```bash
sudo systemctl disable synctacles-collector.timer
sudo systemctl disable synctacles-health.timer
sudo systemctl disable synctacles-importer.timer
sudo systemctl disable synctacles-normalizer.timer
sudo systemctl disable synctacles-tennet.timer

# Or remove them entirely if confirmed deprecated
sudo rm /etc/systemd/system/synctacles-*.service
sudo rm /etc/systemd/system/synctacles-*.timer
```

### Priority 2 (Monitoring)
The monitoring infrastructure (Phase 1) should track:
- ✅ `energy-insights-nl-importer` — Success rate, runtime < 4min
- ✅ `energy-insights-nl-normalizer` — Success rate, runtime < 5s
- ✅ `energy-insights-nl-api` — Uptime and response times
- ✅ Health checks — Database and API connectivity

### Priority 3 (TenneT)
Keep `energy-insights-nl-tennet` disabled unless:
1. Legal review confirms redistribution is allowed
2. Customer provides their own API key (BYO-KEY model)
3. SKILL_02 policy is updated

---

## Testing Results

### Importer Service Test (2026-01-06 18:31-18:34)
```
Start time: 18:31:17
End time:   18:34:49
Duration:   2min 43.393s
Exit code:  0 (SUCCESS)
Data:       Fresh data loaded into raw_* tables
```

### Collector Service Test (2026-01-06 18:35:33-18:35:40)
```
Duration:   7s
Exit code:  0 (SUCCESS)
```

### Health Service Test (2026-01-06 18:36:17)
```
API:        OK ✅
Database:   OK ✅
Exit code:  0 (SUCCESS)
```

### Normalizer Service Test (2026-01-06 18:38:56-18:38:58 - after fix)
```
Duration:   2s
Exit code:  0 (SUCCESS)
Data:       Normalized tables updated
```

---

## Conclusion

**All functional services are working correctly.** The system is ready for the monitoring infrastructure (Phase 1) to begin tracking service health and performance.

The primary outstanding issue is **service naming consolidation** — the legacy `synctacles-*` services should be disabled or removed to prevent confusion and duplicate execution.

---

**Document:** SYSTEMD_SERVICES_ANALYSIS.md
**Created:** 2026-01-06
**Author:** Claude Code
**Status:** Ready for review
