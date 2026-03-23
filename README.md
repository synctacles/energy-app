# Synctacles Energy — Real-Time Electricity Prices for Home Assistant

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8.svg)](https://go.dev)
[![HA App](https://img.shields.io/badge/Home%20Assistant-App-41BDF5.svg)](https://www.home-assistant.io/)

**Free Home Assistant app for real-time electricity prices, smart scheduling and cost optimization across 30 EU bidding zones.**

Track day-ahead energy prices, get GO/WAIT/AVOID action recommendations, find the cheapest hours to run appliances, and monitor live energy costs — all running locally inside Home Assistant.

**All features are 100% free. No subscription, no trial, no payment.**

**Website:** [synctacles.com](https://www.synctacles.com/)

---

## Features

### Price Tracking
- **30 EU bidding zones** across 17 countries (NL, DE, AT, BE, FR, ES, PT, IT, NO, SE, DK, FI, CH, PL, CZ, HU, SI)
- **7 independent price sources** with automatic fallback (EasyEnergy, aWATTar, Energy-Charts, Energi Data Service, SpotHinta, OMIE, Frank Energie)
- **Day-ahead prices** updated daily after 13:00 CET (EPEX/Nordpool publication window)
- **Supplier-specific pricing** for 22 Dutch energy providers via supplier delta calibration

### Smart Recommendations
- **GO/WAIT/AVOID** — real-time action based on current price vs daily average
- **Best window finder** — cheapest consecutive hours (configurable 1-8h) for scheduling appliances
- **Tomorrow outlook** — FAVORABLE/NORMAL/EXPENSIVE preview when day-ahead prices are available

### Sensors & Automation
- **12 Home Assistant sensors** for automations, dashboards and scripts
- **MQTT auto-discovery** — sensors appear automatically when MQTT broker is available
- **Power sensor integration** — connect a P1 meter or power entity for live EUR/hour cost tracking

### Reliability
- **Multi-source fallback** with circuit breaker — if one source fails, the next takes over automatically
- **Local SQLite cache** — 48 hours of prices stored locally for offline resilience
- **Central price server** as primary source (pre-computed, tax-normalized consumer prices)

### Web Dashboard
- Built-in responsive UI with dark/light theme
- Real-time price chart with today and tomorrow prices
- Accessible via Home Assistant sidebar (ingress)

---

## Supported Countries & Zones

| Country | Zones | Primary Source |
|---------|-------|---------------|
| Netherlands | NL | EasyEnergy |
| Germany/Luxembourg | DE-LU | aWATTar |
| Austria | AT | aWATTar |
| Belgium | BE | Energy-Charts |
| France | FR | Energy-Charts |
| Norway | NO1–NO5 | Energi Data Service |
| Sweden | SE1–SE4 | Energi Data Service |
| Denmark | DK1, DK2 | Energi Data Service |
| Finland | FI | SpotHinta |
| Spain | ES | OMIE |
| Portugal | PT | OMIE |
| Italy | 6 regions | Energy-Charts |
| Switzerland | CH | Energy-Charts |
| Poland | PL | Energy-Charts |
| Czech Republic | CZ | Energy-Charts |
| Hungary | HU | Energy-Charts |
| Slovenia | SI | Energy-Charts |

All sources are keyless public APIs — no ENTSO-E token or Nordpool account required.

---

## Installation

### Home Assistant App Store

1. Go to **Settings > Apps > App Store** in Home Assistant
2. Click the three dots (top right) > **Repositories**
3. Add: `https://github.com/synctacles/ha-apps`
4. Refresh the App Store and install **Synctacles Energy**

### Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `zone` | `NL` | Your electricity bidding zone |
| `go_threshold` | `-15` | % below average to trigger GO |
| `avoid_threshold` | `20` | % above average to trigger AVOID |
| `best_window_hours` | `3` | Hours for cheapest window calculation |
| `power_sensor` | — | HA entity ID for live cost tracking |

See [DOCS.md](DOCS.md) for the full configuration guide and sensor reference.

---

## Sensors

| Sensor | Tier | Description |
|--------|------|-------------|
| `sensor.synctacles_energy_price` | Free | Current electricity price (EUR/kWh) |
| `sensor.synctacles_energy_average` | Free | Today's average price |
| `sensor.synctacles_energy_min` | Free | Today's lowest price |
| `sensor.synctacles_energy_max` | Free | Today's highest price |
| `sensor.synctacles_energy_source` | Free | Active price data source |
| `sensor.synctacles_energy_action` | Free | GO/WAIT/AVOID recommendation |
| `sensor.synctacles_energy_best_start` | Free | Cheapest window start hour |
| `sensor.synctacles_energy_best_end` | Free | Cheapest window end hour |
| `sensor.synctacles_energy_best_avg` | Free | Cheapest window average price |
| `sensor.synctacles_energy_tomorrow` | Free | Tomorrow's price outlook |
| `sensor.synctacles_energy_live_cost` | Free | Real-time cost (EUR/h, needs power sensor) |
| `sensor.synctacles_energy_daily_cost` | Free | Cumulative daily cost |

---

## Architecture

```
┌─────────────────────────────────────────┐
│         Home Assistant (Docker)         │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │        Synctacles Energy          │  │
│  │                                   │  │
│  │  Central API ← Primary source     │  │
│  │       ↓ (fallback)               │  │
│  │  Direct collectors (7 sources)    │  │
│  │       ↓ (cache)                  │  │
│  │  Local SQLite (48h retention)     │  │
│  │       ↓                          │  │
│  │  Price engine → GO/WAIT/AVOID     │  │
│  │       ↓                          │  │
│  │  HA sensors (REST + MQTT)         │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

Single compiled Go binary. No Python runtime, no external dependencies.

---

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint

# Cross-compile for all HA architectures
make addon-all
```

Requires Go 1.24+.

---

## Related

- [synctacles/ha-apps](https://github.com/synctacles/ha-apps) — Home Assistant app store repository
- [synctacles/care-app](https://github.com/synctacles/care-app) — Care diagnostics app for Home Assistant
- [synctacles.com](https://www.synctacles.com/) — Project website

---

## License

MIT — See [LICENSE](LICENSE)

---

**Gemaakt met ❤️ op Madeira**
