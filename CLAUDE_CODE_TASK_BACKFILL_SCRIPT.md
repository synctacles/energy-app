# CLAUDE CODE TASK: Automatic Data Backfill Script with Fallback Strategy

## Objective
Create production-ready automatic backfill script that:
1. Runs after API restart/startup
2. Detects data gaps in all sources
3. Backfills missing data automatically
4. Respects fallback strategy (ENTSO-E → Energy-Charts)
5. Logs all operations
6. Integrates with systemd for automation

---

## Requirements

### Functional Requirements

**1. Gap Detection**
- Check latest timestamp per source (raw_* tables)
- Calculate gap from latest → now
- Minimum gap threshold (default: 2 hours) to trigger backfill
- Support per-source configuration

**2. Data Sources to Monitor**

**Primary Sources (ENTSO-E):**
- Generation Mix (A75) → raw_entso_e_a75, norm_entso_e_a75
- Load Data (A65) → raw_entso_e_a65, norm_entso_e_a65
- Day-Ahead Prices (A44) → raw_entso_e_a44, norm_entso_e_a44

**Secondary Sources:**
- TenneT Balance → raw_tennet_balance, norm_tennet_balance

**Fallback Sources:**
- Energy-Charts Prices (when ENTSO-E A44 fails) → raw_prices, norm_prices

**3. Fallback Strategy**

```
For Prices (A44):
1. Try ENTSO-E A44 first
2. If fails or missing → try Energy-Charts
3. If both fail → log error, retry later

For Generation/Load:
1. Try ENTSO-E (only source)
2. If fails → log error, retry later

For Balance:
1. Try TenneT (only source)
2. If fails → log error, retry later
```

**4. Automatic Collection**
- Trigger appropriate collector script for detected gap
- Pass start_time and end_time parameters
- Batch large gaps (max 24h per batch)
- Respect API rate limits (wait between batches)

**5. Verification**
- After backfill, verify gap is filled
- Check normalized data exists (raw + norm pipeline)
- Log success/failure per source

**6. Idempotent**
- Safe to run multiple times
- Skip already-filled data
- No duplicate inserts

---

## Implementation Design

### Script Location
`scripts/admin/backfill_data.py`

### Configuration (.env)

```bash
# Backfill settings
BACKFILL_MIN_GAP_HOURS=2          # Minimum gap to trigger backfill
BACKFILL_MAX_LOOKBACK_DAYS=7      # How far back to check
BACKFILL_BATCH_HOURS=24           # Hours per collection batch
BACKFILL_RATE_LIMIT_SECONDS=5     # Wait between API calls
BACKFILL_LOG_PATH=/opt/energy-insights-nl/logs/backfill
BACKFILL_RETRY_FAILED_HOURS=24    # Retry failed backfills after X hours

# Fallback strategy
BACKFILL_USE_FALLBACK=true
BACKFILL_FALLBACK_DELAY_MINUTES=5  # Wait before trying fallback
```

### Core Components

#### 1. Gap Detector

```python
class GapDetector:
    """Detect data gaps across all sources"""
    
    def __init__(self, db_conn, min_gap_hours=2):
        self.conn = db_conn
        self.min_gap_hours = min_gap_hours
        
        # Source configuration
        self.sources = {
            'entso_a75': {
                'raw_table': 'raw_entso_e_a75',
                'norm_table': 'norm_entso_e_a75',
                'collector': 'collector_entso_a75.py',
                'interval': '1 hour',
                'has_fallback': False
            },
            'entso_a65': {
                'raw_table': 'raw_entso_e_a65',
                'norm_table': 'norm_entso_e_a65',
                'collector': 'collector_entso_a65.py',
                'interval': '1 hour',
                'has_fallback': False
            },
            'entso_a44': {
                'raw_table': 'raw_entso_e_a44',
                'norm_table': 'norm_entso_e_a44',
                'collector': 'collector_entso_a44.py',
                'interval': '1 hour',
                'has_fallback': True,
                'fallback_source': 'energy_charts_prices'
            },
            'tennet_balance': {
                'raw_table': 'raw_tennet_balance',
                'norm_table': 'norm_tennet_balance',
                'collector': 'collector_tennet.py',
                'interval': '15 minutes',
                'has_fallback': False
            },
            'energy_charts_prices': {
                'raw_table': 'raw_prices',
                'norm_table': 'norm_prices',
                'collector': 'collector_energy_charts_prices.py',
                'interval': '1 hour',
                'is_fallback': True,
                'filter': "source = 'energy-charts'"
            }
        }
    
    def get_latest_timestamp(self, source_key):
        """Get latest timestamp for a source"""
        source = self.sources[source_key]
        table = source['raw_table']
        filter_clause = source.get('filter', '')
        
        query = f"""
            SELECT MAX(timestamp) as latest
            FROM {table}
            {f"WHERE {filter_clause}" if filter_clause else ""}
        """
        
        result = self.conn.execute(query).fetchone()
        return result['latest'] if result['latest'] else None
    
    def detect_gap(self, source_key):
        """Detect if source has significant gap"""
        latest = self.get_latest_timestamp(source_key)
        
        if not latest:
            # No data at all - use default start
            return {
                'source': source_key,
                'has_gap': True,
                'gap_start': datetime.now(timezone.utc) - timedelta(days=7),
                'gap_end': datetime.now(timezone.utc),
                'gap_hours': 7 * 24,
                'reason': 'NO_DATA'
            }
        
        # Calculate gap
        now = datetime.now(timezone.utc)
        gap_hours = (now - latest).total_seconds() / 3600
        
        if gap_hours > self.min_gap_hours:
            return {
                'source': source_key,
                'has_gap': True,
                'gap_start': latest + timedelta(hours=1),
                'gap_end': now,
                'gap_hours': gap_hours,
                'reason': 'STALE_DATA'
            }
        
        return {
            'source': source_key,
            'has_gap': False,
            'latest': latest,
            'age_hours': gap_hours
        }
    
    def detect_all_gaps(self):
        """Detect gaps in all sources"""
        gaps = []
        
        for source_key in self.sources.keys():
            # Skip fallback sources (they're triggered by primary failure)
            if self.sources[source_key].get('is_fallback'):
                continue
            
            gap = self.detect_gap(source_key)
            if gap['has_gap']:
                gaps.append(gap)
        
        return gaps
```

#### 2. Collection Trigger with Fallback

```python
class CollectionTrigger:
    """Trigger data collection with fallback support"""
    
    def __init__(self, venv_path, app_path, rate_limit_seconds=5):
        self.venv_python = f"{venv_path}/bin/python"
        self.collectors_path = f"{app_path}/synctacles_db/collectors"
        self.rate_limit = rate_limit_seconds
    
    def trigger_collector(self, collector_script, start_time, end_time, source_key=None):
        """Run collector for specific time range"""
        collector_path = f"{self.collectors_path}/{collector_script}"
        
        cmd = [
            self.venv_python,
            collector_path,
            '--start', start_time.isoformat(),
            '--end', end_time.isoformat(),
            '--backfill'
        ]
        
        logger.info(f"Triggering: {collector_script} for {start_time} → {end_time}")
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=300  # 5 min timeout
        )
        
        if result.returncode != 0:
            logger.error(f"Collector failed: {result.stderr}")
            return None
        
        # Parse output for records collected
        records_match = re.search(r'Collected (\d+) records', result.stdout)
        records = int(records_match.group(1)) if records_match else 0
        
        logger.info(f"✓ Collected {records} records")
        return records
    
    def backfill_with_fallback(self, gap, gap_detector, use_fallback=True):
        """Backfill gap with fallback strategy"""
        source_key = gap['source']
        source_config = gap_detector.sources[source_key]
        
        collector = source_config['collector']
        start = gap['gap_start']
        end = gap['gap_end']
        
        # Try primary source
        logger.info(f"Attempting primary source: {source_key}")
        
        try:
            records = self.trigger_collector(collector, start, end, source_key)
            
            if records and records > 0:
                return {
                    'source': source_key,
                    'status': 'SUCCESS',
                    'method': 'PRIMARY',
                    'records': records,
                    'gap_start': start,
                    'gap_end': end
                }
        except Exception as e:
            logger.error(f"Primary source failed: {e}")
        
        # Try fallback if available and enabled
        if use_fallback and source_config.get('has_fallback'):
            fallback_key = source_config['fallback_source']
            fallback_config = gap_detector.sources[fallback_key]
            
            logger.warning(f"Primary failed, trying fallback: {fallback_key}")
            
            # Wait before fallback (rate limiting)
            time.sleep(int(os.getenv('BACKFILL_FALLBACK_DELAY_MINUTES', 5)) * 60)
            
            try:
                fallback_collector = fallback_config['collector']
                records = self.trigger_collector(fallback_collector, start, end, fallback_key)
                
                if records and records > 0:
                    return {
                        'source': source_key,
                        'status': 'SUCCESS',
                        'method': 'FALLBACK',
                        'fallback_used': fallback_key,
                        'records': records,
                        'gap_start': start,
                        'gap_end': end
                    }
            except Exception as e:
                logger.error(f"Fallback also failed: {e}")
        
        # Both failed
        return {
            'source': source_key,
            'status': 'FAILED',
            'error': 'Both primary and fallback failed',
            'gap_start': start,
            'gap_end': end
        }
    
    def backfill_gap_batched(self, gap, gap_detector, batch_hours=24):
        """Backfill large gap in batches"""
        start = gap['gap_start']
        end = gap['gap_end']
        
        results = []
        current = start
        
        while current < end:
            batch_end = min(current + timedelta(hours=batch_hours), end)
            
            batch_gap = {
                'source': gap['source'],
                'gap_start': current,
                'gap_end': batch_end,
                'gap_hours': (batch_end - current).total_seconds() / 3600
            }
            
            result = self.backfill_with_fallback(batch_gap, gap_detector)
            results.append(result)
            
            # Rate limiting between batches
            if current + timedelta(hours=batch_hours) < end:
                time.sleep(self.rate_limit)
            
            current = batch_end
        
        # Aggregate results
        total_records = sum(r.get('records', 0) for r in results)
        all_success = all(r['status'] == 'SUCCESS' for r in results)
        
        return {
            'source': gap['source'],
            'status': 'SUCCESS' if all_success else 'PARTIAL',
            'batches': len(results),
            'total_records': total_records,
            'gap_start': start,
            'gap_end': end,
            'batch_results': results
        }
```

#### 3. Verification

```python
class BackfillVerifier:
    """Verify backfill completion"""
    
    def __init__(self, db_conn, gap_detector):
        self.conn = db_conn
        self.gap_detector = gap_detector
    
    def verify_gap_filled(self, result):
        """Check if gap was actually filled"""
        source_key = result['source']
        gap_start = result['gap_start']
        gap_end = result['gap_end']
        
        # Re-check gap
        current_gap = self.gap_detector.detect_gap(source_key)
        
        if not current_gap['has_gap']:
            logger.info(f"✓ Gap filled for {source_key}")
            return True
        
        # Check if gap reduced
        if current_gap['gap_hours'] < result.get('original_gap_hours', float('inf')):
            logger.warning(f"⚠ Gap partially filled for {source_key}: {current_gap['gap_hours']:.1f}h remaining")
            return 'PARTIAL'
        
        logger.error(f"✗ Gap still exists for {source_key}")
        return False
    
    def verify_normalization(self, result):
        """Check if normalized data exists"""
        source_key = result['source']
        source_config = self.gap_detector.sources[source_key]
        
        norm_table = source_config['norm_table']
        gap_start = result['gap_start']
        gap_end = result['gap_end']
        
        query = f"""
            SELECT COUNT(*) as count
            FROM {norm_table}
            WHERE timestamp >= %s AND timestamp <= %s
        """
        
        count = self.conn.execute(query, (gap_start, gap_end)).fetchone()['count']
        
        if count > 0:
            logger.info(f"✓ Normalized data exists: {count} records")
            return True
        
        logger.warning(f"⚠ No normalized data yet (normalizer may need to run)")
        return False
```

#### 4. Logging

```python
class BackfillLogger:
    """Comprehensive backfill logging"""
    
    def __init__(self, db_conn, log_dir):
        self.conn = db_conn
        self.log_dir = Path(log_dir)
        self.log_dir.mkdir(parents=True, exist_ok=True)
        
        # Ensure backfill_log table exists
        self.ensure_log_table()
    
    def ensure_log_table(self):
        """Create backfill_log table if not exists"""
        query = """
            CREATE TABLE IF NOT EXISTS backfill_log (
                id SERIAL PRIMARY KEY,
                source VARCHAR(50) NOT NULL,
                gap_start TIMESTAMPTZ NOT NULL,
                gap_end TIMESTAMPTZ NOT NULL,
                gap_hours NUMERIC(10,2),
                status VARCHAR(20) NOT NULL,
                method VARCHAR(20),
                records_collected INTEGER,
                fallback_used VARCHAR(50),
                error TEXT,
                executed_at TIMESTAMPTZ DEFAULT NOW(),
                duration_seconds NUMERIC(10,2)
            );
            
            CREATE INDEX IF NOT EXISTS idx_backfill_log_source 
            ON backfill_log(source);
            
            CREATE INDEX IF NOT EXISTS idx_backfill_log_executed 
            ON backfill_log(executed_at);
        """
        self.conn.execute(query)
        self.conn.commit()
    
    def log_backfill(self, result, duration_seconds):
        """Log backfill operation to database"""
        query = """
            INSERT INTO backfill_log 
            (source, gap_start, gap_end, gap_hours, status, method, 
             records_collected, fallback_used, error, duration_seconds)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """
        
        self.conn.execute(query, (
            result['source'],
            result['gap_start'],
            result['gap_end'],
            result.get('gap_hours'),
            result['status'],
            result.get('method'),
            result.get('records', 0),
            result.get('fallback_used'),
            result.get('error'),
            duration_seconds
        ))
        self.conn.commit()
    
    def save_summary_report(self, results):
        """Save JSON summary report"""
        report_file = self.log_dir / f"backfill_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        
        summary = {
            'timestamp': datetime.now().isoformat(),
            'total_gaps': len(results),
            'successful': sum(1 for r in results if r['status'] == 'SUCCESS'),
            'failed': sum(1 for r in results if r['status'] == 'FAILED'),
            'partial': sum(1 for r in results if r['status'] == 'PARTIAL'),
            'total_records': sum(r.get('records', 0) for r in results),
            'results': results
        }
        
        with open(report_file, 'w') as f:
            json.dump(summary, f, indent=2, default=str)
        
        logger.info(f"Report saved: {report_file}")
```

---

## Main Script Structure

```python
#!/usr/bin/env python3
"""
Automatic Data Backfill Script
Detects and fills data gaps with fallback strategy support
"""

import os
import sys
import logging
from datetime import datetime, timezone
from pathlib import Path

# Add app to path
sys.path.insert(0, '/opt/energy-insights-nl/app')

from synctacles_db.api.database import get_db_connection
from config.settings import settings

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/opt/energy-insights-nl/logs/backfill/backfill.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)


def main():
    """Main backfill execution"""
    logger.info("=" * 80)
    logger.info("BACKFILL: Starting automatic data backfill")
    logger.info("=" * 80)
    
    # Initialize components
    db_conn = get_db_connection()
    gap_detector = GapDetector(db_conn, min_gap_hours=int(os.getenv('BACKFILL_MIN_GAP_HOURS', 2)))
    collector = CollectionTrigger(
        venv_path='/opt/energy-insights-nl/venv',
        app_path='/opt/energy-insights-nl/app',
        rate_limit_seconds=int(os.getenv('BACKFILL_RATE_LIMIT_SECONDS', 5))
    )
    verifier = BackfillVerifier(db_conn, gap_detector)
    logger_obj = BackfillLogger(db_conn, os.getenv('BACKFILL_LOG_PATH', '/opt/energy-insights-nl/logs/backfill'))
    
    # Detect gaps
    logger.info("Detecting data gaps...")
    gaps = gap_detector.detect_all_gaps()
    
    if not gaps:
        logger.info("✓ No gaps detected - data is up to date")
        return
    
    logger.info(f"Found {len(gaps)} gaps to backfill:")
    for gap in gaps:
        logger.info(f"  {gap['source']}: {gap['gap_hours']:.1f} hours ({gap['reason']})")
    
    # Execute backfills
    results = []
    
    for gap in gaps:
        logger.info(f"\nBackfilling {gap['source']}...")
        start_time = datetime.now()
        
        try:
            result = collector.backfill_gap_batched(
                gap, 
                gap_detector,
                batch_hours=int(os.getenv('BACKFILL_BATCH_HOURS', 24))
            )
            
            # Verify
            verification = verifier.verify_gap_filled(result)
            result['verified'] = verification
            
            results.append(result)
            
        except Exception as e:
            logger.error(f"Backfill failed: {e}")
            results.append({
                'source': gap['source'],
                'status': 'FAILED',
                'error': str(e),
                'gap_start': gap['gap_start'],
                'gap_end': gap['gap_end']
            })
        
        # Log to database
        duration = (datetime.now() - start_time).total_seconds()
        logger_obj.log_backfill(results[-1], duration)
    
    # Summary
    logger.info("\n" + "=" * 80)
    logger.info("BACKFILL: Complete")
    logger.info("=" * 80)
    
    successful = sum(1 for r in results if r['status'] == 'SUCCESS')
    logger.info(f"Successful: {successful}/{len(results)}")
    logger.info(f"Total records: {sum(r.get('total_records', 0) for r in results):,}")
    
    # Save report
    logger_obj.save_summary_report(results)


if __name__ == '__main__':
    main()
```

---

## Systemd Integration

### Service File: backfill-startup.service

```ini
[Unit]
Description=SYNCTACLES Automatic Data Backfill (Startup)
After=network.target postgresql.service energy-insights-nl-api.service
Wants=energy-insights-nl-api.service

[Service]
Type=oneshot
User=energy-insights-nl
Group=energy-insights-nl
WorkingDirectory=/opt/energy-insights-nl/app
Environment="PATH=/opt/energy-insights-nl/venv/bin"
EnvironmentFile=/opt/.env

# Wait for API to be ready
ExecStartPre=/bin/sleep 10

# Run backfill
ExecStart=/opt/energy-insights-nl/venv/bin/python /opt/energy-insights-nl/scripts/admin/backfill_data.py

StandardOutput=journal
StandardError=journal

# Restart on failure
Restart=on-failure
RestartSec=300

[Install]
WantedBy=multi-user.target
```

### Timer File: backfill-periodic.timer

```ini
[Unit]
Description=SYNCTACLES Periodic Data Backfill Check

[Timer]
# Run every 6 hours
OnCalendar=*-*-* 00,06,12,18:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

### Service File: backfill-periodic.service

```ini
[Unit]
Description=SYNCTACLES Periodic Data Backfill Check
After=network.target postgresql.service

[Service]
Type=oneshot
User=energy-insights-nl
Group=energy-insights-nl
WorkingDirectory=/opt/energy-insights-nl/app
Environment="PATH=/opt/energy-insights-nl/venv/bin"
EnvironmentFile=/opt/.env

ExecStart=/opt/energy-insights-nl/venv/bin/python /opt/energy-insights-nl/scripts/admin/backfill_data.py

StandardOutput=journal
StandardError=journal
```

---

## Testing Checklist

**Unit Testing:**
- [ ] Gap detection finds known gaps
- [ ] Primary collector triggers correctly
- [ ] Fallback triggers when primary fails
- [ ] Batch processing handles large gaps
- [ ] Verification detects filled gaps
- [ ] Logging writes to database correctly

**Integration Testing:**
- [ ] Script runs successfully on fresh database
- [ ] Script handles partially-filled data
- [ ] Fallback strategy works (simulate ENTSO-E failure)
- [ ] Rate limiting prevents API throttling
- [ ] Systemd service starts on boot
- [ ] Timer triggers periodically

**Production Testing:**
- [ ] Run on actual server after API restart
- [ ] Verify data gaps filled
- [ ] Check normalization pipeline triggered
- [ ] Review logs for errors
- [ ] Verify fallback used when appropriate

---

## Success Criteria

**Script is production-ready when:**
1. ✅ Automatically detects all data gaps
2. ✅ Backfills missing data correctly
3. ✅ Respects fallback strategy (ENTSO-E → Energy-Charts)
4. ✅ Integrates with systemd (runs on startup + periodic)
5. ✅ Logs all operations to database
6. ✅ Idempotent (safe to re-run)
7. ✅ Handles errors gracefully
8. ✅ Works for all data sources

---

## Deliverables

**Files to Create:**
1. `scripts/admin/backfill_data.py` - Main script
2. `systemd/backfill-startup.service.template` - Startup service
3. `systemd/backfill-periodic.timer.template` - Periodic timer
4. `systemd/backfill-periodic.service.template` - Periodic service
5. `scripts/admin/test_backfill.py` - Unit tests (optional but recommended)

**Documentation:**
- README section explaining backfill logic
- Configuration options in .env
- Troubleshooting guide

---

## Priority: HIGH

Critical for production reliability - ensures continuous data availability even after restarts or collection failures.

---

**Implement comprehensive backfill system!** 🚀
