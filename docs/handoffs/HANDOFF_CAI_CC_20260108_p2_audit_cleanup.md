# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Onderwerp:** P2 Audit - Cleanup

---

## PRE-HANDOFF CHECKLIST

- [x] Taak is duidelijk gedefinieerd
- [x] Acceptance criteria zijn concreet
- [x] Geen open vragen die blokkeren
- [x] Relevante SKILLs geïdentificeerd

---

## TASK DESCRIPTION

P2 audit cleanup: ADR nummering fix, reports archiveren, SESSIE_CAI requirement schrappen.

**Verwachte output:**
- SKILL_00 bijgewerkt (2 wijzigingen)
- reports/ opgeschoond
- Git commit gepusht

---

## SPECIFICATIONS

### Deel A: ADR nummering fix

**Locatie:** `docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md`
**Sectie:** O (ADR Protocol), rond regel 750-760

**Zoek naar tekst zoals:**
```
ADR_001 t/m ADR_008  → Bestaand (implicit in SKILLs)
ADR_009+             → Nieuwe beslissingen
```

**Vervang door:**
```
ADR nummering start bij volgende beschikbare nummer.
Check docs/decisions/ voor hoogste bestaande nummer.
```

**Rationale:** "Implicit ADRs" bestaan niet - alleen gedocumenteerde beslissingen tellen.

---

### Deel B: reports/ archiveren

```bash
cd /opt/github/synctacles-api/docs

# 1. Maak archive directory
mkdir -p archived/reports

# 2. Bekijk wat er is (sorteer op datum)
ls -lt reports/

# 3. Behoud ALLEEN de nieuwste file
# Move rest naar archived/reports/
# Voorbeeld (pas aan op basis van ls output):
mv reports/[OUDE_FILES] archived/reports/
```

**Criteria:** Alleen meest recente report blijft in docs/reports/

---

### Deel C: SESSIE_CAI requirement schrappen

**Locatie:** `docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md`
**Sectie:** F (Session Checklist), rond regel 220

**Zoek naar regel met:** `SESSIE_CAI` of vergelijkbare verplichting voor CAI sessie logging

**Verwijder of pas aan:** CAI heeft geen persistent state, dus geen sessie logging requirement.

**Let op:** SESSIE_CC blijft WEL verplicht (CC heeft wel persistent server state)

---

## FILES TO CREATE/MODIFY

| File | Actie | Beschrijving |
|------|-------|--------------|
| docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md | Modify | Sectie O: ADR nummering |
| docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md | Modify | Sectie F: SESSIE_CAI schrappen |
| docs/archived/reports/ | Create | Archive directory |
| docs/reports/* | Move | Oude reports naar archive |

---

## RELEVANT SKILLS

- **SKILL_00 Sectie G:** Git discipline (chown na edits)
- **SKILL_00 Sectie O:** ADR Protocol (wat we aanpassen)
- **SKILL_11:** Git workflow (commit als service user)

---

## OUT OF SCOPE

- Geen nieuwe ADRs schrijven
- Geen inhoudelijke wijzigingen aan reports
- Geen andere SKILL updates

---

## VERIFICATION

```bash
# 1. ADR claim verwijderd
grep -n "ADR_001.*008\|implicit" docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md
# Expected: 0 matches

# 2. Reports gearchiveerd
ls docs/archived/reports/
# Expected: meerdere files

# 3. Reports opgeschoond
ls docs/reports/
# Expected: 1 file (nieuwste)

# 4. SESSIE_CAI niet meer verplicht
grep -n "SESSIE_CAI" docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md
# Expected: 0 matches OF niet als verplichting

# 5. Git gepusht
sudo -u energy-insights-nl git -C /opt/github/synctacles-api log --oneline -1
# Expected: P2 audit commit
```

---

## CONTEXT

Dit is onderdeel van Phase 3 Documentation Audit. P1 (handoff infrastructure) is compleet. P2 ruimt legacy/bloat op.

---

## COMMIT MESSAGE

```
docs: P2 audit cleanup

- Remove implicit ADR_001-008 claim from SKILL_00
- Archive old reports (keep most recent only)
- Remove SESSIE_CAI requirement (CAI has no persistent state)
```

---

## POST-HANDOFF VERIFICATIE

CC bevestigt ontvangst door:
1. Handoff file te plaatsen in `docs/handoffs/`
2. Taken uit te voeren
3. Completion report of nieuwe handoff terug

---
