# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** Bug Fix (Recurring)

---

## PROBLEEM

GitHub Actions "Dependency Security Scan" **faalt opnieuw** op commit b1092f7.

Dit is dezelfde commit waar CC eerder 13 CVEs claimde te hebben gefixed.

**Screenshot toont:**
- Workflow: "Scan Dependencies for Vulnerabilities"
- Status: Failed (1 min 7 sec)
- Commit: b1092f7 (de "CVE fix" commit)

---

## MOGELIJKE OORZAKEN

| Oorzaak | Kans | Check |
|---------|------|-------|
| requirements.txt niet gepusht | HOOG | `git diff origin/main -- requirements.txt` |
| Lokale venv update, requirements.txt niet | HOOG | Compare venv packages vs requirements.txt |
| Nieuwe CVEs sinds fix | LAAG | Check workflow log voor specifieke CVEs |
| Workflow scant andere file | MEDIUM | Check workflow yaml |

---

## DIAGNOSE

### Stap 1: Check wat GitHub heeft

```bash
cd /opt/github/synctacles-api

# Fetch latest
sudo -u energy-insights-nl git fetch origin

# Check requirements.txt op GitHub vs lokaal
sudo -u energy-insights-nl git diff origin/main -- requirements.txt

# Check commit b1092f7 inhoud
sudo -u energy-insights-nl git show b1092f7 --stat
sudo -u energy-insights-nl git show b1092f7 -- requirements.txt
```

### Stap 2: Check workflow logs

```bash
# Bekijk failed workflow
sudo -u energy-insights-nl gh run list --workflow="Dependency Security Scan" --limit 3
sudo -u energy-insights-nl gh run view [RUN_ID] --log-failed
```

### Stap 3: Verify lokale venv vs requirements.txt

```bash
# Wat zit er in venv?
/opt/energy-insights-nl/venv/bin/pip list | grep -E "aiohttp|starlette|multipart|urllib3"

# Wat staat in requirements.txt?
grep -E "aiohttp|starlette|multipart|urllib3" requirements.txt
```

### Stap 4: Check workflow file

```bash
cat .github/workflows/dependency-scan.yml
# Welke file scant de workflow?
```

---

## WAARSCHIJNLIJKE FIX

**Als requirements.txt niet correct gepusht:**

```bash
# 1. Update requirements.txt met huidige venv packages
/opt/energy-insights-nl/venv/bin/pip freeze > requirements.txt

# OF specifiek de gefixte packages:
cat > requirements.txt.patch << 'EOF'
python-multipart==0.0.21
aiohttp==3.13.3
starlette==0.50.0
urllib3==2.6.3
EOF

# 2. Verify
grep -E "aiohttp|starlette|multipart|urllib3" requirements.txt

# 3. Commit en push
sudo -u energy-insights-nl git add requirements.txt
sudo -u energy-insights-nl git commit -m "fix: sync requirements.txt with actual CVE-fixed versions

Previous commit b1092f7 updated venv but requirements.txt was not properly synced.

Fixed versions:
- python-multipart: 0.0.21
- aiohttp: 3.13.3
- starlette: 0.50.0 (via fastapi)
- urllib3: 2.6.3"

sudo -u energy-insights-nl git push origin main
```

---

## VERIFICATION

```bash
# 1. Wacht op nieuwe workflow run
sudo -u energy-insights-nl gh run list --workflow="Dependency Security Scan" --limit 1

# 2. Check status
sudo -u energy-insights-nl gh run view [NEW_RUN_ID]

# 3. Moet "completed successfully" tonen
```

---

## ROOT CAUSE ANALYSE

Na fix, documenteer:

1. **Wat ging er mis bij de originele fix?**
   - Was requirements.txt niet gesaved?
   - Was er een git add vergeten?
   - Scande de workflow een andere file?

2. **Hoe voorkomen we dit?**
   - Altijd `pip freeze > requirements.txt` na venv updates
   - Verify file in git diff voor commit
   - Check workflow na push

---

## DELIVERABLES

1. Root cause van herhaalde failure
2. Correcte requirements.txt gepusht
3. Workflow passing ✅
4. Documentatie van wat mis ging

---

*Template versie: 1.0*
