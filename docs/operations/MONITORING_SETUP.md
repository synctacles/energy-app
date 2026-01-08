# Monitoring Infrastructure - Complete Setup Guide

**Status:** ✅ Operational
**Last Updated:** 2026-01-07
**Security Score:** 9/10

---

## Overview

| Component | Server | URL/Port |
|-----------|--------|----------|
| **Grafana** | CX23 (77.42.41.135) | https://monitor.synctacles.com |
| **Prometheus** | CX23 (Docker) | localhost:9090 (internal) |
| **AlertManager** | CX23 (Docker) | localhost:9093 (internal) |
| **Blackbox Exporter** | CX23 (Docker) | localhost:9115 (internal) |
| **node_exporter** | API server (135.181.255.83) | :9100 |
| **Caddy** | CX23 | :80, :443 (reverse proxy) |

---

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │         CX23 Monitoring Server          │
                    │            77.42.41.135                  │
                    │                                         │
  Internet ──────►  │  ┌─────────┐    ┌──────────────────┐   │
  (HTTPS:443)       │  │  Caddy  │───►│     Grafana      │   │
                    │  │         │    │   (localhost:3000)│   │
                    │  └─────────┘    └──────────────────┘   │
                    │                          │              │
                    │                          ▼              │
                    │                 ┌──────────────────┐   │
                    │                 │   Prometheus     │   │
                    │                 │  (localhost:9090)│   │
                    │                 └────────┬─────────┘   │
                    │                          │              │
                    │         ┌────────────────┼────────┐    │
                    │         ▼                ▼        ▼    │
                    │  ┌───────────┐  ┌───────────┐ ┌──────┐│
                    │  │AlertManager│  │ Blackbox  │ │ ...  ││
                    │  │(:9093)     │  │ (:9115)   │ │      ││
                    │  └─────┬─────┘  └─────┬─────┘ └──────┘│
                    └────────┼──────────────┼───────────────┘
                             │              │
                             ▼              ▼
                    ┌────────────┐   ┌─────────────────┐
                    │   Slack    │   │   API Server    │
                    │ Webhooks   │   │ 135.181.255.83  │
                    │            │   │  node_exporter  │
                    │ #critical  │   │     :9100       │
                    │ #warnings  │   └─────────────────┘
                    │ #info      │
                    └────────────┘
```

---

## Access

### Grafana Dashboard

**URL:** https://monitor.synctacles.com
**Authentication:** Username/password (configured)

**Available Dashboards:**
- System Overview - Server metrics, API health
- Services Status - systemd service monitoring
- API Health - Endpoint monitoring, SSL status

### Slack Alerts

| Channel | Alerts |
|---------|--------|
| `#enin-alerts-critical` | Server down, disk full >90%, service failures |
| `#enin-alerts-warnings` | High CPU/memory, SSL expiry <14 days |
| `#enin-alerts-info` | Informational notifications |

---

## What is Monitored

### API Server (135.181.255.83)

| Metric | Alert Threshold |
|--------|-----------------|
| CPU usage | >80% for 5 min |
| Memory usage | >85% for 5 min |
| Disk usage | >80% warning, >90% critical |
| Load average | >2.0 for 5 min |
| systemd services | energy-insights-nl-*, synctacles-* |

### API Endpoint

| Check | Target | Alert |
|-------|--------|-------|
| HTTP health | https://enin.xteleo.nl/health | Down >1 min |
| SSL certificate | enin.xteleo.nl:443 | Expiry <14 days |

### Pipeline Health (Prometheus Metrics)

**Endpoint:** `https://enin.xteleo.nl/v1/pipeline/metrics`

**Metrics Exposed:**

| Metric | Description | Labels |
|--------|-------------|--------|
| `pipeline_timer_status` | Timer status (1=active, 0=stopped) | timer={collector\|importer\|normalizer\|health} |
| `pipeline_timer_last_trigger_minutes` | Minutes since timer last triggered | timer=... |
| `pipeline_data_status` | Data status (0=FRESH, 1=STALE, 2=UNAVAILABLE, 3=NO_DATA) | source={a44\|a65\|a75} |
| `pipeline_data_freshness_minutes` | Data age in minutes (normalized table) | source=... |

**Data Status Values:**
- **0 (FRESH):** Data <90 min old (A44/A65) or <180 min (A75 normal delay)
- **1 (STALE):** Data 90-180 min old
- **2 (UNAVAILABLE):** Data >180 min old
- **3 (NO_DATA):** No data in database

**Alert Rules:**
```yaml
# Critical: A44 data unavailable (prices are critical for users)
- alert: PipelineDataUnavailableA44
  expr: pipeline_data_status{source="a44"} >= 2
  for: 15m
  labels: {severity: critical}

# Warning: A65/A75 data unavailable (tolerate longer for A75)
- alert: PipelineDataUnavailableA65
  expr: pipeline_data_status{source="a65"} >= 2
  for: 15m
  labels: {severity: warning}

# Critical: Any timer stopped
- alert: PipelineTimerStopped
  expr: pipeline_timer_status == 0
  for: 5m
  labels: {severity: critical}
```

### Services Monitored

```
energy-insights-nl-api.service
energy-insights-nl-collector.service
energy-insights-nl-health.service
energy-insights-nl-importer.service
energy-insights-nl-normalizer.service
synctacles-collector.service (deprecated)
synctacles-health.service (deprecated)
synctacles-importer.service (deprecated)
synctacles-normalizer.service (deprecated)
```

---

## Alert Rules

### Critical (immediate action)

| Alert | Condition | Duration |
|-------|-----------|----------|
| InstanceDown | Instance unreachable | 1 min |
| DiskSpaceCritical | >90% full | immediate |
| ServiceFailed | systemd service failed | immediate |
| APIEndpointDown | HTTP check fails | 1 min |

### Warning (investigate soon)

| Alert | Condition | Duration |
|-------|-----------|----------|
| HighMemoryUsage | >85% | 5 min |
| HighCPUUsage | >80% | 5 min |
| DiskSpaceWarning | >80% | immediate |
| HighLoadAverage | >2.0 | 5 min |
| SSLCertExpiringSoon | <14 days | immediate |

---

## Server Configuration

### CX23 Monitoring Server (77.42.41.135)

**OS:** Ubuntu 24.04
**Resources:** 2 vCPU, 4GB RAM, 40GB disk

**Services:**
```bash
# Docker containers
docker ps
# prometheus, grafana, alertmanager, blackbox

# Reverse proxy
systemctl status caddy

# Security
systemctl status fail2ban
```

**Firewall (Hetzner):**
| Port | Service | Access |
|------|---------|--------|
| 22 | SSH | Open |
| 80 | HTTP (redirect) | Open |
| 443 | HTTPS (Grafana) | Open |
| 3000 | Grafana direct | Closed |
| 9090 | Prometheus | Closed |
| 9093 | AlertManager | Closed |

**⚠️ Known Limitations:**

**DNS Resolution Broken:**
- monitor.synctacles.com cannot resolve external domains (e.g., api.synctacles.com)
- **Impact:** Grafana Infinity plugin and JSON-based datasources will NOT work
- **Solution:** Use Prometheus datasource with IP address + SNI header configuration
- **Do NOT install Grafana Infinity plugin** - it will show "No data" on all panels

**Working Configuration Example:**
```yaml
# Prometheus scrape config that works despite DNS issues
- job_name: "pipeline-health"
  scheme: https
  tls_config:
    server_name: enin.xteleo.nl  # SNI for SSL cert
  static_configs:
    - targets: ["135.181.255.83:443"]  # Direct IP, no DNS lookup
  metrics_path: /v1/pipeline/metrics
```

**Key Files:**
```
/opt/monitoring/
├── docker-compose.yml      # Container definitions
├── prometheus/
│   ├── prometheus.yml      # Scrape config
│   └── alerts.yml          # Alert rules
├── alertmanager/
│   └── alertmanager.yml    # Slack webhooks
├── blackbox/
│   └── blackbox.yml        # HTTP/SSL checks
└── grafana/
    ├── datasources/
    │   └── prometheus.yml  # Datasource config
    └── dashboards/
        ├── system-overview.json
        ├── services-status.json
        └── api-health.json

/etc/caddy/Caddyfile         # Reverse proxy config
```

### API Server (135.181.255.83)

**node_exporter config:**
```
/etc/systemd/system/node_exporter.service
```

**Key settings:**
```ini
ExecStart=/usr/local/bin/node_exporter \
    --collector.systemd \
    --collector.systemd.unit-include="energy-insights-nl-.+|synctacles-.+"
```

---

## Security Configuration

### SSH Access

| User | Access | Notes |
|------|--------|-------|
| monitoring | SSH key | Primary access |
| root | SSH key (prohibit-password) | Emergency only |

**SSH keys authorized for monitoring user:**
- Windows workstation (ftso@coston2)
- API server (root@ENIN-NL)

### Security Measures

| Measure | Status |
|---------|--------|
| HTTPS (Let's Encrypt) | ✅ Auto-renewal |
| SSH key-only | ✅ Password disabled |
| fail2ban | ✅ 24h ban after 3 failures |
| Hetzner firewall | ✅ Only 22, 80, 443 open |
| Grafana on localhost | ✅ Via Caddy only |
| Automatic security updates | ✅ unattended-upgrades |

---

## Operations

### Daily Checks

1. Open https://monitor.synctacles.com
2. Review System Overview dashboard
3. Check for any fired alerts

### Restart Services

```bash
# SSH to CX23
ssh monitoring@77.42.41.135

# Restart all containers
cd /opt/monitoring
sudo docker compose restart

# Restart Caddy
sudo systemctl restart caddy
```

### View Logs

```bash
# Docker container logs
sudo docker logs prometheus
sudo docker logs grafana
sudo docker logs alertmanager

# Caddy logs
sudo journalctl -u caddy -f

# fail2ban
sudo fail2ban-client status sshd
```

### Check Certificate

```bash
# SSL expiry date
echo | openssl s_client -connect monitor.synctacles.com:443 2>/dev/null | openssl x509 -noout -dates
```

### Add New Alert Rule

1. Edit `/opt/monitoring/prometheus/alerts.yml`
2. Restart Prometheus: `sudo docker compose restart prometheus`
3. Verify in Grafana → Alerting → Alert rules

---

## Troubleshooting

### Grafana Not Accessible

```bash
# Check Caddy
sudo systemctl status caddy
sudo journalctl -u caddy --since "5 minutes ago"

# Check Grafana container
sudo docker ps | grep grafana
sudo docker logs grafana --tail 50
```

### No Metrics from API Server

```bash
# On API server
systemctl status node_exporter
curl localhost:9100/metrics | head

# From CX23
curl http://135.181.255.83:9100/metrics | head
```

### Alerts Not Sending to Slack

```bash
# Check AlertManager
sudo docker logs alertmanager --tail 50

# Test webhook manually
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Test alert"}' \
  <SLACK_WEBHOOK_URL>
```

### High Memory on CX23

```bash
# Check container memory
sudo docker stats --no-stream

# If >3.5GB total, consider upgrading to CX33
```

---

## Backup & Recovery

### What to Backup

| Item | Location | Method |
|------|----------|--------|
| Prometheus data | Docker volume | Optional (15 day retention) |
| Grafana dashboards | Docker volume | Export JSON from UI |
| Config files | /opt/monitoring/ | Git repo |
| Caddyfile | /etc/caddy/ | Manual backup |

### Recovery Steps

1. Provision new CX23
2. Copy /opt/monitoring from backup/repo
3. Install Docker, Caddy
4. `docker compose up -d`
5. Configure DNS → new IP
6. Caddy will auto-obtain new SSL cert

---

## Scaling

### Current Capacity

- 1 API server monitored
- 22 alert rules
- 15 day data retention
- ~2GB memory used

### To Add More Servers

1. Install node_exporter on new server
2. Add target to `/opt/monitoring/prometheus/prometheus.yml`
3. Restart Prometheus

### Upgrade to CX33

If memory >85% sustained:
1. Stop CX23 in Hetzner Console
2. Resize to CX33 (8GB RAM)
3. Start server
4. Verify: `docker ps`

---

## Related Documentation

- [SYSTEMD_SERVICES_ANALYSIS.md](../SYSTEMD_SERVICES_ANALYSIS.md) - Service status
- [SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md](../skills/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md) - Logging standards

---

## Dashboards

### Pipeline Health Dashboard

**URL:** [https://monitor.synctacles.com/d/5fd1f7f9-e2bb-4a81-a04e-50f9fbbf0ec0/pipeline-health](https://monitor.synctacles.com/d/5fd1f7f9-e2bb-4a81-a04e-50f9fbbf0ec0/pipeline-health)

**UID:** `5fd1f7f9-e2bb-4a81-a04e-50f9fbbf0ec0`

**Folder:** Energy Insights NL

**Panels:**

**Row 1 - Pipeline Components:**
1. **API Status** - `count(up{job="pipeline-health"})` → Shows "1" (green) if scraping works
2. **Collector Timer** - `pipeline_timer_status{timer="collector"}` → ACTIVE/STOPPED
3. **Importer Timer** - `pipeline_timer_status{timer="importer"}` → ACTIVE/STOPPED
4. **Normalizer Timer** - `pipeline_timer_status{timer="normalizer"}` → ACTIVE/STOPPED
5. **Health Timer** - `pipeline_timer_status{timer="health"}` → ACTIVE/STOPPED

**Row 2 - Data Status:**
6. **A44 Day-Ahead Prices** - `pipeline_data_status{source="a44"}` → FRESH/STALE/UNAVAILABLE
7. **A65 System Load** - `pipeline_data_status{source="a65"}` → FRESH/STALE/UNAVAILABLE
8. **A75 Generation by Source** - `pipeline_data_status{source="a75"}` → FRESH/STALE/UNAVAILABLE

**Row 3 - Trends:**
9. **Data Freshness Over Time** - `pipeline_data_freshness_minutes` (timeseries)

**Color Coding:**
- **Green:** FRESH (<90 min) / ACTIVE
- **Yellow:** STALE (90-180 min)
- **Red:** UNAVAILABLE (>180 min) / STOPPED
- **Gray:** NO_DATA

**Refresh Rate:** 30 seconds (auto)

---

**Document Owner:** Leo
**Last Verified:** 2026-01-08
