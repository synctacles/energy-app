# SYNCTACLES Care - Technical Design Document

**Product:** Home Assistant Maintenance & Security Add-on  
**Version:** 1.0  
**Date:** 2026-01-25  
**Status:** Ready for Implementation

---

## Related Documents

| Document | Locatie | Inhoud |
|----------|---------|--------|
| **SKILL_17_GO_TO_MARKET_V2.md** | `/mnt/project/` of handoff | Business model, pricing, funnels, messaging |
| **SYNCTACLES_CARE_ROADMAP_V2.md** | Handoff | 77 issues, epics, timeline, repo structuur |
| **HANDOFF_CC_CARE_REPO.md** | Handoff | Repo setup opdracht, labels, milestones |
| **SKILL_02_ARCHITECTURE.md** | `/mnt/project/` | Bestaande SYNCTACLES architectuur |
| **SKILL_03_CODING_STANDARDS.md** | `/mnt/project/` | Code conventies |

---

## 1. System Overview

### 1.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER'S HOME ASSISTANT                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────┐       ┌──────────────────────┐            │
│  │  SYNCTACLES Energy   │       │   SYNCTACLES Care    │            │
│  │    (Integration)     │       │      (Add-on)        │            │
│  │                      │       │                      │            │
│  │  - Sensors           │       │  - Web UI (ingress)  │            │
│  │  - Binary sensors    │       │  - Scanner           │            │
│  │  - Services          │       │  - Cleaner           │            │
│  └──────────┬───────────┘       └──────────┬───────────┘            │
│             │                              │                         │
│             │    ┌─────────────────────────┼─────────────┐          │
│             │    │                         │             │          │
│             │    ▼                         ▼             ▼          │
│             │  ┌─────────┐  ┌──────────────────┐  ┌───────────┐    │
│             │  │ HA API  │  │ home-assistant   │  │ Supervisor │    │
│             │  │         │  │ _v2.db           │  │    API     │    │
│             │  └─────────┘  └──────────────────┘  └───────────┘    │
│             │                                                       │
└─────────────┼───────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      SYNCTACLES BACKEND                              │
│                    api.synctacles.com                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐     │
│  │   /api/auth/    │  │  /api/energy/   │  │   /api/care/    │     │
│  │                 │  │                 │  │                 │     │
│  │  - register     │  │  - prices       │  │  - authorize    │     │
│  │  - start-trial  │  │  - actions      │  │  - cleanup/*    │     │
│  │  - status       │  │  - best-window  │  │                 │     │
│  │  - validate     │  │                 │  │                 │     │
│  └────────┬────────┘  └─────────────────┘  └────────┬────────┘     │
│           │                                          │              │
│           └──────────────────┬───────────────────────┘              │
│                              ▼                                       │
│                    ┌─────────────────┐                              │
│                    │   PostgreSQL    │                              │
│                    │  subscriptions  │                              │
│                    └─────────────────┘                              │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Data Flow

```
SCAN FLOW (Gratis):
┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐
│  User  │───▶│ Web UI │───▶│Scanner │───▶│ HA DB  │
│        │    │        │    │        │◀───│        │
│        │◀───│ Report │◀───│ Scores │    │        │
└────────┘    └────────┘    └────────┘    └────────┘

CLEANUP FLOW (Premium):
┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐
│  User  │───▶│ Web UI │───▶│Backend │───▶│Add-on  │───▶│ HA DB  │
│        │    │        │    │ auth   │    │cleaner │    │        │
│        │    │        │◀───│premium?│    │        │    │        │
│        │    │        │    └────────┘    │        │    │        │
│        │    │        │         │        │        │    │        │
│        │    │        │    ┌────┴────┐   │        │    │        │
│        │    │        │    │   NO    │   │        │    │        │
│        │    │ LOCKED │◀───│         │   │        │    │        │
│        │◀───│        │    └─────────┘   │        │    │        │
│        │    │        │                  │        │    │        │
│        │    │        │    ┌────┴────┐   │        │    │        │
│        │    │        │    │   YES   │   │        │    │        │
│        │    │        │───▶│         │──▶│safegrd │───▶│backup  │
│        │    │        │    └─────────┘   │        │    │        │
│        │    │        │                  │cleanup │───▶│DELETE  │
│        │◀───│SUCCESS │◀─────────────────│        │    │        │
└────────┘    └────────┘                  └────────┘    └────────┘
```

### 1.3 Technology Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Add-on Runtime | Python 3.11+ | HA standard, async support |
| Web Framework | aiohttp | Lightweight, async native |
| Frontend | Alpine.js + Tailwind | No build step, small footprint |
| Database Access | sqlite3 | Native Python, HA uses SQLite |
| HA Communication | REST API | Supervisor API voor backups |
| Backend Comms | aiohttp client | Async HTTP naar api.synctacles.com |

---

## 2. Data Model

### 2.1 Backend Database (PostgreSQL)

**Bestaande `subscriptions` tabel - uitbreiding:**

```sql
-- Nieuwe kolommen toevoegen aan bestaande tabel
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    tier VARCHAR(20) DEFAULT 'free'
    CHECK (tier IN ('free', 'trial', 'premium'));

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    features JSONB DEFAULT '{"energy": true, "care": false}';

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    trial_started_at TIMESTAMP;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    trial_ends_at TIMESTAMP;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    care_enabled BOOLEAN DEFAULT false;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    care_last_scan_at TIMESTAMP;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    care_last_cleanup_at TIMESTAMP;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    care_cleanup_count INTEGER DEFAULT 0;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    care_total_bytes_cleaned BIGINT DEFAULT 0;

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS
    country VARCHAR(2) DEFAULT 'NL';

-- Index voor trial expiry checks
CREATE INDEX IF NOT EXISTS idx_subscriptions_trial_ends 
ON subscriptions(trial_ends_at) 
WHERE tier = 'trial';
```

**Features JSONB structuur:**

```json
{
  "energy": true,
  "energy_actions": false,
  "energy_eu": false,
  "care": true,
  "care_cleanup": false,
  "care_scheduled": false
}
```

### 2.2 Add-on Local State

De add-on slaat GEEN state lokaal op. Alles is:
- Realtime uit HA database (health/security data)
- Cached van backend (tier/features)
- Transient (scan results in memory)

**Uitzondering:** Config via HA add-on options:

```yaml
# /data/options.json (managed by HA)
{
  "api_key": "syn_xxxxxxxxxxxx"
}
```

### 2.3 HA Database Tables (Read)

**Relevante tabellen voor scanning:**

```sql
-- Statistics metadata (orphan detection)
SELECT 
    id,
    statistic_id,
    source,
    unit_of_measurement,
    created,
    last_reset
FROM statistics_meta
WHERE statistic_id NOT IN (
    SELECT DISTINCT entity_id FROM states
);

-- Entity registry (via .storage/core.entity_registry)
-- JSON file, niet SQL

-- Database size
SELECT 
    page_count * page_size as size_bytes
FROM pragma_page_count(), pragma_page_size();

-- Fragmentation
SELECT 
    (page_count - freelist_count) * page_size as used_bytes,
    freelist_count * page_size as free_bytes
FROM pragma_page_count(), pragma_freelist_count(), pragma_page_size();
```

### 2.4 API Contracts

#### POST /api/auth/register

**Request:**
```json
{
  "email": "user@example.com",
  "start_trial": true,
  "ha_installation_id": "abc123..."
}
```

**Response:**
```json
{
  "api_key": "syn_xxxxxxxxxxxx",
  "tier": "trial",
  "features": {
    "energy": true,
    "energy_actions": true,
    "energy_eu": false,
    "care": true,
    "care_cleanup": false,
    "care_scheduled": false
  },
  "trial_ends_at": "2026-02-08T12:00:00Z"
}
```

#### GET /api/auth/status

**Request Headers:**
```
X-API-Key: syn_xxxxxxxxxxxx
```

**Response:**
```json
{
  "tier": "trial",
  "features": {
    "energy": true,
    "energy_actions": true,
    "energy_eu": false,
    "care": true,
    "care_cleanup": false,
    "care_scheduled": false
  },
  "trial_ends_at": "2026-02-08T12:00:00Z",
  "trial_days_remaining": 7,
  "care_stats": {
    "last_scan_at": "2026-01-25T10:00:00Z",
    "last_cleanup_at": null,
    "cleanup_count": 0,
    "total_bytes_cleaned": 0
  }
}
```

#### POST /api/care/authorize

**Request:**
```json
{
  "action": "cleanup",
  "dry_run": false,
  "orphan_count": 247,
  "estimated_bytes": 524288000
}
```

**Response (Premium):**
```json
{
  "authorized": true,
  "cleanup_token": "clnp_xxxxxxxxxxxx",
  "expires_at": "2026-01-25T12:30:00Z",
  "cooldown_until": null
}
```

**Response (Not Premium):**
```json
{
  "authorized": false,
  "reason": "premium_required",
  "upgrade_url": "https://synctacles.com/premium"
}
```

**Response (Cooldown):**
```json
{
  "authorized": false,
  "reason": "cooldown",
  "cooldown_until": "2026-01-26T10:00:00Z"
}
```

#### POST /api/care/cleanup/complete

**Request:**
```json
{
  "cleanup_token": "clnp_xxxxxxxxxxxx",
  "success": true,
  "statistics_deleted": 247,
  "entities_deleted": 12,
  "bytes_freed": 524288000,
  "duration_seconds": 45,
  "ha_version": "2024.1.0",
  "backup_created": true
}
```

**Response:**
```json
{
  "recorded": true,
  "total_cleanups": 1,
  "total_bytes_cleaned": 524288000
}
```

---

## 3. Feature Specifications

### 3.1 Health Scanner

**Intentie:** Geef gebruiker inzicht in database gezondheid zonder actie te vereisen.

**Gratis feature** - geen restricties.

#### Metrics

| Metric | Hoe | Threshold |
|--------|-----|-----------|
| Database size | `pragma_page_count * page_size` | Warning >1GB, Critical >5GB |
| Fragmentation | `freelist_count / page_count * 100` | Warning >20%, Critical >40% |
| Orphaned statistics | Query statistics_meta vs states | Warning >100, Critical >500 |
| Orphaned entities | Parse entity_registry vs states | Warning >50, Critical >200 |
| Data age | `MIN(last_updated)` from states | Info only |
| Growth rate | Compare size over 7 days | Info only (V1.1) |

#### Health Score Algorithm

```python
def calculate_health_score(metrics: HealthMetrics) -> str:
    """
    Returns: A, B, C, D, or F
    """
    score = 100
    
    # Database size penalties
    if metrics.db_size_gb > 5:
        score -= 30
    elif metrics.db_size_gb > 2:
        score -= 15
    elif metrics.db_size_gb > 1:
        score -= 5
    
    # Fragmentation penalties
    if metrics.fragmentation_pct > 40:
        score -= 25
    elif metrics.fragmentation_pct > 20:
        score -= 10
    elif metrics.fragmentation_pct > 10:
        score -= 5
    
    # Orphaned statistics penalties
    if metrics.orphaned_statistics > 500:
        score -= 25
    elif metrics.orphaned_statistics > 100:
        score -= 15
    elif metrics.orphaned_statistics > 50:
        score -= 5
    
    # Orphaned entities penalties
    if metrics.orphaned_entities > 200:
        score -= 20
    elif metrics.orphaned_entities > 50:
        score -= 10
    elif metrics.orphaned_entities > 20:
        score -= 5
    
    # Convert to grade
    if score >= 90:
        return "A"
    elif score >= 75:
        return "B"
    elif score >= 60:
        return "C"
    elif score >= 40:
        return "D"
    else:
        return "F"
```

#### Output Model

```python
@dataclass
class HealthReport:
    timestamp: datetime
    grade: str  # A-F
    score: int  # 0-100
    
    db_size_bytes: int
    db_size_human: str  # "1.2 GB"
    fragmentation_pct: float
    
    orphaned_statistics_count: int
    orphaned_statistics_bytes_est: int
    orphaned_statistics_list: list[OrphanedStatistic]  # First 100
    
    orphaned_entities_count: int
    orphaned_entities_list: list[OrphanedEntity]  # First 100
    
    oldest_data: datetime
    newest_data: datetime
    
    recommendations: list[Recommendation]
```

---

### 3.2 Security Scanner

**Intentie:** Eerste HA security audit tool. Geef actionable score.

**Gratis feature** - geen restricties.

#### Checks

| Check | Hoe | Weight | Fix |
|-------|-----|--------|-----|
| 2FA enabled | Parse `auth_providers` in config | 25 | Link naar HA 2FA docs |
| HTTPS/SSL | Check `http` config + cert validity | 20 | Link naar SSL setup |
| HA version | Compare met latest via API | 15 | "Update beschikbaar" |
| Add-ons outdated | Supervisor API versions | 15 | Per add-on update link |
| Trusted networks | Parse `trusted_networks` config | 10 | Warning als te breed |
| IP ban enabled | Check `ip_ban_enabled` | 10 | Recommend enabling |
| Login attempts | Check `login_attempts_threshold` | 5 | Recommend lowering |

#### Security Score Algorithm

```python
def calculate_security_score(checks: list[SecurityCheck]) -> int:
    """
    Returns: 0-100
    """
    total_weight = sum(c.weight for c in checks)
    earned = sum(c.weight for c in checks if c.passed)
    
    return int((earned / total_weight) * 100)
```

#### Check Implementation Details

**2FA Check:**
```python
async def check_2fa(self) -> SecurityCheck:
    """Check if 2FA is enabled."""
    # Read /config/.storage/auth_providers
    auth_file = self.config_path / ".storage" / "auth_providers"
    
    if not auth_file.exists():
        return SecurityCheck(
            name="2fa_enabled",
            passed=False,
            severity="high",
            message="Kan auth configuratie niet lezen"
        )
    
    data = json.loads(auth_file.read_text())
    
    # Look for totp provider
    has_totp = any(
        p.get("type") == "totp" 
        for p in data.get("data", {}).get("providers", [])
    )
    
    return SecurityCheck(
        name="2fa_enabled",
        passed=has_totp,
        weight=25,
        severity="high" if not has_totp else None,
        message="2FA is niet ingeschakeld" if not has_totp else "2FA actief",
        fix_url="https://www.home-assistant.io/docs/authentication/multi-factor-auth/"
    )
```

**HA Version Check:**
```python
async def check_ha_version(self) -> SecurityCheck:
    """Check if HA is up to date."""
    # Get current version from Supervisor API
    current = await self.supervisor.get_core_info()
    current_version = current["version"]
    
    # Get latest from HA API or hardcoded
    # Note: We cache this, don't hit on every scan
    latest = await self.get_latest_ha_version()
    
    is_current = version.parse(current_version) >= version.parse(latest)
    
    return SecurityCheck(
        name="ha_version",
        passed=is_current,
        weight=15,
        severity="medium" if not is_current else None,
        message=f"HA {current_version}" + (f" (latest: {latest})" if not is_current else " (up to date)"),
        fix_url="https://www.home-assistant.io/common-tasks/os/#updating-home-assistant"
    )
```

#### Output Model

```python
@dataclass
class SecurityReport:
    timestamp: datetime
    score: int  # 0-100
    
    checks: list[SecurityCheck]
    
    critical_issues: list[SecurityCheck]  # severity=critical
    high_issues: list[SecurityCheck]      # severity=high
    medium_issues: list[SecurityCheck]    # severity=medium
    
    recommendations: list[Recommendation]

@dataclass
class SecurityCheck:
    name: str
    passed: bool
    weight: int
    severity: Optional[str]  # critical, high, medium, low
    message: str
    fix_url: Optional[str]
    details: Optional[dict]
```

---

### 3.3 Cleanup Engine

**Intentie:** Veilig opruimen van orphaned data. Dit is DE premium feature.

**Premium only** - NOOIT in trial.

#### Safeguards (KRITIEK)

| Safeguard | Implementatie | Failure = |
|-----------|---------------|-----------|
| **Backup verplicht** | Check via Supervisor API dat backup <1h oud | BLOCK cleanup |
| **Ruimte check** | `df -h /config`, need 2x cleanup size free | BLOCK cleanup |
| **HA versie whitelist** | Alleen geteste versies (2024.1+) | BLOCK cleanup |
| **Cooldown 24h** | Server-side check last cleanup | BLOCK cleanup |
| **Max 1000 items** | Batch limit per cleanup | Partial cleanup |
| **Dry-run first** | Altijd simulatie tonen | User confirms |
| **Transaction** | SQLite transaction, rollback on error | No partial state |
| **Verify integrity** | Post-cleanup PRAGMA integrity_check | Report issues |

#### Pre-flight Check Flow

```python
async def preflight_check(self) -> PreflightResult:
    """Run all checks before cleanup is allowed."""
    
    checks = []
    
    # 1. Premium check (via backend)
    auth = await self.backend.authorize_cleanup()
    if not auth.authorized:
        return PreflightResult(
            can_proceed=False,
            blocker=auth.reason,
            checks=[]
        )
    
    # 2. Backup check
    backups = await self.supervisor.list_backups()
    recent_backup = self._find_recent_backup(backups, max_age_hours=1)
    checks.append(PreflightCheck(
        name="backup",
        passed=recent_backup is not None,
        required=True,
        message="Backup vereist" if not recent_backup else f"Backup: {recent_backup.name}"
    ))
    
    # 3. Disk space check
    cleanup_estimate = self.estimate_cleanup_size()
    free_space = self.get_free_space()
    needs_space = cleanup_estimate * 2  # 2x safety margin
    checks.append(PreflightCheck(
        name="disk_space",
        passed=free_space > needs_space,
        required=True,
        message=f"Nodig: {needs_space}MB, Vrij: {free_space}MB"
    ))
    
    # 4. HA version check
    ha_version = await self.supervisor.get_core_version()
    is_supported = self._is_version_supported(ha_version)
    checks.append(PreflightCheck(
        name="ha_version",
        passed=is_supported,
        required=True,
        message=f"HA {ha_version}" + ("" if is_supported else " (niet ondersteund)")
    ))
    
    # 5. Cooldown check (via backend)
    if auth.cooldown_until:
        checks.append(PreflightCheck(
            name="cooldown",
            passed=False,
            required=True,
            message=f"Volgende cleanup mogelijk na {auth.cooldown_until}"
        ))
    
    can_proceed = all(c.passed for c in checks if c.required)
    
    return PreflightResult(
        can_proceed=can_proceed,
        blocker=next((c.name for c in checks if c.required and not c.passed), None),
        checks=checks,
        cleanup_token=auth.cleanup_token if can_proceed else None
    )
```

#### Cleanup Execution

```python
async def execute_cleanup(
    self,
    cleanup_token: str,
    dry_run: bool = False
) -> CleanupResult:
    """Execute the actual cleanup."""
    
    start_time = time.time()
    
    # Connect to database
    db_path = self.config_path / "home-assistant_v2.db"
    conn = sqlite3.connect(str(db_path))
    
    try:
        # Start transaction
        conn.execute("BEGIN TRANSACTION")
        
        # Get orphaned statistics IDs
        orphans = self._get_orphaned_statistics(conn)
        
        if dry_run:
            conn.execute("ROLLBACK")
            return CleanupResult(
                dry_run=True,
                would_delete_statistics=len(orphans),
                would_free_bytes=self._estimate_size(orphans)
            )
        
        # Delete in batches
        deleted_count = 0
        for batch in self._batch(orphans, size=100):
            ids = [o.id for o in batch]
            
            # Delete from statistics first (foreign key)
            conn.execute(
                "DELETE FROM statistics WHERE metadata_id IN ({})".format(
                    ",".join("?" * len(ids))
                ),
                ids
            )
            
            # Delete from statistics_meta
            conn.execute(
                "DELETE FROM statistics_meta WHERE id IN ({})".format(
                    ",".join("?" * len(ids))
                ),
                ids
            )
            
            deleted_count += len(batch)
        
        # VACUUM to reclaim space
        conn.execute("COMMIT")
        conn.execute("VACUUM")
        
        # Verify integrity
        integrity = conn.execute("PRAGMA integrity_check").fetchone()[0]
        
        duration = time.time() - start_time
        
        # Report to backend
        await self.backend.report_cleanup_complete(
            cleanup_token=cleanup_token,
            success=integrity == "ok",
            statistics_deleted=deleted_count,
            duration_seconds=duration
        )
        
        return CleanupResult(
            dry_run=False,
            success=integrity == "ok",
            statistics_deleted=deleted_count,
            bytes_freed=self._calculate_freed_space(),
            duration_seconds=duration,
            integrity_check=integrity
        )
        
    except Exception as e:
        conn.execute("ROLLBACK")
        
        await self.backend.report_cleanup_complete(
            cleanup_token=cleanup_token,
            success=False,
            error=str(e)
        )
        
        raise CleanupError(f"Cleanup failed: {e}")
        
    finally:
        conn.close()
```

#### Supported HA Versions

```python
SUPPORTED_HA_VERSIONS = {
    "min": "2024.1.0",
    "max": None,  # No upper limit yet
    "tested": [
        "2024.1.0", "2024.1.1", "2024.1.2",
        "2024.2.0", "2024.2.1",
        # ... add as tested
    ]
}

def _is_version_supported(self, version_str: str) -> bool:
    v = version.parse(version_str)
    min_v = version.parse(SUPPORTED_HA_VERSIONS["min"])
    
    if v < min_v:
        return False
    
    if SUPPORTED_HA_VERSIONS["max"]:
        max_v = version.parse(SUPPORTED_HA_VERSIONS["max"])
        if v > max_v:
            return False
    
    return True
```

---

### 3.4 Backup Lifecycle

**Intentie:** Gebruiker hoeft niet na te denken over backups. Wij managen het.

**Premium only.**

#### Lifecycle

```
PRE-CLEANUP:
├── Check: Backup <1h oud?
│   ├── JA → Gebruik bestaande, ga door
│   └── NEE → Maak nieuwe backup
│       ├── Tag: "synctacles_care_YYYYMMDD_HHMMSS"
│       └── Wacht tot compleet
│
POST-CLEANUP:
├── Verificatie: Alles OK?
│   ├── JA → Cleanup backup later
│   └── NEE → BEHOUD backup, warn user
│
DAG 7:
├── Notification: "Cleanup 7 dagen geleden. Alles OK?"
│   ├── User confirms → Backup mag weg
│   └── User silent → Wacht
│
DAG 30 (of max 2 Care backups):
├── Auto-delete oudste Care backup
└── Notification: "Oude Care backup verwijderd"
```

#### Implementation

```python
class BackupManager:
    CARE_BACKUP_PREFIX = "synctacles_care_"
    MAX_CARE_BACKUPS = 2
    MIN_RETENTION_DAYS = 7
    AUTO_DELETE_DAYS = 30
    
    async def ensure_backup_exists(self) -> Backup:
        """Ensure a recent backup exists, create if needed."""
        
        backups = await self.supervisor.list_backups()
        
        # Find recent Care backup or any backup
        recent = self._find_recent_backup(backups, max_age_hours=1)
        
        if recent:
            return recent
        
        # Create new backup
        name = f"{self.CARE_BACKUP_PREFIX}{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        
        backup = await self.supervisor.create_backup(
            name=name,
            folders=["homeassistant"],  # Only HA config, not addons
            compressed=True
        )
        
        return backup
    
    async def cleanup_old_backups(self) -> list[str]:
        """Remove old Care backups, respecting limits."""
        
        backups = await self.supervisor.list_backups()
        care_backups = [
            b for b in backups 
            if b.name.startswith(self.CARE_BACKUP_PREFIX)
        ]
        
        # Sort by date, oldest first
        care_backups.sort(key=lambda b: b.date)
        
        deleted = []
        
        # Delete if over max count
        while len(care_backups) > self.MAX_CARE_BACKUPS:
            oldest = care_backups.pop(0)
            if self._days_old(oldest) >= self.MIN_RETENTION_DAYS:
                await self.supervisor.delete_backup(oldest.slug)
                deleted.append(oldest.name)
        
        # Delete if over auto-delete age
        for backup in care_backups[:]:
            if self._days_old(backup) >= self.AUTO_DELETE_DAYS:
                await self.supervisor.delete_backup(backup.slug)
                deleted.append(backup.name)
                care_backups.remove(backup)
        
        return deleted
```

---

### 3.5 Trial & Premium Logic

**Intentie:** Frictionless trial, duidelijke value prop, smooth upgrade.

#### Feature Access Matrix

```python
FEATURE_ACCESS = {
    "free": {
        "energy_prices": True,
        "energy_hours": True,
        "energy_actions": False,
        "energy_best_window": False,
        "energy_live_cost": False,
        "energy_tomorrow": False,
        "energy_eu": False,
        "care_scan": True,
        "care_health_score": True,
        "care_security_score": True,
        "care_orphan_view": True,
        "care_cleanup": False,
        "care_scheduled": False,
        "care_backup_management": False,
    },
    "trial": {
        "energy_prices": True,
        "energy_hours": True,
        "energy_actions": True,        # UNLOCKED
        "energy_best_window": True,    # UNLOCKED
        "energy_live_cost": True,      # UNLOCKED
        "energy_tomorrow": True,       # UNLOCKED
        "energy_eu": False,            # Still locked
        "care_scan": True,
        "care_health_score": True,
        "care_security_score": True,
        "care_orphan_view": True,
        "care_cleanup": False,         # NEVER IN TRIAL
        "care_scheduled": False,       # NEVER IN TRIAL
        "care_backup_management": False,
    },
    "premium": {
        "energy_prices": True,
        "energy_hours": True,
        "energy_actions": True,
        "energy_best_window": True,
        "energy_live_cost": True,
        "energy_tomorrow": True,
        "energy_eu": True,             # UNLOCKED
        "care_scan": True,
        "care_health_score": True,
        "care_security_score": True,
        "care_orphan_view": True,
        "care_cleanup": True,          # UNLOCKED
        "care_scheduled": True,        # UNLOCKED
        "care_backup_management": True,
    },
}
```

#### Feature Check Logic

```python
class FeatureGate:
    def __init__(self, subscription: Subscription):
        self.sub = subscription
    
    def can_access(self, feature: str) -> bool:
        """Check if current subscription can access feature."""
        
        tier = self._effective_tier()
        return FEATURE_ACCESS.get(tier, {}).get(feature, False)
    
    def _effective_tier(self) -> str:
        """Get effective tier considering trial expiry."""
        
        if self.sub.tier == "premium":
            return "premium"
        
        if self.sub.tier == "trial":
            if self.sub.trial_ends_at and self.sub.trial_ends_at > datetime.utcnow():
                return "trial"
            else:
                return "free"  # Trial expired
        
        return "free"
    
    def get_lock_reason(self, feature: str) -> Optional[LockReason]:
        """Get reason why feature is locked, if applicable."""
        
        if self.can_access(feature):
            return None
        
        tier = self._effective_tier()
        
        if tier == "free":
            if feature.startswith("energy_"):
                return LockReason(
                    type="trial_available",
                    message="Start 14-dagen trial om deze feature te unlocken",
                    cta="Start Trial"
                )
            else:
                return LockReason(
                    type="premium_required",
                    message="Upgrade naar Premium voor cleanup features",
                    cta="Upgrade €25/jaar"
                )
        
        if tier == "trial":
            # Trial users trying to access premium features
            return LockReason(
                type="premium_required",
                message="Deze feature is alleen beschikbaar voor Premium",
                cta="Upgrade €25/jaar"
            )
        
        return None
```

#### Trial Flow

```python
class TrialManager:
    TRIAL_DAYS = 14
    
    async def start_trial(self, email: str) -> TrialResult:
        """Start a new trial."""
        
        # Validate email
        if not self._is_valid_email(email):
            raise ValidationError("Ongeldig email adres")
        
        # Check if already has subscription
        existing = await self.backend.get_subscription_by_email(email)
        if existing:
            if existing.tier == "premium":
                raise ValidationError("Je hebt al een Premium account")
            if existing.tier == "trial" and existing.trial_ends_at > datetime.utcnow():
                raise ValidationError("Je hebt al een actieve trial")
        
        # Create trial via backend
        result = await self.backend.register(
            email=email,
            start_trial=True,
            ha_installation_id=self._get_installation_id()
        )
        
        # Store API key locally
        await self._save_api_key(result.api_key)
        
        return TrialResult(
            api_key=result.api_key,
            trial_ends_at=result.trial_ends_at,
            features=result.features
        )
    
    def get_trial_status(self) -> TrialStatus:
        """Get current trial status."""
        
        if self.sub.tier != "trial":
            return TrialStatus(active=False)
        
        now = datetime.utcnow()
        ends = self.sub.trial_ends_at
        
        if ends <= now:
            return TrialStatus(
                active=False,
                expired=True,
                expired_at=ends
            )
        
        days_remaining = (ends - now).days
        
        return TrialStatus(
            active=True,
            days_remaining=days_remaining,
            ends_at=ends,
            show_urgency=days_remaining <= 3
        )
```

---

## 4. UI/UX Flows

### 4.1 Dashboard (Main View)

```
┌─────────────────────────────────────────────────────────────────┐
│  SYNCTACLES Care                                    [⚙️ Settings]│
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────┐  ┌─────────────────────────────┐   │
│  │     HEALTH SCORE        │  │      SECURITY SCORE          │   │
│  │                         │  │                              │   │
│  │         ██████          │  │            72                │   │
│  │        ████████         │  │           /100               │   │
│  │       ██████████        │  │                              │   │
│  │          B              │  │      ████████░░░░            │   │
│  │                         │  │                              │   │
│  │  Last scan: 2h ago      │  │   2 issues gevonden          │   │
│  │                         │  │                              │   │
│  │  [View Details]         │  │   [View Details]             │   │
│  └─────────────────────────┘  └─────────────────────────────┘   │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  ⚠️  247 ORPHANED STATISTICS GEVONDEN                      │  │
│  │                                                            │  │
│  │  Geschatte grootte: ~500 MB                                │  │
│  │                                                            │  │
│  │  [Bekijk Details]                    [🔒 Cleanup Starten]  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  TRIAL: 7 dagen resterend              [Upgrade - €25/jr] │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Locked Feature Modal

```
┌─────────────────────────────────────────────────────────────────┐
│                                                           [X]   │
│                         🔒                                       │
│                                                                  │
│                  PREMIUM FEATURE                                 │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  Je hebt 247 orphaned statistics gevonden (~500 MB)             │
│                                                                  │
│  Upgrade naar Premium om ze veilig te verwijderen.              │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  ✓ One-click cleanup met backup                                 │
│  ✓ Scheduled maintenance                                        │
│  ✓ Energy insights voor heel Europa                             │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│               €25/jaar = €2.08/maand                            │
│                                                                  │
│           ┌─────────────────────────────┐                       │
│           │   Upgrade naar Premium      │                       │
│           └─────────────────────────────┘                       │
│                                                                  │
│                    Misschien later                               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 Cleanup Wizard

**Step 1: Pre-flight**

```
┌─────────────────────────────────────────────────────────────────┐
│  CLEANUP WIZARD                                        Step 1/4 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Pre-flight Checks                                               │
│                                                                  │
│  ✅ Premium account actief                                       │
│  ✅ Backup beschikbaar (synctacles_care_20260125_120000)        │
│  ✅ Voldoende schijfruimte (2.1 GB vrij)                        │
│  ✅ HA versie ondersteund (2024.1.2)                            │
│  ✅ Geen cooldown actief                                         │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│           ┌─────────────────────────────┐                       │
│           │       Volgende →            │                       │
│           └─────────────────────────────┘                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Step 2: Dry Run**

```
┌─────────────────────────────────────────────────────────────────┐
│  CLEANUP WIZARD                                        Step 2/4 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Wat wordt opgeruimd?                                           │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Type                          │ Aantal    │ Geschat       │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ Orphaned statistics           │ 247       │ ~480 MB       │ │
│  │ Orphaned entities             │ 12        │ ~20 MB        │ │
│  │ Database fragmentation        │ 15%       │ ~50 MB        │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ TOTAAL                        │           │ ~550 MB       │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ⚠️  Dit is een simulatie. Er is nog niets verwijderd.          │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  [← Terug]              ┌─────────────────────────────┐         │
│                         │   Bevestig & Start →        │         │
│                         └─────────────────────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Step 3: Progress**

```
┌─────────────────────────────────────────────────────────────────┐
│  CLEANUP WIZARD                                        Step 3/4 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                    Cleanup bezig...                              │
│                                                                  │
│           ████████████████████░░░░░░░░░░  67%                   │
│                                                                  │
│  ✅ Backup geverifieerd                                          │
│  ✅ Statistics verwijderd (247/247)                              │
│  ⏳ Entities verwijderen (8/12)                                  │
│  ⬜ Database optimaliseren                                       │
│  ⬜ Integriteit controleren                                      │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  ⚠️  Sluit dit venster niet tijdens de cleanup                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Step 4: Complete**

```
┌─────────────────────────────────────────────────────────────────┐
│  CLEANUP WIZARD                                        Step 4/4 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                         🎉                                       │
│                                                                  │
│                  CLEANUP VOLTOOID!                               │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  Resultaten:                                                     │
│                                                                  │
│    • Statistics verwijderd: 247                                  │
│    • Entities verwijderd: 12                                     │
│    • Ruimte vrijgemaakt: 487 MB                                  │
│    • Duur: 45 seconden                                           │
│    • Database integriteit: ✅ OK                                 │
│                                                                  │
│  Je Health Score is nu: A (was: B)                              │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  💡 Je backup wordt 7 dagen bewaard. Als je problemen           │
│     ervaart, kan je deze herstellen via HA Settings.            │
│                                                                  │
│           ┌─────────────────────────────┐                       │
│           │     Terug naar Dashboard    │                       │
│           └─────────────────────────────┘                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.4 Trial Banner States

**Active Trial:**
```
┌───────────────────────────────────────────────────────────────┐
│  ⏱️ Trial: 12 dagen resterend                    [Meer info]   │
└───────────────────────────────────────────────────────────────┘
```

**Trial Ending Soon (≤3 days):**
```
┌───────────────────────────────────────────────────────────────┐
│  ⚠️ Trial eindigt over 2 dagen!           [Upgrade - €25/jr]  │
└───────────────────────────────────────────────────────────────┘
```

**Trial Expired:**
```
┌───────────────────────────────────────────────────────────────┐
│  ❌ Trial verlopen. Energy actions zijn nu gelocked.          │
│                                            [Upgrade - €25/jr]  │
└───────────────────────────────────────────────────────────────┘
```

---

## 5. Security & Safety

### 5.1 Wat Mag NOOIT Fout Gaan

| Scenario | Impact | Mitigatie |
|----------|--------|-----------|
| **Data loss door cleanup** | Kritiek | Verplichte backup, dry-run, transaction rollback |
| **Corrupt database** | Kritiek | Integrity check, tested HA versions only |
| **Cleanup tijdens HA restart** | Hoog | Lock file, check HA status before start |
| **Ongeautoriseerde cleanup** | Hoog | Server-side premium check met token |
| **API key leak** | Medium | Key only in options.json, never logged |
| **Infinite cleanup loop** | Medium | 24h cooldown, max 1000 items |

### 5.2 Error Handling

```python
class CleanupError(Exception):
    """Base cleanup error."""
    pass

class PreflightError(CleanupError):
    """Pre-flight check failed."""
    def __init__(self, check: str, message: str):
        self.check = check
        super().__init__(f"Pre-flight failed: {check} - {message}")

class BackupError(CleanupError):
    """Backup creation/verification failed."""
    pass

class DatabaseError(CleanupError):
    """Database operation failed."""
    pass

class AuthorizationError(CleanupError):
    """Backend authorization failed."""
    pass

# In cleanup execution:
try:
    result = await cleanup_engine.execute(...)
except PreflightError as e:
    # Show specific check that failed
    ui.show_error(f"Kan niet starten: {e.message}")
except BackupError as e:
    # Critical - don't proceed
    ui.show_error("Backup mislukt. Cleanup afgebroken.")
except DatabaseError as e:
    # Rollback happened
    ui.show_error(f"Database fout: {e}. Rollback uitgevoerd.")
except AuthorizationError as e:
    # Not premium
    ui.show_premium_modal()
```

### 5.3 Logging

```python
# Wat WEL loggen:
logger.info("Cleanup started", extra={
    "orphan_count": 247,
    "ha_version": "2024.1.2"
})
logger.info("Cleanup completed", extra={
    "deleted": 247,
    "duration_s": 45,
    "bytes_freed": 524288000
})
logger.error("Cleanup failed", extra={
    "error": str(e),
    "rollback": True
})

# Wat NIET loggen:
# - API keys
# - Email addresses
# - Specific entity names (privacy)
```

---

## 6. API Rate Limits

### 6.1 Care Add-on → Backend

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/api/auth/status` | 60 | per uur |
| `/api/care/authorize` | 10 | per uur |
| `/api/care/cleanup/*` | 5 | per dag |

### 6.2 Caching Strategy

```python
class BackendClient:
    # Cache subscription status
    _status_cache: Optional[SubscriptionStatus] = None
    _status_cached_at: Optional[datetime] = None
    STATUS_CACHE_TTL = timedelta(minutes=15)
    
    async def get_status(self, force: bool = False) -> SubscriptionStatus:
        """Get subscription status with caching."""
        
        if not force and self._status_cache and self._status_cached_at:
            age = datetime.utcnow() - self._status_cached_at
            if age < self.STATUS_CACHE_TTL:
                return self._status_cache
        
        status = await self._fetch_status()
        self._status_cache = status
        self._status_cached_at = datetime.utcnow()
        
        return status
```

---

## 7. Testing Strategy

### 7.1 Unit Tests

| Module | Coverage Target | Focus |
|--------|-----------------|-------|
| scanner/health.py | 90% | Score calculation, edge cases |
| scanner/security.py | 90% | All check types |
| cleaner/orphans.py | 95% | SQL queries, batch logic |
| cleaner/safeguards.py | 95% | All preflight checks |
| api/client.py | 80% | Error handling, caching |

### 7.2 Integration Tests

| Scenario | Setup | Verify |
|----------|-------|--------|
| Full scan flow | Mock HA DB with known state | Correct scores |
| Full cleanup flow | Test DB, mock backend | Data deleted, integrity OK |
| Trial flow | Mock backend | Features locked/unlocked |
| Backup flow | Mock Supervisor API | Backup created/listed |

### 7.3 Edge Cases

| Case | Expected Behavior |
|------|-------------------|
| Empty database | Health A, "Geen orphans" |
| 10K+ orphans | Batch delete, progress updates |
| No network | Graceful degradation, cached status |
| HA restarting | Block cleanup, retry later |
| Disk full | Pre-flight fails, clear message |
| Corrupt DB | Integrity check fails, no cleanup |

---

## 8. Deployment

### 8.1 Add-on Versioning

```
MAJOR.MINOR.PATCH

1.0.0 - Initial release
1.0.1 - Bug fixes
1.1.0 - New feature (scheduled scans)
2.0.0 - Breaking change
```

### 8.2 Release Process

1. Create release branch `release/1.0.0`
2. Update version in `config.yaml`
3. Update `CHANGELOG.md`
4. Create GitHub release with tag
5. CI builds multi-arch images
6. Push to add-on repository

### 8.3 Multi-arch Builds

```yaml
# .github/workflows/release.yml
jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        arch: [aarch64, amd64, armhf, armv7, i386]
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3
      - uses: docker/build-push-action@v5
        with:
          context: ./synctacles-care
          platforms: linux/${{ matrix.arch }}
          push: true
          tags: ghcr.io/synctacles/synctacles-care-${{ matrix.arch }}:${{ github.ref_name }}
```

---

## 9. Glossary

| Term | Betekenis |
|------|-----------|
| **Orphaned statistic** | Statistics entry zonder bijbehorende entity |
| **Orphaned entity** | Entity in registry zonder state data |
| **Health Score** | A-F grade gebaseerd op DB gezondheid |
| **Security Score** | 0-100 gebaseerd op security checks |
| **Pre-flight** | Checks voor cleanup mag starten |
| **Dry-run** | Simulatie zonder daadwerkelijke wijzigingen |
| **Cooldown** | Wachttijd tussen cleanups (24h) |
| **Care backup** | Backup gemaakt door Care add-on |

---

## 10. Open Questions (voor CC)

1. **Supervisor API authenticatie:** Hoe krijgen we token? Via `SUPERVISOR_TOKEN` env var?
2. **Entity registry format:** Is dit stabiel tussen HA versies?
3. **Multi-arch testing:** Hoe testen we ARM builds?
4. **Ingress session:** Hoe lang blijft sessie actief?
5. **Backup size estimation:** Hoe schatten we DB backup grootte?

---

*Document Version: 1.0*  
*Last Updated: 2026-01-25*  
*Ready for: CC Implementation*
