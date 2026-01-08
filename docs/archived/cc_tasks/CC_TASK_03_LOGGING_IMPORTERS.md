# CC TASK 03: Logging in Importers

**Project:** SYNCTACLES  
**Datum:** 2026-01-03  
**Vereist:** TASK 01 ✅, TASK 02 ✅  
**Type:** Feature uitbreiding

---

## CONTEXT

Logging infrastructuur staat (TASK 01). Collectors hebben logging (TASK 02).  
Nu: importers voorzien van logging.

**Bewezen pattern uit TASK 02:**
```python
from synctacles_db.core.logging import get_logger
_LOGGER = get_logger(__name__)
```

---

## DOEL

Alle importers voorzien van gestructureerde logging:
- Start/einde met timing en record counts
- Parse details (DEBUG)
- Skipped/invalid records (WARNING, geaggregeerd)
- Errors met volledige context

---

## STAP 1: Inventariseer importers

```bash
ls -la /opt/github/synctacles-api/synctacles_db/importers/

# Noteer:
# - Alle .py bestanden (behalve __init__.py)
# - Classes/functies
# - Huidige logging staat
```

**Rapporteer bevindingen voordat je verder gaat.**

---

## STAP 2: Logging pattern toevoegen

**Per importer bestand, voeg toe aan begin:**

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)
```

---

## STAP 3: Log statements toevoegen

**Pattern voor import functies:**

```python
def import_data(self, file_path):
    """Import data from raw file."""
    _LOGGER.info(f"Importer {self.name} starting: {file_path}")
    start_time = time.time()
    
    inserted = 0
    skipped = 0
    errors = []
    
    try:
        # Parse code...
        _LOGGER.debug(f"Parsing file: {file_path}")
        records = self._parse(file_path)
        _LOGGER.debug(f"Found {len(records)} records to import")
        
        for record in records:
            try:
                self._insert(record)
                inserted += 1
            except DuplicateError:
                skipped += 1
            except Exception as err:
                errors.append(str(err))
        
        elapsed = time.time() - start_time
        
        # Aggregated warnings (niet per record)
        if skipped > 0:
            _LOGGER.debug(f"Skipped {skipped} duplicate records")
        if errors:
            _LOGGER.warning(f"Failed to insert {len(errors)} records")
            _LOGGER.debug(f"Insert errors: {errors[:5]}")  # Max 5
        
        _LOGGER.info(f"Importer {self.name} completed: {inserted} inserted, {skipped} skipped in {elapsed:.2f}s")
        
        return inserted
        
    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"Importer {self.name} failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise
```

**Belangrijke regels:**
- WARNING alleen voor geaggregeerde issues (niet per record spam)
- DEBUG voor individuele record details
- INFO voor totalen

---

## STAP 4: Validatie

```bash
export LOG_LEVEL=debug

cd /opt/github/synctacles-api
source /opt/energy-insights-nl/venv/bin/activate

# Trigger importer (pas aan naar gevonden importer)
python -m synctacles_db.importers.<naam>

# Check output
tail -50 /var/log/synctacles/synctacles.log

export LOG_LEVEL=warning
```

---

## STAP 5: Fix ownership + commit

```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "feat: add logging to importers

- All importer modules now use centralized logging
- INFO: start/end with inserted/skipped counts and timing
- DEBUG: parse details and individual errors
- WARNING: aggregated insert failures

Part 3 of logging implementation series."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## DONE CRITERIA

- [ ] Alle importer .py bestanden geïnventariseerd
- [ ] Elk bestand importeert `get_logger`
- [ ] Elk bestand heeft `_LOGGER = get_logger(__name__)`
- [ ] Import functies loggen start/einde/errors
- [ ] Warnings zijn geaggregeerd (geen spam)
- [ ] Test: importer run produceert log entries
- [ ] Git commit gepusht

---

## RAPPORTAGE

Na voltooiing, rapporteer:
1. Lijst van aangepaste bestanden
2. Aantal importers gevonden vs aangepast
3. Eventuele afwijkingen

---

## VOLGENDE TAAK

Na voltooiing → TASK 04 (normalizers)
