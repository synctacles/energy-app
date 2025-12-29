#!/usr/bin/env bash
# setup_synctacles_server_v2.3.4.sh
# Complete SYNCTACLES Server Installer — Ubuntu 24.04 / Hetzner
#
# CHANGELOG v2.3.4 (2025-12-20)
# - FIX: daemon-reload met info/ok melding vóór enable-loop (cosmetisch)
#
# CHANGELOG v2.3.3 (2025-12-19)
# - ADD: Git safe.directory config for root access
# - ADD: lxml package install in FASE 4
# - FIX: Removed misleading .env symlink message
#
# CHANGELOG v2.3.1 (2025-12-19)
#
# P0 FIXES — Blockers
# - FIX: PYTHONPATH unbound variable — changed "$PYTHONPATH" to "${PYTHONPATH:-}"
#
# CHANGELOG v2.3.0 (2025-12-19)
#
# P0 FIXES — Blockers
# - FIX: Requirements flow — FASE 3 copies requirements.txt from DEV to PROD
# - FIX: FASE 4 installs from $SYNCTACLES_PROD/requirements.txt (not requirements-base.txt)
# - FIX: Gunicorn implementation — synctacles-api.service now uses gunicorn with UvicornWorker
# - FIX: Alembic migration — FASE 5 verifies alembic in venv, runs with PYTHONPATH, exits on failure
#
# P1 FIXES — Production Ready
# - FIX: python3.12-venv explicitly installed in FASE 2
# - FIX: VERSION file validation in FASE 5 (must exist after deploy)
# - FIX: Collector timer copy and enable in FASE 5
#
# P2 FIXES — Polish
# - FIX: pip freeze after install saves to requirements-frozen.txt
# - FIX: Duplicate .env check uses grep -q for SYNCTACLES_LOG_DIR
#
# CHANGELOG v2.2.0 (2025-12-18)
#
# MAJOR: Deploy stap compliance met SKILL 9 (production runtime)
# - Deploy rsync excludes .env en logs/ (voorkomt secret overschrijven)
# - VERSION file check + version tracking in deployment
# - .env symlink naar /opt/synctacles/app/ (collector compatibility)
#
# MAJOR: Log directory standaardisatie (SKILL 9 compliant)
# - Centrale log locatie: /opt/synctacles/logs/
# - Substructuur: api/, scheduler/, collectors/{entso_e_raw,tennet_raw}/, importers/, normalizers/
# - SYNCTACLES_LOG_DIR automatisch toegevoegd aan .env
# - Legacy symlink /var/log/synctacles → /opt/synctacles/logs (backward compatibility)
#
# MAJOR: Systemd unit path validation (kritieke security check)
# - Valideert dat ALLE units /opt/synctacles/app/ gebruiken (NOOIT /opt/github/)
# - Exit 1 bij detectie van DEV repo paths in ExecStart of WorkingDirectory
# - Voorkomt production workloads op git repository
#
# MAJOR: Database migrations (Alembic automation)
# - Automatische `alembic upgrade head` na deploy
# - Schema sync zonder handmatige interventie
# - Error handling voor nieuwe vs. bestaande installaties
#
# MAJOR: Production health check (end-to-end verificatie)
# - API /health endpoint check met version matching
# - Timer count validation (verwacht ≥3 actieve timers)
# - Database connectivity test
# - Version mismatch warning (deployed vs. running)
#
# FIX: Systemd timer copy zonder error bij ontbrekende files
# FIX: Health check wacht 15s voor service stabilization
#
# IMPROVED: Fase 5 exit criteria nu volledig geautomatiseerd
# IMPROVED: Deployment failures nu detecteerbaar via health checks
#
# CHANGELOG V2.1.1 (2025-12-18)
# Fixed (F3.4 GitHub SSH): known_hosts werd root-owned door >> redirection;
# nu correct geschreven als synctacles.
# Fixed (F3.4 GitHub SSH): chown/perms handling aangepast
# (geen chown meer als synctacles → voorkomt Operation not permitted).
# Added (F3.4 GitHub SSH): import flow uitgebreid met "private key only"
# + automatische .pub regeneratie (ssh-keygen -y), naast private+public.
# Improved: strakkere permissies voor GitHub key files en .ssh directory.
# Fixed (F5 systemd): API unit stdout/stderr naar journald gezet om 209/STDOUT te voorkomen
#
# CHANGELOG v2.1.0 (2025-12-17):
# - MAJOR: State-aware SSH key management (root vs synctacles vs GitHub)
#   * Adds installer state file: /var/lib/synctacles-installer/state.env
#   * Root keys: show + fingerprint + safe KEEP/APPEND/REPLACE (backup on change)
#   * Synctacles keys: inherit_root vs custom (diverged) mode, never overwrites after diverge
# - MAJOR: GitHub SSH key separation + enforced verification
#   * GitHub key moved to FASE 3 and stored under synctacles (~/.ssh/id_github)
#   * Interactive: generate or import existing keypair (no mixing with login keys)
#   * Wait loop until GitHub auth is verified (or explicit skip/abort)
#   * Repo clone moved to FASE 3 (runs as synctacles)
# - FIX: .env creation when file does not yet exist (FASE 2.5)
# - SECURITY: remove recursive chmod -R 755 on repo (FASE 6)
#
# CHANGELOG v2.0.4 (2025-12-17):
# - CRITICAL FIX: SSH key handling in fase3
# - Added interactive SSH hardening options
# - Improved fail2ban configuration
# - Added auditd + sudo logging
# - Better directory permissions management
#
# CHANGELOG v2.0.3 (2025-12-17):
# - Merged previous fixes and improved flow
#
# CHANGELOG v2.0.2 (2025-12-15):
# - Added automatic Docker install fix
#
set -euo pipefail

# ========================================================
#   GLOBAL VARIABLES
# ========================================================
SCRIPT_NAME="$(basename "$0")"
# ========================================================
#   BRANDING & ENVIRONMENT VARIABLES
# ========================================================
# Load from /opt/.env (required for fase1-6, generated by fase0)
if [[ -f /opt/.env ]]; then
    source /opt/.env
elif [[ "${1:-}" != "fase0" ]]; then
    # Only fase0 can run without .env (it creates .env)
    echo "❌ ERROR: /opt/.env not found"
    echo "Run: sudo $0 fase0"
    echo "This will prompt for branding configuration and create .env"
    exit 1
fi

# Branding configuration (loaded from .env or fase0)
# No fallback defaults - ensures brand-free repository
# FASE 0 generates .env, other fases require it

# For fase0: use temporary defaults for logging until .env is created
# For fase1-6: variables come from .env
if [[ "${1:-}" == "fase0" ]]; then
    # Temporary logging for fase0 (before .env is created)
    BRAND_SLUG="${BRAND_SLUG:-setup-brand}"
    LOG_DIR="/var/log/setup-temporary"
else
    # fase1-6: use values from .env (already loaded above)
    LOG_DIR="/var/log/${BRAND_SLUG}-setup"
fi

LOG_FILE="$LOG_DIR/setup-$(date +%Y%m%d-%H%M%S).log"

# Paths (fase1-6 only, since fase0 doesn't use these)
if [[ "${1:-}" != "fase0" ]]; then
    SYNCTACLES_PROD="${INSTALL_PATH}"
    SYNCTACLES_DEV="/opt/github/${BRAND_SLUG}-repo"
fi

# Installer state (state-aware reruns)
STATE_DIR="/var/lib/synctacles-installer"
STATE_FILE="$STATE_DIR/state.env"

# GitHub repository (loaded from .env for fase1-6, or derived in fase0)
# Set fallback only for fase0 (which creates .env)
if [[ "${1:-}" == "fase0" ]]; then
    GITHUB_REPO="${GITHUB_REPO:-git@github.com:your-account/your-repo.git}"
fi

# ========================================================
#   LOGGING FUNCTIONS
# ========================================================
setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
    chmod 600 "$LOG_FILE"

    # Redirect all output to both console and log file
    exec > >(tee -a "$LOG_FILE") 2>&1

    echo "======================================================="
    echo "Application Server Setup"
    echo "Started: $(date)"
    echo "Log file: $LOG_FILE"
    echo "======================================================="
    echo
}

# ========================================================
#   HELPER FUNCTIONS
# ========================================================
header() {
    echo
    echo "======================================================="
    echo "$1"
    echo "======================================================="
    echo
}

info() { echo -e "ℹ️  $1"; }
ok()   { echo -e "✅ $1"; }
warn() { echo -e "⚠️  $1"; }
fail() { echo -e "❌ $1"; }

ensure_root() {
    if [[ $EUID -ne 0 ]]; then
        fail "Dit script moet als root draaien. Gebruik: sudo $0"
        exit 1
    fi
}

append_if_not_present() {
    local line="$1"
    local file="$2"
    grep -qxF "$line" "$file" 2>/dev/null || echo "$line" >> "$file"
}

# -----------------------------
# State-aware installer helpers
# -----------------------------
state_init() {
    mkdir -p "$STATE_DIR"
    chmod 700 "$STATE_DIR"
    touch "$STATE_FILE"
    chmod 600 "$STATE_FILE"
}

state_get() {
    local key="$1"
    [[ -f "$STATE_FILE" ]] || { echo ""; return 0; }
    grep -E "^${key}=" "$STATE_FILE" 2>/dev/null | tail -n 1 | cut -d= -f2- || true
}

state_set() {
    local key="$1"
    local value="$2"
    state_init
    if grep -qE "^${key}=" "$STATE_FILE" 2>/dev/null; then
        sed -i "s|^${key}=.*|${key}=${value}|" "$STATE_FILE"
    else
        echo "${key}=${value}" >> "$STATE_FILE"
    fi
}

file_sha256() {
    local f="$1"
    [[ -f "$f" ]] || { echo ""; return 0; }
    sha256sum "$f" | awk '{print $1}'
}

authkeys_has_keys() {
    local f="$1"
    [[ -f "$f" ]] || return 1
    grep -qvE '^[[:space:]]*(#|$)' "$f"
}

backup_file() {
    local f="$1"
    [[ -f "$f" ]] || { echo ""; return 0; }
    local bak="${f}.bak-$(date +%Y%m%d-%H%M%S)"
    cp "$f" "$bak"
    echo "$bak"
}

ensure_ssh_dir_for_user() {
    local user="$1"
    local home="$2"
    mkdir -p "$home/.ssh"
    chmod 700 "$home/.ssh"
    chown "$user:$user" "$home/.ssh"
}

validate_pubkey_line() {
    local line="$1"
    [[ -n "$line" ]] || return 1
    local tmp
    tmp="$(mktemp)"
    printf '%s\n' "$line" > "$tmp"
    ssh-keygen -lf "$tmp" >/dev/null 2>&1
    local rc=$?
    rm -f "$tmp"
    return $rc
}

prompt_pubkey_single_line() {
    local prompt="$1"
    local key=""
    while true; do
        read -rp "$prompt" key
        if [[ -z "$key" ]]; then
            warn "Lege input — probeer opnieuw."
            continue
        fi
        if validate_pubkey_line "$key"; then
            echo "$key"
            return 0
        fi
        warn "Ongeldige SSH public key (ssh-keygen kon hem niet parsen)."
        read -rp "Toch accepteren (op eigen risico)? (y/N): " ANY
        if [[ "${ANY,,}" == "y" ]]; then
            echo "$key"
            return 0
        fi
    done
}

show_authorized_keys() {
    local title="$1"
    local f="$2"
    echo
    echo "---- $title ----"
    if [[ ! -f "$f" ]]; then
        echo "(geen bestand: $f)"
        return 0
    fi
    local nonempty=0
    if authkeys_has_keys "$f"; then
        nonempty=1
    fi
    if [[ $nonempty -eq 0 ]]; then
        echo "(bestand bestaat maar bevat geen keys)"
        return 0
    fi
    # Toon keys (public) — volledig zichtbaar
    cat "$f"
    echo
    echo "Fingerprints:"
    ssh-keygen -lf "$f" 2>/dev/null || echo "(kon fingerprints niet lezen)"
}

github_auth_verified() {
    local output="$1"
    echo "$output" | grep -qi "successfully authenticated"
}

# ========================================================
#   PRE-FLIGHT CHECKS
# ========================================================
preflight_checks() {
    header "Pre-flight checks"

    # Check OS
    if ! grep -q "Ubuntu 24.04" /etc/os-release 2>/dev/null; then
        warn "Script is getest op Ubuntu 24.04. Jij draait: $(grep PRETTY_NAME /etc/os-release | cut -d= -f2)"
        read -rp "Doorgaan? (y/N): " CONTINUE
        [[ "${CONTINUE,,}" == "y" ]] || exit 1
    fi

    ok "OS check passed"

    # Basic tools
    apt-get update -qq >/dev/null 2>&1 || true
    apt-get install -y curl git wget build-essential jq >/dev/null 2>&1

    ok "Basic tools installed"
}

# ========================================================
#   FASE 0 — Brand Configuration (Interactive)
# ========================================================
fase0() {
    header "FASE 0 — Brand Configuration"

    ENV_FILE="/opt/.env"

    # Check if .env already exists
    if [[ -f "$ENV_FILE" ]]; then
        warn ".env already exists at $ENV_FILE"
        read -p "Overwrite? [y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            ok "Using existing .env configuration"
            source "$ENV_FILE"
            return 0
        fi
    fi

    echo ""
    info "Configure branding for this installation"
    echo ""

    # Interactive prompts
    read -p "Brand Name (display): " BRAND_NAME
    read -p "Brand Slug (technical, lowercase-hyphen): " BRAND_SLUG
    read -p "Brand Domain: " BRAND_DOMAIN
    read -p "GitHub Account: " GITHUB_ACCOUNT
    read -p "Repository Name: " REPO_NAME
    read -p "Git Author Email (for commits): " GIT_USER_EMAIL

    # Derived values
    HA_DOMAIN="${BRAND_SLUG//-/_}"  # Replace hyphens with underscores
    SERVICE_USER="$BRAND_SLUG"
    SERVICE_GROUP="$BRAND_SLUG"
    INSTALL_PATH="/opt/$BRAND_SLUG"
    APP_PATH="$INSTALL_PATH/app"
    LOG_PATH="/var/log/$BRAND_SLUG"
    DATA_PATH="/var/lib/$BRAND_SLUG"
    DB_NAME="${BRAND_SLUG//-/_}"
    DB_USER="${BRAND_SLUG//-/_}"
    GITHUB_REPO="git@github.com:${GITHUB_ACCOUNT}/${REPO_NAME}.git"
    GITHUB_REPO_DEV="/opt/github/${REPO_NAME}"
    GIT_USER_NAME="$GITHUB_ACCOUNT"

    # Generate .env
    info "Generating .env at $ENV_FILE..."
    cat > "$ENV_FILE" << EOF
# Generated by setup script on $(date)
# Brand Configuration

## BRANDING & IDENTITY
BRAND_NAME="$BRAND_NAME"
BRAND_SLUG="$BRAND_SLUG"
BRAND_DOMAIN="$BRAND_DOMAIN"

## GITHUB CONFIGURATION
GITHUB_ACCOUNT="$GITHUB_ACCOUNT"
REPO_NAME="$REPO_NAME"
GITHUB_REPO="$GITHUB_REPO"
GITHUB_REPO_DEV="$GITHUB_REPO_DEV"
GIT_USER_NAME="$GIT_USER_NAME"
GIT_USER_EMAIL="$GIT_USER_EMAIL"

## HOME ASSISTANT CONFIGURATION
HA_DOMAIN="$HA_DOMAIN"
HA_COMPONENT_NAME="$BRAND_NAME"

## PATH CONFIGURATION
INSTALL_PATH="$INSTALL_PATH"
APP_PATH="$APP_PATH"
LOG_PATH="$LOG_PATH"
DATA_PATH="$DATA_PATH"

## SERVICE CONFIGURATION
SERVICE_USER="$SERVICE_USER"
SERVICE_GROUP="$SERVICE_GROUP"

## DATABASE CONFIGURATION
DB_NAME="$DB_NAME"
DB_USER="$DB_USER"
DB_HOST="localhost"
DB_PORT="5432"

## API CONFIGURATION
API_HOST="0.0.0.0"
API_PORT="8000"
ADMIN_API_KEY=""
EOF

    chmod 600 "$ENV_FILE"
    ok ".env created at $ENV_FILE"

    # Source it for current script
    source "$ENV_FILE"

    # Generate manifest.json from template
    info "Generating manifest.json from template..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
    TEMPLATE_PATH="$PROJECT_ROOT/custom_components/synctacles/manifest.json.template"
    MANIFEST_PATH="$PROJECT_ROOT/custom_components/synctacles/manifest.json"

    if [[ -f "$TEMPLATE_PATH" ]]; then
        sed -e "s/{{BRAND_SLUG}}/$HA_DOMAIN/g" \
            -e "s/{{BRAND_NAME}}/$BRAND_NAME/g" \
            -e "s/{{GITHUB_ACCOUNT}}/$GITHUB_ACCOUNT/g" \
            -e "s/{{REPO_NAME}}/$REPO_NAME/g" \
            "$TEMPLATE_PATH" > "$MANIFEST_PATH"
        ok "manifest.json generated"
    else
        warn "manifest.json.template not found at $TEMPLATE_PATH, skipping generation"
    fi

    echo ""
    ok "Brand configuration complete"
    echo ""
    echo "Configuration:"
    echo "  Brand:        $BRAND_NAME"
    echo "  Slug:         $BRAND_SLUG"
    echo "  Install Path: $INSTALL_PATH"
    echo "  GitHub:       $GITHUB_ACCOUNT/$REPO_NAME"
    echo ""
}

# ========================================================
#   FASE 1 — System Update + Kernel
# ========================================================
fase1() {
    header "FASE 1 — System Update + Kernel"

    preflight_checks

    info "Update system packages..."
    apt-get update
    apt-get upgrade -y

    ok "System updated"

    # Kernel headers (needed for some packages)
    info "Install kernel headers..."
    apt-get install -y linux-headers-$(uname -r) || true

    ok "Kernel headers installed"

    ok "FASE 1 voltooid — reboot aanbevolen"
    echo
    warn "Als kernel updates zijn geïnstalleerd, reboot nu: sudo reboot"
}

# ========================================================
#   FASE 2 — Software Stack
# ========================================================
fase2() {
    header "FASE 2 — Software Stack (Docker, PostgreSQL, Redis, Monitoring)"

    # NOTE: GitHub SSH key + repo clone is handled in FASE 3 (accounts/SSH) to avoid key mixing.

    # -----------------------------
    # 2.0 Python 3.12 (early install for venv support)
    # -----------------------------
    header "2.0 — Python 3.12 Installation"

    info "Install Python 3.12 with venv support..."
    if ! python3.12 --version >/dev/null 2>&1; then
        apt-get install -y software-properties-common >/dev/null 2>&1
        add-apt-repository ppa:deadsnakes/ppa -y >/dev/null 2>&1 || true
        apt-get update -qq
    fi
    apt-get install -y python3.12 python3.12-venv python3.12-dev >/dev/null 2>&1
    ok "Python 3.12 + venv + dev installed"

    # -----------------------------
    # 2.1 Docker
    # -----------------------------
    header "2.1 — Docker Installatie"

    info "Install Docker..."

    if ! command -v docker >/dev/null 2>&1; then
        apt-get install -y ca-certificates curl gnupg lsb-release >/dev/null 2>&1

        # Add Docker GPG key
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        chmod a+r /etc/apt/keyrings/docker.gpg

        # Add Docker repo
        echo \
          "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
          $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
          tee /etc/apt/sources.list.d/docker.list > /dev/null

        apt-get update -qq
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

        systemctl enable --now docker
        ok "Docker geïnstalleerd en gestart."
    else
        ok "Docker al geïnstalleerd"
    fi

    docker --version && ok "Docker version verified"

    # -----------------------------
    # 2.2 PostgreSQL 16 + TimescaleDB
    # -----------------------------
    header "2.2 — PostgreSQL 16 + TimescaleDB"

    info "Install PostgreSQL 16..."

    if ! command -v psql >/dev/null 2>&1; then
        # Add PostgreSQL repo
        sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
        wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -

        apt-get update -qq
        apt-get install -y postgresql-16 postgresql-client-16

        systemctl enable --now postgresql
        ok "PostgreSQL 16 geïnstalleerd en gestart."
    else
        ok "PostgreSQL al geïnstalleerd"
    fi

    # Install TimescaleDB
    info "Install TimescaleDB..."
    if ! dpkg -l | grep -q timescaledb; then
        # Add TimescaleDB repo
        echo "deb https://packagecloud.io/timescale/timescaledb/ubuntu/ $(lsb_release -cs) main" > /etc/apt/sources.list.d/timescaledb.list
        curl -L https://packagecloud.io/timescale/timescaledb/gpgkey | apt-key add -

        apt-get update -qq
        apt-get install -y timescaledb-2-postgresql-16

        # Configure TimescaleDB
        timescaledb-tune --quiet --yes >/dev/null 2>&1 || true

        systemctl restart postgresql
        ok "TimescaleDB geïnstalleerd en geconfigureerd."
    else
        ok "TimescaleDB al geïnstalleerd"
    fi

    # -----------------------------
    # 2.3 Redis
    # -----------------------------
    header "2.3 — Redis"

    info "Install Redis..."
    if ! command -v redis-server >/dev/null 2>&1; then
        apt-get install -y redis-server

        systemctl enable --now redis-server
        ok "Redis geïnstalleerd en gestart."
    else
        ok "Redis al geïnstalleerd"
    fi

    # -----------------------------
    # 2.4 Monitoring (Grafana + Node Exporter)
    # -----------------------------
    header "2.4 — Monitoring Stack"

    # Grafana
    info "Install Grafana..."
    if ! dpkg -l | grep -q grafana; then
        wget -q -O /usr/share/keyrings/grafana.key https://apt.grafana.com/gpg.key
        echo "deb [signed-by=/usr/share/keyrings/grafana.key] https://apt.grafana.com stable main" > /etc/apt/sources.list.d/grafana.list

        apt-get update -qq
        apt-get install -y grafana

        systemctl enable --now grafana-server
        ok "Grafana geïnstalleerd en gestart."
    else
        ok "Grafana al geïnstalleerd"
    fi

    # Node Exporter
    info "Install Node Exporter..."
    if ! id node_exporter &>/dev/null; then
        useradd --no-create-home --shell /bin/false node_exporter || true

        NODE_EXPORTER_VERSION="1.7.0"
        wget -q "https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
        tar xzf "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
        cp "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter" /usr/local/bin/
        chown node_exporter:node_exporter /usr/local/bin/node_exporter

        cat >/etc/systemd/system/node_exporter.service <<EOF
[Unit]
Description=Node Exporter
After=network.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter
Restart=always

[Install]
WantedBy=multi-user.target
EOF

        systemctl daemon-reload
        systemctl enable --now node_exporter

        # Cleanup
        rm -rf "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64"*

        ok "Node Exporter geïnstalleerd en gestart."
    else
        ok "Node Exporter al geïnstalleerd"
    fi

    if systemctl status node_exporter --no-pager 2>/dev/null | grep -q running; then
        ok "Node Exporter draait"
    else
        warn "Node Exporter lijkt niet te draaien. Check: systemctl status node_exporter"
    fi

    # Continue in next part...
    fase2_database
}

# ========================================================
#   FASE 2.5 — Database Initialization + API Keys
# ========================================================
fase2_database() {
    header "FASE 2.5 — Database Setup"

    # Source ENV if available (allows runtime override)
    if [[ -f /root/.env ]]; then
        source /root/.env
    elif [[ -f /opt/.env ]]; then
        source /opt/.env
    fi

    # Environment variables loaded from /opt/.env (created by fase0)
    # DB_NAME and DB_USER use underscores (from .env: energy_insights_nl)
    # SERVICE_USER may have hyphens (from .env: energy-insights-nl)
    # PostgreSQL doesn't accept hyphens in database names
    # Therefore: NEVER override .env values here - use them as-is

    # Check of database al bestaat
    if ! sudo -u postgres psql -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
        info "Database '$DB_NAME' aanmaken (development mode - geen wachtwoord)..."

        sudo -u postgres psql <<EOF
CREATE DATABASE ${DB_NAME};
CREATE USER ${DB_USER};
ALTER DATABASE ${DB_NAME} OWNER TO ${DB_USER};
GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
\c ${DB_NAME}
CREATE EXTENSION IF NOT EXISTS timescaledb;
EOF

        ok "Database aangemaakt (user: ${SERVICE_USER}, geen wachtwoord)."
    else
        ok "Database '$DB_NAME' bestaat al."
    fi

    # PostgreSQL authentication configureren (trust voor lokaal)
    PG_HBA="/etc/postgresql/16/main/pg_hba.conf"

    info "PostgreSQL authentication configureren..."

    # Backup
    cp "$PG_HBA" "$PG_HBA.bak-$(date +%Y%m%d-%H%M%S)"

    # Add trust rules if not present
    if ! grep -q "local.*${SERVICE_USER}.*${SERVICE_USER}.*trust" "$PG_HBA"; then
        # Add BEFORE the default local all postgres peer line
        sed -i '/^local.*all.*postgres.*peer/i\
# ${BRAND_NAME} Database Access (Development - Trust)\
local   ${SERVICE_USER}      ${SERVICE_USER}                              trust\
host    ${SERVICE_USER}      ${SERVICE_USER}      127.0.0.1/32            trust\
host    ${SERVICE_USER}      ${SERVICE_USER}      ::1/128                 trust' "$PG_HBA"

        systemctl reload postgresql
        ok "PostgreSQL authentication bijgewerkt (trust voor ${SERVICE_USER} user)."
    else
        ok "PostgreSQL authentication al geconfigureerd."
    fi

    # Test database connectie
    if sudo -u postgres psql "${DB_NAME}" -c "SELECT version();" >/dev/null 2>&1; then
        ok "Database connectie geverifieerd."
    else
        fail "Kan niet verbinden met database."
        exit 1
    fi

    # -----------------------------
    # .env configuration (create or overwrite)
    # -----------------------------
    ENV_FILE="$SYNCTACLES_PROD/.env"
    mkdir -p "$SYNCTACLES_PROD"

    CREATE_ENV=1
    if [[ -f "$ENV_FILE" ]]; then
        warn ".env file already exists: $ENV_FILE"
        echo

        # Show existing keys (masked)
        if grep -q "^ENTSOE_API_KEY=." "$ENV_FILE" 2>/dev/null; then
            EXISTING_ENTSOE=$(grep "^ENTSOE_API_KEY=" "$ENV_FILE" | cut -d= -f2-)
            echo "  ENTSO-E key: ${EXISTING_ENTSOE:0:8}...${EXISTING_ENTSOE: -4}"
        else
            echo "  ENTSO-E key: (not set)"
        fi

        if grep -q "^TENNET_API_KEY=." "$ENV_FILE" 2>/dev/null; then
            EXISTING_TENNET=$(grep "^TENNET_API_KEY=" "$ENV_FILE" | cut -d= -f2-)
            echo "  TenneT key:  ${EXISTING_TENNET:0:8}...${EXISTING_TENNET: -4}"
        else
            echo "  TenneT key:  (not set)"
        fi

        echo
        read -rp "Overwrite .env file with new values? (y/N): " OVERWRITE_ENV
        if [[ "${OVERWRITE_ENV,,}" != "y" ]]; then
            ok ".env file preserved (handmatig te wijzigen: $ENV_FILE)"
            CREATE_ENV=0
        else
            warn ".env file wordt overschreven (backup wordt gemaakt)"
            bak="$(backup_file "$ENV_FILE")"
            [[ -n "$bak" ]] && ok "Backup: $bak"
            CREATE_ENV=1
        fi
    fi

    if [[ "$CREATE_ENV" -eq 1 ]]; then
        # -----------------------------
        # Interactive API Key Input
        # -----------------------------
        header "API Keys Configuration"

        info "Je hebt API keys nodig voor:"
        echo "  1. ENTSO-E Transparency Platform (electricity data) — REQUIRED voor SparkCrawler"
        echo "  2. TenneT (settlement prices) — optioneel"
        echo
        info "Tip: Je kunt deze later ook handmatig wijzigen in $ENV_FILE"
        echo

        # ENTSO-E API Key (required for collectors)
        ENTSOE_KEY=""
        while true; do
            read -rp "ENTSO-E API key (laat leeg om te skippen): " ENTSOE_INPUT
            if [[ -z "$ENTSOE_INPUT" ]]; then
                warn "ENTSO-E API key overgeslagen — SparkCrawler werkt niet zonder key!"
                read -rp "Toch doorgaan zonder ENTSO-E key? (y/N): " SKIP_KEY
                if [[ "${SKIP_KEY,,}" == "y" ]]; then
                    break
                fi
            else
                if [[ "$ENTSOE_INPUT" =~ ^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$ ]]; then
                    ENTSOE_KEY="$ENTSOE_INPUT"
                    ok "ENTSO-E API key accepted (UUID format OK)"
                    break
                else
                    warn "API key format lijkt incorrect (verwacht: UUID)."
                    read -rp "Toch gebruiken? (y/N): " USE_ANYWAY
                    if [[ "${USE_ANYWAY,,}" == "y" ]]; then
                        ENTSOE_KEY="$ENTSOE_INPUT"
                        break
                    fi
                fi
            fi
        done

        TENNET_KEY=""
        read -rp "TenneT API key (optioneel, leeg = skip): " TENNET_INPUT
        if [[ -n "$TENNET_INPUT" ]]; then
            TENNET_KEY="$TENNET_INPUT"
            ok "TenneT API key opgeslagen"
        else
            info "TenneT API key overgeslagen"
        fi

        # -----------------------------
        # Write .env file (root-owned for now; ownership fixed in FASE 3)
        # -----------------------------
        info "Aanmaken van .env bestand..."

        cat > "$ENV_FILE" <<EOF
# ========================================================
#   SYNCTACLES Environment Configuration
#   Generated: $(date)
# ========================================================

# Database Configuration (Development - No Password)
DATABASE_URL=postgresql://${SERVICE_USER}@localhost:5432/${DB_NAME}
DB_HOST=localhost
DB_PORT=5432
DB_NAME=${DB_NAME}
DB_USER=${SERVICE_USER}
DB_PASSWORD=

# API Keys
ENTSOE_API_KEY=${ENTSOE_KEY}
TENNET_API_KEY=${TENNET_KEY}

# API Configuration
SECRET_KEY=$(openssl rand -hex 32)
API_HOST=0.0.0.0
API_PORT=8000

# Environment
ENVIRONMENT=development
LOG_LEVEL=DEBUG

# CORS (voor Home Assistant integration) - MUST be valid JSON
CORS_ORIGINS=["*"]

# Rate Limiting (disabled in development)
RATE_LIMIT_ENABLED=false
RATE_LIMIT_FREE_TIER=100

# SparkCrawler Configuration
FETCH_INTERVAL_SECONDS=300
EOF

        chmod 600 "$ENV_FILE"
        chown root:root "$ENV_FILE"
        ok "Environment configuratie opgeslagen in $ENV_FILE"
        warn "Let op: owner staat nu op root; wordt gefixt in FASE 3 (synctacles user)."

        if [[ -z "$ENTSOE_KEY" ]]; then
            warn "Let op: geen ENTSO-E API key — vul deze later aan in $ENV_FILE"
        fi

        echo
        info "Database credentials:"
        echo "  Database: ${DB_NAME}"
        echo "  User:     ${SERVICE_USER}"
        echo "  Password: (geen - trust authentication)"
        echo "  URL:      postgresql://${SERVICE_USER}@localhost:5432/${DB_NAME}"
        echo
    fi

    # -----------------------------
    # Firewall Info
    # -----------------------------
    header "Firewall Configuration"

    info "Firewall wordt beheerd via Hetzner Cloud Firewall."
    info "Configureer via: https://console.hetzner.cloud → Firewalls"
    echo
    info "Aanbevolen regels:"
    echo "  - SSH (22)       → Allow from: Jouw thuis IP"
    echo "  - HTTPS (443)    → Allow from: 0.0.0.0/0"
    echo "  - HTTP (80)      → Allow from: 0.0.0.0/0 (redirect naar 443)"
    echo "  - Grafana (3000) → Allow from: Jouw thuis IP (optioneel)"
    echo

    ok "FASE 2 voltooid — je kunt nu FASE 3 draaien."
}

# ========================================================
#   FASE 3 — Accounts, SSH Keys, Security
# ========================================================
fase3() {
    header "FASE 3 — Security: Accounts, SSH Keys, Hardening"

    state_init
    state_set "LAST_RUN_VERSION" "2.3.3"

    ROOT_AUTH="/root/.ssh/authorized_keys"
    SYN_HOME="/home/${SERVICE_USER}"
    SYN_AUTH="$SYN_HOME/.ssh/authorized_keys"
    GITHUB_KEY="$SYN_HOME/.ssh/id_github"
    GITHUB_PUB="$SYN_HOME/.ssh/id_github.pub"

    # -----------------------------
    # 3.1 Root SSH Login Keys (state-aware)
    # -----------------------------
    header "3.1 — Root SSH Login Keys (state-aware)"

    mkdir -p /root/.ssh
    chmod 700 /root/.ssh

    show_authorized_keys "ROOT: /root/.ssh/authorized_keys" "$ROOT_AUTH"

    echo
    echo "Actie voor root keys:"
    echo "  1) Keep (geen wijzigingen)"
    echo "  2) Append (extra login key toevoegen — behoudt Hetzner key)"
    echo "  3) Replace (gevaarlijk — vervangt alle root keys)"
    read -rp "Keuze (1/2/3) [1]: " ROOT_CHOICE
    ROOT_CHOICE="${ROOT_CHOICE:-1}"

    case "$ROOT_CHOICE" in
        1)
            ok "Root keys: KEEP"
            ;;
        2)
            info "Plak een EXTRA SSH public key voor root login (1 regel):"
            NEW_ROOT_KEY="$(prompt_pubkey_single_line "Public key: ")"
            mkdir -p /root/.ssh
            chmod 700 /root/.ssh
            if [[ -f "$ROOT_AUTH" ]]; then
                bak="$(backup_file "$ROOT_AUTH")"
                [[ -n "$bak" ]] && ok "Backup: $bak"
            fi
            touch "$ROOT_AUTH"
            chmod 600 "$ROOT_AUTH"
            # append only if not already present
            if grep -qxF "$NEW_ROOT_KEY" "$ROOT_AUTH" 2>/dev/null; then
                ok "Key bestond al in root authorized_keys (geen duplicate toegevoegd)."
            else
                echo "$NEW_ROOT_KEY" >> "$ROOT_AUTH"
                ok "Extra root login key toegevoegd."
            fi
            ;;
        3)
            warn "REPLACE betekent: alle bestaande root keys verdwijnen (lockout risico!)."
            warn "Dit mag ALLEEN als je 100% zeker bent dat je nieuwe key werkt."
            read -rp "Type EXACT: REPLACE om door te gaan: " CONFIRM_REPLACE
            if [[ "$CONFIRM_REPLACE" != "REPLACE" ]]; then
                warn "REPLACE geannuleerd — root keys blijven ongewijzigd."
            else
                info "Plak de NIEUWE SSH public key voor root login (1 regel):"
                NEW_ROOT_KEY="$(prompt_pubkey_single_line "Public key: ")"
                if [[ -f "$ROOT_AUTH" ]]; then
                    bak="$(backup_file "$ROOT_AUTH")"
                    [[ -n "$bak" ]] && ok "Backup: $bak"
                fi
                mkdir -p /root/.ssh
                chmod 700 /root/.ssh
                printf '%s\n' "$NEW_ROOT_KEY" > "$ROOT_AUTH"
                chmod 600 "$ROOT_AUTH"
                ok "Root authorized_keys vervangen (REPLACE)."
            fi
            ;;
        *)
            warn "Onbekende keuze — root keys blijven ongewijzigd."
            ;;
    esac

    # Persist root state
    state_set "ROOT_AUTH_KEYS_SHA256" "$(file_sha256 "$ROOT_AUTH")"

    # Hard guard: root must have at least one key before we even think about disabling root login later
    if ! authkeys_has_keys "$ROOT_AUTH"; then
        warn "Root heeft GEEN login keys (authorized_keys leeg of ontbreekt)."
        warn "SSH hardening (PermitRootLogin no) is onveilig tot je dit oplost."
    fi

    # -----------------------------
    # 3.2 Service User Account
    # -----------------------------
    header "3.2 — Service User Account (${SERVICE_USER})"

    info "Maak ${SERVICE_USER} user aan..."
    if ! id "${SERVICE_USER}" >/dev/null 2>&1; then
        adduser --system --group --home "/home/${SERVICE_USER}" --shell /bin/bash "${SERVICE_USER}"
        ok "User '${SERVICE_USER}' aangemaakt."
    else
        ok "User '${SERVICE_USER}' bestaat al."
    fi

    # Add to docker group
    if ! groups "${SERVICE_USER}" | grep -q docker; then
        usermod -aG docker "${SERVICE_USER}"
        ok "${SERVICE_USER} toegevoegd aan docker group."
    fi

    # Ensure SSH dir
    ensure_ssh_dir_for_user "${SERVICE_USER}" "$SYN_HOME"

    # Fix ownership for .env so services can read it
    if [[ -f "$SYNCTACLES_PROD/.env" ]]; then
        chown "${SERVICE_USER}:${SERVICE_GROUP}" "$SYNCTACLES_PROD/.env" 2>/dev/null || true
        chmod 600 "$SYNCTACLES_PROD/.env" 2>/dev/null || true
        ok ".env ownership/perms gezet op ${SERVICE_USER} (600)"
    fi

    # -----------------------------
    # 3.3 Synctacles SSH Login Keys (inherit vs custom)
    # -----------------------------
    header "3.3 — Synctacles SSH Login Keys (inherit_root vs custom)"

    # Determine / infer mode
    MODE="$(state_get "SYNCTACLES_KEY_MODE")"
    if [[ -z "$MODE" ]]; then
        if authkeys_has_keys "$SYN_AUTH" && authkeys_has_keys "$ROOT_AUTH"; then
            if [[ "$(file_sha256 "$SYN_AUTH")" == "$(file_sha256 "$ROOT_AUTH")" ]]; then
                MODE="inherit_root"
            else
                MODE="custom"
            fi
        else
            MODE="inherit_root"
        fi
        state_set "SYNCTACLES_KEY_MODE" "$MODE"
    fi

    info "Current ${SERVICE_USER} key mode: $MODE"
    show_authorized_keys "${SERVICE_USER} authorized keys: $SYN_AUTH" "$SYN_AUTH"

    if ! authkeys_has_keys "$SYN_AUTH"; then
        echo
        if authkeys_has_keys "$ROOT_AUTH"; then
            read -rp "Synctacles keys leeg. Root keys overnemen (inherit)? (Y/n): " INHERIT
            if [[ "${INHERIT,,}" != "n" ]]; then
                bak="$(backup_file "$SYN_AUTH")"
                [[ -n "$bak" ]] && ok "Backup: $bak"
                cp "$ROOT_AUTH" "$SYN_AUTH"
                chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYN_AUTH"
                chmod 600 "$SYN_AUTH"
                MODE="inherit_root"
                state_set "SYNCTACLES_KEY_MODE" "$MODE"
                ok "Synctacles keys overgenomen van root (inherit_root)."
            else
                info "Plak een SSH public key voor ${SERVICE_USER} login (1 regel):"
                NEW_SYN_KEY="$(prompt_pubkey_single_line "Public key: ")"
                bak="$(backup_file "$SYN_AUTH")"
                [[ -n "$bak" ]] && ok "Backup: $bak"
                printf '%s\n' "$NEW_SYN_KEY" > "$SYN_AUTH"
                chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYN_AUTH"
                chmod 600 "$SYN_AUTH"
                MODE="custom"
                state_set "SYNCTACLES_KEY_MODE" "$MODE"
                ok "Synctacles key ingesteld (custom)."
            fi
        else
            warn "Root keys ontbreken/leeg, inherit is niet mogelijk."
            info "Plak een SSH public key voor ${SERVICE_USER} login (1 regel):"
            NEW_SYN_KEY="$(prompt_pubkey_single_line "Public key: ")"
            printf '%s\n' "$NEW_SYN_KEY" > "$SYN_AUTH"
            chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYN_AUTH"
            chmod 600 "$SYN_AUTH"
            MODE="custom"
            state_set "SYNCTACLES_KEY_MODE" "$MODE"
            ok "Synctacles key ingesteld (custom)."
        fi
    else
        echo
        echo "Actie voor ${SERVICE_USER} keys:"
        echo "  1) Keep (geen wijzigingen)"
        echo "  2) Append (extra key toevoegen)  → mode wordt custom"
        echo "  3) Replace (vervang alle keys)   → mode wordt custom"
        if [[ "$MODE" == "inherit_root" ]]; then
            echo "  4) Sync from root (overschrijft ${SERVICE_USER} keys; alleen in inherit_root)"
        fi
        read -rp "Keuze [1]: " SYN_CHOICE
        SYN_CHOICE="${SYN_CHOICE:-1}"

        case "$SYN_CHOICE" in
            1)
                ok "Synctacles keys: KEEP"
                ;;
            2)
                info "Plak een EXTRA SSH public key voor synctacles login (1 regel):"
                NEW_SYN_KEY="$(prompt_pubkey_single_line "Public key: ")"
                bak="$(backup_file "$SYN_AUTH")"
                [[ -n "$bak" ]] && ok "Backup: $bak"
                touch "$SYN_AUTH"
                if grep -qxF "$NEW_SYN_KEY" "$SYN_AUTH" 2>/dev/null; then
                    ok "Key bestond al (geen duplicate)."
                else
                    echo "$NEW_SYN_KEY" >> "$SYN_AUTH"
                    ok "Extra ${SERVICE_USER} login key toegevoegd."
                fi
                chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYN_AUTH"
                chmod 600 "$SYN_AUTH"
                MODE="custom"
                state_set "SYNCTACLES_KEY_MODE" "$MODE"
                ok "Mode switched → custom (diverged). Root mag dit nooit meer overschrijven."
                ;;
            3)
                warn "REPLACE betekent: alle bestaande ${SERVICE_USER} keys verdwijnen (lockout risico!)."
                read -rp "Type EXACT: REPLACE om door te gaan: " CONFIRM_SYN_REPLACE
                if [[ "$CONFIRM_SYN_REPLACE" != "REPLACE" ]]; then
                    warn "REPLACE geannuleerd — ${SERVICE_USER} keys blijven ongewijzigd."
                else
                    info "Plak de NIEUWE SSH public key voor synctacles login (1 regel):"
                    NEW_SYN_KEY="$(prompt_pubkey_single_line "Public key: ")"
                    bak="$(backup_file "$SYN_AUTH")"
                    [[ -n "$bak" ]] && ok "Backup: $bak"
                    printf '%s\n' "$NEW_SYN_KEY" > "$SYN_AUTH"
                    chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYN_AUTH"
                    chmod 600 "$SYN_AUTH"
                    MODE="custom"
                    state_set "SYNCTACLES_KEY_MODE" "$MODE"
                    ok "Synctacles authorized_keys vervangen (custom). Root mag dit nooit meer overschrijven."
                fi
                ;;
            4)
                if [[ "$MODE" != "inherit_root" ]]; then
                    warn "Sync from root is alleen toegestaan in inherit_root mode."
                else
                    if ! authkeys_has_keys "$ROOT_AUTH"; then
                        warn "Root keys leeg — sync niet mogelijk."
                    else
                        warn "Sync from root overschrijft ${SERVICE_USER} authorized_keys."
                        read -rp "Doorgaan met sync? (y/N): " DO_SYNC
                        if [[ "${DO_SYNC,,}" == "y" ]]; then
                            bak="$(backup_file "$SYN_AUTH")"
                            [[ -n "$bak" ]] && ok "Backup: $bak"
                            cp "$ROOT_AUTH" "$SYN_AUTH"
                            chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYN_AUTH"
                            chmod 600 "$SYN_AUTH"
                            ok "Synctacles keys gesynchroniseerd met root (inherit_root)."
                        else
                            info "Sync overgeslagen."
                        fi
                    fi
                fi
                ;;
            *)
                warn "Onbekene keuze — ${SERVICE_USER} keys blijven ongewijzigd."
                ;;
        esac
    fi

    state_set "SYNCTACLES_AUTH_KEYS_SHA256" "$(file_sha256 "$SYN_AUTH")"

    echo
    ok "✓ Je kunt nu inloggen met: ssh ${SERVICE_USER}@$(hostname -I | awk '{print $1}')"
    warn "⚠️  TEST altijd ${SERVICE_USER} login voordat je root-login uitschakelt!"

    # -----------------------------
    # 3.4 GitHub SSH Key (separate) + Repo Clone (runs as ${SERVICE_USER})
    # -----------------------------
    header "3.4 — GitHub Access (separate key, ${SERVICE_USER} user)"

    # Ensure /opt/github exists
    mkdir -p /opt/github
    chmod 755 /opt/github
    chown ${SERVICE_USER}:${SERVICE_GROUP} /opt/github

    # Git author (for ${SERVICE_USER} user, from .env)
    sudo -u ${SERVICE_USER} git config --global user.name "${GIT_USER_NAME}" >/dev/null 2>&1 || true
    sudo -u ${SERVICE_USER} git config --global user.email "${GIT_USER_EMAIL}" >/dev/null 2>&1 || true

    # GitHub key flow (SEPARATE from login keys)
    KEY_EXISTS=0
    if [[ -f "$GITHUB_KEY" && -f "$GITHUB_PUB" ]]; then
        KEY_EXISTS=1
    fi

    if [[ "$KEY_EXISTS" -eq 1 ]]; then
        ok "GitHub SSH key bestaat al: $GITHUB_KEY"
        echo
        info "GitHub public key (${SERVICE_USER}):"
        echo "=================================================="
        cat "$GITHUB_PUB"
        echo "=================================================="
        echo
        echo "Actie voor GitHub key:"
        echo "  1) Keep"
        echo "  2) Regenerate (nieuwe keypair)"
        echo "  3) Import existing key (private only / keypair)"
        read -rp "Keuze [1]: " GH_CHOICE
        GH_CHOICE="${GH_CHOICE:-1}"
        case "$GH_CHOICE" in
            2) GH_ACTION="regen" ;;
            3) GH_ACTION="import" ;;
            *) GH_ACTION="keep" ;;
        esac
    else
        warn "Geen GitHub SSH key gevonden voor ${SERVICE_USER}."
        echo "Actie voor GitHub key:"
        echo "  1) Generate (recommended)"
        echo "  2) Import existing key (private only / keypair)"
        read -rp "Keuze [1]: " GH_CHOICE
        GH_CHOICE="${GH_CHOICE:-1}"
        case "$GH_CHOICE" in
            2) GH_ACTION="import" ;;
            *) GH_ACTION="gen" ;;
        esac
    fi

    ensure_ssh_dir_for_user "${SERVICE_USER}" "$SYN_HOME"

    case "$GH_ACTION" in
        keep)
            ok "GitHub key: KEEP"
            ;;
        gen)
            sudo -u ${SERVICE_USER} ssh-keygen -t ed25519 -f "$GITHUB_KEY" -N "" -C "github@$(hostname)" >/dev/null
            chown ${SERVICE_USER}:${SERVICE_GROUP} "$GITHUB_KEY" "$GITHUB_PUB"
            chmod 600 "$GITHUB_KEY"
            chmod 644 "$GITHUB_PUB"
            ok "GitHub keypair gegenereerd."
            ;;
        regen)
            warn "Regenerate: oude key wordt onbruikbaar in GitHub zodra je hem vervangt."
            if [[ -f "$GITHUB_KEY" ]]; then mv "$GITHUB_KEY" "${GITHUB_KEY}.old-$(date +%Y%m%d-%H%M%S)" || true; fi
            if [[ -f "$GITHUB_PUB" ]]; then mv "$GITHUB_PUB" "${GITHUB_PUB}.old-$(date +%Y%m%d-%H%M%S)" || true; fi
            sudo -u ${SERVICE_USER} ssh-keygen -t ed25519 -f "$GITHUB_KEY" -N "" -C "github@$(hostname)" >/dev/null
            chown ${SERVICE_USER}:${SERVICE_GROUP} "$GITHUB_KEY" "$GITHUB_PUB"
            chmod 600 "$GITHUB_KEY"
            chmod 644 "$GITHUB_PUB"
            ok "Nieuwe GitHub keypair gegenereerd."
            ;;
        import)
            warn "Import GitHub SSH key: je gaat nu PRIVATE KEY materiaal plakken op de server."
            warn "Gebruik dit alleen als je weet wat je doet."
            echo
            echo "Import mode:"
            echo "  1) Private key only (recommended) — public key wordt automatisch geregenereerd"
            echo "  2) Private + public key (je plakt ook de public key)"
            read -rp "Keuze [1]: " GH_IMP_MODE
            GH_IMP_MODE="${GH_IMP_MODE:-1}"

            echo
            info "Plak nu je PRIVATE KEY (multiline). Eindig met een enkele regel: END"
            tmp_priv="$(mktemp)"
            : > "$tmp_priv"
            while IFS= read -r line; do
                [[ "$line" == "END" ]] && break
                printf '%s\n' "$line" >> "$tmp_priv"
            done
            mv "$tmp_priv" "$GITHUB_KEY"
            if ! chown ${SERVICE_USER}:${SERVICE_GROUP} "$GITHUB_KEY"; then
                fail "Kan ownership niet wijzigen naar synctacles (draait script als root?)"
                rm -f "$GITHUB_KEY"
                exit 1
            fi
            chmod 600 "$GITHUB_KEY"

            if [[ "$GH_IMP_MODE" == "2" ]]; then
                echo
                info "Plak nu je PUBLIC KEY (1 regel)."
                PUB_LINE="$(prompt_pubkey_single_line "Public key: ")"
                printf '%s\n' "$PUB_LINE" > "$GITHUB_PUB"
            else
                echo
                info "Public key wordt geregenereerd uit private key..."
                # Run as synctacles so redirection happens with correct ownership
                if ! sudo -u ${SERVICE_USER} bash -lc 'ssh-keygen -y -f ~/.ssh/id_github > ~/.ssh/id_github.pub' >/dev/null 2>&1; then
                    fail "Kon public key niet regenereren uit private key (mogelijk verkeerde key of passphrase nodig)."
                    fail "Tip: run handmatig als synctacles: ssh-keygen -y -f ~/.ssh/id_github > ~/.ssh/id_github.pub"
                    exit 1
                fi
            fi

            chown ${SERVICE_USER}:${SERVICE_GROUP} "$GITHUB_PUB" 2>/dev/null || true
            chmod 644 "$GITHUB_PUB" 2>/dev/null || true

            ok "GitHub SSH key geïmporteerd."
            ;;

        *)
            fail "Interne fout: onbekende GH_ACTION=$GH_ACTION"
            exit 1
            ;;
    esac

    # Configure SSH for GitHub (synctacles user)
    sudo -u ${SERVICE_USER} bash -lc 'mkdir -p ~/.ssh && chmod 700 ~/.ssh'
    if ! sudo -u ${SERVICE_USER} bash -lc 'grep -q "^Host github.com" ~/.ssh/config 2>/dev/null'; then
        sudo -u ${SERVICE_USER} bash -lc "cat >> ~/.ssh/config << 'EOF'

Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_github
    IdentitiesOnly yes
EOF"
        sudo -u ${SERVICE_USER} chmod 600 "$SYN_HOME/.ssh/config"
        sudo -u ${SERVICE_USER} chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYN_HOME/.ssh/config"
        ok "SSH config (synctacles) updated for GitHub"
    else
        ok "SSH config (synctacles) already has github.com host block"
    fi

    # Add github.com to known_hosts as synctacles (redirection must happen as synctacles)
    if ! sudo -u ${SERVICE_USER} -H ssh-keygen -F github.com >/dev/null 2>&1; then
        sudo -u ${SERVICE_USER} -H bash -lc 'ssh-keyscan -t ed25519 github.com >> ~/.ssh/known_hosts' 2>/dev/null || true
    fi

    # Fix perms as root (NOT as synctacles)
    chown ${SERVICE_USER}:${SERVICE_GROUP} /home/synctacles/.ssh/known_hosts 2>/dev/null || true
    chmod 644 /home/synctacles/.ssh/known_hosts 2>/dev/null || true

    # Show public key + instructions
    echo
    echo "=========================================="
    echo "GITHUB SSH KEY SETUP REQUIRED"
    echo "=========================================="
    echo
    echo "GitHub public key (${SERVICE_USER}):"
    echo "=================================================="
    cat "$GITHUB_PUB"
    echo "=================================================="
    echo
    echo "Next steps:"
    echo "1) Voeg deze key toe in GitHub: Settings → SSH and GPG keys → New SSH key"
    echo "2) Title: 'Application Server ($(hostname))'"
    echo

    # Wait until key works (default: retry)
    while true; do
        OUT="$(sudo -u ${SERVICE_USER} ssh -o BatchMode=yes -T git@github.com 2>&1 || true)"
        if github_auth_verified "$OUT"; then
            ok "GitHub SSH connection verified ✓"
            break
        fi
        warn "GitHub SSH nog niet geautoriseerd."
        warn "Output: $OUT"
        echo
        read -rp "ENTER=retry, s=skip (risk), q=abort: " GH_WAIT
        if [[ "${GH_WAIT,,}" == "q" ]]; then
            fail "Afgebroken: GitHub auth niet verified."
            exit 1
        elif [[ "${GH_WAIT,,}" == "s" ]]; then
            warn "Skip gekozen — GitHub clone/pull kan mislukken tot key werkt."
            break
        fi
    done

    # Clone repo (as service user) - check for .git to verify actual repo clone
    if [[ ! -d "${GITHUB_REPO_DEV}/.git" ]]; then
        info "Cloning application repository (as ${SERVICE_USER})..."
        if sudo -u ${SERVICE_USER} git clone "$GITHUB_REPO" "${GITHUB_REPO_DEV}" 2>/dev/null; then
            ok "Repository cloned: ${GITHUB_REPO_DEV}"
            # Allow root to access repo
            git config --global --add safe.directory "${GITHUB_REPO_DEV}"
            ok "Git safe.directory configured for root"
        else
            warn "GitHub clone failed (nog geen toegang of repo niet bereikbaar)."
            warn "Manual step (as ${SERVICE_USER}):"
            warn "  sudo -u ${SERVICE_USER} git clone $GITHUB_REPO ${GITHUB_REPO_DEV}"
        fi
    else
        ok "Repository already exists: ${GITHUB_REPO_DEV}"
        info "KIES workflow: wijzigingen altijd in GitHub, daarna op server: sudo -u ${SERVICE_USER} git -C ${GITHUB_REPO_DEV} pull origin main"
    fi

    # Repo ownership sanity
    if [[ -d "${GITHUB_REPO_DEV}" ]]; then
        chown -R ${SERVICE_USER}:${SERVICE_GROUP} "${GITHUB_REPO_DEV}" 2>/dev/null || true
        ok "Development directory ownership correct"
    fi

    # ====================================
    # 3.5 Deploy Code to Production
    # ====================================
    info "Deploying code from ${GITHUB_REPO_DEV} to ${INSTALL_PATH}/app/..."

    # Create app directory
    mkdir -p "${INSTALL_PATH}/app"

    # Deploy with rsync (always runs, regardless of clone/pull)
    if rsync -a --delete \
        --exclude '.git' \
        --exclude '.github' \
        --exclude '.claude' \
        --exclude '__pycache__' \
        --exclude '*.pyc' \
        --exclude '.env' \
        --exclude 'venv' \
        --exclude '.venv' \
        --exclude 'logs/' \
        --exclude '*.md' \
        --exclude 'tests/' \
        "${GITHUB_REPO_DEV}/" "${INSTALL_PATH}/app/" >/dev/null 2>&1; then

        # Set ownership
        chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "${INSTALL_PATH}/app"
        ok "App deployed: ${INSTALL_PATH}/app"

        # Verify critical files were deployed
        if [[ -f "${INSTALL_PATH}/app/requirements.txt" ]] && [[ -f "${INSTALL_PATH}/app/start_api.py" ]]; then
            info "Critical files verified:"
            [[ -f "${INSTALL_PATH}/app/requirements.txt" ]] && echo "  ✓ requirements.txt"
            [[ -f "${INSTALL_PATH}/app/start_api.py" ]] && echo "  ✓ start_api.py"
            [[ -d "${INSTALL_PATH}/app/synctacles_db" ]] && echo "  ✓ synctacles_db/"
            [[ -d "${INSTALL_PATH}/app/sparkcrawler_db" ]] && echo "  ✓ sparkcrawler_db/"
        else
            warn "Some expected files missing in deployment (may be OK if structure differs)"
        fi
    else
        error "Rsync deployment failed"
        error "Check permissions on ${INSTALL_PATH}"
        exit 1
    fi

    # Store deployment state
    state_set "LAST_DEPLOYMENT_HASH" "$(cd ${GITHUB_REPO_DEV} && git rev-parse HEAD 2>/dev/null || echo 'unknown')"

    # ====================================
    # 3.6 Generate alembic.ini from Template
    # ====================================
    info "Generating alembic.ini from template..."

    ALEMBIC_TEMPLATE="${INSTALL_PATH}/app/alembic.ini.template"
    ALEMBIC_CONFIG="${INSTALL_PATH}/app/alembic.ini"

    if [[ ! -f "${ALEMBIC_TEMPLATE}" ]]; then
        error "Template not found: ${ALEMBIC_TEMPLATE}"
        error "Repository may be outdated - check git status"
        exit 1
    fi

    # Generate alembic.ini with database credentials from .env
    sed -e "s|{{DB_USER}}|${DB_USER}|g" \
        -e "s|{{DB_PASSWORD}}|${DB_PASSWORD}|g" \
        -e "s|{{DB_HOST}}|${DB_HOST}|g" \
        -e "s|{{DB_PORT}}|${DB_PORT}|g" \
        -e "s|{{DB_NAME}}|${DB_NAME}|g" \
        "${ALEMBIC_TEMPLATE}" > "${ALEMBIC_CONFIG}"

    if [[ -f "${ALEMBIC_CONFIG}" ]]; then
        chown ${SERVICE_USER}:${SERVICE_GROUP} "${ALEMBIC_CONFIG}"
        chmod 640 "${ALEMBIC_CONFIG}"  # Readable by service user only
        ok "alembic.ini generated with database credentials"
    else
        error "Failed to generate alembic.ini"
        exit 1
    fi

    # Verify connection string looks correct
    DB_URL=$(grep "^sqlalchemy.url" "${ALEMBIC_CONFIG}" | cut -d'=' -f2- | xargs)
    info "Database URL configured: postgresql://[user]:[pass]@${DB_HOST}:${DB_PORT}/${DB_NAME}"

    # Security check - ensure no literal {{PLACEHOLDERS}} remain
    if grep -q "{{" "${ALEMBIC_CONFIG}"; then
        error "Template placeholders not replaced in alembic.ini"
        error "Check .env variables: DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME"
        exit 1
    fi

    ok "alembic.ini configuration verified (no placeholders)"

    # -----------------------------
    # 3.7 Copy requirements.txt to production (v2.3.0 FIX)
    # -----------------------------
    if [[ -f "${GITHUB_REPO_DEV}/requirements.txt" ]]; then
        info "Copying requirements.txt from DEV to PROD..."
        cp "${GITHUB_REPO_DEV}/requirements.txt" "${INSTALL_PATH}/requirements.txt"
        chown ${SERVICE_USER}:${SERVICE_GROUP} "${INSTALL_PATH}/requirements.txt"
        ok "requirements.txt copied to ${INSTALL_PATH}/"
    else
        warn "requirements.txt not found in development directory — FASE 4 may fail"
    fi

    # Optional: symlink .env into repo (collectors may expect it)
    if [[ -f "${INSTALL_PATH}/.env" && -d "${GITHUB_REPO_DEV}" ]]; then
        ln -sf "${INSTALL_PATH}/.env" "${GITHUB_REPO_DEV}/.env" 2>/dev/null || true
        chown -h ${SERVICE_USER}:${SERVICE_GROUP} "${GITHUB_REPO_DEV}/.env" 2>/dev/null || true
        ok ".env symlink in repo updated (${GITHUB_REPO_DEV}/.env)"
    fi

    # -----------------------------
    # 3.4.2 Directory overview
    # -----------------------------
    info "Directory structuur:"
    echo "  Production:  ${INSTALL_PATH} (runtime)"
    echo "  Development: ${GITHUB_REPO_DEV} (git sync)"

    # Set ownership for production directory (runtime)
    if [[ -d "${INSTALL_PATH}" ]]; then
        chown -R ${SERVICE_USER}:${SERVICE_GROUP} "${INSTALL_PATH}" 2>/dev/null || true
        ok "Ownership van ${INSTALL_PATH} ingesteld op ${SERVICE_USER}:${SERVICE_GROUP}"
    fi

    # -----------------------------
    # 3.5 SSH Hardening (Interactive)
    # -----------------------------
    header "3.5 — SSH Security Hardening"

    echo
    warn "⚠️  BELANGRIJKE VEILIGHEIDSCHECK:"
    echo

    # Check if synctacles has SSH access
    if authkeys_has_keys /home/synctacles/.ssh/authorized_keys; then
        ok "✓ Synctacles user heeft SSH key"
        echo
        info "Test EERST je synctacles login voordat je doorgaat:"
        echo "  ssh ${SERVICE_USER}@$(hostname -I | awk '{print $1}')"
        echo
        read -rp "Heb je synctacles login getest en werkt het? (y/N): " TESTED_LOGIN

        if [[ "${TESTED_LOGIN,,}" != "y" ]]; then
            warn "Login NIET getest - SSH hardening overgeslagen voor veiligheid."
            warn "Test eerst synctacles login, run dan opnieuw: sudo $0 fase3"
            echo
            info "FASE 3 gedeeltelijk voltooid - SSH hardening moet later."
            return 0
        fi
        ok "Login getest - SSH hardening kan veilig uitgevoerd worden."
    else
        fail "✘ Synctacles user heeft GEEN SSH key!"
        warn "SSH hardening overgeslagen - te gevaarlijk zonder backup toegang."
        echo
        info "Voeg eerst een SSH key toe voor synctacles, run dan opnieuw: sudo $0 fase3"
        return 1
    fi

    echo
    SSH_CONF="/etc/ssh/sshd_config"
    SSH_BAK="$SSH_CONF.bak-$(date +%Y%m%d-%H%M%S)"

    cp "$SSH_CONF" "$SSH_BAK"
    ok "Backup van SSH config: $SSH_BAK"

    local modify_ssh=0

    read -rp "Wil je root-login via SSH uitschakelen? (y/N): " DISABLE_ROOT
    if [[ "${DISABLE_ROOT,,}" == "y" ]]; then
        append_if_not_present "PermitRootLogin no" "$SSH_CONF"
        modify_ssh=1
        ok "PermitRootLogin no ingesteld."
    fi

    read -rp "Wil je wachtwoord-login via SSH uitschakelen (alleen key-based)? (y/N): " DISABLE_PW
    if [[ "${DISABLE_PW,,}" == "y" ]]; then
        append_if_not_present "PasswordAuthentication no" "$SSH_CONF"
        append_if_not_present "PubkeyAuthentication yes" "$SSH_CONF"
        modify_ssh=1
        ok "PasswordAuthentication no + PubkeyAuthentication yes ingesteld."
    fi

    read -rp "Wil je MaxAuthTries verlagen naar 3? (y/N): " SET_MAXAUTH
    if [[ "${SET_MAXAUTH,,}" == "y" ]]; then
        append_if_not_present "MaxAuthTries 3" "$SSH_CONF"
        modify_ssh=1
        ok "MaxAuthTries 3 ingesteld."
    fi

    read -rp "Wil je ClientAliveInterval/Count instellen (300s / 2)? (y/N): " SET_CLIENTALIVE
    if [[ "${SET_CLIENTALIVE,,}" == "y" ]]; then
        append_if_not_present "ClientAliveInterval 300" "$SSH_CONF"
        append_if_not_present "ClientAliveCountMax 2" "$SSH_CONF"
        modify_ssh=1
        ok "ClientAliveInterval/Count ingesteld."
    fi

    if [[ "$modify_ssh" -eq 1 ]]; then
        info "Controleer nieuwe SSH-config..."

        if sshd -t 2>/dev/null; then
            echo
            warn "⚠️  SSH config gaat herladen - als er een fout is, blijf je ingelogd!"
            warn "⚠️  Test na reload ALTIJD in een nieuwe terminal: ssh ${SERVICE_USER}@..."
            echo
            read -rp "SSH config herladen? (y/N): " DO_RELOAD

            if [[ "${DO_RELOAD,,}" == "y" ]]; then
                if systemctl reload ssh 2>/dev/null; then
                    ok "SSH config valide — SSH succesvol herladen."
                    echo
                    warn "⚠️  TEST NU in nieuwe terminal: ssh ${SERVICE_USER}@$(hostname -I | awk '{print $1}')"
                    warn "⚠️  Als het NIET werkt, rollback: cp $SSH_BAK $SSH_CONF && systemctl reload ssh"
                else
                    warn "Kon ssh niet herladen. Probeer fallback..."
                    systemctl restart ssh 2>/dev/null && ok "SSH opnieuw gestart (fallback)."
                fi
            else
                info "SSH reload uitgesteld - config is voorbereid maar niet actief."
            fi
        else
            fail "SSH config ongeldig — rollback naar backup."
            cp "$SSH_BAK" "$SSH_CONF"
            systemctl reload ssh 2>/dev/null || systemctl restart ssh 2>/dev/null
        fi
    else
        info "Geen wijzigingen aan SSH-config aangebracht."
    fi

    # -----------------------------
    # 3.6 Auditd + sudo logging
    # -----------------------------
    info "Installeer auditd + sudo logging..."
    apt-get install -y auditd audispd-plugins >/dev/null 2>&1 || true

    echo 'Defaults logfile="/var/log/sudo.log"' >/etc/sudoers.d/99-sudo-logging
    chmod 440 /etc/sudoers.d/99-sudo-logging

    if visudo -cf /etc/sudoers >/dev/null 2>&1; then
        ok "sudo logging geconfigureerd."
    else
        fail "Fout in sudo logging config."
    fi

    systemctl enable --now auditd >/dev/null 2>&1 || true
    ok "auditd actief."

    # -----------------------------
    # 3.7 Fail2ban
    # -----------------------------
    info "Installeer en configureer fail2ban..."
    apt-get install -y fail2ban >/dev/null 2>&1 || true

    mkdir -p /etc/fail2ban/jail.d

    cat >/etc/fail2ban/jail.d/sshd.local <<EOF
[sshd]
enabled  = true
port     = ssh
logpath  = /var/log/auth.log
backend  = systemd
maxretry = 5
bantime  = 3600
findtime = 600
EOF

    systemctl enable --now fail2ban >/dev/null 2>&1 || true

    if systemctl status fail2ban --no-pager 2>/dev/null | grep -q running; then
        ok "fail2ban draait met sshd jail actief."
    else
        fail "fail2ban kon niet worden gestart."
    fi

    ok "FASE 3 voltooid."
    echo
    warn "⚠️  REMINDER: Test synctacles login in nieuwe terminal voordat je uitlogt!"
}

# ========================================================
#   FASE 4 — Python Environment
# ========================================================
fase4() {
    header "FASE 4 — Python Environment Setup (venv)"

    VENV_PATH="$SYNCTACLES_PROD/venv"
    REQUIREMENTS_FILE="$SYNCTACLES_PROD/requirements.txt"
    REQUIREMENTS_FROZEN="$SYNCTACLES_PROD/requirements-frozen.txt"

    # -----------------------------
    # 4.1 Verify Python 3.12
    # -----------------------------
    info "Verifieer Python 3.12..."
    if ! python3.12 --version >/dev/null 2>&1; then
        info "Installeer Python 3.12..."
        apt-get install -y software-properties-common >/dev/null 2>&1
        add-apt-repository ppa:deadsnakes/ppa -y >/dev/null 2>&1 || true
        apt-get update -qq
        apt-get install -y python3.12 python3.12-venv python3.12-dev >/dev/null 2>&1
    fi
    ok "Python 3.12 aanwezig"

    # -----------------------------
    # 4.2 Create Virtual Environment
    # -----------------------------
    if [[ ! -d "$VENV_PATH" ]]; then
        info "Maak virtual environment..."
        python3.12 -m venv "$VENV_PATH"
        ok "VENV aangemaakt: $VENV_PATH"
    else
        ok "VENV bestaat al: $VENV_PATH"
    fi

    # Activate venv
    source "$VENV_PATH/bin/activate"

    # Upgrade pip
    pip install --upgrade pip >/dev/null 2>&1

    # -----------------------------
    # 4.3 Install Requirements (v2.3.0 FIX: use requirements.txt from PROD)
    # -----------------------------
    if [[ -f "$REQUIREMENTS_FILE" ]]; then
        info "Installeer requirements vanuit $REQUIREMENTS_FILE..."
        pip install -r "$REQUIREMENTS_FILE" 2>&1 | tail -5
        ok "Requirements geïnstalleerd"

        # Ensure lxml is installed (required for XML parsing)
        pip install lxml >/dev/null 2>&1
        ok "lxml installed"
    else
        fail "requirements.txt niet gevonden in $SYNCTACLES_PROD"
        fail "Zorg dat FASE 3 correct is uitgevoerd (kopieert requirements.txt van DEV naar PROD)"
        exit 1
    fi

    # -----------------------------
    # 4.4 Freeze requirements (v2.3.0 FIX)
    # -----------------------------
    info "Freeze requirements voor reproduceerbaarheid..."
    pip freeze > "$REQUIREMENTS_FROZEN"
    chown ${SERVICE_USER}:${SERVICE_GROUP} "$REQUIREMENTS_FROZEN"
    ok "Frozen requirements opgeslagen: $REQUIREMENTS_FROZEN"

    # -----------------------------
    # 4.5 Verify critical packages
    # -----------------------------
    info "Verifieer kritieke packages..."

    local missing=0
    for pkg in gunicorn uvicorn fastapi sqlalchemy alembic; do
        if pip show "$pkg" >/dev/null 2>&1; then
            ok "$pkg geïnstalleerd"
        else
            fail "$pkg ONTBREEKT"
            ((missing++))
        fi
    done

    if [[ $missing -gt 0 ]]; then
        fail "$missing kritieke packages ontbreken!"
        exit 1
    fi

    ok "FASE 4 voltooid."
}

# ========================================================
#   FASE 5 — Production Automation (systemd)
# ========================================================
fase5() {
    header "FASE 5 — Production Automation (systemd services + timers)"

    # Production log directories (SKILL 9 compliant)
    info "Creating production log structure..."
    mkdir -p "${INSTALL_PATH}/logs/{api,scheduler,collectors/{entso_e_raw,tennet_raw},importers,normalizers}"
    chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "${INSTALL_PATH}/logs"
    chmod -R 755 "${INSTALL_PATH}/logs"
    ok "Log structure: ${INSTALL_PATH}/logs"

    # Verify repo exists (deployment already ran in FASE 3)
    if [[ ! -d "${GITHUB_REPO_DEV}/systemd" ]]; then
        fail "systemd folder niet gevonden in repo: ${GITHUB_REPO_DEV}/systemd"
        warn "Je repo is mogelijk niet up-to-date."
        warn "Run: sudo -u ${SERVICE_USER} git -C ${GITHUB_REPO_DEV} pull origin main"
        exit 1
    fi

    # Note: Code deployment to ${INSTALL_PATH}/app/ happens in FASE 3
    # FASE 5 focuses on systemd service setup with already-deployed code

    # Verify deployment from FASE 3 was successful
    if [[ ! -f "${INSTALL_PATH}/app/start_api.py" ]]; then
        fail "Application code not deployed. Run FASE 3 first."
        exit 1
    fi
    ok "Application code deployed (from FASE 3)"

    # Continue with systemd setup
    # Verify VERSION file (v2.3.0 FIX)
    # (This file should exist from rsync in FASE 3)
    # -----
    if [[ -f "${INSTALL_PATH}/app/VERSION" ]]; then
        APP_VERSION=$(cat ${INSTALL_PATH}/app/VERSION)
        ok "VERSION file found: $APP_VERSION"
    else
        fail "VERSION file NOT FOUND in ${INSTALL_PATH}/app/"
        fail "Zorg dat VERSION bestaat in de repo root"
        exit 1
    fi

    # Symlink .env into app (collectors expect it there)
    if [[ -f "${INSTALL_PATH}/.env" ]]; then
        ln -sf ${INSTALL_PATH}/.env ${INSTALL_PATH}/app/.env 2>/dev/null || true
        chown -h ${SERVICE_USER}:${SERVICE_GROUP} ${INSTALL_PATH}/app/.env 2>/dev/null || true
        ok ".env accessible from app directory"
    fi

    # Add LOG_DIR to .env if missing (v2.3.0 FIX: use grep -q)
    if [[ -f "${INSTALL_PATH}/.env" ]]; then
        if ! grep -q "^LOG_PATH=" ${INSTALL_PATH}/.env 2>/dev/null; then
            echo "LOG_PATH=${LOG_PATH}" >> ${INSTALL_PATH}/.env
            ok "LOG_PATH added to .env"
        else
            ok "LOG_PATH already in .env"
        fi
    fi

    # Legacy compatibility (symlink for old scripts)
    if [[ ! -L /var/log/application ]]; then
        ln -sf ${LOG_PATH} /var/log/application 2>/dev/null || true
    fi

    # Copy service files
    info "Installeer systemd units..."
    cp "${GITHUB_REPO_DEV}/systemd/"*.service /etc/systemd/system/ 2>/dev/null || true
    cp "${GITHUB_REPO_DEV}/systemd/"*.timer /etc/systemd/system/ 2>/dev/null || true

    # Validate systemd unit paths (CRITICAL)
    info "Validating systemd unit paths..."
    INVALID_PATHS=0

    for unit in /etc/systemd/system/synctacles-*.service; do
        [[ -f "$unit" ]] || continue

        # Check for DEV repo paths (FORBIDDEN)
        if grep -qE "ExecStart=.*/opt/github" "$unit"; then
            fail "INVALID: $(basename "$unit") points to DEV repo"
            ((INVALID_PATHS++))
        fi

        if grep -qE "WorkingDirectory=/opt/github" "$unit"; then
            fail "INVALID: $(basename "$unit") has DEV WorkingDirectory"
            ((INVALID_PATHS++))
        fi
    done

    if [[ $INVALID_PATHS -gt 0 ]]; then
        fail "$INVALID_PATHS systemd units have invalid paths"
        warn "Units MUST use: ${INSTALL_PATH}/app/"
        warn "Fix units in repo: ${GITHUB_REPO_DEV}/systemd/"
        warn "Then re-run: sudo $0 fase5"
        exit 1
    else
        ok "All systemd units use correct paths (${INSTALL_PATH}/app/)"
    fi

    info "Reloading systemd daemon..."
    systemctl daemon-reload
    ok "Systemd daemon reloaded"

    # Enable timers/services (v2.3.0 FIX: explicitly enable collector timer)
    for unit in collector.timer importer.timer normalizer.timer health.timer; do
        UNIT_NAME="synctacles-${unit}"
        if systemctl list-unit-files | grep -q "$UNIT_NAME"; then
            systemctl enable --now "$UNIT_NAME" >/dev/null 2>&1 || true
            ok "Enabled: $UNIT_NAME"
        else
            warn "Unit not found: $UNIT_NAME"
        fi
    done

    # API service
    if systemctl list-unit-files | grep -q "synctacles-api.service"; then
        systemctl enable --now synctacles-api.service >/dev/null 2>&1 || true
        ok "Enabled: synctacles-api.service"
    else
        warn "Unit not found: synctacles-api.service"
    fi

    # Restart API service (pickup new code on redeploy)
    if systemctl is-active --quiet synctacles-api.service 2>/dev/null; then
        info "Restarting API service (code update)..."
        systemctl restart synctacles-api.service
        sleep 3  # Allow restart to complete
        ok "API service restarted"
    else
        # First-time install: start fresh
        info "Starting API service (first install)..."
        systemctl enable --now synctacles-api.service >/dev/null 2>&1 || true
        ok "API service started"
    fi

    # -----------------------------
    # Database Migrations (v2.3.0 FIX: verify alembic + PYTHONPATH)
    # -----------------------------
    header "Database Schema Migration"

    # Verify alembic is installed in venv
    if [[ ! -x "${INSTALL_PATH}/venv/bin/alembic" ]]; then
        fail "Alembic not found in venv!"
        fail "Run: ${INSTALL_PATH}/venv/bin/pip install alembic"
        exit 1
    fi
    ok "Alembic found in venv"

    if [[ -d "${INSTALL_PATH}/app/alembic" ]]; then
        info "Running Alembic migrations..."
        cd ${INSTALL_PATH}/app || exit 1

        # Run with PYTHONPATH set correctly
        export PYTHONPATH="${INSTALL_PATH}/app:${PYTHONPATH:-}"

        if sudo -u ${SERVICE_USER} PYTHONPATH="$PYTHONPATH" ${INSTALL_PATH}/venv/bin/alembic upgrade head 2>&1; then
            ok "Database schema up-to-date"
        else
            fail "Alembic migration FAILED"
            warn "Check: ${INSTALL_PATH}/app/alembic/versions/"
            warn "Manual: cd ${INSTALL_PATH}/app && sudo -u ${SERVICE_USER} PYTHONPATH=. ${INSTALL_PATH}/venv/bin/alembic upgrade head"
            exit 1
        fi

        cd - >/dev/null || true
    else
        info "No Alembic directory found (manual migrations may be needed)"
    fi

    # -----------------------------
    # Production Health Check
    # -----------------------------
    header "Production Health Check"

    info "Waiting for services to stabilize (15s)..."
    sleep 15

    # API health
    if curl -sf http://localhost:8000/health >/dev/null 2>&1; then
        API_RESPONSE=$(curl -s http://localhost:8000/health 2>/dev/null)
        API_STATUS=$(echo "$API_RESPONSE" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
        API_VERSION=$(echo "$API_RESPONSE" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown")

        if [[ "$API_STATUS" == "ok" ]]; then
            ok "API health check passed (status: $API_STATUS, version: $API_VERSION)"
        else
            warn "API responding but status=$API_STATUS"
        fi

        # Compare with deployed version
        if [[ -f "${INSTALL_PATH}/app/VERSION" ]]; then
            DEPLOYED_VER=$(cat ${INSTALL_PATH}/app/VERSION)
            if [[ "$API_VERSION" == "$DEPLOYED_VER" ]]; then
                ok "Version match: $API_VERSION"
            else
                warn "Version mismatch: API=$API_VERSION, Deployed=$DEPLOYED_VER"
            fi
        fi
    else
        warn "API not responding on port 8000"
        warn "Check: journalctl -u synctacles-api.service -n 50"
    fi

    # Timer check
    TIMER_COUNT=$(systemctl list-timers synctacles-* --no-legend 2>/dev/null | wc -l)
    if [[ $TIMER_COUNT -ge 3 ]]; then
        ok "$TIMER_COUNT automation timers active"
    else
        warn "Only $TIMER_COUNT timers active (expected ≥3)"
        warn "Check: systemctl list-timers synctacles-*"
    fi

    # Database connectivity
    if sudo -u ${SERVICE_USER} psql synctacles -c "SELECT 1" >/dev/null 2>&1; then
        ok "Database connection working"
    else
        fail "Database connection failed"
    fi

    # VERSION file check
    if [[ -f "${INSTALL_PATH}/app/VERSION" ]]; then
        ok "VERSION file exists: $(cat ${INSTALL_PATH}/app/VERSION)"
    else
        fail "VERSION file missing after deploy"
    fi

    ok "FASE 5 voltooid — services/timers geïnstalleerd."

    echo
    echo "📊 Monitoring:"
    echo "   Timers:  systemctl list-timers synctacles-*"
    echo "   API:     curl http://localhost:8000/health"
    echo "   Logs:    journalctl -u synctacles-api.service -f"
    echo
    echo "🚀 Next: Test end-to-end pipeline"
    echo "   1. Trigger collector:  systemctl start synctacles-collector.service"
    echo "   2. Trigger importer:   systemctl start synctacles-importer.service"
    echo "   3. Trigger normalizer: systemctl start synctacles-normalizer.service"
    echo "   4. Test API:           curl http://localhost:8000/api/v1/balance | jq"
    echo
}

# ========================================================
#   FASE 6 — Development Tools
# ========================================================
fase6() {
    header "FASE 6 — Development Tools (Git Workflow)"

    # -----------------------------
    # 6.1 Verify Prerequisites
    # -----------------------------
    info "Verifying prerequisites..."

    if [[ ! -d "$SYNCTACLES_DEV" ]]; then
        fail "Development directory niet gevonden: $SYNCTACLES_DEV"
        fail "FASE 3 moet eerst voltooid zijn (GitHub clone)"
        exit 1
    fi

    if ! id "synctacles" &>/dev/null; then
        fail "User 'synctacles' bestaat niet"
        fail "FASE 3 moet eerst voltooid zijn (accounts)"
        exit 1
    fi

    ok "Prerequisites verified"

    # -----------------------------
    # 6.2 Git Repository Check
    # -----------------------------
    info "Verifying git repository..."

    if [[ -d "$SYNCTACLES_DEV/.git" ]]; then
        ok "Git repository already initialized"
    else
        warn "Git repository not initialized"
        info "Initializing git repository..."

        cd "$SYNCTACLES_DEV" || exit 1
        sudo -u ${SERVICE_USER} git init
        sudo -u ${SERVICE_USER} git config user.name "${GIT_USER_NAME}"
        sudo -u ${SERVICE_USER} git config user.email "${GIT_USER_EMAIL}"

        ok "Git repository initialized"
    fi

    # -----------------------------
    # 6.3 Create .gitignore
    # -----------------------------
    if [[ ! -f "$SYNCTACLES_DEV/.gitignore" ]]; then
        info "Creating .gitignore..."

        cat > "$SYNCTACLES_DEV/.gitignore" << 'EOF'
# Python
__pycache__/
*.py[cod]
*$py.class
*.so
.Python
env/
venv/
ENV/
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
*.egg-info/
.installed.cfg
*.egg

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# Environment
.env
.env.*
!.env.example

# Logs
logs/
*.log

# OS
.DS_Store
Thumbs.db

# Project specific
cache/
data/
*.db
*.sqlite
.coverage
htmlcov/
EOF

        chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYNCTACLES_DEV/.gitignore"
        ok ".gitignore created"
    else
        ok ".gitignore already exists"
    fi

    # -----------------------------
    # 6.4 Create Test Script
    # -----------------------------
    if [[ ! -f "$SYNCTACLES_DEV/test_setup.py" ]]; then
        info "Creating test_setup.py..."

        cat > "$SYNCTACLES_DEV/test_setup.py" << 'EOF'
#!/usr/bin/env python3
"""Test that development environment is properly configured"""
import sys

def test_imports():
    """Test all critical imports"""
    print("Testing Python imports...")

    packages = [
        ("entsoe", "entsoe-py"),
        ("pandas", "pandas"),
        ("fastapi", "fastapi"),
        ("sqlalchemy", "sqlalchemy"),
        ("lxml", "lxml"),
    ]

    failed = False
    for module, name in packages:
        try:
            __import__(module)
            print(f"✅ {name}")
        except ImportError as e:
            print(f"❌ {name}: {e}")
            failed = True

    return not failed

def main():
    print("=" * 60)
    print("Application Development Environment Test")
    print("=" * 60)
    print()

    if not test_imports():
        print("\n❌ Import test failed!")
        sys.exit(1)

    print()
    print("=" * 60)
    print("✅ Development environment is ready!")
    print("=" * 60)

if __name__ == "__main__":
    main()
EOF

        chmod +x "$SYNCTACLES_DEV/test_setup.py"
        chown ${SERVICE_USER}:${SERVICE_GROUP} "$SYNCTACLES_DEV/test_setup.py"
        ok "test_setup.py created"
    else
        ok "test_setup.py already exists"
    fi

    # -----------------------------
    # 6.5 Set Final Permissions
    # -----------------------------
    info "Setting final permissions..."
    chown -R ${SERVICE_USER}:${SERVICE_GROUP} "${GITHUB_REPO_DEV}"
    # Do NOT chmod -R the repo; executable bits are tracked by git and blanket chmod can break expectations.
    ok "Ownership set: ${SERVICE_USER}:${SERVICE_GROUP} (permissions unchanged)"

    ok "FASE 6 voltooid — development tools geïnstalleerd"

    echo
    echo "✅ DEVELOPMENT ENVIRONMENT READY"
    echo
    echo "📁 Location: ${GITHUB_REPO_DEV}"
    echo "👤 Owner:    ${SERVICE_USER}:${SERVICE_GROUP}"
    echo "📦 Git:      Initialized"
    echo
    echo "🛠️  Development Workflow:"
    echo "   1. Switch to user:       su - ${SERVICE_USER}"
    echo "   2. Go to repo:           cd ${GITHUB_REPO_DEV}"
    echo "   3. Test setup:           python3 test_setup.py"
    echo "   4. Run collectors:       python3 sparkcrawler_db/collectors/sparkcrawler_entso_e_a75_generation.py"
    echo "   5. Run importers:        python3 sparkcrawler_db/importers/import_entso_e_a75.py"
    echo "   6. Run normalizers:      python3 synctacles_db/normalizers/normalize_entso_e_a75.py"
    echo
}

# ========================================================
#   SETUP SUMMARY
# ========================================================
print_summary() {
    header "Setup Complete!"

    echo
    echo "✅ Application Server Setup voltooid"
    echo
    echo "📁 Directories:"
    echo "   Production:  ${INSTALL_PATH}"
    echo "   Development: ${GITHUB_REPO_DEV}"
    echo "   Logs:        ${LOG_PATH}"
    echo
    echo "🔑 Database:"
    echo "   Name:     ${DB_NAME}"
    echo "   User:     ${DB_USER}"
    echo "   Password: (geen - trust authentication)"
    echo "   URL:      postgresql://${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
    echo
    echo "📄 Configuration:"
    echo "   Environment: ${INSTALL_PATH}/.env"
    echo "   Python venv: ${INSTALL_PATH}/venv/"
    echo
    echo "🚀 Services Running:"
    systemctl is-active --quiet docker && echo "   ✓ Docker" || echo "   ✗ Docker"
    systemctl is-active --quiet postgresql && echo "   ✓ PostgreSQL" || echo "   ✗ PostgreSQL"
    systemctl is-active --quiet redis-server && echo "   ✓ Redis" || echo "   ✗ Redis"
    systemctl is-active --quiet grafana-server && echo "   ✓ Grafana (:3000)" || echo "   ✗ Grafana"
    systemctl is-active --quiet node_exporter && echo "   ✓ Node Exporter (:9100)" || echo "   ✗ Node Exporter"
    systemctl is-active --quiet fail2ban && echo "   ✓ Fail2ban" || echo "   ✗ Fail2ban"
    echo
    echo "📖 Documentation:"
    echo "   Setup log: $LOG_FILE"
    echo
}

# ========================================================
#   MAIN DISPATCHER
# ========================================================
main() {
    ensure_root
    setup_logging

    FASE="${1:-}"

    case "$FASE" in
        fase0)
            fase0
            ;;
        fase1)
            fase1
            ;;
        fase2)
            fase2
            ;;
        fase3)
            fase3
            ;;
        fase4)
            fase4
            ;;
        fase5)
            fase5
            print_summary
            ;;
        fase6)
            fase6
            print_summary
            ;;
        *)
            echo "Gebruik: sudo $0 {fase0|fase1|fase2|fase3|fase4|fase5|fase6}"
            echo
            echo "Fasen:"
            echo "  fase0 - Brand configuration (interactive .env + manifest.json generation)"
            echo "  fase1 - Systeem update + kernel"
            echo "  fase2 - Software stack (Docker, PostgreSQL, Redis, Grafana)"
            echo "  fase3 - Security (accounts, SSH keys, GitHub, SSH hardening, fail2ban)"
            echo "  fase4 - Python environment (venv + packages)"
            echo "  fase5 - Production automation (systemd services + timers)"
            echo "  fase6 - Development tools (git workflow + testing)"
            echo
            echo "Productie server: fase0 → fase1 → fase2 → fase3 → fase4 → fase5"
            echo "Development:      fase0 → fase1 → fase2 → fase3 → fase4 → fase6"
            echo "Beide:            fase0 → fase1 → fase2 → fase3 → fase4 → fase5 → fase6"
            echo
            exit 1
            ;;
    esac
}

# Run
main "$@"
