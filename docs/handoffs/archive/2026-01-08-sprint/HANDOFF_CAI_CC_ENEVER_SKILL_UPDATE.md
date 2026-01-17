# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** Documentation Update

---

## CONTEXT

CAI's SKILLs hebben een blindspot: **Enever.nl** bestaat nergens in de documentatie, maar is volledig geïmplementeerd in de HA component (123 lines, 2 sensors, 19 leveranciers).

Code is verder dan docs. Dit moet gefixed.

---

## TASK

Update 3 SKILL bestanden met Enever informatie.

---

## SKILL_06_DATA_SOURCES.md

**Locatie:** `/opt/github/synctacles-api/docs/SKILL_06_DATA_SOURCES.md`

**Actie:** Voeg nieuwe sectie toe na TenneT sectie:

```markdown
### 4. Enever.nl (Dutch Energy Pricing - BYO-KEY)

⚠️ **LICENSE NOTICE:** Enever.nl data via BYO-key in Home Assistant component only.

**Website:** https://enever.nl/

**What They Provide:**
- Leverancier-specific electricity prices (not just ENTSO-E wholesale)
- Real consumer prices including taxes, markup, delivery costs
- Day-ahead prices (today + tomorrow)
- Support for 19 Dutch energy suppliers

**Supported Leveranciers:**
Tibber, Zonneplan, Frank Energie, ANWB Energie, Greenchoice, Eneco, Vattenfall, 
Essent, Budget Energie, Oxxio, Engie, United Consumers, Vandebron, Next Energy,
Mijndomein Energie, Innova Energie, Energie VanOns, Gewoon Energie, DELTA Energie

**Access Method:** API token (user registers at enever.nl)

**SYNCTACLES Integration:**
- ❌ **NOT available via SYNCTACLES API** (BYO-key only)
- ✅ **Available via Home Assistant component** with user's Enever token
- User registers at https://enever.nl/
- User enters token + selects leverancier in HA config
- Data fetched locally in Home Assistant

**API Details:**
- **Update interval:** 1 hour
- **Smart caching:** ~31 API calls/month (vs ~62 without caching)
- **Resolution:** 60-min default, 15-min for supporters + compatible suppliers
- **Tomorrow prices:** Available after 15:00

**Data Points:**
- Hourly prices today (24 values)
- Hourly prices tomorrow (24 values, after 15:00)
- Price includes all leverancier-specific costs

**Reliability:**
- Dependent on enever.nl uptime
- Fallback to ENTSO-E server prices if unavailable

**Cost:** Free tier available, supporter tier for 15-min resolution
```

**Actie:** Update PRIMARY SOURCES header naar "4 Primary Sources"

---

## SKILL_02_ARCHITECTURE.md

**Locatie:** `/opt/github/synctacles-api/docs/SKILL_02_ARCHITECTURE.md`

**Actie 1:** In System Architecture diagram, voeg toe onder "HOME ASSISTANT":

```
HOME ASSISTANT
├── ServerDataCoordinator (15-min) → Server API
├── TennetDataCoordinator (60s) → TenneT API (BYO-key)
└── EneverDataCoordinator (1hr) → Enever.nl API (BYO-key)
```

**Actie 2:** Voeg nieuwe sectie toe na "TenneT BYO-KEY" sectie:

```markdown
### Enever.nl BYO-Key Architecture

**Purpose:** Leverancier-specific pricing (consumer prices, not wholesale)

**Flow:**
```
User's Enever Token
        ↓
HA Component (EneverDataCoordinator)
        ↓
https://api.enever.nl/
        ↓
Prices Today + Tomorrow sensors
```

**Coordinator Details:**
- **Update interval:** 1 hour
- **Smart caching:** Fetches tomorrow after 15:00, promotes at midnight
- **API reduction:** ~50% fewer calls than naive polling
- **Fallback:** ENTSO-E server prices if Enever unavailable

**Sensors Created (conditional):**
- `sensor.energy_insights_nl_prices_today` - Hourly prices today
- `sensor.energy_insights_nl_prices_tomorrow` - Hourly prices tomorrow

**Resolution Tiers:**
- Default: 60-min (24 points/day)
- Supporter + compatible supplier: 15-min (96 points/day)
```

---

## SKILL_04_PRODUCT_REQUIREMENTS.md

**Locatie:** `/opt/github/synctacles-api/docs/SKILL_04_PRODUCT_REQUIREMENTS.md`

**Actie:** Voeg nieuwe capability sectie toe:

```markdown
### 7. Leverancier-Specific Pricing (Enever BYO-Key)

**What:** Real consumer electricity prices per leverancier

**Data Source:** Enever.nl API (BYO-key in HA component)

**Provides:**
- Hourly prices today (€/kWh, consumer price)
- Hourly prices tomorrow (available after 15:00)
- Leverancier-specific markup and taxes included
- 19 Dutch energy suppliers supported

**Difference from ENTSO-E Prices:**
| Aspect | ENTSO-E (Server) | Enever (BYO) |
|--------|------------------|--------------|
| Price type | Wholesale | Consumer |
| Includes taxes | No | Yes |
| Leverancier markup | No | Yes |
| Resolution | Hourly | Hourly (15-min for supporters) |

**Use Cases:**
- Accurate cost calculation per leverancier
- Compare actual prices vs wholesale
- Optimize based on real consumer costs

**Endpoint:** Via HA component only (BYO-key)
```

---

## VERIFICATION

Na updates, verify:

```bash
# Check Enever mentioned in all 3 files
grep -l "Enever" /opt/github/synctacles-api/docs/SKILL_0{2,4,6}*.md
# Should return 3 files
```

---

## GIT COMMIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add docs/
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: add Enever.nl BYO-key to SKILL documentation

- SKILL_06: Enever as 4th data source
- SKILL_02: EneverDataCoordinator architecture
- SKILL_04: Leverancier-specific pricing capability

Fixes documentation blindspot - code was ahead of docs."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main
```

---

## OUT OF SCOPE

- Geen code changes
- Geen HA component wijzigingen
- Alleen SKILL documentation updates

---

*Template versie: 1.0*
