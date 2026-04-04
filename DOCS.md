# Synctacles Energy — Home Assistant App

Synctacles Energy provides real-time European electricity prices with intelligent GO/WAIT/AVOID recommendations. All price data is fetched from free, public APIs via the Synctacles Worker — no cloud dependency, no account needed.

## How It Works

The app fetches day-ahead electricity prices from multiple European data sources, normalizes them to consumer prices (including VAT, energy tax, and supplier markup where applicable), and publishes the results as Home Assistant sensors via MQTT.

Every 15 minutes, the app:
1. Fetches prices from the highest-priority source for your bidding zone
2. Falls back to alternative sources if the primary is unavailable
3. Calculates the current price, daily statistics, and action recommendation
4. Publishes all sensor values to Home Assistant

### Price Source Hierarchy

Each bidding zone has a prioritized chain of price sources. If the highest-priority source fails, the app automatically tries the next one.

| Tier | Source | Coverage |
|------|--------|----------|
| 1 | Energy-Charts | 26 EU zones (primary) |
| 2 | ENTSO-E | Fallback for EC zones + 11 additional zones |
| 3 | Elering | EE, LV, LT, FI (tertiary) |
| 4 | Elexon BMRS | GB only |
| 5 | Zone correlation | Last resort — estimates from correlated nearby zones |

A 48-hour local SQLite cache provides resilience when all live sources are unavailable.

### GO/WAIT/AVOID Recommendations

The app compares the current hour's price to the daily average:

| Action | Condition | Meaning |
|--------|-----------|---------|
| **GO** | Price is >15% below average | Great time to use electricity |
| **WAIT** | Price is within normal range | No urgency either way |
| **AVOID** | Price is >20% above average | Postpone if possible |

Thresholds are user-configurable. The app also identifies the cheapest/most expensive hours and finds the best consecutive window (configurable from 1 to 8 hours, default 3).

## Supported Countries and Zones

The app supports 46 bidding zones across 28 European countries via the centralized Synctacles Energy Worker (energy-data.synctacles.com).

| Region | Countries | Zones |
|--------|-----------|-------|
| Western Europe | NL, BE, FR, DE, AT, CH | NL, BE, FR, DE-LU, AT, CH |
| Iberian Peninsula | ES, PT | ES, PT |
| Nordic | NO, SE, DK, FI | NO1–NO5, SE1–SE4, DK1, DK2, FI |
| Baltic | EE, LV, LT | EE, LV, LT |
| Central Europe | PL, CZ, HU, SK, SI, HR, RO, BG | PL, CZ, HU, SK, SI, HR, RO, BG |
| Southern Europe | IT, GR, CY | IT (6 zones), GR, CY |
| British Isles | GB, IE | GB, IE |

All sources are free, public APIs. No API keys required.

## Sensors

The app publishes **15 sensors** to Home Assistant — all included for free:

| Sensor | Entity ID | Description |
|--------|-----------|-------------|
| Current Price | `sensor.synctacles_energy_price` | Current hour electricity price (EUR/kWh) |
| Cheapest Hour | `sensor.synctacles_cheapest_hour` | Today's cheapest hour |
| Expensive Hour | `sensor.synctacles_expensive_hour` | Today's most expensive hour |
| Prices Today | `sensor.synctacles_prices_today` | Hourly prices array in attributes |
| Action | `sensor.synctacles_energy_action` | GO / WAIT / AVOID recommendation |
| Best Window | `sensor.synctacles_best_window` | Cheapest consecutive N-hour block |
| Tomorrow Preview | `sensor.synctacles_tomorrow_preview` | FAVORABLE / NORMAL / EXPENSIVE / PENDING |
| Prices Tomorrow | `sensor.synctacles_prices_tomorrow` | Tomorrow's hourly prices in attributes |
| Renewable Share | `sensor.synctacles_renewable_share` | Current renewable energy percentage |
| Green Energy | `binary_sensor.synctacles_green_energy` | ON when current hour is green (high renewables) |
| Cheap Hour | `binary_sensor.synctacles_cheap_hour` | ON when current hour is cheap |
| Live Cost | `sensor.synctacles_live_cost` | Real-time cost in EUR/h (requires power sensor) |
| Savings | `sensor.synctacles_savings` | Daily savings vs average price (requires power sensor) |
| Usage Score | `sensor.synctacles_usage_score` | Usage timing efficiency 0–100 (requires power sensor) |
| Daily Cost | `sensor.synctacles_daily_cost` | Cumulative daily cost in EUR (requires power sensor) |

Live Cost, Savings, Usage Score, and Daily Cost require a power sensor entity to be configured.

## Pricing Modes

The app supports **5 pricing modes** that determine how your electricity price is calculated:

| Mode | Description | Availability |
|------|-------------|--------------|
| **Auto** | Wholesale price + automatic tax profile from Synctacles Worker | All zones |
| **Manual** | Wholesale price + your own tax values (VAT, energy tax, surcharges) | All zones |
| **External Sensor** | Consumer price from any HA sensor (Tibber, Zonneplan, Octopus, etc.) | All zones |
| **P1 Meter** | Tariff from a P1 smart meter sensor | All zones |
| **Meter Tariff** | Meter-based tariff from HA energy integration | All zones |

**Auto** is the default and recommended mode. It uses calibrated tax profiles per country that are updated via the Synctacles Worker. If the Worker is unavailable, the fallback chain provides wholesale prices.

**External Sensor** mode reads the consumer price from any Home Assistant sensor that exposes an electricity tariff in EUR/kWh (or other currency/kWh). The fallback chain still runs in the background for GO/WAIT/AVOID recommendations and tomorrow's prices.

### Supplier Delta Calibration (Netherlands)

For Dutch users, the app supports supplier-specific price calibration. Select your energy supplier in settings to get delta-corrected prices that match your actual consumer price. Supported for 22 Dutch suppliers including Zonneplan, Tibber, Frank Energie, Vattenfall, Essent, and more.

## Automation Blueprints

Five ready-made blueprints for one-click import into Home Assistant:

| Blueprint | Description |
|-----------|-------------|
| **GO Signal** | Turn device on during GO (cheap) hours |
| **AVOID Signal** | Turn device off during AVOID (expensive) hours |
| **Price Alert** | Send notification at configurable price threshold |
| **Best Window** | Schedule automation during the cheapest consecutive hours |
| **Green Energy** | Turn device on during high renewable energy hours |

## Configuration

### Basic Settings

| Option | Default | Description |
|--------|---------|-------------|
| `pricing_mode` | auto | How prices are calculated (see Pricing Modes) |
| `zone` | (auto-detected) | Your electricity bidding zone |
| `go_threshold` | -15 | % below average to recommend GO |
| `avoid_threshold` | 20 | % above average to recommend AVOID |
| `best_window_hours` | 3 | Consecutive hours (1–8) for cheapest-window finder |
| `supplier_id` | — | Your energy supplier (NL only, for delta calibration) |
| `supplier_markup` | 0 | Fixed supplier markup in EUR/kWh (0 = use Worker default) |
| `p1_sensor_entity` | — | HA entity for tariff in EUR/kWh (P1 meter, Tibber, etc.) |
| `power_sensor` | — | HA entity for power consumption (W/kW) — enables cost tracking |
| `debug_mode` | false | Enable verbose logging |

## Troubleshooting

### App shows "Fetching energy prices..."

The app is starting up and fetching prices for the first time. This usually takes 5–15 seconds. If it persists, check the app logs for connection errors.

### Source shows as unhealthy (red dot)

A source's circuit breaker has tripped after a failure. It will automatically recover after 2 hours. The app uses the next available source in the meantime.

### Prices show as "estimated"

Your zone's primary source is temporarily unavailable. The app is using data from a correlated nearby zone as an estimate. Prices will return to "exact" when the primary source recovers.

### Live cost sensors show "unavailable"

Configure a `power_sensor` entity in settings. This must be a Home Assistant entity that reports current power consumption in Watts or kilowatts.
