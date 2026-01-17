# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** Security Fix

---

## CONTEXT

GitHub Actions "Dependency Security Scan" workflow gefaald op DATADIO/synctacles-api.

Screenshot toont:
- Workflow: "Scan Dependencies for Vulnerabilities"
- Status: Failed (1 min 3 sec)
- Repo: DATADIO/synctacles-api

---

## TASK

1. **Diagnose** - Bekijk welke CVEs gevonden zijn
2. **Assess** - Bepaal severity en impact
3. **Fix** - Update dependencies
4. **Verify** - Re-run security scan

---

## DIAGNOSE

```bash
cd /opt/github/synctacles-api

# Check GitHub Actions output (als gh authenticated)
gh run list --workflow="Dependency Security Scan" --limit 1
gh run view [RUN_ID] --log-failed

# Of check lokaal met pip-audit
pip install pip-audit
pip-audit -r requirements.txt

# Of met safety
pip install safety
safety check -r requirements.txt

# Check requirements
cat requirements.txt
cat requirements-dev.txt 2>/dev/null
```

---

## COMMON CVE FIXES

```bash
# Na identificatie, update specifieke packages:
pip install --upgrade [package_name]

# Of update alle packages (voorzichtig):
pip install --upgrade -r requirements.txt

# Regenerate requirements met vaste versies:
pip freeze > requirements.txt
```

---

## VERIFICATION

```bash
# Re-run security scan lokaal
pip-audit -r requirements.txt
# Moet 0 vulnerabilities tonen

# Test dat app nog werkt
cd /opt/github/synctacles-api
source venv/bin/activate
python -m pytest tests/ 2>/dev/null || echo "No tests"

# Check API start
sudo systemctl restart energy-insights-nl-api
curl -s http://localhost:8000/health | jq .
```

---

## GIT COMMIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add requirements*.txt
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "security: update dependencies to fix CVE-XXXX-XXXXX

- Updated [package] from X.Y.Z to A.B.C
- Fixes [CVE ID]: [korte beschrijving]
- Verified no breaking changes"

sudo -u energy-insights-nl git -C /opt/github/synctacles-api push origin main
```

---

## DELIVERABLES

1. Lijst van gevonden CVEs + severity
2. Welke packages geüpdatet
3. Verificatie dat scan slaagt
4. Verificatie dat app nog werkt

---

*Template versie: 1.0*
