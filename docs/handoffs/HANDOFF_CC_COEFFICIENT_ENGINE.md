# HANDOFF: Coefficient Engine Setup

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09  
**Classificatie:** VERTROUWELIJK — Niet delen, niet in publieke repo

---

## STRATEGISCHE CONTEXT

### Waarom Coefficient Engine?

**Probleem:**  
SYNCTACLES Energy Action is gebaseerd op ENTSO-E wholesale prijzen. Maar gebruikers betalen consumer prijzen (Enever). Die kunnen significant afwijken. Energy Action kan daardoor verkeerde aanbevelingen geven.

**Oplossing:**  
Een coefficient die de relatie tussen wholesale en consumer prijzen modelleert. Hiermee wordt Energy Action betrouwbaar zonder dat we Enever data 1-op-1 doorgeven (wat juridisch niet mag).

**Formule:**
```
coefficient = consumer_prijs (Enever) / wholesale_prijs (ENTSO-E)
real_price_estimate = ENTSO-E × coefficient
Energy Action = gebaseerd op real_price_estimate
```

### Business Model Impact

**Zonder coefficient:**
- Energy Action = gebaseerd op wholesale → onbetrouwbaar
- BYO (Enever/TenneT) = user ziet echte prijzen → onze data overbodig
- Geen lock-in

**Met coefficient:**
- Energy Action = gebaseerd op echte prijzen → betrouwbaar
- Coefficient logica = proprietary IP → niet te repliceren
- Server-side berekening → geen licentie = geen Energy Action
- Sterke lock-in

### Architectuur

```
┌─────────────────────────────────────────────────────────────┐
│  COEFFICIENT SERVER (91.99.150.36) — VERTROUWELIJK          │
│  ├── Pulls: Enever + TenneT (Leo's keys)                    │
│  ├── Pulls: ENTSO-E (voor vergelijking)                     │
│  ├── Berekent: coefficient (real-time of lookup)            │
│  ├── Slaat op: historische coefficients                     │
│  └── Exposes: GET /coefficient (alleen voor SYNCTACLES)     │
└─────────────────────────────────────────────────────────────┘
                         │
                         │ IP whitelist: alleen SYNCTACLES server
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  SYNCTACLES SERVER (135.181.255.83)                         │
│  ├── Haalt: coefficient op                                  │
│  ├── Fallback: historische coefficient uit eigen DB         │
│  ├── Berekent: Energy Actions                               │
│  └── Serveert: aan HA components                            │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  HOME ASSISTANT (user)                                      │
│  └── Ziet: USE / WAIT / AVOID                               │
│      (weet niet hoe het berekend is)                        │
└─────────────────────────────────────────────────────────────┘
```

### Toekomstige Uitbreiding

Deze server wordt later B2B API met:
- Grotere historische dataset
- Multi-land support (DE, BE, etc.)
- Mogelijk verkoop van coefficient data aan derden

Daarom: PostgreSQL, geen SQLite.

---

## SERVER DETAILS

| Aspect | Waarde |
|--------|--------|
| Provider | Hetzner |
| Type | CX23 |
| IP | 91.99.150.36 |
| OS | Ubuntu 24.04 (verwacht) |
| Doel | Coefficient Engine + toekomstig B2B |

---

## REPOSITORY

| Aspect | Waarde |
|--------|--------|
| Naam | coefficient-engine |
| Organisatie | DATADIO |
| URL | git@github.com:DATADIO/coefficient-engine.git |
| Visibility | **PRIVATE** |

**BELANGRIJK:** Deze repo is gescheiden van synctacles-api om IP te beschermen. Coefficient logica mag nooit in publieke repo terechtkomen.

---

## FASE 1: SERVER SETUP

### 1.1 Test Installer Script

Leo draait het SYNCTACLES installer script op deze server om het te testen. 

**Verwachte aanpassingen nodig:**
- Brand name: `coefficient-engine` of `datadio`
- Database name: `coefficient_db`
- Service names: `coefficient-*`
- Geen TimescaleDB nodig
- Geen HA component

**Documenteer alle fixes** die nodig zijn voor het installer script.

### 1.2 PIA VPN Setup (NL Exit)

Server staat in Duitsland. Om niet verdacht te lijken bij Enever/TenneT, routeer verkeer via NL.

**Leo heeft PIA account.** Installatie:

```bash
# Download PIA installer
wget https://installers.privateinternetaccess.com/download/pia-linux-3.6.1-08339.run

# Maak executable en installeer (headless)
chmod +x pia-linux-3.6.1-08339.run
sudo ./pia-linux-3.6.1-08339.run --headless

# Login (Leo levert credentials)
piactl login

# Connect NL
piactl set region netherlands
piactl connect

# Verify
piactl get connectionstate
curl ifconfig.me  # Moet NL IP tonen

# Auto-connect na reboot
piactl background enable
```

**Test:**
```bash
# Check IP locatie
curl ipinfo.io
# Moet "country": "NL" tonen
```

### 1.3 Basis Structuur

Na installer, clone repo en setup:

```bash
cd /opt/github
git clone git@github.com:DATADIO/coefficient-engine.git
cd coefficient-engine
```

**Structuur:**
```
coefficient-engine/
├── collectors/
│   ├── __init__.py
│   ├── enever_collector.py      # Pulls Enever API
│   ├── tennet_collector.py      # Pulls TenneT API
│   └── entso_collector.py       # Pulls ENTSO-E (voor vergelijking)
├── analysis/
│   ├── __init__.py
│   ├── coefficient_calc.py      # Berekent coefficients
│   └── stability_report.py      # Analyseert stabiliteit
├── api/
│   ├── __init__.py
│   └── main.py                  # FastAPI: GET /coefficient
├── models/
│   ├── __init__.py
│   └── database.py              # SQLAlchemy models
├── config/
│   ├── __init__.py
│   └── settings.py              # Environment config
├── scripts/
│   ├── backfill_entso.py        # Historische ENTSO-E import
│   └── import_enever_csv.py     # Enever CSV import
├── tests/
│   └── __init__.py
├── .env.example
├── .gitignore
├── requirements.txt
├── README.md                    # Basis, geen details (private repo)
└── ARCHITECTURE.md              # Volledige technische docs
```

---

## FASE 2: DATA VERZAMELEN

### 2.1 ENTSO-E Import (HELE EU, 2022-2025)

Leo heeft CSV's gedownload van ENTSO-E File Library. Import ALLE landen (niet alleen NL).

**CSV locatie (Leo levert aan):**
```
/mnt/user-data/uploads/ of via transfer
- 2022_01_EnergyPrices_12_1_D_r3.csv
- 2022_02_EnergyPrices_12_1_D_r3.csv
- ... (alle maanden 2022-2025)
```

**CSV structuur (tab-delimited):**
```
InstanceCode  DateTime(UTC)  ResolutionCode  AreaCode  AreaDisplayName  AreaTypeCode  MapCode  ContractType  Sequence  Price[Currency/MWh]  Currency  UpdateTime(UTC)
```

**Database schema:**
```sql
CREATE TABLE hist_entso_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    area_code TEXT NOT NULL,           -- 10YNL----------L
    area_name TEXT,                    -- Netherlands (NL)
    country_code TEXT NOT NULL,        -- NL
    price_eur_mwh DECIMAL(10,4),
    resolution TEXT,                   -- PT60M
    currency TEXT DEFAULT 'EUR',
    import_timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_hist_entso_timestamp ON hist_entso_prices(timestamp);
CREATE INDEX idx_hist_entso_country ON hist_entso_prices(country_code);
CREATE INDEX idx_hist_entso_area ON hist_entso_prices(area_code);
CREATE UNIQUE INDEX idx_hist_entso_unique ON hist_entso_prices(timestamp, area_code);
```

**Import script:**
```python
# scripts/import_entso_csv.py
"""
Import ENTSO-E historische prijzen uit CSV (hele EU).
"""

import pandas as pd
from sqlalchemy import create_engine
from pathlib import Path
import os

DATABASE_URL = os.getenv('DATABASE_URL')

def import_entso_csv(csv_path: str):
    """Import single ENTSO-E CSV into database."""
    
    print(f"Importing {csv_path}...")
    
    # Read tab-delimited CSV
    df = pd.read_csv(csv_path, sep='\t', encoding='utf-8')
    
    # Rename columns
    df = df.rename(columns={
        'DateTime(UTC)': 'timestamp',
        'AreaCode': 'area_code',
        'AreaDisplayName': 'area_name',
        'MapCode': 'country_code',
        'Price[Currency/MWh]': 'price_eur_mwh',
        'ResolutionCode': 'resolution',
        'Currency': 'currency'
    })
    
    # Select only needed columns
    df = df[['timestamp', 'area_code', 'area_name', 'country_code', 
             'price_eur_mwh', 'resolution', 'currency']]
    
    # Parse timestamp
    df['timestamp'] = pd.to_datetime(df['timestamp'], utc=True)
    
    # Filter alleen EUR (skip andere currencies)
    df = df[df['currency'] == 'EUR']
    
    # Import to database
    engine = create_engine(DATABASE_URL)
    df.to_sql('hist_entso_prices', engine, if_exists='append', index=False)
    
    print(f"  Imported {len(df)} rows")

def import_all_csvs(folder: str):
    """Import all ENTSO-E CSVs from folder."""
    
    csv_files = sorted(Path(folder).glob('*_EnergyPrices_*.csv'))
    
    for csv_file in csv_files:
        import_entso_csv(str(csv_file))
    
    print(f"\nTotal: {len(csv_files)} files imported")

if __name__ == '__main__':
    import sys
    folder = sys.argv[1] if len(sys.argv) > 1 else '/tmp/entso_data'
    import_all_csvs(folder)
```

**Uitvoeren:**
```bash
# Maak schema
psql coefficient_db -f scripts/schema.sql

# Import alle CSVs
python scripts/import_entso_csv.py /path/to/csv/folder

# Verify
psql coefficient_db -c "SELECT country_code, COUNT(*) FROM hist_entso_prices GROUP BY 1 ORDER BY 2 DESC LIMIT 10;"
```

**Verwachte output:**
```
country_code | count
-------------+--------
DE           | ~105000
FR           | ~105000
NL           | ~26000
...
```

---

### 2.2 Enever Import (LATER - wacht op Leo's Supporter toegang)

Leo krijgt toegang tot Enever Supporter downloads. Tot die tijd: ENTSO-E data is voldoende om infrastructuur te testen.

**Status:** ⏳ WACHTEND op Enever CSV

**Wanneer beschikbaar, database schema:**
```sql
CREATE TABLE hist_enever_prices (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    leverancier TEXT NOT NULL,
    price_eur_kwh DECIMAL(10,6),
    import_timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_hist_enever_timestamp ON hist_enever_prices(timestamp);
CREATE INDEX idx_hist_enever_leverancier ON hist_enever_prices(leverancier);
```

**Import script (voorbereid):**
```python
# scripts/import_enever_csv.py
"""
Import Enever historische prijzen uit CSV.
"""

import pandas as pd
from sqlalchemy import create_engine
import os

DATABASE_URL = os.getenv('DATABASE_URL')

def import_enever_csv(csv_path: str):
    """Import Enever CSV into database."""
    
    # Enever CSV formaat nog te bevestigen
    df = pd.read_csv(csv_path)
    
    # Mapping aanpassen zodra we format kennen
    # df = df.rename(columns={...})
    
    engine = create_engine(DATABASE_URL)
    df.to_sql('hist_enever_prices', engine, if_exists='append', index=False)

if __name__ == '__main__':
    import sys
    import_enever_csv(sys.argv[1])
```

---

## FASE 3: COEFFICIENT ANALYSE (Wacht op Enever data)

**Status:** ⏳ BLOCKED tot Enever CSV beschikbaar

Zodra Enever data geïmporteerd is, kan de analyse starten.

### 3.1 Berekening

```python
# analysis/coefficient_calc.py
"""
Bereken coefficient per uur/dag/maand.
coefficient = consumer_price / wholesale_price
"""

def calculate_coefficients():
    """
    Join Enever en ENTSO-E data, bereken coefficient.
    """
    query = """
    SELECT 
        DATE_TRUNC('hour', e.timestamp) AS hour,
        e.leverancier,
        e.price_eur_kwh AS consumer_price,
        a.price_eur_mwh / 1000 AS wholesale_price_kwh,
        CASE 
            WHEN a.price_eur_mwh > 0 
            THEN e.price_eur_kwh / (a.price_eur_mwh / 1000)
            ELSE NULL 
        END AS coefficient
    FROM hist_enever_prices e
    JOIN hist_entso_a44 a 
        ON DATE_TRUNC('hour', e.timestamp) = DATE_TRUNC('hour', a.timestamp)
    WHERE a.price_eur_mwh > 0
    ORDER BY hour
    """
    return pd.read_sql(query, engine)
```

### 3.2 Stabiliteits Analyse

```python
# analysis/stability_report.py
"""
Analyseer hoe stabiel de coefficient is over tijd.
Output bepaalt of we real-time berekening of lookup tabel nodig hebben.
"""

def stability_report(df):
    """
    Genereer stabiliteitsrapport.
    """
    # Per maand
    monthly = df.groupby(df['hour'].dt.to_period('M')).agg({
        'coefficient': ['mean', 'std', 'min', 'max']
    })
    
    # Per uur van de dag
    hourly = df.groupby(df['hour'].dt.hour).agg({
        'coefficient': ['mean', 'std']
    })
    
    # Per dag van de week
    daily = df.groupby(df['hour'].dt.dayofweek).agg({
        'coefficient': ['mean', 'std']
    })
    
    # Spreiding analyse
    avg_std = df['coefficient'].std()
    avg_mean = df['coefficient'].mean()
    cv = (avg_std / avg_mean) * 100  # Coefficient of variation
    
    print(f"""
    STABILITEITSRAPPORT
    ===================
    Gemiddelde coefficient: {avg_mean:.4f}
    Standaard deviatie: {avg_std:.4f}
    Variatiecoefficient: {cv:.2f}%
    
    CONCLUSIE:
    {"✅ STABIEL - Lookup tabel voldoende" if cv < 10 else "⚠️ INSTABIEL - Real-time berekening nodig"}
    """)
    
    return {
        'monthly': monthly,
        'hourly': hourly,
        'daily': daily,
        'cv': cv,
        'stable': cv < 10
    }
```

### 3.3 Uitkomsten

**Als stabiel (CV < 10%):**
```sql
-- Genereer lookup tabel
CREATE TABLE coefficient_lookup (
    month INT,           -- 1-12
    day_type TEXT,       -- 'weekday' / 'weekend'
    hour INT,            -- 0-23
    coefficient DECIMAL(6,4),
    PRIMARY KEY (month, day_type, hour)
);

-- Vul met gemiddelden
INSERT INTO coefficient_lookup
SELECT 
    EXTRACT(MONTH FROM hour) AS month,
    CASE WHEN EXTRACT(DOW FROM hour) IN (0,6) THEN 'weekend' ELSE 'weekday' END AS day_type,
    EXTRACT(HOUR FROM hour) AS hour,
    AVG(coefficient) AS coefficient
FROM coefficient_analysis
GROUP BY 1, 2, 3;
```

**Als instabiel (CV > 10%):**
- Real-time coefficient berekening nodig
- Coefficient server moet Enever blijven pullen

---

## FASE 4: API

### 4.1 Endpoint

```python
# api/main.py
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

app = FastAPI(title="Coefficient Engine", docs_url=None)  # Geen docs publiek

# IP whitelist
ALLOWED_IPS = ["135.181.255.83"]  # SYNCTACLES server

@app.middleware("http")
async def ip_whitelist(request: Request, call_next):
    client_ip = request.client.host
    if client_ip not in ALLOWED_IPS:
        return JSONResponse(status_code=403, content={"detail": "Forbidden"})
    return await call_next(request)

@app.get("/coefficient")
async def get_coefficient(
    hour: int = None,      # 0-23
    month: int = None,     # 1-12
    day_type: str = None   # weekday/weekend
):
    """
    Retourneer coefficient voor gegeven parameters.
    Als geen parameters: huidige coefficient.
    """
    if hour is None:
        hour = datetime.now().hour
    if month is None:
        month = datetime.now().month
    if day_type is None:
        day_type = 'weekend' if datetime.now().weekday() >= 5 else 'weekday'
    
    coefficient = lookup_coefficient(month, day_type, hour)
    
    return {
        "coefficient": coefficient,
        "month": month,
        "day_type": day_type,
        "hour": hour,
        "source": "lookup"  # of "realtime"
    }

@app.get("/health")
async def health():
    return {"status": "ok"}
```

### 4.2 Security

| Aspect | Implementatie |
|--------|---------------|
| Auth | Geen (IP whitelist) |
| Allowed IPs | Alleen 135.181.255.83 (SYNCTACLES) |
| HTTPS | Via nginx (optioneel, intern verkeer) |
| Docs | Disabled (`docs_url=None`) |
| Visibility | Server IP niet publiek maken |

---

## FASE 5: INTEGRATIE MET SYNCTACLES

Nadat coefficient engine draait, moet SYNCTACLES:

1. **Coefficient ophalen:**
```python
# In synctacles-api
async def get_coefficient():
    try:
        response = await httpx.get("http://91.99.150.36:8000/coefficient")
        return response.json()["coefficient"]
    except:
        return get_fallback_coefficient()  # Uit eigen DB
```

2. **Energy Action berekenen:**
```python
def calculate_energy_action(entso_price: float) -> str:
    coefficient = get_coefficient()
    estimated_real_price = entso_price * coefficient
    
    if estimated_real_price < THRESHOLD_LOW:
        return "USE"
    elif estimated_real_price > THRESHOLD_HIGH:
        return "AVOID"
    else:
        return "WAIT"
```

Dit is een **aparte handoff** nadat coefficient engine werkt.

---

## DELIVERABLES

### Fase 1 (Server Setup)
- [ ] Installer script getest en gedocumenteerd
- [ ] Repository gecloned
- [ ] Basis structuur aangemaakt
- [ ] PostgreSQL draait
- [ ] .env geconfigureerd
- [ ] PIA VPN geïnstalleerd en verbonden met NL

### Fase 2 (Data Import)
- [ ] ENTSO-E CSVs ontvangen van Leo
- [ ] Import script werkt
- [ ] ENTSO-E data heel EU (2022-2025) geïmporteerd
- [ ] Data verificatie: counts per land
- [ ] Enever import script voorbereid (wacht op CSV)

### Fase 3 (Analyse) — BLOCKED tot Enever data
- [ ] Enever CSV ontvangen van Leo
- [ ] Enever data geïmporteerd
- [ ] Coefficient berekening werkt
- [ ] Stabiliteitsrapport gegenereerd
- [ ] Conclusie: lookup of real-time?
- [ ] Lookup tabel of real-time logic geïmplementeerd

### Fase 4 (API)
- [ ] FastAPI endpoint werkt
- [ ] IP whitelist actief
- [ ] Systemd service draait
- [ ] Health check OK

### Fase 5 (Integratie)
- [ ] SYNCTACLES haalt coefficient op
- [ ] Fallback werkt
- [ ] Energy Action gebruikt coefficient
- [ ] End-to-end test passed

---

## GIT WORKFLOW

```bash
# Op coefficient server
sudo -u <service-user> git -C /opt/github/coefficient-engine add -A
sudo -u <service-user> git -C /opt/github/coefficient-engine commit -m "message"
sudo -u <service-user> git -C /opt/github/coefficient-engine push
```

**Let op:** Service user naam hangt af van installer script output.

---

## TIJDLIJN

| Dag | Fase | Activiteit |
|-----|------|------------|
| 1 | Setup | Server + installer test + PIA VPN + repo |
| 1-2 | Data | ENTSO-E CSV import (heel EU) |
| - | WACHT | Enever data (Leo levert wanneer beschikbaar) |
| +1 | Analyse | Coefficient berekening + stabiliteitsrapport |
| +2 | API | Endpoint + security |
| +3 | Integratie | SYNCTACLES koppeling |

**Let op:** Fase 3-5 zijn BLOCKED tot Enever data beschikbaar is. CC kan doorgaan met Fase 1-2 en API skeleton (Fase 4).

---

## VERTROUWELIJKHEID

Dit document en alle coefficient logica is **vertrouwelijk**:

- Niet delen buiten dit project
- Niet in publieke repo's
- Niet bespreken in publieke forums
- coefficient-engine repo blijft **PRIVATE**

De coefficient is het kern-IP van SYNCTACLES.

---

## CONTACT

Bij vragen: overleg met Leo of Claude (Opus) via Claude.ai chat.
