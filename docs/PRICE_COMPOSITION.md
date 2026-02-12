# Price Composition & Tax Profiles

> **Last verified:** 2026-02-11
> **Applies to:** Synctacles Energy Go addon, all EU bidding zones

This document explains how Synctacles Energy calculates consumer electricity prices from wholesale market data. All rates are **verified against official government sources** and stored in YAML configuration files at `internal/countries/data/*.yaml`.

---

## How Consumer Prices Are Calculated

### Formula

```
consumer_price = (wholesale * coefficient + energy_tax + surcharges) * (1 + VAT)
```

| Component | Description |
|-----------|-------------|
| **wholesale** | Day-ahead spot price from EPEX/Nordpool exchange (EUR/kWh) |
| **coefficient** | Supplier markup factor (1.0 = pass-through, 1.05 = 5% markup) |
| **energy_tax** | Government-mandated energy tax, **always excl. VAT** |
| **surcharges** | Additional levies (grid fees, renewable surcharges), **always excl. VAT** |
| **VAT** | Value-added tax applied to the entire consumer price |

### Important: All Rates Exclude VAT

Every `energy_tax` and `surcharges` value in our YAML configs is the **government-published rate excluding VAT**. VAT is applied once at the end to the full sum. This prevents double-taxation.

### Consumer Price Sources (IsConsumer)

Some price sources (e.g., Frank Energie for NL) already return **full consumer prices** including all taxes. These are marked `IsConsumer: true` and bypass normalization entirely.

---

## Country Profiles

### Netherlands (NL)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Energy tax 2024 | 0.10880 | [Belastingdienst - Energiebelasting](https://www.belastingdienst.nl/wps/wcm/connect/bldcontentnl/belastingdienst/zakelijk/overige_belastingen/belastingen_op_milieugrondslag/energiebelasting/energiebelasting) |
| Energy tax 2025 | 0.10154 | Belastingdienst (see above) |
| Energy tax 2026 | **0.09161** | Belastingdienst (see above) |
| Surcharges | 0.0 | - |
| VAT | 21% | Rijksoverheid |
| Coefficient | 1.0 | Dynamic tariff pass-through model |

**Notes:**
- The energiebelasting (EB) includes the former ODE (Opslag Duurzame Energie) since 2025.
- Dutch consumer comparison sites commonly quote **incl. BTW** rates (e.g., 11.08 ct). Our config uses the official **excl. BTW** rate (9.161 ct).
- Verified against Frank Energie API component breakdown: `energyTaxPrice` of 11.085 ct incl. BTW / 1.21 = 9.161 ct excl. BTW.

**Price sources:** EasyEnergy (wholesale+margin), Frank Energie (consumer), Energy-Charts (wholesale)

---

### Germany (DE-LU)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Stromsteuer | 0.02050 | [Zoll.de - Steuersatz](https://www.zoll.de/DE/Fachthemen/Steuern/Verbrauchsteuern/Strom/strom.html) |
| KWKG-Umlage | 0.00446 | [Netztransparenz.de - KWKG 2026](https://www.netztransparenz.de/de-de/Erneuerbare-Energien-und-Umlagen/KWKG/KWKG-Umlage/KWKG-Umlagen-%C3%9Cbersicht/KWKG-Umlage-2026) |
| Offshore-Netzumlage | 0.00941 | [Netztransparenz.de - Offshore 2026](https://www.netztransparenz.de/de-de/Erneuerbare-Energien-und-Umlagen/Sonstige-Umlagen/Offshore-Netzumlage/Offshore-Netzumlagen-%C3%9Cbersicht/Offshore-Netzumlage-2026) |
| S19 StromNEV-Umlage | 0.01559 | [Bundesnetzagentur](https://www.bundesnetzagentur.de/SharedDocs/A_Z_Glossar/P/Par19_StromNEV_Umlage.html) |
| **Total surcharges** | **0.02946** | Sum of above 3 Umlagen |
| VAT (MwSt) | 19% | Bundesfinanzministerium |
| Coefficient | 1.0 | Tibber/aWATTar pass-through model |

**Notes:**
- All Umlagen rates are published annually by the Bundesnetzagentur and TSOs (netto, excl. MwSt).
- The former EEG-Umlage was abolished in July 2022.
- Manufacturing businesses qualify for reduced Stromsteuer (0.05 ct/kWh, EU minimum) from 2026.

**Price sources:** aWATTar (wholesale), Energy-Charts (wholesale)

---

### France (FR)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Accise sur l'electricite 2024 | 0.02110 | [BOFIP - Accise](https://bofip.impots.gouv.fr/bofip/14903-PGP.html) |
| Accise sur l'electricite 2026 | **0.03085** | [EDF Entreprises - Taxes 2026](https://www.edf.fr/entreprises/decryptages/normes-et-reglementations/evolution-des-taxes-et-contributions-appliquees-sur-l-electricite-au-1er-fevrier-2026) |
| Surcharges | 0.0 | - |
| VAT (TVA) | 20% | Direction Generale des Douanes |
| Coefficient | 1.0 | - |

**Notes:**
- Formerly known as TICFE/CSPE, renamed to "Accise sur l'electricite" in 2022.
- Rate of 30.85 EUR/MWh applies to households and equivalent (<36 kVA) from February 2026.
- Rate is HT (hors taxes, excluding TVA 20%).

**Price sources:** Energy-Charts (wholesale)

---

### Austria (AT)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Elektrizitatsabgabe 2024 | 0.01500 | [BMF - Elektrizitatsabgabe](https://www.bmf.gv.at/themen/steuern/steuern-von-a-bis-z/elektrizitaetsabgabe.html) |
| Elektrizitatsabgabe 2026 | **0.00100** | [Parlament AT - Senkung 2026](https://www.parlament.gv.at/fachinfos/budgetdienst/Senkung-der-Elektrizitaetsabgabe-2026) |
| Surcharges | 0.0 | - |
| VAT (MwSt) | 20% | BMF |
| Coefficient | 1.0 | - |

**Notes:**
- Temporarily reduced from 1.5 ct/kWh to **0.1 ct/kWh** for private households in 2026.
- Reduction expires December 31, 2026 (unless extended).
- Saves average household ~60 EUR/year at 3,500 kWh consumption.

**Price sources:** aWATTar (wholesale), Energy-Charts (wholesale)

---

### Belgium (BE)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Bijzondere accijns 2024 | 0.04588 | [FOD Financien - Accijnzen](https://etaamb.openjustice.be/nl/wet-van-19-maart-2023_n2023030776) |
| Bijzondere accijns 2026 | **0.02360** | [Social Energie - Accijnshervorming](https://www.socialenergie.be/nl/accijnshervorming-en-6-btw-voor-elektriciteit-en-gas-2/) |
| Surcharges | 0.0 | - |
| VAT (BTW) | **6%** | FOD Financien (reduced rate for household electricity) |
| Coefficient | 1.0 | - |

**Notes:**
- Belgium applies a **reduced VAT of 6%** on household electricity (not the standard 21%).
- Accijns reformed in 2023; rate decreased for 2026 as part of gradual reform.
- Federal contribution on electricity was abolished.

**Price sources:** Energy-Charts (wholesale)

---

### Norway (NO1-NO5)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Elavgift 2024 | 0.01090 | [Skatteetaten - Electricity](https://www.skatteetaten.no/en/rates/electricity/) |
| Elavgift 2026 | **0.00620** | [Regjeringen - Avgiftssatser 2026](https://www.regjeringen.no/no/tema/okonomi-og-budsjett/skatter-og-avgifter/avgiftssatser-2026/id3121982/) |
| Surcharges | 0.0 | - |
| VAT (MVA) | 25% | Skatteetaten |
| Coefficient | 1.01 | Tibber/Fjordkraft ~1% markup |

**Notes:**
- Elavgift reduced from 12.53 ore/kWh (2024) to 7.13 ore/kWh (2026) for households.
- Converted at 11.5 NOK/EUR. Exchange rate fluctuations may cause ~5% variance.
- Energy-Charts prices are in EUR; Energi Data Service may return NOK.
- 5 bidding zones (NO1 Oslo, NO2 Kristiansand, NO3 Trondheim, NO4 Tromso, NO5 Bergen).

**Price sources:** Energi Data Service (wholesale), Energy-Charts (wholesale), spot-hinta.fi (wholesale)

---

### Sweden (SE1-SE4)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Energiskatt 2024 | 0.03820 | [Skatteverket - Punktskatter](https://www.skatteverket.se/foretag/skatterochavdrag/punktskatter/energiskatter.4.html) |
| Energiskatt 2026 | **0.03130** | [Partille Energi - Reduced Tax 2026](https://partilleenergi.se/en/2025/11/27/sankt-energiskatt-pa-el-fran-arsskiftet/) |
| Surcharges | 0.0 | - |
| VAT (moms) | 25% | Skatteverket |
| Coefficient | 1.0 | Tibber SE pass-through model |

**Notes:**
- Energiskatt reduced from 43.9 ore/kWh to 36.0 ore/kWh for 2026.
- Converted at 11.5 SEK/EUR. Exchange rate fluctuations may cause ~5% variance.
- 4 bidding zones (SE1 Lulea, SE2 Sundsvall, SE3 Stockholm, SE4 Malmo).

**Price sources:** Energi Data Service (wholesale), Energy-Charts (wholesale), spot-hinta.fi (wholesale)

---

### Denmark (DK1-DK2)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Elafgift 2024 | 0.00975 | [Inforevision - Elafgift](https://inforevision.dk/en/aktuelt/stor-lempelse-paa-elafgiften-i-2026-og-2027/) |
| Elafgift 2026 | **0.00107** | [The Local DK - Minimum Tax 2026](https://www.thelocal.dk/20251217/how-much-will-denmarks-new-minimum-electricity-tax-save-you-in-2026) |
| Surcharges | 0.0 | - |
| VAT (moms) | 25% | Skattestyrelsen |
| Coefficient | 1.0 | Barry/Vindstod pass-through model |

**Notes:**
- Elafgift **slashed to EU minimum** (0.8 ore/kWh) for 2026-2027, down from 72.7 ore/kWh. This is a 99% reduction.
- Converted at 7.46 DKK/EUR.
- 2 bidding zones (DK1 West/Jutland, DK2 East/Zealand).

**Price sources:** Energi Data Service (wholesale), Energy-Charts (wholesale)

---

### Finland (FI)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Sahkovero (tax class I) | **0.02253** | [Vero.fi - Electricity Tax Rates](https://www.vero.fi/en/businesses-and-corporations/taxes-and-charges/excise-taxation/sahkovero/Tax-rates-on-electricity-and-certain-fuels/) |
| Surcharges | 0.0 | - |
| VAT (ALV) | 25.5% | Vero.fi |
| Coefficient | 1.0 | - |

**Notes:**
- Rate consists of: energy tax 2.240 ct + strategic stockpile fee 0.013 ct = 2.253 ct/kWh.
- Finland uses EUR natively; no currency conversion needed.
- Tax class I applies to households, agriculture, public sector.

**Price sources:** spot-hinta.fi (wholesale), Energi Data Service (wholesale), Energy-Charts (wholesale)

---

### Spain (ES)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| IEE (estimated flat) | **0.00750** | [AEAT - Impuesto Electricidad](https://sede.agenciatributaria.gob.es/Sede/impuestos-especiales-medioambientales/impuesto-sobre-electricidad.html) |
| Surcharges | 0.0 | - |
| VAT (IVA) | 21% | Agencia Tributaria |
| Coefficient | 1.0 | - |

**Notes:**
- The IEE (Impuesto Especial sobre la Electricidad) is **percentage-based at 5.113%** of the subtotal, not a flat rate. The 0.75 ct/kWh is an approximation for average PVPC household consumption.
- Actual IEE varies with wholesale price: low prices ~0.6 ct, high prices ~0.9 ct.
- Restored to standard 5.113% from July 2024 (was temporarily reduced).

**Price sources:** OMIE (wholesale), Energy-Charts (wholesale)

---

### Portugal (PT)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| IEC | **0.00100** | [EDP - Taxas e Impostos](https://www.edp.pt/particulares/apoio-cliente/perguntas-frequentes/pt/faturas/sobre-a-sua-fatura/que-taxas-impostos-e-contribuicoes-me-sao-cobrados-e-o-que-sao/faq-5740/) |
| Surcharges | 0.0 | - |
| VAT (IVA) | **6%** | Autoridade Tributaria (reduced rate for electricity) |
| Coefficient | 1.0 | - |

**Notes:**
- Portugal applies a **reduced IVA of 6%** on household electricity.
- IEC (Imposto Especial de Consumo) at EU minimum for most consumption.

**Price sources:** OMIE (wholesale), Energy-Charts (wholesale)

---

### Italy (IT)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Accisa sull'energia elettrica | **0.02270** | [Tax Foundation - Excise Duties Electricity EU](https://taxfoundation.org/data/all/eu/excise-duties-electricity-europe-2024/) |
| Surcharges | 0.0 | - |
| VAT (IVA) | 22% | Agenzia delle Entrate |
| Coefficient | 1.0 | - |

**Notes:**
- Accisa is 22.70 EUR/MWh for non-business domestic consumers.
- Primary residences with power <=3 kW: first 150 kWh/month exempt from accisa.
- Italian residential electricity IVA may be 10% for primary residences (vs 22% standard). We use 22% as conservative default.
- 6 bidding zones (IT-North, IT-Centre-North, IT-Centre-South, IT-South, IT-Sicily, IT-Sardinia).

**Price sources:** Energy-Charts (wholesale)

---

### Switzerland (CH)

| Component | Rate (CHF/kWh) | Source |
|-----------|----------------|--------|
| Netzzuschlag (KEV) | 0.02300 | [BFE - Netzzuschlag](https://www.bfe.admin.ch/bfe/de/home/foerderung/erneuerbare-energien/netzzuschlag.html) |
| Winterreserve + SDL | 0.00460 | [Swissgrid - Tariffs 2026](https://www.swissgrid.ch/en/home/newsroom/newsfeed/20250317-01.html) |
| **Total levies** | **0.02760** | Sum of above |
| VAT (MwSt) | 8.1% | Eidgenossische Steuerverwaltung |
| Coefficient | 1.0 | - |

**Notes:**
- Switzerland is **not an EU member** and has no EU-style energy excise tax. Instead, federal levies fund renewable energy (Netzzuschlag/KEV) and grid stability (Winterreserve).
- Prices from Energy-Charts are in EUR; CHF conversion may apply.
- Netzzuschlag = 2.3 Rp/kWh, Winterreserve = 0.41 Rp/kWh, SDL = 0.05 Rp/kWh.

**Price sources:** Energy-Charts (wholesale)

---

### Hungary (HU)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Energiaado | **0.00109** | [PwC - Hungary Corporate Taxes](https://taxsummaries.pwc.com/hungary/corporate/other-taxes) |
| Surcharges | 0.0 | - |
| VAT (AFA) | 27% | NAV (highest in EU) |
| Coefficient | 1.0 | - |

**Notes:**
- Energiaado = 415 HUF/MWh, converted at ~380 HUF/EUR.
- Hungary has the **highest VAT rate in the EU** at 27%.
- Exchange rate fluctuations may cause variance; HUF has been volatile.

**Price sources:** Energy-Charts (wholesale)

---

### Poland (PL)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Akcyza (excise) | 0.00116 | [URE - Excise Tax](https://www.ure.gov.pl/) |
| OZE fee (renewable) | 0.00176 | [Gramwzielone - OZE 2026](https://www.gramwzielone.pl/trendy/20343066/oplata-przejsciowa-zniknie-z-rachunkow-nie-bedzie-podwyzki-oplaty-kogeneracyjnej) |
| Cogeneration fee | 0.00072 | [E-magazyny - Kogeneracja 2026](https://e-magazyny.pl/aktualnosci/oplata-kogeneracyjna-w-2026-roku-bez-zmian-minister-energii-to-rownowaga-miedzy-odbiorcami-a-wytworcami/) |
| **Total surcharges** | **0.00249** | OZE 7.3 PLN/MWh + cogen 3.0 PLN/MWh, at 4.15 PLN/EUR |
| VAT | 23% | Krajowa Administracja Skarbowa |
| Coefficient | 1.0 | - |

**Price sources:** Energy-Charts (wholesale)

---

### Czech Republic (CZ)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Energy tax | 0.00112 | At EU minimum rate (~28.3 CZK/MWh at ~25.3 CZK/EUR) |
| Surcharges | 0.0 | - |
| VAT (DPH) | 21% | Financni sprava |
| Coefficient | 1.0 | - |

**Price sources:** Energy-Charts (wholesale)

---

### Slovenia (SI)

| Component | Rate (EUR/kWh) | Source |
|-----------|----------------|--------|
| Energy tax | 0.00153 | [FURS - Energy Taxation](https://www.fu.gov.si/) |
| Surcharges | 0.0 | - |
| VAT (DDV) | 22% | FURS |
| Coefficient | 1.0 | - |

**Price sources:** Energy-Charts (wholesale)

---

## Price Source Hierarchy

For each bidding zone, we fetch prices from multiple sources with automatic fallback:

| Zone(s) | Tier 1 (primary) | Tier 2 | Tier 3 |
|---------|-------------------|--------|--------|
| NL | EasyEnergy | Frank Energie | Energy-Charts |
| DE-LU | aWATTar | Energy-Charts | - |
| AT | aWATTar | Energy-Charts | - |
| NO1-5, SE1-4 | Energi Data Service | Energy-Charts | spot-hinta.fi |
| DK1-2 | Energi Data Service | Energy-Charts | - |
| FI | spot-hinta.fi | Energi Data Service | Energy-Charts |
| ES, PT | OMIE | Energy-Charts | - |
| BE, FR, IT, HU, PL, CZ, CH, SI | Energy-Charts | - | - |

### Source Types

| Source | Type | Coverage | Auth Required |
|--------|------|----------|---------------|
| EasyEnergy | Wholesale + margin | NL | No |
| Frank Energie | **Consumer price** (IsConsumer) | NL | No |
| aWATTar | Wholesale (EPEX) | DE-LU, AT | No |
| Energy-Charts | Wholesale (EPEX) | All EU zones | No |
| Energi Data Service | Wholesale (Nordpool) | Nordic, Baltic | No |
| spot-hinta.fi | Wholesale (Nordpool) | Nordic | No |
| OMIE | Wholesale (Iberian market) | ES, PT | No |

---

## Currency Conversion

Non-EUR countries require exchange rate conversion for tax rates in YAML:

| Country | Currency | Rate Used | Notes |
|---------|----------|-----------|-------|
| Norway | NOK | 11.5 NOK/EUR | Central bank mid-rate |
| Sweden | SEK | 11.5 SEK/EUR | Central bank mid-rate |
| Denmark | DKK | 7.46 DKK/EUR | ERM-II quasi-fixed rate |
| Hungary | HUF | 380 HUF/EUR | Volatile; updated Feb 2026 |
| Poland | PLN | 4.15 PLN/EUR | Relatively stable |
| Czech Rep. | CZK | 25.3 CZK/EUR | CNB reference rate |
| Switzerland | CHF | ~1.0 CHF/EUR | Prices in CHF natively |

**Note:** Exchange rate fluctuations may cause 3-8% variance for NOK/SEK/HUF zones. DKK is pegged to EUR within a narrow band and is effectively fixed.

---

## Verification Methodology

All energy tax rates were verified on 2026-02-11 using:

1. **Official government tax authorities** as primary source (Belastingdienst, Zoll.de, Skatteetaten, etc.)
2. **Cross-validation** against consumer price APIs where available (Frank Energie component breakdown for NL)
3. **Eurostat** reference data for EU minimum rates
4. **Live comparison** testing: normalized wholesale prices compared against known consumer price sources

### NL Verification Example

The NL energy tax was cross-validated against Frank Energie's API:

```
Frank energyTaxPrice = 11.085 ct (consumer-facing, incl. BTW)
Excl. BTW = 11.085 / 1.21 = 9.161 ct
Matches Belastingdienst rate: 9.161 ct/kWh excl. BTW
```

This confirmed:
- Our normalizer formula `(wholesale + tax) * (1 + VAT)` is correct
- YAML rates must be **excluding VAT** to prevent double-taxation
- The resulting consumer price matches Frank's to within 0.4 ct (supplier margin difference)

---

## Updating Tax Rates

Tax rates typically change on January 1st of each year. To update:

1. Check government sources listed above for new rates
2. Ensure rates are **excluding VAT** (this is the most common error)
3. Add a new `energy_tax` entry with the effective date:
   ```yaml
   energy_tax:
     - from: "2026-01-01"
       rate: 0.09161  # Old rate
     - from: "2027-01-01"
       rate: 0.XXXXX  # New rate (excl. VAT!)
   ```
4. Run tests: `go test ./... -count=1`
5. Verify with live data comparison if consumer price source available
