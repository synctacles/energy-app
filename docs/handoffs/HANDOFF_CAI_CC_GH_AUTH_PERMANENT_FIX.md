# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** CRITICAL (BLOCKER)
**Type:** Infrastructure Fix

---

## PROBLEEM

GitHub CLI (`gh`) authenticatie werkt niet, ondanks dat Leo gisteren een fine-grained PAT heeft aangemaakt op het service account.

Dit probleem komt **keer op keer terug** en blokkeert:
- GitHub issues aanmaken
- PR workflows
- Alle gh CLI operaties

**Dit moet PERMANENT gefixed worden.**

---

## DIAGNOSE

```bash
# 1. Check huidige auth status
gh auth status

# 2. Check welke user/token geconfigureerd is
cat ~/.config/gh/hosts.yml
cat /home/energy-insights-nl/.config/gh/hosts.yml 2>/dev/null

# 3. Check environment variables
env | grep GH
env | grep GITHUB

# 4. Check of PAT token ergens geconfigureerd is
cat ~/.gitconfig | grep -A5 credential
git config --list | grep credential

# 5. Check systemd service user context
whoami
id
```

---

## MOGELIJKE OORZAKEN

| Oorzaak | Check | Fix |
|---------|-------|-----|
| gh niet geauthenticeerd | `gh auth status` toont "not logged in" | `gh auth login` |
| Verkeerde user context | `whoami` ≠ energy-insights-nl | `sudo -u energy-insights-nl gh auth login` |
| Token expired/revoked | gh auth toont error | Nieuwe token genereren |
| Fine-grained PAT permissions | Token mist repo/issues scope | Token permissions updaten |
| Token niet persistent | Werkt in sessie, weg na reboot | Store in config file |

---

## FIX PROCEDURE

### Stap 1: Bepaal correcte user context

```bash
# Alle gh operaties moeten als energy-insights-nl user
sudo -u energy-insights-nl whoami
# Moet "energy-insights-nl" tonen
```

### Stap 2: Check bestaande auth

```bash
sudo -u energy-insights-nl gh auth status
```

### Stap 3: Login met PAT (als niet authenticated)

**Optie A: Interactive login**
```bash
sudo -u energy-insights-nl gh auth login
# Kies: GitHub.com
# Kies: HTTPS
# Kies: Paste authentication token
# Plak de fine-grained PAT die Leo heeft aangemaakt
```

**Optie B: Direct met token (non-interactive)**
```bash
# Als Leo de token kan delen:
echo "ghp_XXXXXXX" | sudo -u energy-insights-nl gh auth login --with-token
```

### Stap 4: Verify auth persistent

```bash
# Check config file bestaat
sudo -u energy-insights-nl ls -la ~/.config/gh/
sudo -u energy-insights-nl cat ~/.config/gh/hosts.yml

# Test gh werkt
sudo -u energy-insights-nl gh auth status
sudo -u energy-insights-nl gh repo list --limit 1
```

### Stap 5: Test issue creation

```bash
cd /opt/github/synctacles-api
sudo -u energy-insights-nl gh issue list --limit 1
sudo -u energy-insights-nl gh issue create --title "[TEST] gh auth test" --body "Test issue - can be closed" --label "test"
# Als succesvol, sluit test issue
sudo -u energy-insights-nl gh issue close [ISSUE_NUMBER]
```

---

## FINE-GRAINED PAT VEREISTEN

De PAT moet deze permissions hebben:

**Repository permissions:**
- `Issues`: Read and write
- `Metadata`: Read (automatisch)
- `Contents`: Read and write (voor commits)
- `Pull requests`: Read and write (indien nodig)

**Repository access:**
- DATADIO/synctacles-api
- ldraaisma/ha-energy-insights-nl (indien nodig)

---

## PERMANENTE OPLOSSING

Om te voorkomen dat dit terugkomt:

### 1. Config file permissions

```bash
# Ensure config directory exists en correct ownership
sudo mkdir -p /home/energy-insights-nl/.config/gh
sudo chown -R energy-insights-nl:energy-insights-nl /home/energy-insights-nl/.config/
sudo chmod 700 /home/energy-insights-nl/.config/gh
```

### 2. Documenteer in SKILL

Na fix, update SKILL_11_REPO_AND_ACCOUNTS.md met:
```markdown
## GitHub CLI Authentication

**User:** energy-insights-nl
**Config:** /home/energy-insights-nl/.config/gh/hosts.yml
**Token type:** Fine-grained PAT
**Permissions:** Issues (RW), Contents (RW), Metadata (R)

**Re-auth procedure:**
sudo -u energy-insights-nl gh auth login
```

### 3. Alias voor CC (optioneel)

```bash
# In /home/energy-insights-nl/.bashrc
alias gh='gh'  # gh already works, but ensure path

# Of script wrapper
cat > /usr/local/bin/gh-issues << 'EOF'
#!/bin/bash
cd /opt/github/synctacles-api
sudo -u energy-insights-nl gh "$@"
EOF
chmod +x /usr/local/bin/gh-issues
```

---

## VERIFICATIE CHECKLIST

- [ ] `gh auth status` toont authenticated
- [ ] `gh repo list` werkt
- [ ] `gh issue list` werkt
- [ ] `gh issue create` werkt
- [ ] Config file bestaat in `/home/energy-insights-nl/.config/gh/`
- [ ] Permissions correct (700 voor dir, 600 voor files)
- [ ] Werkt na nieuwe shell sessie
- [ ] Werkt als energy-insights-nl user

---

## DELIVERABLES

1. Root cause van auth failure
2. Permanente fix geïmplementeerd
3. Verificatie dat gh CLI werkt
4. SKILL_11 update (optioneel)
5. Test: 1 GitHub issue succesvol aangemaakt

---

## NA SUCCESVOLLE FIX

Direct uitvoeren:
1. Maak alle 11 gap audit issues aan (uit GITHUB_ISSUES_TO_CREATE.md)
2. Sluit de 3 HIGH issues als "fixed" met referentie naar commits

---

*Template versie: 1.0*
