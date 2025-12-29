# Claude Code Phase 3 - Installer Overhaul + Bugfixes

**Repo:** `/c/Workbench/DEV/ha-energy-insights-nl`
**Branch:** `git checkout -b refactor/installer-overhaul`

---

## DOEL

Installer volledig fixen zodat:
1. Alle config via één .env file met `export` statements
2. DATABASE_URL automatisch gegenereerd
3. Run scripts gegenereerd uit templates
4. Fail-fast bij ontbrekende configuratie
5. Geen hardcoded paden meer

---

## GEVONDEN ISSUES (te fixen)

| Issue | Locatie | Fix |
|-------|---------|-----|
| .env zonder `export` | FASE 0 | Voeg `export` toe aan alle variabelen |
| DATABASE_URL ontbreekt | FASE 0 | Genereer automatisch |
| Twee .env files | FASE 0/5 | Gebruik alleen `/opt/.env` |
| Scripts niet gegenereerd | FASE 5 | Genereer uit templates |
| settings.py fallbacks | config/settings.py | Fail-fast, geen defaults |
| Hardcoded paden in templates | systemd/*.template | Gebruik placeholders |

---

## STAP 0: Inventarisatie huidige installer

```bash
# Bekijk huidige FASE 0 (env generatie)
grep -A 50 "FASE 0" scripts/setup/setup_synctacles_server_v2.3.4.sh | head -60

# Bekijk huidige FASE 5 (systemd setup)
grep -A 80 "FASE 5" scripts/setup/setup_synctacles_server_v2.3.4.sh | head -90

# Bekijk config/settings.py
cat config/settings.py
```

Toon output voordat je verdergaat.

---

## STAP 1: Fix FASE 0 - .env generatie met export

In `scripts/setup/setup_synctacles_server_v2.3.4.sh`, vervang de FASE 0 .env generatie.

**Zoek naar** de sectie die .env aanmaakt (rond `cat > /opt/.env` of `cat > "$ENV_FILE"`).

**Vervang door:**

```bash
# =============================================================================
# FASE 0: Environment configuratie
# =============================================================================
section "FASE 0: Environment Setup"

ENV_FILE="/opt/.env"

# Backup existing .env if present
if [[ -f "$ENV_FILE" ]]; then
    cp "$ENV_FILE" "${ENV_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    info "Existing .env backed up"
fi

# Generate DATABASE_URL from components
DB_URL="postgresql://${DB_USER}@localhost:5432/${DB_NAME}"

cat > "$ENV_FILE" << EOF
# =============================================================================
# ${BRAND_NAME} Environment Configuration
# Generated: $(date -Iseconds)
# Generator: setup_synctacles_server_v2.3.4.sh
# =============================================================================

# Brand Configuration
export BRAND_NAME="${BRAND_NAME}"
export BRAND_SLUG="${BRAND_SLUG}"

# Installation Paths
export INSTALL_PATH="${INSTALL_PATH}"
export LOG_PATH="${LOG_PATH}"
export COLLECTOR_RAW_PATH="${COLLECTOR_RAW_PATH}"

# Database Configuration
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="${DB_NAME}"
export DB_USER="${DB_USER}"
export DATABASE_URL="${DB_URL}"

# API Keys (user must fill in)
export ENTSOE_API_KEY="${ENTSOE_API_KEY:-}"
export ADMIN_API_KEY="${ADMIN_API_KEY:-}"

# Service Configuration
export SERVICE_USER="${SERVICE_USER}"
export API_PORT="${API_PORT:-8000}"
EOF

chmod 600 "$ENV_FILE"
chown root:root "$ENV_FILE"

success ".env generated at $ENV_FILE"

# Validate critical variables are set
if [[ -z "${ENTSOE_API_KEY:-}" ]]; then
    warn "ENTSOE_API_KEY not set - collectors will fail until configured"
fi
```

---

## STAP 2: Fix FASE 5 - Script generatie uit templates

In `scripts/setup/setup_synctacles_server_v2.3.4.sh`, vervang/update FASE 5.

**Zoek naar** de sectie FASE 5 (systemd setup).

**Voeg toe** (of vervang bestaande script copy logica):

```bash
# =============================================================================
# FASE 5: Generate run scripts from templates
# =============================================================================
section "FASE 5: Systemd & Scripts Setup"

# Directory for generated scripts
SCRIPTS_DIR="${INSTALL_PATH}/app/scripts"
mkdir -p "$SCRIPTS_DIR"

# Template directory (in repo)
TEMPLATE_DIR="${INSTALL_PATH}/app/systemd/scripts"

info "Generating run scripts from templates..."

# Function to generate script from template
generate_script() {
    local template="$1"
    local output="$2"
    
    if [[ ! -f "$template" ]]; then
        warn "Template not found: $template"
        return 1
    fi
    
    sed -e "s|{{INSTALL_PATH}}|${INSTALL_PATH}|g" \
        -e "s|{{LOG_PATH}}|${LOG_PATH}|g" \
        -e "s|{{ENV_FILE}}|/opt/.env|g" \
        -e "s|{{BRAND_SLUG}}|${BRAND_SLUG}|g" \
        -e "s|{{SERVICE_USER}}|${SERVICE_USER}|g" \
        "$template" > "$output"
    
    chmod +x "$output"
    chown "${SERVICE_USER}:${SERVICE_USER}" "$output"
    success "Generated: $output"
}

# Generate all run scripts
generate_script "${TEMPLATE_DIR}/run_collectors.sh.template" "${SCRIPTS_DIR}/run_collectors.sh"
generate_script "${TEMPLATE_DIR}/run_importers.sh.template" "${SCRIPTS_DIR}/run_importers.sh"
generate_script "${TEMPLATE_DIR}/run_normalizers.sh.template" "${SCRIPTS_DIR}/run_normalizers.sh"

# Verify generated scripts
info "Verifying generated scripts..."
for script in run_collectors.sh run_importers.sh run_normalizers.sh; do
    if [[ -f "${SCRIPTS_DIR}/${script}" ]]; then
        # Check no unresolved placeholders
        if grep -q '{{' "${SCRIPTS_DIR}/${script}"; then
            error "Unresolved placeholders in ${script}"
            grep '{{' "${SCRIPTS_DIR}/${script}"
            exit 1
        fi
        success "✓ ${script}"
    else
        error "Missing: ${script}"
        exit 1
    fi
done
```

---

## STAP 3: Fix FASE 5 - Systemd service generatie

**Voeg toe aan FASE 5** (na script generatie):

```bash
# Generate systemd services from templates
info "Generating systemd services..."

SYSTEMD_TEMPLATE_DIR="${INSTALL_PATH}/app/systemd"
SYSTEMD_TARGET_DIR="/etc/systemd/system"

# Function to generate systemd unit from template
generate_systemd() {
    local template="$1"
    local service_name="$2"
    
    if [[ ! -f "$template" ]]; then
        warn "Template not found: $template"
        return 1
    fi
    
    sed -e "s|{{INSTALL_PATH}}|${INSTALL_PATH}|g" \
        -e "s|{{LOG_PATH}}|${LOG_PATH}|g" \
        -e "s|{{BRAND_NAME}}|${BRAND_NAME}|g" \
        -e "s|{{BRAND_SLUG}}|${BRAND_SLUG}|g" \
        -e "s|{{SERVICE_USER}}|${SERVICE_USER}|g" \
        -e "s|{{API_PORT}}|${API_PORT:-8000}|g" \
        "$template" > "${SYSTEMD_TARGET_DIR}/${service_name}"
    
    success "Generated: ${service_name}"
}

# Generate services
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-api.service.template" "${BRAND_SLUG}-api.service"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-collector.service.template" "${BRAND_SLUG}-collector.service"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-collector.timer.template" "${BRAND_SLUG}-collector.timer"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-importer.service.template" "${BRAND_SLUG}-importer.service"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-importer.timer.template" "${BRAND_SLUG}-importer.timer"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-normalizer.service.template" "${BRAND_SLUG}-normalizer.service"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-normalizer.timer.template" "${BRAND_SLUG}-normalizer.timer"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-tennet.service.template" "${BRAND_SLUG}-tennet.service"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-tennet.timer.template" "${BRAND_SLUG}-tennet.timer"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-health.service.template" "${BRAND_SLUG}-health.service"
generate_systemd "${SYSTEMD_TEMPLATE_DIR}/synctacles-health.timer.template" "${BRAND_SLUG}-health.timer"

# Reload systemd
systemctl daemon-reload
success "Systemd daemon reloaded"
```

---

## STAP 4: Fix config/settings.py - Fail-fast

**Vervang** `config/settings.py` volledig:

```python
"""
Synctacles Configuration - Fail-Fast Design

All configuration MUST come from environment variables.
No fallback defaults - missing config = immediate failure.
"""
import os
import sys

class ConfigurationError(Exception):
    """Raised when required configuration is missing."""
    pass

def require_env(key: str, description: str = "") -> str:
    """Get required environment variable or fail immediately."""
    value = os.getenv(key)
    if value is None or value == "":
        msg = f"FATAL: Required environment variable '{key}' is not set."
        if description:
            msg += f"\n  Description: {description}"
        msg += f"\n  Ensure /opt/.env is sourced with 'set -a && source /opt/.env && set +a'"
        print(msg, file=sys.stderr)
        raise ConfigurationError(msg)
    return value

def optional_env(key: str, default: str = "") -> str:
    """Get optional environment variable with explicit default."""
    return os.getenv(key, default)

# =============================================================================
# Required Configuration (fail-fast if missing)
# =============================================================================

# Database - REQUIRED
DATABASE_URL = require_env("DATABASE_URL", "PostgreSQL connection string")
DB_HOST = require_env("DB_HOST", "Database hostname")
DB_PORT = require_env("DB_PORT", "Database port")
DB_NAME = require_env("DB_NAME", "Database name")
DB_USER = require_env("DB_USER", "Database user")

# Paths - REQUIRED
INSTALL_PATH = require_env("INSTALL_PATH", "Base installation directory")
LOG_PATH = require_env("LOG_PATH", "Log directory")
COLLECTOR_RAW_PATH = require_env("COLLECTOR_RAW_PATH", "Raw collector data directory")

# =============================================================================
# Optional Configuration (explicit defaults)
# =============================================================================

# API Keys - Optional (will fail at runtime if needed but not set)
ENTSOE_API_KEY = optional_env("ENTSOE_API_KEY", "")
ADMIN_API_KEY = optional_env("ADMIN_API_KEY", "")

# API Settings
API_PORT = int(optional_env("API_PORT", "8000"))
API_HOST = optional_env("API_HOST", "0.0.0.0")

# Brand (for multi-tenant)
BRAND_NAME = optional_env("BRAND_NAME", "Synctacles")
BRAND_SLUG = optional_env("BRAND_SLUG", "synctacles")
```

---

## STAP 5: Update systemd service templates - ExecStart paden

Check en fix alle service templates die nog hardcoded paden hebben.

**Check:**
```bash
grep -r "sparkcrawler" systemd/
grep -r "/opt/synctacles" systemd/
grep -r "ExecStart=" systemd/*.template
```

**Fix elk bestand** dat nog oude paden bevat.

**synctacles-collector.service.template** moet zijn:
```ini
[Unit]
Description={{BRAND_NAME}} Collector
After=network.target

[Service]
Type=oneshot
User={{SERVICE_USER}}
WorkingDirectory={{INSTALL_PATH}}/app
ExecStart={{INSTALL_PATH}}/app/scripts/run_collectors.sh
StandardOutput=journal
StandardError=journal
```

**synctacles-importer.service.template** moet zijn:
```ini
[Unit]
Description={{BRAND_NAME}} Importer
After=network.target

[Service]
Type=oneshot
User={{SERVICE_USER}}
WorkingDirectory={{INSTALL_PATH}}/app
ExecStart={{INSTALL_PATH}}/app/scripts/run_importers.sh
StandardOutput=journal
StandardError=journal
```

**synctacles-normalizer.service.template** moet zijn:
```ini
[Unit]
Description={{BRAND_NAME}} Normalizer
After=network.target

[Service]
Type=oneshot
User={{SERVICE_USER}}
WorkingDirectory={{INSTALL_PATH}}/app
ExecStart={{INSTALL_PATH}}/app/scripts/run_normalizers.sh
StandardOutput=journal
StandardError=journal
```

---

## STAP 6: Maak health_check.sh.template

De health service verwacht `scripts/health_check.sh`. Maak template:

**Maak** `systemd/scripts/health_check.sh.template`:

```bash
#!/usr/bin/env bash
# =============================================================================
# GENERATED FILE - DO NOT EDIT DIRECTLY
# Generated from: systemd/scripts/health_check.sh.template
# =============================================================================
set -euo pipefail

ENV_FILE="{{ENV_FILE}}"

# Load environment
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

API_PORT="${API_PORT:-8000}"

echo "[$(date)] Running health check..."

# Check API
if curl -sf "http://localhost:${API_PORT}/health" > /dev/null 2>&1; then
    echo "✓ API healthy"
else
    echo "✗ API not responding"
    exit 1
fi

# Check database connection
if command -v psql &> /dev/null; then
    if psql "${DATABASE_URL}" -c "SELECT 1" > /dev/null 2>&1; then
        echo "✓ Database healthy"
    else
        echo "✗ Database not responding"
        exit 1
    fi
fi

echo "[$(date)] Health check complete"
```

**Update** `systemd/synctacles-health.service.template`:

```ini
[Unit]
Description={{BRAND_NAME}} Health Check
After=network.target

[Service]
Type=oneshot
User={{SERVICE_USER}}
ExecStart={{INSTALL_PATH}}/app/scripts/health_check.sh
StandardOutput=journal
StandardError=journal
```

**Update** FASE 5 in installer om ook health_check.sh te genereren:

```bash
generate_script "${TEMPLATE_DIR}/health_check.sh.template" "${SCRIPTS_DIR}/health_check.sh"
```

---

## STAP 7: Update .gitignore

Voeg toe aan `.gitignore`:

```gitignore
# Generated health check script
scripts/health_check.sh
```

---

## VERIFICATIE

```bash
# 1. Installer syntax check
bash -n scripts/setup/setup_synctacles_server_v2.3.4.sh

# 2. Alle templates hebben placeholders
grep -c "{{" systemd/*.template
grep -c "{{" systemd/scripts/*.template

# 3. Geen hardcoded paden in templates
grep -E "/opt/synctacles|/opt/energy-insights" systemd/*.template systemd/scripts/*.template
# MOET LEEG ZIJN

# 4. Geen sparkcrawler referenties
grep -r "sparkcrawler" systemd/ config/
# MOET LEEG ZIJN

# 5. settings.py heeft require_env
grep "require_env" config/settings.py

# 6. Health check template bestaat
ls -la systemd/scripts/health_check.sh.template
```

---

## COMMIT

```bash
git add -A
git commit -m "REFACTOR: Installer overhaul with fail-fast config

Phase 3 of Option B debranding - fixes all discovered issues:

FASE 0 fixes:
- .env now uses 'export' statements for all variables
- DATABASE_URL auto-generated from components
- Single .env location at /opt/.env
- ENTSOE_API_KEY warning if not set

FASE 5 fixes:
- Run scripts generated from templates
- Systemd services generated from templates
- Placeholder validation (no unresolved {{}} allowed)
- health_check.sh template added

config/settings.py:
- Fail-fast design: missing required config = immediate error
- No silent fallback defaults
- Clear error messages with remediation steps

Systemd templates:
- All services use generated run scripts
- No hardcoded paths remaining
- TenneT service uses module syntax

Issues fixed:
- .env without export statements
- Missing DATABASE_URL
- Two .env file confusion
- Scripts not generated from templates
- Silent config failures
"
```

---

## ROLLBACK

```bash
git checkout main
git branch -D refactor/installer-overhaul
```

---

## POST-MERGE: Server Update

Na merge naar main, op de server:

```bash
# 1. Pull latest
cd /opt/github/ha-energy-insights-nl
git pull origin main

# 2. Sync code
rsync -av --delete /opt/github/ha-energy-insights-nl/synctacles_db/ /opt/energy-insights-nl/app/synctacles_db/
rsync -av --delete /opt/github/ha-energy-insights-nl/systemd/ /opt/energy-insights-nl/app/systemd/
rsync -av --delete /opt/github/ha-energy-insights-nl/config/ /opt/energy-insights-nl/app/config/

# 3. Generate scripts from templates
cd /opt/energy-insights-nl/app
INSTALL_PATH="/opt/energy-insights-nl"
LOG_PATH="/var/log/energy-insights-nl"

for template in systemd/scripts/*.template; do
    output="scripts/$(basename ${template%.template})"
    sed -e "s|{{INSTALL_PATH}}|${INSTALL_PATH}|g" \
        -e "s|{{LOG_PATH}}|${LOG_PATH}|g" \
        -e "s|{{ENV_FILE}}|/opt/.env|g" \
        "$template" > "$output"
    chmod +x "$output"
done

# 4. Test
/opt/energy-insights-nl/app/scripts/run_collectors.sh
```
