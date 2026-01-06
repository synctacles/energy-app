# Monitoring & Alerting Infrastructure Project

**Project ID:** #24
**Status:** 🟡 PLANNING → IN PROGRESS
**Created:** 2026-01-06
**Target Completion:** 2026-01-09

---

## 📋 Project Summary

Comprehensive monitoring and alerting infrastructure for Energy Insights NL production environment. This project addresses production blocker #1 from [PRODUCTION_BLOCKERS.md](PRODUCTION_BLOCKERS.md): **Memory Monitoring & Alerting**.

**Why This Project?**
- System experienced OOM crash - need visibility into memory behavior
- No monitoring during load - can't predict capacity
- No alerts - system fails silently until complete crash
- No baseline - don't know healthy vs unhealthy state

**What We're Building:**
- Real-time metrics collection (Prometheus)
- Visual dashboards (Grafana)
- Intelligent alerting (AlertManager → Slack)
- Capacity planning data (from load tests)
- Operational runbooks (for response procedures)

---

## 🎯 Project Goals

### Primary Goal
Achieve complete visibility into production system behavior, with automated alerts and documented capacity limits.

### Success Criteria
- ✅ Memory monitoring active and graphed
- ✅ Alerts firing to Slack within 1 minute of threshold
- ✅ Realistic load baseline established
- ✅ Stress test breaking point documented
- ✅ Capacity limits clear (e.g., "max 7 collectors safely")
- ✅ Operational runbooks complete

---

## 📊 Tech Stack

```
┌─────────────────────────┐         ┌──────────────────────────┐
│  Energy Insights NL     │         │  CX23 Monitoring Server  │
│  (Production)           │         │  (Docker)                │
├─────────────────────────┤         ├──────────────────────────┤
│ - API (port 8000)       │         │ - Prometheus (9090)      │
│ - PostgreSQL (5432)     │         │ - Grafana (3000)         │
│ - Collectors            │         │ - AlertManager (9093)    │
│ - node-exporter (9100)  │────────►│ - AlertManager Webhook   │
│   (system metrics)      │ scrape  │   → Slack                │
└─────────────────────────┘         └──────────────────────────┘
                                            │
                                            │ alerts
                                            ▼
                                    ┌──────────────┐
                                    │ Slack        │
                                    │ SYNCTACLES   │
                                    │ Workspace    │
                                    └──────────────┘
```

**Components:**
- **node-exporter** - Lightweight agent on main server (system metrics: RAM, CPU, disk, I/O)
- **Prometheus** - Time-series database + alerting engine (scrapes every 15 sec)
- **Grafana** - Dashboard and visualization platform
- **AlertManager** - Routes alerts to Slack with severity grouping
- **Slack** - SYNCTACLES workspace for notifications

---

## 📅 Phases & Timeline

### Phase 1: CX23 Server Setup (Day 1)
**Issue:** #25

Tasks:
- Update system packages
- Install Docker + Docker Compose
- Create docker-compose.yml for monitoring stack
- Deploy Prometheus, Grafana, AlertManager containers
- Configure firewall (ports 9090, 3000, 9093)

**Deliverable:** Monitoring server ready, all containers running

---

### Phase 2: node-exporter Setup (Day 1)
**Issue:** #26

Tasks:
- Install node-exporter on main Energy Insights NL server
- Configure systemd service (starts on boot)
- Expose metrics at http://localhost:9100/metrics
- Configure Prometheus scrape job

**Deliverable:** System metrics flowing from main server to monitoring server

---

### Phase 3: AlertManager & Slack Integration (Day 1-2)
**Issue:** #27

Tasks:
- Configure AlertManager webhook to Slack
- Create 5 alert rules (high memory, critical memory, high swap, API down, disk low)
- Test each alert → verify Slack notification
- Create Slack channels (#critical-alerts, #warnings, #info-metrics)
- Route by severity

**Deliverable:** Real-time alerts firing to Slack

---

### Phase 4: Grafana Dashboards (Day 2)
**Issue:** #28

Tasks:
- System Health Dashboard (RAM, CPU, Swap, Page Faults)
- Application Dashboard (Workers, API latency, error rates)
- Database Dashboard (Connections, query performance)
- Load Test Dashboard (skeleton for Phase 5 results)

**Deliverable:** 4 operational dashboards with real-time data

---

### Phase 5: Load Testing (Day 2-3)
**Issue:** #29

Tasks:
- **Part A:** Realistic load test (5 collectors, normal API traffic)
  - Run 30 minutes
  - Baseline memory, CPU, response times
  - Expected: System stable, memory <70%

- **Part B:** Stress test (incremental load until break)
  - Start: 5 collectors
  - Add 1 collector every 5 min
  - Continue until OOM or crash
  - Record breaking point

**Deliverable:** Load test results + capacity limits (e.g., "safe max: 7 collectors")

---

### Phase 6: Documentation (Day 3)
**Issue:** #30

Documents to create:
1. **MONITORING_SETUP_GUIDE.md** - Architecture + deployment
2. **LOAD_TEST_RESULTS.md** - Test data + analysis
3. **CAPACITY_LIMITS.md** - Scaling guidelines
4. **ALERT_RUNBOOK.md** - How to respond to each alert
5. **MONITORING_MAINTENANCE.md** - Daily/weekly/monthly checks
6. **SCALING_GUIDELINES.md** - When/how to scale

**Deliverable:** Complete operational documentation

---

## 🔄 Current Status

### ✅ Complete
- Code quality audit (#23) - 0 credential violations
- GitHub project structure created (#24)
- All 6 phase issues created (#25-30)

### 🟡 Ready to Start
- **Phase 1:** CX23 clean and ready for upgrade
- **Phase 2:** Can start after Phase 1 completes
- **Phase 3:** Slack workspace ready (SYNCTACLES)

### ⏳ Pending
- Phases 1-6 execution (3 days)

---

## 📈 Key Metrics to Monitor

### System Level
- RAM utilization (%)
- Swap usage (%)
- Available memory (MB)
- CPU utilization
- Page faults
- Disk I/O

### Application Level
- Gunicorn worker memory (per worker)
- API request latency (P50/P95/P99)
- Requests per second (throughput)
- HTTP error rate (4xx, 5xx)
- Active connections

### Database Level
- PostgreSQL connections (active/max)
- Connection pool usage (%)
- Query execution time
- Buffer cache effectiveness

### Load Test Specific
- Memory under realistic load (baseline)
- Memory under stress test (breaking point)
- CPU behavior under load
- Response time degradation

---

## 🚨 Alert Rules (Configured in Phase 3)

### CRITICAL Alerts (Immediate Slack notification)
| Rule | Threshold | Action |
|------|-----------|--------|
| CriticalMemoryUsage | >85% RAM | Scale immediately or restrict load |
| APIDown | 2 min no response | Investigate + restart |
| OOMRisk | Swap >50% | Emergency: reduce collectors |

### WARNING Alerts (Slack with lower urgency)
| Rule | Threshold | Action |
|------|-----------|--------|
| HighMemoryUsage | >80% RAM | Monitor trend, plan scaling |
| HighSwapUsage | >20% swap | Investigate memory leaks |
| DiskSpaceLow | <10% available | Cleanup or expand |
| HighErrorRate | >5% 4xx+5xx | Investigate API issues |

### INFO Alerts (Daily digest)
| Rule | Threshold | Action |
|------|-----------|--------|
| MemoryBaseline | Exceeded historical norm | Informational trend |
| PeakLoadMetrics | Daily summary | Track trends |

---

## 📋 Acceptance Criteria (Per Phase)

### Phase 1 ✅
- [ ] CX23 fully upgraded (Ubuntu 24.04)
- [ ] Docker installed and verified
- [ ] docker-compose.yml created
- [ ] Prometheus container running
- [ ] Grafana container running
- [ ] AlertManager container running
- [ ] All 3 ports accessible (9090, 3000, 9093)

### Phase 2 ✅
- [ ] node-exporter installed on main server
- [ ] Systemd service enabled and running
- [ ] Metrics accessible: curl http://localhost:9100/metrics
- [ ] Prometheus successfully scraping (graph shows data)

### Phase 3 ✅
- [ ] AlertManager webhook configured
- [ ] 5 alert rules tested and firing
- [ ] Each alert generates Slack message (tested)
- [ ] Slack channels created and routing working

### Phase 4 ✅
- [ ] 4 Grafana dashboards created
- [ ] All dashboards showing live data
- [ ] Refresh rates optimized (15-30 sec)
- [ ] Dashboards accessible without login (read-only)

### Phase 5 ✅
- [ ] Load test scripts created (realistic + stress)
- [ ] Realistic test baseline captured (30 min)
- [ ] Stress test breaking point identified
- [ ] Graphs showing memory trajectory
- [ ] Capacity limits documented

### Phase 6 ✅
- [ ] 6 documentation files created
- [ ] All procedures tested and verified
- [ ] Links added to main README
- [ ] PRODUCTION_BLOCKERS.md updated with completion

---

## 🔗 Related Issues & Documents

**Blockers Addressed:**
- [PRODUCTION_BLOCKERS.md](PRODUCTION_BLOCKERS.md) - Blocker #1: Memory Monitoring & Alerting

**Verification:**
- [CODE_QUALITY_AUDIT_REPORT.md](CODE_QUALITY_AUDIT_REPORT.md) - Code quality ✅ verified

**GitHub Issues (Sub-tasks):**
- #25 - PHASE 1: CX23 Server Setup
- #26 - PHASE 2: node-exporter Setup
- #27 - PHASE 3: AlertManager & Slack Integration
- #28 - PHASE 4: Grafana Dashboards
- #29 - PHASE 5: Load Testing
- #30 - PHASE 6: Documentation

---

## 🎯 Execution Strategy

**Starting Point: PHASE 1 (CX23 Setup)**

Current prerequisites met:
- ✅ CX23 server clean and ready
- ✅ Slack workspace ready
- ✅ All phases planned and documented

**Next Action:** Begin Phase 1 tasks (Issue #25)

---

## 📞 Contact & Support

- **Project Lead:** Claude Code (Executing)
- **Owner:** Leo (Decision making)
- **Slack Workspace:** SYNCTACLES
- **GitHub Issues:** See Phase issues (#25-30)

---

## 📝 Notes

### Design Decisions
- **Separate monitoring server:** Allows monitoring to stay up if main app crashes
- **Docker deployment:** Easier management, isolation, scalable
- **80% memory threshold:** Conservative, allows time to scale
- **Both realistic + stress tests:** Understand both normal and breaking behavior
- **Comprehensive documentation:** Operational readiness after project

### Trade-offs Considered
- **SaaS vs Self-hosted:** Self-hosted saves cost, full control (chosen)
- **InfluxDB vs Prometheus:** Prometheus is simpler for this use case
- **Email vs Slack:** Slack is real-time, team-visible (chosen)

### Future Enhancements
- Multi-region monitoring (if scaling horizontally)
- Custom metrics from application code
- Predictive scaling based on load patterns
- Integration with PagerDuty (if needed later)

---

**Last Updated:** 2026-01-06
**Status:** Ready for Phase 1 Execution
**Next Milestone:** Phase 1 Complete (CX23 Setup)
