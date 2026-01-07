# Architecture Decision Records (ADR)

This directory contains Architecture Decision Records documenting significant technical decisions.

## Naming Convention

```
ADR_###_[TITEL].md
```

- `###` = Sequential number (001, 002, etc.)
- `TITEL` = Short descriptive title in UPPER_SNAKE_CASE

## When to Create an ADR

Create an ADR for:
- Architecture choices with long-term impact
- Technology selection
- Data model decisions
- API design decisions
- Integration patterns
- Security decisions

Do NOT create an ADR for:
- Bug fixes
- Small refactors
- Documentation updates
- Configuration changes

## ADR Numbering

| Range | Status |
|-------|--------|
| ADR_001 - ADR_008 | Implicit in existing SKILLs |
| ADR_009+ | New explicit decisions |

## Template

See SKILL_00 Section O for the full ADR template.

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [ADR-001](ADR_001_TENNET_BYO_KEY.md) | TenneT Bring-Your-Own-Key Model | Accepted | 2026-01-07 |

## Implicit ADRs (in SKILLs)

- **SKILL_02:** Brand-free architecture
- **SKILL_06:** Data source hierarchy (ENTSO-E primary, Energy-Charts fallback)
- **SKILL_12:** Multi-brand deployment pattern

**Note:** ADR-001 formalizes the TenneT BYO-key decision previously implicit in SKILL_02.
