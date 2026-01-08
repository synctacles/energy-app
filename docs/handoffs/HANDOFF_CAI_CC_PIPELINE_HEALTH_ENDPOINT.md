# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH (P1)
**Type:** Implementation

---

## CONTEXT

Prometheus multi-worker issue omzeild. Nieuwe KISS aanpak:
- Geen Prometheus metrics
- Geen database poorten open
- Simpele JSON API endpoint
- Grafana haalt data via HTTPS (al beveiligd)

**Doel:** Bestaande dashboard uitbreiden: https://monitor.synctacles.com/d/services-status

---

## DEEL 1: REVERT PROMETHEUS CODE

De eerdere Prometheus metrics code is niet nodig. Revert alle uncommitted changes:

```bash
cd /opt/github/synctacles-api

# Check wat er gereverted moet worden
sudo -u energy-insights-nl git status

# Revert alle modified files
sudo -u energy-insights-nl git checkout -- synctacles_db/api/main.py
sudo -u energy-insights-nl git checkout -- synctacles_db/normalizers/normalize_entso_e_a75.py
sudo -u energy-insights-nl git checkout -- synctacles_db/normalizers/normalize_entso_e_a65.py
sudo -u energy-insights-nl git checkout -- synctacles_db/normalizers/normalize_entso_e_a44.py

# Verwijder nieuwe metrics file
rm -f synctacles_db/metrics.py

# Verify clean
sudo -u energy-insights-nl git status
# Moet "nothing to commit, working tree clean" tonen
```

---

## DEEL 2: NIEUWE API ENDPOINT

### File: `synctacles_db/api/routes/pipeline.py` (NIEUW)

```python
"""
Pipeline health endpoint for Grafana monitoring.
KISS approach: JSON endpoint, no Prometheus complexity.
"""
from fastapi import APIRouter
from datetime import datetime, timezone
import subprocess
from sqlalchemy import text
from synctacles_db.database import get_session

router = APIRouter(prefix="/v1/pipeline", tags=["pipeline"])


def get_timer_status(timer_name: str) -> dict:
    """Get systemd timer status."""
    full_name = f"energy-insights-nl-{timer_name}.timer"
    
    # Check if active
    result = subprocess.run(
        ["systemctl", "is-active", full_name],
        capture_output=True, text=True
    )
    is_active = result.stdout.strip() == "active"
    
    # Get last trigger time
    result = subprocess.run(
        ["systemctl", "show", full_name, "--property=LastTriggerUSec"],
        capture_output=True, text=True
    )
    last_trigger = None
    last_trigger_ago_min = None
    
    if "LastTriggerUSec=" in result.stdout:
        timestamp_str = result.stdout.strip().split("=")[1]
        if timestamp_str and timestamp_str != "n/a":
            try:
                # Parse systemd timestamp
                last_trigger = timestamp_str
                # Calculate minutes ago (simplified)
                result2 = subprocess.run(
                    ["systemctl", "show", full_name, "--property=LastTriggerUSecMonotonic"],
                    capture_output=True, text=True
                )
                if "=" in result2.stdout:
                    mono_usec = int(result2.stdout.strip().split("=")[1])
                    # Get current monotonic time
                    with open("/proc/uptime") as f:
                        uptime_sec = float(f.read().split()[0])
                    current_mono_usec = int(uptime_sec * 1_000_000)
                    age_min = (current_mono_usec - mono_usec) / 60_000_000
                    last_trigger_ago_min = round(age_min, 1)
            except:
                pass
    
    return {
        "active": is_active,
        "last_trigger": last_trigger,
        "last_trigger_ago_min": last_trigger_ago_min,
        "status": "OK" if is_active else "STOPPED"
    }


def get_data_freshness(source: str, raw_table: str, norm_table: str) -> dict:
    """Get data freshness from database."""
    session = get_session()
    try:
        # Raw data age (imported_at)
        raw_result = session.execute(text(f"""
            SELECT EXTRACT(EPOCH FROM (NOW() - MAX(imported_at)))/60 as age_min
            FROM {raw_table}
        """)).fetchone()
        raw_age = round(raw_result[0], 1) if raw_result and raw_result[0] else None
        
        # Normalized data age (timestamp)
        norm_result = session.execute(text(f"""
            SELECT EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
            FROM {norm_table}
        """)).fetchone()
        norm_age = round(norm_result[0], 1) if norm_result and norm_result[0] else None
        
        # Determine status
        if norm_age is None:
            status = "NO_DATA"
        elif norm_age < 90:
            status = "FRESH"
        elif norm_age < 180:
            status = "STALE"
        else:
            status = "UNAVAILABLE"
        
        # Detect pipeline gap (raw OK but norm stale = normalizer issue)
        pipeline_gap = None
        if raw_age is not None and norm_age is not None:
            pipeline_gap = round(norm_age - raw_age, 1)
        
        return {
            "raw_age_min": raw_age,
            "norm_age_min": norm_age,
            "pipeline_gap_min": pipeline_gap,
            "status": status
        }
    finally:
        session.close()


@router.get("/health")
def pipeline_health():
    """
    Complete pipeline health status for Grafana dashboard.
    
    Returns timer status and data freshness for all sources.
    """
    now = datetime.now(timezone.utc).isoformat()
    
    return {
        "timestamp": now,
        "timers": {
            "collector": get_timer_status("collector"),
            "importer": get_timer_status("importer"),
            "normalizer": get_timer_status("normalizer"),
            "health": get_timer_status("health")
        },
        "data": {
            "a75": get_data_freshness("a75", "raw_entso_e_a75", "norm_entso_e_a75"),
            "a65": get_data_freshness("a65", "raw_entso_e_a65", "norm_entso_e_a65"),
            "a44": get_data_freshness("a44", "raw_entso_e_a44", "norm_entso_e_a44")
        },
        "api": {
            "status": "OK",
            "workers": 8
        }
    }
```

### Registreer router in main.py

```python
# In synctacles_db/api/main.py, voeg toe bij andere router imports:
from synctacles_db.api.routes.pipeline import router as pipeline_router

# En registreer:
app.include_router(pipeline_router)
```

---

## DEEL 3: TEST ENDPOINT

```bash
# Restart API
sudo systemctl restart energy-insights-nl-api

# Test endpoint
curl -s http://localhost:8000/v1/pipeline/health | jq .

# Verwachte output:
{
  "timestamp": "2026-01-08T14:00:00+00:00",
  "timers": {
    "collector": {"active": true, "last_trigger_ago_min": 3.2, "status": "OK"},
    "importer": {"active": true, "last_trigger_ago_min": 11.6, "status": "OK"},
    "normalizer": {"active": true, "last_trigger_ago_min": 3.4, "status": "OK"},
    "health": {"active": true, "last_trigger_ago_min": 2.1, "status": "OK"}
  },
  "data": {
    "a75": {"raw_age_min": 15.2, "norm_age_min": 12.1, "pipeline_gap_min": -3.1, "status": "FRESH"},
    "a65": {"raw_age_min": 14.8, "norm_age_min": 11.5, "pipeline_gap_min": -3.3, "status": "FRESH"},
    "a44": {"raw_age_min": 18.1, "norm_age_min": 15.4, "pipeline_gap_min": -2.7, "status": "FRESH"}
  },
  "api": {"status": "OK", "workers": 8}
}

# Test via public URL
curl -s https://api.synctacles.com/v1/pipeline/health | jq .
```

---

## DEEL 4: GRAFANA DASHBOARD UPDATE

### Target Dashboard
**URL:** https://monitor.synctacles.com/d/services-status

### Data Source Setup (eenmalig)

1. Login op monitor.synctacles.com
2. Configuration → Data Sources → Add data source
3. Kies: **Infinity** (of **JSON API** plugin)
4. Configureer:
   - Name: `SYNCTACLES API`
   - URL: `https://api.synctacles.com`
   - Auth: None (of API key als nodig)
5. Save & Test

### Dashboard Layout

**Verwijder duplicaten.** Nieuwe layout:

```
┌─────────────────────────────────────────────────────────────────┐
│                       PIPELINE HEALTH                            │
├─────────────────────────────────────────────────────────────────┤
│                          SERVICES                                │
├───────────┬───────────┬───────────┬───────────┬─────────────────┤
│    API    │ Collector │ Importer  │Normalizer │     Health      │
│  🟢 OK    │  🟢 OK    │  🟢 OK    │  🟢 OK    │    🟢 OK        │
│  8 wrkrs  │  3.2 min  │  11.6 min │  3.4 min  │    2.1 min      │
├───────────┴───────────┴───────────┴───────────┴─────────────────┤
│                       DATA FRESHNESS                             │
├───────────┬───────────┬───────────┬───────────┬─────────────────┤
│  Source   │  Raw Age  │ Norm Age  │ Gap       │     Status      │
├───────────┼───────────┼───────────┼───────────┼─────────────────┤
│ A75 (gen) │  15 min   │  12 min   │  -3 min   │    🟢 FRESH     │
├───────────┼───────────┼───────────┼───────────┼─────────────────┤
│ A65 (load)│  14 min   │  11 min   │  -3 min   │    🟢 FRESH     │
├───────────┼───────────┼───────────┼───────────┼─────────────────┤
│ A44 (price)│ 18 min   │  15 min   │  -3 min   │    🟢 FRESH     │
└───────────┴───────────┴───────────┴───────────┴─────────────────┘
```

### Panel Configuratie

**Panel 1: API Status**
- Type: Stat
- Query: `$.api.status`
- Thresholds: OK=green, else=red

**Panel 2-5: Timer Status (Collector, Importer, Normalizer, Health)**
- Type: Stat
- Query: `$.timers.collector.status` (etc.)
- Value: `$.timers.collector.last_trigger_ago_min`
- Unit: minutes
- Thresholds: <20=green, <60=yellow, >=60=red

**Panel 6-8: Data Sources (A75, A65, A44)**
- Type: Stat
- Query: `$.data.a75.status` (etc.)
- Thresholds: FRESH=green, STALE=yellow, UNAVAILABLE=red

**Panel 9-11: Data Age Details (optional table)**
- Type: Table
- Query: `$.data`
- Columns: source, raw_age_min, norm_age_min, pipeline_gap_min, status

### Thresholds Reference

| Metric | Green | Yellow | Red |
|--------|-------|--------|-----|
| Timer age | < 20 min | < 60 min | >= 60 min |
| Data status | FRESH | STALE | UNAVAILABLE |
| Norm age | < 90 min | < 180 min | >= 180 min |
| Pipeline gap | < 30 min | < 60 min | >= 60 min |

---

## DEEL 5: VERWIJDER DUBBELE PANELS

Check bestaand dashboard voor duplicaten:
- Als er al "Importer - Last Run" etc. zijn → vervang door nieuwe panels
- Als er aparte service status panels zijn → consolideer in nieuwe layout
- Doel: 1 coherent dashboard, geen redundantie

---

## GIT COMMIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add synctacles_db/api/routes/pipeline.py synctacles_db/api/main.py
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "feat: add /v1/pipeline/health endpoint for Grafana monitoring

KISS approach - JSON endpoint instead of Prometheus metrics.
- Timer status (collector, importer, normalizer, health)
- Data freshness per source (A75, A65, A44)
- Pipeline gap detection (raw vs normalized age)
- No new ports needed - uses existing HTTPS

Enables complete pipeline monitoring in Grafana dashboard."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main
```

---

## VERIFICATION

1. Endpoint werkt: `curl https://api.synctacles.com/v1/pipeline/health`
2. Grafana data source geconfigureerd
3. Dashboard updated op https://monitor.synctacles.com/d/services-status
4. Geen dubbele panels
5. Thresholds correct (groen/geel/rood werkt)

---

## DELIVERABLES

1. ✅ Prometheus code gereverted
2. ✅ `/v1/pipeline/health` endpoint werkend
3. ✅ Grafana data source geconfigureerd
4. ✅ Dashboard bijgewerkt (geen duplicaten)
5. ✅ Screenshot van werkend dashboard

---

*Template versie: 1.0*
