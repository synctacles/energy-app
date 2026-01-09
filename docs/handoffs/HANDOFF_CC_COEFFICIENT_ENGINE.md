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

### 1.2 Basis Structuur

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

### 2.1 ENTSO-E Backfill (2022-2025)

ENTSO-E API staat historische queries toe. Backfill A44 prijzen:

```python
# scripts/backfill_entso.py
"""
Backfill ENTSO-E A44 prijsdata 2022-2025.
Rate limit: max 400 requests per minuut.
"""

from datetime import datetime, timedelta
import time

def backfill_a44(start_year=2022, end_year=2025):
    """Fetch all A44 price data from ENTSO-E."""
    
    for year in range(start_year, end_year + 1):
        for month in range(1, 13):
            start = datetime(year, month, 1)
            if month == 12:
                end = datetime(year + 1, 1, 1)
            else:
                end = datetime(year, month + 1, 1)
            
            # Skip future dates
            if start > datetime.now():
                continue
                
            print(f"Fetching {year}-{month:02d}...")
            fetch_and_store_a44(start, end)
            time.sleep(1)  # Rate limiting
```

**Database tabel:**
```sql
CREATE TABLE hist_entso_a44 (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    price_eur_mwh DECIMAL(10,4),
    import_timestamp TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(timestamp)
);

CREATE INDEX idx_hist_entso_a44_timestamp ON hist_entso_a44(timestamp);
```

### 2.2 Enever Import

Leo wordt Enever Supporter en download CSV's. Import script:

```python
# scripts/import_enever_csv.py
"""
Import Enever historische prijzen uit CSV.
CSV formaat: timestamp, leverancier, price_eur_kwh
"""

import pandas as pd
from sqlalchemy import create_engine

def import_enever_csv(csv_path: str, leverancier: str):
    """Import Enever CSV into database."""
    
    df = pd.read_csv(csv_path, parse_dates=['timestamp'])
    df['leverancier'] = leverancier
    
    engine = create_engine(DATABASE_URL)
    df.to_sql('hist_enever_prices', engine, if_exists='append', index=False)
```

**Database tabel:**
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

---

## FASE 3: COEFFICIENT ANALYSE

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

### Fase 1 (Server Setup) ✅ VOLTOOID 2026-01-09
- [x] Installer script getest en gedocumenteerd
- [x] Repository gecloned (`/opt/github/coefficient-engine`)
- [x] Basis structuur aangemaakt
- [x] PostgreSQL 16 draait
- [x] .env geconfigureerd (`/opt/coefficient/.env`)

**Server configuratie:**
| Component | Status |
|-----------|--------|
| Hostname | `coefficient` |
| Service user | `coefficient` |
| PostgreSQL 16 | Running |
| Node Exporter | Running (:9100) |
| Fail2ban | Running |
| Python 3.12 + venv | `/opt/coefficient/venv` |
| API service | Running (:8000) |
| SSH hardening | Root disabled, password auth off |

**Toegang:**
```bash
# Via SSH (met geautoriseerde key)
ssh coefficient@91.99.150.36

# API health check
curl http://91.99.150.36:8000/health

# Service status
sudo systemctl status coefficient-api
```

### Fase 2 (Data)
- [ ] ENTSO-E backfill script werkt
- [ ] ENTSO-E data 2022-2025 geïmporteerd
- [ ] Enever CSV import script werkt
- [ ] Enever data geïmporteerd (wacht op Leo's Supporter toegang)

### Fase 3 (Analyse)
- [ ] Coefficient berekening werkt
- [ ] Stabiliteitsrapport gegenereerd
- [ ] Conclusie: lookup of real-time?
- [ ] Lookup tabel of real-time logic geïmplementeerd

### Fase 4 (API) ✅ VOLTOOID 2026-01-09
- [x] FastAPI endpoint werkt
- [x] IP whitelist actief (135.181.255.83)
- [x] Systemd service draait (`coefficient-api.service`)
- [x] Health check OK

### Fase 5 (Integratie)
- [ ] SYNCTACLES haalt coefficient op
- [ ] Fallback werkt
- [ ] Energy Action gebruikt coefficient
- [ ] End-to-end test passed

---

## GIT WORKFLOW

```bash
# Op coefficient server
sudo -u coefficient git -C /opt/github/coefficient-engine add -A
sudo -u coefficient git -C /opt/github/coefficient-engine commit -m "message"
sudo -u coefficient git -C /opt/github/coefficient-engine push
```

**Service user:** `coefficient`

---

## TIJDLIJN

| Dag | Fase | Activiteit |
|-----|------|------------|
| 1 | Setup | Server + installer test + repo |
| 1-2 | Data | ENTSO-E backfill |
| 2 | Data | Enever import (zodra Leo toegang heeft) |
| 2-3 | Analyse | Coefficient berekening + stabiliteitsrapport |
| 3 | API | Endpoint + security |
| 4 | Integratie | SYNCTACLES koppeling |

**Totaal: 4-5 dagen**

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
