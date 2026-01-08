# CC TASK 05: API Request Logging Middleware

**Project:** SYNCTACLES  
**Datum:** 2026-01-03  
**Vereist:** TASK 01-04 ✅  
**Type:** Feature uitbreiding

---

## CONTEXT

Pipeline logging compleet (collectors, importers, normalizers).  
Nu: API request/response logging in **apart bestand** (hoog volume).

**Architectuur beslissing:**
```
/var/log/synctacles/synctacles.log  → pipeline (bestaand, laag volume)
/var/log/synctacles/api.log         → API requests (nieuw, hoog volume)
```

Reden: API traffic kan 10-100× hoger zijn, zou pipeline logs overspoelen.

---

## DOEL

- Apart logbestand voor API requests
- Request/response logging via FastAPI middleware
- Timing per request
- Slow request warnings (>1s)
- Auth failures logging

---

## STAP 1: Uitbreiden logging.py

**Bestand:** `/opt/github/synctacles-api/synctacles_db/core/logging.py`

**Toevoegen na bestaande `setup_logging()` functie:**

```python
DEFAULT_API_LOG_PATH = "/var/log/synctacles/api.log"

_api_initialized = False


def setup_api_logging(
    level: str | None = None,
    log_path: str | None = None,
) -> logging.Logger:
    """
    Initialize separate logging for API requests.
    
    High-volume traffic goes to separate file.
    
    Args:
        level: off|error|warning|info|debug (default: LOG_LEVEL env)
        log_path: Override path (default: LOG_API_PATH env or /var/log/synctacles/api.log)
    
    Returns:
        Logger for API namespace
    """
    global _api_initialized
    if _api_initialized:
        return logging.getLogger("synctacles_db.api.requests")
    
    level = level or os.getenv("LOG_LEVEL", DEFAULT_LOG_LEVEL)
    log_path = log_path or os.getenv("LOG_API_PATH", DEFAULT_API_LOG_PATH)
    level_num = LOG_LEVELS.get(level.lower(), logging.WARNING)
    
    if level.lower() == "off":
        _api_initialized = True
        api_logger = logging.getLogger("synctacles_db.api.requests")
        api_logger.addHandler(logging.NullHandler())
        return api_logger
    
    log_dir = Path(log_path).parent
    log_dir.mkdir(parents=True, exist_ok=True)
    
    handler = RotatingFileHandler(
        log_path,
        maxBytes=MAX_BYTES,
        backupCount=BACKUP_COUNT,
        encoding="utf-8",
    )
    handler.setFormatter(logging.Formatter(
        "%(asctime)s [%(levelname)-7s] %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    ))
    
    api_logger = logging.getLogger("synctacles_db.api.requests")
    api_logger.setLevel(level_num)
    api_logger.addHandler(handler)
    api_logger.propagate = False
    
    _api_initialized = True
    
    # Log init to main log, not api log
    main_logger = logging.getLogger("synctacles_db")
    main_logger.info(f"API logging initialized: level={level}, path={log_path}")
    
    return api_logger


def get_api_logger() -> logging.Logger:
    """Get logger for API request logging."""
    if not _api_initialized:
        setup_api_logging()
    return logging.getLogger("synctacles_db.api.requests")
```

**Update `__all__` in core/__init__.py:**

```python
from synctacles_db.core.logging import setup_logging, get_logger, setup_api_logging, get_api_logger

__all__ = ["setup_logging", "get_logger", "setup_api_logging", "get_api_logger"]
```

---

## STAP 2: Maak middleware bestand

**Bestand:** `/opt/github/synctacles-api/synctacles_db/api/middleware.py`

```python
"""
API request/response logging middleware.

Logs to separate file: /var/log/synctacles/api.log
"""
import time
from fastapi import Request, Response
from starlette.middleware.base import BaseHTTPMiddleware

from synctacles_db.core.logging import get_api_logger

_LOGGER = get_api_logger()

SLOW_REQUEST_THRESHOLD_MS = 1000  # 1 second


class RequestLoggingMiddleware(BaseHTTPMiddleware):
    """Log all API requests with timing."""
    
    async def dispatch(self, request: Request, call_next) -> Response:
        start_time = time.time()
        
        # Extract request info
        method = request.method
        path = request.url.path
        client_ip = request.client.host if request.client else "unknown"
        
        _LOGGER.debug(f">>> {method} {path} from {client_ip}")
        
        try:
            response = await call_next(request)
            
            elapsed_ms = (time.time() - start_time) * 1000
            status = response.status_code
            
            # Log level based on status and timing
            if status >= 500:
                _LOGGER.error(f"<<< {method} {path} {status} ({elapsed_ms:.0f}ms)")
            elif status >= 400:
                _LOGGER.warning(f"<<< {method} {path} {status} ({elapsed_ms:.0f}ms)")
            elif elapsed_ms > SLOW_REQUEST_THRESHOLD_MS:
                _LOGGER.warning(f"<<< {method} {path} {status} SLOW ({elapsed_ms:.0f}ms)")
            else:
                _LOGGER.debug(f"<<< {method} {path} {status} ({elapsed_ms:.0f}ms)")
            
            return response
            
        except Exception as err:
            elapsed_ms = (time.time() - start_time) * 1000
            _LOGGER.error(f"<<< {method} {path} EXCEPTION ({elapsed_ms:.0f}ms): {type(err).__name__}: {err}")
            raise
```

---

## STAP 3: Registreer middleware in main.py

**Bestand:** `/opt/github/synctacles-api/synctacles_db/api/main.py`

**Toevoegen na app = FastAPI(...):**

```python
from synctacles_db.api.middleware import RequestLoggingMiddleware
from synctacles_db.core.logging import setup_api_logging

# Initialize API logging (separate file)
setup_api_logging()

# Add request logging middleware
app.add_middleware(RequestLoggingMiddleware)
```

---

## STAP 4: Update .env bestanden

**Bestand:** `/opt/.env` - toevoegen:

```bash
LOG_API_PATH=/var/log/synctacles/api.log
```

**Bestand:** `.env.example` - toevoegen:

```bash
# API request logging (high volume, separate file)
LOG_API_PATH=/var/log/synctacles/api.log
```

---

## STAP 5: Validatie

```bash
export LOG_LEVEL=debug

# Restart API
sudo systemctl restart energy-insights-nl-api
sleep 3

# Generate requests
curl http://localhost:8000/health
curl http://localhost:8000/v1/generation/current
curl http://localhost:8000/nonexistent  # 404

# Check separate log files
echo "=== Pipeline log ==="
tail -10 /var/log/synctacles/synctacles.log

echo "=== API log ==="
tail -10 /var/log/synctacles/api.log

export LOG_LEVEL=warning
sudo systemctl restart energy-insights-nl-api
```

**Verwachte api.log output:**
```
2026-01-03 15:00:00 [DEBUG  ] >>> GET /health from 127.0.0.1
2026-01-03 15:00:00 [DEBUG  ] <<< GET /health 200 (5ms)
2026-01-03 15:00:01 [DEBUG  ] >>> GET /v1/generation/current from 127.0.0.1
2026-01-03 15:00:01 [DEBUG  ] <<< GET /v1/generation/current 200 (45ms)
2026-01-03 15:00:02 [WARNING] <<< GET /nonexistent 404 (2ms)
```

---

## STAP 6: Fix ownership + commit

```bash
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "feat: add API request logging middleware

- Separate log file for API traffic: /var/log/synctacles/api.log
- RequestLoggingMiddleware with timing per request
- WARNING for slow requests (>1s) and 4xx errors
- ERROR for 5xx and exceptions
- DEBUG for normal request/response flow

Part 5 of logging implementation series."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## DONE CRITERIA

- [ ] `logging.py` uitgebreid met `setup_api_logging()` en `get_api_logger()`
- [ ] `middleware.py` aangemaakt met RequestLoggingMiddleware
- [ ] `main.py` registreert middleware
- [ ] `/var/log/synctacles/api.log` wordt aangemaakt
- [ ] Pipeline logs blijven in `synctacles.log`
- [ ] Slow requests (>1s) genereren WARNING
- [ ] 4xx/5xx genereren WARNING/ERROR
- [ ] Git commit gepusht

---

## LOG BESTANDEN OVERZICHT (na TASK 05)

```
/var/log/synctacles/
├── synctacles.log      → Pipeline: collectors, importers, normalizers
├── synctacles.log.1    → Rotated
├── synctacles.log.2
├── synctacles.log.3
├── api.log             → API requests (hoog volume)
├── api.log.1           → Rotated
├── api.log.2
└── api.log.3
```

---

## VOLGENDE TAAK

Na voltooiing → TASK 06 (HA Integration diagnostics)
