# SYNCTACLES Care - Product Roadmap V2

**Product:** Home Assistant Maintenance & Security Add-on  
**Pricing:** €25/jaar (SYNCTACLES Premium)  
**Target:** NL trial funnel + EU direct sales  
**Last Updated:** 2026-01-25

---

## Vision Statement

> "SYNCTACLES Premium: Slimme energie + zorgeloos onderhoud voor Home Assistant. €25/jaar voor alles."

---

## Business Model

### Tier Structuur

| Tier | Prijs | Registratie | Duur |
|------|-------|-------------|------|
| **Gratis** | €0 | Geen | Onbeperkt |
| **Trial** | €0 | Email | 14 dagen |
| **Premium** | €25/jaar | Betaling | 1 jaar |

### Feature Matrix

| Feature | Gratis | Trial (14d) | Premium |
|---------|--------|-------------|---------|
| **ENERGY** | | | |
| Huidige prijzen NL | ✅ | ✅ | ✅ |
| Goedkoopste uren NL | ✅ | ✅ | ✅ |
| GO/WAIT/AVOID actions | ❌ | ✅ | ✅ |
| Best Window sensor | ❌ | ✅ | ✅ |
| Live Cost sensor | ❌ | ✅ | ✅ |
| Tomorrow preview | ❌ | ✅ | ✅ |
| **EU landen (DE/BE/AT/FR)** | ❌ | ❌ | ✅ |
| **CARE** | | | |
| Health scan | ✅ | ✅ | ✅ |
| Health score (A-F) | ✅ | ✅ | ✅ |
| Security scan | ✅ | ✅ | ✅ |
| Security score (0-100) | ✅ | ✅ | ✅ |
| Orphan lijst (view only) | ✅ | ✅ | ✅ |
| **Cleanup uitvoeren** | 🔒 | 🔒 | ✅ |
| **Scheduled maintenance** | 🔒 | 🔒 | ✅ |
| **Backup management** | 🔒 | 🔒 | ✅ |

### Monetization Strategie

```
┌─────────────────────────────────────────────────────────────────┐
│                     ACQUISITION FUNNEL                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  NL FUNNEL (Trial-based)                                         │
│  ─────────────────────────                                       │
│  Energy gratis → Care scan → Ziet probleem → Trial →             │
│  Ervaart Energy waarde → Wil cleanup → Premium                   │
│                                                                  │
│  EU FUNNEL (Direct)                                              │
│  ──────────────────                                              │
│  Care scan → Ziet probleem → Premium (+ Energy bonus)            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Revenue Projecties

### Conversie Funnel

**NL (Trial-based):**

| Stap | Conversie | Users |
|------|-----------|-------|
| NL HA users dynamisch contract | 30.000 | - |
| Installeert Energy | 10% | 3.000 |
| Installeert Care, ziet probleem | 50% | 1.500 |
| Start trial | 40% | 600 |
| Converteert naar Premium | 25% | 150 |

**EU (Direct):**

| Stap | Conversie | Users |
|------|-----------|-------|
| EU HA users met maintenance pijn | 100.000 | - |
| Hoort van Care | 3% | 3.000 |
| Installeert, ziet probleem | 40% | 1.200 |
| Converteert (direct) | 10% | 120 |

### Revenue Scenarios

| Jaar | NL | EU | Totaal | Revenue | MRR |
|------|----|----|--------|---------|-----|
| Y1 | 150 | 120 | 270 | €6.750 | €562 |
| Y2 | 300 | 400 | 700 | €17.500 | €1.458 |
| Y3 | 500 | 800 | 1.300 | €32.500 | €2.708 |
| Y4 | 700 | 1.200 | 1.900 | €47.500 | €3.958 |
| Y5 | 900 | 1.600 | 2.500 | €62.500 | €5.208 |

**MRR Target €2.5K: Q4 Y3**

---

## V1.0 - MVP Launch 🚀

**Codename:** "Clean Slate"  
**Timeline:** 4 weken  
**Goal:** NL trial funnel live, eerste betalende klanten

### V1.0 WOW Features

| Feature | Concurrentie | Wij |
|---------|--------------|-----|
| Security Score Dashboard | ❌ Bestaat niet | ✅ 0-100 met actionable tips |
| One-Click Orphan Cleanup | ❌ Spook buggy, handmatig | ✅ Safeguards + backup |
| Pre-flight Safety Check | ❌ Niemand | ✅ Ruimte/versie/cooldown |
| Database Health Grade | ❌ Niemand | ✅ A-F rating |
| Trial-to-Premium funnel | ❌ Niemand | ✅ 14-dagen NL trial |
| EU Energy als bonus | ❌ Niemand | ✅ Premium perk |

### V1.0 Technical Scope

#### Backend (api.synctacles.com)

**Database Schema Update:**

```sql
ALTER TABLE subscriptions ADD COLUMN
    features JSONB DEFAULT '{"energy": true, "care": false}',
    tier VARCHAR(20) DEFAULT 'free',           -- free, trial, premium
    trial_started_at TIMESTAMP,
    trial_ends_at TIMESTAMP,
    care_last_cleanup_at TIMESTAMP,
    care_last_scan_at TIMESTAMP,
    care_cleanup_count INT DEFAULT 0,
    country VARCHAR(2) DEFAULT 'NL';
```

**New Endpoints:**

| Endpoint | Functie |
|----------|---------|
| `POST /api/auth/register` | Email registratie, start trial |
| `POST /api/auth/start-trial` | Activeer 14-dagen trial |
| `GET /api/auth/status` | Check tier + trial status |
| `POST /api/care/authorize` | Premium check voor cleanup |
| `POST /api/care/cleanup/prepare` | Pre-authorize met safeguards |
| `POST /api/care/cleanup/complete` | Log resultaat |

**Trial Logic:**

```python
def get_feature_access(subscription, feature):
    # Premium = alles
    if subscription.tier == "premium":
        return True
    
    # Trial = Energy actions, geen Care cleanup
    if subscription.tier == "trial":
        if subscription.trial_ends_at > now():
            if feature in TRIAL_FEATURES:
                return True
            if feature == "care_cleanup":
                return False  # NOOIT in trial
    
    # Gratis = basis
    if feature in FREE_FEATURES:
        return True
    
    return False

FREE_FEATURES = [
    "energy_prices",
    "energy_hours", 
    "care_scan",
    "care_health_score",
    "care_security_score",
    "care_orphan_view"
]

TRIAL_FEATURES = FREE_FEATURES + [
    "energy_actions",
    "energy_best_window",
    "energy_live_cost",
    "energy_tomorrow"
]

PREMIUM_FEATURES = TRIAL_FEATURES + [
    "energy_eu",
    "care_cleanup",
    "care_scheduled",
    "care_backup_management"
]
```

---

### V1.0 Epics & GitHub Issues

#### EPIC 1: Repository Setup
**Label:** `epic:setup`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 1 | Repo structuur bepalen | Single vs mono-repo beslissing | 2h |
| 2 | Repo aanmaken | GitHub setup met juiste naam | 1h |
| 3 | Add-on config.yaml | Metadata, permissions, schema | 2h |
| 4 | Dockerfile basis | Alpine + Python + deps | 4h |
| 5 | run.sh entry point | Startup script met bashio | 2h |
| 6 | CI/CD pipeline | GitHub Actions build/test | 4h |
| 7 | Issue templates | Bug, feature, epic templates | 1h |
| 8 | Labels & milestones | Zoals gespecificeerd | 1h |

**Subtotaal:** 17h

---

#### EPIC 2: Backend Trial & Premium
**Label:** `epic:backend`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 10 | DB schema migratie | Tier, trial, care kolommen | 2h |
| 11 | POST /api/auth/register | Email + start trial optie | 4h |
| 12 | POST /api/auth/start-trial | Trial activatie | 2h |
| 13 | GET /api/auth/status | Tier + features + trial end | 2h |
| 14 | Trial expiry logic | Automatische downgrade | 4h |
| 15 | Feature gate middleware | Check per endpoint | 4h |
| 16 | POST /api/care/authorize | Premium + safeguards check | 4h |
| 17 | POST /api/care/cleanup/prepare | Pre-authorize cleanup | 4h |
| 18 | POST /api/care/cleanup/complete | Log + update counters | 2h |
| 19 | Rate limiting | Anti-abuse | 2h |

**Subtotaal:** 30h

---

#### EPIC 3: Health Scanner
**Label:** `epic:health`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 20 | Database size analyzer | Totaal, per tabel, groei | 4h |
| 21 | Fragmentation detector | SQLite fragmentation % | 2h |
| 22 | Orphaned statistics scanner | Query statistics_meta | 4h |
| 23 | Orphaned entities scanner | Query entity_registry | 4h |
| 24 | Data age analyzer | Oudste/nieuwste, retention | 2h |
| 25 | Health Score algoritme | A-F grade berekening | 4h |
| 26 | Health rapport generator | JSON + readable output | 4h |

**Subtotaal:** 24h

---

#### EPIC 4: Security Scanner ⭐
**Label:** `epic:security`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 30 | 2FA enabled check | Auth provider config | 4h |
| 31 | SSL/HTTPS check | Certificaat validatie | 2h |
| 32 | HA versie check | Vergelijk met latest | 2h |
| 33 | Add-ons outdated check | Supervisor API | 4h |
| 34 | Trusted networks audit | Config parser | 4h |
| 35 | IP ban config check | Config parser | 2h |
| 36 | Security Score algoritme | 0-100 berekening | 4h |
| 37 | Security rapport | Severity + recommendations | 4h |

**Subtotaal:** 26h

---

#### EPIC 5: Cleanup Engine ⭐
**Label:** `epic:cleanup`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 40 | Pre-flight checks | Ruimte, versie, cooldown | 4h |
| 41 | Backup creator | Supervisor API, tagged | 4h |
| 42 | Backup ruimte calculator | Estimate vs available | 2h |
| 43 | Dry-run mode | Simuleer, toon impact | 4h |
| 44 | Orphaned statistics cleaner | DELETE + transaction | 4h |
| 45 | Orphaned entities cleaner | Registry cleanup | 4h |
| 46 | Database VACUUM | Post-cleanup optimize | 2h |
| 47 | Integrity verificatie | Post-cleanup check | 4h |
| 48 | Rollback instructies | Recovery guide | 2h |
| 49 | Cleanup rapport | Wat verwijderd, ruimte | 2h |

**Subtotaal:** 32h

---

#### EPIC 6: Backup Lifecycle
**Label:** `epic:backup`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 50 | Backup tagging | `synctacles_care_YYYYMMDD` | 2h |
| 51 | Backup listing | Toon Care backups | 2h |
| 52 | Backup age tracker | Dagen sinds backup | 1h |
| 53 | Cleanup reminder | Na 7d: "Alles OK?" | 2h |
| 54 | Auto-delete policy | Na 30d, max 2 | 4h |
| 55 | Manual delete optie | User kiest verwijderen | 2h |

**Subtotaal:** 13h

---

#### EPIC 7: Web UI
**Label:** `epic:ui`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 60 | UI framework setup | Alpine.js + Tailwind | 4h |
| 61 | Dashboard view | Health + Security scores | 4h |
| 62 | Health details panel | Stats, orphan counts | 4h |
| 63 | Security details panel | Checks, fix links | 4h |
| 64 | Cleanup wizard | Step-by-step + confirms | 8h |
| 65 | Backup management panel | List, delete, status | 4h |
| 66 | Settings panel | API key config | 2h |
| 67 | Premium gate UI | Locked states + CTA | 4h |
| 68 | Trial status banner | Dagen resterend | 2h |
| 69 | Upgrade flow | Link naar betaling | 2h |

**Subtotaal:** 38h

---

#### EPIC 8: Premium Integration
**Label:** `epic:premium`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 70 | API key input flow | Config of UI setup | 2h |
| 71 | Feature gate client-side | Check features object | 2h |
| 72 | Gratis vs Trial vs Premium UI | Visuele states | 4h |
| 73 | Trial countdown | "Nog X dagen" | 2h |
| 74 | Upgrade CTA's | Strategisch geplaatst | 2h |
| 75 | Locked feature messaging | Wat ze missen | 2h |
| 76 | Offline fallback | Backend down handling | 4h |

**Subtotaal:** 18h

---

#### EPIC 9: Energy Integration Updates
**Label:** `epic:energy`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 80 | Trial feature gates | Actions locked na trial | 4h |
| 81 | EU country support | DE/BE/AT data bronnen | 8h |
| 82 | Country selector | Premium only voor EU | 2h |
| 83 | Energy + Care bundle check | Unified premium status | 2h |

**Subtotaal:** 16h

---

#### EPIC 10: Testing & Docs
**Label:** `epic:testing`

| # | Titel | Beschrijving | Estimate |
|---|-------|--------------|----------|
| 90 | Unit tests scanner | Pytest health/security | 4h |
| 91 | Unit tests cleaner | Pytest met mock DB | 4h |
| 92 | Integration tests | Full flow testing | 8h |
| 93 | Trial flow tests | Edge cases trial | 4h |
| 94 | DOCS.md | Gebruikersdocs | 4h |
| 95 | CHANGELOG.md | Release notes | 1h |
| 96 | Troubleshooting guide | Support docs | 4h |

**Subtotaal:** 29h

---

### V1.0 Totaal

| Epic | Uren |
|------|------|
| Setup | 17h |
| Backend Trial & Premium | 30h |
| Health Scanner | 24h |
| Security Scanner | 26h |
| Cleanup Engine | 32h |
| Backup Lifecycle | 13h |
| Web UI | 38h |
| Premium Integration | 18h |
| Energy Updates | 16h |
| Testing & Docs | 29h |
| **Totaal** | **243h** |

**Timeline:** ~5 weken bij 6-8h/dag

---

## V1.1 - Polish & i18n

**Timeline:** +2 weken na V1.0

| Feature | Beschrijving |
|---------|--------------|
| Multi-language | NL + EN + DE |
| Scheduled health scans | Dagelijks/wekelijks |
| HA Notifications | Native persistent_notification |
| Improved recommendations | Specifiekere fix tips |
| Trial reminder emails | Dag 7, dag 12, dag 14 |

---

## V1.2 - Automation

**Timeline:** +3 weken na V1.1

| Feature | Beschrijving |
|---------|--------------|
| Scheduled cleanup | Maandelijkse auto-cleanup (opt-in) |
| Threshold alerts | Alert bij >1GB of <10% ruimte |
| HA Sensors | `sensor.synctacles_care_*` |
| Automation triggers | Events voor HA automations |
| Blueprints | Pre-built automations |

---

## V2.0 - Advanced

**Timeline:** Q2 2026

| Feature | Premium |
|---------|---------|
| Historical trends | ✅ |
| Predictive alerts | ✅ |
| Custom cleanup rules | ✅ |
| PDF export rapport | ✅ |
| Multi-instance support | ✅ Extra tier |

---

## V3.0 - AI-Powered

**Timeline:** Q4 2026

| Feature | Beschrijving |
|---------|--------------|
| AI recommendations | Claude-powered suggestions |
| Anomaly detection | Ongewone patronen |
| Smart scheduling | Optimale timing |
| Natural language | "Wat is er mis?" |

---

## Repository Structuur

```
synctacles/addon-synctacles-care/    (of mono-repo - CC beslist)
│
├── .github/
│   ├── workflows/
│   │   ├── build.yml
│   │   ├── test.yml
│   │   └── release.yml
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   ├── feature_request.md
│   │   └── epic.md
│   └── PULL_REQUEST_TEMPLATE.md
│
├── synctacles-care/
│   ├── config.yaml
│   ├── Dockerfile
│   ├── build.yaml
│   ├── run.sh
│   ├── DOCS.md
│   ├── CHANGELOG.md
│   ├── icon.png
│   ├── logo.png
│   │
│   ├── care/
│   │   ├── __init__.py
│   │   ├── main.py
│   │   ├── config.py
│   │   │
│   │   ├── scanner/
│   │   │   ├── health.py
│   │   │   ├── security.py
│   │   │   └── models.py
│   │   │
│   │   ├── cleaner/
│   │   │   ├── orphans.py
│   │   │   ├── database.py
│   │   │   ├── backup.py
│   │   │   └── safeguards.py
│   │   │
│   │   ├── api/
│   │   │   ├── client.py
│   │   │   └── models.py
│   │   │
│   │   ├── web/
│   │   │   ├── server.py
│   │   │   ├── routes.py
│   │   │   └── static/
│   │   │
│   │   └── utils/
│   │       ├── ha_api.py
│   │       ├── supervisor.py
│   │       └── logging.py
│   │
│   └── translations/
│       ├── en.yaml
│       └── nl.yaml
│
├── tests/
│   ├── conftest.py
│   ├── test_health.py
│   ├── test_security.py
│   ├── test_cleaner.py
│   └── test_trial.py
│
├── repository.yaml
├── README.md
├── LICENSE
└── pyproject.toml
```

---

## GitHub Labels

```yaml
# Type
- name: "type:bug"
  color: "d73a4a"
- name: "type:feature"
  color: "a2eeef"
- name: "type:docs"
  color: "0075ca"

# Epic
- name: "epic:setup"
  color: "7057ff"
- name: "epic:backend"
  color: "7057ff"
- name: "epic:health"
  color: "7057ff"
- name: "epic:security"
  color: "7057ff"
- name: "epic:cleanup"
  color: "7057ff"
- name: "epic:backup"
  color: "7057ff"
- name: "epic:ui"
  color: "7057ff"
- name: "epic:premium"
  color: "7057ff"
- name: "epic:energy"
  color: "7057ff"
- name: "epic:testing"
  color: "7057ff"

# Priority
- name: "priority:critical"
  color: "b60205"
- name: "priority:high"
  color: "d93f0b"
- name: "priority:medium"
  color: "fbca04"
- name: "priority:low"
  color: "0e8a16"

# Tier
- name: "tier:free"
  color: "c5def5"
- name: "tier:trial"
  color: "bfdadc"
- name: "tier:premium"
  color: "fef2c0"
```

---

## GitHub Milestones

| Milestone | Due Date | Description |
|-----------|----------|-------------|
| V1.0 MVP | 2026-03-01 | Trial funnel + Care cleanup |
| V1.1 Polish | 2026-03-15 | i18n + scheduled scans |
| V1.2 Automation | 2026-04-05 | HA sensors + blueprints |
| V2.0 Advanced | 2026-06-01 | Trends + multi-instance |

---

## Messaging & Copy

### Homepage
> *"SYNCTACLES: Slimme energie + zorgeloos onderhoud voor Home Assistant"*

### Trial CTA
> *"14 dagen gratis proberen - Ontdek hoeveel jouw HA opgeruimd moet worden"*

### Premium CTA
> *"€25/jaar - One-click cleanup + Energy insights voor heel Europa"*

### Locked Cleanup Screen
> *"Je hebt 247 orphaned statistics gevonden (~500MB). Upgrade naar Premium om ze veilig te verwijderen."*

### Trial Ending Banner
> *"Je trial eindigt over 2 dagen. Upgrade nu en behoud toegang tot alle features."*

### Post-Cleanup Success
> *"🎉 487MB vrijgemaakt! Je Health Score is nu A."*

---

## Success Metrics V1.0

| Metric | Target (90 dagen) |
|--------|-------------------|
| Installaties (gratis) | 2.000+ |
| Trial starts | 500+ |
| Trial → Premium conversie | 20%+ |
| Premium users | 100+ |
| Cleanup success rate | >99% |
| Data loss incidents | 0 |
| NPS score | >40 |

---

## Launch Checklist

### Pre-Launch
- [ ] Add-on installeert clean
- [ ] Health + Security scan werkt
- [ ] Cleanup met safeguards werkt
- [ ] Trial flow compleet
- [ ] Premium gate functioneert
- [ ] Payment integration werkt
- [ ] 80% test coverage
- [ ] Docs compleet

### Launch
- [ ] GitHub repo public
- [ ] Add-on repo registreren
- [ ] Landing page live
- [ ] Trial signup werkend
- [ ] HA Community announcement
- [ ] Tweakers post
- [ ] Reddit post

### Post-Launch
- [ ] Monitor installaties
- [ ] Support queue
- [ ] Trial conversies tracken
- [ ] Feedback verzamelen
- [ ] Hotfix ready

---

*Document Version: 2.0*  
*Last Updated: 2026-01-25*  
*Model: NL Trial + EU Premium Bundle*
