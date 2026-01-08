# CC TASK 07: Update SKILL_13 met Definitieve Implementatie

**Project:** SYNCTACLES  
**Datum:** 2026-01-03  
**Vereist:** TASK 01-06 ✅  
**Type:** Documentatie update

---

## CONTEXT

Logging series is compleet. SKILL_13 moet geüpdatet worden met:
- Werkelijke bestandslocaties
- Geïmplementeerde patterns
- Concrete voorbeelden uit codebase

**BELANGRIJK:** Alle toekomstige code MOET aansluiten op SKILL_13. Dit is de standaard.

---

## DOEL

SKILL_13 updaten van "plan/gaps" naar "geïmplementeerde standaard":
- Verwijder "CURRENT STATE: GAPS TO FIX" secties
- Vervang door "IMPLEMENTED STANDARD"
- Voeg concrete code voorbeelden toe uit huidige codebase
- Documenteer exacte log locaties en configuratie

---

## STAP 1: Lees huidige implementatie

```bash
# Backend logging core
cat /opt/github/synctacles-api/synctacles_db/core/logging.py

# Voorbeeld collector met logging
cat /opt/github/synctacles-api/synctacles_db/collectors/entso_e_a75_generation.py | head -50

# Voorbeeld importer met logging  
cat /opt/github/synctacles-api/synctacles_db/importers/import_entso_e_a75.py | head -50

# Voorbeeld normalizer met logging
cat /opt/github/synctacles-api/synctacles_db/normalizers/normalize_entso_e_a75.py | head -50

# API middleware
cat /opt/github/synctacles-api/synctacles_db/api/middleware.py

# HA diagnostics
cat /opt/github/ha-energy-insights-nl/custom_components/ha_energy_insights_nl/diagnostics.py
```

---

## STAP 2: Update SKILL_13

**Bestand:** `/opt/github/synctacles-api/docs/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md`

**Structuur wijzigingen:**

### Verwijder/vervang:
- "CURRENT STATE: GAPS TO FIX" → verwijderen
- "TODO" secties → verwijderen
- "ACTION ITEMS" → vervangen door "MAINTENANCE CHECKLIST"

### Toevoegen/updaten:

**Sectie: IMPLEMENTED INFRASTRUCTURE**
```markdown
## IMPLEMENTED INFRASTRUCTURE

### Log Bestanden

| Bestand | Doel | Rotatie |
|---------|------|---------|
| `/var/log/synctacles/synctacles.log` | Pipeline (collectors, importers, normalizers) | 10MB × 4 |
| `/var/log/synctacles/api.log` | API requests | 10MB × 4 |

### Configuratie

**Backend (.env):**
```bash
LOG_LEVEL=warning          # off|error|warning|info|debug
LOG_PATH_FILE=/var/log/synctacles/synctacles.log
LOG_API_PATH=/var/log/synctacles/api.log
```

**HA (configuration.yaml):**
```yaml
logger:
  logs:
    custom_components.ha_energy_insights_nl: warning
```
```

**Sectie: BACKEND LOGGING PATTERN**
```markdown
## BACKEND LOGGING PATTERN

### Import Statement (VERPLICHT voor alle modules)

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)
```

### Standaard Function Pattern

```python
def process_data(self):
    """Process data with logging."""
    _LOGGER.info(f"{self.name} starting")
    start_time = time.time()
    
    try:
        # ... processing ...
        _LOGGER.debug(f"Processed {count} records")
        
        elapsed = time.time() - start_time
        _LOGGER.info(f"{self.name} completed: {count} records in {elapsed:.2f}s")
        
    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"{self.name} failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise
```
```

**Sectie: LOG LEVELS GEBRUIK**
```markdown
## LOG LEVELS GEBRUIK

| Level | Wanneer | Voorbeeld |
|-------|---------|-----------|
| DEBUG | Request/response details, record counts, timing | `_LOGGER.debug(f"Response: {status}, {size} bytes")` |
| INFO | Start/einde operaties, fallback activaties | `_LOGGER.info(f"Collector completed: {n} records")` |
| WARNING | Stale data, slow responses, aggregated failures | `_LOGGER.warning(f"Data age {age}min exceeds threshold")` |
| ERROR | Operatie failures met context | `_LOGGER.error(f"Failed: {type(err).__name__}: {err}")` |

### Regel: Geen Spam
- WARNING/ERROR: Nooit per-record (aggregeer)
- DEBUG: Mag per-record (alleen bij debug level)
```

**Sectie: HA INTEGRATION PATTERN**
```markdown
## HA INTEGRATION PATTERN

### Import Statement
```python
import logging

_LOGGER = logging.getLogger(__name__)
```

### Diagnostics (VERPLICHT)

Elk HA component moet `diagnostics.py` hebben:

```python
from homeassistant.components.diagnostics import async_redact_data

TO_REDACT = {"api_key", "tennet_api_key", "password", "token"}

async def async_get_config_entry_diagnostics(hass, entry):
    return {
        "config": async_redact_data(dict(entry.data), TO_REDACT),
        "coordinator": {...},
        "entities": {...},
    }
```
```

**Sectie: NIEUWE CODE CHECKLIST**
```markdown
## NIEUWE CODE CHECKLIST

Voordat code gemerged wordt:

### Backend Module
- [ ] `from synctacles_db.core.logging import get_logger`
- [ ] `_LOGGER = get_logger(__name__)`
- [ ] INFO bij start/einde met timing
- [ ] DEBUG voor details
- [ ] ERROR met `{type(err).__name__}: {err}`
- [ ] Geen secrets in logs

### HA Component
- [ ] `import logging` + `_LOGGER = logging.getLogger(__name__)`
- [ ] `diagnostics.py` aanwezig
- [ ] Sensitive data in TO_REDACT
```

---

## STAP 3: Verwijder verouderde secties

Verwijder volledig:
- "CURRENT STATE: GAPS TO FIX"
- "Server-side (Backend) - CRITICAL GAPS" tabel
- "HA Integration - CRITICAL GAPS" tabel  
- "Priority Fix Order"
- "ACTION ITEMS" met TODO lijsten
- "Estimated Total: ~16 hours"

---

## STAP 4: Update version header

```markdown
# SKILL 13 — LOGGING, DIAGNOSTICS & HA INTEGRATION STANDARDS

Logging, Error Handling, and Home Assistant Integration Standards
Version: 2.0 (2026-01-03) - Post-Implementation

---

## STATUS: ✅ IMPLEMENTED

Logging infrastructure volledig geïmplementeerd per TASK 01-06.
Dit document is nu de **verplichte standaard** voor alle nieuwe code.
```

---

## STAP 5: Validatie

```bash
# Check document renderbaar
cat /opt/github/synctacles-api/docs/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md | head -100

# Verify geen "TODO" of "GAPS" meer
grep -i "TODO\|GAPS\|ACTION ITEMS" /opt/github/synctacles-api/docs/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md
# Moet leeg zijn
```

---

## STAP 6: Commit

```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

sudo -u energy-insights-nl git -C /opt/github/synctacles-api add docs/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md

sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: update SKILL_13 to implemented standard

- Remove 'GAPS TO FIX' sections (all implemented)
- Add concrete code patterns from codebase
- Document exact log file locations
- Add new code checklist
- Version 2.0 - post-implementation

SKILL_13 is now the mandatory standard for all new code."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## DONE CRITERIA

- [ ] Alle "TODO/GAPS/ACTION ITEMS" secties verwijderd
- [ ] Concrete code voorbeelden uit huidige codebase
- [ ] Log locaties en configuratie gedocumenteerd
- [ ] "NIEUWE CODE CHECKLIST" toegevoegd
- [ ] Version 2.0 header
- [ ] Git commit gepusht

---

## BELANGRIJK VOOR TOEKOMSTIGE CODE

Na deze update geldt:

1. **Alle nieuwe backend modules** MOETEN `get_logger(__name__)` gebruiken
2. **Alle nieuwe HA componenten** MOETEN `diagnostics.py` hebben
3. **Code review** checkt SKILL_13 compliance
4. **Afwijkingen** vereisen expliciete goedkeuring + documentatie waarom

Dit is geen guideline maar een **standaard**.
