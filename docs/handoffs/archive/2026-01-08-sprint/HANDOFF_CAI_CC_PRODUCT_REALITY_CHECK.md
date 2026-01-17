# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** Audit + GitHub Issues

---

## CONTEXT

CAI had een blindspot (Enever bestond niet in SKILLs maar wel in code). We moeten weten wat de **echte** product status is, niet wat de docs zeggen.

---

## TASK

Complete audit van wat er **IS** (code reality), vergelijk met SKILLs, en maak GitHub issues voor alle gaps.

---

## DEEL 1: CODE AUDIT

### 1.1 synctacles-api repo

```bash
cd /opt/github/synctacles-api

# Endpoints
echo "=== API ENDPOINTS ===" 
grep -r "^@app\.\|^@router\." --include="*.py" | grep -E "(get|post|put|delete)"

# Collectors
echo "=== COLLECTORS ==="
ls -la collectors/ 2>/dev/null || ls -la src/collectors/ 2>/dev/null

# Normalizers  
echo "=== NORMALIZERS ==="
ls -la normalizers/ 2>/dev/null || ls -la src/normalizers/ 2>/dev/null

# Database models/schema
echo "=== DATABASE SCHEMA ==="
grep -r "class.*Model\|CREATE TABLE\|Column(" --include="*.py" | head -50

# Config/constants
echo "=== CONFIG ==="
cat config.py 2>/dev/null || cat src/config.py 2>/dev/null || cat settings.py 2>/dev/null
```

**Documenteer per component:**
- Naam
- Status (actief/disabled/broken)
- Wat doet het
- Dependencies

### 1.2 ha-energy-insights-nl repo

```bash
cd /opt/github/ha-energy-insights-nl/custom_components/ha_energy_insights_nl

# Alle Python files
echo "=== FILES ==="
wc -l *.py

# Sensors
echo "=== SENSORS ==="
grep -E "class.*Sensor" sensor.py

# Config options
echo "=== CONFIG OPTIONS ==="
grep -E "vol\.|CONF_" config_flow.py const.py

# External dependencies
echo "=== DEPENDENCIES ==="
grep -E "^import|^from" *.py | grep -v "homeassistant\|const\|__" | sort -u
```

### 1.3 Productie Server Status

```bash
# Services
echo "=== SYSTEMD SERVICES ==="
systemctl list-units --type=service | grep energy

# Timers
echo "=== SYSTEMD TIMERS ==="
systemctl list-timers | grep energy

# Recent logs (errors only)
echo "=== RECENT ERRORS ==="
journalctl -u "energy-insights-nl*" --since "24 hours ago" -p err --no-pager | tail -30

# Database stats
echo "=== DATABASE STATS ==="
sudo -u postgres psql -d energy_insights -c "
SELECT 
  (SELECT COUNT(*) FROM normalized_data) as normalized_records,
  (SELECT COUNT(*) FROM raw_data) as raw_records,
  (SELECT MAX(timestamp) FROM normalized_data) as latest_data;
"

# Disk usage
echo "=== DISK ==="
df -h /opt /var/log
```

---

## DEEL 2: SKILL VERGELIJKING

Vergelijk audit resultaten met deze SKILLs:
- `/opt/github/synctacles-api/docs/skills/SKILL_02_ARCHITECTURE.md`
- `/opt/github/synctacles-api/docs/skills/SKILL_04_PRODUCT_REQUIREMENTS.md`
- `/opt/github/synctacles-api/docs/skills/SKILL_06_DATA_SOURCES.md`

**Per SKILL, check:**
| Claimed in SKILL | Exists in Code | Works in Prod | Gap? |
|------------------|----------------|---------------|------|
| [feature X] | ✅/❌ | ✅/❌ | [beschrijving] |

---

## DEEL 3: OUTPUT FORMAT

Lever als **PRODUCT_REALITY_CHECK.md**:

```markdown
# PRODUCT REALITY CHECK

**Datum:** 2026-01-08
**Auditor:** CC

---

## SYNCTACLES-API

### Endpoints (Actief)
| Endpoint | Method | Beschrijving | Status |
|----------|--------|--------------|--------|
| /v1/generation-mix | GET | ... | ✅ |

### Endpoints (Gedocumenteerd maar niet gevonden)
| Endpoint | SKILL | Gap |
|----------|-------|-----|

### Collectors
| Naam | Status | Laatste run | Notes |
|------|--------|-------------|-------|

### Normalizers
| Naam | Status | Notes |
|------|--------|-------|

### Database
- Records: X
- Latest data: timestamp
- Schema matches docs: ✅/❌

---

## HA COMPONENT

### Sensors (Actief)
| Sensor | Type | Data Source | Status |
|--------|------|-------------|--------|

### Config Options
| Option | Required | Default | Notes |
|--------|----------|---------|-------|

### BYO-Key Features
| Feature | Implemented | Documented | Gap? |
|---------|-------------|------------|------|

---

## PRODUCTIE SERVER

### Services
| Service | Status | Notes |
|---------|--------|-------|

### Issues Gevonden
| Issue | Severity | Notes |
|-------|----------|-------|

---

## GAPS SUMMARY

### Code exists, NOT in docs
| Item | Location | Action |
|------|----------|--------|

### Docs claim, NOT in code
| Item | SKILL | Action |
|------|-------|--------|

### Broken/Disabled
| Item | Reason | Action |
|------|--------|--------|
```

---

## DEEL 4: GITHUB ISSUES

Voor ELKE gap uit de summary, maak een GitHub issue:

```bash
cd /opt/github/synctacles-api

# Check of gh cli werkt
gh auth status

# Per gap, maak issue:
gh issue create \
  --title "[GAP] Korte beschrijving" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
[Beschrijving wat ontbreekt/niet klopt]

## Source
- SKILL: [welke SKILL]
- Code: [welke file/location]

## Action Required
[ ] Fix code
[ ] Fix docs
[ ] Beide

## Priority
[LOW/MEDIUM/HIGH]" \
  --label "documentation,gap-audit"
```

**Issue labels:** Maak eerst labels aan als ze niet bestaan:
```bash
gh label create "gap-audit" --description "Found during gap audit" --color "FBCA04"
gh label create "docs-code-mismatch" --description "Documentation doesn't match code" --color "D93F0B"
```

---

## DEEL 5: HA REPO ISSUES

Als gaps in HA component:

```bash
cd /opt/github/ha-energy-insights-nl

gh issue create \
  --title "[GAP] ..." \
  --body "..." \
  --label "documentation,gap-audit"
```

---

## VERIFICATION

Na afloop:
```bash
# Toon alle nieuwe issues
gh issue list --repo ldraaisma/synctacles-api --label "gap-audit"
gh issue list --repo ldraaisma/ha-energy-insights-nl --label "gap-audit"
```

---

## DELIVERABLES

1. `PRODUCT_REALITY_CHECK.md` in `/opt/github/synctacles-api/docs/`
2. GitHub issues voor alle gaps (beide repos)
3. Handoff response met summary

---

## OUT OF SCOPE

- Geen fixes (alleen audit + issues)
- Geen code changes
- Geen SKILL updates (dat doet CAI na review)

---

*Template versie: 1.0*
