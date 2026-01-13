# HANDOFF: CC → CAI - Monitoring Expansion Planning

**Van:** Claude Code (CC)
**Aan:** CAI
**Datum:** 2026-01-13 00:15 UTC
**Onderwerp:** Prometheus Monitoring Uitbreiding - Coefficient Server Toevoegen

---

## CONTEXT

Tijdens systeem audit (2026-01-12) zijn beide servers volledig geïnventariseerd:
- ✅ Synctacles API (135.181.255.83) - AL gemonitord
- ⚠️ Coefficient Engine (91.99.150.36) - NIET gemonitord

Leo vroeg: "Kunnen we ook 2 servers monitoren?"

Dit document bevat alle monitoring punten voor beide servers om een complete monitoring strategie te maken.

---

## MONITORING SERVER TOEGANG

### Huidige Configuratie
```
Server:   ENIN-Monitoring (77.42.41.135)
Hostname: monitor.synctacles.com
Stack:    Grafana + Prometheus (Docker)
Config:   /opt/monitoring/prometheus/prometheus.yml
Alerts:   /opt/monitoring/prometheus/alerts.yml
```

### SSH Toegang
```bash
# Vanaf ENIN-NL server (135.181.255.83):
sudo ssh -i /root/.ssh/id_cx23 monitoring@77.42.41.135

# SSH config entry (in /root/.ssh/config):
Host cx23
    HostName 77.42.41.135
    User root
    IdentityFile ~/.ssh/id_cx23
```

### Docker Containers
```
grafana:     grafana/grafana:latest (port 3000)
prometheus:  prom/prometheus:latest (port 9090)
```

---

## SERVER 1: SYNCTACLES API (135.181.255.83) ✅

### Huidige Monitoring Status: ACTIEF

#### System Metrics (Node Exporter)
```yaml
Port: 9100
Job: "node-exporter-main"
Labels:
  - instance: "enin-main"
  - environment: "production"
Metrics:
  - CPU usage
  - Memory available/used
  - Disk space
  - Network I/O
  - System load
```

#### API Health Check (Blackbox Exporter)
```yaml
Job: "blackbox-http"
Endpoint: https://enin.xteleo.nl/health
Module: http_2xx
Metrics:
  - probe_success (0/1)
  - probe_duration_seconds
  - probe_http_status_code
```

#### SSL Certificate Monitoring
```yaml
Job: "blackbox-ssl"
Target: enin.xteleo.nl:443
Module: tls_connect
Metrics:
  - probe_ssl_earliest_cert_expiry
  - Days until expiry
```

#### Pipeline Health Metrics ⭐ NATIVE PROMETHEUS
```yaml
Job: "pipeline-health"
Endpoint: https://135.181.255.83:443/v1/pipeline/metrics
Scrape: 30s
Metrics:
  - pipeline_timer_status{timer} (1=active, 0=stopped)
  - pipeline_timer_last_trigger_minutes{timer}
  - pipeline_data_freshness_minutes{source}
  - pipeline_data_status{source} (0=FRESH, 1=STALE, 2=UNAVAILABLE, 3=NO_DATA)
  - pipeline_raw_norm_gap_minutes{source}

Timers monitored:
  - collector
  - importer
  - normalizer
  - health

Data sources monitored:
  - a44 (Day-ahead prices)
```

#### Systemd Services (via node_exporter)
```yaml
Metrics: node_systemd_unit_state{name,state}

Monitored services:
  - energy-insights-nl-api.service
  - energy-insights-nl-collector.service
  - energy-insights-nl-importer.service
  - energy-insights-nl-normalizer.service

Monitored timers:
  - energy-insights-nl-collector.timer
  - energy-insights-nl-importer.timer
  - energy-insights-nl-normalizer.timer
  - energy-insights-nl-health.timer
```

#### API Endpoints (beschikbaar maar NIET gescraped)
```
/health                    - Health check (AL via blackbox)
/metrics                   - Main Prometheus metrics
/api/v1/energy-action      - Core endpoint (niet gemonitord)
/api/v1/prices             - Price endpoint (niet gemonitord)
/v1/pipeline/health        - JSON health (niet gemonitord, alleen /metrics)
/cache/stats               - Cache statistics (niet gemonitord)
```

#### Database (PostgreSQL)
```
Database: energy_insights_nl
User: energy_insights_nl
Tables:
  - norm_entso_e_a44 (1,432 records, 440 KB)
  - raw_entso_e_a44 (1,528 records, 216 KB)
  - price_cache (3,408 records, 512 KB)
  - raw_tennet_balance (56 MB)
  - norm_tennet_balance (25 MB)

Niet gemonitord:
  - Database size growth
  - Query performance
  - Connection pool status
```

---

## SERVER 2: COEFFICIENT ENGINE (91.99.150.36) ⚠️

### Huidige Monitoring Status: NIET ACTIEF

#### System Metrics (Node Exporter) ✅ DRAAIT AL
```yaml
Port: 9100 ✅ ACTIEF
Process: node_exporter (pid 746)
Status: Draait, maar NIET geconfigureerd in Prometheus

Beschikbare metrics:
  - node_cpu_seconds_total
  - node_memory_MemAvailable_bytes
  - node_memory_MemTotal_bytes
  - node_filesystem_avail_bytes
  - node_disk_io_time_seconds_total
  - node_network_receive_bytes_total
  - node_network_transmit_bytes_total
  - node_load1, node_load5, node_load15
```

**ACTIE:** Voeg scrape target toe aan Prometheus config.

#### API Health Check
```yaml
Endpoint: http://91.99.150.36:8080/health
Response:
  {
    "status": "ok",
    "service": "coefficient-engine",
    "version": "2.0.0",
    "timestamp": "2026-01-13T00:13:52.265712+00:00"
  }

Kan gemonitord worden via:
  1. Blackbox exporter (zoals Synctacles)
  2. Direct HTTP scrape
```

#### Coefficient API Endpoints

##### A. Core Endpoints (Werkend, Niet Gemonitord)
```
GET /health
  Status: ✅ Actief
  Response: {"status": "ok", "version": "2.0.0"}
  Monitoring: Health check

GET /coefficient?country={country}&month={month}&day_type={day_type}&hour={hour}
  Status: ✅ Actief
  Response: {
    "slope": 1.2747,
    "intercept": 0.150204,
    "confidence": 90,
    "sample_size": 70,
    "last_calibrated": "2026-01-12T15:09:42.129427+00:00"
  }
  Monitoring: Response time, success rate

GET /coefficient/current
  Status: ✅ Actief
  Monitoring: Current hour coefficient availability

GET /coefficient/all
  Status: ✅ Actief
  Response: {"count": 576, "coefficients": {...}}
  Monitoring: Total coefficient count, lookup table health
```

##### B. Calibration Endpoints ⭐ KRITIEK
```
GET /calibration/status
  Status: ✅ Actief
  Response: {
    "timestamp": "2026-01-12T19:00:25.618547+00:00",
    "period": "7 days",
    "status": [
      {
        "hour": 3,
        "day_type": "weekday",
        "avg_drift_pct": 19.26,        ← MONITORING PUNT!
        "max_drift_pct": 19.26,
        "sample_count": 2,
        "last_check": "2026-01-12T03:44:49.763983+00:00",
        "needs_attention": true         ← ALERT TRIGGER!
      }
    ]
  }

  MONITORING PUNTEN:
    - avg_drift_pct per hour (threshold: >10%)
    - needs_attention boolean
    - sample_count (te laag = onbetrouwbaar)
    - last_check age (te oud = geen recente validatie)

POST /calibration/run
  Status: ✅ Actief
  Response: {
    "status": "completed",
    "stored": {"slope": 1.285, "intercept": 0.145499},
    "optimal": {"slope": 1.605, "intercept": 0.113, "r_squared": 0.866},
    "drift": {"slope_pct": 24.91, "intercept_pct": 21.96},
    "recommendation": "update"
  }
  Monitoring: Calibration success rate, drift reduction
```

##### C. Consumer Proxy Endpoints (VPN-dependent) ⭐ KRITIEK
```
GET /internal/consumer/prices
  Status: ✅ Actief (via VPN)
  Response: {
    "timestamp": "2026-01-11T21:00:00Z",
    "source": "consumer-proxy",
    "prices_today": {
      "Frank Energie": [{"hour": 0, "price": 0.183}, ...],
      "Tibber": [...],
      "Vattenfall": [...]
    },
    "prices_tomorrow": {...}
  }
  Monitoring:
    - VPN connectivity
    - Provider availability
    - Price data freshness

GET /internal/consumer/health
  Status: ✅ Actief
  Response: {
    "service": "consumer-proxy",
    "status": "ok",
    "token_configured": true,
    "vpn_configured": true
  }
  Monitoring:
    - VPN status
    - Token validity
    - Service health

GET /consumer/frank
  Status: ✅ Actief
  Response: Frank Energie prices
  Monitoring: Frank API availability
```

#### VPN Tunnel Status ⭐ KRITIEK
```yaml
Interface: pia-split (WireGuard)
Purpose: Route to Enever API (84.46.252.107)
Status: ✅ Actief

Current state:
  Handshake: 1 minute 20 seconds ago
  Transfer: 3.53 MiB RX / 1.09 MiB TX
  Peer: IahOocCJ09Dky+9zgOP6qZUqg/Yntbmz3V/hwb4oFWA=
  Endpoint: 158.173.21.230:1337
  Allowed IPs: 84.46.252.107/32

Monitoring via WireGuard:
  sudo wg show pia-split

Metrics beschikbaar:
  - Handshake age (warn if >180s)
  - Transfer bytes
  - Peer status
  - Route availability

WAAROM KRITIEK:
  Consumer price fallback chain tier 1 & 2 vereisen VPN
  Zonder VPN: alleen tier 3-6 beschikbaar (lagere kwaliteit)
```

#### Systemd Services & Timers
```yaml
Services:
  coefficient-api.service
    Status: enabled, active since 2026-01-12 10:40 UTC
    Monitoring: service state, restart count

Timers (alle static/disabled, via timer):
  enever-collector.timer
    Schedule: Daily 00:30 UTC
    Last: 2026-01-12 15:30 UTC
    Next: 2026-01-13 00:30 UTC
    Monitoring: last trigger, success/failure

  consumer-collector.timer
    Schedule: Daily 00:30 UTC
    Next: 2026-01-13 00:30 UTC
    Monitoring: last trigger, success/failure

  frank-live-collector.timer
    Schedule: Daily 01:00 UTC
    Next: 2026-01-13 01:00 UTC
    Monitoring: last trigger, success/failure

  coefficient-calibration.timer
    Schedule: 2x daily (06:17 and 18:17 UTC)
    Last: 2026-01-12 18:17 UTC
    Next: 2026-01-13 06:17 UTC
    Monitoring: last trigger, calibration success
```

#### Database (PostgreSQL)
```yaml
Database: coefficient_db
User: coefficient
Port: 5432 (local only)

Tables:
  hist_entso_prices
    Records: 2,083,437
    Size: 496 MB
    Latest: 2026-01-12 22:45:00+00
    Growth: ~5000 records/day (prices every 15min × 24h × 14 days retention)

  hist_enever_prices
    Records: 361,039
    Size: 64 MB
    Latest: 2026-01-12 23:00:00+00
    Growth: ~24 records/day

  hist_frank_prices
    Records: 26,616
    Size: 4.5 MB
    Latest: 2026-01-13 22:00:00+00 (forecast)
    Growth: ~24 records/day

  coefficient_lookup
    Records: 576 (12 months × 2 day_types × 24 hours)
    Size: 264 KB
    Latest calibration: 2026-01-12 15:09:42+00
    Update frequency: 2x daily

  enever_prices
    Records: 1,200 (50 days × 24 hours)
    Size: 320 KB
    Latest: 2026-01-12 23:00:00+00

Monitoring punten:
  - Database size growth rate
  - Table bloat
  - Query performance
  - Connection count
  - Data freshness per table
```

**OPTIONEEL:** Installeer postgres_exporter (port 9187)

---

## MONITORING PRIORITEITEN

### Priority 1: CRITICAL (Moet Nu)
```
SERVER 1 (Synctacles):
  ✅ Systemd services (API, timers)
  ✅ Pipeline data freshness
  ✅ System resources (CPU, memory, disk)
  ✅ SSL certificate expiry

SERVER 2 (Coefficient):
  ⚠️ Node-exporter toevoegen aan scrape config
  ⚠️ API health check (blackbox of direct)
  ⚠️ Calibration drift monitoring (/calibration/status)
  ⚠️ VPN tunnel status
```

### Priority 2: IMPORTANT (Deze Week)
```
SERVER 1:
  - API endpoint response times (/api/v1/energy-action)
  - Cache hit rates (/cache/stats)
  - Database query performance

SERVER 2:
  - Consumer proxy health (/internal/consumer/health)
  - Systemd timer triggers (collectors, calibration)
  - Database size growth
  - Coefficient lookup table health
```

### Priority 3: NICE TO HAVE (Later)
```
BEIDE SERVERS:
  - Custom Grafana dashboards
  - Database deep metrics (postgres_exporter)
  - Application-level metrics (custom exporters)
  - Log aggregation (Loki)
  - Distributed tracing

SERVER 2 SPECIFIEK:
  - Coefficient accuracy tracking over time
  - Provider availability rates (Frank, Enever, Tibber)
  - Calibration effectiveness metrics
```

---

## VOORGESTELDE IMPLEMENTATIE

### Stap 1: Basis Monitoring Coefficient (10 min)

**Wijzig:** `/opt/monitoring/prometheus/prometheus.yml`

```yaml
# Add to scrape_configs:

  # Coefficient Server - System Metrics
  - job_name: "node-exporter-coefficient"
    static_configs:
      - targets: ["91.99.150.36:9100"]
        labels:
          instance: "coefficient"
          environment: "production"

  # Coefficient Server - API Health
  - job_name: "blackbox-coefficient-http"
    metrics_path: /probe
    params:
      module: [http_2xx]
    static_configs:
      - targets:
          - http://91.99.150.36:8080/health
        labels:
          environment: "production"
          service: "coefficient-api"
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox:9115
```

**Test:**
```bash
cd /opt/monitoring/prometheus
sudo docker exec prometheus promtool check config /etc/prometheus/prometheus.yml
sudo docker restart prometheus
```

### Stap 2: Calibration Drift Monitoring (30 min)

**Optie A: Custom Exporter** (Aanbevolen)
```python
# /opt/monitoring/exporters/coefficient_exporter.py
# Scrape /calibration/status endpoint
# Export as Prometheus metrics:
#   coefficient_drift_pct{hour,day_type}
#   coefficient_needs_attention{hour,day_type}
#   coefficient_sample_count{hour,day_type}
```

**Optie B: Prometheus JSON Exporter**
```yaml
# Use existing json_exporter
# Config to parse /calibration/status
```

### Stap 3: Alert Rules (20 min)

**Voeg toe aan:** `/opt/monitoring/prometheus/alerts.yml`

```yaml
  # Coefficient Server Alerts
  - name: coefficient_alerts
    interval: 60s
    rules:
      # Coefficient server down
      - alert: CoefficientServerDown
        expr: up{job="node-exporter-coefficient"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Coefficient server unreachable"
          description: "Coefficient server (91.99.150.36) has been down for 2+ minutes"

      # Coefficient API down
      - alert: CoefficientAPIDown
        expr: probe_success{job="blackbox-coefficient-http"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Coefficient API health check failing"
          description: "Coefficient API at http://91.99.150.36:8080/health is not responding"

      # High calibration drift (requires custom exporter)
      - alert: CoefficientDriftHigh
        expr: coefficient_drift_pct > 10
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "High calibration drift for hour {{ $labels.hour }}"
          description: "Coefficient drift is {{ $value }}% for hour {{ $labels.hour }} {{ $labels.day_type }}"

      # Calibration not running
      - alert: CalibrationTimerStale
        expr: (time() - node_systemd_timer_last_trigger_seconds{name="coefficient-calibration.timer"}) > 14400
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Calibration timer not triggering"
          description: "Coefficient calibration has not run in 4+ hours (expected 2x daily)"

      # Memory pressure on coefficient server
      - alert: CoefficientMemoryPressure
        expr: (node_memory_MemAvailable_bytes{instance="coefficient"} / node_memory_MemTotal_bytes{instance="coefficient"}) * 100 < 20
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Memory pressure on coefficient server"
          description: "Available memory < 20% on coefficient server"
```

### Stap 4: Grafana Dashboard (60 min)

**Dashboard: "Coefficient Engine Health"**

Panels:
1. **System Resources**
   - CPU usage (gauge)
   - Memory available (gauge)
   - Disk space (gauge)

2. **API Health**
   - Health check status (stat)
   - Response times (graph)
   - Request rate (graph)

3. **Calibration Drift** ⭐
   - Heatmap: drift per hour/day_type
   - List: hours needing attention
   - Graph: drift over time

4. **Data Pipeline**
   - Timer last trigger times
   - Database table sizes
   - Data freshness per table

5. **VPN Status**
   - Tunnel state
   - Handshake age
   - Transfer volume

---

## CUSTOM METRICS EXPORTERS

### Coefficient Calibration Exporter

**Locatie:** `/opt/monitoring/exporters/coefficient_calibration_exporter.py`

```python
#!/usr/bin/env python3
"""
Coefficient Calibration Prometheus Exporter
Scrapes /calibration/status and exports as metrics
"""
import requests
from prometheus_client import start_http_server, Gauge
import time

# Metrics
drift_gauge = Gauge('coefficient_drift_pct',
                   'Coefficient drift percentage',
                   ['hour', 'day_type'])
attention_gauge = Gauge('coefficient_needs_attention',
                       'Whether hour needs calibration attention',
                       ['hour', 'day_type'])
sample_gauge = Gauge('coefficient_sample_count',
                    'Sample count for calibration',
                    ['hour', 'day_type'])

def scrape_calibration_status():
    """Fetch and export calibration status"""
    try:
        resp = requests.get('http://91.99.150.36:8080/calibration/status', timeout=5)
        data = resp.json()

        for item in data.get('status', []):
            hour = str(item['hour'])
            day_type = item['day_type']

            drift_gauge.labels(hour=hour, day_type=day_type).set(item['avg_drift_pct'])
            attention_gauge.labels(hour=hour, day_type=day_type).set(
                1 if item['needs_attention'] else 0
            )
            sample_gauge.labels(hour=hour, day_type=day_type).set(item['sample_count'])

    except Exception as e:
        print(f"Scrape failed: {e}")

if __name__ == '__main__':
    start_http_server(9101)  # Exporter port
    while True:
        scrape_calibration_status()
        time.sleep(300)  # Scrape every 5 minutes
```

**Systemd service:**
```ini
[Unit]
Description=Coefficient Calibration Prometheus Exporter
After=network.target

[Service]
Type=simple
User=monitoring
ExecStart=/usr/bin/python3 /opt/monitoring/exporters/coefficient_calibration_exporter.py
Restart=always

[Install]
WantedBy=multi-user.target
```

**Prometheus scrape:**
```yaml
- job_name: "coefficient-calibration-exporter"
  static_configs:
    - targets: ["localhost:9101"]
```

---

## TESTING CHECKLIST

### Pre-Deployment
```bash
# 1. Backup current config
sudo cp /opt/monitoring/prometheus/prometheus.yml \
       /opt/monitoring/prometheus/prometheus.yml.backup-$(date +%Y%m%d-%H%M%S)

# 2. Validate new config
sudo docker exec prometheus promtool check config /etc/prometheus/prometheus.yml

# 3. Validate alert rules
sudo docker exec prometheus promtool check rules /etc/prometheus/alerts.yml
```

### Post-Deployment
```bash
# 4. Reload Prometheus
sudo docker exec prometheus kill -HUP 1

# 5. Check targets in Prometheus UI
# http://77.42.41.135:9090/targets
# Verify all targets are UP

# 6. Test queries
# coefficient_drift_pct
# up{instance="coefficient"}
# node_memory_MemAvailable_bytes{instance="coefficient"}

# 7. Check Grafana datasource
# https://monitor.synctacles.com/
# Verify Prometheus datasource works

# 8. Trigger test alert
# Manually stop coefficient-api and verify alert fires
```

---

## MONITORING GAPS & RISICO'S

### Huidige Gaps
1. ❌ Coefficient server NIET gemonitord
2. ❌ Calibration drift NIET zichtbaar
3. ❌ VPN tunnel status NIET gemonitord
4. ❌ Consumer proxy health NIET gemonitord
5. ❌ Database growth trends NIET gemonitord

### Risico's Zonder Monitoring
- Calibration drift >10% blijft onopgemerkt → slechte predictions
- VPN tunnel down → tier 1-2 fallback faalt stilletjes
- Coefficient server OOM → geen consumer prices
- Database full → collectors falen
- Timer failures → data wordt stale

---

## RESOURCE REQUIREMENTS

### Monitoring Server
```
Current: 77.42.41.135
Additional load:
  - +1 node-exporter scrape (91.99.150.36:9100)
  - +1 API health check
  - +1 custom exporter (optional)

Estimated impact:
  - CPU: +2-5%
  - Memory: +50-100 MB
  - Storage: +10 MB/day (metrics retention)
```

### Coefficient Server
```
Current: 91.99.150.36
No changes needed:
  - node-exporter al actief (port 9100)
  - API endpoints beschikbaar

Optional additions:
  - postgres_exporter: +50 MB RAM
  - Custom exporters: +20 MB RAM each
```

---

## VOLGENDE STAPPEN VOOR CAI

### Beslissingen Nodig
1. **Welke prioriteit niveau implementeren?**
   - [ ] Priority 1 alleen (basis monitoring)
   - [ ] Priority 1 + 2 (standaard + calibration)
   - [ ] Alle priorities (volledig)

2. **Custom exporters maken?**
   - [ ] Ja - calibration drift exporter
   - [ ] Ja - VPN status exporter
   - [ ] Nee - alleen basis monitoring

3. **Grafana dashboard gewenst?**
   - [ ] Ja - maak coefficient dashboard
   - [ ] Nee - gebruik bestaande system dashboard

4. **Alert routing configureren?**
   - [ ] Email notificaties
   - [ ] Slack webhook
   - [ ] PagerDuty integration

### Informatie Voor Implementatie

**Toegang tot monitoring server:**
```bash
ssh -i /root/.ssh/id_cx23 monitoring@77.42.41.135
# Of via cx23 alias (root)
```

**Config locaties:**
```
Prometheus: /opt/monitoring/prometheus/prometheus.yml
Alerts:     /opt/monitoring/prometheus/alerts.yml
Grafana:    /opt/monitoring/grafana/
```

**Docker commands:**
```bash
# Config validatie
sudo docker exec prometheus promtool check config /etc/prometheus/prometheus.yml

# Reload (graceful)
sudo docker exec prometheus kill -HUP 1

# Restart (hard)
sudo docker restart prometheus

# Logs
sudo docker logs prometheus
sudo docker logs grafana
```

---

## REFERENTIE: HUIDIGE PROMETHEUS CONFIG

**Bestand:** `/opt/monitoring/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

rule_files:
  - /etc/prometheus/alerts.yml

scrape_configs:
  - job_name: "prometheus"
    static_configs:
      - targets: ["localhost:9090"]

  - job_name: "node-exporter-main"
    static_configs:
      - targets: ["135.181.255.83:9100"]
        labels:
          instance: "enin-main"
          environment: "production"

  - job_name: "blackbox-http"
    metrics_path: /probe
    params:
      module: [http_2xx]
    static_configs:
      - targets:
          - https://enin.xteleo.nl/health
        labels:
          environment: "production"
          service: "api"
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox:9115

  - job_name: "blackbox-ssl"
    metrics_path: /probe
    params:
      module: [tls_connect]
    static_configs:
      - targets:
          - enin.xteleo.nl:443
        labels:
          environment: "production"
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox:9115

  - job_name: "pipeline-health"
    scheme: https
    tls_config:
      server_name: enin.xteleo.nl
    static_configs:
      - targets: ["135.181.255.83:443"]
        labels:
          environment: "production"
          service: "pipeline"
          instance: "enin-main"
    metrics_path: /v1/pipeline/metrics
    scrape_interval: 30s
```

---

## CONTACTPERSONEN

**Voor vragen over:**
- Coefficient server: Leo (eigenaar coefficient-engine repo)
- Monitoring server: Leo (beheert cx23)
- Prometheus/Grafana: CAI (implementatie)

---

## HANDOFF COMPLEET

**Status:** 🟢 Klaar voor CAI implementatie

**Geleverd:**
- ✅ Volledige inventarisatie beide servers
- ✅ Alle monitoring punten gedocumenteerd
- ✅ Implementatie plan met priorities
- ✅ Custom exporter voorbeelden
- ✅ Alert rule voorstellen
- ✅ Testing checklist

**Wachtend op:**
- CAI beslissing over prioriteit niveau
- CAI implementatie Prometheus config wijzigingen
- CAI creatie Grafana dashboards (optioneel)

---

**Laatste update:** 2026-01-13 00:15 UTC
**Document versie:** 1.0
