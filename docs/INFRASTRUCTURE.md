# SYNCTACLES Infrastructure

## Server Overview

| Server | Hostname | IP | Purpose |
|--------|----------|-----|---------|
| API Production | ENIN-NL | 135.181.255.83 | API, Database, Collectors, Normalizers |
| Monitoring | monitor.synctacles.com | [IP] | Grafana, Prometheus |

---

## API Server (ENIN-NL)

**Hostname:** ENIN-NL
**IP:** 135.181.255.83

**Services:**
- energy-insights-nl-api (FastAPI, port 8000)
- PostgreSQL (port 5432)
- energy-insights-nl-collector (systemd timer)
- energy-insights-nl-importer (systemd timer)
- energy-insights-nl-normalizer (systemd timer)

**Endpoints:**
- Internal: http://localhost:8000
- External: https://api.synctacles.com (via reverse proxy)
- Metrics: http://localhost:8000/metrics (Prometheus format)

---

## Monitoring Server

**URL:** https://monitor.synctacles.com/
**Proxy:** Caddy

**Services:**
- Grafana (dashboards, alerting)
- Prometheus (metrics collection)

**Access:** Login required

---

## Network Diagram

```
┌─────────────────┐
│   End Users     │
└────────┬────────┘
         │ HTTPS
    ┌────▼─────┐
    │ Home     │ BYO keys: TenneT, Enever
    │ Assistant│
    └────┬─────┘
         │ HTTPS
    ┌────▼──────────┐
    │  API Server   │ ENIN-NL
    │  :8000        │ ENTSO-E, Energy-Charts
    └────┬──────────┘
         │ Prometheus scrape
    ┌────▼──────────┐
    │  Monitoring   │ monitor.synctacles.com
    │  Grafana      │
    └───────────────┘
```

---

## Important Notes

- **Grafana draait NIET op ENIN-NL** - gebruik monitor.synctacles.com
- **Metrics endpoint** beschikbaar op API server voor Prometheus scraping
- **Database** alleen lokaal toegankelijk (localhost:5432)
