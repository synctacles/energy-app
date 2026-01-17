# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** Investigation

---

## CONTEXT

Pipeline health endpoint toont data freshness problemen die onderzocht moeten worden VOORDAT we verder gaan met Grafana dashboard styling.

**Endpoint output:**
```json
{
  "a75": {"norm_age_min": 153.7, "status": "STALE"},
  "a65": {"norm_age_min": -1346.3, "status": "FRESH"},
  "a44": {"norm_age_min": 2373.7, "status": "UNAVAILABLE"}
}
```

---

## PROBLEMEN

### 1. A44 (Prices) - UNAVAILABLE (2373 min = ~40 uur oud)

Dit is een **pipeline failure**. Prijsdata is bijna 2 dagen oud.

**Onderzoek:**
```bash
# Check raw data
sudo -u postgres psql -d energy_insights_nl -c "
SELECT MAX(timestamp) as latest, 
       EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min 
FROM raw_entso_e_a44;"

# Check normalized data
sudo -u postgres psql -d energy_insights_nl -c "
SELECT MAX(timestamp) as latest,
       EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
FROM norm_entso_e_a44;"

# Check collector logs
journalctl -u energy-insights-nl-collector --since "48 hours ago" | grep -i "a44\|price\|error" | tail -50

# Check normalizer logs
journalctl -u energy-insights-nl-normalizer --since "48 hours ago" | grep -i "a44\|price\|error" | tail -50

# Check if ENTSO-E A44 endpoint werkt
# (handmatige curl naar ENTSO-E API indien mogelijk)
```

**Mogelijke oorzaken:**
- ENTSO-E A44 endpoint down/gewijzigd
- Collector haalt A44 niet op
- Normalizer verwerkt A44 niet
- API token verlopen

---

### 2. A65 (Load) - Negatieve age (-1346 min)

Negatieve age betekent: `MAX(timestamp) > NOW()`. 

Dit wijst op **toekomstige timestamps** in de database.

**Onderzoek:**
```bash
# Check timestamps in database
sudo -u postgres psql -d energy_insights_nl -c "
SELECT timestamp, NOW(), timestamp - NOW() as diff
FROM norm_entso_e_a65 
ORDER BY timestamp DESC 
LIMIT 10;"

# Check of er forecast data in zit (kan legitiem zijn)
sudo -u postgres psql -d energy_insights_nl -c "
SELECT COUNT(*) as future_records
FROM norm_entso_e_a65
WHERE timestamp > NOW();"

# Check raw data timestamps
sudo -u postgres psql -d energy_insights_nl -c "
SELECT timestamp
FROM raw_entso_e_a65
ORDER BY timestamp DESC
LIMIT 10;"
```

**Mogelijke oorzaken:**
- ENTSO-E A65 bevat forecast data (legitiem)
- Clock skew tussen servers
- Verkeerde timezone handling
- Test data met toekomstige timestamps

**Als forecast data legitiem is:** Endpoint moet worden aangepast om alleen historische data te meten, of aparte metric voor forecast vs actual.

---

### 3. A75 (Generation) - STALE (153 min)

Net binnen STALE threshold (90-180 min). Minder urgent maar check waarom niet FRESH.

**Onderzoek:**
```bash
# Check laatste data
sudo -u postgres psql -d energy_insights_nl -c "
SELECT MAX(timestamp) as latest,
       EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
FROM norm_entso_e_a75;"

# Check of collector recent draaide
journalctl -u energy-insights-nl-collector --since "3 hours ago" | grep -i "a75\|generation" | tail -20

# Check ENTSO-E A75 beschikbaarheid
# ENTSO-E publiceert met ~60 min vertraging, dus 90-120 min kan normaal zijn
```

---

## DIAGNOSE SAMENVATTING

Na onderzoek, documenteer per source:

```markdown
### A44 (Prices)
- **Raw data latest:** [timestamp]
- **Norm data latest:** [timestamp]  
- **Root cause:** [beschrijving]
- **Fix:** [actie]

### A65 (Load)
- **Future records:** [aantal]
- **Root cause:** [forecast data / clock skew / bug]
- **Fix:** [actie of "legitiem, endpoint aanpassen"]

### A75 (Generation)
- **Age:** [X min]
- **Root cause:** [ENTSO-E delay / collector issue / normaal]
- **Fix:** [actie of "geen actie nodig"]
```

---

## PRIORITEIT

1. **A44 eerst** - 40 uur oud is een echte pipeline failure
2. **A65 daarna** - Begrijpen of negatieve age een bug of feature is
3. **A75 laatste** - Waarschijnlijk normale ENTSO-E vertraging

---

## DELIVERABLES

1. Root cause per data source
2. Fixes waar nodig
3. Endpoint aanpassing als A65 forecast data legitiem is
4. Handoff met bevindingen

---

## GEEN GRAFANA WERK

Focus alleen op data issues. Grafana dashboard kan wachten tot de data correct is.

---

*Template versie: 1.0*
