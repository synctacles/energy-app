#!/usr/bin/env bash
# validate_synctacles_setup_v3.1.0.sh
# Comprehensive validation for SYNCTACLES server setup
#
# =====================================================================
# VERSION: 3.1.0 (2025-12-19)
# =====================================================================
#
# CHANGELOG v3.1.0:
# - Added PAD-CONTRACT checks (no /opt/github/ references in code)
# - Added SYNCTACLES_LOG_DIR validation in .env
# - Added TENNET_API_BASE check in .env
# - Added systemd unit compliance validation (WorkingDirectory, ExecStart)
# - Added database schema checks (Alembic + required tables)
# - Structured output per FASE (matches installer v2.2.0)
# - Exit codes: 0=pass, 1=warn, 2=fail
# - Integrated validate_paths.sh checks
# - Added runtime health checks (API, timers, collector execution)
#
# Validates against:
# - Installer v2.2.0 (FASE 1-5 structure)
# - SKILL 9 PAD-CONTRACT specifications
#
# =====================================================================

set -uo pipefail

# =====================================================================
# GLOBAL CONFIGURATION
# =====================================================================

SCRIPT_VERSION="3.1.0"
INSTALLER_VERSION="2.2.0"

# Paths
SYNCTACLES_PROD="/opt/synctacles"
SYNCTACLES_APP="/opt/synctacles/app"
SYNCTACLES_DEV="/opt/github/synctacles-repo"
SYNCTACLES_LOGS="/opt/synctacles/logs"
SYNCTACLES_VENV="/opt/synctacles/venv"
ENV_FILE="/opt/synctacles/.env"

# Forbidden path pattern (SKILL 9 - no code execution from repo)
FORBIDDEN_PATH="/opt/github/"

# Required tables (database schema)
REQUIRED_TABLES="raw_entso_e_a75 raw_entso_e_a65 raw_tennet_balance norm_entso_e_a75 norm_entso_e_a65 norm_tennet_balance"

# Critical Python packages
CRITICAL_PACKAGES="fastapi sqlalchemy pandas uvicorn pydantic httpx redis tenacity lxml entsoe alembic"

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Per-FASE counters
declare -A FASE_PASSED
declare -A FASE_FAILED
declare -A FASE_WARNINGS

# Current FASE
CURRENT_FASE=""

# =====================================================================
# OUTPUT FUNCTIONS
# =====================================================================

# Check if running interactively (for colors)
if [[ -t 1 ]]; then
    GREEN="\e[32m"
    RED="\e[31m"
    YELLOW="\e[33m"
    BLUE="\e[34m"
    CYAN="\e[36m"
    NC="\e[0m"
    CHECK_PASS="✅"
    CHECK_FAIL="❌"
    CHECK_WARN="⚠️"
else
    GREEN=""
    RED=""
    YELLOW=""
    BLUE=""
    CYAN=""
    NC=""
    CHECK_PASS="[PASS]"
    CHECK_FAIL="[FAIL]"
    CHECK_WARN="[WARN]"
fi

header() {
    echo ""
    echo "==============================================="
    echo "$1"
    echo "==============================================="
    echo ""
}

fase_header() {
    local fase="$1"
    local title="$2"
    CURRENT_FASE="$fase"
    FASE_PASSED[$fase]=0
    FASE_FAILED[$fase]=0
    FASE_WARNINGS[$fase]=0
    echo ""
    echo -e "${BLUE}[${fase}]${NC} ${title}"
}

ok() {
    echo -e "  ${CHECK_PASS} $1"
    ((PASSED++))
    if [[ -n "$CURRENT_FASE" ]]; then
        ((FASE_PASSED[$CURRENT_FASE]++))
    fi
}

fail() {
    echo -e "  ${CHECK_FAIL} $1"
    ((FAILED++))
    if [[ -n "$CURRENT_FASE" ]]; then
        ((FASE_FAILED[$CURRENT_FASE]++))
    fi
}

warn() {
    echo -e "  ${CHECK_WARN} $1"
    ((WARNINGS++))
    if [[ -n "$CURRENT_FASE" ]]; then
        ((FASE_WARNINGS[$CURRENT_FASE]++))
    fi
}

info() {
    echo -e "  ${CYAN}ℹ${NC} $1"
}

# =====================================================================
# PERMISSION CHECK
# =====================================================================

check_permissions() {
    # This script can run as root or synctacles user
    if [[ $EUID -eq 0 ]]; then
        info "Running as root"
        return 0
    elif [[ "$(whoami)" == "synctacles" ]]; then
        info "Running as synctacles user"
        return 0
    else
        info "Running as $(whoami) - some checks may be limited"
        return 0
    fi
}

# =====================================================================
# HELPER FUNCTIONS
# =====================================================================

# Check if command exists
has_command() {
    command -v "$1" >/dev/null 2>&1
}

# Check if service is running
service_running() {
    systemctl is-active --quiet "$1" 2>/dev/null
}

# Check if port is listening
port_listening() {
    ss -tlnp 2>/dev/null | grep -q ":$1 " || netstat -tlnp 2>/dev/null | grep -q ":$1 "
}

# Run command as synctacles if possible
run_as_synctacles() {
    if [[ $EUID -eq 0 ]]; then
        sudo -u synctacles "$@" 2>/dev/null
    elif [[ "$(whoami)" == "synctacles" ]]; then
        "$@" 2>/dev/null
    else
        "$@" 2>/dev/null
    fi
}

# Check if user can access postgres as synctacles
can_access_db() {
    run_as_synctacles psql synctacles -c "SELECT 1" >/dev/null 2>&1
}

# =====================================================================
# FASE 1: SYSTEM PREREQUISITES
# =====================================================================

validate_fase1() {
    fase_header "FASE 1" "System Prerequisites"

    # OS Check
    if [[ -f /etc/os-release ]]; then
        OS_NAME=$(grep "^NAME=" /etc/os-release | cut -d'"' -f2)
        OS_VERSION=$(grep "^VERSION_ID=" /etc/os-release | cut -d'"' -f2)

        if [[ "$OS_NAME" == "Ubuntu" ]] && [[ "$OS_VERSION" == "24.04" ]]; then
            ok "Ubuntu 24.04"
        elif [[ "$OS_NAME" == "Ubuntu" ]]; then
            warn "Ubuntu $OS_VERSION (recommended: 24.04)"
        else
            warn "$OS_NAME $OS_VERSION (recommended: Ubuntu 24.04)"
        fi
    else
        warn "Cannot detect OS version"
    fi

    # Python 3.12+
    if has_command python3.12; then
        PYTHON_VERSION=$(python3.12 --version 2>&1 | awk '{print $2}')
        ok "Python $PYTHON_VERSION"
    elif has_command python3; then
        PYTHON_VERSION=$(python3 --version 2>&1 | awk '{print $2}')
        PYTHON_MAJOR=$(echo "$PYTHON_VERSION" | cut -d. -f1)
        PYTHON_MINOR=$(echo "$PYTHON_VERSION" | cut -d. -f2)

        if [[ "$PYTHON_MAJOR" -eq 3 ]] && [[ "$PYTHON_MINOR" -ge 12 ]]; then
            ok "Python $PYTHON_VERSION"
        else
            fail "Python $PYTHON_VERSION (requires 3.12+)"
        fi
    else
        fail "Python 3.12+ NOT installed"
    fi

    # Required tools
    local tools="curl git wget jq"
    local all_tools_ok=true

    for tool in $tools; do
        if ! has_command "$tool"; then
            all_tools_ok=false
            break
        fi
    done

    if $all_tools_ok; then
        ok "Required tools installed"
    else
        fail "Missing required tools (curl, git, wget, jq)"
    fi
}

# =====================================================================
# FASE 2: SOFTWARE STACK
# =====================================================================

validate_fase2() {
    fase_header "FASE 2" "Software Stack"

    # PostgreSQL 16
    if has_command psql; then
        PG_VERSION=$(psql --version 2>&1 | awk '{print $3}' | cut -d. -f1)
        if [[ "$PG_VERSION" -ge 16 ]]; then
            if service_running postgresql; then
                ok "PostgreSQL $PG_VERSION running"
            else
                fail "PostgreSQL $PG_VERSION installed but NOT running"
            fi
        else
            warn "PostgreSQL $PG_VERSION (recommended: 16+)"
        fi
    else
        fail "PostgreSQL NOT installed"
    fi

    # TimescaleDB extension
    if can_access_db; then
        if run_as_synctacles psql synctacles -t -c "SELECT extname FROM pg_extension WHERE extname='timescaledb';" 2>/dev/null | grep -q "timescaledb"; then
            ok "TimescaleDB extension"
        else
            warn "TimescaleDB extension NOT installed"
        fi
    else
        warn "Cannot verify TimescaleDB (database not accessible)"
    fi

    # Redis
    if service_running redis-server; then
        ok "Redis running"
    elif service_running redis; then
        ok "Redis running"
    else
        fail "Redis NOT running"
    fi

    # .env exists
    if [[ -f "$ENV_FILE" ]]; then
        ok ".env configured"
    else
        fail ".env NOT found ($ENV_FILE)"
    fi
}

# =====================================================================
# FASE 3: SECURITY & REPOSITORY
# =====================================================================

validate_fase3() {
    fase_header "FASE 3" "Security & Repository"

    # synctacles user exists
    if id synctacles >/dev/null 2>&1; then
        ok "synctacles user exists"
    else
        fail "synctacles user NOT found"
    fi

    # GitHub repo cloned
    if [[ -d "$SYNCTACLES_DEV/.git" ]]; then
        ok "GitHub repo cloned"
    else
        warn "GitHub repo NOT cloned ($SYNCTACLES_DEV)"
    fi

    # SSH permissions
    local ssh_dir="/home/synctacles/.ssh"
    if [[ -d "$ssh_dir" ]]; then
        local ssh_perms
        ssh_perms=$(stat -c "%a" "$ssh_dir" 2>/dev/null)
        if [[ "$ssh_perms" == "700" ]]; then
            ok "SSH permissions correct"
        else
            warn "SSH directory permissions: $ssh_perms (should be 700)"
        fi
    else
        warn "SSH directory not found"
    fi
}

# =====================================================================
# FASE 4: PYTHON ENVIRONMENT
# =====================================================================

validate_fase4() {
    fase_header "FASE 4" "Python Environment"

    # venv exists
    if [[ -d "$SYNCTACLES_VENV" ]]; then
        ok "venv exists"
    else
        fail "venv NOT found ($SYNCTACLES_VENV)"
        return
    fi

    # Activate venv and test packages
    # shellcheck disable=SC1091
    if [[ -f "$SYNCTACLES_VENV/bin/activate" ]]; then
        source "$SYNCTACLES_VENV/bin/activate" 2>/dev/null || true

        local pkg_list=""
        local failed_pkgs=0

        for pkg in $CRITICAL_PACKAGES; do
            # Convert package names with hyphens to underscores for import
            local import_name="${pkg//-/_}"

            if python3 -c "import ${import_name}" 2>/dev/null; then
                pkg_list="${pkg_list}${pkg}, "
            else
                ((failed_pkgs++))
            fi
        done

        # Remove trailing comma and space
        pkg_list="${pkg_list%, }"

        if [[ $failed_pkgs -eq 0 ]]; then
            ok "Critical packages: ${pkg_list}"
        else
            warn "Some packages missing ($failed_pkgs failed import)"
        fi

        deactivate 2>/dev/null || true
    else
        fail "venv activate script NOT found"
    fi
}

# =====================================================================
# FASE 5: PRODUCTION DEPLOYMENT
# =====================================================================

validate_fase5() {
    fase_header "FASE 5" "Production Deployment"

    # /opt/synctacles/app/ deployed
    if [[ -d "$SYNCTACLES_APP" ]]; then
        # Check for key subdirectories
        if [[ -d "$SYNCTACLES_APP/sparkcrawler_db" ]] && [[ -d "$SYNCTACLES_APP/synctacles_db" ]]; then
            ok "/opt/synctacles/app/ deployed"
        else
            warn "/opt/synctacles/app/ exists but incomplete"
        fi
    else
        fail "/opt/synctacles/app/ NOT deployed"
    fi

    # Systemd units compliant
    local units_ok=true
    local units_checked=0

    for unit in /etc/systemd/system/synctacles-*.service; do
        [[ -f "$unit" ]] || continue
        ((units_checked++))

        # Check for forbidden /opt/github/ references
        if grep -q "$FORBIDDEN_PATH" "$unit" 2>/dev/null; then
            units_ok=false
            fail "Unit $(basename "$unit") contains $FORBIDDEN_PATH reference"
        fi

        # Check WorkingDirectory
        if ! grep -q "WorkingDirectory=/opt/synctacles/app" "$unit" 2>/dev/null; then
            if grep -q "WorkingDirectory=" "$unit" 2>/dev/null; then
                units_ok=false
                warn "Unit $(basename "$unit") has wrong WorkingDirectory"
            fi
        fi
    done

    if [[ $units_checked -eq 0 ]]; then
        warn "No systemd units found"
    elif $units_ok; then
        ok "Systemd units compliant"
    fi

    # API health OK
    if has_command curl; then
        if curl -sf --connect-timeout 5 http://localhost:8000/health >/dev/null 2>&1; then
            ok "API health OK"
        else
            fail "API NOT responding on :8000"
        fi
    else
        warn "Cannot check API health (curl not available)"
    fi
}

# =====================================================================
# SKILL 9: PAD-CONTRACT COMPLIANCE
# =====================================================================

validate_skill9() {
    fase_header "SKILL 9" "PAD-CONTRACT Compliance"

    # Check 1: No /opt/github/ in shell scripts
    local scripts_ok=true
    if [[ -d "$SYNCTACLES_APP/scripts" ]]; then
        # Use grep with exclusions to avoid self-detection
        if grep -rn --exclude="validate_*.sh" "$FORBIDDEN_PATH" "$SYNCTACLES_APP/scripts/" 2>/dev/null | grep -v "^Binary"; then
            scripts_ok=false
        fi
    fi

    if $scripts_ok; then
        ok "No $FORBIDDEN_PATH in scripts"
    else
        fail "Found $FORBIDDEN_PATH in scripts"
    fi

    # Check 2: No /opt/github/ in Python code
    local python_ok=true
    if [[ -d "$SYNCTACLES_APP/sparkcrawler_db" ]] || [[ -d "$SYNCTACLES_APP/synctacles_db" ]]; then
        if grep -rn "$FORBIDDEN_PATH" "$SYNCTACLES_APP/sparkcrawler_db/" "$SYNCTACLES_APP/synctacles_db/" 2>/dev/null | grep -v "^Binary"; then
            python_ok=false
        fi
    fi

    if $python_ok; then
        ok "No $FORBIDDEN_PATH in Python code"
    else
        fail "Found $FORBIDDEN_PATH in Python code"
    fi

    # Check 3: SYNCTACLES_LOG_DIR configured
    if [[ -f "$ENV_FILE" ]]; then
        if grep -q "^SYNCTACLES_LOG_DIR=" "$ENV_FILE" 2>/dev/null; then
            ok "SYNCTACLES_LOG_DIR configured"
        else
            fail "SYNCTACLES_LOG_DIR NOT in .env"
        fi
    else
        fail ".env file missing"
    fi

    # Check 4: TENNET_API_BASE configured (new requirement)
    if [[ -f "$ENV_FILE" ]]; then
        if grep -q "^TENNET_API_BASE=" "$ENV_FILE" 2>/dev/null; then
            ok "TENNET_API_BASE configured"
        else
            warn "TENNET_API_BASE NOT in .env (optional)"
        fi
    fi

    # Check 5: Log directories exist
    local log_dirs="collectors/entso_e_raw collectors/tennet_raw api scheduler importers normalizers"
    local logs_ok=true

    for subdir in $log_dirs; do
        if [[ ! -d "$SYNCTACLES_LOGS/$subdir" ]]; then
            logs_ok=false
            break
        fi
    done

    if $logs_ok; then
        ok "Log directories exist"
    else
        fail "Log directories incomplete ($SYNCTACLES_LOGS)"
    fi
}

# =====================================================================
# DATABASE SCHEMA VALIDATION
# =====================================================================

validate_database() {
    fase_header "DATABASE" "Schema Validation"

    if ! can_access_db; then
        fail "Cannot connect to database"
        return
    fi

    # Alembic migrations applied
    if run_as_synctacles psql synctacles -c "\dt" 2>/dev/null | grep -q "alembic_version"; then
        ok "Alembic migrations applied"
    else
        warn "Alembic migrations NOT applied"
    fi

    # Required tables exist
    local missing_tables=""
    local tables_found=0

    for table in $REQUIRED_TABLES; do
        if run_as_synctacles psql synctacles -t -c "\dt" 2>/dev/null | grep -qw "$table"; then
            ((tables_found++))
        else
            missing_tables="${missing_tables}${table}, "
        fi
    done

    local total_required
    total_required=$(echo "$REQUIRED_TABLES" | wc -w)

    if [[ $tables_found -eq $total_required ]]; then
        ok "Required tables exist ($tables_found/$total_required)"
    elif [[ $tables_found -gt 0 ]]; then
        warn "Some tables missing ($tables_found/$total_required)"
    else
        fail "No required tables found"
    fi
}

# =====================================================================
# RUNTIME HEALTH CHECKS
# =====================================================================

validate_runtime() {
    fase_header "RUNTIME" "Health Checks"

    # API responding
    if has_command curl; then
        local health_response
        health_response=$(curl -sf --connect-timeout 5 http://localhost:8000/health 2>/dev/null)

        if [[ -n "$health_response" ]]; then
            if has_command jq; then
                local api_version
                api_version=$(echo "$health_response" | jq -r '.version // "unknown"' 2>/dev/null)
                ok "API responding (version: $api_version)"
            else
                ok "API responding"
            fi
        else
            fail "API NOT responding"
        fi
    else
        warn "Cannot check API (curl not available)"
    fi

    # Timers active
    local timer_count
    timer_count=$(systemctl list-timers synctacles-* --no-legend 2>/dev/null | wc -l)

    if [[ $timer_count -ge 4 ]]; then
        ok "Timers active ($timer_count)"
    elif [[ $timer_count -ge 1 ]]; then
        warn "Only $timer_count timer(s) active (expected ≥4)"
    else
        fail "No synctacles timers active"
    fi

    # Recent collector execution (last 2 hours)
    if journalctl -u synctacles-collector.service --since "2 hours ago" 2>/dev/null | grep -qiE "Completed|Success|finished"; then
        ok "Recent collector execution"
    else
        warn "No collector execution in last 2 hours"
    fi
}

# =====================================================================
# SYSTEMD UNIT DETAILED CHECK
# =====================================================================

validate_systemd_units() {
    fase_header "SYSTEMD" "Unit Compliance"

    local units_found=0
    local units_compliant=0

    for unit in /etc/systemd/system/synctacles-*.service; do
        [[ -f "$unit" ]] || continue
        ((units_found++))

        local unit_name
        unit_name=$(basename "$unit")
        local issues=""

        # Check 1: No /opt/github/ in ExecStart
        if grep -qE "ExecStart=.*${FORBIDDEN_PATH}" "$unit" 2>/dev/null; then
            issues="${issues}ExecStart has ${FORBIDDEN_PATH}; "
        fi

        # Check 2: WorkingDirectory should be /opt/synctacles/app
        if grep -q "WorkingDirectory=" "$unit" 2>/dev/null; then
            local workdir
            workdir=$(grep "WorkingDirectory=" "$unit" | head -1 | cut -d= -f2)
            if [[ "$workdir" != "/opt/synctacles/app" ]] && [[ "$workdir" != "/opt/synctacles/app/" ]]; then
                issues="${issues}WorkingDirectory=$workdir; "
            fi
        fi

        if [[ -z "$issues" ]]; then
            ((units_compliant++))
        else
            fail "$unit_name: $issues"
        fi
    done

    if [[ $units_found -eq 0 ]]; then
        warn "No synctacles systemd units found"
    elif [[ $units_compliant -eq $units_found ]]; then
        ok "All $units_found units compliant"
    else
        warn "$units_compliant/$units_found units compliant"
    fi
}

# =====================================================================
# MAIN
# =====================================================================

main() {
    header "SYNCTACLES Setup Validator v${SCRIPT_VERSION}"
    echo "Validates against: Installer v${INSTALLER_VERSION} + SKILL 9"

    check_permissions

    # Run all validations
    validate_fase1
    validate_fase2
    validate_fase3
    validate_fase4
    validate_fase5
    validate_skill9
    validate_database
    validate_runtime
    validate_systemd_units

    # Summary
    echo ""
    echo "==============================================="
    local TOTAL=$((PASSED + FAILED + WARNINGS))

    echo "RESULT: ${PASSED}/${TOTAL} checks passed"

    if [[ $FAILED -gt 0 ]]; then
        echo -e "STATUS: ${CHECK_FAIL} ${RED}FAILURES DETECTED${NC} ($FAILED failures, $WARNINGS warnings)"
    elif [[ $WARNINGS -gt 0 ]]; then
        echo -e "STATUS: ${CHECK_WARN} ${YELLOW}PASSED WITH WARNINGS${NC} ($WARNINGS warnings)"
    else
        echo -e "STATUS: ${CHECK_PASS} ${GREEN}PRODUCTION READY${NC}"
    fi

    echo "==============================================="

    # Exit codes: 0=pass, 1=warnings only, 2=failures
    if [[ $FAILED -gt 0 ]]; then
        exit 2
    elif [[ $WARNINGS -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Run main
main "$@"
