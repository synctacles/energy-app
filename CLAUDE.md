# CLAUDE.md - Energy Go Addon

## Project Overview
Synctacles Energy — a fully local Go-based HA addon for EU electricity price monitoring with GO/WAIT/AVOID recommendations.

**Module:** `github.com/synctacles/energy-go`
**Go version:** 1.24.0
**Binary:** `energy-addon` (single static binary, CGO_ENABLED=0)

## Architecture

```
cmd/energy-addon/           ← Entrypoint
internal/
  config/                   ← EnergyAddonConfig (caarlos0/env)
  models/                   ← HourlyPrice, Action, Zone, TaxProfile
  ha/                       ← HA Supervisor API client (REST)
  state/                    ← JSON state persistence (atomic writes)
  collector/                ← PriceSource interface + implementations
    source.go               ← Interface definition
    easyenergy.go           ← NL (keyless)
    frank.go                ← NL (keyless GraphQL)
    energycharts.go         ← EU-wide (keyless, Fraunhofer ISE)
    energidataservice.go    ← DK/Nordic (keyless)
    awattar.go              ← DE/AT (keyless)
    omie.go                 ← ES/PT (keyless CSV)
    spothinta.go            ← FI/Nordic (keyless)
    ecb.go                  ← ECB exchange rates (EUR→NOK/SEK/DKK)
  engine/                   ← Core business logic
    normalizer.go           ← Wholesale → consumer (TaxProfile)
    fallback.go             ← Multi-source fallback chain
    action.go               ← GO/WAIT/AVOID calculation
    bestwindow.go           ← Sliding window optimum
    tomorrow.go             ← Tomorrow preview
    scheduler.go            ← Fetch scheduling (13:00 CET + jitter)
  store/                    ← SQLite cache (modernc.org/sqlite, pure Go)
  hasensor/                 ← HA sensor publishing
    publisher.go            ← Interface
    rest.go                 ← REST API publisher (always available)
    mqtt.go                 ← MQTT auto-discovery (optional)
    detect.go               ← MQTT broker detection
  license/                  ← License validation (care.synctacles.com)
  countries/                ← Country YAML configs (//go:embed)
  web/                      ← Chi router + embedded SPA
deploy/
  docker/Dockerfile         ← Multi-stage build
  docker/run.sh             ← Bashio options reader
  ha-addon/config.yaml      ← HA addon manifest
```

## Key Design Decisions

1. **Keyless-first**: All primary sources are free/keyless APIs. No ENTSO-E dependency.
2. **No Nordpool**: Explicitly excluded — not usable for commercial redistribution.
3. **Pure Go SQLite**: modernc.org/sqlite (no CGo) → static binary + cross-compile.
4. **Dual sensor publishing**: REST API (always) + MQTT auto-discovery (if broker detected).
5. **Country YAML configs**: Data-driven tax profiles embedded via `//go:embed`.
6. **Free model**: All features (prices, GO/WAIT/AVOID, best window, tomorrow) are free for all users.

## Thresholds (from existing product)

- GO: deviation <= -15% below daily average
- AVOID: deviation >= +20% above daily average
- Best 4 hours: always GO
- Tomorrow FAVORABLE: avg < €0.20/kWh OR avg < today * 0.90
- Tomorrow EXPENSIVE: avg > €0.30/kWh OR avg > today * 1.10
- Data freshness for GO: < 6 hours

## Build Commands

```bash
make addon                 # Build for current platform
make addon-all             # Cross-compile for all HA architectures
make test                  # Run tests
make docker-addon          # Docker build
```

## Related Repos
- **energy-backend**: `synctacles/energy-backend` — Energy API server, shared pkg/
- **care-app**: `synctacles/care-app` — Care HA addon (Go)
- **platform**: `synctacles/platform` — Auth service, licensing
