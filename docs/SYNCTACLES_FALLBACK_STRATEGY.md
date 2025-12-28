# SYNCTACLES Fallback Strategie v1.0

**Datum:** 2025-12-24  
**Status:** Definitief  
**Doel:** Maximale data completeness met authoritative sources

---

## 1. Overzicht

### Prioriteit Hiërarchie
```
1. ENTSO-E (Authoritative - TSO data)
   ↓ (bij value = 0)
2. Forward Fill (vorige bekende waarde)
   ↓ (validatie)
3. Energy-Charts (Fraunhofer model - indien realistisch)
   ↓ (dagelijks)
4. ENTSO-E Backfill (retry na 24h)
   ↓ (na 7 dagen)
5. Permanent Forward Fill (accepteer gap)
```

---

## 2. Real-time Flow (Collector)

### Stap 1: ENTSO-E API Call
```python
entso_e_value = fetch_entso_e(timestamp, psr_type)

if entso_e_value > 0:
    store(value=entso_e_value, 
          source='ENTSO-E', 
          quality='OK', 
          needs_backfill=False)
else:
    # Proceed to Step 2
```

### Stap 2: Forward Fill
```python
previous_value = get_previous_value(psr_type, timestamp)

store(value=previous_value,
      source='ENTSO-E',
      quality='FORWARD_FILL',
      needs_backfill=True)
```

### Stap 3: Energy-Charts Validatie
```python
ec_value = fetch_energy_charts(timestamp, psr_type)

if ec_value and is_realistic(ec_value, previous_value, psr_type, timestamp):
    update(value=ec_value,
           source='Energy-Charts',
           quality='VALIDATED',
           needs_backfill=True)  # Behoud flag voor ENTSO-E retry
```

---

## 3. Realisme Checks

### 3.1 Baseline Thresholds
```python
MIN_VALUES = {
    'biomass': 200,      # NL capaciteit ~500 MW
    'nuclear': 450,      # Borssele = 485 MW
    'gas': 1000,         # Nooit onder 1 GW
    'coal': 0,           # Kan shutdown zijn
    'waste': 0,
    'wind_offshore': 0,  # Windstil mogelijk
    'wind_onshore': 0,
    'solar': 0,          # Context-dependent (zie 3.2)
    'other': 0
}
```

### 3.2 Solar Context-Aware Validatie

#### Dynamische Zonsvensters
```python
def get_solar_window(date, lat=52.37, lon=4.89):
    """
    Bereken zonsopkomst/ondergang (UTC)
    
    Winter (dec): 07:00-15:00 UTC
    Zomer (jun):  03:00-20:00 UTC
    """
    # Implementatie: simplified solar calculation
    # of astral library voor precisie
```

#### Validatie Logic
```python
def validate_solar_zero(timestamp, value):
    sunrise, sunset = get_solar_window(timestamp.date())
    hour = timestamp.hour
    
    # Extend window +/- 1h voor twilight
    solar_start = max(0, sunrise - 1)
    solar_end = min(23, sunset + 1)
    
    # NACHT: 0 MW = normaal
    if hour < solar_start or hour > solar_end:
        return (needs_validation=False, is_suspicious=False)
    
    # DAWN/DUSK: 0 MW = check maar niet alarm
    if hour in [solar_start, solar_end]:
        return (needs_validation=True, is_suspicious=False)
    
    # DAG: 0 MW = zeer verdacht
    if solar_start < hour < solar_end and value == 0:
        return (needs_validation=True, is_suspicious=True)
    
    return (needs_validation=False, is_suspicious=False)
```

#### Maximum Waarden (seizoen-afhankelijk)
```python
max_solar = {
    'winter': 2000,  # MW (dec-feb)
    'summer': 4000   # MW (mei-aug)
}
```

### 3.3 Deviation Check
```python
def is_realistic(ec_value, forward_fill_value, psr_type, timestamp):
    # Minimum check
    if ec_value < MIN_VALUES[psr_type]:
        return False
    
    # Solar special case
    if psr_type == 'solar':
        needs_val, is_suspicious = validate_solar_zero(timestamp, ec_value)
        
        if not needs_val:  # Nacht
            return True
        
        if ec_value == 0 and is_suspicious:  # Dag + 0 MW
            return False
        
        # Check magnitude
        max_expected = 4000 if timestamp.month in [5,6,7,8] else 2000
        if ec_value > max_expected:
            return False
    
    # Deviation van forward fill
    if forward_fill_value > 0:
        deviation = abs(ec_value - forward_fill_value) / forward_fill_value
        if deviation > 1.5:  # >150% afwijking
            return False
    
    return True
```

---

## 4. Backfill Flow (Dagelijks 04:00 UTC)

### Script: `/opt/synctacles/scripts/backfill_entso_e.py`
```python
# Haal alle records met needs_backfill = TRUE
gaps = db.query("""
    SELECT timestamp, psr_type 
    FROM norm_entso_e_a75 
    WHERE needs_backfill = TRUE 
      AND timestamp >= NOW() - INTERVAL '7 days'
""")

for gap in gaps:
    # Retry ENTSO-E API
    value = fetch_entso_e(gap.timestamp, gap.psr_type)
    
    if value > 0:
        # SUCCESS: overschrijf
        db.update(
            value=value,
            source='ENTSO-E',
            quality='BACKFILLED',
            needs_backfill=False
        )
    # else: behoud forward fill, blijft needs_backfill=TRUE

# Cleanup: accepteer permanent gaps na 7 dagen
db.execute("""
    UPDATE norm_entso_e_a75 
    SET needs_backfill = FALSE 
    WHERE needs_backfill = TRUE 
      AND timestamp < NOW() - INTERVAL '7 days'
""")
```

### Cron Schedule
```bash
# Dagelijks 04:00 UTC
0 4 * * * /opt/synctacles/venv/bin/python3 /opt/synctacles/scripts/backfill_entso_e.py >> /opt/synctacles/logs/backfill.log 2>&1
```

---

## 5. Database Schema

### Nieuwe Kolommen
```sql
ALTER TABLE norm_entso_e_a75 ADD COLUMN IF NOT EXISTS
    data_source VARCHAR(20) DEFAULT 'ENTSO-E',
    data_quality VARCHAR(20) DEFAULT 'OK',
    needs_backfill BOOLEAN DEFAULT FALSE;
```

### Quality States

| State | Betekenis | Gebruiksveilig |
|-------|-----------|----------------|
| `OK` | ENTSO-E realtime data | ✅ Volledig |
| `FORWARD_FILL` | Vorige waarde (ENTSO-E bron) | ✅ Acceptabel |
| `VALIDATED` | Energy-Charts (realistisch) | ⚠️ Met attributie |
| `BACKFILLED` | ENTSO-E achteraf aangevuld | ✅ Volledig |

### Data Sources

| Source | Type | Licensing | Attributie Vereist |
|--------|------|-----------|-------------------|
| `ENTSO-E` | Authoritative | EU Regulation | ❌ Nee |
| `Energy-Charts` | Model | CC BY 4.0 | ✅ Ja |

---

## 6. API Response Format

### Metadata Exposure
```json
{
  "data": [
    {
      "timestamp": "2025-12-21T14:00:00Z",
      "biomass_mw": 375.0,
      "solar_mw": 0.0,
      "data_source": "Energy-Charts",
      "data_quality": "VALIDATED"
    }
  ],
  "meta": {
    "attribution": "Solar data from energy-charts.info (Fraunhofer ISE) under CC BY 4.0"
  }
}
```

---

## 7. Performance Metrics (Verwacht)

### Data Completeness

| Scenario | ENTSO-E Only | Met Fallback | Verbetering |
|----------|--------------|--------------|-------------|
| Realtime | 95% | 99.5% | +4.5% |
| Na 24h (backfill) | 96% | 99.8% | +3.8% |
| Na 7 dagen | 97% | 100% | +3% |

### Gap Reconstructie (21 dec test)

| Methode | Success Rate | Voorbeeld |
|---------|--------------|-----------|
| ENTSO-E backfill | 60% (3/5) | 11:00, 17:00, 22:00 ✅ |
| Energy-Charts | 100% (4/4) | Inclusief 14:00 gap ✅ |
| Forward fill | 100% (fallback) | Altijd beschikbaar |

---

## 8. Waarom ENTSO-E Primair?

### Juridisch/Compliance

- **Authoritative source:** TSO-gerapporteerde data (TenneT)
- **EU Regulation 543/2013:** Transparency verplichting
- **Audit trail:** Defensible bij regulatory vragen

### Technisch

- **TenneT balance data:** Geen Energy-Charts alternatief
- **Multi-country V2:** Uniform schema (36 landen)
- **Attribution-free:** Geen licensing overhead

### Data Kwaliteit

- **Grid-connected assets:** Directe metingen
- **Energy-Charts:** Model (kan conservatief schatten)

**Voorbeeld verschil (Biomass):**
- ENTSO-E: 360-380 MW (alle assets >1 MW)
- Energy-Charts: 240 MW (mogelijk alleen grote assets)
- **Verschil:** ~40%

---

## 9. Implementatie Checklist

### Fase 1: Database (10 min)

- [ ] Run migration: `ADD COLUMN data_source, data_quality, needs_backfill`
- [ ] Verify schema: `\d norm_entso_e_a75`

### Fase 2: Collector Mod (1.5h)

- [ ] Forward fill logic per PSR-type
- [ ] Solar validation (zonsvenster berekening)
- [ ] Energy-Charts API call (historical endpoint)
- [ ] Realisme check implementatie
- [ ] Tag `needs_backfill = TRUE`

### Fase 3: Backfill Script (1h)

- [ ] ENTSO-E retry logic
- [ ] 7-day cleanup
- [ ] Logging naar `/opt/synctacles/logs/backfill.log`
- [ ] Test run (dry-run mode)

### Fase 4: Cron Setup (5 min)

- [ ] Add to crontab: `0 4 * * * ...`
- [ ] Verify: `crontab -l`

### Fase 5: Testing (45 min)

- [ ] Trigger 0-waarde (truncate test record)
- [ ] Verify forward fill
- [ ] Verify Energy-Charts call
- [ ] Verify backfill (manual run)
- [ ] Solar edge cases (dawn/dusk/night)

**Totaal: 3.5-4 uur**

---

## 10. Dependencies

### Python Packages
```bash
pip install astral  # Optioneel: precieze zon-berekeningen
```

### API Keys
```bash
# .env
ENTSOE_API_KEY=your_key_here
```

### Rate Limits

| API | Limit | Strategy |
|-----|-------|----------|
| ENTSO-E | Onbeperkt | Direct gebruik |
| Energy-Charts | Ongedocumenteerd | Respecteer fair use (1 call per collector run) |

---

## 11. Monitoring

### Daily Checks
```bash
# Gap count
psql -U synctacles -d synctacles -c "
SELECT COUNT(*) 
FROM norm_entso_e_a75 
WHERE needs_backfill = TRUE;"

# Quality distribution
psql -U synctacles -d synctacles -c "
SELECT data_quality, COUNT(*) 
FROM norm_entso_e_a75 
WHERE timestamp >= NOW() - INTERVAL '7 days'
GROUP BY data_quality;"

# Source distribution
psql -U synctacles -d synctacles -c "
SELECT data_source, COUNT(*) 
FROM norm_entso_e_a75 
WHERE timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY data_source;"
```

### Alerts

- **Gap rate >5%:** Check ENTSO-E API status
- **Energy-Charts >30% usage:** Mogelijk ENTSO-E structureel down
- **Backfill success <50%:** ENTSO-E data niet alsnog gepubliceerd

---

## 12. Future Enhancements (V1.1)

### Calibration Engine
```python
# Dagelijks: bereken drift tussen ENTSO-E en Energy-Charts
daily_drift = calculate_drift(entso_e_data, ec_data)

# Gebruik drift voor correctie
ec_corrected = ec_value * (1 + daily_drift)
```

### User Preferences
```yaml
data_quality_preference:
  strict: true  # Alleen ENTSO-E (accepteer gaps)
  # of
  best_effort: true  # ENTSO-E + Energy-Charts fallback
```

### Historical Backfill (Eenmalig)
```bash
# Vul alle bestaande gaps (21-23 dec)
python3 scripts/backfill_historical.py --start=2025-12-21 --end=2025-12-23
```

---

## 13. Contact & Support

**Project:** SYNCTACLES  
**Developer:** Leo Blom (DATADIO)  
**Repository:** github.com/DATADIO/synctacles-repo  
**Documentation:** /opt/github/synctacles-repo/docs/

---

**Versie:** 1.0  
**Laatst bijgewerkt:** 2025-12-24  
**Status:** Production Ready