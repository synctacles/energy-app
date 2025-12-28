#!/usr/bin/env python3
"""
SYNCTACLES SparkCrawler — TenneT Balance Delta Ingestor
Fetches raw TenneT data and stores COMPLETE records in SparkCrawler database
NO normalization at this layer — this is the source of truth

Data Flow:
1. Fetch raw TenneT API response
2. Parse all points from TimeSeries → Period → points
3. Store EVERY field (power_*, timestamps, metadata, raw JSON)
4. Later: SYNCTACLES normalizer will extract only delta_mw

Author: SYNCTACLES Development
Version: 1.0.0
"""

import os
import sys
import logging
import json
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Optional, Dict, List

import requests
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
        logging.FileHandler(LOG_DIR / "tennet_ingestor.log"),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger(__name__)

# ========================================================
#   CONSTANTS
# ========================================================

COUNTRY = 'NL'
TENNET_API_BASE = os.getenv("TENNET_API_BASE", "https://api.tennet.eu")
TENNET_API_KEY = os.getenv("TENNET_API_KEY", "")

TIMEOUT_SECONDS = 10

# Fields to preserve from raw TenneT response
POWER_FIELDS = [
    'power_afrr_in', 'power_afrr_out',
    'power_igcc_in', 'power_igcc_out',
    'power_mari_in', 'power_mari_out',
    'power_mfrrda_in', 'power_mfrrda_out',
    'power_picasso_in', 'power_picasso_out'
]

METADATA_FIELDS = [
    'mid_price',
    'max_upw_regulation_price',
    'min_downw_regulation_price'
]

TIMESTAMP_FIELDS = [
    'timeInterval_start',
    'timeInterval_end'
]

# ========================================================
#   INGESTOR CLASS
# ========================================================

class TennetBalanceDeltaIngestor:
    """
    Ingests raw TenneT Balance Delta data
    Stores complete records in SparkCrawler (no normalization)
    """
    
    def __init__(self, api_base: str = "", api_key: str = ""):
        """Initialize ingestor"""
        self.logger = logging.getLogger(self.__class__.__name__)
        
        self.api_base = api_base or TENNET_API_BASE
        self.api_key = api_key or TENNET_API_KEY
        
        self.session = requests.Session()
        brand_slug = os.getenv("BRAND_SLUG", "energy-insights")
        self.session.headers.update({
            'User-Agent': f'{brand_slug}-collector/1.0',
            'Accept': 'application/json',
            'apikey': self.api_key  # TenneT uses 'apikey' header
        })
        
        self.raw_records = []
    
    def fetch_latest(self) -> Optional[Dict]:
        """
        Fetch latest balance delta data from TenneT
        
        Returns:
            Raw API response or None if failed
        """
        try:
            endpoint = f"{self.api_base}/publications/v1/balance-delta-high-res/latest"
            
            self.logger.info("Fetching latest TenneT Balance Delta data...")
            self.logger.debug(f"Endpoint: {endpoint}")
            
            response = self.session.get(endpoint, timeout=TIMEOUT_SECONDS)
            
            if response.status_code == 200:
                data = response.json()
                self.logger.info("✓ Received raw data from TenneT")
                return data
            elif response.status_code == 401:
                self.logger.error("Authentication failed (401)")
                return None
            elif response.status_code == 403:
                self.logger.error("Access forbidden (403)")
                return None
            else:
                self.logger.error(f"HTTP {response.status_code}")
                return None
                
        except Exception as e:
            self.logger.error(f"Failed to fetch: {str(e)}")
            return None
    
    def extract_raw_records(self, raw_response: Dict) -> List[Dict]:
        """
        Extract complete raw records from TenneT response
        Preserves ALL fields for later normalization
        
        Args:
            raw_response: Raw API response
        
        Returns:
            List of raw records ready for storage
        """
        records = []
        
        try:
            response = raw_response.get('Response', {})
            timeseries_list = response.get('TimeSeries', [])
            
            if not timeseries_list:
                self.logger.warning("No TimeSeries in response")
                return []
            
            # Extract metadata
            info_type = response.get('informationType', 'UNKNOWN')
            period_interval = response.get('period.timeInterval', {})
            
            # Process each TimeSeries → Period → Point
            for ts_idx, timeseries in enumerate(timeseries_list):
                periods = timeseries.get('Period', [])
                
                for p_idx, period in enumerate(periods):
                    points = period.get('points', [])
                    period_interval_obj = period.get('timeInterval', {})
                    
                    for pt_idx, raw_point in enumerate(points):
                        # Build complete raw record
                        record = self._build_raw_record(
                            raw_point,
                            info_type,
                            period_interval_obj
                        )
                        
                        records.append(record)
            
            self.logger.info(f"Extracted {len(records)} raw records")
            self.raw_records = records
            return records
            
        except Exception as e:
            self.logger.error(f"Failed to extract records: {str(e)}")
            return []
    
    def _build_raw_record(
        self,
        raw_point: Dict,
        info_type: str,
        period_interval: Dict
    ) -> Dict:
        """
        Build a complete raw record for storage
        Includes ALL fields needed for future normalization
        
        Args:
            raw_point: Single point from TenneT
            info_type: Document type (BALANCE_DELTA_HIGH_RES)
            period_interval: Period time window
        
        Returns:
            Raw record ready for database storage
        """
        record = {
            # Timestamps (ALWAYS include both)
            'timestamp_start': raw_point.get('timeInterval_start'),
            'timestamp_end': raw_point.get('timeInterval_end'),
            
            # All power fields (complete for future recalculation)
            'power_afrr_in': self._to_float(raw_point.get('power_afrr_in')),
            'power_afrr_out': self._to_float(raw_point.get('power_afrr_out')),
            'power_igcc_in': self._to_float(raw_point.get('power_igcc_in')),
            'power_igcc_out': self._to_float(raw_point.get('power_igcc_out')),
            'power_mari_in': self._to_float(raw_point.get('power_mari_in')),
            'power_mari_out': self._to_float(raw_point.get('power_mari_out')),
            'power_mfrrda_in': self._to_float(raw_point.get('power_mfrrda_in')),
            'power_mfrrda_out': self._to_float(raw_point.get('power_mfrrda_out')),
            'power_picasso_in': self._to_float(raw_point.get('power_picasso_in')),
            'power_picasso_out': self._to_float(raw_point.get('power_picasso_out')),
            
            # Metadata (useful for analysis/debugging)
            'metadata': {
                'mid_price': self._to_float(raw_point.get('mid_price')),
                'max_upw_regulation_price': self._to_float(raw_point.get('max_upw_regulation_price')),
                'min_downw_regulation_price': self._to_float(raw_point.get('min_downw_regulation_price')),
                'sequence': raw_point.get('sequence'),
                'info_type': info_type,
                'period_start': period_interval.get('start'),
                'period_end': period_interval.get('end')
            },
            
            # Raw JSON (for complete reproducibility)
            'raw_json': raw_point,
            
            # Ingestion metadata
            'ingestion_timestamp': datetime.now(timezone.utc).isoformat() + 'Z',
            'source': 'TenneT',
            'country': COUNTRY,
            'data_type': 'balance_delta_high_res'
        }
        
        return record
    
    def _to_float(self, value) -> Optional[float]:
        """Safely convert value to float"""
        if value is None:
            return None
        try:
            return float(value)
        except (ValueError, TypeError):
            return None
    
    def save_raw_records(
        self,
        output_dir: Optional[Path] = None
    ) -> Optional[str]:
        """
        Save raw records to JSON file (for inspection + backup)
        
        Args:
            output_dir: Directory to save to
        
        Returns:
            File path if successful
        """
        if not output_dir:
            output_dir = LOG_DIR / "collectors" / "tennet_raw"
        
        output_dir.mkdir(parents=True, exist_ok=True)
        
        if not self.raw_records:
            self.logger.debug("No records to save")
            return None
        
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%d_%H%M%S")
        filename = f"tennet_balance_delta_raw_{timestamp}.json"
        filepath = output_dir / filename
        
        try:
            with open(filepath, 'w') as f:
                json.dump(self.raw_records, f, indent=2, default=str)
            
            self.logger.info(f"Saved {len(self.raw_records)} raw records to {filepath}")
            return str(filepath)
            
        except Exception as e:
            self.logger.error(f"Failed to save records: {str(e)}")
            return None
    
    def get_summary(self) -> Dict:
        """
        Get summary of ingestion
        
        Returns:
            Summary statistics
        """
        if not self.raw_records:
            return {'status': 'no_data'}
        
        timestamps = [
            r.get('timestamp_end') or r.get('timestamp_start')
            for r in self.raw_records
        ]
        
        return {
            'status': 'success',
            'records_ingested': len(self.raw_records),
            'timestamp_range': {
                'earliest': min(timestamps) if timestamps else None,
                'latest': max(timestamps) if timestamps else None
            },
            'fields_per_record': {
                'power_fields': len(POWER_FIELDS),
                'metadata_fields': len(METADATA_FIELDS),
                'timestamp_fields': len(TIMESTAMP_FIELDS)
            }
        }

# ========================================================
#   MAIN EXECUTION
# ========================================================

def main():
    """Main entry point"""
    
    logger.info("=" * 70)
    logger.info("SYNCTACLES SparkCrawler — TenneT Balance Delta Ingestor")
    logger.info("=" * 70)
    logger.info("")
    logger.info("Purpose: Fetch and store COMPLETE raw TenneT data")
    logger.info("Strategy: NO normalization at this layer (that's SYNCTACLES job)")
    logger.info("")
    
    # Initialize ingestor
    ingestor = TennetBalanceDeltaIngestor(
        api_base=TENNET_API_BASE,
        api_key=TENNET_API_KEY
    )
    
    # Fetch raw data
    logger.info("Step 1: Fetching raw data from TenneT API...")
    raw_response = ingestor.fetch_latest()
    
    if not raw_response:
        logger.error("Failed to fetch data")
        return 1
    
    # Extract raw records
    logger.info("")
    logger.info("Step 2: Extracting raw records (preserving ALL fields)...")
    records = ingestor.extract_raw_records(raw_response)
    
    if not records:
        logger.error("Failed to extract records")
        return 1
    
    # Save raw records
    logger.info("")
    logger.info("Step 3: Saving raw records for SparkCrawler...")
    saved_file = ingestor.save_raw_records()
    
    if saved_file:
        logger.info(f"Raw records saved: {saved_file}")
    
    # Summary
    logger.info("")
    summary = ingestor.get_summary()
    
    logger.info("=" * 70)
    logger.info("INGESTION SUMMARY")
    logger.info("=" * 70)
    logger.info(f"Status:              {summary['status']}")
    logger.info(f"Records ingested:    {summary['records_ingested']}")
    
    if summary['status'] == 'success':
        logger.info("")
        logger.info("Timestamp range:")
        logger.info(f"  Earliest:        {summary['timestamp_range']['earliest']}")
        logger.info(f"  Latest:          {summary['timestamp_range']['latest']}")
        logger.info("")
        logger.info("Fields preserved per record:")
        logger.info(f"  Power fields:    {summary['fields_per_record']['power_fields']}")
        logger.info(f"  Metadata fields: {summary['fields_per_record']['metadata_fields']}")
        logger.info(f"  Timestamp fields:{summary['fields_per_record']['timestamp_fields']}")
        logger.info("")
        logger.info("📦 This raw data is ready for SYNCTACLES normalizer")
        logger.info("   → normalizer will extract: timestamp + delta_mw")
    
    logger.info("")
    logger.info("=" * 70)
    
    return 0

if __name__ == "__main__":
    try:
        exit_code = main()
        sys.exit(exit_code)
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
        sys.exit(130)
    except Exception as e:
        logger.error(f"Fatal error: {str(e)}", exc_info=True)
        sys.exit(1)