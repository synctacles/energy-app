# Credentials & Access Documentation

Last updated: 2026-01-24
Owner: @synctacles-bot

## Overview

This document tracks all credentials, API keys, SSH keys, and access configurations for the SYNCTACLES infrastructure.

---

## Infrastructure Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          SYNCTACLES INFRASTRUCTURE                          │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   GitHub     │
                              │  synctacles/ │
                              │   backend    │
                              └──────┬───────┘
                                     │
         ┌───────────────────────────┼───────────────────────────┐
         │                           │                           │
         ▼                           ▼                           ▼
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   DEV Server    │      │    cc-hub       │      │   PROD Server   │
│ 135.181.255.83  │──────│  (gateway)      │──────│  46.62.212.227  │
│ synctacles-dev  │      │ 135.181.201.253 │      │   synctacles    │
└─────────────────┘      └────────┬────────┘      └─────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │                           │
                    ▼                           ▼
          ┌─────────────────┐       ┌─────────────────┐
          │  HA PROD        │       │  HA DEV         │
          │  91.99.150.36   │       │ 82.169.33.175   │
          │  ha.synctacles  │       │ ha-dev.synctac  │
          └─────────────────┘       └─────────────────┘
```

---

## SSH Keys Overzicht

### Naming Convention

Alle SSH keys volgen het patroon: `id_<doel>`

| Key Name | Locatie | Doel | Verbindt met |
|----------|---------|------|--------------|
| `id_github` | DEV + PROD | GitHub repository access | github.com |
| `id_ccops_hub` | DEV | Toegang tot cc-hub gateway | cc-hub (135.181.201.253) |
| `id_homeassistant` | cc-hub | Home Assistant PROD | 91.99.150.36 |
| `id_ha` | cc-hub | Home Assistant DEV | 82.169.33.175:22222 |
| `id_prod` | cc-hub | SYNCTACLES PROD server | 46.62.212.227 |
| `id_dev` | cc-hub | SYNCTACLES DEV server | 135.181.255.83 |

### Key Details per Server

**DEV Server (`/home/synctacles-dev/.ssh/`):**
```
id_github          → GitHub (synctacles/backend)
id_ccops_hub       → cc-hub gateway
id_ha              → (backup) HA DEV key
```

**cc-hub (`/home/ccops/.ssh/`):**
```
id_homeassistant   → HA PROD (91.99.150.36, user: ha-user)
id_ha              → HA DEV (82.169.33.175:22222, user: root)
id_prod            → SYNCTACLES PROD (46.62.212.227, user: synctacles)
id_dev             → SYNCTACLES DEV (135.181.255.83, user: synctacles-dev)
```

**PROD Server (`/home/synctacles/.ssh/`):**
```
id_github          → GitHub (synctacles/backend)
```

---

## Server Access

### 1. SYNCTACLES DEV Server

| Property | Value |
|----------|-------|
| **Hostname** | synctacles-dev |
| **IP** | 135.181.255.83 |
| **User** | synctacles-dev |
| **SSH** | Direct (key: `id_dev` op cc-hub) |
| **Doel** | Development, testing, Claude Code |

**Paths:**
```
/opt/github/synctacles-api    # Git repository
/opt/synctacles-dev           # Runtime (venv, logs)
/opt/.env                     # Environment config
```

### 2. SYNCTACLES PROD Server

| Property | Value |
|----------|-------|
| **Hostname** | SYNCTACLES |
| **IP** | 46.62.212.227 |
| **User** | synctacles |
| **SSH** | Via cc-hub (alias: `synct-prod`) |
| **Doel** | Production API |

**Verbinden:**
```bash
# Vanaf DEV server
ssh cc-hub "ssh synct-prod '<command>'"

# Interactieve sessie
ssh -t cc-hub "ssh synct-prod"

# Deployment
~/bin/deploy-prod    # Deploy to PROD
~/bin/prod-status    # Check PROD status
```

### 3. cc-hub Gateway

| Property | Value |
|----------|-------|
| **IP** | 135.181.201.253 |
| **User** | ccops |
| **SSH Key** | `id_ccops_hub` (op DEV) |
| **Doel** | Gateway naar alle servers |

**SSH Config op DEV (`~/.ssh/config`):**
```
Host cc-hub
  HostName 135.181.201.253
  User ccops
  IdentityFile ~/.ssh/id_ccops_hub
```

---

## Home Assistant Infrastructure

### Servers

| Server | IP | SSH Port | User | Web Port | DNS |
|--------|-----|----------|------|----------|-----|
| **HA PROD** | 91.99.150.36 | 22 | ha-user | 8123 | ha.synctacles.com |
| **HA DEV** | 82.169.33.175 | 22222 | root | 8123 | ha-dev.synctacles.com |

### SSH Toegang (ALLEEN via cc-hub!)

**BELANGRIJK:** De DEV server heeft GEEN directe toegang tot de HA servers. Alle verbindingen gaan via cc-hub.

**SSH Config op cc-hub:**
```
Host homeassistant
  HostName 91.99.150.36
  User ha-user
  IdentityFile ~/.ssh/id_homeassistant
  StrictHostKeyChecking yes

Host homeassistant-dev
  HostName 82.169.33.175
  Port 22222
  User root
  IdentityFile ~/.ssh/id_ha
  StrictHostKeyChecking accept-new
```

### Verbinding Commands

**HA PROD:**
```bash
# Command uitvoeren
ssh cc-hub "ssh homeassistant '<command>'"

# Interactieve sessie
ssh -t cc-hub "ssh homeassistant"

# Voorbeeld: check resources
ssh cc-hub "ssh homeassistant 'free -h && df -h /'"
```

**HA DEV:**
```bash
# Command uitvoeren
ssh cc-hub "ssh homeassistant-dev '<command>'"

# Interactieve sessie
ssh -t cc-hub "ssh homeassistant-dev"

# Voorbeeld: check resources
ssh cc-hub "ssh homeassistant-dev 'free -h && df -h /'"
```

### Server Specificaties

**HA PROD (91.99.150.36):**
| Resource | Waarde |
|----------|--------|
| OS | Home Assistant OS (HAOS) |
| CPU | 2 cores |
| RAM | 3.7GB totaal (~2.6GB beschikbaar) |
| Disk | 38GB totaal (~31GB beschikbaar) |
| Services | hassio-supervisor, haos-agent |

**HA DEV (82.169.33.175):**
| Resource | Waarde |
|----------|--------|
| OS | Home Assistant OS (HAOS) |
| CPU | 1 core |
| RAM | 1.9GB totaal (~1.3GB beschikbaar) |
| Disk | 30.8GB totaal (~24.7GB beschikbaar) |
| SSH Port | 22222 |
| Purpose | Development & testing |

### Web Access

| Environment | URL |
|-------------|-----|
| PROD | https://ha.synctacles.com |
| DEV | https://ha-dev.synctacles.com |

---

## GitHub Access

### synctacles-bot Account

| Property | Value |
|----------|-------|
| **Username** | synctacles-bot |
| **Purpose** | Automated operations (CI/CD, commits) |
| **PAT Token** | Stored in gh CLI config |
| **Repository** | synctacles/backend |
| **Branch** | main |

### Git Configuration

**DEV Server:**
```bash
# GitHub CLI
Config: /home/synctacles-dev/.config/gh/hosts.yml
Logged in as: synctacles-bot
Protocol: SSH

# Git Remote
Path: /opt/github/synctacles-api
Remote: git@github.com:synctacles/backend.git
```

**PROD Server:**
```bash
# Git Remote
Path: /opt/github/synctacles-api
Remote: git@github.com:synctacles/backend.git
Auto-update: Disabled (manual deployment only)
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
BRAND_DOMAIN=synctacles.com

DB_NAME=synctacles
DB_USER=synctacles
DATABASE_URL=postgresql://synctacles@localhost:5432/synctacles

INSTALL_PATH=/opt/synctacles
APP_PATH=/opt/github/synctacles-api
LOG_PATH=/var/log/synctacles
API_PORT=8000
```

---

## External Monitoring (UptimeRobot)

### Setup Instructions
1. Log in to https://uptimerobot.com/dashboard
2. Create monitors:

| Monitor | URL | Interval | Alert |
|---------|-----|----------|-------|
| PROD API Health | https://api.synctacles.com/health | 1 min | Email |
| PROD Pipeline Health | https://api.synctacles.com/v1/pipeline/health | 5 min | Email |
| DEV API Health | https://dev.synctacles.com/health | 5 min | Email |

3. Set alert contact to: leo@synctacles.com
4. Enable status page (optional)

### Free Tier Limits
- 50 monitors
- 5-minute interval minimum
- Email alerts only

---

## Troubleshooting

### SSH Connection Issues

**"Permission denied (publickey)" naar HA:**
```bash
# Controleer of je via cc-hub gaat
ssh cc-hub "ssh homeassistant 'echo test'"  # Correct
ssh homeassistant 'echo test'                # FOUT - directe verbinding werkt niet
```

**"Connection timed out" naar HA:**
- HA servers zijn alleen bereikbaar via cc-hub
- Controleer of cc-hub bereikbaar is: `ssh cc-hub "echo OK"`

### Git Permission Issues

**"Permission denied" on git operations (PROD):**
```bash
sudo chown -R synctacles:synctacles /opt/github/synctacles-api/.git
```

**"Permission denied" on git operations (DEV):**
```bash
sudo chown -R synctacles-dev:synctacles-dev /opt/github/synctacles-api/.git
```

### Manual Deployment

**Deploy to PROD (from DEV):**
```bash
~/bin/deploy-prod    # Checks CI, deploys, verifies
```

**Manual git pull on PROD:**
```bash
ssh cc-hub "ssh synct-prod 'sudo -u synctacles git -C /opt/github/synctacles-api pull origin main'"
```

---

## Quick Reference

### SSH Aliases (vanaf DEV server)

```bash
# cc-hub (gateway)
ssh cc-hub

# SYNCTACLES PROD
ssh cc-hub "ssh synct-prod '<command>'"

# Home Assistant PROD
ssh cc-hub "ssh homeassistant '<command>'"

# Home Assistant DEV
ssh cc-hub "ssh homeassistant-dev '<command>'"
```

### Interactieve Sessies

```bash
# PROD server
ssh -t cc-hub "ssh synct-prod"

# HA PROD
ssh -t cc-hub "ssh homeassistant"

# HA DEV
ssh -t cc-hub "ssh homeassistant-dev"
```

---

## Resolved Issues

1. **SSH deploy key**: ✅ Fixed 2026-01-23
   - Org policy enabled deploy keys
   - Key added to GitHub
   - PROD switched from HTTPS to SSH

2. **HA DEV SSH access**: ✅ Fixed 2026-01-24
   - `id_ha` key copied to cc-hub
   - SSH config updated with correct port (22222)
   - Verified connection working

---

**See also:**
- [SKILL_11](skills/SKILL_11_REPO_AND_ACCOUNTS.md) - Repository & Service Accounts
- [SKILL_10](skills/SKILL_10_DEPLOYMENT_WORKFLOW.md) - Deployment Workflow
