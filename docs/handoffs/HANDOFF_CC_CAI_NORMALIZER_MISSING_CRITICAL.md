# HANDOFF: CC → CAI

**Datum:** 2026-01-08
**Van:** CC (Claude Code)
**Naar:** CAI
**Prioriteit:** CRITICAL
**Type:** Architecture Review & Legal Compliance

---

## EXECUTIVE SUMMARY

During Grafana dashboard implementation, discovered that `run_normalizers.sh` only processes **2 out of 3** ENTSO-E data sources. This means **A65 (Load) and A75 (Generation) raw data is potentially being served to customers without normalization**, which may violate ENTSO-E license terms and creates data quality issues.

**Immediate Question for CAI:** Is the API serving raw or normalized data to customers? If raw, this is a critical license compliance and data quality issue.

---

## PROBLEM STATEMENT

### Discovery Timeline

**18:30 UTC - User Question:**
> "Je gaf net aan dat A75 77 min oude data had. Nu laat het dashboard 1.96 hours oud zien. Hoe kan dit?"

**Investigation Result:**
```sql
-- Current Status (2026-01-08 18:29 UTC)
raw_a75:        17:15 (74 min old)  ✅ Being updated
normalized_a75: 16:30 (119 min old) ❌ NOT being updated
```

**Root Cause Found:**
```bash
# /opt/energy-insights-nl/app/scripts/run_normalizers.sh (lines 28-29)
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44  # Only A44
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices       # Price processing

# MISSING:
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65  # Load data
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75  # Generation data
```

**Impact:**
- A44 (Prices): ✅ Being normalized every 15 min → FRESH
- A65 (Load): ❌ NOT being normalized → becomes STALE/UNAVAILABLE
- A75 (Generation): ❌ NOT being normalized → becomes STALE/UNAVAILABLE

---

## CRITICAL QUESTIONS FOR CAI

### 1. Is the API Serving Raw or Normalized Data?

**Need to verify:** Do customer-facing API endpoints query `raw_entso_e_*` or `norm_entso_e_*` tables?

**If Raw Tables:**
- ⚠️ **Legal Risk:** ENTSO-E license violation (no transformation/added value)
- ⚠️ **Data Quality Risk:** Forecast data mixed with historical (negative age bug)
- ⚠️ **No Quality Flags:** Provisional vs validated data not distinguished

**If Normalized Tables:**
- ⚠️ **Stale Data Risk:** A65/A75 data becomes 2+ hours old
- ⚠️ **Incomplete Data:** Customers get outdated information
- ⚠️ **SLA Breach:** If customers expect fresh data

### 2. Is This Intentional or a Bug?

**Possible Explanations:**

**A) Bug - Normalizers Missing from Script**
- A65/A75 normalizers were never added to `run_normalizers.sh`
- System is broken, raw data accumulates without processing
- Fix: Add missing normalizers to script

**B) Intentional - Different Architecture**
- A65/A75 have separate normalization pipeline (not timer-based?)
- Raw data is meant to be served directly for these sources?
- Need documentation of intended architecture

**C) Work in Progress - Incomplete Implementation**
- A65/A75 normalizers exist but aren't in production yet
- System is in transition state
- Need completion of implementation

### 3. What Are ENTSO-E License Requirements?

**Key Question:** Can we legally serve raw ENTSO-E data to commercial customers?

**ENTSO-E Transparency Platform - Typical Terms:**
1. Data must be **transformed/aggregated** for commercial resale
2. Cannot simply **mirror/replicate** ENTSO-E API
3. Must add **value** (normalization, quality checks, gap filling)
4. **Attribution** required

**Need CAI to:**
- Review actual ENTSO-E license agreement
- Confirm if current implementation complies
- Document compliance requirements in code

---

## TECHNICAL EVIDENCE

### File: scripts/run_normalizers.sh

```bash
#!/bin/bash
# Run all data normalizers
# Normalizes raw data to normalized tables

set -e

INSTALL_PATH="${INSTALL_PATH:-/opt/energy-insights-nl}"
VENV_PATH="${VENV_PATH:-${INSTALL_PATH}/venv}"
APP_PATH="${APP_PATH:-${INSTALL_PATH}/app}"
LOG_PATH="${LOG_PATH:-/var/log/energy-insights}"

# Source environment variables
if [ -f "/opt/.env" ]; then
    set -a
    source /opt/.env
    set +a
fi

# Create log directory if needed
mkdir -p "${LOG_PATH}"

# Python path
PYTHON="${VENV_PATH}/bin/python3"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starting normalizer batch..."

# Run normalizers (they handle failures internally)
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Normalizer batch complete"
```

**Analysis:**
- Line 28: Only `normalize_entso_e_a44` (Prices)
- Line 29: Only `normalize_prices` (Price post-processing)
- **Missing:** `normalize_entso_e_a65` (Load)
- **Missing:** `normalize_entso_e_a75` (Generation)

### Normalizer Execution Logs

```bash
# Logs show timer runs successfully every 15 min
Jan 08 18:23:50 ENIN-NL run_normalizers.sh[225846]: [2026-01-08 18:23:50] Starting normalizer batch...
Jan 08 18:23:52 ENIN-NL run_normalizers.sh[225846]: [2026-01-08 18:23:52] Normalizer batch complete

# But NO indication of which sources processed
# NO errors or warnings about skipped sources
```

**Problem:** Silent failure - script succeeds but doesn't process all data.

### Database Evidence

```sql
-- Evidence of growing gap between raw and normalized data

-- A75 (Generation) - 45 minute gap
SELECT
    (SELECT MAX(timestamp) FROM raw_entso_e_a75 WHERE timestamp <= NOW()) as raw_latest,
    (SELECT MAX(timestamp) FROM norm_entso_e_a75 WHERE timestamp <= NOW()) as norm_latest,
    EXTRACT(EPOCH FROM (
        (SELECT MAX(timestamp) FROM raw_entso_e_a75 WHERE timestamp <= NOW()) -
        (SELECT MAX(timestamp) FROM norm_entso_e_a75 WHERE timestamp <= NOW())
    ))/60 as gap_minutes;

-- Result:
--   raw_latest: 2026-01-08 17:15:00
--   norm_latest: 2026-01-08 16:30:00
--   gap_minutes: 45

-- A65 (Load) - Likely similar gap (need verification)
-- A44 (Prices) - No gap (being normalized) ✅
```

### Manual Normalizer Test

**Proof normalizers work when called directly:**

```bash
# Manual run of A75 normalizer (executed by CC at ~17:45 UTC)
sudo -u energy-insights-nl bash -c 'set -a && source /opt/.env && set +a && \
  cd /opt/energy-insights-nl/app && \
  source /opt/energy-insights-nl/venv/bin/activate && \
  /opt/energy-insights-nl/venv/bin/python -m synctacles_db.normalizers.normalize_entso_e_a75'

# Result: SUCCESS
# Normalized data updated from 11:45 → 16:30 (processed ~6 hours of backlog)
```

**Conclusion:** Normalizer scripts exist and work - they're just not being called by the timer.

---

## ARCHITECTURE QUESTIONS

### Current Pipeline (As Understood)

```
ENTSO-E API
    ↓
Collectors (every 15 min)
    ↓
raw_entso_e_a44 / raw_entso_e_a65 / raw_entso_e_a75
    ↓
Importers (every 15 min)
    ↓
raw_entso_e_* tables populated
    ↓
Normalizers (every 15 min) ← **BROKEN HERE**
    ↓ (only A44 processed)
norm_entso_e_a44 ✅ / norm_entso_e_a65 ❌ / norm_entso_e_a75 ❌
    ↓
API Endpoints
    ↓
Customers
```

### Critical Unknown: Which Tables Do API Endpoints Use?

**Need to audit all API endpoints:**

```bash
# Find all endpoints that query ENTSO-E data
grep -r "raw_entso_e" /opt/energy-insights-nl/app/synctacles_db/api/
grep -r "norm_entso_e" /opt/energy-insights-nl/app/synctacles_db/api/
```

**Possible Scenarios:**

**Scenario A: API Uses Normalized Tables (Expected)**
```python
# Good - serving transformed data
SELECT * FROM norm_entso_e_a75 WHERE timestamp BETWEEN ...
```
→ **Impact:** Customers get stale data (2+ hours old)
→ **Risk Level:** MEDIUM (data quality issue)

**Scenario B: API Uses Raw Tables (Problematic)**
```python
# Bad - serving raw ENTSO-E data
SELECT * FROM raw_entso_e_a75 WHERE timestamp BETWEEN ...
```
→ **Impact:** Potential license violation + forecast data leakage
→ **Risk Level:** HIGH (legal + data quality)

**Scenario C: API Uses Both (Confusing)**
```python
# Inconsistent - some endpoints normalized, others raw
SELECT * FROM norm_entso_e_a44 ...  # Prices normalized
SELECT * FROM raw_entso_e_a75 ...   # Generation raw
```
→ **Impact:** Inconsistent data quality across endpoints
→ **Risk Level:** MEDIUM-HIGH (architecture confusion)

---

## DATA QUALITY IMPLICATIONS

### What Normalization Provides

Based on code in `synctacles_db/normalizers/`, normalization typically:

1. **Filters Forecast Data**
   - Removes future timestamps (WHERE timestamp <= NOW())
   - Prevents negative age values in health checks
   - Ensures historical queries return only historical data

2. **Quality Validation**
   - Marks provisional vs validated data (quality_status field)
   - Identifies gaps requiring backfill (needs_backfill flag)
   - Validates data ranges and units

3. **Deduplication**
   - Handles overlapping ENTSO-E requests
   - Uses UNIQUE constraints on (timestamp, country)
   - Prevents duplicate records

4. **Timestamp Standardization**
   - Ensures UTC timezone consistency
   - 15-minute granularity alignment
   - Handles DST transitions

5. **Added Metadata**
   - last_updated timestamp (when normalized)
   - data_source field ('ENTSO-E')
   - Processing audit trail

### What Happens Without Normalization

**If API serves raw data:**

1. ❌ **Forecast data mixed in**
   - Historical queries return future timestamps
   - Causes negative age calculations
   - Confuses analytics/visualizations

2. ❌ **No quality flags**
   - Cannot distinguish provisional vs validated
   - Customers don't know data reliability
   - No indication of backfill needs

3. ❌ **Possible duplicates**
   - Overlapping collector runs create duplicates
   - Customers must deduplicate themselves
   - Inconsistent record counts

4. ❌ **No processing audit**
   - Cannot track when data was last validated
   - Difficult to debug data issues
   - No provenance tracking

---

## LEGAL & COMPLIANCE CONSIDERATIONS

### ENTSO-E License (Typical Requirements)

**Disclaimer:** CC is not a lawyer. CAI must review actual license agreement.

**Typical ENTSO-E Transparency Platform Terms:**

1. **Attribution Required**
   - Must credit ENTSO-E as data source
   - Cannot claim data as own creation
   - ✅ Likely compliant (data_source field)

2. **Transformation Required for Commercial Use**
   - Cannot simply mirror/replicate ENTSO-E API
   - Must add value through processing/analysis
   - ❓ **Unknown compliance status**

3. **No Warranty/Liability**
   - ENTSO-E provides data "as is"
   - Downstream users assume liability
   - ✅ Standard commercial practice

4. **Rate Limiting Respect**
   - Must respect API rate limits
   - Cannot overload ENTSO-E infrastructure
   - ✅ Likely compliant (15-min intervals)

### Commercial Resale Implications

**If serving raw data to paying customers:**

- ⚠️ May constitute "reselling" without transformation
- ⚠️ Customers could go directly to ENTSO-E (no value add)
- ⚠️ Potential breach of terms → loss of API access
- ⚠️ Legal liability if ENTSO-E pursues enforcement

**If serving normalized data:**

- ✅ Clear value add (transformation, validation, quality flags)
- ✅ Justifies commercial pricing
- ✅ Likely compliant with typical license terms
- ✅ Protects against legal issues

---

## PROPOSED INVESTIGATION PLAN

### Phase 1: Architecture Verification (CAI - 1 hour)

**Step 1: Audit API Endpoints**
```bash
# Find all ENTSO-E data queries
cd /opt/energy-insights-nl/app
grep -r "raw_entso_e" synctacles_db/api/ > /tmp/raw_queries.txt
grep -r "norm_entso_e" synctacles_db/api/ > /tmp/norm_queries.txt

# Analyze which endpoints use which tables
# Document findings
```

**Step 2: Review Architecture Documentation**
```bash
# Check if there's existing docs on normalization strategy
find docs/ -name "*normaliz*" -o -name "*architecture*"
```

**Step 3: Check for Alternative Normalization Pipelines**
```bash
# Maybe A65/A75 have different processing?
find /opt/energy-insights-nl -name "*a65*" -o -name "*a75*"
systemctl list-timers | grep -E "a65|a75"
```

### Phase 2: License Review (CAI - 30 min)

**Step 1: Locate ENTSO-E License Agreement**
```bash
# Check for stored license/terms
find /opt -name "*license*" -o -name "*terms*" | grep -i entso
find docs/ -name "*legal*" -o -name "*compliance*"
```

**Step 2: Review Terms**
- Can raw data be served commercially?
- What transformation is required?
- Are quality flags mandatory?
- What attribution is needed?

**Step 3: Document Requirements**
```markdown
# Create: docs/legal/ENTSO_E_LICENSE_REQUIREMENTS.md
- License terms summary
- Compliance checklist
- Implementation requirements
```

### Phase 3: Decision & Implementation (CAI - 2-4 hours)

**Decision Tree:**

```
Does API serve raw data?
├─ YES → CRITICAL FIX NEEDED
│   ├─ Add A65/A75 normalizers to run_normalizers.sh
│   ├─ Update API endpoints to use norm_* tables
│   ├─ Backfill normalized data
│   └─ Add compliance tests
│
└─ NO → Normalized only
    ├─ Add A65/A75 normalizers to run_normalizers.sh
    ├─ Backfill normalized data
    ├─ Add gap monitoring alerts
    └─ Document why they were missing
```

---

## TEMPORARY WORKAROUND (IF NEEDED)

**If CAI confirms this is critical and needs immediate fix:**

### Quick Fix (10 minutes)

```bash
# 1. Add missing normalizers to script
sudo -u energy-insights-nl nano /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# Add these lines after line 29:
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75

# 2. Copy to production
sudo cp /opt/github/synctacles-api/scripts/run_normalizers.sh \
        /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# 3. Run manually to process backlog
sudo -u energy-insights-nl bash /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# 4. Verify next timer run processes all sources
sudo journalctl -u energy-insights-nl-normalizer.service -f
```

**However:** CC recommends CAI review architecture first before applying fixes.

---

## MONITORING IMPROVEMENTS NEEDED

Regardless of root cause resolution, add these safeguards:

### 1. Gap Monitoring

```python
# Add to prometheus metrics (synctacles_db/api/routes/pipeline.py)
raw_normalized_gap_minutes = Gauge(
    'pipeline_raw_normalized_gap_minutes',
    'Minutes between raw and normalized data timestamps',
    ['source']
)

# In /metrics endpoint:
for source in ['a44', 'a65', 'a75']:
    raw_age = get_raw_data_age(db, source)
    norm_age = get_normalized_data_age(db, source)
    gap = norm_age - raw_age
    raw_normalized_gap_minutes.labels(source=source).set(gap)
```

### 2. Alert Rules

```yaml
# Add to /opt/monitoring/prometheus/alerts.yml
- alert: NormalizerLagging
  expr: pipeline_raw_normalized_gap_minutes > 30
  for: 15m
  labels:
    severity: critical
  annotations:
    summary: "Normalizer {{ $labels.source }} lagging >30 min"
    description: "Gap between raw and normalized data is {{ $value }} minutes"
```

### 3. Logging Enhancement

```bash
# Improve run_normalizers.sh logging
echo "[$(date)] Processing A44..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44
echo "[$(date)] A44 complete"

echo "[$(date)] Processing A65..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65
echo "[$(date)] A65 complete"

echo "[$(date)] Processing A75..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75
echo "[$(date)] A75 complete"
```

---

## QUESTIONS FOR CAI TO ANSWER

1. **Architecture:**
   - [ ] Do customer API endpoints use raw_* or norm_* tables?
   - [ ] Is there a separate normalization pipeline for A65/A75?
   - [ ] Why were A65/A75 not included in run_normalizers.sh?

2. **Legal:**
   - [ ] What are actual ENTSO-E license requirements?
   - [ ] Can we serve raw data commercially?
   - [ ] What transformations are required?

3. **Implementation:**
   - [ ] Should A65/A75 normalizers be added to timer script?
   - [ ] Is there missing documentation on normalization strategy?
   - [ ] Are there other missing normalizers we don't know about?

4. **Priority:**
   - [ ] Is this a critical bug requiring immediate fix?
   - [ ] Is this working as intended (different architecture)?
   - [ ] Can this wait for planned refactoring?

---

## DELIVERABLES EXPECTED FROM CAI

1. **Architecture Analysis Report**
   - Document which tables API uses (raw vs normalized)
   - Explain intended normalization architecture
   - Clarify if current state is bug or design

2. **License Compliance Review**
   - ENTSO-E license requirements summary
   - Compliance assessment of current implementation
   - Recommendations for full compliance

3. **Implementation Decision**
   - Fix run_normalizers.sh (add A65/A75)?
   - Update API endpoints (use norm_* tables)?
   - Both?
   - Neither (different architecture)?

4. **Documentation Updates**
   - Architecture diagrams (collector → importer → normalizer → API)
   - Normalization requirements per data source
   - Compliance checklist

5. **Monitoring Improvements**
   - Gap monitoring metrics
   - Alert rules for normalizer lag
   - Enhanced logging

---

## IMPACT ASSESSMENT

### If This Is A Bug (Normalizers Missing)

**Severity:** HIGH to CRITICAL

**Customer Impact:**
- Stale data (2+ hours old) for A65/A75 endpoints
- Possible forecast data leakage
- Inconsistent data quality across sources

**Legal Impact:**
- Potential ENTSO-E license violation (if serving raw commercially)
- Risk of losing API access
- Possible legal liability

**Business Impact:**
- SLA breaches if customers expect fresh data
- Reputation damage if data quality issues discovered
- Competitive disadvantage vs providers with proper normalization

### If This Is Intentional (Different Architecture)

**Severity:** LOW to MEDIUM

**Technical Debt:**
- Confusing architecture (why is A44 different?)
- Lack of documentation causes misunderstanding
- Risk of similar issues with future data sources

**Recommendation:**
- Document intended architecture clearly
- Consider standardizing all sources
- Add tests to prevent regression

---

## TIMELINE

**Created:** 2026-01-08 18:45 UTC
**Priority:** CRITICAL (pending CAI assessment)

**Requested Response:**
- Initial assessment: Within 24 hours
- Full investigation: Within 1 week
- Implementation decision: Within 2 weeks

**Blockers:**
- CC lacks context on intended architecture
- CC cannot assess legal compliance
- CC should not modify production pipeline without CAI approval

---

## REFERENCE LINKS

**Related Handoffs:**
- `HANDOFF_CC_CAI_GRAFANA_DASHBOARD_COMPLETE.md` - Dashboard implementation that discovered this issue
- `HANDOFF_CC_CAI_DATA_FRESHNESS_FIXED.md` - Earlier investigation of data freshness issues

**Key Files:**
- `/opt/energy-insights-nl/app/scripts/run_normalizers.sh` - Normalizer timer script (missing A65/A75)
- `/opt/energy-insights-nl/app/synctacles_db/normalizers/normalize_entso_e_*.py` - Normalizer implementations
- `/opt/energy-insights-nl/app/synctacles_db/api/routes/` - API endpoint implementations

**Database Tables:**
- `raw_entso_e_a44` / `raw_entso_e_a65` / `raw_entso_e_a75` - Raw ENTSO-E data
- `norm_entso_e_a44` / `norm_entso_e_a65` / `norm_entso_e_a75` - Normalized data

---

*Prepared by CC (Claude Code)*
*Date: 2026-01-08 18:45 UTC*
*Template Version: 2.0 (Critical Issue Investigation)*

**⚠️ AWAITING CAI REVIEW - DO NOT IMPLEMENT FIXES WITHOUT APPROVAL ⚠️**
