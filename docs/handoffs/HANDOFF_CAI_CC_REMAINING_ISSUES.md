# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** CRITICAL
**Type:** Remaining Tasks After Documentation

---

## STATUS OVERZICHT

| Categorie | Status |
|-----------|--------|
| Documentatie | ✅ COMPLEET |
| Kritieke Fix | ❌ NOG OPEN |
| Analyse | ❌ NOG OPEN |
| Monitoring | ❌ NOG OPEN |

---

## OPENSTAANDE ISSUES

### 1. CRITICAL: run_normalizers.sh Fix

**Status:** ❌ Alleen gedocumenteerd als "known issue", NOG NIET GEFIXED

**Actie:**
```bash
# Edit script
sudo nano /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# Voeg toe NA regel met normalize_entso_e_a44:
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A65 (load)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Processing A75 (generation)..."
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75

# Run handmatig
sudo -u energy-insights-nl bash /opt/energy-insights-nl/app/scripts/run_normalizers.sh

# Verificatie
sudo -u postgres psql -d energy_insights_nl -c "
SELECT 'A44' as src, MAX(timestamp), EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age FROM norm_entso_e_a44 WHERE timestamp <= NOW()
UNION ALL
SELECT 'A65', MAX(timestamp), EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 FROM norm_entso_e_a65 WHERE timestamp <= NOW()
UNION ALL
SELECT 'A75', MAX(timestamp), EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 FROM norm_entso_e_a75 WHERE timestamp <= NOW();"

# Git sync
sudo cp /opt/energy-insights-nl/app/scripts/run_normalizers.sh /opt/github/synctacles-api/scripts/
cd /opt/github/synctacles-api
sudo -u energy-insights-nl git add scripts/run_normalizers.sh
sudo -u energy-insights-nl git commit -m "fix(critical): add missing A65/A75 normalizers"
sudo -u energy-insights-nl git push origin main
```

---

### 2. HIGH: Raw vs Normalized Schema Analyse

**Status:** ❌ Niet uitgevoerd

**Actie:**
```bash
# Schema vergelijking
sudo -u postgres psql -d energy_insights_nl -c "\d raw_entso_e_a75"
sudo -u postgres psql -d energy_insights_nl -c "\d norm_entso_e_a75"

# Rapporteer kolommen en verschillen in return handoff
```

---

### 3. HIGH: API Endpoint Verificatie

**Status:** ❌ Niet uitgevoerd

**Actie:**
```bash
# Check welke tables API gebruikt
grep -rn "raw_entso_e\|norm_entso_e\|norm_generation\|norm_load" \
  /opt/energy-insights-nl/app/synctacles_db/api/routes/

# Verwacht: ALLEEN norm_* tables
# Rapporteer afwijkingen in return handoff
```

---

### 4. MEDIUM: Gap Monitoring Metric

**Status:** ❌ Niet geïmplementeerd

**Actie:** Voeg toe aan `/opt/energy-insights-nl/app/synctacles_db/api/routes/pipeline.py`:

```python
# In get_pipeline_metrics() of /v1/pipeline/metrics endpoint
# Voeg raw vs normalized gap metric toe per source
```

---

### 5. MEDIUM: Validation Script

**Status:** ❌ Niet aangemaakt

**Actie:** Maak `/opt/github/synctacles-api/scripts/validate_pipeline.sh`:

```bash
#!/bin/bash
# Validate pipeline component counts match
COLLECTORS=$(grep -c "collect_entso_e" scripts/run_collectors.sh 2>/dev/null || echo 0)
IMPORTERS=$(grep -c "import_entso_e" scripts/run_importers.sh 2>/dev/null || echo 0)
NORMALIZERS=$(grep -c "normalize_entso_e" scripts/run_normalizers.sh 2>/dev/null || echo 0)

echo "Collectors: $COLLECTORS | Importers: $IMPORTERS | Normalizers: $NORMALIZERS"

if [ "$COLLECTORS" -ne "$NORMALIZERS" ]; then
    echo "⚠️ MISMATCH - check pipeline configuration"
    exit 1
fi
echo "✅ Pipeline validated"
```

---

## PRIORITEIT VOLGORDE

1. **run_normalizers.sh fix** - CRITICAL (data wordt stale)
2. **API endpoint verificatie** - HIGH (compliance check)
3. **Schema analyse** - HIGH (documentatie)
4. **Gap metric** - MEDIUM (monitoring)
5. **Validation script** - MEDIUM (preventie)

---

## DELIVERABLES

- [ ] run_normalizers.sh gefixed + deployed + committed
- [ ] Alle sources FRESH in pipeline health
- [ ] API endpoint audit resultaat
- [ ] Schema vergelijking output
- [ ] Return handoff met bevindingen

---

*Template versie: 1.0*
