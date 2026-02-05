# SYNCTACLES Platform - Master Architecture

**Last Updated:** 2026-02-05
**Version:** 2.0 (Microservices with Centralized Auth)
**Scope:** Platform-wide architecture (all products)

> **Note:** Product-specific details belong in product repos:
> - Energy: `synctacles/energy/docs/ARCHITECTURE.md`
> - CARE: `synctacles/care/docs/ARCHITECTURE.md`
> - Brains: `synctacles/brains/docs/ARCHITECTURE.md`

---

## Overview

SYNCTACLES is a multi-product SaaS platform for Home Assistant users. This document defines the high-level architecture that ALL products must follow.

## Architecture Principles

1. **Single Source of Truth** - User data lives in Platform API only
2. **Microservices** - Each product is an independent service
3. **Centralized Authentication** - Platform API handles all auth
4. **Stateless Services** - Products store only product-specific data
5. **Event-Driven** - Products communicate via events/webhooks
6. **API-First** - All services expose REST/GraphQL APIs

## System Architecture Diagram

See [PLATFORM_ARCHITECTURE.md](docs/PLATFORM_ARCHITECTURE.md) for full diagram.

**Core Components:**
- **Platform API** (synct-prod): Auth, users, subscriptions
- **Product APIs** (synct-prod): Energy, CARE, future products
- **BRAINS Server**: Knowledge Base (stateless service)
- **Analytics**: Usage tracking (all products)

## Database Ownership

**CRITICAL RULE:** Each database has ONE owner. No duplication of user data.

| Database | Owner | Contains | Has Users? |
|----------|-------|----------|------------|
| platform_db | Platform API | users, subscriptions, tokens | ✅ YES (only here!) |
| energy_db | Energy API | prices, forecasts | ❌ NO (FK only) |
| care_db | CARE API | telegram_links, preferences | ❌ NO (FK only) |
| brains_kb | KB Service | knowledge_base | ❌ NO (stateless) |
| analytics_db | Analytics | query_logs | ❌ NO (FK only) |

## Authentication Flow

All products authenticate users via Platform API:

1. User logs in → Platform API returns JWT
2. Product API validates JWT → Platform API
3. Service-to-service uses API tokens

See full flow in [PLATFORM_ARCHITECTURE.md](docs/PLATFORM_ARCHITECTURE.md#authentication-flow).

## Adding New Products

When creating a new product:
1. NO user tables! Reference platform_db.users(id) via FK
2. Validate JWT/tokens via Platform API
3. Check premium tier via Platform API
4. Document product-specific architecture in product repo

## Product Documentation

- **Platform:** This repo - [docs/PLATFORM_ARCHITECTURE.md](docs/PLATFORM_ARCHITECTURE.md)
- **Energy:** [synctacles/energy](https://github.com/synctacles/energy) *(to be created)*
- **CARE:** [synctacles/care](https://github.com/synctacles/care) *(to be created)*
- **Brains:** [synctacles/brains](https://github.com/synctacles/brains) *(to be created)*
