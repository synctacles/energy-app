# Claude Code Refactor Instructions - Phase 1

**Repo:** `/c/Workbench/DEV/ha-energy-insights-nl`
**Doel:** Consolideer `sparkcrawler_db` naar `synctacles_db`, verwijder alle brand-specifieke prefixes

---

## STAP 0: Inventarisatie (EERST UITVOEREN)

```bash
# Voer uit en toon output VOORDAT je wijzigingen maakt:

# 1. Huidige sparkcrawler structuur
find . -path "*/sparkcrawler_db/*" -name "*.py" -type f

# 2. Huidige synctacles structuur  
find . -path "*/synctacles_db/*" -name "*.py" -type f

# 3. Alle imports die aangepast moeten worden
grep -rn "from sparkcrawler_db\|import sparkcrawler_db" --include="*.py"

# 4. Alle -m module calls in shell scripts
grep -rn "sparkcrawler_db" --include="*.sh"
```

Toon mij deze output voordat je verdergaat.

---

## STAP 1: Maak target directories

```bash
mkdir -p synctacles_db/collectors
mkdir -p synctacles_db/importers
```

Maak `synctacles_db/collectors/__init__.py`:
```python
"""
Synctacles Data Collectors

Collectors halen ruwe data op van externe APIs en schrijven naar raw_* tabellen.
"""
from .entso_e_a75_generation import *
from .entso_e_a65_load import *
from .entso_e_a44_prices import *
from .tennet_ingestor import *
```

Maak `synctacles_db/importers/__init__.py`:
```python
"""
Synctacles Data Importers

Importers verwerken XML/JSON files naar database records.
"""
```

---

## STAP 2: Migreer collectors (COPY + RENAME)

**Bestandsnaam mapping:**

| Bron (sparkcrawler_db/collectors/) | Doel (synctacles_db/collectors/) |
|-----------------------------------|----------------------------------|
| `sparkcrawler_entso_e_a75_generation.py` | `entso_e_a75_generation.py` |
| `sparkcrawler_entso_e_a65_load.py` | `entso_e_a65_load.py` |
| `sparkcrawler_entso_e_a44_prices.py` | `entso_e_a44_prices.py` |
| `sparkcrawler_tennet_ingestor.py` | `tennet_ingestor.py` |
| (andere collectors) | (zonder sparkcrawler_ prefix) |

**Actie:** Kopieer elk bestand naar nieuwe locatie met nieuwe naam.

---

## STAP 3: Update imports IN de gekopieerde bestanden

In ELKE gekopieerde collector, vervang:

```python
# OUD
from sparkcrawler_db.collectors import ...
from sparkcrawler_db.models import ...
from sparkcrawler_db import ...
import sparkcrawler_db

# NIEUW
from synctacles_db.collectors import ...
from synctacles_db.models import ...
from synctacles_db import ...
import synctacles_db
```

---

## STAP 4: Migreer importers (zelfde patroon)

Kopieer alle bestanden uit `sparkcrawler_db/importers/` naar `synctacles_db/importers/`
- Verwijder `sparkcrawler_` prefix uit bestandsnamen
- Update imports in elk bestand

---

## STAP 5: Merge models.py (indien nodig)

Als `sparkcrawler_db/models.py` bestaat:
- Vergelijk met `synctacles_db/models.py`
- Merge unieke models naar `synctacles_db/models.py`
- Update imports naar `synctacles_db.models`

---

## STAP 6: Update ALLE andere bestanden

Zoek en vervang in HELE repo:

```python
# Pattern 1: from imports
"from sparkcrawler_db" → "from synctacles_db"

# Pattern 2: import statements  
"import sparkcrawler_db" → "import synctacles_db"

# Pattern 3: -m module calls in .sh files
"python3 -m sparkcrawler_db.collectors.sparkcrawler_" → "python3 -m synctacles_db.collectors."
"python3 -m sparkcrawler_db." → "python3 -m synctacles_db."
```

**Bestanden om te checken:**
- `app/scripts/run_*.sh`
- `systemd/*.template`
- `config/settings.py`
- Alle `.py` bestanden

---

## STAP 7: Update run scripts

In alle `run_*.sh` bestanden, update module calls:

```bash
# OUD
python3 -m sparkcrawler_db.collectors.sparkcrawler_entso_e_a75_generation

# NIEUW
python3 -m synctacles_db.collectors.entso_e_a75_generation
```

---

## STAP 8: Verwijder oude module

**PAS NA VERIFICATIE:**
```bash
rm -rf sparkcrawler_db/
```

---

## VERIFICATIE COMMANDO'S

Na voltooiing, voer uit:

```bash
# 1. Geen sparkcrawler referenties meer (moet leeg zijn)
grep -rn "sparkcrawler_db" --include="*.py" --include="*.sh"

# 2. Nieuwe structuur correct
find . -path "*/synctacles_db/*" -name "*.py" -type f

# 3. Imports werken (Python syntax check)
python3 -c "from synctacles_db.collectors import entso_e_a75_generation; print('OK')"
python3 -c "from synctacles_db.collectors import entso_e_a65_load; print('OK')"
python3 -c "from synctacles_db.collectors import tennet_ingestor; print('OK')"

# 4. Geen ongebruikte imports
python3 -m py_compile synctacles_db/collectors/*.py
```

---

## ROLLBACK

Als iets misgaat:
```bash
git checkout -- .
git clean -fd
```

---

## COMMIT MESSAGE

Na succesvolle verificatie:
```bash
git add -A
git commit -m "REFACTOR: Consolidate sparkcrawler_db into synctacles_db

- Moved collectors to synctacles_db/collectors/
- Moved importers to synctacles_db/importers/  
- Removed sparkcrawler_ prefix from all filenames
- Updated all imports throughout codebase
- Updated run scripts with new module paths
- Removed deprecated sparkcrawler_db module

Part of Option B debranding refactor.
"
```

---

## BELANGRIJKE REGELS VOOR CLAUDE CODE

1. **EERST inventarisatie** - toon STAP 0 output voordat je wijzigt
2. **GEEN hardcoded paden** - gebruik relatieve paden
3. **BEHOUD bestaande logica** - alleen rename/move, geen functionele wijzigingen
4. **TEST na elke stap** - syntax errors direct fixen
5. **VRAAG bij twijfel** - beter checken dan breken
