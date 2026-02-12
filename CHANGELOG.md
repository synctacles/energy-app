# Changelog

All notable changes to Synctacles Energy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-12

### Added
- EU-wide day-ahead electricity prices for 30 bidding zones across 17 countries
- 7 price sources: EasyEnergy, Frank Energie, Energy-Charts, Energi Data Service, aWATTar, OMIE, spot-hinta.fi
- Multi-source fallback chain with circuit breaker and in-memory cache
- GO/WAIT/AVOID action recommendations based on price deviation from daily average
- Best 3-hour consecutive window finder
- Tomorrow price preview (FAVORABLE/NORMAL/EXPENSIVE)
- 11 Home Assistant sensors (4 free, 7 pro)
- Enever integration for NL all-in consumer prices (24 Dutch suppliers)
- Rate limit protection for Enever API (2h in-memory cache, ~3 calls/day)
- YAML-driven country tax profiles (VAT, energy tax, ODE) with embedded configs
- SQLite price cache with 48h retention (pure Go, no CGO)
- Web dashboard with dark/light theme and real-time price chart
- Source health indicators (green/red) for all configured price APIs
- Settings UI for all addon options (zone, thresholds, Enever, license, power sensor)
- Dual sensor publishing: REST API (always) + MQTT auto-discovery (if broker detected)
- Live Cost, Savings, and Usage Score sensors (requires power sensor entity)
- License validation against api.synctacles.com with 90-day offline grace period
- Freemium model: free tier (price + stats) and pro tier (actions + advanced sensors)
- Cross-compilation for amd64, aarch64, armv7
- Comprehensive English documentation (DOCS.md)
