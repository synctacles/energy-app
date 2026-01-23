# SKILL 11 â€” REPO STRUCTURE & SERVICE ACCOUNTS

Repository Organisation and Account Management
Version: 1.3 (2026-01-22)

---

## PURPOSE

Define the GitHub repository structure, service account conventions, and git workflow rules. This ensures consistent deployments across all environments and prevents permission issues.

---

## REPOSITORY STRUCTURE

### Active Repositories

| Repository | Purpose | Contains |
|------------|---------|----------|
| `synctacles/backend` | Backend API server | Python API, collectors, importers, normalizers, systemd units |
| `synctacles/ha-integration` | Home Assistant integration | HA custom component only (HACS compatible) |

### Archived Repositories

| Repository | Status | Reason |
|------------|--------|--------|
| `synctacles/synctacles-ha` | ARCHIVED | Replaced by `ha-integration` |
| `synctacles/synctacles-repo` | ARCHIVED | Replaced by `backend` |

### Repository Rules

1. **synctacles-api** = ALL backend code (brand-free)
2. **ha-energy-insights-nl** = ONLY HA component code
3. Backend repo NEVER contains HA component in production
4. HA repo NEVER contains backend code

---

## BEIDE REPO'S OP SERVER

| Repo | Server path | Doel |
|------|-------------|------|
| synctacles-api | `/opt/github/synctacles-api` | Backend API development |
| ha-energy-insights-nl | `/opt/github/ha-energy-insights-nl` | HA component development |

**Beide owned by service account:**
```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/ha-energy-insights-nl
```

---

## SERVICE ACCOUNTS

### Server User Convention (2026-01-23)

Different users per environment for safety - prevents accidental PROD edits.

| Server | User | Hostname | SSH Alias (HUB) |
|--------|------|----------|-----------------|
| PROD | `synctacles` | SYNCTACLES | `synct-prod` |
| DEV | `synctacles-dev` | synctacles-dev | `synct-dev` |

**Current Status:**
- PROD: `synctacles` ✅ Active
- DEV: `energy-insights-nl` (legacy, migrate to `synctacles-dev` later)

### PROD Configuration

| Setting | Value |
|---------|-------|
| Service account | `synctacles` |
| Service group | `synctacles` |
| Install path | `/opt/synctacles` |
| Repo path | `/opt/github/synctacles-api` |

### DEV Configuration (Current - Legacy)

| Setting | Value |
|---------|-------|
| Service account | `energy-insights-nl` |
| Service group | `energy-insights-nl` |
| Install path | `/opt/energy-insights-nl` |
| Repo paths | `/opt/github/synctacles-api` + `/opt/github/ha-energy-insights-nl` |

### Why Different Users Per Environment?

- **Safety:** Cannot accidentally edit PROD thinking you're on DEV
- **Prompt clarity:** `synctacles@SYNCTACLES:~$` vs `synctacles-dev@server:~$`
- **Brand = .env only:** Same code, different config per environment
- **Audit trail:** Clear ownership of all operations

---

## GIT WORKFLOW RULES

### CRITICAL: No Root Git Operations

```bash
# NEVER DO THIS
sudo git pull
sudo git status
root@server: git clone ...

# ALWAYS DO THIS
sudo -u energy-insights-nl git pull
sudo -u energy-insights-nl git status
sudo -u energy-insights-nl git clone ...

# Future (after migration):
sudo -u synctacles git pull
```

### Why No Root?

1. **File permissions:** Root-created files break service user access
2. **Security:** Principle of least privilege
3. **Audit trail:** Clear who did what
4. **Consistency:** Same user owns code and runs service

### Common Git Operations

```bash
# Current account: energy-insights-nl
SERVICE_USER="energy-insights-nl"

# Pull latest code (backend)
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api pull

# Pull latest code (HA component)
sudo -u $SERVICE_USER git -C /opt/github/ha-energy-insights-nl pull

# Check status
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api status

# View log
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api log --oneline -5
```

### GitHub CLI (gh) - Issues, PRs, Releases

**Verschil git vs gh:**
| Tool | Doel | Authenticatie |
|------|------|---------------|
| `git` | Code push/pull/commit | SSH key |
| `gh` | Issues, PRs, releases, repo management | Personal Access Token (PAT) |

**gh CLI is geconfigureerd voor service user:**
- Auth storage: `/home/energy-insights-nl/.config/gh/hosts.yml`
- Permissions: `600` (alleen user kan lezen)

**Gebruik:**
```bash
# Issues
sudo -u energy-insights-nl gh issue list
sudo -u energy-insights-nl gh issue close 21
sudo -u energy-insights-nl gh issue create --title "Bug" --body "Description"

# Pull Requests
sudo -u energy-insights-nl gh pr list
sudo -u energy-insights-nl gh pr create --title "Feature" --body "Description"

# Releases
sudo -u energy-insights-nl gh release list
```

**Bij "not authenticated" of "authentication required" errors:**

1. Vraag Leo om PAT (Personal Access Token)
2. Configureer met:
   ```bash
   sudo -u energy-insights-nl gh auth login --with-token <<< "ghp_xxxx"
   ```
3. Verificatie:
   ```bash
   sudo -u energy-insights-nl gh auth status
   ```

**⚠️ NOOIT:**
- PAT in git committen
- PAT in logs tonen
- PAT in `.env` zetten (gebruik gh native storage)

---

## DIRECTORY STRUCTURE

### Current Server Layout

```
/opt/
├── .env                              # Master config (brand settings)
├── github/
│   ├── synctacles-api/               # Backend repo (owned by energy-insights-nl)
│   │   ├── synctacles_db/            # Backend Python code
│   │   ├── config/                   # Configuration files
│   │   ├── alembic/                  # Database migrations
│   │   ├── systemd/                  # Service templates
│   │   ├── scripts/                  # Setup/deployment scripts
│   │   └── docs/                     # Documentation
│   └── ha-energy-insights-nl/        # HA repo (owned by energy-insights-nl)
│       └── custom_components/
│           └── ha_energy_insights_nl/
└── energy-insights-nl/               # Runtime deployment
    ├── app/                          # Deployed code (copy from repo)
    │   ├── synctacles_db/            # Synced from repo
    │   └── config/                   # Synced from repo
    ├── venv/                         # Python virtual environment
    └── logs -> /var/log/energy-insights-nl/

/var/log/energy-insights-nl/          # Log files
/etc/systemd/system/                  # Generated service units
```

---

## BACKEND DEPLOYMENT (Server â†’ Running App)

### KRITIEK: Sync BEIDE directories

```bash
SERVICE_USER="energy-insights-nl"
REPO_PATH="/opt/github/synctacles-api"
APP_PATH="/opt/energy-insights-nl/app"

# 1. Sync synctacles_db/
sudo rsync -av --delete \
    --exclude='.git' \
    --exclude='__pycache__' \
    --exclude='.env' \
    --exclude='venv' \
    $REPO_PATH/synctacles_db/ $APP_PATH/synctacles_db/

# 2. Sync config/ (NIET VERGETEN!)
sudo rsync -av \
    $REPO_PATH/config/ $APP_PATH/config/

# 3. Fix ownership
sudo chown -R $SERVICE_USER:$SERVICE_USER $APP_PATH/

# 4. Restart
sudo systemctl restart energy-insights-nl-api

# 5. Validate
curl http://localhost:8000/health
```

### Validation After Deploy

```bash
# Check service status
sudo systemctl status energy-insights-nl-api

# Check API health
curl http://localhost:8000/health

# Check logs
sudo journalctl -u energy-insights-nl-api -n 20
```

---

## HA COMPONENT DEPLOYMENT

### Repository: ha-energy-insights-nl

**Server path:** `/opt/github/ha-energy-insights-nl`

Contents:
```
ha-energy-insights-nl/
├── custom_components/
│   └── ha_energy_insights_nl/
│       ├── __init__.py
│       ├── config_flow.py
│       ├── sensor.py
│       ├── diagnostics.py
│       ├── const.py
│       ├── manifest.json
│       ├── strings.json
├── hacs.json
└── README.md
```

### Development Workflow (CC â†’ Leo â†’ HA)

**CC heeft GEEN directe toegang tot HA OS.**

Workflow:
1. **CC** edits files in `/opt/github/ha-energy-insights-nl/`
2. **CC** commit + push naar GitHub
3. **Leo** upload handmatig naar HA (via Samba/SFTP/File Editor)
4. **Leo** restart HA integration of core
5. **Leo** deelt logs/screenshots met CC voor debugging

**CC kan NIET:**
- SSH naar HA
- HA logs direct lezen
- HA restart triggeren

**CC kan WEL:**
- Code schrijven/aanpassen in repo
- Syntax valideren
- Debugging suggesties geven op basis van gedeelde logs

### User Installation (Productie)

Users install via HACS:
1. Add custom repository: `synctacles/ha-integration`
2. Install integration
3. Configure API URL + optional Enever API key

---

## BRAND CONFIGURATION

### How Branding Works

**Same code, different .env:**

```bash
# Production (SYNCTACLES)
BRAND_NAME="SYNCTACLES"
BRAND_SLUG="synctacles"
BRAND_DOMAIN="synctacles.com"

# Development
BRAND_NAME="SYNCTACLES [DEV]"
BRAND_SLUG="synctacles"
BRAND_DOMAIN="dev.synctacles.com"
```

### What Changes Per Brand

| Changes | Doesn't Change |
|---------|----------------|
| .env values | Python code |
| Domain/URLs | API logic |
| Display names | Database schema |
| Systemd unit names | Git repository |

---

## CC (CLAUDE CODE) NETWERK & PERMISSIES

### Netwerk Toegang

**CC draait op SYNCTACLES DEV server (NIET in sandbox).**

CC heeft WEL:
- Internet toegang
- Git push/pull naar GitHub (via SSH)
- API calls naar externe services

**NIET zeggen:** "Je moet zelf pushen want ik heb geen internet"
**WEL doen:** Direct pushen na commit

### CC GIT REGEL (KRITIEK)

**ELKE git operatie MOET met service user:**

```bash
# CORRECT - SSH key bestaat voor energy-insights-nl user
sudo -u energy-insights-nl git -C /opt/github/synctacles-api status
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "message"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main

sudo -u energy-insights-nl git -C /opt/github/ha-energy-insights-nl push origin main

# FOUT - root heeft GEEN SSH key, GEEN GitHub toegang
git push
sudo git push
cd /opt/github/synctacles-api && git push
```

**SSH key bestaat ALLEEN voor `energy-insights-nl` user.**
**Root heeft GEEN GitHub toegang.**

Als CC error ziet `Permission denied (publickey)` â†’ verkeerde user context gebruikt.

### Automatic Context Detection

CC moet zelf bepalen welke user context nodig is per operatie:

| Operatie | User | Command Prefix |
|----------|------|----------------|
| Git (status, pull, commit, push) | service user | `sudo -u energy-insights-nl` |
| File edits in repo | root (dan fix) | `sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/...` |
| systemctl (restart, status) | root | `sudo` |
| apt install | root | `sudo` |
| /etc/ configuratie | root | `sudo` |
| alembic migrations | service user | `sudo -u energy-insights-nl` |
| Python/pip in venv | service user | `sudo -u energy-insights-nl` |

### Required Workflow

**Na ELKE file edit in `/opt/github/synctacles-api/`:**
```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
```

**Na ELKE file edit in `/opt/github/ha-energy-insights-nl/`:**
```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/ha-energy-insights-nl/
```

**Voor ALLE git operaties:**
```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api <command>
sudo -u energy-insights-nl git -C /opt/github/ha-energy-insights-nl <command>
```

**NOOIT:**
```bash
# FOUT - veroorzaakt ownership issues
git status
git pull
git commit
```

### Complete Backend Deployment Voorbeeld

```bash
# 1. Pull latest (service user)
sudo -u energy-insights-nl git -C /opt/github/synctacles-api pull

# 2. Edit files (root is OK)
nano /opt/github/synctacles-api/synctacles_db/api/main.py

# 3. Fix ownership (VERPLICHT na edits)
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

# 4. Git commit (service user)
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "fix: update"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push

# 5. Sync to running app
sudo rsync -av --delete \
    --exclude='.git' --exclude='__pycache__' --exclude='.env' --exclude='venv' \
    /opt/github/synctacles-api/synctacles_db/ /opt/energy-insights-nl/app/synctacles_db/
sudo rsync -av /opt/github/synctacles-api/config/ /opt/energy-insights-nl/app/config/
sudo chown -R energy-insights-nl:energy-insights-nl /opt/energy-insights-nl/app/

# 6. Restart service (root)
sudo systemctl restart energy-insights-nl-api
```

### Foutmelding Herkenning

Als CC ziet:
```
fatal: detected dubious ownership in repository
```

**Fix:**
```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
```

**NIET doen:** `git config --global --add safe.directory` (maskeert probleem)

---

## DATABASE ACCOUNTS

### PostgreSQL User Per Environment

**For SYNCTACLES (Current):**

```sql
-- User for running services/scripts
CREATE USER energy_insights_nl WITH PASSWORD 'secret';

-- Permissions
GRANT CONNECT ON DATABASE energy_insights_nl TO energy_insights_nl;
GRANT USAGE ON SCHEMA public TO energy_insights_nl;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO energy_insights_nl;
```

**Service accounts MUST NOT be:**
- `synctacles@localhost` (old, deprecated)
- `root@localhost` (security risk)
- Any root-level user (violates least privilege)

**Each environment has:**
```
Development:  user = energy_insights_nl_dev
Staging:      user = energy_insights_nl_stg
Production:   user = energy_insights_nl (or synctacles - after migration)
```

### DATABASE_URL Format

**Standardized PostgreSQL URL:**

```python
# config/settings.py
DATABASE_URL = "postgresql://energy_insights_nl@localhost:5432/energy_insights_nl"

# Or from environment:
DATABASE_URL = os.getenv("DATABASE_URL")
# Must be in format: postgresql://user:password@host:port/dbname
# OR: postgresql://user@host:port/dbname (password in .env via connection params)
```

### Critical: All Code Uses Same User

| Module | User | Via |
|--------|------|-----|
| normalize_entso_e_a44.py | energy_insights_nl | config.settings.DATABASE_URL |
| normalize_prices.py | energy_insights_nl | config.settings.DATABASE_URL |
| entso_e_a44_prices.py | energy_insights_nl | config.settings.DATABASE_URL |
| API (main.py) | energy_insights_nl | config.settings.DATABASE_URL |

**NOT different per module** (this would cause synchronization bugs).

---

## DEPLOYMENT PROCEDURES

### Pre-Deployment Validation

**ALWAYS verify these before deploying:**

```bash
# 1. All normalizers import from config.settings
grep -r "from config.settings import DATABASE_URL" /opt/github/synctacles-api/synctacles_db/normalizers/
# Should return 4 matches (normalize_entso_e_a44.py, normalize_prices.py, normalize_entso_e_a65.py, normalize_entso_e_a75.py)

# 2. All normalizers call validate_db_connection()
grep -r "validate_db_connection()" /opt/github/synctacles-api/synctacles_db/normalizers/
# Should return 4 matches

# 3. No hardcoded credentials in code
grep -r "synctacles@" /opt/github/synctacles-api/ --exclude-dir=.git
# Should return 0 matches

# 4. Pre-commit hook is installed
ls -la /opt/github/synctacles-api/.git/hooks/pre-commit
# Should show: -rwxr-xr-x
```

### Deployment Steps (Complete)

```bash
SERVICE_USER="energy-insights-nl"
REPO_PATH="/opt/github/synctacles-api"
APP_PATH="/opt/energy-insights-nl/app"

# Step 1: Pull latest from main
sudo -u $SERVICE_USER git -C $REPO_PATH pull origin main

# Step 2: Verify no hardcoded credentials in new code
grep -r "synctacles@" $REPO_PATH --exclude-dir=.git || echo "âœ“ No credentials found"

# Step 3: Sync Python code
sudo rsync -av --delete \
    --exclude='.git' \
    --exclude='__pycache__' \
    --exclude='.env' \
    --exclude='venv' \
    $REPO_PATH/synctacles_db/ $APP_PATH/synctacles_db/

# Step 4: Sync configuration (CRITICAL - contains settings.py)
sudo rsync -av \
    $REPO_PATH/config/ $APP_PATH/config/

# Step 5: Sync scripts (collectors, normalizers)
sudo rsync -av \
    $REPO_PATH/scripts/ $APP_PATH/scripts/

# Step 6: Fix ownership (MUST be service user)
sudo chown -R $SERVICE_USER:$SERVICE_USER $APP_PATH/

# Step 7: Verify config.settings is present
test -f $APP_PATH/config/settings.py && echo "âœ“ settings.py found" || echo "âœ— settings.py MISSING"

# Step 8: Restart services
sudo systemctl restart energy-insights-nl-api
sudo systemctl restart energy-insights-nl-collector.service
sudo systemctl restart energy-insights-nl-normalizer.service

# Step 9: Validate startup
sleep 2
curl http://localhost:8000/health

# Step 10: Check logs for startup errors
sudo journalctl -u energy-insights-nl-api -n 10
```

### Deployment Checklist

Before marking deployment as complete:

```
âœ“ Git pull succeeded (no uncommitted changes blocking)
âœ“ Pre-commit hook didn't block (no hardcoded credentials)
âœ“ rsync completed successfully
âœ“ File ownership is energy-insights-nl:energy-insights-nl
âœ“ config/settings.py is in runtime directory
âœ“ Services restarted without errors
âœ“ curl /health returns 200 OK
âœ“ No "DATABASE_URL" or "role" errors in logs
âœ“ Normalizers ran without connection errors
âœ“ API serving data (check recent query results)
```

---

## TROUBLESHOOTING

### Permission Denied on Git Pull

```bash
# Symptom
fatal: unable to access '...': Permission denied

# Cause
Running git as wrong user (probably root)

# Fix
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api
sudo -u energy-insights-nl git -C /opt/github/synctacles-api pull
```

### Service Can't Read Files

```bash
# Symptom
FileNotFoundError or PermissionError in logs

# Cause
Files created by root or wrong user

# Fix
sudo chown -R energy-insights-nl:energy-insights-nl /opt/energy-insights-nl/app
```

### API Crashes After Deploy

```bash
# Symptom
curl: Failed to connect to localhost port 8000

# Cause
Likely missing config/settings.py sync

# Fix
sudo rsync -av /opt/github/synctacles-api/config/ /opt/energy-insights-nl/app/config/
sudo systemctl restart energy-insights-nl-api
```

---

## QUICK REFERENCE

```bash
# Current service account
SERVICE_USER="energy-insights-nl"

# Git operations (ALWAYS use service user)
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api pull
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api status
sudo -u $SERVICE_USER git -C /opt/github/ha-energy-insights-nl pull

# Service management
sudo systemctl status energy-insights-nl-api
sudo systemctl restart energy-insights-nl-api
sudo journalctl -u energy-insights-nl-api -f

# Fix ownership after edits
sudo chown -R $SERVICE_USER:$SERVICE_USER /opt/github/synctacles-api/
sudo chown -R $SERVICE_USER:$SERVICE_USER /opt/github/ha-energy-insights-nl/

# Full backend deploy
sudo rsync -av --delete --exclude='.git' --exclude='__pycache__' --exclude='.env' --exclude='venv' \
    /opt/github/synctacles-api/synctacles_db/ /opt/energy-insights-nl/app/synctacles_db/
sudo rsync -av /opt/github/synctacles-api/config/ /opt/energy-insights-nl/app/config/
sudo chown -R $SERVICE_USER:$SERVICE_USER /opt/energy-insights-nl/app/
sudo systemctl restart energy-insights-nl-api
```

---

## RELATED SKILLS

- **SKILL 09:** Installer Specs (FASE 0-6 setup)
- **SKILL 10:** Deployment Workflow (deploy procedures)
- **SKILL 12:** Brand-Free Architecture (multi-tenant)
- **SKILL 13:** Logging & Diagnostics (log locations)

---

**Last Updated:** 2026-01-04
**Status:** Active
