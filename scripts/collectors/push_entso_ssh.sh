#!/bin/bash
#
# Push ENTSO-E prices to coefficient server via SSH
# Reads from local norm_entso_e_a44 and pushes to remote hist_entso_prices
#

set -e

# Get today's date
TODAY=$(date -u +"%Y-%m-%d")

# Extract prices from local DB
PRICES=$(psql -t -A -F'|' -d energy_insights_nl -c "
SELECT timestamp, price_eur_mwh
FROM norm_entso_e_a44
WHERE timestamp >= '${TODAY} 00:00:00+00'
ORDER BY timestamp
")

if [ -z "$PRICES" ]; then
    echo "No ENTSO-E prices found for today"
    exit 0
fi

# Count prices
COUNT=$(echo "$PRICES" | wc -l)

# Create SQL for remote insert
SQL="BEGIN;"
while IFS='|' read -r timestamp price; do
    SQL="${SQL}
INSERT INTO hist_entso_prices (timestamp, country_code, area_code, price_eur_mwh)
VALUES ('${timestamp}', 'NL', 'NL-ALL', ${price})
ON CONFLICT (timestamp, area_code) DO UPDATE SET price_eur_mwh = EXCLUDED.price_eur_mwh;"
done <<< "$PRICES"
SQL="${SQL}
COMMIT;"

# Push to coefficient server via SSH
ssh coefficient "sudo -u postgres psql -d coefficient_db -c \"${SQL}\"" > /dev/null 2>&1

echo "Pushed ${COUNT} ENTSO-E prices to coefficient server"
exit 0
