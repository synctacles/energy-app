# CC TASK 06: HA Integration Diagnostics

**Project:** SYNCTACLES  
**Datum:** 2026-01-03  
**Vereist:** TASK 01-05 ✅ (backend logging compleet)  
**Type:** Nieuwe feature

---

## CONTEXT

Backend logging is compleet. Nu: Home Assistant integration diagnostics.

**⚠️ ANDERE REPO:**
```
Backend: /opt/github/synctacles-api/
HA Integration: /opt/github/ha-energy-insights-nl/custom_components/ha_energy_insights_nl/
```

**Doel:**
1. Downloadbare diagnostics JSON (voor support)
2. Gestructureerde logging in HA componenten

---

## STAP 1: Inventariseer HA component

```bash
ls -la /opt/github/ha-energy-insights-nl/custom_components/ha_energy_insights_nl/

# Noteer:
# - Alle .py bestanden
# - Bestaat diagnostics.py al?
# - Huidige logging in bestanden?
```

**Rapporteer bevindingen voordat je verder gaat.**

---

## STAP 2: Maak diagnostics.py

**Bestand:** `/opt/github/ha-energy-insights-nl/custom_components/ha_energy_insights_nl/diagnostics.py`

```python
"""Diagnostics support for Energy Insights NL integration."""
from __future__ import annotations

from typing import Any

from homeassistant.components.diagnostics import async_redact_data
from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant

from .const import DOMAIN

# Keys to redact from diagnostics (privacy/security)
TO_REDACT = {
    "api_key",
    "tennet_api_key", 
    "password",
    "token",
    "secret",
}


async def async_get_config_entry_diagnostics(
    hass: HomeAssistant, entry: ConfigEntry
) -> dict[str, Any]:
    """Return diagnostics for a config entry."""
    
    # Get coordinator data
    data = hass.data.get(DOMAIN, {}).get(entry.entry_id, {})
    coordinator = data.get("coordinator")
    
    diagnostics = {
        "integration_info": {
            "entry_id": entry.entry_id,
            "version": entry.version,
            "domain": DOMAIN,
            "title": entry.title,
        },
        "config": async_redact_data(dict(entry.data), TO_REDACT),
        "options": async_redact_data(dict(entry.options), TO_REDACT),
    }
    
    # Coordinator status
    if coordinator:
        diagnostics["coordinator"] = {
            "last_update_success": coordinator.last_update_success,
            "last_update_time": (
                coordinator.last_update_success_time.isoformat()
                if hasattr(coordinator, "last_update_success_time") 
                and coordinator.last_update_success_time
                else None
            ),
            "update_interval_seconds": (
                coordinator.update_interval.total_seconds()
                if coordinator.update_interval
                else None
            ),
            "data_available": coordinator.data is not None,
        }
        
        # Data summary (no sensitive values)
        if coordinator.data:
            coord_data = coordinator.data
            diagnostics["data_summary"] = {
                "keys": list(coord_data.keys()) if isinstance(coord_data, dict) else "non-dict",
                "quality": coord_data.get("quality") if isinstance(coord_data, dict) else None,
                "source": coord_data.get("source") if isinstance(coord_data, dict) else None,
                "age_seconds": coord_data.get("age_seconds") if isinstance(coord_data, dict) else None,
            }
    else:
        diagnostics["coordinator"] = {"status": "not_initialized"}
    
    # Sensor entity states
    entity_states = {}
    for state in hass.states.async_all():
        if state.entity_id.startswith(f"sensor.{DOMAIN}"):
            entity_states[state.entity_id] = {
                "state": state.state,
                "attributes": dict(state.attributes),
                "last_changed": state.last_changed.isoformat() if state.last_changed else None,
            }
    
    diagnostics["entities"] = entity_states
    
    # Error tracking (if available)
    if coordinator and hasattr(coordinator, "last_exception") and coordinator.last_exception:
        diagnostics["last_error"] = {
            "type": type(coordinator.last_exception).__name__,
            "message": str(coordinator.last_exception),
        }
    
    return diagnostics
```

---

## STAP 3: Logging toevoegen aan bestaande bestanden

**Pattern voor alle HA component bestanden:**

```python
import logging

_LOGGER = logging.getLogger(__name__)
```

**Per bestand specifieke logging:**

### __init__.py
```python
_LOGGER.info(f"Setting up {DOMAIN} integration")
_LOGGER.debug(f"Config: {async_redact_data(entry.data, TO_REDACT)}")

# Bij coordinator setup
_LOGGER.debug("Creating data coordinator")

# Bij errors
_LOGGER.error(f"Failed to setup: {type(err).__name__}: {err}")
```

### config_flow.py
```python
_LOGGER.debug(f"Config flow step: {step_id}")
_LOGGER.debug(f"Validating API connection to {api_url}")
_LOGGER.warning(f"API connection failed: {err}")
_LOGGER.info(f"Integration configured successfully")
```

### sensor.py
```python
_LOGGER.debug(f"Creating sensor: {self._attr_name}")
_LOGGER.debug(f"Sensor {self.entity_id} state: {self.native_value}")

# Bij update errors
_LOGGER.warning(f"Sensor {self.entity_id} unavailable: {err}")
```

### coordinator (indien aanwezig)
```python
_LOGGER.debug(f"Fetching data from {self._api_url}")
_LOGGER.debug(f"Response received: {len(data)} bytes")
_LOGGER.info(f"Data updated successfully, quality={data.get('quality')}")
_LOGGER.warning(f"API returned stale data, age={age_seconds}s")
_LOGGER.error(f"Data fetch failed: {type(err).__name__}: {err}")
```

---

## STAP 4: Redact helper toevoegen aan const.py (indien nodig)

**Bestand:** `const.py` - toevoegen indien niet aanwezig:

```python
# Keys to redact in logs
KEYS_TO_REDACT = {
    "api_key",
    "tennet_api_key",
    "password", 
    "token",
    "secret",
}


def redact_config(config: dict) -> dict:
    """Redact sensitive values from config for logging."""
    return {
        k: "***REDACTED***" if k in KEYS_TO_REDACT else v
        for k, v in config.items()
    }
```

---

## STAP 5: Validatie

```bash
cd /opt/github/ha-energy-insights-nl

# Check syntax
python -m py_compile custom_components/ha_energy_insights_nl/diagnostics.py
python -m py_compile custom_components/ha_energy_insights_nl/__init__.py
python -m py_compile custom_components/ha_energy_insights_nl/sensor.py

# Check imports (basic validation)
python -c "
import sys
sys.path.insert(0, 'custom_components')
from ha_energy_insights_nl.diagnostics import async_get_config_entry_diagnostics
print('diagnostics.py OK')
"
```

**Volledige test in Home Assistant:**
1. Copy naar HA: `/config/custom_components/ha_energy_insights_nl/`
2. Restart HA
3. Settings → Devices & Services → Energy Insights NL → ⋮ → Download diagnostics
4. Check JSON bevat geen API keys

---

## STAP 6: Fix ownership + commit

```bash
# ⚠️ LET OP: ANDERE REPO

sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/ha-energy-insights-nl/

sudo -u energy-insights-nl git -C /opt/github/ha-energy-insights-nl add .
sudo -u energy-insights-nl git -C /opt/github/ha-energy-insights-nl commit -m "feat: add diagnostics and structured logging

- New diagnostics.py for downloadable support info
- Structured logging in all components
- Sensitive data redaction (API keys, tokens)
- Coordinator status and data summary in diagnostics

Part 6 of logging implementation series."

sudo -u energy-insights-nl git -C /opt/github/ha-energy-insights-nl push
```

---

## DONE CRITERIA

- [ ] HA component bestanden geïnventariseerd
- [ ] `diagnostics.py` aangemaakt
- [ ] Alle .py bestanden hebben `_LOGGER = logging.getLogger(__name__)`
- [ ] Logging statements toegevoegd (DEBUG/INFO/WARNING/ERROR)
- [ ] Geen secrets in logs (redacted)
- [ ] Syntax check passed
- [ ] Git commit gepusht naar **ha-energy-insights-nl** repo

---

## DIAGNOSTICS GEBRUIKERSERVARING

Na implementatie kunnen gebruikers:
1. Settings → Devices & Services
2. Energy Insights NL → ⋮ menu
3. "Download diagnostics"
4. JSON bestand delen met support

**JSON bevat:**
- Integration config (API keys redacted)
- Coordinator status (last update, interval)
- Data summary (quality, source, age)
- Entity states (all sensors)
- Last error (if any)

---

## LOGGING NIVEAUS VOOR GEBRUIKERS

Gebruikers configureren in `configuration.yaml`:

```yaml
logger:
  logs:
    custom_components.ha_energy_insights_nl: warning  # Default
    # Of voor debugging:
    custom_components.ha_energy_insights_nl: debug
```

---

## RAPPORTAGE

Na voltooiing, rapporteer:
1. Lijst van aangepaste bestanden
2. Bestaande logging die vervangen is
3. Eventuele coordinator/data structuur bevindingen

---

## SERIES COMPLEET NA DEZE TAAK

✅ TASK 01: Backend Logging Core  
✅ TASK 02: Logging Collectors  
✅ TASK 03: Logging Importers  
✅ TASK 04: Logging Normalizers  
✅ TASK 05: API Request Logging Middleware  
✅ TASK 06: HA Integration Diagnostics  

**Totaal:** Volledig observability framework voor SYNCTACLES
