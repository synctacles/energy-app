# Phase 1: CX23 Server Setup - Handoff Document

**GitHub Issue:** #25
**Status:** 🟡 READY TO START
**Estimated Duration:** 1-2 hours
**Target Completion:** Day 1 (Jan 6-7, 2026)

---

## 📋 What This Phase Does

Setup a new CX23 Hetzner server with Docker and deploy the monitoring stack (Prometheus, Grafana, AlertManager).

**After this phase completes:**
- ✅ CX23 fully upgraded and hardened
- ✅ Docker + Docker Compose installed
- ✅ All containers running (3: Prometheus, Grafana, AlertManager)
- ✅ Monitoring server ready to receive metrics from main server

---

## 🎯 Acceptance Criteria

Before moving to Phase 2, verify:
- [ ] CX23 SSH access working
- [ ] `docker --version` shows v24+
- [ ] `docker-compose --version` shows v2+
- [ ] Firewall rules: ports 9090, 3000, 9093 open
- [ ] `docker ps` shows 3 containers: prometheus, grafana, alertmanager
- [ ] Can access Grafana at http://[cx23-ip]:3000
- [ ] Can access Prometheus at http://[cx23-ip]:9090

---

## 🛠️ Tasks Checklist

### Task 1: System Update
```bash
sudo apt-get update
sudo apt-get upgrade -y
sudo apt-get install -y curl wget git
```

Verify: `uname -a` shows recent kernel

### Task 2: Install Docker
```bash
curl -fsSL https://get.docker.com | bash
sudo usermod -aG docker $USER  # Add current user to docker group
newgrp docker  # Activate group (or logout/login)
docker --version  # Should show v24+
```

### Task 3: Install Docker Compose
```bash
sudo apt-get install -y docker-compose
docker-compose --version  # Should show v2+
```

### Task 4: Create Monitoring Directory
```bash
sudo mkdir -p /opt/monitoring
sudo chown $USER:$USER /opt/monitoring
cd /opt/monitoring
```

### Task 5: Create docker-compose.yml

Create `/opt/monitoring/docker-compose.yml`:

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./alert-rules.yml:/etc/prometheus/alert-rules.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'
    restart: unless-stopped
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin  # CHANGE THIS!
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana-data:/var/lib/grafana
    restart: unless-stopped
    networks:
      - monitoring

  alertmanager:
    image: prom/alertmanager:latest
    container_name: alertmanager
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro
      - alertmanager-data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
    restart: unless-stopped
    networks:
      - monitoring

volumes:
  prometheus-data:
  grafana-data:
  alertmanager-data:

networks:
  monitoring:
    driver: bridge
```

### Task 6: Create Prometheus Configuration

Create `/opt/monitoring/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  scrape_timeout: 10s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - alertmanager:9093

rule_files:
  - 'alert-rules.yml'

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node'
    static_configs:
      - targets: ['[MAIN_SERVER_IP]:9100']  # Will be updated in Phase 2
```

**Replace `[MAIN_SERVER_IP]` later in Phase 2.**

### Task 7: Create Alert Rules (Skeleton)

Create `/opt/monitoring/alert-rules.yml`:

```yaml
groups:
  - name: system_alerts
    interval: 15s
    rules:
      - alert: HighMemoryUsage
        expr: '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 0.80'
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage on {{ $labels.instance }}"
          description: "Memory usage is {{ $value | humanizePercentage }}"

      - alert: CriticalMemoryUsage
        expr: '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 0.85'
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "CRITICAL memory usage on {{ $labels.instance }}"
          description: "Memory usage is {{ $value | humanizePercentage }}"

      - alert: HighSwapUsage
        expr: 'node_memory_SwapFree_bytes / node_memory_SwapTotal_bytes < 0.80'
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High swap usage on {{ $labels.instance }}"
```

### Task 8: Create AlertManager Configuration

Create `/opt/monitoring/alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: 'default'
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 4h

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:5001/webhook'  # Placeholder - will update in Phase 3
```

### Task 9: Start Docker Containers
```bash
cd /opt/monitoring
docker-compose up -d
sleep 10
docker ps
```

Should show 3 running containers:
```
prometheus      prom/prometheus:latest
grafana         grafana/grafana:latest
alertmanager    prom/alertmanager:latest
```

### Task 10: Configure Firewall

If using ufw (Ubuntu):
```bash
sudo ufw allow 9090/tcp  # Prometheus
sudo ufw allow 3000/tcp  # Grafana
sudo ufw allow 9093/tcp  # AlertManager
sudo ufw status
```

### Task 11: Verify Services

Access each service:

**Prometheus:**
```bash
curl http://localhost:9090
# Should return HTML status page
```

**Grafana:**
```bash
curl http://localhost:3000
# Should return HTML login page
```

**AlertManager:**
```bash
curl http://localhost:9093
# Should return HTML status page
```

### Task 12: Test External Connectivity

From your local machine:
```bash
curl http://[CX23_IP]:9090  # Should reach Prometheus
curl http://[CX23_IP]:3000  # Should reach Grafana
curl http://[CX23_IP]:9093  # Should reach AlertManager
```

---

## ⚠️ Important Notes

### Passwords
- **Grafana default:** admin/admin
- **CHANGE THIS IMMEDIATELY** in next task
- Use: http://[cx23-ip]:3000 → Admin menu → Account → Change password

### Networking
- Containers communicate via Docker network (`monitoring`)
- Services exposed on host ports (9090, 3000, 9093)
- No external database needed (data stored in volumes)

### Data Persistence
- Prometheus data: `/var/lib/docker/volumes/prometheus-data`
- Grafana dashboards: `/var/lib/docker/volumes/grafana-data`
- AlertManager config: `/var/lib/docker/volumes/alertmanager-data`

### Backup Strategy
- Monthly: Backup config files (`prometheus.yml`, `alertmanager.yml`)
- Weekly: Backup Grafana dashboards (export as JSON)
- Daily: Prometheus retention set to 30 days

---

## 🔄 Next Phase

Once Phase 1 is complete:
- Move to **Phase 2: node-exporter Setup** (Issue #26)
- Install node-exporter on main Energy Insights NL server
- Update `prometheus.yml` with correct IP address
- Restart Prometheus to start scraping metrics

---

## 📞 Troubleshooting

### Container won't start?
```bash
docker-compose logs prometheus  # Check logs
docker-compose restart prometheus  # Restart specific service
```

### Can't access Grafana?
- Verify firewall: `sudo ufw status`
- Check container: `docker ps -a`
- Restart all: `docker-compose restart`

### Prometheus not scraping?
- Check config: `docker exec prometheus cat /etc/prometheus/prometheus.yml`
- Verify target reachable: `curl http://[target-ip]:9100`
- Wait 30s for first scrape

### Slack webhook not configured yet?
- Normal - Phase 3 handles AlertManager webhook setup
- For now, AlertManager routes to localhost (will be updated)

---

## ✅ Phase 1 Complete Checklist

Before proceeding to Phase 2, verify all below:

- [ ] CX23 SSH access confirmed
- [ ] Docker installed and running
- [ ] Docker Compose installed and working
- [ ] 3 containers deployed and healthy
- [ ] Prometheus accessible at :9090
- [ ] Grafana accessible at :3000
- [ ] AlertManager accessible at :9093
- [ ] Firewall rules configured (9090, 3000, 9093)
- [ ] Grafana password changed from default
- [ ] External connectivity tested from local machine
- [ ] docker-compose.yml backed up in git repo

---

## 📝 Commands Quick Reference

```bash
# View all containers
docker ps -a

# View logs for specific service
docker-compose logs -f prometheus

# Restart a service
docker-compose restart grafana

# Stop all services
docker-compose stop

# Start all services
docker-compose start

# Rebuild (if config changed)
docker-compose down && docker-compose up -d

# Monitor resource usage
docker stats

# SSH into container (if needed for debugging)
docker exec -it prometheus /bin/sh
```

---

**Status:** Ready for Phase 1 Execution
**Next:** Issue #26 (Phase 2: node-exporter Setup)
**Questions?** See MONITORING_PROJECT_OVERVIEW.md for context
