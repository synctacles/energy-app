# Credentials & Access Documentation

Last updated: 2026-01-23

## Overview

This document tracks all credentials, API keys, and access configurations for the SYNCTACLES infrastructure.

---

## GitHub Access

### synctacles-bot Account
- **Username**: `synctacles-bot`
- **Purpose**: Automated operations (CI/CD, auto-update)
- **PAT Token**: `github_pat_11B5FV4LY0gCo19JUG8lQh_...` (stored in gh CLI config)

### Repository: synctacles/backend
- **Access**: synctacles-bot has push access
- **Main branch**: `main`

---

## Server: DEV (energy-insights-nl)

### GitHub CLI
- **Config**: `/home/energy-insights-nl/.config/gh/hosts.yml`
- **Logged in as**: `synctacles-bot`
- **Protocol**: SSH

### Git Remote
- **Path**: `/opt/github/synctacles-api`
- **Remote**: `origin` → `git@github.com:synctacles/backend.git` (SSH)

---

## Server: PROD (synct-prod / 46.62.212.227)

### Git Configuration
- **Path**: `/opt/github/synctacles-api`
- **Remote**: `origin` → `git@github.com:synctacles/backend.git` (SSH)
- **Auto-update**: Every 15 minutes via cron

### SSH Keys
- **Location**: `/home/synctacles/.ssh/`
- **Key file**: `id_github` / `id_github.pub`
- **Key name**: `synctacles-prod-deploy`
- **Status**: ✅ Active (added 2026-01-23)
- **Write access**: Yes (enabled in GitHub)

### Cron Job
- **File**: `/etc/cron.d/synctacles-autoupdate` (or similar)
- **Schedule**: `*/15 * * * *`
- **Script**: `/opt/synctacles/auto-update.sh`
- **Log**: `/var/log/synctacles-update.log`

---

## SSH Infrastructure

### Hub-Spoke Model
```
DEV Server → cc-hub → synct-prod (PROD)
```

### SSH Config (DEV)
```
Host cc-hub
    HostName [hub-ip]
    User [user]

Host synct-prod
    ProxyJump cc-hub
    HostName 46.62.212.227
```

---

## Environment Variables

### Location
- **DEV**: `/opt/.env`
- **PROD**: `/opt/.env`

### Key Variables
```bash
BRAND_NAME=SYNCTACLES
BRAND_SLUG=synctacles
DB_NAME=synctacles
DB_USER=synctacles
DATABASE_URL=postgresql://...
INSTALL_PATH=/opt/synctacles
APP_PATH=/opt/github/synctacles-api
API_PORT=8000
```

---

## Troubleshooting

### "Permission denied" on git operations (PROD)
```bash
sudo chown -R synctacles:synctacles /opt/github/synctacles-api/.git
```

### Test auto-update
```bash
sudo /opt/synctacles/auto-update.sh
```

### Check cron logs
```bash
tail -f /var/log/synctacles-update.log
```

### Manual git pull on PROD
```bash
sudo -u synctacles git -C /opt/github/synctacles-api pull origin main
```

---

## Resolved Issues

1. **SSH deploy key**: ✅ Fixed 2026-01-23
   - Org policy enabled deploy keys
   - Key added to GitHub
   - PROD switched from HTTPS to SSH
