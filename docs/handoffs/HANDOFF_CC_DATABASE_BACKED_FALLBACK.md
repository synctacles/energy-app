# HANDOFF: Database-Backed Fallback Chain

**Van:** Claude Code
**Naar:** Volgende ontwikkelaar / Claude Code
**Datum:** 2026-01-13
**Status:** GEIMPLEMENTEERD
**Classificatie:** TECHNISCH

---

## EXECUTIVE SUMMARY

De prijzen fallback chain is omgebouwd van real-time API calls naar een **database-backed** architectuur. Prijzen worden nu 2x per dag verzameld en lokaal opgeslagen, waardoor de API minder afhankelijk is van externe services.

### Architectuur Wijziging

**OUD (real-time):**
```
API Request → Frank API (real-time) → Response
            ↳ Coefficient Server (fallback)
```

**NIEUW (database-backed):**
```
Systemd Timer (07:00, 15:00 UTC)
    ├── Frank Collector → frank_prices (local DB)
    └── Enever Collector → enever_frank_prices (local DB)

API Request → frank_prices (local) → Response
            ↳ enever_frank_prices (fallback)
            ↳ Frank Direct API (emergency)
```

---

## 1. NIEUWE FALLBACK ARCHITECTUUR

### 1.1 7-Tier Fallback Chain (Database-First)

| Tier | Bron | Type | GO Actie | Data Age Limit |
|------|------|------|----------|----------------|
| **1** | **Frank DB** (local) | LIVE consumer | JA | < 6 uur |
| **2** | **Enever-Frank DB** (local) | LIVE consumer | JA | < 6 uur |
| **3** | Frank Direct API | LIVE consumer (emergency) | JA | n.v.t. |
| **4** | ENTSO-E Fresh + Model | CALCULATED | JA | < 15 min |
| **5** | ENTSO-E Stale + Model | CALCULATED | JA | < 60 min |
| **6** | Energy-Charts + Model | FALLBACK | NEE | - |
| **7** | Cache (memory/PostgreSQL) | CACHED | NEE | - |

### 1.2 GO Action Freshness Rule

**Kritisch:** GO actie is alleen toegestaan als data jonger is dan **6 uur**.

```python
MAX_DATA_AGE_HOURS = 6  # In fallback_manager.py

def _check_data_freshness(data_age_minutes: int) -> bool:
    max_age_minutes = MAX_DATA_AGE_HOURS * 60  # 360 min
    return data_age_minutes < max_age_minutes
```

---

## 2. DATABASE SCHEMA

### 2.1 frank_prices (Tier 1)

```sql
CREATE TABLE frank_prices (
    timestamp     TIMESTAMPTZ PRIMARY KEY,
    price_eur_kwh NUMERIC(10, 6) NOT NULL,
    market_price      NUMERIC(10, 6),  -- Wholesale component
    market_price_tax  NUMERIC(10, 6),  -- BTW
    sourcing_markup   NUMERIC(10, 6),  -- Inkoopkosten
    energy_tax        NUMERIC(10, 6),  -- Energiebelasting
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_frank_prices_timestamp ON frank_prices (timestamp DESC);
```

### 2.2 enever_frank_prices (Tier 2)

```sql
CREATE TABLE enever_frank_prices (
    timestamp     TIMESTAMPTZ PRIMARY KEY,
    price_eur_kwh NUMERIC(10, 6) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_enever_frank_prices_timestamp ON enever_frank_prices (timestamp DESC);
```

---

## 3. COLLECTOR SCRIPTS

### 3.1 Frank Collector

**Bestand:** `scripts/collectors/frank_collector.py`

```bash
# Handmatig draaien
python scripts/collectors/frank_collector.py --tomorrow

# Exit codes:
# 0: Success
# 1: No prices collected
# 2: Database error
```

**Functionaliteit:**
- Haalt prijzen direct van Frank Energie GraphQL API
- Slaat alle 4 prijscomponenten op
- UPSERT logica (update bij conflict)
- `--tomorrow` flag voor day-ahead prijzen (na 14:00 CET)

### 3.2 Enever-Frank Collector

**Bestand:** `scripts/collectors/enever_frank_collector.py`

```bash
# Handmatig draaien
python scripts/collectors/enever_frank_collector.py --tomorrow
```

**Functionaliteit:**
- Haalt Frank prijzen via Coefficient Server (Enever data)
- Fallback voor wanneer Frank Direct niet werkt
- Zelfde UPSERT logica

---

## 4. SYSTEMD TIMERS

### 4.1 Timer Schema

| Timer | Service | Tijden (UTC) |
|-------|---------|--------------|
| `synctacles-frank-collector.timer` | Frank collector | 07:00, 15:00 |
| `synctacles-enever-frank-collector.timer` | Enever collector | 07:05, 15:05 |

### 4.2 Template Bestanden

```
systemd/
├── synctacles-frank-collector.service.template
├── synctacles-frank-collector.timer.template
├── synctacles-enever-frank-collector.service.template
└── synctacles-enever-frank-collector.timer.template
```

### 4.3 Deployment

Na deployment moeten de timers ingeschakeld worden:

```bash
# Kopieer templates naar /etc/systemd/system/ met variabele substitutie
# Enable en start timers
sudo systemctl enable synctacles-frank-collector.timer
sudo systemctl enable synctacles-enever-frank-collector.timer
sudo systemctl start synctacles-frank-collector.timer
sudo systemctl start synctacles-enever-frank-collector.timer

# Controleer status
systemctl list-timers | grep synctacles
```

---

## 5. FALLBACK MANAGER WIJZIGINGEN

### 5.1 Nieuwe Methodes

**`_get_frank_from_db()`** - Leest Frank prijzen uit lokale DB
**`_get_enever_frank_from_db()`** - Leest Enever-Frank prijzen uit lokale DB
**`_check_data_freshness()`** - Controleert of data vers genoeg is voor GO actie

### 5.2 Gewijzigde Tier Volgorde

```python
async def get_prices_with_fallback(...):
    # TIER 1: Frank DB (local)
    frank_prices, frank_age = await FallbackManager._get_frank_from_db()
    if frank_prices:
        allow_go = FallbackManager._check_data_freshness(frank_age)
        return (frank_prices, "Frank DB", quality, allow_go)

    # TIER 2: Enever-Frank DB (local)
    enever_prices, enever_age = await FallbackManager._get_enever_frank_from_db()
    if enever_prices:
        allow_go = FallbackManager._check_data_freshness(enever_age)
        return (enever_prices, "Enever-Frank DB", quality, allow_go)

    # TIER 3: Frank Direct API (emergency fallback)
    frank_direct = await FrankEnergieClient.get_prices_today()
    ...
```

---

## 6. BESTANDEN GEWIJZIGD/TOEGEVOEGD

| Bestand | Status | Beschrijving |
|---------|--------|--------------|
| `synctacles_db/models.py` | GEWIJZIGD | FrankPrices, EneverFrankPrices models |
| `synctacles_db/fallback/fallback_manager.py` | GEWIJZIGD | DB read methods, nieuwe tier volgorde |
| `scripts/collectors/frank_collector.py` | **NIEUW** | Frank price collector |
| `scripts/collectors/enever_frank_collector.py` | **NIEUW** | Enever-Frank collector |
| `systemd/synctacles-frank-collector.*` | **NIEUW** | Systemd service + timer |
| `systemd/synctacles-enever-frank-collector.*` | **NIEUW** | Systemd service + timer |
| `alembic/versions/20260113_add_frank_prices_tables.py` | **NIEUW** | Migratie (tabellen handmatig aangemaakt) |
| `requirements.txt` | GEWIJZIGD | asyncpg==0.30.0 toegevoegd |
| `scripts/test/test_db_fallback.py` | **NIEUW** | Test script |

---

## 7. VALIDATIE

### 7.1 Test Resultaten (2026-01-13)

```
============================================================
DATABASE-BACKED FALLBACK CHAIN - TEST SUITE
============================================================

[TEST 1] Frank Collector - API Fetch
  PASS: Fetched 24 prices from Frank API

[TEST 2] FallbackManager Database Methods
  PASS: Frank DB has 23 prices, age 1 min
  PASS: Enever-Frank DB has 24 prices

[TEST 3] Full Fallback Chain
  PASS: Got 23 prices
  Source: Frank DB
  Quality: FRESH
  GO Allowed: True

TEST SUMMARY: Passed: 3/3 - ALL TESTS PASSED
```

### 7.2 Database Staat

```sql
-- Huidige data
SELECT table_name, COUNT(*) as records
FROM (
    SELECT 'frank_prices' as table_name, * FROM frank_prices
    UNION ALL
    SELECT 'enever_frank_prices', * FROM enever_frank_prices
) t GROUP BY table_name;

-- Resultaat:
-- frank_prices        | 24 records
-- enever_frank_prices | 24 records
```

---

## 8. VOORDELEN

| Aspect | Oud (Real-time) | Nieuw (DB-backed) |
|--------|-----------------|-------------------|
| Latency | ~300-500ms | ~10-50ms |
| Externe dependencies | Bij elke request | 2x per dag |
| Failure resilience | Laag | Hoog (data cached) |
| API rate limits | Risk | Geen impact |
| Frank IP blocking | Directe impact | 6 uur buffer |

---

## 9. VOLGENDE STAPPEN

### Nog Te Doen

1. **Systemd timers deployen** op productie server
2. **Monitoring instellen** voor collector failures
3. **Fase 2 (Remote):** Coefficient API endpoint `/internal/enever/frank`
   - Nog niet nodig zolang `/internal/consumer/prices` werkt

### Deployment Checklist

- [ ] Pull latest code
- [ ] Run `pip install -r requirements.txt` (voor asyncpg)
- [ ] Verify tabellen bestaan in database
- [ ] Deploy systemd timer templates
- [ ] Enable en start timers
- [ ] Monitor logs voor eerste collector runs

---

## CHANGELOG

| Datum | Wijziging |
|-------|-----------|
| 2026-01-13 | Database schema aangemaakt |
| 2026-01-13 | Frank en Enever collectors geïmplementeerd |
| 2026-01-13 | FallbackManager omgebouwd naar DB-first |
| 2026-01-13 | Systemd timer templates toegevoegd |
| 2026-01-13 | Alle tests geslaagd |
