# Energy API Go - Production Deployment Guide

## Target Server: ENERGY-PROD (energy.synctacles.com)

**Status:** READY TO DEPLOY ✅

**Benchmark Results:**
- 2.13x faster response time (34.6ms → 16.2ms)
- 97% less memory (490 MB → 16 MB)
- 2.13x higher throughput (28.9 → 61.7 req/sec)

---

## Pre-Deployment Checklist

### 1. Verify Services on ENERGY-PROD

```bash
# Check from DEV (we are here)
ssh cc-hub "ssh energy-prod 'systemctl status auth-service postgresql'"
```

Expected:
- ✅ Auth service running on port 8000
- ✅ PostgreSQL running with `energy_prod` database

### 2. Verify Database Permissions

```bash
ssh cc-hub "ssh energy-prod 'sudo -u postgres psql -d energy_prod -c \"SELECT COUNT(*) FROM prices;\"'"
```

Expected: Price data exists

### 3. Check Current Python API

```bash
ssh cc-hub "ssh energy-prod 'systemctl status energy-prod-api'"
```

This will be running on port 8001 (will run parallel with Go on 8002)

---

## Deployment Steps

### Step 1: Transfer Files to ENERGY-PROD

```bash
# From DEV server (where we are)
cd /opt/github/energy-go

# Create tarball of Go code + deployment scripts
tar czf energy-go-prod.tar.gz \
  cmd/ \
  internal/ \
  deploy/ \
  go.mod \
  go.sum

# Transfer to ENERGY-PROD via cc-hub
scp energy-go-prod.tar.gz cc-hub:/tmp/
ssh cc-hub "scp /tmp/energy-go-prod.tar.gz energy-prod:/tmp/"
```

### Step 2: Setup on ENERGY-PROD

```bash
# SSH to ENERGY-PROD
ssh cc-hub "ssh energy-prod"

# Create directory and extract
sudo mkdir -p /opt/github/energy-go
sudo chown synctacles-dev:synctacles-dev /opt/github/energy-go
cd /opt/github/energy-go
tar xzf /tmp/energy-go-prod.tar.gz

# Create production .env
cat > .env << 'EOF'
PORT=8002
DATABASE_URL=postgres://synctacles_dev@localhost:5432/energy_prod?sslmode=disable
AUTH_SERVICE_URL=http://localhost:8000
LOG_LEVEL=info
LOG_FILE=/var/log/energy-api/energy-api.log
EOF

chmod 600 .env
```

### Step 3: Grant Database Permissions

```bash
# On ENERGY-PROD
sudo -u postgres psql -d energy_prod << 'EOF'
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO synctacles_dev;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO synctacles_dev;
EOF
```

### Step 4: Deploy

```bash
# On ENERGY-PROD
cd /opt/github/energy-go
sudo ./deploy/install.sh
```

This will:
- Build the Go binary
- Create log directory
- Install systemd service
- Start the service on port 8002

### Step 5: Verify Deployment

```bash
# On ENERGY-PROD
systemctl status energy-api

# Test health endpoint
curl http://localhost:8002/health

# Test with API key (use production key)
curl -H "X-API-Key: YOUR_PROD_KEY" http://localhost:8002/api/v1/prices?zone=NL
```

---

## Configuration Differences: DEV vs PROD

| Config | DEV | PROD |
|--------|-----|------|
| **Port** | 8002 | 8002 |
| **Database** | energy_dev | energy_prod |
| **DB User** | synctacles_dev | synctacles_dev |
| **Auth URL** | http://localhost:8000 | http://localhost:8000 |
| **Log Level** | debug | info |
| **Domain** | dev.synctacles.com | energy.synctacles.com |

---

## Parallel Deployment Strategy

The Go API will run on **port 8002** alongside Python on **port 8001**:

```
┌─────────────────────────────────────┐
│  ENERGY-PROD Server                 │
│  ├─ Python API → :8001 (existing)  │
│  └─ Go API     → :8002 (new)       │
└─────────────────────────────────────┘
```

### Testing Both APIs

```bash
# Python API (port 8001) - existing
curl http://energy.synctacles.com:8001/api/v1/prices?zone=NL

# Go API (port 8002) - new
curl http://energy.synctacles.com:8002/api/v1/prices?zone=NL
```

---

## Cutover Plan

### Phase 1: Parallel Running (Week 1)
- Both APIs active
- Monitor metrics
- Compare error rates, response times

### Phase 2: Traffic Split (Week 2)
Option 1: Update clients to use port 8002
Option 2: Nginx reverse proxy with weighted routing

### Phase 3: Full Cutover (Week 3)
- Redirect all traffic to Go API (:8002)
- Keep Python API as backup
- Monitor for issues

### Phase 4: Deprecation (Week 4)
- Stop Python API
- Update documentation
- Celebrate! 🎉

---

## Rollback Plan

If issues arise:

```bash
# Stop Go API
sudo systemctl stop energy-api

# Verify Python API still running
systemctl status energy-prod-api
curl http://localhost:8001/health
```

Python API continues unaffected on port 8001.

---

## Monitoring

### Service Status

```bash
systemctl status energy-api
journalctl -u energy-api -f
tail -f /var/log/energy-api/energy-api.log
```

### Health Checks

```bash
# Local
curl http://localhost:8002/health

# Public (if firewall allows)
curl http://energy.synctacles.com:8002/health
```

### Prometheus Metrics

```bash
curl http://localhost:8002/metrics
```

---

## Firewall Configuration

If you want to expose the Go API publicly:

```bash
# On ENERGY-PROD
sudo ufw allow 8002/tcp comment 'Energy API Go'
sudo ufw status
```

**Note:** Consider using Nginx reverse proxy instead of direct port exposure.

---

## Performance Monitoring

Compare metrics after 24 hours:

```bash
# Memory usage
ps aux | grep -E "(uvicorn|energy-api)" | grep -v grep

# Request counts (from auth service logs)
sudo -u postgres psql -d auth_prod -c "
  SELECT product_id, SUM(used_today) as total_requests
  FROM rate_limits
  GROUP BY product_id;
"
```

---

## Troubleshooting

### Service won't start

```bash
# Check logs
journalctl -u energy-api -n 50

# Verify database connection
sudo -u postgres psql -d energy_prod -c "SELECT COUNT(*) FROM prices;"
```

### Port conflict

```bash
# Check what's using port 8002
sudo ss -tlnp | grep :8002
```

### Permission errors

```bash
# Grant database permissions
sudo -u postgres psql -d energy_prod -c "
  GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO synctacles_dev;
"
```

---

## Success Criteria

After deployment, verify:
- ✅ Service active and stable
- ✅ Health endpoint responds
- ✅ Auth integration works
- ✅ Database queries succeed
- ✅ Memory usage < 20 MB
- ✅ Response time < 30ms
- ✅ No errors in logs

---

## Next Steps After PROD Deployment

1. Update HA addon to use port 8002 (or keep using port 8001 until cutover)
2. Monitor for 7 days
3. Compare metrics vs Python
4. Plan full cutover
5. Deprecate Python API
