# SKILL 11 â€” REPO STRUCTURE & SERVICE ACCOUNTS

Repository Organisation and Account Management
Version: 1.0 (2026-01-02)

---

## PURPOSE

Define the GitHub repository structure, service account conventions, and git workflow rules. This ensures consistent deployments across all environments and prevents permission issues.

---

## REPOSITORY STRUCTURE

### Active Repositories

| Repository | Purpose | Contains |
|------------|---------|----------|
| `DATADIO/synctacles-api` | Backend API server | Python API, collectors, importers, normalizers, systemd units |
| `DATADIO/ha-energy-insights-nl` | Home Assistant integration | HA custom component only (HACS compatible) |

### Archived Repositories

| Repository | Status | Reason |
|------------|--------|--------|
| `DATADIO/synctacles-ha` | ARCHIVED | Replaced by `ha-energy-insights-nl` |
| `DATADIO/synctacles-repo` | ARCHIVED | Replaced by `synctacles-api` |

### Repository Rules

1. **synctacles-api** = ALL backend code (brand-free)
2. **ha-energy-insights-nl** = ONLY HA component code
3. Backend repo NEVER contains HA component in production
4. HA repo NEVER contains backend code

---

## SERVICE ACCOUNTS

### Current Configuration

| Setting | Value |
|---------|-------|
| Service account | `energy-insights-nl` |
| Service group | `energy-insights-nl` |
| Git owner | `energy-insights-nl` |
| Install path | `/opt/energy-insights-nl` |
| Repo path | `/opt/github/synctacles-api` |

### Future Configuration (Post-Migration)

| Setting | Value |
|---------|-------|
| Service account | `synctacles` |
| Service group | `synctacles` |
| Git owner | `synctacles` |
| Install path | `/opt/synctacles` |
| Repo path | `/opt/github/synctacles-api` |

### Why Single Generic Account?

- **Simplicity:** 1 account for all brands
- **Brand = .env only:** Same code, different config
- **No code changes:** Switch brand via environment variables
- **Audit trail:** Clear ownership of all operations

---

## GIT WORKFLOW RULES

### CRITICAL: No Root Git Operations

```bash
# âŒ NEVER DO THIS
sudo git pull
sudo git status
root@server: git clone ...

# âœ… ALWAYS DO THIS
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

# Pull latest code
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api pull

# Check status
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api status

# View log
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api log --oneline -5

# Clone new repo (if needed)
sudo -u $SERVICE_USER git clone https://github.com/DATADIO/synctacles-api.git /opt/github/synctacles-api
```

---

## DIRECTORY STRUCTURE

### Current Server Layout

```
/opt/
â”œâ”€â”€ .env                              # Master config (brand settings)
â”œâ”€â”€ github/
â”‚   â””â”€â”€ synctacles-api/               # Git repo (owned by energy-insights-nl)
â”‚       â”œâ”€â”€ synctacles_db/            # Backend Python code
â”‚       â”œâ”€â”€ alembic/                  # Database migrations
â”‚       â”œâ”€â”€ systemd/                  # Service templates
â”‚       â”œâ”€â”€ scripts/                  # Setup/deployment scripts
â”‚       â””â”€â”€ docs/                     # Documentation
â””â”€â”€ energy-insights-nl/               # Runtime deployment
    â”œâ”€â”€ app/                          # Deployed code (copy from repo)
    â”œâ”€â”€ venv/                         # Python virtual environment
    â””â”€â”€ logs -> /var/log/energy-insights-nl/

/var/log/energy-insights-nl/          # Log files
/etc/systemd/system/                  # Generated service units
```

### Future Server Layout (Post-Migration)

```
/opt/
â”œâ”€â”€ .env                              # Master config
â”œâ”€â”€ github/
â”‚   â””â”€â”€ synctacles-api/               # Git repo (owned by synctacles)
â””â”€â”€ synctacles/                       # Runtime deployment
    â”œâ”€â”€ app/
    â”œâ”€â”€ venv/
    â””â”€â”€ logs -> /var/log/synctacles/
```

---

## BRAND CONFIGURATION

### How Branding Works

**Same code, different .env:**

```bash
# Netherlands (Energy Insights NL)
BRAND_NAME="Energy Insights NL"
BRAND_SLUG="energy-insights-nl"
BRAND_DOMAIN="energy-insights.nl"

# Commercial (SYNCTACLES)
BRAND_NAME="SYNCTACLES"
BRAND_SLUG="synctacles"
BRAND_DOMAIN="synctacles.io"
```

### What Changes Per Brand

| Changes | Doesn't Change |
|---------|----------------|
| .env values | Python code |
| Domain/URLs | API logic |
| Display names | Database schema |
| Systemd unit names | Git repository |

---

## DEPLOYMENT WORKFLOW

### Pull & Deploy (Current)

```bash
SERVICE_USER="energy-insights-nl"
REPO_PATH="/opt/github/synctacles-api"
APP_PATH="/opt/energy-insights-nl/app"

# 1. Pull latest
sudo -u $SERVICE_USER git -C $REPO_PATH pull

# 2. Sync to app directory
sudo rsync -av --delete \
    --exclude='.git' \
    --exclude='__pycache__' \
    --exclude='.env' \
    $REPO_PATH/synctacles_db/ $APP_PATH/synctacles_db/

# 3. Run migrations
sudo -u $SERVICE_USER $APP_PATH/../venv/bin/alembic upgrade head

# 4. Restart services
sudo systemctl restart energy-insights-nl-api
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

**This repo is for HACS distribution only.**

Contents:
```
ha-energy-insights-nl/
â”œâ”€â”€ custom_components/
â”‚   â””â”€â”€ ha_energy_insights_nl/
â”‚       â”œâ”€â”€ __init__.py
â”‚       â”œâ”€â”€ config_flow.py
â”‚       â”œâ”€â”€ sensor.py
â”‚       â”œâ”€â”€ const.py
â”‚       â”œâ”€â”€ manifest.json
â”‚       â”œâ”€â”€ strings.json
â”‚       â””â”€â”€ tennet_client.py      # TenneT BYO-key client
â”œâ”€â”€ hacs.json
â””â”€â”€ README.md
```

### User Installation

Users install via HACS:
1. Add custom repository: `DATADIO/ha-energy-insights-nl`
2. Install integration
3. Configure API URL + optional TenneT key

---

## MIGRATION CHECKLIST (Future)

When migrating from `energy-insights-nl` to `synctacles`:

```bash
# 1. Create new user
sudo useradd -r -s /bin/bash synctacles

# 2. Transfer ownership
sudo chown -R synctacles:synctacles /opt/github/synctacles-api

# 3. Update .env
sudo sed -i 's/energy-insights-nl/synctacles/g' /opt/.env

# 4. Create new directories
sudo mkdir -p /opt/synctacles/{app,venv,logs}
sudo chown -R synctacles:synctacles /opt/synctacles

# 5. Regenerate systemd units (from templates)
# 6. Enable new services, disable old
# 7. Test everything
# 8. Remove old user (after validation)
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

---

## RELATED SKILLS

- **SKILL 09:** Installer Specs (FASE 0-6 setup)
- **SKILL 10:** Deployment Workflow (deploy procedures)
- **SKILL 12:** Brand-Free Architecture (multi-tenant)

---

## CC (CLAUDE CODE) PERMISSION RULES

### Automatic Context Detection

CC moet zelf bepalen welke user context nodig is per operatie:

| Operatie | User | Command Prefix |
|----------|------|----------------|
| Git (status, pull, commit, push) | service user | `sudo -u energy-insights-nl` |
| File edits in repo | root (dan fix) | `sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/` |
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

**Voor ALLE git operaties:**
```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api <command>
```

**NOOIT:**
```bash
# FOUT - veroorzaakt ownership issues
git status
git pull
git commit
```

### Complete Deployment Voorbeeld

```bash
# 1. Pull latest (service user)
sudo -u energy-insights-nl git -C /opt/github/synctacles-api pull

# 2. Edit files (root is OK)
nano /opt/github/synctacles-api/docs/api-reference.md

# 3. Fix ownership (VERPLICHT na edits)
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

# 4. Git commit (service user)
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: update"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push

# 5. Restart service (root)
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

## QUICK REFERENCE

```bash
# Current service account
SERVICE_USER="energy-insights-nl"

# Git operations (ALWAYS use service user)
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api pull
sudo -u $SERVICE_USER git -C /opt/github/synctacles-api status

# Service management
sudo systemctl status energy-insights-nl-api
sudo systemctl restart energy-insights-nl-api
sudo journalctl -u energy-insights-nl-api -f

# Fix ownership after edits
sudo chown -R $SERVICE_USER:$SERVICE_USER /opt/github/synctacles-api/
```

---

**Last Updated:** 2026-01-02
**Status:** Active
