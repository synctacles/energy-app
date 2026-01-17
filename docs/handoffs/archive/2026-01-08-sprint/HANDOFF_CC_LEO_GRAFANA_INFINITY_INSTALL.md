# HANDOFF: CC → LEO

**Datum:** 2026-01-08
**Van:** CC (Claude Code)
**Naar:** Leo
**Prioriteit:** MEDIUM
**Type:** Manual Execution Required

---

## SITUATIE

De handoff `HANDOFF_CAI_CC_GRAFANA_INFINITY_DASHBOARD.md` vraagt om Grafana Infinity plugin installatie op monitor.synctacles.com. CC heeft geen SSH toegang naar die server, dus dit moet jij uitvoeren.

---

## WAT UITGEVOERD MOET WORDEN

### Voorbereiding
- SSH naar monitor.synctacles.com (77.42.41.135)
- Gebruik service account indien beschikbaar, anders root

---

## STAP 1: INFINITY PLUGIN INSTALLEREN

```bash
# Installeer plugin
sudo grafana-cli plugins install yesoreyeram-infinity-datasource

# Restart Grafana
sudo systemctl restart grafana-server

# Verify installatie
grafana-cli plugins ls | grep infinity
```

**Verwacht:** `yesoreyeram-infinity-datasource @ x.x.x`

---

## STAP 2: DATA SOURCE CONFIGUREREN

**Via Grafana UI:** http://monitor.synctacles.com → Configuration → Data Sources → Add data source

| Setting | Waarde |
|---------|--------|
| Type | Infinity |
| Name | `SYNCTACLES-API` |
| URL | `https://api.synctacles.com` |
| Auth | None |

**Test:** Save & Test → moet "Success" tonen

---

## STAP 3: DASHBOARD PANELS CONFIGUREREN

**Dashboard:** https://monitor.synctacles.com/d/services-status (of nieuw aanmaken)

### Panel A: API Status (Stat)
```
Query URL: /v1/pipeline/health
Parser: JSON
JSONPath: $.api.status
Thresholds: OK=green, else=red
```

### Panel B: Timer Status (4x Stat in row)
```
Collector:  $.timers.collector.status
Importer:   $.timers.importer.status
Normalizer: $.timers.normalizer.status
Health:     $.timers.health.status

Thresholds: OK=green, STOPPED=red
```

### Panel C: Data Freshness (3x Stat in row)
```
A75: $.data.a75.status
A65: $.data.a65.status
A44: $.data.a44.status

Thresholds: FRESH=green, STALE=yellow, UNAVAILABLE=red
```

### Panel D: Data Age Table (Optional)
```
Query URL: /v1/pipeline/health
Parser: JSON
JSONPath: $.data
Columns: source, norm_age_min, status
```

---

## VERIFICATIE

Nadat je dit hebt uitgevoerd, check:

1. ✅ Plugin visible in Grafana plugins list
2. ✅ Data source test succeeds
3. ✅ Dashboard toont real-time data
4. ✅ Kleuren correct (A44/A65 green, A75 red/yellow als verwacht)

---

## API ENDPOINT REFERENTIE

**Endpoint:** `https://api.synctacles.com/v1/pipeline/health`

**Response structuur:**
```json
{
  "timers": {
    "collector": {"status": "OK", "last_run": "2026-01-08T15:44:00Z"},
    "importer": {"status": "OK", "last_run": "2026-01-08T15:45:00Z"},
    "normalizer": {"status": "OK", "last_run": "2026-01-08T15:46:00Z"},
    "health": {"status": "OK", "last_run": "2026-01-08T15:47:00Z"}
  },
  "data": {
    "a75": {"status": "UNAVAILABLE", "norm_age_min": 189.7},
    "a65": {"status": "FRESH", "norm_age_min": 9.7},
    "a44": {"status": "FRESH", "norm_age_min": 9.7}
  },
  "api": {"status": "OK"}
}
```

**Current Status (2026-01-08 16:15):**
- A44 (Prices): FRESH (9.7 min) ✅
- A65 (Load): FRESH (9.7 min) ✅
- A75 (Generation): UNAVAILABLE (189 min) - expected ENTSO-E delay ✅

---

## WAAROM CC DIT NIET KAN

CC heeft geen SSH toegang naar monitor.synctacles.com:
- Geprobeerd als root: Permission denied
- Geprobeerd als energy-insights-nl: Permission denied
- SSH key van ENIN-NL naar monitor bestaat niet

---

## DELIVERABLES

Na uitvoering door jou:
1. ✅ Infinity plugin geïnstalleerd
2. ✅ Data source geconfigureerd
3. ✅ Dashboard panels werkend
4. ✅ Screenshot of bevestiging aan CC

---

## REFERENTIE DOCUMENTATIE

- CAI's originele handoff: `/opt/github/synctacles-api/docs/handoffs/HANDOFF_CAI_CC_GRAFANA_INFINITY_DASHBOARD.md`
- Grafana setup guide: `/opt/github/synctacles-api/docs/GRAFANA_DASHBOARD_SETUP.md`
- Data freshness fix: `/opt/github/synctacles-api/docs/handoffs/HANDOFF_CC_CAI_DATA_FRESHNESS_FIXED.md`

---

*Template versie: 1.0*
*Prepared by CC: 2026-01-08 16:20 UTC*
