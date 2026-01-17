# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH (P1)
**Type:** Implementation + Cleanup

---

## CONTEXT

Pipeline Dashboard was eerder geschat op 7-10 uur. Na correctie (Grafana al operationeel op monitor.synctacles.com) is dit nu 2-4 uur.

Vandaag's normalizer bug (2.5 dagen stale data) was instant zichtbaar geweest met dit dashboard.

---

## DEEL 1: GRAFANA CLEANUP OP API SERVER

### Waarom

- Grafana draait op **monitor.synctacles.com** (dedicated)
- Op ENIN-NL is grafana-server.service inactive en nutteloos
- Veroorzaakt verwarring (CC checkte verkeerde server)
- Clean separation: API server ≠ monitoring server
- Minder attack surface + resources

### Task: Verwijder Grafana van ENIN-NL

```bash
# 1. Stop en disable (indien actief)
sudo systemctl stop grafana-server 2>/dev/null
sudo systemctl disable grafana-server 2>/dev/null

# 2. Verwijder Grafana packages
sudo apt-get remove --purge grafana grafana-enterprise -y
sudo apt-get autoremove -y

# 3. Verwijder config/data directories
sudo rm -rf /etc/grafana
sudo rm -rf /var/lib/grafana
sudo rm -rf /var/log/grafana

# 4. Verify volledig verwijderd
which grafana-server  # Moet niets returnen
systemctl list-units | grep grafana  # Moet niets returnen
dpkg -l | grep grafana  # Moet niets returnen
```

### Verificatie

```bash
# Moet allemaal LEEG zijn:
systemctl status grafana-server 2>&1  # "Unit not found"
ls /etc/grafana 2>&1  # "No such file"
ls /var/lib/grafana 2>&1  # "No such file"
```

---

## DEEL 2: PIPELINE DASHBOARD IMPLEMENTATIE

### Prerequisites Check

```bash
# 1. Verify Grafana accessible
curl -sI https://monitor.synctacles.com/ | head -3
# Moet HTTP 200 of 302 (redirect naar login) tonen

# 2. Verify API metrics endpoint
curl -s http://localhost:8000/metrics | head -10
# Moet Prometheus metrics tonen

# 3. Check of Prometheus API metrics al scraped
# (Dit moet in Grafana UI - login required)
```

### Dashboard Concept

```
┌─────────────────────────────────────────────────────────────────┐
│                    PIPELINE HEALTH DASHBOARD                     │
├─────────────┬─────────────┬─────────────┬─────────────┬─────────┤
│   Source    │  Collector  │  Importer   │ Normalizer  │   API   │
├─────────────┼─────────────┼─────────────┼─────────────┼─────────┤
│    A75      │   🟢 OK     │   🟢 OK     │   🟢 OK     │  🟢 OK  │
│ (generation)│  3.2 min    │  11.6 min   │  3.4 min    │  fresh  │
├─────────────┼─────────────┼─────────────┼─────────────┼─────────┤
│    A65      │   🟢 OK     │   🟢 OK     │   🟢 OK     │  🟢 OK  │
│   (load)    │  3.2 min    │  11.6 min   │  3.4 min    │  fresh  │
├─────────────┼─────────────┼─────────────┼─────────────┼─────────┤
│    A44      │   🟢 OK     │   🟢 OK     │   🟢 OK     │  🟢 OK  │
│  (prices)   │  3.2 min    │  11.6 min   │  3.4 min    │  fresh  │
└─────────────┴─────────────┴─────────────┴─────────────┴─────────┘
```

### Benodigde Metrics

**Check of deze metrics bestaan in /metrics:**

```bash
curl -s http://localhost:8000/metrics | grep -E "collector|importer|normalizer|data_age|freshness"
```

**Indien metrics ontbreken, moeten ze toegevoegd worden aan de code:**

```python
# Voorbeeld metrics die nodig zijn:
pipeline_collector_last_success_timestamp{source="a75"}
pipeline_importer_last_success_timestamp{source="a75"}
pipeline_normalizer_last_success_timestamp{source="a75"}
pipeline_data_age_minutes{source="a75", layer="normalized"}
```

### Dashboard Panels (Grafana)

**Panel 1: Collector Status per Source**
- Type: Stat
- Query: `time() - pipeline_collector_last_success_timestamp{source="$source"}`
- Thresholds: Green < 20min, Yellow < 60min, Red >= 60min

**Panel 2: Importer Status per Source**
- Type: Stat
- Query: `time() - pipeline_importer_last_success_timestamp{source="$source"}`
- Thresholds: Green < 20min, Yellow < 60min, Red >= 60min

**Panel 3: Normalizer Status per Source**
- Type: Stat
- Query: `time() - pipeline_normalizer_last_success_timestamp{source="$source"}`
- Thresholds: Green < 20min, Yellow < 60min, Red >= 60min

**Panel 4: Data Freshness**
- Type: Stat
- Query: `pipeline_data_age_minutes{source="$source", layer="normalized"}`
- Thresholds: Green < 90, Yellow < 180, Red >= 180

### Alert Rules (Optional)

```yaml
# Collector stuk
- alert: CollectorStale
  expr: time() - pipeline_collector_last_success_timestamp > 3600
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Collector {{ $labels.source }} stale > 1 hour"

# Normalizer stuk (vandaag's bug)
- alert: NormalizerStale
  expr: time() - pipeline_normalizer_last_success_timestamp > 3600
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Normalizer {{ $labels.source }} stale > 1 hour"

# Data te oud
- alert: DataUnavailable
  expr: pipeline_data_age_minutes > 180
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Data {{ $labels.source }} older than 3 hours"
```

---

## DEEL 3: METRICS INSTRUMENTATIE (INDIEN NODIG)

Als de benodigde metrics niet bestaan, voeg toe aan de code:

**File:** `src/metrics.py` (of waar prometheus metrics gedefinieerd zijn)

```python
from prometheus_client import Gauge

# Pipeline metrics
pipeline_collector_last_success = Gauge(
    'pipeline_collector_last_success_timestamp',
    'Timestamp of last successful collector run',
    ['source']
)

pipeline_importer_last_success = Gauge(
    'pipeline_importer_last_success_timestamp',
    'Timestamp of last successful importer run',
    ['source']
)

pipeline_normalizer_last_success = Gauge(
    'pipeline_normalizer_last_success_timestamp',
    'Timestamp of last successful normalizer run',
    ['source']
)

pipeline_data_age = Gauge(
    'pipeline_data_age_minutes',
    'Age of data in minutes',
    ['source', 'layer']
)
```

**Update collectors/importers/normalizers:**
```python
# Na succesvolle run:
from src.metrics import pipeline_collector_last_success
pipeline_collector_last_success.labels(source='a75').set_to_current_time()
```

---

## VOLGORDE

1. **Grafana cleanup** op ENIN-NL (5 min)
2. **Check metrics** bestaan in /metrics endpoint (5 min)
3. **Als metrics ontbreken:** instrumentatie toevoegen (1-2u)
4. **Verify Prometheus scraping** op monitor.synctacles.com (15 min)
5. **Create dashboard** in Grafana (1-2u)
6. **Test** met simulated failure (30 min)
7. **Optional:** Alert rules toevoegen (30 min)

---

## DELIVERABLES

1. ✅ Grafana volledig verwijderd van ENIN-NL
2. ✅ Pipeline metrics beschikbaar in /metrics
3. ✅ Prometheus scraping verified
4. ✅ Pipeline Health Dashboard in Grafana
5. ✅ Documentatie update (MONITORING.md)
6. Optional: Alert rules geconfigureerd

---

## GIT COMMIT (indien code changes)

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add .
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "feat: add pipeline health metrics for Grafana dashboard

- Add collector/importer/normalizer success timestamp metrics
- Add data_age_minutes metric per source
- Enables pipeline health monitoring in Grafana

Related: Product Reality Check audit - proactive monitoring"

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main
```

---

## VERIFICATIE

Na implementatie:
1. Grafana cleanup: `dpkg -l | grep grafana` moet leeg zijn
2. Metrics: `curl localhost:8000/metrics | grep pipeline` toont metrics
3. Dashboard: Screenshot van werkend dashboard
4. Test: Stop normalizer, verify dashboard toont rood

---

*Template versie: 1.0*
