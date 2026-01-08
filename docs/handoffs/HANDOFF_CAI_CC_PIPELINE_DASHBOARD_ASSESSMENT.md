# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** LOW
**Type:** Assessment

---

## CONTEXT

Leo wil een Pipeline Health Dashboard in Grafana:
- Elke data source (A75, A65, A44) als rij
- Elke pipeline laag (Collector → Importer → Normalizer → API) als kolom
- Blokken worden groen/rood gebaseerd op health

Vandaag's bug (normalizer stuk) was dan instant zichtbaar geweest.

**Geschatte effort:** 7-10 uur

---

## VRAAG

Review de huidige GitHub issues en backlog. Bepaal:

1. **Prioriteit** - Waar past "Pipeline Health Dashboard" t.o.v. bestaande taken?
2. **Dependencies** - Zijn er taken die eerst moeten?
3. **Advies** - Nu doen, later doen, of backlog?

---

## BENODIGDE INFO

```bash
# GitHub issues
gh issue list --repo ldraaisma/synctacles-api --state open

# Of als gh niet authenticated:
cat /opt/github/synctacles-api/docs/GITHUB_ISSUES_TO_CREATE.md

# Bestaande Grafana setup
ls -la /etc/grafana/provisioning/dashboards/
cat /etc/grafana/provisioning/dashboards/*.json 2>/dev/null | head -100
```

---

## OUTPUT FORMAT

```markdown
## PIPELINE DASHBOARD PRIORITY ASSESSMENT

### Huidige Backlog
| # | Issue | Priority | Status |
|---|-------|----------|--------|

### Dependencies
- [ ] [Dependency 1]
- [ ] [Dependency 2]

### Prioriteit Advies
[HIGH/MEDIUM/LOW] - [Reden]

### Aanbevolen Actie
[Nu / Sprint X / Backlog]

### Alternatief (indien niet nu)
[Quick win optie indien beschikbaar]
```

---

## OUT OF SCOPE

- Geen implementatie
- Alleen assessment + advies

---

*Template versie: 1.0*
