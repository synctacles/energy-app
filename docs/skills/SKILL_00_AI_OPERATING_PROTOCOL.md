# SKILL 00 — AI OPERATING PROTOCOL

**MANDATORY READING BEFORE ANY ACTION**
Version: 2.0 (2026-01-07)
Status: ENFORCED
Scope: Claude Code (CC) + Claude AI (CAI)

---

## ⛔ STOP — READ THIS FIRST

Dit document is VERPLICHT voor elke AI sessie (CC én CAI).
Geen acties, geen fixes, geen edits voordat je dit hebt gelezen EN bewezen dat je het snapt.

**Incident aanleiding:** 4.5 uur verspild door CC die:
- SKILLs niet las
- Aannames maakte zonder verificatie
- Productie ging "fixen" zonder toestemming
- Deprecated services probeerde te repareren
- TenneT aanraakte terwijl dit OFF-LIMITS is

---

# DEEL 1: ALGEMEEN PROTOCOL (CC + CAI)

---

## SECTIE A: MANDATORY SKILL READING

### Lees ALTIJD voor je begint:

| SKILL | Bestand | Waarom verplicht |
|-------|---------|------------------|
| **SKILL 00** | SKILL_00_AI_OPERATING_PROTOCOL.md | Dit document |
| **SKILL 01** | SKILL_01_HARD_RULES.md | Non-negotiable regels, fail-fast, KISS |
| **SKILL 02** | SKILL_02_ARCHITECTURE.md | System design, TenneT BYO-KEY model |
| **SKILL 11** | SKILL_11_REPO_AND_ACCOUNTS.md | Git workflow, service accounts, GEEN ROOT |

### Lees INDIEN RELEVANT:

| SKILL | Bestand | Wanneer |
|-------|---------|---------|
| SKILL 03 | SKILL_03_CODING_STANDARDS.md | Bij code schrijven |
| SKILL 06 | SKILL_06_DATA_SOURCES.md | Bij data pipeline werk |
| SKILL 09 | SKILL_09_INSTALLER_SPECS.md | Bij deployment/setup |
| SKILL 10 | SKILL_10_DEPLOYMENT_WORKFLOW.md | Bij deployment |
| SKILL 13 | SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md | Bij logging/debugging |

### Bewijs dat je gelezen hebt:

**START ELKE SESSIE MET:**
```
"Ik heb SKILL_00, SKILL_01, SKILL_02, SKILL_11 gelezen.
Key points:
- [1 bullet SKILL_00]
- [1 bullet SKILL_01]
- [1 bullet SKILL_02]
- [1 bullet SKILL_11]

Mag ik beginnen?"
```

**WACHT OP GOEDKEURING VOORDAT JE VERDERGAAT.**

---

## SECTIE B: PROTECT MODE = DEFAULT

### Wat PROTECT MODE betekent:

```
✅ TOEGESTAAN:
- Lezen (cat, view, less)
- Analyseren (grep, find, ls)
- Vragen stellen
- Documenteren
- Plannen maken

❌ VERBODEN:
- Bestanden aanpassen
- Services herstarten
- Git commits
- Database wijzigingen
- "Even snel fixen"
```

### Wanneer PROTECT MODE eindigt:

**ALLEEN** wanneer Leo expliciet zegt:
- "1" of "go" of "execute"
- "Ja, pas aan"
- "Fix it"
- "Akkoord, uitvoeren"

**NIET** bij:
- "Interessant"
- "Hmm"
- "Kun je kijken naar..."
- Stilte

---

## SECTIE C: VERIFICATIE VOOR CONCLUSIES

### ❌ VERBODEN gedrag:

```
"Script mist" → zonder `ls -la` output
"Service is broken" → zonder te vragen of het deprecated is
"Ik fix even" → zonder expliciete toestemming
"Volgens mij..." → zonder verificatie
```

### ✅ VERPLICHT gedrag:

```
STAP 1: Observatie
$ ls -la /opt/energy-insights-nl/app/scripts/
[toon output]

STAP 2: Vraag
"Ik zie dat run_importers.sh ontbreekt. Is dit:
 a) Deprecated (niet nodig)
 b) Broken (moet gerepareerd)
 c) Anders?"

STAP 3: WACHT op antwoord

STAP 4: Pas DAN actie voorstellen
```

---

## SECTIE D: OFF-LIMITS GEBIEDEN

### Raak NOOIT aan zonder expliciete instructie:

| Gebied | Reden | Documentatie |
|--------|-------|--------------|
| **TenneT services** | Juridisch: geen redistributie | SKILL_02 §TenneT BYO-KEY |
| **synctacles-* services** | Deprecated (oude naming) | SKILL_11 |
| **/opt/.env** | Productie secrets | SKILL_01 |
| **Database credentials** | Security | SKILL_03 |

### Bij twijfel:

```
"Ik wil [X] aanpassen. Dit raakt [Y gebied].
Is dit toegestaan of off-limits?"
```

---

## SECTIE E: ESCALATIE

### Wanneer STOPPEN en VRAGEN:

1. **Onduidelijke scope** - "Moet dit gerepareerd of is het deprecated?"
2. **Meerdere mogelijke oorzaken** - "Ik zie 3 mogelijkheden, welke eerst?"
3. **Productie impact** - "Dit raakt live services, mag ik doorgaan?"
4. **Twijfel over off-limits** - "Raakt dit TenneT/secrets/deprecated code?"
5. **Conflicterende informatie** - "SKILL zegt X, maar ik zie Y"

### Format:

```
⚠️ ESCALATIE:
- Situatie: [wat ik zie]
- Twijfel: [waarom ik stop]
- Opties: [a, b, c]
- Vraag: [concrete vraag]
```

---

## SECTIE F: SESSION CHECKLIST

### Start sessie:

```
□ SKILL_00 gelezen (dit document)
□ SKILL_01, SKILL_02, SKILL_11 gelezen
□ Key points samengevat aan Leo
□ Goedkeuring ontvangen om te beginnen
□ PROTECT MODE = actief
□ STATUS_MERGED_CURRENT.md gelezen (indien bestaat)
```

### Tijdens sessie:

```
□ Verificatie VOOR conclusies
□ Vragen VOOR acties
□ Geen edits zonder "1"/"go"
□ Failed services: vraag deprecated vs broken
□ Off-limits gebieden: niet aanraken
□ chown DIRECT na file edits (CC only)
```

### Einde sessie:

```
□ Alle wijzigingen gedocumenteerd
□ STATUS_[CC|CAI]_CURRENT.md bijgewerkt
□ SESSIE_[CC|CAI]_[DATUM].md opgeleverd (bij significante sessies)
□ Git commits met accountability (CC only)
□ Status gerapporteerd aan Leo
```

---

# DEEL 2: CLAUDE CODE (CC) SPECIFIEK

---

## SECTIE G: CC GIT DISCIPLINE

### ELKE git operatie:

```bash
# CORRECT - SSH key bestaat voor energy-insights-nl user
sudo -u energy-insights-nl git -C /opt/github/synctacles-api <command>

# FOUT - NOOIT DOEN
git <command>
sudo git <command>
cd /opt/github/synctacles-api && git push
```

### Na file edits (KRITIEK):

**⚠️ Na ELKE file creatie of edit → DIRECT chown uitvoeren**

Niet wachten tot het einde van de sessie. Niet wachten tot voor git commit.
Direct na de edit, vóór de volgende actie.

```bash
# NA ELKE FILE EDIT - GEEN UITZONDERINGEN
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
```

**Waarom direct?**
- Root-owned files blokkeren git operations
- Service user kan root-owned files niet lezen
- Problemen stapelen op als je wacht

**Patroon:**
```bash
# 1. Edit file (als root is OK)
nano /opt/github/synctacles-api/docs/file.md

# 2. DIRECT daarna - niet later
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

# 3. Dan pas volgende actie
```

**Bij meerdere files:**
```bash
# Edit file 1
# Edit file 2
# Edit file 3
# chown (eenmalig voor batch is OK, maar VOOR volgende stap)
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
# Dan pas git of andere acties
```

### Commit messages:

```
<type>: <wat>

<waarom>

<accountability als relevant>
```

---

## SECTIE H: CC FAILED SERVICES PROTOCOL

### Bij `systemctl list-units --failed`:

**STAP 1:** Toon output, GEEN interpretatie

**STAP 2:** Vraag ALTIJD:
```
"Ik zie X failed services:
- service-a
- service-b
- service-c

Welke zijn:
1. Deprecated (negeren)
2. Intentioneel uit (bijv. TenneT)
3. Daadwerkelijk broken (onderzoeken)
?"
```

**STAP 3:** WACHT op antwoord

**STAP 4:** Onderzoek ALLEEN wat Leo aanwijst als "broken"

### ⛔ NOOIT:

- Aannemen dat failed = moet gerepareerd
- Alle services tegelijk proberen te fixen
- TenneT services aanraken (BYO-KEY model, zie SKILL_02)

---

## SECTIE I: CC NETWERK & PERMISSIES

### CC draait op ENIN-NL server (NIET in sandbox)

CC heeft WEL:
- Internet toegang
- Git push/pull naar GitHub (via SSH)
- API calls naar externe services

**NIET zeggen:** "Je moet zelf pushen want ik heb geen internet"
**WEL doen:** Direct pushen na commit

### User Context

| Operatie | User | Command Prefix |
|----------|------|----------------|
| Git (status, pull, commit, push) | service user | `sudo -u energy-insights-nl` |
| File edits in repo | root | Direct na edit: `sudo chown -R energy-insights-nl:...` |
| systemctl (restart, status) | root | `sudo` |
| apt install | root | `sudo` |
| /etc/ configuratie | root | `sudo` |
| alembic migrations | service user | `sudo -u energy-insights-nl` |
| Python/pip in venv | service user | `sudo -u energy-insights-nl` |

**⚠️ File edits:** Root mag editen, maar ownership DIRECT fixen (zie Sectie G).

---

# DEEL 3: CLAUDE AI (CAI) SPECIFIEK

---

## SECTIE J: CAI VERANTWOORDELIJKHEDEN

### CAI doet WEL:

```
✅ Architectuur design en review
✅ Planning en projectmanagement
✅ Documentatie schrijven en structureren
✅ Code review (op basis van gedeelde code)
✅ SKILL updates en uitbreidingen
✅ ADR's opstellen
✅ Strategische adviezen
✅ Troubleshooting analyse (zonder server toegang)
```

### CAI doet NIET:

```
❌ Directe server toegang
❌ Git commits (geen repo toegang)
❌ Service restarts
❌ File edits op server
❌ Database queries
❌ API calls naar productie
```

### CAI's output = altijd voor Leo/CC om uit te voeren

---

# DEEL 4: SHARED KNOWLEDGE PROTOCOL

---

## SECTIE K: NAAMCONVENTIE

### Document Naming Pattern

```
[TYPE]_[BRON]_[BESCHRIJVING]_[DATUM].md

BRON codes:
- CC    → Claude Code gemaakt
- CAI   → Claude AI gemaakt  
- LEO   → Leo gemaakt
- MERGED → Geconsolideerd door Leo
```

### Per Document Type

| Type | Pattern | Locatie |
|------|---------|---------|
| Skills | `SKILL_##_[NAAM].md` | `docs/skills/` |
| Status CC | `STATUS_CC_CURRENT.md` | `docs/status/` |
| Status CAI | `STATUS_CAI_CURRENT.md` | `docs/status/` |
| Status SSOT | `STATUS_MERGED_CURRENT.md` | `docs/status/` |
| Actions | `NEXT_ACTIONS.md` | `docs/status/` |
| Sessie CC | `SESSIE_CC_[YYYYMMDD].md` | `docs/sessions/` |
| Sessie CAI | `SESSIE_CAI_[YYYYMMDD].md` | `docs/sessions/` |
| ADR | `ADR_###_[TITEL].md` | `docs/decisions/` |

### Voorbeelden

```
STATUS_CC_CURRENT.md              → CC's huidige status
STATUS_CAI_CURRENT.md             → CAI's huidige status  
STATUS_MERGED_CURRENT.md          → SSOT (Leo's merged versie)
SESSIE_CC_20260107.md             → CC sessie samenvatting
SESSIE_CAI_20260107.md            → CAI sessie samenvatting
ADR_009_TENNET_BYO_KEY.md         → Architecture Decision Record
```

---

## SECTIE L: DIRECTORY STRUCTUUR

### Officiële docs/ structuur

```
/opt/github/synctacles-api/docs/
│
├── README.md                           # Index van alle documentatie
│
├── skills/                             # SKILL documenten
│   ├── SKILL_00_AI_OPERATING_PROTOCOL.md
│   ├── SKILL_01_HARD_RULES.md
│   ├── SKILL_02_ARCHITECTURE.md
│   ├── SKILL_03_CODING_STANDARDS.md
│   ├── SKILL_04_PRODUCT_REQUIREMENTS.md
│   ├── SKILL_05_COMMUNICATION_RULES.md
│   ├── SKILL_06_DATA_SOURCES.md
│   ├── SKILL_08_HARDWARE_PROFILE.md
│   ├── SKILL_09_INSTALLER_SPECS.md
│   ├── SKILL_10_DEPLOYMENT_WORKFLOW.md
│   ├── SKILL_11_REPO_AND_ACCOUNTS.md
│   ├── SKILL_12_BRAND_FREE_ARCHITECTURE.md
│   └── SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md
│
├── status/                             # Live state files
│   ├── STATUS_MERGED_CURRENT.md        # SSOT (Leo's versie)
│   ├── STATUS_CC_CURRENT.md            # CC's laatste status
│   ├── STATUS_CAI_CURRENT.md           # CAI's laatste status
│   └── NEXT_ACTIONS.md                 # Geprioriteerde backlog
│
├── sessions/                           # Sessie samenvattingen
│   ├── README.md                       # Index + instructies
│   ├── SESSIE_CC_[YYYYMMDD].md
│   ├── SESSIE_CAI_[YYYYMMDD].md
│   └── archive/                        # Oudere sessies (>30 dagen)
│
├── decisions/                          # Architecture Decision Records
│   ├── README.md                       # ADR index + nummering
│   └── ADR_###_[TITEL].md
│
├── templates/                          # Reusable templates
│   ├── TEMPLATE_STATUS_CC.md
│   ├── TEMPLATE_STATUS_CAI.md
│   ├── TEMPLATE_SESSIE.md
│   ├── TEMPLATE_HANDOFF_CAI_CC.md
│   ├── TEMPLATE_HANDOFF_CC_CAI.md
│   └── TEMPLATE_ADR.md
│
├── CC_communication/                   # CC specifieke communicatie
├── operations/                         # Operationele docs
├── tasks/                              # Taak tracking
├── reports/                            # Rapporten
├── incidents/                          # Incident logs
├── api/                                # API specifieke docs
├── archived/                           # Deprecated docs
│
├── ARCHITECTURE.md                     # Systeem architectuur
├── api-reference.md                    # API documentatie
├── troubleshooting.md                  # Troubleshooting guide
├── user-guide.md                       # Gebruikershandleiding
└── SYSTEMD_SERVICES_ANALYSIS.md        # Service analyse
```

### Waar hoort wat?

| Content Type | Locatie |
|--------------|---------|
| Regels, procedures, standaarden | `docs/skills/` |
| Huidige project staat | `docs/status/` |
| Sessie verslagen | `docs/sessions/` |
| Architectuur beslissingen | `docs/decisions/` |
| Reusable templates | `docs/templates/` |
| API documentatie | `docs/api/` of root |
| CC specifieke zaken | `docs/CC_communication/` |
| Operationele zaken | `docs/operations/` |
| Oude/deprecated docs | `docs/archived/` |

---

## SECTIE M: DUAL STATUS MODEL

### Principe

```
┌─────────────────────────────────────────────────────┐
│                    LEO (Owner)                       │
│              STATUS_MERGED_CURRENT.md                │
│                 (Single Source of Truth)             │
│                    ▲      ▲                          │
│          ┌────────┘      └────────┐                  │
│          │                        │                  │
│  STATUS_CC_CURRENT.md    STATUS_CAI_CURRENT.md      │
│  ├─ Server state          ├─ Project context         │
│  ├─ Code changes          ├─ Architectural state     │
│  ├─ Git status            ├─ Planning status         │
│  ├─ Service health        ├─ Open beslissingen       │
│  └─ Runtime issues        └─ Dependencies            │
│                                                      │
│     Claude Code               Claude AI              │
└─────────────────────────────────────────────────────┘
```

### Workflow

1. **Start sessie:** Lees `STATUS_MERGED_CURRENT.md` (SSOT)
2. **Tijdens sessie:** Houd eigen status bij
3. **Einde sessie:** Update eigen `STATUS_[CC|CAI]_CURRENT.md`
4. **Leo merged:** Combineert tot nieuwe `STATUS_MERGED_CURRENT.md`
5. **Volgende sessie:** Lees nieuwe SSOT

### CC Status bevat:

```markdown
## STATUS_CC_CURRENT.md

### Server State
- Services: [running/failed/unknown]
- Disk: [usage]
- Last deploy: [timestamp]

### Code Changes (uncommitted)
- [ ] file1.py - [beschrijving]
- [ ] file2.md - [beschrijving]

### Git Status
- Branch: main
- Last commit: [hash] [message]
- Uncommitted: [yes/no]

### Open Issues
- [ ] Issue 1
- [ ] Issue 2

### Blocked By
- [dependencies]

### Last Updated
[timestamp] by CC
```

### CAI Status bevat:

```markdown
## STATUS_CAI_CURRENT.md

### Project Phase
- Current: [phase/sprint]
- Next milestone: [date/description]

### Architectural State
- Open decisions: [list]
- Recent ADRs: [list]

### Planning Status
- Sprint: [naam]
- Progress: [X/Y taken]

### Documentation State
- Updates needed: [list]
- Reviews pending: [list]

### Open Questions for Leo
- [ ] Question 1
- [ ] Question 2

### Blocked By
- [dependencies]

### Last Updated
[timestamp] by CAI
```

---

## SECTIE N: HANDOFF PROTOCOL

### Handoff Opslag

**Locatie:** `docs/handoffs/`

**Naamconventie:** `HANDOFF_[BRON]_[DOEL]_YYYYMMDD_[topic].md`

**Voorbeelden:**
- `HANDOFF_CAI_CC_20260108_p1_audit_fixes.md`
- `HANDOFF_CC_CAI_20260108_logging_review.md`

**Templates:** `docs/templates/TEMPLATE_HANDOFF_[CAI|CC]_[CC|CAI].md`

### Wanneer Handoff VERPLICHT

| Situatie | Handoff nodig? |
|----------|----------------|
| Sessie-einde met onafgerond werk | ✅ JA |
| Taak overdracht CC ↔ CAI | ✅ JA |
| Volledige taak afgerond, geen follow-up | ❌ NEE |
| Mini-taak < 5 min zonder context | ❌ NEE |

### CC → CAI Handoff

**Trigger:** CC klaar met taak, CAI input nodig (review, planning, docs)

**Template:** `docs/templates/TEMPLATE_HANDOFF_CC_CAI.md`

**CC levert in `docs/handoffs/HANDOFF_CC_CAI_YYYYMMDD_[topic].md`:**
```markdown
## HANDOFF: CC → CAI

### PRE-HANDOFF CHECKLIST
- [ ] Alle wijzigingen gecommit
- [ ] Services stabiel
- [ ] Geen blocking errors

### Completed Work
- [wat is gedaan]
- [welke files gewijzigd]

### Current State
- [server status]
- [git status]

### Needs from CAI
- [ ] Review van [X]
- [ ] Documentatie update voor [Y]
- [ ] Planning advies voor [Z]

### Context
- [relevante achtergrond]

### Files to Review
- path/to/file1.py
- path/to/file2.md

### POST-HANDOFF VERIFICATIE
CAI bevestigt: [ ] Ontvangen en begrepen
```

### CAI → CC Handoff

**Trigger:** CAI klaar met planning/docs, CC executie nodig

**Template:** `docs/templates/TEMPLATE_HANDOFF_CAI_CC.md`

**CAI levert in `docs/handoffs/HANDOFF_CAI_CC_YYYYMMDD_[topic].md`:**
```markdown
## HANDOFF: CAI → CC

### PRE-HANDOFF CHECKLIST
- [ ] Taak is concreet en uitvoerbaar
- [ ] Acceptance criteria gedefinieerd
- [ ] Relevante SKILLs geïdentificeerd
- [ ] Out of scope duidelijk

### Task Description
- [wat moet CC doen]
- [verwachte output]

### Specifications
- [technische details]
- [acceptance criteria]

### Files to Create/Modify
- [ ] path/to/file1.py - [instructies]
- [ ] path/to/file2.md - [instructies]

### Relevant SKILLs
- SKILL_XX voor [aspect]
- SKILL_YY voor [aspect]

### Out of Scope
- [wat NIET doen]
- [off-limits gebieden]

### Verification
- [ ] Test 1
- [ ] Test 2

### POST-HANDOFF VERIFICATIE
CC bevestigt: [ ] Ontvangen, begrepen, kan starten
```

### Leo → AI Handoff

**Leo specificeert:**
- Welke AI (CC of CAI)
- Taak beschrijving
- Prioriteit
- Deadline (indien van toepassing)
- Go/No-go voor uitvoering

### Handoff Archivering

**Retentie:** Handoffs ouder dan 30 dagen → `docs/archived/handoffs/`

**Cleanup:** Maandelijks door CC bij sessie-start

---

*Sectie N versie: 2.0 (2026-01-07)*
*Reden update: Templates verplicht gesteld, enforcement toegevoegd*

---

## SECTIE O: ADR PROTOCOL

### Wanneer ADR maken?

```
✅ ADR NODIG:
- Architectuur keuze met lange termijn impact
- Technologie selectie
- Data model beslissingen
- API design beslissingen
- Integratie patronen
- Security beslissingen

❌ GEEN ADR:
- Bug fixes
- Kleine refactors
- Documentatie updates
- Configuratie wijzigingen
```

### ADR Template

```markdown
# ADR-XXX: [Titel]

**Status:** Proposed | Accepted | Deprecated | Superseded
**Date:** YYYY-MM-DD
**Author:** Leo | CAI | CC
**Supersedes:** ADR-YYY (indien van toepassing)

## Context

Wat is het probleem of de beslissing die genomen moet worden?
Wat is de huidige situatie?

## Decision

Wat hebben we besloten?
Concrete, uitvoerbare beslissing.

## Consequences

### Positief
- [voordelen]

### Negatief
- [nadelen]

### Risico's
- [wat kan misgaan]

## Alternatives Considered

### Optie A: [naam]
- Beschrijving
- Waarom niet gekozen

### Optie B: [naam]
- Beschrijving
- Waarom niet gekozen

## Implementation

- [ ] Stap 1
- [ ] Stap 2
- [ ] Stap 3

## References

- [links naar relevante docs]
- [SKILLs die geraakt worden]
```

### ADR Nummering

```
ADR_001 t/m ADR_008  → Bestaand (implicit in SKILLs)
ADR_009+             → Nieuwe beslissingen
```

### ADR Workflow

1. **CAI** stelt ADR op (Proposed)
2. **Leo** reviewt en keurt goed
3. **Status** → Accepted
4. **CC** implementeert (indien nodig)
5. **Update** relevante SKILLs

---

## SECTIE P: VERANTWOORDELIJKHEDEN MATRIX

### Beslissingsboom

```
┌─────────────────────────────────────────────────────┐
│ BESLISSING NODIG                                    │
├─────────────────────────────────────────────────────┤
│                                                     │
│ Architectuur/Design?  ─────► CAI adviseert          │
│                              Leo beslist            │
│                              CC executes            │
│                                                     │
│ Code implementatie?   ─────► CC doet                │
│                              CAI review (optioneel) │
│                              Leo approves           │
│                                                     │
│ Productie impact?     ─────► Leo ALTIJD approve     │
│                              Geen uitzonderingen    │
│                                                     │
│ Quick fix < 15 min?   ─────► CC mag uitvoeren       │
│                              MITS: verified +       │
│                              documented             │
│                                                     │
│ Scope change?         ─────► CAI signaleert         │
│                              Leo beslist            │
│                                                     │
│ Off-limits gebied?    ─────► STOP. Vraag Leo.       │
│                              Geen uitzonderingen    │
│                                                     │
│ Documentatie?         ─────► CAI schrijft           │
│                              CC commit              │
│                              Leo reviews            │
│                                                     │
│ Planning/Roadmap?     ─────► CAI stelt voor         │
│                              Leo beslist            │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### RACI Matrix

| Activiteit | Leo | CAI | CC |
|------------|-----|-----|-----|
| Architectuur beslissingen | A | R | I |
| Code schrijven | A | C | R |
| Code review | A | R | I |
| Git commits | A | I | R |
| Service restarts | A | I | R |
| SKILL updates | A | R | C |
| ADR schrijven | A | R | C |
| Planning | A | R | I |
| Prioriteiten | R/A | C | I |
| Documentatie | A | R | C |
| Troubleshooting | A | C | R |

**R** = Responsible (doet het werk)
**A** = Accountable (eindverantwoordelijk)
**C** = Consulted (input gevraagd)
**I** = Informed (op de hoogte gehouden)

---

## SECTIE Q: SESSIE SAMENVATTING TEMPLATE

### Voor significante sessies (>1 uur of belangrijke wijzigingen)

```markdown
# SESSIE SAMENVATTING

**Datum:** YYYY-MM-DD
**Bron:** CC | CAI
**Duur:** X uur
**Focus:** [hoofdonderwerp]

---

## UITGEVOERDE WERK

### Completed
- [x] Taak 1 - [beschrijving]
- [x] Taak 2 - [beschrijving]

### Gewijzigde Files
| File | Actie | Beschrijving |
|------|-------|--------------|
| path/file.py | Modified | [wat] |
| path/new.md | Created | [wat] |

### Git Commits
- `abc1234` - [message]
- `def5678` - [message]

---

## BESLISSINGEN

| Beslissing | Rationale | ADR? |
|------------|-----------|------|
| [beslissing] | [waarom] | Nee / ADR-XXX |

---

## OPEN ITEMS

### Blocked
- [ ] [item] - blocked by [wat]

### TODO (volgende sessie)
- [ ] [item]
- [ ] [item]

### Vragen voor Leo
- [ ] [vraag]

---

## HANDOFF NOTES

### Voor CC (indien CAI sessie)
- [instructies voor CC]

### Voor CAI (indien CC sessie)
- [input nodig van CAI]

---

## STATUS UPDATE

[Korte update voor STATUS_[CC|CAI]_CURRENT.md]
```

---

# DEEL 5: QUICK REFERENCE

---

## SECTIE R: QUICK REFERENCE CARD

```
┌─────────────────────────────────────────────────────┐
│  AI OPERATING PROTOCOL - QUICK REFERENCE            │
├─────────────────────────────────────────────────────┤
│                                                     │
│  1. LEES SKILLS EERST (00, 01, 02, 11 minimum)     │
│  2. BEWIJS DAT JE ZE GELEZEN HEBT                  │
│  3. LEES STATUS_MERGED_CURRENT.md                   │
│  4. PROTECT MODE = DEFAULT                          │
│  5. VERIFICATIE VOOR CONCLUSIES                     │
│  6. VRAAG VOOR ACTIES                               │
│  7. FAILED ≠ MOET GEREPAREERD                       │
│  8. TENNET = OFF-LIMITS                             │
│  9. GIT = ALTIJD ALS SERVICE USER (CC)              │
│ 10. CHOWN DIRECT NA FILE EDITS (CC)                 │
│ 11. BIJ TWIJFEL: STOP EN VRAAG                      │
│ 12. GEEN "IK FIX EVEN"                              │
│ 13. UPDATE STATUS BIJ SESSIE EINDE                  │
│                                                     │
├─────────────────────────────────────────────────────┤
│  NAAMCONVENTIE                                      │
├─────────────────────────────────────────────────────┤
│  STATUS_CC_CURRENT.md    → CC's status              │
│  STATUS_CAI_CURRENT.md   → CAI's status             │
│  STATUS_MERGED_CURRENT.md → SSOT (Leo)              │
│  SESSIE_[CC|CAI]_YYYYMMDD.md → Sessie log          │
│  ADR_###_[TITEL].md      → Decision record          │
│                                                     │
├─────────────────────────────────────────────────────┤
│  DIRECTORY STRUCTUUR                                │
├─────────────────────────────────────────────────────┤
│  docs/skills/    → SKILL documenten                 │
│  docs/status/    → Live state files                 │
│  docs/sessions/  → Sessie samenvattingen            │
│  docs/decisions/ → ADRs                             │
│  docs/templates/ → Reusable templates               │
│                                                     │
├─────────────────────────────────────────────────────┤
│  HANDOFF                                            │
├─────────────────────────────────────────────────────┤
│  CC → CAI: Completed work + needs from CAI          │
│  CAI → CC: Task specs + files to modify             │
│  Both: Update own STATUS file                       │
│  Leo: Merge to SSOT                                 │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## SECTIE S: GERELATEERDE SKILLS

| SKILL | Focus |
|-------|-------|
| SKILL_01 | Hard rules, fail-fast, KISS |
| SKILL_02 | Architecture, TenneT BYO-KEY |
| SKILL_03 | Coding standards |
| SKILL_05 | Communication rules |
| SKILL_06 | Data sources |
| SKILL_09 | Installer specs |
| SKILL_10 | Deployment workflow |
| SKILL_11 | Repo structure, git discipline |
| SKILL_13 | Logging, diagnostics |

---

## SECTIE T: CONSEQUENCES

### Bij overtreding van dit protocol:

1. **Leo stopt de sessie**
2. **Wijzigingen worden teruggedraaid**
3. **Tijd is verspild**

### De 4.5-uur incident bewees:

- AI las SKILLs niet → TenneT rabbit hole (45 min verspild)
- AI verifieerde niet → Verkeerde script geblamed (60 min verspild)
- AI vroeg niet → Productie bijna gesloopt
- AI nam aan → 3.5 uur verspild van 4.5 uur totaal

**Dit protocol bestaat om herhaling te voorkomen.**

---

**LAATSTE WAARSCHUWING:**

Dit protocol is geen suggestie. Het is een vereiste.
Elke sessie begint met bewijs dat je dit gelezen hebt.
Geen uitzonderingen.

---

**Document Owner:** Leo
**Enforcement:** Strict
**Version:** 2.0
**Last Updated:** 2026-01-07
