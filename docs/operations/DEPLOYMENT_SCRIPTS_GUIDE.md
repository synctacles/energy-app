# Deployment Scripts Guide

**Date:** 2026-01-12 (Updated)
**Status:** COMPLETED
**Architecture:** Git repo = Working directory (V2.0)

---

## Overview

Systemd timer scripts are version-controlled in the git repository and executed directly from the git repo. No rsync or sync scripts needed.

**Deployment workflow:**
```bash
cd /opt/github/synctacles-api
git pull origin main
sudo systemctl restart energy-insights-nl-api
```

---

## Directory Structure

```
/opt/github/synctacles-api/          <- Git repo = Working directory
├── scripts/
│   ├── run_collectors.sh            <- Data collection
│   ├── run_importers.sh             <- Data import
│   ├── run_normalizers.sh           <- Data normalization
│   └── health_check.sh              <- Health monitoring
├── synctacles_db/                   <- Application code
├── config/                          <- Configuration modules
├── .env                             <- Environment config (in .gitignore)
└── docs/                            <- Documentation

/opt/energy-insights-nl/
└── venv/                            <- Python virtual environment (shared)
```

---

## Scripts

### run_collectors.sh
- **Purpose:** Run ENTSO-E A44 price collector
- **Triggered by:** `energy-insights-nl-collector.timer` (every 15 minutes)
- **Location:** `/opt/github/synctacles-api/scripts/run_collectors.sh`

### run_importers.sh
- **Purpose:** Import raw data to database
- **Triggered by:** `energy-insights-nl-importer.timer`
- **Location:** `/opt/github/synctacles-api/scripts/run_importers.sh`

### run_normalizers.sh
- **Purpose:** Normalize raw data to normalized tables
- **Triggered by:** `energy-insights-nl-normalizer.timer` (every 15 minutes)
- **Location:** `/opt/github/synctacles-api/scripts/run_normalizers.sh`

### health_check.sh
- **Purpose:** System health monitoring
- **Triggered by:** `energy-insights-nl-health.timer`
- **Location:** `/opt/github/synctacles-api/scripts/health_check.sh`

---

## Environment Variables

Environment file: `/opt/github/synctacles-api/.env`

```bash
# Database
DATABASE_URL=postgresql://energy_insights_nl@localhost:5432/energy_insights_nl
DB_HOST=localhost
DB_PORT=5432
DB_NAME=energy_insights_nl
DB_USER=energy_insights_nl

# Paths
INSTALL_PATH=/opt/energy-insights-nl
LOG_PATH=/var/log/energy-insights-nl

# API Keys
ENTSOE_API_KEY=your_key_here
SECRET_KEY=your_secret_here

# Environment
ENVIRONMENT=development
LOG_LEVEL=DEBUG
```

---

## Systemd Configuration

### API Service
```ini
# /etc/systemd/system/energy-insights-nl-api.service
[Service]
EnvironmentFile=/opt/github/synctacles-api/.env
WorkingDirectory=/opt/github/synctacles-api
Environment="PYTHONPATH=/opt/github/synctacles-api"
ExecStart=/opt/energy-insights-nl/venv/bin/gunicorn \
    synctacles_db.api.main:app \
    --workers 8 \
    --worker-class uvicorn.workers.UvicornWorker \
    --bind 127.0.0.1:8000 \
    --chdir /opt/github/synctacles-api
```

### Timer Services
```ini
# /etc/systemd/system/energy-insights-nl-collector.service
[Service]
Type=oneshot
User=energy-insights-nl
WorkingDirectory=/opt/github/synctacles-api
ExecStart=/opt/github/synctacles-api/scripts/run_collectors.sh
```

---

## Deployment Procedure

### Standard Deploy
```bash
cd /opt/github/synctacles-api
git pull origin main
sudo systemctl restart energy-insights-nl-api
```

### Deploy with Migrations
```bash
cd /opt/github/synctacles-api
git pull origin main
source /opt/energy-insights-nl/venv/bin/activate
alembic upgrade head
sudo systemctl restart energy-insights-nl-api
```

### Verify Deployment
```bash
# Check service status
sudo systemctl status energy-insights-nl-api

# Test health endpoint
curl -s http://localhost:8000/health | jq .

# Check logs
journalctl -u energy-insights-nl-api -n 50 --no-pager
```

---

## Design Principles

### Git Repo = Working Directory
- No rsync or sync scripts
- No separate `/app` directory
- Services run directly from git repo
- `git pull && systemctl restart` = deploy

### Brand-Free Scripts
Scripts use environment variables for all paths:

```bash
# Good (brand-free):
VENV_PATH="${VENV_PATH:-/opt/energy-insights-nl/venv}"
"${VENV_PATH}/bin/python3" -m synctacles_db.collectors.entso_e_a44_prices

# Bad (hardcoded):
/opt/energy-insights-nl/venv/bin/python3 -m synctacles_db.collectors.entso_e_a44_prices
```

### Environment Isolation
- `.env` file in git repo (protected by `.gitignore`)
- File permissions: `600` (owner read/write only)
- Never commit secrets to git

---

## Troubleshooting

### Service Won't Start
```bash
# Check logs
journalctl -u energy-insights-nl-api -n 100 --no-pager

# Verify .env exists
ls -la /opt/github/synctacles-api/.env

# Test import manually
cd /opt/github/synctacles-api
source /opt/energy-insights-nl/venv/bin/activate
python -c "from synctacles_db.api.main import app; print('OK')"
```

### Scripts Not Executable
```bash
chmod +x /opt/github/synctacles-api/scripts/*.sh
```

### Timer Not Running
```bash
# Check timer status
systemctl list-timers energy-insights-nl-*

# Check service logs
journalctl -u energy-insights-nl-collector -n 50
```

---

## Adding New Scripts

1. Create script in git repo:
   ```bash
   vim /opt/github/synctacles-api/scripts/new_script.sh
   chmod +x /opt/github/synctacles-api/scripts/new_script.sh
   ```

2. Test locally:
   ```bash
   bash /opt/github/synctacles-api/scripts/new_script.sh
   ```

3. Commit to git:
   ```bash
   git add scripts/new_script.sh
   git commit -m "feat: add new_script for xyz"
   git push origin main
   ```

4. Create systemd service/timer if needed

---

## Multi-Server Deployment

### Synctacles Server (135.181.255.83)
```bash
cd /opt/github/synctacles-api
git pull origin main
sudo systemctl restart energy-insights-nl-api
```

### Coefficient Server (91.99.150.36)
```bash
cd /opt/github/coefficient-engine
git pull origin main
sudo systemctl restart coefficient-api
```

---

## References

- [SKILL_10_DEPLOYMENT_WORKFLOW.md](../skills/SKILL_10_DEPLOYMENT_WORKFLOW.md) - Full deployment documentation
- [SKILL_12_BRAND_FREE_ARCHITECTURE.md](../skills/SKILL_12_BRAND_FREE_ARCHITECTURE.md) - Brand-free design principles

---

**Status:** Ready for Production
