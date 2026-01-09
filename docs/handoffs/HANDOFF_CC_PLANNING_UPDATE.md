# HANDOFF: Planning Update + GitHub Reorganisatie

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09

---

## CONTEXT

Launch plan is herzien. Coefficient Engine is toegevoegd als Fase 0 — dit is blocker voor product waarde (niet alleen kwaliteit).

**Nieuwe prioriteit:**
```
Fase 0: Coefficient Engine (NIEUW - product waarde)
Fase 1: OTP/Tests (kwaliteit)
Fase 2: Docs/Monitoring (operationeel)
Fase 3: Beta launch (commercieel)
```

---

## OPDRACHTEN

### 1. Launch Plan Opslaan

**Update bestaand of maak nieuw:**
```
/opt/github/synctacles-api/docs/LAUNCH_PLAN.md
```

Gebruik content uit `/mnt/user-data/outputs/LAUNCH_PLAN_V1.md`

### 2. GitHub Issues Reorganiseren

```bash
# Nieuwe issue voor Coefficient Engine
sudo -u energy-insights-nl gh issue create \
  --repo synctacles/synctacles-api \
  --title "F0 - Coefficient Engine Setup" \
  --body "Setup coefficient server en integratie. Zie HANDOFF_CC_COEFFICIENT_ENGINE.md. Server: 91.99.150.36, Repo: DATADIO/coefficient-engine" \
  --label "critical"

# Update bestaande priorities
sudo -u energy-insights-nl gh issue edit 5 --add-label "critical" --milestone "V1 Launch"
sudo -u energy-insights-nl gh issue edit 47 --add-label "high" --milestone "V1 Launch"

# Sluit #47 als docs klaar zijn
# (wacht tot HA component gedocumenteerd is)
```

### 3. Coefficient Engine Handoff

Zie apart document: `HANDOFF_CC_COEFFICIENT_ENGINE.md`

Dit is de primaire taak. Start hiermee zodra Leo het installer script heeft gedraaid.

### 4. Git Commit

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add docs/LAUNCH_PLAN.md
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: update launch plan met Fase 0 Coefficient Engine"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## VOLGORDE VANDAAG

1. Wacht op Leo: installer script test op coefficient server
2. Na installer: start HANDOFF_CC_COEFFICIENT_ENGINE.md
3. Parallel: update planning in synctacles-api repo

---

## DOCUMENTS OVERZICHT

| Document | Locatie | Inhoud |
|----------|---------|--------|
| Launch Plan | synctacles-api/docs/LAUNCH_PLAN.md | Overzicht alle fases |
| Coefficient Handoff | HANDOFF_CC_COEFFICIENT_ENGINE.md | Gedetailleerde technische setup |
| Unit Tests Handoff | HANDOFF_CAI_UNIT_TESTS.md | Test suite (Fase 1) |

---

## TIJDLIJN

| Dag | Focus |
|-----|-------|
| 1-3 | Coefficient Engine (Fase 0) |
| 4-5 | Unit Tests + OTP (Fase 1) |
| 6 | Docs + Monitoring (Fase 2) |
| 7+ | Beta launch (Fase 3) |

Coefficient Engine is nu de primaire focus.
