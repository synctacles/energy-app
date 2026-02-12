# Hybrid Architecture — Synctacles Energy v1.3.0

## Overview

The energy addon uses a **hybrid architecture**: a central price server as primary source, with the full client-side collector chain as emergency fallback.

```
Normal:    Addon → Synctacles Server → pre-computed consumer prices
                   (1 API call)

Server down: Addon → [Energy-Charts, aWATTar, EnergiDataService, ...]
                     (existing fallback chain, direct to third-party APIs)

All down:  Addon → SQLite cache (48h)
                   → WAIT action
```

## Components

### energy-addon (HA addon binary)
- **Location**: `cmd/energy-addon/main.go`
- **Role**: Runs inside Home Assistant, publishes HA sensors
- **Source chain**:
  - Tier 0: Synctacles API (central server, `https://energy.synctacles.com`)
  - Tier 1+: Direct third-party sources (fallback only)
  - Tier N: SQLite cache (48h retention)
- **Key change in v1.3.0**: Synctacles API prepended as first source in chain

### energy-server (central proxy binary)
- **Location**: `cmd/energy-server/main.go`
- **Role**: Polls all 30 EU zones, normalizes prices, serves REST API
- **Stack**: Go binary + PostgreSQL
- **Collectors**: Reuses same `internal/collector/` code as addon
- **API**: `GET /api/v1/prices?zone=NL&date=2026-02-12`

### Shared code (`internal/`)
Both binaries share:
- `collector/` — All 7 price source implementations
- `engine/` — FallbackManager, ActionEngine, Normalizer
- `models/` — HourlyPrice, ZoneInfo, TaxProfile
- `countries/` — 30 zone configs with tax profiles

## Data flow

### Server side (every 15 minutes)
```
30 zones → 6 source APIs (concurrent, max 5 parallel)
         → Normalizer (wholesale → consumer per country)
         → PostgreSQL (upsert)
         → In-memory cache (for API serving)
```

### Addon side (every 15 minutes)
```
GET /api/v1/prices?zone=NL → JSON response
    → SensorData → HA sensors
    ↓ (on failure, circuit breaker opens)
Direct: Energy-Charts, EasyEnergy, Frank, etc.
    → Normalizer → SensorData → HA sensors
    ↓ (all fail)
SQLite cache (48h) → SensorData → HA sensors (WAIT action)
```

## 429 Rate Limit Handling

All HTTP clients now handle 429 Too Many Requests:

1. `Retry-After` header is parsed (seconds or HTTP-date format)
2. Circuit breaker opens for the specified duration (instead of fixed 2h)
3. Addon falls back to next source immediately
4. Source auto-recovers when Retry-After expires

This protects Energy-Charts during mass fallback events (e.g., Synctacles server outage where thousands of addons simultaneously fall back to direct polling).

## API Reference

### `GET /api/v1/prices`
Returns pre-computed consumer prices for a zone.

**Parameters:**
- `zone` (required): Bidding zone code (e.g., `NL`, `DE-LU`, `NO1`)
- `date` (optional): Filter by date (`YYYY-MM-DD`), default: today + tomorrow

**Response:**
```json
{
  "zone": "NL",
  "source": "easyenergy",
  "quality": "live",
  "prices": [
    {
      "timestamp": "2026-02-12T00:00:00Z",
      "price_eur": 0.2834,
      "unit": "EUR/kWh",
      "is_consumer": true
    }
  ]
}
```

### `GET /api/v1/zones`
Returns all supported zone codes.

### `GET /api/v1/health`
Health check endpoint.

## Deployment

### Server (ENERGY-PROD)
```bash
# Build
cd /opt/github/energy-go
CGO_ENABLED=0 go build -o dist/energy-server ./cmd/energy-server

# Deploy
scp dist/energy-server energy-prod:/opt/energy-server/
ssh energy-prod 'sudo systemctl restart energy-server'
```

**Environment variables:**
- `DATABASE_URL` (required): PostgreSQL connection string
- `LISTEN_ADDR` (default `:8080`): HTTP listen address
- `DEBUG_MODE` (default `false`): Enable debug logging

**Systemd unit:** See `deploy/server/energy-server.service`

### Addon (HA)
Standard addon build + deploy (see MEMORY.md for procedure).
The `SYNCTACLES_URL` defaults to `https://energy.synctacles.com`.

## Database Schema

```sql
CREATE TABLE prices (
    zone        TEXT        NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    price_eur   NUMERIC(10,6) NOT NULL,
    unit        TEXT        NOT NULL DEFAULT 'kWh',
    source      TEXT        NOT NULL,
    quality     TEXT        NOT NULL DEFAULT 'live',
    is_consumer BOOLEAN     NOT NULL DEFAULT false,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (zone, timestamp)
);
```

**Storage projection:**
- 30 zones × 24h × 365d = 262,800 rows/year
- ~54 MB/year (with indexes)
- 7-day retention for served data; historical data optional

## Zones Covered

30 EU bidding zones across 18 countries:

| Country | Zones | Primary Source |
|---------|-------|----------------|
| Austria | AT | aWATTar |
| Belgium | BE | Energy-Charts |
| Switzerland | CH | Energy-Charts |
| Czech Republic | CZ | Energy-Charts |
| Germany | DE-LU | aWATTar |
| Denmark | DK1, DK2 | EnergiDataService |
| Spain | ES | OMIE |
| Finland | FI | SpotHinta |
| France | FR | Energy-Charts |
| Hungary | HU | Energy-Charts |
| Italy | IT-North, IT-CN, IT-CS, IT-South, IT-Sicily, IT-Sardinia | Energy-Charts |
| Netherlands | NL | EasyEnergy |
| Norway | NO1-NO5 | EnergiDataService |
| Poland | PL | Energy-Charts |
| Portugal | PT | OMIE |
| Sweden | SE1-SE4 | EnergiDataService |
| Slovenia | SI | Energy-Charts |
