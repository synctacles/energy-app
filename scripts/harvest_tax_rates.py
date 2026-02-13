#!/usr/bin/env python3
"""
Tax Rate Harvester for Energy-Go
Automatically scrapes official government websites to extract electricity tax rates.

Usage:
    python harvest_tax_rates.py [--country CODE] [--dry-run]

Examples:
    python harvest_tax_rates.py                    # Harvest all missing countries
    python harvest_tax_rates.py --country BG       # Harvest only Bulgaria
    python harvest_tax_rates.py --dry-run          # Preview without saving
"""

import argparse
import json
import os
import re
import sys
from dataclasses import dataclass
from datetime import date, datetime
from pathlib import Path
from typing import Dict, List, Optional

import requests
import yaml
from groq import Groq

# Constants
SCRIPT_DIR = Path(__file__).parent
REPO_ROOT = SCRIPT_DIR.parent
COUNTRIES_DIR = REPO_ROOT / "internal" / "countries" / "data"
CONFIG_FILE = SCRIPT_DIR / "missing_countries.yaml"

# Groq API
GROQ_API_KEY = os.getenv("GROQ_API_KEY")
if not GROQ_API_KEY:
    print("ERROR: GROQ_API_KEY environment variable not set!")
    print("Get your free API key at: https://console.groq.com/keys")
    sys.exit(1)

groq_client = Groq(api_key=GROQ_API_KEY)


@dataclass
class TaxData:
    """Extracted tax rate data."""
    energy_tax: float  # EUR/kWh excluding VAT
    vat_rate: float  # Decimal (0.21 = 21%)
    surcharges: float  # EUR/kWh excluding VAT
    effective_from: str  # YYYY-MM-DD
    source_url: str
    source_quote: str  # Verbatim text proof
    confidence: str  # high/medium/low
    currency: str = "EUR"


def load_config() -> dict:
    """Load missing countries configuration."""
    with open(CONFIG_FILE) as f:
        return yaml.safe_load(f)


def fetch_url(url: str, timeout: int = 15) -> str:
    """Fetch HTML content from URL."""
    headers = {
        "User-Agent": "SynctaclesEnergyHarvester/1.0 (Tax Rate Research)"
    }
    try:
        response = requests.get(url, headers=headers, timeout=timeout)
        response.raise_for_status()
        return response.text
    except requests.RequestException as e:
        raise Exception(f"Failed to fetch {url}: {e}")


def extract_tax_rates_with_ai(html: str, url: str, country_code: str, currency: str) -> TaxData:
    """
    Extract tax rates from HTML using Groq AI.

    Args:
        html: HTML content of government webpage
        url: Source URL
        country_code: ISO country code (e.g., "BG")
        currency: Local currency code (e.g., "BGN", "EUR")

    Returns:
        TaxData object with extracted information
    """
    # Truncate HTML to fit in context (keep first 15k chars)
    html_snippet = html[:15000]

    prompt = f"""You are extracting electricity tax information for {country_code} from an official government website.

Extract the following information and return ONLY valid JSON (no markdown, no code blocks):

{{
  "energy_tax_per_kwh": 0.0062,     // Energy excise tax in {currency}/kWh
  "vat_rate_decimal": 0.25,         // VAT rate as decimal (25% = 0.25)
  "surcharges_per_kwh": 0.0,        // Additional levies/surcharges in {currency}/kWh
  "effective_from": "2026-01-01",   // When these rates apply (YYYY-MM-DD)
  "source_quote": "exact text...",  // Verbatim snippet proving this (max 200 chars)
  "confidence": "high"              // high/medium/low based on clarity
}}

CRITICAL RULES:
1. Energy tax and surcharges must EXCLUDE VAT (pre-tax rates)
2. Convert rates to per kWh (if given per MWh, divide by 1000)
3. VAT must be decimal format (19% = 0.19, not 19)
4. Source quote must be verbatim text from the page
5. If uncertain, use confidence: "medium" or "low"
6. If you cannot find clear data, return confidence: "low" with best estimates

Website URL: {url}
Currency: {currency}

HTML content:
{html_snippet}

Return ONLY the JSON object, nothing else:"""

    try:
        response = groq_client.chat.completions.create(
            model="llama-3.3-70b-versatile",  # Updated model (3.1 deprecated)
            messages=[{"role": "user", "content": prompt}],
            temperature=0.1,  # Low temperature for deterministic extraction
            max_tokens=500
        )

        content = response.choices[0].message.content.strip()

        # Remove markdown code blocks if present
        content = re.sub(r'^```json\s*', '', content)
        content = re.sub(r'\s*```$', '', content)
        content = content.strip()

        data = json.loads(content)

        return TaxData(
            energy_tax=float(data["energy_tax_per_kwh"]),
            vat_rate=float(data["vat_rate_decimal"]),
            surcharges=float(data["surcharges_per_kwh"]),
            effective_from=data["effective_from"],
            source_url=url,
            source_quote=data["source_quote"],
            confidence=data["confidence"],
            currency=currency
        )

    except json.JSONDecodeError as e:
        raise Exception(f"AI returned invalid JSON: {content[:200]}")
    except Exception as e:
        raise Exception(f"AI extraction failed: {e}")


def convert_to_eur(amount: float, currency: str, exchange_rate: Optional[float]) -> float:
    """Convert amount to EUR if not already in EUR."""
    if currency == "EUR":
        return amount
    if exchange_rate is None:
        raise Exception(f"Missing exchange rate for {currency}")
    return amount / exchange_rate


def harvest_country(country: dict, dry_run: bool = False) -> Optional[TaxData]:
    """
    Harvest tax data for a single country.

    Args:
        country: Country config dict
        dry_run: If True, don't save files

    Returns:
        TaxData if successful, None otherwise
    """
    code = country["code"]
    name = country["name"]
    currency = country.get("currency", "EUR")
    exchange_rate = country.get("typical_exchange_rate")
    urls = country["tax_authority_urls"]

    print(f"\n{'='*60}")
    print(f"Processing {code} ({name})")
    print(f"{'='*60}")

    best_result = None
    best_confidence_score = 0

    confidence_scores = {"high": 3, "medium": 2, "low": 1}

    for i, url in enumerate(urls, 1):
        print(f"\n[{i}/{len(urls)}] Fetching: {url}")

        try:
            # Fetch HTML
            html = fetch_url(url)
            print(f"  ✓ Fetched {len(html):,} bytes")

            # Extract with AI
            print(f"  ⏳ Extracting with Groq AI...")
            tax_data = extract_tax_rates_with_ai(html, url, code, currency)

            # Convert to EUR if needed
            if currency != "EUR":
                tax_data.energy_tax = convert_to_eur(tax_data.energy_tax, currency, exchange_rate)
                tax_data.surcharges = convert_to_eur(tax_data.surcharges, currency, exchange_rate)
                print(f"  💱 Converted {currency} → EUR (rate: {exchange_rate})")

            # Display results
            print(f"  ✓ Extracted (confidence: {tax_data.confidence})")
            print(f"     Energy tax: €{tax_data.energy_tax:.5f}/kWh (excl. VAT)")
            print(f"     VAT rate: {tax_data.vat_rate*100:.1f}%")
            print(f"     Surcharges: €{tax_data.surcharges:.5f}/kWh")
            print(f"     Effective: {tax_data.effective_from}")
            print(f"     Quote: \"{tax_data.source_quote[:100]}...\"")

            # Track best result
            score = confidence_scores.get(tax_data.confidence, 0)
            if score > best_confidence_score:
                best_confidence_score = score
                best_result = tax_data

            # If high confidence, we're done
            if tax_data.confidence == "high":
                print(f"  🎯 High confidence achieved, using this result")
                break

        except Exception as e:
            print(f"  ✗ Failed: {e}")
            continue

    if best_result is None:
        print(f"\n❌ {code}: No usable data extracted")
        return None

    if not dry_run:
        # Save YAML file
        save_country_yaml(country, best_result)
        print(f"\n✅ {code}: Successfully harvested and saved!")
    else:
        print(f"\n✅ {code}: Would save (dry-run mode)")

    return best_result


def save_country_yaml(country: dict, tax_data: TaxData):
    """Generate and save YAML file for country."""
    code = country["code"].lower()
    name = country["name"]
    timezone = country["timezone"]

    yaml_content = f"""country: {code.upper()}
name: {name}
currency: EUR
zones:
  - code: "{code.upper()}"
    name: "{name}"
    timezone: "{timezone}"
tax_profile:
  vat_rate: {tax_data.vat_rate}
  coefficient: 1.0
  energy_tax:
    - from: "{tax_data.effective_from}"
      rate: {tax_data.energy_tax:.5f}  # EUR/kWh excl. VAT
  surcharges: {tax_data.surcharges:.5f}  # EUR/kWh excl. VAT
sources:
  - name: energycharts
    priority: 1

# AI-extracted on {date.today()} from:
# {tax_data.source_url}
# Confidence: {tax_data.confidence}
# Source quote: "{tax_data.source_quote[:100]}..."
"""

    output_file = COUNTRIES_DIR / f"{code}.yaml"
    output_file.write_text(yaml_content)
    print(f"  💾 Saved: {output_file}")


def main():
    parser = argparse.ArgumentParser(description="Harvest electricity tax rates from government websites")
    parser.add_argument("--country", help="Harvest specific country only (e.g., BG)")
    parser.add_argument("--dry-run", action="store_true", help="Preview without saving files")
    parser.add_argument("--eu-only", action="store_true", help="Only harvest EU countries (priority)")
    args = parser.parse_args()

    print("="*60)
    print("Energy Tax Rate Harvester")
    print("="*60)
    print(f"Groq API: {'✓ Configured' if GROQ_API_KEY else '✗ Missing'}")
    print(f"Output: {COUNTRIES_DIR}")
    print(f"Mode: {'DRY RUN (no files saved)' if args.dry_run else 'LIVE (will save files)'}")

    # Load config
    config = load_config()
    countries = config["missing_countries"]

    # Filter if needed
    if args.country:
        countries = [c for c in countries if c["code"] == args.country.upper()]
        if not countries:
            print(f"\n❌ Country {args.country} not found in config")
            return

    if args.eu_only:
        # EU countries have EUR currency or are in EU
        eu_codes = {"BG", "CY", "EE", "GR", "HR", "IE", "LT", "LV", "RO", "SK"}
        countries = [c for c in countries if c["code"] in eu_codes]

    print(f"Countries to process: {len(countries)}")

    # Harvest
    results = []
    for country in countries:
        result = harvest_country(country, dry_run=args.dry_run)
        if result:
            results.append((country["code"], result))

    # Summary
    print("\n" + "="*60)
    print("SUMMARY")
    print("="*60)
    print(f"Successfully harvested: {len(results)}/{len(countries)} countries")

    if results:
        print("\nResults:")
        for code, data in results:
            print(f"  ✓ {code}: tax={data.energy_tax:.5f}, VAT={data.vat_rate*100:.0f}%, confidence={data.confidence}")

    if not args.dry_run and results:
        print(f"\n💾 Generated {len(results)} YAML files in: {COUNTRIES_DIR}")
        print("\nNext steps:")
        print("  1. Review the generated YAML files")
        print("  2. git add internal/countries/data/*.yaml")
        print("  3. git commit -m 'feat: add {len(results)} countries via AI harvesting'")
        print("  4. Test with: go test ./...")


if __name__ == "__main__":
    main()
