# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** MEDIUM
**Type:** Documentation Updates

---

## CONTEXT

Na Grafana dashboard implementatie zijn meerdere learnings gedocumenteerd die vastgelegd moeten worden in SKILLs om herhaling te voorkomen.

---

## UPDATE 1: SKILL_08_HARDWARE_PROFILE.md

**Locatie:** `/mnt/project/SKILL_08_HARDWARE_PROFILE.md`

**Sectie:** INFRASTRUCTURE ACCESS (CC/AI) - **TOEVOEGEN** aan bestaande sectie:

```markdown
### DNS Limitatie Monitor Server

**KRITIEK:** DNS resolution werkt NIET op monitor.synctacles.com voor externe domains.

```bash
# Dit faalt:
curl https://api.synctacles.com/v1/pipeline/health
# Error: Could not resolve host: api.synctacles.com
```

**Impact:**
- Geen JSON/Infinity datasources mogelijk in Grafana
- Geen directe API calls naar externe hosts

**Workaround:** Prometheus met IP targets + SNI configuratie.

**TODO:** Onderzoek `/etc/resolv.conf` en firewall configuratie.
```

---

## UPDATE 2: SKILL_06_DATA_SOURCES.md

**Locatie:** `/mnt/project/SKILL_06_DATA_SOURCES.md`

**Toevoegen:** Nieuwe sectie over ENTSO-E data delays:

```markdown
## ENTSO-E DATA DELAYS

### A75 (Generation by Source) - VERWACHTE DELAYS

**Normaal gedrag:**
- Delay: 2-4 uur na timestamp
- STALE status (90-180 min) is NORMAAL
- UNAVAILABLE (>180 min) is NIET normaal → onderzoek nodig

**Update patronen (uit DB analyse):**
| Tijd (UTC) | Type | Records |
|------------|------|---------|
| ~13:00 | Grote batch | ~104 timestamps (24h) |
| ~03:00 | Kleinere update | Backfill |

**Monitoring:**
- Alert alleen op UNAVAILABLE, niet op STALE
- Check raw vs normalized gap (>30 min = probleem)

### A44 (Day-Ahead Prices) - FORECAST DATA

**Karakteristiek:** Bevat TOEKOMSTIGE timestamps (day-ahead).

**Database queries:** ALTIJD filteren met `WHERE timestamp <= NOW()`

```sql
-- FOUT (geeft negatieve age):
SELECT MAX(timestamp) FROM norm_entso_e_a44;

-- CORRECT:
SELECT MAX(timestamp) FROM norm_entso_e_a44 WHERE timestamp <= NOW();
```

### A65 (System Load) - FORECAST DATA

**Karakteristiek:** Bevat 24u forecast data.

**Zelfde filtering vereist als A44.**
```

---

## UPDATE 3: SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md

**Locatie:** `/mnt/project/SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md`

**Toevoegen:** Sectie over normalizer logging requirements:

```markdown
## NORMALIZER LOGGING REQUIREMENTS

### Huidige Situatie (ONVOLDOENDE)

```
[2026-01-08 17:38:50] Starting normalizer batch...
[2026-01-08 17:38:52] Normalizer batch complete
Consumed 1.668s CPU time
```

**Probleem:** Geen indicatie welke sources verwerkt werden. Silent failures mogelijk.

### Vereiste Logging

```
[2026-01-08 17:38:50] Starting normalizer batch...
[2026-01-08 17:38:50] Processing A75 (generation)... 104 records
[2026-01-08 17:38:51] Processing A65 (load)... 96 records  
[2026-01-08 17:38:51] Processing A44 (prices)... 24 records
[2026-01-08 17:38:52] Normalizer batch complete (224 total, 0 errors)
Consumed 1.668s CPU time (A75: 0.8s, A65: 0.5s, A44: 0.3s)
```

### Vereiste Log Levels

| Event | Level | Voorbeeld |
|-------|-------|-----------|
| Start/complete | INFO | "Starting normalizer batch..." |
| Per source | INFO | "Processing A75... 104 records" |
| Skip (geen data) | DEBUG | "A75: skipped - no new raw data" |
| Error | ERROR | "A75: failed - database timeout" |
| Timing | DEBUG | "A75: 0.8s" |

### Alert Triggers

- ERROR level → immediate alert
- Source skipped 3x consecutief → warning alert
- Raw/Normalized gap >30 min → warning alert
```

---

## UPDATE 4: Nieuw bestand MONITORING.md of update SKILL_XX

**Locatie:** `/mnt/project/docs/MONITORING.md` of nieuwe SKILL

**Toevoegen:** Grafana best practices:

```markdown
## GRAFANA DASHBOARD BEST PRACTICES

### Datasource Keuze

| Optie | Status | Reden |
|-------|--------|-------|
| Prometheus | ✅ GEBRUIK | Native, historisch, alerting |
| Infinity (JSON) | ❌ VERMIJD | DNS issues op monitor server |
| PostgreSQL Direct | ⚠️ ALLEEN INDIEN NODIG | Extra security surface |

### Dashboard Design Regels

1. **Complete pipeline visibility**
   - Toon ALLE componenten (timers + data status + trends)
   - Niet alleen data, ook service status

2. **Standaard layout**
   - Row 1: Service/Timer status (Collector, Importer, Normalizer, Health)
   - Row 2: Data status per source (A44, A65, A75)
   - Row 3: Timeseries trend (optioneel)

3. **Formatting**
   - `textMode: "value"` voor stat panels (niet "value_and_name")
   - Professionele titels: "Day-Ahead Prices (A44)" niet "A44"
   - Descriptions op alle panels

4. **Color coding (standaard)**
   - Green: FRESH (<90 min) / ACTIVE / OK
   - Yellow: STALE (90-180 min)
   - Red: UNAVAILABLE (>180 min) / STOPPED / ERROR
   - Gray: NO_DATA

### Prometheus Metrics Endpoint

Elke API moet `/metrics` of `/v1/pipeline/metrics` exposen:

```python
# Vereiste metrics
pipeline_timer_status{timer="collector|importer|normalizer|health"}  # 1=active, 0=stopped
pipeline_data_status{source="a44|a65|a75"}  # 1=fresh, 2=stale, 3=unavailable
pipeline_data_freshness_minutes{source="a44|a65|a75"}  # float
```
```

---

## UPDATE 5: run_normalizers.sh verificatie

**Locatie:** `/opt/github/synctacles-api/scripts/run_normalizers.sh`

**Actie:** Verifieer dat A75 normalizer NIET ontbreekt (zoals eerder A44 importer ontbrak).

```bash
# Check huidige inhoud
cat /opt/github/synctacles-api/scripts/run_normalizers.sh

# Moet bevatten:
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a75
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a65
"${PYTHON}" -m synctacles_db.normalizers.normalize_entso_e_a44
```

**Als A75 ontbreekt:** Toevoegen + commit + deploy.

---

## DELIVERABLES

1. [ ] SKILL_08: DNS limitatie monitor server toegevoegd
2. [ ] SKILL_06: ENTSO-E data delays sectie toegevoegd
3. [ ] SKILL_13: Normalizer logging requirements toegevoegd
4. [ ] MONITORING.md: Grafana best practices toegevoegd
5. [ ] run_normalizers.sh: A75 aanwezigheid geverifieerd
6. [ ] Alle wijzigingen committed en gepushed

---

## PRIORITEIT

| Item | Prioriteit | Reden |
|------|------------|-------|
| run_normalizers.sh check | HIGH | Potentiële root cause A75 failure |
| SKILL_08 DNS | MEDIUM | Voorkomt herhaling Infinity debacle |
| SKILL_06 delays | MEDIUM | Voorkomt false alarms |
| SKILL_13 logging | MEDIUM | Betere debugging |
| MONITORING.md | LOW | Best practices vastleggen |

---

*Template versie: 1.0*
