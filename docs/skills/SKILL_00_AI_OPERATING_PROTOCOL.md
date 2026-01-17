# SKILL 00 â€” AI OPERATING PROTOCOL

**MANDATORY READING BEFORE ANY ACTION**
Version: 2.2 (2026-01-16)
Status: ENFORCED
Scope: Claude Code (CC) + Claude AI (CAI)

---

## â›” STOP â€” READ THIS FIRST

Dit document is VERPLICHT voor elke AI sessie (CC Ã©n CAI).
Geen acties, geen fixes, geen edits voordat je dit hebt gelezen EN bewezen dat je het snapt.

**Incident aanleiding:** 4.5 uur verspild door CC die:
- SKILLs niet las
- Aannames maakte zonder verificatie
- Productie ging "fixen" zonder toestemming
- Deprecated services probeerde te repareren
- TenneT aanraakte terwijl dit OFF-LIMITS is

---

# DEEL 1: ALGEMEEN PROTOCOL (CC + CAI)

---

## SECTIE A: MANDATORY SKILL READING

### Lees ALTIJD voor je begint:

| SKILL | Bestand | Waarom verplicht |
|-------|---------|------------------|
| **SKILL 00** | SKILL_00_AI_OPERATING_PROTOCOL.md | Dit document |
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
"Ik heb SKILL_00, SKILL_01, SKILL_02, SKILL_11 gelezen.
Key points:
- [1 bullet SKILL_00]
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
âœ… TOEGESTAAN:
- Lezen (cat, view, less)
- Analyseren (grep, find, ls)
- Vragen stellen
- Documenteren
- Plannen maken

âŒ VERBODEN:
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
- "Akkoord, uitvoeren"

**NIET** bij:
- "Interessant"
- "Hmm"
- "Kun je kijken naar..."
- Stilte

---

## SECTIE C: VERIFICATIE VOOR CONCLUSIES

### âŒ VERBODEN gedrag:

```
"Script mist" â†’ zonder `ls -la` output
"Service is broken" â†’ zonder te vragen of het deprecated is
"Ik fix even" â†’ zonder expliciete toestemming
"Volgens mij..." â†’ zonder verificatie
```

### âœ… VERPLICHT gedrag:

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

## SECTIE C2: CC SESSION MODES

### Twee operationele modes:

**SUPERVISED MODE (default):**
```
1. CC presenteert plan: "Ik ga X doen"
2. CC wacht op expliciete goedkeuring: "go" / "1" / "execute"
3. CC voert uit
4. CC rapporteert resultaat
```

**AUTONOMOUS MODE (opt-in):**
```
1. CC voert direct uit zonder te vragen
2. CC rapporteert alleen resultaat
```

### Wanneer SUPERVISED verplicht:

- **ALTIJD** voor HIGH risk servers (productie)
- **ALTIJD** voor destructieve acties (delete, drop, restart prod services)
- **ALTIJD** voor nieuwe/onbekende taken
- **ALTIJD** bij twijfel over impact
- **ALTIJD** voor off-limits gebieden

### Wanneer AUTONOMOUS toegestaan:

**Alleen als ALLE voorwaarden waar zijn:**
- Leo heeft expliciet "autonomous mode" aangegeven voor deze sessie
- Server is LOW risk (dev/test/monitor)
- Actie staat op whitelist (zie beneden)
- Geen productie impact
- Reversible zonder data loss

### Action Whitelist (autonomous safe):

```
âœ… READ-ONLY:
- cat, less, view, grep, find, ls
- systemctl status
- journalctl (read)
- git log, git status, git diff
- Database SELECT queries (read-only)

âœ… LOW-RISK WRITES:
- git pull (op dev/test)
- Documentation updates
- Log file analysis
- Test runs (niet-productie)

âŒ ALTIJD SUPERVISED:
- systemctl restart/stop/start
- git push/commit
- File edits (code, config)
- Database INSERT/UPDATE/DELETE
- Service installs/updates
- Firewall changes
- User management
```

### Server Risk Levels:

| Server | Risk | Default Mode | Override |
|--------|------|--------------|----------|
| synct-prd (prod) | HIGH | SUPERVISED | Geen |
| coefficient (prod) | HIGH | SUPERVISED | Geen |
| synct-dev (ENIN-NL) | LOW | SUPERVISED | Autonomous OK |
| synct-tst (test) | LOW | SUPERVISED | Autonomous OK |
| monitor | LOW | SUPERVISED | Autonomous OK |
| sideproject | MEDIUM | SUPERVISED | Alleen voor specifieke taken |

### Mode Activation:

**Leo activeert autonomous:**
```
"CC autonomous mode voor deze sessie op synct-dev"
"CC, autonomous deployment naar test"
```

**Leo schakelt terug naar supervised:**
```
"CC supervised mode"
"CC, vraag toestemming voor acties"
```

**CC gedrag:**
- **Zonder expliciete autonomous activatie** → ALTIJD SUPERVISED
- **Bij activatie** → Check server risk + action whitelist
- **Bij twijfel** → SUPERVISED (veiligste optie)
- **Rapporteer modus** aan start sessie

### Voorbeeld Flow:

```
# SUPERVISED (default)
CC: "Ik ga API service herstarten. Plan:
     1. systemctl restart energy-insights-nl-api
     2. Check logs
     3. Verify health endpoint
     
     Execute? (go/1)"
Leo: "go"
CC: [uitvoeren + rapporteren]

# AUTONOMOUS (opt-in, whitelisted action)
Leo: "CC autonomous mode, check logs op dev"
CC: [direct uitvoeren]
CC: "Logs geanalyseerd. Laatste 50 entries tonen geen errors."
```

### Safety Override:

**CC moet ALTIJD stoppen en vragen bij:**
- Onverwachte situatie tijdens autonomous mode
- Impact groter dan verwacht
- Twijfel over juiste actie
- Potentieel data loss

**Autonomous betekent NIET "blindly execute"**
**Het betekent: "execute pre-approved safe actions without asking"**

---

## SECTIE D: OFF-LIMITS GEBIEDEN

### Raak NOOIT aan zonder expliciete instructie:

| Gebied | Reden | Documentatie |
|--------|-------|--------------|
| **TenneT services** | Juridisch: geen redistributie | SKILL_02 Â§TenneT BYO-KEY |
| **synctacles-* services** | Deprecated (oude naming) | SKILL_11 |
| **/opt/.env** | Productie secrets | SKILL_01 |
| **Database credentials** | Security | SKILL_03 |

### Bij twijfel:

```
"Ik wil [X] aanpassen. Dit raakt [Y gebied].
Is dit toegestaan of off-limits?"
```

---

## SECTIE E: ESCALATIE

### Wanneer STOPPEN en VRAGEN:

1. **Onduidelijke scope** - "Moet dit gerepareerd of is het deprecated?"
2. **Meerdere mogelijke oorzaken** - "Ik zie 3 mogelijkheden, welke eerst?"
3. **Productie impact** - "Dit raakt live services, mag ik doorgaan?"
4. **Twijfel over off-limits** - "Raakt dit TenneT/secrets/deprecated code?"
5. **Conflicterende informatie** - "SKILL zegt X, maar ik zie Y"

### Format:

```
âš ï¸ ESCALATIE:
- Situatie: [wat ik zie]
- Twijfel: [waarom ik stop]
- Opties: [a, b, c]
- Vraag: [concrete vraag]
```

---

## SECTIE F: SESSION CHECKLIST

### Start sessie:

```
â–¡ SKILL_00 gelezen (dit document)
â–¡ SKILL_01, SKILL_02, SKILL_11 gelezen
â–¡ Key points samengevat aan Leo
â–¡ Goedkeuring ontvangen om te beginnen
â–¡ PROTECT MODE = actief
â–¡ STATUS_MERGED_CURRENT.md gelezen (indien bestaat)
```

### Tijdens sessie:

```
â–¡ Verificatie VOOR conclusies
â–¡ Vragen VOOR acties
â–¡ Geen edits zonder "1"/"go"
â–¡ Failed services: vraag deprecated vs broken
â–¡ Off-limits gebieden: niet aanraken
â–¡ chown DIRECT na file edits (CC only)
```

### Einde sessie:

```
â–¡ Alle wijzigingen gedocumenteerd
â–¡ STATUS_[CC|CAI]_CURRENT.md bijgewerkt
â–¡ SESSIE_CC_[DATUM].md opgeleverd bij significante CC sessies
â–¡ Git commits met accountability (CC only)
â–¡ Status gerapporteerd aan Leo
```

---

# DEEL 2: CLAUDE CODE (CC) SPECIFIEK

---

## SECTIE G: CC GIT DISCIPLINE

### ELKE git operatie:

```bash
# CORRECT - SSH key bestaat voor energy-insights-nl user
sudo -u energy-insights-nl git -C /opt/github/synctacles-api <command>

# FOUT - NOOIT DOEN
git <command>
sudo git <command>
cd /opt/github/synctacles-api && git push
```

### Na file edits (KRITIEK):

**âš ï¸ Na ELKE file creatie of edit â†’ DIRECT chown uitvoeren**

Niet wachten tot het einde van de sessie. Niet wachten tot voor git commit.
Direct na de edit, vÃ³Ã³r de volgende actie.

```bash
# NA ELKE FILE EDIT - GEEN UITZONDERINGEN
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
```

**Waarom direct?**
- Root-owned files blokkeren git operations
- Service user kan root-owned files niet lezen
- Problemen stapelen op als je wacht

**Patroon:**
```bash
# 1. Edit file (als root is OK)
nano /opt/github/synctacles-api/docs/file.md

# 2. DIRECT daarna - niet later
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/

# 3. Dan pas volgende actie
```

**Bij meerdere files:**
```bash
# Edit file 1
# Edit file 2
# Edit file 3
# chown (eenmalig voor batch is OK, maar VOOR volgende stap)
sudo chown -R energy-insights-nl:energy-insights-nl /opt/github/synctacles-api/
# Dan pas git of andere acties
```

### Commit messages:

```
<type>: <wat>

<waarom>

<accountability als relevant>
```

---

## SECTIE H: CC FAILED SERVICES PROTOCOL

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

### â›” NOOIT:

- Aannemen dat failed = moet gerepareerd
- Alle services tegelijk proberen te fixen
- TenneT services aanraken (BYO-KEY model, zie SKILL_02)

---

## SECTIE H2: HUB-SPOKE INFRASTRUCTURE

### Vanaf 2026-01-16: CC werkt via Hub-Spoke model

```
â"Œâ"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"
â"‚                    CC HUB                       â"‚
â"‚             135.181.201.253                     â"‚
â"‚          User: ccops (dedicated)                â"‚
â"‚                                                 â"‚
â"‚  â"Œâ"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"  â"‚
â"‚  â"‚  SSH Config + Spoke Keys:            â"‚  â"‚
â"‚  â"‚  - coefficient  (91.99.150.36)        â"‚  â"‚
â"‚  â"‚  - monitor      (77.42.41.135)        â"‚  â"‚
â"‚  â"‚  - synct-dev    (135.181.255.83)      â"‚  â"‚
â"‚  â"‚  - synct-tst    (TBD)                 â"‚  â"‚
â"‚  â"‚  - synct-prd    (TBD)                 â"‚  â"‚
â"‚  â"‚  - sideproject  (TBD)                 â"‚  â"‚
â"‚  â""â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"˜  â"‚
â""â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"˜
         â"‚          â"‚          â"‚
    â"Œâ"€â"€â"€â"€â"´â"€â"€â"    â"Œâ"€â"€â"´â"€â"€â"    â"Œâ"€â"€â"´â"€â"€â"
    â"‚ SPOKE 1â"‚    â"‚ ... â"‚    â"‚ SPOKE Nâ"‚
    â""â"€â"€â"€â"€â"€â"€â"€â"€â"˜    â""â"€â"€â"€â"€â"€â"˜    â""â"€â"€â"€â"€â"€â"€â"€â"€â"˜
```

### CC Workflow (NIEUWE STANDAARD):

**ALTIJD via Hub:**
```bash
# CC draait op synct-dev (135.181.255.83)
# CC SSH't naar hub als ccops
ssh cc-hub

# Vanuit hub naar spoke servers
ssh coefficient
ssh monitor
ssh synct-dev  # (ja, terug naar synct-dev is mogelijk)
```

**Directe SSH = ALLEEN in noodgevallen**

### Key Architecture:

**Hub (ccops user):**
- 6 spoke key pairs (`id_coefficient`, `id_monitor`, etc.)
- SSH config met alle spoke aliassen
- Centrale security boundary

**Spoke servers:**
- Accepteren ALLEEN hub's public key
- Firewall: SSH poort 22 alleen open voor hub IP
- Service-specifieke users (coefficient, energy-insights-nl, monitoring, etc.)

### CC Access Pattern:

```bash
# CC heeft private key: id_ccops_hub
# Opgeslagen op synct-dev: ~/.ssh/id_ccops_hub

# CC → Hub
ssh -i ~/.ssh/id_ccops_hub ccops@135.181.201.253

# Hub → Spoke (via SSH config)
ssh coefficient  # Automatisch: id_coefficient key
```

### Security Voordelen:

- **1 inbound SSH** per spoke (alleen hub IP)
- **Centrale audit trail** (alle CC acties via hub)
- **Key rotation** simpeler (1x op hub)
- **Schaalbaarheid** (nieuwe servers = nieuwe spoke config)

### Bidirectioneel vs Unidirectioneel

**CRITICAL - Belangrijke distinctie:**

**synct-dev = BIDIRECTIONEEL (synct-dev ↔ hub):**
```
synct-dev heeft:
- id_ccops_hub (private key naar hub)
- SSH config met cc-hub entry
- Kan naar hub verbinden

WAAROM:
- CC draait op synct-dev (operator locatie)
- CC moet handoffs lezen op hub
- CC moet via hub naar spokes
- Dit is CC's "werkplek"
```

**Alle andere spokes = UNIDIRECTIONEEL (hub → spoke only):**
```
coefficient, monitor, synct-tst, synct-prd hebben GEEN:
- Private key naar hub
- SSH config voor hub
- Mogelijkheid om hub te bereiken

WAAROM:
- CC draait daar NIET
- Geen operationele behoefte
- Security: least privilege
- Managed servers, geen operator locaties
```

**Regel van duim:**
- **Operator locatie** (waar CC draait) = bidirectioneel
- **Managed servers** (CC beheert via hub) = unidirectioneel

**Toekomst:**
Als CC later ook op andere servers draait (bijv. monitoring server voor local tasks), dan wordt die ook bidirectioneel. Het gaat om "waar opereert CC", niet om "alle spokes symmetrisch".

### Emergency Bypass:

**Als hub down is:**
- Leo kan direct SSH naar spoke servers
- Leo's persoonlijke SSH key blijft altijd werkend op alle servers
- CC kan tijdelijk direct vanaf synct-dev (na toestemming Leo)

### Migration Status:

- âœ… Hub opgezet (135.181.201.253)
- âœ… ccops user aangemaakt
- âœ… Spoke keys gegenereerd
- âœ… SSH config aangemaakt
- â¬œ Public keys deployen naar spokes (in progress)
- â¬œ Firewall restricties (hub-only)
- â¬œ Test alle connections

**VANAF NU: CC gebruikt ALLEEN hub-spoke model voor server toegang**

---

## SECTIE H3: HOME ASSISTANT DEPLOYMENT ACCESS

### Architectuur Positie

Home Assistant is **GEEN spoke server** in de hub-spoke infrastructuur:
- HA is een **consumer test device** (development/test doeleinden)
- Draait lokaal op Leo's netwerk (VM op 192.168.2.1)
- Niet deel van productie infrastructuur
- Outside de managed server topology

### Connection Details

**Direct Access (synct-dev → HA):**
```
Host: ha
  HostName: 82.169.33.175 (public IP)
  Port: 22222
  User: root
  IdentityFile: ~/.ssh/id_ha
  IdentitiesOnly: yes
```

**Network Path:**
```
CC (synct-dev) → Internet → Router NAT (82.169.33.175:22222)
                 → Port Forward → HA VM (192.168.2.1:22222)
```

**Security Model:**
- SSH key-based authentication (no password)
- Router firewall: Source IP restricted to 135.181.255.83 (synct-dev only)
- HA SSH add-on: authorized_keys configured with CC's public key
- Test device = acceptable exposure via port forwarding

### SSH Key Setup

**Private key location (synct-dev):**
```bash
~/.ssh/id_ha
```

**Public key (added to HA SSH add-on config):**
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAII9/glpxNjnwimQRVJXVlSnK03fwKsjA9DmZAD8u4IBH cc-to-homeassistant
```

**Key fingerprint:**
```
SHA256:gtLeDOfanRpD7oZa6+xRsr0GGuezRVhpup8AWu58+9o
```

### HA Component Deployment

**Custom Component Path:**
```
/config/custom_components/ha_energy_insights_nl/
```

**Deployment Commands:**
```bash
# Test connection
ssh ha 'hostname && ls /config/custom_components/'

# Upload single file
scp sensor.py ha:/config/custom_components/ha_energy_insights_nl/

# Sync entire directory
rsync -av --delete local_dir/ ha:/config/custom_components/ha_energy_insights_nl/

# Dashboard deployment (if needed)
scp dashboard.yaml ha:/config/dashboards/
```

### IP Address Changes

**Internal IP kan wijzigen** (192.168.2.1 → 192.168.2.X):
- CC verbindt naar **public IP** (82.169.33.175:22222)
- Leo past **port forwarding** aan in router (nieuwe interne destination)
- CC SSH config hoeft **niet** gewijzigd (public IP blijft stabiel)

**Bij public IP wijziging:**
- Leo update SSH config op synct-dev: `HostName: NEW.PUBLIC.IP`
- Of gebruik DynDNS voor stabiele hostname

### Deployment Protocol

**SUPERVISED MODE:**
- HA is test device = lage risk
- Maar blijf protocol volgen:
  1. CC genereert/wijzigt code lokaal
  2. Toont diff aan Leo
  3. Wacht op "Go" approval
  4. Deploy via `scp` of `rsync`
  5. Leo restart HA integration indien nodig

**NOT via hub:**
- HA deployment gaat **niet** via hub
- Hub is voor managed servers (coefficient, monitor, etc.)
- HA is consumer device met directe toegang

### Troubleshooting

**Connection failed:**
```bash
# Test port reachability
nc -zv 82.169.33.175 22222

# Test auth (should succeed)
ssh -i ~/.ssh/id_ha -p 22222 root@82.169.33.175 'echo SUCCESS'

# Check source IP (should be 135.181.255.83)
curl -4 ifconfig.me
```

**Permission denied:**
- Check HA SSH add-on running
- Verify public key in HA authorized_keys config
- Check router firewall allows source IP 135.181.255.83

**Port forwarding not working:**
- Verify router NAT rule: 22222 → 192.168.2.1:22222
- Check HA internal IP hasn't changed
- Test from local network first (192.168.2.1:22222)

---

## SECTIE I: CC NETWERK & PERMISSIES

### CC draait op ENIN-NL server (NIET in sandbox)

CC heeft WEL:
- Internet toegang
- Git push/pull naar GitHub (via SSH)
- API calls naar externe services

**NIET zeggen:** "Je moet zelf pushen want ik heb geen internet"
**WEL doen:** Direct pushen na commit

### User Context

| Operatie | User | Command Prefix |
|----------|------|----------------|
| Git (status, pull, commit, push) | service user | `sudo -u energy-insights-nl` |
| File edits in repo | root | Direct na edit: `sudo chown -R energy-insights-nl:...` |
| systemctl (restart, status) | root | `sudo` |
| apt install | root | `sudo` |
| /etc/ configuratie | root | `sudo` |
| alembic migrations | service user | `sudo -u energy-insights-nl` |
| Python/pip in venv | service user | `sudo -u energy-insights-nl` |

**âš ï¸ File edits:** Root mag editen, maar ownership DIRECT fixen (zie Sectie G).

---

# DEEL 3: CLAUDE AI (CAI) SPECIFIEK

---

## SECTIE J: CAI VERANTWOORDELIJKHEDEN

### CAI doet WEL:

```
âœ… Architectuur design en review
âœ… Planning en projectmanagement
âœ… Documentatie schrijven en structureren
âœ… Code review (op basis van gedeelde code)
âœ… SKILL updates en uitbreidingen
âœ… ADR's opstellen
âœ… Strategische adviezen
âœ… Troubleshooting analyse (zonder server toegang)
```

### CAI doet NIET:

```
âŒ Directe server toegang
âŒ Git commits (geen repo toegang)
âŒ Service restarts
âŒ File edits op server
âŒ Database queries
âŒ API calls naar productie
```

### CAI's output = altijd voor Leo/CC om uit te voeren

---

# DEEL 4: SHARED KNOWLEDGE PROTOCOL

---

## SECTIE K: NAAMCONVENTIE

### Document Naming Pattern

```
[TYPE]_[BRON]_[BESCHRIJVING]_[DATUM].md

BRON codes:
- CC    â†’ Claude Code gemaakt
- CAI   â†’ Claude AI gemaakt  
- LEO   â†’ Leo gemaakt
- MERGED â†’ Geconsolideerd door Leo
```

### Per Document Type

| Type | Pattern | Locatie |
|------|---------|---------|
| Skills | `SKILL_##_[NAAM].md` | `docs/skills/` |
| Status CC | `STATUS_CC_CURRENT.md` | `docs/status/` |
| Status CAI | `STATUS_CAI_CURRENT.md` | `docs/status/` |
| Status SSOT | `STATUS_MERGED_CURRENT.md` | `docs/status/` |
| Actions | `NEXT_ACTIONS.md` | `docs/status/` |
| Sessie CC | `SESSIE_CC_[YYYYMMDD].md` | `docs/sessions/` |
| ADR | `ADR_###_[TITEL].md` | `docs/decisions/` |

### Voorbeelden

```
STATUS_CC_CURRENT.md              â†’ CC's huidige status
STATUS_CAI_CURRENT.md             â†’ CAI's huidige status
STATUS_MERGED_CURRENT.md          â†’ SSOT (Leo's merged versie)
SESSIE_CC_20260107.md             â†’ CC sessie samenvatting
ADR_001_TENNET_BYO_KEY.md         â†’ Architecture Decision Record
```

---

## SECTIE L: DIRECTORY STRUCTUUR

### OfficiÃ«le docs/ structuur

```
/opt/github/synctacles-api/docs/
â”‚
â”œâ”€â”€ README.md                           # Index van alle documentatie
â”‚
â”œâ”€â”€ skills/                             # SKILL documenten
â”‚   â”œâ”€â”€ SKILL_00_AI_OPERATING_PROTOCOL.md
â”‚   â”œâ”€â”€ SKILL_01_HARD_RULES.md
â”‚   â”œâ”€â”€ SKILL_02_ARCHITECTURE.md
â”‚   â”œâ”€â”€ SKILL_03_CODING_STANDARDS.md
â”‚   â”œâ”€â”€ SKILL_04_PRODUCT_REQUIREMENTS.md
â”‚   â”œâ”€â”€ SKILL_05_COMMUNICATION_RULES.md
â”‚   â”œâ”€â”€ SKILL_06_DATA_SOURCES.md
â”‚   â”œâ”€â”€ SKILL_08_HARDWARE_PROFILE.md
â”‚   â”œâ”€â”€ SKILL_09_INSTALLER_SPECS.md
â”‚   â”œâ”€â”€ SKILL_10_DEPLOYMENT_WORKFLOW.md
â”‚   â”œâ”€â”€ SKILL_11_REPO_AND_ACCOUNTS.md
â”‚   â”œâ”€â”€ SKILL_12_BRAND_FREE_ARCHITECTURE.md
â”‚   â””â”€â”€ SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md
â”‚
â”œâ”€â”€ status/                             # Live state files
â”‚   â”œâ”€â”€ STATUS_MERGED_CURRENT.md        # SSOT (Leo's versie)
â”‚   â”œâ”€â”€ STATUS_CC_CURRENT.md            # CC's laatste status
â”‚   â”œâ”€â”€ STATUS_CAI_CURRENT.md           # CAI's laatste status
â”‚   â””â”€â”€ NEXT_ACTIONS.md                 # Geprioriteerde backlog
â”‚
â”œâ”€â”€ sessions/                           # Sessie samenvattingen (CC only)
â”‚   â”œâ”€â”€ README.md                       # Index + instructies
â”‚   â”œâ”€â”€ SESSIE_CC_[YYYYMMDD].md
â”‚   â””â”€â”€ archive/                        # Oudere sessies (>30 dagen)
â”‚
â”œâ”€â”€ decisions/                          # Architecture Decision Records
â”‚   â”œâ”€â”€ README.md                       # ADR index + nummering
â”‚   â””â”€â”€ ADR_###_[TITEL].md
â”‚
â”œâ”€â”€ templates/                          # Reusable templates
â”‚   â”œâ”€â”€ TEMPLATE_STATUS_CC.md
â”‚   â”œâ”€â”€ TEMPLATE_STATUS_CAI.md
â”‚   â”œâ”€â”€ TEMPLATE_SESSIE.md
â”‚   â”œâ”€â”€ TEMPLATE_HANDOFF_CAI_CC.md
â”‚   â”œâ”€â”€ TEMPLATE_HANDOFF_CC_CAI.md
â”‚   â””â”€â”€ TEMPLATE_ADR.md
â”‚
â”œâ”€â”€ CC_communication/                   # CC specifieke communicatie
â”œâ”€â”€ operations/                         # Operationele docs
â”œâ”€â”€ tasks/                              # Taak tracking
â”œâ”€â”€ reports/                            # Rapporten
â”œâ”€â”€ incidents/                          # Incident logs
â”œâ”€â”€ api/                                # API specifieke docs
â”œâ”€â”€ archived/                           # Deprecated docs
â”‚
â”œâ”€â”€ ARCHITECTURE.md                     # Systeem architectuur
â”œâ”€â”€ api-reference.md                    # API documentatie
â”œâ”€â”€ troubleshooting.md                  # Troubleshooting guide
â”œâ”€â”€ user-guide.md                       # Gebruikershandleiding
â””â”€â”€ SYSTEMD_SERVICES_ANALYSIS.md        # Service analyse
```

### Waar hoort wat?

| Content Type | Locatie |
|--------------|---------|
| Regels, procedures, standaarden | `docs/skills/` |
| Huidige project staat | `docs/status/` |
| Sessie verslagen | `docs/sessions/` |
| Architectuur beslissingen | `docs/decisions/` |
| Reusable templates | `docs/templates/` |
| API documentatie | `docs/api/` of root |
| CC specifieke zaken | `docs/CC_communication/` |
| Operationele zaken | `docs/operations/` |
| Oude/deprecated docs | `docs/archived/` |

---

## SECTIE M: DUAL STATUS MODEL

### Principe

```
â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
â”‚                    LEO (Owner)                       â”‚
â”‚              STATUS_MERGED_CURRENT.md                â”‚
â”‚                 (Single Source of Truth)             â”‚
â”‚                    â–²      â–²                          â”‚
â”‚          â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”˜      â””â”€â”€â”€â”€â”€â”€â”€â”€â”                  â”‚
â”‚          â”‚                        â”‚                  â”‚
â”‚  STATUS_CC_CURRENT.md    STATUS_CAI_CURRENT.md      â”‚
â”‚  â”œâ”€ Server state          â”œâ”€ Project context         â”‚
â”‚  â”œâ”€ Code changes          â”œâ”€ Architectural state     â”‚
â”‚  â”œâ”€ Git status            â”œâ”€ Planning status         â”‚
â”‚  â”œâ”€ Service health        â”œâ”€ Open beslissingen       â”‚
â”‚  â””â”€ Runtime issues        â””â”€ Dependencies            â”‚
â”‚                                                      â”‚
â”‚     Claude Code               Claude AI              â”‚
â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
```

### Workflow

1. **Start sessie:** Lees `STATUS_MERGED_CURRENT.md` (SSOT)
2. **Tijdens sessie:** Houd eigen status bij
3. **Einde sessie:** Update eigen `STATUS_[CC|CAI]_CURRENT.md`
4. **Leo merged:** Combineert tot nieuwe `STATUS_MERGED_CURRENT.md`
5. **Volgende sessie:** Lees nieuwe SSOT

### CC Status bevat:

```markdown
## STATUS_CC_CURRENT.md

### Server State
- Services: [running/failed/unknown]
- Disk: [usage]
- Last deploy: [timestamp]

### Code Changes (uncommitted)
- [ ] file1.py - [beschrijving]
- [ ] file2.md - [beschrijving]

### Git Status
- Branch: main
- Last commit: [hash] [message]
- Uncommitted: [yes/no]

### Open Issues
- [ ] Issue 1
- [ ] Issue 2

### Blocked By
- [dependencies]

### Last Updated
[timestamp] by CC
```

### CAI Status bevat:

```markdown
## STATUS_CAI_CURRENT.md

### Project Phase
- Current: [phase/sprint]
- Next milestone: [date/description]

### Architectural State
- Open decisions: [list]
- Recent ADRs: [list]

### Planning Status
- Sprint: [naam]
- Progress: [X/Y taken]

### Documentation State
- Updates needed: [list]
- Reviews pending: [list]

### Open Questions for Leo
- [ ] Question 1
- [ ] Question 2

### Blocked By
- [dependencies]

### Last Updated
[timestamp] by CAI
```

---

## SECTIE N: HANDOFF PROTOCOL

### Handoff Opslag

**Locatie:** `docs/handoffs/`

**Naamconventie:** `HANDOFF_[BRON]_[DOEL]_YYYYMMDD_[topic].md`

**Voorbeelden:**
- `HANDOFF_CAI_CC_20260108_p1_audit_fixes.md`
- `HANDOFF_CC_CAI_20260108_logging_review.md`

**Templates:** `docs/templates/TEMPLATE_HANDOFF_[CAI|CC]_[CC|CAI].md`

### Wanneer Handoff VERPLICHT

| Situatie | Handoff nodig? |
|----------|----------------|
| Sessie-einde met onafgerond werk | âœ… JA |
| Taak overdracht CC â†” CAI | âœ… JA |
| Volledige taak afgerond, geen follow-up | âŒ NEE |
| Mini-taak < 5 min zonder context | âŒ NEE |

### CC â†’ CAI Handoff

**Trigger:** CC klaar met taak, CAI input nodig (review, planning, docs)

**Template:** `docs/templates/TEMPLATE_HANDOFF_CC_CAI.md`

**CC levert in `docs/handoffs/HANDOFF_CC_CAI_YYYYMMDD_[topic].md`:**
```markdown
## HANDOFF: CC â†’ CAI

### PRE-HANDOFF CHECKLIST
- [ ] Alle wijzigingen gecommit
- [ ] Services stabiel
- [ ] Geen blocking errors

### Completed Work
- [wat is gedaan]
- [welke files gewijzigd]

### Current State
- [server status]
- [git status]

### Needs from CAI
- [ ] Review van [X]
- [ ] Documentatie update voor [Y]
- [ ] Planning advies voor [Z]

### Context
- [relevante achtergrond]

### Files to Review
- path/to/file1.py
- path/to/file2.md

### POST-HANDOFF VERIFICATIE
CAI bevestigt: [ ] Ontvangen en begrepen
```

### CAI â†’ CC Handoff

**Trigger:** CAI klaar met planning/docs, CC executie nodig

**Template:** `docs/templates/TEMPLATE_HANDOFF_CAI_CC.md`

**CAI levert in `docs/handoffs/HANDOFF_CAI_CC_YYYYMMDD_[topic].md`:**
```markdown
## HANDOFF: CAI â†’ CC

### PRE-HANDOFF CHECKLIST
- [ ] Taak is concreet en uitvoerbaar
- [ ] Acceptance criteria gedefinieerd
- [ ] Relevante SKILLs geÃ¯dentificeerd
- [ ] Out of scope duidelijk

### Task Description
- [wat moet CC doen]
- [verwachte output]

### Specifications
- [technische details]
- [acceptance criteria]

### Files to Create/Modify
- [ ] path/to/file1.py - [instructies]
- [ ] path/to/file2.md - [instructies]

### Relevant SKILLs
- SKILL_XX voor [aspect]
- SKILL_YY voor [aspect]

### Out of Scope
- [wat NIET doen]
- [off-limits gebieden]

### Verification
- [ ] Test 1
- [ ] Test 2

### POST-HANDOFF VERIFICATIE
CC bevestigt: [ ] Ontvangen, begrepen, kan starten
```

### Leo â†’ AI Handoff

**Leo specificeert:**
- Welke AI (CC of CAI)
- Taak beschrijving
- Prioriteit
- Deadline (indien van toepassing)
- Go/No-go voor uitvoering

### Handoff Archivering

**Retentie:** Handoffs ouder dan 30 dagen â†’ `docs/archived/handoffs/`

**Cleanup:** Maandelijks door CC bij sessie-start

---

*Sectie N versie: 2.0 (2026-01-07)*
*Reden update: Templates verplicht gesteld, enforcement toegevoegd*

---

## SECTIE N2: HUB-BASED HANDOFF LOCATIES

### VANAF 2026-01-16: Handoffs op HUB server

**Oude workflow (DEPRECATED):**
- Handoffs in GitHub repo: `/opt/github/synctacles-api/docs/handoffs/`
- CC leest vanuit repo
- Git commits nodig

**Nieuwe workflow (ACTIEF):**
- Handoffs op HUB: `/home/ccops/handoffs/`
- CC SSH't naar hub, leest handoff
- Geen Git overhead
- Geen GitHub pollution

### Directory Structuur op HUB

```
/home/ccops/
├── handoffs/
│   ├── HANDOFF_CAI_CC_YYYYMMDD_[topic].md
│   ├── HANDOFF_CC_CAI_YYYYMMDD_[topic].md
│   └── archive/                    # Completed handoffs >30 dagen
├── status/
│   ├── [server]-cleanup.txt
│   └── deployment-status.txt
└── logs/
    └── cc-activity-YYYYMMDD.log
```

### CC Workflow

**1. Leo informeert CC:**
"Handoff beschikbaar op hub: HANDOFF_CAI_CC_20260116_ssh_cleanup.md"

**2. CC leest handoff:**
```bash
# Vanaf synct-dev
ssh cc-hub
cat /home/ccops/handoffs/HANDOFF_CAI_CC_20260116_ssh_cleanup.md
```

**3. CC voert uit volgens handoff**

**4. CC schrijft completion status:**
```bash
# Op hub als ccops
cat > /home/ccops/status/ssh-cleanup-synctdev-complete.txt << 'EOF'
Completed: 2026-01-16 14:30 UTC
Task: SSH cleanup synct-dev
Status: SUCCESS
Details: [samenvatting]
EOF
```

**5. CC archiveert handoff (optional):**
```bash
mv /home/ccops/handoffs/HANDOFF_CAI_CC_20260116_ssh_cleanup.md \
   /home/ccops/handoffs/archive/
```

### CAI & CC Workflow

**SKILLs (blijven in GitHub repo):**
- CAI maakt/update SKILL docs via project knowledge
- CAI gebruikt present_files tool voor download
- Leo commit naar GitHub (synctacles-api repo)
- **CC leest SKILLs:** `/opt/github/synctacles-api/docs/skills/` op synct-dev
- Geen HUB cache nodig (CC heeft repo toegang)

**Handoffs (operationeel op HUB):**
- CAI maakt handoff document
- CAI gebruikt present_files tool
- Leo download + upload naar HUB `/home/ccops/handoffs/`
- **CC leest handoffs:** Via `ssh cc-hub` → `/home/ccops/handoffs/`

**Scheiding:**
- SKILLs = Development (GitHub repo op synct-dev)
- Handoffs = Operations (HUB server)

### Permissions op HUB

```bash
# Setup (eenmalig door Leo)
sudo -u ccops mkdir -p /home/ccops/{handoffs,status,logs,handoffs/archive}
sudo -u ccops chmod 700 /home/ccops/{handoffs,status,logs}
```

### File Naming (unchanged)

```
HANDOFF_[BRON]_[DOEL]_YYYYMMDD_[topic].md

Examples:
- HANDOFF_CAI_CC_20260116_ssh_cleanup_synctdev.md
- HANDOFF_CC_CAI_20260116_coefficient_deployment_complete.md
```

### CRITICAL: Waar CC leest

**SKILLs (op synct-dev):**
```
✅ CORRECT:
cat /opt/github/synctacles-api/docs/skills/SKILL_00.md
grep "TenneT" /opt/github/synctacles-api/docs/skills/SKILL_02.md
```

**Handoffs (op HUB):**
```
❌ FOUT:
cat /opt/github/synctacles-api/docs/handoffs/HANDOFF_*.md

✅ CORRECT:
ssh cc-hub
cat /home/ccops/handoffs/HANDOFF_*.md
```

**Waarom deze scheiding:**
- CC draait op synct-dev → repo direct beschikbaar
- SKILLs = development docs → blijven in version control
- Handoffs = operationele taken → leven op operationele hub
- Geen dubbele administratie (KISS principe)

---

*Sectie N2 versie: 1.1 (2026-01-16)*
*Reden: Verduidelijkt SKILL locatie (synct-dev repo), handoffs op HUB, geen cache*

---

## SECTIE O: ADR PROTOCOL

### Wanneer ADR maken?

```
âœ… ADR NODIG:
- Architectuur keuze met lange termijn impact
- Technologie selectie
- Data model beslissingen
- API design beslissingen
- Integratie patronen
- Security beslissingen

âŒ GEEN ADR:
- Bug fixes
- Kleine refactors
- Documentatie updates
- Configuratie wijzigingen
```

### ADR Template

```markdown
# ADR-XXX: [Titel]

**Status:** Proposed | Accepted | Deprecated | Superseded
**Date:** YYYY-MM-DD
**Author:** Leo | CAI | CC
**Supersedes:** ADR-YYY (indien van toepassing)

## Context

Wat is het probleem of de beslissing die genomen moet worden?
Wat is de huidige situatie?

## Decision

Wat hebben we besloten?
Concrete, uitvoerbare beslissing.

## Consequences

### Positief
- [voordelen]

### Negatief
- [nadelen]

### Risico's
- [wat kan misgaan]

## Alternatives Considered

### Optie A: [naam]
- Beschrijving
- Waarom niet gekozen

### Optie B: [naam]
- Beschrijving
- Waarom niet gekozen

## Implementation

- [ ] Stap 1
- [ ] Stap 2
- [ ] Stap 3

## References

- [links naar relevante docs]
- [SKILLs die geraakt worden]
```

### ADR Nummering

```
ADR nummering start bij volgende beschikbare nummer.
Check docs/decisions/ voor hoogste bestaande nummer.
```

### ADR Workflow

1. **CAI** stelt ADR op (Proposed)
2. **Leo** reviewt en keurt goed
3. **Status** â†’ Accepted
4. **CC** implementeert (indien nodig)
5. **Update** relevante SKILLs

---

## SECTIE P: VERANTWOORDELIJKHEDEN MATRIX

### Beslissingsboom

```
â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
â”‚ BESLISSING NODIG                                    â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚                                                     â”‚
â”‚ Architectuur/Design?  â”€â”€â”€â”€â”€â–º CAI adviseert          â”‚
â”‚                              Leo beslist            â”‚
â”‚                              CC executes            â”‚
â”‚                                                     â”‚
â”‚ Code implementatie?   â”€â”€â”€â”€â”€â–º CC doet                â”‚
â”‚                              CAI review (optioneel) â”‚
â”‚                              Leo approves           â”‚
â”‚                                                     â”‚
â”‚ Productie impact?     â”€â”€â”€â”€â”€â–º Leo ALTIJD approve     â”‚
â”‚                              Geen uitzonderingen    â”‚
â”‚                                                     â”‚
â”‚ Quick fix < 15 min?   â”€â”€â”€â”€â”€â–º CC mag uitvoeren       â”‚
â”‚                              MITS: verified +       â”‚
â”‚                              documented             â”‚
â”‚                                                     â”‚
â”‚ Scope change?         â”€â”€â”€â”€â”€â–º CAI signaleert         â”‚
â”‚                              Leo beslist            â”‚
â”‚                                                     â”‚
â”‚ Off-limits gebied?    â”€â”€â”€â”€â”€â–º STOP. Vraag Leo.       â”‚
â”‚                              Geen uitzonderingen    â”‚
â”‚                                                     â”‚
â”‚ Documentatie?         â”€â”€â”€â”€â”€â–º CAI schrijft           â”‚
â”‚                              CC commit              â”‚
â”‚                              Leo reviews            â”‚
â”‚                                                     â”‚
â”‚ Planning/Roadmap?     â”€â”€â”€â”€â”€â–º CAI stelt voor         â”‚
â”‚                              Leo beslist            â”‚
â”‚                                                     â”‚
â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
```

### RACI Matrix

| Activiteit | Leo | CAI | CC |
|------------|-----|-----|-----|
| Architectuur beslissingen | A | R | I |
| Code schrijven | A | C | R |
| Code review | A | R | I |
| Git commits | A | I | R |
| Service restarts | A | I | R |
| SKILL updates | A | R | C |
| ADR schrijven | A | R | C |
| Planning | A | R | I |
| Prioriteiten | R/A | C | I |
| Documentatie | A | R | C |
| Troubleshooting | A | C | R |

**R** = Responsible (doet het werk)
**A** = Accountable (eindverantwoordelijk)
**C** = Consulted (input gevraagd)
**I** = Informed (op de hoogte gehouden)

---

## SECTIE Q: SESSIE SAMENVATTING TEMPLATE

### Voor significante sessies (>1 uur of belangrijke wijzigingen)

```markdown
# SESSIE SAMENVATTING

**Datum:** YYYY-MM-DD
**Bron:** CC | CAI
**Duur:** X uur
**Focus:** [hoofdonderwerp]

---

## UITGEVOERDE WERK

### Completed
- [x] Taak 1 - [beschrijving]
- [x] Taak 2 - [beschrijving]

### Gewijzigde Files
| File | Actie | Beschrijving |
|------|-------|--------------|
| path/file.py | Modified | [wat] |
| path/new.md | Created | [wat] |

### Git Commits
- `abc1234` - [message]
- `def5678` - [message]

---

## BESLISSINGEN

| Beslissing | Rationale | ADR? |
|------------|-----------|------|
| [beslissing] | [waarom] | Nee / ADR-XXX |

---

## OPEN ITEMS

### Blocked
- [ ] [item] - blocked by [wat]

### TODO (volgende sessie)
- [ ] [item]
- [ ] [item]

### Vragen voor Leo
- [ ] [vraag]

---

## HANDOFF NOTES

### Voor CC (indien CAI sessie)
- [instructies voor CC]

### Voor CAI (indien CC sessie)
- [input nodig van CAI]

---

## STATUS UPDATE

[Korte update voor STATUS_[CC|CAI]_CURRENT.md]
```

---

# DEEL 5: QUICK REFERENCE

---

## SECTIE R: QUICK REFERENCE CARD

```
â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
â”‚  AI OPERATING PROTOCOL - QUICK REFERENCE            â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚                                                     â”‚
â”‚  1. LEES SKILLS EERST (00, 01, 02, 11 minimum)     â”‚
â”‚  2. BEWIJS DAT JE ZE GELEZEN HEBT                  â”‚
â”‚  3. LEES STATUS_MERGED_CURRENT.md                   â”‚
â”‚  4. PROTECT MODE = DEFAULT                          â”‚
â”‚  5. VERIFICATIE VOOR CONCLUSIES                     â”‚
â”‚  6. VRAAG VOOR ACTIES                               â”‚
â”‚  7. FAILED â‰  MOET GEREPAREERD                       â”‚
â”‚  8. TENNET = OFF-LIMITS                             â”‚
â”‚  9. GIT = ALTIJD ALS SERVICE USER (CC)              â”‚
â”‚ 10. CHOWN DIRECT NA FILE EDITS (CC)                 â”‚
â”‚ 11. BIJ TWIJFEL: STOP EN VRAAG                      â”‚
â”‚ 12. GEEN "IK FIX EVEN"                              â”‚
â”‚ 13. UPDATE STATUS BIJ SESSIE EINDE                  â”‚
â”‚                                                     â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  NAAMCONVENTIE                                      â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  STATUS_CC_CURRENT.md    â†’ CC's status              â”‚
â”‚  STATUS_CAI_CURRENT.md   â†’ CAI's status             â”‚
â”‚  STATUS_MERGED_CURRENT.md â†’ SSOT (Leo)              â”‚
â”‚  SESSIE_[CC|CAI]_YYYYMMDD.md â†’ Sessie log          â”‚
â”‚  ADR_###_[TITEL].md      â†’ Decision record          â”‚
â”‚                                                     â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  DIRECTORY STRUCTUUR                                â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  docs/skills/    â†’ SKILL documenten                 â”‚
â”‚  docs/status/    â†’ Live state files                 â”‚
â”‚  docs/sessions/  â†’ Sessie samenvattingen            â”‚
â”‚  docs/decisions/ â†’ ADRs                             â”‚
â”‚  docs/templates/ â†’ Reusable templates               â”‚
â”‚                                                     â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  HANDOFF                                            â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  CC â†’ CAI: Completed work + needs from CAI          â”‚
â”‚  CAI â†’ CC: Task specs + files to modify             â”‚
â”‚  Both: Update own STATUS file                       â”‚
â”‚  Leo: Merge to SSOT                                 â”‚
â”‚                                                     â”‚
â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
```

---

## SECTIE S: GERELATEERDE SKILLS

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

## SECTIE T: CONSEQUENCES

### Bij overtreding van dit protocol:

1. **Leo stopt de sessie**
2. **Wijzigingen worden teruggedraaid**
3. **Tijd is verspild**

### De 4.5-uur incident bewees:

- AI las SKILLs niet â†’ TenneT rabbit hole (45 min verspild)
- AI verifieerde niet â†’ Verkeerde script geblamed (60 min verspild)
- AI vroeg niet â†’ Productie bijna gesloopt
- AI nam aan â†’ 3.5 uur verspild van 4.5 uur totaal

**Dit protocol bestaat om herhaling te voorkomen.**

---

**LAATSTE WAARSCHUWING:**

Dit protocol is geen suggestie. Het is een vereiste.
Elke sessie begint met bewijs dat je dit gelezen hebt.
Geen uitzonderingen.

---

**Document Owner:** Leo
**Enforcement:** Strict
**Version:** 2.2
**Last Updated:** 2026-01-16 (Added H3: HA Deployment Access)
