# Claude Code Refactor - UITVOERING

**Repo:** `/c/Workbench/DEV/ha-energy-insights-nl`
**Branch:** Maak EERST nieuwe branch: `git checkout -b refactor/module-consolidation`

---

## FASE A: Collectors migreren

### A1. Maak directories
```bash
mkdir -p synctacles_db/collectors
mkdir -p synctacles_db/importers
```

### A2. Kopieer en hernoem collectors

| Bron | Doel |
|------|------|
| `sparkcrawler_db/collectors/sparkcrawler_entso_e_a75_generation.py` | `synctacles_db/collectors/entso_e_a75_generation.py` |
| `sparkcrawler_db/collectors/sparkcrawler_entso_e_a65_load.py` | `synctacles_db/collectors/entso_e_a65_load.py` |
| `sparkcrawler_db/collectors/sparkcrawler_entso_e_a44_prices.py` | `synctacles_db/collectors/entso_e_a44_prices.py` |
| `sparkcrawler_db/collectors/sparkcrawler_tennet_ingestor.py` | `synctacles_db/collectors/tennet_ingestor.py` |
| `sparkcrawler_db/collectors/energy_charts_prices.py` | `synctacles_db/collectors/energy_charts_prices.py` |

### A3. Maak `synctacles_db/collectors/__init__.py`
```python
"""
Synctacles Data Collectors

Collectors halen ruwe data op van externe APIs.
Output: XML/JSON files in logs/ directory.
"""
```

---

## FASE B: Importers migreren

### B1. Kopieer importers (namen blijven gelijk)

| Bron | Doel |
|------|------|
| `sparkcrawler_db/importers/import_entso_e_a75.py` | `synctacles_db/importers/import_entso_e_a75.py` |
| `sparkcrawler_db/importers/import_entso_e_a65.py` | `synctacles_db/importers/import_entso_e_a65.py` |
| `sparkcrawler_db/importers/import_entso_e_a44.py` | `synctacles_db/importers/import_entso_e_a44.py` |
| `sparkcrawler_db/importers/import_tennet_balance.py` | `synctacles_db/importers/import_tennet_balance.py` |
| `sparkcrawler_db/importers/import_energy_charts_prices.py` | `synctacles_db/importers/import_energy_charts_prices.py` |

### B2. Maak `synctacles_db/importers/__init__.py`
```python
"""
Synctacles Data Importers

Importers lezen XML/JSON files en schrijven naar raw_* tabellen.
"""
```

---

## FASE C: Models mergen

### C1. Vergelijk models
Open beide bestanden:
- `sparkcrawler_db/models.py`
- `synctacles_db/models.py`

### C2. Merge strategie
De `sparkcrawler_db/models.py` bevat RAW models:
- `RawEntsoeA75`
- `RawEntsoeA65`
- `RawEntsoeA44`
- `RawTennetBalance`

Kopieer deze classes naar `synctacles_db/models.py` als ze daar nog niet staan.
Behoud `Base` declaratie.

---

## FASE D: Imports updaten

### D1. In gekopieerde importers (synctacles_db/importers/*.py)
Vervang:
```python
# OUD
from sparkcrawler_db.models import RawEntsoeA75

# NIEUW
from synctacles_db.models import RawEntsoeA75
```

Bestanden:
- `synctacles_db/importers/import_entso_e_a75.py` (regel 22)
- `synctacles_db/importers/import_entso_e_a65.py` (regel 19)
- `synctacles_db/importers/import_tennet_balance.py` (regel 19)

### D2. In normalizers (KRITIEK - deze importeren NOG van sparkcrawler!)
Vervang in:
- `synctacles_db/normalizers/normalize_entso_e_a75.py` (regel 17)
- `synctacles_db/normalizers/normalize_entso_e_a65.py` (regel 17)
- `synctacles_db/normalizers/normalize_tennet_balance.py` (regel 17)

```python
# OUD
from sparkcrawler_db.models import RawEntsoeA75

# NIEUW
from synctacles_db.models import RawEntsoeA75
```

### D3. In alembic/env.py (regel 14)
```python
# OUD
from sparkcrawler_db.models import Base as SparkBase

# NIEUW
from synctacles_db.models import Base as SparkBase
```
OF verwijder als niet meer nodig (check of SparkBase nog gebruikt wordt in dat bestand).

---

## FASE E: Shell scripts updaten

### E1. `scripts/run_collectors.sh` (regels 27-29)
```bash
# OUD
python3 -m sparkcrawler_db.collectors.sparkcrawler_entso_e_a75_generation
python3 -m sparkcrawler_db.collectors.sparkcrawler_entso_e_a65_load
python3 -m sparkcrawler_db.collectors.sparkcrawler_entso_e_a44_prices

# NIEUW
python3 -m synctacles_db.collectors.entso_e_a75_generation
python3 -m synctacles_db.collectors.entso_e_a65_load
python3 -m synctacles_db.collectors.entso_e_a44_prices
```

### E2. `scripts/run_importers.sh` (regels 27-29)
```bash
# OUD
python3 -m sparkcrawler_db.importers.import_entso_e_a75
python3 -m sparkcrawler_db.importers.import_entso_e_a65
python3 -m sparkcrawler_db.importers.import_tennet_balance

# NIEUW
python3 -m synctacles_db.importers.import_entso_e_a75
python3 -m synctacles_db.importers.import_entso_e_a65
python3 -m synctacles_db.importers.import_tennet_balance
```

### E3. `scripts/deploy/deploy_fase2.sh`
Vervang alle `sparkcrawler_db` referenties:
- Regel 67: `SPARKCRAWLER_DIR` → verwijder of rename naar `SYNCTACLES_DIR`
- Regel 112-114: Update directory copy logic

### E4. `scripts/setup/setup_synctacles_server_v2.3.4.sh`
- Regel 1425: Check voor `synctacles_db` ipv `sparkcrawler_db`
- Regel 2260-2261: Update voorbeeldcommando's

### E5. `scripts/test/validate_synctacles_setup_v3.1.0.sh`
- Regel 397, 471, 472: Verwijder `sparkcrawler_db` checks, alleen `synctacles_db`

### E6. `scripts/validate_paths.sh`
- Regel 27, 36, 45: Verwijder `sparkcrawler_db` referenties

---

## FASE F: Cleanup

### F1. Verwijder oude module
```bash
rm -rf sparkcrawler_db/
```

### F2. Update .gitignore indien nodig
Check of `sparkcrawler_db` ergens expliciet genoemd wordt.

---

## VERIFICATIE

Na alle wijzigingen:

```bash
# 1. Geen sparkcrawler referenties meer
grep -rn "sparkcrawler_db" --include="*.py" --include="*.sh"
# MOET LEEG ZIJN

# 2. Python syntax check
python3 -m py_compile synctacles_db/collectors/*.py
python3 -m py_compile synctacles_db/importers/*.py
python3 -m py_compile synctacles_db/normalizers/*.py

# 3. Import test
python3 -c "from synctacles_db.collectors import entso_e_a75_generation; print('collectors OK')"
python3 -c "from synctacles_db.importers import import_entso_e_a75; print('importers OK')"
python3 -c "from synctacles_db.models import RawEntsoeA75; print('models OK')"
```

---

## COMMIT

```bash
git add -A
git commit -m "REFACTOR: Consolidate sparkcrawler_db into synctacles_db

Phase 1 of Option B debranding:
- Migrated collectors to synctacles_db/collectors/
- Migrated importers to synctacles_db/importers/
- Merged models into synctacles_db/models.py
- Updated all imports throughout codebase
- Updated shell scripts with new module paths
- Removed deprecated sparkcrawler_db module

Breaking change: All -m module calls now use synctacles_db.*
"
```

---

## ROLLBACK (indien nodig)

```bash
git checkout main
git branch -D refactor/module-consolidation
```
