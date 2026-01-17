# HANDOFF: A75 UNAVAILABLE - Root Cause Analysis

**Server:** SYNCTACLES API (ENIN-NL)  
**IP:** 135.181.255.83  
**SSH:** `ssh leo@135.181.255.83` (of je standaard SSH alias)  
**Prioriteit:** HOOG (33% core metrics down)  
**Geschatte tijd:** 30-45 min

---

## PROBLEEM

Grafana dashboard toont:
- ✅ API Status: OK
- ✅ Day-Ahead Prices (A44): FRESH
- ✅ System Load (A65): FRESH
- ❌ **Generation by Source (A75): UNAVAILABLE**

---

## DIAGNOSE STAPPEN

### 1. Check Service Status

```bash
# Collector service status
sudo systemctl status energy-insights-nl-collector

# Recent logs
sudo journalctl -u energy-insights-nl-collector --since "2 hours ago" | grep -i "A75\|generation\|error"

# Normalizer status
sudo systemctl status energy-insights-nl-normalizer
sudo journalctl -u energy-insights-nl-normalizer --since "2 hours ago" | grep -i "A75\|generation"
```

### 2. Check Database

```bash
cd /opt/synctacles/app
source venv/bin/activate

# Laatste A75 data
psql -d synctacles -c "
SELECT source, MAX(timestamp) as last_data, COUNT(*) as records_24h
FROM generation_by_source
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY source
ORDER BY last_data DESC;
"

# Raw vs Normalized vergelijking
psql -d synctacles -c "
SELECT 
    'raw' as type, MAX(timestamp) as last 
FROM raw_generation_by_source
UNION ALL
SELECT 
    'normalized' as type, MAX(timestamp) as last 
FROM generation_by_source;
"
```

### 3. Test ENTSO-E A75 Direct

```bash
# Load API token
source /opt/synctacles/app/.env

# Test A75 request
curl -s "https://web-api.tp.entsoe.eu/api?securityToken=${ENTSOE_TOKEN}&documentType=A75&processType=A16&In_Domain=10YNL----------L&periodStart=$(date -u +%Y%m%d0000)&periodEnd=$(date -u +%Y%m%d2359)" | head -100

# Check response
# - Empty/error = ENTSO-E down
# - Valid XML = Parser/DB issue
```

### 4. Check Fallback Status

```bash
# Is Energy-Charts fallback geïmplementeerd?
grep -r "energy-charts\|energycharts" /opt/synctacles/app/

# Check collector code voor fallback logic
cat /opt/synctacles/app/collectors/generation_collector.py | grep -A20 "fallback\|except\|error"
```

---

## MOGELIJKE OORZAKEN & FIXES

| Oorzaak | Diagnose | Fix |
|---------|----------|-----|
| ENTSO-E A75 down | curl geeft error/empty | Wachten + fallback activeren |
| Rate limit | 429 in logs | Backoff implementeren |
| Parser error | XML OK, DB empty | Fix parser code |
| NL data niet beschikbaar | Valid XML, geen NL | Check In_Domain parameter |
| Normalizer niet in run script | A75 ontbreekt in script | Toevoegen (was eerder issue) |
| Weekend vertraging | Zaterdag, data komt later | Documenteren als bekend gedrag |

---

## DELIVERABLE

Maak bestand: `/opt/synctacles/docs/RCA_A75_UNAVAILABLE_20260110.md`

Inhoud:
1. **Root Cause:** [extern/intern]
2. **Tijdlijn:** Wanneer begon het falen
3. **Fallback Status:** Geïmplementeerd? Zo ja, waarom niet getriggerd?
4. **Tijdelijke Fix:** Wat is nu gedaan
5. **Permanente Fix:** Wat moet nog
6. **Preventie:** Monitoring/alerting verbeteringen

---

## SUCCESS CRITERIA

- [ ] Root cause geïdentificeerd
- [ ] A75 data weer FRESH of fallback actief
- [ ] RCA document geschreven
- [ ] Preventie stappen gedocumenteerd
