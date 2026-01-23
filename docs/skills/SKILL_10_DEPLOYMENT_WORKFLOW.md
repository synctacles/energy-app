# SKILL 10 — DEPLOYMENT WORKFLOW

Deployment Strategy for SYNCTACLES
Version: 2.0 (2026-01-12)

---

## GOAL

Simple, reliable deployment:
```bash
git pull && sudo systemctl restart synctacles-api
```

No symlinks. No sync scripts. Git repo = Production directory.

---

## ARCHITECTURE (V2.0)

### Design Principle
**Git repository IS the working directory** for systemd services.

```
/opt/github/synctacles-api/          ← Git repo AND working directory
├── synctacles_db/                   ← Application code
├── scripts/                         ← Operational scripts
├── config/                          ← Configuration modules
├── alembic/                         ← Database migrations
├── .env                             ← Environment config (in .gitignore)
└── docs/                            ← Documentation
```

### Environment File
- Location: `/opt/github/synctacles-api/.env`
- Protected by `.gitignore`
- Contains secrets (API keys, database credentials)
- File permissions: `600` (owner read/write only)

### Virtual Environment
- Location: `/opt/synctacles-dev/venv/`
- Shared across git repo (not inside repo)
- Managed separately from deployments

---

## SYSTEMD CONFIGURATION

### API Service
```ini
# /etc/systemd/system/synctacles-api.service
[Service]
EnvironmentFile=/opt/github/synctacles-api/.env
WorkingDirectory=/opt/github/synctacles-api
Environment="PYTHONPATH=/opt/github/synctacles-api"

ExecStart=/opt/synctacles-dev/venv/bin/gunicorn \
    synctacles_db.api.main:app \
    --workers 8 \
    --worker-class uvicorn.workers.UvicornWorker \
    --bind 127.0.0.1:8000 \
    --chdir /opt/github/synctacles-api
```

### Background Services
All services use consistent paths:
- `WorkingDirectory=/opt/github/synctacles-api`
- `ExecStart=/opt/github/synctacles-api/scripts/<script>.sh`

---

## DEPLOYMENT WORKFLOW

### Standard Deploy
```bash
cd /opt/github/synctacles-api
git pull origin main
sudo systemctl restart synctacles-api
```

### With Migrations
```bash
cd /opt/github/synctacles-api
git pull origin main
source /opt/synctacles-dev/venv/bin/activate
alembic upgrade head
sudo systemctl restart synctacles-api
```

### Validation
```bash
# Check service status
sudo systemctl status synctacles-api

# Test health endpoint
curl -s http://localhost:8000/health | jq .

# Check logs for errors
journalctl -u synctacles-api -n 50 --no-pager
```

---

## ROLLBACK

### Quick Rollback (previous commit)
```bash
cd /opt/github/synctacles-api
git checkout HEAD~1
sudo systemctl restart synctacles-api
```

### Rollback to Specific Tag
```bash
cd /opt/github/synctacles-api
git checkout v1.0.0
sudo systemctl restart synctacles-api
```

### Return to Latest
```bash
cd /opt/github/synctacles-api
git checkout main
git pull origin main
sudo systemctl restart synctacles-api
```

---

## PROHIBITED PATTERNS

### NO symlinks for configuration
```bash
# WRONG - creates maintenance burden
ln -s /opt/.env /opt/github/synctacles-api/.env

# CORRECT - direct file in repo
cp /opt/.env /opt/github/synctacles-api/.env
```

### NO separate app directory
```bash
# WRONG - requires sync
/opt/synctacles-dev/app/     ← Separate from git

# CORRECT - git IS app
/opt/github/synctacles-api/      ← Git repo = app directory
```

### NO sync scripts
```bash
# WRONG - complex, error-prone
rsync git-repo/ /opt/app/

# CORRECT - direct git operations
git pull && systemctl restart
```

---

## ENVIRONMENT VARIABLES

### Required Variables
```bash
# Database
DATABASE_URL=postgresql://user@localhost:5432/dbname
DB_HOST=localhost
DB_PORT=5432
DB_NAME=energy_insights_nl
DB_USER=energy_insights_nl

# API Keys (sensitive)
ENTSOE_API_KEY=<uuid>
SECRET_KEY=<hex>

# API Configuration
API_HOST=0.0.0.0
API_PORT=8000

# Paths
INSTALL_PATH=/opt/synctacles-dev
LOG_PATH=/var/log/synctacles-dev
```

### .gitignore Protection
```
# Environment (NEVER commit)
.env
.env.local
.env.*.local
```

---

## MULTI-SERVER DEPLOYMENT

### Synctacles Server (135.181.255.83)
```bash
# Deploy synctacles-api
cd /opt/github/synctacles-api
git pull && sudo systemctl restart synctacles-api
```

### Coefficient Server (91.99.150.36)
```bash
# Deploy coefficient-engine
cd /opt/github/coefficient-engine
git pull && sudo systemctl restart coefficient-engine
```

### Cross-Server Dependencies
1. Synctacles calls Coefficient at `http://91.99.150.36:8080`
2. IP whitelist on Coefficient: `ALLOWED_IPS=135.181.255.83`
3. Deploy Coefficient BEFORE Synctacles if API changes

---

## EMERGENCY PROCEDURES

### API Not Starting
```bash
# 1. Check logs
journalctl -u synctacles-api -n 100 --no-pager

# 2. Check .env exists and has correct permissions
ls -la /opt/github/synctacles-api/.env
# Should show: -rw------- 1 energy-insights-nl ...

# 3. Test import manually
cd /opt/github/synctacles-api
source /opt/synctacles-dev/venv/bin/activate
python -c "from synctacles_db.api.main import app; print('OK')"
```

### Database Connection Failed
```bash
# 1. Check PostgreSQL
sudo systemctl status postgresql

# 2. Test connection
psql -U energy_insights_nl -d energy_insights_nl -c "SELECT 1"

# 3. Check DATABASE_URL in .env
grep DATABASE_URL /opt/github/synctacles-api/.env
```

### Missing Module Import
```bash
# Common cause: __init__.py imports non-existent module
grep "from \. import" /opt/github/synctacles-api/synctacles_db/api/endpoints/__init__.py

# Fix: remove import for archived modules
```

---

## VERSION TRACKING

### Git Tags
```bash
# Create release tag
git tag -a v1.0.1 -m "Release 1.0.1 - bugfix"
git push origin v1.0.1

# Deploy specific version
git checkout v1.0.1
sudo systemctl restart synctacles-api
```

### Health Endpoint Version
```bash
curl -s http://localhost:8000/health | jq '.version'
# Returns: "1.0.0"
```

---

## RELATED SKILLS

- SKILL 8: Hardware Profile (server specs)
- SKILL 9: Installer Specs (initial setup)
- SKILL 12: Brand-Free Architecture (naming conventions)
