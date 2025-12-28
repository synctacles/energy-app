"""Import Energy-Charts price JSON to raw_prices table."""
import os
import json
from datetime import datetime, timezone
from pathlib import Path
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://synctacles@localhost:5432/synctacles")
LOG_DIR = Path(os.getenv("LOG_PATH", "/var/log/energy-insights"))
INPUT_DIR = LOG_DIR / "collectors" / "energy_charts_raw"

engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)

def import_prices(file_path: Path, country: str = "NL"):
    """Import prices from JSON file to database."""
    with open(file_path) as f:
        data = json.load(f)
    
    unix_seconds = data.get("unix_seconds", [])
    prices = data.get("price", [])
    
    if len(unix_seconds) != len(prices):
        raise ValueError("Mismatch: unix_seconds vs prices length")
    
    session = Session()
    imported = 0
    
    for ts_unix, price in zip(unix_seconds, prices):
        timestamp = datetime.fromtimestamp(ts_unix, tz=timezone.utc)
        
        # Upsert
        session.execute("""
            INSERT INTO raw_prices (timestamp, country, price_eur_mwh, source, source_file)
            VALUES (:ts, :country, :price, 'energy-charts', :file)
            ON CONFLICT (timestamp, country, source) DO UPDATE
            SET price_eur_mwh = EXCLUDED.price_eur_mwh
        """, {
            "ts": timestamp,
            "country": country,
            "price": price,
            "file": str(file_path)
        })
        imported += 1
    
    session.commit()
    session.close()
    
    print(f"✅ Imported {imported} price records from {file_path.name}")

def import_all():
    """Import all unprocessed JSON files."""
    if not INPUT_DIR.exists():
        print(f"❌ Directory not found: {INPUT_DIR}")
        return
    
    files = sorted(INPUT_DIR.glob("prices_NL_*.json"))
    
    for file_path in files:
        try:
            import_prices(file_path)
        except Exception as e:
            print(f"❌ Failed {file_path.name}: {e}")

if __name__ == "__main__":
    import_all()