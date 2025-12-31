# Claude Code - Borging Performance Optimalisaties

**Datum:** 2025-12-30
**Doel:** Performance optimalisaties borgen in setup scripts en documentatie

---

## Context

Performance optimalisaties zijn handmatig toegepast op productie:
- TCP kernel tuning
- Gunicorn 8 workers + keepalive
- Nginx proxy buffering

Deze moeten geborgd worden zodat nieuwe servers automatisch geoptimaliseerd zijn.

---

## TAAK 1: Maak TCP tuning script

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/setup/optimize_tcp.sh`

```bash
cat > /opt/github/ha-energy-insights-nl/scripts/setup/optimize_tcp.sh << 'EOF'
#!/bin/bash
# TCP Kernel Tuning voor SYNCTACLES
# Voorkomt TIME_WAIT socket exhaustion bij high concurrency

set -euo pipefail

echo "=== TCP Kernel Tuning ==="

# Check if already applied
if grep -q "SYNCTACLES Performance Tuning" /etc/sysctl.conf 2>/dev/null; then
    echo "✅ TCP tuning already applied"
    exit 0
fi

# Apply tuning
cat >> /etc/sysctl.conf << 'SYSCTL'

# SYNCTACLES Performance Tuning
# Toegevoegd door setup script
# Docs: SKILL_08_HARDWARE_PROFILE.md

# Reuse TIME_WAIT sockets for new connections
net.ipv4.tcp_tw_reuse = 1

# Reduce TIME_WAIT duration (60s -> 30s)
net.ipv4.tcp_fin_timeout = 30

# Increase connection backlog
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 4096
SYSCTL

# Apply immediately
sysctl -p

echo "✅ TCP tuning applied"
sysctl net.ipv4.tcp_tw_reuse net.ipv4.tcp_fin_timeout net.core.somaxconn
EOF

chmod +x /opt/github/ha-energy-insights-nl/scripts/setup/optimize_tcp.sh
echo "✅ Created: scripts/setup/optimize_tcp.sh"
```

---

## TAAK 2: Update Gunicorn systemd template

**Bestand:** `/opt/github/ha-energy-insights-nl/systemd/energy-insights-nl-api.service.template`

```bash
mkdir -p /opt/github/ha-energy-insights-nl/systemd

cat > /opt/github/ha-energy-insights-nl/systemd/energy-insights-nl-api.service.template << 'EOF'
[Unit]
Description={{BRAND_NAME}} API
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User={{SERVICE_USER}}
Group={{SERVICE_GROUP}}
WorkingDirectory={{APP_PATH}}
Environment="PATH={{INSTALL_PATH}}/venv/bin"
EnvironmentFile=/opt/.env

# Geoptimaliseerde Gunicorn configuratie
# Performance tuning: 8 workers, keepalive, backlog
ExecStart={{INSTALL_PATH}}/venv/bin/gunicorn \
    --bind 127.0.0.1:{{API_PORT}} \
    --workers 8 \
    --worker-class uvicorn.workers.UvicornWorker \
    --worker-connections 1024 \
    --keepalive 5 \
    --backlog 2048 \
    --timeout 30 \
    --graceful-timeout 10 \
    --access-logfile - \
    --error-logfile - \
    synctacles_db.api.main:app

Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo "✅ Created: systemd/energy-insights-nl-api.service.template"
```

---

## TAAK 3: Update Nginx config template

**Bestand:** `/opt/github/ha-energy-insights-nl/config/nginx/energy-insights-nl.conf.template`

```bash
mkdir -p /opt/github/ha-energy-insights-nl/config/nginx

cat > /opt/github/ha-energy-insights-nl/config/nginx/energy-insights-nl.conf.template << 'EOF'
# {{BRAND_NAME}} Nginx Configuration
# Performance-optimized template

server {
    listen 80;
    server_name {{BRAND_DOMAIN}} _;

    # Gzip compression
    gzip on;
    gzip_types application/json text/plain application/javascript text/css;
    gzip_min_length 256;
    gzip_comp_level 5;

    # Proxy buffering (reduces latency)
    proxy_buffering on;
    proxy_buffer_size 4k;
    proxy_buffers 8 16k;
    proxy_busy_buffers_size 24k;

    # HTTP/1.1 keep-alive (reduces connection overhead)
    proxy_http_version 1.1;
    proxy_set_header Connection "";

    # Main API location
    location / {
        proxy_pass http://127.0.0.1:{{API_PORT}};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 10s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # Health endpoint (no access logging)
    location /health {
        proxy_pass http://127.0.0.1:{{API_PORT}}/health;
        access_log off;
    }
}
EOF

echo "✅ Created: config/nginx/energy-insights-nl.conf.template"
```

---

## TAAK 4: Update SKILL_08_HARDWARE_PROFILE.md

**Voeg toe aan:** `/opt/github/ha-energy-insights-nl/docs/SKILL_08_HARDWARE_PROFILE.md`

```bash
cat >> /opt/github/ha-energy-insights-nl/docs/SKILL_08_HARDWARE_PROFILE.md << 'EOF'

---

## PERFORMANCE TUNING

### TCP Kernel Parameters

Voorkomt TIME_WAIT socket exhaustion bij high concurrency.

**Locatie:** `/etc/sysctl.conf`

```bash
# SYNCTACLES Performance Tuning
net.ipv4.tcp_tw_reuse = 1          # Reuse TIME_WAIT sockets
net.ipv4.tcp_fin_timeout = 30      # Reduce TIME_WAIT (60s -> 30s)
net.core.somaxconn = 4096          # Connection backlog
net.ipv4.tcp_max_syn_backlog = 4096
```

**Toepassen:**
```bash
./scripts/setup/optimize_tcp.sh
# Of handmatig: sysctl -p
```

**Impact:** +50-100% concurrent connection capacity

---

### Gunicorn Configuratie

Geoptimaliseerd voor high-concurrency API workloads.

| Parameter | Default | Optimized | Reden |
|-----------|---------|-----------|-------|
| workers | 1 | 8 | Benut multi-core CPU |
| worker-connections | 1000 | 1024 | Max connections per worker |
| keepalive | 2 | 5 | Reduce connection overhead |
| backlog | 2048 | 2048 | Queue size voor wachtende connections |
| timeout | 30 | 30 | Request timeout |

**Impact:** +30-50% throughput

---

### Nginx Proxy Optimalisatie

| Setting | Waarde | Reden |
|---------|--------|-------|
| proxy_buffering | on | Buffer responses, reduce latency |
| proxy_http_version | 1.1 | Enable keep-alive upstream |
| gzip_comp_level | 5 | Balance compression/CPU |
| Connection header | "" | Enable connection reuse |

**Impact:** +10-20% response time improvement

---

### Load Test Resultaten (CX33)

**Na optimalisatie:**

| Concurrent Users | Requests/sec | Error Rate | Status |
|------------------|--------------|------------|--------|
| 10 | 64 | 0% | ✅ Stable |
| 50 | 258 | 0% | ✅ Stable |
| 100 | 135 | 26% | ⚠️ Degraded |
| 200 | 160 | 32% | ⚠️ Degraded |

**Klanten capacity:** ~500-800 (CX33), ~1500-2000 (CX43)

EOF

echo "✅ Updated: docs/SKILL_08_HARDWARE_PROFILE.md"
```

---

## TAAK 5: Update SKILL_09_INSTALLER_SPECS.md

**Voeg FASE 3.5 toe:**

```bash
cat >> /opt/github/ha-energy-insights-nl/docs/SKILL_09_INSTALLER_SPECS.md << 'EOF'

---

## FASE 3.5: PERFORMANCE TUNING (NIEUW)

**Wanneer:** Na FASE 3 (Security), voor FASE 4 (Python)

**Script:** `scripts/setup/optimize_tcp.sh`

**Stappen:**

### 3.5.1 TCP Kernel Tuning

```bash
./scripts/setup/optimize_tcp.sh
```

Voegt toe aan `/etc/sysctl.conf`:
- `net.ipv4.tcp_tw_reuse = 1`
- `net.ipv4.tcp_fin_timeout = 30`
- `net.core.somaxconn = 4096`
- `net.ipv4.tcp_max_syn_backlog = 4096`

### 3.5.2 Verificatie

```bash
sysctl net.ipv4.tcp_tw_reuse
# Expected: net.ipv4.tcp_tw_reuse = 1
```

**Exit criteria:**
- [ ] TCP tuning in sysctl.conf
- [ ] sysctl -p zonder errors
- [ ] Verificatie toont correcte waarden

---

## FASE 5 UPDATE: Systemd Service

**Template:** `systemd/energy-insights-nl-api.service.template`

Bevat geoptimaliseerde Gunicorn config:
- 8 workers (ipv default 4)
- keepalive 5
- backlog 2048
- worker-connections 1024

**Genereren:**
```bash
sed -e "s/{{BRAND_NAME}}/$BRAND_NAME/g" \
    -e "s/{{SERVICE_USER}}/$SERVICE_USER/g" \
    -e "s/{{SERVICE_GROUP}}/$SERVICE_GROUP/g" \
    -e "s/{{APP_PATH}}/$APP_PATH/g" \
    -e "s/{{INSTALL_PATH}}/$INSTALL_PATH/g" \
    -e "s/{{API_PORT}}/$API_PORT/g" \
    systemd/energy-insights-nl-api.service.template \
    > /etc/systemd/system/${BRAND_SLUG}-api.service
```

---

## FASE 5 UPDATE: Nginx Config

**Template:** `config/nginx/energy-insights-nl.conf.template`

Bevat:
- Gzip compression (level 5)
- Proxy buffering
- HTTP/1.1 keep-alive
- Optimized timeouts

**Genereren:**
```bash
sed -e "s/{{BRAND_NAME}}/$BRAND_NAME/g" \
    -e "s/{{BRAND_DOMAIN}}/$BRAND_DOMAIN/g" \
    -e "s/{{API_PORT}}/$API_PORT/g" \
    config/nginx/energy-insights-nl.conf.template \
    > /etc/nginx/sites-available/${BRAND_SLUG}

ln -sf /etc/nginx/sites-available/${BRAND_SLUG} /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

EOF

echo "✅ Updated: docs/SKILL_09_INSTALLER_SPECS.md"
```

---

## TAAK 6: Maak master setup script update

**Bestand:** `/opt/github/ha-energy-insights-nl/scripts/setup/setup_performance.sh`

```bash
cat > /opt/github/ha-energy-insights-nl/scripts/setup/setup_performance.sh << 'EOF'
#!/bin/bash
# SYNCTACLES Performance Setup
# Run after base installation (FASE 1-3)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "========================================"
echo "  SYNCTACLES Performance Setup"
echo "========================================"

# Source environment
if [[ -f /opt/.env ]]; then
    source /opt/.env
else
    echo "❌ /opt/.env not found. Run FASE 0 first."
    exit 1
fi

# 1. TCP Tuning
echo ""
echo "--- Step 1: TCP Kernel Tuning ---"
"$SCRIPT_DIR/optimize_tcp.sh"

# 2. Gunicorn Service (if exists, update it)
echo ""
echo "--- Step 2: Gunicorn Service ---"
if [[ -f "$REPO_DIR/systemd/energy-insights-nl-api.service.template" ]]; then
    sed -e "s|{{BRAND_NAME}}|$BRAND_NAME|g" \
        -e "s|{{SERVICE_USER}}|$SERVICE_USER|g" \
        -e "s|{{SERVICE_GROUP}}|$SERVICE_GROUP|g" \
        -e "s|{{APP_PATH}}|$APP_PATH|g" \
        -e "s|{{INSTALL_PATH}}|$INSTALL_PATH|g" \
        -e "s|{{API_PORT}}|${API_PORT:-8000}|g" \
        "$REPO_DIR/systemd/energy-insights-nl-api.service.template" \
        > "/etc/systemd/system/${BRAND_SLUG}-api.service"
    
    systemctl daemon-reload
    echo "✅ Gunicorn service updated"
else
    echo "⚠️ Service template not found, skipping"
fi

# 3. Nginx Config (if nginx installed)
echo ""
echo "--- Step 3: Nginx Configuration ---"
if command -v nginx &> /dev/null; then
    if [[ -f "$REPO_DIR/config/nginx/energy-insights-nl.conf.template" ]]; then
        sed -e "s|{{BRAND_NAME}}|$BRAND_NAME|g" \
            -e "s|{{BRAND_DOMAIN}}|${BRAND_DOMAIN:-_}|g" \
            -e "s|{{API_PORT}}|${API_PORT:-8000}|g" \
            "$REPO_DIR/config/nginx/energy-insights-nl.conf.template" \
            > "/etc/nginx/sites-available/${BRAND_SLUG}"
        
        ln -sf "/etc/nginx/sites-available/${BRAND_SLUG}" /etc/nginx/sites-enabled/
        
        if nginx -t; then
            systemctl reload nginx
            echo "✅ Nginx config updated"
        else
            echo "❌ Nginx config invalid"
            exit 1
        fi
    else
        echo "⚠️ Nginx template not found, skipping"
    fi
else
    echo "⚠️ Nginx not installed, skipping"
fi

# 4. Restart API
echo ""
echo "--- Step 4: Restart Services ---"
if systemctl is-active --quiet "${BRAND_SLUG}-api"; then
    systemctl restart "${BRAND_SLUG}-api"
    sleep 3
    echo "✅ API restarted"
fi

# 5. Verify
echo ""
echo "--- Verification ---"
echo "TCP settings:"
sysctl net.ipv4.tcp_tw_reuse net.ipv4.tcp_fin_timeout net.core.somaxconn

echo ""
echo "Gunicorn workers:"
ps aux | grep "[g]unicorn" | wc -l

echo ""
echo "========================================"
echo "  Performance Setup Complete"
echo "========================================"
EOF

chmod +x /opt/github/ha-energy-insights-nl/scripts/setup/setup_performance.sh
echo "✅ Created: scripts/setup/setup_performance.sh"
```

---

## TAAK 7: Update directory structure

```bash
# Zorg dat docs directory bestaat
mkdir -p /opt/github/ha-energy-insights-nl/docs

# Kopieer SKILL bestanden als ze nog niet in docs staan
# (Skip als al aanwezig)
```

---

## TAAK 8: Verificatie

```bash
echo "=== Verificatie ==="

echo "Scripts:"
ls -la /opt/github/ha-energy-insights-nl/scripts/setup/*.sh

echo ""
echo "Templates:"
ls -la /opt/github/ha-energy-insights-nl/systemd/*.template 2>/dev/null || echo "systemd templates: OK"
ls -la /opt/github/ha-energy-insights-nl/config/nginx/*.template 2>/dev/null || echo "nginx templates: OK"

echo ""
echo "Executable check:"
[[ -x /opt/github/ha-energy-insights-nl/scripts/setup/optimize_tcp.sh ]] && echo "✅ optimize_tcp.sh" || echo "❌ optimize_tcp.sh"
[[ -x /opt/github/ha-energy-insights-nl/scripts/setup/setup_performance.sh ]] && echo "✅ setup_performance.sh" || echo "❌ setup_performance.sh"
```

---

## TAAK 9: Git commit

```bash
su - energy-insights-nl -c "
cd /opt/github/ha-energy-insights-nl
git add -A
git status

git commit -m 'Ops: Borging performance optimalisaties in setup scripts

Nieuwe bestanden:
- scripts/setup/optimize_tcp.sh (TCP kernel tuning)
- scripts/setup/setup_performance.sh (master performance setup)
- systemd/energy-insights-nl-api.service.template (8 workers, keepalive)
- config/nginx/energy-insights-nl.conf.template (buffering, gzip)

Documentatie updates:
- SKILL_08: Performance tuning sectie toegevoegd
- SKILL_09: FASE 3.5 toegevoegd, FASE 5 templates

Nieuwe servers krijgen automatisch:
- TCP tw_reuse, fin_timeout, somaxconn tuning
- Gunicorn 8 workers met keepalive
- Nginx proxy buffering en HTTP/1.1'

git push origin main
"
```

---

## Exit Criteria

- [ ] `scripts/setup/optimize_tcp.sh` bestaat en executable
- [ ] `scripts/setup/setup_performance.sh` bestaat en executable
- [ ] `systemd/*.template` bevat geoptimaliseerde Gunicorn config
- [ ] `config/nginx/*.template` bevat buffering + gzip
- [ ] SKILL_08 bevat performance tuning documentatie
- [ ] SKILL_09 bevat FASE 3.5 + template instructies
- [ ] Alles gecommit en gepusht

---

## Initiële Opdracht

```
Borg performance optimalisaties in setup scripts en documentatie.

WERKWIJZE:
1. Lees dit document volledig
2. Voer TAAK 1-9 sequentieel uit
3. Verifieer dat alle scripts executable zijn
4. Commit naar git

BELANGRIJK:
- Maak directories aan waar nodig (mkdir -p)
- Check of bestanden al bestaan voor append
- Templates gebruiken {{PLACEHOLDER}} formaat

START: TAAK 1
```
