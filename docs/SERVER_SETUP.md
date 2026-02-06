# Server Setup & Disaster Recovery Guide

**Last Updated:** 2026-02-05
**Purpose:** Complete reinstallation procedures for all Synctacles servers

---

## Quick Reference

| Server | IP | OS | Purpose |
|--------|-----|-----|---------|
| ENERGY-DEV | 135.181.255.83 | Ubuntu 22.04 | Development |
| ENERGY-PROD | 46.62.212.227 | Ubuntu 22.04 | Production Energy API |
| CARE-PROD | 173.249.55.109 | Ubuntu 22.04 | Production Care/KB |
| MONITOR | 77.42.41.135 | Ubuntu 22.04 | Prometheus/Grafana |

---

## ENERGY-PROD Server Setup

### 1. Base System

```bash
# Update system
apt update && apt upgrade -y

# Install dependencies
apt install -y \
    python3.12 python3.12-venv python3.12-dev \
    postgresql-16 postgresql-contrib \
    nginx certbot python3-certbot-nginx \
    git curl jq

# Create system user
useradd -r -m -s /bin/bash energy
```

### 2. PostgreSQL Setup

```bash
# Create database and user
sudo -u postgres psql << 'EOF'
CREATE USER energy WITH PASSWORD 'GENERATE_SECURE_PASSWORD';
CREATE DATABASE energy_prod OWNER energy;
GRANT ALL PRIVILEGES ON DATABASE energy_prod TO energy;
\c energy_prod
GRANT ALL ON SCHEMA public TO energy;
EOF
```

### 3. Application Setup

```bash
# Create directories
mkdir -p /opt/energy-prod /opt/github
chown energy:energy /opt/energy-prod /opt/github

# Clone repository (as energy user)
su - energy
cd /opt/github
git clone git@github.com-energy:synctacles/energy.git synctacles-energy

# Setup Python environment
python3.12 -m venv /opt/energy-prod/venv
source /opt/energy-prod/venv/bin/activate
pip install -r /opt/github/synctacles-energy/requirements.txt
pip install -e /opt/github/synctacles-energy
```

### 4. Environment Configuration

Create `/opt/energy-prod/.env`:

```bash
# Database
DATABASE_URL="postgresql://energy:PASSWORD@localhost:5432/energy_prod"
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="energy_prod"
DB_USER="energy"

# Branding
BRAND_NAME="Energy PROD"
BRAND_SLUG="energy-prod"

# API
API_HOST="0.0.0.0"
API_PORT="8001"
LOG_LEVEL="warning"

# External APIs
ENTSOE_API_KEY="YOUR_KEY_HERE"

# Paths
INSTALL_PATH="/opt/energy-prod"
LOG_PATH="/var/log/energy-prod"
GITHUB_ACCOUNT="synctacles"
REPO_NAME="ha-integration"
```

Set permissions:
```bash
chmod 600 /opt/energy-prod/.env
chown energy:energy /opt/energy-prod/.env
```

### 5. Systemd Services

Copy templates and configure:

```bash
# API Service
cat > /etc/systemd/system/energy-prod-api.service << 'EOF'
[Unit]
Description=Energy PROD API
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=energy
Group=energy
WorkingDirectory=/opt/github/synctacles-energy
EnvironmentFile=/opt/energy-prod/.env
Environment="PYTHONPATH=/opt/github/synctacles-energy"
ExecStart=/opt/energy-prod/venv/bin/gunicorn energy_api.api.main:app \
    --workers 8 \
    --worker-class uvicorn.workers.UvicornWorker \
    --bind 127.0.0.1:8001 \
    --access-logfile - \
    --error-logfile -
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Collector Timer
cat > /etc/systemd/system/energy-prod-collector.timer << 'EOF'
[Unit]
Description=Energy PROD Collector Timer

[Timer]
OnCalendar=*:0/15
Persistent=true

[Install]
WantedBy=timers.target
EOF

# Collector Service
cat > /etc/systemd/system/energy-prod-collector.service << 'EOF'
[Unit]
Description=Energy PROD Collector
After=network.target postgresql.service

[Service]
Type=oneshot
User=energy
Group=energy
WorkingDirectory=/opt/github/synctacles-energy
EnvironmentFile=/opt/energy-prod/.env
Environment="PYTHONPATH=/opt/github/synctacles-energy"
ExecStart=/opt/github/synctacles-energy/scripts/run_collectors.sh

[Install]
WantedBy=multi-user.target
EOF
```

Repeat for: `importer`, `normalizer`, `health`, `frank-collector`

Enable services:
```bash
systemctl daemon-reload
systemctl enable --now energy-prod-api
systemctl enable --now energy-prod-collector.timer
systemctl enable --now energy-prod-importer.timer
systemctl enable --now energy-prod-normalizer.timer
systemctl enable --now energy-prod-health.timer
systemctl enable --now energy-prod-frank-collector.timer
```

### 6. Nginx Configuration

```bash
cat > /etc/nginx/sites-available/energy << 'EOF'
upstream api_backend {
    server 127.0.0.1:8001;
    keepalive 32;
}

server {
    listen 80;
    server_name energy.synctacles.com;

    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name energy.synctacles.com;

    ssl_certificate /etc/letsencrypt/live/synctacles.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/synctacles.com/privkey.pem;

    location / {
        proxy_pass http://api_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

ln -s /etc/nginx/sites-available/energy /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

### 7. SSL Certificate

```bash
# Temporarily allow HTTP
ufw allow 80/tcp

# Get certificate
certbot --nginx -d energy.synctacles.com

# Block HTTP again
ufw deny 80/tcp
```

### 8. Firewall

```bash
ufw allow 22/tcp      # SSH
ufw allow 443/tcp     # HTTPS
ufw allow 9100/tcp    # Node exporter (from MONITOR only)
ufw enable
```

### 9. Git Deploy Key (Read-Only)

```bash
# Generate key
ssh-keygen -t ed25519 -C "energy-prod-deploy" -f ~/.ssh/id_energy_deploy -N ""

# Add to GitHub as read-only deploy key
cat ~/.ssh/id_energy_deploy.pub

# Configure SSH
cat >> ~/.ssh/config << 'EOF'
Host github.com-energy
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_energy_deploy
EOF
```

---

## ENERGY-DEV Server Setup

Same as ENERGY-PROD with these differences:

| Setting | DEV Value |
|---------|-----------|
| Database | `energy_dev` |
| User | `energy_dev` |
| Port | 8001 |
| Domain | `dev.synctacles.com` |
| Service prefix | `energy-dev-*` |
| Git access | **Read/Write** (push enabled) |

---

## CARE-PROD Server Setup

### 1. Base System

```bash
apt update && apt upgrade -y

apt install -y \
    python3.12 python3.12-venv python3.12-dev \
    postgresql-16 postgresql-contrib postgresql-16-pgvector \
    nodejs npm \
    git curl jq

# Create system user
useradd -r -m -s /bin/bash brains
```

### 2. PostgreSQL with pgvector

```bash
sudo -u postgres psql << 'EOF'
CREATE USER brains_admin WITH PASSWORD 'GENERATE_SECURE_PASSWORD';
CREATE DATABASE brains_kb OWNER brains_admin;
\c brains_kb
CREATE EXTENSION IF NOT EXISTS vector;
CREATE SCHEMA kb;
GRANT ALL ON SCHEMA kb TO brains_admin;
GRANT ALL ON SCHEMA public TO brains_admin;
ALTER DATABASE brains_kb SET search_path TO kb, public;
EOF
```

### 3. Ollama Installation

```bash
curl -fsSL https://ollama.com/install.sh | sh
systemctl enable --now ollama

# Pull required models
ollama pull phi3:mini
ollama pull nomic-embed-text
```

### 4. Application Setup

```bash
mkdir -p /opt/openclaw/harvesters /opt/openclaw/mcp /opt/openclaw/logs
chown -R brains:brains /opt/openclaw

su - brains
cd /opt/openclaw/harvesters
python3.12 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

### 5. Environment Configuration

Create `/opt/openclaw/harvesters/.env`:

```bash
# Database
DATABASE_URL="postgresql://brains_admin:PASSWORD@localhost:5432/brains_kb"

# Telegram
TELEGRAM_BOT_TOKEN_SUPPORT="YOUR_BOT_TOKEN"
TELEGRAM_GROUP_ID="-1003846489213"

# External APIs
GROQ_API_KEY="YOUR_KEY"
ANTHROPIC_API_KEY="YOUR_KEY"

# GitHub
GITHUB_REPO="home-assistant/core"
```

### 6. Systemd Services

```bash
# Support Bot
cat > /etc/systemd/system/care-prod-support.service << 'EOF'
[Unit]
Description=Care PROD Support Bot
After=network.target postgresql.service ollama.service
Wants=postgresql.service ollama.service

[Service]
Type=simple
User=brains
Group=brains
WorkingDirectory=/opt/openclaw/harvesters
EnvironmentFile=/opt/openclaw/harvesters/.env
ExecStart=/opt/openclaw/harvesters/venv/bin/python support_agent/main.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Harvest Timer (hourly)
cat > /etc/systemd/system/care-prod-harvest.timer << 'EOF'
[Unit]
Description=Care PROD Harvest Timer

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
EOF
```

Enable services:
```bash
systemctl daemon-reload
systemctl enable --now care-prod-support
systemctl enable --now care-prod-harvest.timer
```

### 7. Node Exporter

```bash
wget https://github.com/prometheus/node_exporter/releases/download/v1.7.0/node_exporter-1.7.0.linux-amd64.tar.gz
tar xvf node_exporter-*.tar.gz
mv node_exporter-*/node_exporter /usr/local/bin/

cat > /etc/systemd/system/node_exporter.service << 'EOF'
[Unit]
Description=Node Exporter

[Service]
User=nobody
ExecStart=/usr/local/bin/node_exporter

[Install]
WantedBy=multi-user.target
EOF

systemctl enable --now node_exporter
```

---

## MONITOR Server Setup

### 1. Prometheus

```bash
# Create user
useradd -r -s /bin/false prometheus

# Download and install
wget https://github.com/prometheus/prometheus/releases/download/v2.48.0/prometheus-2.48.0.linux-amd64.tar.gz
tar xvf prometheus-*.tar.gz
mv prometheus-*/prometheus /usr/local/bin/
mv prometheus-*/promtool /usr/local/bin/

mkdir -p /etc/prometheus /var/lib/prometheus
chown prometheus:prometheus /var/lib/prometheus
```

### 2. Prometheus Configuration

Create `/etc/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 30s

scrape_configs:
  - job_name: 'energy-prod-node'
    static_configs:
      - targets: ['46.62.212.227:9100']
        labels:
          environment: 'prod'
          server: 'energy-prod'

  - job_name: 'energy-prod-api'
    scheme: https
    metrics_path: /metrics
    static_configs:
      - targets: ['energy.synctacles.com:443']
        labels:
          environment: 'prod'

  - job_name: 'energy-dev-node'
    static_configs:
      - targets: ['135.181.255.83:9100']
        labels:
          environment: 'dev'

  - job_name: 'care-prod-node'
    static_configs:
      - targets: ['173.249.55.109:9100']
        labels:
          environment: 'prod'
```

### 3. Grafana

```bash
apt install -y apt-transport-https software-properties-common
wget -q -O - https://packages.grafana.com/gpg.key | apt-key add -
add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
apt update && apt install grafana
systemctl enable --now grafana-server
```

---

## Backup Procedures

### Database Backups

```bash
# ENERGY-PROD
pg_dump -U energy energy_prod | gzip > /backup/energy_prod_$(date +%Y%m%d).sql.gz

# CARE-PROD
pg_dump -U brains_admin brains_kb | gzip > /backup/brains_kb_$(date +%Y%m%d).sql.gz
```

### Restore from Backup

```bash
gunzip -c backup_file.sql.gz | psql -U owner database_name
```

---

## Verification Checklist

After setup, verify:

- [ ] API responds: `curl https://energy.synctacles.com/health`
- [ ] Services running: `systemctl status energy-prod-api`
- [ ] Timers active: `systemctl list-timers energy-prod-*`
- [ ] Database accessible: `psql -h localhost -U energy energy_prod`
- [ ] Monitoring working: Check Prometheus targets
- [ ] SSL certificate valid: `curl -I https://energy.synctacles.com`

---

## Emergency Contacts

- **Prometheus Alerts:** Slack #critical-alerts
- **On-call:** Check PagerDuty rotation
