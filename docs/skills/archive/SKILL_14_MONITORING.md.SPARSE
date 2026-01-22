# SKILL 14 — MONITORING INFRASTRUCTURE

Observability, Dashboards, and Alerting
Version: 1.0 (2026-01-08)

---

## PURPOSE

Document de monitoring infrastructuur voor SYNCTACLES.

---

## KEY INFORMATION

### Monitoring Server

| Aspect | Value |
|--------|-------|
| URL | https://monitor.synctacles.com/ |
| Stack | Grafana + Prometheus |
| Proxy | Caddy |

**⚠️ BELANGRIJK:** Grafana draait NIET op ENIN-NL (API server). Gebruik altijd monitor.synctacles.com.

---

## Grafana

**Access:** https://monitor.synctacles.com/

**Dashboards:**
- Services Status
- [TODO: andere dashboards]

**Data Source:** Prometheus

---

## Prometheus

**Scrape targets:**
- ENIN-NL:8000/metrics (API server)

**Metrics beschikbaar:**
- Python runtime metrics
- FastAPI metrics
- Custom application metrics

---

## API Metrics Endpoint

**URL:** http://localhost:8000/metrics (op ENIN-NL)

**Format:** Prometheus OpenMetrics

**Test:**
```bash
curl -s http://localhost:8000/metrics | head -20
```

---

## Adding New Metrics

1. Add instrumentation in Python code
2. Verify metric in /metrics endpoint
3. Prometheus scrapes automatically
4. Create Grafana panel with PromQL

---

## Related Documents

- docs/INFRASTRUCTURE.md - Server layout
- docs/MONITORING.md - Detailed monitoring docs
- SKILL_02 - API architecture
