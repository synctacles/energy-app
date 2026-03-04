# Changelog

All notable changes to Synctacles Energy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-02-12

### Added
- Power sensor auto-detection — automatically finds P1 meter or power sensor from HA entities

### Removed
- License/trial system — all features are now unconditionally free

## [1.1.0] - 2026-02-12

### Added
- `binary_sensor.synctacles_cheap_hour` — ON during GO periods
- `sensor.synctacles_daily_cost` — cumulative daily cost sensor (resets at midnight, requires power sensor)
- Configurable best window duration (`best_window_hours`, 1-8 hours, default 3)

### Fixed
- SQLite cache now writes to `/data` instead of read-only `/config` mount

### Changed
- Sensor count: 11 → 12
- Best window sensor description updated for configurable duration

## [1.0.0] - 2026-02-12

### Added
- EU-wide day-ahead electricity prices for 30 bidding zones across 17 countries
- 7 price sources: EasyEnergy, Frank Energie, Energy-Charts, Energi Data Service, aWATTar, OMIE, spot-hinta.fi
- Multi-source fallback chain with circuit breaker and in-memory cache
- GO/WAIT/AVOID action recommendations based on price deviation from daily average
- Best 3-hour consecutive window finder
- Tomorrow price preview (FAVORABLE/NORMAL/EXPENSIVE)
- 12 Home Assistant sensors
- Enever integration for NL all-in consumer prices (24 Dutch suppliers)
- Rate limit protection for Enever API (2h in-memory cache, ~3 calls/day)
- YAML-driven country tax profiles (VAT, energy tax, ODE) with embedded configs
- SQLite price cache with 48h retention (pure Go, no CGO)
- Web dashboard with dark/light theme and real-time price chart
- Source health indicators (green/red) for all configured price APIs
- Settings UI for all addon options (zone, thresholds, Enever, power sensor)
- Dual sensor publishing: REST API (always) + MQTT auto-discovery (if broker detected)
- Live Cost, Savings, and Usage Score sensors (requires power sensor entity)
- Cross-compilation for amd64, aarch64, armv7
- Comprehensive English documentation (DOCS.md)
