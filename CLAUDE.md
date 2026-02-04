# CLAUDE.md - Project Context for Claude Code

## ⚠️ SCOPE WARNING

**Dit is de PLATFORM repo. Product-specifieke code hoort hier NIET.**

| Component | Juiste repo |
|-----------|-------------|
| Energy API | `synctacles/energy` |
| Care/Moltbot | `synctacles/care` |
| Brains/AI | `synctacles/brains` |
| HA Integration | `synctacles/ha-integration` |

**Deze repo is ALLEEN voor:**
- Auth Service (centralized authentication)
- Shared libraries (gebruikt door alle products)
- Infrastructure as Code (deployment scripts, systemd templates)
- Cross-product utilities
- Platform documentation

**NIET in deze repo:**
- ❌ Product-specifieke API endpoints
- ❌ Product-specifieke business logic
- ❌ Product-specifieke collectors/workers
- ❌ Product-specifieke tests

---

## Error Handling Protocol

**CRITICAL: When errors are detected, follow this protocol:**

1. **Immediate Investigation**
   - Stop current work and investigate the error immediately
   - Identify root cause and impact
   - Document findings

2. **Priority Assessment**
   - **High Priority** (blocking, security, data loss): Fix immediately
   - **Medium Priority** (bugs, broken features): Create GitHub issue in this repo
   - **Low Priority** (nice-to-have, optimization): Create GitHub issue with "enhancement" label

3. **Issue Creation Template**
   ```bash
   gh issue create --repo synctacles/platform \
     --title "Error: [Brief description]" \
     --body "## Problem
   [Error description]

   ## Context
   [When/where detected]

   ## Investigation
   [Root cause analysis]

   ## Impact
   [What's affected]

   ## Suggested Fix
   [Potential solution]" \
     --label "bug"
   ```

4. **Never Ignore Errors**
   - All errors must be either fixed immediately or tracked in GitHub
   - Silent failures are unacceptable
   - Always inform user of error and tracking status

---

## Project Overview
SYNCTACLES Platform - Shared infrastructure and services for all Synctacles products (Energy, Care, Brains).

## Repository Structure
```
platform/
├── auth/                    # Auth Service (centralized authentication)
│   ├── main.py
│   ├── models.py
│   └── middleware.py
├── shared/                  # Shared libraries
│   ├── database/           # Common database utilities
│   ├── logging/            # Logging configuration
│   ├── monitoring/         # Prometheus metrics
│   └── utils/              # Common utilities
├── infrastructure/         # IaC templates
│   ├── systemd/           # Systemd service templates
│   ├── nginx/             # Nginx configs
│   └── deployment/        # Deployment scripts
├── docs/
│   ├── ARCHITECTURE.md    # Platform architecture
│   ├── DEPLOYMENT.md      # Deployment procedures
│   └── API.md             # Auth API documentation
└── tests/
    ├── test_auth.py
    └── test_shared.py
```

## Infrastructure

### Servers
| Server | Purpose | Access |
|--------|---------|--------|
| cc-hub | Central monitoring hub | Direct (current session) |
| DEV | Quick testing & development (synctacles-dev) | Via `ssh cc-hub "ssh synct-dev '...'"` |
| PROD | Production Energy API (synctacles-prod) | Via `ssh cc-hub "ssh synct-prod '...'"` |
| BRAINS | **PRODUCTION** OpenClaw, KB & Ollama (brains.synctacles.com) | Via `ssh cc-hub "ssh brains '...'"` |
| MONITOR | Prometheus & Grafana (77.42.41.135) | Via `ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 '...'"` |

**Note:** As of 2026-02-04, the Knowledge Base and OpenClaw run exclusively on BRAINS (production). DEV server is used for quick code testing only.

### GitHub Account
- **Bot account**: `synctacles-bot`
- **Repository**: `synctacles/platform`
- **Authentication**: PAT token (configured in gh CLI)

### Product Repositories
- **Energy**: `synctacles/energy` - Price API, collectors
- **Care**: `synctacles/care` - Support bot, KB
- **Brains**: `synctacles/brains` - AI/ML services
- **HA Integration**: `synctacles/ha-integration` - Home Assistant addon

## Development Workflow

### Local Development
```bash
# Setup
python3.12 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Run Auth Service
uvicorn platform.auth.main:app --reload --port 8000

# Run tests
pytest

# Code quality
ruff format .
ruff check .
```

## Deployment

### Shared Libraries
When shared libraries are updated, all dependent products must be notified:
```bash
# After merging shared lib changes
gh issue create --repo synctacles/energy --title "Update shared lib dependency"
gh issue create --repo synctacles/care --title "Update shared lib dependency"
```

### Auth Service
Auth service updates require coordination across all products:
```bash
# Deploy Auth Service
~/bin/deploy-platform

# Verify all products can authenticate
~/bin/verify-auth-integration
```

## Server Setup

### Brains Server (OpenClaw & KB)

**Server Details:**
- **Hostname:** brains.synctacles.com
- **IP:** 173.249.55.109
- **SSH:** Via cc-hub (`ssh cc-hub "ssh brains '...'"`)
- **User:** `brains` (non-root dedicated user)
- **Purpose:** OpenClaw API (HA community communication) + Knowledge Base + AI inference

**Architecture (2026-02-04 - UPDATED):**
The Knowledge Base system runs on BRAINS as a single production environment:
- **OpenClaw Gateway:** Telegram bot for HA community support (v2026.2.2-3)
- **Knowledge Base:** PostgreSQL 16 database with pgvector extension
- **Ollama:** Local LLM inference (phi3:mini, llama3:8b, nomic-embed-text)
- **MCP Server:** KB search tool for OpenClaw agents
- **No separate DEV environment:** Direct production deployment

**Software Stack:**
- Node.js 22.22.0
- OpenClaw 2026.2.2-3 with MCP SDK
- PostgreSQL 16 + pgvector 0.6.0
- Ollama with phi3:mini (2.2GB) and nomic-embed-text (274MB)

**Required Services:**
```bash
# OpenClaw Gateway (Telegram bot + agents)
systemctl status openclaw

# KB Database (PostgreSQL)
systemctl status postgresql

# Ollama (Local LLM inference)
systemctl status ollama

# Prometheus Node Exporter (for monitoring)
systemctl status node_exporter
```

**Service Status Check:**
```bash
ssh cc-hub 'ssh brains "systemctl is-active postgresql ollama openclaw node_exporter"'
```

**Deployment Strategy:**
- Single production environment (no dev/staging)
- Short outages acceptable with community notification
- Daily automated database backups
- Git-based rollback capability
- Quick smoke tests in screen before systemctl restart

**Monitoring Setup:**
The brains server is monitored via:
- **Prometheus:** Scrapes metrics from `https://brains.synctacles.com/metrics` and `http://173.249.55.109:9100/metrics`
- **Alertmanager:** Sends critical alerts to Slack #critical-alerts
- **Dashboard:** Status visible on cc-hub monitoring dashboard

**Configuration & Credentials:**
- **OpenClaw Config:** `/etc/openclaw/openclaw.json`
- **Secrets:** `/etc/openclaw/secrets.env` (chmod 600, contains Telegram token)
- **DB Credentials:** `/root/.openclaw-credentials/` (admin + reader passwords)
- **Workspace:** `/opt/openclaw/workspace/`
- **MCP Server:** `/opt/openclaw/mcp/kb-search.js`
- **Setup Scripts:** `/home/brains/setup/scripts/brains-setup/`

**Database:**
- **Name:** `brains_kb`
- **Schema:** `kb` (tables: `entries`, `query_log`)
- **Admin User:** `brains_admin` (full access)
- **OpenClaw User:** `openclaw_reader` (read-only on entries, insert on query_log)
- **Connection:** `postgresql://openclaw_reader:***@localhost:5432/brains_kb`

**Telegram Bot:**
- **Token:** Configured in `/etc/openclaw/secrets.env`
- **Group ID:** -1003846489213 (migrated from Moltbot)
- **Bot Name:** Brains KB Bot (via OpenClaw)

**Common Commands:**
```bash
# Restart OpenClaw
ssh cc-hub 'ssh brains "sudo systemctl restart openclaw"'

# View logs
ssh cc-hub 'ssh brains "sudo journalctl -u openclaw -f"'

# Database access (as admin)
ssh cc-hub 'ssh brains "sudo -u postgres psql -d brains_kb"'

# Test Ollama
ssh cc-hub 'ssh brains "ollama list"'
```

**SSH Key:**
The public key for cc-hub→brains access:
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPpL62iQAPP12ih0TZaXlMAH31cYdV6a9ZHaO+GF0Iie ccops@cc-hub->brains
```
This must be added to `/home/brains/.ssh/authorized_keys` on the brains server (as user `brains`, NOT root).

## CI/CD Pipeline
GitHub Actions runs on every push:
- Ruff linting & formatting
- Pytest (unit + integration tests)
- Auth service validation
- Shared lib compatibility checks

## Related Repos
- **Energy:** https://github.com/synctacles/energy
- **Care:** https://github.com/synctacles/care
- **Brains:** https://github.com/synctacles/brains
- **HA Integration:** https://github.com/synctacles/ha-integration

## Migration Status (2026-02-04)

This repository was renamed from `synctacles/backend` to `synctacles/platform` as part of the multi-repo migration:
- ✅ Energy code extracted to `synctacles/energy`
- ✅ Platform/Auth code remains here
- ✅ Shared libraries organized
- ✅ **KB system consolidated to BRAINS (single production env)**
- 🚧 Care extraction (planned)
- 🚧 Brains extraction (planned)

**Architecture Decision (2026-02-04 - IMPLEMENTED):**
Knowledge Base and OpenClaw now run exclusively on BRAINS server as a single production environment:
- **Status:** ✅ Fully deployed and operational (2026-02-04 13:27)
- **Rationale:** KB data is identical for dev/prod, community tolerates short outages, faster iteration
- **Safety nets:** Daily backups, git-based rollback, systemd auto-restart, status notifications
- **DEV server:** Used only for quick code testing in screen/tmux before production deployment
- **Migration:** Moltbot Telegram tokens migrated to OpenClaw configuration

**Installation Date:** 2026-02-04
**Installed By:** Claude Code (automated via handoff)
**Services:** All operational (PostgreSQL, Ollama, OpenClaw, node_exporter)
**KB Data:** Fresh start (0 entries - ready for population)

**Note:** Most active development happens in product repos. This repo focuses on:
- Cross-product infrastructure
- Authentication services
- Shared libraries
- BRAINS server setup scripts
