# HANDOFF: CC → CAI

**Datum:** 2026-01-08
**Van:** CC
**Naar:** CAI
**Type:** Implementation Blocker + Decision Required
**Prioriteit:** MEDIUM

---

## STATUS

⚠️ **BLOCKED** - Pipeline Dashboard implementation 60% complete, blocked by Prometheus metrics architecture issue

---

## EXECUTIVE SUMMARY

Started implementing Pipeline Health Dashboard per HANDOFF_CAI_CC_PIPELINE_DASHBOARD_IMPLEMENTATION.md.

**Completed (60%):**
- ✅ Grafana cleanup op ENIN-NL (verwijderd, 709 MB freed)
- ✅ Pipeline metrics module created (`synctacles_db/metrics.py`)
- ✅ Metrics instrumentation added to 3 normalizers (A75, A65, A44)
- ✅ API imports metrics module

**Blocked (40%):**
- ❌ Metrics not appearing in /metrics endpoint
- ❌ Gunicorn multi-worker architecture issue
- ❌ Cannot proceed with Grafana dashboard without metrics

**Root Cause:** Prometheus metrics in Gunicorn multi-worker setup require shared state, which our current implementation doesn't have.

---

## WIJZIGINGEN GEDAAN

### 1. Grafana Verwijderd van ENIN-NL

**Actie:**
```bash
sudo systemctl disable grafana-server
sudo apt-get remove --purge grafana -y
sudo apt-get autoremove -y
sudo rm -rf /etc/grafana /var/lib/grafana /var/log/grafana
```

**Verificatie:**
```bash
dpkg -l | grep grafana  # Empty ✅
which grafana-server    # Empty ✅
systemctl list-units | grep grafana  # Empty ✅
```

**Impact:**
- ✅ Freed 709 MB disk space
- ✅ Removed confusion about Grafana location
- ✅ Clean separation: API server ≠ monitoring server
- ⚠️ IRREVERSIBLE without reinstall (not needed - Grafana on monitor.synctacles.com)

**Rollback:** Not needed. Grafana was inactive and unnecessary on API server.

---

### 2. Nieuwe File: `synctacles_db/metrics.py`

**Locatie:** `/opt/github/synctacles-api/synctacles_db/metrics.py`

**Inhoud:**
```python
"""
Pipeline health metrics for Prometheus/Grafana monitoring.
"""
from prometheus_client import Gauge

# Pipeline health metrics
pipeline_collector_last_success = Gauge(
    'pipeline_collector_last_success_timestamp',
    'Timestamp of last successful collector run',
    ['source']
)

pipeline_importer_last_success = Gauge(
    'pipeline_importer_last_success_timestamp',
    'Timestamp of last successful importer run',
    ['source']
)

pipeline_normalizer_last_success = Gauge(
    'pipeline_normalizer_last_success_timestamp',
    'Timestamp of last successful normalizer run',
    ['source']
)

pipeline_data_age_minutes = Gauge(
    'pipeline_data_age_minutes',
    'Age of data in minutes',
    ['source', 'layer']
)

# Helper functions
def record_collector_success(source: str):
    pipeline_collector_last_success.labels(source=source).set_to_current_time()

def record_importer_success(source: str):
    pipeline_importer_last_success.labels(source=source).set_to_current_time()

def record_normalizer_success(source: str):
    pipeline_normalizer_last_success.labels(source=source).set_to_current_time()

def record_data_age(source: str, layer: str, age_minutes: float):
    pipeline_data_age_minutes.labels(source=source, layer=layer).set(age_minutes)
```

**Impact:**
- ✅ NEW FILE - defines pipeline metrics
- ✅ Safe to keep - doesn't break anything
- ⚠️ NOT FUNCTIONAL until Gunicorn multi-worker issue resolved

**Rollback:** Safe to delete file if needed.

---

### 3. Modified: `synctacles_db/api/main.py`

**Changes:**

**Import added:**
```python
from prometheus_client import Counter, Histogram, Gauge, generate_latest, CONTENT_TYPE_LATEST
# Added: Gauge import
```

**Import added:**
```python
# Import pipeline metrics to register them with Prometheus
import synctacles_db.metrics  # noqa: F401
```

**Impact:**
- ✅ Imports Gauge class for metrics
- ✅ Imports metrics module to register with Prometheus
- ✅ No breaking changes - API starts successfully
- ⚠️ Metrics not appearing due to Gunicorn multi-worker issue

**Rollback:**
```bash
# Remove these lines from main.py:
# from prometheus_client import Counter, Histogram, Gauge, generate_latest, CONTENT_TYPE_LATEST
# Change back to:
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST

# Remove this line:
import synctacles_db.metrics  # noqa: F401
```

**Status:** ✅ API OPERATIONAL (tested, no errors)

---

### 4. Modified: `synctacles_db/normalizers/normalize_entso_e_a75.py`

**Changes:**

**Import added:**
```python
from synctacles_db.metrics import record_normalizer_success, record_data_age
```

**After successful normalization (line ~130):**
```python
# Record metrics for monitoring
record_normalizer_success('a75')

sample = session.query(NormEntsoeA75).order_by(NormEntsoeA75.timestamp.desc()).first()
if sample:
    # ... existing debug log ...
    # Record data age
    now = datetime.now(timezone.utc)
    age_minutes = (now - sample.timestamp).total_seconds() / 60
    record_data_age('a75', 'normalized', age_minutes)
```

**Impact:**
- ✅ Calls metrics recording functions after successful normalization
- ✅ No breaking changes - normalizer runs successfully
- ⚠️ Metrics not visible in /metrics endpoint (Gunicorn issue)

**Testing:**
```bash
sudo -u energy-insights-nl bash scripts/run_normalizers.sh
# Result: [2026-01-08 13:01:32] Normalizer batch complete ✅
```

**Rollback:**
```bash
# Remove these lines:
- from synctacles_db.metrics import record_normalizer_success, record_data_age
- record_normalizer_success('a75')
- # Record data age block (6 lines)
```

**Status:** ✅ NORMALIZER OPERATIONAL (tested, runs successfully)

---

### 5. Modified: `synctacles_db/normalizers/normalize_entso_e_a65.py`

**Changes:** Identical pattern to A75

**Import added:**
```python
from synctacles_db.metrics import record_normalizer_success, record_data_age
```

**After successful normalization:**
```python
# Record metrics for monitoring
record_normalizer_success('a65')

sample = session.query(NormEntsoeA65).order_by(NormEntsoeA65.timestamp.desc()).first()
if sample:
    # ... existing debug log ...
    # Record data age
    now = datetime.now(timezone.utc)
    age_minutes = (now - sample.timestamp).total_seconds() / 60
    record_data_age('a65', 'normalized', age_minutes)
```

**Impact:** Same as A75

**Rollback:** Remove same lines

**Status:** ✅ NORMALIZER OPERATIONAL

---

### 6. Modified: `synctacles_db/normalizers/normalize_entso_e_a44.py`

**Changes:** Identical pattern to A75

**Import added:**
```python
from synctacles_db.metrics import record_normalizer_success, record_data_age
```

**After successful normalization:**
```python
# Record metrics for monitoring
record_normalizer_success('a44')

# Record data age
sample = session.query(NormEntsoeA44).order_by(NormEntsoeA44.timestamp.desc()).first()
if sample:
    now = datetime.now(timezone.utc)
    age_minutes = (now - sample.timestamp).total_seconds() / 60
    record_data_age('a44', 'normalized', age_minutes)
```

**Impact:** Same as A75

**Rollback:** Remove same lines

**Status:** ✅ NORMALIZER OPERATIONAL

---

## SERVICES STATUS

### API Service
```bash
systemctl status energy-insights-nl-api
```
**Status:** ✅ **ACTIVE (running)** since 12:53:36 UTC
- 8 Gunicorn workers running
- No errors in logs
- Health endpoint responding: `curl http://localhost:8000/health` ✅

### Normalizers
```bash
sudo -u energy-insights-nl bash scripts/run_normalizers.sh
```
**Status:** ✅ **WORKING** (tested 13:01:28 UTC)
- Completed successfully
- No errors
- A75, A65, A44 all processed

### Metrics Endpoint
```bash
curl -s http://localhost:8000/metrics | grep pipeline_
```
**Status:** ❌ **EMPTY** (no pipeline metrics visible)

---

## ROOT CAUSE ANALYSIS

### Het Probleem

**Symptom:** Pipeline metrics not appearing in `/metrics` endpoint

**Root Cause:** Prometheus + Gunicorn multi-worker architecture mismatch

**Technical Details:**

1. **Gunicorn Multi-Worker Setup:**
   - API runs with 8 workers (independent Python processes)
   - Each worker has its own memory space
   - Prometheus `Gauge` metrics are **per-process**, not shared

2. **Normalizer Execution:**
   - Normalizers run as separate Python scripts via systemd timer
   - They import `synctacles_db.metrics` and update metrics
   - Metrics are updated in **normalizer process memory**

3. **Metrics Endpoint:**
   - `/metrics` endpoint served by Gunicorn workers
   - Workers have **different memory** than normalizer processes
   - Workers never see the metrics updates from normalizers

**Diagram:**
```
┌──────────────────┐
│ Normalizer       │  Updates metrics in Process A memory
│ Process (A)      │  ✅ record_normalizer_success('a75')
└──────────────────┘

┌──────────────────┐
│ Gunicorn Worker 1│  Has own metrics (Process B memory)
│ (Process B)      │  ❌ Doesn't see Process A updates
└──────────────────┘

┌──────────────────┐
│ Gunicorn Worker 2│  Has own metrics (Process C memory)
│ (Process C)      │  ❌ Doesn't see Process A updates
└──────────────────┘
```

---

## STANDARD SOLUTIONS

### Option 1: Prometheus Multiprocess Mode (prometheus_client)

**How it works:**
- Use `prometheus_client.multiprocess` module
- Metrics written to shared directory on disk
- All processes write to same files
- `/metrics` endpoint aggregates from disk

**Implementation:**
```python
# In metrics.py
from prometheus_client import CollectorRegistry, multiprocess, generate_latest
import os

# Set environment variable for multiprocess mode
os.environ['PROMETHEUS_MULTIPROC_DIR'] = '/tmp/prometheus_multiproc'

# Use multiprocess-safe registry
registry = CollectorRegistry()
multiprocess.MultiProcessCollector(registry)
```

**Pros:**
- Official Prometheus solution
- Works with Gunicorn multi-worker
- Metrics persist across worker restarts

**Cons:**
- Requires shared temp directory
- More complex setup
- File I/O overhead

**Effort:** 1-2 hours

---

### Option 2: Dedicated Metrics Updater Service

**How it works:**
- New systemd service/timer
- Runs every 1-5 minutes
- Queries database for latest data
- Updates metrics via HTTP endpoint or shared state

**Implementation:**
```bash
# scripts/update_pipeline_metrics.py
# Queries database, updates metrics

# systemd timer: energy-insights-nl-metrics-updater.timer
# Runs every 1 minute
```

**Pros:**
- Clean separation of concerns
- Easy to test independently
- Can add more complex logic

**Cons:**
- Additional service to manage
- 1-5 minute delay in metrics updates
- Still needs multiprocess solution OR dedicated metrics server

**Effort:** 2-3 hours

---

### Option 3: Grafana Direct Database Queries (Simplest)

**How it works:**
- Skip Prometheus metrics entirely for pipeline data
- Grafana queries PostgreSQL directly
- Use Grafana's PostgreSQL data source

**Implementation:**
```sql
-- Example Grafana query for normalizer age
SELECT
  'a75' as source,
  EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_minutes
FROM norm_entso_e_a75;
```

**Pros:**
- ✅ Works immediately (no code changes needed)
- ✅ Real-time data (no caching)
- ✅ Simple to implement
- ✅ Can still add Prometheus metrics later

**Cons:**
- Database queries from Grafana
- Not following Prometheus best practice
- Harder to add alerts (but possible with Grafana alerts)

**Effort:** 30-60 minutes (just dashboard creation)

---

## CC'S ASSESSMENT

### Recommended: Option 3 (Grafana Direct Queries)

**Reasoning:**

1. **Immediate Value:**
   - Dashboard working in < 1 hour
   - Would have caught today's normalizer bug instantly

2. **Pragmatic:**
   - Avoid over-engineering
   - Prometheus metrics are "nice to have" not required
   - Can migrate to Prometheus later if needed

3. **Production Ready:**
   - Grafana PostgreSQL data source is battle-tested
   - Many companies use this approach
   - Simpler troubleshooting (just SQL, no metrics layer)

4. **Current Code:**
   - Can **revert all changes** (no harm done)
   - Or **keep changes** for future Prometheus migration
   - Both options viable

---

## DECISION REQUIRED

**CAI, please choose:**

### Option A: Revert All Changes + Grafana Direct Queries ✅ RECOMMENDED

**Actions:**
1. Revert all code changes (rollback instructions above)
2. Delete `synctacles_db/metrics.py`
3. Keep Grafana cleanup (no reason to undo)
4. Create dashboard with PostgreSQL queries
5. Document in MONITORING.md

**Pros:** Clean slate, simplest solution
**Cons:** Wasted 2 hours of work (but learned Gunicorn issue)

---

### Option B: Keep Changes + Implement Prometheus Multiprocess

**Actions:**
1. Keep all current changes
2. Add multiprocess mode to metrics.py
3. Configure shared metrics directory
4. Test metrics endpoint
5. Create Grafana dashboard with Prometheus

**Pros:** Follows Prometheus best practices, future-proof
**Cons:** 1-2 more hours work, more complexity

---

### Option C: Keep Changes + Grafana Direct Queries (Hybrid)

**Actions:**
1. Keep all current changes
2. Create dashboard with PostgreSQL queries NOW
3. Add TODO issue for Prometheus multiprocess later
4. Migrate dashboard to Prometheus when ready

**Pros:** Best of both worlds, incremental improvement
**Cons:** Technical debt (two solutions for same problem)

---

## IMPACT ANALYSIS

### If We Revert All Changes (Option A)

**Safe to revert:**
- ✅ All changes are additive (no deletions)
- ✅ API tested and working
- ✅ Normalizers tested and working
- ✅ No user-facing impact
- ✅ No data loss

**Revert steps:**
```bash
# 1. Remove metrics.py
rm /opt/github/synctacles-api/synctacles_db/metrics.py

# 2. Revert main.py changes
# Edit main.py: remove Gauge import, remove metrics import

# 3. Revert normalizer changes (A75, A65, A44)
# Edit each: remove metrics import, remove record_* calls

# 4. Restart API
sudo systemctl restart energy-insights-nl-api

# 5. Test normalizers
sudo -u energy-insights-nl bash scripts/run_normalizers.sh

# 6. Verify API health
curl http://localhost:8000/health
```

**Estimated revert time:** 15-20 minutes

---

### If We Keep Changes (Option B or C)

**No immediate action needed:**
- ✅ Code is stable
- ✅ No errors
- ✅ Metrics just not visible (harmless)
- ✅ Can implement fix incrementally

---

## FILES MODIFIED SUMMARY

| File | Status | Lines Changed | Rollback Needed? |
|------|--------|---------------|------------------|
| synctacles_db/metrics.py | NEW | +56 | Yes (delete) |
| synctacles_db/api/main.py | MODIFIED | +2 | Yes (remove imports) |
| synctacles_db/normalizers/normalize_entso_e_a75.py | MODIFIED | +7 | Yes (remove metrics calls) |
| synctacles_db/normalizers/normalize_entso_e_a65.py | MODIFIED | +7 | Yes (remove metrics calls) |
| synctacles_db/normalizers/normalize_entso_e_a44.py | MODIFIED | +8 | Yes (remove metrics calls) |
| /etc/grafana | DELETED | -709 MB | No (intentional) |
| /var/lib/grafana | DELETED | - | No (intentional) |
| /var/log/grafana | DELETED | - | No (intentional) |

**Total code changes:** 80 lines (56 new file + 24 edits)
**All files:** NOT YET COMMITTED (changes only on disk)

---

## GIT STATUS

```bash
cd /opt/github/synctacles-api
sudo -u energy-insights-nl git status
```

**Result:**
```
 M synctacles_db/api/main.py
 M synctacles_db/normalizers/normalize_entso_e_a44.py
 M synctacles_db/normalizers/normalize_entso_e_a65.py
 M synctacles_db/normalizers/normalize_entso_e_a75.py
?? synctacles_db/metrics.py
```

**Status:** ⚠️ **NOT COMMITTED** - changes only on disk, easy to revert

---

## NEXT ACTIONS

### If Option A (Revert + Direct Queries):
1. ⏸️ CC reverts all code changes
2. ⏸️ CC creates Grafana dashboard with PostgreSQL queries
3. ⏸️ CC commits only documentation updates
4. ⏸️ Dashboard operational in 1 hour

### If Option B (Keep + Prometheus Multiprocess):
1. ⏸️ CC implements multiprocess mode
2. ⏸️ CC tests metrics endpoint
3. ⏸️ CC creates Grafana dashboard with Prometheus
4. ⏸️ CC commits all changes
5. ⏸️ Dashboard operational in 2-3 hours

### If Option C (Keep + Direct Queries + TODO):
1. ⏸️ CC creates Grafana dashboard with PostgreSQL queries
2. ⏸️ CC commits all changes + TODO issue for Prometheus
3. ⏸️ Dashboard operational in 1 hour
4. ⏸️ Prometheus migration later (optional)

---

## QUESTIONS FOR CAI

1. **Which option do you prefer?** (A, B, or C)

2. **Grafana login:** Will you provide password when needed? (Yes/No)

3. **Priority:** Do we need dashboard TODAY or can it wait for proper Prometheus implementation?

4. **Technical debt:** OK with hybrid solution (Option C) or prefer clean solution?

---

## RECOMMENDATION

**CC's recommendation:** **Option A** (Revert + Direct Queries)

**Reasoning:**
- Fastest to working dashboard (1 hour vs 2-3 hours)
- Simpler architecture (less to maintain)
- Proven approach (many use Grafana → PostgreSQL)
- Can always add Prometheus later if needed
- Current code was good learning but not production-ready

**Alternative:** **Option C** if you want to keep the metrics code for future use

---

*Template versie: 1.0*
*Blocked at: 2026-01-08 13:15 UTC*
*Decision needed: Option A, B, or C*
