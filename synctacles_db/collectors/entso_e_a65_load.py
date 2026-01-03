#!/usr/bin/env python3
"""
SYNCTACLES SparkCrawler — ENTSO-E A65 Load Parser
Fetches and parses ENTSO-E Load (A65) data
National electricity consumption/demand for Netherlands

Document Type: A65 (Total Load)
Country: Netherlands (NL)
Resolution: Per 15-minute intervals

Author: SYNCTACLES Development
Version: 1.0.0
"""

import os
import sys
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Optional, Dict
import json

import pandas as pd
from entsoe import EntsoeRawClient
from dotenv import load_dotenv

from synctacles_db.core.logging import get_logger

# ========================================================
#   CONFIGURATION
# ========================================================

# Load environment
load_dotenv()

# Fix API Key inconsistency
api_key = os.getenv("ENTSOE_API_KEY") or os.getenv("ENTSO_E_API_KEY")
if not api_key:
    print("ERROR: ENTSOE_API_KEY not found")

# Logging setup
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))
LOG_DIR.mkdir(parents=True, exist_ok=True)

_LOGGER = get_logger(__name__)

# ========================================================
#   CONSTANTS
# ========================================================

COUNTRY_CODE = 'NL'  # Netherlands
DOCUMENT_TYPE = 'A65'  # Total Load document

# ========================================================
#   PARSER CLASS
# ========================================================

class EntsoeLoadParser:
    """
    Fetches and parses ENTSO-E Load (A65) data
    National electricity consumption/demand
    Stores raw responses for later normalization
    """

    def __init__(self, api_key: str, country_code: str = 'NL'):
        """Initialize ENTSO-E client"""
        self.client = EntsoeRawClient(api_key=api_key)
        self.country_code = country_code
        self.logger = _LOGGER
        self.results = {}
    
    def fetch_load(
        self,
        hours_back: int = 24,
        load_type: str = 'total'
    ) -> Optional[str]:
        """
        Fetch load data (consumption)
        
        Args:
            hours_back: How many hours back to fetch
            load_type: 'total' (national), 'actual', 'forecast', etc.
        
        Returns:
            Raw XML response as string, or None if failed
        """
        try:
            # Calculate time range
            end = pd.Timestamp.now(tz='UTC')
            start = end - pd.Timedelta(hours=hours_back)
            
            self.logger.info(
                f"Fetching Load data ({load_type}) "
                f"from {start.strftime('%Y-%m-%d %H:%M')} to {end.strftime('%Y-%m-%d %H:%M')}"
            )
            
            # Query ENTSO-E API for Load
            xml_response = self.client.query_load(
                country_code=self.country_code,
                start=start,
                end=end
            )
            
            if xml_response:
                self.logger.info(
                    f"✓ Received {len(xml_response)} bytes from ENTSO-E (A65)"
                )
                return xml_response
            else:
                self.logger.warning(f"Empty response for Load data")
                return None
                
        except Exception as e:
            self.logger.error(f"Failed to fetch Load data: {str(e)}")
            return None
    
    def fetch_load_forecast(
        self,
        hours_ahead: int = 24
    ) -> Optional[str]:
        """
        Fetch load FORECAST data (Day-ahead)
        
        Args:
            hours_ahead: How many hours ahead to forecast
        
        Returns:
            Raw XML response as string, or None if failed
        """
        try:
            start = pd.Timestamp.now(tz='UTC')
            end = start + pd.Timedelta(hours=hours_ahead)
            
            self.logger.info(
                f"Fetching Load FORECAST "
                f"from {start.strftime('%Y-%m-%d %H:%M')} to {end.strftime('%Y-%m-%d %H:%M')}"
            )
            
            xml_response = self.client.query_load_forecast(
                country_code=self.country_code,
                start=start,
                end=end
            )
            
            if xml_response:
                self.logger.info(
                    f"✓ Received {len(xml_response)} bytes forecast from ENTSO-E"
                )
                return xml_response
            else:
                self.logger.warning(f"Empty forecast response")
                return None
                
        except Exception as e:
            self.logger.error(f"Failed to fetch Load forecast: {str(e)}")
            return None
    
    def save_to_file(
        self,
        output_dir: Optional[Path] = None
    ) -> Dict[str, str]:
        """
        Save all fetched responses to files (for inspection)
        
        Returns:
            Dictionary: {type: file_path}
        """
        if not output_dir:
            output_dir = LOG_DIR / "collectors" / "entso_e_raw"
        
        output_dir.mkdir(parents=True, exist_ok=True)
        timestamp = datetime.now(timezone.utc).strftime("%Y%m%d_%H%M%S")
        
        saved_files = {}
        
        for data_type, xml_data in self.results.items():
            if xml_data:
                filename = f"entso_e_a65_load_{data_type}_{timestamp}.xml"
                filepath = output_dir / filename
                
                try:
                    with open(filepath, 'w') as f:
                        f.write(xml_data)
                    
                    saved_files[data_type] = str(filepath)
                    self.logger.debug(f"Saved {data_type} to {filepath}")
                    
                except Exception as e:
                    self.logger.error(f"Failed to save {data_type}: {str(e)}")
        
        return saved_files
    
    def get_summary(self) -> Dict:
        """
        Get summary of fetched data
        
        Returns:
            Summary statistics
        """
        summary = {
            'timestamp': datetime.now(timezone.utc).isoformat(),
            'document_type': DOCUMENT_TYPE,
            'country': self.country_code,
            'data_types_requested': len(self.results),
            'data_types_succeeded': sum(1 for v in self.results.values() if v),
            'data_types_failed': sum(1 for v in self.results.values() if not v),
            'details': {}
        }
        
        for data_type, xml in self.results.items():
            summary['details'][data_type] = {
                'status': 'success' if xml else 'failed',
                'size_bytes': len(xml) if xml else 0
            }
        
        return summary

# ========================================================
#   MAIN EXECUTION
# ========================================================

def main():
    """Main entry point"""
    _LOGGER.info("ENTSO-E A65 Load Collector starting")
    start_time = time.time()

    try:
        # Initialize parser
        parser = EntsoeLoadParser(api_key=api_key, country_code=COUNTRY_CODE)

        # Fetch Load data (last 24 hours)
        _LOGGER.info(f"Fetching Load (A65) actual data for {COUNTRY_CODE}...")
        load_data = parser.fetch_load(hours_back=24)
        parser.results['actual'] = load_data

        # Fetch Load forecast (next 24 hours)
        _LOGGER.info(f"Fetching Load (A65) forecast data...")
        forecast_data = parser.fetch_load_forecast(hours_ahead=24)
        parser.results['forecast'] = forecast_data

        # Save raw data
        _LOGGER.info("Saving raw XML responses...")
        saved_files = parser.save_to_file()
        _LOGGER.debug(f"Saved {len(saved_files)} files to output directory")

        # Print summary
        summary = parser.get_summary()
        _LOGGER.info(f"A65 collector: {summary['data_types_succeeded']} successful, {summary['data_types_failed']} failed")

        # Return exit code based on success
        elapsed = time.time() - start_time
        if summary['data_types_failed'] > 0:
            _LOGGER.warning(f"A65 collector: {summary['data_types_failed']} data types failed to fetch")
            _LOGGER.info(f"ENTSO-E A65 Load Collector completed with errors in {elapsed:.2f}s")
            return 1
        else:
            _LOGGER.info(f"ENTSO-E A65 Load Collector completed successfully in {elapsed:.2f}s")
            return 0

    except Exception as err:
        elapsed = time.time() - start_time
        _LOGGER.error(f"A65 collector failed after {elapsed:.2f}s: {type(err).__name__}: {err}")
        raise

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
