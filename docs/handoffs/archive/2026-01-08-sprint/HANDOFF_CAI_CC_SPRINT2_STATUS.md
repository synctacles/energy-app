# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** NORMAL
**Sprint:** 2 Kickoff

---

## TASK DESCRIPTION

Status check voor Sprint 2 start. Rapporteer huidige server/repo staat.

---

## GEVRAAGDE INFO

### 1. Server Status
```bash
# Services
systemctl status energy-insights-nl-api --no-pager
systemctl list-units --failed

# Disk
df -h /opt

# Recent logs (errors only)
journalctl -u energy-insights-nl-api --since "1 hour ago" -p err --no-pager
```

### 2. Git Status
```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api status
sudo -u energy-insights-nl git -C /opt/github/synctacles-api log --oneline -5
```

### 3. HA Component Repo
```bash
sudo -u energy-insights-nl git -C /opt/github/ha-energy-insights-nl status
sudo -u energy-insights-nl git -C /opt/github/ha-energy-insights-nl log --oneline -3
```

### 4. Database Quick Check
```bash
sudo -u energy-insights-nl psql -d energy_insights_nl -c "SELECT COUNT(*) FROM norm_entso_e_a75 WHERE timestamp > NOW() - INTERVAL '24 hours';"
```

---

## OUTPUT FORMAT

Rapporteer in dit format:

```markdown
## STATUS REPORT

### Server
- API: [running/failed]
- Disk: [X/Y used]
- Errors last hour: [count]

### Git (synctacles-api)
- Branch: [main]
- Clean: [yes/no]
- Last commit: [hash] [message]

### Git (ha-energy-insights-nl)
- Branch: [main]
- Clean: [yes/no]
- Last commit: [hash] [message]

### Database
- Records 24h: [count]

### Blocking Issues
- [none / list]
```

---

## OUT OF SCOPE

- Geen fixes uitvoeren
- Geen deployments
- Alleen observeren en rapporteren

---

## CONTEXT

Sprint 2 focus: HA Component TenneT BYO-key implementation.
Status check bepaalt of we kunnen starten of eerst issues moeten oplossen.

---

*Template versie: 1.0*
