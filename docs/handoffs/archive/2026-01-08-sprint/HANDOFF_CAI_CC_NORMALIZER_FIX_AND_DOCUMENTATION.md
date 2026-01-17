# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** CRITICAL
**Type:** Bug Fix + Investigation + Documentation

---

## EXECUTIVE SUMMARY

A65 en A75 normalizers ontbreken in `run_normalizers.sh`. Dit is een bug, geen design keuze. Fix direct, documenteer wat normalisatie doet, en update SKILLs.

---

## DEEL 1: KRITIEKE FIX

### Fix run_normalizers.sh

**Locatie:** `/opt/energy-insights-nl/app/scripts/run_normalizers.sh`

**Huidige inhoud (BROKEN):**
```bash
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices
```

**Nieuwe inhoud (FIXED):**
```bash
# ENTSO-E normalizers (alle 3 sources)
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A44 (prices)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A65 (load)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A75 (generation)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75

# Price post-processing
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing price aggregation..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_prices
```

### Deployment

```bash
# 1. Edit script
sudo nano /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# 2. Kopieer naar git repo
sudo cp /opt/energy-insights-nl/app/scripts/run_normalizers.sh \
        /opt/github/synctacles-api/scripts/run_normalizers.sh

# 3. Run handmatig om backlog te verwerken
sudo -u energy-insights-nl bash /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# 4. Verificatie
sudo -u postgres psql -d energy_insights_nl -c "
SELECT 
    'A44' as source,
    MAX(timestamp) as latest,
    EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
FROM norm_entso_e_a44 WHERE timestamp <= NOW()
UNION ALL
SELECT 
    'A65' as source,
    MAX(timestamp) as latest,
    EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
FROM norm_entso_e_a65 WHERE timestamp <= NOW()
UNION ALL
SELECT 
    'A75' as source,
    MAX(timestamp) as latest,
    EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
FROM norm_entso_e_a75 WHERE timestamp <= NOW();"

# 5. Git commit
cd /opt/github/synctacles-api
sudo -u energy-insights-nl git add scripts/run_normalizers.sh
sudo -u energy-insights-nl git commit -m "fix(critical): add missing A65/A75 normalizers to run_normalizers.sh

A65 (load) and A75 (generation) normalizers were never added to the
timer script. This caused normalized data to become stale while raw
data was being collected correctly.

Added:
- normalize_entso_e_a65 (load data)
- normalize_entso_e_a75 (generation data)
- Per-source logging for debugging

Fixes: Silent normalizer failure discovered during Grafana dashboard implementation"

sudo -u energy-insights-nl git push origin main
```

---

## DEEL 2: RAW VS NORMALIZED ANALYSE

### Opdracht

Onderzoek en documenteer het exacte verschil tussen raw en normalized data.

### Onderzoeksstappen

```bash
# 1. Vergelijk schema's
sudo -u postgres psql -d energy_insights_nl -c "\d raw_entso_e_a75"
sudo -u postgres psql -d energy_insights_nl -c "\d norm_entso_e_a75"

sudo -u postgres psql -d energy_insights_nl -c "\d raw_entso_e_a65"
sudo -u postgres psql -d energy_insights_nl -c "\d norm_entso_e_a65"

sudo -u postgres psql -d energy_insights_nl -c "\d raw_entso_e_a44"
sudo -u postgres psql -d energy_insights_nl -c "\d norm_entso_e_a44"

# 2. Bekijk sample data
sudo -u postgres psql -d energy_insights_nl -c "
SELECT * FROM raw_entso_e_a75 ORDER BY timestamp DESC LIMIT 5;"

sudo -u postgres psql -d energy_insights_nl -c "
SELECT * FROM norm_entso_e_a75 ORDER BY timestamp DESC LIMIT 5;"

# 3. Bekijk normalizer code
cat /opt/energy-insights-nl/app/synctacles_db/normalizers/normalize_entso_e_a75.py
cat /opt/energy-insights-nl/app/synctacles_db/normalizers/normalize_entso_e_a65.py
cat /opt/energy-insights-nl/app/synctacles_db/normalizers/normalize_entso_e_a44.py
```

### Verwachte Transformaties (uit architectuur docs)

| Aspect | Raw Tables | Normalized Tables |
|--------|------------|-------------------|
| **Structuur** | 1 row per PSR type | Pivot: PSR types als kolommen |
| **Quality metadata** | ❌ Geen | ✅ data_source, data_quality, age_seconds |
| **Confidence** | ❌ Geen | ✅ confidence_score (0-100) |
| **Backfill tracking** | ❌ Geen | ✅ needs_backfill boolean |
| **Timestamps** | source + import | + normalized_timestamp |
| **Fallback** | ❌ Geen | ✅ FORWARD_FILL, Energy-Charts |
| **Aggregatie** | ❌ Geen | ✅ total_mw, renewable_percentage |

### A75 Specifiek (Generation)

**Raw:** 9 separate rows per timestamp (1 per PSR type)
```
timestamp | psr_type      | value_mw
----------|---------------|----------
10:15     | solar         | 450
10:15     | wind_onshore  | 1200
10:15     | nuclear       | 3200
...
```

**Normalized:** 1 row per timestamp met alle PSR types als kolommen
```
timestamp | solar_mw | wind_onshore_mw | nuclear_mw | total_mw | data_quality | age_seconds
----------|----------|-----------------|------------|----------|--------------|------------
10:15     | 450      | 1200            | 3200       | 10450    | FRESH        | 120
```

---

## DEEL 3: API ENDPOINT VERIFICATIE

### Controleer dat API norm_* tables gebruikt

```bash
# Zoek alle database queries in API code
grep -rn "raw_entso_e" /opt/energy-insights-nl/app/synctacles_db/api/
grep -rn "norm_entso_e" /opt/energy-insights-nl/app/synctacles_db/api/
grep -rn "norm_generation" /opt/energy-insights-nl/app/synctacles_db/api/
grep -rn "norm_load" /opt/energy-insights-nl/app/synctacles_db/api/
grep -rn "norm_prices" /opt/energy-insights-nl/app/synctacles_db/api/
```

**Verwacht resultaat:** Alleen `norm_*` tables in API routes.

**Als raw_* gevonden wordt:** Documenteer welke endpoints en rapporteer in handoff terug.

---

## DEEL 4: DOCUMENTATIE UPDATES

### Update 1: SKILL_02_ARCHITECTURE.md

**Locatie:** Project knowledge (of `/opt/github/synctacles-api/docs/`)

**Toevoegen aan Layer 3: Normalizers sectie:**

```markdown
### Normalizer Transformaties per Source

#### A75 (Generation)
| Raw | Normalized |
|-----|------------|
| 9 rows per timestamp (1/PSR) | 1 row met PSR kolommen |
| value_mw per type | solar_mw, wind_mw, nuclear_mw, etc. |
| Geen metadata | data_quality, age_seconds, confidence_score |
| Geen aggregatie | total_mw, renewable_percentage |

#### A65 (Load)
| Raw | Normalized |
|-----|------------|
| Separate actual/forecast | Merged met load_difference_mw |
| Geen metadata | data_quality, age_seconds |
| Inclusief forecast | Gefilterd: alleen historical (timestamp <= NOW) |

#### A44 (Prices)
| Raw | Normalized |
|-----|------------|
| Inclusief day-ahead (future) | Gefilterd voor freshness queries |
| Geen metadata | data_quality, age_seconds |
| Geen fallback | Fallback naar Energy-Charts indien stale |

### CRITICAL: run_normalizers.sh moet ALLE normalizers bevatten

```bash
# Alle ENTSO-E sources MOETEN aanwezig zijn:
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75
```
```

### Update 2: SKILL_06_DATA_SOURCES.md

**Toevoegen:**

```markdown
## RAW VS NORMALIZED DATA

### Waarom Normalisatie Cruciaal Is

1. **ENTSO-E Compliance**
   - Raw data = directe API mirror (niet toegestaan voor commercieel gebruik)
   - Normalized data = getransformeerd met toegevoegde waarde

2. **Quality Metadata**
   - data_quality: FRESH | STALE | FALLBACK | UNAVAILABLE
   - age_seconds: Hoe oud is de data
   - confidence_score: 0-100 betrouwbaarheid

3. **Automation Safety**
   - API retourneert `allow_go_action` flag
   - Alleen FRESH data van primaire bron = true
   - Voorkomt automatisering op onbetrouwbare data

4. **Data Structuur**
   - Raw: Ruwe API response (meerdere rows per timestamp)
   - Normalized: Geoptimaliseerd voor queries (1 row per timestamp)

### API Endpoints

**KRITIEK:** API endpoints MOETEN altijd `norm_*` tables gebruiken, NOOIT `raw_*`.

Dit garandeert:
- Quality metadata in responses
- Gefilterde forecast data
- Correcte freshness berekening
- ENTSO-E license compliance
```

### Update 3: SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md

**Toevoegen aan Normalizers Pattern:**

```markdown
### run_normalizers.sh Logging

**Vereist format:**
```bash
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A44 (prices)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A65 (load)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A75 (generation)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75
```

**Waarom:** Zonder per-source logging is het onmogelijk te zien welke normalizers daadwerkelijk draaien. Dit veroorzaakte de A75 silent failure bug.
```

### Update 4: Nieuwe Validatie Script

**Maak bestand:** `/opt/github/synctacles-api/scripts/validate_pipeline.sh`

```bash
#!/bin/bash
# Validate that all pipeline components are configured correctly

set -e

echo "=== Pipeline Validation ==="

# Check collectors
echo -n "Collectors in run_collectors.sh: "
grep -c "collect_entso_e" scripts/run_collectors.sh || echo "0"

# Check importers  
echo -n "Importers in run_importers.sh: "
grep -c "import_entso_e" scripts/run_importers.sh || echo "0"

# Check normalizers
echo -n "Normalizers in run_normalizers.sh: "
grep -c "normalize_entso_e" scripts/run_normalizers.sh || echo "0"

# Validate counts match
COLLECTORS=$(grep -c "collect_entso_e" scripts/run_collectors.sh 2>/dev/null || echo 0)
IMPORTERS=$(grep -c "import_entso_e" scripts/run_importers.sh 2>/dev/null || echo 0)
NORMALIZERS=$(grep -c "normalize_entso_e" scripts/run_normalizers.sh 2>/dev/null || echo 0)

if [ "$COLLECTORS" -ne "$IMPORTERS" ] || [ "$IMPORTERS" -ne "$NORMALIZERS" ]; then
    echo ""
    echo "⚠️  WARNING: Pipeline component mismatch!"
    echo "   Collectors: $COLLECTORS"
    echo "   Importers:  $IMPORTERS"
    echo "   Normalizers: $NORMALIZERS"
    echo ""
    echo "   Each collector should have matching importer AND normalizer."
    exit 1
else
    echo ""
    echo "✅ Pipeline validated: $COLLECTORS sources configured correctly"
fi
```

---

## DEEL 5: MONITORING VERBETERING

### Gap Metric Toevoegen

**Locatie:** `/opt/energy-insights-nl/app/synctacles_db/api/routes/pipeline.py`

**Toevoegen aan /v1/pipeline/metrics:**

```python
# Raw vs Normalized gap metric
for source in ['a44', 'a65', 'a75']:
    raw_table = f"raw_entso_e_{source}"
    norm_table = f"norm_entso_e_{source}"
    
    gap_result = session.execute(text(f"""
        SELECT 
            EXTRACT(EPOCH FROM (
                (SELECT MAX(timestamp) FROM {raw_table} WHERE timestamp <= NOW()) -
                (SELECT MAX(timestamp) FROM {norm_table} WHERE timestamp <= NOW())
            ))/60 as gap_minutes
    """)).fetchone()
    
    gap = gap_result[0] if gap_result and gap_result[0] else 0
    
    # Expose as Prometheus metric
    output.append(f'pipeline_raw_norm_gap_minutes{{source="{source}"}} {gap}')
```

### Alert Rule Toevoegen

**Locatie:** Prometheus alerts op monitor.synctacles.com

```yaml
- alert: NormalizerLagging
  expr: pipeline_raw_norm_gap_minutes > 30
  for: 15m
  labels:
    severity: critical
  annotations:
    summary: "Normalizer {{ $labels.source }} lagging >30 min behind raw data"
    description: "Gap between raw and normalized data is {{ $value }} minutes. Check run_normalizers.sh."
```

---

## DELIVERABLES CHECKLIST

### Kritieke Fix
- [ ] A65 normalizer toegevoegd aan run_normalizers.sh
- [ ] A75 normalizer toegevoegd aan run_normalizers.sh
- [ ] Per-source logging toegevoegd
- [ ] Handmatig gedraaid om backlog te verwerken
- [ ] Geverifieerd dat alle sources FRESH zijn
- [ ] Git commit + push

### Analyse
- [ ] Schema's raw vs normalized gedocumenteerd
- [ ] Transformaties per source beschreven
- [ ] API endpoints gecontroleerd (alleen norm_* tables)
- [ ] Bevindingen gerapporteerd in return handoff

### Documentatie
- [ ] SKILL_02 update (normalizer transformaties)
- [ ] SKILL_06 update (raw vs normalized)
- [ ] SKILL_13 update (logging requirements)
- [ ] validate_pipeline.sh script aangemaakt

### Monitoring
- [ ] Gap metric toegevoegd aan /v1/pipeline/metrics
- [ ] Alert rule geconfigureerd
- [ ] Grafana dashboard geüpdatet (indien nodig)

---

## RETURN HANDOFF VERWACHT

Rapporteer:
1. Schema vergelijking output (raw vs normalized kolommen)
2. API endpoint audit resultaat (welke tables gebruikt)
3. Eventuele afwijkingen van verwachte architectuur
4. Bevestiging dat alle fixes deployed zijn

---

*Template versie: 1.0*
