# SKILL_08_HARDWARE_PROFILE

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
