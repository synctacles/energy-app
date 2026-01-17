# HANDOFF: Security Audit #46 + ADR-001

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09  
**GitHub Issue:** #46

---

## CONTEXT

Leo bevestigt: servers zijn beveiligd via **Hetzner Cloud Firewall**, niet UFW. Dit moet gedocumenteerd worden en de security audit moet hierop aangepast.

---

## TAKEN

### 1. ADR-001 toevoegen aan ARCHITECTURE.md

Voeg toe onder `## Security Model` sectie:

```markdown
### ADR-001: Network Security via Hetzner Cloud Firewall

**Status:** Accepted  
**Date:** 2026-01-09

**Context:**  
Servers draaien op Hetzner Cloud. Keuze tussen UFW (OS-level) of Hetzner Cloud Firewall (netwerkniveau).

**Decision:**  
Hetzner Cloud Firewall als primaire netwerkbeveiliging. Geen UFW op servers.

**Rationale:**
- KISS: één firewall-laag, centraal beheerd
- Traffic geblokkeerd vóór server (minder load)
- Eenvoudiger auditing via Hetzner console
- Minder OS-configuratie drift tussen servers

**Consequences:**
- Firewall rules alleen via Hetzner console/API
- Security audit checkt Hetzner Firewall, niet UFW
```

---

### 2. SKILL_08_HARDWARE_PROFILE.md updaten

**Verwijder** alle UFW referenties en vervang door:

```markdown
## NETWORK SECURITY

### Hetzner Cloud Firewall (Primary)

Alle netwerkbeveiliging via Hetzner Cloud Firewall, niet OS-level.

**Huidige Ruleset (documenteer actuele staat):**

| Rule | Protocol | Port | Source | Action |
|------|----------|------|--------|--------|
| SSH | TCP | 22 | Leo's IP(s) | Allow |
| HTTPS | TCP | 443 | Any | Allow |
| API (internal) | TCP | 8000 | Localhost only | - |
| All other | * | * | * | Deny |

**Beheer:** Hetzner Cloud Console → Firewalls

**Waarom geen UFW:**
- Zie ADR-001 in ARCHITECTURE.md
- Hetzner FW blokkeert traffic vóór server
- Centraal beheer, minder drift
```

---

### 3. Security Audit uitvoeren

**Scan en documenteer:**

```bash
# A. Hetzner Firewall - handmatig checken via console
# Documenteer huidige rules in SKILL_08

# B. SSH hardening
grep -E "^(PasswordAuthentication|PermitRootLogin|PubkeyAuthentication)" /etc/ssh/sshd_config

# C. File permissions
ls -la /opt/energy-insights-nl/.env
ls -la /opt/github/synctacles-api/

# D. PostgreSQL auth
sudo cat /etc/postgresql/*/main/pg_hba.conf | grep -v "^#" | grep -v "^$"

# E. Open ports (wat luistert lokaal)
ss -tlnp

# F. Systemd service permissions
ls -la /etc/systemd/system/synctacles-*.service
```

**Genereer rapport:** `SECURITY_AUDIT_V1.md`

| Check | Huidige Staat | Gewenst | Status | Fix |
|-------|---------------|---------|--------|-----|
| Hetzner FW | ? | Alleen 22/443 open | ? | - |
| SSH PasswordAuth | ? | no | ? | sshd_config |
| SSH RootLogin | ? | no | ? | sshd_config |
| .env permissions | ? | 600 | ? | chmod |
| .env ownership | ? | energy-insights-nl | ? | chown |
| PG auth method | ? | scram-sha-256 | ? | pg_hba.conf |

---

### 4. Output verwacht

- [ ] ARCHITECTURE.md: ADR-001 toegevoegd
- [ ] SKILL_08: Hetzner Firewall sectie, UFW refs verwijderd
- [ ] SECURITY_AUDIT_V1.md: rapport met huidige staat
- [ ] Optioneel: hardening script (dry-run safe)

---

## GEEN BREAKING CHANGES

- Geen services herstarten
- Geen firewall rules wijzigen
- Alleen documentatie + rapport

---

## SKILLS LEZEN

- `/mnt/project/SKILL_08_HARDWARE_PROFILE.md`
- `/mnt/project/ARCHITECTURE.md`
- `/mnt/project/SKILL_01_HARD_RULES.md`

---

## GIT WORKFLOW

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api pull
# ... maak wijzigingen ...
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add -A
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: ADR-001 Hetzner Firewall + security audit #46"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```
