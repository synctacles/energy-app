# HANDOFF: CC → CAI

**Datum:** 2026-01-08
**Van:** CC
**Naar:** CAI
**Type:** Pipeline Dashboard Assessment

---

## STATUS

✅ **COMPLETE** - Assessment done

---

## PIPELINE DASHBOARD PRIORITY ASSESSMENT

### Context

Leo wil een Pipeline Health Dashboard in Grafana om data pipeline health te monitoren.
Vandaag's normalizer bug zou instant zichtbaar geweest zijn met zo'n dashboard.

**Geschatte effort:** 7-10 uur

---

### Huidige Backlog (Gap Audit Issues)

Van PRODUCT_REALITY_CHECK.md (11 gaps gevonden):

| # | Issue | Priority | Status |
|---|-------|----------|--------|
| 1 | Database not updating (normalizer bug) | HIGH | ✅ FIXED (commit bd8518e) |
| 2 | TenneT service failing | HIGH | ✅ FIXED (commit bd8518e) |
| 3 | API staleness warnings | HIGH | ✅ VERIFIED WORKING |
| 4 | Undocumented endpoint `/v1/now` | MEDIUM | Open |
| 5 | Undocumented endpoint `/metrics` | MEDIUM | Open |
| 6 | Undocumented cache endpoints (3x) | MEDIUM | Open |
| 7 | Undocumented admin endpoint | MEDIUM | Open |
| 8 | Orphaned TenneT tables | MEDIUM | Open |
| 9 | Verify endpoint aliases | LOW | Open |
| 10 | Incomplete timer documentation | LOW | Open |
| 11 | Missing API reference doc | LOW | Open |

**Plus nieuw:**
| 12 | CVE Security vulnerabilities (13 CVEs) | HIGH | ✅ FIXED (commit b1092f7) |
| 13 | Pipeline Health Dashboard (this assessment) | TBD | Assessment |

---

### Current System Status

**Grafana:**
- Status: Installed, maar inactive (dead since 2026-01-06 13:21:56)
- Location: `/etc/grafana/provisioning/dashboards/` (te checken)
- Action needed: Start service + create dashboard

**Monitoring Infrastructure:**
- Prometheus: Status unknown (not checked)
- Existing dashboards: None found
- Metrics endpoints: `/metrics` exists maar undocumented (Issue #5)

**Data Pipeline:**
- Collectors: ✅ Running (every 15 min)
- Importers: ✅ Running (every 15 min)
- Normalizers: ✅ Fixed (now includes A75, A65, A44, prices)
- API: ✅ Operational
- Health: `/health` endpoint exists

---

### Dependencies

**Before Pipeline Dashboard can be implemented:**

1. **Grafana activation:**
   - [ ] Start grafana-server service
   - [ ] Verify Grafana accessible (http://localhost:3000)
   - [ ] Setup Prometheus data source

2. **Prometheus setup (indien niet actief):**
   - [ ] Install/start Prometheus
   - [ ] Configure Prometheus to scrape `/metrics` endpoint
   - [ ] Verify metrics collection

3. **Metrics instrumentation:**
   - [ ] Document `/metrics` endpoint (Issue #5)
   - [ ] Verify metrics exist for:
     - Collector success/failure per source (A75, A65, A44)
     - Importer success/failure per source
     - Normalizer success/failure per source
     - API endpoint response times
     - Data age per source

4. **Optional - GitHub issues:**
   - [ ] Create remaining 8 open issues from gap audit
   - [ ] Create issue for Pipeline Dashboard

---

### Prioriteit Advies

**MEDIUM-HIGH Priority** - Waarom:

**Arguments FOR (HIGH):**
1. ✅ **Proven value:** Bug vandaag (normalizer) zou instant zichtbaar geweest zijn
2. ✅ **Quick detection:** Stale data detection binnen minuten vs dagen
3. ✅ **Proactive monitoring:** Problemen spotten voor users ze merken
4. ✅ **Leo's request:** Expliciet gevraagd door product owner
5. ✅ **Limited complexity:** 7-10 uur is beheersbaar

**Arguments AGAINST (LOWER):**
1. ⚠️ **Dependencies:** Requires Grafana + Prometheus setup first
2. ⚠️ **Current fixes working:** Normalizer fixed, TenneT disabled, no active production issues
3. ⚠️ **Documentation gaps:** 8 open issues from gap audit
4. ⚠️ **Grafana inactive:** Service needs restart/config

**Net Assessment: MEDIUM-HIGH**
- Value is clear (prevent today's bug)
- Dependencies manageable (Grafana + Prometheus)
- But not blocking (production operational)

---

### Aanbevolen Actie

**Option A: Next Sprint (RECOMMENDED)**

**Timeline:** Sprint 3 (after current Sprint 2 - HA Component)

**Rationale:**
- Production nu stabiel (fixes compleet)
- Grafana/Prometheus setup first (dependencies)
- Complete gap audit issues first (finish what we started)
- Then dashboard as proactive monitoring enhancement

**Sprint Plan:**
```
Sprint 2 (current): HA Component documentation ✅
Sprint 3 (next):
  - Week 1: Grafana/Prometheus setup + metrics documentation
  - Week 2: Pipeline Health Dashboard implementation
  - Week 3: Testing + alerting rules
```

**Pros:**
- Dependencies resolved first
- Documentation gaps closed
- Fresh start with clear foundation
- Leo gets visual monitoring tool

**Cons:**
- Not immediate (maar production is nu OK)
- Another normalizer bug zou nog niet zichtbaar zijn (maar unlikely - we fixed the root cause)

---

**Option B: Urgent (binnen 2 dagen)**

**Only if:**
- Leo considers this blocking for stakeholder demo
- OR production incidents frequency increases
- OR regulatory/compliance requirement

**Rationale:** Production is operational, no active incidents, documented backlog exists.

---

**Option C: Backlog (NOT RECOMMENDED)**

**Rationale:** Too much value demonstrated today. Proactive monitoring > reactive firefighting.

---

### Alternatief (Quick Win)

**Minimal Viable Monitoring (2-3 uur):**

If dashboard te complex nu, overweeg:

1. **Simple health check script:**
```bash
#!/bin/bash
# /opt/github/synctacles-api/scripts/pipeline_health_check.sh

# Check each pipeline stage
echo "=== PIPELINE HEALTH ==="

# Collectors (check last run < 20 min)
collector_age=$(systemctl show energy-insights-nl-collector.service -p ActiveExitTimestampMonotonic --value)
echo "Collector: $(if [ $collector_age -lt 1200000000 ]; then echo "OK"; else echo "STALE"; fi)"

# Normalizers (check database age)
db_age=$(sudo -u postgres psql -d energy_insights_nl -t -c "SELECT EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 FROM norm_entso_e_a75")
echo "A75 age: ${db_age} minutes (FRESH < 90, STALE < 180)"

# Repeat for A65, A44
# API health check
curl -sf http://localhost:8000/health > /dev/null && echo "API: OK" || echo "API: FAIL"
```

2. **Cronjob alert:**
```bash
# Run every 15 minutes, alert if issues
*/15 * * * * /opt/github/synctacles-api/scripts/pipeline_health_check.sh | grep -E "STALE|FAIL" && /usr/bin/mail -s "Pipeline Health Alert" admin@domain.com
```

**Effort:** 2-3 uur
**Value:** Basic monitoring without Grafana dependency
**Limitation:** No visualization, just alerts

---

### Implementation Plan (if approved for Sprint 3)

**Phase 1: Infrastructure (Week 1)**
1. Start Grafana service
2. Setup Prometheus (if needed)
3. Configure Prometheus scraping
4. Document `/metrics` endpoint
5. Verify metrics collection

**Phase 2: Dashboard (Week 2)**
1. Create Grafana dashboard template
2. Add panels per data source (A75, A65, A44)
3. Add panels per pipeline stage (Collector → Importer → Normalizer → API)
4. Color coding (green/yellow/red based on thresholds)
5. Data age display per source

**Phase 3: Alerting (Week 3)**
1. Configure alert rules
2. Test alert firing
3. Setup notification channels (email/Slack)
4. Document dashboard usage

**Deliverable:**
Pipeline Health Dashboard showing:
- 3 rows (A75, A65, A44)
- 4 columns (Collector, Importer, Normalizer, API)
- Green/red status per block
- Data age per source
- Alert if any block red > 15 min

---

### Risk Assessment

**Risk: Dashboard shows false negatives**
- Mitigation: Test thoroughly with known failures
- Likelihood: LOW (we now understand pipeline)

**Risk: Grafana/Prometheus setup takes longer**
- Mitigation: Budget 2-3 uur extra
- Likelihood: MEDIUM (onbekende config)

**Risk: Leo wants it NOW**
- Mitigation: Offer quick win alternative (health check script)
- Likelihood: LOW (production stable)

---

## DELIVERABLES

1. ✅ Current backlog assessment (13 issues: 4 fixed, 8 open, 1 assessed)
2. ✅ Dependencies identified (Grafana, Prometheus, metrics)
3. ✅ Priority recommendation (MEDIUM-HIGH, Sprint 3)
4. ✅ Quick win alternative (health check script)
5. ✅ Implementation plan (3-week phased approach)

---

## RECOMMENDED NEXT ACTIONS

**Immediate (CC):**
1. ✅ Security fixes complete (commit b1092f7)
2. ⏳ Create GitHub issues for 8 remaining gaps
3. ⏳ Create GitHub issue for Pipeline Dashboard

**Short-term (CAI decides):**
- Approve/reject Pipeline Dashboard for Sprint 3
- OR request quick win alternative (health check script)
- OR defer to backlog

**Sprint 3 (if approved):**
- Week 1: Infrastructure setup
- Week 2: Dashboard implementation
- Week 3: Alerting + testing

---

## CONTEXT FOR CAI

**Assessment Request:** Determine priority for Pipeline Health Dashboard

**Analysis:**
- Value proven today (normalizer bug would be instantly visible)
- Dependencies manageable (Grafana + Prometheus)
- Current production stable (no urgency)
- Backlog exists (8 open gap audit issues)

**Recommendation:** Sprint 3 (MEDIUM-HIGH priority)
- Complete gap audit issues first
- Setup infrastructure properly
- Implement dashboard with alerting
- Deliver comprehensive monitoring solution

**Alternative:** Quick win health check script (2-3 uur) if dashboard te complex

**Leo's Call:** Decision needed on timing (Sprint 3 vs Now vs Backlog)

---

*Template versie: 1.0*
*Response to: HANDOFF_CAI_CC_PIPELINE_DASHBOARD_ASSESSMENT.md*
*Completed: 2026-01-08 11:40 UTC*
