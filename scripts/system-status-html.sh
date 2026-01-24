#!/bin/bash
# =============================================================================
# SYNCTACLES System Status Report - HTML Version
# =============================================================================
# Generates an HTML status page for all SYNCTACLES systems
# Designed to run on cc-hub as central monitoring point
#
# Usage:
#   ./system-status-html.sh > /var/www/html/status.html
#   Or with auto-refresh cron: */5 * * * * /opt/scripts/system-status-html.sh > /var/www/html/status.html
# =============================================================================

# Configuration
PROMETHEUS_HOST="monitoring@77.42.41.135"
PROMETHEUS_KEY="~/.ssh/id_monitoring"
PROD_HOST="synct-prod"
DEV_HOST="synct-dev"

# Thresholds
CPU_WARN=70
CPU_CRIT=90
MEM_WARN=80
MEM_CRIT=95
DISK_WARN=80
DISK_CRIT=90
SSL_WARN_DAYS=14
SSL_CRIT_DAYS=7

# Timestamp
GENERATED=$(date '+%Y-%m-%d %H:%M:%S')
REFRESH_SECONDS=300

# =============================================================================
# Helper Functions
# =============================================================================

status_class() {
    case $1 in
        "up"|"ok"|"healthy"|"active"|"running") echo "status-ok" ;;
        "warn"|"warning") echo "status-warn" ;;
        "down"|"error"|"critical"|"failed"|"inactive") echo "status-error" ;;
        *) echo "status-unknown" ;;
    esac
}

status_icon() {
    case $1 in
        "up"|"ok"|"healthy"|"active"|"running") echo "✓" ;;
        "warn"|"warning") echo "⚠" ;;
        "down"|"error"|"critical"|"failed"|"inactive") echo "✗" ;;
        *) echo "?" ;;
    esac
}

query_prometheus() {
    local query=$1
    ssh -i $PROMETHEUS_KEY $PROMETHEUS_HOST "curl -s 'http://localhost:9090/api/v1/query?query=$query'" 2>/dev/null
}

# =============================================================================
# Collect Data
# =============================================================================

# API Health
prod_api_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://api.synctacles.com/health 2>/dev/null)
prod_pipeline_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://api.synctacles.com/v1/pipeline/health 2>/dev/null)
dev_api_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://dev.synctacles.com/health 2>/dev/null)
dev_pipeline_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 https://dev.synctacles.com/v1/pipeline/health 2>/dev/null)

[ "$prod_api_code" = "200" ] && prod_api_status="ok" || prod_api_status="error"
[ "$prod_pipeline_code" = "200" ] && prod_pipeline_status="ok" || prod_pipeline_status="error"
[ "$dev_api_code" = "200" ] && dev_api_status="ok" || dev_api_status="error"
[ "$dev_pipeline_code" = "200" ] && dev_pipeline_status="ok" || dev_pipeline_status="error"

# SSL Certificates
get_ssl_days() {
    local domain=$1
    local expiry_date=$(echo | openssl s_client -servername "$domain" -connect "$domain:443" 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
    if [ -n "$expiry_date" ]; then
        local expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null)
        local now_epoch=$(date +%s)
        echo $(( (expiry_epoch - now_epoch) / 86400 ))
    else
        echo "N/A"
    fi
}

ssl_api=$(get_ssl_days "api.synctacles.com")
ssl_main=$(get_ssl_days "synctacles.com")
ssl_dev=$(get_ssl_days "dev.synctacles.com")

get_ssl_status() {
    local days=$1
    if [ "$days" = "N/A" ]; then echo "error"
    elif [ "$days" -lt $SSL_CRIT_DAYS ]; then echo "error"
    elif [ "$days" -lt $SSL_WARN_DAYS ]; then echo "warn"
    else echo "ok"
    fi
}

# System Resources from Prometheus
get_metric() {
    local env=$1
    local metric=$2
    local query=""

    case $metric in
        cpu) query="100%20-%20(avg(rate(node_cpu_seconds_total%7Benvironment%3D%22${env}%22%2Cmode%3D%22idle%22%7D%5B5m%5D))%20*%20100)" ;;
        mem) query="(1%20-%20(node_memory_MemAvailable_bytes%7Benvironment%3D%22${env}%22%7D%20%2F%20node_memory_MemTotal_bytes%7Benvironment%3D%22${env}%22%7D))%20*%20100" ;;
        disk) query="(1%20-%20(node_filesystem_avail_bytes%7Benvironment%3D%22${env}%22%2Cmountpoint%3D%22%2F%22%7D%20%2F%20node_filesystem_size_bytes%7Benvironment%3D%22${env}%22%2Cmountpoint%3D%22%2F%22%7D))%20*%20100" ;;
    esac

    local result=$(ssh -i $PROMETHEUS_KEY $PROMETHEUS_HOST "curl -s 'http://localhost:9090/api/v1/query?query=${query}'" 2>/dev/null)
    echo "$result" | grep -oP '"value":\[\d+\.?\d*,"\K[^"]+' | head -1
}

prod_cpu=$(get_metric "prod" "cpu")
prod_mem=$(get_metric "prod" "mem")
prod_disk=$(get_metric "prod" "disk")
dev_cpu=$(get_metric "dev" "cpu")
dev_mem=$(get_metric "dev" "mem")
dev_disk=$(get_metric "dev" "disk")

get_resource_status() {
    local value=$1
    local warn=$2
    local crit=$3
    if [ -z "$value" ] || [ "$value" = "null" ]; then echo "error"
    elif (( $(echo "$value > $crit" | bc -l 2>/dev/null) )); then echo "error"
    elif (( $(echo "$value > $warn" | bc -l 2>/dev/null) )); then echo "warn"
    else echo "ok"
    fi
}

# Services
get_service_status() {
    local host=$1
    local service=$2
    if [ "$host" = "local" ]; then
        systemctl is-active "$service" 2>/dev/null
    else
        ssh "$host" "systemctl is-active $service" 2>/dev/null
    fi
}

# Prometheus Targets
targets_json=$(ssh -i $PROMETHEUS_KEY $PROMETHEUS_HOST "curl -s http://localhost:9090/api/v1/targets" 2>/dev/null)
targets_up=$(echo "$targets_json" | grep -o '"health":"up"' | wc -l)
targets_down=$(echo "$targets_json" | grep -o '"health":"down"' | wc -l)
targets_total=$((targets_up + targets_down))

# Alerts
alerts_json=$(ssh -i $PROMETHEUS_KEY $PROMETHEUS_HOST "curl -s http://localhost:9093/api/v2/alerts" 2>/dev/null)
alerts_count=$(echo "$alerts_json" | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d))" 2>/dev/null || echo "0")

# =============================================================================
# Generate HTML
# =============================================================================

cat << 'HTMLHEAD'
<!DOCTYPE html>
<html lang="nl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
HTMLHEAD

echo "    <meta http-equiv=\"refresh\" content=\"${REFRESH_SECONDS}\">"

cat << 'HTMLSTYLE'
    <title>SYNCTACLES Status</title>
    <style>
        :root {
            --bg-dark: #0d1117;
            --bg-card: #161b22;
            --bg-card-hover: #1c2128;
            --border: #30363d;
            --text: #c9d1d9;
            --text-muted: #8b949e;
            --accent: #58a6ff;
            --success: #3fb950;
            --warning: #d29922;
            --error: #f85149;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
            background: var(--bg-dark);
            color: var(--text);
            line-height: 1.5;
            min-height: 100vh;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem;
        }

        header {
            text-align: center;
            margin-bottom: 2rem;
            padding-bottom: 2rem;
            border-bottom: 1px solid var(--border);
        }

        .logo {
            font-size: 2.5rem;
            font-weight: 700;
            color: var(--accent);
            letter-spacing: 0.1em;
            margin-bottom: 0.5rem;
        }

        .subtitle {
            color: var(--text-muted);
            font-size: 1rem;
        }

        .timestamp {
            color: var(--text-muted);
            font-size: 0.875rem;
            margin-top: 1rem;
        }

        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }

        .card {
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 8px;
            overflow: hidden;
        }

        .card-header {
            background: rgba(88, 166, 255, 0.1);
            padding: 1rem 1.5rem;
            border-bottom: 1px solid var(--border);
            font-weight: 600;
            font-size: 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .card-header svg {
            width: 20px;
            height: 20px;
            fill: var(--accent);
        }

        .card-body {
            padding: 1rem 1.5rem;
        }

        .status-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.75rem 0;
            border-bottom: 1px solid var(--border);
        }

        .status-row:last-child {
            border-bottom: none;
        }

        .status-label {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .status-icon {
            width: 24px;
            height: 24px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            font-size: 14px;
        }

        .status-ok .status-icon {
            background: rgba(63, 185, 80, 0.2);
            color: var(--success);
        }

        .status-warn .status-icon {
            background: rgba(210, 153, 34, 0.2);
            color: var(--warning);
        }

        .status-error .status-icon {
            background: rgba(248, 81, 73, 0.2);
            color: var(--error);
        }

        .status-unknown .status-icon {
            background: rgba(139, 148, 158, 0.2);
            color: var(--text-muted);
        }

        .status-value {
            font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
            font-size: 0.875rem;
        }

        .status-ok .status-value { color: var(--success); }
        .status-warn .status-value { color: var(--warning); }
        .status-error .status-value { color: var(--error); }

        .env-badge {
            display: inline-block;
            padding: 0.125rem 0.5rem;
            border-radius: 4px;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            margin-right: 0.5rem;
        }

        .env-prod {
            background: rgba(248, 81, 73, 0.2);
            color: var(--error);
        }

        .env-dev {
            background: rgba(63, 185, 80, 0.2);
            color: var(--success);
        }

        .progress-bar {
            height: 8px;
            background: var(--border);
            border-radius: 4px;
            overflow: hidden;
            margin-top: 0.25rem;
        }

        .progress-fill {
            height: 100%;
            border-radius: 4px;
            transition: width 0.3s ease;
        }

        .progress-ok { background: var(--success); }
        .progress-warn { background: var(--warning); }
        .progress-error { background: var(--error); }

        .summary-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 1rem;
            margin-bottom: 2rem;
        }

        .summary-card {
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 1.5rem;
            text-align: center;
        }

        .summary-value {
            font-size: 2.5rem;
            font-weight: 700;
            line-height: 1;
        }

        .summary-label {
            color: var(--text-muted);
            font-size: 0.875rem;
            margin-top: 0.5rem;
        }

        .summary-ok .summary-value { color: var(--success); }
        .summary-warn .summary-value { color: var(--warning); }
        .summary-error .summary-value { color: var(--error); }
        .summary-total .summary-value { color: var(--accent); }

        .targets-list {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 0.5rem;
        }

        .target-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem;
            background: var(--bg-dark);
            border-radius: 4px;
            font-size: 0.8rem;
            font-family: monospace;
        }

        .target-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
        }

        .target-dot.up { background: var(--success); }
        .target-dot.down { background: var(--error); }

        footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-muted);
            font-size: 0.875rem;
            border-top: 1px solid var(--border);
        }

        @media (max-width: 768px) {
            .container { padding: 1rem; }
            .grid { grid-template-columns: 1fr; }
            .summary-grid { grid-template-columns: repeat(2, 1fr); }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="logo">SYNCTACLES</div>
            <div class="subtitle">System Status Dashboard</div>
HTMLSTYLE

echo "            <div class=\"timestamp\">Last updated: ${GENERATED} • Auto-refresh: ${REFRESH_SECONDS}s</div>"

cat << 'HTMLBODY1'
        </header>

        <!-- Summary Cards -->
        <div class="summary-grid">
HTMLBODY1

# Calculate totals
total_ok=0
total_warn=0
total_error=0

# Count API statuses
if [ "$prod_api_status" = "ok" ]; then ((total_ok++)); else ((total_error++)); fi
if [ "$prod_pipeline_status" = "ok" ]; then ((total_ok++)); else ((total_error++)); fi
if [ "$dev_api_status" = "ok" ]; then ((total_ok++)); else ((total_error++)); fi
if [ "$dev_pipeline_status" = "ok" ]; then ((total_ok++)); else ((total_error++)); fi

# Count SSL statuses
for ssl in "$ssl_api" "$ssl_main" "$ssl_dev"; do
    status=$(get_ssl_status "$ssl")
    [ "$status" = "ok" ] && ((total_ok++))
    [ "$status" = "warn" ] && ((total_warn++))
    [ "$status" = "error" ] && ((total_error++))
done

# Count resource statuses
for val in "$prod_cpu" "$prod_mem" "$prod_disk" "$dev_cpu" "$dev_mem" "$dev_disk"; do
    if [ -n "$val" ] && [ "$val" != "null" ]; then
        ((total_ok++))
    else
        ((total_error++))
    fi
done

total_checks=$((total_ok + total_warn + total_error))

echo "            <div class=\"summary-card summary-total\"><div class=\"summary-value\">${total_checks}</div><div class=\"summary-label\">Total Checks</div></div>"
echo "            <div class=\"summary-card summary-ok\"><div class=\"summary-value\">${total_ok}</div><div class=\"summary-label\">Healthy</div></div>"
echo "            <div class=\"summary-card summary-warn\"><div class=\"summary-value\">${total_warn}</div><div class=\"summary-label\">Warnings</div></div>"
echo "            <div class=\"summary-card summary-error\"><div class=\"summary-value\">${total_error}</div><div class=\"summary-label\">Critical</div></div>"

cat << 'HTMLBODY2'
        </div>

        <div class="grid">
            <!-- API Health -->
            <div class="card">
                <div class="card-header">
                    <svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>
                    API & Pipeline Health
                </div>
                <div class="card-body">
HTMLBODY2

# Production API
echo "                    <div class=\"status-row $(status_class $prod_api_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $prod_api_status)</div><span class=\"env-badge env-prod\">PROD</span>API Health</div>"
echo "                        <div class=\"status-value\">HTTP ${prod_api_code}</div>"
echo "                    </div>"

echo "                    <div class=\"status-row $(status_class $prod_pipeline_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $prod_pipeline_status)</div><span class=\"env-badge env-prod\">PROD</span>Pipeline Health</div>"
echo "                        <div class=\"status-value\">HTTP ${prod_pipeline_code}</div>"
echo "                    </div>"

echo "                    <div class=\"status-row $(status_class $dev_api_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $dev_api_status)</div><span class=\"env-badge env-dev\">DEV</span>API Health</div>"
echo "                        <div class=\"status-value\">HTTP ${dev_api_code}</div>"
echo "                    </div>"

echo "                    <div class=\"status-row $(status_class $dev_pipeline_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $dev_pipeline_status)</div><span class=\"env-badge env-dev\">DEV</span>Pipeline Health</div>"
echo "                        <div class=\"status-value\">HTTP ${dev_pipeline_code}</div>"
echo "                    </div>"

cat << 'HTMLBODY3'
                </div>
            </div>

            <!-- SSL Certificates -->
            <div class="card">
                <div class="card-header">
                    <svg viewBox="0 0 24 24"><path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/></svg>
                    SSL Certificates
                </div>
                <div class="card-body">
HTMLBODY3

ssl_api_status=$(get_ssl_status "$ssl_api")
ssl_main_status=$(get_ssl_status "$ssl_main")
ssl_dev_status=$(get_ssl_status "$ssl_dev")

echo "                    <div class=\"status-row $(status_class $ssl_api_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $ssl_api_status)</div>api.synctacles.com</div>"
echo "                        <div class=\"status-value\">${ssl_api} days</div>"
echo "                    </div>"

echo "                    <div class=\"status-row $(status_class $ssl_main_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $ssl_main_status)</div>synctacles.com</div>"
echo "                        <div class=\"status-value\">${ssl_main} days</div>"
echo "                    </div>"

echo "                    <div class=\"status-row $(status_class $ssl_dev_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $ssl_dev_status)</div>dev.synctacles.com</div>"
echo "                        <div class=\"status-value\">${ssl_dev} days</div>"
echo "                    </div>"

cat << 'HTMLBODY4'
                </div>
            </div>

            <!-- System Resources PROD -->
            <div class="card">
                <div class="card-header">
                    <svg viewBox="0 0 24 24"><path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14zM7 10h2v7H7zm4-3h2v10h-2zm4 6h2v4h-2z"/></svg>
                    <span class="env-badge env-prod">PROD</span> System Resources
                </div>
                <div class="card-body">
HTMLBODY4

# PROD Resources
prod_cpu_status=$(get_resource_status "$prod_cpu" $CPU_WARN $CPU_CRIT)
prod_mem_status=$(get_resource_status "$prod_mem" $MEM_WARN $MEM_CRIT)
prod_disk_status=$(get_resource_status "$prod_disk" $DISK_WARN $DISK_CRIT)

prod_cpu_fmt=$(printf "%.1f" "${prod_cpu:-0}" 2>/dev/null || echo "N/A")
prod_mem_fmt=$(printf "%.1f" "${prod_mem:-0}" 2>/dev/null || echo "N/A")
prod_disk_fmt=$(printf "%.1f" "${prod_disk:-0}" 2>/dev/null || echo "N/A")

echo "                    <div class=\"status-row $(status_class $prod_cpu_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $prod_cpu_status)</div>CPU Usage</div>"
echo "                        <div class=\"status-value\">${prod_cpu_fmt}%</div>"
echo "                    </div>"
echo "                    <div class=\"progress-bar\"><div class=\"progress-fill progress-${prod_cpu_status}\" style=\"width: ${prod_cpu_fmt}%\"></div></div>"

echo "                    <div class=\"status-row $(status_class $prod_mem_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $prod_mem_status)</div>Memory Usage</div>"
echo "                        <div class=\"status-value\">${prod_mem_fmt}%</div>"
echo "                    </div>"
echo "                    <div class=\"progress-bar\"><div class=\"progress-fill progress-${prod_mem_status}\" style=\"width: ${prod_mem_fmt}%\"></div></div>"

echo "                    <div class=\"status-row $(status_class $prod_disk_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $prod_disk_status)</div>Disk Usage</div>"
echo "                        <div class=\"status-value\">${prod_disk_fmt}%</div>"
echo "                    </div>"
echo "                    <div class=\"progress-bar\"><div class=\"progress-fill progress-${prod_disk_status}\" style=\"width: ${prod_disk_fmt}%\"></div></div>"

cat << 'HTMLBODY5'
                </div>
            </div>

            <!-- System Resources DEV -->
            <div class="card">
                <div class="card-header">
                    <svg viewBox="0 0 24 24"><path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14zM7 10h2v7H7zm4-3h2v10h-2zm4 6h2v4h-2z"/></svg>
                    <span class="env-badge env-dev">DEV</span> System Resources
                </div>
                <div class="card-body">
HTMLBODY5

# DEV Resources
dev_cpu_status=$(get_resource_status "$dev_cpu" $CPU_WARN $CPU_CRIT)
dev_mem_status=$(get_resource_status "$dev_mem" $MEM_WARN $MEM_CRIT)
dev_disk_status=$(get_resource_status "$dev_disk" $DISK_WARN $DISK_CRIT)

dev_cpu_fmt=$(printf "%.1f" "${dev_cpu:-0}" 2>/dev/null || echo "N/A")
dev_mem_fmt=$(printf "%.1f" "${dev_mem:-0}" 2>/dev/null || echo "N/A")
dev_disk_fmt=$(printf "%.1f" "${dev_disk:-0}" 2>/dev/null || echo "N/A")

echo "                    <div class=\"status-row $(status_class $dev_cpu_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $dev_cpu_status)</div>CPU Usage</div>"
echo "                        <div class=\"status-value\">${dev_cpu_fmt}%</div>"
echo "                    </div>"
echo "                    <div class=\"progress-bar\"><div class=\"progress-fill progress-${dev_cpu_status}\" style=\"width: ${dev_cpu_fmt}%\"></div></div>"

echo "                    <div class=\"status-row $(status_class $dev_mem_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $dev_mem_status)</div>Memory Usage</div>"
echo "                        <div class=\"status-value\">${dev_mem_fmt}%</div>"
echo "                    </div>"
echo "                    <div class=\"progress-bar\"><div class=\"progress-fill progress-${dev_mem_status}\" style=\"width: ${dev_mem_fmt}%\"></div></div>"

echo "                    <div class=\"status-row $(status_class $dev_disk_status)\">"
echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $dev_disk_status)</div>Disk Usage</div>"
echo "                        <div class=\"status-value\">${dev_disk_fmt}%</div>"
echo "                    </div>"
echo "                    <div class=\"progress-bar\"><div class=\"progress-fill progress-${dev_disk_status}\" style=\"width: ${dev_disk_fmt}%\"></div></div>"

cat << 'HTMLBODY6'
                </div>
            </div>

            <!-- Services PROD -->
            <div class="card">
                <div class="card-header">
                    <svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 17h-2v-2h2v2zm2.07-7.75l-.9.92C13.45 12.9 13 13.5 13 15h-2v-.5c0-1.1.45-2.1 1.17-2.83l1.24-1.26c.37-.36.59-.86.59-1.41 0-1.1-.9-2-2-2s-2 .9-2 2H8c0-2.21 1.79-4 4-4s4 1.79 4 4c0 .88-.36 1.68-.93 2.25z"/></svg>
                    <span class="env-badge env-prod">PROD</span> Systemd Services
                </div>
                <div class="card-body">
HTMLBODY6

# PROD Services
for svc in synctacles-api synctacles-collector.timer synctacles-importer.timer synctacles-normalizer.timer synctacles-health.timer; do
    status=$(ssh $PROD_HOST "systemctl is-active $svc" 2>/dev/null)
    [ "$status" = "active" ] && svc_status="ok" || svc_status="error"
    svc_name=$(echo $svc | sed 's/synctacles-//' | sed 's/.timer//')
    echo "                    <div class=\"status-row $(status_class $svc_status)\">"
    echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $svc_status)</div>${svc_name}</div>"
    echo "                        <div class=\"status-value\">${status:-unknown}</div>"
    echo "                    </div>"
done

cat << 'HTMLBODY7'
                </div>
            </div>

            <!-- Services DEV -->
            <div class="card">
                <div class="card-header">
                    <svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 17h-2v-2h2v2zm2.07-7.75l-.9.92C13.45 12.9 13 13.5 13 15h-2v-.5c0-1.1.45-2.1 1.17-2.83l1.24-1.26c.37-.36.59-.86.59-1.41 0-1.1-.9-2-2-2s-2 .9-2 2H8c0-2.21 1.79-4 4-4s4 1.79 4 4c0 .88-.36 1.68-.93 2.25z"/></svg>
                    <span class="env-badge env-dev">DEV</span> Systemd Services
                </div>
                <div class="card-body">
HTMLBODY7

# DEV Services (note: DEV uses synctacles-dev-* naming)
for svc in synctacles-dev-api synctacles-dev-collector.timer synctacles-dev-importer.timer synctacles-dev-normalizer.timer synctacles-dev-health.timer; do
    status=$(ssh $DEV_HOST "systemctl is-active $svc" 2>/dev/null)
    [ "$status" = "active" ] && svc_status="ok" || svc_status="error"
    svc_name=$(echo $svc | sed 's/synctacles-//' | sed 's/.timer//')
    echo "                    <div class=\"status-row $(status_class $svc_status)\">"
    echo "                        <div class=\"status-label\"><div class=\"status-icon\">$(status_icon $svc_status)</div>${svc_name}</div>"
    echo "                        <div class=\"status-value\">${status:-unknown}</div>"
    echo "                    </div>"
done

cat << 'HTMLBODY8'
                </div>
            </div>
        </div>

        <!-- Prometheus Targets -->
        <div class="card">
            <div class="card-header">
                <svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>
                Prometheus Monitoring Targets
            </div>
            <div class="card-body">
HTMLBODY8

echo "                <p style=\"margin-bottom: 1rem; color: var(--text-muted);\">Total: <strong>${targets_total}</strong> targets (${targets_up} up, ${targets_down} down)</p>"
echo "                <div class=\"targets-list\">"

# Parse and display targets
echo "$targets_json" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    targets = data.get('data', {}).get('activeTargets', [])
    for t in sorted(targets, key=lambda x: (x.get('labels', {}).get('environment', ''), x.get('labels', {}).get('job', ''))):
        job = t.get('labels', {}).get('job', 'unknown')
        env = t.get('labels', {}).get('environment', '?')
        health = t.get('health', 'unknown')
        dot_class = 'up' if health == 'up' else 'down'
        print(f'                    <div class=\"target-item\"><span class=\"target-dot {dot_class}\"></span>[{env}] {job}</div>')
except:
    print('                    <div class=\"target-item\">Unable to load targets</div>')
" 2>/dev/null

cat << 'HTMLFOOTER'
                </div>
            </div>
        </div>

        <footer>
            SYNCTACLES Monitoring • Powered by Prometheus & Alertmanager
        </footer>
    </div>
</body>
</html>
HTMLFOOTER
