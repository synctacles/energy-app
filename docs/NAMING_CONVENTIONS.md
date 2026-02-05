# Synctacles Naming Conventions

**Last Updated:** 2026-02-05
**Version:** 1.0

This document defines the official naming conventions for all Synctacles infrastructure.

---

## Core Principle: Brand-Free Infrastructure

**SYNCTACLES** is a brand name for users. Infrastructure uses **product names** only.

| Context | Use Brand? | Example |
|---------|------------|---------|
| User-facing content | Yes | "Welcome to SYNCTACLES Energy" |
| Service names | No | `energy-prod-api` (not `synctacles-energy-api`) |
| Database names | No | `energy_prod` (not `synctacles`) |
| File names | No | `energy.conf` (not `synctacles.conf`) |
| Domains | Yes | `energy.synctacles.com` |

---

## Service Naming

### Format

```
{product}-{environment}-{task}
```

### Components

| Component | Values | Description |
|-----------|--------|-------------|
| `{product}` | `energy`, `care`, `auth`, `platform` | The product/service |
| `{environment}` | `dev`, `prod` | Environment |
| `{task}` | `api`, `collector`, `importer`, `normalizer`, `health`, `support`, `harvest` | What the service does |

### Examples

| Service | Description |
|---------|-------------|
| `energy-prod-api` | Production Energy API |
| `energy-prod-collector` | Production price data collector |
| `energy-prod-importer` | Production data importer |
| `energy-prod-normalizer` | Production data normalizer |
| `energy-prod-health` | Production health checker |
| `energy-prod-frank-collector` | Production Frank Energie collector |
| `energy-dev-api` | Development Energy API |
| `care-prod-support` | Production Care support bot |
| `care-prod-harvest` | Production KB harvester |
| `care-prod-update` | Production weekly KB update |

### Systemd Files

```
/etc/systemd/system/{product}-{env}-{task}.service
/etc/systemd/system/{product}-{env}-{task}.timer
```

---

## Database Naming

### Format

```
{product}_{environment}
```

### Examples

| Database | Server | Purpose |
|----------|--------|---------|
| `energy_dev` | ENERGY-DEV | Development price data |
| `energy_prod` | ENERGY-PROD | Production price data |
| `auth_prod` | ENERGY-PROD | User authentication |
| `brains_kb` | CARE-PROD | Knowledge Base |

### Database Users

```
{product}_{environment}  (e.g., energy_dev, energy_prod)
```

---

## Server Naming

### Format

```
{PRODUCT}-{ENVIRONMENT}
```

### Current Servers

| Name | IP | Purpose |
|------|-----|---------|
| **ENERGY-DEV** | 135.181.255.83 | Energy development |
| **ENERGY-PROD** | 46.62.212.227 | Energy production |
| **CARE-PROD** | 173.249.55.109 | Care/KB production |
| **MONITOR** | 77.42.41.135 | Prometheus/Grafana |

### SSH Aliases (in cc-hub ~/.ssh/config)

| Alias | Server |
|-------|--------|
| `synct-dev` | ENERGY-DEV |
| `energy-prod` | ENERGY-PROD |
| `brains` | CARE-PROD |

---

## Prometheus Job Naming

### Format

```
{product}-{environment}-{metric_type}
```

### Examples

| Job Name | Target | Description |
|----------|--------|-------------|
| `energy-prod-node` | 46.62.212.227:9100 | ENERGY-PROD system metrics |
| `energy-prod-api` | energy.synctacles.com:443/metrics | ENERGY-PROD API metrics |
| `energy-prod-pipeline` | energy.synctacles.com:443/v1/pipeline/metrics | ENERGY-PROD pipeline metrics |
| `energy-dev-node` | 135.181.255.83:9100 | ENERGY-DEV system metrics |
| `energy-dev-api` | dev.synctacles.com:443/metrics | ENERGY-DEV API metrics |
| `care-prod-node` | 173.249.55.109:9100 | CARE-PROD system metrics |

---

## Domain Naming

### Format

```
{product}.synctacles.com       (production)
dev.synctacles.com             (development)
```

### Current Domains

| Domain | Server | Product |
|--------|--------|---------|
| `energy.synctacles.com` | ENERGY-PROD | Energy API |
| `dev.synctacles.com` | ENERGY-DEV | Dev Energy API |
| `api.synctacles.com` | ENERGY-PROD | Legacy (redirects to energy) |
| `care.synctacles.com` | CARE-PROD | Care API (future) |

---

## Environment Variables

### Standard Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | Database connection | `postgresql://energy_prod@localhost/energy_prod` |
| `API_HOST` | API bind address | `0.0.0.0` |
| `API_PORT` | API port | `8001` |
| `LOG_LEVEL` | Logging level | `warning` |
| `BRAND_NAME` | Display brand name | `SYNCTACLES` |
| `BRAND_SLUG` | URL-safe brand | `synctacles` |

### Environment File Location

```
/opt/{product}/{environment}/.env
```

Or for development:
```
/opt/github/synctacles-{product}/.env
```

---

## Git Repository Naming

### Format

```
synctacles/{product}
```

### Repositories

| Repo | Purpose |
|------|---------|
| `synctacles/platform` | Auth service, shared libraries, infrastructure |
| `synctacles/energy` | Energy API, price collectors |
| `synctacles/care` | Care/support bot (planned) |
| `synctacles/brains` | AI/ML services (planned) |
| `synctacles/ha-integration` | Home Assistant addon |

---

## Migration Reference

### Old → New Mapping

| Old Name | New Name |
|----------|----------|
| `synctacles-energy.service` | `energy-prod-api.service` |
| `synctacles-energy-collector.timer` | `energy-prod-collector.timer` |
| `synctacles_dev` (database) | `energy_dev` |
| `synctacles` (database) | `energy_prod` |
| `synct-prod` (server) | `ENERGY-PROD` |
| `brains` (server) | `CARE-PROD` |
| `openclaw-support.service` | `care-prod-support.service` |
| `openclaw-harvest.timer` | `care-prod-harvest.timer` |

---

## Quick Reference Card

```
SERVICE:   {product}-{env}-{task}          → energy-prod-api
DATABASE:  {product}_{env}                  → energy_prod
SERVER:    {PRODUCT}-{ENV}                  → ENERGY-PROD
DOMAIN:    {product}.synctacles.com         → energy.synctacles.com
PROM JOB:  {product}-{env}-{type}           → energy-prod-node
```
