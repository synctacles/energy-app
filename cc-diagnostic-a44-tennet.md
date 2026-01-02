# Diagnostic Report: A44 Prices & TenneT Auth Status

**Generated**: 2026-01-01 23:58 UTC
**Scope**: A44 price collection pipeline + TenneT authentication

---

## Part 1: A44 Price Collector Status

### ✅ Collector is Running Successfully

**Evidence**:
```
Jan 01 23:56:43 ENTSO-E A44 Price Collector
├─ Fetched 97 price records (TODAY: 2026-01-01)
├─ Saved to a44_NL_prices_20260101.csv ✅
├─ Fetched 92 price records (TOMORROW: 2026-01-02)
├─ Saved to a44_NL_prices_20260102.csv ✅
└─ Status: COMPLETE
```

**Collection Schedule**: Runs every 5 minutes (via timer)
- 2026-01-01 23:40:43 ✅
- 2026-01-01 23:41:43 ✅
- 2026-01-01 23:56:43 ✅

### ⚠️ Problem: Importer NOT Processing New A44 Files

**Files Collected** (Raw):
```bash
Latest files from collector (Jan 1, 23:56):
- a44_NL_prices_20260101.csv  (3122 bytes)
- a44_NL_prices_20260102.csv  (2971 bytes)

Older files (not processed):
- a44_NL_prices_20251231.csv  (2983 bytes) - Dec 30 20:28
- a44_NL_prices_20251230.csv  (3152 bytes) - Dec 30 20:28
- a44_NL_prices_20251229.csv  (3164 bytes) - Dec 29 23:56
```

**Files in Database** (Normalized):
```sql
SELECT COUNT(*), MAX(timestamp), MIN(timestamp)
FROM norm_entso_e_a44;

Result: 92 records
Max: 2025-12-24 22:45:00+00 (8 days old!)
Min: 2025-12-24 00:00:00+00
```

### 🔴 Root Cause: Data Gap

| Layer | Status | Age | Notes |
|-------|--------|-----|-------|
| **Collector** | ✅ Fresh | Just fetched | Collecting TODAY + TOMORROW |
| **Raw Files** | ✅ Fresh | Jan 1, 23:56 | 5 CSV files on disk |
| **Imported (Raw DB)** | ❌ Stale | 8 days old | Not importing new files |
| **Normalized (API)** | ❌ Stale | 8 days old | Not normalizing new data |

### Why New Files Aren't Being Imported

**Hypothesis**: The importer script is not configured to process A44 CSV files

Check the importer configuration:
```bash
grep -r "a44\|A44\|prices" /opt/energy-insights-nl/app/scripts/run_importers.sh
```

**Most Likely Issue**:
- Importer only handles ENTSO-E XML files (A75, A65)
- A44 CSV files are collected but not being processed
- Need to add A44 importer step

---

## Part 2: TenneT API Authentication

### ✅ API Key is Loaded Correctly

```bash
grep TENNET /opt/.env
TENNET_API_KEY="de1f44f9-7085-4d93-81ca-710d2233c84c"
```

### ✅ API Endpoint Configured

```
TENNET_API_BASE = https://api.tennet.eu
```

### 🔴 Problem: 401 Authentication Error Persists

**Last Error** (from journalctl):
```
ERROR TennetBalanceDeltaIngestor: Authentication failed (401)
```

**Possible Root Causes**:
1. API key is **expired or revoked**
2. API key has **insufficient permissions**
3. API endpoint **changed** or requires additional headers
4. Request format **incorrect** (wrong header name?)

**Check Request Headers**:
```python
# From tennet_ingestor.py
self.session.headers.update({
    'User-Agent': f'{brand_slug}-collector/1.0',
    'Accept': 'application/json',
    'apikey': self.api_key  # TenneT expects 'apikey' header
})
```

---

## Priority Analysis

### Current Data Status

| Data | Freshness | Impact | Fix Priority |
|------|-----------|--------|--------------|
| **A75 Generation** | Mixed (raw fresh, normalized updating) | High - Used for analysis | Monitor (will auto-update) |
| **A65 Load** | Fresh (2026-01-01 23:56) | High - Used for analysis | ✅ No action needed |
| **A44 Prices** | Stale (8 days old) | Medium - Historical only | **HIGH - Action Required** |
| **TenneT Balance** | Current (2025-12-29) | Medium - Balance data | **MEDIUM - Investigate** |

---

## 🎯 Priority Recommendation

### **PRIORITY 1: FIX A44 IMPORTER (URGENT)** 🔴

**Why**:
- Collector is working ✅ (fetching fresh data)
- Raw files are accumulating ✅ (on disk since Jan 1)
- But NOT being imported ❌ (database still 8 days old)
- This is a low-hanging fruit (likely just missing importer step)

**Estimated Impact**:
- Once fixed: A44 will jump from 8 days old to current
- Fixes visible in API immediately
- No dependencies on other components

**Action Items**:
1. Check if A44 importer is configured in `run_importers.sh`
2. If not, add step to process A44 CSV files
3. Verify import works: `python3 -m synctacles_db.importers.import_a44`
4. Run importer manually
5. Verify normalized table updates

---

### **PRIORITY 2: INVESTIGATE TENNET 401 ERROR (MEDIUM)** 🟡

**Why**:
- TenneT is optional (nice-to-have, not blocking)
- But if key is expired, it will keep failing
- Better to fix before it becomes critical

**Investigation Steps**:
1. Test API key directly:
   ```bash
   curl -v -H "apikey: de1f44f9-7085-4d93-81ca-710d2233c84c" \
        "https://api.tennet.eu/..."
   ```
2. Check TenneT documentation for current endpoint
3. Verify key permissions in TenneT portal
4. Contact TenneT if key expired

---

### **PRIORITY 3: MONITOR A75 NORMALIZATION (LOW)** 🟢

**Why**:
- Already working (raw data is fresh)
- Normalized table will auto-update on next normalizer run
- Just requires patience (~5 min cycles)

**Status**:
- Last run: 2026-01-01 23:41
- Next run: ~2026-01-02 00:00
- Expected result: A75 normalized table updates to fresh data

---

## Recommendation Summary

```
🚨 DO THIS FIRST:
   A44 Importer Issue - Fix the import pipeline
   Effort: Low (likely 1-2 lines of config/code)
   Impact: High (fixes 8-day-old price data)
   Timeline: 5-10 minutes

⚠️  THEN INVESTIGATE:
   TenneT 401 Error - Check if key is expired/revoked
   Effort: Medium (may need external contact)
   Impact: Medium (nice-to-have, not blocking)
   Timeline: After A44 is fixed

✅ THEN MONITOR:
   A75 Normalization - Just watch it update
   Effort: None (automatic)
   Impact: High (generation mix data)
   Timeline: Next cycle (~5 min)
```

---

## Next Steps

### Immediate (Now - 10 min)
1. [ ] Find and examine `run_importers.sh`
2. [ ] Check if A44 importer is included
3. [ ] If missing, add import step
4. [ ] Run importer manually
5. [ ] Verify A44 data updated in database

### Short-term (10-30 min)
6. [ ] Test TenneT API key with curl
7. [ ] Check TenneT portal for key status
8. [ ] Update key if expired OR check API documentation

### Monitoring (Ongoing)
9. [ ] Watch A75 normalized table (should update)
10. [ ] Monitor collector timers (should keep running)
11. [ ] Check fetch_log for successful entries

---

## Files to Check

**A44 Importer**:
```bash
ls /opt/energy-insights-nl/app/importers/ | grep -i a44
grep -r "a44\|A44\|prices" /opt/energy-insights-nl/app/scripts/
```

**Run Scripts**:
```bash
cat /opt/energy-insights-nl/app/scripts/run_importers.sh
```

**TenneT Config**:
```bash
grep -r "tennet\|TENNET" /opt/.env /opt/energy-insights-nl/.env
```

---

**Generated**: 2026-01-01 23:58 UTC
**Status**: Ready for action
