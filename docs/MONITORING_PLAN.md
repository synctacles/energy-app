# SYNCTACLES Monitoring Plan

**Issue:** #6 - Setup Monitoring & Alerting
**Status:** Plan
**Date:** 2026-01-23

---

## 1. OVERZICHT

### Huidige Situatie
De API heeft al basis Prometheus metrics:
- `/metrics` - HTTP request metrics (requests_total, duration)
- `/v1/pipeline/metrics` - Pipeline-specifieke metrics (timer status, data freshness)
- `/v1/pipeline/health` - JSON health endpoint

### Doel
Centraal monitoring dashboard op een dedicated monitoring server met:
- Real-time visibility op alle SYNCTACLES componenten
- Pro-actieve alerting bij problemen
- Historische data voor trend-analyse

### Scope
| Omgeving | Metrics | Dashboards | Alerting |
|----------|---------|------------|----------|
| **PROD** | Ja | Ja | Ja (critical + warning) |
| **DEV** | Ja | Ja | Nee (alleen logging) |

> DEV wordt wel gescraped voor vergelijkende dashboards, maar genereert geen alerts om alert fatigue te voorkomen.

---

## 2. WAT MOET GEMONITORD WORDEN

### 2.1 API Health (PROD + DEV)

| Metric | Beschrijving | Threshold Alert |
|--------|--------------|-----------------|
| `http_requests_total` | Totaal requests per endpoint | N/A (info) |
| `http_request_duration_seconds` | Response latency | P95 > 2s |
| `http_requests_total{status=5xx}` | Server errors | > 5/min |
| `http_requests_total{status=4xx}` | Client errors | > 100/min |
| API uptime | Health endpoint bereikbaar | DOWN > 1 min |

### 2.2 Pipeline Health

| Metric | Beschrijving | Threshold Alert |
|--------|--------------|-----------------|
| `pipeline_timer_status` | Systemd timer actief | 0 (stopped) |
| `pipeline_timer_last_trigger_minutes` | Minuten sinds laatste run | > 120 min |
| `pipeline_data_freshness_minutes` | Data ouderdom | > 90 min (STALE) |
| `pipeline_data_status` | Data status code | > 0 (niet FRESH) |
| `pipeline_raw_norm_gap_minutes` | Gap raw→normalized | > 30 min |

### 2.3 Database Health

| Metric | Beschrijving | Threshold Alert |
|--------|--------------|-----------------|
| PostgreSQL connections | Actieve connecties | > 80% max |
| PostgreSQL slow queries | Queries > 1s | > 10/min |
| Database size | Totale grootte | > 80% disk |
| Replication lag | (indien HA) | > 10s |
| Table row counts | Records per tabel | Afname > 10% |

### 2.4 System Resources (per server)

| Metric | Beschrijving | Threshold Alert |
|--------|--------------|-----------------|
| CPU usage | CPU percentage | > 80% sustained 10 min |
| Memory usage | RAM percentage | > 85% |
| Disk usage | Disk percentage | > 80% |
| Network I/O | Bytes in/out | Anomaly detection |
| Process count | Gunicorn workers | < 8 (PROD) |

### 2.5 SSL Certificates

| Metric | Beschrijving | Threshold Alert |
|--------|--------------|-----------------|
| Certificate expiry | Dagen tot expiry | < 30 dagen (warning) |
| Certificate expiry | Dagen tot expiry | < 7 dagen (critical) |
| Certificate valid | Cert geldig | Invalid = critical |

### 2.6 Business Events (Info Alerts)

| Event | Beschrijving | Alert Type |
|-------|--------------|------------|
| API Key Created | Nieuwe gebruiker registratie | Info (email) |
| API Key Revoked | Key ingetrokken | Info (email) |
| Rate Limit Hit | Gebruiker bereikt limiet | Warning (log) |

### 2.7 Collector/Importer Status

| Metric | Beschrijving | Threshold Alert |
|--------|--------------|-----------------|
| ENTSO-E collector | A44 data ophalen | Failure > 2x |
| Frank collector | Frank Energie data | Failure > 2x |
| Importer success | Records geimporteerd | 0 records > 2 runs |
| Normalizer success | Records genormaliseerd | 0 records > 2 runs |

### 2.8 External Dependencies

| Metric | Beschrijving | Threshold Alert |
|--------|--------------|-----------------|
| ENTSO-E API | Bereikbaarheid | DOWN > 30 min |
| Frank Energie API | Bereikbaarheid | DOWN > 30 min |
| GitHub API | Voor auto-update checks | DOWN > 1 uur |

---

## 3. MONITORING STACK ARCHITECTUUR

### 3.1 Componenten

```
┌─────────────────────────────────────────────────────────────────┐
│                    MONITORING SERVER                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Prometheus  │→→│   Grafana    │→→│ Alertmanager │          │
│  │  (scraper)   │  │ (dashboards) │  │   (alerts)   │          │
│  └──────┬───────┘  └──────────────┘  └──────┬───────┘          │
│         │                                    │                   │
│         │ scrape /metrics                    │ alerts            │
└─────────┼────────────────────────────────────┼───────────────────┘
          │                                    │
          ▼                                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐
│   PROD Server   │  │   DEV Server    │  │   Email/    │
│   :8000/metrics │  │   :8000/metrics │  │   Slack     │
└─────────────────┘  └─────────────────┘  └─────────────┘
```

### 3.2 Prometheus Configuratie

```yaml
# prometheus.yml
global:
  scrape_interval: 30s
  evaluation_interval: 30s

scrape_configs:
  # PROD API metrics
  - job_name: 'synctacles-prod-api'
    static_configs:
      - targets: ['46.62.212.227:8000']
    metrics_path: /metrics

  # PROD Pipeline metrics
  - job_name: 'synctacles-prod-pipeline'
    static_configs:
      - targets: ['46.62.212.227:8000']
    metrics_path: /v1/pipeline/metrics

  # DEV API metrics
  - job_name: 'synctacles-dev-api'
    static_configs:
      - targets: ['135.181.255.83:8000']
    metrics_path: /metrics

  # DEV Pipeline metrics
  - job_name: 'synctacles-dev-pipeline'
    static_configs:
      - targets: ['135.181.255.83:8000']
    metrics_path: /v1/pipeline/metrics

  # Node Exporter (system metrics)
  - job_name: 'node-prod'
    static_configs:
      - targets: ['46.62.212.227:9100']

  - job_name: 'node-dev'
    static_configs:
      - targets: ['135.181.255.83:9100']

  # PostgreSQL Exporter
  - job_name: 'postgres-prod'
    static_configs:
      - targets: ['46.62.212.227:9187']

  - job_name: 'postgres-dev'
    static_configs:
      - targets: ['135.181.255.83:9187']

  # SSL Certificate monitoring (Blackbox Exporter)
  - job_name: 'ssl-certs'
    metrics_path: /probe
    params:
      module: [http_2xx]
    static_configs:
      - targets:
        - https://api.synctacles.com
        - https://synctacles.com
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: localhost:9115  # Blackbox exporter
```

### 3.3 Alerting Rules

```yaml
# alerts.yml
groups:
  - name: synctacles-api
    rules:
      - alert: APIDown
        expr: up{job=~"synctacles-.*-api"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "SYNCTACLES API is down"

      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High 5xx error rate"

      - alert: SlowResponses
        expr: histogram_quantile(0.95, http_request_duration_seconds_bucket) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API P95 latency > 2s"

  - name: synctacles-pipeline
    rules:
      - alert: TimerStopped
        expr: pipeline_timer_status == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Pipeline timer {{ $labels.timer }} is stopped"

      - alert: DataStale
        expr: pipeline_data_status > 0
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "Data for {{ $labels.source }} is stale"

      - alert: PipelineGap
        expr: pipeline_raw_norm_gap_minutes > 30
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Raw-to-normalized pipeline gap > 30 min"

  - name: synctacles-system
    rules:
      - alert: HighCPU
        expr: 100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage on {{ $labels.instance }}"

      - alert: HighMemory
        expr: (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 > 85
        for: 5m
        labels:
          severity: warning

      - alert: DiskFull
        expr: (1 - node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 > 80
        for: 5m
        labels:
          severity: critical

  - name: synctacles-certificates
    rules:
      - alert: CertificateExpiringSoon
        expr: probe_ssl_earliest_cert_expiry - time() < 86400 * 30
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "SSL certificate expires in < 30 days"

      - alert: CertificateExpiringCritical
        expr: probe_ssl_earliest_cert_expiry - time() < 86400 * 7
        for: 1h
        labels:
          severity: critical
        annotations:
          summary: "SSL certificate expires in < 7 days!"

      - alert: CertificateInvalid
        expr: probe_ssl_earliest_cert_expiry == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "SSL certificate is invalid or expired"
```

---

## 4. GRAFANA DASHBOARDS

### 4.1 Dashboard Structuur

```
SYNCTACLES Monitoring
├── Overview Dashboard (startpagina)
│   ├── Status indicators: PROD ✓ DEV ✓
│   ├── Last data update: 2 min ago
│   ├── Active alerts: 0
│   └── Quick links naar detail dashboards
│
├── API Performance Dashboard
│   ├── Requests per endpoint (grafiek)
│   ├── Response time histogram
│   ├── Error rate trend
│   └── Status code breakdown
│
├── Pipeline Health Dashboard
│   ├── Timer status indicators
│   ├── Data freshness gauges
│   ├── Pipeline gap timeline
│   └── Collector/Importer logs
│
├── Database Dashboard
│   ├── Connection pool usage
│   ├── Query duration histogram
│   ├── Table sizes
│   └── Row counts per tabel
│
└── System Resources Dashboard
    ├── CPU/Memory/Disk per server
    ├── Network I/O
    └── Process list
```

### 4.2 Overview Dashboard Panels

| Panel | Type | Data Source |
|-------|------|-------------|
| PROD Status | Stat (green/red) | Prometheus: up{job="synctacles-prod-api"} |
| DEV Status | Stat (green/red) | Prometheus: up{job="synctacles-dev-api"} |
| Data Freshness | Gauge | pipeline_data_freshness_minutes |
| Request Rate | Graph | rate(http_requests_total[5m]) |
| Error Rate | Graph | rate(http_requests_total{status=~"5.."}[5m]) |
| Active Alerts | Alert list | Prometheus alerts |
| Last 24h Uptime | Stat (%) | avg_over_time(up[24h]) * 100 |

### 4.3 Data Freshness Panel (Visual)

```
┌─────────────────────────────────────────────────┐
│  DATA FRESHNESS                                  │
│                                                  │
│  A44 Prices     [██████████████████████] 2 min  │
│  Frank Prices   [████████████████░░░░░░] 45 min │
│                                                  │
│  ● FRESH (< 90 min)  ○ STALE  ○ UNAVAILABLE    │
└─────────────────────────────────────────────────┘
```

---

## 5. BENODIGDE INSTALLATIES

### 5.1 Op Monitoring Server (77.42.41.135)

De monitoring server heeft al een werkende Docker stack. De configuratie wordt beheerd via:
- `/opt/monitoring/docker-compose.yml`
- `/opt/monitoring/prometheus/` (config files)
- `/opt/monitoring/alertmanager/` (config files)

**Deployment:**
```bash
# Vanaf DEV server, via cc-hub
ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'cd /opt/monitoring && docker-compose down && docker-compose up -d'"

# Reload Prometheus config (zonder restart)
ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'curl -X POST http://localhost:9090/-/reload'"
```

**Docker Compose stack:**
```yaml
# /opt/monitoring/docker-compose.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus:/etc/prometheus
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--web.enable-lifecycle'  # Enable reload via API
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
    restart: unless-stopped

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager:/etc/alertmanager
    restart: unless-stopped

  blackbox-exporter:
    image: prom/blackbox-exporter:latest
    ports:
      - "9115:9115"
    volumes:
      - ./blackbox:/etc/blackbox_exporter
    restart: unless-stopped

volumes:
  prometheus_data:
  grafana_data:
```

### 5.2 Op PROD + DEV Servers

```bash
# Node Exporter (system metrics)
sudo apt install prometheus-node-exporter
sudo systemctl enable prometheus-node-exporter

# PostgreSQL Exporter
docker run -d \
  --name postgres-exporter \
  -p 9187:9187 \
  -e DATA_SOURCE_NAME="postgresql://user:pass@localhost:5432/db?sslmode=disable" \
  prometheuscommunity/postgres-exporter
```

### 5.3 Firewall Rules

```bash
# Op PROD/DEV: Allow Prometheus scraping van monitoring server
sudo ufw allow from [MONITORING_IP] to any port 8000   # API metrics
sudo ufw allow from [MONITORING_IP] to any port 9100   # Node exporter
sudo ufw allow from [MONITORING_IP] to any port 9187   # Postgres exporter

# Op Monitoring server: Allow external access to Grafana
sudo ufw allow 3000/tcp  # Grafana (of via reverse proxy op 443)
```

---

## 6. ALERTING CHANNELS

### 6.1 Slack Webhooks (al geconfigureerd)

| Severity | Channel | Webhook ID |
|----------|---------|------------|
| Critical | #critical-alerts | `T0A6YTF8X45/B0A7A4HDFAQ/ehALUsVzas8as0ZaeB5bHkVu` |
| Warning | #warnings | `T0A6YTF8X45/B0A80RKSSPJ/B77Gri7FrYvGUwUEjl9Rvr2o` |
| Info | #info-metrics | `T0A6YTF8X45/B0A705BFFA7/XFA1G62gWEJkDngkw7LVO3np` |

### 6.2 Alertmanager Config

```yaml
# alertmanager.yml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'slack-warnings'
  routes:
    - match:
        severity: critical
      receiver: 'slack-critical'
      repeat_interval: 1h
    - match:
        severity: warning
      receiver: 'slack-warnings'
      repeat_interval: 4h
    - match:
        severity: info
      receiver: 'slack-info'
      repeat_interval: 12h

receivers:
  - name: 'slack-critical'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/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'
        channel: '#critical-alerts'
        send_resolved: true

  - name: 'slack-warnings'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/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'
        channel: '#warnings'
        send_resolved: true

  - name: 'slack-info'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/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'
        channel: '#info-metrics'
        send_resolved: true

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'instance']
```

---

## 7. IMPLEMENTATIE FASES

### Fase 1: Basis Setup (Week 1)
- [ ] Monitoring server provisionen
- [ ] Prometheus + Grafana installeren
- [ ] Node Exporter op PROD + DEV
- [ ] Basis dashboard met uptime

### Fase 2: Pipeline Monitoring (Week 2)
- [ ] Pipeline metrics dashboard
- [ ] Data freshness alerts
- [ ] Timer status alerts

### Fase 3: Database Monitoring (Week 3)
- [ ] PostgreSQL Exporter setup
- [ ] Database dashboard
- [ ] Connection pool alerts

### Fase 4: Alerting (Week 4)
- [ ] Alertmanager setup
- [ ] Email configuratie
- [ ] Alert rules fine-tuning
- [ ] Runbook documentatie

---

## 8. METRICS DIE TOEGEVOEGD MOETEN WORDEN

### 8.1 Collector Metrics (nieuw)

```python
# In collectors - toe te voegen
collector_runs_total = Counter(
    "collector_runs_total",
    "Total collector runs",
    ["collector", "status"]  # status: success/failure
)

collector_records_fetched = Gauge(
    "collector_records_fetched",
    "Records fetched in last run",
    ["collector"]
)

collector_duration_seconds = Histogram(
    "collector_duration_seconds",
    "Collector run duration",
    ["collector"]
)
```

### 8.2 Database Metrics (nieuw)

```python
# In API - toe te voegen
db_table_rows = Gauge(
    "db_table_rows",
    "Row count per table",
    ["table"]
)

db_query_duration_seconds = Histogram(
    "db_query_duration_seconds",
    "Database query duration",
    ["query_type"]  # select/insert/update
)
```

### 8.3 Business Events Metrics (nieuw)

```python
# In auth.py - toe te voegen
from prometheus_client import Counter, Info

api_key_events = Counter(
    "api_key_events_total",
    "API key lifecycle events",
    ["event_type"]  # created, revoked, expired
)

# Bij key aanmaak:
api_key_events.labels(event_type="created").inc()

# Bij key revoke:
api_key_events.labels(event_type="revoked").inc()
```

**Info alert via webhook (alternatief):**
```python
# In auth.py - bij create_api_key()
import aiohttp

async def notify_key_created(email: str, key_name: str):
    """Send info notification when API key is created."""
    webhook_url = os.getenv("ALERTMANAGER_WEBHOOK")
    if webhook_url:
        payload = {
            "alerts": [{
                "status": "firing",
                "labels": {
                    "alertname": "APIKeyCreated",
                    "severity": "info"
                },
                "annotations": {
                    "summary": f"New API key created: {key_name}",
                    "description": f"User {email} created a new API key"
                }
            }]
        }
        async with aiohttp.ClientSession() as session:
            await session.post(webhook_url, json=payload)
```

---

## 9. MONITORING INFRASTRUCTURE

### 9.1 Monitoring Server

| Property | Value |
|----------|-------|
| **Hostname** | monitoring.synctacles.com |
| **IP** | 77.42.41.135 |
| **SSH Access** | Via cc-hub: `ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135"` |
| **Config Path** | `/opt/monitoring/` |

### 9.2 Server Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│  MONITORING SERVER (77.42.41.135)                                    │
│  monitoring.synctacles.com                                           │
│                                                                      │
│  /opt/monitoring/                                                    │
│  ├── docker-compose.yml                                              │
│  ├── prometheus/                                                     │
│  │   ├── prometheus.yml                                              │
│  │   └── alerts.yml                                                  │
│  ├── alertmanager/                                                   │
│  │   └── alertmanager.yml                                            │
│  └── blackbox/                                                       │
│      └── blackbox.yml                                                │
│                                                                      │
│  Services:                                                           │
│  ├── prometheus:9090                                                 │
│  ├── grafana:3000                                                    │
│  ├── alertmanager:9093                                               │
│  └── blackbox-exporter:9115                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### 9.3 Target Servers

| Server | IP | Role | Alerts |
|--------|-----|------|--------|
| **PROD** | 46.62.212.227 | Production | Yes (critical + warning) |
| **DEV** | 135.181.255.83 | Development | No (metrics only) |

### 9.4 Ports to Scrape

| Port | Service | PROD | DEV |
|------|---------|------|-----|
| 8000 | API /metrics | ✓ | ✓ |
| 8000 | Pipeline /v1/pipeline/metrics | ✓ | ✓ |
| 9100 | Node Exporter | ✓ | ✓ |
| 9187 | PostgreSQL Exporter | ✓ | ✓ |

### 9.5 Firewall Configuratie

#### Monitoring Server (77.42.41.135)

**Hetzner Cloud Firewall:**
| Richting | Poort | Protocol | Bron | Doel |
|----------|-------|----------|------|------|
| Inkomend | 22 | TCP | Jouw IP, cc-hub IP | SSH |
| Inkomend | 443 | TCP | Jouw IP | Grafana HTTPS |

**Linux Firewall (ufw) - Certbot met tijdelijke poort 80:**
```bash
# Certbot renewal met pre/post hooks
# Poort 80 wordt ALLEEN geopend tijdens renewal

sudo certbot renew \
  --pre-hook "ufw allow 80/tcp" \
  --post-hook "ufw deny 80/tcp"

# Of configureer in /etc/letsencrypt/cli.ini:
# pre-hook = ufw allow 80/tcp
# post-hook = ufw deny 80/tcp
```

**Uitgaand (voor scraping):**
| Poort | Protocol | Doel |
|-------|----------|------|
| 8000 | TCP | DEV + PROD API metrics |
| 9100 | TCP | DEV + PROD node exporter |
| 443 | TCP | SSL certificate checks |

#### DEV Server (135.181.255.83)

**Hetzner Cloud Firewall:**
| Richting | Poort | Protocol | Bron |
|----------|-------|----------|------|
| Inkomend | 22 | TCP | Jouw IP, cc-hub IP |
| Inkomend | 443 | TCP | 0.0.0.0/0 (API publiek) |
| Inkomend | 8000 | TCP | 77.42.41.135 (monitoring) |
| Inkomend | 9100 | TCP | 77.42.41.135 (monitoring) |

**Linux Firewall (ufw) - Certbot:**
```bash
# Zelfde aanpak als monitoring server
sudo certbot renew \
  --pre-hook "ufw allow 80/tcp" \
  --post-hook "ufw deny 80/tcp"
```

#### PROD Server (46.62.212.227)

**Hetzner Cloud Firewall:**
| Richting | Poort | Protocol | Bron |
|----------|-------|----------|------|
| Inkomend | 22 | TCP | Jouw IP, cc-hub IP |
| Inkomend | 443 | TCP | 0.0.0.0/0 (API publiek) |
| Inkomend | 8000 | TCP | 77.42.41.135 (monitoring) |
| Inkomend | 9100 | TCP | 77.42.41.135 (monitoring) |

**Linux Firewall (ufw) - Certbot:**
```bash
sudo certbot renew \
  --pre-hook "ufw allow 80/tcp" \
  --post-hook "ufw deny 80/tcp"
```

#### Certbot Hooks Installeren (alle servers)

```bash
# Maak hook scripts
sudo tee /etc/letsencrypt/renewal-hooks/pre/open-port-80.sh << 'EOF'
#!/bin/bash
ufw allow 80/tcp
EOF

sudo tee /etc/letsencrypt/renewal-hooks/post/close-port-80.sh << 'EOF'
#!/bin/bash
ufw deny 80/tcp
EOF

# Maak executable
sudo chmod +x /etc/letsencrypt/renewal-hooks/pre/open-port-80.sh
sudo chmod +x /etc/letsencrypt/renewal-hooks/post/close-port-80.sh

# Test
sudo certbot renew --dry-run
```

---

## 10. REFERENTIES

- Prometheus docs: https://prometheus.io/docs/
- Grafana dashboards: https://grafana.com/grafana/dashboards/
- Node Exporter: https://github.com/prometheus/node_exporter
- PostgreSQL Exporter: https://github.com/prometheus-community/postgres_exporter
