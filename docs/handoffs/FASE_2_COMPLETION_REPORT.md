# FASE 2 COMPLETION REPORT

Energy Action Focus - Soft Delete
Completed: 2026-01-11

---

## SUMMARY

Fase 2 implements the "soft delete" approach for SYNCTACLES Energy Action focus:
- Grid endpoints return HTTP 410 Gone with migration guidance
- HA component reduced from 12 to 6 sensors
- Importers/normalizers skip A65 (load) and A75 (generation)
- All changes are reversible (code not removed, just disabled)

---

## API CHANGES

### New File: `synctacles_db/api/endpoints/deprecated.py`

Centralized 410 Gone handlers for discontinued endpoints.

```python
DISCONTINUED_MESSAGE = {
    "error": "Gone",
    "message": "This endpoint has been discontinued. SYNCTACLES now focuses exclusively on Energy Action.",
    "documentation": "https://github.com/DATADIO/synctacles-api#energy-action-focus",
    "migration": {
        "energy_action": "/api/v1/energy-action",
        "prices": "/api/v1/prices/today",
        "rationale": "Energy Action provides actionable insights without raw grid data."
    }
}
```

### Endpoint Status

| Endpoint | Status | Response |
|----------|--------|----------|
| `/api/v1/energy-action` | ✅ Active | JSON with action, quality, confidence |
| `/api/v1/prices/today` | ✅ Active | Price data for today |
| `/api/v1/prices/tomorrow` | ✅ Active | Price data for tomorrow |
| `/api/v1/now` | ✅ Active | Unified current state |
| `/api/v1/generation-mix` | ❌ 410 Gone | Migration to energy-action |
| `/api/v1/load` | ❌ 410 Gone | Migration to energy-action |
| `/api/v1/signals` | ❌ 410 Gone | Migration to energy-action |
| `/api/v1/balance` | ❌ 410 Gone | TenneT BYO-key guidance |

### Modified Files

#### `synctacles_db/api/main.py`

```python
# Changed imports
from synctacles_db.api.endpoints import balance, now, prices, auth, energy_action
from synctacles_db.api.endpoints.deprecated import router as deprecated_router, signals_router as deprecated_signals_router

# Changed router registrations
# V1 endpoints - Active
app.include_router(balance.router, prefix="/api/v1", tags=["balance"])
app.include_router(now.router, prefix="/api/v1", tags=["Unified"])
app.include_router(prices.router, prefix="/api/v1", tags=["prices"])

# V1 endpoints - Deprecated (410 Gone)
app.include_router(deprecated_router, prefix="/api/v1", tags=["deprecated"])
app.include_router(deprecated_signals_router, prefix="/api", tags=["deprecated"])
```

#### `synctacles_db/api/endpoints/__init__.py`

```python
# Active endpoints
from . import balance
from . import now
from . import prices
from . import energy_action

# Deprecated endpoints (410 Gone) - Fase 2: Soft Delete (2026-01-11)
# from . import generation_mix  # DISCONTINUED
# from . import load            # DISCONTINUED
# from . import signals         # DISCONTINUED
```

---

## HA COMPONENT CHANGES

Repository: `ha-energy-insights-nl`

### Sensor Changes (12 → 6 entities)

#### Sensors KEPT (6)

| Sensor | Entity ID | Description |
|--------|-----------|-------------|
| `PriceCurrentSensor` | `sensor.electricity_price` | Current wholesale price |
| `CheapestHourSensor` | `sensor.cheapest_hour` | Time of cheapest hour today |
| `ExpensiveHourSensor` | `sensor.expensive_hour` | Time of most expensive hour |
| `EnergyActionSensor` | `sensor.energy_action` | GO/WAIT/AVOID recommendation |
| `PricesTodaySensor` | `sensor.prices_today` | Hourly prices today (BYO Enever) |
| `PricesTomorrowSensor` | `sensor.prices_tomorrow` | Hourly prices tomorrow (BYO Enever) |

#### Sensors DISCONTINUED (6)

| Sensor | Entity ID | Reason |
|--------|-----------|--------|
| `GenerationTotalSensor` | `sensor.generation_total` | Grid data discontinued |
| `LoadActualSensor` | `sensor.load_actual` | Grid data discontinued |
| `PriceStatusSensor` | `sensor.price_status` | Redundant with EnergyAction |
| `PriceLevelSensor` | `sensor.price_level` | Redundant with EnergyAction |
| `BalanceDeltaSensor` | `sensor.balance_delta` | TenneT BYO discontinued |
| `GridStressSensor` | `sensor.grid_stress` | TenneT BYO discontinued |

### Modified Files

#### `custom_components/ha_energy_insights_nl/sensor.py`

- `async_setup_entry()` now creates only 6 core sensors
- Discontinued sensors are commented out with explanation
- Warning logged if TenneT BYO-key is still configured

#### `custom_components/ha_energy_insights_nl/__init__.py`

- `ServerDataCoordinator` only fetches `/prices` endpoint
- No longer fetches `/generation-mix` or `/load` (return 410 Gone)
- Reduced API calls from 3 to 1 per update cycle

---

## SCRIPT CHANGES

### `scripts/run_importers.sh`

```bash
# Phase 2: Energy Action Focus (2026-01-11)
# A65 (load) and A75 (generation) importers are SKIPPED
# Only A44 (prices) is needed for Energy Action

# SKIPPED: "${PYTHON}" -m synctacles_db.importers.import_entso_e_a75  # Generation
# SKIPPED: "${PYTHON}" -m synctacles_db.importers.import_entso_e_a65  # Load
"${PYTHON}" -m synctacles_db.importers.import_entso_e_a44  # Prices - ACTIVE
```

### `scripts/run_normalizers.sh`

```bash
# Phase 2: Energy Action Focus (2026-01-11)
# Only A44 (prices) normalizer is needed for Energy Action
# A65 (load) and A75 (generation) normalizers are SKIPPED

"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44  # ACTIVE
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65  # SKIPPED
# "${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75  # SKIPPED
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices  # ACTIVE
```

---

## GIT COMMITS

### synctacles-api

| Commit | Message |
|--------|---------|
| `be799e5` | feat: Phase 2 - Energy Action Focus (soft delete) |

### ha-energy-insights-nl

| Commit | Message |
|--------|---------|
| `40b9fff` | feat: Phase 2 - Energy Action Focus (reduce to 6 sensors) |

---

## GITHUB ISSUES CLOSED

| Issue | Title | Status |
|-------|-------|--------|
| #63 | Disable grid/generation endpoints (410) | ✅ Closed |
| #64 | Reduce HA component to 6 entities | ✅ Closed |
| #65 | Skip TenneT and A65/A75 processing | ✅ Closed |

---

## DOCUMENTATION UPDATED (Fase 1)

As part of Fase 2, the following Fase 1 documentation was also completed:

| Document | Changes |
|----------|---------|
| `FASE_1_COMPLETION_REPORT.md` | Created - full Fase 1 report |
| `ARCHITECTURE.md` | Added price_cache table, 5-tier fallback chain, energy-action endpoint |
| `SKILL_02_ARCHITECTURE.md` | Added fallback chain diagram, price_cache schema |
| `SKILL_06_DATA_SOURCES.md` | Added price fallback chain, Energy-Charts API |
| `SKILL_13_LOGGING.md` | Added PART D: Quality Indicator Logging |

---

## ROLLBACK PROCEDURE

If Fase 2 needs to be reverted:

### API (synctacles-api)

1. Revert `main.py` imports to include `generation_mix`, `load`, `signals`
2. Remove `deprecated.py`
3. Restore `__init__.py` imports
4. Uncomment importers in `run_importers.sh`
5. Uncomment normalizers in `run_normalizers.sh`

### HA Component (ha-energy-insights-nl)

1. Restore `ServerDataCoordinator` to fetch all 3 endpoints
2. Restore all 12 sensors in `async_setup_entry()`

---

## REMAINING WORK (Fase 3+)

Fase 2 is "soft delete" - code is disabled but not removed. The following phases will:

- **Fase 3:** Hard delete (remove code entirely)
  - #66: Remove TenneT integration code
  - #67: Remove A65/A75 (load/generation) code
  - #68: Remove grid stress calculation
  - #69: Clean up unused dependencies

- **Fase 4:** Documentation cleanup
  - #70-#73: Update SKILL files, API docs, archive issues

---

## NOTES

1. **Graceful Degradation:** Existing HA users won't get errors - discontinued sensors simply won't be created. Automations using these sensors will need to be updated.

2. **TenneT BYO-Key:** Users with TenneT API keys configured will see a warning in logs. The TenneT coordinator still runs but creates no sensors.

3. **API Backwards Compatibility:** Clients calling discontinued endpoints get a helpful 410 response with migration guidance, not a confusing 404.

4. **ENTSO-E API Savings:** By skipping A65/A75 imports, we reduce ENTSO-E API calls by ~66%, lowering risk of rate limiting.

---

*Report generated: 2026-01-11*
*Author: Claude Opus 4.5*
