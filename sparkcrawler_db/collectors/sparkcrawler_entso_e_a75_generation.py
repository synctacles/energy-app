#!/usr/bin/env python3
"""
SYNCTACLES SparkCrawler - ENTSO-E A75 Generation Mix Collector
Fetches ENTSO-E Generation Mix (A75) per PSR-type
Stores raw XML to files for later import into database

Document Type: A75 (Generation per PSR-type)
Country: Netherlands (NL)
PSR-types: B01, B04, B05, B14, B16, B17, B18, B19, B20

Output: logs/entso_e_raw/a75_NL_{psr_type}_{timestamp}.xml

Author: SYNCTACLES Development
Version: 2.0.0 (file-based, no direct DB)
"""

import os
import sys
import logging
import argparse
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Optional, Dict

import pandas as pd
from entsoe import EntsoeRawClient
from dotenv import load_dotenv

# ========================================================
#   CONFIGURATION
# ========================================================

load_dotenv()

# Fix API Key inconsistency
api_key = os.getenv("ENTSOE_API_KEY") or os.getenv("ENTSO_E_API_KEY")
if not api_key:
    print("ERROR: ENTSOE_API_KEY not found")

LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))
LOG_DIR.mkdir(parents=True, exist_ok=True)

RAW_OUTPUT_DIR = LOG_DIR / "collectors" / "entso_e_raw"
RAW_OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s] %(levelname)-8s %(name)s: %(message)s",
    handlers=[
        logging.FileHandler(LOG_DIR / "entso_e_a75_collector.log"),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger(__name__)

# ========================================================
#   PSR-TYPE DEFINITIONS (Netherlands)
# ========================================================

PSR_TYPES = {
    'B16': {'name': 'Solar', 'priority': 1},
    'B18': {'name': 'Wind Offshore', 'priority': 2},
    'B19': {'name': 'Wind Onshore', 'priority': 3},
    'B04': {'name': 'Fossil Gas', 'priority': 4},
    'B05': {'name': 'Fossil Hard Coal', 'priority': 5},
    'B01': {'name': 'Biomass', 'priority': 6},
    'B14': {'name': 'Nuclear', 'priority': 7},
    'B17': {'name': 'Waste', 'priority': 8},
    'B20': {'name': 'Other', 'priority': 9},
}

COUNTRY_CODE = 'NL'

# ========================================================
#   COLLECTOR CLASS
# ========================================================

class EntsoEA75Collector:
    """Fetches ENTSO-E A75 Generation Mix data, stores raw XML to files."""

    def __init__(self, api_key: str, country_code: str = 'NL'):
        self.client = EntsoeRawClient(api_key=api_key)
        self.country_code = country_code
        self.logger = logging.getLogger(self.__class__.__name__)
        self.results = {}

    def fetch_generation_mix(self, psr_type: str, start: pd.Timestamp = None, end: pd.Timestamp = None, hours_back: int = 24) -> Optional[str]:
        """Fetch generation data for specific PSR-type.

        Args:
            psr_type: PSR-type code (B01-B20)
            start: Start timestamp (if None, calculated from hours_back)
            end: End timestamp (if None, uses now)
            hours_back: Hours to look back (only used if start is None)
        """
        try:
            # Determine time range
            if end is None:
                end = pd.Timestamp(datetime.now(timezone.utc))
            if start is None:
                start = end - timedelta(hours=hours_back)

            self.logger.info(
                f"Fetching {PSR_TYPES[psr_type]['name']} ({psr_type}) "
                f"from {start.strftime('%Y-%m-%d %H:%M')} to {end.strftime('%Y-%m-%d %H:%M')}"
            )

            xml_response = self.client.query_generation(
                country_code=self.country_code,
                start=start,
                end=end,
                psr_type=psr_type
            )

            if xml_response:
                self.logger.info(f"Received {len(xml_response)} bytes")
                return xml_response
            else:
                self.logger.warning(f"Empty response for {psr_type}")
                return None

        except Exception as e:
            self.logger.error(f"Failed to fetch {psr_type}: {str(e)}")
            return None

    def fetch_all_psr_types(self, start: pd.Timestamp = None, end: pd.Timestamp = None, hours_back: int = 24, rate_limit_seconds: int = 0) -> Dict[str, Optional[str]]:
        """Fetch generation data for ALL PSR-types.

        Args:
            start: Start timestamp
            end: End timestamp
            hours_back: Hours to look back (only if start/end not specified)
            rate_limit_seconds: Seconds to wait between API calls
        """
        self.logger.info(f"Starting bulk fetch for {len(PSR_TYPES)} PSR-types...")

        results = {}
        success_count = 0
        fail_count = 0

        for psr_type in sorted(PSR_TYPES.keys()):
            xml = self.fetch_generation_mix(psr_type, start=start, end=end, hours_back=hours_back)
            results[psr_type] = xml
            if xml:
                success_count += 1
            else:
                fail_count += 1

            # Rate limiting between requests
            if rate_limit_seconds > 0 and psr_type != sorted(PSR_TYPES.keys())[-1]:
                self.logger.info(f"Rate limiting: {rate_limit_seconds}s delay before next request")
                time.sleep(rate_limit_seconds)

        self.logger.info(f"Bulk fetch complete: {success_count} success, {fail_count} failed")
        self.results = results
        return results

    def save_to_files(self, output_dir: Optional[Path] = None) -> Dict[str, str]:
        """Save all fetched responses to XML files."""
        if not output_dir:
            output_dir = RAW_OUTPUT_DIR

        output_dir.mkdir(parents=True, exist_ok=True)
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%d_%H%M%S")

        saved_files = {}

        for psr_type, xml_data in self.results.items():
            if xml_data:
                filename = f"a75_{self.country_code}_{psr_type}_{timestamp}.xml"
                filepath = output_dir / filename

                try:
                    with open(filepath, 'w', encoding='utf-8') as f:
                        f.write(xml_data)
                    saved_files[psr_type] = str(filepath)
                    self.logger.info(f"Saved {psr_type} to {filepath}")
                except Exception as e:
                    self.logger.error(f"Failed to save {psr_type}: {str(e)}")

        return saved_files

    def get_summary(self) -> Dict:
        """Get summary of fetched data."""
        summary = {
            'timestamp': datetime.now(timezone.utc).isoformat(),
            'country': self.country_code,
            'document_type': 'A75',
            'psr_types_requested': len(PSR_TYPES),
            'psr_types_succeeded': sum(1 for v in self.results.values() if v),
            'psr_types_failed': sum(1 for v in self.results.values() if not v),
            'data': {}
        }

        for psr_type, xml in self.results.items():
            summary['data'][psr_type] = {
                'name': PSR_TYPES[psr_type]['name'],
                'status': 'success' if xml else 'failed',
                'size_bytes': len(xml) if xml else 0
            }

        return summary


# ========================================================
#   MAIN EXECUTION
# ========================================================

def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description='ENTSO-E A75 Generation Mix Collector',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Default: last 24 hours
  python sparkcrawler_entso_e_a75_generation.py

  # Specific date range
  python sparkcrawler_entso_e_a75_generation.py --start 2025-12-01T00:00:00 --end 2025-12-02T00:00:00

  # Backfill mode: batched with rate limiting
  python sparkcrawler_entso_e_a75_generation.py --start 2025-12-01T00:00:00 --end 2025-12-31T23:59:59 --backfill
        """
    )
    parser.add_argument('--start', type=str, help='Start datetime (ISO format, e.g., 2025-12-01T00:00:00Z)')
    parser.add_argument('--end', type=str, help='End datetime (ISO format, e.g., 2025-12-02T00:00:00Z)')
    parser.add_argument('--backfill', action='store_true', help='Backfill mode (batched with rate limiting)')
    args = parser.parse_args()

    logger.info("=" * 60)
    logger.info("SYNCTACLES SparkCrawler - ENTSO-E A75 Collector")
    logger.info("=" * 60)

    # Parse dates if provided
    start_ts = None
    end_ts = None

    if args.start:
        try:
            start_ts = pd.Timestamp(args.start, tz='UTC')
            logger.info(f"Start: {start_ts}")
        except Exception as e:
            logger.error(f"Failed to parse start datetime: {e}")
            return 1

    if args.end:
        try:
            end_ts = pd.Timestamp(args.end, tz='UTC')
            logger.info(f"End: {end_ts}")
        except Exception as e:
            logger.error(f"Failed to parse end datetime: {e}")
            return 1

    collector = EntsoEA75Collector(api_key=api_key, country_code=COUNTRY_CODE)

    logger.info(f"Fetching Generation Mix (A75) for {COUNTRY_CODE}...")
    logger.info(f"PSR-types: {', '.join(sorted(PSR_TYPES.keys()))}")

    # Determine rate limiting based on backfill mode
    rate_limit = 5 if args.backfill else 0
    if args.backfill:
        logger.info("Backfill mode enabled: 5s rate limit between PSR-type requests")

    collector.fetch_all_psr_types(start=start_ts, end=end_ts, hours_back=24, rate_limit_seconds=rate_limit)

    logger.info("Saving raw XML responses...")
    saved_files = collector.save_to_files()
    logger.info(f"Saved {len(saved_files)} files to {RAW_OUTPUT_DIR}")

    summary = collector.get_summary()

    logger.info("=" * 60)
    logger.info("SUMMARY")
    logger.info("=" * 60)
    logger.info(f"Timestamp:  {summary['timestamp']}")
    logger.info(f"Country:    {summary['country']}")
    logger.info(f"Successful: {summary['psr_types_succeeded']}")
    logger.info(f"Failed:     {summary['psr_types_failed']}")

    for psr_type, data in sorted(summary['data'].items()):
        status_icon = "OK" if data['status'] == 'success' else "FAIL"
        logger.info(f"  [{status_icon}] {psr_type} {data['name']:20s} ({data['size_bytes']:6d} bytes)")

    logger.info(f"Output: {RAW_OUTPUT_DIR}")
    logger.info("=" * 60)

    return 1 if summary['psr_types_failed'] > 0 else 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        logger.info("Interrupted")
        sys.exit(130)
    except Exception as e:
        logger.error(f"Fatal: {str(e)}", exc_info=True)
        sys.exit(1)
