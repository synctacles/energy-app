# SKILL BUSINESS 00 — GO-TO-MARKET STRATEGY V2

Business Model, Pricing, and Adoption Strategy  
**Version: 2.0 (2026-01-25)**

> **Kernbeslissing:** Care is kernproduct (€25/jaar), Energy is gratis acquisition funnel.  
> NL trial + EU premium bundle.

---

## EXECUTIVE SUMMARY

SYNCTACLES positioneert als **Premium Bundle**: Care (maintenance) + Energy (insights) voor €25/jaar. 

- **Care** = universele pijn (maintenance), converteert
- **Energy** = regionale waarde (NL/EU), acquireert + sweetens deal
- **NL** = trial funnel (14 dagen gratis Energy actions)
- **EU** = direct sales (Care-first, Energy als bonus)

---

## PRODUCT BUNDELING

### Tier Structuur

| Tier | Prijs | Energy | Care |
|------|-------|--------|------|
| **Gratis** | €0 | NL: prijzen + uren | Scan + view |
| **Trial (14d)** | €0 | NL: + actions/best window | Scan + view |
| **Premium** | €25/jaar | **Heel EU** + alles | **Cleanup + scheduled** |

### Feature Matrix

| Feature | Gratis | Trial (14d) | Premium |
|---------|--------|-------------|---------|
| **ENERGY** | | | |
| Prijzen NL | ✅ | ✅ | ✅ |
| Goedkoopste uren NL | ✅ | ✅ | ✅ |
| GO/WAIT/AVOID | ❌ | ✅ | ✅ |
| Best Window | ❌ | ✅ | ✅ |
| Live Cost | ❌ | ✅ | ✅ |
| Tomorrow preview | ❌ | ✅ | ✅ |
| EU landen | ❌ | ❌ | ✅ |
| **CARE** | | | |
| Health scan + score | ✅ | ✅ | ✅ |
| Security scan + score | ✅ | ✅ | ✅ |
| Orphan view | ✅ | ✅ | ✅ |
| **Cleanup** | 🔒 | 🔒 | ✅ |
| **Scheduled** | 🔒 | 🔒 | ✅ |
| **Backup mgmt** | 🔒 | 🔒 | ✅ |

### Kritieke Regel

```
Care cleanup is NOOIT in trial.
Dat is de conversie driver.
```

---

## WAARDE PROPOSITIE

### Messaging

**Primair:**
> "€25/jaar voor Care + Energy insights voor heel Europa gratis erbij"

**Alternatief:**
> "SYNCTACLES Premium: One-click HA cleanup + slimme energie voor €2.08/maand"

### Wat Lost Het Op?

| Probleem | Oplossing | Waarde |
|----------|-----------|--------|
| "7000 orphaned statistics, handmatig klikken" | One-click cleanup | Uren werk bespaard |
| "Database 7GB, help!" | Health scan + cleanup | Ruimte + snelheid |
| "Is mijn HA veilig?" | Security Score 0-100 | Peace of mind |
| "Wanneer energie goedkoop?" | GO/WAIT/AVOID | €50-150/jaar besparing |

---

## USER JOURNEY

### NL Funnel (Trial-based)

```
DISCOVERY
└── Installeert Energy Integration (HACS)
└── Geen registratie nodig
└── Ziet: prijzen, goedkoopste uren
    ↓
GRATIS WAARDE
└── "Handig! Wat is Care?"
└── Installeert Care Add-on
└── Ziet: Health B, Security 65/100
└── Ziet: "247 orphaned statistics" 😱
└── Klikt Cleanup → 🔒 Premium
    ↓
TRIAL START
└── "14 dagen gratis"
└── Email registratie
└── Energy actions unlocken
└── Care cleanup blijft 🔒
    ↓
TRIAL EXPERIENCE (14 dagen)
└── Geniet van GO/WAIT/AVOID
└── Ziet dagelijks: "247 orphans wachten"
└── Dag 12: "Trial eindigt over 2 dagen"
    ↓
CONVERSION
└── "€25/jaar = €2.08/maand"
└── "Inclusief Energy voor heel EU"
└── Betaalt → Alles unlocked
└── Eerste cleanup: "487MB vrijgemaakt!" 🎉
```

### EU Funnel (Direct)

```
DISCOVERY
└── Hoort van Care (Reddit, forum, YouTube)
└── Maintenance pijn herkenbaar
    ↓
INSTALL
└── Installeert Care Add-on
└── Ziet: Health C, Security 45/100
└── Ziet: "1.2GB aan orphans" 😱
    ↓
DIRECT CONVERSION
└── "€25/jaar voor cleanup"
└── "Bonus: Energy voor jouw land"
└── Betaalt → Alles unlocked
```

---

## PSYCHOLOGICAL TRIGGERS

| Trigger | Implementatie |
|---------|---------------|
| **Loss aversion** | "247 orphans vreten ruimte" |
| **Sunk cost** | 14 dagen invested in setup |
| **Social proof** | "1.2GB vrijgemaakt door 500+ users" |
| **Urgency** | "Trial eindigt over 2 dagen" |
| **Anchoring** | "€2.08/maand" ipv "€25/jaar" |
| **Bonus framing** | "Energy EU gratis erbij" |

---

## COMPETITIVE POSITIONING

### vs Spook (gratis)

| Aspect | Spook | SYNCTACLES Care |
|--------|-------|-----------------|
| Orphan listing | ✅ Buggy | ✅ Betrouwbaar |
| One-click cleanup | ❌ "Fix All" gevraagd sinds 2022 | ✅ Met safeguards |
| Statistics cleanup | ❌ Alleen listen | ✅ Daadwerkelijk delete |
| Security audit | ❌ | ✅ Score 0-100 |
| Database optimize | ❌ | ✅ VACUUM |
| Backup-first | ❌ | ✅ Verplicht |
| Scheduled | ❌ | ✅ Premium |

**Conclusie:** Spook = feature voor devs, SYNCTACLES Care = product voor users.

### vs HA Native Tools

| Tool | Functie | SYNCTACLES Care |
|------|---------|-----------------|
| Developer Tools > Statistics | Één voor één fixen | One-click alle |
| Recorder purge | Oude data | + Orphans + VACUUM |
| auto_repack | Config (advanced) | Zero-config |

**Conclusie:** HA tools = primitives, SYNCTACLES Care = UX layer.

---

## REVENUE PROJECTIONS

### Conversie Funnel

**NL (Trial):**
| Stap | Conversie | Users |
|------|-----------|-------|
| NL HA users (dynamisch) | 30K | - |
| Installeert Energy | 10% | 3.000 |
| Installeert Care | 50% | 1.500 |
| Start trial | 40% | 600 |
| Converteert | 25% | **150** |

**EU (Direct):**
| Stap | Conversie | Users |
|------|-----------|-------|
| EU maintenance pijn | 100K | - |
| Hoort van Care | 3% | 3.000 |
| Installeert | 40% | 1.200 |
| Converteert | 10% | **120** |

### Revenue Scenarios

| Jaar | NL | EU | Totaal | Revenue | MRR |
|------|----|----|--------|---------|-----|
| Y1 | 150 | 120 | 270 | €6.750 | €562 |
| Y2 | 300 | 400 | 700 | €17.500 | €1.458 |
| Y3 | 500 | 800 | 1.300 | €32.500 | €2.708 |

**MRR Target €2.5K: Q4 Y3**

### Break-even

- Server kosten: ~€60/jaar
- Break-even: **3 premium users**

---

## TRIAL MECHANICS

### Backend Implementation

```python
TRIAL_DAYS = 14

def start_trial(email: str) -> APIKey:
    """Start 14-dag trial."""
    key = generate_api_key()
    db.insert(
        api_key=key,
        email=email,
        tier="trial",
        trial_ends_at=now() + timedelta(days=14)
    )
    return key

def check_feature(sub: Subscription, feature: str) -> bool:
    """Check feature access."""
    # Premium = alles
    if sub.tier == "premium":
        return True
    
    # Trial actief
    if sub.tier == "trial" and sub.trial_ends_at > now():
        if feature in TRIAL_FEATURES:
            return True
        if feature == "care_cleanup":
            return False  # NOOIT
    
    # Gratis
    if feature in FREE_FEATURES:
        return True
    
    return False
```

### Email Triggers

| Dag | Onderwerp | Inhoud |
|-----|-----------|--------|
| 0 | "Welkom bij SYNCTACLES" | Setup tips, Care install hint |
| 7 | "Halverwege je trial" | Orphan count reminder |
| 12 | "Nog 2 dagen" | Urgency, upgrade CTA |
| 14 | "Trial verlopen" | Features gelocked, upgrade |

---

## ANTI-ABUSE

### Rate Limiting

| Tier | Limiet | Enforcement |
|------|--------|-------------|
| Geen key | 100/dag/IP | IP blocking |
| Gratis key | 500/dag | Key blocking |
| Trial | 1000/dag | Key blocking |
| Premium | 2000/dag | Key blocking |

### Abuse Detectie

- Request patterns monitoring
- HA installation_id binding (premium)
- Automated blocking bij spikes

---

## PRICING PSYCHOLOGY

### Waarom €25/jaar

| Factor | Rationale |
|--------|-----------|
| < €30 | Impulsaankoop drempel |
| > €10 | Genoeg om serieus te nemen |
| Jaarlijks | Minder churn, hogere LTV |
| Round number | Geen nickle-and-diming gevoel |

### Framing

| Frame | Toepassing |
|-------|------------|
| Per maand | "€2.08/maand" in CTA's |
| Per jaar | "€25/jaar" in pricing page |
| Bundle | "Care + Energy EU" altijd samen |
| Vergelijking | "Minder dan 1 koffie/maand" |

---

## GO-TO-MARKET PLAN

### Pre-Launch (Week -2 to 0)

- [ ] Care add-on feature-complete
- [ ] Trial flow werkend
- [ ] Payment integratie
- [ ] Landing page live
- [ ] Docs compleet

### Soft Launch (Week 1-2)

- [ ] Invite-only beta (50 users)
- [ ] Feedback verzamelen
- [ ] Bug fixes
- [ ] Conversion optimalisatie

### Public Launch (Week 3)

- [ ] HA Community Forum post
- [ ] Tweakers forum
- [ ] Reddit r/homeassistant
- [ ] Home Assistant subreddit
- [ ] GitHub repository public

### Growth (Week 4+)

- [ ] YouTube outreach
- [ ] Feature requests implementeren
- [ ] EU expansion (DE/BE/AT)
- [ ] Word-of-mouth incentives

---

## MARKETING CHANNELS

### Organic (Gratis)

| Kanaal | Verwachte Impact |
|--------|------------------|
| HA Community Forum | Hoog - core audience |
| Reddit r/homeassistant | Hoog - 800K members |
| Tweakers forum | Medium - NL focus |
| GitHub trending | Low - devs only |

### Earned Media

| Type | Strategie |
|------|-----------|
| YouTube reviews | Outreach naar HA YouTubers |
| Blog posts | Guest posts op HA blogs |
| Podcast mentions | HA-focused podcasts |

### Paid (Later)

- Geen paid marketing Y1
- Mogelijk Reddit ads na product-market fit
- ROI moet >3x zijn

---

## SUCCESS METRICS

### V1.0 (90 dagen)

| Metric | Target |
|--------|--------|
| Total installs | 2.000+ |
| Trial starts | 500+ |
| Trial → Premium | 20%+ |
| Premium users | 100+ |
| Revenue | €2.500+ |
| Cleanup success | >99% |
| Data loss | 0 |
| NPS | >40 |

### Y1

| Metric | Target |
|--------|--------|
| Premium users | 270+ |
| Revenue | €6.750+ |
| MRR | €562+ |
| Countries | NL + DE |

---

## RISK MITIGATION

| Risico | Impact | Mitigatie |
|--------|--------|-----------|
| Spook improves | Medium | UX + security differentiatie |
| HA adds native cleanup | High | Security focus, ahead blijven |
| Low trial conversion | Medium | A/B test CTAs, messaging |
| Support overload | Medium | Docs, self-service, Discord |
| Abuse/scraping | Low | Rate limits, IP blocking |

---

## EU EXPANSION ROADMAP

### Phase 1: NL (Launch)
- Frank Energie
- EasyEnergy  
- Enever

### Phase 2: DE (Q2 2026)
- ENTSO-E (basis)
- aWATTar
- Tibber

### Phase 3: BE/AT (Q3 2026)
- ENTSO-E data
- Local providers research

### Phase 4: FR/ES (Q4 2026)
- Market research
- API integraties

---

## COMMUNICATION TEMPLATES

### Launch Announcement

> **SYNCTACLES Premium: One-click HA maintenance + slimme energie**
>
> Nieuw: SYNCTACLES Care add-on
> - Security Score: Is je HA veilig?
> - Health Score: Hoe gezond is je database?
> - One-click Cleanup: Verwijder orphans veilig
>
> + Energy insights voor heel Europa
>
> 14 dagen gratis proberen → synctacles.com

### Trial Ending Email

> **Je SYNCTACLES trial eindigt over 2 dagen**
>
> Je hebt 247 orphaned statistics gevonden (~500MB).
> 
> Upgrade naar Premium voor €25/jaar:
> ✓ One-click cleanup
> ✓ Scheduled maintenance
> ✓ Energy voor heel EU
>
> [Upgrade nu]

### Premium Upsell (In-app)

> **🔒 Premium Feature**
>
> Je hebt 247 orphaned statistics (~500MB).
> Upgrade om ze in één klik te verwijderen.
>
> €25/jaar = €2.08/maand
> + Energy insights voor heel Europa
>
> [Upgrade naar Premium]

---

## BESLISSINGEN LOG

| Datum | Beslissing | Rationale |
|-------|------------|-----------|
| 2026-01-21 | Gratis lanceren | First-mover, adoptie |
| 2026-01-25 | Care = kernproduct | Universele pijn, converteert |
| 2026-01-25 | Energy = acquisition | NL sweetener, EU bonus |
| 2026-01-25 | €25/jaar bundle | Impulsaankoop, hoge perceived value |
| 2026-01-25 | 14-dag trial (NL) | Ervaar Energy, wil Care cleanup |
| 2026-01-25 | Cleanup nooit in trial | Conversie driver |

---

## SAMENVATTING

```
SYNCTACLES PREMIUM (€25/jaar)
├── Care (global) = kernproduct
│   ├── Health scan + score (gratis)
│   ├── Security scan + score (gratis)
│   ├── Orphan view (gratis)
│   └── Cleanup + scheduled (PREMIUM)
│
└── Energy (NL → EU) = acquisition + bonus
    ├── Prijzen + uren (gratis)
    ├── Actions + Best Window (trial/premium)
    └── EU landen (PREMIUM)

FUNNEL:
NL: Gratis Energy → Care scan → Trial → Premium
EU: Care scan → Premium (+ Energy bonus)

TARGET:
Y1: 270 users, €6.750
Y3: 1.300 users, €32.500, €2.7K MRR
```

---

*Versie: 2.0*  
*Datum: 2026-01-25*  
*Model: NL Trial + EU Premium Bundle*
