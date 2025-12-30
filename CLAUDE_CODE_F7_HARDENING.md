# Claude Code - F7 Technical Hardening

**Datum:** 2025-12-30
**Server:** 135.181.255.83 (SSH: ha-energy-insights-nl)
**Workspace:** /opt/github/ha-energy-insights-nl

---

## Context

Je werkt op een Ubuntu 24.04 server via VS Code Remote SSH.
- API draait op `:8000` via systemd service `energy-insights-nl-api`
- Nginx reverse proxy op `:80` (geen SSL - komt later met domein)
- Database: PostgreSQL `energy_insights_nl`
- Git user: `energy-insights-nl` (voor commits)
- Runtime user: `energy-insights-nl`
- Current user in terminal: `root`

**Huidige staat:**
- 6 API endpoints functioneel
- Database indexes aangemaakt
- Nginx + gzip actief
- `cachetools` geïnstalleerd
- Cache module bestaat: `synctacles_db/api/cache.py`

---

## TAAK 1: Integreer caching in API endpoints

### 1.1 Check bestaande cache module

```bash
cat /opt/github/ha-energy-insights-nl/synctacles_db/api/cache.py
```

### 1.2 Update generation endpoint

**Bestand:** `/opt/github/ha-energy-insights-nl/synctacles_db/api/endpoints/generation.py`

Voeg toe bovenaan imports:
```python
from synctacles_db.api.cache import cached, generation_cache
```

Voeg `@cached(generation_cache)` decorator toe boven de endpoint functie.

### 1.3 Update load endpoint

**Bestand:** `/opt/github/ha-energy-insights-nl/synctacles_db/api/endpoints/load.py`

```python
from synctacles_db.api.cache import cached, load_cache
```

### 1.4 Update prices endpoint

**Bestand:** `/opt/github/ha-energy-insights-nl/synctacles_db/api/endpoints/prices.py`

```python
from synctacles_db.api.cache import cached, prices_cache
```

### 1.5 Update balance endpoint

**Bestand:** `/opt/github/ha-energy-insights-nl/synctacles_db/api/endpoints/balance.py`

```python
from synctacles_db.api.cache import cached, balance_cache
```

### 1.6 Update signals endpoint

**Bestand:** `/opt/github/ha-energy-insights-nl/synctacles_db/api/endpoints/signals.py`

```python
from synctacles_db.api.cache import cached, signals_cache
```

### 1.7 Sync naar productie en test

```bash
# Sync cache module naar app
cp /opt/github/ha-energy-insights-nl/synctacles_db/api/cache.py /opt/energy-insights-nl/app/synctacles_db/api/

# Sync endpoints
rsync -av /opt/github/ha-energy-insights-nl/synctacles_db/api/endpoints/ /opt/energy-insights-nl/app/synctacles_db/api/endpoints/

# Restart API
systemctl restart energy-insights-nl-api

# Test caching (2e request moet sneller zijn)
echo "Request 1:"
time curl -s http://localhost:8000/api/v1/generation-mix > /dev/null
echo "Request 2 (should be cached):"
time curl -s http://localhost:8000/api/v1/generation-mix > /dev/null
```

---

## TAAK 2: Maak scripts directory structuur

```bash
mkdir -p /opt/github/ha-energy-insights-nl/scripts/{deploy,maintenance,validation}
```

---

## TAAK 3: Maak deploy script

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/deploy/deploy.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "=== SYNCTACLES Deploy ==="
echo "Started: $(date)"

# Variables
REPO_DIR="/opt/github/ha-energy-insights-nl"
APP_DIR="/opt/energy-insights-nl/app"
BACKUP_BASE="/opt/energy-insights-nl/backups"

# 1. Pre-checks
echo ""
echo "--- Pre-checks ---"
cd "$REPO_DIR"

if [[ -n $(git status --porcelain) ]]; then
    echo "⚠️  Uncommitted changes detected"
    git status --short
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi
echo "✅ Git status OK"

# 2. Backup current
echo ""
echo "--- Creating backup ---"
BACKUP_DIR="$BACKUP_BASE/deploy-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp -r "$APP_DIR" "$BACKUP_DIR/"
echo "✅ Backup created: $BACKUP_DIR"

# 3. Pull latest
echo ""
echo "--- Pulling latest ---"
git pull origin main
echo "✅ Git pulled"

# 4. Sync files
echo ""
echo "--- Syncing files ---"
rsync -av --delete \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    --exclude='.git' \
    "$REPO_DIR/synctacles_db/" "$APP_DIR/synctacles_db/"

rsync -av \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    "$REPO_DIR/config/" "$APP_DIR/config/"

rsync -av \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    "$REPO_DIR/alembic/" "$APP_DIR/alembic/"

echo "✅ Files synced"

# 5. Run migrations (if any)
echo ""
echo "--- Database migrations ---"
cd "$APP_DIR"
source /opt/energy-insights-nl/venv/bin/activate
export PYTHONPATH="$APP_DIR"
if alembic current 2>/dev/null; then
    alembic upgrade head
    echo "✅ Migrations complete"
else
    echo "⚠️  Alembic not configured, skipping migrations"
fi

# 6. Restart services
echo ""
echo "--- Restarting services ---"
systemctl restart energy-insights-nl-api
sleep 3
echo "✅ API restarted"

# 7. Health check
echo ""
echo "--- Health check ---"
if curl -sf http://localhost:8000/health > /dev/null; then
    echo "✅ Health check passed"
    curl -s http://localhost:8000/health | jq .
else
    echo "❌ Health check FAILED!"
    echo "Rolling back..."
    cp -r "$BACKUP_DIR/app/"* "$APP_DIR/"
    systemctl restart energy-insights-nl-api
    exit 1
fi

echo ""
echo "=== Deploy Complete ==="
echo "Finished: $(date)"
```

**Maak executable:**
```bash
chmod +x /opt/github/ha-energy-insights-nl/scripts/deploy/deploy.sh
```

---

## TAAK 4: Maak rollback script

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/deploy/rollback.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "=== SYNCTACLES Rollback ==="

BACKUP_BASE="/opt/energy-insights-nl/backups"
APP_DIR="/opt/energy-insights-nl/app"

# List available backups
echo "Available backups:"
ls -lt "$BACKUP_BASE" | head -10

# Select backup
if [[ -n "${1:-}" ]]; then
    BACKUP_DIR="$BACKUP_BASE/$1"
else
    LATEST=$(ls -t "$BACKUP_BASE" | head -1)
    BACKUP_DIR="$BACKUP_BASE/$LATEST"
    echo ""
    read -p "Rollback to $LATEST? [y/N] " -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi

if [[ ! -d "$BACKUP_DIR/app" ]]; then
    echo "❌ Backup not found: $BACKUP_DIR"
    exit 1
fi

# Restore
echo "Restoring from: $BACKUP_DIR"
rm -rf "$APP_DIR"
cp -r "$BACKUP_DIR/app" "$APP_DIR"

# Restart
systemctl restart energy-insights-nl-api
sleep 3

# Verify
if curl -sf http://localhost:8000/health > /dev/null; then
    echo "✅ Rollback successful"
    curl -s http://localhost:8000/health | jq .
else
    echo "❌ Rollback FAILED - manual intervention required!"
    exit 1
fi
```

**Maak executable:**
```bash
chmod +x /opt/github/ha-energy-insights-nl/scripts/deploy/rollback.sh
```

---

## TAAK 5: Maak backup script

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/maintenance/backup.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "=== SYNCTACLES Database Backup ==="

BACKUP_DIR="/opt/energy-insights-nl/backups/db"
DB_NAME="energy_insights_nl"
RETENTION_DAYS=30

mkdir -p "$BACKUP_DIR"

# Create backup
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_$(date +%Y%m%d_%H%M%S).sql.gz"
sudo -u postgres pg_dump "$DB_NAME" | gzip > "$BACKUP_FILE"

echo "✅ Backup created: $BACKUP_FILE"
echo "   Size: $(du -h "$BACKUP_FILE" | cut -f1)"

# Cleanup old backups
echo ""
echo "Cleaning backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete -print

echo ""
echo "Current backups:"
ls -lh "$BACKUP_DIR"/*.sql.gz 2>/dev/null | tail -5

echo ""
echo "=== Backup Complete ==="
```

**Maak executable:**
```bash
chmod +x /opt/github/ha-energy-insights-nl/scripts/maintenance/backup.sh
```

---

## TAAK 6: Maak restore script

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/maintenance/restore.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "=== SYNCTACLES Database Restore ==="

BACKUP_DIR="/opt/energy-insights-nl/backups/db"
DB_NAME="energy_insights_nl"

# List backups
echo "Available backups:"
ls -lh "$BACKUP_DIR"/*.sql.gz 2>/dev/null | tail -10

if [[ -z "${1:-}" ]]; then
    echo ""
    echo "Usage: $0 <backup_file>"
    echo "Example: $0 ${DB_NAME}_20251230_120000.sql.gz"
    exit 1
fi

BACKUP_FILE="$BACKUP_DIR/$1"
if [[ ! -f "$BACKUP_FILE" ]]; then
    echo "❌ Backup not found: $BACKUP_FILE"
    exit 1
fi

echo ""
echo "⚠️  WARNING: This will DROP and recreate the database!"
read -p "Continue with restore from $1? [y/N] " -n 1 -r
echo
[[ ! $REPLY =~ ^[Yy]$ ]] && exit 1

# Stop API
systemctl stop energy-insights-nl-api

# Restore
echo "Restoring..."
sudo -u postgres dropdb --if-exists "$DB_NAME"
sudo -u postgres createdb "$DB_NAME" -O energy_insights_nl
gunzip -c "$BACKUP_FILE" | sudo -u postgres psql "$DB_NAME"

# Start API
systemctl start energy-insights-nl-api
sleep 3

# Verify
if curl -sf http://localhost:8000/health > /dev/null; then
    echo "✅ Restore successful"
else
    echo "❌ Restore FAILED - check logs!"
    exit 1
fi
```

**Maak executable:**
```bash
chmod +x /opt/github/ha-energy-insights-nl/scripts/maintenance/restore.sh
```

---

## TAAK 7: Maak health-check script

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/maintenance/health-check.sh`

```bash
#!/bin/bash

echo "=== SYNCTACLES Health Check ==="
echo "Time: $(date)"
echo ""

ERRORS=0

# Check API service
echo "--- Services ---"
if systemctl is-active --quiet energy-insights-nl-api; then
    echo "✅ API service: running"
else
    echo "❌ API service: NOT running"
    ((ERRORS++))
fi

# Check timers
TIMER_COUNT=$(systemctl list-timers energy-insights-nl-* --no-pager 2>/dev/null | grep -c energy-insights || echo 0)
if [[ $TIMER_COUNT -ge 3 ]]; then
    echo "✅ Timers active: $TIMER_COUNT"
else
    echo "⚠️  Timers active: $TIMER_COUNT (expected >= 3)"
fi

# Check nginx
if systemctl is-active --quiet nginx; then
    echo "✅ Nginx: running"
else
    echo "⚠️  Nginx: NOT running"
fi

# Check PostgreSQL
if systemctl is-active --quiet postgresql; then
    echo "✅ PostgreSQL: running"
else
    echo "❌ PostgreSQL: NOT running"
    ((ERRORS++))
fi

# Check API endpoints
echo ""
echo "--- API Endpoints ---"
for endpoint in health "api/v1/generation-mix" "api/v1/load" "api/v1/balance" "api/v1/signals"; do
    if curl -sf "http://localhost:8000/$endpoint" > /dev/null 2>&1; then
        echo "✅ /$endpoint"
    else
        echo "❌ /$endpoint"
        ((ERRORS++))
    fi
done

# Check database
echo ""
echo "--- Database ---"
if sudo -u postgres psql -d energy_insights_nl -c "SELECT 1" > /dev/null 2>&1; then
    echo "✅ Database connection OK"
    
    # Row counts
    for table in norm_entso_e_a75 norm_entso_e_a65 norm_tennet_balance; do
        COUNT=$(sudo -u postgres psql -d energy_insights_nl -t -c "SELECT COUNT(*) FROM $table" 2>/dev/null | tr -d ' ')
        echo "   $table: $COUNT rows"
    done
else
    echo "❌ Database connection FAILED"
    ((ERRORS++))
fi

# Check disk space
echo ""
echo "--- Disk Space ---"
DISK_USAGE=$(df -h /opt | tail -1 | awk '{print $5}' | tr -d '%')
if [[ $DISK_USAGE -lt 80 ]]; then
    echo "✅ Disk usage: ${DISK_USAGE}%"
else
    echo "⚠️  Disk usage: ${DISK_USAGE}% (warning: >80%)"
fi

# Summary
echo ""
echo "=== Summary ==="
if [[ $ERRORS -eq 0 ]]; then
    echo "✅ All critical checks passed"
    exit 0
else
    echo "❌ $ERRORS critical error(s) found"
    exit 1
fi
```

**Maak executable:**
```bash
chmod +x /opt/github/ha-energy-insights-nl/scripts/maintenance/health-check.sh
```

---

## TAAK 8: Maak validation script

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/validation/validate_setup.sh`

```bash
#!/bin/bash
set -e

echo "========================================"
echo "  SYNCTACLES Setup Validation"
echo "  $(date)"
echo "========================================"
echo ""

ERRORS=0
WARNINGS=0

check_pass() { echo "✅ $1"; }
check_fail() { echo "❌ $1"; ((ERRORS++)); }
check_warn() { echo "⚠️  $1"; ((WARNINGS++)); }

# 1. Environment
echo "--- Environment ---"
[[ -f /opt/.env ]] && check_pass ".env exists" || check_fail ".env missing"

source /opt/.env 2>/dev/null || true
[[ -n "${BRAND_NAME:-}" ]] && check_pass "BRAND_NAME set: $BRAND_NAME" || check_fail "BRAND_NAME not set"
[[ -n "${DATABASE_URL:-}" ]] && check_pass "DATABASE_URL set" || check_fail "DATABASE_URL not set"

# 2. Services
echo ""
echo "--- Services ---"
systemctl is-active --quiet energy-insights-nl-api && check_pass "API service running" || check_fail "API service not running"
systemctl is-active --quiet postgresql && check_pass "PostgreSQL running" || check_fail "PostgreSQL not running"
systemctl is-active --quiet nginx && check_pass "Nginx running" || check_warn "Nginx not running"

# 3. Timers
echo ""
echo "--- Scheduled Tasks ---"
for timer in collector importer normalizer; do
    if systemctl list-timers | grep -q "energy-insights-nl-${timer}"; then
        check_pass "Timer: $timer"
    else
        check_warn "Timer missing: $timer"
    fi
done

# 4. API Endpoints
echo ""
echo "--- API Endpoints ---"
API_BASE="http://localhost:8000"

for endpoint in /health /api/v1/generation-mix /api/v1/load /api/v1/balance /api/v1/prices /api/v1/signals; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE$endpoint" 2>/dev/null || echo "000")
    if [[ "$STATUS" == "200" ]]; then
        check_pass "$endpoint (HTTP $STATUS)"
    else
        check_fail "$endpoint (HTTP $STATUS)"
    fi
done

# 5. Database
echo ""
echo "--- Database ---"
if sudo -u postgres psql -d energy_insights_nl -c "SELECT 1" > /dev/null 2>&1; then
    check_pass "Database connection"
    
    # Check tables
    for table in norm_entso_e_a75 norm_entso_e_a65 norm_entso_e_a44 norm_tennet_balance; do
        if sudo -u postgres psql -d energy_insights_nl -c "SELECT 1 FROM $table LIMIT 1" > /dev/null 2>&1; then
            check_pass "Table exists: $table"
        else
            check_warn "Table missing or empty: $table"
        fi
    done
else
    check_fail "Database connection failed"
fi

# 6. File System
echo ""
echo "--- File System ---"
[[ -d /opt/energy-insights-nl/app ]] && check_pass "App directory exists" || check_fail "App directory missing"
[[ -d /opt/energy-insights-nl/venv ]] && check_pass "Venv exists" || check_fail "Venv missing"
[[ -d /opt/github/ha-energy-insights-nl ]] && check_pass "Git repo exists" || check_fail "Git repo missing"

# 7. Permissions
echo ""
echo "--- Permissions ---"
if [[ $(stat -c %U /opt/energy-insights-nl/app) == "energy-insights-nl" ]]; then
    check_pass "App owned by service user"
else
    check_warn "App ownership incorrect"
fi

# Summary
echo ""
echo "========================================"
echo "  Validation Complete"
echo "========================================"
echo "  Errors:   $ERRORS"
echo "  Warnings: $WARNINGS"
echo ""

if [[ $ERRORS -eq 0 ]]; then
    echo "✅ Setup is VALID"
    exit 0
else
    echo "❌ Setup has ERRORS - fix before production"
    exit 1
fi
```

**Maak executable:**
```bash
chmod +x /opt/github/ha-energy-insights-nl/scripts/validation/validate_setup.sh
```

---

## TAAK 9: Setup backup cron job

```bash
# Add daily backup at 03:00
(crontab -l 2>/dev/null | grep -v "backup.sh"; echo "0 3 * * * /opt/github/ha-energy-insights-nl/scripts/maintenance/backup.sh >> /var/log/synctacles-backup.log 2>&1") | crontab -

# Verify
crontab -l
```

---

## TAAK 10: Git commit alles

```bash
# Switch to git user for commit
su - energy-insights-nl -c "
cd /opt/github/ha-energy-insights-nl

git add -A

git commit -m 'F7: Technical hardening - caching, deploy/maintenance scripts

- Added in-memory caching for API endpoints (TTL: 1-60 min)
- Created deploy.sh with backup and rollback support
- Created rollback.sh for quick recovery
- Created backup.sh + restore.sh for database
- Created health-check.sh for monitoring
- Created validate_setup.sh for setup verification
- Added daily backup cron job (03:00)

Scripts structure:
  scripts/deploy/      - deploy.sh, rollback.sh
  scripts/maintenance/ - backup.sh, restore.sh, health-check.sh
  scripts/validation/  - validate_setup.sh'

git push origin main
"
```

---

## TAAK 11: Run validation

```bash
# Run full validation
/opt/github/ha-energy-insights-nl/scripts/validation/validate_setup.sh

# Run health check
/opt/github/ha-energy-insights-nl/scripts/maintenance/health-check.sh
```

---

## Exit Criteria

Na uitvoering moeten alle checks groen zijn:
- [ ] Cache werkt (2e request sneller)
- [ ] `./scripts/deploy/deploy.sh` bestaat en is executable
- [ ] `./scripts/maintenance/backup.sh` bestaat en is executable
- [ ] `./scripts/validation/validate_setup.sh` alle checks groen
- [ ] Cron job voor daily backup actief
- [ ] Alles gecommit en gepusht naar GitHub

**Test commando's:**
```bash
# Test cache
time curl -s http://localhost/api/v1/generation-mix > /dev/null
time curl -s http://localhost/api/v1/generation-mix > /dev/null

# Test scripts
/opt/github/ha-energy-insights-nl/scripts/maintenance/health-check.sh
/opt/github/ha-energy-insights-nl/scripts/validation/validate_setup.sh

# Check cron
crontab -l | grep backup
```
