# Grafana Dashboard Setup - Pipeline Health Monitoring

**Datum:** 2026-01-08
**Status:** Grafana configuratie uitgesteld - Infinity plugin niet beschikbaar
**Alternatief:** Gebruik bestaand Prometheus-based dashboard

---

## SITUATIE

### Pipeline Health API

**Endpoint:** `https://api.synctacles.com/v1/pipeline/health`
**Status:** ✅ Operationeel (commit 051a847)

**Response:**
```json
{
  "timestamp": "2026-01-08T14:18:40+00:00",
  "timers": {
    "collector": {"active": true, "status": "OK", "last_trigger": "Thu 2026-01-08 14:14:48 UTC"},
    "importer": {"active": true, "status": "OK"},
    "normalizer": {"active": true, "status": "OK"},
    "health": {"active": true, "status": "OK"}
  },
  "data": {
    "a75": {"raw_age_min": 78.7, "norm_age_min": 153.7, "pipeline_gap_min": 75.0, "status": "STALE"},
    "a65": {"raw_age_min": -1406.3, "norm_age_min": -1346.3, "pipeline_gap_min": 60.0, "status": "FRESH"},
    "a44": {"raw_age_min": 2373.7, "norm_age_min": 2373.7, "pipeline_gap_min": 0.0, "status": "UNAVAILABLE"}
  },
  "api": {"status": "OK", "workers": 8}
}
```

### Grafana Setup

**URL:** https://monitor.synctacles.com
**Dashboard:** https://monitor.synctacles.com/d/services-status/services-status
**Grafana Version:** 12.3.1

**Bestaande panels:**
- API Service (stat)
- Importer Service (stat)
- Normalizer Service (stat)
- Collector Service (stat)
- Importer - Last Run (stat)
- Normalizer - Last Run (stat)
- Collector - Last Run (stat)
- Timer Status (stat)

---

## PROBLEEM

**Infinity plugin niet beschikbaar** in Grafana 12.3.1
→ Kan JSON API niet direct als data source gebruiken

**Beschikbare plugins:**
- Prometheus ✅ (al in gebruik)
- PostgreSQL ✅ (maar Grafana draait op andere server)
- Testdata ✅ (voor demo)

---

## OPTIES

### Optie A: Installeer Infinity Plugin (AANBEVOLEN)

```bash
# Op monitor.synctacles.com server
grafana-cli plugins install yesoreyeram-infinity-datasource
systemctl restart grafana-server
```

Daarna volg [Optie A instructies](#optie-a-infinity-plugin) hieronder.

### Optie B: PostgreSQL Direct Query

Grafana kan direct naar de database als:
1. Database poort 5432 open staat voor monitor.synctacles.com
2. PostgreSQL user voor Grafana aangemaakt

**Nadelen:**
- Security: extra database access
- Duplicatie: zelfde queries als API endpoint

### Optie C: Gebruik Bestaand Prometheus Dashboard

**Huidige status:** Dashboard gebruikt al Prometheus metrics voor services.

**Actie:** Niets - huidige dashboard is voldoende voor basic monitoring.

**Nadelen:**
- Geen data freshness metrics
- Geen pipeline gap detection

---

## IMPLEMENTATIE: OPTIE A (Infinity Plugin)

### Stap 1: Installeer Plugin

```bash
ssh monitor.synctacles.com
sudo grafana-cli plugins install yesoreyeram-infinity-datasource
sudo systemctl restart grafana-server
```

### Stap 2: Configureer Data Source

1. Login: https://monitor.synctacles.com
   - Username: `admin`
   - Password: `?T4Ew7Vo@Mij%i2q=v+=kpgro`

2. **Configuration → Data Sources → Add data source**

3. Zoek: **Infinity**

4. Configuratie:
   - **Name:** `SYNCTACLES Pipeline API`
   - **URL:** `https://api.synctacles.com`
   - **Auth:** None (API is public)
   - **Allowed hosts:** `api.synctacles.com`

5. **Save & Test** → Should show "Data source is working"

### Stap 3: Update Dashboard

Open: https://monitor.synctacles.com/d/services-status/services-status

#### Panel 1: API Status

**Add Panel → Stat**
- **Title:** API Status
- **Data source:** SYNCTACLES Pipeline API
- **Query Type:** JSON
- **URL:** `/v1/pipeline/health`
- **Parser:** Backend
- **Rows / Root:** `api`
- **Columns:**
  - Field: `status` (Type: String)

**Thresholds:**
- OK → Green
- Default → Red

**Grid Position:** Top row, first panel

#### Panel 2-5: Timer Status (Collector, Importer, Normalizer, Health)

**Example: Collector Timer**

**Add Panel → Stat**
- **Title:** Collector Service
- **Data source:** SYNCTACLES Pipeline API
- **Query Type:** JSON
- **URL:** `/v1/pipeline/health`
- **Parser:** Backend
- **Rows / Root:** `timers.collector`
- **Columns:**
  - Field: `status` (Type: String)
  - Field: `last_trigger` (Type: String) - voor subtitle

**Value Options:**
- **Show:** All values
- **Calculation:** Last (not null)

**Thresholds:**
- OK → Green
- STOPPED → Red
- Default → Yellow

**Text:**
- **Title:** Collector Service
- **Value:** `${__field.name:status}`
- **Subtitle:** Last: `${__field.name:last_trigger}`

Herhaal voor `importer`, `normalizer`, `health` met `timers.X` als root.

#### Panel 6-8: Data Freshness (A75, A65, A44)

**Example: A75 Generation Data**

**Add Panel → Stat**
- **Title:** A75 - Generation Data
- **Data source:** SYNCTACLES Pipeline API
- **Query Type:** JSON
- **URL:** `/v1/pipeline/health`
- **Parser:** Backend
- **Rows / Root:** `data.a75`
- **Columns:**
  - Field: `status` (Type: String)
  - Field: `norm_age_min` (Type: Number)
  - Field: `pipeline_gap_min` (Type: Number)

**Value Options:**
- Display status with age as subtitle

**Thresholds:**
- FRESH → Green
- STALE → Yellow
- UNAVAILABLE → Red
- NO_DATA → Gray

**Text:**
- **Title:** A75 Generation
- **Value:** `${__field.name:status}`
- **Subtitle:** Age: `${__field.name:norm_age_min}` min (Gap: `${__field.name:pipeline_gap_min}` min)

Herhaal voor A65 (Load) en A44 (Prices).

#### Panel 9: Data Age Table (Optional)

**Add Panel → Table**
- **Title:** Pipeline Data Status
- **Data source:** SYNCTACLES Pipeline API
- **Query Type:** JSON
- **URL:** `/v1/pipeline/health`
- **Parser:** Backend
- **Rows / Root:** `data`
- **Columns:**
  - Source (String)
  - raw_age_min (Number)
  - norm_age_min (Number)
  - pipeline_gap_min (Number)
  - status (String)

**Transform:**
- Add field from calculation: Row index → Source name
- Organize fields: reorder columns

**Column Styles:**
- status → Color by thresholds (FRESH=green, STALE=yellow, UNAVAILABLE=red)
- *_age_min → Unit: minutes

### Stap 4: Opschonen Duplicaten

**Te verwijderen panels:**
- "Importer - Last Run" (vervangen door nieuwe timer panels)
- "Normalizer - Last Run" (vervangen)
- "Collector - Last Run" (vervangen)
- "Timer Status" (vervangen door individuele panels)

**Behouden:**
- API Service ✅
- [Importer/Normalizer/Collector] Service ✅ (update data source naar nieuwe API)

---

## IMPLEMENTATIE: OPTIE B (PostgreSQL)

**Vereisten:**
1. Open PostgreSQL poort op ENIN-NL voor monitor.synctacles.com IP
2. Maak readonly user voor Grafana

### Stap 1: PostgreSQL User

```bash
# Op ENIN-NL server
sudo -u postgres psql energy_insights_nl

CREATE USER grafana_readonly WITH PASSWORD 'secure_password_here';
GRANT CONNECT ON DATABASE energy_insights_nl TO grafana_readonly;
GRANT USAGE ON SCHEMA public TO grafana_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO grafana_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO grafana_readonly;
```

### Stap 2: PostgreSQL Config

```bash
# /etc/postgresql/XX/main/pg_hba.conf
# Voeg toe (vervang XX.XX.XX.XX met monitor.synctacles.com IP):
host    energy_insights_nl    grafana_readonly    XX.XX.XX.XX/32    scram-sha-256

# Restart
sudo systemctl restart postgresql
```

### Stap 3: Grafana Data Source

1. **Configuration → Data Sources → Add data source**
2. **PostgreSQL**
3. Configuratie:
   - **Host:** `ENIN-NL-IP:5432`
   - **Database:** `energy_insights_nl`
   - **User:** `grafana_readonly`
   - **Password:** (from step 1)
   - **SSL Mode:** disable (or require als SSL configured)

### Stap 4: Panel Queries

**Example: A75 Data Freshness**

```sql
SELECT
  'a75' as source,
  EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as norm_age_min,
  CASE
    WHEN EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 < 90 THEN 'FRESH'
    WHEN EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 < 180 THEN 'STALE'
    ELSE 'UNAVAILABLE'
  END as status
FROM norm_entso_e_a75;
```

Herhaal voor A65, A44 met hun respectievelijke tabellen.

**Voordeel:** Direct database queries
**Nadeel:** Security (extra database access)

---

## THRESHOLD REFERENTIE

### Data Freshness

| Status | Minutes | Color | Meaning |
|--------|---------|-------|---------|
| FRESH | < 90 | Green | Data is up-to-date |
| STALE | 90-180 | Yellow | Data is outdated |
| UNAVAILABLE | >= 180 | Red | No recent data |
| NO_DATA | N/A | Gray | No data in table |

### Service Timers

| Status | Color | Meaning |
|--------|-------|---------|
| OK | Green | Timer active |
| STOPPED | Red | Timer inactive |

### Pipeline Gap

| Gap (minutes) | Color | Meaning |
|---------------|-------|---------|
| < 30 | Green | Normal processing lag |
| 30-60 | Yellow | Slow normalizer |
| >= 60 | Red | Normalizer problem |

Negative gap = normalized data newer than raw (impossible, indicates time skew).

---

## ALTERNATIEF: GEBRUIK PROMETHEUS

**Huidige situatie:** Dashboard gebruikt Prometheus metrics.

**Optie:** Extend API met Prometheus metrics endpoint (al aanwezig: `/metrics`).

**Prometheus Metrics Toevoegen:**

```python
# synctacles_db/metrics.py (reuse van eerdere Grafana poging)
from prometheus_client import Gauge

pipeline_data_age = Gauge(
    'pipeline_data_age_minutes',
    'Age of normalized data in minutes',
    ['source']
)

# In normalizers: na succesvolle run
from synctacles_db.metrics import pipeline_data_age
pipeline_data_age.labels(source='a75').set(age_minutes)
```

**Probleem:** Gunicorn multi-worker (zelfde blocker als eerder).

---

## AANBEVELING

**Voor nu:** Behoud bestaand Prometheus dashboard.

**Toekomst:** Installeer Infinity plugin (Optie A) voor volledige pipeline visibility.

**Reden:**
- Infinity plugin = 5 minuten installatie
- Direct JSON API integratie
- Geen extra database access
- Follows KISS principe

---

## TROUBLESHOOTING

### "Data source is working" maar geen data

**Check:**
1. API endpoint bereikbaar: `curl https://api.synctacles.com/v1/pipeline/health`
2. CORS instellingen als Grafana browser-based request doet
3. Infinity plugin correct geconfigureerd (Backend parser, niet Browser)

### Panels tonen "No data"

**Check:**
1. Query URL correct (`/v1/pipeline/health` not `https://...`)
2. Parser: Backend (niet Browser of Manual)
3. Root path correct (`data.a75` niet `$.data.a75`)
4. Column selectors matchen response structure

### Thresholds werken niet

**Check:**
1. Field type = String voor status values
2. Thresholds in Value mappings of Overrides
3. Threshold mode = "Absolute" niet "Percentage"

---

## REFERENTIES

- **API Endpoint:** [synctacles_db/api/routes/pipeline.py](../synctacles_db/api/routes/pipeline.py)
- **Handoff:** [HANDOFF_CC_CAI_PIPELINE_DASHBOARD_PROGRESS.md](handoffs/HANDOFF_CC_CAI_PIPELINE_DASHBOARD_PROGRESS.md)
- **Infinity Plugin:** https://grafana.com/grafana/plugins/yesoreyeram-infinity-datasource/
- **Grafana Dashboard:** https://monitor.synctacles.com/d/services-status

---

*Document versie: 1.0*
*Laatste update: 2026-01-08*
