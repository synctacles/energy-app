# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** CRITICAL
**Type:** Bug Fix

---

## CONTEXT

Product Reality Check vond 3 HIGH priority issues. Productie is effectief stuk. Fix ALLE drie voordat GitHub issues worden aangemaakt.

---

## ISSUE 1: Database stale (2.5 dagen)

### Diagnose

```bash
# 1. Check collector logs
journalctl -u energy-insights-nl-collector --since "2026-01-05" --no-pager | tail -100

# 2. Check importer logs
journalctl -u energy-insights-nl-importer --since "2026-01-05" --no-pager | tail -100

# 3. Check normalizer logs
journalctl -u energy-insights-nl-normalizer --since "2026-01-05" --no-pager | tail -100

# 4. Check ENTSO-E API token
cat /opt/github/synctacles-api/.env | grep ENTSO

# 5. Test collector handmatig
cd /opt/github/synctacles-api
sudo -u energy-insights-nl /opt/github/synctacles-api/venv/bin/python -m collectors.entso_e_collector

# 6. Check database connectie
sudo -u energy-insights-nl /opt/github/synctacles-api/venv/bin/python -c "
from src.database import get_db_connection
conn = get_db_connection()
print('DB OK' if conn else 'DB FAIL')
"
```

### Fix

Gebaseerd op diagnose output:
- Als ENTSO-E token expired → update `.env`
- Als DB connectie faalt → check credentials
- Als collector error → fix code/config
- Als timer niet runt → `systemctl enable/start`

### Verificatie

```bash
# Wacht 15 min na fix, dan check:
sudo -u postgres psql -d energy_insights_nl -c "
SELECT MAX(timestamp) as latest FROM norm_entso_e_a75;
"
# Moet timestamp van < 30 min geleden zijn
```

---

## ISSUE 2: TenneT service disablen

### Actie

```bash
# Stop en disable TenneT service + timer
sudo systemctl stop energy-insights-nl-tennet.service
sudo systemctl stop energy-insights-nl-tennet.timer
sudo systemctl disable energy-insights-nl-tennet.service
sudo systemctl disable energy-insights-nl-tennet.timer

# Mask om per ongeluk starten te voorkomen
sudo systemctl mask energy-insights-nl-tennet.service
sudo systemctl mask energy-insights-nl-tennet.timer
```

### Verificatie

```bash
# Check status
systemctl status energy-insights-nl-tennet.service
systemctl status energy-insights-nl-tennet.timer
# Beide moeten "masked" of "disabled" zijn

# Check geen errors meer in logs
journalctl -u energy-insights-nl-tennet --since "10 minutes ago"
# Moet leeg zijn
```

---

## ISSUE 3: API staleness warnings

### Diagnose

```bash
# Check huidige API response
curl -s http://localhost:8000/v1/generation-mix | jq '.quality, .timestamp'

# Check of quality metadata bestaat
grep -r "quality\|stale\|fresh" /opt/github/synctacles-api/src/api/ --include="*.py"
```

### Fix Optie A: Quick fix (add staleness check)

Als staleness check niet bestaat, voeg toe aan API endpoints:

```python
# In relevante endpoint files
from datetime import datetime, timedelta

def get_quality_status(data_timestamp: datetime) -> dict:
    age = datetime.utcnow() - data_timestamp
    age_minutes = age.total_seconds() / 60
    
    if age_minutes < 90:
        status = "FRESH"
    elif age_minutes < 180:
        status = "STALE"
    else:
        status = "UNAVAILABLE"
    
    return {
        "quality_status": status,
        "data_age_minutes": int(age_minutes),
        "data_timestamp": data_timestamp.isoformat()
    }
```

### Fix Optie B: Config change

Als staleness check al bestaat maar thresholds verkeerd:

```bash
# Check config
grep -r "FRESH\|STALE\|90\|180" /opt/github/synctacles-api/src/
# Update thresholds indien nodig
```

### Verificatie

```bash
# Test endpoint
curl -s http://localhost:8000/v1/generation-mix | jq '.quality_status, .data_age_minutes'
# Moet UNAVAILABLE tonen als data > 3h oud
```

---

## VOLGORDE

1. **EERST** Issue 2 (TenneT disable) - quickest win, stopt log pollution
2. **DAN** Issue 1 (Database) - diagnose + fix data pipeline
3. **LAATSTE** Issue 3 (Staleness) - alleen relevant als Issue 1 gefixed

---

## GIT COMMITS

Per fix een aparte commit:

```bash
# Na Issue 2
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit --allow-empty -m "ops: disable TenneT server-side collector (BYO-only policy)"

# Na Issue 1 (als code change nodig)
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "fix: restore data pipeline - [beschrijf root cause]"

# Na Issue 3 (als code change nodig)
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "fix: add staleness warnings to API responses"

# Push alles
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main
```

---

## DELIVERABLES

1. TenneT service disabled + masked
2. Data pipeline werkend (verse data < 30 min)
3. API toont staleness status
4. Handoff response met root cause analyse

---

## NA FIXES

Pas NA alle fixes verified:
1. Update STATUS_CC_CURRENT.md
2. Maak GitHub issues aan voor MEDIUM/LOW gaps (of sluit als niet meer relevant)
3. Handoff naar CAI met results

---

*Template versie: 1.0*
