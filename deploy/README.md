# Energy API Go - Deployment Guide

## Quick Deployment (DEV server)

```bash
# On DEV server (we are here)
cd /opt/github/energy-go
sudo ./deploy/install.sh
```

This will:
- Build the Go binary
- Create log directory `/var/log/energy-api/`
- Install systemd service
- Start the service on port 8002

## Manual Steps

### 1. Build Binary

```bash
cd /opt/github/energy-go
go build -o energy-api ./cmd/energy-api/
```

### 2. Create Log Directory

```bash
sudo mkdir -p /var/log/energy-api
sudo chown synctacles_dev:synctacles_dev /var/log/energy-api
```

### 3. Install Service

```bash
sudo cp deploy/systemd/energy-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable energy-api
sudo systemctl start energy-api
```

### 4. Verify

```bash
# Check service status
systemctl status energy-api

# Check logs
journalctl -u energy-api -f
tail -f /var/log/energy-api/energy-api.log

# Test endpoints
curl http://localhost:8002/health
curl -H "X-API-Key: sk_free_..." http://localhost:8002/api/v1/prices?zone=NL
```

## Configuration

Environment variables (set in `/opt/github/energy-go/.env` or systemd service):

```bash
PORT=8002
DATABASE_URL=postgres://synctacles_dev@localhost:5432/energy_dev?sslmode=disable
AUTH_SERVICE_URL=http://localhost:8000
LOG_LEVEL=info              # debug, info, warn, error
LOG_FILE=/var/log/energy-api/energy-api.log
```

## Service Management

```bash
# Start
sudo systemctl start energy-api

# Stop
sudo systemctl stop energy-api

# Restart
sudo systemctl restart energy-api

# Status
systemctl status energy-api

# Logs
journalctl -u energy-api -f
tail -f /var/log/energy-api/energy-api.log

# Enable on boot
sudo systemctl enable energy-api

# Disable
sudo systemctl disable energy-api
```

## Parallel Deployment (with Python API)

The Go API runs on **port 8002** alongside the Python API on **port 8001**.

### Testing Both APIs

```bash
# Python API (port 8001)
curl http://localhost:8001/api/v1/prices?zone=NL

# Go API (port 8002)
curl http://localhost:8002/api/v1/prices?zone=NL
```

### Performance Comparison

```bash
# Benchmark Python API
ab -n 1000 -c 10 -H "X-API-Key: sk_free_..." http://localhost:8001/api/v1/prices?zone=NL

# Benchmark Go API
ab -n 1000 -c 10 -H "X-API-Key: sk_free_..." http://localhost:8002/api/v1/prices?zone=NL
```

## Monitoring

### Health Check

```bash
curl http://localhost:8002/health
```

Expected response:
```json
{
  "status": "ok",
  "version": "2.0.0",
  "service": "energy-api",
  "time": "2026-02-13T01:00:00Z"
}
```

### Prometheus Metrics

```bash
curl http://localhost:8002/metrics
```

Expected response:
```
# HELP energy_api_info Energy API information
# TYPE energy_api_info gauge
energy_api_info{version="2.0.0",service="energy-api"} 1

# HELP energy_api_up Service up status
# TYPE energy_api_up gauge
energy_api_up 1
```

## Troubleshooting

### Service won't start

```bash
# Check logs
journalctl -u energy-api -n 50

# Check permissions
ls -la /opt/github/energy-go/energy-api
ls -la /var/log/energy-api/

# Verify database connection
sudo -u synctacles_dev psql -d energy_dev -c "SELECT COUNT(*) FROM prices;"
```

### Port already in use

```bash
# Check what's on port 8002
sudo ss -tlnp | grep :8002

# Kill old process
sudo systemctl stop energy-api
pkill -f energy-api
```

### Database permission errors

```bash
# Grant permissions
sudo -u postgres psql -d energy_dev -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO synctacles_dev;"
```

## Deployment to PROD

**TODO**: Separate deployment guide for ENERGY-PROD server.

Key differences:
- Use production database (`energy_prod`)
- Use production auth service URL
- Set LOG_LEVEL=info (not debug)
- Configure firewall rules
- Set up monitoring alerts

## Rollback

If issues arise, rollback to Python API:

```bash
# Stop Go API
sudo systemctl stop energy-api

# Verify Python API is still running
curl http://localhost:8001/health
```

The Python API should continue running unaffected on port 8001.

## Next Steps

After successful DEV deployment:

1. Run parallel for 24-48 hours
2. Compare metrics (response time, error rate, memory)
3. A/B test (50% traffic to each)
4. If stable, deprecate Python API
5. Repeat for PROD server
