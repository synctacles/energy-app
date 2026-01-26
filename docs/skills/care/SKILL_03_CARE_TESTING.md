# SKILL CARE 03 — TESTING INFRASTRUCTURE

Unit Tests, Integration Tests, and CI Pipeline
**Version: 1.0 (2026-01-26)**

> **Philosophy:** Tests run on DEV machine, not on customer HA.
> Integration tests validate against real HA database via SSH.

---

## TEST OVERVIEW

| Type | Count | Location | Runs Against |
|------|-------|----------|--------------|
| Unit Tests | 105 | `tests/test_*.py` | Mock database (local) |
| Integration Tests | 12 | `tests/test_integration_ha.py` | Real HA DEV database |

---

## RUNNING TESTS

### Unit Tests (Default)
```bash
cd /opt/github/addon-synctacles-care

# Run all unit tests (excludes integration by default)
python3 -m pytest tests/ -v

# Run with coverage
python3 -m pytest tests/ --cov=synctacles-care/care --cov-report=term-missing
```

### Integration Tests
```bash
# Requires SSH access to HA DEV (see Section H3 in core/SKILL_00_AI_PROTOCOL)
python3 -m pytest -m integration tests/test_integration_ha.py -v
```

### All Tests
```bash
python3 -m pytest tests/ -v -m ""  # Empty marker = all tests
```

---

## PYTEST CONFIGURATION

**File:** `pyproject.toml`

```toml
[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]
markers = [
    "integration: marks tests as integration tests (require HA DEV access)",
]
# Exclude integration tests by default
addopts = "-m 'not integration'"
```

---

## UNIT TEST STRUCTURE

### Fixtures (conftest.py)
```python
@pytest.fixture
def mock_config_path(tmp_path):
    """Create a mock HA config directory with database."""
    # Creates:
    # - tmp_path/home-assistant_v2.db (SQLite with test data)
    # - tmp_path/.storage/core.entity_registry (JSON)
    # - tmp_path/.storage/auth (JSON)
    # - tmp_path/.HA_VERSION
    return tmp_path

@pytest.fixture
def mock_db_path(mock_config_path):
    """Return path to mock database."""
    return mock_config_path / "home-assistant_v2.db"
```

### Test Database Schema
The mock database includes:
- `states` table (with test entity states)
- `states_meta` table (entity metadata)
- `statistics_meta` table (statistics metadata)
- `statistics` table (historical data)
- `statistics_short_term` table

### Test Data Scenarios
```python
# Active entity with state and statistics
"sensor.active_sensor" → In states_meta, has statistics

# Orphaned statistic (no matching entity)
"sensor.deleted_sensor" → In statistics_meta, NOT in states_meta

# External statistic (should be ignored by cleanup)
"sensor.external" → source="external_source", not "recorder"
```

---

## UNIT TEST COVERAGE

### Scanner Tests

| File | Tests | Coverage |
|------|-------|----------|
| `test_scanner_health.py` | 25 | HealthScanner class |
| `test_scanner_security.py` | 18 | SecurityScanner class |
| `test_scanner_schema.py` | 15 | Schema detection |
| `test_scanner_cleanup.py` | 12 | DatabaseCleaner |

### Cleaner Tests

| File | Tests | Coverage |
|------|-------|----------|
| `test_cleaner_backup.py` | 10 | BackupManager |
| `test_cleaner_orphan_stats.py` | 8 | Orphan cleanup |
| `test_cleaner_database.py` | 7 | Transaction handling |
| `test_cleaner_safety.py` | 5 | Safety checks |

### Utility Tests

| File | Tests | Coverage |
|------|-------|----------|
| `test_utils_version.py` | 5 | Version comparison |
| `test_utils_supervisor.py` | - | Supervisor API mocks |

---

## INTEGRATION TESTS

### Purpose
Integration tests connect to the **real HA DEV** environment to verify:
1. Schema detection works with actual HA database
2. Orphan detection finds real orphans
3. Scanner handles real-world data structures

### Architecture
```
┌─────────────────────────────────────────────────────────────────┐
│                    SYNCT-DEV (This Machine)                      │
│                    Hetzner VPS (Germany)                         │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                   pytest runner                             │ │
│  │                                                             │ │
│  │  1. SSH connect to HA DEV (82.169.33.175:22222)            │ │
│  │  2. SCP copy database to local temp directory              │ │
│  │  3. Run scanners against local copy                         │ │
│  │  4. Verify results                                          │ │
│  └─────────────────────────────┬───────────────────────────────┘ │
│                                │                                 │
└────────────────────────────────┼─────────────────────────────────┘
                                 │ SSH/SCP
                                 │ Port 22222
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                       HA DEV (Home)                              │
│                    82.169.33.175:22222                           │
│                                                                  │
│  /config/                                                        │
│  ├── home-assistant_v2.db     (~78 MB, schema 53)               │
│  ├── .storage/                                                   │
│  │   └── core.entity_registry                                    │
│  └── .HA_VERSION              (2026.x.x)                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Connection Details
**Source:** core/SKILL_00_AI_PROTOCOL.md Section H3

```python
# tests/test_integration_ha.py
HA_HOST = "82.169.33.175"
HA_PORT = "22222"
HA_USER = "root"
HA_KEY = os.path.expanduser("~/.ssh/id_ha")
HA_DB_PATH = "/config/home-assistant_v2.db"
HA_STORAGE_PATH = "/config/.storage"
```

### Integration Test Classes

#### TestSchemaDetectionIntegration
```python
def test_detects_real_schema_version(self, ha_database_copy):
    """Test that we correctly detect the schema version of real HA."""
    info = get_schema_version(db_path)
    assert info.version >= 41  # Minimum supported
    assert info.supported is True
    assert info.has_statistics_meta is True
    assert info.has_states_meta is True
```

#### TestHealthScannerIntegration
```python
async def test_health_scan_completes(self, ha_database_copy):
    """Test that health scan completes without errors on real HA data."""
    scanner = HealthScanner(ha_database_copy)
    report = await scanner.scan()

    assert report.grade in ["A", "B", "C", "D", "F"]
    assert 0 <= report.score <= 100
    assert report.db_size_bytes > 0
    assert report.schema_version >= 41
```

#### TestDatabaseCleanerIntegration
```python
def test_cleaner_lock_check(self, ha_database_copy):
    """Test database lock checking works."""
    cleaner = DatabaseCleaner(ha_database_copy)
    is_locked = cleaner.is_database_locked()
    assert is_locked is False  # Local copy shouldn't be locked
```

#### TestRemoteHAQueries
```python
def test_schema_version_query(self, ha_available):
    """Query schema version directly via SSH."""
    result = ssh_command(
        f"sqlite3 {HA_DB_PATH} "
        "'SELECT schema_version FROM schema_changes ORDER BY change_id DESC LIMIT 1'"
    )
    version = int(result.stdout.strip())
    assert version >= 41
```

### Database Corruption Handling
HA may be writing to the database during copy. Tests handle this gracefully:

```python
async def test_health_scan_completes(self, ha_database_copy):
    try:
        report = await scanner.scan()
    except sqlite3.DatabaseError as e:
        if "malformed" in str(e):
            pytest.skip("Database corrupted during copy (HA was writing)")
        raise
```

---

## CI PIPELINE

### GitHub Actions Workflow
**File:** `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Python 3.11
        uses: actions/setup-python@v5
        with:
          python-version: "3.11"

      - name: Install dependencies
        run: |
          pip install -e ".[dev]"

      - name: Ruff format check
        run: ruff format --check synctacles-care/care/

      - name: Ruff lint check
        run: ruff check synctacles-care/care/

      - name: Run tests
        run: pytest tests/ -v --tb=short
```

### Pre-Push Hook
**File:** `.git/hooks/pre-push`

```bash
#!/bin/bash
set -e

echo "Running pre-push checks..."

# Ruff format check
ruff format --check synctacles-care/care/

# Ruff lint check
ruff check synctacles-care/care/

# Run tests (unit only, not integration)
python3 -m pytest tests/ -q

echo "All checks passed!"
```

---

## TEST COMMANDS SUMMARY

| Command | Description |
|---------|-------------|
| `pytest tests/ -v` | Run unit tests (verbose) |
| `pytest -m integration -v` | Run integration tests only |
| `pytest tests/ --cov=synctacles-care/care` | Run with coverage |
| `pytest tests/ -k "health"` | Run tests matching "health" |
| `pytest tests/ --tb=short` | Short traceback on failure |
| `ruff format --check synctacles-care/care/` | Check formatting |
| `ruff check synctacles-care/care/` | Run linter |

---

## TROUBLESHOOTING

### Integration Tests Fail with Connection Refused
```
E       subprocess.TimeoutExpired: Command timed out
```
**Fix:** Ensure SSH key exists and HA DEV is online:
```bash
ssh -i ~/.ssh/id_ha -p 22222 root@82.169.33.175 "echo OK"
```

### Database Malformed Error
```
sqlite3.DatabaseError: database disk image is malformed
```
**Cause:** HA was writing during SCP copy.
**Fix:** Test automatically skips. Retry later when HA is idle.

### Module Not Found
```
ModuleNotFoundError: No module named 'care'
```
**Fix:** Integration test adds path automatically:
```python
sys.path.insert(0, str(Path(__file__).parent.parent / "synctacles-care"))
```

---

## RELATED SKILLS

| Skill | Description |
|-------|-------------|
| [SKILL_00_CARE_OVERVIEW.md](SKILL_00_CARE_OVERVIEW.md) | Product overview |
| [SKILL_01_CARE_SAFETY.md](SKILL_01_CARE_SAFETY.md) | Safety mitigations tested |
| [../core/SKILL_00_AI_PROTOCOL.md](../core/SKILL_00_AI_PROTOCOL.md) | Section H3: HA DEV access |

---

*Generated: 2026-01-26*
