"""
Energy-Charts Day-Ahead Price Collector
Fetches NL electricity prices from Fraunhofer ISE API
"""
import os
import json
import requests
from datetime import datetime, timedelta, timezone
from pathlib import Path

BASE_URL = "https://api.energy-charts.info/price"
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))
OUTPUT_DIR = LOG_DIR / "collectors" / "energy_charts_raw"

def fetch_prices(country: str = "NL", days: int = 2) -> dict:
    """Fetch day-ahead prices for country."""
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    
    today = datetime.now(timezone.utc).date()
    start = today.isoformat()
    end = (today + timedelta(days=days)).isoformat()
    
    params = {"bzn": country, "start": start, "end": end}
    
    response = requests.get(BASE_URL, params=params, timeout=30)
    response.raise_for_status()
    
    data = response.json()
    
    # Save raw response
    timestamp = datetime.now(timezone.utc).strftime("%Y%m%d_%H%M%S")
    output_file = OUTPUT_DIR / f"prices_{country}_{timestamp}.json"
    
    with open(output_file, "w") as f:
        json.dump(data, f, indent=2)
    
    print(f"✅ Fetched {len(data.get('price', []))} price points")
    print(f"📁 Saved: {output_file}")
    
    return data

if __name__ == "__main__":
    fetch_prices()