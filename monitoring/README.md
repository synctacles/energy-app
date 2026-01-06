# SYNCTACLES Monitoring Infrastructure

Hybrid monitoring setup using Docker for the central monitoring server and native node_exporter on application servers.

## Quick Start

### Phase 1: Setup Monitoring Server

Copy this entire directory to your CX23 server and run:

```bash
cd /opt/monitoring
docker-compose up -d
```

### Phase 2: Setup Application Servers

On each application server:

```bash
sudo bash /path/to/setup_application_server.sh <monitoring_server_ip>
```

Then add to Prometheus:

```bash
./scripts/monitoring/add_target.sh <app_server_ip> <server_name>
```

## Directory Structure

```
monitoring/
├── README.md                           # This file
├── docker-compose.yml                  # Docker Compose configuration
├── prometheus/
│   ├── prometheus.yml.template        # Prometheus main config
│   ├── alerts.yml                     # Alert rules (OOM-focused)
│   └── recording_rules.yml.template   # Optional: pre-calculated metrics
├── alertmanager/
│   └── alertmanager.yml.template      # Alert routing configuration
└── grafana/
    └── datasources/
        └── prometheus.yml             # Grafana datasource config
```

## Files to Copy to CX23

When setting up on CX23, copy these files to `/opt/monitoring/`:

1. **docker-compose.yml** ← Copy as-is
2. **prometheus/prometheus.yml.template** → Copy as `prometheus.yml`
3. **prometheus/alerts.yml** ← Copy as-is
4. **alertmanager/alertmanager.yml.template** → Copy as `alertmanager.yml`
5. **grafana/datasources/prometheus.yml** ← Create directory structure

### Setup Commands

```bash
# Create directory
mkdir -p /opt/monitoring/grafana/datasources

# Copy files
cp monitoring/docker-compose.yml /opt/monitoring/
cp monitoring/prometheus/prometheus.yml.template /opt/monitoring/prometheus.yml
cp monitoring/prometheus/alerts.yml /opt/monitoring/
cp monitoring/alertmanager/alertmanager.yml.template /opt/monitoring/alertmanager.yml
cp monitoring/grafana/datasources/prometheus.yml /opt/monitoring/grafana/datasources/

# Or all at once from repo
cp -r monitoring/* /opt/monitoring/
```

## Access URLs

After `docker-compose up -d`:

- **Prometheus:** http://<CX23_IP>:9090
- **Grafana:** http://<CX23_IP>:3000 (admin/admin)
- **AlertManager:** http://<CX23_IP>:9093

## Alert Rules

See `prometheus/alerts.yml` for:
- Memory pressure alerts (OOM prevention)
- Disk space alerts
- CPU usage alerts
- Service health alerts

## Documentation

- [SKILL_14_MONITORING_INFRASTRUCTURE.md](../SKILL_14_MONITORING_INFRASTRUCTURE.md) - Architecture
- [MONITORING_SETUP.md](../docs/operations/MONITORING_SETUP.md) - Setup guide
- [PHASE_1_EXECUTION_CHECKLIST.md](../PHASE_1_EXECUTION_CHECKLIST.md) - Step-by-step
