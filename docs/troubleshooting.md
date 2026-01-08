# Energy Insights NL Troubleshooting Guide

**Quick diagnosis and fixes for common issues**

---

## API Issues

### Response Data Not an Array (Parser Error)

**Symptom:** Error parsing API response, expected object but got array

**Cause:** API returns `data` as array `[{...}]`, not object `{...}`

**Example:**
```python
# ❌ WRONG - treats data as object
response = requests.get(f"{BASE_URL}/api/v1/generation-mix", headers=headers)
total_mw = response.json()['data']['total_mw']  # Error: list indices must be integers

# ✅ CORRECT - treats data as array
response = requests.get(f"{BASE_URL}/api/v1/generation-mix", headers=headers)
total_mw = response.json()['data'][0]['total_mw']  # Access first element [0]
```

**API Reference:**
- `/api/v1/generation-mix` → `data` is **array**
- `/api/v1/load` → `data` is **array**

---

### Load Field Name "load_mw" Not "load_actual_mw"

**Symptom:** Field `load_actual_mw` not found in /api/v1/load response

**Cause:** API uses field name `load_mw` (not `load_actual_mw`)

**Example:**
```python
# ❌ WRONG - field doesn't exist
response = requests.get(f"{BASE_URL}/api/v1/load", headers=headers)
load = response.json()['data'][0]['load_actual_mw']  # KeyError: 'load_actual_mw'

# ✅ CORRECT - use correct field name
response = requests.get(f"{BASE_URL}/api/v1/load", headers=headers)
load = response.json()['data'][0]['load_mw']  # ✓ Correct field

# Also available:
load_forecast = response.json()['data'][0]['load_forecast_mw']
load_diff = response.json()['data'][0]['load_difference_mw']
```

**API Reference:**
- Load endpoint: `/api/v1/load`
- Actual load field: `load_mw`
- Forecast load field: `load_forecast_mw`

---

### Old Endpoint "/v1/generation/current" Returns 404

**Symptom:** `curl /v1/generation/current` returns 404 Not Found

**Cause:** Old endpoint path changed to new format

**Migration:**
```bash
# ❌ OLD (deprecated)
curl https://synctacles.io/v1/generation/current

# ✅ NEW (current)
curl https://synctacles.io/api/v1/generation-mix
curl https://synctacles.io/api/v1/load
```

**Updated Endpoints:**
| Old | New |
|-----|-----|
| `/v1/generation/current` | `/api/v1/generation-mix` |
| `/v1/load/current` | `/api/v1/load` |
| `/v1/balance/current` | `/api/v1/balance` (501 Not Implemented) |
| `/v1/prices/today` | `/api/v1/prices` |

---

### 401 Unauthorized

**Symptom:**
```json
{"detail": "Invalid API key"}
```

**Diagnosis:**
```bash
# Test API key (development)
curl -H "X-API-Key: YOUR_KEY" http://localhost:8000/health

# Or (production)
curl -H "X-API-Key: YOUR_KEY" https://your-api-domain/health

# Expected: {"status":"healthy", ...}
# Actual: 401 Unauthorized
```

**Fixes:**

1. **Regenerate API key:**
   ```bash
   curl -X POST -H "X-API-Key: YOUR_CURRENT_KEY" \
     https://synctacles.io/auth/regenerate-key
   ```

2. **Update Home Assistant:**
   - Settings → Integrations → SYNCTACLES → Configure
   - Paste new API key
   - Restart integration

3. **Verify .env file (server):**
   ```bash
   grep API_KEY /opt/synctacles/.env
   # Should NOT show your user API key (different keys)
   ```

---

### 429 Rate Limit Exceeded

**Symptom:**
```json
{"detail": "Rate limit exceeded. Daily limit: 1000 requests."}
```

**Diagnosis:**
```bash
curl -H "X-API-Key: YOUR_KEY" https://synctacles.io/auth/stats

# Check: usage_today vs rate_limit_daily
```

**Fixes:**

1. **Wait for reset (00:00 UTC):**
   ```bash
   # Current time
   date -u

   # Reset in:
   echo "$(( (86400 - $(date +%s) % 86400) / 3600 )) hours"
   ```

2. **Reduce polling frequency (HA):**
   ```yaml
   # configuration.yaml
   synctacles:
     scan_interval: 1800  # 30 min (48 req/day)
   ```

3. **Identify excessive calls:**
   ```bash
   # Check HA logs
   grep "synctacles" /config/home-assistant.log | wc -l
   ```

**Prevention:**
- Default 15 min polling = 96 requests/day (10x margin)
- Avoid manual template sensors that poll continuously

---

### 503 Service Unavailable

**Symptom:** API not responding

**Diagnosis:**
```bash
# Server health
systemctl status synctacles-api

# API logs
journalctl -u synctacles-api -n 50

# Database connection
sudo -u synctacles psql synctacles -c "SELECT 1"
```

**Fixes:**

1. **Restart API service:**
   ```bash
   sudo systemctl restart synctacles-api
   sleep 5
   curl http://localhost:8000/health
   ```

2. **Check database:**
   ```bash
   sudo systemctl status postgresql
   sudo systemctl restart postgresql
   ```

3. **Verify timers:**
   ```bash
   systemctl list-timers synctacles-*
   # Expected: ≥3 active timers
   ```

---

## Home Assistant Integration Issues

### Integration Not Found

**Symptom:** Cannot find SYNCTACLES in Add Integration

**Diagnosis:**
```bash
# SSH into HA
ls /config/custom_components/synctacles/

# Expected files:
# __init__.py, sensor.py, config_flow.py, const.py, manifest.json
```

**Fixes:**

1. **Reinstall via HACS:**
   - HACS → Integrations → ⋮ → Custom repositories
   - Remove SYNCTACLES
   - Re-add: `https://github.com/DATADIO/synctacles-ha`
   - Install → Restart HA

2. **Manual install:**
   ```bash
   cd /config/custom_components
   git clone https://github.com/DATADIO/synctacles-ha.git synctacles
   ha core restart
   ```

3. **Clear cache:**
   ```bash
   rm -rf /config/.storage/core.config_entries
   ha core restart
   ```
   ⚠️ **WARNING:** Removes ALL integrations - reconfigure manually

---

### Sensors Show "Unavailable"

**Symptom:** All 3 sensors unavailable in HA

**Diagnosis:**
```yaml
# Developer Tools → States
sensor.synctacles_generation_total: unavailable
sensor.synctacles_load_actual: unavailable
sensor.synctacles_balance_delta: unavailable
```

**Fixes:**

1. **Check HA logs:**
   ```
   Settings → System → Logs
   Search: "synctacles"
   ```

   **Common errors:**
   - `ConnectionError` → Network issue (check DNS, firewall)
   - `401 Unauthorized` → Invalid API key
   - `Timeout` → API slow (check server load)

2. **Restart integration:**
   ```
   Settings → Devices & Services → SYNCTACLES → ⋮ → Reload
   ```

3. **Verify API endpoint:**
   ```bash
   # From HA machine
   curl -I https://synctacles.io/health
   # Expected: HTTP/2 200
   ```

---

### Sensors Show Old Data

**Symptom:** Timestamp > 1 hour old

**Diagnosis:**
```yaml
# Developer Tools → States
sensor.synctacles_generation_total:
  attributes:
    quality_status: STALE  # or NO_DATA
    data_age_seconds: 3600  # 1 hour+
```

**Fixes:**

1. **Wait for update cycle (15 min):**
   ```bash
   # Check next expected update
   curl https://synctacles.io/api/v1/generation-mix | jq '.meta.next_update_utc'
   ```

2. **Verify server timers:**
   ```bash
   # SSH into server
   systemctl list-timers synctacles-collector

   # Expected: Next trigger < 15 min
   ```

3. **Manual trigger (server):**
   ```bash
   sudo systemctl start synctacles-collector
   sleep 60  # Allow import + normalization
   sudo systemctl start synctacles-importer
   sudo systemctl start synctacles-normalizer
   ```

4. **Check database:**
   ```sql
   -- Latest data timestamp
   SELECT MAX(timestamp), COUNT(*)
   FROM norm_entso_e_a75
   WHERE country = 'NL';
   ```

---

## Server Issues

### Collectors Failing

**Symptom:**
```bash
journalctl -u synctacles-collector -n 20
# Error: 403 Forbidden (ENTSO-E)
```

**Diagnosis:**
```bash
# Test API keys
source /opt/synctacles/.env
curl -H "Accept: application/xml" \
  "https://web-api.tp.entsoe.eu/api?documentType=A75&securityToken=$ENTSOE_API_KEY"
```

**Fixes:**

1. **Verify ENTSO-E key:**
   ```bash
   grep ENTSOE_API_KEY /opt/synctacles/.env
   # Format: UUID (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
   ```

2. **Regenerate ENTSO-E key:**
   - Visit: https://transparency.entsoe.eu/usrm/user/myAccountSettings
   - Security Token → Regenerate
   - Update `/opt/synctacles/.env`
   - Restart collectors: `sudo systemctl restart synctacles-collector`

3. **TenneT key not configured (optional):**
   - TenneT is now BYO-key only (Home Assistant component)
   - Server no longer collects TenneT data
   - No TENNET_API_KEY needed in server .env

---

## TenneT BYO-Key Issues

### Balance Sensors Missing (No TenneT Key)

**Symptom:** No `sensor.energy_insights_nl_balance_delta` or `sensor.energy_insights_nl_grid_stress`

**Cause:** TenneT API key not configured in Home Assistant

**Expected Behavior:**
- Without TenneT key: Only 2 sensors (generation + load) ✓
- With TenneT key: 4 sensors (+ balance + grid stress) ✓

**Fix:**
1. Settings → Devices & Services → Energy Insights NL
2. Click **Configure**
3. Enter your personal TenneT API key (obtained from https://www.tennet.eu/developer-portal/)
4. Submit → Restart integration
5. Wait 1 minute for sensors to appear

**If still missing after 1 minute:**
1. Check HA logs: Settings → System → Logs → Search "energy_insights"
2. Look for TenneT-related errors
3. Verify key format (should be Bearer token)

---

### TenneT "401 Unauthorized" or "Invalid API Key"

**Symptom:** Balance sensors show "unavailable", logs show auth error

**Diagnosis:**
```bash
# SSH into HA and test key directly
curl -H "Authorization: Bearer YOUR_TENNET_KEY" \
  https://api.tennet.eu/v1/balance-delta-high-res
```

**Fixes:**
1. **Regenerate key at TenneT Developer Portal:**
   - Visit: https://www.tennet.eu/developer-portal/
   - Account → API Credentials
   - Regenerate new key
   - Copy completely (no spaces)

2. **Update Home Assistant:**
   - Settings → Devices & Services → Energy Insights NL → Configure
   - Paste new key
   - Submit

3. **Verify key format:**
   - Should be token string (no "Bearer " prefix in HA config)
   - No leading/trailing spaces
   - Should work in curl test above

---

### TenneT "429 Too Many Requests"

**Symptom:** Balance data intermittent, 429 errors in logs, then recovers

**Cause:** TenneT limit is 100 requests/minute; hitting rate limit

**Normal behavior:**
- HA component polls every 5 minutes = 12 requests/hour = safe

**If 429 errors occur:**
1. **Check polling frequency:**
   ```yaml
   # Settings → Devices & Services → Energy Insights NL → Configure
   # Verify TenneT polling interval (default: 5 min, max safe: 1 min)
   ```

2. **Multiple HA instances sharing key?**
   - Create separate TenneT API keys for each HA instance
   - Or increase polling interval to 10+ minutes

3. **Wait for limit reset:**
   - TenneT resets every minute
   - Sensors will auto-recover after 1-2 minutes

---

### TenneT "403 Forbidden"

**Symptom:** Authentication succeeds (401 doesn't appear) but data fetch fails with 403

**Cause:** API key lacks required scope/permissions

**Fix:**
1. Check TenneT Developer Portal that API key has:
   - `balance-delta-high-res` endpoint access
2. If not, regenerate key with full permissions
3. Restart HA integration

---

### Database Corruption

**Symptom:**
```bash
psql -U synctacles -d synctacles -c "SELECT 1"
# Error: relation does not exist
```

**Diagnosis:**
```bash
# Check tables
psql -U synctacles -d synctacles -c "\dt"

# Check migrations
cd /opt/synctacles/app
source /opt/synctacles/venv/bin/activate
alembic current
```

**Fixes:**

1. **Re-run migrations:**
   ```bash
   cd /opt/synctacles/app
   source /opt/synctacles/venv/bin/activate
   export PYTHONPATH=/opt/synctacles/app
   alembic upgrade head
   ```

2. **Restore from backup:**
   ```bash
   # List backups
   ls -lh /opt/synctacles/backups/

   # Restore (example: 20251220_033000.sql.gz)
   gunzip < /opt/synctacles/backups/synctacles_20251220_033000.sql.gz | \
     psql -U synctacles -d synctacles
   ```

3. **Nuclear option (fresh install):**
   ```bash
   # ⚠️ DESTROYS ALL DATA
   sudo -u postgres psql -c "DROP DATABASE synctacles;"
   sudo -u postgres psql -c "CREATE DATABASE synctacles;"
   sudo -u postgres psql -c "GRANT ALL ON DATABASE synctacles TO synctacles;"
   
   cd /opt/synctacles/app
   alembic upgrade head
   
   # Restart collectors to re-fetch data
   sudo systemctl start synctacles-collector
   ```

---

### Nginx SSL Issues

**Symptom:**
```bash
curl https://synctacles.io/health
# Error: SSL certificate problem
```

**Diagnosis:**
```bash
# Check certificate
openssl s_client -connect synctacles.io:443 -servername synctacles.io

# Check Nginx config
nginx -t
```

**Fixes:**

1. **Renew Let's Encrypt:**
   ```bash
   sudo certbot renew
   sudo systemctl reload nginx
   ```

2. **Force renewal:**
   ```bash
   sudo certbot renew --force-renewal
   ```

3. **Check auto-renewal:**
   ```bash
   systemctl status certbot.timer
   # Should be active
   ```

---

## Pipeline Health Issues

### Normalizer Running But Not Processing Data (Silent Failure)

**Symptom:** Pipeline timers show as "active" but A65/A75 normalized data becomes increasingly stale (2+ hours old) while raw data is fresh.

**Diagnosis:**
```bash
# Check gap between raw and normalized data
psql -U energy-insights-nl -d energy_insights_nl <<EOF
SELECT
    'a75_gap' as source,
    EXTRACT(EPOCH FROM (
        (SELECT MAX(timestamp) FROM raw_entso_e_a75 WHERE timestamp <= NOW()) -
        (SELECT MAX(timestamp) FROM norm_entso_e_a75 WHERE timestamp <= NOW())
    ))/60 as gap_minutes;

SELECT
    'a65_gap' as source,
    EXTRACT(EPOCH FROM (
        (SELECT MAX(timestamp) FROM raw_entso_e_a65 WHERE timestamp <= NOW()) -
        (SELECT MAX(timestamp) FROM norm_entso_e_a65 WHERE timestamp <= NOW())
    ))/60 as gap_minutes;
EOF

# Gap >30 minutes indicates normalizer not processing data
```

**Root Cause:** Check if normalizers are missing from `/opt/energy-insights-nl/app/scripts/run_normalizers.sh`

**Fix:**
```bash
# Verify normalizer script includes all sources
cat /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# Should include:
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65  # Load data
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75  # Generation data
# "${PYTHON}" -m synctacles_db.normalizers.normalize_prices

# Manual run to process backlog
sudo -u energy-insights-nl bash -c 'set -a && source /opt/.env && set +a && \
  cd /opt/energy-insights-nl/app && \
  source /opt/energy-insights-nl/venv/bin/activate && \
  /opt/energy-insights-nl/venv/bin/python -m synctacles_db.normalizers.normalize_entso_e_a75'

# Restart API to pick up fresh normalized data
sudo systemctl restart energy-insights-nl-api.service
```

**Prevention:**
- Add logging to normalizer scripts indicating which sources are processed
- Monitor raw vs normalized gap with Prometheus metrics
- Alert if gap exceeds 30 minutes

---

### ENTSO-E Forecast Data Causing Negative Age Values

**Symptom:** API health checks show negative age values (e.g., -1346 minutes) for data freshness

**Example:**
```json
{
  "a65": {"norm_age_min": -1346.7, "status": "FRESH"},
  "a44": {"norm_age_min": -1911.2, "status": "FRESH"}
}
```

**Root Cause:** ENTSO-E API includes forecast data with future timestamps. Queries using `MAX(timestamp)` without filtering select future records.

**Affected Sources:**
- **A65 (Load):** Includes 24-hour forecast (88+ future records)
- **A44 (Prices):** Includes day-ahead prices (tomorrow's data)
- **A75 (Generation):** Minimal forecast data

**Fix Pattern:**
```sql
-- WRONG - Selects future timestamps
SELECT MAX(timestamp) FROM norm_entso_e_a65;

-- CORRECT - Filters to historical only
SELECT MAX(timestamp) FROM norm_entso_e_a65 WHERE timestamp <= NOW();
```

**Rule:** ALL queries selecting `MAX(timestamp)` on ENTSO-E data MUST include `WHERE timestamp <= NOW()`

**Locations to Check:**
- Pipeline health endpoints ([synctacles_db/api/routes/pipeline.py](../synctacles_db/api/routes/pipeline.py))
- Grafana dashboard queries
- Alert rule expressions
- Data export scripts

---

### A75 Generation Data Shows STALE Status

**Symptom:** A75 data consistently shows "STALE" (90-180 min old) when other sources are FRESH

**Expected Behavior:** This is NORMAL for A75 due to ENTSO-E publishing delay

**ENTSO-E A75 Characteristics:**
- **Normal delay:** 2-4 hours after actual generation
- **Update schedule:**
  - 13:01 UTC daily: Large batch (~104 timestamps, last 24h data)
  - 03:54 UTC daily: Smaller update (~20h backfill)

**Alert Thresholds:**
- **FRESH (<90 min):** Ideal but rare for A75
- **STALE (90-180 min):** **NORMAL** - expected ENTSO-E publishing delay
- **UNAVAILABLE (>180 min):** **ABNORMAL** - investigate normalizer

**Only investigate if:**
1. A75 shows UNAVAILABLE (>180 min old)
2. Raw data is fresh but normalized is stale (indicates normalizer issue)
3. Other sources (A44, A65) also affected (indicates broader pipeline problem)

---

## Data Quality Issues

### Quality Status = FALLBACK

**Symptom:**
```json
{
  "meta": {
    "source": "Energy-Charts",
    "quality_status": "FALLBACK"
  }
}
```

**Meaning:** Primary ENTSO-E source failed, using secondary Energy-Charts.

**Fixes:**

1. **Check ENTSO-E status:**
   - Visit: https://transparency.entsoe.eu
   - Check for outages/maintenance

2. **Wait for primary restore:**
   - Fallback auto-switches back when primary available
   - Check `meta.source` in next API response

3. **Verify collector:**
   ```bash
   journalctl -u synctacles-collector -n 50
   # Look for: "Fetching from ENTSO-E"
   ```

**Acceptable:** Fallback accuracy ~90-93% (primary ~95%+)

---

### Quality Status = NO_DATA

**Symptom:**
```json
{
  "data": [],
  "meta": {
    "quality_status": "NO_DATA"
  }
}
```

**Diagnosis:**
```bash
# Check database
psql -U synctacles -d synctacles -c "
SELECT COUNT(*), MAX(timestamp)
FROM norm_entso_e_a75
WHERE country = 'NL';
"
```

**Fixes:**

1. **Trigger full pipeline:**
   ```bash
   sudo systemctl start synctacles-collector
   sleep 120  # Wait for import + normalization
   
   # Verify data
   curl http://localhost:8000/api/v1/generation-mix | jq '.meta.quality_status'
   # Expected: "OK"
   ```

2. **Check logs:**
   ```bash
   journalctl -u synctacles-collector -n 50
   journalctl -u synctacles-importer -n 50
   journalctl -u synctacles-normalizer -n 50
   ```

3. **Database sanity check:**
   ```sql
   -- Should have recent data
   SELECT COUNT(*) FROM norm_entso_e_a75
   WHERE timestamp > NOW() - INTERVAL '1 hour';
   
   -- If 0: pipeline broken
   ```

---

## Performance Issues

### Slow API Responses (> 2s)

**Diagnosis:**
```bash
# Measure response time
time curl http://localhost:8000/api/v1/generation-mix > /dev/null

# Check database load
psql -U synctacles -d synctacles -c "
SELECT pid, query, state
FROM pg_stat_activity
WHERE datname = 'synctacles';
"
```

**Fixes:**

1. **Add database indexes:**
   ```sql
   CREATE INDEX IF NOT EXISTS idx_norm_a75_country_time
   ON norm_entso_e_a75(country, timestamp DESC);
   ```

2. **Vacuum database:**
   ```bash
   psql -U synctacles -d synctacles -c "VACUUM ANALYZE;"
   ```

3. **Check server resources:**
   ```bash
   htop
   # Look for: high CPU, memory usage
   ```

---

## Emergency Procedures

### API Completely Down

**Diagnosis:**
```bash
curl -I https://synctacles.io/health
# No response or 502 Bad Gateway
```

**Nuclear Restart:**
```bash
sudo systemctl restart postgresql
sudo systemctl restart redis-server
sudo systemctl restart synctacles-api
sleep 10
curl http://localhost:8000/health
```

---

### Rollback Failed Deployment

```bash
cd /opt/github/synctacles-repo
sudo ./deployment/rollback.sh

# Select most recent backup (interactive)
# OR specify backup timestamp
sudo ./deployment/rollback.sh 20251220-153045
```

---

## Diagnostic Commands Reference

```bash
# API health
curl http://localhost:8000/health
curl http://localhost:8000/metrics

# Services
systemctl status synctacles-api
systemctl list-timers synctacles-*

# Database
psql -U synctacles -d synctacles -c "SELECT version();"
psql -U synctacles -d synctacles -c "\dt"

# Logs
journalctl -u synctacles-api -f
tail -f /opt/synctacles/logs/api/*.log

# Disk space
df -h /opt/synctacles

# Network
ss -tlnp | grep 8000
```

---

## Contact Support

**When self-diagnosis fails:**

1. **Gather diagnostics:**
   ```bash
   /opt/github/synctacles-repo/scripts/diagnostics.sh > diagnostics.txt
   ```

2. **Email support:**
   - To: support@synctacles.io
   - Attach: diagnostics.txt
   - Include: API key (first 8 chars only)

3. **GitHub issue:**
   - Repository: https://github.com/DATADIO/synctacles-repo/issues
   - Template: Bug report
   - Label: bug, help wanted

**Response time:** < 24 hours (business days)

---

---

## Grafana Dashboard Issues

### Grafana Infinity Plugin Shows "No Data"

**Symptom:** Infinity datasource configured correctly, API endpoint works in browser, but all Grafana panels show "No data"

**Root Cause:** DNS resolution broken on monitor.synctacles.com for external domains

**Diagnosis:**
```bash
# SSH to monitor server
ssh monitoring@monitor.synctacles.com

# Test DNS resolution
curl https://api.synctacles.com/v1/pipeline/health
# Result: "Could not resolve host: api.synctacles.com"
```

**Solution:** Do NOT use Infinity plugin on monitor.synctacles.com. Use Prometheus datasource instead.

**Recommended Approach:**
1. Expose Prometheus metrics endpoint in API ([/v1/pipeline/metrics](../synctacles_db/api/routes/pipeline.py:134-173))
2. Configure Prometheus to scrape with IP address + SNI header
3. Create Grafana dashboard using Prometheus datasource

**Working Prometheus Configuration:**
```yaml
# /opt/monitoring/prometheus/prometheus.yml
- job_name: "pipeline-health"
  scheme: https
  tls_config:
    server_name: enin.xteleo.nl  # SNI header for SSL cert match
  static_configs:
    - targets: ["135.181.255.83:443"]  # Direct IP, no DNS
  metrics_path: /v1/pipeline/metrics
  scrape_interval: 30s
```

**Why Prometheus Works:**
- Uses direct IP address (no DNS lookup required)
- SNI header matches actual SSL certificate domain
- Native HTTP client with proper TLS handling

**Why Infinity Plugin Fails:**
- Relies on host system DNS resolution
- monitor.synctacles.com DNS configuration is broken
- Cannot bypass DNS with IP + SNI header (plugin limitation)

---

**Last Updated:** 2026-01-08
**Status:** Production Ready
**See Also:**
- [Architecture Guide](ARCHITECTURE.md) - System design & data flow
- [API Reference](api-reference.md) - Complete API documentation
- [Monitoring Setup](operations/MONITORING_SETUP.md) - Grafana & Prometheus configuration
- [Pipeline Health Dashboard](https://monitor.synctacles.com/d/5fd1f7f9-e2bb-4a81-a04e-50f9fbbf0ec0/pipeline-health) - Live dashboard
