# HANDOFF: Claude AI → Claude Code
## Coefficient Validatie: Frank API vs Enever Historisch

**Datum:** 2026-01-10
**Van:** Claude AI (analyse)
**Naar:** Claude Code (uitvoering)
**Server:** 91.99.150.36 (coefficient server)

---

## OPDRACHT

Valideer of de historische Enever coefficient data betrouwbaar is door te vergelijken met real-time Frank API data.

---

## CONTEXT

We hebben:
1. **Frank API** - real-time consumer prices (100% accuraat voor Frank klanten)
2. **Enever historisch** - 3+ jaar CSV data, coefficients berekend

**Vraag:** Klopt Enever's berekening? Is er systematische bias?

---

## STAPPEN

### Stap 1: Frank API ophalen voor vandaag

```bash
ssh coefficient@91.99.150.36

curl -s -X POST https://graphql.frankenergie.nl \
  -H "Content-Type: application/json" \
  -d '{"query":"{ marketPricesElectricity(startDate:\"2026-01-10\", endDate:\"2026-01-11\") { from till marketPrice marketPriceTax sourcingMarkupPrice energyTaxPrice }}"}' | python3 -m json.tool
```

**Verwachte output:** 24 uur-records met:
- `marketPrice` - wholesale (EPEX)
- `marketPriceTax` - BTW op wholesale
- `sourcingMarkupPrice` - Frank's marge (~€0.018)
- `energyTaxPrice` - energiebelasting (~€0.111)

### Stap 2: Bereken Frank's werkelijke markup per uur

```
frank_markup = marketPriceTax + sourcingMarkupPrice + energyTaxPrice
```

(Dit is alles behalve de kale wholesale prijs)

### Stap 3: Vergelijk met Enever coefficients

Locatie: `~/coefficients_enever_historical.json`

```bash
cat ~/coefficients_enever_historical.json | python3 -c "
import json, sys
data = json.load(sys.stdin)
frank = data['FrankEnergie']['hourly']
for h in range(24):
    print(f\"Hour {h:02d}: mean={frank[str(h)]['mean']:.4f}, std={frank[str(h)]['std']:.4f}\")
"
```

### Stap 4: Maak vergelijkingstabel

Voor elk uur (0-23):
- Frank real-time markup
- Enever historisch gemiddelde markup
- Verschil (€)
- Verschil (%)

---

## ALTERNATIEF: Database Query

Als de Enever data in PostgreSQL zit:

```bash
psql -h localhost -U coefficient -d coefficient
```

Query voor FrankEnergie consumer vs wholesale:

```sql
-- Check welke tabellen er zijn
\dt

-- Als er consumer_prices en wholesale_prices tabellen zijn:
SELECT 
    date_trunc('hour', timestamp) as hour,
    AVG(consumer_price - wholesale_price) as avg_markup
FROM prices
WHERE provider = 'FrankEnergie'
GROUP BY 1
ORDER BY 1;
```

---

## GEWENSTE OUTPUT

Lever een rapport met:

1. **Tabel: Frank Real-time vs Enever Historisch**
   - Per uur (0-23)
   - Frank markup vandaag
   - Enever historisch gemiddelde
   - Verschil

2. **Conclusie:**
   - Is er systematische bias? (bijv. Enever consistent hoger/lager)
   - Hoe groot is de afwijking? (€ en %)
   - Is Enever data bruikbaar als baseline?

3. **Aanbeveling:**
   - Kunnen we Enever coefficients vertrouwen?
   - Zo niet, wat is de correctiefactor?

---

## BRONNEN OP SERVER

```
~/
├── coefficients_enever_historical.json   # Berekende coefficients
├── data/enever_csv/FrankEnergie.csv      # Ruwe historische data
├── validate_enever.py                     # Bestaand validatiescript
└── FRANK_ENERGIE_API_REPORT.md           # API test resultaten
```

---

## DEADLINE

Geen harde deadline, maar dit blokkeert beslissing over coefficient accuracy claims.

---

*Gegenereerd door Claude AI - 10 januari 2026*
