# SKILL 00 — CLAUDE CODE OPERATING PROTOCOL

**MANDATORY READING BEFORE ANY ACTION**
Version: 1.0 (2026-01-06)
Status: ENFORCED

---

## ⛔ STOP — READ THIS FIRST

Dit document is VERPLICHT voor elke Claude Code sessie.
Geen acties, geen fixes, geen edits voordat je dit hebt gelezen EN bewezen dat je het snapt.

**Incident aanleiding:** 4.5 uur verspild door CC die:
- SKILLs niet las
- Aannames maakte zonder verificatie
- Productie ging "fixen" zonder toestemming
- Deprecated services probeerde te repareren
- TenneT aanraakte terwijl dit OFF-LIMITS is

---

## SECTIE A: MANDATORY SKILL READING

### Lees ALTIJD voor je begint:

| SKILL | Bestand | Waarom verplicht |
|-------|---------|------------------|
| **SKILL 01** | SKILL_01_HARD_RULES.md | Non-negotiable regels, fail-fast, KISS |
| **SKILL 02** | SKILL_02_ARCHITECTURE.md | System design, TenneT BYO-KEY model |
| **SKILL 11** | SKILL_11_REPO_AND_ACCOUNTS.md | Git workflow, service accounts, GEEN ROOT |

### Lees INDIEN RELEVANT:

| SKILL | Bestand | Wanneer |
|-------|---------|---------|
| SKILL 03 | SKILL_03_CODING_STANDARDS.md | Bij code schrijven |
| SKILL 06 | SKILL_06_DATA_SOURCES.md | Bij data pipeline werk |
| SKILL 09 | SKILL_09_INSTALLER_SPECS.md | Bij deployment/setup |
| SKILL 10 | SKILL_10_DEPLOYMENT_WORKFLOW.md | Bij deployment |
| SKILL 13 | SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md | Bij logging/debugging |

### Bewijs dat je gelezen hebt:

**START ELKE SESSIE MET:**
```
"Ik heb SKILL_01, SKILL_02, SKILL_11 gelezen.
Key points:
- [1 bullet SKILL_01]
- [1 bullet SKILL_02]
- [1 bullet SKILL_11]

Mag ik beginnen?"
```

**WACHT OP GOEDKEURING VOORDAT JE VERDERGAAT.**

---

## SECTIE B: PROTECT MODE = DEFAULT

### Wat PROTECT MODE betekent:

```
✅ TOEGESTAAN:
- Lezen (cat, view, less)
- Analyseren (grep, find, ls)
- Vragen stellen
- Documenteren

❌ VERBODEN:
- Bestanden aanpassen
- Services herstarten
- Git commits
- Database wijzigingen
- "Even snel fixen"
```

### Wanneer PROTECT MODE eindigt:

**ALLEEN** wanneer Leo expliciet zegt:
- "1" of "go" of "execute"
- "Ja, pas aan"
- "Fix it"

**NIET** bij:
- "Interessant"
- "Hmm"
- "Kun je kijken naar..."
- Stilte

---

## SECTIE C: VERIFICATIE VOOR CONCLUSIES

### ❌ VERBODEN gedrag:

```
"Script mist" → zonder `ls -la` output
"Service is broken" → zonder te vragen of het deprecated is
"Ik fix even" → zonder expliciete toestemming
"Volgens mij..." → zonder verificatie
```

### ✅ VERPLICHT gedrag:

```
STAP 1: Observatie
$ ls -la /opt/energy-insights-nl/app/scripts/
[toon output]

STAP 2: Vraag
"Ik zie dat run_importers.sh ontbreekt. Is dit:
 a) Deprecated (niet nodig)
 b) Broken (moet gerepareerd)
 c) Anders?"

STAP 3: WACHT op antwoord

STAP 4: Pas DAN actie voorstellen
```

---

## SECTIE D: FAILED SERVICES PROTOCOL

### Bij `systemctl list-units --failed`:

**STAP 1:** Toon output, GEEN interpretatie

**STAP 2:** Vraag ALTIJD:
```
"Ik zie X failed services:
- service-a
- service-b
- service-c

Welke zijn:
1. Deprecated (negeren)
2. Intentioneel uit (bijv. TenneT)
3. Daadwerkelijk broken (onderzoeken)
?"
```

**STAP 3:** WACHT op antwoord

**STAP 4:** Onderzoek ALLEEN wat Leo aanwijst als "broken"

### ⛔ NOOIT:

- Aannemen dat failed = moet gerepareerd
- Alle services tegelijk proberen te fixen
- TenneT services aanraken (BYO-KEY model, zie SKILL_02)

---

## SECTIE E: OFF-LIMITS GEBIEDEN

### Raak NOOIT aan zonder expliciete instructie:

| Gebied | Reden | Documentatie |
|--------|-------|--------------|
| **TenneT services** | Juridisch: geen redistributie | SKILL_02 §TenneT BYO-KEY |
| **synctacles-* services** | Deprecated (oude naming) | SKILL_11 |
| **/opt/.env** | Productie secrets | SKILL_01 |
| **Database credentials** | Security | SKILL_03 |

### Bij twijfel:

```
"Ik wil [X] aanpassen. Dit raakt [Y gebied].
Is dit toegestaan of off-limits?"
```

---

## SECTIE F: GIT DISCIPLINE

### ELKE git operatie:

```bash
# CORRECT
sudo -u energy-insights-nl git -C /opt/github/synctacles-api <command>

# FOUT - NOOIT DOEN
git <command>
sudo git <command>
```

### Na file edits:

```bash
# VERPLICHT
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
```

### Commit messages:

```
<type>: <wat>

<waarom>

<accountability als relevant>
```

---

## SECTIE G: ESCALATIE

### Wanneer STOPPEN en VRAGEN:

1. **Onduidelijke scope** - "Moet dit gerepareerd of is het deprecated?"
2. **Meerdere mogelijke oorzaken** - "Ik zie 3 mogelijkheden, welke eerst?"
3. **Productie impact** - "Dit raakt live services, mag ik doorgaan?"
4. **Twijfel over off-limits** - "Raakt dit TenneT/secrets/deprecated code?"
5. **Conflicterende informatie** - "SKILL zegt X, maar ik zie Y"

### Format:

```
⚠️ ESCALATIE:
- Situatie: [wat ik zie]
- Twijfel: [waarom ik stop]
- Opties: [a, b, c]
- Vraag: [concrete vraag]
```

---

## SECTIE H: SESSION CHECKLIST

### Start sessie:

```
□ SKILL_00 gelezen (dit document)
□ SKILL_01, SKILL_02, SKILL_11 gelezen
□ Key points samengevat aan Leo
□ Goedkeuring ontvangen om te beginnen
□ PROTECT MODE = actief
```

### Tijdens sessie:

```
□ Verificatie VOOR conclusies
□ Vragen VOOR acties
□ Geen edits zonder "1"/"go"
□ Failed services: vraag deprecated vs broken
□ Off-limits gebieden: niet aanraken
```

### Einde sessie:

```
□ Alle wijzigingen gedocumenteerd
□ Git commits met accountability
□ Ownership fixes uitgevoerd
□ Status gerapporteerd
```

---

## SECTIE I: CONSEQUENCES

### Bij overtreding van dit protocol:

1. **Leo stopt de sessie**
2. **Wijzigingen worden teruggedraaid**
3. **Tijd is verspild**

### De 4.5-uur incident bewees:

- CC las SKILLs niet → TenneT rabbit hole (45 min verspild)
- CC verifieerde niet → Verkeerde script geblamed (60 min verspild)
- CC vroeg niet → Productie bijna gesloopt
- CC nam aan → 3.5 uur verspild van 4.5 uur totaal

**Dit protocol bestaat om herhaling te voorkomen.**

---

## SECTIE J: QUICK REFERENCE

```
┌─────────────────────────────────────────────────┐
│  CC OPERATING PROTOCOL - QUICK REFERENCE        │
├─────────────────────────────────────────────────┤
│                                                 │
│  1. LEES SKILLS EERST (01, 02, 11 minimum)     │
│  2. BEWIJS DAT JE ZE GELEZEN HEBT              │
│  3. PROTECT MODE = DEFAULT                      │
│  4. VERIFICATIE VOOR CONCLUSIES                 │
│  5. VRAAG VOOR ACTIES                           │
│  6. FAILED ≠ MOET GEREPAREERD                   │
│  7. TENNET = OFF-LIMITS                         │
│  8. GIT = ALTIJD ALS SERVICE USER               │
│  9. BIJ TWIJFEL: STOP EN VRAAG                  │
│ 10. GEEN "IK FIX EVEN"                          │
│                                                 │
└─────────────────────────────────────────────────┘
```

---

## GERELATEERDE SKILLS

| SKILL | Focus |
|-------|-------|
| SKILL_01 | Hard rules, fail-fast, KISS |
| SKILL_02 | Architecture, TenneT BYO-KEY |
| SKILL_03 | Coding standards |
| SKILL_05 | Communication rules |
| SKILL_06 | Data sources |
| SKILL_09 | Installer specs |
| SKILL_10 | Deployment workflow |
| SKILL_11 | Repo structure, git discipline |
| SKILL_13 | Logging, diagnostics |

---

**LAATSTE WAARSCHUWING:**

Dit protocol is geen suggestie. Het is een vereiste.
Elke sessie begint met bewijs dat je dit gelezen hebt.
Geen uitzonderingen.

---

**Document Owner:** Leo
**Enforcement:** Strict
**Last Updated:** 2026-01-06
