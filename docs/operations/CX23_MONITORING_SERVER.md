# CX23 Monitoring Server - Technical Documentation

**Hostname:** ENIN-Monitoring
**IP:** 77.42.41.135
**Provider:** Hetzner Cloud
**Type:** CX23 (2 vCPU, 4GB RAM, 40GB SSD)
**OS:** Ubuntu 24.04 LTS
**Purpose:** Centralized monitoring for Energy Insights NL

---

## Access

### SSH

```bash
# Primary access (recommended)
ssh monitoring@77.42.41.135

# Emergency access
ssh root@77.42.41.135  # Key-only, password disabled
```

### Web

| Service | URL |
|---------|-----|
| Grafana | https://monitor.synctacles.com |

---

## Installed Services

### Docker Containers

```bash
sudo docker ps
```

| Container | Image | Port | Purpose |
|-----------|-------|------|---------|
| prometheus | prom/prometheus:latest | 9090 | Metrics & alerting |
| grafana | grafana/grafana:latest | 3000 | Dashboards |
| alertmanager | prom/alertmanager:latest | 9093 | Alert routing |
| blackbox | prom/blackbox-exporter:latest | 9115 | HTTP/SSL probes |

### System Services

```bash
systemctl status caddy
systemctl status fail2ban
systemctl status docker
```

| Service | Purpose |
|---------|---------|
| caddy | Reverse proxy with auto-SSL |
| fail2ban | SSH brute-force protection |
| docker | Container runtime |

---

## Directory Structure

```
/opt/monitoring/
├── docker-compose.yml
├── prometheus/
│   ├── prometheus.yml
│   └── alerts.yml
├── alertmanager/
│   └── alertmanager.yml
├── blackbox/
│   └── blackbox.yml
└── grafana/
    ├── datasources/
    │   └── prometheus.yml
    └── dashboards/
        ├── system-overview.json
        ├── services-status.json
        └── api-health.json

/etc/caddy/
└── Caddyfile

/var/lib/caddy/
└── .local/share/caddy/    # SSL certificates

/home/monitoring/
└── .ssh/
    └── authorized_keys    # SSH keys
```

---

## Network Configuration

### Hetzner Firewall Rules

| Direction | Port | Protocol | Source | Action |
|-----------|------|----------|--------|--------|
| Inbound | 22 | TCP | Any | Allow |
| Inbound | 80 | TCP | Any | Allow |
| Inbound | 443 | TCP | Any | Allow |
| Inbound | 3000 | TCP | Any | **Deny** |
| Inbound | 9090 | TCP | Any | **Deny** |
| Inbound | 9093 | TCP | Any | **Deny** |

### Internal Port Bindings

| Port | Service | Binding |
|------|---------|---------|
| 3000 | Grafana | 127.0.0.1:3000 |
| 9090 | Prometheus | 0.0.0.0:9090 (firewall blocked) |
| 9093 | AlertManager | 0.0.0.0:9093 (firewall blocked) |

---

## Security Configuration

### SSH

**File:** `/etc/ssh/sshd_config`

```
PermitRootLogin prohibit-password
PasswordAuthentication no
PubkeyAuthentication yes
```

### fail2ban

**File:** `/etc/fail2ban/jail.local`

```ini
[sshd]
enabled = true
port = ssh
maxretry = 3
bantime = 24h
```

**Commands:**
```bash
# Check banned IPs
sudo fail2ban-client status sshd

# Unban an IP
sudo fail2ban-client set sshd unbanip <IP>
```

### Caddy (HTTPS)

**File:** `/etc/caddy/Caddyfile`

```
monitor.synctacles.com {
    reverse_proxy localhost:3000
}
```

**SSL Certificate:**
- Provider: Let's Encrypt
- Auto-renewal: Yes (30 days before expiry)
- Storage: `/var/lib/caddy/.local/share/caddy/`

### Automatic Updates

**Package:** unattended-upgrades

```bash
# Check config
cat /etc/apt/apt.conf.d/20auto-upgrades

# Should show:
# APT::Periodic::Update-Package-Lists "1";
# APT::Periodic::Unattended-Upgrade "1";
```

---

## User Accounts

### monitoring (primary)

```bash
id monitoring
# uid=108(monitoring) gid=110(monitoring) groups=110(monitoring),998(docker)
```

**SSH Keys:**
- ftso@coston2 (Windows workstation)
- root@ENIN-NL (API server)

### root

SSH key-only access, password disabled.

---

## Maintenance Commands

### Docker

```bash
# Restart all containers
cd /opt/monitoring
sudo docker compose restart

# View resource usage
sudo docker stats --no-stream

# Clean up old images
sudo docker system prune -a

# View container logs
sudo docker logs <container> --tail 100 -f
```

### Caddy

```bash
# Restart
sudo systemctl restart caddy

# Check status
sudo systemctl status caddy

# View logs
sudo journalctl -u caddy -f

# Validate config
sudo caddy validate --config /etc/caddy/Caddyfile

# List certificates
sudo caddy list-certs
```

### System

```bash
# Disk usage
df -h

# Memory usage
free -h

# Running processes
htop

# Check for updates
sudo apt update && sudo apt list --upgradable
```

---

## Monitoring Data

### Retention

- **Prometheus:** 15 days (configured in docker-compose.yml)
- **Grafana:** Persistent volume (grafana-data)
- **AlertManager:** Persistent volume (alertmanager-data)

### Storage Location

```bash
# Docker volumes
sudo docker volume ls

# Inspect volume
sudo docker volume inspect monitoring_prometheus-data
```

---

## Recovery Procedures

### Container Won't Start

```bash
# Check logs
sudo docker logs <container>

# Remove and recreate
cd /opt/monitoring
sudo docker compose down
sudo docker compose up -d
```

### Caddy SSL Error

```bash
# Force certificate renewal
sudo systemctl stop caddy
sudo rm -rf /var/lib/caddy/.local/share/caddy/certificates/
sudo systemctl start caddy
```

### SSH Locked Out

1. Use Hetzner Console (web-based console access)
2. Check fail2ban: `fail2ban-client status sshd`
3. Check sshd: `systemctl status sshd`

### Full Server Recovery

1. Create new CX23 in Hetzner
2. Install Docker:
   ```bash
   curl -fsSL https://get.docker.com | sh
   ```
3. Install Caddy:
   ```bash
   apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
   curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
   curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
   apt update && apt install -y caddy
   ```
4. Copy /opt/monitoring from backup
5. `docker compose up -d`
6. Update DNS A record
7. Caddy will auto-obtain SSL cert

---

## Contact Points

### Alerting

Slack channels:
- #enin-alerts-critical
- #enin-alerts-warnings
- #enin-alerts-info

### Infrastructure

- Hetzner Cloud Console: https://console.hetzner.cloud
- DNS: synctacles.com zone

---

## Changelog

| Date | Change |
|------|--------|
| 2026-01-06 | Initial CX23 provisioning |
| 2026-01-06 | Docker monitoring stack deployed |
| 2026-01-06 | Slack alerting configured |
| 2026-01-07 | SSH key migrated to monitoring user |
| 2026-01-07 | fail2ban installed |
| 2026-01-07 | Root password login disabled |
| 2026-01-07 | Caddy installed, HTTPS enabled |
| 2026-01-07 | Hetzner firewall configured |

---

**Last Updated:** 2026-01-07
**Document Owner:** Leo
