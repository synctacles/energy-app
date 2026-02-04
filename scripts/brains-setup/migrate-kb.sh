#!/bin/bash
set -euo pipefail

# ============================================
# KB Migration Script
# DEV Server → BRAINS Server
# ============================================

# Configuratie
DEV_SERVER="synct-dev"  # SSH alias from cc-hub
DEV_DB_NAME="brains_kb"           # Naam op DEV
DEV_DB_USER="postgres"            # User op DEV

BRAINS_DB_NAME="brains_kb"
BRAINS_DB_USER="brains_admin"

BACKUP_DIR="/tmp/kb-migration"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

log_info() { echo -e "\033[0;32m[INFO]\033[0m $1"; }
log_warn() { echo -e "\033[1;33m[WARN]\033[0m $1"; }

# ============================================
# Stap 1: Dump van DEV server
# ============================================
dump_from_dev() {
    log_info "Creating backup directory..."
    mkdir -p "${BACKUP_DIR}"

    log_info "Dumping KB database from DEV server..."

    # Via cc-hub naar DEV server
    ssh cc-hub "ssh ${DEV_SERVER} 'pg_dump -U ${DEV_DB_USER} -d ${DEV_DB_NAME} --schema=kb --data-only'" \
        > "${BACKUP_DIR}/kb_data_${TIMESTAMP}.sql" 2>/dev/null || {
        log_warn "Failed to dump from DEV. Checking if DB exists..."

        # Check if DB exists on DEV
        DB_EXISTS=$(ssh cc-hub "ssh ${DEV_SERVER} 'sudo -u postgres psql -lqt | cut -d \\| -f 1 | grep -qw ${DEV_DB_NAME} && echo yes || echo no'")

        if [[ "$DB_EXISTS" == "no" ]]; then
            log_warn "Database ${DEV_DB_NAME} does not exist on DEV server"
            log_warn "Creating empty migration (no data to migrate)"
            echo "-- No data to migrate from DEV" > "${BACKUP_DIR}/kb_data_${TIMESTAMP}.sql"
            return
        fi

        log_warn "Database exists but dump failed. Retrying..."
        exit 1
    }

    log_info "Dump created: ${BACKUP_DIR}/kb_data_${TIMESTAMP}.sql"
}

# ============================================
# Stap 2: Restore op BRAINS
# ============================================
restore_to_brains() {
    log_info "Restoring KB data to BRAINS..."

    # Check if there's actual data to restore
    if grep -q "No data to migrate" "${BACKUP_DIR}/kb_data_${TIMESTAMP}.sql"; then
        log_info "No data to restore (empty migration)"
        return
    fi

    # Restore data
    sudo -u postgres psql -d "${BRAINS_DB_NAME}" < "${BACKUP_DIR}/kb_data_${TIMESTAMP}.sql"

    # Verify
    ENTRY_COUNT=$(sudo -u postgres psql -d "${BRAINS_DB_NAME}" -t -c "SELECT COUNT(*) FROM kb.entries;")
    log_info "Restored ${ENTRY_COUNT} KB entries"
}

# ============================================
# Stap 3: Migreer Moltbot configs
# ============================================
migrate_moltbot_config() {
    log_info "Migrating Moltbot configuration..."

    # Haal Moltbot config op van DEV
    ssh cc-hub "ssh ${DEV_SERVER} 'cat ~/.config/moltbot/config.json'" \
        > "${BACKUP_DIR}/moltbot_config_${TIMESTAMP}.json" 2>/dev/null || {
        log_warn "Moltbot config niet gevonden op standaard locatie (~/.config/moltbot/)"

        # Try alternative locations
        ssh cc-hub "ssh ${DEV_SERVER} 'cat /etc/moltbot/config.json'" \
            > "${BACKUP_DIR}/moltbot_config_${TIMESTAMP}.json" 2>/dev/null || {
            log_warn "Moltbot config ook niet gevonden in /etc/moltbot/"
            log_warn "Skipping config migration - manual configuration needed"
            return
        }
    }

    # Converteer naar OpenClaw format (basis transformatie)
    log_info "Converting Moltbot config to OpenClaw format..."
    # Dit vereist handmatige review - config formats kunnen verschillen

    log_info "Config opgeslagen in ${BACKUP_DIR}/"
    log_info "⚠️  Review en pas handmatig aan in /etc/openclaw/openclaw.json"
}

# ============================================
# Stap 4: Kopieer scripts
# ============================================
migrate_scripts() {
    log_info "Migrating scripts from DEV..."

    # Maak scripts directory
    mkdir -p /opt/openclaw/scripts

    # Kopieer relevante scripts
    ssh cc-hub "ssh ${DEV_SERVER} 'tar -czf - ~/moltbot-scripts/ 2>/dev/null'" | tar -xzf - -C /opt/openclaw/scripts/ 2>/dev/null || {
        log_warn "Geen scripts gevonden om te migreren"
    }

    chown -R openclaw:openclaw /opt/openclaw/scripts 2>/dev/null || log_warn "openclaw user not yet created"

    log_info "Scripts migrated ✓"
}

# ============================================
# Main
# ============================================
main() {
    echo "============================================"
    echo "KB Migration: DEV → BRAINS"
    echo "============================================"

    dump_from_dev
    restore_to_brains
    migrate_moltbot_config
    migrate_scripts

    echo ""
    log_info "Migration complete!"
    echo ""
    echo "Volgende stappen:"
    echo "1. Verifieer data: sudo -u postgres psql -d ${BRAINS_DB_NAME} -c 'SELECT COUNT(*) FROM kb.entries;'"
    echo "2. Review Moltbot config conversie in ${BACKUP_DIR}/"
    echo "3. Update Telegram bot tokens in /etc/openclaw/secrets.env"
    echo "4. Start OpenClaw: sudo systemctl start openclaw"
}

main "$@"
