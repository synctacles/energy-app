# SKILL 14 — COEFFICIENT BUSINESS MODEL

**Classificatie:** VERTROUWELIJK  
**Versie:** 1.0  
**Datum:** 2026-01-09

---

## OVERZICHT

Dit document beschrijft het coefficient-gebaseerde business model voor SYNCTACLES. De coefficient is het kern-IP dat het product onderscheidt van gratis alternatieven.

---

## PROBLEEM

### Huidige Situatie

| Component | Bron | Probleem |
|-----------|------|----------|
| Energy Action | ENTSO-E wholesale | Niet wat user betaalt |
| BYO Enever | User's key | Data mag niet doorgestuurd |
| BYO TenneT | User's key | Data mag niet doorgestuurd |

### Gevolg

- Energy Action kan verkeerde aanbevelingen geven
- User met Enever ziet echte prijzen, vindt onze data overbodig
- Geen lock-in, geen betalingsreden

---

## OPLOSSING: COEFFICIENT MODEL

### Concept

```
coefficient = consumer_prijs / wholesale_prijs

SYNCTACLES berekent coefficient uit:
- Eigen Enever key (niet user's key)
- Vergelijking met ENTSO-E

User krijgt:
- Energy Action gebaseerd op realistische prijzen
- Zonder te weten hoe het berekend wordt
```

### Waarom Dit Werkt

| Aspect | Voordeel |
|--------|----------|
| Juridisch | Geen 1-op-1 data doorgifte |
| IP | Coefficient logica is geheim |
| Accuraat | Gebaseerd op echte consumer prijzen |
| Lock-in | Zonder licentie geen Energy Action |

---

## ARCHITECTUUR

### Twee Servers

```
COEFFICIENT SERVER (vertrouwelijk)
├── Locatie: 91.99.150.36
├── Repo: DATADIO/coefficient-engine (PRIVATE)
├── Functie: Berekent coefficient
└── Access: Alleen SYNCTACLES IP

SYNCTACLES SERVER (publiek)
├── Locatie: 135.181.255.83
├── Repo: synctacles/synctacles-api
├── Functie: Energy Actions, API voor users
└── Access: Publiek via enin.xteleo.nl
```

### Waarom Gescheiden

- Als SYNCTACLES lekt, is coefficient logica veilig
- Verschillende access policies
- Toekomstige B2B uitbreiding op coefficient server

---

## FEATURE GATING

### Gratis vs Betaald

| Feature | Gratis (ENIN) | Betaald (SYNCTACLES) |
|---------|---------------|----------------------|
| Prijzen (ENTSO-E) | ✅ | ✅ |
| Generation/Load | ✅ | ✅ |
| Grid Stress | ✅ | ✅ |
| BYO Display | ✅ | ✅ |
| **Energy Action** | ❌ | ✅ |
| **Cheapest Hour** | ❌ | ✅ |
| **Most Expensive Hour** | ❌ | ✅ |
| Priority Support | ❌ | ✅ |

### Implementatie

```
Geen licentie → API geeft geen Energy Action
BYO data = alleen display, geen intelligence
Killer features = server-side, niet in HA component
```

---

## COEFFICIENT STABILITEIT

### Analyse Nodig

| Vraag | Impact |
|-------|--------|
| Is coefficient stabiel per maand? | Lookup tabel vs real-time |
| Varieert per seizoen? | Seizoens-correctie nodig? |
| Varieert per leverancier? | Per-leverancier coefficient? |

### Scenarios

**Als stabiel (CV < 10%):**
```sql
-- Statische lookup tabel
SELECT coefficient 
FROM coefficient_lookup 
WHERE month = 1 AND day_type = 'weekday' AND hour = 14;
```

**Als instabiel (CV > 10%):**
```python
# Real-time berekening
coefficient = calculate_from_live_enever()
```

---

## MIGRATIE STRATEGIE

### Van ENIN naar SYNCTACLES

| Type | User Experience | Effort |
|------|-----------------|--------|
| Hard | Nieuwe integration | 0 uur |
| Soft | Config aanpassen | 2-4 uur |
| Smooth | Automatisch | 8-16 uur |

### Gekozen: Soft Migratie

```yaml
# User past aan:
energy_insights_nl:
  api_url: https://api.synctacles.io  # was enin.xteleo.nl
  api_key: sk_live_xxxxx              # nieuw
```

Entities blijven werken, automations intact.

---

## CRITERIA VOOR VOLGENDE MARKT

### Wanneer NL Verlaten

| Criterium | Threshold |
|-----------|-----------|
| Betalende users | ≥50 |
| MRR | ≥€500 |
| Churn | <10%/maand |
| Support load | ≤2 uur/week |
| Uptime | >99% over 30 dagen |

Als 3/5 gehaald → klaar voor DE.

### EU Uitbreiding Uitdaging

| Land | Enever-equivalent | Status |
|------|-------------------|--------|
| DE | aWATTar? Tibber? | ❓ Research nodig |
| BE | Onbekend | ❓ Research nodig |
| AT | aWATTar | ❓ Research nodig |

**Fallback:** User voert eigen tarief in (opslag + belasting).

---

## TOEKOMSTIGE B2B

Coefficient server evolueert naar:

| Fase | Functie |
|------|---------|
| Nu | Coefficient voor SYNCTACLES |
| Later | Historische data API |
| Toekomst | B2B coefficient service |

Mogelijk verdienmodel: verkoop coefficient aan andere energie-apps.

---

## VERTROUWELIJKHEID

### Wat Geheim Is

- Coefficient berekening logica
- Coefficient server IP/locatie
- Enever/TenneT API keys (Leo's keys)
- Historische coefficient data

### Wat Publiek Mag

- Dat Energy Action bestaat
- Dat het "intelligente" aanbevelingen geeft
- Dat het meerdere bronnen combineert

### Nooit Delen

- Hoe coefficient berekend wordt
- coefficient-engine repo
- Dit SKILL document

---

## GERELATEERDE DOCUMENTEN

- HANDOFF_CC_COEFFICIENT_ENGINE.md (technische setup)
- LAUNCH_PLAN.md (planning)
- ADR-003 (nog te schrijven: coefficient architectuur beslissing)

---

## CHANGELOG

| Versie | Datum | Wijziging |
|--------|-------|-----------|
| 1.0 | 2026-01-09 | Initiële versie |
