# Naming Convention & Infrastructure Standards

**Version:** 1.0
**Date:** 2026-02-05
**Status:** Approved
**Applies to:** All Synctacles products and infrastructure

---

## Overview

This document defines the official naming conventions for all Synctacles infrastructure components including servers, services, databases, directories, and APIs.

**Core Principle:** Brand-free, product-focused naming.

- No "SYNCTACLES" branding in service names, files, or daemons
- Only: `{product}-{environment}-{task}`

---

## Products

| Product | Code | Description |
|---------|------|-------------|
| **Energy** | `energy` | Energy price API, collectors, Home Assistant integration |
| **Care** | `care` | Knowledge Base, Support Bot, Community assistance |
| **Auth** | `auth` | Centralized authentication service |

---

## Environments

| Environment | Code | Purpose |
|-------------|------|---------|
| **Development** | `dev` | All products, testing, development |
| **Production** | `prod` | Single product per server |

---

## Servers

| Server | Hostname | IP | Products | Databases |
|--------|----------|-----|----------|-----------|
| **DEV** | dev.synctacles.com | 135.181.255.83 | Energy, Care, Auth | `energy_dev`, `care_dev`, `auth_dev` |
| **ENERGY-PROD** | energy.synctacles.com | 46.62.212.227 | Energy, Auth | `energy_prod`, `auth_prod` |
| **CARE-PROD** | care.synctacles.com | 173.249.55.109 | Care | `care_prod` |

### SSH Configuration (cc-hub)

```bash
# ~/.ssh/config

Host dev
  HostName 135.181.255.83
  User synctacles-dev
  IdentityFile ~/.ssh/id_dev

Host energy-prod
  HostName 46.62.212.227
  User energy
  IdentityFile ~/.ssh/id_energy_prod

Host care-prod
  HostName 173.249.55.109
  User care
  IdentityFile ~/.ssh/id_care_prod
```

---

## Systemd Services

### Naming Pattern

```
{product}-{environment}-{task}.service
{product}-{environment}-{task}.timer
```

### Energy Services

| DEV | PROD | Description |
|-----|------|-------------|
| `energy-dev-api.service` | `energy-prod-api.service` | FastAPI server (Gunicorn) |
| `energy-dev-collector.service` | `energy-prod-collector.service` | ENTSO-E data collector |
| `energy-dev-collector.timer` | `energy-prod-collector.timer` | Collector schedule (15 min) |
| `energy-dev-frank.service` | `energy-prod-frank.service` | Frank Energie price collector |
| `energy-dev-frank.timer` | `energy-prod-frank.timer` | Frank schedule (07:00, 15:00 UTC) |
| `energy-dev-importer.service` | `energy-prod-importer.service` | Raw data importer |
| `energy-dev-importer.timer` | `energy-prod-importer.timer` | Importer schedule (15 min) |
| `energy-dev-normalizer.service` | `energy-prod-normalizer.service` | Data normalizer |
| `energy-dev-normalizer.timer` | `energy-prod-normalizer.timer` | Normalizer schedule (15 min) |
| `energy-dev-health.service` | `energy-prod-health.service` | Health check |
| `energy-dev-health.timer` | `energy-prod-health.timer` | Health schedule (5 min) |

### Care Services

| DEV | PROD | Description |
|-----|------|-------------|
| `care-dev-api.service` | `care-prod-api.service` | Care API server |
| `care-dev-support.service` | `care-prod-support.service` | Telegram support bot |
| `care-dev-harvest.service` | `care-prod-harvest.service` | KB harvester |
| `care-dev-harvest.timer` | `care-prod-harvest.timer` | Harvest schedule (hourly) |
| `care-dev-update.service` | `care-prod-update.service` | Weekly software update |
| `care-dev-update.timer` | `care-prod-update.timer` | Update schedule (weekly) |

### Auth Services

| DEV | PROD | Description |
|-----|------|-------------|
| `auth-dev-api.service` | `auth-prod-api.service` | Authentication API |

---

## Databases

### Naming Pattern

```
{product}_{environment}
```

### Database Allocation

| Server | Database | Owner | Description |
|--------|----------|-------|-------------|
| **DEV** | `energy_dev` | `energy_dev` | Energy price data, ENTSO-E, Frank |
| **DEV** | `care_dev` | `care_dev` | KB test data |
| **DEV** | `auth_dev` | `auth_dev` | Users, sessions (test) |
| **ENERGY-PROD** | `energy_prod` | `energy` | Energy production data |
| **ENERGY-PROD** | `auth_prod` | `auth` | Users, sessions (production) |
| **CARE-PROD** | `care_prod` | `care` | Knowledge Base production |

### Energy Database Tables

```
energy_{env}
├── frank_prices          # Frank Energie hourly prices
├── norm_prices           # Normalized consumer prices
├── raw_entso_e_a44       # Raw ENTSO-E day-ahead prices
├── raw_entso_e_a65       # Raw ENTSO-E load data (discontinued)
├── raw_entso_e_a75       # Raw ENTSO-E generation data (discontinued)
├── hist_entso_prices     # Historical ENTSO-E prices
└── coefficient_*         # Price coefficient tables
```

### Care Database Tables

```
care_{env}
├── kb.knowledge_base           # KB entries
├── kb.knowledge_base_categories # KB categories
├── kb.knowledge_base_feedback  # User feedback
├── kb.knowledge_base_usage     # Usage statistics
└── public.harvest_state        # Harvester state tracking
```

### Auth Database Tables

```
auth_{env}
├── users                 # User accounts
├── sessions              # Active sessions
├── api_keys              # API keys
└── service_tokens        # Inter-service tokens
```

---

## Directories

### Naming Pattern

```
/opt/{product}-{environment}/
```

### Directory Structure

| Server | Directory | Purpose |
|--------|-----------|---------|
| **DEV** | `/opt/energy-dev/` | Energy venv, config |
| **DEV** | `/opt/care-dev/` | Care venv, config |
| **DEV** | `/opt/auth-dev/` | Auth venv, config |
| **ENERGY-PROD** | `/opt/energy-prod/` | Energy venv, config |
| **ENERGY-PROD** | `/opt/auth-prod/` | Auth venv, config |
| **CARE-PROD** | `/opt/care-prod/` | Care venv, config |

### Standard Directory Layout

```
/opt/{product}-{environment}/
├── venv/                 # Python virtual environment
├── .env                  # Environment configuration
├── logs/                 # Application logs (symlink to /var/log/)
└── data/                 # Local data cache (optional)

/opt/github/synctacles-{product}/
└── (git repository)      # Source code

/var/log/{product}-{environment}/
└── (log files)           # Systemd journal + app logs
```

---

## API Standards

### Port Allocation

| Port | Service | Notes |
|------|---------|-------|
| **8000** | Auth API | Authentication service |
| **8001** | Energy API | Main product API |
| **8002** | Care API | Only on DEV (multi-product) |

**Rule:** Each product's main API runs on port 8001 on its own production server.

### URL Structure

```
https://{product}.synctacles.com/v1/{resource}
```

### Energy API Endpoints

```
GET  /v1/prices/today           # Today's hourly prices
GET  /v1/prices/tomorrow        # Tomorrow's prices (after 14:00)
GET  /v1/prices/{date}          # Historical prices
GET  /v1/windows/cheapest       # Cheapest time windows
GET  /v1/health                 # Health check
GET  /v1/metrics                # Prometheus metrics
```

### Auth API Endpoints (on energy.synctacles.com)

```
POST /v1/auth/login             # User login
POST /v1/auth/refresh           # Token refresh
POST /v1/auth/logout            # User logout
GET  /v1/auth/me                # Current user info
POST /v1/auth/service-token     # Service-to-service token
```

### Care API Endpoints

```
GET  /v1/kb/search              # Search knowledge base
GET  /v1/kb/entry/{id}          # Get KB entry
POST /v1/kb/feedback            # Submit feedback
GET  /v1/health                 # Health check
GET  /v1/metrics                # Prometheus metrics
```

### API Versioning

- Always use `/v1/` prefix
- Increment version for breaking changes only
- Old versions supported for minimum 6 months after deprecation

---

## Python Packages

| Package | Repository | Description |
|---------|------------|-------------|
| `energy_api` | synctacles/energy | Energy API application |
| `care_api` | synctacles/care | Care API application |
| `auth_api` | synctacles/platform | Auth API application |
| `synctacles_shared` | synctacles/platform | Shared utilities |

**Deprecated:**
- `synctacles_db` - Replaced by `energy_api`

---

## Environment Files

### Naming Pattern

```
/opt/{product}-{environment}/.env
```

### Required Variables (Energy)

```bash
# Database
DATABASE_URL="postgresql://{user}@localhost:5432/{database}"
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="{product}_{environment}"
DB_USER="{product}_{environment}"

# Paths
INSTALL_PATH="/opt/{product}-{environment}"
APP_PATH="/opt/github/synctacles-{product}"
LOG_PATH="/var/log/{product}-{environment}"
VENV_PATH="/opt/{product}-{environment}/venv"

# API
API_HOST="0.0.0.0"
API_PORT="8001"
LOG_LEVEL="warning"

# Service
SERVICE_USER="{user}"
```

---

## Monitoring

### Prometheus Targets

```yaml
# prometheus.yml

scrape_configs:
  - job_name: 'energy-dev'
    static_configs:
      - targets: ['dev.synctacles.com:8001']
    metrics_path: /v1/metrics

  - job_name: 'energy-prod'
    static_configs:
      - targets: ['energy.synctacles.com:8001']
    metrics_path: /v1/metrics

  - job_name: 'care-prod'
    static_configs:
      - targets: ['care.synctacles.com:8001']
    metrics_path: /v1/metrics

  - job_name: 'auth-prod'
    static_configs:
      - targets: ['energy.synctacles.com:8000']
    metrics_path: /v1/metrics
```

---

## Migration Reference

### Old → New Mapping

| Component | Old Name | New Name |
|-----------|----------|----------|
| **Server** | synct-dev | dev |
| **Server** | synct-prod | energy-prod |
| **Server** | brains | care-prod |
| **Domain** | api.synctacles.com | energy.synctacles.com |
| **Domain** | brains.synctacles.com | care.synctacles.com |
| **Service** | synctacles-dev-api | energy-dev-api |
| **Service** | synctacles-prod-api | energy-prod-api |
| **Service** | openclaw-support | care-prod-support |
| **Database** | synctacles_dev | energy_dev |
| **Database** | synctacles | energy_prod |
| **Database** | brains_kb | care_prod |
| **Directory** | /opt/synctacles-dev/ | /opt/energy-dev/ |
| **Directory** | /opt/synctacles/ | /opt/energy-prod/ |
| **Directory** | /opt/openclaw/ | /opt/care-prod/ |
| **User** | synctacles | energy |
| **User** | brains | care |
| **Package** | synctacles_db | energy_api |

---

## Checklist for New Products

When adding a new product:

1. [ ] Choose product code (lowercase, single word)
2. [ ] Create GitHub repository: `synctacles/{product}`
3. [ ] Create Python package: `{product}_api`
4. [ ] Create DEV database: `{product}_dev`
5. [ ] Create DEV directory: `/opt/{product}-dev/`
6. [ ] Create DEV services: `{product}-dev-*.service`
7. [ ] Add to monitoring configuration
8. [ ] Update this document

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-05 | Claude Code | Initial version |
