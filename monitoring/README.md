# Monitoring Stack Configuration

**Status:** ✅ Deployed and operational on CX23 (77.42.41.135)
**URL:** https://monitor.synctacles.com

---

## Directory Structure (on CX23)

```
/opt/monitoring/
├── docker-compose.yml          # Container orchestration
├── prometheus/
│   ├── prometheus.yml          # Scrape configuration
│   └── alerts.yml              # Alert rules (22 rules)
├── alertmanager/
│   └── alertmanager.yml        # Slack webhook routing
├── blackbox/
│   └── blackbox.yml            # HTTP/SSL probe config
└── grafana/
    ├── datasources/
    │   └── prometheus.yml      # Prometheus datasource
    └── dashboards/
        ├── system-overview.json
        ├── services-status.json
        └── api-health.json
```

---

## Components

| Service | Port | Purpose |
|---------|------|---------|
| **Prometheus** | 9090 | Metrics collection & alerting |
| **Grafana** | 3000 | Dashboards (via Caddy :443) |
| **AlertManager** | 9093 | Alert routing to Slack |
| **Blackbox** | 9115 | HTTP/SSL endpoint probes |

---

## Quick Commands (on CX23)

```bash
# Start/restart all
cd /opt/monitoring
sudo docker compose up -d

# View status
sudo docker ps

# View logs
sudo docker logs prometheus --tail 50
sudo docker logs grafana --tail 50
sudo docker logs alertmanager --tail 50

# Restart single service
sudo docker compose restart prometheus
```

---

## Configuration Files

### docker-compose.yml

Key settings:
- Prometheus: 15 day retention
- Grafana: bound to localhost (Caddy handles external)
- All containers on `monitoring` network

### prometheus/prometheus.yml

Scrape targets:
- `node-exporter-main` (135.181.255.83:9100) - API server metrics
- `blackbox-http` - HTTP health checks
- `blackbox-ssl` - SSL certificate monitoring

### prometheus/alerts.yml

22 alert rules in groups:
- `memory_alerts` - Memory usage warnings
- `disk_alerts` - Disk space warnings/critical
- `cpu_alerts` - CPU usage
- `load_alerts` - System load
- `service_alerts` - systemd service failures
- `service_health_alerts` - Service health checks
- `endpoint_alerts` - HTTP endpoint monitoring

### alertmanager/alertmanager.yml

Routes alerts by severity:
- `critical` → #enin-alerts-critical
- `warning` → #enin-alerts-warnings
- `info` → #enin-alerts-info

### blackbox/blackbox.yml

Probes:
- HTTP 2xx checker for /health endpoint
- TCP SSL certificate checker

---

## Grafana Dashboards

### System Overview
- API Health status (UP/DOWN)
- Memory usage %
- Disk usage %
- CPU usage %

### Services Status
- systemd service states
- Service uptime
- Recent restarts

### API Health
- HTTP response time
- SSL certificate expiry
- Endpoint availability

---

## Adding New Targets

1. Edit `/opt/monitoring/prometheus/prometheus.yml`
2. Add target under appropriate job:
   ```yaml
   - job_name: 'node'
     static_configs:
       - targets: ['<new-ip>:9100']
         labels:
           instance: '<server-name>'
   ```
3. Restart: `sudo docker compose restart prometheus`

---

## Deployment

This directory is a reference. The actual deployment is on CX23.

To redeploy from scratch:
1. Copy this directory to `/opt/monitoring/` on new server
2. Install Docker
3. `docker compose up -d`
4. Configure Caddy for HTTPS
5. Update DNS

---

## Related Documentation

- [MONITORING_SETUP.md](../docs/operations/MONITORING_SETUP.md) - Complete setup guide
- [SYSTEMD_SERVICES_ANALYSIS.md](../docs/SYSTEMD_SERVICES_ANALYSIS.md) - Service analysis

---

**Last Updated:** 2026-01-07
