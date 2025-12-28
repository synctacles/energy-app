# Rebranding Task: SYNCTACLES → Energy Insights NL

## Objectives
1. Rename all directories and modules
2. Replace all brand strings
3. Translate Dutch comments to English
4. Make brand ENV-driven where needed

## Specific Changes

### Directory Renames
```
custom_components/synctacles/ → custom_components/ha_energy_insights_nl/
synctacles_db/ → energy_insights_db/
sparkcrawler_db/ → data_collector/
```

### String Replacements
- `"SYNCTACLES"` → `"ENERGY_INSIGHTS_NL"` (uppercase constants)
- `"synctacles"` → `"energy_insights"` (lowercase identifiers)
- `"Synctacles"` → `"Energy Insights NL"` (display names)
- `"/opt/synctacles"` → `"/opt/energy-insights"`
- `"synctacles-"` → `"energy-insights-"` (systemd service names)

### Translate All Dutch Comments to English

**Search patterns for Dutch comments:**
- `# ` comments in .py files
- `# ` comments in .sh files
- Docstrings with Dutch text

**Examples:**
- `# Haal data op` → `# Fetch data`
- `# Controleer verbinding` → `# Check connection`
- `# Start de service` → `# Start the service`

### Files Priority Order

**Phase 1: Custom Component (Critical for HA)**
```
custom_components/synctacles/__init__.py
custom_components/synctacles/manifest.json
custom_components/synctacles/sensor.py
custom_components/synctacles/config_flow.py
custom_components/synctacles/const.py
custom_components/synctacles/strings.json
```

**Phase 2: Core Database Module**
```
synctacles_db/__init__.py
synctacles_db/models.py
synctacles_db/api/main.py
synctacles_db/api/endpoints/*.py
synctacles_db/normalizers/*.py
```

**Phase 3: Data Collectors**
```
sparkcrawler_db/__init__.py
sparkcrawler_db/collectors/*.py
sparkcrawler_db/importers/*.py
```

**Phase 4: System Services**
```
systemd/synctacles-api.service
systemd/synctacles-collector.service
systemd/synctacles-collector.timer
systemd/synctacles-importer.service
systemd/synctacles-importer.timer
systemd/synctacles-normalizer.service
systemd/synctacles-normalizer.timer
systemd/synctacles-tennet.service
systemd/synctacles-tennet.timer
systemd/synctacles-health.service
systemd/synctacles-health.timer
```

**Phase 5: Scripts**
```
scripts/setup/setup_synctacles_server_v2.3.3.sh
scripts/test/validate_synctacles_setup_v3.1.0.sh
scripts/run_collectors.sh
scripts/run_importers.sh
scripts/run_normalizers.sh
scripts/health_check.sh
scripts/backup_database.sh
scripts/cleanup_logs.sh
```

**Phase 6: Configuration**
```
.env.example
config/settings.py
alembic.ini
requirements.txt
```

**Phase 7: Documentation**
```
docs/*.md
README.md (already done)
```

## Special Considerations

### Do NOT Change:
- Git history
- External API endpoints (ENTSO-E, TenneT, Energy-Charts)
- Database table names (migration complexity)
- Variable names that are technically correct (e.g., `psr_type`)

### ENV-Driven Config (Add to .env.example):
```bash
# Branding
BRAND_NAME="Energy Insights NL"
BRAND_SLUG="energy-insights"
BRAND_DOMAIN="energy-insights.example.com"

# Paths
INSTALL_PATH="/opt/energy-insights"
LOG_PATH="/var/log/energy-insights"
DATA_PATH="/var/lib/energy-insights"
```

### Manifest.json Updates:
```json
{
  "domain": "ha_energy_insights_nl",
  "name": "Energy Insights NL",
  "documentation": "https://github.com/DATADIO/ha-energy-insights-nl",
  "issue_tracker": "https://github.com/DATADIO/ha-energy-insights-nl/issues"
}
```

## Testing Checklist After Rebranding

### Import Tests
- [ ] `from energy_insights_db import models` works
- [ ] `from data_collector.collectors import *` works
- [ ] Custom component imports work

### Service Tests
- [ ] Systemd service files reference correct paths
- [ ] Service user/group names updated
- [ ] Working directory paths correct

### Path Tests
- [ ] No hardcoded `/opt/synctacles` references
- [ ] Log paths updated
- [ ] Data paths updated

### String Tests
- [ ] No "SYNCTACLES" in user-facing strings
- [ ] API responses don't mention old brand
- [ ] Error messages use new brand

### Translation Tests
- [ ] All Dutch comments translated
- [ ] Technical accuracy maintained
- [ ] No broken references from translation

## Execution Strategy

1. **Backup First:** Commit current state
2. **Directory Renames:** Use `git mv` to preserve history
3. **Bulk Replace:** Use find/replace with caution
4. **Import Updates:** Fix all import statements
5. **Test:** Run basic imports and syntax checks
6. **Commit:** Stage changes incrementally by phase

## Notes for Claude Code

- Use regex for bulk replacements where safe
- Manually review systemd service files (critical)
- Test import statements after each phase
- Preserve technical accuracy in translations
- Flag any ambiguous Dutch phrases for review
