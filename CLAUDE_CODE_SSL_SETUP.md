# Claude Code - SSL Setup

**Datum:** 2025-12-30
**Domein:** enin.xteleo.nl
**Server IP:** 135.181.255.83
**Doel:** HTTPS activeren via Let's Encrypt

---

## Vereiste

**DNS moet eerst ingesteld zijn:**
```
A-record: enin.xteleo.nl → 135.181.255.83
```

---

## TAAK 1: Verify DNS propagation

```bash
echo "=== DNS Verificatie ==="

# Check DNS resolution
RESOLVED_IP=$(dig +short enin.xteleo.nl)
EXPECTED_IP="135.181.255.83"

echo "Domain: enin.xteleo.nl"
echo "Expected IP: $EXPECTED_IP"
echo "Resolved IP: $RESOLVED_IP"

if [[ "$RESOLVED_IP" == "$EXPECTED_IP" ]]; then
    echo "✅ DNS correct geconfigureerd"
else
    echo "❌ DNS nog niet correct. Wacht op propagation of check DNS settings."
    echo "Huidige resolutie: $RESOLVED_IP"
    exit 1
fi
```

---

## TAAK 2: Update Nginx config met domein

```bash
echo "=== Nginx Config Update ==="

# Backup huidige config
cp /etc/nginx/sites-available/energy-insights-nl /tmp/nginx_pre_ssl_backup.conf

# Update config met correct domein
cat > /etc/nginx/sites-available/energy-insights-nl << 'EOF'
server {
    listen 80;
    server_name enin.xteleo.nl;

    # Gzip compression
    gzip on;
    gzip_types application/json text/plain application/javascript text/css;
    gzip_min_length 256;
    gzip_comp_level 5;

    # Proxy buffering
    proxy_buffering on;
    proxy_buffer_size 4k;
    proxy_buffers 8 16k;
    proxy_busy_buffers_size 24k;

    # HTTP/1.1 keep-alive
    proxy_http_version 1.1;
    proxy_set_header Connection "";

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
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

# Test en reload
nginx -t && systemctl reload nginx
echo "✅ Nginx config updated voor enin.xteleo.nl"
```

---

## TAAK 3: Test HTTP toegang extern

```bash
echo "=== HTTP Test ==="

# Test via domein
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://enin.xteleo.nl/health)

if [[ "$HTTP_STATUS" == "200" ]]; then
    echo "✅ HTTP werkt: http://enin.xteleo.nl/health (HTTP $HTTP_STATUS)"
else
    echo "❌ HTTP failed (HTTP $HTTP_STATUS)"
    echo "Check firewall: ufw status"
    echo "Check nginx: systemctl status nginx"
    exit 1
fi
```

---

## TAAK 4: Request SSL certificaat

```bash
echo "=== Let's Encrypt SSL ==="

# Certbot met nginx plugin
certbot --nginx -d enin.xteleo.nl --non-interactive --agree-tos --email admin@xteleo.nl --redirect

echo "✅ SSL certificaat aangevraagd"
```

**Opmerking:** Vervang `admin@xteleo.nl` door een geldig e-mailadres als dit niet klopt.

---

## TAAK 5: Verify SSL configuratie

```bash
echo "=== SSL Verificatie ==="

# Check HTTPS
HTTPS_STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://enin.xteleo.nl/health)

if [[ "$HTTPS_STATUS" == "200" ]]; then
    echo "✅ HTTPS werkt: https://enin.xteleo.nl/health"
else
    echo "❌ HTTPS failed (HTTP $HTTPS_STATUS)"
    exit 1
fi

# Check certificaat details
echo ""
echo "Certificaat info:"
echo | openssl s_client -servername enin.xteleo.nl -connect enin.xteleo.nl:443 2>/dev/null | openssl x509 -noout -dates -subject

# Check HTTP redirect naar HTTPS
echo ""
echo "HTTP redirect test:"
curl -sI http://enin.xteleo.nl | head -3
```

---

## TAAK 6: Test alle endpoints via HTTPS

```bash
echo "=== Endpoint Tests (HTTPS) ==="

BASE_URL="https://enin.xteleo.nl"

for endpoint in /health /api/v1/generation-mix /api/v1/load /api/v1/balance /api/v1/prices /api/v1/signals; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL$endpoint")
    if [[ "$STATUS" == "200" ]]; then
        echo "✅ $endpoint (HTTP $STATUS)"
    else
        echo "❌ $endpoint (HTTP $STATUS)"
    fi
done
```

---

## TAAK 7: Check auto-renewal

```bash
echo "=== Auto-Renewal Check ==="

# Certbot timer status
systemctl status certbot.timer --no-pager

# Test renewal (dry-run)
certbot renew --dry-run

echo ""
echo "✅ Auto-renewal geconfigureerd"
```

---

## TAAK 8: Update .env met domein

```bash
echo "=== Update .env ==="

# Backup
cp /opt/.env /opt/.env.backup_pre_ssl

# Update BRAND_DOMAIN als deze anders is
if grep -q "BRAND_DOMAIN=" /opt/.env; then
    sed -i 's|BRAND_DOMAIN=.*|BRAND_DOMAIN=enin.xteleo.nl|' /opt/.env
    echo "✅ BRAND_DOMAIN updated"
else
    echo "BRAND_DOMAIN=enin.xteleo.nl" >> /opt/.env
    echo "✅ BRAND_DOMAIN added"
fi

# Show current value
grep BRAND_DOMAIN /opt/.env
```

---

## TAAK 9: Firewall check

```bash
echo "=== Firewall Status ==="

ufw status

# Zorg dat HTTPS open is
ufw allow 443/tcp
ufw status | grep -E "80|443"

echo "✅ Firewall configured"
```

---

## TAAK 10: Final summary

```bash
echo "========================================"
echo "  SSL SETUP COMPLETE"
echo "========================================"
echo ""
echo "Domain:     enin.xteleo.nl"
echo "HTTP:       http://enin.xteleo.nl → redirects to HTTPS"
echo "HTTPS:      https://enin.xteleo.nl ✅"
echo ""
echo "Endpoints:"
echo "  https://enin.xteleo.nl/health"
echo "  https://enin.xteleo.nl/api/v1/generation-mix"
echo "  https://enin.xteleo.nl/api/v1/load"
echo "  https://enin.xteleo.nl/api/v1/balance"
echo "  https://enin.xteleo.nl/api/v1/prices"
echo "  https://enin.xteleo.nl/api/v1/signals"
echo ""
echo "Certificate:"
echo "  Issuer: Let's Encrypt"
echo "  Auto-renewal: Active (certbot.timer)"
echo ""
echo "========================================"
```

---

## Exit Criteria

- [ ] DNS resolves naar 135.181.255.83
- [ ] HTTP werkt op poort 80
- [ ] SSL certificaat geïnstalleerd
- [ ] HTTPS werkt op poort 443
- [ ] HTTP redirect naar HTTPS actief
- [ ] Alle 6 endpoints bereikbaar via HTTPS
- [ ] Auto-renewal geconfigureerd
- [ ] .env updated met BRAND_DOMAIN
- [ ] Firewall poort 443 open

---

## Troubleshooting

**DNS niet resolved:**
```bash
# Check met verschillende DNS servers
dig enin.xteleo.nl @8.8.8.8
dig enin.xteleo.nl @1.1.1.1
# Wacht 5-30 min voor propagation
```

**Certbot failed:**
```bash
# Check logs
journalctl -u certbot -n 50

# Common issues:
# - DNS niet gepropageerd
# - Poort 80 niet bereikbaar
# - Rate limit (max 5 certs per week per domein)
```

**HTTPS niet bereikbaar:**
```bash
# Check firewall
ufw status
iptables -L -n | grep 443

# Check nginx
nginx -t
systemctl status nginx
```

---

## Initiële Opdracht

```
Configureer SSL voor enin.xteleo.nl via Let's Encrypt.

WERKWIJZE:
1. Lees dit document
2. Voer TAAK 1-10 sequentieel uit
3. Bij DNS error in TAAK 1: STOP en meld aan gebruiker
4. Bij certbot error: check troubleshooting sectie

VEREISTE:
DNS A-record moet al ingesteld zijn:
enin.xteleo.nl → 135.181.255.83

START: TAAK 1 (DNS verificatie)
```
