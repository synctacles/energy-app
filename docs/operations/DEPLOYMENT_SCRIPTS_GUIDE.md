# Deployment Scripts Guide - Brand-Free Version Control

**Date:** 2026-01-05
**Status:** COMPLETED
**Versions:** run_collectors.sh, run_normalizers.sh

---

## Overview

Essential systemd timer scripts are now version-controlled in the git repository. These scripts use environment variables exclusively - no hardcoded brand strings or paths.

## Scripts Included

### run_collectors.sh
- **Purpose:** Run all data collectors (A44, A65, A75, Energy-Charts)
- **Triggered by:** `energy-insights-nl-collector.timer` (every 15 minutes)
- **Location:** `/opt/github/synctacles-api/scripts/run_collectors.sh`

### run_normalizers.sh
- **Purpose:** Normalize raw data to normalized tables
- **Triggered by:** `energy-insights-nl-normalizer.timer` (every 15 minutes)
- **Location:** `/opt/github/synctacles-api/scripts/run_normalizers.sh`

---

## Environment Variables

All scripts respect these environment variables (fallback to defaults):

```bash
INSTALL_PATH         # Base installation directory
                     # Default: /opt/energy-insights-nl

VENV_PATH            # Python virtual environment
                     # Default: ${INSTALL_PATH}/venv

APP_PATH             # Application root
                     # Default: ${INSTALL_PATH}/app

LOG_PATH             # Log directory
                     # Default: /var/log/energy-insights
```

## Configuration

Set environment variables in `/opt/.env`:

```bash
export INSTALL_PATH=/opt/energy-insights-nl
export VENV_PATH=${INSTALL_PATH}/venv
export APP_PATH=${INSTALL_PATH}/app
export LOG_PATH=/var/log/energy-insights
export LOG_LEVEL=INFO

# Database
export DATABASE_URL=postgresql://energy_insights_nl@localhost:5432/energy_insights_nl
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=energy_insights_nl
export DB_USER=energy_insights_nl

# API Keys (optional)
export ENTSOE_API_KEY=your_key_here
```

## Deployment Procedure

### 1. Sync from Source Repository

```bash
# Sync all source code
sudo rsync -av --delete \
    --exclude='.git' \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    --exclude='.env' \
    --exclude='venv' \
    /opt/github/synctacles-api/synctacles_db/ \
    /opt/energy-insights-nl/app/synctacles_db/

# Sync configuration
sudo rsync -av --delete \
    /opt/github/synctacles-api/config/ \
    /opt/energy-insights-nl/app/config/

# Sync scripts (preserve executable permissions!)
sudo rsync -av \
    /opt/github/synctacles-api/scripts/ \
    /opt/energy-insights-nl/app/scripts/

# Fix ownership
sudo chown -R energy-insights-nl:energy-insights-nl /opt/energy-insights-nl/app/
```

### 2. Verify Scripts

```bash
# Check scripts exist
ls -la /opt/energy-insights-nl/app/scripts/run_*.sh

# Verify they're executable
file /opt/energy-insights-nl/app/scripts/run_normalizers.sh
```

### 3. Restart Services

```bash
sudo systemctl restart energy-insights-nl-collector.service
sudo systemctl restart energy-insights-nl-normalizer.service

# Monitor
sudo journalctl -u energy-insights-nl-collector.service -f
sudo journalctl -u energy-insights-nl-normalizer.service -f
```

---

## Design Principles

### Brand-Free Scripts
- ✅ Use environment variables for ALL paths
- ✅ Work with any deployment directory
- ✅ Can be deployed to different hosts
- ✅ Support multiple environments (dev, staging, prod)

Example:
```bash
# Bad (hardcoded):
PYTHON="/opt/energy-insights-nl/venv/bin/python3"

# Good (brand-free):
PYTHON="${VENV_PATH}/bin/python3"
```

### Version Control Protection
- ✅ Scripts in git repository
- ✅ Protected from accidental deletion
- ✅ Deployment via rsync preserves scripts
- ✅ Enables safe deployments with `--delete` flag

### Environment Isolation
- ✅ Source environment from `/opt/.env`
- ✅ Works with systemd service environment
- ✅ Respects admin configuration
- ✅ Fails fast if required vars missing

---

## Troubleshooting

### Scripts Not Found After Deployment

**Problem:** rsync deleted scripts

**Solution:** Ensure rsync command does NOT use `--delete` on scripts directory:
```bash
# Wrong:
rsync --delete /source/ /runtime/

# Right:
rsync /source/scripts/ /runtime/scripts/  # Explicit, no --delete
```

### Scripts Won't Execute

**Problem:** Scripts are not executable

**Solution:**
```bash
chmod +x /opt/energy-insights-nl/app/scripts/run_*.sh
```

### Environment Variables Not Loaded

**Problem:** Scripts error about missing python

**Solution:** Ensure `/opt/.env` exists and contains `INSTALL_PATH`:
```bash
test -f /opt/.env && echo "Config exists" || echo "Missing /opt/.env"
```

### Collections/Normalizers Not Running

**Problem:** Timers active but no execution

**Solution:** Verify script is in correct location:
```bash
systemctl cat energy-insights-nl-collector.service | grep ExecStart
# Should show: ExecStart=/opt/energy-insights-nl/app/scripts/run_collectors.sh
```

---

## Git Management

### Adding New Scripts

1. Write script in runtime directory
2. Test thoroughly
3. Copy to source repo:
   ```bash
   cp /opt/energy-insights-nl/app/scripts/new_script.sh \
      /opt/github/synctacles-api/scripts/
   ```
4. Commit to git:
   ```bash
   git add scripts/new_script.sh
   git commit -m "feat: add new_script for xyz"
   git push origin main
   ```

### Updating Existing Scripts

1. Edit script in source repo
2. Test changes
3. Sync to runtime:
   ```bash
   rsync /opt/github/synctacles-api/scripts/run_normalizers.sh \
         /opt/energy-insights-nl/app/scripts/
   ```
4. Verify functionality
5. Commit to git with descriptive message

---

## Service Integration

### Systemd Timer Units

**File:** `/etc/systemd/system/energy-insights-nl-collector.timer`
```ini
[Unit]
Description=Energy Insights NL Collector Timer
Requires=energy-insights-nl-collector.service

[Timer]
OnBootSec=5min
OnUnitActiveSec=15min
Persistent=true

[Install]
WantedBy=timers.target
```

**File:** `/etc/systemd/system/energy-insights-nl-collector.service`
```ini
[Unit]
Description=Energy Insights NL Data Collector
After=network.target postgresql.service

[Service]
Type=oneshot
User=energy-insights-nl
WorkingDirectory=/opt/energy-insights-nl/app
EnvironmentFile=/opt/.env
ExecStart=/opt/energy-insights-nl/app/scripts/run_collectors.sh

[Install]
WantedBy=multi-user.target
```

---

## Migration from Hardcoded Scripts

If you have hardcoded scripts, migrate to brand-free version:

### Before (Hardcoded)
```bash
#!/bin/bash
PYTHON="/opt/energy-insights-nl/venv/bin/python3"
cd /opt/energy-insights-nl/app
"$PYTHON" -m synctacles_db.collectors.entso_e_a44_prices
```

### After (Brand-Free)
```bash
#!/bin/bash
INSTALL_PATH="${INSTALL_PATH:-/opt/energy-insights-nl}"
VENV_PATH="${VENV_PATH:-${INSTALL_PATH}/venv}"
APP_PATH="${APP_PATH:-${INSTALL_PATH}/app}"

source /opt/.env  # Load deployment config
cd "${APP_PATH}"
"${VENV_PATH}/bin/python3" -m synctacles_db.collectors.entso_e_a44_prices
```

---

## Testing Scripts Locally

```bash
# Test with current environment
bash /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# Test with custom environment
INSTALL_PATH=/tmp/test bash /opt/github/synctacles-api/scripts/run_normalizers.sh

# Dry-run (check syntax)
bash -n /opt/github/synctacles-api/scripts/run_collectors.sh
```

---

## Changelog

- **2026-01-05:** Scripts added to git repository as brand-free version
  - run_collectors.sh
  - run_normalizers.sh
  - Both use env vars for paths
  - Both tested and verified

---

## References

- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [Systemd Timer Guide](https://www.freedesktop.org/software/systemd/man/systemd.timer.html)
- [Environment Variables Setup](/opt/.env)

---

**Status:** Ready for Production ✅

All scripts are version-controlled, brand-free, and safe for deployment.
