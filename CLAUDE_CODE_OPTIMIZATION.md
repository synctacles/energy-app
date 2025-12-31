# Claude Code - Performance Optimalisaties

**Datum:** 2025-12-30
**Server:** 135.181.255.83
**Doel:** Zero-risk optimalisaties + hertest

---

## Context

Load test toonde bottleneck bij 30-40 concurrent users door:
- TIME_WAIT socket accumulation
- Suboptimale Gunicorn config
- Nginx buffering niet geoptimaliseerd

**Baseline (vóór optimalisatie):**
- Max concurrent: ~30-40
- Requests/sec: ~65-70
- First errors: 50 concurrent
- p95 latency: 253.9ms

---

## TAAK 1: TCP Kernel Tuning

```bash
echo "=== TCP Kernel Tuning ==="

# Backup huidige settings
sysctl net.ipv4.tcp_tw_reuse net.ipv4.tcp_fin_timeout net.core.somaxconn > /tmp/sysctl_backup.txt

# Apply optimalisaties
cat >> /etc/sysctl.conf << 'EOF'

# SYNCTACLES Performance Tuning (2025-12-30)
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 4096
EOF

sysctl -p

echo "✅ TCP tuning applied"
sysctl net.ipv4.tcp_tw_reuse net.ipv4.tcp_fin_timeout net.core.somaxconn
```

---

## TAAK 2: Gunicorn Optimalisatie

```bash
echo "=== Gunicorn Optimalisatie ==="

# Backup huidige service
cp /etc/systemd/system/energy-insights-nl-api.service /tmp/api_service_backup.service

# Check huidige config
grep -E "ExecStart|workers" /etc/systemd/system/energy-insights-nl-api.service

# Update service met geoptimaliseerde settings
cat > /etc/systemd/system/energy-insights-nl-api.service << 'EOF'
[Unit]
Description=Energy Insights NL API
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=energy-insights-nl
Group=energy-insights-nl
WorkingDirectory=/opt/energy-insights-nl/app
Environment="PATH=/opt/energy-insights-nl/venv/bin"
EnvironmentFile=/opt/.env
ExecStart=/opt/energy-insights-nl/venv/bin/gunicorn \
    --bind 127.0.0.1:8000 \
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

systemctl daemon-reload
systemctl restart energy-insights-nl-api
sleep 3

# Verify
systemctl is-active energy-insights-nl-api && echo "✅ API running with 8 workers" || echo "❌ API failed"
ps aux | grep gunicorn | grep -v grep | wc -l
```

---

## TAAK 3: Nginx Proxy Buffering

```bash
echo "=== Nginx Proxy Buffering ==="

# Backup
cp /etc/nginx/sites-available/energy-insights-nl /tmp/nginx_backup.conf

# Update nginx config
cat > /etc/nginx/sites-available/energy-insights-nl << 'EOF'
server {
    listen 80;
    server_name _;

    # Gzip compression
    gzip on;
    gzip_types application/json text/plain;
    gzip_min_length 256;
    gzip_comp_level 5;

    # Proxy buffering
    proxy_buffering on;
    proxy_buffer_size 4k;
    proxy_buffers 8 16k;
    proxy_busy_buffers_size 24k;

    # Connection optimizations
    proxy_http_version 1.1;
    proxy_set_header Connection "";

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 10s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    location /health {
        proxy_pass http://127.0.0.1:8000/health;
        access_log off;
    }
}
EOF

nginx -t && systemctl reload nginx
echo "✅ Nginx optimized"
```

---

## TAAK 4: Verify Services

```bash
echo "=== Service Verification ==="

# Check all services
systemctl is-active energy-insights-nl-api && echo "✅ API" || echo "❌ API"
systemctl is-active nginx && echo "✅ Nginx" || echo "❌ Nginx"
systemctl is-active postgresql && echo "✅ PostgreSQL" || echo "❌ PostgreSQL"

# Check workers
echo ""
echo "Gunicorn workers:"
ps aux | grep "[g]unicorn" | wc -l

# Check endpoints
echo ""
echo "Endpoint check:"
curl -s -o /dev/null -w "%{http_code}" http://localhost/health && echo " /health OK"
curl -s -o /dev/null -w "%{http_code}" http://localhost/api/v1/generation-mix && echo " /generation-mix OK"

# Check TCP settings
echo ""
echo "TCP settings:"
sysctl net.ipv4.tcp_tw_reuse net.ipv4.tcp_fin_timeout net.core.somaxconn
```

---

## TAAK 5: Hertest - Baseline (10 concurrent)

```bash
echo "=== HERTEST: Baseline (10 users, 30s) ==="
echo "Waiting 5 seconds for services to stabilize..."
sleep 5

hey -n 1000 -c 10 -z 30s http://localhost/api/v1/generation-mix
```

---

## TAAK 6: Hertest - Moderate (50 concurrent)

```bash
echo "=== HERTEST: Moderate (50 users, 60s) ==="
sleep 5

hey -n 5000 -c 50 -z 60s http://localhost/api/v1/generation-mix
```

---

## TAAK 7: Hertest - Stress (100 concurrent)

```bash
echo "=== HERTEST: Stress (100 users, 60s) ==="
sleep 5

hey -n 10000 -c 100 -z 60s http://localhost/api/v1/generation-mix
```

---

## TAAK 8: Hertest - Extreme (200 concurrent)

```bash
echo "=== HERTEST: Extreme (200 users, 30s) ==="
sleep 5

hey -n 10000 -c 200 -z 30s http://localhost/api/v1/generation-mix
```

---

## TAAK 9: Resource Check During Load

```bash
echo "=== Resource Status ==="

# TCP connections
echo "TCP connections:"
ss -s | grep -E "TCP|estab|time-wait"

# Memory
echo ""
echo "Memory:"
free -h

# CPU load
echo ""
echo "Load average:"
uptime

# Gunicorn workers
echo ""
echo "Gunicorn workers:"
ps aux | grep "[g]unicorn" --color=never | awk '{print $2, $3"%CPU", $4"%MEM", $11}'
```

---

## TAAK 10: Genereer Vergelijkingsrapport

```bash
cat > /opt/github/ha-energy-insights-nl/OPTIMIZATION_RESULTS.md << 'EOF'
# Performance Optimization Results

**Datum:** $(date +%Y-%m-%d)
**Server:** Hetzner CX33 (4 vCPU, 8GB RAM)

---

## Wijzigingen

1. **TCP Tuning**
   - tcp_tw_reuse: 0 → 1
   - tcp_fin_timeout: 60 → 30
   - somaxconn: 4096
   - tcp_max_syn_backlog: 4096

2. **Gunicorn**
   - workers: 4 → 8
   - worker-connections: default → 1024
   - keepalive: default → 5
   - backlog: default → 2048

3. **Nginx**
   - proxy_buffering: on
   - proxy_http_version: 1.1
   - Connection: "" (keep-alive)

---

## Resultaten Vergelijking

### Baseline (10 concurrent)

| Metric | VOOR | NA | Verbetering |
|--------|------|-----|-------------|
| Requests/sec | 63.87 | XXX | XXX% |
| Latency p50 | 135.6 ms | XXX | XXX% |
| Latency p95 | 253.9 ms | XXX | XXX% |
| Error rate | 0% | XXX | - |

### Moderate (50 concurrent)

| Metric | VOOR | NA | Verbetering |
|--------|------|-----|-------------|
| Requests/sec | 66.43 | XXX | XXX% |
| Latency p95 | 266.8 ms | XXX | XXX% |
| Error rate | 2.9% | XXX | XXX |
| Timeouts | 115 | XXX | XXX |

### Stress (100 concurrent)

| Metric | VOOR | NA | Verbetering |
|--------|------|-----|-------------|
| Requests/sec | 5.0 | XXX | XXX% |
| Error rate | 100% | XXX | XXX |

### Extreme (200 concurrent)

| Metric | VOOR | NA | Verbetering |
|--------|------|-----|-------------|
| Requests/sec | N/A | XXX | - |
| Error rate | N/A | XXX | - |

---

## Nieuwe Limieten

| Metric | VOOR | NA |
|--------|------|-----|
| Max concurrent users | 30-40 | XXX |
| Max requests/sec | 65-70 | XXX |
| First errors at | 50 | XXX |
| Complete failure at | 100 | XXX |

---

## Conclusie

**Verbetering behaald:** XXX%
**Nieuwe klanten ceiling:** XXX
**Production ready:** ✅/❌

EOF

echo "✅ Template created: OPTIMIZATION_RESULTS.md"
echo "Vul in met bovenstaande testresultaten"
```

---

## TAAK 11: Git Commit

```bash
su - energy-insights-nl -c "
cd /opt/github/ha-energy-insights-nl
git add -A
git commit -m 'Perf: TCP tuning, Gunicorn 8 workers, Nginx buffering

Optimalisaties:
- TCP: tw_reuse=1, fin_timeout=30, somaxconn=4096
- Gunicorn: 8 workers, keepalive=5, backlog=2048
- Nginx: proxy buffering, HTTP/1.1 keep-alive

Load test resultaten: zie OPTIMIZATION_RESULTS.md'
git push origin main
"
```

---

## Exit Criteria

- [ ] TCP tuning actief (sysctl -p zonder errors)
- [ ] 8 Gunicorn workers draaien
- [ ] Nginx config valid (nginx -t)
- [ ] Alle endpoints responding
- [ ] Hertest voltooid (4 load levels)
- [ ] OPTIMIZATION_RESULTS.md ingevuld
- [ ] Committed naar git

---

## Initiële Opdracht

```
Voer performance optimalisaties uit en hertest.

WERKWIJZE:
1. Lees dit document
2. Voer TAAK 1-11 sequentieel uit
3. Vul OPTIMIZATION_RESULTS.md in met echte waarden
4. Vergelijk VOOR/NA metrics

BELANGRIJK:
- Backup wordt automatisch gemaakt per taak
- Bij service failure: check logs met journalctl -u energy-insights-nl-api -n 50
- Wacht 5 sec tussen load tests

START: TAAK 1
```
