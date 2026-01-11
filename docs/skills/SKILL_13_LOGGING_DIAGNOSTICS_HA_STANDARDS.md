# SKILL 13 — LOGGING, DIAGNOSTICS & HA INTEGRATION STANDARDS

Logging, Error Handling, and Home Assistant Integration Standards
Version: 2.0 (2026-01-03) - Post-Implementation

---

## STATUS: ✅ IMPLEMENTED

Logging infrastructure fully implemented per TASK 01-06 of SYNCTACLES project.
This document is now the **mandatory standard** for all new code.

---

## PURPOSE

Define standards for:
1. **Server-side logging** - Backend API, collectors, importers, normalizers
2. **Client-side logging** - Home Assistant integration
3. **Diagnostics** - Troubleshooting without direct access
4. **Error handling** - Graceful degradation, clear error states
5. **HA integration patterns** - Config flow, coordinators, sensors

**Implementation coverage:**
- ✅ Collectors (4 modules): Centralized logging with timing metrics
- ✅ Importers (4 modules): Structured logging with record tracking
- ✅ Normalizers (4 modules): Quality tracking and fallback logging
- ✅ API middleware: Request/response logging with auth context
- ✅ HA integration: Coordinator logging + diagnostics module

---

## IMPLEMENTED INFRASTRUCTURE

### Log Files

| File | Purpose | Rotation |
|------|---------|----------|
| `/var/log/synctacles/synctacles.log` | Backend pipeline (collectors, importers, normalizers) | 10MB × 4 files (40MB max) |
| `/var/log/synctacles/api.log` | API requests/responses (via middleware) | 10MB × 4 files (40MB max) |

### Configuration

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

---

## PART A: BACKEND LOGGING STANDARD

### Import Statement (MANDATORY for all modules)

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)
```

All backend modules **MUST** use centralized logging via `synctacles_db.core.logging`.
Never use `logging.basicConfig()` or `logging.getLogger()` directly.

### Core Implementation

**File:** `synctacles_db/core/logging.py`

```python
def get_logger(name: str) -> logging.Logger:
    """Get logger with centralized configuration."""
    logger = logging.getLogger(name)

    # RotatingFileHandler: 10MB per file × 4 = 40MB max
    handler = RotatingFileHandler(
        LOG_PATH_FILE,
        maxBytes=10 * 1024 * 1024,
        backupCount=3
    )

    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    handler.setFormatter(formatter)
    logger.addHandler(handler)
    logger.setLevel(LOG_LEVEL)

    return logger
```

### Standard Function Pattern

```python
def process_data(self):
    """Process data with logging."""
    _LOGGER.info(f"{self.name} starting")
    start_time = time.time()

    try:
        # ... processing logic ...
        _LOGGER.debug(f"Processed {count} records")

        elapsed = time.time() - start_time
        _LOGGER.info(f"{self.name} completed: {count} records in {elapsed:.2f}s")

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"{self.name} failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise
```

### Logging Levels Usage

| Level | When | Example |
|-------|------|---------|
| **DEBUG** | Request/response details, record counts, parsing steps | `_LOGGER.debug(f"Parsing {file}: {size} bytes")` |
| **INFO** | Start/end of operations, record counts, quality status | `_LOGGER.info(f"Normalized {n} records in {t:.2f}s")` |
| **WARNING** | Stale data, slow responses, fallback activation, aggregated failures | `_LOGGER.warning(f"Data age {age}min exceeds threshold")` |
| **ERROR** | Operation failures with exception context | `_LOGGER.error(f"Failed: {type(err).__name__}: {err}")` |

**Golden Rule: No Spam**
- WARNING/ERROR: Never log per-record (aggregate into summary)
- DEBUG: May log per-record, but only at debug log level
- INFO: Start/end operations only (not per-record)

### Real Examples from Codebase

#### Collectors Pattern
**File:** `synctacles_db/collectors/entso_e_a75_generation.py`

```python
from synctacles_db.core.logging import get_logger
_LOGGER = get_logger(__name__)

async def fetch_generation(self, country: str = "NL") -> dict:
    """Fetch generation data."""
    _LOGGER.info(f"A75 collector starting: country={country}")
    start_time = time.time()

    try:
        _LOGGER.debug(f"Request URL: {url}, timeout: 30s")
        response = await self._session.get(url, headers=headers)

        if response.status == 200:
            data = await response.json()
            count = len(data.get("data", []))
            _LOGGER.debug(f"A75 response received: {count} records")

            elapsed = time.time() - start_time
            _LOGGER.info(f"A75 collector completed: {count} records in {elapsed:.2f}s")
            return data
        else:
            _LOGGER.warning(f"A75 API returned {response.status}")
            return None

    except Exception as e:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75 collector failed after {elapsed:.2f}s: {type(e).__name__}: {e}")
        raise
```

#### Importers Pattern
**File:** `synctacles_db/importers/import_entso_e_a75.py`

```python
def import_a75_file(filepath: Path, session) -> tuple[int, int]:
    """Import single A75 XML file."""
    _LOGGER.info(f"A75 XML importer starting: {filepath.name}")
    start_time = time.time()

    try:
        _LOGGER.debug(f"Parsing XML file: {filepath}")
        tree = etree.parse(str(filepath))
        root = tree.getroot()

        # ... parsing logic ...

        elapsed = time.time() - start_time
        _LOGGER.info(f"A75 XML importer completed: {len(records)} records in {elapsed:.2f}s")
        return len(records), 0

    except Exception as e:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75 XML import failed after {elapsed:.2f}s: {type(e).__name__}: {e}")
        return 0, 1
```

#### Normalizers Pattern
**File:** `synctacles_db/normalizers/normalize_entso_e_a75.py`

```python
def normalize_a75_generation():
    """Pivot raw_entso_e_a75 → norm_entso_e_a75."""
    _LOGGER.info("A75 normalizer starting")
    start_time = time.time()

    try:
        latest_raw = session.query(func.max(RawEntsoeA75.timestamp)).scalar()
        quality_status = calculate_quality_status(latest_raw)

        _LOGGER.debug(f"Latest raw timestamp: {latest_raw}")
        _LOGGER.debug(f"Quality status: {quality_status}")

        # ... pivot logic ...

        elapsed = time.time() - start_time
        _LOGGER.info(f"A75 normalizer completed: {len(records)} records normalized in {elapsed:.2f}s")

    except Exception as e:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A75 normalizer failed after {elapsed:.2f}s: {type(e).__name__}: {e}")
        raise
```

### Quality Status Calculation

All normalizers log quality assessment:

```python
def calculate_quality_status(latest_timestamp: datetime) -> str:
    """Calculate data quality based on age."""
    if latest_timestamp is None:
        return 'NO_DATA'

    now = datetime.now(timezone.utc)
    age_minutes = (now - latest_timestamp).total_seconds() / 60

    if age_minutes < 15:
        return 'OK'
    elif age_minutes < 1440:  # 24 hours
        return 'STALE'
    else:
        return 'CACHED'
```

---

## PART B: API REQUEST LOGGING (FastAPI Middleware)

**File:** `synctacles_db/api/middleware.py`

### HTTP Logging Middleware

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)

async def http_logging_middleware(request: Request, call_next):
    """Log HTTP requests and responses with timing."""
    start_time = time.time()

    # Log request at DEBUG level
    _LOGGER.debug(
        f"HTTP request: {method} {path}",
        extra={
            "method": request.method,
            "path": request.url.path,
            "query": str(request.url.query),
            "client": request.client.host if request.client else "unknown",
        }
    )

    response = await call_next(request)
    elapsed = time.time() - start_time

    # Log response with appropriate level
    if 200 <= response.status_code < 400:
        # Success - INFO level
        _LOGGER.info(
            f"HTTP response: {request.method} {request.url.path} {response.status_code}",
            extra={"duration_ms": elapsed * 1000}
        )
    else:
        # Error - WARNING level
        _LOGGER.warning(
            f"HTTP error: {request.method} {request.url.path} {response.status_code}",
            extra={"duration_ms": elapsed * 1000}
        )

    response.headers["X-Response-Time"] = f"{elapsed:.3f}"
    return response
```

### Auth Middleware Logging

```python
async def auth_middleware(request: Request, call_next):
    """Validate API key with logging."""
    path = request.url.path

    # Exempt paths
    if path in EXEMPT_PATHS:
        _LOGGER.debug(f"Auth exempt path: {path}")
        return await call_next(request)

    # Validate key
    api_key = request.headers.get("X-API-Key")
    if not api_key:
        _LOGGER.warning(f"Auth failed: missing X-API-Key header for {path}")
        return JSONResponse(status_code=401, content={"detail": "API key required"})

    try:
        user = auth_service.validate_api_key(db, api_key)
        if not user:
            _LOGGER.warning(f"Auth failed: invalid API key for {path}")
            return JSONResponse(status_code=401, content={"detail": "Invalid API key"})

        _LOGGER.debug(f"Auth success: user {user.id} for {path}")
        request.state.user = user

    except Exception as e:
        _LOGGER.error(f"Auth error: {type(e).__name__}: {e} for {path}")
        return JSONResponse(status_code=401, content={"detail": "Auth failed"})

    return await call_next(request)
```

### Rate Limit Middleware Logging

```python
async def rate_limit_middleware(request: Request, call_next):
    """Rate limit with usage logging."""
    user = getattr(request.state, "user", None)

    # Check limit
    usage_count = db.query(APIUsage).filter(...).count()
    if usage_count >= user.rate_limit_daily:
        _LOGGER.warning(
            f"Rate limit exceeded: user {user.id}, daily limit {user.rate_limit_daily}",
            extra={"limit": user.rate_limit_daily, "usage": usage_count}
        )
        return JSONResponse(status_code=429, content={"detail": "Rate limit exceeded"})

    _LOGGER.debug(
        f"Rate limit check: user {user.id}, usage {usage_count}/{user.rate_limit_daily}",
        extra={"remaining": user.rate_limit_daily - usage_count}
    )

    response = await call_next(request)

    # Log usage
    _LOGGER.debug(
        f"API usage logged: user {user.id}, path {request.url.path}, status {response.status_code}"
    )

    return response
```

---

## PART C: HOME ASSISTANT INTEGRATION

### Import Statement

```python
import logging

_LOGGER = logging.getLogger(__name__)
```

HA components use standard Python logging (not centralized, as HA manages logs).

### Coordinator Logging

**File:** `custom_components/ha_energy_insights_nl/__init__.py`

```python
async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Set up Energy Insights NL from a config entry."""
    _LOGGER.info("Setting up Energy Insights NL integration (entry_id=%s)", entry.entry_id)

    api_url = entry.data.get(CONF_API_URL, "").rstrip("/")
    has_api_key = bool(entry.data.get(CONF_API_KEY))
    has_tennet_key = bool(entry.data.get(CONF_TENNET_API_KEY))

    _LOGGER.debug(
        "Configuration: api_url=%s, has_api_key=%s, has_tennet_key=%s",
        api_url[:20] if api_url else "not set",
        has_api_key,
        has_tennet_key
    )

    # Server coordinator
    server_coordinator = ServerDataCoordinator(hass, session, api_url, api_key)
    await server_coordinator.async_config_entry_first_refresh()

    _LOGGER.info(
        "Server coordinator initialized: generation=%s, load=%s, prices=%s",
        server_coordinator.data.get("generation") is not None if server_coordinator.data else False,
        server_coordinator.data.get("load") is not None if server_coordinator.data else False,
        server_coordinator.data.get("prices") is not None if server_coordinator.data else False
    )
```

### ServerDataCoordinator Logging

```python
async def _async_update_data(self) -> dict[str, Any]:
    """Fetch data from server API."""
    _LOGGER.debug("Server coordinator update starting")
    data = {}

    # Fetch generation
    try:
        _LOGGER.debug("Fetching generation data from %s%s", self._api_url[:20], ENDPOINT_GENERATION)
        async with async_timeout.timeout(30):
            async with self._session.get(...) as response:
                if response.status == 200:
                    data["generation"] = await response.json()
                    _LOGGER.debug("Generation data fetched successfully")
                else:
                    _LOGGER.warning("Generation API returned %s", response.status)
    except Exception as err:
        _LOGGER.error("Error fetching generation: %s", err)

    # ... similar for load and prices ...

    _LOGGER.info(
        "Server coordinator update complete: generation=%s, load=%s, prices=%s",
        bool(data.get("generation")),
        bool(data.get("load")),
        bool(data.get("prices"))
    )

    return data
```

### TenneT Coordinator Logging

```python
async def _async_update_data(self) -> dict[str, Any]:
    """Fetch data from TenneT API."""
    _LOGGER.debug("TenneT coordinator update starting")

    try:
        _LOGGER.debug("Fetching TenneT balance data from %s%s", TENNET_BASE_URL[:20], TENNET_BALANCE_ENDPOINT)
        async with self._session.get(...) as response:
            if response.status == 200:
                raw_data = await response.json()
                _LOGGER.debug("TenneT data fetched successfully, parsing response")
                parsed = self._parse_tennet_response(raw_data)

                _LOGGER.info(
                    "TenneT coordinator update complete: balance=%.2f MW, quality=%s",
                    parsed.get("balance_delta_mw") or 0,
                    parsed.get("quality")
                )
                return parsed
            elif response.status == 401:
                _LOGGER.error("TenneT API key invalid or expired (401)")
                raise UpdateFailed("TenneT API key invalid or expired")
    except aiohttp.ClientError as err:
        _LOGGER.error("TenneT connection error: %s", err)
        raise UpdateFailed(f"TenneT connection error: {err}") from err
```

### Diagnostics Module (MANDATORY)

**File:** `custom_components/ha_energy_insights_nl/diagnostics.py`

```python
import logging

_LOGGER = logging.getLogger(__name__)

async def async_get_config_entry_diagnostics(hass, entry):
    """Return diagnostics for config entry."""
    _LOGGER.debug("Generating diagnostics for %s", entry.entry_id)

    data = hass.data[DOMAIN][entry.entry_id]
    server_coordinator = data["server_coordinator"]
    tennet_coordinator = data.get("tennet_coordinator")

    diagnostics = {
        "entry_id": entry.entry_id,
        "config": {
            "api_url": entry.data.get("api_url", "")[:20],  # Redact for privacy
            "has_api_key": bool(entry.data.get("api_key")),
            "has_tennet_key": bool(entry.data.get("tennet_api_key")),
        },
        "server_coordinator": {
            "name": server_coordinator.name,
            "last_update_success": server_coordinator.last_update_success,
            "last_update_time": server_coordinator.last_update_time,
        },
        "data_status": {
            "has_generation": bool(server_coordinator.data.get("generation")) if server_coordinator.data else False,
            "has_load": bool(server_coordinator.data.get("load")) if server_coordinator.data else False,
            "has_prices": bool(server_coordinator.data.get("prices")) if server_coordinator.data else False,
        },
    }

    if tennet_coordinator:
        diagnostics["tennet_coordinator"] = {
            "name": tennet_coordinator.name,
            "last_update_success": tennet_coordinator.last_update_success,
            "last_update_time": tennet_coordinator.last_update_time,
        }

    _LOGGER.info("Diagnostics generated: generation=%s, load=%s, prices=%s",
                 diagnostics["data_status"].get("has_generation"),
                 diagnostics["data_status"].get("has_load"),
                 diagnostics["data_status"].get("has_prices"))

    return diagnostics
```

---

## STARTUP VALIDATION PATTERN

**CRITICAL FOR PRODUCTION RELIABILITY:** All services, collectors, normalizers, and importers must validate dependencies at startup before attempting data processing.

### Database Connection Validation

**File:** `synctacles_db/normalizers/base.py`

```python
"""Base utilities for all normalizers - validation at startup."""
from sqlalchemy import create_engine, text
from config.settings import DATABASE_URL
import logging

_LOGGER = logging.getLogger(__name__)

def validate_db_connection():
    """
    Fail-fast database validation at startup.

    Ensures:
    - Database is reachable
    - Credentials are correct
    - User has necessary permissions

    Returns:
        SQLAlchemy Engine instance if validation succeeds

    Raises:
        SystemExit(1) if validation fails (prevents silent failures)
    """
    try:
        engine = create_engine(DATABASE_URL)
        with engine.connect() as conn:
            conn.execute(text("SELECT 1"))
        _LOGGER.info("✓ Database connectie gevalideerd")
        return engine
    except Exception as e:
        _LOGGER.critical(f"✗ Database connectie FAILED: {e}")
        _LOGGER.critical("  Check DATABASE_URL in /opt/.env")
        _LOGGER.critical("  Verwacht user: energy_insights_nl")
        raise SystemExit(1)
```

### Usage in All Normalizers

**Pattern for ALL normalizer modules:**

```python
# normalize_entso_e_a44.py

import sys
from pathlib import Path
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from config.settings import DATABASE_URL
from synctacles_db.normalizers.base import validate_db_connection
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)

# CRITICAL: Validate at startup (module import time)
validate_db_connection()

# Only proceed if connection is valid
engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)

def main():
    """Main normalizer logic runs AFTER validation passes."""
    _LOGGER.info("A44 normalizer starting")
    # ... data processing ...

if __name__ == '__main__':
    main()
```

### Why This Pattern?

1. **Fail-Fast:** Service exits immediately with clear error message instead of running silently and failing on first query
2. **Clear Error Message:** Admin knows exactly what's wrong:
   - "Database unreachable at localhost" → check DB_HOST
   - "role synctacles does not exist" → check DB_USER
   - "permission denied" → check database permissions
3. **Early Detection:** Startup validation catches misconfiguration before systemd timer runs for 15 minutes with no output
4. **Logging Integration:** Uses standard _LOGGER so error appears in journalctl

### Applied To

All normalizer modules call `validate_db_connection()` at startup:
- ✅ `normalize_entso_e_a44.py`
- ✅ `normalize_prices.py`
- ✅ `normalize_entso_e_a65.py`
- ✅ `normalize_entso_e_a75.py`

All collector modules validate at startup (data collection doesn't start if DB down):
- ✅ `entso_e_a44_prices.py`
- ✅ `entso_e_a65_load.py`
- ✅ `entso_e_a75_generation.py`
- ✅ `energy_charts_prices.py`

### Logging Example (systemd logs)

**When validation passes:**
```
Jan 05 10:15:00 server[12345]: ✓ Database connectie gevalideerd
Jan 05 10:15:00 server[12345]: A44 normalizer starting
Jan 05 10:15:15 server[12345]: A44 normalizer completed: 856 records in 15.2s
```

**When validation fails (DATABASE_URL wrong):**
```
Jan 05 10:15:00 server[12345]: ✗ Database connectie FAILED: (psycopg2.OperationalError) could not connect to server
Jan 05 10:15:00 server[12345]:   Check DATABASE_URL in /opt/.env
Jan 05 10:15:00 server[12345]:   Verwacht user: energy_insights_nl
```

Service exits cleanly with SystemExit(1), systemd logs failure, admin immediately sees the issue.

### Anti-Pattern (DO NOT USE)

```python
# ❌ WRONG - Silent failure
engine = create_engine(DATABASE_URL)  # No validation
Session = sessionmaker(bind=engine)

def main():
    session = Session()
    try:
        # Fails on first query with confusing error
        data = session.query(SomeTable).first()
```

---

## NEW CODE CHECKLIST

All new code must pass this checklist before merge:

### Backend Module Checklist

- [ ] Import: `from synctacles_db.core.logging import get_logger`
- [ ] Logger: `_LOGGER = get_logger(__name__)`
- [ ] Function start: `_LOGGER.info(f"{name} starting")`
- [ ] Timing: `start_time = time.time()` + elapsed calculation
- [ ] Function end: `_LOGGER.info(f"{name} completed in {elapsed:.2f}s")`
- [ ] Error handler: `_LOGGER.error(f"... {type(err).__name__}: {err}")`
- [ ] No secrets in logs (check API keys, tokens, passwords)
- [ ] No per-record logging at WARNING/ERROR level
- [ ] Passes: `python -m py_compile module.py`

### HA Component Checklist

- [ ] Import: `import logging` + `_LOGGER = logging.getLogger(__name__)`
- [ ] Setup: Log entry creation with entry_id
- [ ] Coordinator: Log update start/end with data availability
- [ ] Errors: Log specific HTTP status codes and exception types
- [ ] Diagnostics: `diagnostics.py` file exists and implemented
- [ ] Config: Redact sensitive data in diagnostics (api_key, tokens)
- [ ] Logging: Consistent with HA logging conventions
- [ ] Passes: `python -m py_compile component.py`

---

## MAINTENANCE CHECKLIST

### For Operations Teams

- **Weekly:** Check log rotation (files should be ~10MB)
  ```bash
  ls -lh /var/log/synctacles/
  ```

- **Monthly:** Archive old logs (older than 30 days)
  ```bash
  find /var/log/synctacles -name "*.log.*" -mtime +30 -delete
  ```

- **Alert on errors:** Set up monitoring
  ```bash
  grep "ERROR" /var/log/synctacles/synctacles.log | wc -l
  ```

### For Developers

1. **Before submitting PR:** Verify code passes checklist above
2. **In code review:** Check logging compliance with SKILL_13
3. **During testing:** Validate log output at DEBUG level
4. **After deployment:** Monitor logs for new ERROR patterns

---

## MANDATORY FOR FUTURE CODE

After this update:

1. **All new backend modules MUST use:** `get_logger(__name__)`
2. **All new HA components MUST have:** `diagnostics.py`
3. **Code review MUST verify:** SKILL_13 compliance
4. **Exceptions MUST be logged** with type and message context
5. **No backwards-compatibility exceptions** - consistency is mandatory

Deviations require:
- Explicit review approval
- Documentation of why standard doesn't apply
- Alternative logging pattern must be equivalent or better

---

## PART D: QUALITY INDICATOR LOGGING (Energy Action Endpoint)

### Overview

The `/api/v1/energy-action` endpoint includes quality metadata indicating data freshness and reliability. This section documents the quality logging patterns for the 5-tier fallback chain.

### Quality Levels

| Quality | Confidence | Tier | Log Level | Description |
|---------|------------|------|-----------|-------------|
| `live` | 100% | 1-2 | DEBUG | Fresh ENTSO-E data (< 60 min) |
| `estimated` | 70-85% | 3 | INFO | Energy-Charts fallback or stale data |
| `cached` | 50% | 4a-4b | WARNING | In-memory or PostgreSQL cache |
| `unavailable` | 0% | 5 | ERROR | No data available |

### FallbackManager Logging Pattern

**File:** `synctacles_db/fallback/fallback_manager.py`

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)

async def get_prices_with_fallback(self, country: str = "NL") -> dict:
    """Get prices with fallback chain and quality logging."""

    # Tier 1: Fresh ENTSO-E
    entsoe_data = await self._get_entsoe_prices(country)
    if entsoe_data and self._is_fresh(entsoe_data, threshold_minutes=15):
        _LOGGER.debug(f"Tier 1 (ENTSO-E Fresh): {entsoe_data['price']:.4f} EUR/kWh")
        return {"price": entsoe_data["price"], "quality": "live", "source": "ENTSO-E", "tier": 1}

    # Tier 2: Stale ENTSO-E
    if entsoe_data and self._is_acceptable(entsoe_data, threshold_minutes=60):
        _LOGGER.debug(f"Tier 2 (ENTSO-E Stale): {entsoe_data['price']:.4f} EUR/kWh, age={entsoe_data['age_min']}min")
        return {"price": entsoe_data["price"], "quality": "live", "source": "ENTSO-E", "tier": 2}

    # Tier 3: Energy-Charts
    ec_data = await self._get_energy_charts_prices(country)
    if ec_data:
        _LOGGER.info(f"Tier 3 (Energy-Charts): {ec_data['price']:.4f} EUR/kWh (ENTSO-E unavailable)")
        return {"price": ec_data["price"], "quality": "estimated", "source": "Energy-Charts", "tier": 3}

    # Tier 4a: In-Memory Cache
    cached = self._memory_cache.get(f"prices_{country}")
    if cached:
        _LOGGER.warning(f"Tier 4a (Memory Cache): {cached['price']:.4f} EUR/kWh, age={cached['age_min']}min")
        return {"price": cached["price"], "quality": "cached", "source": "Cache", "tier": 4}

    # Tier 4b: PostgreSQL Cache
    from synctacles_db.services.price_cache import price_cache_service
    db_cached = price_cache_service.get_last_known(country)
    if db_cached:
        _LOGGER.warning(f"Tier 4b (PostgreSQL Cache): {db_cached['price']:.4f} EUR/kWh from {db_cached['source']}")
        return {"price": db_cached["price"], "quality": "cached", "source": "Cache", "tier": 4}

    # Tier 5: Unavailable
    _LOGGER.error(f"Tier 5 (UNAVAILABLE): All sources failed for {country}")
    return {"price": None, "quality": "unavailable", "source": None, "tier": 5}
```

### Price Cache Service Logging

**File:** `synctacles_db/services/price_cache.py`

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)

def store_price(self, price: float, source: str, quality: str, country: str = "NL") -> bool:
    """Store price with logging."""
    try:
        # ... insert logic ...
        _LOGGER.debug(f"Price cached: {price:.4f} EUR/kWh from {source}")
        return True
    except Exception as e:
        _LOGGER.error(f"Failed to cache price: {type(e).__name__}: {e}")
        return False

def get_last_known(self, country: str = "NL") -> Optional[dict]:
    """Get last known price with logging."""
    try:
        row = session.query(PriceCache)...
        if row:
            _LOGGER.debug(f"Cache hit: {row.price_eur_kwh:.4f} EUR/kWh from {row.source}")
            return {...}
        _LOGGER.debug(f"Cache miss for {country}")
        return None
    except Exception as e:
        _LOGGER.error(f"Cache lookup failed: {type(e).__name__}: {e}")
        return None

def cleanup_old_entries(self, hours: int = 24) -> int:
    """Clean old cache entries with logging."""
    try:
        # ... delete logic ...
        if deleted_count > 0:
            _LOGGER.info(f"Cache cleanup: removed {deleted_count} entries older than {hours}h")
        return deleted_count
    except Exception as e:
        _LOGGER.error(f"Cache cleanup failed: {type(e).__name__}: {e}")
        return 0
```

### Energy Action Endpoint Logging

**File:** `synctacles_db/api/endpoints/energy_action.py`

```python
from synctacles_db.core.logging import get_logger

_LOGGER = get_logger(__name__)

@router.get("/v1/energy-action")
async def get_energy_action(country: str = "NL"):
    """Get energy action recommendation with quality logging."""
    _LOGGER.debug(f"Energy action request: country={country}")
    start_time = time.time()

    try:
        result = await fallback_manager.get_prices_with_fallback(country)

        action = calculate_action(result["price"], daily_average)

        elapsed = time.time() - start_time
        _LOGGER.info(
            f"Energy action: action={action}, quality={result['quality']}, "
            f"price={result['price']:.4f}, source={result['source']}, "
            f"tier={result['tier']}, elapsed={elapsed:.3f}s"
        )

        return {
            "action": action,
            "price_eur_kwh": result["price"],
            "quality": result["quality"],
            "source": result["source"],
            "confidence": CONFIDENCE_MAP[result["quality"]],
            ...
        }

    except Exception as e:
        elapsed = time.time() - start_time
        _LOGGER.error(f"Energy action failed after {elapsed:.3f}s: {type(e).__name__}: {e}")
        raise HTTPException(status_code=500, detail="Internal error")
```

### Quality Monitoring Alerts

**Recommended Alert Rules:**

| Metric | Threshold | Severity | Action |
|--------|-----------|----------|--------|
| quality=cached | > 10 requests/hour | WARNING | Check ENTSO-E and Energy-Charts |
| quality=unavailable | > 1 request/hour | CRITICAL | Immediate investigation |
| Tier 4b hits | > 5% of requests | WARNING | Verify fallback chain |
| Cache cleanup failures | Any | ERROR | Check database permissions |

### Log Examples

**Normal operation (Tier 1-2):**
```
2026-01-11 12:00:00 - fallback_manager - DEBUG - Tier 1 (ENTSO-E Fresh): 0.0912 EUR/kWh
2026-01-11 12:00:00 - energy_action - INFO - Energy action: action=WAIT, quality=live, price=0.0912, source=ENTSO-E, tier=1, elapsed=0.015s
```

**Fallback activated (Tier 3):**
```
2026-01-11 12:00:00 - fallback_manager - INFO - Tier 3 (Energy-Charts): 0.0895 EUR/kWh (ENTSO-E unavailable)
2026-01-11 12:00:00 - energy_action - INFO - Energy action: action=WAIT, quality=estimated, price=0.0895, source=Energy-Charts, tier=3, elapsed=0.234s
```

**Cache fallback (Tier 4b):**
```
2026-01-11 12:00:00 - fallback_manager - WARNING - Tier 4b (PostgreSQL Cache): 0.0920 EUR/kWh from entsoe
2026-01-11 12:00:00 - energy_action - INFO - Energy action: action=WAIT, quality=cached, price=0.0920, source=Cache, tier=4, elapsed=0.008s
```

**Complete failure (Tier 5):**
```
2026-01-11 12:00:00 - fallback_manager - ERROR - Tier 5 (UNAVAILABLE): All sources failed for NL
2026-01-11 12:00:00 - energy_action - INFO - Energy action: action=WAIT, quality=unavailable, price=null, source=null, tier=5, elapsed=0.450s
```

---

## CONTACT & ESCALATION

Questions about logging standards:
1. Check this document first (SKILL_13 v2.1)
2. Review examples in `/opt/github/synctacles-api/synctacles_db/`
3. Contact architecture team if unclear

---

**Last Updated:** 2026-01-11
**Status:** ✅ Implemented and Mandatory
**Next Review:** Quarterly or after major architectural changes

---

## CHANGELOG

### v2.1 (2026-01-11)
- Added PART D: Quality Indicator Logging for Energy Action endpoint
- Added 5-tier fallback chain logging patterns
- Added Price Cache Service logging examples
- Added quality monitoring alerts

### v2.0 (2026-01-03)
- Initial implementation post-TASK 01-06
- Centralized backend logging via `get_logger()`
- API middleware logging patterns
- HA integration logging standards
