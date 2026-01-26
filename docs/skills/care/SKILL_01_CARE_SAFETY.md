# SKILL CARE 01 — SAFETY MITIGATIONS

P0-P2 Safety Framework for Database Operations
**Version: 1.0 (2026-01-26)**

> **Philosophy:** We operate on customer databases. Data loss = dead product.
> Every write operation requires explicit safeguards.

---

## PRIORITY MATRIX

| Priority | Impact | Response Time | Examples |
|----------|--------|---------------|----------|
| **P0** | Data destruction | Block release | Cascade delete without backup |
| **P1** | Data loss | Fix before test | Cleanup without dry-run |
| **P2** | User confusion | Fix in sprint | Wrong schema detection |

---

## P0: CRITICAL MITIGATIONS

### P0 #20: Pre-Cleanup Backup
**Status:** ✅ Implemented

**What:** Automatic backup creation before any cleanup operation.

**Implementation:**
```python
# care/cleaner/backup.py
class BackupManager:
    async def create_backup(self, db_path: Path) -> BackupResult:
        """Create timestamped backup before cleanup."""
        backup_name = f"care_backup_{datetime.now():%Y%m%d_%H%M%S}.db"
        backup_path = self.backup_dir / backup_name
        shutil.copy2(db_path, backup_path)
        return BackupResult(path=backup_path, size=backup_path.stat().st_size)
```

**Test Coverage:**
- `test_cleaner_backup.py::test_backup_created_before_cleanup`
- `test_cleaner_backup.py::test_backup_naming_convention`
- `test_cleaner_backup.py::test_backup_file_integrity`

---

### P0 #21: Dry-Run Default
**Status:** ✅ Implemented

**What:** All cleanup operations default to dry-run mode. Actual deletion requires explicit `execute=True`.

**Implementation:**
```python
# care/scanner/cleanup.py
class DatabaseCleaner:
    async def clean_orphaned_statistics(
        self,
        execute: bool = False  # DRY-RUN BY DEFAULT
    ) -> CleanupResult:
        """Remove orphaned statistics.

        Args:
            execute: If False (default), only simulate. If True, actually delete.
        """
        if not execute:
            logger.info("DRY-RUN: Would delete %d rows", count)
            return CleanupResult(deleted=0, would_delete=count, dry_run=True)

        # Actual deletion only with execute=True
        ...
```

**Test Coverage:**
- `test_cleaner_orphan_stats.py::test_dry_run_does_not_delete`
- `test_cleaner_orphan_stats.py::test_execute_true_required_for_deletion`

---

### P0 #22: Transaction Rollback
**Status:** ✅ Implemented

**What:** All write operations wrapped in transactions with automatic rollback on error.

**Implementation:**
```python
# care/cleaner/database.py
async def execute_in_transaction(conn, operations):
    """Execute operations in transaction with rollback on error."""
    try:
        conn.execute("BEGIN TRANSACTION")
        for op in operations:
            conn.execute(op)
        conn.execute("COMMIT")
    except Exception as e:
        conn.execute("ROLLBACK")
        logger.error("Transaction rolled back: %s", e)
        raise
```

**Test Coverage:**
- `test_cleaner_database.py::test_transaction_rollback_on_error`
- `test_cleaner_database.py::test_partial_failure_rolls_back_all`

---

### P0 #23: No Cascade Deletes
**Status:** ✅ Enforced

**What:** Never use `ON DELETE CASCADE`. All deletions are explicit and logged.

**SQL Policy:**
```sql
-- FORBIDDEN: CASCADE deletes
DELETE FROM statistics_meta WHERE id = ?;  -- Would cascade to statistics!

-- REQUIRED: Explicit child-first deletion
DELETE FROM statistics WHERE metadata_id = ?;
DELETE FROM statistics_short_term WHERE metadata_id = ?;
DELETE FROM statistics_meta WHERE id = ?;
```

**Test Coverage:**
- `test_cleaner_safety.py::test_no_cascade_in_delete_queries`
- `test_cleaner_safety.py::test_child_tables_deleted_first`

---

## P1: IMPORTANT MITIGATIONS

### P1 #24: HA Shutdown Check
**Status:** ✅ Implemented

**What:** Verify Home Assistant recorder is stopped before cleanup.

**Implementation:**
```python
# care/cleaner/safeguards.py
class SafeguardChecker:
    async def check_ha_stopped(self) -> bool:
        """Check if HA recorder is stopped (safe for cleanup)."""
        # Check via Supervisor API
        core_info = await self.supervisor.get_core_info()
        return core_info.state == "stopped"

    def is_database_locked(self) -> bool:
        """Check if database has active locks."""
        try:
            conn = sqlite3.connect(str(self.db_path), timeout=1)
            conn.execute("BEGIN EXCLUSIVE")
            conn.execute("ROLLBACK")
            conn.close()
            return False  # No lock
        except sqlite3.OperationalError:
            return True  # Locked
```

**Test Coverage:**
- `test_cleaner_safeguards.py::test_cleanup_blocked_when_ha_running`
- `test_cleaner_safeguards.py::test_database_lock_detection`

---

### P1 #26: Purge Window Conflict
**Status:** ✅ Implemented

**What:** Detect if HA is currently purging (06:12 UTC nightly purge).

**Implementation:**
```python
# care/scanner/cleanup.py
PURGE_WINDOW_START = time(6, 0)   # 06:00 UTC
PURGE_WINDOW_END = time(7, 0)     # 07:00 UTC

class DatabaseCleaner:
    def is_purge_window(self) -> bool:
        """Check if currently in HA nightly purge window."""
        now = datetime.now(timezone.utc).time()
        return PURGE_WINDOW_START <= now <= PURGE_WINDOW_END
```

**Test Coverage:**
- `test_scanner_cleanup.py::test_purge_window_detection`
- `test_scanner_cleanup.py::test_cleanup_warns_during_purge`

---

## P2: MINOR MITIGATIONS

### P2 #25: HA Version Compatibility
**Status:** ✅ Implemented

**What:** Check Home Assistant version and warn if below minimum or known issues.

**File:** `care/utils/version.py`

**Configuration:**
```python
MIN_HA_VERSION = "2024.1.0"  # Minimum supported

KNOWN_ISSUES: dict[str, str] = {
    # "2024.8.0": "Database schema migration issues",
}
```

**Output:**
```python
@dataclass
class VersionCompatibility:
    version: str
    supported: bool = True
    min_version: str = MIN_HA_VERSION
    warnings: list[str] = field(default_factory=list)
    known_issue: Optional[str] = None
```

**Test Coverage:**
- `test_utils_version.py::test_version_below_minimum`
- `test_utils_version.py::test_known_issue_detection`
- `test_utils_version.py::test_version_comparison`

---

### P2 #27: Schema Version Detection
**Status:** ✅ Implemented

**What:** Detect database schema version and refuse operations on incompatible schemas.

**File:** `care/scanner/schema.py`

**Configuration:**
```python
MIN_SCHEMA_VERSION = 41  # states_meta table required

SCHEMA_VERSIONS: dict[int, str] = {
    38: "metadata_id added to states",
    41: "states_meta table created",
    43: "last_reported_ts added",
    44: "events refactored, event_types table",
    45: "states refactored, states_meta table",
    # ... up to 52+
}
```

**Detection Methods:**
1. `schema_changes` table (modern, HA 2021.7+)
2. `schema_version` table (legacy)
3. Table structure inference (very old)

**Output:**
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

**Test Coverage:**
- `test_scanner_schema.py::test_schema_version_detection`
- `test_scanner_schema.py::test_unsupported_schema_warning`
- `test_scanner_schema.py::test_missing_required_tables`
- Integration: `test_integration_ha.py::test_detects_real_schema_version`

---

## SAFETY CHECKLIST (Pre-Release)

Before any cleanup feature release:

- [ ] P0 #20: Backup created before cleanup?
- [ ] P0 #21: Dry-run is default?
- [ ] P0 #22: Transaction rollback on error?
- [ ] P0 #23: No cascade deletes in SQL?
- [ ] P1 #24: HA stopped check implemented?
- [ ] P1 #26: Purge window conflict detected?
- [ ] P2 #25: Version compatibility checked?
- [ ] P2 #27: Schema version validated?

---

## ERROR HANDLING PHILOSOPHY

### Fail-Safe Defaults
```python
# WRONG: Assume success
if cleanup_possible:
    do_cleanup()

# RIGHT: Assume failure, require explicit confirmation
if all_safeguards_passed and user_confirmed and execute_mode:
    do_cleanup()
else:
    log_why_blocked()
```

### Logging Policy
```python
# Log EVERY decision point
logger.info("Safeguard check: HA state = %s", ha_state)
logger.info("Safeguard check: DB locked = %s", is_locked)
logger.info("Safeguard check: In purge window = %s", in_purge)

# Log the decision
if blocked:
    logger.warning("Cleanup BLOCKED: %s", reason)
else:
    logger.info("Cleanup ALLOWED: all safeguards passed")
```

---

## RELATED SKILLS

| Skill | Description |
|-------|-------------|
| [SKILL_00_CARE_OVERVIEW.md](SKILL_00_CARE_OVERVIEW.md) | Product overview |
| [SKILL_03_CARE_TESTING.md](SKILL_03_CARE_TESTING.md) | Test coverage for safety |

---

*Generated: 2026-01-26*
