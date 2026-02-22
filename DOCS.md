# Synctacles Energy — Home Assistant Addon

Synctacles Energy is a fully local Home Assistant addon that provides EU day-ahead electricity prices with intelligent GO/WAIT/AVOID recommendations. All price data is fetched directly from free, public European energy APIs — no cloud dependency.

## How Energy Prices Work

Energy prices in Europe are determined **one day in advance** through the EPEX Spot day-ahead auction. This is what that means for you:

### Prices are static — not real-time

- **Today's prices** were determined yesterday at 13:00 CET. They will not change.
- **Tomorrow's prices** are published today at 13:00 CET. After that, they're fixed.
- There is nothing "real-time" about day-ahead energy prices. All 24 hourly prices for a given day are known in advance and are immutable once published.

### What the addon does

The addon fetches day-ahead electricity prices from multiple European data sources, normalizes them to consumer prices (including VAT, energy tax, and supplier markup where applicable), and publishes the results as Home Assistant sensors.

1. Fetches today's 24 hourly prices from your configured source
2. At 13:00 CET, fetches tomorrow's 24 hourly prices
3. Stores all prices locally in a **persistent cache that survives reboots**
4. Shows GO/WAIT/AVOID recommendations based on the current hour's price

### Why you see "stored" in the source bar

After a reboot, the addon uses locally stored prices instead of making new API calls. This is safe because the prices never change after publication. The "stored" label indicates prices are served from the local cache — they are identical to what the API would return.

### Source health vs. data quality

The source bar separates two independent concepts:

- **Green/red dot**: Whether the API source is currently reachable (circuit breaker status)
- **[serving]/[stored]**: Where your current price data comes from

A source can show a red dot (API temporarily down) while still serving valid stored data. The prices are correct regardless — only the source's reachability changed.

| Badge | Meaning |
|-------|---------|
| `live` | Prices just fetched from a live API source |
| `live (memory)` | Same live prices, served from in-memory cache |
| `live (stored)` | Same live prices, restored from persistent disk cache (e.g. after reboot) |
| `cached` | Fallback prices from disk cache (no live source available) — GO recommendations disabled |

### Price Source Fallback Chain

Each bidding zone has a prioritized list of price sources. If the highest-priority source fails, the addon automatically tries the next one. A circuit breaker prevents hammering failed sources (2-hour cooldown).

| Tier | Description | GO Allowed |
|------|-------------|------------|
| 1-3 | Live API sources (zone-specific priority) | Yes |
| Disk warm | Persistent cache with live-quality data (after reboot) | Yes |
| 4 | SQLite cache fallback (48h retention) | No |

### GO/WAIT/AVOID Recommendations

The addon compares the current hour's price to the daily average:

| Action | Condition | Meaning |
|--------|-----------|---------|
| **GO** | Price is >15% below average | Great time to use electricity |
| **WAIT** | Price is within normal range | No urgency either way |
| **AVOID** | Price is >20% above average | Postpone if possible |

Thresholds are user-configurable. The addon also identifies the cheapest/most expensive hours and finds the best consecutive window (configurable from 1 to 8 hours, default 3).

## Supported Countries and Zones

The addon supports 30 bidding zones across 17 European countries:

| Country | Zones | Primary Source | Fallback Sources |
|---------|-------|---------------|------------------|
| Netherlands | NL | EasyEnergy | Frank Energie, Energy-Charts |
| Germany/Luxembourg | DE-LU | aWATTar | Energy-Charts |
| Austria | AT | aWATTar | Energy-Charts |
| Belgium | BE | Energy-Charts | — |
| France | FR | Energy-Charts | — |
| Norway | NO1, NO2, NO3, NO4, NO5 | Energi Data Service | Energy-Charts, spot-hinta.fi |
| Sweden | SE1, SE2, SE3, SE4 | Energi Data Service | Energy-Charts, spot-hinta.fi |
| Denmark | DK1, DK2 | Energi Data Service | Energy-Charts |
| Finland | FI | spot-hinta.fi | Energi Data Service, Energy-Charts |
| Spain | ES | OMIE | Energy-Charts |
| Portugal | PT | OMIE | Energy-Charts |
| Italy | IT-North, IT-Centre-North, IT-Centre-South, IT-South, IT-Sicily, IT-Sardinia | Energy-Charts | — |
| Switzerland | CH | Energy-Charts | — |
| Poland | PL | Energy-Charts | — |
| Czech Republic | CZ | Energy-Charts | — |
| Hungary | HU | Energy-Charts | — |
| Slovenia | SI | Energy-Charts | — |

All sources are free, public APIs. No API keys required (except for the optional Enever integration).

## Sensors

The addon publishes up to 12 sensors to Home Assistant:

### Free Tier

| Sensor | Entity ID | State | Description |
|--------|-----------|-------|-------------|
| Current Price | `sensor.synctacles_energy_price` | EUR/kWh | Current hour electricity price |
| Cheapest Hour | `sensor.synctacles_cheapest_hour` | HH:00 | Today's cheapest hour |
| Expensive Hour | `sensor.synctacles_expensive_hour` | HH:00 | Today's most expensive hour |
| Prices Today | `sensor.synctacles_prices_today` | count | Hourly prices array in attributes |
| Cheap Hour | `binary_sensor.synctacles_cheap_hour` | on/off | ON during GO periods, OFF during WAIT/AVOID |

### Pro Tier

| Sensor | Entity ID | State | Description |
|--------|-----------|-------|-------------|
| Action | `sensor.synctacles_energy_action` | GO/WAIT/AVOID | Current recommendation |
| Best Window | `sensor.synctacles_best_window` | HH:00 - HH:00 | Best consecutive window (configurable 1-8h) |
| Tomorrow Preview | `sensor.synctacles_tomorrow_preview` | FAVORABLE/NORMAL/EXPENSIVE/PENDING | Tomorrow's price outlook |
| Prices Tomorrow | `sensor.synctacles_prices_tomorrow` | count | Tomorrow's hourly prices in attributes |
| Live Cost | `sensor.synctacles_live_cost` | EUR/h | Real-time cost based on power sensor |
| Savings | `sensor.synctacles_savings` | EUR | Daily savings vs average price |
| Usage Score | `sensor.synctacles_usage_score` | 0-100 | How well you use cheap hours |
| Daily Cost | `sensor.synctacles_daily_cost` | EUR | Cumulative daily cost, resets at midnight |

Live Cost, Savings, Usage Score, and Daily Cost require a power sensor entity to be configured.

## Configuration

### Basic Settings

| Option | Default | Description |
|--------|---------|-------------|
| `zone` | NL | Your electricity bidding zone (see table above) |
| `go_threshold` | -15 | % below average to recommend GO |
| `avoid_threshold` | 20 | % above average to recommend AVOID |
| `best_window_hours` | 3 | Duration of the best consecutive window in hours (1-8) |
| `coefficient` | 0 | Price coefficient override (0 = use country default tax profile) |
| `license_key` | — | Pro license key for premium sensors |
| `power_sensor` | — | HA entity ID for power consumption (e.g. `sensor.power_consumption`) |
| `debug_mode` | false | Enable verbose logging |

### Price Coefficient

The coefficient adjusts the final consumer price. A value of `0` uses the country's default tax profile (VAT, energy tax, ODE levy). Set a custom value to override — for example, `1.05` adds a 5% markup on top of the wholesale price.

## Enever Integration (Netherlands Only)

[Enever](https://enever.nl) is a Dutch price comparison service that provides real-time **all-in consumer prices** for your specific energy supplier. It is not an energy supplier itself — it aggregates pricing data from 24+ Dutch suppliers.

### Why Use Enever?

Without Enever, prices are calculated from wholesale rates plus standard tax profiles. With Enever, you get the **exact price your supplier charges**, including their specific markup, discounts, and fee structure.

### Setup

1. Visit [enever.nl](https://enever.nl) and request a free API token
2. In the addon settings, enable Enever and enter your token
3. Select your energy supplier from the dropdown
4. Restart the addon

### Rate Limits

The free Enever API tier allows **250 calls per month**. The addon uses an in-memory cache with a 2-hour TTL, resulting in approximately **3 API calls per day** — well within the free tier limit.

If you need higher rate limits or want to support the Enever project, consider becoming an [Enever Supporter](https://enever.nl/supporter).

### Supported Suppliers

Enever provides pricing data for 24 Dutch energy suppliers:

ANWB Energie, Budget Energie, Coolblue Energie, EasyEnergy, Energie Direct, Energie van Ons, Energiek, EnergyZero, Essent, Frank Energie, Groene Stroom Lokaal, Hegg, Innova Energie, Mijn Domein Energie, NextEnergy, Pure Energie, Quatt, SamSam, Tibber, Vandebron, Vattenfall, Vrij op naam, Wout Energie, Zonneplan.

## Freemium Model

| Feature | Free | Pro |
|---------|------|-----|
| Current price sensor | Yes | Yes |
| Cheapest/expensive hour | Yes | Yes |
| Prices today (hourly array) | Yes | Yes |
| GO/WAIT/AVOID action | — | Yes |
| Cheap hour binary sensor | Yes | Yes |
| Best window (configurable hours) | — | Yes |
| Tomorrow preview | — | Yes |
| Prices tomorrow | — | Yes |
| Live cost (needs power sensor) | — | Yes |
| Savings tracking | — | Yes |
| Usage score | — | Yes |
| Daily cost tracking | — | Yes |

### 14-Day Free Trial

Every new installation gets **14 days of full Pro access** — no license key needed. All Pro sensors and features are unlocked immediately. The trial countdown starts when the addon first runs.

After the trial ends, the addon reverts to the free tier. Enter a license key in Settings to keep Pro features permanently.

Pro licenses are available at [synctacles.com](https://synctacles.com).

## Troubleshooting

### Addon shows "Waiting for first price update"

The addon is starting up and fetching prices for the first time. This usually takes 5-15 seconds. If it persists, check the addon logs for errors.

### Enever shows "no prices in response"

This is normal outside of data availability windows. Day-ahead prices are typically published after 13:00 CET. The addon will automatically fall back to the next source in the chain.

### Source shows as unhealthy (red dot)

A source's circuit breaker has tripped after a failure. It will automatically recover after 2 hours. The addon uses the next available source in the meantime.

### Version shows "dev"

The Docker image was built without version injection. Rebuild with `make docker-addon` or use the `--build-arg VERSION=x.y.z` flag.
