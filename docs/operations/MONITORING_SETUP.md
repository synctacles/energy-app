# SYNCTACLES Monitoring Setup Guide

**For:** Energy Insights NL production monitoring
**Architecture:** Hybrid (Docker on monitoring server + native node_exporter on app servers)
**Status:** Ready for deployment

---

## Quick Start (5 minutes)

On your **monitoring server (CX23)**:

```bash
# 1. Ensure /opt/monitoring directory with docker-compose.yml
# (should be in PHASE_1_HANDOFF.md)

# 2. Go to directory
cd /opt/monitoring

# 3. Start monitoring
docker-compose up -d

# 4. Verify
docker ps
curl http://localhost:9090
curl http://localhost:3000
```

**Then add app servers:**

```bash
# On MONITORING server
./scripts/monitoring/add_target.sh 10.0.1.5 production-api
./scripts/monitoring/add_target.sh 10.0.1.6 collector-01
```

---

## Detailed Setup

### Phase 1: Monitoring Server (CX23)

**Prerequisites:**
- Ubuntu 24.04 clean installation
- 4GB RAM (CX23 recommended)
- 40GB disk
- SSH access as root
- Internet connectivity

**Steps:**

#### 1.1 Prepare Directory Structure

```bash
mkdir -p /opt/monitoring
cd /opt/monitoring

# Create docker-compose.yml
# (From PHASE_1_HANDOFF.md or copy from repo)
# Should include: prometheus, grafana, alertmanager services
```

#### 1.2 Create Prometheus Configuration

```bash
# Create prometheus.yml
cat > prometheus.yml <<'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'synctacles'
    environment: 'production'

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - localhost:9093

rule_files:
  - /etc/prometheus/alerts.yml

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']

  - job_name: 'synctacles-servers'
    static_configs:
      - targets: []
        # AUTO-GENERATED - DO NOT EDIT MANUALLY
        # Use: ./scripts/monitoring/add_target.sh <ip> <name>
EOF
```

#### 1.3 Create Alert Rules

```bash
# Copy from repository
cp /opt/github/synctacles-api/monitoring/prometheus/alerts.yml ./

# Or create manually from SKILL_14
```

#### 1.4 Start Containers

```bash
# Make sure docker-compose.yml exists
docker-compose up -d

# Wait 10 seconds for startup
sleep 10

# Verify all running
docker ps
# Should show: prometheus, grafana, alertmanager (all healthy)
```

#### 1.5 Verify Access

```bash
# Prometheus
curl http://localhost:9090/-/ready
# Should return: "OK"

# Grafana
curl http://localhost:3000
# Should return HTML

# Node exporter (on monitoring server)
curl http://localhost:9100/metrics | head
# Should show: node_cpu_seconds_total, node_memory_MemTotal_bytes, etc.
```

#### 1.6 Change Grafana Password

```bash
# Access: http://<monitoring_server_ip>:3000
# Username: admin
# Password: admin (default)

# Click: Admin → Account → Change Password
# Set strong password
```

**✅ Phase 1 Complete:** Monitoring server operational

---

### Phase 2: Application Servers (node_exporter)

**For each app server:**

#### 2.1 Copy Setup Script

```bash
# On your LOCAL MACHINE or monitoring server
scp /opt/github/synctacles-api/scripts/monitoring/setup_application_server.sh \
    root@<app_server_ip>:/tmp/
```

#### 2.2 Run Installation

```bash
# SSH to app server
ssh root@<app_server_ip>

# Run script (provide monitoring server IP)
sudo bash /tmp/setup_application_server.sh <monitoring_server_ip>

# Example:
# sudo bash /tmp/setup_application_server.sh 10.0.1.100
```

**Script will:**
- Install node_exporter
- Start systemd service
- Configure firewall (allow monitoring server only)
- Verify metrics available at :9100

#### 2.3 Add to Prometheus (On Monitoring Server)

```bash
# Back on monitoring server
cd /opt/monitoring

./scripts/monitoring/add_target.sh <app_server_ip> <server_name>

# Example:
# ./scripts/monitoring/add_target.sh 10.0.1.5 production-api
```

#### 2.4 Verify Metrics (Wait 30 seconds)

```bash
# In Prometheus UI:
# http://<monitoring_server_ip>:9090/targets

# Should show green "UP" for each server

# OR in Grafana:
# http://<monitoring_server_ip>:3000
# Wait 30 seconds, metrics should appear in System Health dashboard
```

**Repeat for each app server**

**✅ Phase 2 Complete:** App servers sending metrics

---

## Testing & Validation

### Comprehensive Test

```bash
# On monitoring server
cd /opt/github/synctacles-api

./scripts/monitoring/test_monitoring.sh
```

**Should show:**
- ✅ All containers running
- ✅ Prometheus API responsive
- ✅ Grafana API responsive
- ✅ node_exporter metrics available
- ✅ Alert rules loaded
- ✅ Firewall configured
- ✅ Disk space OK

### Quick Checks

**Check Prometheus targets:**
```bash
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | .labels.instance'
```

**Check metrics flowing:**
```bash
curl http://localhost:9090/api/v1/query?query=up | jq '.data.result[].value'
```

**Check alerts:**
```bash
curl http://localhost:9090/api/v1/rules | jq '.data.groups[0].rules | length'
```

---

## Accessing Monitoring

### Prometheus

**URL:** http://<monitoring_server_ip>:9090

**Pages:**
- `/targets` - Server status and health
- `/alerts` - Active/fired alerts
- `/graph` - Query metrics
- `/config` - Current configuration

### Grafana

**URL:** http://<monitoring_server_ip>:3000

**Dashboards:**
- System Health - Overview of all servers
- Memory Analysis - OOM prevention focus (when available)
- API Performance - Request metrics (when available)

### AlertManager

**URL:** http://<monitoring_server_ip>:9093

**Shows:**
- Active alerts
- Alert groups
- Notification history

---

## Troubleshooting

### Containers Won't Start

```bash
# Check logs
docker-compose logs prometheus
docker-compose logs grafana
docker-compose logs alertmanager

# Restart all
docker-compose restart

# Rebuild if needed
docker-compose down
docker-compose up -d
```

### Prometheus Not Scraping

```bash
# Check targets
curl http://localhost:9090/api/v1/targets

# Should show "UP" for all targets
# If "DOWN", check:
# 1. Node exporter running: systemctl status prometheus-node-exporter (on app server)
# 2. Firewall allows: ufw status (on app server)
# 3. Network: ping from monitoring to app server
```

### Metrics Not Appearing in Grafana

```bash
# Wait 30 seconds after adding target

# Check datasource
curl http://localhost:3000/api/datasources

# Should show Prometheus datasource

# Manually query
curl http://localhost:9090/api/v1/query?query=up

# Should return data
```

### Memory Usage High

```bash
# Check container stats
docker stats

# If > 3.5GB total, consider upgrading CX23 → CX33
# Can do via Hetzner console (2-5 min downtime)
```

### No Swap Configured

If alert fires: "No swap configured"

```bash
# On app server that triggered alert
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Make persistent
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

---

## Operations

### Daily

- Check Grafana System Health dashboard for anomalies
- Review recent alerts
- Monitor memory usage trend

### Weekly

- Verify all servers reporting metrics
- Check Prometheus disk usage
- Review alert thresholds

### Monthly

- Test alert notifications (trigger manually)
- Review retention needs (default: 15 days)
- Update capacity baseline
- Security updates

### Scaling to CX33

If memory usage > 85% sustained:

1. **Via Hetzner Console:**
   - Stop CX23
   - Upgrade to CX33 (8GB RAM)
   - Start CX23 → becomes CX33

2. **On Server:**
   ```bash
   # Docker restarts automatically
   docker ps

   # Verify
   ./scripts/monitoring/test_monitoring.sh
   ```

---

## Integration Points

### With Application Servers

- node_exporter exposes metrics at `http://<app_server>:9100/metrics`
- Prometheus scrapes every 15 seconds
- No configuration needed on app servers (except opening firewall)

### With Slack (Future)

- AlertManager can send to Slack webhook
- Configure in docker-compose.yml environment
- See ALERT_RUNBOOK.md for responses

### With Logging (SKILL_13)

- System metrics (Prometheus): time-series
- Application logs (journal): events
- Both visible in operational view

---

## Security Considerations

### Monitoring Server (CX23)

- UFW firewall: Only allow needed ports (22, 9090, 3000)
- fail2ban: SSH brute-force protection
- SSH: Key-based only (recommend disabling root later)
- Docker: Services isolated in containers
- Secrets: None stored in config (no API keys needed)

### Application Servers

- Firewall: Port 9100 allowed for monitoring server IP only
- No additional services running
- node_exporter: Read-only access to metrics
- No outbound connections needed (monitoring server pulls)

---

## Capacity Planning

### Current (CX23 + 3-5 app servers)

- **Monitoring memory:** ~2.5GB used, 1.5GB buffer
- **Data retention:** 15 days
- **Disk usage:** ~10GB per 15 days

### Upgrade to CX33 (10-15 app servers)

- **Monitoring memory:** ~3GB used, 5GB buffer
- **Data retention:** 30+ days possible
- **Disk usage:** ~30GB per 30 days

### Further scaling (30+ servers)

- Consider federation (multiple Prometheus servers)
- Or external metrics service (Datadog, New Relic, etc.)

---

## Next Steps

1. ✅ Phase 1: Monitoring server running
2. ✅ Phase 2: App servers configured
3. ⏳ Phase 3: [MONITORING_RUNBOOK.md](MONITORING_RUNBOOK.md) - Operations manual
4. ⏳ Phase 4: Grafana dashboards (create via UI or from JSON)
5. ⏳ Phase 5: Load testing (verify system behavior)

---

## Support

**For technical details:**
- [SKILL_14_MONITORING_INFRASTRUCTURE.md](../../SKILL_14_MONITORING_INFRASTRUCTURE.md) - Architecture reference
- [MONITORING_RUNBOOK.md](MONITORING_RUNBOOK.md) - Operations guide
- [PHASE_1_HANDOFF.md](../../PHASE_1_HANDOFF.md) - Docker setup details

**GitHub Issues:**
- #27 - AlertManager & Slack Integration
- #28 - Grafana Dashboards
- #29 - Load Testing

---

**Last Updated:** 2026-01-06
**Status:** Ready for deployment
