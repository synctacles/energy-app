# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** MEDIUM
**Type:** Documentation Update

---

## CONTEXT

SKILL docs zijn bijgewerkt met Enever (commit 41a00c7). User-facing docs (README, user-guide) noemen Enever nog niet.

---

## TASK

Update README.md en user-guide.md met Enever informatie.

---

## README.md

**Locatie:** `/opt/github/synctacles-api/README.md`

**Actie 1:** In "Features" of "What it provides" sectie, voeg toe:

```markdown
### Data Sources
- **ENTSO-E** - European grid data (generation, load, wholesale prices)
- **TenneT** - Dutch grid balance (BYO-key in HA component)
- **Enever.nl** - Leverancier-specific consumer prices (BYO-key in HA component)
- **Energy-Charts** - Fallback data source
```

**Actie 2:** In HA component sectie (als aanwezig), voeg toe:

```markdown
### BYO-Key Features (Home Assistant)

De HA component ondersteunt optionele BYO-keys voor extra functionaliteit:

| Feature | Data | Update Interval |
|---------|------|-----------------|
| TenneT API key | Real-time grid balance | 60 seconden |
| Enever.nl token | Leverancier-specific prijzen | 1 uur |

**Enever voordelen:**
- Echte consumentenprijzen (niet wholesale)
- 19 Nederlandse leveranciers ondersteund
- Inclusief belastingen en leveranciers-opslag
```

---

## user-guide.md

**Locatie:** `/opt/github/synctacles-api/docs/user-guide.md`

**Actie 1:** Voeg nieuwe sectie toe "Enever.nl Integratie" (na TenneT sectie als die bestaat):

```markdown
## Enever.nl Integratie (Optioneel)

Enever.nl biedt leverancier-specifieke stroomprijzen - de prijs die je daadwerkelijk betaalt, niet de wholesale prijs.

### Waarom Enever?

| Aspect | ENTSO-E (Server) | Enever (BYO) |
|--------|------------------|--------------|
| Prijstype | Wholesale | Consument |
| Inclusief BTW | Nee | Ja |
| Leverancier opslag | Nee | Ja |
| Resolutie | Uurlijks | Uurlijks (15-min voor supporters) |

### Setup

1. Registreer op https://enever.nl/
2. Kopieer je API token
3. In Home Assistant: Instellingen → Integraties → Energy Insights NL → Configureren
4. Voer token in + selecteer je leverancier

### Ondersteunde Leveranciers

Tibber, Zonneplan, Frank Energie, ANWB Energie, Greenchoice, Eneco, Vattenfall, 
Essent, Budget Energie, Oxxio, Engie, United Consumers, Vandebron, Next Energy,
Mijndomein Energie, Innova Energie, Energie VanOns, Gewoon Energie, DELTA Energie

### Sensors

Na configuratie verschijnen 2 extra sensors:
- `sensor.energy_insights_nl_prices_today` - Uurprijzen vandaag
- `sensor.energy_insights_nl_prices_tomorrow` - Uurprijzen morgen (na 15:00)

### Smart Caching

De component haalt morgen-prijzen automatisch op na 15:00 en promoveert deze om middernacht. Dit resulteert in ~31 API calls/maand in plaats van ~62.
```

**Actie 2:** Als er een "Sensors" overzicht is, voeg Enever sensors toe:

```markdown
### Enever Sensors (indien geconfigureerd)
| Sensor | Beschrijving |
|--------|--------------|
| `prices_today` | 24 uurprijzen vandaag (€/kWh) |
| `prices_tomorrow` | 24 uurprijzen morgen (na 15:00) |
```

---

## VERIFICATION

```bash
grep -c "Enever" /opt/github/synctacles-api/README.md
# Verwacht: > 0

grep -c "Enever" /opt/github/synctacles-api/docs/user-guide.md
# Verwacht: > 0
```

---

## GIT COMMIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add README.md docs/user-guide.md
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: add Enever.nl to user-facing documentation

- README: Add Enever as data source, BYO-key features table
- user-guide: Add Enever setup instructions and sensor list

Completes Enever documentation across all doc types."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main
```

---

## OUT OF SCOPE

- api-reference.md (Enever = HA-only, geen server endpoints)
- Geen code changes

---

*Template versie: 1.0*
