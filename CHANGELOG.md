# Changelog

## 1.0.0

### Crowdsource pipeline hersteld (ADR_018)
- Wizard tax submission stuurt nu naar het correcte `/submit-tax` endpoint (was `/submit-price` — 404 na refactor in maart)

### Initial release
- EU-wide day-ahead electricity prices for 30 bidding zones across 17 countries
- 7 price sources: EasyEnergy, Frank Energie, Energy-Charts, Energi Data Service, aWATTar, OMIE, spot-hinta.fi
- Multi-source fallback chain with circuit breaker and in-memory cache
- GO/WAIT/AVOID action recommendations based on price deviation from daily average
- Best 3-hour consecutive window finder
- Tomorrow price preview (FAVORABLE/NORMAL/EXPENSIVE)
- 12 Home Assistant sensors
- Direct Price Production (ADR_016) — consumer prices berekend uit publieke bronnen zonder Enever afhankelijkheid
- Power sensor auto-detection — automatically finds P1 meter or power sensor from HA entities
- Web dashboard with dark/light theme and real-time price chart
- Settings UI for all addon options (zone, thresholds, power sensor)
- Dual sensor publishing: REST API (always) + MQTT auto-discovery (if broker detected)
- Cross-compilation for amd64, aarch64, armv7
- Comprehensive English documentation (DOCS.md)
