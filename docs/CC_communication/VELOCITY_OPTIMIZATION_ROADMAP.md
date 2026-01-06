# 🚀 VELOCITY & EFFICIENCY OPTIMALISATIE ROADMAP

**Datum:** 2026-01-06  
**Versie:** 1.0  
**Status:** POST-LAUNCH implementatie (niet pre-launch)  
**Planning:** V1.1 (Feb 2026)

---

## 📋 EXECUTIVE SUMMARY

**Huidige velocity:** 6x sneller dan industry baseline (exceptioneel)  
**Potentiële velocity:** 10-12x met systematische optimalisaties  
**Timing:** IMPLEMENTEER NA V1 LAUNCH (niet voor)

**Waarom na launch:**
1. Launch window sluit (prioriteit #1)
2. Huidige 6x velocity is voldoende voor launch
3. Optimization costs 16 uur, saves slechts 8 uur pre-launch = negatieve ROI
4. Post-launch production data maakt optimalisatie effectiever

**Post-launch planning:**
- Week 1 (Feb 1-7): Stabilisatie + production metrics verzamelen
- Week 2-3 (Feb 8-21): Velocity optimization sprint
- Week 4+ (Feb 22+): Germany expansion met geoptimaliseerde velocity

---

## 🎯 CATEGORIEËN & PRIORITEIT

### Tier 1: QUICK WINS (Hoogste ROI)
**Implementatie:** Week 2-3 Feb (8 uur totaal)  
**Impact:** 2-3x extra velocity boost  
**Return:** Proven patterns, low risk

1. Template-driven development (2 uur)
2. Pre-commit validation hooks (2 uur)
3. Automated deployment validation (1 uur)
4. Hot-reload development (1 uur)
5. Smart test selection (1 uur)

### Tier 2: WORKFLOW OPTIMALISATIES
**Implementatie:** Week 3-4 Feb (12 uur totaal)  
**Impact:** 30-50% efficiency gain  
**Return:** Process improvements, cultural change

6. Parallel sprint execution (0 uur - planning only)
7. Batch similar tasks (0 uur - discipline)
8. Context-preserving summaries (2 uur template)
9. Decision log (ADR process) (2 uur setup)
10. Track cycle time metrics (4 uur dashboard)

### Tier 3: TECHNICAL OPTIMALISATIES
**Implementatie:** March 2026 (20 uur totaal)  
**Impact:** 10-20% performance gains  
**Return:** Data-driven after production metrics

11. Database query optimization (8 uur)
12. Collector template inheritance (6 uur)
13. Automated fallback testing (4 uur)
14. Database reset script (2 uur)

---

## 💎 TIER 1: QUICK WINS (Hoogste ROI)

### 1. Template-Driven Development
**Geschatte tijd:** 2 uur  
**Impact:** 5x sneller nieuwe collectors/normalizers  
**Status:** Post-launch (Feb 8-14)

#### Implementatie
```bash
#!/bin/bash
# scripts/generate-collector.sh

if [ -z "$1" ]; then
    echo "Usage: ./generate-collector.sh <name>"
    echo "Example: ./generate-collector.sh entso_e_a46_reserves"
    exit 1
fi

NAME=$1
CLASS_NAME=$(echo "$NAME" | sed 's/_/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) tolower(substr($i,2))}1' | sed 's/ //g')

cat > "synctacles_db/collectors/${NAME}.py" <<EOF
"""
Collector for ${NAME}
Auto-generated on $(date +%Y-%m-%d)
"""
from synctacles_db.core.logging import get_logger
import time
import asyncio

_LOGGER = get_logger(__name__)

class ${CLASS_NAME}Collector:
    """Collector for ${NAME} data source."""
    
    def __init__(self):
        self.name = "${NAME}"
    
    async def collect(self):
        """Main collection entry point."""
        _LOGGER.info(f"{self.name} starting")
        start_time = time.time()
        
        try:
            # TODO: Implement collection logic
            data = await self._fetch_data()
            
            elapsed = time.time() - start_time
            _LOGGER.info(f"{self.name} completed in {elapsed:.2f}s")
            return data
            
        except Exception as err:
            elapsed = time.time() - start_time
            _LOGGER.error(f"{self.name} failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
            raise
    
    async def _fetch_data(self):
        """Fetch data from source API."""
        # TODO: Implement actual fetch logic
        raise NotImplementedError("Implement _fetch_data method")

if __name__ == '__main__':
    collector = ${CLASS_NAME}Collector()
    asyncio.run(collector.collect())
EOF

echo "✅ Generated synctacles_db/collectors/${NAME}.py"
echo "Next steps:"
echo "  1. Edit ${NAME}.py and implement _fetch_data()"
echo "  2. Run: python synctacles_db/collectors/${NAME}.py"
echo "  3. Create corresponding importer and normalizer"
```

#### Ook genereren
- `scripts/generate-importer.sh` (similar pattern)
- `scripts/generate-normalizer.sh` (similar pattern)
- `scripts/generate-api-endpoint.sh` (FastAPI route template)

#### Testing
```bash
# Test generator
./scripts/generate-collector.sh test_collector
python synctacles_db/collectors/test_collector.py
# Should run without syntax errors
```

---

### 2. Pre-Commit Validation Hooks
**Geschatte tijd:** 2 uur  
**Impact:** 90% bugs caught voor commit  
**Status:** Post-launch (Feb 8-14)

#### Implementatie
```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "🔍 Running pre-commit validation..."

# 1. Python syntax check
echo "  → Checking Python syntax..."
python_files=$(git diff --cached --name-only --diff-filter=ACM | grep '\.py$')
for file in $python_files; do
    python -m py_compile "$file" || {
        echo "❌ Syntax error in $file"
        exit 1
    }
done

# 2. No hardcoded credentials
echo "  → Checking for hardcoded credentials..."
if git diff --cached | grep -E "(synctacles@|postgresql://[a-z_]+@)" | grep -v "template\|example"; then
    echo "❌ Hardcoded credentials detected"
    echo "   Use config.settings.DATABASE_URL instead"
    exit 1
fi

# 3. No TODO markers in production code
echo "  → Checking for TODO markers..."
if git diff --cached synctacles_db/ | grep -E "TODO|FIXME" | grep -v "tests/"; then
    echo "⚠️  TODO markers found in production code"
    echo "   Move to GitHub issues or remove before commit"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 4. Required imports in collectors/normalizers
echo "  → Checking required imports..."
for file in $python_files; do
    if [[ $file == synctacles_db/collectors/* ]] || [[ $file == synctacles_db/normalizers/* ]]; then
        if ! grep -q "from synctacles_db.core.logging import get_logger" "$file"; then
            echo "❌ Missing required import in $file"
            echo "   Add: from synctacles_db.core.logging import get_logger"
            exit 1
        fi
    fi
done

# 5. No secrets in .env files
echo "  → Checking for committed secrets..."
if git diff --cached --name-only | grep -E "\.env$" | grep -v "\.env\.example"; then
    echo "❌ Attempting to commit .env file"
    echo "   Add to .gitignore: echo '.env' >> .gitignore"
    exit 1
fi

echo "✅ All pre-commit checks passed"
exit 0
```

#### Installation
```bash
# Make hook executable
chmod +x .git/hooks/pre-commit

# Test it
echo "# TODO: test" > test_file.py
git add test_file.py
git commit -m "test"  # Should trigger warning
```

---

### 3. Automated Deployment Validation (Optimized)
**Geschatte tijd:** 1 uur  
**Impact:** 20x sneller deployment verification (10 min → 30 sec)  
**Status:** Sprint 1 implementation, optimization post-launch

#### Optimized Implementation
```bash
#!/bin/bash
# scripts/post-deploy-verify.sh (optimized version)

set -e

echo "🔍 Post-Deploy Verification (Optimized)"
START_TIME=$(date +%s)

# 1. Health check (quick fail if API down)
echo "  → API health check..."
curl -sf http://localhost:8000/health > /dev/null || {
    echo "❌ API health check failed"
    exit 1
}

# 2. All endpoints in parallel (not sequential!)
echo "  → Testing all endpoints (parallel)..."
ENDPOINTS=(
    "/api/v1/generation-mix"
    "/api/v1/load"
    "/api/v1/prices/today"
    "/api/v1/signals/is-green"
    "/api/v1/signals/should-charge"
    "/api/v1/signals/charge-speed"
)

for endpoint in "${ENDPOINTS[@]}"; do
    (curl -sf "http://localhost:8000$endpoint" > /dev/null || {
        echo "❌ Endpoint failed: $endpoint"
        exit 1
    }) &
done

# Wait for all parallel requests
wait

# 3. Database check (1 query instead of 5)
echo "  → Database connectivity & data freshness..."
psql -U energy_insights_nl -d energy_insights_nl -t -A -c "
SELECT 
    CASE 
        WHEN COUNT(*) FILTER (WHERE timestamp > NOW() - INTERVAL '1 hour') > 0 
        THEN 'OK' 
        ELSE 'STALE' 
    END as status
FROM norm_entso_e_a75
" | grep -q "OK" || {
    echo "❌ Database has stale data (> 1 hour old)"
    exit 1
}

# 4. Systemd timers active
echo "  → Systemd timers status..."
systemctl is-active energy-insights-nl-collector.timer > /dev/null || {
    echo "❌ Collector timer not active"
    exit 1
}

systemctl is-active energy-insights-nl-normalizer.timer > /dev/null || {
    echo "❌ Normalizer timer not active"
    exit 1
}

# 5. Recent collector runs successful
echo "  → Recent collector status..."
LAST_COLLECTOR=$(systemctl status energy-insights-nl-collector.service | grep "Active:" | grep -q "ago")
if [ $? -ne 0 ]; then
    echo "⚠️  Collector has not run recently"
fi

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

echo "✅ All deployment checks passed in ${ELAPSED}s"
exit 0
```

#### Integration in Deployment
```bash
# scripts/deploy.sh (add verification)
rsync -av ...
systemctl restart energy-insights-nl-api

# Wait for restart
sleep 5

# Verify deployment
./scripts/post-deploy-verify.sh || {
    echo "❌ Deployment verification failed - rolling back"
    ./scripts/rollback.sh
    exit 1
}

echo "✅ Deployment verified and successful"
```

---

### 4. Hot-Reload Development Environment
**Geschatte tijd:** 1 uur  
**Impact:** 24x sneller iteration (2 min → 5 sec)  
**Status:** Post-launch (Feb 8-14)

#### Implementatie
```bash
# scripts/dev-server.sh
#!/bin/bash

echo "🔥 Starting hot-reload development server..."

# API with auto-reload
uvicorn synctacles_db.api.main:app \
    --reload \
    --reload-dir synctacles_db \
    --reload-exclude "*.pyc" \
    --reload-exclude "__pycache__" \
    --log-level debug \
    --host 0.0.0.0 \
    --port 8000

# Alternative: watchdog for collectors/normalizers
# pip install watchdog[watchmedo]
# watchmedo auto-restart \
#     --patterns="*.py" \
#     --recursive \
#     --directory synctacles_db/collectors \
#     -- python synctacles_db/collectors/entso_e_a75_generation.py
```

#### Usage
```bash
# Development mode
./scripts/dev-server.sh

# Edit any file in synctacles_db/
# Server automatically restarts on save
# Test immediately without manual restart
```

---

### 5. Smart Test Selection (pytest-testmon)
**Geschatte tijd:** 1 uur  
**Impact:** 10x sneller CI (5 min → 30 sec)  
**Status:** Post-launch (Feb 8-14)

#### Implementatie
```bash
# Install pytest-testmon
pip install pytest-testmon

# Add to requirements-dev.txt
echo "pytest-testmon==2.1.0" >> requirements-dev.txt

# Configure pytest.ini
cat >> pytest.ini <<EOF

[pytest]
testmon = true
testmon_config_file = .testmondata
EOF

# First run (establishes baseline)
pytest --testmon

# Subsequent runs (only affected tests)
pytest --testmon  # Runs only tests affected by changed files
```

#### CI/CD Integration
```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Cache testmon data
        uses: actions/cache@v3
        with:
          path: .testmondata
          key: testmon-${{ github.sha }}
          restore-keys: testmon-
      
      - name: Run smart tests
        run: pytest --testmon
```

---

## 🔧 TIER 2: WORKFLOW OPTIMALISATIES

### 6. Parallel Sprint Execution
**Geschatte tijd:** 0 uur (planning only)  
**Impact:** 25% sneller (4 weken → 3 weken)  
**Status:** APPLY NOW (pre-launch)

#### Current Planning
```
Week 1: Sprint 1 (Security)       - solo
Week 2: Sprint 2 (Monitoring)     - solo
Week 3: Sprint 3 (Testing)        - solo
Week 4: Sprint 4 (Database)       - solo
```

#### Optimized Planning
```
Week 1: Sprint 1 (Security)                    - solo [BLOCKER]
Week 2: Sprint 2 (Monitoring) + Sprint 3 start - parallel
Week 3: Sprint 3 finish + Sprint 4 start       - parallel
Week 4: Sprint 4 finish + buffer               - cleanup
```

**Dependencies:**
- Sprint 1 blocks launch (CORS fix critical)
- Sprint 2, 3, 4 are independent → can overlap

---

### 7. Batch Similar Tasks
**Geschatte tijd:** 0 uur (discipline)  
**Impact:** 20-30% less context switching  
**Status:** APPLY NOW (pre-launch)

#### Bad Workflow (Context Switching)
```
09:00 - Write collector A
10:00 - Write test for collector A
11:00 - Write normalizer B
12:00 - Fix API bug
13:00 - Update documentation
14:00 - Write collector C
```
**Problem:** 6 context switches = 1-2 uur wasted

#### Good Workflow (Batched)
```
Morning Block (09:00-12:00):
  - ALL collector work (context loaded once)
  - Complete 3 collectors without interruption

Afternoon Block (13:00-15:00):
  - ALL test writing (testing mindset)
  - Cover all morning's work

Evening Block (15:00-17:00):
  - Documentation ONLY
  - Update all relevant docs
```
**Benefit:** 3 context switches instead of 6 = 30% less overhead

---

### 8. Context-Preserving Session Summaries
**Geschatte tijd:** 2 uur (template + discipline)  
**Impact:** 30% less rework from context loss  
**Status:** Post-launch (Feb 8-14)

#### Template
```markdown
# SESSIE_SAMENVATTING_YYYYMMDD.md

**Datum:** YYYY-MM-DD
**Sessie:** [Sprint name / Feature name]
**Deelnemers:** Leo + Claude AI / Claude Code

---

## ✅ WAT IS BEREIKT

- [x] Task 1: Specific deliverable
- [x] Task 2: Specific deliverable
- [x] Task 3: Specific deliverable

**Code changes:**
- `file/path/here.py`: What changed and why
- `another/file.py`: What changed and why

**Database changes:**
- Migration `XXXX_description.py`: What schema changed

---

## 🎯 BESLISSINGEN GENOMEN

**Decision 1: [Title]**
- Context: Why this was needed
- Options considered: A, B, C
- Chosen: B
- Reasoning: Because X, Y, Z
- Documented in: ADR-XXX or SKILL-XX

**Decision 2: [Title]**
- ...

---

## 🚧 OPEN ITEMS

- [ ] Follow-up task A (blocked by: X)
- [ ] Follow-up task B (needs: Y)
- [ ] Technical debt item C (defer to: V1.1)

---

## 📍 CONTEXT VOOR VOLGENDE SESSIE

**Where we left off:**
- Currently working in: `synctacles_db/collectors/new_feature.py`
- Last working state: Tests passing, ready for deployment
- Known edge cases: [List any edge cases discovered]

**Next steps:**
1. Complete implementation of X
2. Add tests for Y
3. Deploy to staging

**Relevant files:**
- `/path/to/file1.py` - Main implementation
- `/path/to/file2.py` - Supporting module
- `tests/test_feature.py` - Test coverage

**Known issues/limitations:**
- Issue A: Temporary workaround in place (TODO: proper fix)
- Limitation B: Performance acceptable but could optimize later
```

#### Usage
```bash
# At end of each work session
cp templates/SESSIE_SAMENVATTING.md docs/sessions/SESSIE_$(date +%Y%m%d).md
# Fill in details
git add docs/sessions/
git commit -m "docs: session summary $(date +%Y-%m-%d)"
```

---

### 9. Decision Log (ADR Process)
**Geschatte tijd:** 2 uur setup  
**Impact:** 5-10% time saved from re-discussions  
**Status:** Post-launch (Feb 8-14)

#### Template
```markdown
# ADR-XXX: [Title of Decision]

**Date:** YYYY-MM-DD
**Status:** [Proposed | Accepted | Deprecated | Superseded]
**Deciders:** [List of people involved]

---

## Context

[Describe the problem/situation requiring a decision]

Example:
> We need to decide how to handle TenneT API data given license restrictions.

---

## Decision Drivers

- Factor 1: License compliance
- Factor 2: User experience
- Factor 3: Development complexity

---

## Considered Options

1. **Option A:** Server-side aggregation
2. **Option B:** Client-side (BYO-key)
3. **Option C:** Hybrid approach

---

## Decision Outcome

**Chosen option:** Option B (BYO-key)

**Reasoning:**
- ✅ License compliant (users bring own key)
- ✅ No server-side redistribution risk
- ✅ Users maintain direct API access
- ⚠️ Slightly more setup friction (acceptable tradeoff)

---

## Consequences

### Positive
- Legal risk eliminated
- Transparent to users (they see API usage)
- Scalable without TenneT rate limit concerns

### Negative
- Requires user registration with TenneT
- Additional configuration step in HA
- Cannot provide TenneT data to users without API key

### Neutral
- Documentation update required
- HA component code change needed

---

## Links

- Related: ADR-007 (Data source architecture)
- Supersedes: [None]
- Implementation: SKILL_06 update, HA component PR #XX
```

#### Directory Structure
```
docs/decisions/
├── README.md                    # Index of all ADRs
├── ADR-001-three-layer-pipeline.md
├── ADR-002-brand-free-templates.md
├── ADR-008-tennet-byo-key.md
└── template.md                  # Blank template
```

---

### 10. Track Cycle Time Metrics
**Geschatte tijd:** 4 uur (dashboard setup)  
**Impact:** Continuous improvement visibility  
**Status:** Post-launch (Feb 15-21)

#### Metrics to Track
```yaml
Development Metrics:
  feature_request_to_deploy:
    target: < 3 days
    measure: From GitHub issue creation to production deploy
  
  bug_fix_time:
    p1_target: < 4 hours
    p2_target: < 24 hours
    measure: From bug report to fix deployed
  
  code_review_time:
    target: < 4 hours
    measure: From PR creation to merge
  
  deploy_to_verified:
    target: < 5 minutes
    measure: From deploy start to verification complete

Quality Metrics:
  test_coverage:
    target: > 60%
    measure: pytest --cov
  
  defect_escape_rate:
    target: < 5%
    measure: Bugs found in production vs caught in dev
  
  deployment_success_rate:
    target: > 95%
    measure: Successful deploys / total deploys

Operational Metrics:
  api_uptime:
    target: > 99.5%
    measure: Health check pass rate
  
  data_freshness:
    target: < 15 minutes
    measure: Latest data timestamp vs current time
  
  cache_hit_rate:
    target: > 80%
    measure: Cache hits / total requests
```

#### Simple Dashboard
```bash
# scripts/metrics-dashboard.sh
#!/bin/bash

echo "📊 SYNCTACLES Velocity Metrics Dashboard"
echo "=========================================="
echo ""

# Deployment frequency
echo "Deployment Frequency (last 30 days):"
git log --since="30 days ago" --grep="deploy\|release" --oneline | wc -l

# Average time to merge PRs (if using GitHub)
# gh pr list --state merged --limit 10 --json createdAt,mergedAt
# Calculate average

# Bug fix time (from GitHub issues with label "bug")
# gh issue list --label bug --state closed --limit 10

# Test coverage
echo ""
echo "Test Coverage:"
pytest --cov=synctacles_db --cov-report=term-missing | grep "TOTAL"

# Recent deploy success rate
echo ""
echo "Recent Deployments (last 5):"
journalctl -u energy-insights-nl-api --since "7 days ago" | grep "Deployment" | tail -5

echo ""
echo "Run './scripts/detailed-metrics.py' for full analysis"
```

---

## 🛠️ TIER 3: TECHNICAL OPTIMALISATIES

### 11. Database Query Optimization
**Geschatte tijd:** 8 uur  
**Impact:** 10x sneller API responses (200ms → 20ms)  
**Status:** Post-launch + production data (March 2026)

#### Analysis Phase (2 uur)
```sql
-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Find slowest queries
SELECT 
    substring(query, 1, 100) as query_snippet,
    calls,
    mean_exec_time,
    total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan ASC;

-- Table bloat analysis
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

#### Optimization Phase (6 uur)
```sql
-- Add missing indexes (based on query analysis)
CREATE INDEX CONCURRENTLY idx_norm_a75_timestamp_quality
ON norm_entso_e_a75(timestamp DESC, data_quality)
WHERE data_quality IN ('OK', 'STALE');

CREATE INDEX CONCURRENTLY idx_norm_a65_timestamp
ON norm_entso_e_a65(timestamp DESC);

CREATE INDEX CONCURRENTLY idx_norm_prices_timestamp
ON norm_prices(timestamp DESC);

-- Partial index for recent data (80% of queries)
CREATE INDEX CONCURRENTLY idx_norm_a75_recent
ON norm_entso_e_a75(timestamp DESC)
WHERE timestamp > NOW() - INTERVAL '24 hours';

-- Vacuum and analyze
VACUUM ANALYZE norm_entso_e_a75;
VACUUM ANALYZE norm_entso_e_a65;
VACUUM ANALYZE norm_prices;

-- Configure autovacuum for heavy-write tables
ALTER TABLE raw_entso_e_a75 SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.02
);
```

---

### 12. Collector Template Inheritance
**Geschatte tijd:** 6 uur  
**Impact:** 5x sneller new collectors (50 lines → 10 lines)  
**Status:** Post-launch (Feb 15-21)

#### Base Class Implementation
```python
# synctacles_db/collectors/base.py
"""Base collector with retry, logging, error handling."""
from synctacles_db.core.logging import get_logger
import asyncio
import aiohttp
from typing import Optional, Callable

class BaseCollector:
    """Base collector with common functionality."""
    
    def __init__(self, name: str):
        self.name = name
        self.logger = get_logger(f"collectors.{name}")
        self._session: Optional[aiohttp.ClientSession] = None
    
    async def __aenter__(self):
        """Async context manager entry."""
        self._session = aiohttp.ClientSession()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        if self._session:
            await self._session.close()
    
    async def collect_with_retry(
        self, 
        fetch_func: Callable, 
        max_retries: int = 3,
        backoff_base: float = 2.0
    ):
        """
        Execute fetch_func with exponential backoff retry.
        
        Args:
            fetch_func: Async function to execute
            max_retries: Maximum retry attempts
            backoff_base: Base for exponential backoff (seconds)
        
        Returns:
            Result from fetch_func
        
        Raises:
            Exception from fetch_func after max_retries
        """
        for attempt in range(max_retries):
            try:
                return await fetch_func()
            except Exception as e:
                if attempt == max_retries - 1:
                    self.logger.error(
                        f"Failed after {max_retries} attempts: {type(e).__name__}: {e}"
                    )
                    raise
                
                wait_time = backoff_base ** attempt
                self.logger.warning(
                    f"Attempt {attempt + 1}/{max_retries} failed: {e}. "
                    f"Retrying in {wait_time}s..."
                )
                await asyncio.sleep(wait_time)
    
    async def fetch_json(self, url: str, headers: dict = None, timeout: int = 30):
        """Fetch JSON data from URL with error handling."""
        if not self._session:
            raise RuntimeError("Session not initialized. Use 'async with' context.")
        
        async with self._session.get(url, headers=headers, timeout=timeout) as response:
            response.raise_for_status()
            return await response.json()
    
    async def fetch_xml(self, url: str, headers: dict = None, timeout: int = 30):
        """Fetch XML data from URL with error handling."""
        if not self._session:
            raise RuntimeError("Session not initialized. Use 'async with' context.")
        
        async with self._session.get(url, headers=headers, timeout=timeout) as response:
            response.raise_for_status()
            return await response.text()
```

#### Usage Example
```python
# synctacles_db/collectors/entso_e_a75_generation.py
from synctacles_db.collectors.base import BaseCollector
import time

class EntsoEA75Collector(BaseCollector):
    """ENTSO-E A75 generation collector."""
    
    def __init__(self):
        super().__init__("entso_e_a75")
        self.api_url = "https://web-api.tp.entsoe.eu/api"
    
    async def collect(self):
        """Main collection entry point."""
        self.logger.info(f"{self.name} starting")
        start_time = time.time()
        
        try:
            # Use base class retry mechanism
            data = await self.collect_with_retry(self._fetch_a75)
            
            elapsed = time.time() - start_time
            self.logger.info(f"{self.name} completed in {elapsed:.2f}s")
            return data
            
        except Exception as err:
            elapsed = time.time() - start_time
            self.logger.error(f"{self.name} failed: {type(err).__name__}: {err}")
            raise
    
    async def _fetch_a75(self):
        """Fetch A75 data (business logic only, no boilerplate)."""
        params = {
            "documentType": "A75",
            "processType": "A16",
            "in_Domain": "10YNL----------L",
            # ... other params
        }
        
        # Use base class fetch method
        return await self.fetch_xml(self.api_url, params=params)

# Usage
async def main():
    async with EntsoEA75Collector() as collector:
        await collector.collect()
```

**Benefit:** Went from 50+ lines to ~15 lines = 70% less code

---

### 13. Automated Fallback Testing
**Geschatte tijd:** 4 uur  
**Impact:** Zero fear of outages  
**Status:** Post-launch (Feb 15-21)

#### Chaos Testing Implementation
```python
# tests/chaos/test_fallback.py
"""Chaos engineering tests for fallback scenarios."""
import pytest
from unittest.mock import patch, MagicMock
from requests.exceptions import Timeout, ConnectionError
from synctacles_db.normalizers.normalize_entso_e_a75 import normalize_generation

class TestFallbackScenarios:
    """Test automatic fallback to Energy-Charts."""
    
    @pytest.mark.chaos
    def test_fallback_on_entso_e_timeout(self):
        """Simulate ENTSO-E timeout, verify Energy-Charts fallback."""
        with patch('synctacles_db.collectors.entso_e_a75.fetch') as mock_fetch:
            mock_fetch.side_effect = Timeout("ENTSO-E timeout")
            
            result = normalize_generation()
            
            # Verify fallback activated
            assert result['meta']['source'] == 'Energy-Charts'
            assert result['meta']['quality_status'] == 'FALLBACK'
            assert result['data']['total_mw'] > 0  # Has data despite timeout
    
    @pytest.mark.chaos
    def test_fallback_on_entso_e_500_error(self):
        """Simulate ENTSO-E 500 error, verify graceful fallback."""
        with patch('synctacles_db.collectors.entso_e_a75.fetch') as mock_fetch:
            mock_fetch.return_value.status_code = 500
            mock_fetch.return_value.raise_for_status.side_effect = ConnectionError()
            
            result = normalize_generation()
            
            assert result['meta']['source'] == 'Energy-Charts'
            assert result['meta']['quality_status'] == 'FALLBACK'
    
    @pytest.mark.chaos
    def test_stale_data_triggers_fallback(self):
        """Verify fallback when ENTSO-E data > 3 hours old."""
        # Mock stale timestamp in database
        with patch('synctacles_db.normalizers.normalize_entso_e_a75.get_latest_timestamp') as mock_ts:
            mock_ts.return_value = datetime.now(timezone.utc) - timedelta(hours=4)
            
            result = normalize_generation()
            
            # Should fallback due to staleness
            assert result['meta']['quality_status'] in ['FALLBACK', 'STALE']
    
    @pytest.mark.chaos
    def test_both_sources_fail_returns_cache(self):
        """Verify cache usage when all sources fail."""
        with patch('synctacles_db.collectors.entso_e_a75.fetch') as mock_entso, \
             patch('synctacles_db.collectors.energy_charts.fetch') as mock_ec:
            
            mock_entso.side_effect = Timeout()
            mock_ec.side_effect = ConnectionError()
            
            result = normalize_generation()
            
            # Should use cache as last resort
            assert result['meta']['source'] == 'CACHE'
            assert result['meta']['quality_status'] == 'CACHED'
```

#### Run Chaos Tests
```bash
# Run chaos tests specifically
pytest tests/chaos/ -v -m chaos

# Schedule chaos tests in CI (weekly)
# .github/workflows/chaos-test.yml
name: Chaos Testing
on:
  schedule:
    - cron: '0 2 * * 0'  # Every Sunday at 2am
```

---

### 14. Database Reset Script
**Geschatte tijd:** 2 uur  
**Impact:** 60x sneller clean testing  
**Status:** Post-launch (Feb 8-14)

#### Implementation
```bash
#!/bin/bash
# scripts/reset-dev-db.sh

set -e

DB_NAME="energy_insights_nl_dev"
DB_USER="energy_insights_nl"

echo "🔄 Resetting development database: $DB_NAME"

# 1. Drop database (if exists)
echo "  → Dropping existing database..."
sudo -u postgres psql -c "DROP DATABASE IF EXISTS $DB_NAME;"

# 2. Create fresh database
echo "  → Creating fresh database..."
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

# 3. Run migrations
echo "  → Running Alembic migrations..."
cd /opt/energy-insights-nl/app
source /opt/energy-insights-nl/venv/bin/activate
alembic upgrade head

# 4. Seed test data (optional)
if [ -f scripts/seed-test-data.py ]; then
    echo "  → Seeding test data..."
    python scripts/seed-test-data.py
fi

# 5. Verify
echo "  → Verifying schema..."
psql -U $DB_USER -d $DB_NAME -c "\dt" | grep "norm_entso_e_a75" || {
    echo "❌ Schema verification failed"
    exit 1
}

echo "✅ Development database reset complete"
echo ""
echo "Next steps:"
echo "  1. Run collectors to populate data"
echo "  2. Run normalizers to process data"
echo "  3. Start API server"
```

#### Test Data Seeder (Optional)
```python
# scripts/seed-test-data.py
"""Seed development database with realistic test data."""
from datetime import datetime, timedelta, timezone
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from config.settings import DATABASE_URL
from synctacles_db.models import NormEntsoeA75, NormEntsoeA65, NormPrices

engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)

def seed_generation_data():
    """Seed 7 days of realistic generation data."""
    session = Session()
    
    now = datetime.now(timezone.utc)
    
    for hours_ago in range(0, 168, 1):  # 7 days, hourly
        timestamp = now - timedelta(hours=hours_ago)
        
        # Realistic NL generation mix
        record = NormEntsoeA75(
            timestamp=timestamp,
            biomass_mw=350 + (hours_ago % 50),
            wind_onshore_mw=1200 + (hours_ago * 10 % 800),
            wind_offshore_mw=600 + (hours_ago * 5 % 400),
            solar_mw=0 if timestamp.hour < 6 or timestamp.hour > 20 else 800 - abs(timestamp.hour - 13) * 60,
            nuclear_mw=485,
            gas_mw=2000 + (hours_ago * 20 % 1500),
            coal_mw=200 if hours_ago % 3 == 0 else 0,
            waste_mw=120,
            other_mw=100,
            data_source='SEEDED',
            data_quality='OK'
        )
        
        session.add(record)
    
    session.commit()
    print(f"✅ Seeded {168} generation records")

def seed_load_data():
    """Seed 7 days of load data."""
    # Similar pattern...
    pass

def seed_price_data():
    """Seed 7 days of price data."""
    # Similar pattern...
    pass

if __name__ == '__main__':
    seed_generation_data()
    seed_load_data()
    seed_price_data()
```

---

## 📅 IMPLEMENTATIE ROADMAP

### Pre-Launch (Jan 6 - Jan 25)
**Focus:** Launch V1 met huidige velocity (6x)

**Implementeer ALLEEN:**
- ✅ Parallel sprint execution (0 uur - planning change)
- ✅ Batch similar tasks (0 uur - discipline)

**Reden:** Launch window prioriteit, 6x velocity is voldoende

---

### Post-Launch Week 1 (Feb 1-7)
**Focus:** Stabilisatie + metrics verzamelen

**Activiteiten:**
- Fix launch bugs (4-8 uur)
- Monitor production metrics (daily)
- Collect user feedback (daily)
- NO new development (stabilisatie periode)

---

### Post-Launch Week 2-3 (Feb 8-21): VELOCITY SPRINT
**Focus:** Tier 1 Quick Wins implementatie

**Sprint planning:**
```
Day 1-2 (Feb 8-9): Template generators
  - Collector template (4 uur)
  - Normalizer template (2 uur)
  - API endpoint template (2 uur)

Day 3-4 (Feb 10-11): Development environment
  - Pre-commit hooks (2 uur)
  - Hot-reload setup (1 uur)
  - Smart test selection (1 uur)

Day 5 (Feb 12): Deployment
  - Optimized post-deploy verification (1 uur)
  - Database reset script (2 uur)

Week 3 (Feb 15-21): Workflow + Technical
  - Session summary templates (2 uur)
  - ADR process setup (2 uur)
  - Metrics dashboard (4 uur)
  - Collector base class (6 uur)
  - Chaos tests (4 uur)
```

**Total: 33 uur over 2 weken = 17 uur/week = 3.5 uur/dag**

---

### March 2026: DATA-DRIVEN OPTIMIZATION
**Focus:** Tier 3 technical optimalisaties based on production data

**Activiteiten:**
- Database query analysis (2 uur)
- Index optimization (6 uur)
- Performance monitoring (4 uur)

---

## 🎯 GITHUB ISSUES VOOR CC

### Instructies voor Claude Code

**CC:** Maak de volgende GitHub issues aan in `DATADIO/synctacles-api` repository:

#### Issue Template Format
```markdown
## [VELOCITY] Title

**Category:** Velocity Optimization
**Priority:** P2 (Post-Launch)
**Estimated effort:** X hours
**Target:** V1.1 (Feb 2026)
**Depends on:** V1 Launch (Jan 25)

### Description
[Brief description]

### Implementation Checklist
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

### Success Criteria
- Criterion 1
- Criterion 2

### Related
- Tier: [1/2/3]
- Impact: [velocity boost estimate]
- Document: `/docs/VELOCITY_OPTIMIZATION.md`
```

---

### Issues to Create

#### Tier 1: Quick Wins (5 issues)

**Issue #1: Template-Driven Development**
```markdown
## [VELOCITY] Template-Driven Development

**Category:** Velocity Optimization
**Priority:** P2 (Post-Launch)
**Estimated effort:** 2 hours
**Target:** V1.1 (Feb 8-14, 2026)
**Depends on:** V1 Launch

### Description
Create generator scripts for collectors, normalizers, and API endpoints to eliminate boilerplate code.

### Implementation Checklist
- [ ] Create `scripts/generate-collector.sh`
- [ ] Create `scripts/generate-normalizer.sh`
- [ ] Create `scripts/generate-api-endpoint.sh`
- [ ] Add templates to `templates/` directory
- [ ] Test generators with new component
- [ ] Document usage in CONTRIBUTING.md

### Success Criteria
- New collector generated in < 2 minutes
- Generated code passes syntax check
- 80% less boilerplate code

### Related
- Tier: 1 (Quick Win)
- Impact: 5x faster new components
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 1
```

**Issue #2: Pre-Commit Validation Hooks**
```markdown
## [VELOCITY] Pre-Commit Validation Hooks

**Category:** Velocity Optimization
**Priority:** P2 (Post-Launch)
**Estimated effort:** 2 hours
**Target:** V1.1 (Feb 8-14, 2026)

### Description
Implement pre-commit hooks to catch 90% of bugs before commit (syntax errors, hardcoded credentials, TODO markers, missing imports).

### Implementation Checklist
- [ ] Create `.git/hooks/pre-commit` script
- [ ] Add Python syntax validation
- [ ] Add credential scanning
- [ ] Add TODO marker detection
- [ ] Add required imports check
- [ ] Add .env file protection
- [ ] Test with intentional violations
- [ ] Document in CONTRIBUTING.md

### Success Criteria
- Blocks commits with syntax errors
- Blocks commits with hardcoded credentials
- Warns on TODO markers in production code
- 90% bug catch rate before push

### Related
- Tier: 1 (Quick Win)
- Impact: 2x less debugging time
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 2
```

**Issue #3: Optimized Deployment Validation**
```markdown
## [VELOCITY] Optimize Post-Deploy Verification

**Category:** Velocity Optimization
**Priority:** P2 (Post-Launch)
**Estimated effort:** 1 hour
**Target:** V1.1 (Feb 8-14, 2026)

### Description
Optimize post-deploy verification script from 10 minutes to 30 seconds using parallel endpoint testing and single database query.

### Implementation Checklist
- [ ] Refactor endpoint tests to run in parallel
- [ ] Combine multiple DB queries into one
- [ ] Add timing measurements
- [ ] Test on staging environment
- [ ] Update deployment scripts to use optimized version

### Success Criteria
- Verification completes in < 60 seconds
- All checks still thorough
- Zero false positives

### Related
- Tier: 1 (Quick Win)
- Impact: 20x faster deployment validation
- Depends on: Sprint 1 (initial post-deploy script)
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 3
```

**Issue #4: Hot-Reload Development Environment**
```markdown
## [VELOCITY] Hot-Reload Development Setup

**Category:** Velocity Optimization
**Priority:** P2 (Post-Launch)
**Estimated effort:** 1 hour
**Target:** V1.1 (Feb 8-14, 2026)

### Description
Setup auto-reload for API and collectors to eliminate manual restart cycle (2 min → 5 sec).

### Implementation Checklist
- [ ] Create `scripts/dev-server.sh` with uvicorn --reload
- [ ] Configure reload directories
- [ ] Add watchdog for collectors/normalizers
- [ ] Test auto-reload on file save
- [ ] Document in LOCAL_DEVELOPMENT.md

### Success Criteria
- API restarts automatically on code change
- Restart time < 5 seconds
- No manual intervention needed

### Related
- Tier: 1 (Quick Win)
- Impact: 24x faster iteration
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 4
```

**Issue #5: Smart Test Selection (pytest-testmon)**
```markdown
## [VELOCITY] Smart Test Selection with pytest-testmon

**Category:** Velocity Optimization
**Priority:** P2 (Post-Launch)
**Estimated effort:** 1 hour
**Target:** V1.1 (Feb 8-14, 2026)

### Description
Implement pytest-testmon to run only affected tests instead of full suite (5 min → 30 sec).

### Implementation Checklist
- [ ] Install pytest-testmon
- [ ] Add to requirements-dev.txt
- [ ] Configure pytest.ini
- [ ] Setup .testmondata caching in CI/CD
- [ ] Test with file changes
- [ ] Document in CONTRIBUTING.md

### Success Criteria
- Only affected tests run on subsequent runs
- First run establishes baseline
- CI/CD runs 10x faster for small changes

### Related
- Tier: 1 (Quick Win)
- Impact: 10x faster CI
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 5
```

---

#### Tier 2: Workflow (4 issues)

**Issue #6: Session Summary Template**
```markdown
## [VELOCITY] Context-Preserving Session Summaries

**Category:** Velocity Optimization
**Priority:** P3 (Post-Launch)
**Estimated effort:** 2 hours
**Target:** V1.1 (Feb 15-21, 2026)

### Description
Create standardized session summary template to prevent context loss between work sessions (30% less rework).

### Implementation Checklist
- [ ] Create `templates/SESSIE_SAMENVATTING.md`
- [ ] Document template usage
- [ ] Add example filled template
- [ ] Create `docs/sessions/` directory
- [ ] Add to workflow documentation

### Success Criteria
- Template covers: achievements, decisions, open items, context
- Easy to fill (< 10 minutes)
- Prevents context loss

### Related
- Tier: 2 (Workflow)
- Impact: 30% less rework
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 8
```

**Issue #7: ADR Process Setup**
```markdown
## [VELOCITY] Architecture Decision Records Process

**Category:** Velocity Optimization
**Priority:** P3 (Post-Launch)
**Estimated effort:** 2 hours
**Target:** V1.1 (Feb 15-21, 2026)

### Description
Setup ADR (Architecture Decision Record) process to avoid re-discussing decisions.

### Implementation Checklist
- [ ] Create `docs/decisions/` directory
- [ ] Create ADR template
- [ ] Document existing decisions (ADR-001 through ADR-008)
- [ ] Add ADR process to CONTRIBUTING.md
- [ ] Create INDEX of all ADRs

### Success Criteria
- Template covers: context, options, decision, consequences
- Easy to create (< 30 minutes per ADR)
- 5-10% time saved from re-discussions

### Related
- Tier: 2 (Workflow)
- Impact: 5-10% time saved
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 9
```

**Issue #8: Cycle Time Metrics Dashboard**
```markdown
## [VELOCITY] Track Development Cycle Time Metrics

**Category:** Velocity Optimization
**Priority:** P3 (Post-Launch)
**Estimated effort:** 4 hours
**Target:** V1.1 (Feb 15-21, 2026)

### Description
Setup metrics tracking for feature delivery, bug fixes, code review, and deployment times to enable continuous improvement.

### Implementation Checklist
- [ ] Define metrics to track (feature/bug/review/deploy time)
- [ ] Create `scripts/metrics-dashboard.sh`
- [ ] Setup automated collection (GitHub Actions)
- [ ] Create simple visualization
- [ ] Document targets and baselines

### Success Criteria
- Tracks 4 key cycle time metrics
- Dashboard updates weekly
- Visibility enables improvement

### Related
- Tier: 2 (Workflow)
- Impact: Continuous improvement visibility
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 10
```

**Issue #9: Database Reset Script**
```markdown
## [VELOCITY] Database Reset Script for Clean Testing

**Category:** Velocity Optimization
**Priority:** P3 (Post-Launch)
**Estimated effort:** 2 hours
**Target:** V1.1 (Feb 8-14, 2026)

### Description
Create one-command database reset for development/testing (10 min manual → 10 sec automated).

### Implementation Checklist
- [ ] Create `scripts/reset-dev-db.sh`
- [ ] Add Alembic migration execution
- [ ] Add optional test data seeder
- [ ] Add schema verification
- [ ] Test on development environment
- [ ] Document in LOCAL_DEVELOPMENT.md

### Success Criteria
- Single command resets database
- Completes in < 30 seconds
- Optionally seeds test data

### Related
- Tier: 2 (Workflow)
- Impact: 60x faster clean testing
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 14
```

---

#### Tier 3: Technical (3 issues)

**Issue #10: Database Query Optimization**
```markdown
## [VELOCITY] Database Query Performance Optimization

**Category:** Velocity Optimization
**Priority:** P3 (Post-Launch)
**Estimated effort:** 8 hours
**Target:** March 2026 (data-driven)
**Depends on:** Production metrics

### Description
Analyze production query performance and optimize slow queries with indexes and query rewrites.

### Implementation Checklist
- [ ] Enable pg_stat_statements
- [ ] Analyze slowest 20 queries
- [ ] Review index usage
- [ ] Add missing indexes
- [ ] Optimize slow queries
- [ ] Vacuum and analyze tables
- [ ] Configure autovacuum settings
- [ ] Measure before/after performance

### Success Criteria
- p95 query time < 50ms
- No full table scans on hot paths
- 10x faster API responses

### Related
- Tier: 3 (Technical)
- Impact: 10x faster API
- Requires: Production data
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 11
```

**Issue #11: Collector Base Class Inheritance**
```markdown
## [VELOCITY] Collector Template Inheritance Pattern

**Category:** Velocity Optimization
**Priority:** P3 (Post-Launch)
**Estimated effort:** 6 hours
**Target:** V1.1 (Feb 15-21, 2026)

### Description
Refactor collectors to use base class with shared retry logic, logging, and error handling.

### Implementation Checklist
- [ ] Create `synctacles_db/collectors/base.py`
- [ ] Implement BaseCollector with retry/logging
- [ ] Add async context manager support
- [ ] Add fetch_json() and fetch_xml() helpers
- [ ] Refactor existing collectors to inherit from base
- [ ] Add tests for base class
- [ ] Document pattern in ARCHITECTURE.md

### Success Criteria
- New collector = 10 lines vs 50 lines
- All collectors use consistent patterns
- Zero duplication of retry/error logic

### Related
- Tier: 3 (Technical)
- Impact: 5x faster new collectors
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 12
```

**Issue #12: Automated Fallback Chaos Testing**
```markdown
## [VELOCITY] Automated Fallback Chaos Testing

**Category:** Velocity Optimization
**Priority:** P3 (Post-Launch)
**Estimated effort:** 4 hours
**Target:** V1.1 (Feb 15-21, 2026)

### Description
Implement automated chaos tests to verify fallback behavior under various failure scenarios.

### Implementation Checklist
- [ ] Create `tests/chaos/` directory
- [ ] Implement ENTSO-E timeout test
- [ ] Implement ENTSO-E 500 error test
- [ ] Implement stale data test
- [ ] Implement both-sources-fail test
- [ ] Add chaos tests to CI (weekly schedule)
- [ ] Document chaos testing approach

### Success Criteria
- 4+ chaos scenarios covered
- Tests run automatically weekly
- Zero fear of outages

### Related
- Tier: 3 (Technical)
- Impact: Confidence in fallback reliability
- Document: `/docs/VELOCITY_OPTIMIZATION.md` section 13
```

---

### GitHub Labels to Create

```
velocity-optimization
tier-1-quick-win
tier-2-workflow
tier-3-technical
post-launch
v1.1
```

---

### Issue Assignment
- Assign all to: `@DATADIO` (Leo)
- Milestone: `V1.1 - Velocity Optimization`
- Label: `velocity-optimization` + appropriate tier label

---

## 📊 EXPECTED RESULTS

### Before Optimization (Current)
```
Velocity: 6x baseline
New collector: 30 min
New endpoint: 45 min
Deploy verification: 10 min
Test suite: 5 min (full)
Bug catch rate: 70% (pre-deploy)
Context loss: 30% rework
```

### After Optimization (Target)
```
Velocity: 10-12x baseline
New collector: 5 min (template)
New endpoint: 10 min (template)
Deploy verification: 30 sec (parallel)
Test suite: 30 sec (smart selection)
Bug catch rate: 90% (pre-commit hooks)
Context loss: 5% rework (session summaries)
```

**Overall Impact:** 2x additional velocity boost = **12x total vs baseline**

---

## ⚠️ CRITICAL: TIMING & PRIORITY

**DO NOT implement before V1 launch (Jan 25)**

**Reasons:**
1. Launch window closing (prioriteit #1)
2. Current 6x velocity sufficient for launch
3. Optimization costs 33 uur, saves only ~15 uur pre-launch
4. Production data needed for smart optimization

**Implement after launch:**
- Week 1 (Feb 1-7): Stabilization
- Week 2-3 (Feb 8-21): Tier 1 Quick Wins
- March: Tier 3 Data-Driven Optimization

**Exception:** Parallel sprint execution and batched workflows (0 uur cost, immediate benefit)

---

**Status:** Ready for GitHub issue creation by CC  
**Priority:** Post-launch V1.1 implementation  
**Expected ROI:** 2x velocity boost (6x → 12x baseline)
