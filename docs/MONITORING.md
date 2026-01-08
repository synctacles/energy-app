# SYNCTACLES Monitoring

## Overview

SYNCTACLES gebruikt een dedicated monitoring server voor observability.

**URL:** https://monitor.synctacles.com/

---

## Stack

| Component | Purpose | Location |
|-----------|---------|----------|
| Grafana | Dashboards, alerting | monitor.synctacles.com |
| Prometheus | Metrics collection | monitor.synctacles.com |
| Caddy | Reverse proxy, HTTPS | monitor.synctacles.com |

---

## Metrics Collection

### API Metrics Endpoint

**URL:** http://ENIN-NL:8000/metrics (internal)
**Format:** Prometheus OpenMetrics

**Beschikbare metrics:**
- Python runtime (GC, memory, CPU)
- FastAPI request metrics
- Custom application metrics

### Prometheus Scraping

Prometheus scraped de API metrics endpoint periodiek.

**Verify scraping werkt:**
1. Login op monitor.synctacles.com
2. Ga naar Explore
3. Query: `up{job="synctacles-api"}`
4. Moet `1` returnen

---

## Grafana Dashboards

### Bestaande Dashboards

- Services Status (zie screenshot in docs)
- [TODO: list andere dashboards]

### Dashboard Toevoegen

1. Login op https://monitor.synctacles.com/
2. Dashboards → New Dashboard
3. Add visualization
4. Select Prometheus data source
5. Write PromQL query
6. Save dashboard

---

## Alerting

[TODO: Document alert rules]

---

## Troubleshooting

**Grafana niet bereikbaar:**
- Check https://monitor.synctacles.com/
- Verify Caddy proxy status

**Metrics niet zichtbaar in Grafana:**
1. Check API metrics endpoint: `curl http://ENIN-NL:8000/metrics`
2. Check Prometheus scrape config
3. Check Prometheus targets in Grafana
