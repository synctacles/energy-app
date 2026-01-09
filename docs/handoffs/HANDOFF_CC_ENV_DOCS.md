# HANDOFF: Documenteer .env Dependencies

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09  
**Context:** Incident - services down na verwijderen /opt/.env

---

## OPDRACHT

Voeg sectie toe aan SKILL_08_HARDWARE_PROFILE.md:

```markdown
## Environment Files

### /opt/.env (Primary)

**Gebruikt door:**
- `energy-insights-nl-api.service` (EnvironmentFile)
- `energy-insights-nl-collector.service` (EnvironmentFile)
- `energy-insights-nl-importer.service` (EnvironmentFile)
- `energy-insights-nl-normalizer.service` (EnvironmentFile)

**Bevat:**
- DATABASE_URL
- INSTALL_PATH
- API configuratie

**⚠️ NIET VERWIJDEREN** — alle services zijn hiervan afhankelijk.

### /opt/energy-insights-nl/.env (Symlink/Copy)

**Relatie:** Identiek aan /opt/.env of symlink ernaar.

**Backup locatie:** `/opt/energy-insights-nl/backups/`

### Verificatie

```bash
# Check welke services .env gebruiken
grep -r "EnvironmentFile" /etc/systemd/system/energy-insights-nl-*.service
```
```

---

## GIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add -A
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: environment file dependencies SKILL_08"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```
