# Energy Insights NL Troubleshooting Guide

**Quick diagnosis and fixes for common issues**

---

## API Issues

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

### Balance Sensors Missing

**Symptom:** No `balance_delta` or `grid_stress` sensors

**Cause:** TenneT API key not configured

**Fix:**
1. Settings → Devices & Services → SYNCTACLES → Configure
2. Add your personal TenneT API key
3. Restart integration

---

### TenneT "Invalid API Key"

**Symptom:** Balance sensors show "unavailable", logs show auth error

**Diagnosis:**
```bash
# Test your key directly
curl -H "Authorization: Bearer YOUR_TENNET_KEY" \
  https://api.tennet.eu/v1/balance-delta-high-res/latest
```

**Fixes:**
1. Regenerate key at TenneT Developer Portal
2. Verify key copied correctly (no extra spaces)
3. Check key hasn't expired

---

### TenneT Rate Limit

**Symptom:** Balance data intermittent, 429 errors in log

**Cause:** TenneT limit is 100 requests/minute

**Fix:**
- HA component polls every 5 minutes (safe margin)
- If multiple HA instances share key: use separate keys

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

**Last Updated:** 2025-12-30
**Status:** Production Ready
**See Also:**
- [Architecture Guide](ARCHITECTURE.md) - System design & data flow
- [API Reference](api-reference.md) - Complete API documentation
- [Deployment Guide](deployment.md) - Installation & operations
