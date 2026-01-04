# CC TASK 04: Logging in Normalizers

**Project:** SYNCTACLES  
**Datum:** 2026-01-03  
**Vereist:** TASK 01 ✅, TASK 02 ✅, TASK 03 ✅  
**Type:** Feature uitbreiding

---

## CONTEXT

Logging infrastructuur staat. Collectors en importers hebben logging.  
Nu: normalizers - de kritieke laag voor data quality en fallback beslissingen.

**Bewezen pattern:**
```python
from synctacles_db.core.logging import get_logger
_LOGGER = get_logger(__name__)
```

---

## DOEL

Normalizers logging is extra belangrijk voor:
- **Fallback activaties** - wanneer en waarom
- **Quality decisions** - waarom data STALE/FRESH/FALLBACK is
- **Data age tracking** - zichtbaarheid in data versheid
- **Transformation issues** - wat ging mis bij normalisatie

---

## STAP 1: Inventariseer normalizers

```bash
ls -la /opt/github/synctacles-api/synctacles_db/normalizers/

# Noteer:
# - Alle .py bestanden (behalve __init__.py)
# - Classes/functies
# - Fallback logica aanwezig?
# - Quality scoring aanwezig?
```

**Rapporteer bevindingen voordat je verder gaat.**

---

## STAP 2: Logging pattern toevoegen

**Per normalizer bestand, voeg toe aan begin:**

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)
```

---

## STAP 3: Log statements toevoegen

**Pattern voor normalize functies:**

```python
def normalize(self):
    """Normalize raw data with quality metadata."""
    _LOGGER.info(f"Normalizer {self.name} starting")
    start_time = time.time()
    
    try:
        # Fetch raw data
        raw_data = self._get_raw_data()
        _LOGGER.debug(f"Raw data: {len(raw_data)} records")
        
        # Check data age
        age_minutes = self._calculate_age(raw_data)
        _LOGGER.debug(f"Data age: {age_minutes:.1f} minutes")
        
        # Quality decision logging
        if age_minutes > self.STALE_THRESHOLD:
            _LOGGER.warning(f"Data age {age_minutes:.0f}min exceeds threshold {self.STALE_THRESHOLD}min, quality=STALE")
        
        # Fallback logging
        if self._needs_fallback(raw_data):
            _LOGGER.info(f"Fallback activated: using {self.fallback_source} instead of {self.primary_source}")
            raw_data = self._get_fallback_data()
            _LOGGER.debug(f"Fallback data: {len(raw_data)} records")
        
        # Transform
        normalized = self._transform(raw_data)
        _LOGGER.debug(f"Normalized: {len(normalized)} records")
        
        # Quality score
        quality = self._calculate_quality(normalized)
        _LOGGER.debug(f"Quality score: {quality}")
        
        elapsed = time.time() - start_time
        _LOGGER.info(f"Normalizer {self.name} completed: {len(normalized)} records, quality={quality} in {elapsed:.2f}s")
        
        return normalized
        
    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"Normalizer {self.name} failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise
```

**Specifieke log momenten:**

| Situatie | Level | Voorbeeld |
|----------|-------|-----------|
| Fallback geactiveerd | INFO | "Fallback activated: using Energy-Charts" |
| Data te oud | WARNING | "Data age 95min exceeds threshold 60min" |
| Quality berekening | DEBUG | "Quality score: 0.85" |
| Transformation details | DEBUG | "Transformed 150 raw → 12 normalized" |
| Geen data beschikbaar | ERROR | "No raw data available, cannot normalize" |

---

## STAP 4: Validatie

```bash
export LOG_LEVEL=debug

cd /opt/github/synctacles-api
source /opt/energy-insights-nl/venv/bin/activate

# Trigger normalizer (pas aan naar gevonden normalizer)
python -m synctacles_db.normalizers.<naam>

# Check output - let op fallback en quality logs
tail -50 /var/log/synctacles/synctacles.log

export LOG_LEVEL=warning
```

**Verwachte output (voorbeeld):**
```
2026-01-03 14:00:00 [INFO   ] synctacles_db.normalizers.generation: Normalizer generation starting
2026-01-03 14:00:00 [DEBUG  ] synctacles_db.normalizers.generation: Raw data: 150 records
2026-01-03 14:00:00 [DEBUG  ] synctacles_db.normalizers.generation: Data age: 35.2 minutes
2026-01-03 14:00:00 [DEBUG  ] synctacles_db.normalizers.generation: Quality score: 0.92
2026-01-03 14:00:00 [INFO   ] synctacles_db.normalizers.generation: Normalizer completed: 12 records, quality=0.92 in 0.15s
```

---

## STAP 5: Fix ownership + commit

```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "feat: add logging to normalizers

- All normalizer modules now use centralized logging
- INFO: start/end, fallback activations
- DEBUG: data age, quality scores, transformation details
- WARNING: stale data threshold exceeded

Part 4 of logging implementation series."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## DONE CRITERIA

- [ ] Alle normalizer .py bestanden geïnventariseerd
- [ ] Elk bestand importeert `get_logger`
- [ ] Elk bestand heeft `_LOGGER = get_logger(__name__)`
- [ ] Fallback beslissingen worden gelogd (INFO)
- [ ] Quality/age issues worden gelogd (WARNING/DEBUG)
- [ ] Test: normalizer run produceert log entries
- [ ] Git commit gepusht

---

## RAPPORTAGE

Na voltooiing, rapporteer:
1. Lijst van aangepaste bestanden
2. Welke fallback logica gevonden
3. Quality scoring aanwezig? Hoe gelogd?

---

## VOLGENDE TAAK

Na voltooiing → TASK 05 (API middleware)
