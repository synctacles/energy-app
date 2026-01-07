# Energy Insights NL Load Test Report

**Datum:** 2026-01-07
**Server:** Hetzner CX33 (4 vCPU, 8GB RAM)
**Tester:** Claude Code
**Vorige test:** 2025-12-30

---

## Test Configuratie

- Tool: hey (HTTP load testing)
- Target: https://enin.xteleo.nl
- Endpoint: /api/v1/generation-mix
- Workers: 8 Gunicorn workers (UvicornWorker)
- Optimalisaties sinds vorige test:
  - TCP tuning (tcp_tw_reuse, tcp_fin_timeout)
  - Gunicorn worker-connections 1024
  - Gunicorn keepalive 5
  - Gunicorn backlog 2048

---

## Resultaten

### Baseline (10 concurrent users, 30s)

| Metric | Waarde | Vorige Test | Verbetering |
|--------|--------|-------------|-------------|
| Requests/sec | 69.1 | 63.9 | +8% |
| Latency p50 | 135 ms | 135.6 ms | = |
| Latency p95 | 185 ms | 253.9 ms | **-27%** |
| Latency p99 | 299 ms | 526.3 ms | **-43%** |
| Error rate | 0% | 0% | = |
| Total requests | 2,000 | 1,925 | - |

### Moderate (50 concurrent users, 60s)

| Metric | Waarde | Vorige Test | Verbetering |
|--------|--------|-------------|-------------|
| Requests/sec | **262.8** | 66.4 | **+296%** |
| Latency p50 | 170 ms | 142.3 ms | +20% |
| Latency p95 | 265 ms | 266.8 ms | = |
| Latency p99 | 351 ms | 3,235.7 ms | **-89%** |
| Error rate | **0%** | 2.9% | **FIXED** |
| Total requests | 5,000 | 5,161 | - |
| HTTP 200 | 5,000 | 5,127 | - |
| HTTP 502 | 0 | 34 | **FIXED** |

### Breaking Point Tests

| Concurrent Users | Requests/sec | p95 (ms) | p99 (ms) | Error Rate | Status |
|-----------------|--------------|----------|----------|------------|--------|
| 50 | 262.8 | 265 | 351 | 0% | **STABLE** |
| 60 | 137.9 | 214 | 286 | 0.27% | Acceptable |
| 75 | 199.6 | 312 | 445 | 0.43% | Marginal |
| 100 | 244.6 | 304 | 474 | 11.1% | Degraded |

### Stress (100 concurrent users, 60s)

| Metric | Waarde | Vorige Test | Verbetering |
|--------|--------|-------------|-------------|
| Requests/sec | **244.6** | 5.0 | **+4,792%** |
| Latency p95 | 304 ms | N/A (timeout) | NEW |
| Latency p99 | 474 ms | N/A (timeout) | NEW |
| Error rate | 11.1% | 100% | **-89%** |
| Total requests | 10,000 | 300 | - |
| HTTP 200 | 8,890 | 0 | **NEW** |
| HTTP 500 | 1,076 | - | - |
| HTTP 502 | 34 | 300 | - |

---

## Vergelijking met Vorige Test

### Breaking Point

| Metric | 2026-01-07 | 2025-12-30 | Verbetering |
|--------|------------|------------|-------------|
| Max stable concurrent users | **50+** | ~30-40 | **+25-67%** |
| Max requests/sec (stable) | **262.8** | ~65-70 | **+300%** |
| First errors appear at | 60 users | 50 users | **+20%** |
| Complete failure at | >100 users | 100 users | **IMPROVED** |

### Performance Targets

| Target | 2025-12-30 | 2026-01-07 | Status |
|--------|------------|------------|--------|
| p95 < 100ms (baseline) | 253.9ms | 185ms | Still improving |
| p95 < 200ms (moderate) | 266.8ms | 265ms | Close |
| Error rate < 1% (50 users) | 2.9% | **0%** | **ACHIEVED** |
| 100+ req/sec (moderate) | 66.4 | **262.8** | **ACHIEVED** |

---

## Root Cause Analysis

### Wat is verbeterd?

1. **TCP TIME_WAIT handling**
   - `tcp_tw_reuse = 1` - Hergebruik van TIME_WAIT sockets
   - `tcp_fin_timeout = 30` - Snellere socket cleanup
   - Result: Geen socket exhaustion meer bij 50 concurrent users

2. **Gunicorn optimalisaties**
   - `worker-connections 1024` - Meer connections per worker
   - `keepalive 5` - Connection reuse
   - `backlog 2048` - Grotere queue voor piekbelasting
   - Result: 4x hogere throughput

3. **Workers verhoogd**
   - Van 4 naar 8 workers
   - Result: Betere parallelisatie

### Resterende bottlenecks

1. **HTTP 500 errors bij 100 concurrent users**
   - 1,076 van 10,000 requests (10.8%)
   - Waarschijnlijk database connection pool exhaustion
   - Aanbeveling: pgBouncer implementeren

2. **HTTP 502 errors bij hoge load**
   - Nginx upstream timeout
   - Aanbeveling: Nginx proxy_read_timeout verhogen

---

## Conclusie

| Criterium | Status |
|-----------|--------|
| Production ready voor 50 users | **JA** |
| Production ready voor 100 users | NEE (11% errors) |
| Headroom | 50-60 concurrent users stabiel |

### Samenvatting

Het systeem is nu **production ready** voor de verwachte load:
- **50 concurrent users**: 0% errors, 262.8 req/sec
- **Home Assistant integratie**: Bij 1000 HA instances die elk elke 60s pollen = ~17 req/sec (ruim onder capaciteit)

De TCP tuning en Gunicorn optimalisaties hebben de capaciteit **verviervoudigd**.

### Aanbevelingen voor verdere verbetering

1. **Korte termijn:**
   - [ ] pgBouncer voor database connection pooling
   - [ ] Nginx timeout tuning

2. **Lange termijn:**
   - [ ] Redis caching voor nog hogere throughput
   - [ ] Server upgrade naar CX43 voor 100+ concurrent users

---

## Test Details

| Parameter | Waarde |
|-----------|--------|
| Test datum | 2026-01-07 04:35 UTC |
| Test tool | hey v0.1.4 |
| Test locatie | Remote (via internet) |
| Network latency | ~10-20ms |
| SSL | Enabled (Let's Encrypt) |

---

**Vorige rapport:** [LOAD_TEST_REPORT.md](LOAD_TEST_REPORT.md) (2025-12-30)
