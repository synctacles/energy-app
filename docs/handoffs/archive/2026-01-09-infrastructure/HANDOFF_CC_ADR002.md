# HANDOFF: ADR-002 PostgreSQL Trust Authentication

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09  
**GitHub Issue:** #46 (afsluiting)

---

## OPDRACHT

Voeg ADR-002 toe aan ARCHITECTURE.md, direct na ADR-001.

---

## CONTENT

```markdown
### ADR-002: PostgreSQL Trust Authentication (No Password)

**Status:** Accepted  
**Date:** 2026-01-09

**Context:**  
Security audit adviseerde scram-sha-256 (password auth) voor PostgreSQL. Standaard best practice is "altijd wachtwoord".

**Decision:**  
Behoud `trust` authentication voor localhost connections. Geen password.

**Rationale:**

Threat model analyse:

| Aanvaller shell als... | trust | password in .env |
|------------------------|-------|------------------|
| service user | DB access | Leest .env → DB access |
| andere user | DB access | .env onleesbaar (600) → geblokkeerd |
| root | DB access | DB access |

Password beschermt alleen tegen niet-service-user én niet-root shell access. Op deze server:
- Geen andere users
- Aanvaller is service user (app exploit) of root
- Password in plaintext .env = security theater

Echte bescherming zit in perimeter:
1. Hetzner FW — voorkomt toegang
2. SSH keys-only — voorkomt toegang
3. localhost binding — DB niet extern bereikbaar

Binnen = binnen.

**Consequences:**
- Eenvoudiger setup (KISS)
- Geen secrets rotation nodig voor DB
- Geen .env credential sprawl
- Bewuste keuze, geen oversight
```

---

## LOCATIE

In ARCHITECTURE.md, onder `## Security Model`, na ADR-001.

---

## GIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add -A
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: ADR-002 PostgreSQL trust authentication rationale #46"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## DAARNA

Sluit GitHub Issue #46.
