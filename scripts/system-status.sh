#!/bin/bash
# =============================================================================
# SYNCTACLES System Status Report
# =============================================================================
# Shows health status of all systems in one consolidated report
# Run from DEV server: ./scripts/system-status.sh
# =============================================================================

set -o pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
PROMETHEUS_URL="http://localhost:9090"
MONITORING_SSH="ssh cc-hub \"ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135\""

# Thresholds
CPU_WARN=70
CPU_CRIT=90
MEM_WARN=80
MEM_CRIT=95
DISK_WARN=80
DISK_CRIT=90
SSL_WARN_DAYS=14
SSL_CRIT_DAYS=7

# =============================================================================
# Helper Functions
# =============================================================================

print_header() {
    echo ""
    echo -e "${BOLD}${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${BLUE}  $1${NC}"
    echo -e "${BOLD}${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
}

print_section() {
    echo ""
    echo -e "${BOLD}${CYAN}─── $1 ───${NC}"
}

status_icon() {
    case $1 in
        "up"|"ok"|"healthy"|"active"|"running")
            echo -e "${GREEN}✓${NC}"
            ;;
        "warn"|"warning")
            echo -e "${YELLOW}⚠${NC}"
            ;;
        "down"|"error"|"critical"|"failed"|"inactive")
            echo -e "${RED}✗${NC}"
            ;;
        *)
            echo -e "${YELLOW}?${NC}"
            ;;
    esac
}

format_status() {
    local status=$1
    case $status in
        "up"|"ok"|"healthy"|"active"|"running")
            echo -e "${GREEN}$status${NC}"
            ;;
        "warn"|"warning")
            echo -e "${YELLOW}$status${NC}"
            ;;
        "down"|"error"|"critical"|"failed"|"inactive")
            echo -e "${RED}$status${NC}"
            ;;
        *)
            echo -e "${YELLOW}$status${NC}"
            ;;
    esac
}

query_prometheus() {
    local query=$1
    ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'curl -s \"http://localhost:9090/api/v1/query?query=$query\"'" 2>/dev/null
}

# =============================================================================
# Report Header
# =============================================================================

clear
echo -e "${BOLD}"
echo "  ____  _  _ _  _  ___ _____  _    ___ _    ___ ____  "
echo " / ___|| || | \\| |/ __|_   _|/_\\  / __| |  | __/ ___| "
echo " \\___ \\| || | .\` | (__  | | / _ \\| (__| |__| _|\\___ \\ "
echo " |____/ \\_, |_|\\_|\\___| |_|/_/ \\_\\\\___|____|___|____/ "
echo "        |__/                                          "
echo -e "${NC}"
echo -e "${BOLD}System Status Report${NC}"
echo -e "Generated: $(date '+%Y-%m-%d %H:%M:%S')"
echo -e "Server: $(hostname)"

# =============================================================================
# API Health Check
# =============================================================================

print_header "API & PIPELINE HEALTH"

print_section "Production (api.synctacles.com)"

# PROD API Health
prod_api=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://api.synctacles.com/health 2>/dev/null)
if [ "$prod_api" = "200" ]; then
    prod_api_status="up"
else
    prod_api_status="down"
fi
echo -e "  $(status_icon $prod_api_status) API Health:      $(format_status $prod_api_status) (HTTP $prod_api)"

# PROD Pipeline Health
prod_pipeline_response=$(curl -s --connect-timeout 5 https://api.synctacles.com/v1/pipeline/health 2>/dev/null)
prod_pipeline_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://api.synctacles.com/v1/pipeline/health 2>/dev/null)
if [ "$prod_pipeline_code" = "200" ]; then
    prod_pipeline_status="up"
    # Extract pipeline details if available
    prod_collector=$(echo "$prod_pipeline_response" | grep -o '"collector":[^,}]*' | cut -d: -f2 | tr -d '"' 2>/dev/null)
    prod_importer=$(echo "$prod_pipeline_response" | grep -o '"importer":[^,}]*' | cut -d: -f2 | tr -d '"' 2>/dev/null)
else
    prod_pipeline_status="down"
fi
echo -e "  $(status_icon $prod_pipeline_status) Pipeline Health: $(format_status $prod_pipeline_status) (HTTP $prod_pipeline_code)"

print_section "Development (dev.synctacles.com)"

# DEV API Health
dev_api=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://dev.synctacles.com/health 2>/dev/null)
if [ "$dev_api" = "200" ]; then
    dev_api_status="up"
else
    dev_api_status="down"
fi
echo -e "  $(status_icon $dev_api_status) API Health:      $(format_status $dev_api_status) (HTTP $dev_api)"

# DEV Pipeline Health
dev_pipeline_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://dev.synctacles.com/v1/pipeline/health 2>/dev/null)
if [ "$dev_pipeline_code" = "200" ]; then
    dev_pipeline_status="up"
else
    dev_pipeline_status="down"
fi
echo -e "  $(status_icon $dev_pipeline_status) Pipeline Health: $(format_status $dev_pipeline_status) (HTTP $dev_pipeline_code)"

# =============================================================================
# SSL Certificates
# =============================================================================

print_header "SSL CERTIFICATES"

check_ssl_expiry() {
    local domain=$1
    local expiry_date=$(echo | openssl s_client -servername "$domain" -connect "$domain:443" 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)

    if [ -n "$expiry_date" ]; then
        local expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null)
        local now_epoch=$(date +%s)
        local days_left=$(( (expiry_epoch - now_epoch) / 86400 ))

        local status="ok"
        if [ $days_left -lt $SSL_CRIT_DAYS ]; then
            status="critical"
        elif [ $days_left -lt $SSL_WARN_DAYS ]; then
            status="warn"
        fi

        echo -e "  $(status_icon $status) $domain: ${days_left} days remaining"
    else
        echo -e "  $(status_icon error) $domain: Unable to check"
    fi
}

check_ssl_expiry "api.synctacles.com"
check_ssl_expiry "synctacles.com"
check_ssl_expiry "dev.synctacles.com"

# =============================================================================
# System Resources (from Prometheus)
# =============================================================================

print_header "SYSTEM RESOURCES"

get_resource_metrics() {
    local env=$1
    local label=$2

    print_section "$label"

    # URL encode the queries
    local cpu_query="100%20-%20(avg(rate(node_cpu_seconds_total%7Benvironment%3D%22${env}%22%2Cmode%3D%22idle%22%7D%5B5m%5D))%20*%20100)"
    local mem_query="(1%20-%20(node_memory_MemAvailable_bytes%7Benvironment%3D%22${env}%22%7D%20%2F%20node_memory_MemTotal_bytes%7Benvironment%3D%22${env}%22%7D))%20*%20100"
    local disk_query="(1%20-%20(node_filesystem_avail_bytes%7Benvironment%3D%22${env}%22%2Cmountpoint%3D%22%2F%22%7D%20%2F%20node_filesystem_size_bytes%7Benvironment%3D%22${env}%22%2Cmountpoint%3D%22%2F%22%7D))%20*%20100"

    # CPU Usage
    local cpu_result=$(ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'curl -s \"http://localhost:9090/api/v1/query?query=${cpu_query}\"'" 2>/dev/null)
    local cpu=$(echo "$cpu_result" | grep -oP '"value":\[\d+\.?\d*,"\K[^"]+' | head -1)
    if [ -n "$cpu" ] && [ "$cpu" != "null" ]; then
        cpu=$(printf "%.1f" "$cpu")
        local cpu_status="ok"
        if (( $(echo "$cpu > $CPU_CRIT" | bc -l) )); then
            cpu_status="critical"
        elif (( $(echo "$cpu > $CPU_WARN" | bc -l) )); then
            cpu_status="warn"
        fi
        echo -e "  $(status_icon $cpu_status) CPU Usage:    ${cpu}%"
    else
        echo -e "  $(status_icon error) CPU Usage:    N/A"
    fi

    # Memory Usage
    local mem_result=$(ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'curl -s \"http://localhost:9090/api/v1/query?query=${mem_query}\"'" 2>/dev/null)
    local mem=$(echo "$mem_result" | grep -oP '"value":\[\d+\.?\d*,"\K[^"]+' | head -1)
    if [ -n "$mem" ] && [ "$mem" != "null" ]; then
        mem=$(printf "%.1f" "$mem")
        local mem_status="ok"
        if (( $(echo "$mem > $MEM_CRIT" | bc -l) )); then
            mem_status="critical"
        elif (( $(echo "$mem > $MEM_WARN" | bc -l) )); then
            mem_status="warn"
        fi
        echo -e "  $(status_icon $mem_status) Memory Usage: ${mem}%"
    else
        echo -e "  $(status_icon error) Memory Usage: N/A"
    fi

    # Disk Usage
    local disk_result=$(ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'curl -s \"http://localhost:9090/api/v1/query?query=${disk_query}\"'" 2>/dev/null)
    local disk=$(echo "$disk_result" | grep -oP '"value":\[\d+\.?\d*,"\K[^"]+' | head -1)
    if [ -n "$disk" ] && [ "$disk" != "null" ]; then
        disk=$(printf "%.1f" "$disk")
        local disk_status="ok"
        if (( $(echo "$disk > $DISK_CRIT" | bc -l) )); then
            disk_status="critical"
        elif (( $(echo "$disk > $DISK_WARN" | bc -l) )); then
            disk_status="warn"
        fi
        echo -e "  $(status_icon $disk_status) Disk Usage:   ${disk}%"
    else
        echo -e "  $(status_icon error) Disk Usage:   N/A"
    fi
}

get_resource_metrics "prod" "Production (46.62.212.227)"
get_resource_metrics "dev" "Development (135.181.255.83)"

# =============================================================================
# Systemd Services
# =============================================================================

print_header "SYSTEMD SERVICES"

print_section "Production (46.62.212.227)"
for svc in synctacles-api synctacles-collector.timer synctacles-importer.timer synctacles-normalizer.timer synctacles-health.timer; do
    status=$(ssh cc-hub "ssh synct-prod 'systemctl is-active $svc'" 2>/dev/null)
    if [ "$status" = "active" ]; then
        echo -e "  $(status_icon active) $svc"
    else
        echo -e "  $(status_icon failed) $svc (${status:-unknown})"
    fi
done

print_section "Development (135.181.255.83)"
for svc in synctacles-dev-api synctacles-dev-collector.timer synctacles-dev-importer.timer synctacles-dev-normalizer.timer synctacles-dev-health.timer; do
    status=$(systemctl is-active $svc 2>/dev/null)
    if [ "$status" = "active" ]; then
        echo -e "  $(status_icon active) $svc"
    else
        echo -e "  $(status_icon failed) $svc (${status:-unknown})"
    fi
done

# =============================================================================
# Prometheus Targets
# =============================================================================

print_header "PROMETHEUS MONITORING TARGETS"

targets_json=$(query_prometheus "" | sed 's/query=/targets/g')
# Alternative: direct targets API
targets_result=$(ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'curl -s http://localhost:9090/api/v1/targets'" 2>/dev/null)

if [ -n "$targets_result" ]; then
    echo "$targets_result" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    targets = data.get('data', {}).get('activeTargets', [])

    up_count = sum(1 for t in targets if t.get('health') == 'up')
    down_count = sum(1 for t in targets if t.get('health') != 'up')

    print(f'  Total targets: {len(targets)} ({up_count} up, {down_count} down)')
    print()

    for t in sorted(targets, key=lambda x: (x.get('labels', {}).get('environment', ''), x.get('labels', {}).get('job', ''))):
        job = t.get('labels', {}).get('job', 'unknown')
        instance = t.get('labels', {}).get('instance', 'unknown')
        env = t.get('labels', {}).get('environment', 'unknown')
        health = t.get('health', 'unknown')

        icon = '\033[0;32m✓\033[0m' if health == 'up' else '\033[0;31m✗\033[0m'
        status_color = '\033[0;32m' if health == 'up' else '\033[0;31m'

        print(f'  {icon} [{env:4}] {job:25} {status_color}{health}\033[0m')
except Exception as e:
    print(f'  Error parsing targets: {e}')
" 2>/dev/null
else
    echo -e "  $(status_icon error) Unable to fetch Prometheus targets"
fi

# =============================================================================
# Recent Alerts
# =============================================================================

print_header "ALERTMANAGER STATUS"

alerts_result=$(ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'curl -s http://localhost:9093/api/v2/alerts'" 2>/dev/null)

if [ -n "$alerts_result" ] && [ "$alerts_result" != "[]" ]; then
    alert_count=$(echo "$alerts_result" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null)
    if [ "$alert_count" -gt 0 ] 2>/dev/null; then
        echo -e "  ${YELLOW}⚠ Active alerts: $alert_count${NC}"
        echo "$alerts_result" | python3 -c "
import json, sys
alerts = json.load(sys.stdin)
for a in alerts[:5]:
    name = a.get('labels', {}).get('alertname', 'unknown')
    severity = a.get('labels', {}).get('severity', 'unknown')
    print(f'    - {name} ({severity})')
" 2>/dev/null
    else
        echo -e "  $(status_icon ok) No active alerts"
    fi
else
    echo -e "  $(status_icon ok) No active alerts"
fi

# =============================================================================
# Summary
# =============================================================================

print_header "SUMMARY"

# Count statuses
total_checks=0
ok_checks=0
warn_checks=0
fail_checks=0

# API checks
if [ "$prod_api_status" = "up" ]; then ((ok_checks++)); else ((fail_checks++)); fi; ((total_checks++))
if [ "$prod_pipeline_status" = "up" ]; then ((ok_checks++)); else ((fail_checks++)); fi; ((total_checks++))
if [ "$dev_api_status" = "up" ]; then ((ok_checks++)); else ((fail_checks++)); fi; ((total_checks++))
if [ "$dev_pipeline_status" = "up" ]; then ((ok_checks++)); else ((fail_checks++)); fi; ((total_checks++))

echo ""
echo -e "  ${GREEN}✓ Healthy:${NC}  $ok_checks"
echo -e "  ${YELLOW}⚠ Warning:${NC}  $warn_checks"
echo -e "  ${RED}✗ Critical:${NC} $fail_checks"
echo ""

if [ $fail_checks -eq 0 ] && [ $warn_checks -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}All systems operational${NC}"
elif [ $fail_checks -eq 0 ]; then
    echo -e "  ${YELLOW}${BOLD}Systems operational with warnings${NC}"
else
    echo -e "  ${RED}${BOLD}Some systems require attention${NC}"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
echo -e "Report completed at $(date '+%H:%M:%S')"
echo ""
