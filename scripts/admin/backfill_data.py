#!/usr/bin/env python3
"""
Automatic Data Backfill Orchestrator

Detects gaps in time-series energy data and triggers collection/import to fill them.
Supports multiple data sources with fallback strategies.

Usage:
  python backfill_data.py --source all --dry-run
  python backfill_data.py --source a44 --max-gap-days 30
"""

import os
import sys
import logging
import argparse
import json
import subprocess
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass

from sqlalchemy import create_engine, func
from sqlalchemy.orm import sessionmaker, Session

# Import database models
sys.path.insert(0, str(Path(__file__).parent.parent.parent / 'app'))

from synctacles_db.models import (
    RawEntsoeA75, RawEntsoeA65, RawEntsoeA44, RawTennetBalance,
    BackfillLog
)

from dotenv import load_dotenv

# ========================================================
#   CONFIGURATION
# ========================================================

load_dotenv()

LOG_DIR = Path(os.getenv("LOG_PATH", "/opt/energy-insights/logs"))
LOG_DIR.mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s] %(levelname)-8s %(name)s: %(message)s",
    handlers=[
        logging.FileHandler(LOG_DIR / "backfill_data.log"),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger(__name__)

DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://synctacles@localhost:5432/synctacles')
COLLECTORS_PATH = Path(os.getenv("INSTALL_PATH", "/opt/energy-insights") / "app" / "sparkcrawler_db" / "collectors")
VENV_PATH = Path(os.getenv("INSTALL_PATH", "/opt/energy-insights") / "venv")
VENV_PYTHON = VENV_PATH / "bin" / "python3"

# ========================================================
#   DATA CLASSES
# ========================================================

@dataclass
class Gap:
    """Represents a data gap"""
    source: str  # 'a75', 'a65', 'a44', 'tennet', 'energy_charts'
    category: Optional[str]  # psr_type, platform, or None
    start: datetime
    end: datetime

    def __str__(self):
        gap_hours = (self.end - self.start).total_seconds() / 3600
        return f"{self.source}({self.category or '-'}): {self.start} → {self.end} ({gap_hours:.0f}h)"


@dataclass
class CollectionResult:
    """Result of collection attempt"""
    status: str  # 'SUCCESS', 'FAILED', 'NO_DATA'
    stdout: str = ""
    stderr: str = ""
    duration: float = 0
    fallback_source: Optional[str] = None


# ========================================================
#   DATABASE SESSION
# ========================================================

def get_db_session() -> Session:
    """Create database session"""
    engine = create_engine(DATABASE_URL)
    SessionLocal = sessionmaker(bind=engine)
    return SessionLocal()


# ========================================================
#   GAP DETECTOR
# ========================================================

class GapDetector:
    """Detects gaps in time-series data"""

    def __init__(self, session: Session):
        self.session = session
        self.logger = logging.getLogger(self.__class__.__name__)

    def detect_gaps(self, source: str, country: str = 'NL') -> List[Gap]:
        """
        Detect gaps for specified source

        Returns:
            List of Gap objects
        """
        if source == 'a75':
            return self._detect_a75_gaps(country)
        elif source == 'a65':
            return self._detect_a65_gaps(country)
        elif source == 'a44':
            return self._detect_a44_gaps(country)
        elif source == 'tennet':
            return self._detect_tennet_gaps()
        else:
            self.logger.warning(f"Unknown source: {source}")
            return []

    def _detect_a75_gaps(self, country: str) -> List[Gap]:
        """Detect gaps in A75 data per PSR type"""
        gaps = []
        psr_types = ['B01', 'B04', 'B05', 'B14', 'B16', 'B17', 'B18', 'B19', 'B20']

        for psr_type in psr_types:
            try:
                # Find max timestamp for this PSR type
                max_ts = self.session.query(func.max(RawEntsoeA75.timestamp)).filter(
                    RawEntsoeA75.country == country,
                    RawEntsoeA75.psr_type == psr_type
                ).scalar()

                if max_ts is None:
                    # No data at all - suggest backfill from 30 days ago
                    max_ts = datetime.now(timezone.utc) - timedelta(days=30)
                    self.logger.info(f"A75 {psr_type}: No data, suggesting backfill from {max_ts}")

                # Calculate gap
                now = datetime.now(timezone.utc)
                gap_seconds = (now - max_ts).total_seconds()

                if gap_seconds > 3600:  # Gap > 1 hour
                    gaps.append(Gap(
                        source='a75',
                        category=psr_type,
                        start=max_ts,
                        end=now
                    ))
                    self.logger.info(f"A75 {psr_type}: Gap detected ({gap_seconds/3600:.1f} hours)")

            except Exception as e:
                self.logger.error(f"Failed to detect A75 {psr_type} gap: {e}")

        return gaps

    def _detect_a65_gaps(self, country: str) -> List[Gap]:
        """Detect gaps in A65 data per type (actual/forecast)"""
        gaps = []
        types = ['actual', 'forecast']

        for data_type in types:
            try:
                max_ts = self.session.query(func.max(RawEntsoeA65.timestamp)).filter(
                    RawEntsoeA65.country == country,
                    RawEntsoeA65.type == data_type
                ).scalar()

                if max_ts is None:
                    max_ts = datetime.now(timezone.utc) - timedelta(days=30)

                now = datetime.now(timezone.utc)
                gap_seconds = (now - max_ts).total_seconds()

                if gap_seconds > 3600:
                    gaps.append(Gap(
                        source='a65',
                        category=data_type,
                        start=max_ts,
                        end=now
                    ))
                    self.logger.info(f"A65 {data_type}: Gap detected ({gap_seconds/3600:.1f} hours)")

            except Exception as e:
                self.logger.error(f"Failed to detect A65 {data_type} gap: {e}")

        return gaps

    def _detect_a44_gaps(self, country: str) -> List[Gap]:
        """Detect gaps in A44 prices data"""
        gaps = []

        try:
            max_ts = self.session.query(func.max(RawEntsoeA44.timestamp)).filter(
                RawEntsoeA44.country == country
            ).scalar()

            if max_ts is None:
                max_ts = datetime.now(timezone.utc) - timedelta(days=30)

            now = datetime.now(timezone.utc)
            gap_seconds = (now - max_ts).total_seconds()

            if gap_seconds > 3600:
                gaps.append(Gap(
                    source='a44',
                    category=None,
                    start=max_ts,
                    end=now
                ))
                self.logger.info(f"A44: Gap detected ({gap_seconds/3600:.1f} hours)")

        except Exception as e:
            self.logger.error(f"Failed to detect A44 gap: {e}")

        return gaps

    def _detect_tennet_gaps(self) -> List[Gap]:
        """Detect gaps in TenneT balance data"""
        gaps = []
        platforms = ['aFRR', 'IGCC', 'MARI', 'mFRRda', 'PICASSO']

        for platform in platforms:
            try:
                max_ts = self.session.query(func.max(RawTennetBalance.timestamp)).filter(
                    RawTennetBalance.platform == platform
                ).scalar()

                if max_ts is None:
                    # TenneT only has recent data, don't suggest historical backfill
                    self.logger.info(f"TenneT {platform}: No data available")
                    continue

                now = datetime.now(timezone.utc)
                gap_seconds = (now - max_ts).total_seconds()

                if gap_seconds > 300:  # Gap > 5 minutes for real-time data
                    gaps.append(Gap(
                        source='tennet',
                        category=platform,
                        start=max_ts,
                        end=now
                    ))
                    self.logger.info(f"TenneT {platform}: Gap detected ({gap_seconds/60:.0f} minutes)")

            except Exception as e:
                self.logger.error(f"Failed to detect TenneT {platform} gap: {e}")

        return gaps


# ========================================================
#   COLLECTION TRIGGER
# ========================================================

class CollectionTrigger:
    """Triggers collector scripts with fallback support"""

    def __init__(self):
        self.logger = logging.getLogger(self.__class__.__name__)

    def trigger_collection(self, gap: Gap) -> CollectionResult:
        """Trigger collection for specified gap"""

        if gap.source == 'a75':
            return self._trigger_a75(gap)
        elif gap.source == 'a65':
            return self._trigger_a65(gap)
        elif gap.source == 'a44':
            return self._trigger_a44_with_fallback(gap)
        elif gap.source == 'tennet':
            return self._trigger_tennet(gap)
        elif gap.source == 'energy_charts':
            return self._trigger_energy_charts(gap)
        else:
            return CollectionResult(status='FAILED', stderr=f"Unknown source: {gap.source}")

    def _trigger_a75(self, gap: Gap) -> CollectionResult:
        """Trigger A75 collection"""
        return self._run_collector(
            'sparkcrawler_entso_e_a75_generation.py',
            gap.start,
            gap.end
        )

    def _trigger_a65(self, gap: Gap) -> CollectionResult:
        """Trigger A65 collection"""
        return self._run_collector(
            'sparkcrawler_entso_e_a65_load.py',
            gap.start,
            gap.end
        )

    def _trigger_a44_with_fallback(self, gap: Gap) -> CollectionResult:
        """Try A44 first, fall back to Energy-Charts"""
        result = self._trigger_a44(gap)
        if result.status == 'SUCCESS':
            return result

        self.logger.warning(f"A44 failed, attempting Energy-Charts fallback")
        result = self._trigger_energy_charts(gap)
        if result.status == 'SUCCESS':
            result.fallback_source = 'energy_charts'
        return result

    def _trigger_a44(self, gap: Gap) -> CollectionResult:
        """Trigger A44 collection"""
        return self._run_collector(
            'sparkcrawler_entso_e_a44_prices.py',
            gap.start,
            gap.end
        )

    def _trigger_tennet(self, gap: Gap) -> CollectionResult:
        """Trigger TenneT collection"""
        return self._run_collector(
            'sparkcrawler_tennet_ingestor.py',
            gap.start,
            gap.end
        )

    def _trigger_energy_charts(self, gap: Gap) -> CollectionResult:
        """Trigger Energy-Charts collection"""
        return self._run_collector(
            'energy_charts_prices.py',
            gap.start,
            gap.end
        )

    def _run_collector(self, script_name: str, start: datetime, end: datetime) -> CollectionResult:
        """Execute collector script via subprocess"""

        script_path = COLLECTORS_PATH / script_name
        if not script_path.exists():
            return CollectionResult(status='FAILED', stderr=f"Script not found: {script_path}")

        cmd = [
            str(VENV_PYTHON),
            str(script_path),
            '--start', start.isoformat(),
            '--end', end.isoformat(),
            '--backfill'
        ]

        self.logger.info(f"Running: {' '.join(cmd)}")

        start_time = time.time()
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
            duration = time.time() - start_time

            return CollectionResult(
                status='SUCCESS' if result.returncode == 0 else 'FAILED',
                stdout=result.stdout,
                stderr=result.stderr,
                duration=duration
            )
        except subprocess.TimeoutExpired:
            duration = time.time() - start_time
            return CollectionResult(
                status='FAILED',
                stderr=f"Timeout after {duration:.0f}s",
                duration=duration
            )
        except Exception as e:
            duration = time.time() - start_time
            return CollectionResult(
                status='FAILED',
                stderr=str(e),
                duration=duration
            )


# ========================================================
#   BACKFILL LOGGER
# ========================================================

class BackfillLogger:
    """Logs backfill operations to database"""

    def __init__(self, session: Session):
        self.session = session
        self.logger = logging.getLogger(self.__class__.__name__)

    def log_backfill(self, gap: Gap, result: CollectionResult, records_inserted: int = 0):
        """Log backfill operation"""

        log_entry = BackfillLog(
            source_type=gap.source,
            data_category=gap.category,
            country='NL',
            gap_start=gap.start,
            gap_end=gap.end,
            status=result.status,
            records_inserted=records_inserted if result.status == 'SUCCESS' else 0,
            records_failed=0 if result.status == 'SUCCESS' else 1,
            error_message=result.stderr[:500] if result.stderr else None,
            execution_duration_seconds=result.duration,
            fallback_source_used=result.fallback_source
        )

        try:
            self.session.add(log_entry)
            self.session.commit()
            self.logger.info(f"Logged backfill: {gap.source} - {result.status}")
        except Exception as e:
            self.logger.error(f"Failed to log backfill: {e}")


# ========================================================
#   MAIN ORCHESTRATOR
# ========================================================

def main():
    """Main backfill orchestration"""

    parser = argparse.ArgumentParser(
        description='Automatic Data Backfill Orchestrator',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Detect gaps for all sources (no backfill)
  python backfill_data.py --source all --dry-run

  # Backfill A44 prices (max 30 days)
  python backfill_data.py --source a44 --max-gap-days 30

  # Backfill all sources (max 7 days)
  python backfill_data.py --source all --max-gap-days 7
        """
    )

    parser.add_argument('--source', choices=['a75', 'a65', 'a44', 'tennet', 'energy_charts', 'all'], default='all',
                        help='Data source to backfill')
    parser.add_argument('--dry-run', action='store_true', help='Detect gaps only, do not backfill')
    parser.add_argument('--max-gap-days', type=int, default=30, help='Maximum gap size to backfill (days)')
    args = parser.parse_args()

    logger.info("=" * 80)
    logger.info("BACKFILL ORCHESTRATOR")
    logger.info("=" * 80)

    # Initialize
    session = get_db_session()
    detector = GapDetector(session)
    trigger = CollectionTrigger()
    logger_obj = BackfillLogger(session)

    # Determine sources
    sources = ['a75', 'a65', 'a44', 'tennet'] if args.source == 'all' else [args.source]

    total_gaps = 0
    total_backfilled = 0

    try:
        for source in sources:
            logger.info(f"\n--- Checking {source.upper()} ---")

            gaps = detector.detect_gaps(source)
            logger.info(f"Found {len(gaps)} gap(s)")
            total_gaps += len(gaps)

            for gap in gaps:
                logger.info(f"Gap: {gap}")

                # Filter by max gap size
                gap_days = (gap.end - gap.start).days
                if gap_days > args.max_gap_days:
                    logger.warning(f"Skipping: gap size ({gap_days}d) exceeds max ({args.max_gap_days}d)")
                    logger_obj.log_backfill(gap, CollectionResult(status='SKIPPED'), 0)
                    continue

                if args.dry_run:
                    logger.info("DRY-RUN: Would backfill this gap")
                    continue

                # Trigger collection
                logger.info(f"Triggering collection...")
                result = trigger.trigger_collection(gap)
                logger.info(f"Result: {result.status}")

                if result.stderr:
                    logger.error(f"Error: {result.stderr[:200]}")

                # Log result
                logger_obj.log_backfill(gap, result)

                if result.status == 'SUCCESS':
                    total_backfilled += 1

    except Exception as e:
        logger.error(f"Fatal error: {e}", exc_info=True)
        return 1
    finally:
        session.close()

    logger.info("\n" + "=" * 80)
    logger.info(f"SUMMARY: {total_backfilled}/{total_gaps} gaps backfilled")
    logger.info("=" * 80)

    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
        sys.exit(130)
    except Exception as e:
        logger.error(f"Fatal error: {e}", exc_info=True)
        sys.exit(1)
