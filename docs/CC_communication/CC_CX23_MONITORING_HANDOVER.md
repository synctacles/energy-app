# CC Handover: CX23 Monitoring Server Setup

**Datum:** 2026-01-07
**Status:** Volledig operationeel

---

## Wat is CX23?

Hetzner Cloud server (77.42.41.135) voor centralized monitoring van Energy Insights NL.

**Toegang:** https://monitor.synctacles.com

---

## Geïnstalleerde Stack

```
CX23 (77.42.41.135)
├── Docker
│   ├── Prometheus (metrics, :9090 intern)
│   ├── Grafana (dashboards, :3000 intern)
│   ├── AlertManager (Slack alerts, :9093 intern)
│   └── Blackbox (HTTP/SSL probes, :9115 intern)
├── Caddy (reverse proxy, :80/:443)
├── fail2ban (SSH protection)
└── unattended-upgrades (auto security updates)
```

---

## Configuratie Locaties

**Op CX23:**
```
/opt/monitoring/
├── docker-compose.yml
├── prometheus/prometheus.yml    # Scrape targets
├── prometheus/alerts.yml        # 22 alert rules
├── alertmanager/alertmanager.yml # Slack webhooks
├── blackbox/blackbox.yml
└── grafana/dashboards/*.json    # 3 dashboards

/etc/caddy/Caddyfile              # HTTPS reverse proxy
```

---

## Wat Wordt Gemonitord

| Target | Metrics |
|--------|---------|
| API server (135.181.255.83:9100) | CPU, memory, disk, systemd services |
| https://enin.xteleo.nl/health | HTTP uptime |
| enin.xteleo.nl:443 | SSL certificate expiry |

**Services gemonitord via node_exporter:**
- energy-insights-nl-* (actief)
- synctacles-* (deprecated)

---

## Alerting

Slack webhooks naar:
- `#enin-alerts-critical` - Server down, disk >90%, service failures
- `#enin-alerts-warnings` - High CPU/memory, SSL expiry <14 dagen
- `#enin-alerts-info` - Informatief

---

## Security (Score: 9/10)

| Maatregel | Status |
|-----------|--------|
| HTTPS (Let's Encrypt) | ✅ Auto-renewal via Caddy |
| SSH | ✅ Key-only (monitoring user) |
| Root login | ✅ prohibit-password |
| fail2ban | ✅ 24h ban na 3 pogingen |
| Hetzner firewall | ✅ Alleen 22, 80, 443 open |
| Prometheus/AlertManager | ✅ Niet publiek (firewall) |
| Auto updates | ✅ unattended-upgrades |

---

## SSH Toegang

```bash
# Primair
ssh monitoring@77.42.41.135

# Nood
ssh root@77.42.41.135
```

**Geautoriseerde keys:**
- Windows werkstation (ftso@coston2)
- API server (root@ENIN-NL)

---

## Veelgebruikte Commands

```bash
# Containers beheren
cd /opt/monitoring
sudo docker compose restart
sudo docker logs <container> --tail 50

# Caddy
sudo systemctl restart caddy
sudo journalctl -u caddy -f

# fail2ban
sudo fail2ban-client status sshd
```

---

## Documentatie

- [MONITORING_SETUP.md](../operations/MONITORING_SETUP.md) - Volledige setup guide
- [CX23_MONITORING_SERVER.md](../operations/CX23_MONITORING_SERVER.md) - Server details
- [monitoring/README.md](../../monitoring/README.md) - Stack overview

---

## Belangrijk voor CC

1. **SSH naar CX23** werkt alleen via `monitoring` user of `root` (key-only)
2. **Prometheus/AlertManager** zijn NIET publiek bereikbaar (firewall)
3. **Grafana** alleen via https://monitor.synctacles.com
4. **SSL certificaat** wordt automatisch vernieuwd door Caddy
5. **Config wijzigingen** op CX23 zijn NIET in git (alleen templates in repo)

---

**Laatste update:** 2026-01-07
