# SKILL_09_INSTALLER_SPECS

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
