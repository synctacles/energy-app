# SKILL 17 — GO-TO-MARKET STRATEGY

Business Model, Pricing, and Adoption Strategy
**Version: 1.0 (2026-01-21)**

> **Kernbeslissing:** Gratis lanceren, zero frictie, maximale adoptie.
> Premium features = roadmap, niet V1.

---

## EXECUTIVE SUMMARY

SYNCTACLES lanceert als **volledig gratis** product zonder API key vereiste. De strategische rationale: first-mover advantage in een markt waar AI-tools snel concurrerende oplossingen kunnen bouwen.

**Waarde propositie:** Niet data (Enever biedt dat voor €12.50/jaar), maar **intelligence + simpliciteit** — GO/WAIT/AVOID zonder configuratie.

---

## STRATEGISCHE RATIONALE

### Waarom Gratis?

| Factor | Betaald (€1-2/maand) | Gratis |
|--------|----------------------|--------|
| Adoptie snelheid | Langzaam | Snel |
| Concurrent risico | Hoog | Laag |
| Community/reputatie | Matig | Sterk |
| Tijd tot kritische massa | 12-24 maanden | 3-6 maanden |

### De Concurrent Dreiging

```
AI-tools kunnen dit concept binnen weken nabouwen.
Wie eerst 5K+ users heeft met goede reputatie, wint.
Gratis = claim de niche voordat iemand anders het doet.
```

### Monetization Pad

1. **V1:** Gratis — bouw userbase + reputatie
2. **V2:** Premium tier — multi-property, multi-country, AI support
3. **V3:** B2B — energieleveranciers, laadpaal operators

---

## GEEN API KEY (V1)

### Beslissing

Geen API key vereist voor gebruik. Zero frictie = maximale adoptie.

### Anti-Abuse

| Mechanisme | Implementatie |
|------------|---------------|
| Rate limiting | Per IP (niet per key) |
| Metrics | HA installation_id (anoniem) |
| Abuse detectie | Request patterns monitoring |

### Communicatiekanalen (zonder email)

| Kanaal | Doel |
|--------|------|
| HA integration changelog | Release notes |
| GitHub releases | Technische updates |
| Discord/community | Support, announcements |
| README | Documentatie |

---

## KEIHARDE WAARDE SYNCTACLES

Wat rechtvaardigt gebruik (ook al is het gratis)?

| Waarde | Omschrijving |
|--------|--------------|
| **Zero-config intelligence** | GO/WAIT/AVOID zonder nadenken |
| **6-tier fallback stack** | Altijd werkend, zelf bouwen = nightmare |
| **Onderhoud** | API's veranderen, SYNCTACLES fixt |
| **Bundeling** | Eén integratie vs 3-4 losse bronnen |
| **Best Window algoritme** | Variabele duration, server-berekend |

### SYNCTACLES vs Enever

| Aspect | Enever (€12.50/jaar) | SYNCTACLES (gratis) |
|--------|----------------------|---------------------|
| Levert | Data (prijzen) | Intelligence (acties) |
| GO/WAIT/AVOID | ❌ | ✅ |
| Best Window berekening | ❌ | ✅ |
| Zero-config | ❌ | ✅ |
| Install and forget | ❌ | ✅ |

---

## PREMIUM MODEL FILOSOFIE

### Kernprincipe

```
Premium = gemak + support, NIET exclusiviteit.
Client-side features zijn niet afdwingbaar (open source).
```

### Implicaties

| Aspect | Betekenis |
|--------|-----------|
| Pricing | Laag houden (€5 max), anders bouwt dev het zelf |
| Value prop | "Bespaar jezelf de moeite" |
| Doelgroep premium | Niet-technische HA users, tijd > geld |
| Doelgroep gratis | Devs, tinkerers, prijsbewust |

### Enforcement Realiteit

| Waar | Afdwingbaar? |
|------|--------------|
| Server-side berekening | ✅ Ja |
| Client-side (HA integration) | ❌ Nee, open source |

---

## PREMIUM FEATURES ANALYSE

### ❌ AFGEWEZEN voor Premium

| Feature | Reden |
|---------|-------|
| Historical data (30+ dagen) | Geen behoefte bij HA users |
| Live Cost + Savings | Gratis voor stickiness, client-side |
| 15-min resolution | Enever's upsell, niet de onze |
| Best Window | Gratis (best_4_hours schrappen, best_window met variabele duration) |
| Tomorrow Preview | Niet actionable genoeg |
| API access | Rate limits = anti-abuse, niet paywall |
| CSV exports | Niemand wil historical data export |
| Webhook notifications | HA doet dit al |
| YAML Generator / Claude Agent | Niet KISS, te veel bewegende delen |
| Priority support | Niet schaalbaar als solo developer |
| Dashboard/webapp | Andere doelgroep, geen actie-waarde |
| Leverancier vergelijking | Bestaat al (Pricewise, etc.) |
| Hardware integraties | Buiten scope, HA lost dit op |

### ✅ GRATIS als Adoptie Driver

| Feature | Waarom gratis |
|---------|---------------|
| Blueprints | Publiek deelbaar (YAML), geen paywall mogelijk |
| Live Cost + Savings | Stickiness, "ik bespaar €X" gevoel |
| Best Window | Core value, variabele duration |
| Tomorrow Preview | Nice-to-have awareness |

### ⏸️ ROADMAP Premium (Later)

| Feature | Rationale | Enforcement |
|---------|-----------|-------------|
| Multi-property | Meerdere HA instances, zakelijk gebruik | HA installation_id binding |
| Multi-country (DE/FR/BE) | Extra infra/maintenance | Geografische waarde-afbakening |
| AI Support | Schaalbaar, API kosten = per gebruik | Server-side |

---

## MULTI-PROPERTY ENFORCEMENT

### Mechanisme

```
1. API key (later) bindt aan eerste HA installation_id
2. 2e instance met zelfde key → reject
3. Multi-property premium → unlock meerdere instance IDs per key
```

### Realiteit Check

| User type | Gaat spoofen? |
|-----------|---------------|
| Casual user | Nee |
| Tech-savvy | Misschien, maar effort vs €5/maand |
| Determined freeloader | Ja, maar betaalt sowieso nooit |

**Conclusie:** "Good enough" enforcement. 90% gestopt, 10% was toch nooit klant.

---

## B2B STRATEGIE

### Timing

Na 500+ B2C users (bewijs van waarde).

### Targets

| Partij | Potentieel |
|--------|------------|
| Energieleverancier (white-label) | €500-2000/maand |
| Laadpaal operators | €1000-5000/maand |
| Installateurs | €200-500/maand |
| Woningcorporaties | €2000-10000/project |

### Pricing Framework (B2B)

| Model | Prijs |
|-------|-------|
| Per request | €0.001 |
| Per user/maand | €1-1.50 |
| Flat fee | €500-2000/maand |
| Enterprise | Custom |

**Onderhandelingsprincipe:** Wie eerst prijs noemt, verliest.

---

## EXIT STRATEGIE

### Oud Denken vs Nieuw Denken

| Oud | Nieuw |
|-----|-------|
| MRR opbouwen (€2.5K/maand) | Userbase opbouwen (10K+ users) |
| SaaS metrics | Platform metrics |
| Revenue = waarde | Adoptie + data = waarde |

### Exit Waarde

```
Platform met:
- 10K+ actieve users
- Unieke historische dataset
- Community + reputatie
- B2B partnerships
```

---

## RATE LIMITS ANALYSE

### Realiteit Check

| Data type | Calls nodig |
|-----------|-------------|
| Uurprijzen (vooraf bekend) | ~2/dag = 60/maand |
| Smart caching (huidige impl) | ~31/maand |
| Actieve HA user | ~150/dag max |

**Conclusie:** 1000 req/dag = ruim voldoende voor iedereen. Rate limits zijn anti-abuse tool, niet monetization tool.

### Finale Rate Limit Strategie

| Gebruik | Limiet |
|---------|--------|
| Normaal gebruik | 1000/dag (ruim) |
| Abuse/scraping | IP block |
| B2B | Custom afspraken |

---

## TECHNISCHE BESLISSINGEN (Uit Strategie)

### best_4_hours → best_window ✅ GEÏMPLEMENTEERD

**Beslissing:** Schrap `best_4_hours` attribuut, behoud `best_window` sensor met variabele duration parameter (user configureerbaar, default 3 uur).

**Rationale:** Dubbele functionaliteit, best_window is completere abstractie.

**Status:** Geïmplementeerd in v2.3.0 (2026-01-21). `best_4_hours` attribuut verwijderd uit CheapestHourSensor.

### Lokale Scheduling ✅ GEÏMPLEMENTEERD

**Verbetering:** HA zet lokale timers op basis van ontvangen prijsarray, actions triggeren exact op uurgrens i.p.v. polling-afhankelijk.

**Rationale:** Prijzen zijn vooraf bekend, geen 14-min delay meer op uurtransities.

**Status:** Geïmplementeerd in v2.4.0 (2026-01-21). `_schedule_hourly_updates()` in ServerDataCoordinator.

---

## LAUNCH CHECKLIST

### Vóór Launch

- [ ] API key requirement verwijderen
- [ ] Rate limiting op IP implementeren
- [ ] HA installation_id metrics (anoniem)
- [ ] Blueprints publiceren (GitHub)
- [ ] Discord/community opzetten
- [ ] README + documentatie compleet

### Na Launch

- [ ] Adoptie metrics monitoren
- [ ] Community feedback verzamelen
- [ ] Blueprints uitbreiden op basis van vraag
- [ ] B2B outreach na 500 users

---

## COMMUNICATIE BOODSCHAP

### Naar Users

> "SYNCTACLES is gratis. Voor altijd? Geen idee. Maar nu wel. 
> Install, configure, forget. Wij regelen de rest."

### Naar Community

> "Open source HA integration. Data van ENTSO-E, Enever, Energy-Charts.
> Wij bundelen, normaliseren, en geven je GO/WAIT/AVOID.
> Geen account nodig, geen API key, geen bullshit."

### Naar Potentiële B2B

> "X actieve HA users in Nederland vertrouwen op onze energy actions.
> White-label mogelijk. Laten we praten."

---

## RISICO'S EN MITIGATIE

| Risico | Impact | Mitigatie |
|--------|--------|-----------|
| Concurrent bouwt zelfde (gratis) | Hoog | First-mover, community, reputatie |
| Server kosten zonder revenue | Matig | Hetzner goedkoop, schaalbaar |
| Abuse/scraping | Laag | IP rate limiting |
| Support overload | Matig | Docs, Discord, community-driven |
| Geen pad naar monetization | Laag | B2B + premium roadmap |

---

## SAMENVATTING

```
V1 STRATEGIE:
├── Prijs: GRATIS
├── API key: GEEN
├── Rate limits: Anti-abuse, niet paywall
├── Premium: Later (multi-property, multi-country, AI support)
├── B2B: Na 500+ users
└── Exit: Userbase + data + partnerships
```

**Kernbeslissing:** De €12-24/jaar die we mislopen is niets vs de marktpositie die we winnen.

---

*Versie: 1.0*
*Datum: 2026-01-21*
*Gebaseerd op: Strategische sessie CAI*
