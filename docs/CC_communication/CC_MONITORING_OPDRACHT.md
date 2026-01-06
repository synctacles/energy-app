# CC OPDRACHT: Monitoring & Alerting Infrastructure

**Datum:** 2026-01-06
**Prioriteit:** HIGH
**Geschatte tijd:** 4-6 uur
**Doel:** Veilige monitoring server setup met volledige automatisering

---

## CONTEXT

Leo richt aparte monitoring server in (Hetzner CX23) voor Prometheus + Grafana.
Deze server moet 3-5 SYNCTACLES application servers monitoren met focus op:
- Memory usage (OOM prevention)
- CPU load
- Disk space
- Service health (API, collectors, normalizers)
- Database connections

**Huidige situatie:**
- Monitoring verwijderd van application server (memory besparing)
- Nieuwe server: Ubuntu 24.04, root SSH werkt
- Target servers: energy-insights-nl-api (+ 2-4 toekomstige deployments)

**Recent incident:** OOM crash op application server door:
- Geen swap (nu 4GB toegevoegd)
- PostgreSQL shared_buffers te hoog (nu 1GB)
- 9 Gunicorn workers (540MB) + Grafana + collectors = memory spike

**Doel monitoring:** Voorkomen dat dit opnieuw gebeurt via proactive alerting.

---

## DELIVERABLES

### 1. SKILL Document
**Bestand:** `SKILL_14_MONITORING_INFRASTRUCTURE.md`

**Inhoud:**
- **Purpose:** Waarom aparte monitoring server (memory isolation, central oversight)
- **Architecture:** 1 monitoring server → N application servers (scrape model)
- **Hardware requirements:** 
  - CX23 (2 vCPU, 4GB RAM) voor 3-5 servers
  - CX33 (4 vCPU, 8GB RAM) voor 10-15 servers
  - Upgrade path: on-the-fly via Hetzner console (2-5 min downtime)
- **Installation workflow:** Van clean Ubuntu 24.04 naar production-ready
- **Security model:** Dedicated `monitoring` user (analoog aan `energy-insights-nl` pattern in SKILL_11)
- **Metrics collected:** 
  - node_exporter: CPU, memory, disk, network
  - API metrics: request rate, latency, errors (future)
  - Database metrics: connections, query time (future)
- **Alert rules:** 
  - Memory > 80% (warning)
  - Memory > 90% (critical)
  - Swap > 50% (warning - indicates pressure)
  - Disk > 85% (warning)
  - Service down > 2min (critical)
  - CPU > 80% for 10min (warning)
- **Dashboards:** 
  - Overview: All servers at a glance
  - Memory Analysis: OOM prevention focus (swap usage, buffers, cache)
  - API Performance: Latency, throughput, errors
- **Maintenance:** Backup (Prometheus data retention 15 days), updates, scaling strategy

**Referenties:**
- SKILL_08: Hardware profiles (memory tuning guidelines)
- SKILL_11: Service account patterns (dedicated user per service)
- SKILL_13: Logging standards (integration with existing logs)

**Format:** Volg SKILL template structuur (zie andere SKILL files als voorbeeld)

---

### 2. Directory Structure
**Locatie:** `/opt/github/synctacles-api/`

```
monitoring/
├── README.md                           # Quick start guide
├── prometheus/
│   ├── prometheus.yml.template         # Main config with {{PLACEHOLDERS}}
│   ├── alerts.yml.template             # Alert rules
│   └── recording_rules.yml.template    # Aggregations (optional)
├── grafana/
│   ├── dashboards/
│   │   ├── synctacles_overview.json    # Main dashboard
│   │   ├── memory_analysis.json        # OOM prevention focus
│   │   └── api_performance.json        # Request latency, errors
│   └── datasources/
│       └── prometheus.yml              # Datasource config
└── node_exporter/
    └── node_exporter.service.template  # For application servers

scripts/
└── monitoring/
    ├── install_monitoring.sh           # MAIN INSTALLER (zie spec hieronder)
    ├── configure_monitoring.sh         # Interactive .env generation
    ├── add_target.sh                   # Add server to monitoring
    ├── setup_application_server.sh     # Install node_exporter on app servers
    └── test_monitoring.sh              # Validation script

docs/
└── operations/
    ├── MONITORING_SETUP.md             # User guide (step-by-step)
    └── MONITORING_RUNBOOK.md           # Troubleshooting, common tasks
```

---

### 3. MAIN INSTALLER SCRIPT
**Bestand:** `scripts/monitoring/install_monitoring.sh`

**Requirements:**
- **Fail-fast:** Check `.env` exists voor installatie start
- **Idempotent:** Kan herhaald worden zonder errors
- **Logging:** Alle acties naar `/var/log/monitoring-install.log` + stdout
- **Phases:** Duidelijke stappen met progress indicators
- **Rollback:** Bij failure, log error + suggest manual cleanup

**Script header:**
```bash
#!/bin/bash
# SYNCTACLES Monitoring Server Installer
# Installs: Prometheus, Grafana, Node Exporter, Alertmanager
# Requires: Ubuntu 24.04, root access, /opt/monitoring/.env
# Usage: sudo ./install_monitoring.sh

set -euo pipefail  # Fail-fast
exec 1> >(tee -a /var/log/monitoring-install.log)
exec 2>&1

echo "════════════════════════════════════════════════"
echo "SYNCTACLES Monitoring Server Installation"
echo "Started: $(date)"
echo "════════════════════════════════════════════════"
```

**Phases (8 stappen):**

#### Phase 0: Pre-flight Checks
```bash
echo ""
echo "[Phase 0/8] Pre-flight checks..."

# Check Ubuntu version
if ! grep -q "Ubuntu 24.04" /etc/os-release; then
    echo "❌ ERROR: Ubuntu 24.04 required"
    exit 1
fi

# Check root
if [[ $EUID -ne 0 ]]; then
    echo "❌ ERROR: Must run as root"
    exit 1
fi

# Check .env exists
if [[ ! -f /opt/monitoring/.env ]]; then
    echo "❌ ERROR: /opt/monitoring/.env not found"
    echo "Run: sudo ./configure_monitoring.sh first"
    exit 1
fi

# Check internet
if ! ping -c 1 8.8.8.8 &>/dev/null; then
    echo "❌ ERROR: No internet connectivity"
    exit 1
fi

# Check disk space
AVAILABLE=$(df /opt | awk 'NR==2 {print $4}')
if [[ $AVAILABLE -lt 3000000 ]]; then
    echo "❌ ERROR: Insufficient disk space (need 3GB free)"
    exit 1
fi

echo "✅ Pre-flight checks passed"
```

#### Phase 1: System Update
```bash
echo ""
echo "[Phase 1/8] System update..."

apt update
apt upgrade -y
apt full-upgrade -y
apt autoremove -y
apt autoclean

# Check reboot required
if [ -f /var/run/reboot-required ]; then
    echo "⚠️  Kernel updated - reboot required after installation"
    echo "   Command: sudo reboot"
fi

echo "✅ System updated"
```

#### Phase 2: Security Hardening
```bash
echo ""
echo "[Phase 2/8] Security hardening..."

# Firewall setup
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    comment "SSH"
ufw allow 9090/tcp  comment "Prometheus"
ufw allow 3000/tcp  comment "Grafana"
ufw --force enable

# Fail2ban installation
apt install -y fail2ban
systemctl enable --now fail2ban

# SSH hardening (⚠️ ROOT BLIJFT ENABLED)
# Backup original config
cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup.$(date +%Y%m%d)

# Harden SSH (maar root blijft!)
sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config
# ⚠️ SKIP: PermitRootLogin blijft op yes (Leo's requirement)

systemctl restart sshd

# Unattended security updates
apt install -y unattended-upgrades
dpkg-reconfigure -plow unattended-upgrades

echo "✅ Security hardened"
echo "⚠️  NOTE: Root SSH still enabled (manual disable recommended after testing)"
```

**⚠️ KRITIEK: Root SSH wordt NIET uitgeschakeld**
- Leo moet expliciet toestemming geven voor `PermitRootLogin no`
- Script print duidelijke waarschuwing in output
- Documenteer in MONITORING_SETUP.md hoe root handmatig uit te schakelen

#### Phase 3: Create Monitoring User
```bash
echo ""
echo "[Phase 3/8] Creating monitoring user..."

# Create dedicated monitoring user (analoog aan SKILL_11)
if ! id -u monitoring &>/dev/null; then
    adduser --system --group --home /home/monitoring --shell /bin/bash monitoring
    echo "Created user: monitoring"
else
    echo "User monitoring already exists"
fi

# Sudo rights
echo "monitoring ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/monitoring
chmod 440 /etc/sudoers.d/monitoring

# SSH keys (copy from root if available)
mkdir -p /home/monitoring/.ssh
if [[ -f /root/.ssh/authorized_keys ]]; then
    cp /root/.ssh/authorized_keys /home/monitoring/.ssh/
    echo "Copied SSH keys from root"
fi
chown -R monitoring:monitoring /home/monitoring/.ssh
chmod 700 /home/monitoring/.ssh
chmod 600 /home/monitoring/.ssh/authorized_keys 2>/dev/null || true

# Git config
sudo -u monitoring git config --global user.name "Monitoring Server"
sudo -u monitoring git config --global user.email "monitoring@synctacles.io"

echo "✅ Monitoring user created"
```

#### Phase 4: Install Prometheus
```bash
echo ""
echo "[Phase 4/8] Installing Prometheus..."

# Install from official repo (not snap - per best practice)
apt install -y prometheus prometheus-node-exporter

# Load .env
source /opt/monitoring/.env

# Configure from template
if [[ -f /opt/github/synctacles-api/monitoring/prometheus/prometheus.yml.template ]]; then
    cp /opt/github/synctacles-api/monitoring/prometheus/prometheus.yml.template \
       /etc/prometheus/prometheus.yml
    
    # Substitute placeholders
    sed -i "s/{{SCRAPE_INTERVAL}}/${SCRAPE_INTERVAL:-15s}/g" /etc/prometheus/prometheus.yml
    sed -i "s/{{RETENTION}}/${RETENTION:-15d}/g" /etc/prometheus/prometheus.yml
else
    echo "⚠️  WARNING: prometheus.yml.template not found, using defaults"
fi

# Copy alert rules
if [[ -f /opt/github/synctacles-api/monitoring/prometheus/alerts.yml.template ]]; then
    cp /opt/github/synctacles-api/monitoring/prometheus/alerts.yml.template \
       /etc/prometheus/alerts.yml
fi

# Ownership
chown -R prometheus:prometheus /etc/prometheus
chown -R prometheus:prometheus /var/lib/prometheus

# Enable and start
systemctl enable --now prometheus
systemctl enable --now prometheus-node-exporter

# Validate
sleep 5
if curl -sf http://localhost:9090/-/ready > /dev/null; then
    echo "✅ Prometheus installed and running"
else
    echo "❌ ERROR: Prometheus failed to start"
    journalctl -u prometheus -n 50
    exit 1
fi
```

#### Phase 5: Install Grafana
```bash
echo ""
echo "[Phase 5/8] Installing Grafana..."

# Add Grafana repo
apt install -y software-properties-common apt-transport-https
wget -q -O - https://packages.grafana.com/gpg.key | apt-key add -
add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
apt update
apt install -y grafana

# Configure datasource
mkdir -p /etc/grafana/provisioning/datasources
if [[ -f /opt/github/synctacles-api/monitoring/grafana/datasources/prometheus.yml ]]; then
    cp /opt/github/synctacles-api/monitoring/grafana/datasources/prometheus.yml \
       /etc/grafana/provisioning/datasources/
fi

# Configure dashboards directory
mkdir -p /etc/grafana/provisioning/dashboards
cat > /etc/grafana/provisioning/dashboards/synctacles.yml <<EOF
apiVersion: 1
providers:
  - name: 'SYNCTACLES'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
EOF

# Copy dashboards
mkdir -p /var/lib/grafana/dashboards
if [[ -d /opt/github/synctacles-api/monitoring/grafana/dashboards ]]; then
    cp /opt/github/synctacles-api/monitoring/grafana/dashboards/*.json \
       /var/lib/grafana/dashboards/ 2>/dev/null || echo "No dashboards found"
fi

# Ownership
chown -R grafana:grafana /etc/grafana
chown -R grafana:grafana /var/lib/grafana

# Enable and start
systemctl enable --now grafana-server

# Validate
sleep 5
if curl -sf http://localhost:3000/api/health > /dev/null; then
    echo "✅ Grafana installed and running"
else
    echo "❌ ERROR: Grafana failed to start"
    journalctl -u grafana-server -n 50
    exit 1
fi
```

#### Phase 6: Install Alertmanager (Optional)
```bash
echo ""
echo "[Phase 6/8] Installing Alertmanager..."

source /opt/monitoring/.env

# Only install if SMTP configured
if [[ -n "${SMTP_HOST:-}" ]]; then
    apt install -y prometheus-alertmanager
    
    # Configure email alerts
    cat > /etc/prometheus/alertmanager.yml <<EOF
global:
  smtp_smarthost: '${SMTP_HOST}:${SMTP_PORT:-587}'
  smtp_from: '${ALERT_EMAIL_FROM:-monitoring@synctacles.io}'
  smtp_auth_username: '${SMTP_USER:-}'
  smtp_auth_password: '${SMTP_PASS:-}'
  smtp_require_tls: true

route:
  receiver: 'email'
  group_by: ['alertname', 'instance']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: 'email'
    email_configs:
      - to: '${ALERT_EMAIL_TO:-${ALERT_EMAIL}}'
        headers:
          Subject: '[SYNCTACLES] {{ .GroupLabels.alertname }}'
EOF

    chown prometheus:prometheus /etc/prometheus/alertmanager.yml
    systemctl enable --now prometheus-alertmanager
    
    echo "✅ Alertmanager installed and configured"
else
    echo "⚠️  SMTP not configured - skipping Alertmanager"
    echo "   Configure later: /etc/prometheus/alertmanager.yml"
fi
```

#### Phase 7: Post-Install Validation
```bash
echo ""
echo "[Phase 7/8] Post-install validation..."

ERRORS=0

# Service health
for service in prometheus grafana-server prometheus-node-exporter; do
    if systemctl is-active --quiet $service; then
        echo "  ✅ $service: active"
    else
        echo "  ❌ $service: inactive"
        ((ERRORS++))
    fi
done

# Connectivity
if curl -sf http://localhost:9090/-/ready > /dev/null; then
    echo "  ✅ Prometheus API: responsive"
else
    echo "  ❌ Prometheus API: unreachable"
    ((ERRORS++))
fi

if curl -sf http://localhost:3000/api/health > /dev/null; then
    echo "  ✅ Grafana API: responsive"
else
    echo "  ❌ Grafana API: unreachable"
    ((ERRORS++))
fi

# Firewall
if ufw status | grep -qE "(9090|3000|22).*ALLOW"; then
    echo "  ✅ Firewall: monitoring ports open"
else
    echo "  ❌ Firewall: ports not configured"
    ((ERRORS++))
fi

# Disk space
DISK_USAGE=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
if [[ "$DISK_USAGE" -lt 85 ]]; then
    echo "  ✅ Disk space: ${DISK_USAGE}% used"
else
    echo "  ⚠️  Disk space: ${DISK_USAGE}% used (> 85%)"
fi

if [[ $ERRORS -gt 0 ]]; then
    echo ""
    echo "❌ $ERRORS validation error(s) - check logs above"
    exit 1
fi

echo "✅ Validation passed"
```

#### Phase 8: Summary & Next Steps
```bash
echo ""
echo "════════════════════════════════════════════════"
echo "✅ MONITORING SERVER INSTALLATION COMPLETE"
echo "════════════════════════════════════════════════"
echo ""
echo "Services:"
echo "  Prometheus:    http://$(hostname -I | awk '{print $1}'):9090"
echo "  Grafana:       http://$(hostname -I | awk '{print $1}'):3000"
echo "  Node Exporter: http://$(hostname -I | awk '{print $1}'):9100/metrics"
echo ""
echo "Credentials:"
echo "  Grafana:     admin / admin"
echo "  ⚠️  CHANGE PASSWORD ON FIRST LOGIN!"
echo ""
echo "⚠️  SECURITY NOTICE:"
echo "  - Root SSH is still ENABLED (as requested)"
echo "  - Recommended: Disable after testing with monitoring user"
echo "  - Command: sudo sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config"
echo "  - Then: sudo systemctl restart sshd"
echo ""
echo "Next steps:"
echo "  1. Change Grafana password: http://$(hostname -I | awk '{print $1}'):3000"
echo "  2. Add application servers:"
echo "     ./scripts/monitoring/add_target.sh <server_ip> <server_name>"
echo "  3. Install node_exporter on application servers:"
echo "     ./scripts/monitoring/setup_application_server.sh"
echo "  4. Test monitoring:"
echo "     ./scripts/monitoring/test_monitoring.sh"
echo ""
echo "Logs: /var/log/monitoring-install.log"
echo ""
echo "Completed: $(date)"
echo "════════════════════════════════════════════════"

# Check reboot needed
if [ -f /var/run/reboot-required ]; then
    echo ""
    echo "⚠️⚠️⚠️  REBOOT REQUIRED (kernel update)  ⚠️⚠️⚠️"
    echo "Run: sudo reboot"
    echo ""
fi
```

**Script footer:**
```bash
exit 0
```

---

### 4. Configuration Script
**Bestand:** `scripts/monitoring/configure_monitoring.sh`

**Interactive prompts voor .env generatie:**

```bash
#!/bin/bash
# SYNCTACLES Monitoring Configuration
# Interactive .env generation for monitoring server

set -euo pipefail

echo "════════════════════════════════════════════════"
echo "SYNCTACLES Monitoring Configuration"
echo "════════════════════════════════════════════════"
echo ""

# Check if .env already exists
if [[ -f /opt/monitoring/.env ]]; then
    echo "⚠️  /opt/monitoring/.env already exists"
    read -p "Overwrite? (y/n): " OVERWRITE
    if [[ "$OVERWRITE" != "y" ]]; then
        echo "Aborted. Using existing .env"
        exit 0
    fi
fi

# Scrape interval
read -p "Scrape interval (default: 15s): " SCRAPE_INTERVAL
SCRAPE_INTERVAL=${SCRAPE_INTERVAL:-15s}

# Retention
read -p "Data retention (default: 15d): " RETENTION
RETENTION=${RETENTION:-15d}

# Alert email
echo ""
echo "Alert Configuration:"
read -p "Alert email address: " ALERT_EMAIL
while [[ ! "$ALERT_EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; do
    echo "❌ Invalid email format"
    read -p "Alert email address: " ALERT_EMAIL
done

# SMTP (optional)
echo ""
read -p "Configure SMTP for email alerts? (y/n): " SMTP_CONFIG
if [[ "$SMTP_CONFIG" == "y" ]]; then
    read -p "SMTP host (e.g., smtp.gmail.com): " SMTP_HOST
    read -p "SMTP port (default: 587): " SMTP_PORT
    SMTP_PORT=${SMTP_PORT:-587}
    read -p "SMTP username: " SMTP_USER
    read -sp "SMTP password: " SMTP_PASS
    echo ""
    read -p "Alert 'from' email (default: monitoring@synctacles.io): " ALERT_EMAIL_FROM
    ALERT_EMAIL_FROM=${ALERT_EMAIL_FROM:-monitoring@synctacles.io}
else
    SMTP_HOST=""
    SMTP_PORT=""
    SMTP_USER=""
    SMTP_PASS=""
    ALERT_EMAIL_FROM="monitoring@synctacles.io"
fi

# Generate .env
echo ""
echo "Generating configuration..."
mkdir -p /opt/monitoring
cat > /opt/monitoring/.env <<EOF
# SYNCTACLES Monitoring Configuration
# Generated: $(date)
# WARNING: Contains sensitive data - do not commit to git

# Prometheus
SCRAPE_INTERVAL=$SCRAPE_INTERVAL
RETENTION=$RETENTION

# Alerting
ALERT_EMAIL=$ALERT_EMAIL
ALERT_EMAIL_TO=$ALERT_EMAIL
ALERT_EMAIL_FROM=$ALERT_EMAIL_FROM

# SMTP (leave empty to disable email alerts)
SMTP_HOST=$SMTP_HOST
SMTP_PORT=$SMTP_PORT
SMTP_USER=$SMTP_USER
SMTP_PASS=$SMTP_PASS

# Grafana (defaults - change on first login)
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASS=admin
EOF

chmod 600 /opt/monitoring/.env
chown root:root /opt/monitoring/.env

echo ""
echo "✅ Configuration saved to /opt/monitoring/.env"
echo ""
echo "Review: cat /opt/monitoring/.env"
echo ""
echo "Next: Run installation script"
echo "  sudo ./scripts/monitoring/install_monitoring.sh"
echo ""
```

---

### 5. Alert Rules
**Bestand:** `monitoring/prometheus/alerts.yml.template`

**Memory-focused alert rules (OOM prevention):**

```yaml
groups:
  - name: synctacles_memory_alerts
    interval: 30s
    rules:
      # Memory alerts (OOM prevention - highest priority)
      - alert: MemoryPressureWarning
        expr: (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 < 20
        for: 5m
        labels:
          severity: warning
          category: memory
        annotations:
          summary: "Memory pressure on {{ $labels.instance }}"
          description: "Available memory < 20% (current: {{ $value | humanize }}%)\nAction: Investigate memory usage, consider adding swap or reducing services."

      - alert: MemoryPressureCritical
        expr: (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 < 10
        for: 2m
        labels:
          severity: critical
          category: memory
        annotations:
          summary: "CRITICAL memory pressure on {{ $labels.instance }}"
          description: "Available memory < 10% (current: {{ $value | humanize }}%) - OOM imminent!\nImmediate action required: Reduce memory usage or add capacity."

      - alert: SwapUsageHigh
        expr: (node_memory_SwapTotal_bytes - node_memory_SwapFree_bytes) / node_memory_SwapTotal_bytes * 100 > 50
        for: 10m
        labels:
          severity: warning
          category: memory
        annotations:
          summary: "High swap usage on {{ $labels.instance }}"
          description: "Swap usage > 50% (current: {{ $value | humanize }}%)\nIndicates memory pressure - investigate application memory leaks."

      - alert: NoSwapConfigured
        expr: node_memory_SwapTotal_bytes == 0
        for: 5m
        labels:
          severity: warning
          category: memory
        annotations:
          summary: "No swap configured on {{ $labels.instance }}"
          description: "Server has no swap space - vulnerable to OOM crashes.\nRecommended: Add 4GB swap for emergency buffer."

  - name: synctacles_disk_alerts
    interval: 60s
    rules:
      - alert: DiskSpaceLow
        expr: (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100 < 15
        for: 5m
        labels:
          severity: warning
          category: disk
        annotations:
          summary: "Low disk space on {{ $labels.instance }}"
          description: "Disk space < 15% (current: {{ $value | humanize }}%)\nAction: Clean up logs, old backups, or expand disk."

      - alert: DiskSpaceCritical
        expr: (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100 < 5
        for: 2m
        labels:
          severity: critical
          category: disk
        annotations:
          summary: "CRITICAL disk space on {{ $labels.instance }}"
          description: "Disk space < 5% (current: {{ $value | humanize }}%)\nImmediate action: Services may fail soon!"

  - name: synctacles_service_alerts
    interval: 30s
    rules:
      - alert: ServiceDown
        expr: up{job="synctacles-servers"} == 0
        for: 2m
        labels:
          severity: critical
          category: service
        annotations:
          summary: "Service down: {{ $labels.instance }}"
          description: "{{ $labels.job }} unreachable for 2+ minutes\nCheck: systemctl status, network connectivity, firewall."

      - alert: NodeExporterDown
        expr: up{job="node"} == 0
        for: 2m
        labels:
          severity: critical
          category: service
        annotations:
          summary: "Node exporter down on {{ $labels.instance }}"
          description: "Cannot collect metrics - check node_exporter service."

  - name: synctacles_cpu_alerts
    interval: 60s
    rules:
      - alert: HighCPUUsage
        expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 10m
        labels:
          severity: warning
          category: cpu
        annotations:
          summary: "High CPU usage on {{ $labels.instance }}"
          description: "CPU usage > 80% for 10+ minutes (current: {{ $value | humanize }}%)\nAction: Investigate high CPU processes (top, htop)."

      - alert: CriticalCPUUsage
        expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 95
        for: 5m
        labels:
          severity: critical
          category: cpu
        annotations:
          summary: "CRITICAL CPU usage on {{ $labels.instance }}"
          description: "CPU usage > 95% for 5+ minutes (current: {{ $value | humanize }}%)\nServer may become unresponsive - immediate action required."

  - name: synctacles_load_alerts
    interval: 60s
    rules:
      - alert: HighSystemLoad
        expr: node_load15 / count(node_cpu_seconds_total{mode="idle"}) without (cpu,mode) > 1.5
        for: 10m
        labels:
          severity: warning
          category: load
        annotations:
          summary: "High system load on {{ $labels.instance }}"
          description: "15-min load average > 1.5x CPU count (current: {{ $value | humanize }})\nAction: Check running processes and resource bottlenecks."
```

**Note:** Deze alert rules zijn specifiek gefocust op OOM prevention (recent incident) en hebben duidelijke actionable descriptions.

---

### 6. Prometheus Configuration Template
**Bestand:** `monitoring/prometheus/prometheus.yml.template`

```yaml
# SYNCTACLES Prometheus Configuration
# Template version - substitute {{PLACEHOLDERS}} during installation

global:
  scrape_interval: {{SCRAPE_INTERVAL}}
  evaluation_interval: {{SCRAPE_INTERVAL}}
  external_labels:
    cluster: 'synctacles'
    environment: 'production'

# Alertmanager configuration
alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - localhost:9093

# Load alert rules
rule_files:
  - /etc/prometheus/alerts.yml

# Scrape configurations
scrape_configs:
  # Monitoring server self-monitoring
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
        labels:
          server_type: 'monitoring'

  # Node exporter on monitoring server
  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']
        labels:
          server_type: 'monitoring'
          server_name: 'monitoring-server'

  # Application servers (add via add_target.sh)
  - job_name: 'synctacles-servers'
    static_configs:
      - targets:
          # AUTO-GENERATED - DO NOT EDIT MANUALLY
          # Use: ./scripts/monitoring/add_target.sh <server_ip> <server_name>
          # Example: - targets: ['10.0.1.5:9100']  # enin-nl-prod
        labels:
          server_type: 'application'
```

---

### 7. Helper Scripts

#### `scripts/monitoring/add_target.sh`
```bash
#!/bin/bash
# Add application server to Prometheus monitoring

set -euo pipefail

# Usage
if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <server_ip> [server_name]"
    echo ""
    echo "Example:"
    echo "  $0 10.0.1.5 enin-nl-prod"
    echo "  $0 192.168.1.100"
    exit 1
fi

SERVER_IP=$1
SERVER_NAME=${2:-"server-$SERVER_IP"}

# Validate IP format
if [[ ! "$SERVER_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "❌ Invalid IP format: $SERVER_IP"
    exit 1
fi

# Check if already exists
if grep -q "$SERVER_IP:9100" /etc/prometheus/prometheus.yml; then
    echo "⚠️  Server $SERVER_IP already in monitoring"
    exit 0
fi

# Backup config
cp /etc/prometheus/prometheus.yml /etc/prometheus/prometheus.yml.backup.$(date +%Y%m%d-%H%M%S)

# Add to targets (insert after # AUTO-GENERATED comment)
sed -i "/# AUTO-GENERATED - DO NOT EDIT MANUALLY/a \          - targets: ['$SERVER_IP:9100']  # $SERVER_NAME" \
    /etc/prometheus/prometheus.yml

# Validate config
if ! promtool check config /etc/prometheus/prometheus.yml; then
    echo "❌ Config validation failed - restoring backup"
    mv /etc/prometheus/prometheus.yml.backup.* /etc/prometheus/prometheus.yml
    exit 1
fi

# Reload Prometheus
systemctl reload prometheus

echo "✅ Added $SERVER_NAME ($SERVER_IP) to monitoring"
echo ""
echo "Next steps:"
echo "  1. Install node_exporter on $SERVER_IP:"
echo "     scp scripts/monitoring/setup_application_server.sh root@$SERVER_IP:/tmp/"
echo "     ssh root@$SERVER_IP 'bash /tmp/setup_application_server.sh'"
echo ""
echo "  2. Check target health (wait 15-30s for first scrape):"
echo "     http://$(hostname -I | awk '{print $1}'):9090/targets"
echo ""
echo "  3. View metrics in Grafana:"
echo "     http://$(hostname -I | awk '{print $1}'):3000"
```

#### `scripts/monitoring/setup_application_server.sh`
```bash
#!/bin/bash
# Install node_exporter on SYNCTACLES application server
# Run this script on the APPLICATION server (not monitoring server)

set -euo pipefail

echo "════════════════════════════════════════════════"
echo "Installing node_exporter on application server"
echo "════════════════════════════════════════════════"
echo ""

# Check root
if [[ $EUID -ne 0 ]]; then
    echo "❌ ERROR: Must run as root"
    exit 1
fi

# Get monitoring server IP
read -p "Monitoring server IP: " MONITORING_IP
if [[ ! "$MONITORING_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "❌ Invalid IP format"
    exit 1
fi

# Install node_exporter
echo "[1/4] Installing node_exporter..."
apt update
apt install -y prometheus-node-exporter

# Enable and start
echo "[2/4] Starting node_exporter..."
systemctl enable --now prometheus-node-exporter

# Configure firewall (allow monitoring server only)
echo "[3/4] Configuring firewall..."
ufw allow from $MONITORING_IP to any port 9100 comment "Prometheus scraping"

# Validate
echo "[4/4] Validating installation..."
sleep 2
if curl -sf http://localhost:9100/metrics | head -5 > /dev/null; then
    echo "✅ Node exporter installed and running"
    echo ""
    echo "Metrics endpoint: http://$(hostname -I | awk '{print $1}'):9100/metrics"
    echo ""
    echo "Next: Add this server to monitoring:"
    echo "  On monitoring server, run:"
    echo "  ./scripts/monitoring/add_target.sh $(hostname -I | awk '{print $1}') $(hostname)"
else
    echo "❌ Node exporter failed to start"
    journalctl -u prometheus-node-exporter -n 20
    exit 1
fi
```

#### `scripts/monitoring/test_monitoring.sh`
```bash
#!/bin/bash
# Comprehensive monitoring stack validation

set -euo pipefail

echo "════════════════════════════════════════════════"
echo "SYNCTACLES Monitoring Validation"
echo "════════════════════════════════════════════════"
echo ""

ERRORS=0

# Test 1: Services running
echo "[1/8] Checking services..."
for service in prometheus grafana-server prometheus-node-exporter; do
    if systemctl is-active --quiet $service; then
        echo "  ✅ $service: active"
    else
        echo "  ❌ $service: inactive"
        ((ERRORS++))
    fi
done

# Test 2: Prometheus API
echo "[2/8] Testing Prometheus API..."
if curl -sf http://localhost:9090/-/ready > /dev/null; then
    echo "  ✅ Prometheus API: responsive"
else
    echo "  ❌ Prometheus API: unreachable"
    ((ERRORS++))
fi

# Test 3: Grafana API
echo "[3/8] Testing Grafana API..."
if curl -sf http://localhost:3000/api/health > /dev/null; then
    echo "  ✅ Grafana API: responsive"
else
    echo "  ❌ Grafana API: unreachable"
    ((ERRORS++))
fi

# Test 4: Node exporter metrics
echo "[4/8] Testing node exporter..."
if curl -sf http://localhost:9100/metrics | grep -q "node_cpu_seconds_total"; then
    echo "  ✅ Node exporter: publishing metrics"
else
    echo "  ❌ Node exporter: no metrics"
    ((ERRORS++))
fi

# Test 5: Targets configured
echo "[5/8] Checking Prometheus targets..."
TARGET_COUNT=$(curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets | length' 2>/dev/null || echo "0")
if [[ "$TARGET_COUNT" -gt 0 ]]; then
    echo "  ✅ Prometheus targets: $TARGET_COUNT configured"
else
    echo "  ⚠️  Prometheus targets: none configured yet"
    echo "     Add with: ./scripts/monitoring/add_target.sh <ip> <name>"
fi

# Test 6: Alert rules loaded
echo "[6/8] Checking alert rules..."
RULE_COUNT=$(curl -s http://localhost:9090/api/v1/rules | jq '.data.groups[0].rules | length' 2>/dev/null || echo "0")
if [[ "$RULE_COUNT" -gt 0 ]]; then
    echo "  ✅ Alert rules: $RULE_COUNT loaded"
else
    echo "  ❌ Alert rules: none loaded"
    ((ERRORS++))
fi

# Test 7: Firewall rules
echo "[7/8] Checking firewall..."
if ufw status | grep -qE "(9090|3000|22).*ALLOW"; then
    echo "  ✅ Firewall: monitoring ports open"
else
    echo "  ❌ Firewall: ports not configured"
    ((ERRORS++))
fi

# Test 8: Disk space
echo "[8/8] Checking disk space..."
DISK_USAGE=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
if [[ "$DISK_USAGE" -lt 85 ]]; then
    echo "  ✅ Disk space: ${DISK_USAGE}% used"
else
    echo "  ⚠️  Disk space: ${DISK_USAGE}% used (> 85%)"
fi

echo ""
echo "════════════════════════════════════════════════"
if [[ "$ERRORS" -eq 0 ]]; then
    echo "✅ ALL TESTS PASSED"
    echo ""
    echo "Monitoring server is operational!"
    echo ""
    echo "Access URLs:"
    echo "  Prometheus: http://$(hostname -I | awk '{print $1}'):9090"
    echo "  Grafana:    http://$(hostname -I | awk '{print $1}'):3000"
    echo ""
    echo "Next steps:"
    echo "  1. Add application servers: ./scripts/monitoring/add_target.sh <ip>"
    echo "  2. Configure Grafana dashboards"
    echo "  3. Set up email alerts (if not done)"
    exit 0
else
    echo "❌ $ERRORS TEST(S) FAILED"
    echo ""
    echo "Troubleshooting:"
    echo "  - Check Prometheus logs: journalctl -u prometheus -n 50"
    echo "  - Check Grafana logs: journalctl -u grafana-server -n 50"
    echo "  - Verify config: promtool check config /etc/prometheus/prometheus.yml"
    echo "  - Check services: systemctl status prometheus grafana-server"
    exit 1
fi
```

---

### 8. Grafana Datasource Configuration
**Bestand:** `monitoring/grafana/datasources/prometheus.yml`

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://localhost:9090
    isDefault: true
    editable: true
    jsonData:
      timeInterval: "15s"
      queryTimeout: "60s"
      httpMethod: "POST"
```

---

### 9. Documentation Files

#### `monitoring/README.md`
**Quick start guide:**

```markdown
# SYNCTACLES Monitoring

Quick setup for monitoring server.

## Prerequisites

- Ubuntu 24.04
- Root access
- 4GB RAM minimum (CX23 or equivalent)
- 40GB disk space

## Installation

1. **Clone repository:**
   ```bash
   git clone <repo-url>
   cd synctacles-api
   ```

2. **Configure monitoring:**
   ```bash
   sudo ./scripts/monitoring/configure_monitoring.sh
   ```

3. **Install monitoring stack:**
   ```bash
   sudo ./scripts/monitoring/install_monitoring.sh
   ```

4. **Validate installation:**
   ```bash
   sudo ./scripts/monitoring/test_monitoring.sh
   ```

5. **Add application servers:**
   ```bash
   ./scripts/monitoring/add_target.sh <server_ip> <server_name>
   ```

## Access

- Prometheus: http://<server-ip>:9090
- Grafana: http://<server-ip>:3000 (admin/admin)

## Documentation

- Full guide: [docs/operations/MONITORING_SETUP.md](../../docs/operations/MONITORING_SETUP.md)
- Troubleshooting: [docs/operations/MONITORING_RUNBOOK.md](../../docs/operations/MONITORING_RUNBOOK.md)
- Architecture: [SKILL_14_MONITORING_INFRASTRUCTURE.md](../../SKILL_14_MONITORING_INFRASTRUCTURE.md)
```

#### `docs/operations/MONITORING_SETUP.md`
**Comprehensive user guide with:**
- Prerequisites checklist
- Step-by-step installation
- Configuration options explained
- Adding application servers
- Dashboard setup
- Alert configuration
- Common issues

#### `docs/operations/MONITORING_RUNBOOK.md`
**Operations manual with:**
- Daily/weekly maintenance tasks
- Troubleshooting decision tree
- Common alerts and responses
- Backup and recovery
- Scaling procedures
- Security best practices

---

## TECHNICAL REQUIREMENTS

### Code Quality
- ✅ **Fail-fast patterns** (SKILL_01): Check requirements before proceeding
- ✅ **Clear error messages** (SKILL_05): [WHAT] - [WHY] - [HOW TO FIX]
- ✅ **Idempotent scripts**: Can re-run safely without breaking
- ✅ **Logging**: All output to file + stdout (tee)
- ✅ **Exit codes**: 0 = success, 1 = failure

### Security
- ✅ **Dedicated user** (SKILL_11): `monitoring` user for services
- ✅ **SSH hardening**: Key-only, but root stays enabled per Leo requirement
- ✅ **Firewall**: UFW with minimal ports (22, 9090, 3000)
- ✅ **Fail2ban**: Brute-force protection
- ✅ **Automatic updates**: Unattended-upgrades for security patches
- ⚠️ **Root SSH warning**: Prominent in installation output

### Documentation
- ✅ **SKILL_14**: Comprehensive monitoring guide
- ✅ **Inline comments**: Script sections explained
- ✅ **User guide**: Step-by-step MONITORING_SETUP.md
- ✅ **Runbook**: Troubleshooting MONITORING_RUNBOOK.md
- ✅ **Templates**: All configs with {{PLACEHOLDERS}}

### Testing
- ✅ **Pre-flight checks**: Validate before installation
- ✅ **Post-install validation**: test_monitoring.sh
- ✅ **Config validation**: promtool check config
- ✅ **Service health**: systemctl status checks

---

## DELIVERABLES CHECKLIST

Voor commit, verifieer:

**Code:**
- [ ] `scripts/monitoring/install_monitoring.sh` - 8 phases compleet, executable
- [ ] `scripts/monitoring/configure_monitoring.sh` - Interactive prompts, executable
- [ ] `scripts/monitoring/add_target.sh` - IP validation, executable
- [ ] `scripts/monitoring/setup_application_server.sh` - Firewall config, executable
- [ ] `scripts/monitoring/test_monitoring.sh` - Comprehensive checks, executable

**Configuration:**
- [ ] `monitoring/prometheus/prometheus.yml.template` - {{PLACEHOLDERS}}
- [ ] `monitoring/prometheus/alerts.yml.template` - Memory-focused rules
- [ ] `monitoring/grafana/datasources/prometheus.yml` - Datasource config
- [ ] `monitoring/grafana/dashboards/*.json` - At least overview dashboard

**Documentation:**
- [ ] `SKILL_14_MONITORING_INFRASTRUCTURE.md` - Complete reference
- [ ] `monitoring/README.md` - Quick start
- [ ] `docs/operations/MONITORING_SETUP.md` - Step-by-step guide
- [ ] `docs/operations/MONITORING_RUNBOOK.md` - Operations manual

**Quality:**
- [ ] All scripts have headers with usage info
- [ ] All scripts fail-fast with clear errors
- [ ] All templates documented in README
- [ ] Root SSH warning prominent in output
- [ ] Logging to /var/log/monitoring-install.log

---

## SUCCESS CRITERIA

**Installation succesvol als:**
1. ✅ Clean Ubuntu 24.04 → running monitoring in < 10 minutes
2. ✅ Prometheus + Grafana + Node Exporter active
3. ✅ Firewall configured, fail2ban running
4. ✅ test_monitoring.sh passes all 8 checks
5. ✅ Grafana accessible on :3000, dashboards loaded
6. ✅ Alert rules loaded in Prometheus (check :9090/alerts)
7. ✅ Root SSH still enabled with prominent warning
8. ✅ `monitoring` user can SSH + sudo

**Documentatie volledig als:**
1. ✅ SKILL_14 standalone leesbaar (no missing context)
2. ✅ User guide has zero ambiguous steps
3. ✅ All {{PLACEHOLDERS}} documented
4. ✅ Troubleshooting covers common issues
5. ✅ Integration with SKILL_08, SKILL_11, SKILL_13 clear

---

## PRIORITY BREAKDOWN

**Phase 1 (Critical - Day 1):**
1. `install_monitoring.sh` with 8 phases
2. Security hardening (firewall, fail2ban, SSH)
3. Basic alert rules (memory, disk, service down)
4. `test_monitoring.sh` validation
5. `configure_monitoring.sh` for .env generation

**Phase 2 (High - Day 2):**
6. Helper scripts (add_target.sh, setup_application_server.sh)
7. SKILL_14 document
8. User guide (MONITORING_SETUP.md)
9. Prometheus configuration template
10. Grafana datasource config

**Phase 3 (Medium - Day 3):**
11. Grafana dashboards (overview, memory analysis, API performance)
12. Runbook (MONITORING_RUNBOOK.md)
13. Advanced alert rules (CPU, load, API errors)
14. Alertmanager email configuration

**Geschatte tijd:** 4-6 uur total (spread over 2-3 days)

---

## GRAFANA DASHBOARDS (Optional but Recommended)

Create JSON exports for:

### 1. Overview Dashboard
**Bestand:** `monitoring/grafana/dashboards/synctacles_overview.json`

**Panels:**
- All servers status (up/down)
- Memory usage (all servers)
- CPU usage (all servers)
- Disk usage (all servers)
- Active alerts count
- Network I/O

### 2. Memory Analysis Dashboard
**Bestand:** `monitoring/grafana/dashboards/memory_analysis.json`

**Panels (OOM prevention focus):**
- Available memory % (time series)
- Swap usage % (time series)
- Memory pressure gauge
- Top memory processes (future)
- Memory alerts timeline
- Buffer/cache breakdown

### 3. API Performance Dashboard
**Bestand:** `monitoring/grafana/dashboards/api_performance.json`

**Panels (future when API metrics available):**
- Request rate (req/s)
- Response time (p50, p95, p99)
- Error rate (%)
- Active connections
- Top slow endpoints

**Note:** If time is limited, skip dashboards and document placeholder structure. Leo can create dashboards via Grafana UI later.

---

## COMMIT INSTRUCTIONS

**Na voltooiing:**

```bash
cd /opt/github/synctacles-api

# Verify structure
tree monitoring/ scripts/monitoring/ docs/operations/ | head -50

# Verify all scripts executable
chmod +x scripts/monitoring/*.sh

# Add all monitoring files
git add monitoring/ scripts/monitoring/ docs/operations/MONITORING_*.md SKILL_14_*.md

# Commit with detailed message
git commit -m "feat: monitoring infrastructure (SKILL_14)

Complete monitoring server setup with focus on OOM prevention:

Installation:
- 8-phase installer with fail-fast validation
- Security hardening (firewall, fail2ban, SSH hardening)
- Prometheus + Grafana + Node Exporter + Alertmanager
- Interactive configuration via .env generation

Alert Rules (Memory Focus):
- Memory < 20%: Warning (investigate)
- Memory < 10%: Critical (OOM imminent)
- Swap > 50%: Warning (memory pressure)
- No swap: Warning (vulnerable to OOM)
- Service down: Critical
- Disk > 85%: Warning
- CPU > 80% for 10min: Warning

Helper Scripts:
- add_target.sh: Add servers to monitoring
- setup_application_server.sh: Install node_exporter
- test_monitoring.sh: Comprehensive validation

Documentation:
- SKILL_14: Complete monitoring reference
- User guide: Step-by-step setup
- Runbook: Operations and troubleshooting

Security Note:
Root SSH remains enabled per requirement.
Manual disable recommended after testing.

Tested on: Ubuntu 24.04, Hetzner CX23 (4GB RAM)
Target: 3-5 SYNCTACLES application servers

Resolves: OOM prevention monitoring gap (2026-01-06 incident)"

# Push to main
git push origin main
```

---

## TESTING INSTRUCTIONS (For CC)

**Before committing, test locally if possible:**

```bash
# Syntax check all scripts
for script in scripts/monitoring/*.sh; do
    bash -n "$script" && echo "✅ $script" || echo "❌ $script"
done

# Test configure_monitoring.sh (dry run)
# Manually verify prompts and .env generation

# Verify templates have no syntax errors
promtool check config monitoring/prometheus/prometheus.yml.template || echo "⚠️ Fix template syntax"

# Check all files are in git
git status
```

---

## NOTES FOR CC

1. **Root SSH:** Leo explicitly wants root SSH to stay enabled. Print warning but do NOT disable.

2. **Memory focus:** Recent OOM crash motivates this project. Alert rules should prioritize memory monitoring.

3. **SKILL alignment:** Follow patterns from SKILL_11 (service accounts), SKILL_08 (hardware specs), SKILL_13 (logging).

4. **Fail-fast:** Scripts must validate requirements before proceeding. Clear error messages with fixes.

5. **Documentation:** SKILL_14 is main reference. User guide is step-by-step. Runbook is operations manual.

6. **Grafana dashboards:** If time limited, create placeholder structure in monitoring/grafana/dashboards/ with README explaining dashboards can be created via UI.

7. **Alertmanager:** Only configure if SMTP credentials provided in .env (optional feature).

8. **Testing:** Run test_monitoring.sh before declaring success.

---

## QUESTIONS FOR LEO (If Any)

If you encounter blockers or need decisions:

1. **Grafana dashboards:** Create JSON exports or document manual setup?
2. **SMTP details:** Should configure_monitoring.sh validate SMTP connection?
3. **Alert thresholds:** Are memory < 20% (warning) and < 10% (critical) appropriate?
4. **Retention:** 15 days default OK or different?
5. **Additional metrics:** PostgreSQL connection metrics needed in V1?

Document questions in `MONITORING_QUESTIONS.md` for Leo review.

---

**BEGIN IMPLEMENTATION. GOOD LUCK CC! 🚀**

**Timeline:** Aim for Phase 1 complete within 4 hours, full completion within 2-3 days.

**Status updates:** Commit after each phase completion for progress visibility.
