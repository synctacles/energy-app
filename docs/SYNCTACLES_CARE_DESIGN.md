# Synctacles Care - Technical Design Document

**Product:** Home Assistant Maintenance Add-on
**Version:** 1.0 Draft
**Date:** 2026-01-25
**Status:** Design Phase

---

## Executive Summary

Synctacles Care is a Home Assistant add-on that solves the universal problem of HA maintenance - orphaned entities, bloated databases, and accumulated cruft that every HA user experiences but few can fix.

### Value Proposition

| For Users | For Synctacles |
|-----------|----------------|
| One-click cleanup | Unique market position |
| No technical knowledge needed | Recurring revenue stream |
| Scheduled maintenance | Brand awareness |
| Peace of mind | Funnel to Energy product |

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     USER'S HOME ASSISTANT                        │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Synctacles Care Add-on                       │   │
│  │                                                           │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │   │
│  │  │   Scanner   │  │   Cleaner   │  │   Scheduler     │   │   │
│  │  │   (Local)   │  │ (API-gated) │  │   (API-gated)   │   │   │
│  │  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘   │   │
│  │         │                │                   │            │   │
│  │         ▼                ▼                   ▼            │   │
│  │  ┌─────────────────────────────────────────────────────┐ │   │
│  │  │              Care API Client                         │ │   │
│  │  └─────────────────────────┬───────────────────────────┘ │   │
│  │                            │                              │   │
│  └────────────────────────────┼──────────────────────────────┘   │
│                               │                                   │
│  ┌────────────────┐           │          ┌───────────────────┐   │
│  │ Entity Registry│◄──────────┤          │ HA Database       │   │
│  │ (.storage/)    │           │          │ (home-assistant   │   │
│  └────────────────┘           │          │  _v2.db)          │   │
│                               │          └───────────────────┘   │
└───────────────────────────────┼───────────────────────────────────┘
                                │
                                │ HTTPS
                                ▼
┌───────────────────────────────────────────────────────────────────┐
│                    SYNCTACLES BACKEND                              │
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                     Care API                                 │  │
│  │                                                              │  │
│  │  POST /api/care/authorize    - Validate subscription        │  │
│  │  POST /api/care/cleanup      - Get cleanup commands         │  │
│  │  POST /api/care/backup       - Cloud backup upload          │  │
│  │  GET  /api/care/history      - Cleanup history              │  │
│  │  POST /api/care/schedule     - Set maintenance schedule     │  │
│  │                                                              │  │
│  └──────────────────────────────┬──────────────────────────────┘  │
│                                 │                                  │
│  ┌──────────────────────────────▼──────────────────────────────┐  │
│  │                 Subscription Service                         │  │
│  │                                                              │  │
│  │  - API Key validation                                        │  │
│  │  - Tier checking (free/basic/premium)                        │  │
│  │  - Usage tracking                                            │  │
│  │  - Analytics                                                 │  │
│  │                                                              │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

---

## 2. Feature Specification

### 2.1 Free Tier Features (Local)

These run entirely on the user's HA instance, no API calls needed.

#### Scan Engine

```python
class CareScanner:
    """Scans HA for maintenance issues."""

    def scan_orphaned_entities(self) -> ScanResult:
        """Find entities with orphaned_timestamp in registry."""

    def scan_orphaned_statistics(self) -> ScanResult:
        """Find statistics_meta entries without matching entities."""

    def scan_database_health(self) -> DatabaseHealth:
        """Check database size, fragmentation, age of data."""

    def scan_unused_entities(self) -> ScanResult:
        """Find entities not used in automations/dashboards."""

    def generate_report(self) -> CareReport:
        """Generate human-readable maintenance report."""
```

#### Report Output

```json
{
  "scan_timestamp": "2026-01-25T12:00:00Z",
  "ha_version": "2025.1.0",
  "issues": {
    "orphaned_entities": {
      "count": 12,
      "severity": "medium",
      "items": [
        {"entity_id": "sensor.old_device", "platform": "mqtt", "orphaned_days": 30}
      ]
    },
    "orphaned_statistics": {
      "count": 47,
      "severity": "low",
      "items": [
        {"statistic_id": "sensor.energy_insights_nl_price", "rows": 15000}
      ]
    },
    "database_health": {
      "size_mb": 2300,
      "recommended_max_mb": 1000,
      "oldest_data_days": 365,
      "severity": "high"
    }
  },
  "recommendations": [
    {
      "action": "clean_orphaned_statistics",
      "impact": "Recover ~500MB disk space",
      "risk": "low"
    }
  ]
}
```

### 2.2 Premium Features (API-Gated)

#### One-Click Cleanup

```python
class CareCleaner:
    """Performs cleanup operations (requires premium)."""

    async def authorize(self, api_key: str) -> AuthResult:
        """Validate API key with Synctacles backend."""

    async def cleanup_orphaned_statistics(
        self,
        dry_run: bool = True
    ) -> CleanupResult:
        """Remove orphaned statistics from database."""

    async def cleanup_orphaned_entities(
        self,
        dry_run: bool = True
    ) -> CleanupResult:
        """Remove orphaned entities from registry."""

    async def optimize_database(self) -> OptimizeResult:
        """Run VACUUM and optimize database."""

    async def purge_old_data(
        self,
        keep_days: int = 30
    ) -> PurgeResult:
        """Purge data older than specified days."""
```

#### Scheduled Maintenance

```python
class CareScheduler:
    """Scheduled maintenance (requires premium)."""

    async def set_schedule(
        self,
        frequency: str,  # "daily", "weekly", "monthly"
        time: str,       # "03:00"
        actions: list    # ["orphaned_stats", "optimize_db"]
    ) -> Schedule:
        """Configure automatic maintenance schedule."""

    async def get_schedule(self) -> Schedule:
        """Get current maintenance schedule."""

    async def get_history(self) -> list[MaintenanceEvent]:
        """Get maintenance history from backend."""
```

#### Cloud Backup

```python
class CareBackup:
    """Backup service (requires premium)."""

    async def backup_before_cleanup(self) -> BackupResult:
        """Create backup before any destructive operation."""
        # Uploads to Synctacles cloud storage
        # Retained for 30 days

    async def list_backups(self) -> list[Backup]:
        """List available backups."""

    async def restore_backup(self, backup_id: str) -> RestoreResult:
        """Restore from backup."""
```

---

## 3. API Endpoints (Backend)

### 3.1 Authorization

```
POST /api/care/authorize
```

**Request:**
```json
{
  "api_key": "user_api_key_here",
  "ha_instance_id": "unique_ha_id",
  "requested_features": ["cleanup", "schedule", "backup"]
}
```

**Response:**
```json
{
  "authorized": true,
  "tier": "premium",
  "features": {
    "cleanup": true,
    "schedule": true,
    "backup": true,
    "max_backups": 5,
    "backup_retention_days": 30
  },
  "token": "temporary_action_token",
  "token_expires": "2026-01-25T13:00:00Z"
}
```

### 3.2 Cleanup Authorization

```
POST /api/care/cleanup
```

**Request:**
```json
{
  "token": "temporary_action_token",
  "action": "orphaned_statistics",
  "items_count": 47,
  "estimated_recovery_mb": 500
}
```

**Response:**
```json
{
  "approved": true,
  "cleanup_id": "cleanup_abc123",
  "backup_required": true,
  "backup_url": "https://api.synctacles.com/api/care/backup/upload"
}
```

### 3.3 Log Cleanup Event

```
POST /api/care/cleanup/{cleanup_id}/complete
```

**Request:**
```json
{
  "token": "temporary_action_token",
  "cleanup_id": "cleanup_abc123",
  "result": {
    "success": true,
    "items_removed": 47,
    "space_recovered_mb": 487,
    "duration_seconds": 12
  }
}
```

### 3.4 Schedule Management

```
POST /api/care/schedule
GET /api/care/schedule
DELETE /api/care/schedule
```

### 3.5 Backup Management

```
POST /api/care/backup/upload
GET /api/care/backup/list
GET /api/care/backup/{backup_id}/download
```

---

## 4. HA Add-on Structure

```
synctacles-care/
├── config.yaml              # Add-on configuration
├── Dockerfile               # Build configuration
├── run.sh                   # Startup script
│
├── care/
│   ├── __init__.py
│   ├── main.py              # Entry point
│   ├── scanner.py           # Scan functionality
│   ├── cleaner.py           # Cleanup functionality
│   ├── scheduler.py         # Scheduled tasks
│   ├── backup.py            # Backup functionality
│   ├── api_client.py        # Synctacles API client
│   │
│   ├── analyzers/
│   │   ├── entity_registry.py
│   │   ├── statistics.py
│   │   ├── database.py
│   │   └── automations.py
│   │
│   └── web/
│       ├── server.py        # Internal web server
│       ├── routes.py        # API routes
│       └── static/          # Web UI assets
│
├── rootfs/
│   └── etc/
│       └── services.d/
│           └── care/
│               └── run      # S6 service definition
│
└── translations/
    ├── en.yaml
    └── nl.yaml
```

### config.yaml

```yaml
name: "Synctacles Care"
description: "Home Assistant Maintenance Made Easy"
version: "1.0.0"
slug: "synctacles_care"
url: "https://github.com/synctacles/ha-care"
arch:
  - amd64
  - aarch64
  - armv7

startup: application
boot: auto

ports:
  5580/tcp: 5580
ports_description:
  5580/tcp: "Web UI"

options:
  api_key: ""
  auto_scan: true
  scan_interval_hours: 24

schema:
  api_key: str?
  auto_scan: bool
  scan_interval_hours: int(1,168)

ingress: true
ingress_port: 5580
panel_icon: "mdi:broom"
panel_title: "Synctacles Care"

map:
  - config:rw          # Access to HA config
  - ssl:ro             # Read SSL certs
  - share:rw           # Shared storage for backups
```

---

## 5. Lovelace Card Design

### 5.1 Summary Card

```yaml
type: custom:synctacles-care-card
title: System Health
show_last_scan: true
show_quick_actions: true
```

**Rendered:**

```
┌─────────────────────────────────────────────┐
│  🧹 Synctacles Care                         │
├─────────────────────────────────────────────┤
│                                             │
│  System Health: ⚠️ 3 Issues Found           │
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │ 📊 Orphaned Statistics     47       │    │
│  │ 👻 Orphaned Entities       12       │    │
│  │ 💾 Database Size          2.3 GB    │    │
│  └─────────────────────────────────────┘    │
│                                             │
│  Last scan: 2 hours ago                     │
│                                             │
│  [🔍 Scan Now]  [🧹 Clean All]             │
│                                             │
│  ────────────────────────────────────────   │
│  🔒 Premium features: Active                │
│  📅 Next scheduled: Tomorrow 03:00          │
│                                             │
└─────────────────────────────────────────────┘
```

### 5.2 Detailed Panel

```
┌─────────────────────────────────────────────────────────────────┐
│  🧹 Synctacles Care - Maintenance Dashboard                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─── Current Status ──────────────────────────────────────┐    │
│  │                                                          │    │
│  │  Overall Health:  ⚠️ Needs Attention                     │    │
│  │  Last Scan:       25 Jan 2026, 10:30                     │    │
│  │  Last Cleanup:    20 Jan 2026, 03:00                     │    │
│  │                                                          │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─── Issues Found ────────────────────────────────────────┐    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │ 📊 Orphaned Statistics                           │   │    │
│  │  │                                                  │   │    │
│  │  │ 47 statistics entries without matching entity    │   │    │
│  │  │ Est. recovery: 487 MB                            │   │    │
│  │  │                                                  │   │    │
│  │  │ Examples:                                        │   │    │
│  │  │  • sensor.energy_insights_nl_price (15,230 rows) │   │    │
│  │  │  • sensor.old_growatt_power (8,102 rows)         │   │    │
│  │  │  • sensor.deleted_device_temp (4,551 rows)       │   │    │
│  │  │                                                  │   │    │
│  │  │ [View All]                    [🧹 Clean Now]     │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │ 👻 Orphaned Entities                             │   │    │
│  │  │                                                  │   │    │
│  │  │ 12 entities no longer claimed by integration     │   │    │
│  │  │                                                  │   │    │
│  │  │ [View All]                    [🧹 Clean Now]     │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │ 💾 Database Health                               │   │    │
│  │  │                                                  │   │    │
│  │  │ Size: 2.3 GB (recommended: < 1 GB)               │   │    │
│  │  │ Oldest data: 365 days                            │   │    │
│  │  │ Fragmentation: 15%                               │   │    │
│  │  │                                                  │   │    │
│  │  │ [⚙️ Optimize]          [🗑️ Purge Old Data]       │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  │                                                          │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─── Schedule ────────────────────────────────────────────┐    │
│  │                                                          │    │
│  │  Automatic Maintenance: ✅ Enabled                       │    │
│  │  Frequency: Weekly (Sunday 03:00)                        │    │
│  │  Actions: Clean orphans, Optimize DB                     │    │
│  │                                                          │    │
│  │  [⚙️ Configure Schedule]                                 │    │
│  │                                                          │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─── History ─────────────────────────────────────────────┐    │
│  │                                                          │    │
│  │  📅 20 Jan 2026 03:00 - Scheduled cleanup                │    │
│  │     ✓ Removed 23 orphaned statistics                     │    │
│  │     ✓ Recovered 234 MB                                   │    │
│  │                                                          │    │
│  │  📅 13 Jan 2026 03:00 - Scheduled cleanup                │    │
│  │     ✓ Removed 8 orphaned entities                        │    │
│  │     ✓ Database optimized                                 │    │
│  │                                                          │    │
│  │  [View Full History]                                     │    │
│  │                                                          │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 6. Database Schema (Backend)

```sql
-- Care subscriptions (extends existing user table)
CREATE TABLE care_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    tier VARCHAR(20) NOT NULL DEFAULT 'free',  -- free, basic, premium
    started_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    ha_instance_id VARCHAR(64),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Cleanup events (analytics & history)
CREATE TABLE care_cleanup_events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    cleanup_type VARCHAR(50) NOT NULL,  -- orphaned_stats, orphaned_entities, optimize_db
    items_count INTEGER,
    space_recovered_mb DECIMAL(10,2),
    duration_seconds INTEGER,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Scheduled maintenance configs
CREATE TABLE care_schedules (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    frequency VARCHAR(20) NOT NULL,  -- daily, weekly, monthly
    time_utc TIME NOT NULL,
    day_of_week INTEGER,  -- 0-6 for weekly
    day_of_month INTEGER,  -- 1-28 for monthly
    actions JSONB NOT NULL,  -- ["orphaned_stats", "optimize_db"]
    enabled BOOLEAN DEFAULT TRUE,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Cloud backups
CREATE TABLE care_backups (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    backup_type VARCHAR(20) NOT NULL,  -- pre_cleanup, manual, scheduled
    file_path VARCHAR(255) NOT NULL,
    file_size_mb DECIMAL(10,2),
    ha_version VARCHAR(20),
    cleanup_id INTEGER REFERENCES care_cleanup_events(id),
    expires_at TIMESTAMP,  -- Auto-delete after 30 days
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_care_cleanup_user ON care_cleanup_events(user_id);
CREATE INDEX idx_care_cleanup_date ON care_cleanup_events(created_at);
CREATE INDEX idx_care_schedules_next ON care_schedules(next_run_at) WHERE enabled = TRUE;
CREATE INDEX idx_care_backups_expires ON care_backups(expires_at);
```

---

## 7. Implementation Roadmap

### Phase 1: MVP (2 weeks)

| Task | Priority | Effort |
|------|----------|--------|
| Scanner: Orphaned statistics | High | 2 days |
| Scanner: Orphaned entities | High | 2 days |
| Scanner: Database health | High | 1 day |
| Report generation | High | 1 day |
| Basic web UI | High | 2 days |
| Add-on packaging | High | 1 day |
| Documentation | Medium | 1 day |

**Deliverable:** Free tier working - scan & report

### Phase 2: Premium Features (2 weeks)

| Task | Priority | Effort |
|------|----------|--------|
| Backend: Care API endpoints | High | 2 days |
| Backend: Subscription checking | High | 1 day |
| Cleaner: Orphaned statistics | High | 2 days |
| Cleaner: Orphaned entities | High | 2 days |
| Cleaner: Database optimize | Medium | 1 day |
| Backup before cleanup | High | 2 days |

**Deliverable:** One-click cleanup working

### Phase 3: Scheduler & Polish (2 weeks)

| Task | Priority | Effort |
|------|----------|--------|
| Scheduler implementation | High | 3 days |
| Cloud backup storage | Medium | 2 days |
| Cleanup history | Medium | 1 day |
| Lovelace card | Medium | 2 days |
| Testing & bugfixes | High | 2 days |

**Deliverable:** Full product ready

### Phase 4: Launch (1 week)

| Task | Priority | Effort |
|------|----------|--------|
| HACS submission | High | 1 day |
| Documentation site | Medium | 2 days |
| Marketing materials | Medium | 1 day |
| Community announcement | High | 1 day |

---

## 8. Pricing Strategy

### Recommended Tiers

| Tier | Price | Features |
|------|-------|----------|
| **Free** | €0 | Scan, report, manual instructions |
| **Care Basic** | €1.99/mo | + One-click cleanup |
| **Care Premium** | €4.99/mo | + Scheduled + Backup + History |
| **Energy Bundle** | €7.99/mo | Care Premium + Energy features |

### Alternative: One-time Purchase

| Option | Price |
|--------|-------|
| Care Lifetime | €29 |
| Energy + Care Lifetime | €49 |

---

## 9. Success Metrics

### Product Metrics

| Metric | Target (3 months) |
|--------|-------------------|
| Add-on installs | 1,000+ |
| Free → Paid conversion | 5% |
| Monthly Active Users | 500+ |
| Cleanup actions performed | 10,000+ |

### Technical Metrics

| Metric | Target |
|--------|--------|
| Scan completion time | < 30 seconds |
| Cleanup success rate | > 99% |
| API response time | < 200ms |
| Uptime | 99.9% |

---

## 10. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| HA breaking changes | Medium | High | Version pinning, quick updates |
| Data loss during cleanup | Low | Critical | Mandatory backup, dry-run first |
| Competition (Spook etc.) | Medium | Medium | Better UX, premium features |
| Low conversion rate | Medium | Medium | Adjust pricing, add features |

---

## 11. Open Questions

1. **Backup storage**: Use S3/R2 or self-hosted MinIO?
2. **Pricing model**: Subscription vs one-time vs usage-based?
3. **Branding**: "Synctacles Care" vs "HA Janitor" vs other?
4. **Free tier limits**: How generous before paywall?

---

## Appendix A: Competitor Analysis

### Spook

- **Pros:** Open source, HACS available
- **Cons:** Limited features, entities-only, no scheduling
- **Gap:** No statistics cleanup, no database maintenance

### Manual Cleanup

- **Pros:** Free
- **Cons:** Requires technical knowledge, time-consuming, error-prone
- **Gap:** No guidance, no automation

### Our Advantage

- Complete solution (entities + statistics + database)
- One-click cleanup with safety net
- Scheduled maintenance
- Cloud backup
- Professional support

---

## Appendix B: Technical References

- HA Entity Registry: `/config/.storage/core.entity_registry`
- HA Database: `/config/home-assistant_v2.db`
- Statistics tables: `statistics_meta`, `statistics`, `statistics_short_term`
- HA Recorder docs: https://www.home-assistant.io/integrations/recorder/

