# HANDOFF: CAI → CC

**Datum:** 2026-01-07
**Van:** Claude AI (CAI)
**Naar:** Claude Code (CC)
**Prioriteit:** P0
**Geschatte tijd:** 30 min

---

## TASK DESCRIPTION

Implementeer Phase 1 van Shared Knowledge Architecture:
1. Maak nieuwe directories aan
2. Plaats SKILL_00 v2.0
3. Maak STATUS templates
4. Maak NEXT_ACTIONS.md

---

## STAP 1: DIRECTORIES AANMAKEN

```bash
# Als service user
sudo -u energy-insights-nl mkdir -p /opt/github/synctacles-api/docs/status
sudo -u energy-insights-nl mkdir -p /opt/github/synctacles-api/docs/sessions
sudo -u energy-insights-nl mkdir -p /opt/github/synctacles-api/docs/sessions/archive
sudo -u energy-insights-nl mkdir -p /opt/github/synctacles-api/docs/decisions
sudo -u energy-insights-nl mkdir -p /opt/github/synctacles-api/docs/templates
```

---

## STAP 2: SKILL_00 PLAATSEN

Leo heeft SKILL_00_AI_OPERATING_PROTOCOL.md ontvangen van CAI.

**Actie CC:** Kopieer content naar:
```
/opt/github/synctacles-api/docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md
```

**Verwijder oude versie (indien bestaat):**
```
/opt/github/synctacles-api/docs/skills/SKILL_00_CC_OPERATING_PROTOCOL.md
```

---

## STAP 3: STATUS FILES AANMAKEN

### 3a. STATUS_MERGED_CURRENT.md

**Locatie:** `/opt/github/synctacles-api/docs/status/STATUS_MERGED_CURRENT.md`

```markdown
# STATUS_MERGED_CURRENT.md

**Last Updated:** 2026-01-07
**Updated By:** Leo (initial)

---

## PROJECT STATE

### Current Phase
- Sprint 1: Technical Foundation (Jan 7-14)
- Parallel: Shared Knowledge Architecture

### Active Work
- [x] SKILL_00 v2.0 created (CAI)
- [x] Directory structure implemented (CC)
- [ ] HA Component TenneT BYO-key (pending)

### Blockers
- None

---

## SERVER STATE

### Services
- energy-insights-nl-api: running
- energy-insights-nl-collector: running
- energy-insights-nl-normalizer: running

### Last Deploy
- Date: [CC vult in]
- Commit: [CC vult in]

---

## NEXT PRIORITIES

1. Complete Sprint 1 tasks
2. HA Component development
3. Documentation cleanup (Phase 3)

---

## OPEN DECISIONS

- None pending

---

## NOTES

Initial SSOT created as part of Shared Knowledge Architecture Phase 1.
```

### 3b. STATUS_CC_CURRENT.md

**Locatie:** `/opt/github/synctacles-api/docs/status/STATUS_CC_CURRENT.md`

```markdown
# STATUS_CC_CURRENT.md

**Last Updated:** 2026-01-07
**Updated By:** CC

---

## SERVER STATE

### Services
| Service | Status | Last Check |
|---------|--------|------------|
| energy-insights-nl-api | [check] | [timestamp] |
| energy-insights-nl-collector | [check] | [timestamp] |
| energy-insights-nl-normalizer | [check] | [timestamp] |

### Disk Usage
- /opt: [check]
- /var/log: [check]

### Last Deploy
- Timestamp: [check]
- Commit: [check]

---

## CODE CHANGES (uncommitted)

- None

---

## GIT STATUS

- Branch: main
- Last commit: [hash] [message]
- Uncommitted changes: No

---

## OPEN ISSUES

- None

---

## BLOCKED BY

- None

---

## LAST SESSION

- Date: 2026-01-07
- Focus: Phase 1 State Files implementation
- Outcome: [pending]
```

### 3c. STATUS_CAI_CURRENT.md

**Locatie:** `/opt/github/synctacles-api/docs/status/STATUS_CAI_CURRENT.md`

```markdown
# STATUS_CAI_CURRENT.md

**Last Updated:** 2026-01-07
**Updated By:** CAI

---

## PROJECT PHASE

- Current: Sprint 1 - Technical Foundation
- Next Milestone: Jan 14 - Sprint 1 complete
- Launch Target: Jan 25

---

## ARCHITECTURAL STATE

### Recent Decisions
- TenneT BYO-key model (ADR in SKILL_02)
- Dual status model for AI coordination

### Open Decisions
- None pending

### Recent ADRs
- None formalized yet (historical in SKILLs)

---

## PLANNING STATUS

### Sprint 1 (Jan 7-14)
- [ ] HA Component foundations
- [ ] API hardening
- [x] Shared Knowledge Phase 1

### Parallel Work
- [x] SKILL_00 v2.0 expansion
- [ ] Documentation audit (Phase 3)

---

## DOCUMENTATION STATE

### Updates Needed
- README.md index update (add new directories)
- SKILL_11 minor update (reference SKILL_00)

### Reviews Pending
- None

---

## OPEN QUESTIONS FOR LEO

- None

---

## BLOCKED BY

- None

---

## HANDOFF NOTES

### For CC
- Phase 1 implementation ready
- See this handoff document

### From Last Session
- Shared Knowledge Architecture planning complete
- SKILL_00 v2.0 delivered
```

---

## STAP 4: NEXT_ACTIONS.md AANMAKEN

**Locatie:** `/opt/github/synctacles-api/docs/status/NEXT_ACTIONS.md`

```markdown
# NEXT_ACTIONS.md

**Last Updated:** 2026-01-07
**Owner:** Leo

---

## P0 - BLOCKING / URGENT

- None

---

## P1 - THIS SPRINT (Jan 7-14)

### Technical
- [ ] HA Component TenneT BYO-key implementation
- [ ] API endpoint hardening
- [ ] Error handling improvements

### Documentation
- [ ] Phase 2: Handoff protocol formalization
- [ ] Phase 3: Documentation audit

---

## P2 - NEXT SPRINT (Jan 15-21)

- [ ] HA Component testing
- [ ] Phase 4: Documentation cleanup
- [ ] Pre-launch checklist

---

## P3 - BACKLOG

- [ ] Multi-country support planning
- [ ] Payment integration research
- [ ] YouTube creator outreach

---

## COMPLETED (Recent)

- [x] 2026-01-07: SKILL_00 v2.0 (CAI)
- [x] 2026-01-07: Phase 1 directories (CC)
- [x] 2026-01-07: Status files created (CC)

---

## NOTES

Priority definitions:
- **P0:** Blocking launch or breaking production
- **P1:** Must complete this sprint
- **P2:** Should complete next sprint
- **P3:** Backlog / when time permits
```

---

## STAP 5: SESSIONS README AANMAKEN

**Locatie:** `/opt/github/synctacles-api/docs/sessions/README.md`

```markdown
# Sessions Directory

Sessie samenvattingen van CC en CAI werkzaamheden.

## Naming Convention

```
SESSIE_[BRON]_[YYYYMMDD].md
```

- `BRON`: CC of CAI
- `YYYYMMDD`: Datum van sessie

## When to Create

- Sessies > 1 uur
- Significante wijzigingen
- Belangrijke beslissingen
- Handoff momenten

## Archive

Sessies ouder dan 30 dagen worden verplaatst naar `archive/`.
```

---

## STAP 6: DECISIONS README AANMAKEN

**Locatie:** `/opt/github/synctacles-api/docs/decisions/README.md`

```markdown
# Architecture Decision Records

## ADR Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| 001-008 | (Historical, in SKILLs) | Accepted | Pre-2026 |
| 009+ | (Future decisions) | - | - |

## When to Create ADR

- Architecture choices with long-term impact
- Technology selection
- Data model decisions
- API design decisions
- Integration patterns
- Security decisions

## Template

See SKILL_00 Section O for ADR template.

## Workflow

1. CAI drafts ADR (Status: Proposed)
2. Leo reviews and approves
3. Status → Accepted
4. CC implements (if needed)
5. Update relevant SKILLs
```

---

## STAP 7: TEMPLATES AANMAKEN

### 7a. TEMPLATE_STATUS_CC.md

**Locatie:** `/opt/github/synctacles-api/docs/templates/TEMPLATE_STATUS_CC.md`

```markdown
# STATUS_CC_CURRENT.md

**Last Updated:** [DATUM]
**Updated By:** CC

---

## SERVER STATE

### Services
| Service | Status | Last Check |
|---------|--------|------------|
| energy-insights-nl-api | [status] | [timestamp] |
| energy-insights-nl-collector | [status] | [timestamp] |
| energy-insights-nl-normalizer | [status] | [timestamp] |

### Disk Usage
- /opt: [usage]
- /var/log: [usage]

### Last Deploy
- Timestamp: [timestamp]
- Commit: [hash]

---

## CODE CHANGES (uncommitted)

- [ ] [file] - [beschrijving]

---

## GIT STATUS

- Branch: [branch]
- Last commit: [hash] [message]
- Uncommitted changes: [yes/no]

---

## OPEN ISSUES

- [ ] [issue]

---

## BLOCKED BY

- [blocker of "None"]

---

## LAST SESSION

- Date: [datum]
- Focus: [onderwerp]
- Outcome: [resultaat]
```

### 7b. TEMPLATE_STATUS_CAI.md

**Locatie:** `/opt/github/synctacles-api/docs/templates/TEMPLATE_STATUS_CAI.md`

```markdown
# STATUS_CAI_CURRENT.md

**Last Updated:** [DATUM]
**Updated By:** CAI

---

## PROJECT PHASE

- Current: [phase/sprint]
- Next Milestone: [datum] - [beschrijving]
- Launch Target: [datum]

---

## ARCHITECTURAL STATE

### Recent Decisions
- [beslissing]

### Open Decisions
- [beslissing of "None"]

### Recent ADRs
- [ADR of "None"]

---

## PLANNING STATUS

### Current Sprint
- [ ] [taak]
- [ ] [taak]

### Parallel Work
- [ ] [taak]

---

## DOCUMENTATION STATE

### Updates Needed
- [doc of "None"]

### Reviews Pending
- [doc of "None"]

---

## OPEN QUESTIONS FOR LEO

- [ ] [vraag of "None"]

---

## BLOCKED BY

- [blocker of "None"]

---

## HANDOFF NOTES

### For CC
- [instructie of "None pending"]

### From Last Session
- [samenvatting]
```

### 7c. TEMPLATE_SESSIE.md

**Locatie:** `/opt/github/synctacles-api/docs/templates/TEMPLATE_SESSIE.md`

```markdown
# SESSIE SAMENVATTING

**Datum:** [YYYY-MM-DD]
**Bron:** [CC | CAI]
**Duur:** [X] uur
**Focus:** [hoofdonderwerp]

---

## UITGEVOERDE WERK

### Completed
- [x] [taak] - [beschrijving]
- [x] [taak] - [beschrijving]

### Gewijzigde Files
| File | Actie | Beschrijving |
|------|-------|--------------|
| [path] | [Created/Modified/Deleted] | [wat] |

### Git Commits
- `[hash]` - [message]

---

## BESLISSINGEN

| Beslissing | Rationale | ADR? |
|------------|-----------|------|
| [beslissing] | [waarom] | [Nee / ADR-XXX] |

---

## OPEN ITEMS

### Blocked
- [ ] [item] - blocked by [wat]

### TODO (volgende sessie)
- [ ] [item]

### Vragen voor Leo
- [ ] [vraag]

---

## HANDOFF NOTES

### Voor CC (indien CAI sessie)
- [instructies]

### Voor CAI (indien CC sessie)
- [input nodig]

---

## STATUS UPDATE

[Korte update voor STATUS_[CC|CAI]_CURRENT.md]
```

### 7d. TEMPLATE_HANDOFF_CAI_CC.md

**Locatie:** `/opt/github/synctacles-api/docs/templates/TEMPLATE_HANDOFF_CAI_CC.md`

```markdown
# HANDOFF: CAI → CC

**Datum:** [YYYY-MM-DD]
**Van:** Claude AI (CAI)
**Naar:** Claude Code (CC)
**Prioriteit:** [P0/P1/P2]
**Geschatte tijd:** [X] min

---

## TASK DESCRIPTION

[Wat moet CC doen]
[Verwachte output]

---

## SPECIFICATIONS

[Technische details]
[Acceptance criteria]

---

## FILES TO CREATE/MODIFY

- [ ] [path/to/file] - [instructies]
- [ ] [path/to/file] - [instructies]

---

## RELEVANT SKILLS

- SKILL_XX voor [aspect]
- SKILL_YY voor [aspect]

---

## OUT OF SCOPE

- [wat NIET doen]
- [off-limits gebieden]

---

## VERIFICATION

- [ ] [test 1]
- [ ] [test 2]

---

**CC: Wacht op Leo's "go" voordat je uitvoert.**
```

### 7e. TEMPLATE_HANDOFF_CC_CAI.md

**Locatie:** `/opt/github/synctacles-api/docs/templates/TEMPLATE_HANDOFF_CC_CAI.md`

```markdown
# HANDOFF: CC → CAI

**Datum:** [YYYY-MM-DD]
**Van:** Claude Code (CC)
**Naar:** Claude AI (CAI)

---

## COMPLETED WORK

- [wat is gedaan]
- [welke files gewijzigd]

---

## CURRENT STATE

### Server Status
- [services status]

### Git Status
- Branch: [branch]
- Last commit: [hash] [message]

---

## NEEDS FROM CAI

- [ ] Review van [X]
- [ ] Documentatie update voor [Y]
- [ ] Planning advies voor [Z]

---

## CONTEXT

[Relevante achtergrond]

---

## FILES TO REVIEW

- [path/to/file1]
- [path/to/file2]
```

### 7f. TEMPLATE_ADR.md

**Locatie:** `/opt/github/synctacles-api/docs/templates/TEMPLATE_ADR.md`

```markdown
# ADR-XXX: [Titel]

**Status:** Proposed | Accepted | Deprecated | Superseded
**Date:** [YYYY-MM-DD]
**Author:** [Leo | CAI | CC]
**Supersedes:** [ADR-YYY of "None"]

---

## Context

[Wat is het probleem of de beslissing die genomen moet worden?]
[Wat is de huidige situatie?]

---

## Decision

[Wat hebben we besloten?]
[Concrete, uitvoerbare beslissing.]

---

## Consequences

### Positief
- [voordeel]

### Negatief
- [nadeel]

### Risico's
- [risico]

---

## Alternatives Considered

### Optie A: [naam]
- Beschrijving: [wat]
- Waarom niet gekozen: [reden]

### Optie B: [naam]
- Beschrijving: [wat]
- Waarom niet gekozen: [reden]

---

## Implementation

- [ ] Stap 1
- [ ] Stap 2
- [ ] Stap 3

---

## References

- [links naar relevante docs]
- [SKILLs die geraakt worden]
```

---

## STAP 8: GIT COMMIT

```bash
cd /opt/github/synctacles-api

# Fix ownership
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

# Add all new files
sudo -u energy-insights-nl git add docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md
sudo -u energy-insights-nl git add docs/status/
sudo -u energy-insights-nl git add docs/sessions/
sudo -u energy-insights-nl git add docs/decisions/
sudo -u energy-insights-nl git add docs/templates/

# Remove old SKILL_00 if exists
sudo -u energy-insights-nl git rm docs/skills/SKILL_00_CC_OPERATING_PROTOCOL.md 2>/dev/null || true

# Commit
sudo -u energy-insights-nl git commit -m "feat: Shared Knowledge Architecture Phase 1

- Add SKILL_00 v2.0 (AI Operating Protocol for CC + CAI)
- Create docs/status/ with STATUS files and NEXT_ACTIONS
- Create docs/sessions/ for session logs
- Create docs/decisions/ for ADRs
- Create docs/templates/ with 6 reusable templates
- Implement dual status model (CC/CAI/MERGED)

Part of: Shared Knowledge Architecture project
Ref: 4.5h incident prevention"

# Push
sudo -u energy-insights-nl git push origin main
```

---

## VERIFICATION

Na uitvoering, bevestig:

```bash
# Directory structure
ls -la /opt/github/synctacles-api/docs/status/
ls -la /opt/github/synctacles-api/docs/sessions/
ls -la /opt/github/synctacles-api/docs/decisions/
ls -la /opt/github/synctacles-api/docs/templates/

# Files exist
cat /opt/github/synctacles-api/docs/status/STATUS_MERGED_CURRENT.md | head -5
ls /opt/github/synctacles-api/docs/templates/TEMPLATE_*.md | wc -l  # Should be 6
```

---

## OUT OF SCOPE

- Geen wijzigingen aan bestaande code
- Geen service restarts
- Geen database changes
- Geen andere SKILL files aanpassen (komt later)

---

## RELEVANT SKILLS

- SKILL_00: Dit document definieert alles
- SKILL_11: Git discipline (service user!)

---

## ACCEPTANCE CRITERIA

- [ ] 4 nieuwe directories bestaan (status, sessions, decisions, templates)
- [ ] SKILL_00 v2.0 in docs/skills/
- [ ] 3 STATUS files in docs/status/
- [ ] NEXT_ACTIONS.md in docs/status/
- [ ] README.md in sessions/ en decisions/
- [ ] 6 templates in docs/templates/
- [ ] Git commit + push succesvol
- [ ] Oude SKILL_00_CC_OPERATING_PROTOCOL.md verwijderd

---

**CC: Wacht op Leo's "go" voordat je uitvoert.**
