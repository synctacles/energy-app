# CC PLAN: TenneT BYO-Key Migration

**Datum:** 2025-01-02
**Doel:** TenneT data verplaatsen van server-side naar HA component (BYO-key)
**Reden:** TenneT API licentie verbiedt server-side redistributie

---

## CONTEXT

TenneT API Gateway General Terms verbieden "distributing, selling, or sharing data obtained through the APIs with third parties". Onze server-side TenneT collector/API endpoint is een schending.

**Oplossing:** TenneT data wordt lokaal opgehaald in de HA component met user's eigen API key (BYO = Bring Your Own).

---

## FASE 1: SERVER-SIDE CLEANUP

### 1.1 Systemd Services Stoppen

**Locatie:** Server via SSH

```bash
# Stop en disable TenneT timer/service
sudo systemctl stop energy-insights-nl-tennet.timer 2>/dev/null || true
sudo systemctl stop energy-insights-nl-tennet.service 2>/dev/null || true
sudo systemctl disable energy-insights-nl-tennet.timer 2>/dev/null || true
sudo systemctl disable energy-insights-nl-tennet.service 2>/dev/null || true

# Verifieer
systemctl list-units "energy-insights-nl-tennet*"
# Verwacht: geen resultaten
```

### 1.2 Run Scripts Aanpassen

**Bestand:** `/opt/energy-insights-nl/app/scripts/run_collectors.sh`

**Actie:** Verwijder TenneT sectie (indien aanwezig)

```bash
# Zoek en verwijder regels met "tennet" (case insensitive)
# Bewaar backup eerst
```

**Bestand:** `/opt/energy-insights-nl/app/scripts/run_importers.sh`

**Actie:** Verwijder TenneT sectie (indien aanwezig)

**Bestand:** `/opt/energy-insights-nl/app/scripts/run_normalizers.sh`

**Actie:** Verwijder TenneT sectie (indien aanwezig)

### 1.3 API Endpoint Aanpassen

**Bestand:** `/opt/energy-insights-nl/app/synctacles_db/api/routes/` (zoek balance route)

**Actie:** Vervang `/v1/balance/current` implementatie met stub:

```python
@router.get("/v1/balance/current")
async def get_balance_current():
    """
    Grid balance data - Available via BYO-key in Home Assistant component.
    
    TenneT API license prohibits server-side redistribution.
    Configure your TenneT API key in Home Assistant for real-time balance data.
    """
    return JSONResponse(
        status_code=501,
        content={
            "error": "Not Implemented",
            "message": "Balance data available via BYO-key in HA component",
            "documentation": "https://github.com/DATADIO/ha-energy-insights-nl#tennet-byo-key",
            "reason": "TenneT API license prohibits server-side redistribution"
        }
    )
```

### 1.4 Environment Cleanup

**Bestand:** `/opt/.env`

**Actie:** Verwijder of comment TENNET_API_KEY regel

```bash
# Backup eerst
sudo cp /opt/.env /opt/.env.backup.$(date +%Y%m%d)

# Verwijder TENNET_API_KEY regel
sudo sed -i '/^TENNET_API_KEY=/d' /opt/.env

# Of comment uit:
# sudo sed -i 's/^TENNET_API_KEY=/#TENNET_API_KEY=/' /opt/.env
```

### 1.5 Database Archiveren

**Actie:** Rename TenneT tabellen naar archive_*

```sql
-- Voer uit via psql -U synctacles -d synctacles

-- Archiveer raw TenneT data
ALTER TABLE IF EXISTS raw_tennet_balance 
RENAME TO archive_raw_tennet_balance;

-- Archiveer normalized TenneT data  
ALTER TABLE IF EXISTS norm_grid_balance 
RENAME TO archive_norm_grid_balance;

-- Verifieer
\dt *tennet*
\dt *balance*
-- Verwacht: alleen archive_* tabellen
```

### 1.6 Collector/Importer/Normalizer Files

**Actie:** Verplaats naar archive folder (niet verwijderen)

```bash
cd /opt/energy-insights-nl/app/synctacles_db

# Maak archive folders
mkdir -p collectors/archive
mkdir -p importers/archive
mkdir -p normalizers/archive

# Verplaats TenneT files (indien aanwezig)
mv collectors/tennet_ingestor.py collectors/archive/ 2>/dev/null || true
mv collectors/tennet_*.py collectors/archive/ 2>/dev/null || true
mv importers/import_tennet*.py importers/archive/ 2>/dev/null || true
mv normalizers/normalize_tennet*.py normalizers/archive/ 2>/dev/null || true
```

---

## FASE 2: DOCUMENTATIE UPDATES

### 2.1 SKILL_06_DATA_SOURCES.md

**Locatie:** `/mnt/project/SKILL_06_DATA_SOURCES.md`

**Wijzigingen:**

1. **Sectie "2. TenneT"** - Herschrijf naar:

```markdown
### 2. TenneT (Dutch Transmission System Operator) - BYO-KEY ONLY

**⚠️ LICENSE NOTICE:** TenneT API terms prohibit server-side redistribution.
TenneT data is available via BYO-key (Bring Your Own) in the Home Assistant component only.

**Website:** https://www.tennet.eu/

**What They Provide:**
- Dutch grid-specific data
- Frequency, reserve margins
- Grid stress events

**Access Method:** HTTP API with personal API key

**SYNCTACLES Integration:**
- ❌ NOT available via SYNCTACLES API (license restriction)
- ✅ Available via Home Assistant component with user's own TenneT API key
- User registers at TenneT Developer Portal
- User enters personal API key in HA integration config
- Data fetched locally, never passes through SYNCTACLES servers

**Key Data Points:**
[... rest blijft hetzelfde ...]
```

2. **Sectie "AUTHENTICATION & CREDENTIALS"** - Update TenneT deel:

```markdown
### TenneT (BYO-Key in HA Component)

**NOT configured on server** - TenneT API key is personal and configured in Home Assistant.

Users obtain their own key from: https://www.tennet.eu/developer-portal/

**HA Configuration:**
```yaml
# Home Assistant configuration
synctacles:
  api_key: "YOUR_SYNCTACLES_API_KEY"
  tennet_api_key: "YOUR_PERSONAL_TENNET_KEY"  # Optional, enables balance data
```
```

3. **Sectie "FALLBACK STRATEGY"** - Verwijder TenneT uit server fallback, voeg note toe:

```markdown
### Balance Data

**Server-side:** Not available (TenneT license restriction)

**HA Component (with BYO-key):**
- Real-time from TenneT API (5-min updates)
- No fallback (TenneT-only data)
- If no BYO-key configured: balance sensors disabled
```

4. **Sectie "ERROR HANDLING"** - Update "When TenneT Fails":

```markdown
### When TenneT Fails (HA Component)

TenneT errors are handled locally in the Home Assistant component.
Server-side has no TenneT dependency.

**Common Causes (user's HA):**
1. Invalid/expired personal API key
2. TenneT rate limit exceeded
3. Network issues

**HA Component Response:**
- Sensor shows "unavailable"
- Logs error to HA system log
- Does not affect other SYNCTACLES sensors
```

5. **Sectie "MONITORING DATA SOURCES"** - Verwijder TenneT uit server health check:

```python
@app.get("/health/sources")
async def source_health():
    """Report status of each data source."""
    return {
        "entso_e_a75": { ... },
        "entso_e_a65": { ... },
        "entso_e_a44": { ... },
        "energy_charts": { ... }
        # TenneT removed - BYO-key only in HA component
    }
```

---

### 2.2 SKILL_02_ARCHITECTURE.md

**Locatie:** `/mnt/project/SKILL_02_ARCHITECTURE.md`

**Wijzigingen:**

1. **Component Overview diagram** - Update:

```
EXTERNAL SOURCES
├── ENTSO-E (Generation, Load, Prices)
├── Energy-Charts (Fallback)
└── TenneT (BYO-key via HA only, not server)
```

2. **Sectie "LAYER 1: COLLECTORS"** - Verwijder `tennet_ingestor.py` uit lijst

3. **Database schema** - Voeg note toe bij norm_grid_balance:

```markdown
**Note:** `norm_grid_balance` table archived. Balance data now via BYO-key in HA component.
```

---

### 2.3 api-reference.md

**Locatie:** `/mnt/project/api-reference.md`

**Wijzigingen:**

1. **Sectie "GET /v1/balance/current"** - Vervang met:

```markdown
### GET /v1/balance/current
**⚠️ DEPRECATED - Returns 501 Not Implemented**

Grid balance data is no longer available via the SYNCTACLES API due to TenneT license restrictions.

**Response:**
```json
{
  "error": "Not Implemented",
  "message": "Balance data available via BYO-key in HA component",
  "documentation": "https://github.com/DATADIO/ha-energy-insights-nl#tennet-byo-key",
  "reason": "TenneT API license prohibits server-side redistribution"
}
```

**Alternative:** Configure your personal TenneT API key in the Home Assistant integration to enable real-time balance data locally.

**Update Interval:** N/A (endpoint deprecated)
**Data Source:** Available via BYO-key in HA component only
```

2. **Sectie "Data Sources"** - Update:

```markdown
## Data Sources

**Server-side (via API):**
- ENTSO-E Transparency Platform (Generation, Load, Prices)
- Energy-Charts (Fraunhofer ISE) - Fallback

**Client-side (HA Component with BYO-key):**
- TenneT TSO API (Grid Balance) - requires user's personal API key

**Attribution:**
- ENTSO-E data: [transparency.entsoe.eu](https://transparency.entsoe.eu)
- Energy-Charts: CC BY 4.0, Fraunhofer ISE
- TenneT: User's personal API key, local processing only
```

---

### 2.4 user-guide.md

**Locatie:** `/mnt/project/user-guide.md`

**Wijzigingen:**

1. **Sectie "Configure Integration"** - Voeg TenneT BYO-key toe:

```markdown
### 3. Configure Integration

1. Settings → Devices & Services → **Add Integration**
2. Search: **Energy Insights NL**
3. Enter:
   - **API Endpoint:** Your API server URL
   - **API Key:** (paste from step 1)
   - **TenneT API Key (optional):** Your personal TenneT key for balance data
4. Submit → **Sensors created** ✓

**TenneT BYO-Key (Optional):**
To enable real-time grid balance data:
1. Register at [TenneT Developer Portal](https://www.tennet.eu/developer-portal/)
2. Create an API key (free, personal use)
3. Enter in HA integration config
4. Balance sensors will appear after restart
```

2. **Sectie "Verify Installation"** - Update expected entities:

```markdown
**Expected entities (without TenneT key):**
- `sensor.energy_insights_nl_generation_total`
- `sensor.energy_insights_nl_load_actual`

**Additional entities (with TenneT BYO-key):**
- `sensor.energy_insights_nl_balance_delta`
- `sensor.energy_insights_nl_grid_stress`
```

3. **Nieuwe sectie "TenneT BYO-Key Setup":**

```markdown
## TenneT BYO-Key Setup (Optional)

Real-time grid balance data requires your personal TenneT API key.

### Why BYO-Key?

TenneT's API license prohibits server-side redistribution. Your personal key fetches data directly to your Home Assistant - it never passes through SYNCTACLES servers.

### Get Your TenneT Key

1. Visit: https://www.tennet.eu/developer-portal/
2. Create account (free)
3. Generate API key
4. Copy key securely

### Configure in Home Assistant

1. Settings → Devices & Services → SYNCTACLES
2. Click **Configure**
3. Enter TenneT API Key
4. Restart integration

### Available Sensors (with BYO-key)

| Sensor | Description |
|--------|-------------|
| `sensor.synctacles_balance_delta` | Grid balance MW (+surplus/-deficit) |
| `sensor.synctacles_grid_stress` | Grid stress indicator (0-100) |

### Troubleshooting

**Sensor shows "unavailable":**
- Verify TenneT key is correct
- Check HA logs for TenneT errors
- TenneT may have rate limits (100 req/min)
```

---

### 2.5 troubleshooting.md

**Locatie:** `/mnt/project/troubleshooting.md`

**Wijzigingen:**

1. **Nieuwe sectie "TenneT BYO-Key Issues":**

```markdown
## TenneT BYO-Key Issues

### Balance Sensors Missing

**Symptom:** No `balance_delta` or `grid_stress` sensors

**Cause:** TenneT API key not configured

**Fix:**
1. Settings → Devices & Services → SYNCTACLES → Configure
2. Add your personal TenneT API key
3. Restart integration

---

### TenneT "Invalid API Key"

**Symptom:** Balance sensors show "unavailable", logs show auth error

**Diagnosis:**
```bash
# Test your key directly
curl -H "Authorization: Bearer YOUR_TENNET_KEY" \
  https://api.tennet.eu/v1/balance
```

**Fixes:**
1. Regenerate key at TenneT Developer Portal
2. Verify key copied correctly (no extra spaces)
3. Check key hasn't expired

---

### TenneT Rate Limit

**Symptom:** Balance data intermittent, 429 errors in log

**Cause:** TenneT limit is 100 requests/minute

**Fix:**
- HA component polls every 5 minutes (safe margin)
- If multiple HA instances share key: use separate keys
```

---

### 2.6 ARCHITECTURE.md (main docs)

**Locatie:** `/mnt/project/ARCHITECTURE.md`

**Wijzigingen:**

1. **Executive Summary** - Update key capabilities:

```markdown
**Key Capabilities:**
- Real-time Dutch generation mix (9 PSR types) updated every 15 minutes
- Grid load forecasts with actual values
- Day-ahead electricity prices
- Grid balance data via BYO-key in HA component (TenneT license restriction)
```

2. **Component Overview diagram** - Verwijder TenneT uit server flow

3. **ADR toevoegen - ADR-008: TenneT BYO-Key Architecture**

```markdown
### ADR-008: TenneT BYO-Key Architecture

**Context:** TenneT API Gateway General Terms prohibit "distributing, selling, or sharing data obtained through the APIs with third parties". Server-side redistribution violates these terms.

**Decision:** Move TenneT data fetching from server-side to client-side (Home Assistant component). Users provide their own TenneT API key (BYO = Bring Your Own).

**Consequences:**
- ✅ Legally compliant (user's personal key, local processing)
- ✅ No server-side TenneT infrastructure to maintain
- ✅ Users control their own rate limits
- ✅ Demonstrates thoughtful legal architecture
- ❌ Requires user to obtain TenneT key separately
- ❌ Balance data optional (not all users will configure)

**Alternatives Considered:**
- Server-side redistribution (illegal)
- ENTSO-E proxy (60-90 min delay, inferior quality)
- Remove balance feature entirely (loses competitive advantage)
```

4. **Fallback Strategy** - Update Balance sectie:

```markdown
### Balance Data (HA Component Only)

Grid balance is fetched directly by the Home Assistant component using the user's personal TenneT API key.

**No server-side fallback** - TenneT is the only source for real-time balance data.

**If BYO-key not configured:**
- Balance sensors not created
- Grid stress signal unavailable
- Other signals (is_green, is_cheap) still work via server
```

---

## FASE 3: HA COMPONENT SPECIFICATIE (NIEUW)

### 3.1 HA Component Config Flow Update

**Bestand:** `custom_components/synctacles/config_flow.py`

**Wijzigingen:**

```python
# Voeg toe aan config schema
STEP_USER_DATA_SCHEMA = vol.Schema({
    vol.Required(CONF_API_KEY): str,
    vol.Required(CONF_API_ENDPOINT): str,
    vol.Optional(CONF_TENNET_API_KEY): str,  # NIEUW: BYO-key
})
```

### 3.2 HA Component TenneT Client

**Nieuw bestand:** `custom_components/synctacles/tennet_client.py`

```python
"""TenneT API client for BYO-key balance data."""

class TennetClient:
    """Client for fetching TenneT balance data locally."""
    
    def __init__(self, api_key: str):
        self.api_key = api_key
        self.base_url = "https://api.tennet.eu/v1"
    
    async def get_balance(self) -> dict:
        """Fetch current grid balance."""
        # Implementation
        pass
```

### 3.3 HA Component Sensor Updates

**Bestand:** `custom_components/synctacles/sensor.py`

**Wijzigingen:**

```python
# Conditional balance sensors
async def async_setup_entry(hass, entry, async_add_entities):
    sensors = [
        GenerationSensor(...),
        LoadSensor(...),
    ]
    
    # Add balance sensors only if TenneT key configured
    if entry.data.get(CONF_TENNET_API_KEY):
        sensors.extend([
            BalanceDeltaSensor(...),
            GridStressSensor(...),
        ])
    
    async_add_entities(sensors)
```

### 3.4 HA Component strings.json

**Bestand:** `custom_components/synctacles/strings.json`

**Toevoegen:**

```json
{
  "config": {
    "step": {
      "user": {
        "data": {
          "api_key": "SYNCTACLES API Key",
          "api_endpoint": "API Endpoint URL",
          "tennet_api_key": "TenneT API Key (optional, for balance data)"
        },
        "description": "Enter your SYNCTACLES credentials. TenneT key is optional but enables real-time grid balance data."
      }
    }
  }
}
```

---

## FASE 4: VERIFICATIE

### 4.1 Server Verificatie

```bash
# Geen TenneT services actief
systemctl list-units "*tennet*"
# Verwacht: leeg

# Geen TENNET_API_KEY in .env
grep -i tennet /opt/.env
# Verwacht: leeg of gecomment

# API endpoint returns 501
curl http://localhost:8000/v1/balance/current
# Verwacht: {"error": "Not Implemented", ...}

# Database tabellen gearchiveerd
psql -U synctacles -d synctacles -c "\dt *tennet*"
# Verwacht: alleen archive_* tabellen
```

### 4.2 Documentatie Verificatie

```bash
# Zoek naar server-side TenneT referenties (moeten minimaal zijn)
grep -r "TenneT" docs/ --include="*.md" | grep -v "BYO\|component\|optional"
# Verwacht: minimale hits, alleen historische context
```

---

## NIET AANPASSEN (Historische documenten)

Deze bestanden bevatten TenneT referenties maar zijn completion reports - niet wijzigen:

- `DAG_1_HA_Integration_Bevindingen.md`
- `DAG_2_COMPLETION_REPORT.md`
- `DAG_3_Completion_Report_Unified_A.md`
- `F4-LITE_COMPLETION_REPORT.md`
- `F5_COMPLETION_REPORT.md`
- `F6_COMPLETION_REPORT.md`
- `F7_COMPLETION_REPORT.md`
- `F8A_COMPLETION_REPORT.md`
- `F8B_COMPLETION_REPORT.md`
- `WEEK_1_COMPLETION_REPORT.md`
- `DEBRANDING_SESSION_SUMMARY.md`

---

## SAMENVATTING WIJZIGINGEN

| Categorie | Bestand | Actie |
|-----------|---------|-------|
| **Server** | systemd services | Stop + disable |
| **Server** | run_*.sh scripts | Verwijder TenneT secties |
| **Server** | API route | 501 stub |
| **Server** | /opt/.env | Verwijder TENNET_API_KEY |
| **Server** | Database | Rename naar archive_* |
| **Server** | Collector/Importer/Normalizer | Move naar archive/ |
| **Docs** | SKILL_06_DATA_SOURCES.md | BYO-key uitleg |
| **Docs** | SKILL_02_ARCHITECTURE.md | Verwijder server TenneT |
| **Docs** | api-reference.md | 501 deprecated |
| **Docs** | user-guide.md | BYO-key setup guide |
| **Docs** | troubleshooting.md | BYO-key troubleshooting |
| **Docs** | ARCHITECTURE.md | ADR-008 + updates |
| **HA** | config_flow.py | Optional TenneT key field |
| **HA** | tennet_client.py | Nieuw: lokale TenneT fetch |
| **HA** | sensor.py | Conditional balance sensors |
| **HA** | strings.json | UI labels |

---

## PRIORITEIT

1. **KRITISCH:** Server cleanup (Fase 1) - juridische compliance
2. **HOOG:** Documentatie updates (Fase 2) - consistency
3. **MEDIUM:** HA component spec (Fase 3) - feature completeness
4. **LAAG:** Verificatie (Fase 4) - quality assurance

---

**Einde CC Plan**
