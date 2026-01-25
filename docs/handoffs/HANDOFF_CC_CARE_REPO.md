# HANDOFF: SYNCTACLES Care Repository Setup

**Van:** CAI (Claude AI)  
**Naar:** CC (Claude Code)  
**Datum:** 2026-01-25  
**Prioriteit:** Hoog

---

## TL;DR

Maak een HA Add-on repository voor SYNCTACLES Care. Dit is een maintenance/security tool die samen met de bestaande Energy integration het SYNCTACLES Premium bundle (€25/jaar) vormt.

---

## Business Context

### Product Model

```
SYNCTACLES Premium (€25/jaar)
├── Care Add-on (global) = revenue driver
│   ├── Health scan + score → GRATIS
│   ├── Security scan + score → GRATIS
│   ├── Cleanup uitvoeren → PREMIUM
│   └── Scheduled maintenance → PREMIUM
│
└── Energy Integration (NL/EU) = acquisition
    ├── Prijzen + uren → GRATIS
    ├── Actions + Best Window → TRIAL (14d) / PREMIUM
    └── EU landen → PREMIUM
```

### User Funnel

**NL:**
```
Gratis Energy → Care scan → Ziet "247 orphans" → Trial → Premium
```

**EU:**
```
Care scan → Ziet probleem → Premium (+ Energy bonus)
```

---

## Opdracht

### Fase 1: Repository Setup

**Beslissingen (jij bepaalt):**

1. **Repo structuur:**
   - Optie A: `synctacles/ha-addons` (mono-repo)
   - Optie B: `synctacles/addon-synctacles-care` (dedicated)
   - Optie C: Anders

2. **Naamgeving:**
   - Repository naam
   - Add-on directory naam
   - Add-on slug
   - Display naam

**Aanmaken:**

- [ ] Repository met gekozen structuur
- [ ] README.md met project overview
- [ ] LICENSE (MIT)
- [ ] repository.yaml (indien mono-repo)
- [ ] .github/ISSUE_TEMPLATE/ (bug, feature, epic)
- [ ] .github/PULL_REQUEST_TEMPLATE.md
- [ ] .github/workflows/ (build, test, release skeletons)

### Fase 2: Labels

```yaml
# Type
- "type:bug" (#d73a4a)
- "type:feature" (#a2eeef)
- "type:docs" (#0075ca)

# Epic
- "epic:setup" (#7057ff)
- "epic:backend" (#7057ff)
- "epic:health" (#7057ff)
- "epic:security" (#7057ff)
- "epic:cleanup" (#7057ff)
- "epic:backup" (#7057ff)
- "epic:ui" (#7057ff)
- "epic:premium" (#7057ff)
- "epic:energy" (#7057ff)
- "epic:testing" (#7057ff)

# Priority
- "priority:critical" (#b60205)
- "priority:high" (#d93f0b)
- "priority:medium" (#fbca04)
- "priority:low" (#0e8a16)

# Tier
- "tier:free" (#c5def5)
- "tier:trial" (#bfdadc)
- "tier:premium" (#fef2c0)
```

### Fase 3: Milestones

| Milestone | Due Date |
|-----------|----------|
| V1.0 MVP | 2026-03-01 |
| V1.1 Polish | 2026-03-15 |
| V1.2 Automation | 2026-04-05 |
| V2.0 Advanced | 2026-06-01 |

### Fase 4: Issues

Maak alle V1.0 issues aan. Zie `SYNCTACLES_CARE_ROADMAP_V2.md` voor volledige lijst.

**Quick reference - Epics:**

| Epic | # Issues |
|------|----------|
| Setup | 8 |
| Backend | 10 |
| Health | 7 |
| Security | 8 |
| Cleanup | 10 |
| Backup | 6 |
| UI | 10 |
| Premium | 7 |
| Energy | 4 |
| Testing | 7 |
| **Totaal** | **77** |

---

## Technical Specs

### Add-on config.yaml Template

```yaml
name: "SYNCTACLES Care"
version: "0.0.1"
slug: synctacles_care
description: "Database maintenance & security audit for Home Assistant"
url: "https://synctacles.com/care"
arch:
  - aarch64
  - amd64
  - armhf
  - armv7
  - i386

# Permissions
map:
  - type: homeassistant_config
    read_only: false
  - type: backup
  - type: ssl

homeassistant_api: true
hassio_api: true

# UI
ingress: true
ingress_port: 8099
panel_icon: "mdi:database-cog"
panel_title: "Care"

# Options
options:
  api_key: ""
schema:
  api_key: "str?"
```

### Tech Stack

| Component | Technologie |
|-----------|-------------|
| Runtime | Python 3.11+ |
| Web | aiohttp |
| UI | Alpine.js + Tailwind |
| DB access | sqlite3 |
| HA API | REST |

### Bestaande Repos

- `synctacles/synctacles-api` - Backend
- `synctacles/synctacles-ha` - Energy Integration

---

## Referenties

### Voorbeelden

| Add-on | Wat te leren |
|--------|--------------|
| hassio-addons/addon-sqlite-web | DB access |
| hassio-addons/addon-node-red | Web UI + ingress |
| thomasmauerer/hassio-addons | Backup integratie |

### Docs

- https://developers.home-assistant.io/docs/add-ons/configuration/
- https://developers.home-assistant.io/docs/add-ons/tutorial/

### Project Skills (LEES)

- `/mnt/project/SKILL_00_AI_OPERATING_PROTOCOL.md`
- `/mnt/project/SKILL_02_ARCHITECTURE.md`
- `/mnt/project/SKILL_03_CODING_STANDARDS.md`

---

## Vragen

1. Mono-repo of dedicated? Waarom?
2. Hoe versioning tussen add-on en backend?
3. HACS default of eigen repo?
4. CI/CD voor multi-arch builds?

---

## Deliverables

- [ ] Beslisdocument (repo structuur + rationale)
- [ ] Repository aangemaakt
- [ ] README, LICENSE, repository.yaml
- [ ] Labels, milestones
- [ ] Issue templates, PR template
- [ ] CI/CD skeletons
- [ ] Alle 77 V1.0 issues

---

## Bijlagen

**LEES DEZE DOCUMENTEN:**

| Document | Inhoud | Prioriteit |
|----------|--------|------------|
| `SYNCTACLES_CARE_DESIGN.md` | **Technisch design** - architectuur, data model, feature specs, safeguards, UI flows | **EERST** |
| `SYNCTACLES_CARE_ROADMAP_V2.md` | Alle 77 issues, epics, timeline | Voor issue creatie |
| `SKILL_17_GO_TO_MARKET_V2.md` | Business context, pricing, funnels | Achtergrond |

**Het Design Document bevat:**
- System overview + component diagram
- Data model (DB schema, API contracts)
- Feature specs per module (health, security, cleanup, backup, trial)
- Safeguards en wat NOOIT fout mag gaan
- UI wireframes in ASCII
- Testing strategy
- Open questions voor jou

Met deze docs kan je zelfstandig goede Acceptance Criteria schrijven per issue.

---

*Einde handoff*
