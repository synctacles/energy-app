# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** MEDIUM
**Type:** Implementation

---

## DOEL

Grafana Infinity plugin installeren op monitor.synctacles.com en Pipeline Health dashboard configureren.

---

## STAP 1: INFINITY PLUGIN INSTALLEREN

```bash
# SSH naar monitor server
ssh monitor.synctacles.com

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

**Grafana UI:** Configuration → Data Sources → Add data source

| Setting | Waarde |
|---------|--------|
| Type | Infinity |
| Name | `SYNCTACLES-API` |
| URL | `https://api.synctacles.com` |
| Auth | None |

**Test:** Save & Test → moet "Success" tonen

---

## STAP 3: DASHBOARD PANELS

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

1. ✅ Plugin visible in Grafana plugins list
2. ✅ Data source test succeeds
3. ✅ Dashboard toont real-time data
4. ✅ Kleuren correct (A44/A65 green, A75 red/yellow)

---

## REFERENTIE

**API Endpoint:** `https://api.synctacles.com/v1/pipeline/health`

**Response structuur:**
```json
{
  "timers": {"collector": {"status": "OK"}, ...},
  "data": {"a75": {"status": "STALE", "norm_age_min": 189.7}, ...},
  "api": {"status": "OK"}
}
```

**Docs:** `/opt/github/synctacles-api/docs/GRAFANA_DASHBOARD_SETUP.md`

---

## DELIVERABLES

1. ✅ Infinity plugin geïnstalleerd
2. ✅ Data source geconfigureerd
3. ✅ Dashboard panels werkend
4. ✅ Screenshot of bevestiging

---

*Template versie: 1.0*
