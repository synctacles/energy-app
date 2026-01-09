# HANDOFF: Launch Plan Implementatie

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09

---

## OPDRACHT

### 1. Launch Plan opslaan

Kopieer `/mnt/user-data/outputs/LAUNCH_PLAN_V1.md` naar:

```
/opt/github/synctacles-api/docs/LAUNCH_PLAN.md
```

Dit is het master launch document. Root van docs folder.

### 2. GitHub Issues Reorganiseren

```bash
# FASE 1 - OTP (CRITICAL)
sudo -u energy-insights-nl gh issue edit 5 --add-label "critical" --milestone "V1 Launch"

# FASE 2 - Launch Prep
sudo -u energy-insights-nl gh issue edit 44 --add-label "high" --milestone "V1 Launch"
sudo -u energy-insights-nl gh issue edit 45 --add-label "high" --milestone "V1 Launch"
sudo -u energy-insights-nl gh issue edit 50 --add-label "medium" --milestone "V1 Launch"

# FASE 3 - Beta
sudo -u energy-insights-nl gh issue edit 51 --add-label "high" --milestone "V1 Launch"
sudo -u energy-insights-nl gh issue edit 52 --add-label "low" --milestone "V1 Launch"
sudo -u energy-insights-nl gh issue edit 53 --add-label "high" --milestone "V1 Launch"

# Sluit #47 (HA component is klaar)
sudo -u energy-insights-nl gh issue close 47 --comment "HA Component v1.0.0 implemented and working. Documentation added."
```

### 3. Git Commit

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add docs/LAUNCH_PLAN.md
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: add V1 launch plan with OTP structure"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

### 4. Start Unit Tests

Na bovenstaande, voer `HANDOFF_CAI_UNIT_TESTS.md` uit.

---

## VOLGORDE

```
1. Launch plan opslaan in repo
2. GitHub issues reorganiseren
3. Commit + push
4. Start #5 (unit tests)
```

---

## EXIT CRITERIA

- [ ] `docs/LAUNCH_PLAN.md` bestaat in repo
- [ ] Issues correct gelabeld
- [ ] #47 gesloten
- [ ] Unit tests gestart
