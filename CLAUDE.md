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
| DEV | Development | Direct (`/opt/github/synctacles-api`) |
| PROD | Production | Via `ssh cc-hub "ssh synct-prod '...'"` |

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

## Migration Status (2026-02-03)

This repository was renamed from `synctacles/backend` to `synctacles/platform` as part of the multi-repo migration:
- ✅ Energy code extracted to `synctacles/energy`
- ✅ Platform/Auth code remains here
- ✅ Shared libraries organized
- 🚧 Care extraction (planned)
- 🚧 Brains extraction (planned)

**Note:** Most active development happens in product repos. This repo focuses on:
- Cross-product infrastructure
- Authentication services
- Shared libraries
