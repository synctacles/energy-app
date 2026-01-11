# RESEARCH OPDRACHT: EU Consumer Electricity Price Data

**Voor:** ChatGPT + Claude Code  
**Van:** Claude (Opus) + Leo  
**Datum:** 2026-01-11  
**Doel:** Identificeer en valideer bronnen voor consumer electricity prices in EU-landen om coefficient lookup tables te bouwen

---

## TAAKVERDELING

| AI | Focus | Acties |
|----|-------|--------|
| **ChatGPT** | Research & interpretatie | Bronnen identificeren, rapporten lezen, regulators doorzoeken |
| **Claude Code** | Technische validatie | API's testen (`curl`), CSV samples downloaden, data structuur valideren |

Beide AI's leveren resultaten in hetzelfde format. Consolidatie gebeurt daarna.

---

## CONTEXT: HOE DEZE OPDRACHT TOT STAND KWAM

### Het Product: SYNCTACLES

SYNCTACLES is een energie data platform voor Home Assistant gebruikers. De "killer feature" is **Energy Action** — een simpel signaal (USE/WAIT/AVOID) dat gebruikers vertelt wanneer ze energie moeten gebruiken gebaseerd op de werkelijke consumer prijs.

### Het Probleem

ENTSO-E (Europese energie data) levert alleen **wholesale prijzen** (€/MWh). Maar consumenten betalen een **consumer prijs** die bestaat uit:

```
consumer_prijs = wholesale_prijs + netkosten + belastingen + leveranciersmarge
```

Energy Action gebaseerd op alleen wholesale prijzen geeft verkeerde aanbevelingen omdat de verhouding wholesale/consumer varieert per uur, dag en seizoen.

### De Huidige Oplossing (Nederland)

In Nederland gebruiken we **Enever.nl** — een gratis API die real-time consumer prijzen levert voor alle Nederlandse leveranciers. Hiermee berekenen we een **coefficient**:

```
coefficient = consumer_prijs / wholesale_prijs
```

Na analyse van 27.895 uur data (nov 2022 - jan 2026) bleek een **additief model** beter te werken:

```python
# Finaal model (89% accuraat voor ranking)
consumer_prijs = wholesale_prijs + HOURLY_OFFSET[uur]

HOURLY_OFFSET = {
    0: 0.1934, 1: 0.1903, 2: 0.1879, 3: 0.1819,
    4: 0.1705, 5: 0.1667, 6: 0.1789, 7: 0.1989,
    8: 0.2132, 9: 0.2099, 10: 0.2030, 11: 0.1968,
    12: 0.1899, 13: 0.1768, 14: 0.1669, 15: 0.1599,
    16: 0.1508, 17: 0.1571, 18: 0.1723, 19: 0.2009,
    20: 0.2085, 21: 0.2050, 22: 0.2006, 23: 0.1945
}
```

**Resultaat:** Met alleen ENTSO-E data + deze 24 getallen pakken we 89% van de maximale besparing.

### De Uitdaging: Schalen naar EU

Enever bestaat alleen in Nederland. Voor andere EU-landen moeten we:

1. **Ideaal:** Vergelijkbare live API's vinden (zoals aWATTar in DE/AT)
2. **Fallback:** Historische consumer prijzen verzamelen → lookup table maken

---

## WAT WE ZOEKEN (in volgorde van prioriteit)

### Prioriteit 1: Live Consumer Price API's
Net zoals Enever in Nederland — real-time consumer prijzen per uur.

**Voordelen:**
- Coefficient kan continu geüpdatet worden
- Vangt tariefwijzigingen en seizoenseffecten op
- Hoogste accuraatheid (100% vs ~89% voor lookup)

**Per API documenteer:**
- Endpoint URL
- Authenticatie (geen / API key / OAuth)
- Rate limits
- Response format (JSON/XML)
- Gratis of betaald
- Geo-restricties (IP check?)

**Claude Code:** Test elke gevonden API met `curl` en lever sample response.

### Prioriteit 2: Historische Consumer Price Data
Als fallback wanneer geen live API beschikbaar is.

**Per bron documenteer:**
- Download URL
- Data formaat (CSV/Excel/PDF)
- Periode beschikbaar
- Granulariteit (uur/dag/maand)
- Inclusief belastingen/netkosten?

**Claude Code:** Download sample files en valideer structuur.

---

## DETAILS PER PRIORITEIT

### Voor Live API's (Prioriteit 1):

1. **Endpoint informatie**
   - Base URL
   - Authenticatie methode
   - Voorbeeld request

2. **Data die geleverd wordt**
   - Consumer prijs (€/kWh of €/MWh)
   - Tijdresolutie (15-min / uur)
   - Vooruit beschikbaar (day-ahead?)

3. **Beperkingen**
   - Rate limits
   - Geo-restricties
   - Licentievoorwaarden

### Voor Historische Data (Prioriteit 2):

1. **Historische consumer electricity prices** (€/kWh of €/MWh)
   - Minimaal 12 maanden data (liefst 24+)
   - Uurresolutie (of daggemiddelde als uur niet beschikbaar)
   - Huishoudtarief (niet industrie)

2. **Bronvermelding**
   - Naam van de bron (regulator, leverancier, overheid, onderzoeksinstituut)
   - URL waar data te vinden is
   - Type data (CSV download, API, PDF rapport, interactieve tool)
   - Gratis of betaald

3. **Data kwaliteit indicatie**
   - Volledigheid (hoeveel maanden/jaren)
   - Granulariteit (uur/dag/maand)
   - Inclusief belastingen? Netkosten?

**Claude Code:** Download minimaal 1 sample bestand per bruikbare bron.

---

## PRIORITEIT LANDEN

### Tier 1 (Eerste focus - grote HA markten)
| Land | Code | Geschatte HA users | Prioriteit |
|------|------|-------------------|------------|
| Duitsland | DE | 60.000+ | CRITICAL |
| Frankrijk | FR | 20.000+ | HIGH |
| België | BE | 8.000+ | HIGH |
| Oostenrijk | AT | 5.000+ | HIGH |

### Tier 2 (Secundair)
| Land | Code | Prioriteit |
|------|------|------------|
| Spanje | ES | MEDIUM |
| Italië | IT | MEDIUM |
| Polen | PL | MEDIUM |
| Denemarken | DK | MEDIUM |
| Zweden | SE | MEDIUM |
| Noorwegen | NO | MEDIUM |

### Tier 3 (Later)
Overige EU-27 landen

---

## BEKENDE BRONNEN (Startpunt)

### Prijsfeed API's (real-time, mogelijk ook historisch)
| Land | Bron | URL | Status |
|------|------|-----|--------|
| DE/AT | aWATTar | https://www.awattar.de/services/api | Gratis, fair use |
| DE | SMARD.de | https://www.smard.de/ | Overheid, gratis |
| DE | Energy-Charts (Fraunhofer) | https://energy-charts.info/ | Gratis |
| EU | ENTSO-E | https://transparency.entsoe.eu/ | Wholesale only |

### Regulators (vaak historische rapporten)
| Land | Regulator | URL |
|------|-----------|-----|
| DE | Bundesnetzagentur | https://www.bundesnetzagentur.de/ |
| AT | E-Control | https://www.e-control.at/ |
| FR | CRE | https://www.cre.fr/ |
| BE | CREG (federaal) | https://www.creg.be/ |
| BE | VREG (Vlaanderen) | https://www.vreg.be/ |
| ES | CNMC | https://www.cnmc.es/ |

### Statistische bronnen
| Bron | URL | Type |
|------|-----|------|
| Eurostat | https://ec.europa.eu/eurostat/ | Halfjaarlijks, alle EU |
| ACER | https://www.acer.europa.eu/ | Marktmonitoring |

---

## GEWENST OUTPUT FORMAAT

### Per land, lever:

```markdown
## [LAND] ([CODE])

### Bron 1: [Naam]
- **URL:** [link]
- **Type:** [CSV/API/PDF/Tool]
- **Gratis:** Ja/Nee
- **Data beschikbaar:** [periode, bv. 2020-2025]
- **Granulariteit:** [uur/dag/maand]
- **Inclusief:** [belastingen/netkosten/leveranciersmarge]
- **Download instructies:** [hoe te verkrijgen]
- **Notities:** [beperkingen, licentie, etc.]

### Bron 2: [Naam]
...
```

### Samenvattingstabel

```markdown
| Land | Beste bron | Type | Periode | Granulariteit | Gratis |
|------|------------|------|---------|---------------|--------|
| DE   | SMARD.de   | CSV  | 2015-now | uur         | Ja     |
| FR   | ...        | ...  | ...     | ...           | ...    |
```

---

## SPECIFIEKE VRAGEN

1. **Duitsland:** SMARD.de heeft wholesale data — hebben ze ook consumer/retail prijzen? Zo niet, welke bron heeft historische retail prijzen?

2. **België:** VREG heeft tariefvergelijkers — exporteren die historische data of alleen actuele tarieven?

3. **Frankrijk:** EDF publiceert tarieven — zijn historische versies beschikbaar? Is er een open data portaal?

4. **Oostenrijk:** E-Control heeft een tarifkalkulator — kan je historische tarieven extraheren?

5. **Overige landen:** Zijn er pan-Europese datasets (Eurostat, ACER) met voldoende granulariteit (minimaal dagelijks)?

---

## WAT WE NIET NODIG HEBBEN

- Individuele leverancierstarieven (gemiddelde consumer prijs volstaat)
- Industrietarieven (alleen huishouden)
- Verbruiksdata (alleen prijzen)
- Wholesale-only bronnen (ENTSO-E hebben we al)

---

## CONTEXT VOOR INTERPRETATIE

### Waarom dit werkt

De relatie tussen wholesale en consumer prijs is **redelijk stabiel per uur van de dag**. Netkosten en belastingen zijn vaak vast, de leveranciersmarge varieert maar middelt uit. 

Door historische data te analyseren kunnen we per land een lookup table maken zoals we voor Nederland hebben gedaan. Zelfs met 80% accuraatheid (vs 89% NL) is Energy Action waardevol.

### Acceptabele kwaliteit

| Granulariteit | Acceptabel |
|---------------|------------|
| Uurlijks | ✅ Ideaal |
| Dagelijks | ✅ Goed |
| Maandelijks | ⚠️ Bruikbaar voor grove estimate |
| Jaarlijks | ❌ Onvoldoende |

| Periode | Acceptabel |
|---------|------------|
| 24+ maanden | ✅ Ideaal |
| 12-24 maanden | ✅ Voldoende |
| 6-12 maanden | ⚠️ Marginaal |
| < 6 maanden | ❌ Onvoldoende |

---

## DELIVERABLE

### ChatGPT levert:
1. Per Tier 1 land: minimaal 2 bronnen onderzocht
2. Per Tier 2 land: minimaal 1 bron onderzocht  
3. Samenvattingstabel van alle gevonden bronnen
4. Aanbeveling welke bronnen prioriteit hebben
5. Eventuele showstoppers of landen waar geen bruikbare data lijkt te bestaan

### Claude Code levert:
1. Per gevonden API: `curl` test + sample response
2. Per gevonden CSV/download: sample bestand + structuur analyse
3. Validatie: welke bronnen technisch bruikbaar zijn
4. Eventuele geo-restricties of auth-blokkades geïdentificeerd

### Gecombineerd eindresultaat:

```markdown
| Land | Beste bron | Type | Live API? | Getest? | Bruikbaar? |
|------|------------|------|-----------|---------|------------|
| DE   | aWATTar    | API  | Ja        | ✅      | ✅         |
| FR   | ...        | CSV  | Nee       | ✅      | ⚠️         |
```

---

## VERVOLGSTAP

Na deze research (beide AI's):
1. Leo consolideert resultaten van ChatGPT + CC
2. Per land met live API: coefficient-engine uitbreiden (zoals NL met Enever)
3. Per land zonder live API: historische data downloaden → analyse → lookup table genereren
4. Multi-country support in coefficient-engine implementeren

---

*Einde research opdracht*
