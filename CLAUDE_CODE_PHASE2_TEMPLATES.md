# Claude Code Phase 2 - Run Script Templates

**Repo:** `/c/Workbench/DEV/ha-energy-insights-nl`
**Branch:** `git checkout -b refactor/run-script-templates`

---

## DOEL

Converteer hardcoded run scripts naar templates met `{{PLACEHOLDERS}}`.
Installer (FASE 5) genereert dan brand-specifieke scripts.

---

## EXTRA: Fix systemd/synctacles-tennet.service.template

**Regel 9 vervangen van:**
```
ExecStart={{INSTALL_PATH}}/venv/bin/python3 {{INSTALL_PATH}}/app/sparkcrawler_db/collectors/sparkcrawler_tennet_ingestor.py
```

**Naar:**
```
ExecStart={{INSTALL_PATH}}/venv/bin/python3 -m synctacles_db.collectors.tennet_ingestor
```

---

## STAP 0: Inventarisatie

```bash
# Toon huidige run scripts
cat scripts/run_collectors.sh
cat scripts/run_importers.sh
cat scripts/run_normalizers.sh 2>/dev/null || echo "Niet gevonden"
cat scripts/run_tennet.sh 2>/dev/null || echo "Niet gevonden"
cat scripts/run_health.sh 2>/dev/null || echo "Niet gevonden"

# Check bestaande systemd templates
ls -la systemd/*.template
```

Toon output voordat je verdergaat.

---

## STAP 1: Maak template directory

```bash
mkdir -p systemd/scripts
```

---

## STAP 2: Maak run_collectors.sh.template

Maak bestand `systemd/scripts/run_collectors.sh.template`:

```bash
#!/usr/bin/env bash
# =============================================================================
# GENERATED FILE - DO NOT EDIT DIRECTLY
# Generated from: systemd/scripts/run_collectors.sh.template
# Regenerate with: setup script FASE 5
# =============================================================================
set -euo pipefail

# Paths from template substitution
APP_DIR="{{INSTALL_PATH}}/app"
VENV_DIR="{{INSTALL_PATH}}/venv"
LOG_DIR="{{LOG_PATH}}/scheduler"
ENV_FILE="{{ENV_FILE}}"

mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/collectors_$(date +%Y%m%d_%H%M%S).log"

# Activate venv
source "$VENV_DIR/bin/activate"

# Set PYTHONPATH
export PYTHONPATH="$APP_DIR:${PYTHONPATH:-}"

# Load environment variables
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

cd "$APP_DIR"

echo "[$(date)] Starting collectors..." | tee -a "$LOG_FILE"
python3 -m synctacles_db.collectors.entso_e_a75_generation 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.collectors.entso_e_a65_load 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.collectors.entso_e_a44_prices 2>&1 | tee -a "$LOG_FILE"
echo "[$(date)] Collectors complete" | tee -a "$LOG_FILE"
```

---

## STAP 3: Maak run_importers.sh.template

Maak bestand `systemd/scripts/run_importers.sh.template`:

```bash
#!/usr/bin/env bash
# =============================================================================
# GENERATED FILE - DO NOT EDIT DIRECTLY
# Generated from: systemd/scripts/run_importers.sh.template
# Regenerate with: setup script FASE 5
# =============================================================================
set -euo pipefail

# Paths from template substitution
APP_DIR="{{INSTALL_PATH}}/app"
VENV_DIR="{{INSTALL_PATH}}/venv"
LOG_DIR="{{LOG_PATH}}/scheduler"
ENV_FILE="{{ENV_FILE}}"

mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/importers_$(date +%Y%m%d_%H%M%S).log"

# Activate venv
source "$VENV_DIR/bin/activate"

# Set PYTHONPATH
export PYTHONPATH="$APP_DIR:${PYTHONPATH:-}"

# Load environment variables
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

cd "$APP_DIR"

echo "[$(date)] Starting importers..." | tee -a "$LOG_FILE"
python3 -m synctacles_db.importers.import_entso_e_a75 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.importers.import_entso_e_a65 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.importers.import_tennet_balance 2>&1 | tee -a "$LOG_FILE"
echo "[$(date)] Importers complete" | tee -a "$LOG_FILE"
```

---

## STAP 4: Maak run_normalizers.sh.template

Maak bestand `systemd/scripts/run_normalizers.sh.template`:

```bash
#!/usr/bin/env bash
# =============================================================================
# GENERATED FILE - DO NOT EDIT DIRECTLY
# Generated from: systemd/scripts/run_normalizers.sh.template
# Regenerate with: setup script FASE 5
# =============================================================================
set -euo pipefail

# Paths from template substitution
APP_DIR="{{INSTALL_PATH}}/app"
VENV_DIR="{{INSTALL_PATH}}/venv"
LOG_DIR="{{LOG_PATH}}/scheduler"
ENV_FILE="{{ENV_FILE}}"

mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/normalizers_$(date +%Y%m%d_%H%M%S).log"

# Activate venv
source "$VENV_DIR/bin/activate"

# Set PYTHONPATH
export PYTHONPATH="$APP_DIR:${PYTHONPATH:-}"

# Load environment variables
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

cd "$APP_DIR"

echo "[$(date)] Starting normalizers..." | tee -a "$LOG_FILE"
python3 -m synctacles_db.normalizers.normalize_entso_e_a75 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.normalizers.normalize_entso_e_a65 2>&1 | tee -a "$LOG_FILE"
python3 -m synctacles_db.normalizers.normalize_tennet_balance 2>&1 | tee -a "$LOG_FILE"
echo "[$(date)] Normalizers complete" | tee -a "$LOG_FILE"
```

---

## STAP 5: Maak run_tennet.sh.template

Maak bestand `systemd/scripts/run_tennet.sh.template`:

```bash
#!/usr/bin/env bash
# =============================================================================
# GENERATED FILE - DO NOT EDIT DIRECTLY
# Generated from: systemd/scripts/run_tennet.sh.template
# Regenerate with: setup script FASE 5
# =============================================================================
set -euo pipefail

# Paths from template substitution
APP_DIR="{{INSTALL_PATH}}/app"
VENV_DIR="{{INSTALL_PATH}}/venv"
LOG_DIR="{{LOG_PATH}}/scheduler"
ENV_FILE="{{ENV_FILE}}"

mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/tennet_$(date +%Y%m%d_%H%M%S).log"

# Activate venv
source "$VENV_DIR/bin/activate"

# Set PYTHONPATH
export PYTHONPATH="$APP_DIR:${PYTHONPATH:-}"

# Load environment variables
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

cd "$APP_DIR"

echo "[$(date)] Starting TenneT collector..." | tee -a "$LOG_FILE"
python3 -m synctacles_db.collectors.tennet_ingestor 2>&1 | tee -a "$LOG_FILE"
echo "[$(date)] TenneT collector complete" | tee -a "$LOG_FILE"
```

---

## STAP 6: Maak run_health.sh.template

Maak bestand `systemd/scripts/run_health.sh.template`:

```bash
#!/usr/bin/env bash
# =============================================================================
# GENERATED FILE - DO NOT EDIT DIRECTLY
# Generated from: systemd/scripts/run_health.sh.template
# Regenerate with: setup script FASE 5
# =============================================================================
set -euo pipefail

# Paths from template substitution
APP_DIR="{{INSTALL_PATH}}/app"
VENV_DIR="{{INSTALL_PATH}}/venv"
LOG_DIR="{{LOG_PATH}}/scheduler"
ENV_FILE="{{ENV_FILE}}"

mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/health_$(date +%Y%m%d_%H%M%S).log"

# Activate venv
source "$VENV_DIR/bin/activate"

# Set PYTHONPATH
export PYTHONPATH="$APP_DIR:${PYTHONPATH:-}"

# Load environment variables
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

cd "$APP_DIR"

echo "[$(date)] Running health check..." | tee -a "$LOG_FILE"
# Add health check commands here
curl -s http://localhost:8000/health || echo "API not responding"
echo "[$(date)] Health check complete" | tee -a "$LOG_FILE"
```

---

## STAP 7: Update .gitignore

Voeg toe aan `.gitignore`:

```gitignore
# Generated run scripts (from templates)
scripts/run_collectors.sh
scripts/run_importers.sh
scripts/run_normalizers.sh
scripts/run_tennet.sh
scripts/run_health.sh
```

---

## STAP 8: Verwijder oude hardcoded scripts uit git tracking

```bash
git rm --cached scripts/run_collectors.sh 2>/dev/null || true
git rm --cached scripts/run_importers.sh 2>/dev/null || true
git rm --cached scripts/run_normalizers.sh 2>/dev/null || true
git rm --cached scripts/run_tennet.sh 2>/dev/null || true
git rm --cached scripts/run_health.sh 2>/dev/null || true
```

**Let op:** Verwijder NIET de fysieke bestanden, alleen uit git tracking.

---

## VERIFICATIE

```bash
# 1. Templates bestaan
ls -la systemd/scripts/*.template

# 2. Placeholders correct
grep -c "{{INSTALL_PATH}}" systemd/scripts/*.template
grep -c "{{LOG_PATH}}" systemd/scripts/*.template
grep -c "{{ENV_FILE}}" systemd/scripts/*.template

# 3. Geen hardcoded paden in templates
grep -E "/opt/synctacles|/opt/energy-insights" systemd/scripts/*.template
# MOET LEEG ZIJN

# 4. Module paden correct (synctacles_db, niet sparkcrawler_db)
grep "synctacles_db" systemd/scripts/*.template | head -5
grep "sparkcrawler_db" systemd/scripts/*.template
# TWEEDE MOET LEEG ZIJN

# 5. .gitignore updated
grep "run_collectors.sh" .gitignore
```

---

## COMMIT

```bash
git add -A
git commit -m "REFACTOR: Convert run scripts to templates

Phase 2 of Option B debranding:
- Created systemd/scripts/ directory for run script templates
- Added 5 template files with {{PLACEHOLDERS}}:
  - run_collectors.sh.template
  - run_importers.sh.template
  - run_normalizers.sh.template
  - run_tennet.sh.template
  - run_health.sh.template
- Updated .gitignore to exclude generated scripts
- Removed hardcoded scripts from git tracking

Placeholders used:
- {{INSTALL_PATH}} - Base installation directory
- {{LOG_PATH}} - Log directory path
- {{ENV_FILE}} - Environment file location

Installer FASE 5 will generate brand-specific scripts from these templates.
"
```

---

## PLACEHOLDER REFERENTIE

| Placeholder | Typical Value | Source |
|-------------|---------------|--------|
| `{{INSTALL_PATH}}` | `/opt/energy-insights-nl` | .env: INSTALL_PATH |
| `{{LOG_PATH}}` | `/var/log/energy-insights-nl` | .env: LOG_PATH |
| `{{ENV_FILE}}` | `/opt/.env` | Hardcoded in installer |
| `{{BRAND_SLUG}}` | `energy-insights-nl` | .env: BRAND_SLUG |

---

## ROLLBACK

```bash
git checkout main
git branch -D refactor/run-script-templates
```
