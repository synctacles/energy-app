# SYNCTACLES Platform - Master Architecture

**Last Updated:** 2026-02-05
**Version:** 3.0 (Product-based naming, brand-free infrastructure)
**Scope:** Platform-wide architecture (all products)

> **Note:** Product-specific details belong in product repos:
> - Energy: `synctacles/energy/docs/ARCHITECTURE.md`
> - CARE: `synctacles/care/docs/ARCHITECTURE.md`
> - Brains: `synctacles/brains/docs/ARCHITECTURE.md`

---

## Overview

SYNCTACLES is a multi-product SaaS platform for Home Assistant users. This document defines the high-level architecture that ALL products must follow.

## Naming Conventions

See [NAMING_CONVENTIONS.md](NAMING_CONVENTIONS.md) for the complete naming standard.

**Key rules:**
- **Services:** `{product}-{environment}-{task}` (e.g., `energy-prod-api`)
- **Databases:** `{product}_{environment}` (e.g., `energy_prod`)
- **No brand names** in infrastructure (only in user-facing content)

## Architecture Principles

1. **Single Source of Truth** - User data lives in Platform API only
2. **Microservices** - Each product is an independent service
3. **Centralized Authentication** - Platform API handles all auth
4. **Stateless Services** - Products store only product-specific data
5. **Event-Driven** - Products communicate via events/webhooks
6. **API-First** - All services expose REST/GraphQL APIs
7. **Brand-Free Infrastructure** - No SYNCTACLES in service/file names

## Infrastructure

### Servers

| Server | Purpose | IP | SSH Alias |
|--------|---------|-----|-----------|
| **ENERGY-DEV** | Energy development | 135.181.255.83 | `synct-dev` |
| **ENERGY-PROD** | Production Energy API | 46.62.212.227 | `energy-prod` |
| **CARE-PROD** | Production Care/KB | 173.249.55.109 | `brains` |
| **MONITOR** | Prometheus/Grafana | 77.42.41.135 | via key |

### Databases

| Database | Server | Owner | Purpose |
|----------|--------|-------|---------|
| `energy_dev` | ENERGY-DEV | energy-dev | Dev price data |
| `energy_prod` | ENERGY-PROD | energy-prod | Prod price data |
| `brains_kb` | CARE-PROD | care-prod | Knowledge Base |

### Public Endpoints

| Domain | Server | Product |
|--------|--------|---------|
| `energy.synctacles.com` | ENERGY-PROD | Energy API |
| `dev.synctacles.com` | ENERGY-DEV | Dev Energy API |
| `care.synctacles.com` | CARE-PROD | Care API (future) |

## System Architecture Diagram

```
                    ┌─────────────────┐
                    │   Cloudflare    │
                    │  (DNS/CDN/SSL)  │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  ENERGY-PROD    │ │  ENERGY-DEV     │ │  CARE-PROD      │
│  46.62.212.227  │ │  135.181.255.83 │ │  173.249.55.109 │
│                 │ │                 │ │                 │
│ energy-prod-api │ │ energy-dev-api  │ │ care-prod-*     │
│ energy-prod-*   │ │ energy-dev-*    │ │                 │
│                 │ │                 │ │ PostgreSQL+KB   │
│ PostgreSQL      │ │ PostgreSQL      │ │ Ollama          │
└─────────────────┘ └─────────────────┘ └─────────────────┘
         │                   │                   │
         └───────────────────┼───────────────────┘
                             │
                    ┌────────▼────────┐
                    │    MONITOR      │
                    │  77.42.41.135   │
                    │                 │
                    │ Prometheus      │
                    │ Grafana         │
                    │ Alertmanager    │
                    └─────────────────┘
```

## Database Ownership

**CRITICAL RULE:** Each database has ONE owner. No duplication of user data.

| Database | Server | Owner | Contains | Has Users? |
|----------|--------|-------|----------|------------|
| `auth_prod` | ENERGY-PROD | auth-prod | users, subscriptions, tokens | ✅ YES (only here!) |
| `energy_prod` | ENERGY-PROD | energy-prod | prices, forecasts | ❌ NO (FK only) |
| `energy_dev` | ENERGY-DEV | energy-dev | dev prices | ❌ NO (FK only) |
| `brains_kb` | CARE-PROD | care-prod | knowledge_base | ❌ NO (stateless) |

## Authentication Flow

All products authenticate users via Platform API:

1. User logs in → Platform API returns JWT
2. Product API validates JWT → Platform API
3. Service-to-service uses API tokens

## Git Workflow

**Development → Production deployment:**

1. **DEV server** has push access (write deploy key)
2. **PROD servers** have pull-only access (read-only deploy keys)
3. Code changes are made and tested on DEV
4. Changes are pushed to GitHub from DEV
5. PROD servers pull changes via `git pull`

```bash
# On DEV: push changes
ssh cc-hub "ssh synct-dev 'cd /opt/github/synctacles-energy && git push origin main'"

# On PROD: pull changes
ssh cc-hub "ssh energy-prod 'cd /opt/github/synctacles-energy && git pull origin main'"
```

## Adding New Products

When creating a new product:
1. Follow naming convention: `{product}-{env}-{task}`
2. NO user tables! Reference auth database via FK
3. Validate JWT/tokens via Platform API
4. Check premium tier via Platform API
5. Document product-specific architecture in product repo
6. Set up separate databases per environment

## Product Documentation

- **Platform:** This repo - [docs/PLATFORM_ARCHITECTURE.md](PLATFORM_ARCHITECTURE.md)
- **Energy:** [synctacles/energy](https://github.com/synctacles/energy)
- **CARE:** [synctacles/care](https://github.com/synctacles/care) *(planned)*
- **Brains:** [synctacles/brains](https://github.com/synctacles/brains) *(planned)*

## Monitoring

All servers are monitored via Prometheus (MONITOR server):

| Job Name | Target | Metrics |
|----------|--------|---------|
| `energy-prod-node` | 46.62.212.227:9100 | System metrics |
| `energy-prod-api` | energy.synctacles.com:443/metrics | API metrics |
| `energy-dev-node` | 135.181.255.83:9100 | System metrics |
| `care-prod-node` | 173.249.55.109:9100 | System metrics |

See [NAMING_CONVENTIONS.md](NAMING_CONVENTIONS.md) for complete naming rules.
