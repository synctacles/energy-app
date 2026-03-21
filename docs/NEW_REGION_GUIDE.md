# Adding a New Region/Country

This guide documents how to add a new country or bidding zone to the Energy App. GB (Great Britain) was the first non-ENTSO-E region added and serves as the reference implementation.

## Overview

Adding a new region requires changes in two repos:

| Repo | What | Why |
|------|------|-----|
| `synctacles-platform` (energy-data worker) | Wholesale price fetching + tax seed | Worker provides prices and tax profiles to the app |
| `synctacles/energy-app` | Country YAML + zone config | App needs zone metadata, suppliers, and tax defaults |

## Step 1: Country YAML (`energy-app`)

Create `pkg/countries/data/{cc}.yaml` where `{cc}` is the ISO-2 country code (lowercase).

### Required fields

```yaml
country: GB                    # ISO-2 country code
name: United Kingdom           # Display name
currency: GBP                  # ISO-4217 currency code (EUR, GBP, SEK, NOK, etc.)

tax_defaults:                  # Fallback when Worker tax profile unavailable
  vat_rate: 0.05               # VAT as decimal (5% = 0.05)
  energy_tax: 0.00079          # Energy tax per kWh in LOCAL CURRENCY
  surcharges: 0                # Additional surcharges per kWh
  network_tariff_avg: 0.082    # Average network tariff per kWh (informational)
  valid_from: "2025-04-01"     # When these rates became effective

suppliers:                     # Known electricity suppliers
  - id: octopus_agile          # Lowercase, underscores, unique per country
    name: Octopus Energy Agile # Display name
    markup: 0.04               # Initial estimate in LOCAL CURRENCY per kWh

zones:                         # Bidding zones (most countries have one)
  - code: GB                   # Must match Worker's zone code exactly
    name: Great Britain
    country: GB
    timezone: Europe/London    # IANA timezone
    lat: 51.51                 # Centroid latitude (for auto-detection)
    lon: -0.13                 # Centroid longitude
```

### Non-ENTSO-E zones

For zones without ENTSO-E EIC codes (e.g., GB post-Brexit), add the `wholesale` flag:

```yaml
zones:
  - code: GB
    wholesale: true            # Explicit flag — has wholesale via alternative source
    # NO eic field needed
```

This ensures `HasWholesale()` returns `true` even without an EIC code.

### Supplier markup strategy

**Initial markups are estimates.** The delta pipeline (ADR_010) automatically refines them:

1. **Day 0:** All suppliers use the estimated markup from YAML
2. **Sensor submissions arrive:** Users with price sensors submit per-hour deltas
3. **Worker aggregates:** Median delta per (zone, supplier, hour) stored in `energy_supplier_deltas`
4. **App applies:** Supplier-specific delta replaces fixed markup
5. **No sensor?** Suppliers without submissions receive the `_average` delta (median across all suppliers)

This means markup values in the YAML only matter until the first sensors come online. Err on the side of a reasonable estimate (2-5 ct/kWh for most EU markets).

### Supplier tiers

Organize suppliers by how they track wholesale:

| Tier | Description | Example |
|------|-------------|---------|
| 1 | True wholesale pass-through (hourly/half-hourly) | Octopus Agile, Tibber |
| 2 | Time-of-use / EV tariffs (fixed bands) | Octopus Go, E.ON Smart Saver |
| 3 | Standard variable (no wholesale tracking) | OVO, British Gas |

All tiers benefit from the delta pipeline. Tier 1 suppliers will have the most accurate deltas (sensor data matches wholesale closely). Tier 3 suppliers rely on the `_average` delta.

## Step 2: Worker Configuration (`synctacles-platform`)

### Tax seed

Ensure `energy_tax_seed` has an entry for the country:

```sql
INSERT OR REPLACE INTO energy_tax_seed
  (country_code, vat_pct, energy_tax_kwh, surcharges_kwh, network_cost_kwh, valid_from)
VALUES ('GB', 0.05, 0.00079, 0, 0.082, '2025-04-01');
```

### Zone registration

Ensure `energy_zones` has the zone:

```sql
INSERT OR REPLACE INTO energy_zones
  (zone, country_code, country_name, currency, seed_needed, primary_source)
VALUES ('GB', 'GB', 'United Kingdom', 'GBP', 1, 'elexon');
```

### Wholesale source

If the zone uses a non-standard wholesale source (not ENTSO-E or Energy-Charts), implement a dedicated fetcher in the energy-data worker. Example: `fetchZoneAgileRates()` for GB (ADR_013).

## Step 3: Validation Checklist

- [ ] `gb.yaml` loads without error: `go test ./pkg/countries/...`
- [ ] `HasWholesale()` returns `true` for the zone
- [ ] Worker returns prices: `curl .../api/v1/energy/prices?zone=GB&from=...&to=...`
- [ ] Worker returns tax profile: check `tax_profile` in price response
- [ ] Worker returns supplier deltas (once sensors online): `curl .../api/v1/energy/supplier-deltas?zone=GB`
- [ ] Energy-app detects zone and shows prices on dashboard
- [ ] Sensor detection works for local integrations (e.g., Octopus HA integration)

## Reference: Currency Handling

The delta formula is **currency-agnostic**:

```
delta = consumer_price / (1 + VAT) - wholesale - energy_tax - surcharges
```

All inputs must be in the **same local currency**. The Worker serves wholesale in local currency (EUR for EU, GBP for GB). Tax profiles are in local currency. Sensor prices are in local currency. No cross-currency conversion is needed within a zone.

The `PriceEUR` field name in the Go code is a historical artifact — the value is always in the zone's local currency regardless of the field name.

## Reference: GB Implementation (ADR_013)

GB was added as the first non-ENTSO-E, non-EUR zone:

| Aspect | EU zones | GB |
|--------|----------|-----|
| Wholesale source | Energy-Charts / ENTSO-E | EPEX Spot + Nord Pool (via agilerates.uk) / Elexon BMRS |
| Resolution | PT15M or PT60M | PT30M |
| Currency | EUR | GBP |
| EIC code | Yes | None (uses `wholesale: true` flag) |
| Day-ahead timing | ~13:00 CET | EPEX hourly ~09:30, N2EX ~10:00, EPEX 30-min ~16:00 UTC |
