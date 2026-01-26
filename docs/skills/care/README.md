# SYNCTACLES Care Skills

Technical documentation for the Care add-on (database maintenance & security audit).

## Quick Reference

| Skill | Description |
|-------|-------------|
| [SKILL_00_CARE_OVERVIEW.md](SKILL_00_CARE_OVERVIEW.md) | Product overview, features, tier matrix |
| [SKILL_01_CARE_SAFETY.md](SKILL_01_CARE_SAFETY.md) | P0-P2 safety mitigations |
| [SKILL_02_CARE_ARCHITECTURE.md](SKILL_02_CARE_ARCHITECTURE.md) | Code structure, data models, APIs |
| [SKILL_03_CARE_TESTING.md](SKILL_03_CARE_TESTING.md) | Unit tests, integration tests, CI |

## Repository

**Code:** `addon-synctacles-care`
**Location:** `/opt/github/addon-synctacles-care`

## Current Status (2026-01-26)

- **Version:** 1.0 MVP
- **Tests:** 105 unit + 12 integration (all passing)
- **Safety:** P0-P2 mitigations implemented
- **CI:** GitHub Actions + pre-push hook

## Key Principles

1. **Data Safety First** - Every write operation has safeguards
2. **Dry-Run Default** - All cleanup operations simulate by default
3. **Schema Aware** - Validates HA database schema before operations
4. **Test Against Real Data** - Integration tests run against HA DEV

---

*See main skills at: [../README.md](../README.md)*
