# SKILL CARE 00 — OVERVIEW

Product Overview and Features
**Version: 1.0 (2026-01-26)**

> **Purpose:** SYNCTACLES Care is a Home Assistant add-on for database maintenance and security audits.
> Care is the kernproduct (€25/jaar), Energy is the acquisition funnel.

---

## EXECUTIVE SUMMARY

SYNCTACLES Care provides Home Assistant users with:
- **Health Scan**: Database health analysis with A-F grading
- **Security Scan**: Security posture scoring (0-100)
- **Orphan Detection**: Find orphaned statistics/entities
- **Cleanup** (Premium): One-click cleanup of orphans
- **Scheduled Maintenance** (Premium): Automated maintenance windows

**Repository:** `addon-synctacles-care`
**Runtime:** Docker container inside HA add-ons
**Python:** 3.11+

---

## PRODUCT TIER MATRIX

| Feature | Gratis | Trial (14d) | Premium |
|---------|--------|-------------|---------|
| Health scan + grade | ✅ | ✅ | ✅ |
| Security scan + score | ✅ | ✅ | ✅ |
| Orphan view | ✅ | ✅ | ✅ |
| **Cleanup** | 🔒 | 🔒 | ✅ |
| **Scheduled** | 🔒 | 🔒 | ✅ |
| **Backup mgmt** | 🔒 | 🔒 | ✅ |

> **Kritieke Regel:** Care cleanup is NOOIT in trial. Dat is de conversie driver.

---

## KEY VALUE PROPOSITION

| Problem | Solution | Value |
|---------|----------|-------|
| "7000 orphaned statistics, handmatig klikken" | One-click cleanup | Uren werk bespaard |
| "Database 7GB, help!" | Health scan + cleanup | Ruimte + snelheid |
| "Is mijn HA veilig?" | Security Score 0-100 | Peace of mind |

---

## ARCHITECTURE OVERVIEW

```
┌─────────────────────────────────────────────────────────────┐
│                    HOME ASSISTANT HOST                       │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────────┐ │
│  │              SYNCTACLES CARE ADD-ON                     │ │
│  │                   (Docker Container)                    │ │
│  │  ┌─────────────┬─────────────┬────────────────────────┐│ │
│  │  │   Scanner   │   Cleaner   │      Web UI            ││ │
│  │  │  - Health   │  - Orphans  │   (aiohttp)            ││ │
│  │  │  - Security │  - Backup   │                        ││ │
│  │  │  - Schema   │  - Database │   Port 8099            ││ │
│  │  └─────────────┴─────────────┴────────────────────────┘│ │
│  │         │                │                              │ │
│  │         │ READ-ONLY      │ WRITE (Premium)              │ │
│  │         ▼                ▼                              │ │
│  │  ┌────────────────────────────────────────────────────┐│ │
│  │  │           /config (mounted volume)                 ││ │
│  │  │  - home-assistant_v2.db (SQLite)                   ││ │
│  │  │  - .storage/core.entity_registry                   ││ │
│  │  │  - .storage/auth                                   ││ │
│  │  └────────────────────────────────────────────────────┘│ │
│  └────────────────────────────────────────────────────────┘ │
│                              │                              │
│                              │ Supervisor API               │
│                              ▼                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Home Assistant Supervisor                  │ │
│  │  - HA Core info (version)                              │ │
│  │  - Add-on management                                    │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

## HEALTH SCAN OUTPUT

```json
{
  "grade": "B",
  "score": 78,
  "db_size_bytes": 524288000,
  "db_size_human": "500 MB",
  "fragmentation_pct": 15.2,
  "schema_version": 53,
  "schema_supported": true,
  "orphaned_statistics_count": 247,
  "orphaned_entities_count": 12,
  "recommendations": [
    {
      "type": "cleanup",
      "severity": "warning",
      "message": "247 orphaned statistics found",
      "action": "Run cleanup to remove"
    }
  ]
}
```

### Grading Scale

| Grade | Score Range | Meaning |
|-------|-------------|---------|
| A | 90-100 | Excellent health |
| B | 75-89 | Good, minor issues |
| C | 60-74 | Fair, attention needed |
| D | 40-59 | Poor, cleanup recommended |
| F | 0-39 | Critical, immediate action |

---

## CODE STRUCTURE

```
addon-synctacles-care/
├── synctacles-care/
│   └── care/
│       ├── __init__.py
│       ├── main.py              # Entry point
│       ├── config.py            # Configuration
│       │
│       ├── scanner/             # Read-only analysis (FREE)
│       │   ├── health.py        # HealthScanner class
│       │   ├── security.py      # SecurityScanner class
│       │   ├── schema.py        # Schema detection (P2 #27)
│       │   ├── cleanup.py       # DatabaseCleaner class
│       │   └── models.py        # Pydantic models
│       │
│       ├── cleaner/             # Write operations (PREMIUM)
│       │   ├── orphans.py       # Orphan cleanup
│       │   ├── backup.py        # Backup management
│       │   ├── database.py      # DB operations
│       │   └── safeguards.py    # Pre-cleanup safety checks
│       │
│       ├── utils/
│       │   ├── version.py       # HA version check (P2 #25)
│       │   ├── supervisor.py    # Supervisor API client
│       │   ├── ha_api.py        # HA REST API client
│       │   └── logging.py       # Structured logging
│       │
│       ├── api/
│       │   ├── client.py        # SYNCTACLES API client
│       │   └── models.py        # API models
│       │
│       └── web/
│           ├── server.py        # aiohttp server
│           └── routes.py        # HTTP endpoints
│
├── tests/
│   ├── conftest.py              # Pytest fixtures
│   ├── test_scanner_*.py        # Scanner unit tests
│   ├── test_cleaner_*.py        # Cleaner unit tests
│   └── test_integration_ha.py   # Integration tests (HA DEV)
│
├── pyproject.toml               # Dependencies + pytest config
└── Dockerfile                   # HA add-on container
```

---

## CURRENT STATUS (2026-01-26)

### Implemented (V1.0 MVP)

| Component | Status | Description |
|-----------|--------|-------------|
| Health Scanner | ✅ | Database analysis, grading |
| Security Scanner | ✅ | Security posture scoring |
| Schema Detection | ✅ | P2 #27 - Schema compatibility |
| Version Check | ✅ | P2 #25 - HA version validation |
| Orphan Detection | ✅ | Find orphaned statistics/entities |
| Unit Tests | ✅ | 105 tests, all passing |
| Integration Tests | ✅ | 12 tests against HA DEV |
| CI Pipeline | ✅ | GitHub Actions (ruff, pytest) |

### Planned (V1.1+)

| Component | Status | Description |
|-----------|--------|-------------|
| Cleanup | 🚧 | One-click orphan removal |
| Scheduled | 🚧 | Automated maintenance windows |
| Backup Mgmt | 🚧 | Backup creation/restoration |
| Web UI | 🚧 | Full ingress UI |

---

## RELATED SKILLS

| Skill | Description |
|-------|-------------|
| [SKILL_01_CARE_SAFETY.md](SKILL_01_CARE_SAFETY.md) | Safety mitigations (P0-P2) |
| [SKILL_02_CARE_ARCHITECTURE.md](SKILL_02_CARE_ARCHITECTURE.md) | Technical architecture |
| [SKILL_03_CARE_TESTING.md](SKILL_03_CARE_TESTING.md) | Testing infrastructure |
| [../business/SKILL_00_GO_TO_MARKET.md](../business/SKILL_00_GO_TO_MARKET.md) | Business model & pricing |

---

*Generated: 2026-01-26*
