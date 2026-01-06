# 🧹 Opschonings- & Herindelingsplan Documentatie

**Datum:** 6 januari 2026
**Doel:** Alles onder `/docs`, verwijderen wat overbodig is
**Status:** Plan gereed voor uitvoering

---

## 📊 HUIDIGE SITUATIE

### Root Level (6 .md files)
```
/opt/github/synctacles-api/
├── README.md                                      ← Project intro
├── CHANGELOG.md                                   ← Version history
├── SYNCTACLES_PROJECT_VOORTGANGSRAPPORT.md       ← Status report (50+ pages)
├── SYNCTACLES_GLOBALE_POSITIE.md                 ← Strategy & launch (70+ pages)
├── CC_FEEDBACK_ON_CLAUDE_INVENTARISATIE.md       ← Analysis feedback
└── DOCUMENTATIE_INDEX.md                         ← This index
```

### /docs Structure (Current)
```
/docs/
├── ARCHITECTURE.md                               ✅ Keep
├── user-guide.md                                 ✅ Keep
├── api-reference.md                              ✅ Keep
├── troubleshooting.md                            ✅ Keep
├── DEPLOYMENT_SCRIPTS_GUIDE.md                   ⚠️ MOVE to /docs/operations/
├── SKILL_08_HARDWARE_PROFILE.md                  ⚠️ MOVE to /docs/skills/
├── SKILL_09_INSTALLER_SPECS.md                   ⚠️ MOVE to /docs/skills/
├── ENERGY_CHARTS_*.md (4 files)                  ⚠️ CONSOLIDATE to /docs/api/
├── /skills/                                      ✅ Keep & organize
├── /CC_communication/                            ✅ Keep (issue tracking)
├── /incidents/                                   ✅ Keep
├── /reports/                                     ✅ Keep
├── /api/                                         ✅ Keep
└── /github/                                      ⚠️ OPTIONAL (low value)
```

---

## 🎯 OPSCHONINGSPLAN

### FASE 1: VERPLAATSEN & CONSOLIDEREN

#### 1.1 Verplaats Root-Level Documenten naar /docs

**Documenten om te verplaatsen:**

| Document | Destination | Reden | Action |
|----------|-------------|-------|--------|
| SYNCTACLES_PROJECT_VOORTGANGSRAPPORT.md | `/docs/reports/` | Status report | Move + rename to `STATUS_COMPLETE_2026-01-06.md` |
| SYNCTACLES_GLOBALE_POSITIE.md | `/docs/reports/` | Strategy | Move + rename to `STRATEGY_LAUNCH_2026-01-06.md` |
| CC_FEEDBACK_ON_CLAUDE_INVENTARISATIE.md | `/docs/reports/` | Analysis | Move + rename to `ANALYSIS_FEEDBACK_2026-01-06.md` |
| DOCUMENTATIE_INDEX.md | `/docs/` | Index | Keep in root (primary reference) |
| DEPLOYMENT_SCRIPTS_GUIDE.md | `/docs/operations/` | Operations | Move (create new folder) |

**Documenten om te HOUDEN in root:**

| Document | Reden |
|----------|-------|
| README.md | GitHub default entry point |
| CHANGELOG.md | Version history (convention) |

---

#### 1.2 Consolideer ENERGY_CHARTS Documentatie

**Huidige situatie:**
```
/docs/
├── ENERGY_CHARTS_API_RESPONSE_FORMAT.md
├── ENERGY_CHARTS_INTEGRATION_GUIDE.md
├── ENERGY_CHARTS_API_SUMMARY.md
├── ENERGY_CHARTS_DOCUMENTATION_INDEX.md
```

**Consolidatie plan:**
1. Merge alle 4 bestanden naar: `/docs/api/ENERGY_CHARTS_INTEGRATION.md`
2. Structure:
   - Summary
   - Integration guide
   - API response format
   - Troubleshooting

**Result:** 1 bestand ipv 4

---

#### 1.3 Verplaats & Deduplicate SKILL_* Files

**Huidige duplicates:**
```
/docs/SKILL_08_HARDWARE_PROFILE.md        (in root)
/docs/skills/SKILL_08_HARDWARE_PROFILE.md (in skills)

/docs/SKILL_09_INSTALLER_SPECS.md         (in root)
/docs/skills/SKILL_09_INSTALLER_SPECS.md  (in skills)
```

**Plan:**
- Delete root-level copies
- Keep only in `/docs/skills/`
- Update all references

---

#### 1.4 Maak `/docs/operations/` Folder

**Inhoud:**
```
/docs/operations/
├── DEPLOYMENT_SCRIPTS_GUIDE.md    (moved from root)
├── BACKUP_STRATEGY.md              (extract from SKILL_10)
├── MONITORING_SETUP.md             (extract from SKILL_13)
└── RUNBOOKS.md                     (extract from incident reports)
```

---

### FASE 2: VERWIJDER OVERBODIG

#### 2.1 Verwijder of Merge `/docs/github/`

**Huidige inhoud:**
```
/docs/github/github_setup_and_commands.md
```

**Evaluatie:**
- ❌ Low value (Git basics are standard knowledge)
- ⚠️ Could be merged into README.md as "Contributing" section
- Recommendation: **DELETE** (or extract to CONTRIBUTING.md in root if needed)

---

#### 2.2 Evalueer `/docs/reports/` Content

**Huidige inhoud:**
```
LOAD_TEST_REPORT.md                  ✅ KEEP (performance baseline)
OPTIMIZATION_RESULTS.md              ✅ KEEP (past work)
WERKELIJK_QUERY_RESULTATEN.md        ❓ UNCLEAR
FASE_*.md (3 files)                  ⚠️ ARCHIVE or DELETE?
CC_PLAN_TENNET_BYO_MIGRATION.md      ⚠️ Historical (TenneT migration)
VOORTGANGSRAPPORT_TENNET_MIGRATION.md ⚠️ Historical (TenneT migration)
```

**Recommendation:**
- **KEEP:**
  - LOAD_TEST_REPORT.md (performance baseline)
  - OPTIMIZATION_RESULTS.md (what we optimized)

- **ARCHIVE to `/docs/archived/historical/`:**
  - FASE_1_COMPLETION_REPORT.md
  - FASE_2_COMPLETION_REPORT.md
  - FASE_3_SPECIFICATION.md
  - CC_PLAN_TENNET_BYO_MIGRATION.md
  - VOORTGANGSRAPPORT_TENNET_MIGRATION.md
  - WERKELIJK_QUERY_RESULTATEN.md

**Reden:** These are historical records of completed work, not operational docs.

---

#### 2.3 Evalueer `/docs/incidents/` Content

**Huidige inhoud:**
```
cc-incident-report-fallback-outage.md    ✅ KEEP (proof of fallback)
cc-diagnostic-a44-tennet.md              ✅ KEEP (diagnostic reference)
cc-action-report-a44-fix.md              ✅ KEEP (how we fixed it)
```

**Decision:** ✅ **KEEP ALL** (incident documentation is valuable)

---

### FASE 3: REORGANISEREN

#### 3.1 Nieuwe Folder Structure

```
/docs/
├── README.md                          ← Entry point
├── ARCHITECTURE.md                    ← System design
├── api-reference.md                   ← API endpoints
├── user-guide.md                      ← User instructions
├── troubleshooting.md                 ← Problem solving
│
├── /skills/                           ← Operational procedures
│   ├── README.md                      (overview)
│   ├── SKILL_01_HARD_RULES.md
│   ├── SKILL_02_ARCHITECTURE.md
│   ├── ... (all SKILL_* files)
│   └── SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md
│
├── /operations/                       ← NEW: Deployment & Operations
│   ├── DEPLOYMENT_SCRIPTS_GUIDE.md
│   ├── BACKUP_STRATEGY.md
│   ├── MONITORING_SETUP.md
│   └── RUNBOOKS.md
│
├── /api/                              ← API integrations
│   ├── signals.md
│   └── ENERGY_CHARTS_INTEGRATION.md   (consolidated)
│
├── /CC_communication/                 ← Issue tracking
│   ├── INDEX.md
│   └── CC_TASK_*.md (all issues)
│
├── /incidents/                        ← Incident reports
│   ├── cc-incident-report-fallback-outage.md
│   ├── cc-diagnostic-a44-tennet.md
│   └── cc-action-report-a44-fix.md
│
├── /reports/                          ← Technical reports
│   ├── LOAD_TEST_REPORT.md
│   ├── OPTIMIZATION_RESULTS.md
│   └── STATUS_COMPLETE_2026-01-06.md  (moved from root)
│
└── /archived/                         ← Historical records
    └── /historical/
        ├── FASE_1_COMPLETION_REPORT.md
        ├── FASE_2_COMPLETION_REPORT.md
        ├── FASE_3_SPECIFICATION.md
        ├── TENNET_MIGRATION_PLAN.md
        ├── TENNET_MIGRATION_REPORT.md
        └── QUERY_RESULTS.md
```

---

## ✅ ACTIE ITEMS (In Volgorde)

### Stap 1: Backup Maken
```bash
git add .
git commit -m "docs: backup before reorganization"
git tag backup-2026-01-06
```

### Stap 2: Verwijder Root-Level Duplicates
```bash
rm /docs/SKILL_08_HARDWARE_PROFILE.md
rm /docs/SKILL_09_INSTALLER_SPECS.md
```

### Stap 3: Consolideer ENERGY_CHARTS
```bash
# Merge 4 bestanden → 1
# mv /docs/ENERGY_CHARTS_*.md → /docs/api/ENERGY_CHARTS_INTEGRATION.md
```

### Stap 4: Maak Nieuwe Folders
```bash
mkdir -p /docs/operations
mkdir -p /docs/archived/historical
```

### Stap 5: Verplaats Bestanden
```bash
# Root → /docs/reports
mv SYNCTACLES_PROJECT_VOORTGANGSRAPPORT.md → /docs/reports/STATUS_COMPLETE_2026-01-06.md
mv SYNCTACLES_GLOBALE_POSITIE.md → /docs/reports/STRATEGY_LAUNCH_2026-01-06.md
mv CC_FEEDBACK_ON_CLAUDE_INVENTARISATIE.md → /docs/reports/ANALYSIS_FEEDBACK_2026-01-06.md

# Root → /docs/operations
mv DEPLOYMENT_SCRIPTS_GUIDE.md → /docs/operations/

# /docs/reports → archived
mv FASE_*.md → /docs/archived/historical/
mv CC_PLAN_TENNET_*.md → /docs/archived/historical/
mv VOORTGANGSRAPPORT_TENNET_*.md → /docs/archived/historical/
mv WERKELIJK_QUERY_RESULTATEN.md → /docs/archived/historical/
```

### Stap 6: Verwijder Lage-Waarde Folders
```bash
# Option A: Delete entirely
rm -rf /docs/github/

# Option B: Extract to root (if needed)
# Create /CONTRIBUTING.md with git/github info
```

### Stap 7: Update References
- Update all internal links in documents
- Update DOCUMENTATIE_INDEX.md
- Update any hardcoded file references

### Stap 8: Create New Index Docs

**`/docs/README.md`** (Entry point)
```markdown
# SYNCTACLES Documentation

- [Architecture](ARCHITECTURE.md) - System design
- [API Reference](api-reference.md) - All endpoints
- [User Guide](user-guide.md) - How to use
- [Troubleshooting](troubleshooting.md) - Problem solving

## By Category
- [Skills & Procedures](skills/) - How to do things
- [Operations](operations/) - Deployment & monitoring
- [API Integrations](api/) - Data sources
- [Issue Tracking](CC_communication/) - All issues
- [Incident Reports](incidents/) - Past incidents
- [Technical Reports](reports/) - Performance & status
```

### Stap 9: Create `/docs/operations/README.md`
```markdown
# Operations & Deployment

- [Deployment Scripts](DEPLOYMENT_SCRIPTS_GUIDE.md) - How to deploy
- [Backup Strategy](BACKUP_STRATEGY.md) - Data protection
- [Monitoring Setup](MONITORING_SETUP.md) - Observability
- [Runbooks](RUNBOOKS.md) - Emergency procedures
```

### Stap 10: Update Root Files
- Keep README.md (GitHub convention)
- Keep CHANGELOG.md (version history)
- Keep DOCUMENTATIE_INDEX.md (if still useful, or deprecate)

### Stap 11: Final Commit
```bash
git add -A
git commit -m "docs: reorganize documentation structure

- Move root docs to /docs/reports
- Create /docs/operations for deployment docs
- Consolidate ENERGY_CHARTS files
- Archive historical reports
- Remove low-value /docs/github folder
- Update all references and indexes"
```

---

## 📊 RESULTAAT NA OPSCHONING

### Voordelen
- ✅ Alles onder `/docs/` (clean root)
- ✅ Duidelijk georganiseerd (skills, operations, api, etc)
- ✅ Historische docs gearchiveerd (niet in main flow)
- ✅ Geen duplicates
- ✅ Gemakkelijker navigeren

### Statistieken
- **Before:** 55+ files, root level is messy
- **After:** ~45 files, root level clean, `/docs` organized

### File Count Reduction
```
Root level:
  Before: 6 .md files
  After:  2 .md files (README, CHANGELOG)

/docs level:
  Before: Scattered structure
  After:  Clear hierarchy with /operations, /archived/

Total consolidation:
  4 ENERGY_CHARTS files → 1
  Phase reports → archived
  Duplicates removed
```

---

## ⚠️ VOORZICHTIGHEID

### Niet Verwijderen
- ❌ Do NOT delete CC_communication (issue tracking is valuable)
- ❌ Do NOT delete incidents (proof of system resilience)
- ❌ Do NOT delete SKILL_* docs (operational procedures)

### Well-Maintained During Reorganization
- ✅ Git history is preserved (moves, not deletes)
- ✅ Update all cross-references
- ✅ Test all links work after move
- ✅ Update CI/CD if any docs are referenced

### Rollback Plan
```bash
# If something goes wrong:
git reset --hard backup-2026-01-06
```

---

## 🎯 SUMMARY

| Action | Items | Impact |
|--------|-------|--------|
| Move to /docs/reports | 3 large reports | Clean root |
| Create /docs/operations | 1 moved + 3 new | Better operations visibility |
| Consolidate ENERGY_CHARTS | 4 → 1 | Easier to maintain |
| Archive historical | 6 files | Focus on current docs |
| Delete duplicates | 2 files | No redundancy |
| Delete low-value | /docs/github | Cleaner structure |
| **TOTAL:** | ~12 files changed | Well-organized, clean |

---

**Status:** Plan complete & ready to execute
**Estimated effort:** 2-3 hours (including testing links)
**Risk level:** LOW (git history preserved, rollback available)

Zal ik dit plan uitvoeren? 🚀
