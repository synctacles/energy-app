# SYNCTACLES Fallback System

**Version:** V1.0  
**Date:** 26 December 2025

## Overview

Component-based fallback system that automatically switches to Energy-Charts API when ENTSO-E data is unavailable or stale.

---

## Supported Endpoints

### ✅ WITH FALLBACK

| Endpoint | Primary Source | Fallback Source | Component |
|----------|---------------|-----------------|-----------|
| `/api/v1/generation-mix` | ENTSO-E A75 | Energy-Charts | generation_mix |
| `/api/v1/load` | ENTSO-E A65 | Energy-Charts | load |
| `/api/v1/signals` | ENTSO-E A75 | Energy-Charts | generation_mix |

### ❌ NO FALLBACK

| Endpoint | Source | Reason |
|----------|--------|--------|
| `/api/v1/prices` | ENTSO-E A44 | No price data in Energy-Charts |
| `/api/v1/balance` | TenneT | NL-specific data, no alternative |

---

## Fallback Logic (4 Tiers)

### Tier 1: Fresh Database Data (Primary)
```
Age < 30 min → Quality: FRESH
Use immediately, optimal data quality
```

### Tier 2: Stale Database Data (Acceptable)
```
Age 30-150 min → Quality: STALE
Still usable (ENTSO-E A75 has structural delay)
```

### Tier 3: Energy-Charts Fallback
```
Age > 150 min OR missing → Quality: FALLBACK
Fetch from Energy-Charts API
Cache result (5 min TTL)
```

### Tier 4: Cache (Last Resort)
```
All sources fail → Quality: CACHED
Use cached fallback data (<5 min old)
```

### Tier 5: Complete Failure
```
No data available → Quality: UNAVAILABLE
Return null, safe defaults for signals
```

---

## Thresholds by Component

```python
{
    "generation_mix": {
        "fresh": 30,   # < 30 min
        "stale": 150,  # 30-150 min (accept ENTSO-E delay)
    },
    "load": {
        "fresh": 15,   # < 15 min
        "stale": 60,   # 15-60 min
    },
}
```

**Rationale:**
- ENTSO-E A75 (generation): 60-90 min structural delay
- ENTSO-E A65 (load): 5-30 min typical delay
- Energy-Charts: Often 150+ min old (not always better)

---

## Response Metadata

### Example: ENTSO-E Primary (Normal)
```json
{
  "metadata": {
    "source": "ENTSO-E",
    "quality": "STALE",
    "age_minutes": 71,
    "renewable_percentage": 91.2
  }
}
```

### Example: Energy-Charts Fallback
```json
{
  "metadata": {
    "source": "Energy-Charts",
    "quality": "FALLBACK",
    "age_minutes": 167,
    "renewable_percentage": 24.8
  }
}
```

### Example: Unavailable
```json
{
  "metadata": {
    "source": "None",
    "quality": "UNAVAILABLE"
  }
}
```

---

## Quality States

| State | Meaning | Action |
|-------|---------|--------|
| `FRESH` | Data < fresh threshold | Use immediately |
| `STALE` | Data < stale threshold | Use but not optimal |
| `FALLBACK` | Using backup source | Energy-Charts active |
| `CACHED` | Using expired cache | Last resort |
| `UNAVAILABLE` | All sources failed | Return null |

---

## Infrastructure

### Collection Schedule
- **Collector:** Every 15 minutes (systemd timer)
- **Normalizer:** Every 15 minutes (processes raw data)
- **Fallback cache:** 5 minute TTL

### Expected Behavior

**Normal operation (95% of time):**
- ENTSO-E data 60-90 min old
- Quality: STALE
- Fallback NOT triggered (within 150 min threshold)

**Fallback activation (5% of time):**
- ENTSO-E data >150 min old OR missing
- Quality: FALLBACK
- Energy-Charts provides backup data

**Complete failure (<1% of time):**
- Both ENTSO-E and Energy-Charts unavailable
- Quality: UNAVAILABLE
- Signals use safe defaults (is_green = false, etc.)

---

## API Examples

### Check Fallback Status
```bash
# Generation mix
curl -H "X-API-Key: YOUR_KEY" \
  https://api.synctacles.io/api/v1/generation-mix | jq '.metadata'

# Signals renewable component
curl -H "X-API-Key: YOUR_KEY" \
  https://api.synctacles.io/api/v1/signals | jq '.metadata.data_quality'
```

### Expected Uptime

**Component availability:**
- ENTSO-E A75: ~95% (within 150 min threshold)
- Energy-Charts: ~98% (backup)
- **Combined: ~99.9%** (both down simultaneously rare)

---

## Monitoring

### Health Check
Monitor `metadata.quality` field:
- `FRESH` or `STALE` → Normal ✅
- `FALLBACK` → Backup active ⚠️
- `UNAVAILABLE` → Service degraded ❌

### Alerts (Recommended)
- `quality = UNAVAILABLE` for >15 min → Critical
- `quality = FALLBACK` for >6 hours → Warning
- `age_minutes > 180` → Warning

---

## Troubleshooting

### Fallback Always Active
**Symptom:** `source = Energy-Charts` always  
**Cause:** ENTSO-E collector failing  
**Check:** `sudo journalctl -u synctacles-collector`

### Old Data (>180 min)
**Symptom:** `age_minutes > 180`  
**Cause:** Both sources stale  
**Check:** ENTSO-E API status, Energy-Charts availability

### Unavailable Data
**Symptom:** `quality = UNAVAILABLE`  
**Cause:** All sources failed  
**Check:** Network connectivity, API credentials

---

## Version History

**V1.0 (26 Dec 2025):**
- Component-based fallback manager
- Energy-Charts integration
- generation-mix, load, signals endpoints
- Optimized thresholds (150 min for A75)

---

## Next Steps (V1.1)

**Planned improvements:**
- Suspicious data detection (solar=0 at noon)
- Per-country threshold tuning
- Fallback for prices endpoint (alternative source)
- Advanced cache strategies

---

*For technical implementation details, see:*
- `/synctacles_db/fallback/fallback_manager.py`
- `/synctacles_db/fallback/energy_charts_client.py`
