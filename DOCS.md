# Synctacles Energy — Home Assistant Addon

Synctacles Energy is a fully local Home Assistant addon that provides real-time EU electricity prices with intelligent GO/WAIT/AVOID recommendations. All price data is fetched directly from free, public European energy APIs — no cloud dependency.

## How It Works

The addon fetches day-ahead electricity prices from multiple European data sources, normalizes them to consumer prices (including VAT, energy tax, and supplier markup where applicable), and publishes the results as Home Assistant sensors.

Every 15 minutes, the addon:
1. Fetches prices from the highest-priority source for your bidding zone
2. Falls back to alternative sources if the primary is unavailable
3. Calculates the current price, daily statistics, and action recommendation
4. Publishes all sensor values to Home Assistant

### Price Source Fallback Chain

Each bidding zone has a prioritized list of price sources. If the highest-priority source fails, the addon automatically tries the next one. A circuit breaker prevents hammering failed sources (2-hour cooldown).

| Tier | Description | GO Allowed |
|------|-------------|------------|
| 1-3 | Live API sources (zone-specific priority) | Yes |
| 4 | SQLite cache (48h retention) | No |

### GO/WAIT/AVOID Recommendations

The addon compares the current hour's price to the daily average:

| Action | Condition | Meaning |
|--------|-----------|---------|
| **GO** | Price is >15% below average | Great time to use electricity |
| **WAIT** | Price is within normal range | No urgency either way |
| **AVOID** | Price is >20% above average | Postpone if possible |

Thresholds are user-configurable. The addon also identifies the cheapest/most expensive hours and finds the best consecutive 3-hour window.

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

The addon publishes up to 11 sensors to Home Assistant:

### Free Tier

| Sensor | Entity ID | State | Description |
|--------|-----------|-------|-------------|
| Current Price | `sensor.synctacles_energy_price` | EUR/kWh | Current hour electricity price |
| Cheapest Hour | `sensor.synctacles_cheapest_hour` | HH:00 | Today's cheapest hour |
| Expensive Hour | `sensor.synctacles_expensive_hour` | HH:00 | Today's most expensive hour |
| Prices Today | `sensor.synctacles_prices_today` | count | Hourly prices array in attributes |

### Pro Tier

| Sensor | Entity ID | State | Description |
|--------|-----------|-------|-------------|
| Action | `sensor.synctacles_energy_action` | GO/WAIT/AVOID | Current recommendation |
| Best Window | `sensor.synctacles_best_window` | HH:00 - HH:00 | Best 3-hour consecutive window |
| Tomorrow Preview | `sensor.synctacles_tomorrow_preview` | FAVORABLE/NORMAL/EXPENSIVE/PENDING | Tomorrow's price outlook |
| Prices Tomorrow | `sensor.synctacles_prices_tomorrow` | count | Tomorrow's hourly prices in attributes |
| Live Cost | `sensor.synctacles_live_cost` | EUR/h | Real-time cost based on power sensor |
| Savings | `sensor.synctacles_savings` | EUR | Daily savings vs average price |
| Usage Score | `sensor.synctacles_usage_score` | 0-100 | How well you use cheap hours |

Live Cost, Savings, and Usage Score require a power sensor entity to be configured.

## Configuration

### Basic Settings

| Option | Default | Description |
|--------|---------|-------------|
| `zone` | NL | Your electricity bidding zone (see table above) |
| `go_threshold` | -15 | % below average to recommend GO |
| `avoid_threshold` | 20 | % above average to recommend AVOID |
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
| Best 3-hour window | — | Yes |
| Tomorrow preview | — | Yes |
| Prices tomorrow | — | Yes |
| Live cost (needs power sensor) | — | Yes |
| Savings tracking | — | Yes |
| Usage score | — | Yes |

Pro licenses are available at [synctacles.com](https://synctacles.com).

## Troubleshooting

### Addon shows "Waiting for first price update"

The addon is starting up and fetching prices for the first time. This usually takes 5-15 seconds. If it persists, check the addon logs for errors.

### Enever shows "no prices in response"

This is normal outside of data availability windows. Day-ahead prices are typically published after 13:00 CET. The addon will automatically fall back to the next source in the chain.

### Source shows as unhealthy (red dot)

A source's circuit breaker has tripped after a failure. It will automatically recover after 2 hours. The addon uses the next available source in the meantime.

### SQLite cache disabled

If you see "SQLite cache disabled" in logs, the addon couldn't create the cache database. This doesn't affect functionality — the addon works fine with just the in-memory cache. Prices will be re-fetched from live sources after each restart.

### Version shows "dev"

The Docker image was built without version injection. Rebuild with `make docker-addon` or use the `--build-arg VERSION=x.y.z` flag.
