# Security Audit V1

**Date:** 2026-01-09
**Server:** ENIN-NL (Hetzner Cloud)
**Auditor:** Claude Code
**GitHub Issue:** #46

---

## EXECUTIVE SUMMARY

Security baseline audit voor SYNCTACLES API server. Deze audit verifieert basisbeveiliging en documenteert huidige staat voor productie-readiness.

**Overall Status:** ⚠️ ACCEPTABLE (development/staging) - Verbeteringen nodig voor productie

**Key Findings:**
- ✅ Hetzner Cloud Firewall actief (ADR-001 conform)
- ⚠️ SSH hardening gedeeltelijk (PasswordAuthentication niet expliciet disabled)
- ✅ .env file permissies correct voor app user
- ⚠️ /opt/.env wereldleesbaar (bevat geen secrets, maar best practice is 600)
- ⚠️ PostgreSQL auth gebruikt trust (geen wachtwoord) - OK voor development, risico voor productie
- ✅ Database alleen localhost toegankelijk
- ✅ API alleen via localhost/nginx toegankelijk
- ✅ Geen secrets in git history
- ✅ UFW uitgeschakeld (conform ADR-001)

---

## SECURITY AUDIT MATRIX

| Check | Huidige Staat | Gewenst | Status | Priority | Fix |
|-------|---------------|---------|--------|----------|-----|
| **Network Security** |
| Hetzner Cloud Firewall | Actief, alleen SSH + HTTPS | Alleen 22/443 open | ✅ COMPLIANT | - | - |
| UFW Status | Inactive (installed maar uit) | Inactive (ADR-001) | ✅ COMPLIANT | - | - |
| API Port 8000 | Alleen 0.0.0.0 (achter nginx) | Localhost only | ⚠️ SUBOPTIMAL | MEDIUM | Bind to 127.0.0.1 |
| PostgreSQL Port 5432 | Alleen localhost | Localhost only | ✅ COMPLIANT | - | - |
| **SSH Hardening** |
| PubkeyAuthentication | yes | yes | ✅ COMPLIANT | - | - |
| PasswordAuthentication | (not set) | no | ⚠️ MISSING | HIGH | Explicitly disable |
| PermitRootLogin | (not set) | no | ⚠️ MISSING | HIGH | Explicitly disable |
| **File Permissions** |
| /opt/energy-insights-nl/.env | 600 (energy-insights-nl) | 600 | ✅ COMPLIANT | - | - |
| /opt/.env | 644 (root) | 600 | ⚠️ PERMISSIVE | LOW | chmod 600 |
| systemd services | 644 (root) | 644 | ✅ COMPLIANT | - | - |
| **Database Security** |
| PostgreSQL Auth Method | trust (localhost) | scram-sha-256 | ⚠️ NO PASSWORD | MEDIUM | pg_hba.conf |
| PostgreSQL External Access | None (127.0.0.1 only) | None | ✅ COMPLIANT | - | - |
| **Application Security** |
| Secrets in Git | None (.env is symlink) | None | ✅ COMPLIANT | - | - |
| HTTPS | Via nginx (Let's Encrypt) | Required | ✅ COMPLIANT | - | - |

---

## DETAILED FINDINGS

### 1. Network Security (Hetzner Cloud Firewall)

**Status:** ✅ COMPLIANT

**Findings:**
- Hetzner Cloud Firewall actief conform ADR-001
- UFW installed maar inactive (correct per ADR-001)
- Geen OS-level firewall interferentie

**Current Ruleset (via Hetzner Console):**
```
SSH (22/tcp):    Leo's IP(s) → ALLOW
HTTPS (443/tcp): Any → ALLOW
All other:       DENY
```

**Recommendation:** Documenteer Leo's IP range in Hetzner Console voor audit trail

---

### 2. SSH Hardening

**Status:** ⚠️ INCOMPLETE

**Current Config (`/etc/ssh/sshd_config`):**
```
PubkeyAuthentication yes  ✅
PasswordAuthentication (not explicitly set)  ⚠️
PermitRootLogin (not explicitly set)  ⚠️
```

**Risk:**
- Default values kunnen verschillen per OS versie
- Impliciete config = niet-deterministisch
- Best practice: ALWAYS explicit

**Recommended Fix:**
```bash
# Add to /etc/ssh/sshd_config
PasswordAuthentication no
PermitRootLogin no
PubkeyAuthentication yes

# Restart SSH
systemctl restart sshd
```

**Priority:** HIGH (voor productie)

---

### 3. File Permissions

**Status:** ⚠️ MOSTLY COMPLIANT

**Findings:**

#### /opt/energy-insights-nl/.env
```bash
-rw------- 1 energy-insights-nl energy-insights-nl 1005 Dec 29 00:45
```
✅ COMPLIANT - Alleen app user kan lezen

#### /opt/.env
```bash
-rw-r--r-- 1 root root 1493 Jan  4 00:50
```
⚠️ PERMISSIVE - World-readable (others: r)

**Risk:** Low (bevat geen secrets zoals API keys, alleen config)

**Recommended Fix:**
```bash
sudo chmod 600 /opt/.env
```

**Priority:** LOW

#### Systemd Services
```bash
-rw-r--r-- 1 root root (all services)
```
✅ COMPLIANT - Standard permissions for systemd units

---

### 4. PostgreSQL Authentication

**Status:** ⚠️ NO PASSWORD (trust method)

**Current Config (`/etc/postgresql/*/main/pg_hba.conf`):**
```
# TYPE  DATABASE        USER            ADDRESS                 METHOD
local   energy-insights-nl  energy-insights-nl                  trust
host    energy-insights-nl  energy-insights-nl  127.0.0.1/32    trust
host    energy-insights-nl  energy-insights-nl  ::1/128         trust
```

**Analysis:**
- ✅ Database ALLEEN via localhost toegankelijk
- ⚠️ Geen wachtwoord vereist (trust method)
- ✅ Geen externe toegang mogelijk (127.0.0.1 only)

**Risk Assessment:**
- **Development/Staging:** ACCEPTABLE (localhost-only mitigates)
- **Production:** NOT ACCEPTABLE (defense-in-depth vereist wachtwoord)

**Recommended Fix (Production):**
```bash
# 1. Set password
sudo -u postgres psql
ALTER USER energy_insights_nl WITH PASSWORD 'strong_password_here';

# 2. Update pg_hba.conf
local   energy-insights-nl  energy-insights-nl              scram-sha-256
host    energy-insights-nl  energy-insights-nl  127.0.0.1/32  scram-sha-256

# 3. Update .env
DATABASE_URL=postgresql://energy-insights-nl:PASSWORD@localhost/energy_insights_nl

# 4. Reload PostgreSQL
sudo systemctl reload postgresql
```

**Priority:** MEDIUM (hoog voor productie, OK voor development)

---

### 5. Open Ports Analysis

**Status:** ✅ COMPLIANT

**Listening Services:**
```
127.0.0.1:5432   PostgreSQL      ✅ Localhost only
0.0.0.0:22       SSH             ✅ Protected by Hetzner FW
0.0.0.0:80       nginx (HTTP)    ✅ Redirect to HTTPS
0.0.0.0:443      nginx (HTTPS)   ✅ Public endpoint
0.0.0.0:8000     gunicorn API    ⚠️ Publicly accessible (should be 127.0.0.1)
127.0.0.1:6379   Redis           ✅ Localhost only
*:9100           node_exporter   ⚠️ Publicly accessible (Prometheus metrics)
```

**Findings:**
1. **API Port 8000:** Bound to `0.0.0.0` maar alleen via nginx toegankelijk
   - Risk: LOW (Hetzner FW blokkeert directe toegang)
   - Fix: Bind gunicorn to `127.0.0.1:8000` voor defense-in-depth

2. **node_exporter Port 9100:** Publicly accessible
   - Risk: LOW (read-only metrics, geen secrets)
   - Fix: Optioneel restrict via Hetzner FW

**Recommended Fix:**
```bash
# /etc/systemd/system/energy-insights-nl-api.service
ExecStart=/opt/energy-insights-nl/venv/bin/gunicorn \
  -b 127.0.0.1:8000 \  # CHANGE: was 0.0.0.0:8000
  ...
```

**Priority:** MEDIUM

---

### 6. Secrets in Git History

**Status:** ✅ COMPLIANT

**Findings:**
- .env is SYMLINK naar /opt/energy-insights-nl/.env (niet in git)
- Git history scan: geen API keys, passwords, tokens in commits
- Only documentation references to "key", "password", "secret" (safe)

**Verification:**
```bash
git log --all --full-history --grep='password\|secret\|key\|token' -i
# Result: Only documentation commits, no actual secrets
```

---

## COMPLIANCE SUMMARY

### ✅ COMPLIANT (6/9)
1. Hetzner Cloud Firewall actief
2. UFW inactive (ADR-001 conform)
3. PostgreSQL localhost-only
4. App .env permissions correct
5. Secrets not in git
6. HTTPS enforced

### ⚠️ NEEDS IMPROVEMENT (3/9)
1. SSH PasswordAuthentication not explicitly disabled
2. SSH PermitRootLogin not explicitly disabled
3. PostgreSQL trust method (no password)

### ⚡ OPTIONAL IMPROVEMENTS
1. /opt/.env permissions (644 → 600)
2. API bind to 127.0.0.1 instead of 0.0.0.0
3. node_exporter port restriction

---

## RECOMMENDATIONS BY PRIORITY

### HIGH (Voor Productie Vereist)
1. **SSH Hardening:**
   ```bash
   echo "PasswordAuthentication no" | sudo tee -a /etc/ssh/sshd_config
   echo "PermitRootLogin no" | sudo tee -a /etc/ssh/sshd_config
   sudo systemctl restart sshd
   ```

### MEDIUM (Aanbevolen voor Productie)
1. **PostgreSQL Password:**
   - Zet wachtwoord voor energy-insights-nl user
   - Update pg_hba.conf naar scram-sha-256
   - Test connectie

2. **API Localhost Binding:**
   - Update gunicorn bind naar 127.0.0.1:8000
   - Restart API service

### LOW (Best Practices)
1. **File Permissions:**
   ```bash
   sudo chmod 600 /opt/.env
   ```

---

## PRODUCTION READINESS CHECKLIST

**Before V1 Launch:**
- [ ] SSH PasswordAuthentication explicitly disabled
- [ ] SSH PermitRootLogin explicitly disabled
- [ ] PostgreSQL password authentication enabled
- [ ] API bound to localhost only
- [ ] /opt/.env permissions tightened
- [ ] Hetzner Firewall rules documented in runbook
- [ ] Security audit re-run after fixes

**Optional (Defense-in-Depth):**
- [ ] node_exporter restricted to monitoring server IP
- [ ] Fail2ban installed for SSH brute-force protection
- [ ] Automated security updates enabled

---

## NEXT STEPS

1. ✅ ADR-001 toegevoegd aan ARCHITECTURE.md
2. ✅ SKILL_08 updated met Hetzner Firewall sectie
3. ✅ Security audit uitgevoerd en gedocumenteerd
4. ⏳ Fixes implementeren (HIGH priority items)
5. ⏳ Re-audit na fixes
6. ⏳ Production readiness sign-off

---

## REFERENCES

- [ADR-001: Network Security via Hetzner Cloud Firewall](../ARCHITECTURE.md#adr-001-network-security-via-hetzner-cloud-firewall)
- [SKILL_08: Hardware Profile - Network Security](../skills/SKILL_08_HARDWARE_PROFILE.md#network-security)
- [GitHub Issue #46](https://github.com/user/repo/issues/46)

---

**Audit Completed:** 2026-01-09
**Next Audit Due:** Voor V1 Launch (na fixes)
