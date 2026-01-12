#!/usr/bin/env python3
"""
Download historische Frank Energie prijzen naar CSV.

Data range: 2023-01-01 tot vandaag (Frank API heeft geen data voor 2022)
Output: /opt/coefficient/data/frank-historisch/

Gebruik:
    python download_frank_historical.py
    python download_frank_historical.py --output /custom/path/

API Endpoint: https://graphql.frankenergie.nl/
Authenticatie: Geen (publiek toegankelijk)
"""

import argparse
import csv
import os
import sys
import time
from datetime import datetime, timedelta
from pathlib import Path

try:
    import requests
except ImportError:
    print("ERROR: requests module not installed. Run: pip install requests")
    sys.exit(1)

DEFAULT_OUTPUT_DIR = Path("/opt/coefficient/data/frank-historisch")
ENDPOINT = "https://graphql.frankenergie.nl/"
START_YEAR = 2023  # Frank API heeft geen data voor 2022


def fetch_prices(date_str: str) -> list | None:
    """Fetch prices for a single date from Frank Energie API."""
    query = """query MarketPrices($date: String!) {
        marketPrices(date: $date) {
            electricityPrices {
                from
                till
                marketPrice
                marketPriceTax
                sourcingMarkupPrice
                energyTaxPrice
            }
        }
    }"""

    try:
        response = requests.post(
            ENDPOINT,
            json={"query": query, "variables": {"date": date_str}},
            headers={"Content-Type": "application/json"},
            timeout=15,
        )
        data = response.json()

        if "data" in data and data["data"] and data["data"].get("marketPrices"):
            return data["data"]["marketPrices"].get("electricityPrices", [])
        return []
    except Exception as e:
        print(f"  ERROR {date_str}: {e}")
        return None


def download_month(year: int, month: int, output_dir: Path) -> bool:
    """Download one month of data and save to CSV."""
    start_date = datetime(year, month, 1)

    # Calculate end of month
    if month == 12:
        end_date = datetime(year + 1, 1, 1) - timedelta(days=1)
    else:
        end_date = datetime(year, month + 1, 1) - timedelta(days=1)

    filename = output_dir / f"frank_electricity_{year}_{month:02d}.csv"

    # Skip if already exists and has data
    if filename.exists() and filename.stat().st_size > 1000:
        print(f"  Skipping {year}-{month:02d} (already exists)")
        return True

    print(f"  Downloading {year}-{month:02d}...")
    all_records = []

    current = start_date
    while current <= end_date:
        date_str = current.strftime("%Y-%m-%d")
        prices = fetch_prices(date_str)

        if prices is None:
            # Error occurred, retry once
            time.sleep(1)
            prices = fetch_prices(date_str)

        if prices:
            all_records.extend(prices)
        elif prices is None:
            print(f"    Failed: {date_str}")

        current += timedelta(days=1)
        time.sleep(0.05)  # Rate limiting

    if not all_records:
        print(f"    No data for {year}-{month:02d}")
        return False

    # Write CSV
    with open(filename, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(
            [
                "timestamp_from",
                "timestamp_till",
                "market_price_eur",
                "market_tax_eur",
                "sourcing_markup_eur",
                "energy_tax_eur",
                "total_price_eur",
            ]
        )

        for r in all_records:
            total = (
                r["marketPrice"]
                + r["marketPriceTax"]
                + r["sourcingMarkupPrice"]
                + r["energyTaxPrice"]
            )
            writer.writerow(
                [
                    r["from"],
                    r["till"],
                    f"{r['marketPrice']:.6f}",
                    f"{r['marketPriceTax']:.6f}",
                    f"{r['sourcingMarkupPrice']:.6f}",
                    f"{r['energyTaxPrice']:.6f}",
                    f"{total:.6f}",
                ]
            )

    print(f"    Saved {len(all_records)} records to {filename.name}")
    return True


def main():
    parser = argparse.ArgumentParser(
        description="Download historische Frank Energie prijzen naar CSV",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Voorbeelden:
  python download_frank_historical.py
  python download_frank_historical.py --output /tmp/frank-data/

CSV formaat:
  timestamp_from, timestamp_till, market_price_eur, market_tax_eur,
  sourcing_markup_eur, energy_tax_eur, total_price_eur

Notities:
  - Data beschikbaar vanaf 2023-01-01 (geen 2022 data in Frank API)
  - Bestaande bestanden worden overgeslagen
  - ~720 records per maand (24 uur x ~30 dagen)
        """,
    )
    parser.add_argument(
        "--output",
        "-o",
        type=Path,
        default=DEFAULT_OUTPUT_DIR,
        help=f"Output directory (default: {DEFAULT_OUTPUT_DIR})",
    )

    args = parser.parse_args()
    output_dir = args.output

    print("=" * 60)
    print("FRANK ENERGIE HISTORISCHE DATA DOWNLOAD")
    print("=" * 60)
    print(f"Output directory: {output_dir}")
    print(f"Data range: {START_YEAR}-01 tot heden")
    print(f"API endpoint: {ENDPOINT}")
    print("")

    output_dir.mkdir(parents=True, exist_ok=True)

    # Download from START_YEAR-01 to current month
    today = datetime.now()

    months_to_download = []
    current = datetime(START_YEAR, 1, 1)
    while current <= today:
        months_to_download.append((current.year, current.month))
        if current.month == 12:
            current = datetime(current.year + 1, 1, 1)
        else:
            current = datetime(current.year, current.month + 1, 1)

    print(f"Months to process: {len(months_to_download)}")
    print("")

    success = 0
    failed = 0

    for year, month in months_to_download:
        if download_month(year, month, output_dir):
            success += 1
        else:
            failed += 1
        time.sleep(0.2)  # Between months

    print("")
    print("=" * 60)
    print(f"COMPLETE: {success} months downloaded, {failed} failed")
    print("=" * 60)

    # List files
    print("")
    print("Downloaded files:")
    total_records = 0
    for f in sorted(output_dir.glob("frank_electricity_*.csv")):
        size_kb = f.stat().st_size / 1024
        with open(f) as csvfile:
            records = sum(1 for _ in csvfile) - 1  # Minus header
            total_records += records
        print(f"  {f.name}: {size_kb:.1f} KB ({records} records)")

    print("")
    print(f"Total: {total_records} price records")

    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
