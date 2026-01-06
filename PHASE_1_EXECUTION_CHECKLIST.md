# Phase 1 Execution Checklist

**Start Date:** 2026-01-06
**Goal:** Get monitoring server (CX23) operational with Docker
**Time:** ~45 minutes

---

## Pre-Execution Checklist

- [ ] CX23 server provisioned (Ubuntu 24.04)
- [ ] SSH access working as root
- [ ] Internet connectivity verified
- [ ] /opt/monitoring directory ready (or will create)
- [ ] This checklist printed/accessible

---

## Step 1: System Preparation (5 min)

**On CX23:**

```bash
# Login
ssh root@<CX23_IP>

# Update system
sudo apt-get update
sudo apt-get upgrade -y

# Verify Ubuntu version
lsb_release -a
# Should show: Ubuntu 24.04
```

**Checklist:**
- [ ] SSH successful
- [ ] apt-get update completed
- [ ] Ubuntu 24.04 confirmed

---

## Step 2: Install Docker (5 min)

```bash
# Install Docker
curl -fsSL https://get.docker.com | bash

# Verify installation
docker --version
# Should show: Docker version 24+

# Install Docker Compose
apt-get install -y docker-compose

# Verify
docker-compose --version
# Should show: Docker Compose version 2+

# Test Docker
docker ps
# Should show: CONTAINER ID (empty list OK)
```

**Checklist:**
- [ ] Docker installed (version 24+)
- [ ] Docker Compose installed (version 2+)
- [ ] docker ps runs successfully

---

## Step 3: Create Directory Structure (2 min)

```bash
# Create monitoring directory
mkdir -p /opt/monitoring
cd /opt/monitoring

# Verify
pwd
# Should show: /opt/monitoring
```

**Checklist:**
- [ ] /opt/monitoring directory created
- [ ] Current directory is /opt/monitoring

---

## Step 4: Create docker-compose.yml (3 min)

**Create file:** /opt/monitoring/docker-compose.yml

```bash
cat > /opt/monitoring/docker-compose.yml <<'EOF'
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./alerts.yml:/etc/prometheus/alerts.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=15d'
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
      - GF_SECURITY_ADMIN_PASSWORD=admin
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
EOF
```

**Checklist:**
- [ ] docker-compose.yml created
- [ ] 3 services defined: prometheus, grafana, alertmanager
- [ ] File saved in /opt/monitoring

---

## Step 5: Create Prometheus Configuration (3 min)

**Create file:** /opt/monitoring/prometheus.yml

```bash
cat > /opt/monitoring/prometheus.yml <<'EOF'
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

**Checklist:**
- [ ] prometheus.yml created
- [ ] Contains 3 scrape jobs
- [ ] File saved in /opt/monitoring

---

## Step 6: Copy Alert Rules (2 min)

```bash
# Copy from repository
cp /opt/github/synctacles-api/monitoring/prometheus/alerts.yml /opt/monitoring/

# Verify
ls -lh /opt/monitoring/alerts.yml
# Should show: alerts.yml with content
```

**Checklist:**
- [ ] alerts.yml copied
- [ ] File exists in /opt/monitoring

---

## Step 7: Create AlertManager Config (2 min)

**Create file:** /opt/monitoring/alertmanager.yml

```bash
cat > /opt/monitoring/alertmanager.yml <<'EOF'
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
      - url: 'http://localhost:5001/webhook'
EOF
```

**Checklist:**
- [ ] alertmanager.yml created
- [ ] File saved in /opt/monitoring

---

## Step 8: Start Docker Containers (5 min)

```bash
# From /opt/monitoring directory
cd /opt/monitoring

# Start all containers
docker-compose up -d

# Wait for startup
sleep 10

# Check running containers
docker ps
# Should show 3 containers: prometheus, grafana, alertmanager (all healthy)

# View logs (if needed for debugging)
docker-compose logs
```

**Checklist:**
- [ ] docker-compose up -d successful
- [ ] docker ps shows 3 running containers
- [ ] All containers have status "Up"

---

## Step 9: Verify Services (5 min)

```bash
# Test Prometheus
curl http://localhost:9090/-/ready
# Should return: OK

# Test Grafana
curl http://localhost:3000
# Should return HTML (not an error)

# Test node_exporter (on monitoring server)
curl http://localhost:9100/metrics | head -5
# Should show metric lines like: node_cpu_seconds_total

# Check alert rules loaded
curl http://localhost:9090/api/v1/rules | grep -o '"name":"' | wc -l
# Should show: at least 1 (number of alert rule groups)
```

**Checklist:**
- [ ] Prometheus API responds with "OK"
- [ ] Grafana HTML returned
- [ ] node_exporter metrics available
- [ ] Alert rules loaded (>0)

---

## Step 10: Access Web UIs (3 min)

From your local machine (not SSH):

1. **Prometheus:**
   ```
   http://<CX23_IP>:9090
   ```
   - Should show Prometheus UI
   - Check `/targets` page (should be empty for now - will add servers later)
   - Check `/alerts` page (should show alert groups)

2. **Grafana:**
   ```
   http://<CX23_IP>:3000
   ```
   - Should show login page
   - Username: admin
   - Password: admin
   - **DO NOT SKIP:** Change password on first login!
     - Click: Admin menu (top left) → Account → Change Password
     - Set strong password

3. **AlertManager:**
   ```
   http://<CX23_IP>:9093
   ```
   - Should show AlertManager UI

**Checklist:**
- [ ] Prometheus accessible and showing targets page
- [ ] Grafana accessible and login works
- [ ] Grafana password changed from default
- [ ] AlertManager accessible

---

## Step 11: Run Validation Tests (3 min)

```bash
# Copy test script
cp /opt/github/synctacles-api/scripts/monitoring/test_monitoring.sh /opt/monitoring/

# Make executable
chmod +x /opt/monitoring/test_monitoring.sh

# Run tests
cd /opt/monitoring
./test_monitoring.sh
```

**Expected output:**
- ✅ All containers running
- ✅ Prometheus API: responsive
- ✅ Grafana API: responsive
- ✅ node_exporter: publishing metrics
- ✅ Alert rules: loaded
- ✅ Firewall: ports open
- ✅ Disk space: OK

**Checklist:**
- [ ] test_monitoring.sh runs without errors
- [ ] All 8 tests pass (or mostly pass with warnings OK)

---

## Step 12: Firewall Configuration (3 min)

```bash
# Check if UFW enabled
sudo ufw status

# If not enabled and you want to enable:
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    comment "SSH"
sudo ufw allow 9090/tcp  comment "Prometheus"
sudo ufw allow 3000/tcp  comment "Grafana"
sudo ufw allow 9093/tcp  comment "AlertManager"
sudo ufw --force enable

# Verify
sudo ufw status
# Should show: Status: active (with rules)
```

**Checklist:**
- [ ] UFW status checked
- [ ] If enabling: all ports allowed
- [ ] Firewall verified active (if applicable)

---

## ✅ Phase 1 Complete Verification

```bash
# Final verification commands
echo "=== Docker Containers ==="
docker ps -a

echo ""
echo "=== Services Health ==="
curl -s http://localhost:9090/-/ready && echo "Prometheus: OK" || echo "Prometheus: FAILED"
curl -s http://localhost:3000 > /dev/null && echo "Grafana: OK" || echo "Grafana: FAILED"
curl -s http://localhost:9100/metrics > /dev/null && echo "node_exporter: OK" || echo "node_exporter: FAILED"

echo ""
echo "=== Directory Structure ==="
ls -lh /opt/monitoring/

echo ""
echo "=== Memory Usage ==="
free -h
```

**Phase 1 Complete Checklist:**
- [ ] All 12 steps completed
- [ ] All containers running
- [ ] All web UIs accessible
- [ ] Validation tests passing
- [ ] Grafana password changed
- [ ] Firewall configured
- [ ] CX23_IP noted for later use

---

## 📝 Important Notes

### IP Address to Remember
Your CX23 IP: `___________________`

(Fill in your actual CX23 IP - you'll need this for Phase 2)

### Credentials to Change
- Grafana default password (admin/admin) → **ALREADY CHANGED** ✅

### For Phase 2
You'll need:
1. CX23 monitoring server IP
2. Scripts copied to repo (already done):
   - setup_application_server.sh
   - add_target.sh

### If Something Goes Wrong
1. Check logs: `docker-compose logs`
2. Restart: `docker-compose restart`
3. Rebuild: `docker-compose down && docker-compose up -d`
4. See MONITORING_SETUP.md troubleshooting section

---

## ⏱️ Estimated Timeline

| Step | Time | Cumulative |
|------|------|-----------|
| 1. System Preparation | 5 min | 5 min |
| 2. Install Docker | 5 min | 10 min |
| 3. Create Directory | 2 min | 12 min |
| 4. docker-compose.yml | 3 min | 15 min |
| 5. Prometheus Config | 3 min | 18 min |
| 6. Copy Alert Rules | 2 min | 20 min |
| 7. AlertManager Config | 2 min | 22 min |
| 8. Start Containers | 5 min | 27 min |
| 9. Verify Services | 5 min | 32 min |
| 10. Access Web UIs | 3 min | 35 min |
| 11. Run Tests | 3 min | 38 min |
| 12. Firewall | 3 min | 41 min |
| **Total** | - | **~45 min** |

---

## ✅ When Complete

Message me:
```
Phase 1 complete! CX23 IP: <your_ip>
Monitoring server operational.
Ready for Phase 2: node_exporter setup
```

Then I will:
1. Update GitHub issue #25 with completion
2. Provide Phase 2 checklist
3. Help with Phase 2 execution
