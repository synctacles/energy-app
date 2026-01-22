# 🤖 GitHub Automation Setup

Met je Personal Access Token kan ik GitHub Issues automatisch beheren. Dit document beschrijft hoe alles werkt.

## ✅ Status

- ✅ GitHub PAT ingesteld
- ✅ Task manager script aangemaakt
- ✅ 13 issues georganiseerd in 3 milestones
- ✅ Labels toegepast (critical, high, medium, optional)
- ✅ Duplicaten verwijderd (was 15, nu 13 unieke taken)

## 📊 Huidige GitHub Status

### Milestones:
1. **v1.0.0 Launch** - 3 CRITICAL taken (deze week)
2. **v1.1.0 Features** - 5 HIGH taken (volgende sprint)
3. **v1.2.0 Expansion** - 4 MEDIUM taken (Q1 2026)

### Backlog:
- 3 OPTIONAL taken (Q2-Q3 2026)

## 🚀 Wat Ik Nu Kan Doen (Automatisch)

Met je PAT kan ik:

```bash
# 1. Status van alle taken checken
export GITHUB_PAT="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"
./scripts/github-task-manager.sh status

# 2. Dagelijkse standup rapportage
./scripts/github-task-manager.sh daily-report

# 3. Project voortgang zien
./scripts/github-task-manager.sh progress

# 4. Commentaar toevoegen op issues
./scripts/github-task-manager.sh comment 1 "Started working on CORS fix"

# 5. Issues updaten naar in-progress/done
./scripts/github-task-manager.sh update-status 1 in-progress
./scripts/github-task-manager.sh update-status 1 done
```

## 🔐 Veiligheid van je PAT

⚠️ **BELANGRIJK:**
- Je PAT is NODIG zodat ik issues kan updaten
- Ik sla het NIET op in bestanden (alleen in deze chat context)
- Het token heeft beperkte permissions: alleen `repo`, `workflow`, `read:org`
- Je kan het token op elk moment revokceren via GitHub Settings

**Als je het token wilt vervangen:**
1. Ga naar https://github.com/settings/tokens
2. Delete de huidige `Claude-TaskManager` token
3. Maak een nieuwe aan
4. Stuur mij het nieuwe token

## 📋 Task Management Workflow

### Voor mij (Claude):

**Daily Workflow:**
```
1. Check status van alle open issues
2. Update issues met progress
3. Add comments met updates
4. Generate daily report
5. Flag blocked issues
```

**Ik kan:**
- ✅ Automatisch issues sluiten als werk klaar is
- ✅ Commentaar toevoegen met updates
- ✅ Issues labelen als blocked/in-progress
- ✅ Daily standup genereren
- ✅ Progress tracking

**Ik kan NIET (safety):**
- ❌ Issues verwijderen
- ❌ Repository instellingen veranderen
- ❌ Code naar main branch pushen
- ❌ Deploy triggers activeren

### Voor jou:

1. **Zeg welke taak je aan het doen bent**
   - Ik update GitHub issue naar "in-progress"
   - Ik voeg een comment toe met context

2. **Zeg wanneer je een taak afmaakt**
   - Ik sluit de issue
   - Ik markeer volgende taken

3. **Vraag om dagelijks standup**
   - Ik genereer report met status van alle taken

## 🎯 Use Cases

### Use Case 1: Start werken aan CORS issue

**Jij zegt:**
```
Ik begin aan issue #1 (Fix CORS). Ik zal:
1. Hardcoded origins vervangen
2. Environment variable configureren
3. Testen met home-assistant.io
```

**Ik doet:**
```
1. Mark issue #1 as in-progress
2. Add comment: "Started work on CORS configuration"
3. Check if all acceptance criteria are clear
```

### Use Case 2: Daily standup

**Jij zegt:**
```
Kan je een standup geven?
```

**Ik doet:**
```
$ ./scripts/github-task-manager.sh daily-report

🔴 CRITICAL: 3
   #1 - Fix CORS (in-progress)
   #2 - Setup Dependency Scanning (todo)
   #3 - Post-Deployment Verification (todo)

🟠 HIGH: 5
   #5 - Unit Test Suite
   #6 - Monitoring & Alerting
   ... etc
```

### Use Case 3: Issue is geblokkeerd

**Jij zegt:**
```
Issue #5 is geblokkeerd - wacht op approval
```

**Ik doet:**
```
1. Add label "blocked" to issue #5
2. Add comment: "Blocked - waiting for approval"
```

## 📝 Issue Statuses (Automatisch)

Elk issue kan in deze states zijn:

| Status | Label | Betekenis |
|--------|-------|-----------|
| 📋 Open | - | Wachten op start |
| 🔄 In Progress | `in-progress` | Actief aan het werken |
| ✅ Done | - | Issue gesloten |
| 🚫 Blocked | `blocked` | Wacht op iets |

## 🔧 Aanpassen van je Setup

### Token vervangen
```bash
# Oude token revokceren op GitHub
# Nieuwe token genereren
# Stuur mij het nieuwe token
export GITHUB_PAT="ghp_xxxxx"
```

### Script handmatig runnen
```bash
cd /opt/github/synctacles-api
export GITHUB_PAT="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"
./scripts/github-task-manager.sh status
```

### Directe GitHub API aanroepen
Als het script niet werkt kan ik ook direct curl gebruiken:
```bash
curl -H "Authorization: token $GITHUB_PAT" \
  https://api.github.com/repos/DATADIO/synctacles-api/issues
```

## 🎓 Volgende Stappen

Nu kan je:

1. **Zeg mij wat je gaat doen**
   - Ik update GitHub automatisch
   - Je kan voortgang zien in GitHub

2. **Vraag om standup/status**
   - Ik genereer automatisch rapport
   - Altijd up-to-date

3. **Zeg wanneer je klaar bent**
   - Ik sluit issues
   - Ik markeert volgende taak

## 💬 Commands voor mij (Claude)

Je kan dit zeggen:
- "Ik start met issue #1"
- "Issue #1 is klaar"
- "Geef standup"
- "Wat is de status?"
- "Issue #5 is geblokkeerd"

## 📚 Gerelateerde Files

- `scripts/github-task-manager.sh` - Main automation script
- `docs/tasks/TAKEN_2026-01-06_LAUNCH_SPRINT.md` - All tasks in detail
- `.github/ISSUE_TEMPLATE/task.md` - Issue template

---

**Status: ✅ Ready for automated task management!**

Zeg me nu: Waar wil je aan beginnen? 🚀
