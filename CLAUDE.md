# CLAUDE.md - Energy App

## ⚠️ Principle #1: Always Think Like the End User

**Before shipping any feature or fix, stand in the shoes of every end user — both technical and non-technical — and verify the result is actually usable.**

Ask yourself:
- Can a non-technical HA user understand what this does?
- Is the UI clear without documentation?
- Does clicking a button produce visible, understandable feedback?
- Can the user recover from errors without reading logs?
- If a feature requires user action (API key, settings, etc.), is the path to that action obvious?

**If the answer to any of these is "maybe" or "no": stop and fix it first.**

---

## Project Overview
Synctacles Energy — a fully local Go-based HA app for EU electricity price monitoring with GO/WAIT/AVOID recommendations.

**Module:** `github.com/synctacles/energy-app`
**Go version:** 1.24.0
**Binary:** `energy-addon` (single static binary, CGO_ENABLED=0)

## Architecture

```
cmd/energy-addon/
  main.go                   ← Entrypoint, scheduler, delta init, sensor publishing

internal/
  config/                   ← EnergyAddonConfig (caarlos0/env)
  delta/                    ← Supplier delta pipeline (ADR_010)
    cache.go                ← Delta cache: fetch, persist to disk, RunFetcher loop
    submitter.go            ← Sensor → delta calculation → POST to Worker
    forecast.go             ← Parse sensor forecast attributes (Zonneplan, NordPool, etc.)
    wholesale.go            ← Fetch wholesale prices from Worker for delta calc
  gate/                     ← Feature gating
  ha/                       ← HA Supervisor API client (REST)
  hasensor/                 ← HA sensor publishing + sensor auto-detection
    publisher.go            ← SensorSet + PublishAll (13 sensors)
    rest.go                 ← REST API publisher (always available)
    mqtt.go                 ← MQTT auto-discovery (optional)
    detect.go               ← DetectAllTariffSensors (multi-supplier)
  heartbeat/                ← Heartbeat sender to platform Worker
  state/                    ← JSON state persistence (atomic writes)
  telemetry/                ← Anonymous telemetry
  web/                      ← Chi router + embedded SPA
    server.go               ← API endpoints (/api/dashboard, /api/sources, etc.)
    static/index.html       ← SPA (i18n, chart, settings, 8 languages)

pkg/
  collector/                ← PriceSource interface + implementations
    source.go               ← Interface definition
    synctacles.go           ← Synctacles Worker API (primary, all zones)
    energycharts.go         ← Energy-Charts direct (keyless, EU-wide fallback)
    ecb.go                  ← ECB exchange rates (EUR→NOK/SEK/DKK)
    httpclient.go           ← Shared HTTP client with retry
  countries/                ← Country YAML configs (//go:embed) — tax profiles
  engine/                   ← Core business logic
    normalizer.go           ← Wholesale → consumer (TaxProfile + delta application)
    fallback.go             ← Multi-source fallback chain + FetchResult
    action.go               ← GO/WAIT/AVOID calculation (wholesale mode)
    action_regulated.go     ← Fixed-rate and TOU action modes
    bestwindow.go           ← Sliding window optimum
    bestwindow_regulated.go ← TOU offpeak window
    scheduler.go            ← Fetch scheduling (13:00 CET + jitter)
    taxcache.go             ← Tax profile cache (disk-persisted)
    tomorrow.go             ← Tomorrow preview
    tou.go                  ← Time-of-Use tariff engine
  kb/                       ← Knowledge base client
  lease/                    ← Distributed lease management
  models/                   ← HourlyPrice, Action, Zone, TaxProfile, etc.
  platform/                 ← Platform API client (HMAC signing)
  store/                    ← SQLite cache (modernc.org/sqlite, pure Go)
```

## Key Design Decisions

1. **Keyless-first**: No ENTSO-E key required. App fetches from Synctacles Worker (which handles EC/ENTSO-E).
2. **No Nordpool**: Explicitly excluded — not usable for commercial redistribution.
3. **Pure Go SQLite**: modernc.org/sqlite (no CGo) → static binary + cross-compile.
4. **Dual sensor publishing**: REST API (always) + MQTT auto-discovery (if broker detected).
5. **Country YAML configs**: Data-driven tax profiles embedded via `//go:embed`.
6. **Free model**: All features are free for all users. No account, no registration.

## Source Chain

The app fetches prices from the Synctacles Worker (`energy-data.synctacles.com`), which serves Energy-Charts (primary) and ENTSO-E (fallback) data. The app also has a direct Energy-Charts collector as local fallback.

```
Tier 1: Synctacles Worker (energy-data.synctacles.com/prices)
  └─ Returns EC or ENTSO-E wholesale + consumer prices
  └─ Source info propagated to UI via FetchResult.UpstreamSource

Tier 2: Energy-Charts direct (keyless, local fallback)
  └─ Used when Worker is unreachable

Tier 3: SQLite disk cache (offline capable)
  └─ Persisted prices from previous successful fetch
```

Implementation: `pkg/engine/fallback.go` (FetchResult, source chain), `pkg/collector/synctacles.go` (Worker client).

## Pricing Modes

| Mode | Description | Delta applied? |
|------|-------------|---------------|
| `auto` | Dynamic wholesale + tax profile + delta | Yes |
| `external_sensor` | Sensor live price for display, wholesale+delta for chart | Yes (chart only) |
| `p1_meter` | P1 meter tariff sensor | Yes (chart only) |
| `meter_tariff` | Generic meter tariff | No |
| `manual` | Manual markup over wholesale | No |
| `fixed` | Fixed rate (no dynamic prices) | No |
| `tou` | Time-of-Use regulated tariff | No |

All dynamic modes (auto, external_sensor, p1_meter) route through `normalizeAuto()` in the normalizer — this is critical for delta application. No early returns that skip delta logic.

## Supplier Delta Pipeline (ADR_010)

Per-hour correction factors that calibrate wholesale prices to actual supplier consumer prices.

### Delta Submission (sensor installs)

```
internal/delta/submitter.go:
  1. Starts 3 min after boot (stabilization wait)
  2. DetectAllTariffSensors() → finds all HA sensors with known supplier patterns
  3. At each hour boundary + 15s:
     a. Read forecast from sensor attributes (internal/delta/forecast.go)
     b. Fetch wholesale from Worker (internal/delta/wholesale.go)
     c. Calculate: delta = consumer_excl_VAT - wholesale - energy_tax - surcharges
     d. POST /supplier-deltas to Worker (HMAC signed)
  4. Live correction: if current sensor price deviates >0.25ct from predicted, submit immediate correction
```

### Delta Consumption (all installs)

```
internal/delta/cache.go:
  1. Initialized BEFORE scheduler.Run() (critical ordering in main.go)
  2. Supplier resolution: cfg.SupplierID > SupplierHintFromEntity() > "_average" fallback
  3. Fetches GET /supplier-deltas every 15 min + at hour boundary + 30s
  4. Persists to disk (delta_cache.json) for offline resilience (max 72h age)
  5. Provides deltaLookup function to normalizer via SetDeltaLookup()
```

### Delta Application (normalizer)

```
pkg/engine/normalizer.go:
  normalizeAuto() called for ALL dynamic pricing modes:
    → deltaLookup(timestamp) → supplier-specific delta from cache
    → p.PriceEUR += delta × (1 + VAT)

  Sensor mode specifics:
    → Supplier-specific delta: always applied
    → _average delta: skipped (would reduce accuracy for known supplier)
    → Sensor live reading overrides CurrentPrice display only (not chart bars)
```

### Timing

```
HH:00:00  — Hour boundary
HH:00:15  — Submitter reads forecast + calculates delta → POST to Worker
HH:00:30  — Delta cache fetches updated deltas from Worker
HH:00:45  — Scheduler fetches prices → normalizer applies deltas → chart updated
```

## Thresholds

- GO: deviation <= -15% below daily average
- AVOID: deviation >= +20% above daily average
- Best 4 hours: always GO
- Tomorrow FAVORABLE: avg < €0.20/kWh OR avg < today × 0.90
- Tomorrow EXPENSIVE: avg > €0.30/kWh OR avg > today × 1.10
- Data freshness for GO: < 6 hours

## Build Commands

```bash
make addon                 # Build for current platform
make addon-all             # Cross-compile for all HA architectures
make test                  # Run tests
make docker-addon          # Docker build
go vet ./...               # Lint check (must pass before commit)
```

## Related Repos
- **platform**: `synctacles/platform` — Cloudflare Workers + D1, shared infrastructure, ADR docs
- **care-app**: `synctacles/care-app` — Care HA app (Go)
- **ha-apps**: `synctacles/ha-apps` — HA app store distribution (config.yaml, Docker images)
