# Energy Insights NL

Real-time Dutch electricity grid data for Home Assistant automation.

## What Makes This Different

**Single integration for ALL Dutch energy data:**
- ✅ Generation mix (wind/solar/gas/nuclear) - live breakdown
- ✅ Grid balance (import/export) - optimize usage timing  
- ✅ Load forecasts - plan ahead
- ✅ Normalized data - no API complexity

**vs Other Solutions:**
- ENTSO-E integration: Raw data only, no normalization
- Energy prices integrations: Only prices, no grid data
- Custom API scripts: Requires coding skills

**Key Benefits:**
- 15-minute updates
- Quality indicators (OK/STALE/NO_DATA)
- Fallback sources (ENTSO-E → Energy-Charts)
- Ready for automations (binary sensors)

## Pricing

**Beta (Now):** Free - test and provide feedback
**After Launch:** Paid subscription model
**Early Contributor Perk:** 1 year free for beta participants who contribute

Details: [pricing page after launch]

## Installation

Via HACS (recommended):
1. Add custom repository
2. Search "Energy Insights NL"
3. Configure with API endpoint

Manual: [link to docs]

## Use Cases

- EV charging when renewable % high
- Appliance scheduling on low-load periods
- Energy dashboards with live grid mix
- Home automation based on grid balance

## Technical

- Data sources: ENTSO-E, TenneT, Energy-Charts
- Update interval: 15 minutes
- Sensors: 8 entities
- Requirements: Home Assistant 2024.1+

## Support

Discord: [link]
Issues: GitHub Issues
