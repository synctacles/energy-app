# CC LOGGING IMPLEMENTATIE - OVERZICHT

**Project:** SYNCTACLES
**Datum:** 2026-01-03
**Totaal taken:** 6

---

## TAKENREEKS

| Task | Bestand | Doel | Status |
|------|---------|------|--------|
| 01 | `CC_TASK_01_BACKEND_LOGGING_CORE.md` | Logging module + rotatie + config | TODO |
| 02 | `CC_TASK_02_LOGGING_COLLECTORS.md` | Logging in alle collectors | WACHT |
| 03 | `CC_TASK_03_LOGGING_IMPORTERS.md` | Logging in alle importers | WACHT |
| 04 | `CC_TASK_04_LOGGING_NORMALIZERS.md` | Logging in alle normalizers | WACHT |
| 05 | `CC_TASK_05_LOGGING_API_MIDDLEWARE.md` | Request/response logging | WACHT |
| 06 | `CC_TASK_06_HA_DIAGNOSTICS.md` | HA Integration diagnostics | WACHT |

---

## VOLGORDE

**Strikt sequentieel uitvoeren.** Elke taak bouwt voort op de vorige.

```
TASK 01 ──► TASK 02 ──► TASK 03 ──► TASK 04 ──► TASK 05 ──► TASK 06
  │
  └── Moet EERST voltooid zijn (logging module is dependency)
```

---

## REFERENTIES

- **Standaarden:** `SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md`
- **Service account:** `energy-insights-nl` (zie SKILL_11)
- **Repo:** `/opt/github/synctacles-api`
- **Log locatie:** `/var/log/synctacles/synctacles.log`

---

## LOG NIVEAUS

| Niveau | Wanneer gebruiken |
|--------|-------------------|
| `off` | Logging volledig uit |
| `error` | Alleen fatale fouten |
| `warning` | **STANDAARD PRODUCTIE** - onverwacht maar niet fataal |
| `info` | Normale operatie milestones |
| `debug` | Troubleshooting - hoge disk I/O! |

---

## WAT TE LOGGEN PER COMPONENT

### Collectors (TASK 02)
```
INFO:  Start + einde met record count
DEBUG: Request URL, response status, response size, timing
ERROR: Failures met context (URL, status, error type)
```

### Importers (TASK 03)
```
INFO:  Start + einde met inserted/skipped counts
DEBUG: Parse details, raw data structure
WARNING: Skipped records (niet per record, aggregated)
ERROR: Parse failures met context
```

### Normalizers (TASK 04)
```
INFO:  Fallback activaties, quality changes
DEBUG: Raw → normalized transformation details
WARNING: Data age threshold exceeded
ERROR: Transformation failures
```

### API (TASK 05)
```
DEBUG: Inkomende requests (kan hoog volume zijn)
WARNING: Slow responses (>1s), auth failures
ERROR: Endpoint failures
```

---

## GIT WORKFLOW

**Na ELKE taak:**

```bash
# Fix ownership
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

# Commit
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "feat: [beschrijving]"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## VALIDATIE PER TAAK

Elke taak bevat eigen validatiestappen. Minimaal:

1. Python import zonder errors
2. Logging produceert output in logbestand
3. Bestaande functionaliteit werkt nog
4. API health check OK

---

## START

**Begin met:** `CC_TASK_01_BACKEND_LOGGING_CORE.md`
