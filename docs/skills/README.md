# SYNCTACLES Skills - Master Index

Technical documentation for SYNCTACLES products, infrastructure, and development protocols.

**Last Updated:** 2026-01-26

---

## Quick Start for AI

**MANDATORY READS (in order):**
1. [core/SKILL_00_AI_PROTOCOL.md](core/SKILL_00_AI_PROTOCOL.md) - Operating protocol (VERPLICHT)
2. [core/SKILL_01_HARD_RULES.md](core/SKILL_01_HARD_RULES.md) - Non-negotiable rules

**For Infrastructure Work:**
- [infrastructure/SKILL_01_DEPLOYMENT.md](infrastructure/SKILL_01_DEPLOYMENT.md)
- [infrastructure/SKILL_02_REPOS_ACCOUNTS.md](infrastructure/SKILL_02_REPOS_ACCOUNTS.md)

**For Energy Product:**
- [energy/SKILL_00_ENERGY_OVERVIEW.md](energy/SKILL_00_ENERGY_OVERVIEW.md)
- [energy/SKILL_01_ARCHITECTURE.md](energy/SKILL_01_ARCHITECTURE.md)

**For Care Product:**
- [care/SKILL_00_CARE_OVERVIEW.md](care/SKILL_00_CARE_OVERVIEW.md)
- [care/SKILL_01_CARE_SAFETY.md](care/SKILL_01_CARE_SAFETY.md)

---

## Product Overview

### SYNCTACLES Energy
**Type:** Price data API + Home Assistant integration
**Purpose:** Real-time electricity prices & smart energy recommendations
**Role:** Acquisition funnel (gratis/trial)
**Repos:** `synctacles-api` (API), `ha-integration` (HA component)

**Key Features:**
- 6-tier fallback stack for price data
- GO/WAIT/AVOID energy action recommendations
- 24 Dutch provider integrations (Enever BYO-key)
- Live cost calculation from P1 meters

[→ Energy Skills](energy/)

### SYNCTACLES Care
**Type:** Home Assistant add-on
**Purpose:** Database maintenance & security audit
**Role:** Kernproduct (€25/jaar)
**Repo:** `addon-synctacles-care`

**Key Features:**
- Health scan + A-F grading
- Security audit + 0-100 score
- Orphan detection & cleanup (Premium)
- P0-P2 safety mitigations

[→ Care Skills](care/)

---

## Skill Categories

### 📋 [Core Skills](core/)
Fundamental AI protocols and development standards.

| # | Skill | Description |
|---|-------|-------------|
| 00 | [AI Protocol](core/SKILL_00_AI_PROTOCOL.md) | Operating protocol (VERPLICHT) |
| 01 | [Hard Rules](core/SKILL_01_HARD_RULES.md) | Non-negotiable development rules |
| 02 | [Coding Standards](core/SKILL_02_CODING_STANDARDS.md) | Code style, testing, docs |
| 03 | [Communication](core/SKILL_03_COMMUNICATION.md) | User communication rules |
| 04 | [Development](core/SKILL_04_DEVELOPMENT.md) | Development workflow |

### 🏗️ [Infrastructure Skills](infrastructure/)
Server management, deployment, and operations.

| # | Skill | Description |
|---|-------|-------------|
| 00 | [Hardware](infrastructure/SKILL_00_HARDWARE.md) | Server hardware profile |
| 01 | [Deployment](infrastructure/SKILL_01_DEPLOYMENT.md) | Deployment workflow |
| 02 | [Repos & Accounts](infrastructure/SKILL_02_REPOS_ACCOUNTS.md) | Git repos & bot accounts |
| 03 | [Brand-Free](infrastructure/SKILL_03_BRAND_FREE.md) | Brand-free architecture |
| 04 | [Logging](infrastructure/SKILL_04_LOGGING.md) | Logging & diagnostics |
| 05 | [Monitoring](infrastructure/SKILL_05_MONITORING.md) | System monitoring |
| 06 | [Backup](infrastructure/SKILL_06_BACKUP.md) | Backup & recovery |
| 07 | [Installer](infrastructure/SKILL_07_INSTALLER.md) | Installer scripts |

### ⚡ [Energy Skills](energy/)
Price data API, HA integration, smart recommendations.

| # | Skill | Description |
|---|-------|-------------|
| 00 | [Overview](energy/SKILL_00_ENERGY_OVERVIEW.md) | Product overview & features |
| 01 | [Architecture](energy/SKILL_01_ARCHITECTURE.md) | API architecture, fallback stack |
| 02 | [Product](energy/SKILL_02_PRODUCT.md) | Product requirements |
| 03 | [Data Sources](energy/SKILL_03_DATA_SOURCES.md) | Data providers & APIs |
| 04 | [Price Engine](energy/SKILL_04_PRICE_ENGINE.md) | Consumer price calculation |

### 🛡️ [Care Skills](care/)
Database maintenance, security audit, cleanup.

| # | Skill | Description |
|---|-------|-------------|
| 00 | [Overview](care/SKILL_00_CARE_OVERVIEW.md) | Product overview & tier matrix |
| 01 | [Safety](care/SKILL_01_CARE_SAFETY.md) | P0-P2 safety mitigations |
| 02 | [Architecture](care/SKILL_02_CARE_ARCHITECTURE.md) | Code structure, data models |
| 03 | [Testing](care/SKILL_03_CARE_TESTING.md) | Unit tests, integration tests, CI |

### 💼 [Business Skills](business/)
Go-to-market, pricing, user context.

| # | Skill | Description |
|---|-------|-------------|
| 00 | [Go-to-Market](business/SKILL_00_GO_TO_MARKET.md) | Pricing, tiers, user journeys |
| 01 | [Personal Profile](business/SKILL_01_PERSONAL_PROFILE.md) | User profile & preferences |

---

## Repository Map

| Repository | Location | Purpose |
|------------|----------|---------|
| `synctacles-api` | `/opt/github/synctacles-api` | Energy API server (FastAPI) |
| `ha-integration` | `/opt/github/ha-integration` | Energy HA component |
| `addon-synctacles-care` | `/opt/github/addon-synctacles-care` | Care HA add-on |

---

## Server Environments

| Environment | Host | Access | Purpose |
|-------------|------|--------|---------|
| **DEV** | This machine | Direct | Development (synct-dev) |
| **PROD** | 46.62.212.227 | Via cc-hub | Production API |
| **HA DEV** | 82.169.33.175:22222 | SSH | Care integration tests |

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                         SYNCTACLES                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────┐     ┌──────────────────────────────┐  │
│  │   Energy API Server  │     │    Care Add-on               │  │
│  │   (synctacles-api)   │     │  (addon-synctacles-care)     │  │
│  │                      │     │                              │  │
│  │  - Price data        │     │  - Health scanner            │  │
│  │  - Fallback stack    │     │  - Security audit            │  │
│  │  - PostgreSQL        │     │  - Orphan cleanup (Premium)  │  │
│  │  - FastAPI           │     │  - Docker container          │  │
│  └──────────┬───────────┘     └───────────┬──────────────────┘  │
│             │                              │                     │
│             │ REST API                     │ Local files         │
│             ▼                              ▼                     │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │            Home Assistant Installation                      ││
│  │                                                             ││
│  │  ┌────────────────────┐    ┌──────────────────────────┐   ││
│  │  │ Energy Integration │    │   Care Add-on            │   ││
│  │  │ (custom_component) │    │   (container)            │   ││
│  │  │                    │    │                          │   ││
│  │  │ - 10+ sensors      │    │ - /config access         │   ││
│  │  │ - Enever BYO       │    │ - DB analysis            │   ││
│  │  │ - Coordinators     │    │ - Web UI (ingress)       │   ││
│  │  └────────────────────┘    └──────────────────────────┘   ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

---

## Key Principles

### Development
- **Brand-Free:** All scripts use environment variables from `/opt/.env`
- **Safety First:** P0-P2 mitigations for database operations
- **Test Coverage:** Unit + integration tests for all features
- **CI/CD:** GitHub Actions + pre-push/pre-commit hooks

### Products
- **Energy:** Acquisition funnel (gratis → trial → premium)
- **Care:** Kernproduct (€25/jaar) - converts users
- **Bundle:** Care + Energy together = premium experience

### Infrastructure
- **DEV:** Development machine (direct access)
- **PROD:** Production server (via SSH tunnel)
- **Deploy:** `~/bin/deploy-prod` (checks CI before deploy)

---

## For Claude.ai (CAI)

When reading these skills:
1. Start with **core/SKILL_00_AI_PROTOCOL.md** (mandatory)
2. Read **core/SKILL_01_HARD_RULES.md** (non-negotiable)
3. Choose product-specific skills based on task:
   - Energy work → [energy/](energy/)
   - Care work → [care/](care/)
   - Infrastructure → [infrastructure/](infrastructure/)
4. Reference **business/** for pricing/GTM context

All skills are written for you to understand the system completely.

---

*Skills maintained by: synctacles-dev*
*AI Operating Protocol Version: See core/SKILL_00_AI_PROTOCOL.md*
