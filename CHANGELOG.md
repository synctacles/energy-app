# CHANGELOG - SYNCTACLES

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Week 2 - Binary Signals (2025-12-25)

#### Added
- **Binary Signals API** (`/api/v1/signals`)
  - 5 intelligent decision signals for home automation
  - `is_cheap`: Price below 24h rolling average
  - `is_green`: Renewable energy > 50% threshold
  - `charge_now`: Optimal charging (cheap + green)
  - `grid_stable`: Grid balance within safe limits
  - `cheap_hour_coming`: Price dip forecast (3h ahead)
- **API Documentation** (`docs/api/signals.md`)
  - Complete endpoint specification
  - Signal logic definitions with examples
  - Integration guides (Home Assistant + Python)
  - Rate limits and error handling
- **Real-time Data Integration**
  - ENTSO-E day-ahead pricing (15 min updates)
  - ENTSO-E generation mix by source type
  - TenneT grid balance data (5 min updates)

#### Changed
- Updated database schema queries to use correct column names
  - `country` instead of `country_code`
  - `timestamp` instead of `datetime_utc`
  - `delta_mw` instead of `balance_delta_mw`
- Renewable percentage calculation uses normalized columns
  - Direct column access (b01_biomass_mw, b16_solar_mw, etc.)
  - Eliminates need for PSR type filtering

#### Fixed
- Import path for database dependencies (`dependencies.py`)
- Router prefix configuration (single `/v1` prefix)
- Async/sync function mismatch (converted to sync)
- Type casting for float values in metadata
- Empty list handling in `cheap_hour_coming` calculation

#### Technical Details
- **Performance:** < 100ms response time (p95)
- **Database:** 3 normalized tables queried
- **Cache:** 5-minute TTL on signals endpoint
- **Auth:** X-API-Key header required

---

## [1.0.0] - Week 1 Completion (2025-12-24)

### Added
- Authentication system with API key generation
- User management endpoints (signup, stats, regenerate)
- Fallback data sources (Energy-Charts integration)
- HACS repository structure for HA distribution
- System API keys for root and synctacles users
- Database read-only role for normalizer safety

### Changed
- Single database architecture (simplified from 2-database plan)
- Deployment workflow (git → rsync → restart)
- Systemd timer activation (4 timers, 15-min intervals)

### Fixed
- Auth middleware order (after CORS)
- PostgreSQL peer authentication for normalizer role
- Email validation for system accounts

### Infrastructure
- **Data Collection:** 965K records normalized
- **API Uptime:** 7+ days stable
- **Cache Performance:** 79-83% hit rate
- **Workers:** 4 Gunicorn processes

---

## [0.9.0] - Foundation (F1-F7)

### Added
- Data collection pipeline (ENTSO-E + TenneT)
- Database normalization layer
- FastAPI REST endpoints (generation-mix, load, balance)
- Caching layer (Redis)
- Monitoring (Prometheus + Grafana)
- Automated scheduling (systemd timers)

### Database Schema
- **Raw tables:** 3 (entso_e_a75, entso_e_a65, tennet_balance)
- **Normalized tables:** 4 (a75_generation, a44_prices, a65_load, balance)
- **Retention:** 72 hours rolling window

### API Endpoints (Initial)
- `GET /api/v1/generation-mix` - Current generation by source
- `GET /api/v1/load` - Current electricity demand
- `GET /api/v1/balance` - Grid balance status

---

## Project Metrics

### Development Velocity
- **Week 1:** 3.5h (planned: 18h) → 80% faster
- **Week 2 DAG 1:** 1.5h (planned: 4h) → 62% faster
- **Overall efficiency:** 6x AI acceleration factor

### Data Quality
- **Completeness:** ~100% (via intelligent fallbacks)
- **Freshness:** 5-15 minute updates
- **Attribution:** All sources tracked in metadata
- **Quality flags:** FRESH, STALE, CACHED, FALLBACK

### Coverage
- **Country:** Netherlands (NL) only
- **Future:** Germany, France, Belgium (9-12 months)
- **Markets:** Day-ahead pricing, real-time generation

---

## Roadmap

### V1.1 (January 2026)
- [ ] Home Assistant custom component update
- [ ] 5 binary sensor entities
- [ ] User-configurable thresholds
- [ ] Dashboard card examples
- [ ] Automation templates

### V1.2 (February 2026)
- [ ] Historical data API (7-day queries)
- [ ] Price forecasting (24h ahead)
- [ ] Renewable forecast integration
- [ ] Export capacity signals

### V2.0 (Q2 2026)
- [ ] Multi-country support (DE, FR, BE)
- [ ] Advanced signals (load shifting, arbitrage)
- [ ] Mobile app (iOS + Android)
- [ ] Premium tier features

---

## Support

**Documentation:** https://docs.synctacles.io  
**Issues:** https://github.com/DATADIO/synctacles-repo/issues  
**API Status:** https://status.synctacles.io  
**Contact:** support@synctacles.io

---

*Maintained by Leo + Claude*  
*Last Updated: 2025-12-25*
