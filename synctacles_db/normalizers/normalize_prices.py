"""Normalize raw_prices to norm_prices with quality checks."""
import os
from datetime import datetime, timezone, timedelta
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://synctacles@localhost:5432/synctacles")
engine = create_engine(DATABASE_URL)
Session = sessionmaker(bind=engine)

def normalize_prices(country: str = "NL"):
    """Normalize prices with quality status."""
    session = Session()
    now = datetime.now(timezone.utc)
    
    # Get latest raw data
    raw_records = session.execute("""
        SELECT timestamp, price_eur_mwh, fetch_time
        FROM raw_prices
        WHERE country = :country
        AND timestamp >= NOW() - INTERVAL '48 hours'
        ORDER BY timestamp
    """, {"country": country}).fetchall()
    
    normalized = 0
    
    for record in raw_records:
        timestamp, price, fetch_time = record
        
        # Quality check
        age = (now - fetch_time).total_seconds() / 3600
        if age < 1:
            quality = "OK"
        elif age < 25:
            quality = "STALE"
        else:
            quality = "NO_DATA"
        
        # Upsert
        session.execute("""
            INSERT INTO norm_prices (timestamp, country, price_eur_mwh, quality_status)
            VALUES (:ts, :country, :price, :quality)
            ON CONFLICT (timestamp, country) DO UPDATE
            SET price_eur_mwh = EXCLUDED.price_eur_mwh,
                quality_status = EXCLUDED.quality_status,
                normalized_at = NOW()
        """, {
            "ts": timestamp,
            "country": country,
            "price": price,
            "quality": quality
        })
        normalized += 1
    
    session.commit()
    session.close()
    
    print(f"✅ Normalized {normalized} price records")

if __name__ == "__main__":
    normalize_prices()