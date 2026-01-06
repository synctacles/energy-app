# 📋 Takenlijst Management - Hoe Het Werkt

## 🎯 3 Manieren om Mij Taken te Geven

### **Optie 1: Simple To-Do Markdown (✨ AANBEVOLEN)**

Geef mij een .md bestand met taken, en ik zal deze:
- In GitHub Issues omzetten
- In een project board plaatsen
- In de codebase integreren
- Automatisch vervolgen

**Voorbeeld formaat:**

```markdown
# Taken voor SYNCTACLES - Q1 2026

## 🔴 CRITICAL (Blocking Launch)

- [ ] Fix CORS configuration in main.py
  - Priority: Immediate
  - Effort: 1 hour
  - Owner: Cloud DevOps
  - Description: Replace allow_origins=["*"] with environment-based config

- [ ] Add post-deployment verification script
  - Priority: High
  - Effort: 3 hours
  - Owner: Deployment
  - Description: Test all 7 endpoints after deploy

## 🟠 HIGH (Next Sprint)

- [ ] Add unit tests for data validation
  - Priority: High
  - Effort: 8 hours
  - Owner: Backend
  - Acceptance Criteria:
    - 60%+ test coverage
    - All critical paths tested
    - CI/CD integration working

- [ ] Setup Dependabot for dependency scanning
  - Priority: High
  - Effort: 2 hours
  - Owner: DevOps
```

---

### **Optie 2: CSV/Spreadsheet Format (als je van Excel houdt)**

Geef mij een CSV met kolommen:
```
Taak,Prioriteit,Effort,Owner,Status,Beschrijving
Fix CORS,CRITICAL,1h,DevOps,TODO,Replace allow_origins config
Add tests,HIGH,8h,Backend,TODO,Reach 60% coverage
```

Ik zet dit om naar GitHub Issues + Markdown takenlijsten.

---

### **Optie 3: GitHub Issues Direct (als je GitHub al gebruikt)**

Je kan rechtstreeks issues in GitHub aanmaken, en ik kan:
- Issues lezen en prioriteren
- Labels toevoegen (bug, feature, enhancement)
- Issues koppelen aan pull requests
- Automatisch sluiten na completion

---

## 🔄 Hoe Ik Met Taken Werk

### **Stap 1: Takenlijst Ontvangen**
Je geeft mij een takenlijst (Markdown, CSV, of GitHub Issues)

### **Stap 2: Parsing & Organisatie**
Ik parse alle taken en zet op in:
- `/TAKEN_*.md` bestand (traceable)
- GitHub Issues (synchronisatie)
- Project board (visual overview)

### **Stap 3: Implementatie**
Voor elke taak:
- Maak een branch: `feature/TAAK-X-beschrijving`
- Update `TAKEN_*.md` bij start (status: IN_PROGRESS)
- Commit regelmatig (zie progress)
- Sluit task af met commit message

### **Stap 4: Reporting**
Ik geef je:
- Daily standup (welke taken afgerond)
- Weekly progress (velocity tracking)
- Status updates (blockers, risks)

---

## 📝 Voorbeeld: Takenlijst Aanmaken

**Jij geeft dit:**
```markdown
# Sprint Tasks - Week 1 (Jan 6-12)

## CRITICAL
- [ ] Fix CORS configuration
- [ ] Add security headers
- [ ] Enable Dependabot

## HIGH
- [ ] Write unit tests
- [ ] Setup monitoring alerts
- [ ] Create deployment verification script
```

**Ik doe dit:**
1. ✅ Maak `/docs/tasks/TAKEN_2026-01-06_SPRINT-1.md`
2. ✅ Zet om naar GitHub Issues (met labels, assignees)
3. ✅ Update TodoWrite during work (real-time progress)
4. ✅ Create feature branches (`feature/fix-cors`, etc)
5. ✅ Track completion met commits
6. ✅ Give daily standups

---

## 🎯 Takenlijst Structuur (Wat Ik Verwacht)

**Minimaal:**
```
- [ ] Task beschrijving
```

**Beter:**
```
- [ ] Task beschrijving
  - Priority: HIGH/MEDIUM/LOW
  - Effort: 2-3 hours
  - Owner: [naam/role]
```

**Best:**
```
- [ ] Task beschrijving
  - Priority: HIGH
  - Effort: 3 hours
  - Owner: DevOps Team
  - Acceptance Criteria:
    - All endpoints responding
    - Health check passes
    - No errors in logs
  - Related Issues: #123, #456
  - Blocks: Launch readiness
```

---

## 💻 GitHub Integration (Optioneel)

Als je GitHub Issues al gebruikt, kan ik:

### **Automatisch Issues Synchroniseren**
```bash
# Ik kan issues lezen:
gh issue list --state open --json title,number,labels

# Ik kan issues creëren/updaten:
gh issue create --title "Fix CORS" --body "..." --label "critical"
```

### **Project Board Management**
```bash
# Ik kan taken op project board plaatsen:
gh project item add [project-id] --content-url [issue-url]
```

---

## 📊 Takenlijst Formats Die Ik Ondersteun

| Format | Voordeel | Hoe Geven |
|--------|----------|-----------|
| **Markdown (.md)** | Traceable in git, duidelijk | Zet in `/docs/tasks/` folder |
| **CSV/Excel** | Makkelijk in Excel maken | Upload als CSV file |
| **GitHub Issues** | Native integration | Direct in GitHub aanmaken |
| **Slack/Discord** | Real-time | Paste in message |
| **Plain Text** | Snel** | Any format, ik parse het |

---

## 🚀 Best Practice: Takenlijsten Aanmaken

### **Stap 1: Maak een Takenlijst Bestand**

```markdown
# TAKEN - Sprint Januari 6-20, 2026
Created: 2026-01-06
Owner: Leo
Status: ACTIVE

## 🔴 CRITICAL - Week 1 (Jan 6-12)

### Security Hardening
- [ ] Fix CORS configuration
  Priority: IMMEDIATE
  Effort: 1h
  Owner: CC
  Status: TODO

- [ ] Add security headers
  Priority: IMMEDIATE
  Effort: 1h
  Owner: CC

### Automation
- [ ] Create post-deploy verification script
  Priority: IMMEDIATE
  Effort: 3h
  Owner: CC

## 🟠 HIGH - Week 2 (Jan 13-20)

- [ ] Enable Dependabot
- [ ] Add Bandit security scanning
- [ ] Create security policy document

## 🟡 MEDIUM - Q1 2026

- [ ] Add unit tests (60%+ coverage)
- [ ] Setup monitoring dashboards
- [ ] Create on-call playbook
```

### **Stap 2: Geef Mij Dit Bestand**

Zet in `/docs/tasks/` of stuur via prompt:
```
"Hier is mijn takenlijst. Zet deze om naar:
1. GitHub Issues
2. Project board items
3. Markdown tracking
Zet prioriteit op CRITICAL taken."
```

### **Stap 3: Ik Track De Voortgang**

Ik zal:
- ✅ Taken parsen
- ✅ Issues in GitHub aanmaken
- ✅ Branches maken per task
- ✅ Daily updates geven
- ✅ Weekly reports maken

---

## 📈 Voorbeeld Workflow

### Day 1: Takenlijst Ontvangen
```markdown
# Sprint 1 - Security Hardening (Jan 6-12)

Critical:
- [ ] Fix CORS
- [ ] Add security headers
- [ ] Enable Dependabot
- [ ] Create verification script
```

### Day 2: Ik Begin
```
Parsed 4 tasks:
✅ Created GitHub Issues (SYNCTACLES #1-4)
✅ Added labels: security, critical, immediate
✅ Set milestone: v1.0.0 (Launch)
✅ Created branches:
   - feature/security-cors-fix
   - feature/security-headers
   - feature/security-dependabot
   - feature/deploy-verification
```

### Day 3-5: Daily Updates
```
PROGRESS UPDATE - Jan 8
======================
✅ DONE (2/4):
  - Fix CORS configuration (#1) → PR #42
  - Add security headers (#2) → PR #43

🔄 IN PROGRESS (2/4):
  - Enable Dependabot (#3) - 50% done, testing today
  - Create verification script (#4) - Started, testing tomorrow

Blockers: None
Risks: None
```

### Day 10: Sprint Review
```
SPRINT COMPLETE - Jan 12
========================
✅ 4/4 DONE
  - Fix CORS (#1)
  - Add security headers (#2)
  - Enable Dependabot (#3)
  - Create verification script (#4)

📊 Metrics:
  Velocity: 4 tasks / week
  Quality: 0 bugs found in PR review
  Estimate accuracy: 95%

🚀 Ready for next sprint!
```

---

## 🔗 Folder Structure voor Taken

```
/docs/tasks/                    ← Alle takenlijsten
├── TAKEN_2026-01-06_SPRINT-1.md
├── TAKEN_2026-01-20_SPRINT-2.md
├── TAKEN_2026-02-03_SPRINT-3.md
└── README.md                   (this guide)
```

---

## ✅ Klaar?

**Dit is het proces:**

1. **Jij:** Geef mij een takenlijst (Markdown, CSV, GitHub Issues)
2. **Ik:** Parse taken, maak issues, track voortgang
3. **Jij:** Check daily updates en PR's
4. **Ik:** Report weekly progress
5. **Jij:** Give feedback, adjust priorities
6. **Repeat:** Each sprint

---

## 🎯 Hoe Start Je Nu

### **Optie A: Direct Markdown**
```
Geef mij dit in een message:

# Tasks - Week 1

Critical:
- [ ] Task 1 (Priority: HIGH, Effort: 3h)
- [ ] Task 2 (Priority: CRITICAL, Effort: 1h)

High:
- [ ] Task 3 (Priority: HIGH, Effort: 8h)

Zet om naar GitHub Issues + tracking markdown
```

### **Optie B: Bestand Aanmaken**
```bash
# Ik kan dit voor je doen:
1. Maak /docs/tasks/TAKEN_YYYY-MM-DD_SPRINT-X.md
2. Zet je takenlijst erin
3. Zet om naar GitHub Issues
4. Begin met implementatie
```

### **Optie C: GitHub Issues Direct**
```bash
# Zet rechtstreeks issues in GitHub, ik zal:
1. Issues lezen
2. Labels toevoegen
3. Project board managen
4. Track voortgang
```

---

## 📞 Vragen?

- **"Hoe geef ik je taken?"** → Markdown format hierboven
- **"Kan je GitHub Issues gebruiken?"** → Ja! Natuurlijk
- **"Hoe track je voortgang?"** → Daily updates + weekly reports
- **"Kan je prioriteiten veranderen?"** → Ja, zeg het me in een message

---

**Status:** Ready for tasks! 🚀
**Format:** Markdown, CSV, GitHub Issues (all supported)
**Frequency:** Daily updates, weekly reports
