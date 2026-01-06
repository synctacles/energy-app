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

## Production Monitoring

This project is moving toward production. Monitoring infrastructure is being set up.

**Current Status:**
- ✅ Code quality: 100% SKILL_11 compliant
- 🟡 Monitoring: In progress (Issue #24)
- 📊 Load testing: Planned

**Documentation:**
- [Code Quality Audit Report](CODE_QUALITY_AUDIT_REPORT.md) - 0 credential violations, all checks passed
- [Production Blockers](PRODUCTION_BLOCKERS.md) - Current production readiness status
- [Monitoring Project Overview](MONITORING_PROJECT_OVERVIEW.md) - Full monitoring infrastructure plan
- [Quick Start Guide](MONITORING_QUICK_START.md) - Get started quickly

**GitHub Project Issues:**
- [#24 - Main Monitoring Project](https://github.com/DATADIO/synctacles-api/issues/24)
- [#25 - Phase 1: CX23 Server Setup](https://github.com/DATADIO/synctacles-api/issues/25)
- [#26 - Phase 2: node-exporter Setup](https://github.com/DATADIO/synctacles-api/issues/26)
- [#27 - Phase 3: AlertManager & Slack](https://github.com/DATADIO/synctacles-api/issues/27)
- [#28 - Phase 4: Grafana Dashboards](https://github.com/DATADIO/synctacles-api/issues/28)
- [#29 - Phase 5: Load Testing](https://github.com/DATADIO/synctacles-api/issues/29)
- [#30 - Phase 6: Documentation](https://github.com/DATADIO/synctacles-api/issues/30)

## Support

Discord: [link]
Issues: GitHub Issues
