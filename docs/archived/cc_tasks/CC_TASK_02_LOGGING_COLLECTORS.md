# CC TASK 02: Logging in Collectors

**Project:** SYNCTACLES  
**Datum:** 2026-01-03  
**Vereist:** TASK 01 voltooid ✅  
**Type:** Feature uitbreiding

---

## CONTEXT

TASK 01 heeft de logging infrastructuur opgezet:
- `synctacles_db/core/logging.py` met `get_logger()`
- Log bestand: `/var/log/synctacles/synctacles.log`
- Config via `LOG_LEVEL` env var

Nu: logging toevoegen aan alle collector modules.

---

## DOEL

Alle collectors voorzien van gestructureerde logging:
- Start/einde met timing en record counts
- API request/response details (DEBUG)
- Errors met volledige context

---

## STAP 1: Inventariseer collectors

**Voer uit en documenteer welke bestanden gevonden worden:**

```bash
ls -la /opt/github/synctacles-api/synctacles_db/collectors/

# Noteer:
# - Alle .py bestanden (behalve __init__.py)
# - Welke classes/functies bevatten ze
# - Huidige staat (wel/geen logging)
```

**Rapporteer bevindingen voordat je verder gaat.**

---

## STAP 2: Logging pattern toevoegen

**Per collector bestand, voeg toe aan begin:**

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)
```

---

## STAP 3: Log statements toevoegen

**Pattern voor elke collect functie/method:**

```python
def collect(self):
    """Collect data from source."""
    _LOGGER.info(f"Collector {self.name} starting")
    start_time = time.time()
    
    try:
        # Bestaande request code...
        _LOGGER.debug(f"Request URL: {url}")
        
        response = self._fetch(url)
        _LOGGER.debug(f"Response status: {response.status_code}, size: {len(response.content)} bytes")
        
        # Bestaande parse code...
        records = self._parse(response)
        
        elapsed = time.time() - start_time
        _LOGGER.info(f"Collector {self.name} completed: {len(records)} records in {elapsed:.2f}s")
        
        return records
        
    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"Collector {self.name} failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise
```

**Belangrijke regels:**
- Geen secrets loggen (sanitize URLs met tokens)
- DEBUG voor request/response details
- INFO voor start/einde
- ERROR voor failures met context

---

## STAP 4: Sanitize helper (indien nodig)

Als URLs tokens bevatten, voeg toe aan logging.py of utils:

```python
import re

def sanitize_url(url: str) -> str:
    """Remove tokens/keys from URL for safe logging."""
    return re.sub(r'(token|key|apikey|secret)=[^&]+', r'\1=***', url, flags=re.IGNORECASE)
```

---

## STAP 5: Validatie

```bash
# Set debug level tijdelijk
export LOG_LEVEL=debug

# Trigger een collector handmatig (pas aan naar gevonden collector)
cd /opt/github/synctacles-api
source /opt/energy-insights-nl/venv/bin/activate
python -m synctacles_db.collectors.<naam>

# Check log output
tail -50 /var/log/synctacles/synctacles.log

# Zet terug naar warning
export LOG_LEVEL=warning
```

---

## STAP 6: Fix ownership + commit

```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "feat: add logging to collectors

- All collector modules now use centralized logging
- INFO: start/end with record counts and timing
- DEBUG: request/response details
- ERROR: failures with full context

Part 2 of logging implementation series."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## DONE CRITERIA

- [ ] Alle collector .py bestanden geïnventariseerd
- [ ] Elk bestand importeert `get_logger`
- [ ] Elk bestand heeft `_LOGGER = get_logger(__name__)`
- [ ] Collect functies loggen start/einde/errors
- [ ] Geen secrets in logs (URLs gesanitized)
- [ ] Test: collector run produceert log entries
- [ ] Git commit gepusht

---

## RAPPORTAGE

Na voltooiing, rapporteer:
1. Lijst van aangepaste bestanden
2. Eventuele afwijkingen van het pattern
3. Problemen/beslissingen onderweg

---

## VOLGENDE TAAK

Na voltooiing ontvangt Leo feedback → TASK 03 (importers) wordt aangemaakt.
