# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Sprint:** 1 Phase 4 (FINAL)

---

## TASK DESCRIPTION

Consolideer documentatie van 21 → 17 files door archivering van obsolete/duplicate content.

**Exit Criteria:**
- 4 files gearchiveerd
- README.md updated als index
- Git commit met alle changes

---

## FILES TO ARCHIVE

Verplaats naar `docs/archived/`:

| File | Nieuwe locatie | Reden |
|------|----------------|-------|
| `SESSIE_SAMENVATTING_20260102.md` | `docs/archived/sessions/` | Oud sessie log |
| `HA_CUSTOMIZATION_CONTEXT.md` | `docs/archived/` | Sessie-specifiek, niet meer actueel |
| `SKILL_00_CC_OPERATING_PROTOCOL.md` | `docs/archived/` | Superseded door SKILL_00_AI v2.0 |
| `ARCHITECTURE.md` | `docs/archived/` | 90% overlap met SKILL_02 |

---

## FILES TO MODIFY

### 1. README.md - Update als Documentation Index

Voeg sectie toe met navigatie naar alle actuele docs:

```markdown
## Documentation Index

### Core SKILLs (Mandatory Reading)
| SKILL | Focus | Priority |
|-------|-------|----------|
| SKILL_00 | AI Operating Protocol | CC+CAI: Always |
| SKILL_01 | Hard Rules | Everyone: Always |
| SKILL_02 | Architecture | Developers: Always |
| SKILL_11 | Repo & Accounts | CC: Always |

### Reference Documentation
- api-reference.md - API endpoints
- troubleshooting.md - Common issues
- user-guide.md - End user guide

### All SKILLs
[existing SKILL listing - already in README]
```

---

## EXECUTION STEPS

```bash
SERVICE_USER="energy-insights-nl"
REPO="/opt/github/synctacles-api"

# 1. Ensure directories exist
mkdir -p $REPO/docs/archived/sessions

# 2. Archive files (vanuit repo root, niet /mnt/project)
git -C $REPO mv docs/SESSIE_SAMENVATTING_20260102.md docs/archived/sessions/ 2>/dev/null || \
  mv $REPO/SESSIE_SAMENVATTING_20260102.md $REPO/docs/archived/sessions/ 2>/dev/null

git -C $REPO mv docs/HA_CUSTOMIZATION_CONTEXT.md docs/archived/ 2>/dev/null || \
  mv $REPO/HA_CUSTOMIZATION_CONTEXT.md $REPO/docs/archived/ 2>/dev/null

git -C $REPO mv docs/SKILL_00_CC_OPERATING_PROTOCOL.md docs/archived/ 2>/dev/null || \
  mv $REPO/SKILL_00_CC_OPERATING_PROTOCOL.md $REPO/docs/archived/ 2>/dev/null

git -C $REPO mv docs/ARCHITECTURE.md docs/archived/ 2>/dev/null || \
  mv $REPO/ARCHITECTURE.md $REPO/docs/archived/ 2>/dev/null

# 3. Fix ownership
sudo chown -R $SERVICE_USER:$SERVICE_USER $REPO/

# 4. Update README.md met index sectie
# [Edit README.md - add Documentation Index section]

# 5. Git commit
sudo -u $SERVICE_USER git -C $REPO add .
sudo -u $SERVICE_USER git -C $REPO commit -m "docs: Phase 4 cleanup - consolidate 21→17 files

Archived obsolete/duplicate documentation:
- SESSIE_SAMENVATTING_20260102.md (old session log)
- HA_CUSTOMIZATION_CONTEXT.md (session-specific, outdated)
- SKILL_00_CC_OPERATING_PROTOCOL.md (superseded by SKILL_00_AI v2.0)
- ARCHITECTURE.md (90% overlap with SKILL_02)

Updated README.md with documentation index.

Sprint 1 Phase 4: COMPLETE"

# 6. Push
sudo -u $SERVICE_USER git -C $REPO push origin main
```

---

## VERIFICATION

Na uitvoering:
```bash
# File count check
find $REPO -maxdepth 1 -name "*.md" -type f | wc -l
# Expected: 17 of minder in root

# Archived files exist
ls $REPO/docs/archived/
# Expected: 4 nieuwe files

# Git status clean
sudo -u $SERVICE_USER git -C $REPO status
# Expected: nothing to commit
```

---

## OUT OF SCOPE

- GEEN SKILL content wijzigingen
- GEEN file renames (alleen moves)
- GEEN nieuwe documentatie schrijven

---

## RELEVANT SKILLS

- SKILL_00 Sectie G: Git discipline (chown na edits)
- SKILL_11: Repo structure, git workflow

---

## NOTES

- Files in `/mnt/project/` zijn READ-ONLY copies
- Werk in `/opt/github/synctacles-api/`
- Check eerst waar files daadwerkelijk staan op server

---

**Status:** Ready for CC execution
