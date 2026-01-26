# SKILL CARE 02 — TECHNICAL ARCHITECTURE

Code Structure, Data Flow, and Implementation Details
**Version: 1.0 (2026-01-26)**

> **Runtime Environment:** Docker container running as HA add-on with access to /config volume.

---

## DEPLOYMENT ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         HOME ASSISTANT HOST                              │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                    SUPERVISOR (hassio)                              │ │
│  │                                                                      │ │
│  │  ┌─────────────────────┐  ┌─────────────────────────────────────┐  │ │
│  │  │   HA Core Container │  │    SYNCTACLES Care Add-on          │  │ │
│  │  │                     │  │         (Docker)                    │  │ │
│  │  │  - Recorder         │  │                                     │  │ │
│  │  │  - SQLite writes    │  │  Python 3.11 + aiohttp             │  │ │
│  │  │  - Nightly purge    │  │                                     │  │ │
│  │  └──────────┬──────────┘  │  ┌──────────────────────────────┐  │  │ │
│  │             │             │  │         Web Server           │  │  │ │
│  │             │             │  │       (port 8099)            │  │  │ │
│  │             │             │  │                              │  │  │ │
│  │             │             │  │  GET /api/health-scan        │  │  │ │
│  │             │             │  │  GET /api/security-scan      │  │  │ │
│  │             │             │  │  POST /api/cleanup (Premium) │  │  │ │
│  │             │             │  └──────────────────────────────┘  │  │ │
│  │             │             │              │                      │  │ │
│  │             │             │              │ Scanner/Cleaner      │  │ │
│  │             │             │              ▼                      │  │ │
│  │             │             │  ┌──────────────────────────────┐  │  │ │
│  │             │             │  │     /config (volume)         │  │  │ │
│  │             │             │  │                              │  │  │ │
│  │             ▼             │  │  - home-assistant_v2.db      │  │  │ │
│  │  ┌──────────────────┐    │  │  - .storage/*                 │  │  │ │
│  │  │   /config        │◀───┼──│  - .HA_VERSION                │  │  │ │
│  │  │   (shared vol)   │    │  └──────────────────────────────┘  │  │ │
│  │  └──────────────────┘    └─────────────────────────────────────┘  │ │
│  │                                         │                          │ │
│  │                                         │ Supervisor API           │ │
│  │                                         │ (core info, addon mgmt)  │ │
│  │                                         ▼                          │ │
│  │                          ┌────────────────────────┐                │ │
│  │                          │   Supervisor REST API   │                │ │
│  │                          │   SUPERVISOR_TOKEN      │                │ │
│  │                          └────────────────────────┘                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## CODE STRUCTURE

```
addon-synctacles-care/
├── synctacles-care/                    # Python package root
│   └── care/
│       ├── __init__.py                 # Package init
│       ├── main.py                     # Entry point (run via supervisor)
│       ├── config.py                   # Configuration from options.json
│       │
│       ├── scanner/                    # READ-ONLY operations (FREE tier)
│       │   ├── __init__.py
│       │   ├── health.py               # HealthScanner class
│       │   ├── security.py             # SecurityScanner class
│       │   ├── schema.py               # Schema detection (P2 #27)
│       │   ├── cleanup.py              # DatabaseCleaner (dry-run)
│       │   └── models.py               # HealthReport, SecurityReport, etc.
│       │
│       ├── cleaner/                    # WRITE operations (PREMIUM tier)
│       │   ├── __init__.py
│       │   ├── orphans.py              # Orphan cleanup logic
│       │   ├── backup.py               # BackupManager
│       │   ├── database.py             # Transaction handling
│       │   └── safeguards.py           # Pre-cleanup safety checks
│       │
│       ├── utils/
│       │   ├── __init__.py
│       │   ├── version.py              # HA version check (P2 #25)
│       │   ├── supervisor.py           # Supervisor API client
│       │   ├── ha_api.py               # HA REST API client
│       │   └── logging.py              # Structured logging setup
│       │
│       ├── api/
│       │   ├── __init__.py
│       │   ├── client.py               # SYNCTACLES cloud API client
│       │   └── models.py               # LicenseInfo, etc.
│       │
│       └── web/
│           ├── __init__.py
│           ├── server.py               # aiohttp Application
│           └── routes.py               # HTTP route handlers
│
├── tests/                              # Pytest test suite
│   ├── conftest.py                     # Shared fixtures
│   ├── test_scanner_health.py          # HealthScanner tests
│   ├── test_scanner_security.py        # SecurityScanner tests
│   ├── test_scanner_schema.py          # Schema detection tests
│   ├── test_scanner_cleanup.py         # DatabaseCleaner tests
│   ├── test_cleaner_*.py               # Cleaner module tests
│   ├── test_utils_*.py                 # Utility tests
│   └── test_integration_ha.py          # Integration tests (HA DEV)
│
├── config.yaml                         # HA add-on configuration
├── Dockerfile                          # Container build
├── run.sh                              # Container entrypoint
└── pyproject.toml                      # Dependencies + pytest config
```

---

## KEY CLASSES

### HealthScanner
**File:** `care/scanner/health.py`

```python
class HealthScanner:
    """Scans Home Assistant database for health issues."""

    # Thresholds
    SIZE_WARNING_GB = 1.0
    SIZE_CRITICAL_GB = 5.0
    FRAG_WARNING_PCT = 20.0
    FRAG_CRITICAL_PCT = 40.0

    def __init__(self, config_path: Path):
        self.config_path = config_path
        self.db_path = config_path / "home-assistant_v2.db"
        self.entity_registry_path = config_path / ".storage" / "core.entity_registry"

    async def scan(self) -> HealthReport:
        """Run complete health scan."""
        # 1. Check schema version (P2 #27)
        # 2. Analyze database size
        # 3. Calculate fragmentation
        # 4. Find orphaned statistics
        # 5. Find orphaned entities
        # 6. Calculate grade
        return HealthReport(...)
```

### SecurityScanner
**File:** `care/scanner/security.py`

```python
class SecurityScanner:
    """Scans Home Assistant for security issues."""

    def __init__(self, config_path: Path):
        self.storage_path = config_path / ".storage"

    async def scan(self) -> SecurityReport:
        """Run security audit."""
        # 1. Check MFA status
        # 2. Check exposed ports
        # 3. Check SSL/HTTPS
        # 4. Check password policies
        # 5. Calculate score (0-100)
        return SecurityReport(...)
```

### DatabaseCleaner
**File:** `care/scanner/cleanup.py`

```python
class DatabaseCleaner:
    """Database cleanup operations (Premium feature)."""

    def __init__(self, config_path: Path):
        self.db_path = config_path / "home-assistant_v2.db"

    def is_database_locked(self) -> bool:
        """Check if database has active locks."""

    def is_purge_window(self) -> bool:
        """Check if in HA nightly purge window (06:00-07:00 UTC)."""

    async def clean_orphaned_statistics(
        self,
        execute: bool = False  # DRY-RUN DEFAULT (P0 #21)
    ) -> CleanupResult:
        """Remove orphaned statistics."""
```

### SupervisorClient
**File:** `care/utils/supervisor.py`

```python
class SupervisorClient:
    """Client for Home Assistant Supervisor API."""

    def __init__(self, token: str):
        self.token = token
        self.base_url = "http://supervisor"

    async def get_core_info(self) -> CoreInfo:
        """Get HA Core information (version, state)."""

    async def stop_core(self) -> bool:
        """Stop HA Core (for safe cleanup)."""

    async def start_core(self) -> bool:
        """Start HA Core after cleanup."""
```

---

## DATA MODELS

### HealthReport
**File:** `care/scanner/models.py`

```python
@dataclass
class HealthReport:
    grade: str                          # A, B, C, D, F
    score: int                          # 0-100
    db_size_bytes: int
    db_size_human: str                  # "500 MB"
    fragmentation_pct: float
    schema_version: int
    schema_supported: bool
    orphaned_statistics_count: int
    orphaned_statistics_list: list[OrphanedStatistic]
    orphaned_entities_count: int
    orphaned_entities_list: list[OrphanedEntity]
    recommendations: list[Recommendation]
    errors: list[ScanError]
    scan_duration_ms: int
    timestamp: datetime
```

### OrphanedStatistic
```python
@dataclass
class OrphanedStatistic:
    statistic_id: str                   # e.g., "sensor.deleted_device"
    metadata_id: int
    source: str                         # "recorder" or external
    unit_of_measurement: str
    row_count: int                      # Number of data rows
    estimated_size_bytes: int
    oldest_record: datetime
    newest_record: datetime
```

### SchemaInfo
```python
@dataclass
class SchemaInfo:
    version: int
    supported: bool = True
    min_version: int = MIN_SCHEMA_VERSION
    description: str = ""
    warnings: list[str] = field(default_factory=list)
    has_states_meta: bool = False
    has_statistics_meta: bool = False
    has_events_meta: bool = False
```

---

## DATABASE QUERIES

### Orphaned Statistics Detection
```sql
-- Find statistics_meta entries with no matching entity
SELECT
    sm.id AS metadata_id,
    sm.statistic_id,
    sm.source,
    sm.unit_of_measurement,
    COUNT(s.id) AS row_count
FROM statistics_meta sm
LEFT JOIN states_meta stm ON sm.statistic_id = stm.entity_id
LEFT JOIN statistics s ON s.metadata_id = sm.id
WHERE stm.entity_id IS NULL
  AND sm.source = 'recorder'  -- Only HA-generated, not external
GROUP BY sm.id
ORDER BY row_count DESC;
```

### Database Size & Fragmentation
```sql
-- Get page counts for fragmentation calculation
PRAGMA page_count;      -- Total pages
PRAGMA freelist_count;  -- Free (fragmented) pages

-- Fragmentation % = (freelist_count / page_count) * 100
```

### Schema Version Detection
```sql
-- Modern (HA 2021.7+)
SELECT schema_version
FROM schema_changes
ORDER BY change_id DESC
LIMIT 1;

-- Legacy fallback
SELECT version FROM schema_version LIMIT 1;
```

---

## WEB ENDPOINTS

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/` | GET | No | Web UI (ingress) |
| `/api/health-scan` | GET | Bearer | Run health scan |
| `/api/security-scan` | GET | Bearer | Run security scan |
| `/api/cleanup` | POST | Bearer + Premium | Execute cleanup |
| `/api/status` | GET | No | Add-on status |

### Example Response
```http
GET /api/health-scan
Authorization: Bearer <ingress_token>

HTTP/1.1 200 OK
Content-Type: application/json

{
  "grade": "B",
  "score": 78,
  "db_size_bytes": 524288000,
  "db_size_human": "500 MB",
  "fragmentation_pct": 15.2,
  "schema_version": 53,
  "orphaned_statistics_count": 247,
  "orphaned_entities_count": 12,
  "recommendations": [
    {
      "type": "cleanup",
      "severity": "warning",
      "message": "247 orphaned statistics found"
    }
  ]
}
```

---

## CONFIGURATION

### config.yaml (Add-on Config)
```yaml
name: "SYNCTACLES Care"
version: "1.0.0"
slug: "synctacles_care"
description: "Database maintenance & security audit for Home Assistant"
url: "https://github.com/synctacles/addon-synctacles-care"
arch:
  - aarch64
  - amd64
  - armv7
  - i386
init: false
homeassistant_api: true
ingress: true
ingress_port: 8099
panel_icon: "mdi:database-check"
panel_title: "Care"
map:
  - config:rw  # Read-write access to /config
options:
  log_level: info
schema:
  log_level: list(debug|info|warning|error)
```

### Environment Variables
```bash
SUPERVISOR_TOKEN    # Auto-injected by Supervisor
CONFIG_PATH         # /config (mounted volume)
```

---

## LOGGING

### Log Levels
```python
from care.utils.logging import get_logger

logger = get_logger()

logger.debug("Detailed trace info")
logger.info("Normal operation")
logger.warning("Non-critical issue")
logger.error("Error occurred")
```

### Log Format
```
2026-01-26 14:32:15 [INFO] care.scanner.health: Starting health scan
2026-01-26 14:32:15 [INFO] care.scanner.schema: Database schema version: 53 (supported)
2026-01-26 14:32:16 [INFO] care.scanner.health: Found 247 orphaned statistics
2026-01-26 14:32:16 [INFO] care.scanner.health: Health scan complete: grade=B, score=78
```

---

## DEPENDENCIES

### Runtime (pyproject.toml)
```toml
dependencies = [
    "aiohttp>=3.9.0",
    "pyyaml>=6.0",
]
```

### Development
```toml
[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "pytest-asyncio>=0.21",
    "pytest-cov>=4.0",
    "ruff>=0.1",
]
```

---

## RELATED SKILLS

| Skill | Description |
|-------|-------------|
| [SKILL_00_CARE_OVERVIEW.md](SKILL_00_CARE_OVERVIEW.md) | Product overview |
| [SKILL_01_CARE_SAFETY.md](SKILL_01_CARE_SAFETY.md) | Safety mitigations |
| [SKILL_03_CARE_TESTING.md](SKILL_03_CARE_TESTING.md) | Testing infrastructure |

---

*Generated: 2026-01-26*
