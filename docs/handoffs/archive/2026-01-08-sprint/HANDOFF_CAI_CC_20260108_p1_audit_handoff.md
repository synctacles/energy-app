# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Taak:** Documentation Audit P1 Fixes + Handoff Infrastructure
**Prioriteit:** P1
**Geschatte tijd:** 45 min

---

## PRE-HANDOFF CHECKLIST (CAI verifieert)

- [x] Taak is concreet en uitvoerbaar
- [x] Acceptance criteria gedefinieerd
- [x] Relevante SKILLs geïdentificeerd
- [x] Out of scope duidelijk afgebakend
- [x] Geen open architectuur vragen

---

## TASK DESCRIPTION

### Wat moet CC doen

**Deel A: Handoff Infrastructure**
1. Maak directory `docs/handoffs/`
2. Update SKILL_00 Sectie N met nieuwe content (zie bijlage)
3. Plaats deze handoff in `docs/handoffs/`

**Deel B: P1 Audit Fixes**
4. Inventory check: `ls docs/*.md`
5. SKILL_07 in .gitignore toevoegen
6. Archiveer completed CC tasks (01-07) naar `docs/archived/cc_tasks/`

**Deel C: Commit**
7. Git commit + push met alle wijzigingen

### Verwachte output
- `docs/handoffs/` directory bestaat
- SKILL_00 Sectie N bevat handoff opslag locatie
- CC_communication/ bevat alleen actieve taken (08, 09, 10)
- Alles gecommit naar git

---

## SPECIFICATIONS

### Technische details

**Stap 1: Handoffs directory**
```bash
mkdir -p /opt/github/synctacles-api/docs/handoffs/
```

**Stap 2: SKILL_00 update**
Vervang regel 592-664 in `docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md` met content uit `SKILL_00_SECTIE_N_UPDATE.md` (bijlage van Leo).

**Stap 3: Plaats deze handoff**
```bash
# Leo levert handoff file, CC plaatst in:
/opt/github/synctacles-api/docs/handoffs/HANDOFF_CAI_CC_20260108_p1_audit_handoff.md
```

**Stap 4: Inventory**
```bash
ls -la /opt/github/synctacles-api/docs/*.md
```

**Stap 5: SKILL_07 in .gitignore**
```bash
echo "docs/skills/SKILL_07_PERSONAL_PROFILE.md" >> /opt/github/synctacles-api/.gitignore
```

**Stap 6: Archiveer CC tasks**
```bash
mkdir -p /opt/github/synctacles-api/docs/archived/cc_tasks/
mv /opt/github/synctacles-api/docs/CC_communication/CC_TASK_0[1-7]*.md \
   /opt/github/synctacles-api/docs/archived/cc_tasks/
```

**Stap 7: Git commit**
```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: handoff infrastructure + P1 audit fixes

- Add docs/handoffs/ for handoff storage
- Update SKILL_00 Sectie N with handoff location
- Add SKILL_07 to gitignore (personal info)
- Archive completed CC_TASK_01-07"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

### Files to create/modify

| File | Actie | Instructies |
|------|-------|-------------|
| `docs/handoffs/` | Create | Nieuwe directory |
| `docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md` | Modify | Vervang Sectie N (592-664) |
| `docs/handoffs/HANDOFF_CAI_CC_20260108_p1_audit_handoff.md` | Create | Deze file |
| `.gitignore` | Modify | Voeg SKILL_07 path toe |
| `docs/archived/cc_tasks/` | Create | Nieuwe directory |
| `docs/CC_communication/CC_TASK_01-07*.md` | Move | Naar archived/cc_tasks/ |

---

## RELEVANT SKILLS

| SKILL | Sectie | Waarom relevant |
|-------|--------|-----------------|
| SKILL_00 | Sectie G | Git discipline, chown na edits |
| SKILL_00 | Sectie N | Handoff protocol (wordt geüpdatet) |
| SKILL_11 | Git workflow | Service user voor git operaties |

---

## OUT OF SCOPE

**CC doet NIET:**
- SKILL_07 content aanmaken (Leo doet zelf)
- P2 acties (reports consolidatie, ADR nummering)
- Content wijzigen in gearchiveerde files
- Templates wijzigen (alleen SKILL_00 Sectie N)

**Bij twijfel:** Stop en vraag Leo

---

## VERIFICATION

### Tests door CC

```bash
# 1. Handoffs directory exists
test -d /opt/github/synctacles-api/docs/handoffs/ && echo "✓ handoffs dir"

# 2. SKILL_07 in gitignore
grep "SKILL_07" /opt/github/synctacles-api/.gitignore && echo "✓ gitignore"

# 3. Archived tasks count
ls /opt/github/synctacles-api/docs/archived/cc_tasks/ | wc -l
# Expected: 7

# 4. Remaining CC tasks
ls /opt/github/synctacles-api/docs/CC_communication/CC_TASK* | wc -l
# Expected: 3 (08, 09, 10)

# 5. Handoff file exists
test -f /opt/github/synctacles-api/docs/handoffs/HANDOFF_CAI_CC_20260108_p1_audit_handoff.md && echo "✓ handoff"

# 6. Git status clean
sudo -u energy-insights-nl git -C /opt/github/synctacles-api status
# Expected: nothing to commit
```

### Success indicators
- Alle 6 verificatie checks passed
- Git push succesvol
- Geen errors in output

---

## CONTEXT

**Waarom deze taak:**
- CAI hield zich niet aan eigen handoff regels
- SKILL_00 Sectie N miste opslag locatie voor handoffs
- Phase 3 Documentation Audit identificeerde cleanup nodig

**Phase 3 Audit bevindingen:**
- SKILL_07 referenties zonder file → gitignore oplossing
- CC_TASK_01-07 completed maar niet gearchiveerd
- Handoff locatie niet gedefinieerd in SKILL_00

---

## POST-HANDOFF VERIFICATIE

**CC bevestigt ontvangst:**
- [ ] Taak begrepen
- [ ] Scope duidelijk  
- [ ] Kan starten zonder verdere input
- [ ] SKILL_00, SKILL_11 gelezen

---

*Handoff gemaakt volgens: docs/templates/TEMPLATE_HANDOFF_CAI_CC.md*
*Opslag locatie: docs/handoffs/HANDOFF_CAI_CC_20260108_p1_audit_handoff.md*
