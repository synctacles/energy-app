# Claude Code Phase 4 - Architectuur & Documentatie

**Repo:** `/c/Workbench/DEV/ha-energy-insights-nl`
**Branch:** `git checkout -b docs/architecture-documentation`

---

## DOEL

Complete documentatie suite voor:
- Toekomstige Leo (6 maanden)
- Potentiële team members
- Production readiness
- Multi-tenant deployment

---

## STAP 0: Maak docs directory structuur

```bash
mkdir -p docs
```

---

## STAP 1: ARCHITECTURE.md (P2.7 - ADRs)

**Maak** `docs/ARCHITECTURE.md`:

```markdown
# SYNCTACLES Architecture

## Overview

SYNCTACLES is een energy data aggregation platform dat ruwe data van externe bronnen (ENTSO-E, TenneT) verzamelt, normaliseert, en via REST API beschikbaar stelt voor Home Assistant integratie.

## Design Principles

1. **KISS** - Keep It Simple, Stupid
2. **Fail-Fast** - Geen silent failures, expliciete errors
3. **Brand-Free** - Configuratie via templates, geen hardcoded branding
4. **Three-Layer Architecture** - RAW → Normalized → API

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        EXTERNAL SOURCES                          │
├─────────────────┬─────────────────┬─────────────────────────────┤
│    ENTSO-E      │     TenneT      │      Energy-Charts          │
│  (A75/A65/A44)  │   (Balance)     │       (Fallback)            │
└────────┬────────┴────────┬────────┴─────────────┬───────────────┘
         │                 │                      │
         ▼                 ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      LAYER 1: COLLECTORS                         │
│  synctacles_db/collectors/                                       │
│  ├── entso_e_a75_generation.py   (15-min interval)              │
│  ├── entso_e_a65_load.py         (15-min interval)              │
│  ├── entso_e_a44_prices.py       (hourly)                       │
│  ├── tennet_ingestor.py          (rate-limited)                 │
│  └── energy_charts_prices.py     (fallback)                     │
│                                                                  │
│  Output: /var/log/{brand}/collectors/entso_e_raw/*.xml          │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      LAYER 2: IMPORTERS                          │
│  synctacles_db/importers/                                        │
│  ├── import_entso_e_a75.py   → raw_entso_e_a75 table            │
│  ├── import_entso_e_a65.py   → raw_entso_e_a65 table            │
│  ├── import_entso_e_a44.py   → raw_entso_e_a44 table            │
│  └── import_tennet_balance.py → raw_tennet_balance table        │
│                                                                  │
│  Parse XML/CSV → Insert to PostgreSQL RAW tables                 │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     LAYER 3: NORMALIZERS                         │
│  synctacles_db/normalizers/                                      │
│  ├── normalize_entso_e_a75.py  → norm_generation table          │
│  ├── normalize_entso_e_a65.py  → norm_load table                │
│  └── normalize_tennet_balance.py → norm_grid_balance table      │
│                                                                  │
│  RAW tables → Normalized tables with quality metadata            │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        LAYER 4: API                              │
│  synctacles_db/api/                                              │
│  ├── main.py                 (FastAPI app)                       │
│  ├── routes/                 (endpoint definitions)              │
│  └── schemas/                (Pydantic models)                   │
│                                                                  │
│  Endpoints:                                                      │
│  ├── /v1/generation/current  → Current generation mix           │
│  ├── /v1/load/current        → Current grid load                │
│  ├── /v1/prices/today        → Today's electricity prices       │
│  ├── /v1/signals/charge_now  → EV charging recommendation       │
│  └── /health                 → System health status             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data Flow

```
ENTSO-E API                    PostgreSQL                    REST API
    │                              │                            │
    │  XML Response                │                            │
    ▼                              │                            │
┌─────────┐                        │                            │
│Collector│ ─── saves XML ───────► /var/log/.../raw/            │
└─────────┘                        │                            │
    │                              │                            │
    │  trigger                     │                            │
    ▼                              │                            │
┌─────────┐                        │                            │
│Importer │ ─── parses XML ──────► raw_entso_e_* tables         │
└─────────┘                        │                            │
    │                              │                            │
    │  trigger                     │                            │
    ▼                              │                            │
┌──────────┐                       │                            │
│Normalizer│ ─── transforms ─────► norm_* tables                │
└──────────┘                       │                            │
    │                              │                            │
    │                              │  query                     │
    │                              ▼                            │
    │                         ┌─────────┐                       │
    │                         │   API   │ ◄─── HTTP request ────┤
    │                         └─────────┘                       │
    │                              │                            │
    │                              │  JSON response             │
    │                              ▼                            │
    │                         Home Assistant                    │
```

---

## Module Structure

```
synctacles_db/
├── __init__.py
├── models.py                 # SQLAlchemy models (RAW + Normalized)
├── collectors/
│   ├── __init__.py
│   ├── entso_e_a75_generation.py
│   ├── entso_e_a65_load.py
│   ├── entso_e_a44_prices.py
│   ├── tennet_ingestor.py
│   └── energy_charts_prices.py
├── importers/
│   ├── __init__.py
│   ├── import_entso_e_a75.py
│   ├── import_entso_e_a65.py
│   ├── import_entso_e_a44.py
│   └── import_tennet_balance.py
├── normalizers/
│   ├── __init__.py
│   ├── normalize_entso_e_a75.py
│   ├── normalize_entso_e_a65.py
│   └── normalize_tennet_balance.py
├── api/
│   ├── __init__.py
│   ├── main.py
│   ├── routes/
│   └── schemas/
└── fallback/
    └── energy_charts_client.py
```

---

## Database Schema

### RAW Tables (Layer 2 output)

| Table | Source | Key Fields |
|-------|--------|------------|
| `raw_entso_e_a75` | ENTSO-E A75 | timestamp, psr_type, value_mw |
| `raw_entso_e_a65` | ENTSO-E A65 | timestamp, load_mw, forecast_mw |
| `raw_entso_e_a44` | ENTSO-E A44 | timestamp, price_eur_mwh |
| `raw_tennet_balance` | TenneT | timestamp, balance_mw, state |

### Normalized Tables (Layer 3 output)

| Table | Purpose | Quality Fields |
|-------|---------|----------------|
| `norm_generation` | Aggregated generation mix | status, source, age_seconds |
| `norm_load` | Grid load + forecast | status, source, age_seconds |
| `norm_grid_balance` | Grid balance state | status, source, age_seconds |

---

## Architecture Decision Records (ADR)

### ADR-001: Three-Layer Data Pipeline
**Context:** Need reliable data processing with clear separation of concerns.
**Decision:** Separate collectors, importers, and normalizers into distinct modules.
**Consequences:** 
- (+) Each layer can fail independently
- (+) Easy to debug (check RAW data first)
- (+) Can replay/reprocess any layer
- (-) More moving parts to orchestrate

### ADR-002: Fail-Fast Configuration
**Context:** Silent failures caused hours of debugging.
**Decision:** All required config must be present at startup, or fail immediately.
**Consequences:**
- (+) Immediate feedback on misconfiguration
- (+) No "works on my machine" issues
- (-) Requires complete .env before any testing

### ADR-003: Template-Based Deployment
**Context:** Need to deploy same codebase with different branding.
**Decision:** Use `{{PLACEHOLDER}}` templates, generate at install time.
**Consequences:**
- (+) Single codebase, multiple brands
- (+) No git conflicts between deployments
- (-) Extra generation step during install

### ADR-004: XML File Caching
**Context:** ENTSO-E has rate limits; need ability to replay/debug.
**Decision:** Collectors save raw XML to disk before processing.
**Consequences:**
- (+) Can replay imports without API calls
- (+) Debug data issues by inspecting raw files
- (+) Natural backup of source data
- (-) Disk space usage (mitigated by log rotation)

### ADR-005: Quality Metadata
**Context:** Users need to know if data is fresh/stale/missing.
**Decision:** All normalized data includes status (OK/STALE/NO_DATA) and age.
**Consequences:**
- (+) Home Assistant can show data quality
- (+) Automation can use fallback logic
- (-) Slightly more complex API responses

---

## Scheduling

| Service | Interval | Purpose |
|---------|----------|---------|
| collector.timer | 15 min | Fetch new data from sources |
| importer.timer | 15 min | Parse XML into RAW tables |
| normalizer.timer | 15 min | Transform to normalized tables |
| tennet.timer | 5 min | TenneT balance (rate-limited) |
| health.timer | 5 min | System health check |

---

## Technology Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Runtime | Python | 3.12 |
| Web Framework | FastAPI | 0.100+ |
| Database | PostgreSQL + TimescaleDB | 16 |
| ORM | SQLAlchemy | 2.0 |
| Migrations | Alembic | 1.12+ |
| Process Manager | systemd | - |
| OS | Ubuntu | 24.04 LTS |

---

## Security Model (Current)

⚠️ **Development Mode** - Not production ready

| Aspect | Current State | Production Required |
|--------|---------------|---------------------|
| API Auth | None | API key / JWT |
| DB Auth | Local peer auth | Password + SSL |
| Secrets | Plain text .env | Vault / encrypted |
| Network | Open ports | Firewall + reverse proxy |

See `SECRETS_MANAGEMENT.md` for production hardening.
```

---

## STAP 2: DEPLOYMENT.md (P1.6)

**Maak** `docs/DEPLOYMENT.md`:

```markdown
# SYNCTACLES Deployment Guide

## Prerequisites

- Ubuntu 24.04 LTS
- Root access
- GitHub access to repository
- ENTSO-E API key (get from: https://transparency.entsoe.eu/)

---

## Fresh Installation

### 1. Clone Repository

```bash
cd /opt
git clone git@github.com:DATADIO/ha-energy-insights-nl.git github/ha-energy-insights-nl
```

### 2. Run Installer

```bash
cd /opt/github/ha-energy-insights-nl/scripts/setup
chmod +x setup_synctacles_server_v2.3.4.sh
sudo ./setup_synctacles_server_v2.3.4.sh
```

### 3. Configure API Keys

Edit `/opt/.env`:
```bash
export ENTSOE_API_KEY="your-key-here"
```

### 4. Start Services

```bash
systemctl start ${BRAND_SLUG}-collector.timer
systemctl start ${BRAND_SLUG}-importer.timer
systemctl start ${BRAND_SLUG}-normalizer.timer
systemctl start ${BRAND_SLUG}-tennet.timer
systemctl start ${BRAND_SLUG}-api.service
```

### 5. Verify

```bash
# Check timers
systemctl list-timers "${BRAND_SLUG}-*"

# Check API
curl http://localhost:8000/health

# Check database
sudo -u postgres psql -d ${DB_NAME} -c "SELECT COUNT(*) FROM raw_entso_e_a75;"
```

---

## Upgrade Path

### Minor Updates (code changes only)

```bash
# 1. Pull latest code
cd /opt/github/ha-energy-insights-nl
git pull origin main

# 2. Sync to app directory
rsync -av --delete synctacles_db/ /opt/${BRAND_SLUG}/app/synctacles_db/

# 3. Regenerate scripts from templates
cd /opt/${BRAND_SLUG}/app
for template in systemd/scripts/*.template; do
    output="scripts/$(basename ${template%.template})"
    sed -e "s|{{INSTALL_PATH}}|/opt/${BRAND_SLUG}|g" \
        -e "s|{{LOG_PATH}}|/var/log/${BRAND_SLUG}|g" \
        -e "s|{{ENV_FILE}}|/opt/.env|g" \
        "$template" > "$output"
    chmod +x "$output"
done

# 4. Restart services
systemctl restart ${BRAND_SLUG}-api.service
```

### Major Updates (database migrations)

```bash
# 1. Stop services
systemctl stop ${BRAND_SLUG}-*.timer
systemctl stop ${BRAND_SLUG}-api.service

# 2. Backup database
sudo -u postgres pg_dump ${DB_NAME} > backup_$(date +%Y%m%d).sql

# 3. Pull and sync (as above)

# 4. Run migrations
cd /opt/${BRAND_SLUG}/app
source /opt/${BRAND_SLUG}/venv/bin/activate
set -a && source /opt/.env && set +a
alembic upgrade head

# 5. Restart services
systemctl start ${BRAND_SLUG}-*.timer
systemctl start ${BRAND_SLUG}-api.service
```

---

## Rollback

### Code Rollback

```bash
cd /opt/github/ha-energy-insights-nl
git log --oneline -10  # Find commit to rollback to
git checkout <commit-hash>

# Re-sync as per upgrade steps
```

### Database Rollback

```bash
# Stop services first
systemctl stop ${BRAND_SLUG}-*.timer

# Restore from backup
sudo -u postgres psql -c "DROP DATABASE ${DB_NAME};"
sudo -u postgres psql -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};"
sudo -u postgres psql ${DB_NAME} < backup_YYYYMMDD.sql

# Restart
systemctl start ${BRAND_SLUG}-*.timer
```

---

## Directory Structure

```
/opt/
├── .env                          # Environment configuration
├── github/
│   └── ha-energy-insights-nl/    # Git repository (read-only)
└── ${BRAND_SLUG}/
    ├── app/                      # Application code
    │   ├── synctacles_db/        # Python modules
    │   ├── config/               # Configuration
    │   ├── scripts/              # Generated run scripts
    │   └── systemd/              # Templates
    ├── venv/                     # Python virtual environment
    └── logs/                     # Application logs

/var/log/${BRAND_SLUG}/
├── collectors/
│   └── entso_e_raw/              # Raw XML files
├── scheduler/                    # Run script logs
└── api/                          # API logs

/etc/systemd/system/
└── ${BRAND_SLUG}-*.service|timer # Systemd units
```

---

## Verification Checklist

- [ ] All timers running: `systemctl list-timers "${BRAND_SLUG}-*"`
- [ ] API responding: `curl http://localhost:8000/health`
- [ ] Database has data: `psql -c "SELECT COUNT(*) FROM raw_entso_e_a75"`
- [ ] Logs rotating: `ls -la /var/log/${BRAND_SLUG}/`
- [ ] No errors in journal: `journalctl -u "${BRAND_SLUG}-*" --since "1 hour ago"`
```

---

## STAP 3: BACKUP_RESTORE.md (P1.5)

**Maak** `docs/BACKUP_RESTORE.md`:

```markdown
# SYNCTACLES Backup & Restore Procedures

## Backup Strategy

### What to Backup

| Component | Location | Frequency | Retention |
|-----------|----------|-----------|-----------|
| Database | PostgreSQL | Daily | 30 days |
| Configuration | /opt/.env | On change | Versioned |
| Raw XML files | /var/log/*/collectors/ | Weekly | 7 days |

### What NOT to Backup

- Generated scripts (regenerate from templates)
- Python venv (reinstall from requirements.txt)
- Application code (restore from Git)

---

## Database Backup

### Manual Backup

```bash
# Full database dump
sudo -u postgres pg_dump ${DB_NAME} > /backup/synctacles_$(date +%Y%m%d_%H%M%S).sql

# Compressed
sudo -u postgres pg_dump ${DB_NAME} | gzip > /backup/synctacles_$(date +%Y%m%d).sql.gz
```

### Automated Daily Backup

Create `/etc/cron.daily/synctacles-backup`:

```bash
#!/bin/bash
BACKUP_DIR="/backup/synctacles"
DB_NAME="energy_insights_nl"
RETENTION_DAYS=30

mkdir -p "$BACKUP_DIR"

# Create backup
sudo -u postgres pg_dump "$DB_NAME" | gzip > "$BACKUP_DIR/db_$(date +%Y%m%d).sql.gz"

# Remove old backups
find "$BACKUP_DIR" -name "db_*.sql.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup complete: $BACKUP_DIR/db_$(date +%Y%m%d).sql.gz"
```

```bash
chmod +x /etc/cron.daily/synctacles-backup
```

---

## Database Restore

### Full Restore

```bash
# 1. Stop services
systemctl stop ${BRAND_SLUG}-*.timer
systemctl stop ${BRAND_SLUG}-api.service

# 2. Drop and recreate database
sudo -u postgres psql << EOF
DROP DATABASE IF EXISTS ${DB_NAME};
CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};
EOF

# 3. Restore
gunzip -c /backup/synctacles_YYYYMMDD.sql.gz | sudo -u postgres psql ${DB_NAME}

# 4. Verify
sudo -u postgres psql -d ${DB_NAME} -c "SELECT COUNT(*) FROM raw_entso_e_a75;"

# 5. Restart services
systemctl start ${BRAND_SLUG}-*.timer
systemctl start ${BRAND_SLUG}-api.service
```

### Partial Restore (single table)

```bash
# Extract specific table from backup
gunzip -c backup.sql.gz | grep -A 1000 "COPY raw_entso_e_a75" | head -n 1000 > table_data.sql

# Or use pg_restore with custom format backups
```

---

## Configuration Backup

### Manual

```bash
cp /opt/.env /backup/env_$(date +%Y%m%d).backup
```

### Restore

```bash
cp /backup/env_YYYYMMDD.backup /opt/.env
chmod 600 /opt/.env
```

---

## Disaster Recovery

### Complete Server Loss

1. Provision new Ubuntu 24.04 server
2. Clone repository
3. Run installer
4. Restore .env from backup
5. Restore database from backup
6. Verify all services

### Estimated Recovery Time

| Scenario | RTO |
|----------|-----|
| Code issue (rollback) | 5 min |
| Database corruption | 15 min |
| Complete server loss | 1 hour |

---

## Testing Backups

Monthly backup verification:

```bash
# 1. Create test database
sudo -u postgres createdb synctacles_test

# 2. Restore latest backup
gunzip -c /backup/synctacles_latest.sql.gz | sudo -u postgres psql synctacles_test

# 3. Verify row counts match
sudo -u postgres psql -d synctacles_test -c "SELECT COUNT(*) FROM raw_entso_e_a75;"

# 4. Cleanup
sudo -u postgres dropdb synctacles_test
```
```

---

## STAP 4: MULTI_TENANT.md

**Maak** `docs/MULTI_TENANT.md`:

```markdown
# SYNCTACLES Multi-Tenant Deployment

## Overview

SYNCTACLES uses a template-based architecture that allows the same codebase to be deployed with different branding on multiple servers.

---

## Template System

### Placeholders

| Placeholder | Description | Example Value |
|-------------|-------------|---------------|
| `{{BRAND_NAME}}` | Display name | "Energy Insights NL" |
| `{{BRAND_SLUG}}` | URL/file-safe identifier | "energy-insights-nl" |
| `{{INSTALL_PATH}}` | Base installation directory | "/opt/energy-insights-nl" |
| `{{LOG_PATH}}` | Log directory | "/var/log/energy-insights-nl" |
| `{{ENV_FILE}}` | Environment file location | "/opt/.env" |
| `{{SERVICE_USER}}` | System user for services | "energy-insights-nl" |
| `{{DB_NAME}}` | Database name | "energy_insights_nl" |
| `{{DB_USER}}` | Database user | "energy_insights_nl" |
| `{{API_PORT}}` | API listen port | "8000" |

### Template Locations

```
systemd/
├── synctacles-api.service.template
├── synctacles-collector.service.template
├── synctacles-collector.timer.template
├── synctacles-importer.service.template
├── synctacles-importer.timer.template
├── synctacles-normalizer.service.template
├── synctacles-normalizer.timer.template
├── synctacles-tennet.service.template
├── synctacles-tennet.timer.template
├── synctacles-health.service.template
├── synctacles-health.timer.template
└── scripts/
    ├── run_collectors.sh.template
    ├── run_importers.sh.template
    ├── run_normalizers.sh.template
    └── health_check.sh.template
```

---

## Deploying New Brand

### 1. Configure Brand Variables

At the start of installer, set:

```bash
BRAND_NAME="My Energy Service"
BRAND_SLUG="my-energy-service"
BRAND_DOMAIN="myenergy.example.com"
```

### 2. Run Installer

The installer will:
- Create user: `my-energy-service`
- Create directories: `/opt/my-energy-service/`
- Create database: `my_energy_service`
- Generate systemd units: `my-energy-service-*.service`
- Generate run scripts from templates

### 3. Result

```
/opt/my-energy-service/           # Installation
/var/log/my-energy-service/       # Logs
/etc/systemd/system/my-energy-service-*.service  # Services
```

---

## Multiple Brands on Same Server

Possible but not recommended for production. Each brand needs:
- Unique BRAND_SLUG
- Unique API_PORT
- Separate database

```bash
# Brand 1: Port 8000
BRAND_SLUG="energy-nl"
API_PORT="8000"

# Brand 2: Port 8001
BRAND_SLUG="energy-de"
API_PORT="8001"
```

---

## Code Module Naming

Currently, Python modules are named `synctacles_db` which is brand-neutral.
This is acceptable for V1 but could be made fully configurable in future.

### Current State
```python
from synctacles_db.collectors import entso_e_a75_generation
```

### Future State (if needed)
```python
from app.collectors import entso_e_a75_generation  # Generic 'app' module
```

---

## Git Workflow

### Single Repository, Multiple Deployments

```
GitHub: DATADIO/ha-energy-insights-nl (main repo)
    │
    ├── Server NL: /opt/energy-insights-nl/
    ├── Server DE: /opt/energy-insights-de/
    └── Server BE: /opt/energy-insights-be/
```

Each server:
1. Clones same repo
2. Runs installer with different BRAND_* variables
3. Gets brand-specific configuration

### No Git Conflicts

Because:
- Templates use placeholders (not hardcoded values)
- Generated files are in .gitignore
- .env is server-specific (not in git)
```

---

## STAP 5: TROUBLESHOOTING.md (P2.8)

**Maak** `docs/TROUBLESHOOTING.md`:

```markdown
# SYNCTACLES Troubleshooting Playbook

## Quick Diagnostics

```bash
# System overview
systemctl list-timers "${BRAND_SLUG}-*" --no-pager
journalctl -u "${BRAND_SLUG}-*" --since "1 hour ago" --no-pager | tail -50

# Database check
sudo -u postgres psql -d ${DB_NAME} -c "SELECT COUNT(*), MAX(timestamp) FROM raw_entso_e_a75;"

# API check
curl -s http://localhost:8000/health | jq .
```

---

## Common Issues

### 1. "ENTSOE_API_KEY not set"

**Symptom:**
```
ERROR - ENTSOE_API_KEY not set
```

**Cause:** Environment variable not loaded

**Fix:**
```bash
# Check .env has the key
grep ENTSOE_API_KEY /opt/.env

# Ensure export statement
# Should be: export ENTSOE_API_KEY="..."
# Not: ENTSOE_API_KEY="..."

# If missing export, fix:
sed -i 's/^ENTSOE_API_KEY=/export ENTSOE_API_KEY=/' /opt/.env
```

---

### 2. "DATABASE_URL not set" / Database Connection Failed

**Symptom:**
```
FATAL: Required environment variable 'DATABASE_URL' is not set
```

**Cause:** .env missing DATABASE_URL or not sourced properly

**Fix:**
```bash
# Check DATABASE_URL exists
grep DATABASE_URL /opt/.env

# If missing, add:
echo 'export DATABASE_URL="postgresql://${DB_USER}@localhost:5432/${DB_NAME}"' >> /opt/.env

# Test connection
set -a && source /opt/.env && set +a
psql "$DATABASE_URL" -c "SELECT 1"
```

---

### 3. Duplicate Key Violation

**Symptom:**
```
duplicate key value violates unique constraint "raw_entso_e_a75_pkey"
```

**Cause:** Database sequence out of sync

**Fix:**
```bash
sudo -u postgres psql -d ${DB_NAME} -c "
SELECT setval('raw_entso_e_a75_id_seq', (SELECT MAX(id) FROM raw_entso_e_a75));
"
```

---

### 4. Permission Denied on Scripts

**Symptom:**
```
Permission denied: /opt/.../scripts/run_collectors.sh
```

**Cause:** Scripts not executable or wrong owner

**Fix:**
```bash
chmod +x /opt/${BRAND_SLUG}/app/scripts/*.sh
chown -R ${SERVICE_USER}:${SERVICE_USER} /opt/${BRAND_SLUG}/app/scripts/
```

---

### 5. Collector Returns Empty Data

**Symptom:** Collector runs but no data in database

**Diagnosis:**
```bash
# Check raw XML files exist
ls -la /var/log/${BRAND_SLUG}/collectors/entso_e_raw/

# Check file contents
head -50 /var/log/${BRAND_SLUG}/collectors/entso_e_raw/a75_*.xml

# Check importer ran
journalctl -u ${BRAND_SLUG}-importer --since "1 hour ago"
```

**Common causes:**
- ENTSO-E rate limited (wait and retry)
- Invalid API key
- Network issue

---

### 6. API Returns 500 Error

**Symptom:** `/health` or other endpoints return 500

**Diagnosis:**
```bash
# Check API logs
journalctl -u ${BRAND_SLUG}-api --since "10 min ago"

# Check if API is running
systemctl status ${BRAND_SLUG}-api

# Check database connectivity
sudo -u postgres psql -d ${DB_NAME} -c "SELECT 1"
```

---

### 7. Timers Not Running

**Symptom:** Data not updating

**Diagnosis:**
```bash
# Check timer status
systemctl list-timers "${BRAND_SLUG}-*"

# Check if enabled
systemctl is-enabled ${BRAND_SLUG}-collector.timer
```

**Fix:**
```bash
systemctl enable ${BRAND_SLUG}-collector.timer
systemctl start ${BRAND_SLUG}-collector.timer
```

---

### 8. Stale Data (status: STALE)

**Symptom:** API returns `"status": "STALE"`

**Meaning:** Data is older than threshold (typically 30 min)

**Diagnosis:**
```bash
# Check last successful collector run
journalctl -u ${BRAND_SLUG}-collector --since "2 hours ago" | grep -i success

# Check raw data timestamps
sudo -u postgres psql -d ${DB_NAME} -c "
SELECT MAX(timestamp), NOW() - MAX(timestamp) as age 
FROM raw_entso_e_a75;
"
```

---

## Log Locations

| Component | Log Location |
|-----------|-------------|
| Collectors | `/var/log/${BRAND_SLUG}/scheduler/collectors_*.log` |
| Importers | `/var/log/${BRAND_SLUG}/scheduler/importers_*.log` |
| Normalizers | `/var/log/${BRAND_SLUG}/scheduler/normalizers_*.log` |
| API | `journalctl -u ${BRAND_SLUG}-api` |
| Systemd | `journalctl -u ${BRAND_SLUG}-*` |

---

## Getting Help

1. Check this playbook
2. Check `journalctl` logs
3. Check raw XML files in collector output
4. Database direct query for data state
5. Create GitHub issue with:
   - Error message
   - Relevant log snippets
   - Steps to reproduce
```

---

## STAP 6: DEVELOPER_ONBOARDING.md (P2.9)

**Maak** `docs/DEVELOPER_ONBOARDING.md`:

```markdown
# SYNCTACLES Developer Onboarding

## Welcome

This guide gets you from zero to running SYNCTACLES locally in 30 minutes.

---

## Prerequisites

- Python 3.12+
- PostgreSQL 16+ with TimescaleDB
- Git
- ENTSO-E API key

---

## Local Development Setup

### 1. Clone Repository

```bash
git clone git@github.com:DATADIO/ha-energy-insights-nl.git
cd ha-energy-insights-nl
```

### 2. Create Virtual Environment

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

### 3. Create Local Database

```bash
sudo -u postgres psql << EOF
CREATE USER synctacles_dev WITH PASSWORD 'dev_password';
CREATE DATABASE synctacles_dev OWNER synctacles_dev;
\c synctacles_dev
CREATE EXTENSION IF NOT EXISTS timescaledb;
EOF
```

### 4. Create .env File

```bash
cat > .env << EOF
export BRAND_NAME="Synctacles Dev"
export BRAND_SLUG="synctacles-dev"
export INSTALL_PATH="$(pwd)"
export LOG_PATH="./logs"
export COLLECTOR_RAW_PATH="./logs/collectors"

export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="synctacles_dev"
export DB_USER="synctacles_dev"
export DATABASE_URL="postgresql://synctacles_dev:dev_password@localhost:5432/synctacles_dev"

export ENTSOE_API_KEY="your-key-here"
export API_PORT="8000"
EOF
```

### 5. Run Migrations

```bash
set -a && source .env && set +a
alembic upgrade head
```

### 6. Test Collector

```bash
mkdir -p logs/collectors/entso_e_raw
python -m synctacles_db.collectors.entso_e_a75_generation
```

### 7. Run API

```bash
uvicorn synctacles_db.api.main:app --reload --port 8000
```

---

## Project Structure

```
ha-energy-insights-nl/
├── synctacles_db/          # Main Python package
│   ├── collectors/         # Data fetchers
│   ├── importers/          # XML → Database
│   ├── normalizers/        # RAW → Normalized
│   └── api/                # FastAPI app
├── config/                 # Configuration
├── alembic/                # Database migrations
├── systemd/                # Service templates
├── scripts/                # Utility scripts
├── docs/                   # Documentation
└── tests/                  # Test suite
```

---

## Development Workflow

### Adding New Feature

1. Create feature branch: `git checkout -b feature/my-feature`
2. Make changes
3. Test locally
4. Commit: `git commit -m "feat: description"`
5. Push: `git push origin feature/my-feature`
6. Create PR

### Running Tests

```bash
pytest tests/
```

### Code Style

```bash
# Format
black synctacles_db/
isort synctacles_db/

# Lint
flake8 synctacles_db/
mypy synctacles_db/
```

---

## Key Concepts

### Fail-Fast Configuration

All required config must be present at startup:

```python
from config.settings import require_env

DATABASE_URL = require_env("DATABASE_URL")  # Fails if missing
```

### Three-Layer Pipeline

1. **Collector** → Fetches XML, saves to disk
2. **Importer** → Parses XML, inserts to RAW table
3. **Normalizer** → Transforms RAW → Normalized

### Quality Metadata

All normalized data includes:
- `status`: OK / STALE / NO_DATA
- `source`: Where data came from
- `age_seconds`: How old the data is

---

## Useful Commands

```bash
# Run single collector
python -m synctacles_db.collectors.entso_e_a75_generation

# Run single importer
python -m synctacles_db.importers.import_entso_e_a75

# Check database
psql $DATABASE_URL -c "SELECT COUNT(*) FROM raw_entso_e_a75"

# Create migration
alembic revision --autogenerate -m "Add new table"

# API docs
open http://localhost:8000/docs
```

---

## Getting Help

- Architecture: `docs/ARCHITECTURE.md`
- Troubleshooting: `docs/TROUBLESHOOTING.md`
- Ask in team chat
```

---

## STAP 7: MONITORING.md (P2.10)

**Maak** `docs/MONITORING.md`:

```markdown
# SYNCTACLES Monitoring Setup

## Overview

Monitoring stack for SYNCTACLES production deployments.

---

## Quick Health Check

```bash
# One-liner status
curl -s http://localhost:8000/health | jq -r '.status'
```

---

## Health Endpoint

`GET /health`

```json
{
  "status": "healthy",
  "components": {
    "database": "ok",
    "collectors": {
      "a75_generation": {"status": "ok", "last_run": "2025-12-29T23:45:00Z"},
      "a65_load": {"status": "ok", "last_run": "2025-12-29T23:45:00Z"},
      "a44_prices": {"status": "ok", "last_run": "2025-12-29T23:00:00Z"}
    },
    "data_freshness": {
      "generation": {"age_seconds": 300, "status": "ok"},
      "load": {"age_seconds": 300, "status": "ok"},
      "prices": {"age_seconds": 3600, "status": "ok"}
    }
  }
}
```

---

## Prometheus Metrics (Future)

Planned metrics endpoint: `GET /metrics`

```
# Data freshness
synctacles_data_age_seconds{type="generation"} 300
synctacles_data_age_seconds{type="load"} 300
synctacles_data_age_seconds{type="prices"} 3600

# Collector success rate
synctacles_collector_success_total{collector="a75"} 1440
synctacles_collector_failure_total{collector="a75"} 2

# API request latency
synctacles_api_request_duration_seconds{endpoint="/v1/generation"} 0.05
```

---

## Alerting Rules (Prometheus/Alertmanager)

```yaml
groups:
  - name: synctacles
    rules:
      - alert: DataStale
        expr: synctacles_data_age_seconds > 1800
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Data is stale"
          
      - alert: CollectorFailing
        expr: increase(synctacles_collector_failure_total[1h]) > 3
        labels:
          severity: critical
        annotations:
          summary: "Collector failing repeatedly"
          
      - alert: APIDown
        expr: up{job="synctacles-api"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "API is down"
```

---

## Grafana Dashboard

### Key Panels

1. **Data Freshness** - Gauge showing age of each data type
2. **Collector Success Rate** - 24h rolling success percentage
3. **API Latency** - P50/P95/P99 response times
4. **Database Size** - Table row counts over time
5. **Error Rate** - Errors per hour

### Dashboard JSON

See `monitoring/grafana-dashboard.json` (to be created)

---

## Log Monitoring

### Loki/Promtail Setup

```yaml
# promtail config
scrape_configs:
  - job_name: synctacles
    static_configs:
      - targets:
          - localhost
        labels:
          job: synctacles
          __path__: /var/log/${BRAND_SLUG}/**/*.log
```

### Useful Log Queries

```
# Errors in last hour
{job="synctacles"} |= "ERROR"

# Collector runs
{job="synctacles"} |= "Collector" |= "complete"

# Slow API requests
{job="synctacles"} |= "request" | duration > 1s
```

---

## Uptime Monitoring

### External Checks (Uptime Robot, Pingdom, etc.)

- **Endpoint:** `https://your-domain.com/health`
- **Method:** GET
- **Expected:** HTTP 200, body contains `"status": "healthy"`
- **Interval:** 5 minutes
- **Alert:** If down > 2 checks

---

## Current State

| Feature | Status |
|---------|--------|
| /health endpoint | ✅ Basic |
| Component status | ⏳ Planned |
| Prometheus metrics | ⏳ Planned |
| Grafana dashboard | ⏳ Planned |
| Log aggregation | ⏳ Planned |
| Alerting | ⏳ Planned |
```

---

## STAP 8: SECRETS_MANAGEMENT.md (P1.4)

**Maak** `docs/SECRETS_MANAGEMENT.md`:

```markdown
# SYNCTACLES Secrets Management

## Current State (Development)

⚠️ **Not Production Ready**

Secrets are currently stored in plain text in `/opt/.env`:

```bash
export ENTSOE_API_KEY="abc123..."
export DATABASE_URL="postgresql://..."
```

This is acceptable for development but must be hardened for production.

---

## Secrets Inventory

| Secret | Current Location | Risk Level |
|--------|------------------|------------|
| ENTSOE_API_KEY | /opt/.env | Medium |
| DATABASE_URL | /opt/.env | High |
| ADMIN_API_KEY | /opt/.env | High |
| DB password | PostgreSQL peer auth | Low |

---

## Production Recommendations

### Option 1: Encrypted .env (Simple)

```bash
# Encrypt
gpg --symmetric --cipher-algo AES256 /opt/.env
# Creates /opt/.env.gpg

# Decrypt at runtime (in systemd service)
ExecStartPre=/bin/bash -c 'gpg --decrypt /opt/.env.gpg > /run/synctacles/.env'
```

### Option 2: HashiCorp Vault (Enterprise)

```bash
# Store secret
vault kv put secret/synctacles entsoe_api_key="abc123"

# Retrieve in application
vault kv get -field=entsoe_api_key secret/synctacles
```

### Option 3: systemd Credentials (Linux)

```ini
[Service]
LoadCredential=entsoe_api_key:/etc/synctacles/secrets/entsoe_api_key
Environment=ENTSOE_API_KEY=%d/entsoe_api_key
```

---

## Database Security

### Current: Peer Authentication

PostgreSQL uses peer auth (no password, local Unix socket only).

### Production: Password + SSL

```bash
# 1. Set password
sudo -u postgres psql -c "ALTER USER ${DB_USER} PASSWORD 'strong_password';"

# 2. Update pg_hba.conf
# Change: local all all peer
# To: local all all scram-sha-256

# 3. Update DATABASE_URL
export DATABASE_URL="postgresql://${DB_USER}:password@localhost:5432/${DB_NAME}?sslmode=require"
```

---

## API Authentication

### Current: None

API is open to localhost.

### Production Options

1. **API Key Header**
   ```bash
   curl -H "X-API-Key: secret" http://api/v1/...
   ```

2. **JWT Tokens**
   ```bash
   curl -H "Authorization: Bearer eyJ..." http://api/v1/...
   ```

3. **Reverse Proxy Auth**
   - Nginx/Caddy handles auth
   - API trusts authenticated requests

---

## File Permissions

```bash
# .env should be readable only by root and service user
chmod 600 /opt/.env
chown root:${SERVICE_USER} /opt/.env

# Or even stricter
chmod 400 /opt/.env
chown root:root /opt/.env
# Use systemd LoadCredential for service access
```

---

## Checklist for Production

- [ ] Database uses password auth (not peer)
- [ ] DATABASE_URL uses SSL
- [ ] .env file permissions are 600 or stricter
- [ ] API requires authentication
- [ ] Secrets are encrypted at rest
- [ ] Secrets are not in git history
- [ ] Backup encryption keys stored separately
- [ ] Regular secret rotation schedule
```

---

## STAP 9: ENV_REFERENCE.md

**Maak** `docs/ENV_REFERENCE.md`:

```markdown
# SYNCTACLES Environment Variables Reference

## Required Variables

These MUST be set or the application will fail to start.

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | Full PostgreSQL connection string | `postgresql://user@localhost:5432/dbname` |
| `DB_HOST` | Database hostname | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_NAME` | Database name | `energy_insights_nl` |
| `DB_USER` | Database user | `energy_insights_nl` |
| `INSTALL_PATH` | Base installation directory | `/opt/energy-insights-nl` |
| `LOG_PATH` | Log directory | `/var/log/energy-insights-nl` |
| `COLLECTOR_RAW_PATH` | Raw collector data | `/var/log/.../collectors` |

---

## Optional Variables

These have sensible defaults but can be customized.

| Variable | Description | Default |
|----------|-------------|---------|
| `ENTSOE_API_KEY` | ENTSO-E Transparency API key | (empty) |
| `ADMIN_API_KEY` | Admin API authentication | (empty) |
| `API_PORT` | API listen port | `8000` |
| `API_HOST` | API listen address | `0.0.0.0` |
| `BRAND_NAME` | Display name | `Synctacles` |
| `BRAND_SLUG` | URL-safe identifier | `synctacles` |

---

## Brand Variables

Used for multi-tenant deployments.

| Variable | Description | Example |
|----------|-------------|---------|
| `BRAND_NAME` | Human-readable name | `Energy Insights NL` |
| `BRAND_SLUG` | File/URL safe name | `energy-insights-nl` |
| `BRAND_DOMAIN` | Public domain | `energy.example.com` |
| `SERVICE_USER` | System user | `energy-insights-nl` |

---

## Template Variables

Used in systemd/script templates (not in .env).

| Placeholder | Replaced By |
|-------------|-------------|
| `{{INSTALL_PATH}}` | INSTALL_PATH value |
| `{{LOG_PATH}}` | LOG_PATH value |
| `{{ENV_FILE}}` | `/opt/.env` |
| `{{BRAND_NAME}}` | BRAND_NAME value |
| `{{BRAND_SLUG}}` | BRAND_SLUG value |
| `{{SERVICE_USER}}` | SERVICE_USER value |
| `{{API_PORT}}` | API_PORT value |

---

## Example .env File

```bash
# =============================================================================
# Energy Insights NL Environment Configuration
# Generated: 2025-12-29T23:00:00+00:00
# =============================================================================

# Brand Configuration
export BRAND_NAME="Energy Insights NL"
export BRAND_SLUG="energy-insights-nl"

# Installation Paths
export INSTALL_PATH="/opt/energy-insights-nl"
export LOG_PATH="/var/log/energy-insights-nl"
export COLLECTOR_RAW_PATH="/var/log/energy-insights-nl/collectors"

# Database Configuration
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="energy_insights_nl"
export DB_USER="energy_insights_nl"
export DATABASE_URL="postgresql://energy_insights_nl@localhost:5432/energy_insights_nl"

# API Keys
export ENTSOE_API_KEY="your-api-key-here"
export ADMIN_API_KEY=""

# Service Configuration
export SERVICE_USER="energy-insights-nl"
export API_PORT="8000"
```

---

## Validation

The application validates required variables at startup:

```python
# config/settings.py
DATABASE_URL = require_env("DATABASE_URL", "PostgreSQL connection string")
```

If a required variable is missing, you'll see:

```
FATAL: Required environment variable 'DATABASE_URL' is not set.
  Description: PostgreSQL connection string
  Ensure /opt/.env is sourced with 'set -a && source /opt/.env && set +a'
```
```

---

## STAP 10: DOCUMENTATION_STATUS.md (Overzicht ontbrekende docs)

**Maak** `docs/DOCUMENTATION_STATUS.md`:

```markdown
# SYNCTACLES Documentation Status

## Overview

Last updated: 2025-12-30

---

## Completed Documentation

| Document | Priority | Status | Description |
|----------|----------|--------|-------------|
| ARCHITECTURE.md | P2.7 | ✅ Complete | System overview, ADRs, data flow |
| DEPLOYMENT.md | P1.6 | ✅ Complete | Install, upgrade, rollback |
| BACKUP_RESTORE.md | P1.5 | ✅ Complete | Backup procedures |
| MULTI_TENANT.md | - | ✅ Complete | Brand-free deployment |
| TROUBLESHOOTING.md | P2.8 | ✅ Complete | Common issues playbook |
| DEVELOPER_ONBOARDING.md | P2.9 | ✅ Complete | Getting started guide |
| MONITORING.md | P2.10 | ✅ Complete | Monitoring setup |
| SECRETS_MANAGEMENT.md | P1.4 | ✅ Complete | Production security |
| ENV_REFERENCE.md | - | ✅ Complete | Environment variables |

---

## Pending Documentation

| Document | Priority | Status | Description |
|----------|----------|--------|-------------|
| API_REFERENCE.md | P1 | ⏳ Pending | Full API endpoint documentation |
| DATA_DICTIONARY.md | P2 | ⏳ Pending | Database schema details |
| RUNBOOK.md | P1 | ⏳ Pending | Operational procedures |
| CHANGELOG.md | P2 | ⏳ Pending | Version history |
| CONTRIBUTING.md | P3 | ⏳ Pending | Contribution guidelines |
| TESTING.md | P2 | ⏳ Pending | Test strategy and procedures |
| PERFORMANCE.md | P3 | ⏳ Pending | Performance tuning guide |
| HOME_ASSISTANT.md | P1 | ⏳ Pending | HA integration guide |

---

## Documentation Debt

### High Priority (before production)

1. **API_REFERENCE.md** - Users need to know endpoint contracts
2. **RUNBOOK.md** - Operators need step-by-step procedures
3. **HOME_ASSISTANT.md** - Primary use case documentation

### Medium Priority (before team expansion)

4. **DATA_DICTIONARY.md** - Understanding data model
5. **TESTING.md** - How to run and write tests
6. **CHANGELOG.md** - Track changes between versions

### Low Priority (nice to have)

7. **CONTRIBUTING.md** - Open source readiness
8. **PERFORMANCE.md** - Optimization guide

---

## Auto-Generated Documentation

| Document | Source | Status |
|----------|--------|--------|
| API Docs (Swagger) | FastAPI /docs | ✅ Auto-generated |
| API Docs (ReDoc) | FastAPI /redoc | ✅ Auto-generated |
| Code Docstrings | Python files | 🟡 Partial |

---

## Documentation Improvements Needed

### ARCHITECTURE.md
- [ ] Add sequence diagrams for data flow
- [ ] Add component interaction diagram
- [ ] Document error handling strategy

### DEPLOYMENT.md
- [ ] Add Ansible/Terraform automation
- [ ] Add container deployment option
- [ ] Add load balancer configuration

### TROUBLESHOOTING.md
- [ ] Add more edge cases
- [ ] Add performance troubleshooting
- [ ] Add network debugging

### DEVELOPER_ONBOARDING.md
- [ ] Add Windows setup instructions
- [ ] Add Docker-based dev environment
- [ ] Add IDE configuration

---

## Next Steps

1. Complete pending P1 documentation
2. Add diagrams to ARCHITECTURE.md
3. Create API_REFERENCE.md from OpenAPI spec
4. Write HOME_ASSISTANT.md integration guide

---

## Review Schedule

| Document | Review Frequency | Last Review |
|----------|------------------|-------------|
| ARCHITECTURE.md | On major changes | 2025-12-30 |
| DEPLOYMENT.md | Quarterly | 2025-12-30 |
| TROUBLESHOOTING.md | Monthly | 2025-12-30 |
| ENV_REFERENCE.md | On config changes | 2025-12-30 |
```

---

## VERIFICATIE

```bash
# Check all docs exist
ls -la docs/

# Word count
wc -l docs/*.md

# Check for broken internal links (basic)
grep -r "\[.*\](.*\.md)" docs/ | grep -v "http"
```

---

## COMMIT

```bash
git add -A
git commit -m "DOCS: Complete documentation suite (Phase 4)

Added comprehensive documentation:

P1 - Production Critical:
- DEPLOYMENT.md - Install, upgrade, rollback procedures
- BACKUP_RESTORE.md - Database backup/restore
- SECRETS_MANAGEMENT.md - Production security guide

P2 - Quality of Life:
- ARCHITECTURE.md - System overview, ADRs, data flow
- TROUBLESHOOTING.md - Common issues playbook
- DEVELOPER_ONBOARDING.md - Getting started guide
- MONITORING.md - Monitoring setup guide

Reference:
- MULTI_TENANT.md - Brand-free deployment
- ENV_REFERENCE.md - All environment variables
- DOCUMENTATION_STATUS.md - Doc inventory and gaps

Total: 10 documentation files covering architecture,
deployment, operations, and development.
"
```

---

## ROLLBACK

```bash
git checkout main
git branch -D docs/architecture-documentation
```
