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
Synctacles Platform - Shared infrastructure and services for all Synctacles products (Energy, Care).

**Key Documentation:**
- [docs/NAMING_CONVENTIONS.md](docs/NAMING_CONVENTIONS.md) - **Official naming standards** for services, databases, directories, APIs
- [docs/PLATFORM_ARCHITECTURE.md](docs/PLATFORM_ARCHITECTURE.md) - Microservices design, authentication flow, adding new products

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

> **Naming Convention:** See [docs/NAMING_CONVENTIONS.md](docs/NAMING_CONVENTIONS.md) for complete naming standards.

### Servers
| Server | Domain | Purpose | SSH Alias | Access |
|--------|--------|---------|-----------|--------|
| cc-hub | - | Central monitoring hub | - | Direct (current session) |
| ENERGY-DEV | dev.synctacles.com | Multi-product development | `synct-dev` | `ssh cc-hub "ssh synct-dev '...'"` |
| ENERGY-PROD | energy.synctacles.com | Energy API (production) | `energy-prod` | `ssh cc-hub "ssh energy-prod '...'"` |
| CARE-PROD | care.synctacles.com | Care/KB Support (production) | `brains` | `ssh cc-hub "ssh brains '...'"` |
| MONITOR | 77.42.41.135 | Prometheus & Grafana | - | `ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 '...'"` |

### Databases per Server
| Server | Databases |
|--------|-----------|
| DEV | `energy_dev`, `care_dev`, `auth_dev` |
| ENERGY-PROD | `energy_prod`, `auth_prod` |
| CARE-PROD | `care_prod` |

**Note:** As of 2026-02-05, new naming convention applied. Old names (synct-dev, synct-prod, brains) are deprecated.

### GitHub Account
- **Bot account**: `synctacles-bot`
- **Repository**: `synctacles/platform`
- **Authentication**: PAT token (configured in gh CLI)

### Product Repositories
| Product | Repository | Server | Description |
|---------|------------|--------|-------------|
| **Energy** | `synctacles/energy` | ENERGY-PROD | Price API, collectors, HA integration |
| **Care** | `synctacles/care` | CARE-PROD | KB Support bot, harvesters |
| **Platform** | `synctacles/platform` | All | Auth service, shared libs, IaC |
| **HA Integration** | `synctacles/ha-integration` | - | Home Assistant addon (client-side) |

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

### CARE-PROD Server (KB Support & Harvesters)

> **Migration Note (2026-02-05):** Server renamed from "BRAINS" to "CARE-PROD". Domain changing from brains.synctacles.com to care.synctacles.com.

**Server Details:**
- **Hostname:** care.synctacles.com (DNS pending, currently brains.synctacles.com)
- **IP:** 173.249.55.109
- **SSH Alias:** `brains` (via cc-hub: `ssh cc-hub "ssh brains '...'"`)
- **User:** `brains` (system user)
- **Database:** `brains_kb` (KB schema: `kb.*`)
- **Purpose:** Knowledge Base support bot + KB harvesters + AI inference

**Architecture (2026-02-05 - PRODUCTION):**
The Care product runs on CARE-PROD as a single production environment:
- **Support Bot:** Python Telegram bot for HA community support (@SynctaclesSupportBot)
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
# Support Bot (Telegram)
systemctl status care-prod-support

# KB Harvester (oneshot, runs hourly via timer)
systemctl status care-prod-harvest

# Harvester Timer
systemctl status care-prod-harvest.timer

# KB Database (PostgreSQL)
systemctl status postgresql

# Ollama (Local LLM inference)
systemctl status ollama

# Prometheus Node Exporter (monitoring)
systemctl status node_exporter
```

**Service Status Check:**
```bash
ssh cc-hub 'ssh brains "systemctl is-active postgresql ollama care-prod-support care-prod-harvest.timer node_exporter"'
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
- **Environment:** `/opt/care-prod/.env` (chmod 600, contains all secrets)
- **Python Code:** `/opt/care-prod/harvesters/` (support_agent, tools/scanners, shared)
- **Virtual Environment:** `/opt/care-prod/venv/`
- **MCP Server:** `/opt/care-prod/mcp/kb-search.js` (Node.js)
- **Logs:** `/var/log/care-prod/` (harvest.log)
- **Setup Scripts:** `/opt/github/synctacles-platform/scripts/care-setup/`

> **Note:** Directory migration from `/opt/openclaw/` to `/opt/care-prod/` pending.

**Version Control (2026-02-04):**
- **Git Repo:** `/opt/openclaw/harvesters/.git` (initialized 2026-02-04)
- **Current Branch:** `master`
- **Latest Commit:** `cab15d5` (docs: add comprehensive README with dev workflow)
- **Pre-commit Hook:** Python syntax validation (blocks commits with syntax errors)
- **Workflow:** Feature branches → test → merge to master → rollback if needed
- **README:** `/opt/openclaw/harvesters/README.md` (dev workflow, rollback procedures)
- **Note:** Always commit working states before making changes for easy rollback

**Database:**
- **Name:** `care_prod` (was: `brains_kb`)
- **Schemas:**
  - `kb` (knowledge_base, knowledge_base_categories, knowledge_base_feedback, knowledge_base_usage)
  - `public` (harvest_state for tracking scanner progress)
- **Admin User:** `care` (was: `brains_admin`)
- **Connection:** `postgresql://care:***@localhost:5432/care_prod?sslmode=disable`
- **Search Path:** `kb, public` (set at database level)
- **Data:** 17,297 active KB entries, 24 categories, avg confidence 0.78

**Telegram Bot:**
- **Username:** @SynctaclesSupportBot
- **Token:** In `/opt/care-prod/.env` as `TELEGRAM_BOT_TOKEN_SUPPORT`
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
ssh cc-hub 'ssh brains "sudo systemctl restart care-prod-support"'

# View Support Bot logs
ssh cc-hub 'ssh brains "sudo journalctl -u care-prod-support -f"'

# Manually trigger harvest
ssh cc-hub 'ssh brains "sudo systemctl start care-prod-harvest"'

# View harvest logs
ssh cc-hub 'ssh brains "sudo journalctl -u care-prod-harvest -f"'

# Database access (as admin)
ssh cc-hub 'ssh brains "sudo -u postgres psql -d care_prod"'

# Check KB statistics
ssh cc-hub 'ssh brains "sudo -u postgres psql -d care_prod -c \"SELECT COUNT(*) FROM kb.knowledge_base WHERE is_active = true;\""'

# Test Ollama models
ssh cc-hub 'ssh brains "ollama list"'

# Check all service status
ssh cc-hub 'ssh brains "systemctl status care-prod-support care-prod-harvest.timer postgresql ollama --no-pager"'
```

**Troubleshooting:**
```bash
# Check harvest state
ssh cc-hub 'ssh brains "sudo -u postgres psql -d care_prod -c \"SELECT * FROM public.harvest_state;\""'

# Test Telegram bot token
ssh cc-hub 'ssh brains "curl -s https://api.telegram.org/bot\$(grep TELEGRAM_BOT_TOKEN_SUPPORT /opt/care-prod/.env | cut -d= -f2)/getMe | jq"'

# Verify database permissions
ssh cc-hub 'ssh brains "sudo -u postgres psql -d care_prod -c \"\\du care\""'

# Check service resource usage
ssh cc-hub 'ssh brains "systemctl status care-prod-support --no-pager | grep -E '\''Memory|CPU'\''"'
```

**SSH Key:**
The public key for cc-hub→care-prod access:
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPpL62iQAPP12ih0TZaXlMAH31cYdV6a9ZHaO+GF0Iie ccops@cc-hub->care-prod
```
This must be added to `/home/care/.ssh/authorized_keys` on the care-prod server (as user `care`, NOT root).

## CI/CD Pipeline
GitHub Actions runs on every push:
- Ruff linting & formatting
- Pytest (unit + integration tests)
- Auth service validation
- Shared lib compatibility checks

## Related Repos
- **Energy:** https://github.com/synctacles/energy
- **Care:** https://github.com/synctacles/care
- **Platform:** https://github.com/synctacles/platform
- **HA Integration:** https://github.com/synctacles/ha-integration

## Migration Status (2026-02-05)

### Infrastructure Overhaul Project
GitHub Project: [DEV Infrastructure Overhaul](https://github.com/orgs/synctacles/projects/6)

**Naming Convention Migration:**
- ✅ Naming convention documented ([docs/NAMING_CONVENTIONS.md](docs/NAMING_CONVENTIONS.md))
- 🚧 DEV server migration (Energy/Care/Auth databases)
- 🚧 ENERGY-PROD server migration (was: PROD)
- 🚧 CARE-PROD server migration (was: BRAINS)

**Server Renames:**
| Old | New | Status |
|-----|-----|--------|
| synct-dev | dev | 🚧 Pending |
| synct-prod | energy-prod | 🚧 Pending |
| brains | care-prod | 🚧 Pending |

**Domain Changes:**
| Old | New | Status |
|-----|-----|--------|
| api.synctacles.com | energy.synctacles.com | ✅ DNS Ready |
| brains.synctacles.com | care.synctacles.com | ✅ DNS Ready |

**Architecture (2026-02-05):**
- **ENERGY-PROD:** Energy API + Auth API (production)
- **CARE-PROD:** Care API + Support Bot (production)
- **DEV:** All products for development

**Note:** Most active development happens in product repos. This repo focuses on:
- Cross-product infrastructure
- Authentication services
- Shared libraries
- Server setup scripts
