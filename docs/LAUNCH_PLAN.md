# LAUNCH PLAN: Gecontroleerde V1 Release

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09  
**Laatste update:** 2026-01-09 (Coefficient Engine toegevoegd)  
**Doel:** OTP-structuur opzetten voor professionele, gecontroleerde launch

---

## CONTEXT

Leo's launch criteria:
1. Niet overspoeld worden met bugs
2. Professioneel fundament voor doorgroei
3. OTP-omgeving (geen ad-hoc fixes in productie)
4. Controle over het product

Huidige staat:
- O (Ontwikkeling): ✅ Werkt
- T (Test): ❌ Ontbreekt
- P (Productie): ✅ Draait
- Coefficient Engine: ❌ Ontbreekt (KRITIEK)

---

## FASE 0: COEFFICIENT ENGINE (Blocker voor product waarde)

**Waarom eerst?**  
Zonder coefficient is Energy Action gebaseerd op wholesale prijzen, niet wat de gebruiker betaalt. Dit is de kern van het product en ons IP.

**Server:** 91.99.150.36 (Hetzner CX23)  
**Repo:** git@github.com:DATADIO/coefficient-engine.git (PRIVATE)  
**Handoff:** HANDOFF_CC_COEFFICIENT_ENGINE.md

### 0.1 Server Setup
- [ ] Installer script testen op nieuwe server
- [ ] PostgreSQL configureren
- [ ] Repo clonen en structuur opzetten

### 0.2 Data Verzamelen
- [ ] ENTSO-E backfill (2022-2025)
- [ ] Enever CSV import (wacht op Leo's Supporter toegang)

### 0.3 Analyse
- [ ] Coefficient berekening
- [ ] Stabiliteitsrapport genereren
- [ ] Besluit: lookup tabel of real-time?

### 0.4 API
- [ ] GET /coefficient endpoint
- [ ] IP whitelist (alleen SYNCTACLES server)
- [ ] Health check

### 0.5 Integratie
- [ ] SYNCTACLES haalt coefficient op
- [ ] Energy Action gebruikt coefficient
- [ ] Fallback naar historische coefficient

**Geschatte tijd:** 4-5 dagen

---

## FASE 1: OTP FUNDAMENT (Blocker voor kwaliteit)

### 1.1 Unit Test Suite (#5)

**Prioriteit:** CRITICAL  
**Geschatte tijd:** 2-3 uur

Voer HANDOFF_CAI_UNIT_TESTS.md uit:
- [ ] pytest.ini configuratie
- [ ] conftest.py met fixtures
- [ ] FallbackManager tests
- [ ] API endpoint tests
- [ ] Normalizer tests
- [ ] GitHub Actions workflow (.github/workflows/test.yml)

**Exit criteria:**
```bash
pytest -v  # Alle tests groen
```

### 1.2 Branch Protection

**Prioriteit:** CRITICAL  
**Geschatte tijd:** 15 min

```bash
# Na tests werken in CI:
gh api -X PUT /repos/synctacles/synctacles-api/branches/main/protection \
  -f required_status_checks='{"strict":true,"contexts":["test"]}' \
  -f enforce_admins=false \
  -f required_pull_request_reviews=null \
  -f restrictions=null
```

**Exit criteria:**
- Push naar main zonder groene tests = BLOCKED

### 1.3 HA Component Documentatie (#47)

**Prioriteit:** HIGH  
**Geschatte tijd:** 1 uur

HA component bestaat maar is niet gedocumenteerd. Voeg toe:

**ARCHITECTURE.md:**
```markdown
## Home Assistant Integration

**Status:** Implemented, v1.0.0

**Installatie:** Custom component via HACS of handmatig

**Entities (12):**
| Entity | Type | Beschrijving |
|--------|------|--------------|
| Balance Delta | sensor | Grid balance (MW) |
| Cheapest Hour | sensor | Goedkoopste uur vandaag |
| Electricity Price | sensor | Huidige prijs (€/MWh) |
| Energy Action | sensor | Aanbevolen actie (WAIT/USE/AVOID) |
| Generation Total | sensor | Totale opwek (MW) |
| Grid Stress | sensor | Netbelasting (%) |
| Load Actual | sensor | Huidig verbruik (MW) |
| Most Expensive Hour | sensor | Duurste uur vandaag |
| Price Level | sensor | Prijsniveau (%) |
| Price Status | sensor | Status (NORMAL/HIGH/LOW) |
| Prices Today | sensor | Prijzen vandaag (lijst) |
| Prices Tomorrow | sensor | Prijzen morgen (lijst) |

**Configuratie:** Via UI flow (Settings → Integrations → Add → Energy Insights NL)

**Vereist:** API URL (https://enin.xteleo.nl)
```

**user-guide.md:** Installatie-instructies voor eindgebruikers

**Exit criteria:**
- Nieuwe AI/developer kan HA component vinden in docs
- Installatie-instructies compleet

---

## FASE 2: LAUNCH VOORBEREIDING

### 2.1 Documentation Cleanup (#50)

**Prioriteit:** MEDIUM  
**Geschatte tijd:** 2 uur

- [ ] api-reference.md compleet (alle endpoints)
- [ ] user-guide.md eindgebruiker-ready
- [ ] README.md voor GitHub repo

### 2.2 External Monitoring (#44)

**Prioriteit:** HIGH  
**Geschatte tijd:** 30 min

Minimaal: UptimeRobot of vergelijkbaar op:
- https://enin.xteleo.nl/health
- Alert naar Leo bij downtime

### 2.3 Backup Test (#45)

**Prioriteit:** HIGH  
**Geschatte tijd:** 1 uur

```bash
# Test backup procedure
pg_dump synctacles_nl > /tmp/test_backup.sql

# Test restore (op test DB)
createdb synctacles_nl_test
psql synctacles_nl_test < /tmp/test_backup.sql

# Verify
psql synctacles_nl_test -c "SELECT COUNT(*) FROM norm_generation"

# Cleanup
dropdb synctacles_nl_test
```

Document procedure in SKILL_08.

---

## FASE 3: BETA LAUNCH

### 3.1 Landing Page (#52)

**Prioriteit:** MEDIUM  
**Geschatte tijd:** 4 uur (of skip voor V1)

Opties:
- A: Simpele static page op enin.xteleo.nl
- B: Carrd.co of vergelijkbaar (30 min)
- C: Skip, direct via HA community posts

**Aanbeveling:** B of C voor snelheid

### 3.2 Beta User Onboarding (#53)

**Prioriteit:** HIGH  
**Geschatte tijd:** 2 uur

- [ ] Onboarding doc schrijven (stap-voor-stap)
- [ ] Support kanaal bepalen (Discord? GitHub Issues?)
- [ ] Feedback formulier
- [ ] Max 5-10 beta users selecteren

### 3.3 Payment Gateway (#51)

**Prioriteit:** HIGH (maar na beta feedback)  
**Geschatte tijd:** 4-8 uur

Opties:
- Mollie (NL-native, iDEAL)
- Stripe (internationaal)

**Aanbeveling:** Mollie voor NL markt, simpelste integratie

---

## GITHUB REORGANISATIE

### Issues Herclassificeren

```bash
# FASE 1 - OTP (CRITICAL)
gh issue edit 5 --add-label "critical" --remove-label "high" --milestone "V1 Launch"

# FASE 2 - Launch Prep (HIGH)
gh issue edit 44 --add-label "high" --milestone "V1 Launch"
gh issue edit 45 --add-label "high" --milestone "V1 Launch"
gh issue edit 50 --add-label "medium" --milestone "V1 Launch"

# FASE 3 - Beta (HIGH)
gh issue edit 51 --add-label "high" --milestone "V1 Launch"
gh issue edit 52 --add-label "low" --milestone "V1 Launch"
gh issue edit 53 --add-label "high" --milestone "V1 Launch"

# Sluit reeds afgerond
gh issue close 47 --comment "HA Component implemented and documented"
```

### Nieuwe Milestone

```bash
gh api -X PATCH /repos/synctacles/synctacles-api/milestones/1 \
  -f title="V1 Launch" \
  -f description="Gecontroleerde launch met OTP-structuur" \
  -f due_on="2026-01-15T00:00:00Z"
```

---

## KRITIEKE PAD

```
Dag 1-2:  Fase 0 - Coefficient server setup + ENTSO-E backfill
Dag 2-3:  Fase 0 - Enever import + analyse + API
Dag 3-4:  Fase 0 - SYNCTACLES integratie
Dag 4-5:  Fase 1 - #5 Unit Tests + Branch Protection
Dag 5-6:  Fase 1 - #47 Docs + #44 Monitoring + #45 Backup Test
Dag 7:    Fase 3 - #53 Beta Onboarding + eerste 5 users
Week 2:   Feedback verwerken + #51 Payment
          ↓
       V1 PUBLIC LAUNCH
```

---

## NIET VOOR V1 (Parking Lot)

| Issue | Waarom later |
|-------|--------------|
| #8 DB HA | Overkill voor <100 users |
| #7 CI/CD uitgebreid | Basis is genoeg |
| #6 Advanced Monitoring | Grafana werkt |
| #14 Multi-region | V2 |
| #17 Mobile App | V2+ |

---

## EXIT CRITERIA V1 LAUNCH

- [ ] Coefficient engine draait en levert data
- [ ] Energy Action gebruikt coefficient (niet alleen wholesale)
- [ ] Tests in CI, branch protection aan
- [ ] Docs compleet (HA, API, user guide)
- [ ] External monitoring actief
- [ ] Backup procedure getest
- [ ] 5-10 beta users actief
- [ ] Feedback kanaal open
- [ ] Payment ready (of handmatig voor beta)

---

## START COMMANDO

Begin met Fase 1.1:

```bash
cd /opt/github/synctacles-api
# Voer HANDOFF_CAI_UNIT_TESTS.md uit
```

Rapporteer na elke fase-completion.
