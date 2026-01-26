# SKILL INFRASTRUCTURE 05 — MONITORING INFRASTRUCTURE

**Version:** 1.0
**Date:** 2026-01-06
**Status:** Approved for Implementation
**Scope:** Production monitoring for SYNCTACLES distributed systems

---

## Purpose & Philosophy

**Why separate monitoring infrastructure?**

A monitoring system must remain operational even when the systems it monitors fail. A production server with OOM crashes, network issues, or service failures should not affect visibility into what happened.

**SYNCTACLES Monitoring Principles:**
1. **Isolation:** Monitoring server separate from application servers
2. **Efficiency:** Minimal resource overhead (native agents, not Docker on app servers)
3. **Visibility:** Real-time metrics with historical data retention
4. **Alerting:** Proactive notification of problems (not reactive discovery)
5. **Simplicity:** Easy to deploy, troubleshoot, and scale

---

## Architecture

### Overview

```
┌─────────────────────────────────┐
│  Monitoring Server (CX23/CX33)  │
│  - Prometheus (metrics store)   │
│  - Grafana (dashboards)         │
│  - AlertManager (notifications) │
│  - Docker containers            │
└──────────────────┬──────────────┘
                   │ scrape (15s)
        ┌──────────┴──────────┬──────────┐
        ▼                     ▼          ▼
┌──────────────────┐  ┌──────────────┐  ┌──────────────┐
│  App Server 1    │  │  App Server 2│  │  App Server N│
│  node_exporter   │  │ node_exporter│  │node_exporter │
│  (native systemd)│  │ (native)     │  │ (native)     │
└──────────────────┘  └──────────────┘  └──────────────┘
```

**Data Flow:**
1. node_exporter on each app server exposes metrics at `http://localhost:9100/metrics`
2. Prometheus scrapes every 15 seconds (configurable)
3. Prometheus evaluates alert rules every 15 seconds
4. Alert triggered → AlertManager → Slack webhook
5. Grafana reads from Prometheus for dashboards

---

## Hardware Requirements

### Monitoring Server (CX23 recommended, CX33 if needed)

| Size | CPU | RAM | Disk | App Servers | Notes |
|------|-----|-----|------|-------------|-------|
| **CX23** | 2 vCPU | 4GB | 40GB | 3-5 | **Recommended starting point** |
| **CX33** | 4 vCPU | 8GB | 80GB | 10-15 | Upgrade if CX23 RAM exceeds 85% |
| **CX43** | 8 vCPU | 16GB | 160GB | 30+ | Enterprise scale |

**CX23 Resource Breakdown:**
- Ubuntu 24.04: ~600MB
- Prometheus container: ~200MB
- Grafana container: ~150MB
- AlertManager container: ~50MB
- node_exporter (host): ~20MB
- Buffer: ~1GB
- **Total: ~2.5GB / 4GB = 62.5% usage**

**Upgrade Trigger:** If memory > 85% for sustained period, upgrade to CX33.

### Application Servers (Existing)

**Per-server monitoring overhead:**
- node_exporter: ~20MB RAM, ~0.1% CPU
- Network: ~1-2 KB/s upstream (metrics to monitoring server)
- Disk: negligible

---

## Installation Architecture

### Monitoring Server Setup

**Approach:** Docker Compose (hybrid model)

```
Monitoring Server (CX23)
├── docker-compose.yml (3 services)
│   ├── prometheus (prom/prometheus)
│   ├── grafana (grafana/grafana)
│   └── alertmanager (prom/alertmanager)
├── UFW firewall (port 22, 9090, 3000)
├── fail2ban (SSH brute-force protection)
└── Monitoring user account
```

**Why Docker here?**
- Simple: `docker-compose up -d`
- Portable: Move to new server by copying volumes
- Updates: Pull latest image, restart
- Isolation: Services don't interfere with OS

### Application Servers (Node Exporter)

**Approach:** Native systemd services

```bash
# Simple installation
sudo apt install prometheus-node-exporter

# Starts automatically via systemd
# Firewall: allow monitoring server IP only
```

**Why native here?**
- Lightweight (~20MB, not 500MB Docker)
- Direct access to system metrics
- Follows SKILL_02_REPOS_ACCOUNTS pattern (service account)
- No Docker dependency

---

## Deployment

### Phase 1: Monitoring Server (CX23)

**Time: ~45 minutes**

1. Provision clean Ubuntu 24.04 on CX23
2. Clone repository
3. Run installation script:
   ```bash
   sudo ./scripts/monitoring/install_monitoring.sh
   ```
4. Answer configuration prompts
5. Verify: `sudo ./scripts/monitoring/test_monitoring.sh`

**Script handles:**
- System updates
- UFW firewall (SSH, Prometheus, Grafana)
- fail2ban for security
- Docker + Docker Compose
- Prometheus container with alert rules
- Grafana container with datasources
- AlertManager container (if SMTP configured)

### Phase 2: Application Servers (node_exporter)

**Time: ~15 minutes per server**

1. Copy setup script to app server:
   ```bash
   scp scripts/monitoring/setup_application_server.sh root@<app_server>:/tmp/
   ```

2. Run on app server:
   ```bash
   ssh root@<app_server> 'sudo bash /tmp/setup_application_server.sh <monitoring_server_ip>'
   ```

3. Add to Prometheus targets:
   ```bash
   ./scripts/monitoring/add_target.sh <app_server_ip> <server_name>
   ```

---

## Metrics & Alert Rules

### Collected Metrics

**System Level (node_exporter):**
- Memory: total, available, used, swap
- CPU: usage per core, context switches
- Disk: usage per mount, I/O operations
- Network: bytes in/out, errors, dropped packets
- Load average: 1min, 5min, 15min
- Processes: running, sleeping, zombie

**Process Level (future):**
- Gunicorn workers: memory per worker
- PostgreSQL: connections, query time
- API: request rate, latency, errors

### Alert Rules (OOM-Focused)

**File:** `monitoring/prometheus/alerts.yml`

#### Memory Alerts (Highest Priority)

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| **MemoryPressure Warning** | Available < 20% | warning | Investigate memory usage |
| **MemoryPressure Critical** | Available < 10% | critical | OOM imminent - reduce load immediately |
| **HighSwap Usage** | Swap > 50% | warning | Indicates memory pressure - check leaks |
| **No Swap** | Swap == 0 | warning | Vulnerable to OOM - add 4GB swap |

#### Service Level Alerts

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| **Service Down** | No metrics for 2 min | critical | Check systemctl, network, firewall |
| **Node Exporter Down** | Metrics missing | critical | Restart node_exporter service |

#### Disk Alerts

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| **Disk Low** | < 15% free | warning | Clean logs or expand disk |
| **Disk Critical** | < 5% free | critical | Services may fail - immediate action |

#### CPU Alerts

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| **High CPU** | > 80% for 10 min | warning | Investigate CPU processes |
| **Critical CPU** | > 95% for 5 min | critical | Server may become unresponsive |

---

## Dashboards

### Required Dashboards

#### 1. System Health Dashboard
**Purpose:** Overview of all monitored servers

**Panels:**
- Server status (up/down gauge)
- Memory utilization (%)
- CPU utilization (%)
- Disk usage (%)
- Available memory (MB)
- Swap usage (%)
- Active alerts count

#### 2. Memory Analysis Dashboard
**Purpose:** OOM prevention focus

**Panels:**
- Available memory over time (line graph)
- Memory utilization trend
- Swap usage over time
- Memory pressure gauge
- Alert timeline
- Top memory processes (future)

#### 3. API Performance Dashboard
**Purpose:** Request/response metrics (when API instrumentation added)

**Panels:**
- Request rate (req/s)
- Response time (P50, P95, P99)
- Error rate (%)
- Active connections
- Latency by endpoint

---

## Configuration & Customization

### Prometheus Configuration

**File:** `monitoring/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s           # Can be customized
  evaluation_interval: 15s
  external_labels:
    cluster: 'synctacles'
    environment: 'production'

scrape_configs:
  - job_name: 'prometheus'       # Self-monitoring
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node'             # App servers
    static_configs:
      - targets:
          - '10.0.1.5:9100'      # Added via add_target.sh
          - '10.0.1.6:9100'
```

### Alert Rules

**File:** `monitoring/prometheus/alerts.yml`

Alert rules are templates with PromQL expressions. See implementation for specific thresholds.

### Grafana Configuration

**Datasource:** Prometheus at `http://prometheus:9090`

**Dashboards:** Imported from JSON files in `monitoring/grafana/dashboards/`

**Authentication:**
- Default: admin/admin
- **CHANGE ON FIRST LOGIN**
- Disable Grafana signup in settings

---

## Security

### Monitoring Server (CX23)

**Firewall (UFW):**
```
SSH (22):         Allow all (for management)
Prometheus (9090): Allow monitoring clients only (optional)
Grafana (3000):   Allow monitoring clients only (optional)
```

**SSH Hardening:**
- Key-based authentication only
- Root login: enabled (per requirement) with warning
- fail2ban: brute-force protection
- Unattended security updates: enabled

**Service Accounts:**
- `monitoring` user: dedicated for Prometheus/Grafana/AlertManager
- Runs with minimal privileges
- No shell access

### Application Servers

**Firewall (UFW):**
```
node_exporter (9100): Allow monitoring server IP only
```

**node_exporter:**
- Runs as `prometheus` system user
- No shell access
- Read-only access to system metrics

### Data Isolation

- Monitoring server: **cannot SSH to app servers** (unidirectional)
- App servers: **pull metrics via HTTP only** (read-only)
- No sensitive data in metrics (IP addresses only)
- HTTPS optional (for future)

---

## Operations & Maintenance

### Daily Tasks
- Check Grafana dashboards for anomalies
- Review alert history
- Monitor disk usage trend

### Weekly Tasks
- Verify all servers reporting metrics
- Check Prometheus retention (default: 15 days)
- Review alert rule effectiveness

### Monthly Tasks
- Archive old metrics (optional)
- Update baseline thresholds
- Capacity planning review
- Security updates verification

### Upgrading CX23 → CX33

**If memory consistently > 85%:**

1. **On Hetzner console:**
   - Upgrade CX23 → CX33 (2-5 min downtime)
   - Monitoring server reboots

2. **After reboot:**
   - Docker containers restart automatically
   - Check: `docker ps`
   - Verify: `./scripts/monitoring/test_monitoring.sh`

3. **No data loss:**
   - Prometheus volumes preserved
   - Grafana dashboards preserved
   - Alert history preserved

---

## Scaling

### Adding New Application Servers

```bash
# 1. Install node_exporter on new server
ssh root@<new_server> 'sudo bash setup_application_server.sh'

# 2. Add to Prometheus targets
./scripts/monitoring/add_target.sh <new_server_ip> <server_name>

# 3. Verify metrics appear
# Dashboard updates within 15 seconds
```

### Adding More Monitoring Servers (Future)

For federation/multi-region:
- Secondary monitoring server runs independently
- Each scrapes its regional app servers
- Global Grafana for cross-region dashboards

---

## Integration with Other SKILLs

### SKILL_02_REPOS_ACCOUNTS: Service Accounts
- Monitoring user: `monitoring` (dedicated)
- Pattern: Similar to `energy-insights-nl` on app servers
- Principle: Each service has minimal privileges

### SKILL_00_HARDWARE: Hardware Profiles
- CX23: 3-5 servers (our current scale)
- CX33: 10-15 servers (upgrade path)
- Guidelines: Monitor RAM, upgrade at 85%

### SKILL_04_LOGGING: Logging
- Prometheus data: time-series (metrics)
- Journal logs: application events
- Integration: Both visible in operational dashboards

---

## Troubleshooting

### Monitoring Server Issues

**Docker containers not starting:**
```bash
docker-compose logs prometheus
docker-compose logs grafana
systemctl restart docker
```

**Prometheus not scraping:**
- Check: `http://localhost:9090/targets`
- Verify node_exporter running on app server
- Check firewall: `sudo ufw status`

**Grafana can't connect to Prometheus:**
- Check datasource: Grafana UI → Admin → Data Sources
- Verify Prometheus healthy: `curl http://localhost:9090/-/ready`

**High memory usage:**
- Check: `docker stats`
- Prometheus retention: reduce or upgrade to CX33
- Grafana dashboards: disable auto-refresh on unused

### Application Server Issues

**node_exporter not running:**
```bash
sudo systemctl status prometheus-node-exporter
sudo journalctl -u prometheus-node-exporter -n 50
```

**Metrics not appearing in Prometheus:**
- Check firewall: `sudo ufw status | grep 9100`
- Verify listening: `sudo netstat -tulpn | grep 9100`
- Test manually: `curl http://localhost:9100/metrics | head`

**Firewall blocking metrics:**
```bash
# From monitoring server
curl http://<app_server_ip>:9100/metrics

# If fails, check app server firewall
sudo ufw allow from <monitoring_ip> to any port 9100
```

---

## Future Enhancements

### Phase 2 (Future)
- Application metrics (API request rate, latency, errors)
- Database metrics (PostgreSQL connections, query time)
- Custom alerts for business logic
- Email alerting via Alertmanager

### Phase 3 (Future)
- Prometheus federation (multi-region)
- High availability (2x monitoring servers)
- Backup automation (Prometheus snapshots)
- Audit logging (who changed what)

### Phase 4 (Future)
- Machine learning for anomaly detection
- Predictive scaling recommendations
- Cost optimization insights

---

## References

**Related SKILLs:**
- SKILL_02_REPOS_ACCOUNTS: Service accounts and permissions
- SKILL_00_HARDWARE: Hardware profiles and tuning
- SKILL_04_LOGGING: Logging standards

**External Resources:**
- Prometheus: https://prometheus.io/docs/
- Grafana: https://grafana.com/docs/
- node_exporter: https://github.com/prometheus/node_exporter

**SYNCTACLES Documents:**
- [PRODUCTION_BLOCKERS.md](PRODUCTION_BLOCKERS.md) - Context
- [CODE_QUALITY_AUDIT_REPORT.md](CODE_QUALITY_AUDIT_REPORT.md) - Code verification
- [MONITORING_SETUP.md](docs/operations/MONITORING_SETUP.md) - Step-by-step guide
- [MONITORING_RUNBOOK.md](docs/operations/MONITORING_RUNBOOK.md) - Operations manual

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-06 | Initial SKILL document - hybrid architecture |

---

**Status:** Approved for Implementation
**Last Review:** 2026-01-06
**Next Review:** 2026-02-06
