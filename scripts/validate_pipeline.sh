#!/bin/bash
# Validate that all pipeline components are configured correctly

set -e

echo "=== Pipeline Validation ==="

# Check collectors (use entso_e pattern since collectors use different naming)
COLLECTORS=$(grep -c "synctacles_db.collectors.entso_e" scripts/run_collectors.sh 2>/dev/null || echo 0)
echo "Collectors in run_collectors.sh: $COLLECTORS"

# Check importers
IMPORTERS=$(grep -c "import_entso_e" scripts/run_importers.sh 2>/dev/null || echo 0)
echo "Importers in run_importers.sh: $IMPORTERS"

# Check normalizers
NORMALIZERS=$(grep -c "normalize_entso_e" scripts/run_normalizers.sh 2>/dev/null || echo 0)
echo "Normalizers in run_normalizers.sh: $NORMALIZERS"

if [ "$COLLECTORS" -ne "$IMPORTERS" ] || [ "$IMPORTERS" -ne "$NORMALIZERS" ]; then
    echo ""
    echo "⚠️  WARNING: Pipeline component mismatch!"
    echo "   Collectors: $COLLECTORS"
    echo "   Importers:  $IMPORTERS"
    echo "   Normalizers: $NORMALIZERS"
    echo ""
    echo "   Each collector should have matching importer AND normalizer."
    exit 1
else
    echo ""
    echo "✅ Pipeline validated: $COLLECTORS sources configured correctly"
fi
