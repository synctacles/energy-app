# HANDOFF: SSH Hardening + Security Fixes #46

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09  
**GitHub Issue:** #46

---

## CONTEXT

Security audit V1 afgerond. Nu fixes implementeren. **KRITIEK:** SSH hardening kan lockout veroorzaken. Strikte volgorde vereist.

---

## FASE 1: PRE-FLIGHT CHECKS (GEEN WIJZIGINGEN)

### 1.1 Controleer sudo rechten energy-insights-nl

```bash
# Check of user in sudo group zit
groups energy-insights-nl

# Check sudoers
sudo cat /etc/sudoers | grep energy-insights-nl
sudo cat /etc/sudoers.d/* | grep energy-insights-nl

# Test sudo werkt
sudo -u energy-insights-nl sudo whoami
# Verwacht: root
```

**STOP als sudo niet werkt. Fix eerst:**
```bash
sudo usermod -aG sudo energy-insights-nl
```

### 1.2 Controleer SSH keys voor energy-insights-nl

```bash
# Check of authorized_keys bestaat en gevuld is
ls -la /home/energy-insights-nl/.ssh/
cat /home/energy-insights-nl/.ssh/authorized_keys

# Vergelijk met root's keys (moeten matchen)
cat /root/.ssh/authorized_keys
```

**Als keys ontbreken, kopieer van root:**
```bash
mkdir -p /home/energy-insights-nl/.ssh
cp /root/.ssh/authorized_keys /home/energy-insights-nl/.ssh/
chown -R energy-insights-nl:energy-insights-nl /home/energy-insights-nl/.ssh
chmod 700 /home/energy-insights-nl/.ssh
chmod 600 /home/energy-insights-nl/.ssh/authorized_keys
```

### 1.3 Test SSH login (HANDMATIG - LEO + CC)

**STOP HIER - Wacht op bevestiging van Leo:**

Leo test:
```bash
ssh energy-insights-nl@<server-ip>
sudo whoami  # moet "root" returnen
```

CC test (nieuwe sessie):
```bash
ssh energy-insights-nl@<server-ip>
sudo whoami  # moet "root" returnen
```

**PAS NA BEVESTIGING BEIDE TESTS GESLAAGD → GA NAAR FASE 2**

---

## FASE 2: SSH HARDENING

### 2.1 Backup huidige config

```bash
sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup.$(date +%Y%m%d)
```

### 2.2 SSH hardening toepassen

```bash
# PasswordAuthentication uitschakelen
echo "PasswordAuthentication no" | sudo tee -a /etc/ssh/sshd_config

# Root login uitschakelen
echo "PermitRootLogin no" | sudo tee -a /etc/ssh/sshd_config
```

### 2.3 Valideer config syntax

```bash
sudo sshd -t
# Verwacht: geen output = OK
# Als errors: STOP, fix config
```

### 2.4 Restart SSH

```bash
sudo systemctl restart sshd
```

### 2.5 TEST ONMIDDELLIJK (nieuwe terminal, sluit huidige NIET)

```bash
# Nieuwe terminal:
ssh energy-insights-nl@<server-ip>

# Als dit faalt: gebruik nog-open sessie om te reverten:
sudo cp /etc/ssh/sshd_config.backup.* /etc/ssh/sshd_config
sudo systemctl restart sshd
```

---

## FASE 3: OVERIGE FIXES

### 3.1 Onderzoek /opt/.env

```bash
# Wat staat erin?
cat /opt/.env

# Vergelijk met app .env
diff /opt/.env /opt/energy-insights-nl/.env
```

**Rapporteer inhoud aan Leo. Mogelijke acties:**
- Verwijderen (als duplicate/cruft)
- chmod 600 (als nodig)
- Niets (als onschuldig)

### 3.2 API bind naar localhost

```bash
# Zoek huidige gunicorn config
grep -r "0.0.0.0:8000" /etc/systemd/system/

# Edit service file
sudo nano /etc/systemd/system/synctacles-api.service
# OF
sudo nano /etc/systemd/system/energy-insights-nl-api.service

# Wijzig:
# -b 0.0.0.0:8000 → -b 127.0.0.1:8000

# Reload en restart
sudo systemctl daemon-reload
sudo systemctl restart synctacles-api  # of energy-insights-nl-api
```

**Verificatie:**
```bash
ss -tlnp | grep 8000
# Verwacht: 127.0.0.1:8000, NIET 0.0.0.0:8000
```

---

## FASE 4: DOCUMENTATIE UPDATE

Update SECURITY_AUDIT_V1.md met resultaten:
- [ ] SSH hardening: ✅ COMPLIANT
- [ ] /opt/.env: [resultaat onderzoek]
- [ ] API localhost bind: ✅ COMPLIANT
- [ ] node_exporter: ✅ Hetzner FW restricted

---

## NIET NODIG: PostgreSQL Password Auth

**Besluit:** Definitief geschrapt. Trust method blijft.

**Rationale:**
- Password in .env = plaintext op disk
- Als aanvaller shell heeft als service user → leest .env → heeft password
- Dus: binnen = binnen, password voegt niets toe
- Echte bescherming zit in perimeter (Hetzner FW) en localhost-only binding

**Security model:**
```
Hetzner FW (alleen 22/443) → SSH keys-only → localhost DB binding
```

Als aanvaller door alle drie komt, is DB password irrelevant.

---

## VOLGORDE SAMENVATTING

```
1. Pre-flight checks (sudo + SSH keys)
2. LEO + CC testen SSH login als energy-insights-nl
3. ══════ WACHT OP BEVESTIGING ══════
4. SSH hardening (PasswordAuth=no, PermitRootLogin=no)
5. Test nieuwe SSH sessie VOORDAT je huidige sluit
6. /opt/.env onderzoek + rapport
7. API bind 127.0.0.1
8. Docs updaten
```

---

## ROLLBACK PROCEDURES

### SSH lockout

Als je buitengesloten bent:
1. Hetzner Console → Server → Rescue mode
2. Mount filesystem
3. Revert sshd_config van backup
4. Reboot

### API niet bereikbaar na bind change

```bash
# Revert naar 0.0.0.0
sudo sed -i 's/127.0.0.1:8000/0.0.0.0:8000/' /etc/systemd/system/*-api.service
sudo systemctl daemon-reload
sudo systemctl restart synctacles-api
```

---

## EXIT CRITERIA

- [ ] SSH login werkt voor Leo als energy-insights-nl
- [ ] SSH login werkt voor CC als energy-insights-nl  
- [ ] PasswordAuthentication=no in sshd_config
- [ ] PermitRootLogin=no in sshd_config
- [ ] /opt/.env onderzocht en gerapporteerd
- [ ] API bindt op 127.0.0.1:8000
- [ ] SECURITY_AUDIT_V1.md updated
- [ ] Geen service downtime

---

## SKILLS LEZEN

- `/mnt/project/SKILL_08_HARDWARE_PROFILE.md`
- `/mnt/project/SKILL_11_REPO_AND_ACCOUNTS.md`
