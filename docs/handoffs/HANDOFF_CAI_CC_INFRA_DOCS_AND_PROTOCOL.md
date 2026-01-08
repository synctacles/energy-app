# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** Documentation + Protocol Update

---

## CONTEXT

1. CC checkte verkeerde server voor Grafana (ENIN-NL ipv monitor.synctacles.com)
2. CC's gh auth check ontbrak, waardoor issues niet aangemaakt konden worden
3. Beide problemen komen voort uit ontbrekende documentatie en protocol gaps

---

## DEEL 1: ARCHITECTUUR DOCUMENTATIE

### Task 1.1: Create docs/INFRASTRUCTURE.md

**Locatie:** `/opt/github/synctacles-api/docs/INFRASTRUCTURE.md`

**Inhoud:**

```markdown
# SYNCTACLES Infrastructure

## Server Overview

| Server | Hostname | IP | Purpose |
|--------|----------|-----|---------|
| API Production | ENIN-NL | 135.181.255.83 | API, Database, Collectors, Normalizers |
| Monitoring | monitor.synctacles.com | [IP] | Grafana, Prometheus |

---

## API Server (ENIN-NL)

**Hostname:** ENIN-NL
**IP:** 135.181.255.83

**Services:**
- energy-insights-nl-api (FastAPI, port 8000)
- PostgreSQL (port 5432)
- energy-insights-nl-collector (systemd timer)
- energy-insights-nl-importer (systemd timer)
- energy-insights-nl-normalizer (systemd timer)

**Endpoints:**
- Internal: http://localhost:8000
- External: https://api.synctacles.com (via reverse proxy)
- Metrics: http://localhost:8000/metrics (Prometheus format)

---

## Monitoring Server

**URL:** https://monitor.synctacles.com/
**Proxy:** Caddy

**Services:**
- Grafana (dashboards, alerting)
- Prometheus (metrics collection)

**Access:** Login required

---

## Network Diagram

```
┌─────────────────┐
│   End Users     │
└────────┬────────┘
         │ HTTPS
    ┌────▼─────┐
    │ Home     │ BYO keys: TenneT, Enever
    │ Assistant│
    └────┬─────┘
         │ HTTPS
    ┌────▼──────────┐
    │  API Server   │ ENIN-NL
    │  :8000        │ ENTSO-E, Energy-Charts
    └────┬──────────┘
         │ Prometheus scrape
    ┌────▼──────────┐
    │  Monitoring   │ monitor.synctacles.com
    │  Grafana      │
    └───────────────┘
```

---

## Important Notes

- **Grafana draait NIET op ENIN-NL** - gebruik monitor.synctacles.com
- **Metrics endpoint** beschikbaar op API server voor Prometheus scraping
- **Database** alleen lokaal toegankelijk (localhost:5432)
```

---

### Task 1.2: Create docs/MONITORING.md

**Locatie:** `/opt/github/synctacles-api/docs/MONITORING.md`

**Inhoud:**

```markdown
# SYNCTACLES Monitoring

## Overview

SYNCTACLES gebruikt een dedicated monitoring server voor observability.

**URL:** https://monitor.synctacles.com/

---

## Stack

| Component | Purpose | Location |
|-----------|---------|----------|
| Grafana | Dashboards, alerting | monitor.synctacles.com |
| Prometheus | Metrics collection | monitor.synctacles.com |
| Caddy | Reverse proxy, HTTPS | monitor.synctacles.com |

---

## Metrics Collection

### API Metrics Endpoint

**URL:** http://ENIN-NL:8000/metrics (internal)
**Format:** Prometheus OpenMetrics

**Beschikbare metrics:**
- Python runtime (GC, memory, CPU)
- FastAPI request metrics
- Custom application metrics

### Prometheus Scraping

Prometheus scraped de API metrics endpoint periodiek.

**Verify scraping werkt:**
1. Login op monitor.synctacles.com
2. Ga naar Explore
3. Query: `up{job="synctacles-api"}`
4. Moet `1` returnen

---

## Grafana Dashboards

### Bestaande Dashboards

- Services Status (zie screenshot in docs)
- [TODO: list andere dashboards]

### Dashboard Toevoegen

1. Login op https://monitor.synctacles.com/
2. Dashboards → New Dashboard
3. Add visualization
4. Select Prometheus data source
5. Write PromQL query
6. Save dashboard

---

## Alerting

[TODO: Document alert rules]

---

## Troubleshooting

**Grafana niet bereikbaar:**
- Check https://monitor.synctacles.com/
- Verify Caddy proxy status

**Metrics niet zichtbaar in Grafana:**
1. Check API metrics endpoint: `curl http://ENIN-NL:8000/metrics`
2. Check Prometheus scrape config
3. Check Prometheus targets in Grafana
```

---

### Task 1.3: Create SKILL_14_MONITORING.md

**Locatie:** `/opt/github/synctacles-api/docs/skills/SKILL_14_MONITORING.md`

**Inhoud:**

```markdown
# SKILL 14 — MONITORING INFRASTRUCTURE

Observability, Dashboards, and Alerting
Version: 1.0 (2026-01-08)

---

## PURPOSE

Document de monitoring infrastructuur voor SYNCTACLES.

---

## KEY INFORMATION

### Monitoring Server

| Aspect | Value |
|--------|-------|
| URL | https://monitor.synctacles.com/ |
| Stack | Grafana + Prometheus |
| Proxy | Caddy |

**⚠️ BELANGRIJK:** Grafana draait NIET op ENIN-NL (API server). Gebruik altijd monitor.synctacles.com.

---

## Grafana

**Access:** https://monitor.synctacles.com/

**Dashboards:**
- Services Status
- [TODO: andere dashboards]

**Data Source:** Prometheus

---

## Prometheus

**Scrape targets:**
- ENIN-NL:8000/metrics (API server)

**Metrics beschikbaar:**
- Python runtime metrics
- FastAPI metrics
- Custom application metrics

---

## API Metrics Endpoint

**URL:** http://localhost:8000/metrics (op ENIN-NL)

**Format:** Prometheus OpenMetrics

**Test:**
```bash
curl -s http://localhost:8000/metrics | head -20
```

---

## Adding New Metrics

1. Add instrumentation in Python code
2. Verify metric in /metrics endpoint
3. Prometheus scrapes automatically
4. Create Grafana panel with PromQL

---

## Related Documents

- docs/INFRASTRUCTURE.md - Server layout
- docs/MONITORING.md - Detailed monitoring docs
- SKILL_02 - API architecture
```

---

## DEEL 2: SKILL_00 PROTOCOL UPDATE

### Task 2.1: Update SKILL_00_AI_OPERATING_PROTOCOL.md

**Locatie:** `/opt/github/synctacles-api/docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md`

**Actie:** Voeg nieuwe sectie toe in "Session Start Protocol" of equivalent:

```markdown
## SESSION START CHECKLIST

Bij elke nieuwe sessie MOET CC deze verificaties uitvoeren:

### 1. GitHub CLI Authentication

```bash
sudo -u energy-insights-nl gh auth status
```

**Verwacht:** "Logged in to github.com account [account]"

**Bij failure:**
- STOP met gh-gerelateerde taken
- Meld aan user dat gh auth nodig is
- Verwijs naar SKILL_11 voor auth procedure

### 2. Server Context Awareness

**Reminder:** SYNCTACLES heeft meerdere servers:
- **ENIN-NL** (135.181.255.83) - API, Database, Collectors
- **monitor.synctacles.com** - Grafana, Prometheus

Check INFRASTRUCTURE.md voor server layout voordat je services zoekt.

### 3. Git Status

```bash
cd /opt/github/synctacles-api
sudo -u energy-insights-nl git status
```

**Verify:** Clean working directory of document uncommitted changes.
```

---

### Task 2.2: Create Session Start Script (Optional)

**Locatie:** `/opt/github/synctacles-api/scripts/cc_session_start.sh`

**Inhoud:**

```bash
#!/bin/bash
# CC Session Start Verification Script
# Run at beginning of each CC session

echo "=== CC SESSION START VERIFICATION ==="
echo ""

# 1. GitHub CLI
echo "1. GitHub CLI Authentication:"
if sudo -u energy-insights-nl gh auth status 2>&1 | grep -q "Logged in"; then
    echo "   ✅ gh auth OK"
else
    echo "   ❌ gh auth FAILED - Run: sudo -u energy-insights-nl gh auth login"
fi
echo ""

# 2. Git status
echo "2. Git Repository Status:"
cd /opt/github/synctacles-api
if sudo -u energy-insights-nl git status --porcelain | grep -q .; then
    echo "   ⚠️ Uncommitted changes present"
    sudo -u energy-insights-nl git status --short
else
    echo "   ✅ Working directory clean"
fi
echo ""

# 3. Services
echo "3. Critical Services:"
systemctl is-active --quiet energy-insights-nl-api && echo "   ✅ API running" || echo "   ❌ API not running"
echo ""

# 4. Server reminder
echo "4. Infrastructure Reminder:"
echo "   - API Server: ENIN-NL (this server)"
echo "   - Monitoring: monitor.synctacles.com (NOT this server)"
echo ""

echo "=== VERIFICATION COMPLETE ==="
```

**Permissions:**
```bash
chmod +x /opt/github/synctacles-api/scripts/cc_session_start.sh
chown energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/scripts/cc_session_start.sh
```

---

## DEEL 3: GH AUTH PERMANENT FIX

**Referentie:** Voer ook HANDOFF_CAI_CC_GH_AUTH_PERMANENT_FIX.md uit als gh auth nog niet werkt.

Na succesvolle gh auth:
1. Maak alle 11 gap audit issues aan
2. Test met session start script

---

## VERIFICATION CHECKLIST

- [ ] docs/INFRASTRUCTURE.md aangemaakt
- [ ] docs/MONITORING.md aangemaakt
- [ ] docs/skills/SKILL_14_MONITORING.md aangemaakt
- [ ] SKILL_00 updated met session start checklist
- [ ] scripts/cc_session_start.sh aangemaakt (optional)
- [ ] gh auth working
- [ ] Session start script getest

---

## GIT COMMIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add docs/ scripts/
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: add infrastructure, monitoring docs + session start protocol

- INFRASTRUCTURE.md: Server layout (ENIN-NL vs monitor.synctacles.com)
- MONITORING.md: Grafana/Prometheus documentation
- SKILL_14_MONITORING.md: New skill for monitoring infrastructure
- SKILL_00: Added session start checklist (gh auth, server awareness)
- scripts/cc_session_start.sh: Verification script for CC sessions

Fixes documentation gaps that caused wrong server assumptions."

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main
```

---

## DELIVERABLES

1. docs/INFRASTRUCTURE.md
2. docs/MONITORING.md
3. docs/skills/SKILL_14_MONITORING.md
4. SKILL_00 session start checklist update
5. scripts/cc_session_start.sh
6. gh auth working + tested

---

*Template versie: 1.0*
