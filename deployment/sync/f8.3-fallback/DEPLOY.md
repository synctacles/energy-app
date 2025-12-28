# F8.3 Fallback APIs - Deployment Manifest

## Files
- `energy_charts.py` → `/opt/synctacles/app/synctacles_db/fallback/`
- `manager.py` → `/opt/synctacles/app/synctacles_db/fallback/`

## Dependencies
Add to requirements.txt:
- cachetools==5.3.2

## Validation
```bash
# Test fallback (truncate DB to force fallback)
psql -U synctacles -d synctacles -c "TRUNCATE norm_entso_e_a75;"
curl http://localhost:8000/api/v1/generation-mix | jq .meta
# Should show: "source": "Energy-Charts", "quality_status": "FALLBACK"
```
