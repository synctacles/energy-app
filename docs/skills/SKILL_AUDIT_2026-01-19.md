# SKILL AUDIT

Audit van alle SKILL bestanden - Status en aanbevelingen
**Datum: 2026-01-19**

---

## SAMENVATTING

| SKILL | Versie | Status | Actie |
|-------|--------|--------|-------|
| SKILL_01_HARD_RULES | 1.0 | Actueel | Geen actie |
| SKILL_02_ARCHITECTURE | 3.0 | **Verouderd** | **Vervangen door v4.0** |
| SKILL_03_CODING_STANDARDS | - | Actueel | Geen actie |
| SKILL_04_PRODUCT_REQUIREMENTS | 2.0 | **Verouderd** | **Vervangen door v3.0** |
| SKILL_05_COMMUNICATION_RULES | - | Actueel | Geen actie |
| SKILL_06_DATA_SOURCES | 2.0 | **Gedeeltelijk verouderd** | Update nodig |
| SKILL_07_PERSONAL_PROFILE | - | N/A | Persoonlijk profiel |
| SKILL_08_HARDWARE_PROFILE | - | N/A | Hardware specs |
| SKILL_09_INSTALLER_SPECS | - | Actueel | Geen actie |
| SKILL_10_DEPLOYMENT_WORKFLOW | - | Actueel | Geen actie |
| SKILL_10_COEFFICIENT_VPN | - | **Verouderd** | **Coefficient server niet meer nodig** |
| SKILL_11_REPO_AND_ACCOUNTS | - | Actueel | Geen actie |
| SKILL_12_BRAND_FREE_ARCHITECTURE | - | Actueel | Geen actie |
| SKILL_13_LOGGING_DIAGNOSTICS | - | Actueel | Geen actie |
| SKILL_14_COEFFICIENT_BUSINESS_MODEL | - | **Verouderd** | Coefficient server vervangen door static offsets |
| SKILL_14_MONITORING | - | Actueel | Geen actie |
| SKILL_15_CONSUMER_PRICE_ENGINE | 2.1 | **Gedeeltelijk verouderd** | KISS Stack vervangt coefficient server |
| SKILL_16_BACKUP_RECOVERY | - | Actueel | Geen actie |

---

## DETAILANALYSE

### SKILL_01_HARD_RULES.md

**Versie:** 1.0 (2025-12-30)
**Status:** ✅ ACTUEEL

**Bevindingen:**
- Alle regels zijn nog van toepassing
- KISS, fail-fast, brand-free principes correct
- Gecentraliseerde database configuratie (Rule 11) correct

**Actie:** Geen

---

### SKILL_02_ARCHITECTURE.md

**Versie:** 3.0 (2026-01-11)
**Status:** ❌ VEROUDERD

**Verouderde informatie:**
1. Mist `/api/v1/dashboard` endpoint (bundled data)
2. Mist `/api/v1/best-window` endpoint
3. Mist `/api/v1/tomorrow` endpoint
4. Refereert naar TenneT integration (discontinued)
5. Mist P1/Live Cost sensor architectuur
6. Mist Savings sensor
7. Vermeldt 19 Enever providers (nu 24)
8. Mist 6-tier fallback stack (KISS Migration)
9. Refereert naar coefficient server (vervangen door static offsets)
10. Coordinator details incorrect (geen apart coordinator.py)

**Actie:** **VERVANGEN door SKILL_02_ARCHITECTURE_v4.md**

---

### SKILL_03_CODING_STANDARDS.md

**Status:** ✅ ACTUEEL

**Bevindingen:**
- Code standards zijn generiek en blijven geldig
- Python conventies correct

**Actie:** Geen

---

### SKILL_04_PRODUCT_REQUIREMENTS.md

**Versie:** 2.0 (2026-01-11)
**Status:** ❌ VEROUDERD

**Verouderde informatie:**
1. Mist Live Cost sensor (state, attributes, calculation)
2. Mist Savings sensor (state, attributes, RestoreEntity)
3. Mist Best Window sensor
4. Mist Tomorrow Preview sensor
5. Entity IDs pattern incorrect (oude naamgeving)
6. Config flow mist power sensor step
7. Mist options flow threshold configuratie
8. Automation examples verouderd
9. Versienummer incorrect (toont oude structuur)
10. Vermeldt 19 providers (nu 24)

**Actie:** **VERVANGEN door SKILL_04_PRODUCT_REQUIREMENTS_v3.md**

---

### SKILL_05_COMMUNICATION_RULES.md

**Status:** ✅ ACTUEEL

**Bevindingen:**
- Communicatie richtlijnen blijven geldig

**Actie:** Geen

---

### SKILL_06_DATA_SOURCES.md

**Versie:** 2.0 (2026-01-11)
**Status:** ⚠️ GEDEELTELIJK VEROUDERD

**Verouderde informatie:**
1. Vermeldt TenneT (discontinued)
2. 5-tier fallback incorrect (nu 6-tier KISS Stack)
3. Mist Frank DB als Tier 1
4. Mist EasyEnergy als Tier 3
5. Mist static offsets uitleg
6. A75 fallback collector niet meer nodig (Energy Action focus)
7. Vermeldt 19 Enever providers (nu 24)

**Correcte informatie:**
- ENTSO-E A44 documentatie correct
- Energy-Charts documentatie correct
- Frank API documentatie correct
- Rate limits correct

**Actie:** Update nodig - nieuwe fallback stack documenteren

---

### SKILL_07_PERSONAL_PROFILE.md

**Status:** N/A (Persoonlijk profiel Leo)

**Actie:** Geen

---

### SKILL_08_HARDWARE_PROFILE.md

**Status:** N/A (Server hardware specs)

**Actie:** Geen

---

### SKILL_09_INSTALLER_SPECS.md

**Status:** ✅ ACTUEEL

**Bevindingen:**
- FASE 0-6 installatie correct
- Systemd templates correct

**Actie:** Geen

---

### SKILL_10_DEPLOYMENT_WORKFLOW.md

**Status:** ✅ ACTUEEL

**Bevindingen:**
- 6-fase deployment correct
- SCP/SSH procedures correct

**Actie:** Geen

---

### SKILL_10_COEFFICIENT_VPN.md

**Status:** ❌ VEROUDERD

**Reden:**
- KISS Migration v2.0.0 heeft coefficient server vervangen door static offsets
- VPN configuratie niet meer nodig voor prijsberekening
- Enever access nu alleen via HA component (BYO-key)

**Actie:** Markeren als DEPRECATED of archiveren

---

### SKILL_11_REPO_AND_ACCOUNTS.md

**Status:** ✅ ACTUEEL

**Actie:** Geen

---

### SKILL_12_BRAND_FREE_ARCHITECTURE.md

**Status:** ✅ ACTUEEL

**Bevindingen:**
- Template systeem correct
- ENV-driven configuratie correct

**Actie:** Geen

---

### SKILL_13_LOGGING_DIAGNOSTICS_HA_STANDARDS.md

**Status:** ✅ ACTUEEL

**Bevindingen:**
- Logging patterns correct
- Diagnostics correct

**Actie:** Geen

---

### SKILL_14_COEFFICIENT_BUSINESS_MODEL.md

**Status:** ❌ VEROUDERD

**Reden:**
- Coefficient server vervangen door static offsets in KISS Migration
- Dual-source validation niet meer server-side
- Business model voor coefficient data niet meer relevant

**Actie:** Markeren als DEPRECATED of archiveren

---

### SKILL_14_MONITORING.md

**Status:** ✅ ACTUEEL

**Bevindingen:**
- Prometheus metrics correct
- Health checks correct

**Actie:** Geen

---

### SKILL_15_CONSUMER_PRICE_ENGINE.md

**Versie:** 2.1 (2026-01-12)
**Status:** ⚠️ GEDEELTELIJK VEROUDERD

**Verouderde informatie:**
1. Coefficient server architectuur niet meer primair
2. Dual-source validation (Frank + Enever) nu alleen voor BYO
3. VPN setup niet meer nodig
4. Hourly lookup table vervangen door static offsets

**Correcte informatie:**
- Linear regression model documentatie correct
- Frank API endpoint documentatie correct
- Accuracy metrics historisch relevant

**Actie:** Update nodig - KISS Stack (static offsets) als primaire methode documenteren

---

### SKILL_16_BACKUP_RECOVERY.md

**Status:** ✅ ACTUEEL

**Actie:** Geen

---

## AANBEVELINGEN

### Hoge Prioriteit

1. **SKILL_02_ARCHITECTURE**: Vervang v3.0 door v4.0
   - Nieuwe file: `SKILL_02_ARCHITECTURE_v4.md` ✅ AANGEMAAKT

2. **SKILL_04_PRODUCT_REQUIREMENTS**: Vervang v2.0 door v3.0
   - Nieuwe file: `SKILL_04_PRODUCT_REQUIREMENTS_v3.md` ✅ AANGEMAAKT

### Medium Prioriteit

3. **SKILL_06_DATA_SOURCES**: Update naar v3.0
   - Voeg 6-tier KISS Stack toe
   - Verwijder TenneT referenties
   - Update Enever providers (24)

4. **SKILL_15_CONSUMER_PRICE_ENGINE**: Update naar v3.0
   - Documenteer static offsets als primaire methode
   - Markeer coefficient server als legacy

### Lage Prioriteit

5. **SKILL_10_COEFFICIENT_VPN**: Archiveer
   - Verplaats naar `docs/skills/archive/`
   - Of markeer header met `DEPRECATED`

6. **SKILL_14_COEFFICIENT_BUSINESS_MODEL**: Archiveer
   - Verplaats naar `docs/skills/archive/`
   - Of markeer header met `DEPRECATED`

---

## NIEUWE SKILL BESTANDEN

De volgende bestanden zijn gegenereerd uit broncode:

1. `SKILL_02_ARCHITECTURE_v4.md` - Volledige architectuur uit code
2. `SKILL_04_PRODUCT_REQUIREMENTS_v3.md` - Volledige sensor specs uit code
3. `SKILL_AUDIT_2026-01-19.md` - Dit bestand

---

## BRONCODE GESCAND

| Repository | Bestanden |
|------------|-----------|
| synctacles-api | windows.py, prices.py, fallback_manager.py, frank_energie_client.py, easyenergy_client.py, main.py |
| ha-energy-insights-nl | __init__.py, sensor.py, config_flow.py, enever_client.py, const.py, manifest.json |

---

*Audit uitgevoerd: 2026-01-19*
*Door: Claude Code (CC)*
