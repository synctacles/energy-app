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

**Note:** As of 2026-02-04, the Knowledge Base and OpenClaw run exclusively on BRAINS (production). DEV server moltbot services removed (2026-02-04) - all harvest notifications now come directly from BRAINS via corrected import paths.

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

### Brains Server (OpenClaw KB & Harvesters)

**Server Details:**
- **Hostname:** brains.synctacles.com
- **IP:** 173.249.55.109
- **SSH:** Via cc-hub (`ssh cc-hub "ssh brains '...'"`)
- **User:** `brains` (non-root dedicated user)
- **Purpose:** Knowledge Base support bot + KB harvesters + AI inference

**Architecture (2026-02-04 - PRODUCTION):**
The Knowledge Base system runs on BRAINS as a single production environment:
- **OpenClaw Support Bot:** Python Telegram bot for HA community support (@SynctaclesSupportBot)
- **KB Harvesters:** Automated scanners for GitHub, Forums, Reddit, StackOverflow
- **Knowledge Base:** PostgreSQL 16 database with pgvector extension (17,297+ active entries)
- **Ollama:** Local LLM inference for KB query processing
- **MCP Server:** KB search tool (Node.js) for OpenClaw agents
- **No separate DEV environment:** Direct production deployment with automated backups

**Software Stack:**
- Python 3.12 with venv (support bot + harvesters)
- Node.js 22.22.0 (MCP server)
- PostgreSQL 16 + pgvector 0.6.0
- Ollama with phi3:mini (2 GB) and nomic-embed-text (0.3 GB)
- python-telegram-bot library

**Required Services:**
```bash
# OpenClaw Support Bot (Telegram)
systemctl status openclaw-support

# KB Harvester (oneshot, runs hourly via timer)
systemctl status openclaw-harvest

# Harvester Timer
systemctl status openclaw-harvest.timer

# KB Database (PostgreSQL)
systemctl status postgresql

# Ollama (Local LLM inference)
systemctl status ollama

# Prometheus Node Exporter (monitoring)
systemctl status node_exporter
```

**Service Status Check:**
```bash
ssh cc-hub 'ssh brains "systemctl is-active postgresql ollama openclaw-support openclaw-harvest.timer node_exporter"'
```

**Deployment Strategy:**
- Single production environment (no dev/staging)
- Short outages acceptable with community notification
- Daily automated database backups
- Git-based rollback capability
- systemd auto-restart on failure

**Monitoring Setup:**
The brains server is monitored via:
- **Prometheus:** Scrapes metrics from `http://173.249.55.109:9100/metrics`
- **Alertmanager:** Sends critical alerts to Slack #critical-alerts
- **Dashboard:** Status visible on cc-hub monitoring dashboard
- **Telegram:** Harvest notifications sent to group topic 3 (monitoring)

**Configuration & Credentials:**
- **Environment:** `/opt/openclaw/harvesters/.env` (chmod 600, contains all secrets)
- **Python Code:** `/opt/openclaw/harvesters/` (support_agent, tools/scanners, shared)
- **Virtual Environment:** `/opt/openclaw/harvesters/venv/`
- **MCP Server:** `/opt/openclaw/mcp/kb-search.js` (Node.js)
- **Logs:** `/opt/openclaw/logs/` (harvest.log)
- **Setup Scripts:** `/opt/github/synctacles-api/scripts/brains-setup/`

**Version Control (2026-02-04):**
- **Git Repo:** `/opt/openclaw/harvesters/.git` (initialized 2026-02-04)
- **Current Branch:** `master`
- **Latest Commit:** `cab15d5` (docs: add comprehensive README with dev workflow)
- **Pre-commit Hook:** Python syntax validation (blocks commits with syntax errors)
- **Workflow:** Feature branches → test → merge to master → rollback if needed
- **README:** `/opt/openclaw/harvesters/README.md` (dev workflow, rollback procedures)
- **Note:** Always commit working states before making changes for easy rollback

**Database:**
- **Name:** `brains_kb`
- **Schemas:**
  - `kb` (knowledge_base, knowledge_base_categories, knowledge_base_feedback, knowledge_base_usage)
  - `public` (harvest_state for tracking scanner progress)
- **Admin User:** `brains_admin` (full access, used by support bot and harvesters)
- **Connection:** `postgresql://brains_admin:***@localhost:5432/brains_kb?sslmode=disable`
- **Search Path:** `kb, public` (set at database level)
- **Data:** 17,297 active KB entries, 24 categories, avg confidence 0.78

**Telegram Bot:**
- **Username:** @SynctaclesSupportBot
- **Token:** In `/opt/openclaw/harvesters/.env` as `TELEGRAM_BOT_TOKEN_SUPPORT`
- **Group ID:** -1003846489213
- **Topics:** 2 (support), 3 (monitoring)
- **Commands:** /help, /status, /faq, /analyze
- **Status:** Admin in group with privacy mode enabled

**API Keys (in .env):**
- **GROQ_API_KEY:** Free tier LLM for harvester processing
- **ANTHROPIC_API_KEY:** Claude API for advanced processing
- **GITHUB_REPO:** home-assistant/core (for issue harvesting)

**Common Commands:**
```bash
# Restart Support Bot
ssh cc-hub 'ssh brains "sudo systemctl restart openclaw-support"'

# View Support Bot logs
ssh cc-hub 'ssh brains "sudo journalctl -u openclaw-support -f"'

# Manually trigger harvest
ssh cc-hub 'ssh brains "sudo systemctl start openclaw-harvest"'

# View harvest logs
ssh cc-hub 'ssh brains "sudo journalctl -u openclaw-harvest -f"'

# Database access (as admin)
ssh cc-hub 'ssh brains "sudo -u postgres psql -d brains_kb"'

# Check KB statistics
ssh cc-hub 'ssh brains "sudo -u postgres psql -d brains_kb -c \"SELECT COUNT(*) FROM kb.knowledge_base WHERE is_active = true;\""'

# Test Ollama models
ssh cc-hub 'ssh brains "ollama list"'

# Check all service status
ssh cc-hub 'ssh brains "systemctl status openclaw-support openclaw-harvest.timer postgresql ollama --no-pager"'
```

**Troubleshooting:**
```bash
# Check harvest state
ssh cc-hub 'ssh brains "sudo -u postgres psql -d brains_kb -c \"SELECT * FROM public.harvest_state;\""'

# Test Telegram bot token
ssh cc-hub 'ssh brains "curl -s https://api.telegram.org/bot\$(grep TELEGRAM_BOT_TOKEN_SUPPORT /opt/openclaw/harvesters/.env | cut -d= -f2)/getMe | jq"'

# Verify database permissions
ssh cc-hub 'ssh brains "sudo -u postgres psql -d brains_kb -c \"\\du brains_admin\""'

# Check service resource usage
ssh cc-hub 'ssh brains "systemctl status openclaw-support --no-pager | grep -E '\''Memory|CPU'\''"'
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

**Architecture Decision (2026-02-04 - FULLY OPERATIONAL):**
Knowledge Base and OpenClaw now run exclusively on BRAINS server as a single production environment:
- **Status:** ✅ Fully deployed and operational (2026-02-04 14:10)
- **Rationale:** KB data is identical for dev/prod, community tolerates short outages, faster iteration
- **Safety nets:** Hourly harvest backups, git-based rollback, systemd auto-restart, Telegram notifications
- **DEV server:** Platform monitoring bots remain (moltbot-monitor, moltbot-dev)
- **Migration:** Complete KB data (18,413 entries from DEV), moltbot-support → openclaw-support

**Installation Date:** 2026-02-04
**Completed By:** Claude Code (automated migration + troubleshooting)
**Services:** All operational (PostgreSQL, Ollama, openclaw-support, openclaw-harvest.timer, node_exporter)
**KB Data:** 17,297 active entries (24 categories, avg confidence 0.78)
**Harvesters:** GitHub, Forum, Reddit, StackOverflow (hourly automated scans)

**Note:** Most active development happens in product repos. This repo focuses on:
- Cross-product infrastructure
- Authentication services
- Shared libraries
- BRAINS server setup scripts
