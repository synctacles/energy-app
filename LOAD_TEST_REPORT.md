# SYNCTACLES Load Test Report

**Datum:** 2025-12-30
**Server:** Hetzner CX33 (4 vCPU, 8GB RAM)
**Tester:** Claude Code

---

## Test Configuratie

- Tool: hey + wrk
- Target: http://localhost (via nginx)
- Endpoints: /api/v1/generation-mix (primary)
- Caching: TTL 300s (generation)
- Workers: 4 Gunicorn workers (UvicornWorker)

---

## Health Check Resultaten

Pre-test status:
- ✅ API running (systemctl energy-insights-nl-api: active)
- ✅ PostgreSQL running (active)
- ✅ Nginx running (active)
- ✅ /health: HTTP 200
- ✅ /api/v1/generation-mix: HTTP 200
- ✅ /api/v1/load: HTTP 200
- ✅ /api/v1/balance: HTTP 200

Baseline single request timing:
- Request 1: 0.356056s
- Request 2: 0.252694s

---

## Resultaten

### Baseline (10 concurrent users, 30s)

| Metric | Waarde | Target | Status |
|--------|--------|--------|--------|
| Requests/sec | 63.87 | 50+ | ✅ |
| Latency p50 | 135.6 ms | < 20ms | ⚠️ |
| Latency p95 | 253.9 ms | < 50ms | ❌ |
| Latency p99 | 526.3 ms | < 100ms | ❌ |
| Error rate | 0% | < 0.1% | ✅ |
| Total requests | 1925 | - | - |
| Slowest request | 978.2 ms | - | - |

### Moderate (50 concurrent users, 60s)

| Metric | Waarde | Target | Status |
|--------|--------|--------|--------|
| Requests/sec | 66.43 | 100+ | ❌ |
| Latency p50 | 142.3 ms | < 30ms | ❌ |
| Latency p95 | 266.8 ms | < 100ms | ❌ |
| Latency p99 | 3235.7 ms | < 200ms | ❌ |
| Error rate | 2.9% | < 1% | ❌ |
| Total requests | 5161 | - | - |
| HTTP 200 responses | 5127 | - | - |
| HTTP 502 errors | 34 | - | - |
| Timeout errors | 115 | - | - |
| Slowest request | 19659.2 ms | - | - |

**Observatie:** Server begint al fouten te veroorzaken bij 50 concurrent users.

### Stress (100 concurrent users, 60s)

| Metric | Waarde | Target | Status |
|--------|--------|--------|--------|
| Requests/sec | 5.0 | 100+ | ❌ |
| Latency p95 | N/A | < 200ms | ❌ |
| Error rate | 100% | < 5% | ❌ |
| Total requests attempted | 300 | - | - |
| Timeout errors | 300 | - | - |

**Observatie:** Server is volledig overbelast, vrijwel alle requests timeout.

### Breaking Point

| Metric | Waarde |
|--------|--------|
| Max concurrent users | ~30-40 |
| Max requests/sec | ~65-70 |
| First errors appear at | 50 concurrent users |
| Complete failure at | 100 concurrent users |

---

## Cache Effectiveness

| Scenario | Avg Response Time |
|----------|-------------------|
| Cold cache | 307.6 ms |
| Warm cache | 268.1 ms |
| Improvement | 12.8% |

**Observatie:** Cache effectiviteit is zeer laag (verwacht 2-10x verbetering). Dit suggereert dat:
1. Cache-hits zijn niet optimaal geconfigureerd
2. TTL van 300s kan onvoldoende zijn
3. Requests hebben mogelijk variabele parameters die cache-hits voorkomen

---

## Database Performance

Query performance (zeer goed):

| Query | Response Time |
|-------|----------------|
| norm_entso_e_a75 (generation) | 6.24 ms |
| norm_entso_e_a65 (load) | 0.73 ms |
| norm_tennet_balance (balance) | 0.84 ms |

Table sizes:
- raw_tennet_balance: 56 MB (293,020 rows)
- norm_tennet_balance: 25 MB (58,660 rows)
- raw_entso_e_a75: 2.1 MB (8,911 rows)
- raw_entso_e_a65: 1.4 MB (2,126 rows)
- norm_entso_e_a75: 776 KB (1,062 rows)
- norm_entso_e_a65: 592 KB (1,212 rows)

**Conclusie:** Database is niet het bottleneck (queries zijn snel).

---

## Resource Usage (Peak)

Gemeten tijdens 50 concurrent users test:

| Resource | Usage |
|----------|-------|
| CPU active | 6.5% (low) |
| Memory used | 3990.8 MB / 7751.2 MB (51.5%) |
| Memory available | 3760.4 MB |
| Swap used | 0 MB |
| DB connections | 888 TCP (estab: 228, closed: 647, time_wait: 641) |
| Load average | 0.55 |

**Kritieke observatie:** 641 connections in TIME_WAIT state! Dit is waarschijnlijk het bottleneck.

Gunicorn worker status:
- 4 workers actief
- PID 21963-22020
- Geheugengebruik per worker: ~93 MB
- CPU per worker: ~6%

---

## Bottlenecks Identified

1. ✅ **TIME_WAIT socket connections** (PRIMAIR BOTTLENECK)
   - 641 connections in time_wait state bij 50 concurrent users
   - Dit beperkt het aantal nieuwe verbindingen dat kan worden aanvaard
   - Linux default TIME_WAIT duration: 60 seconden

2. ✅ **Insufficient connection pooling**
   - Gunicorn/Uvicorn connection handling niet geoptimaliseerd voor high concurrency
   - Mogelijk onvoldoende connection pool size voor database

3. ✅ **Cache effectiviteit zeer laag**
   - Slechts 12.8% verbetering van cold naar warm cache
   - Cache hits worden niet volledig benut

4. ⚠️ **API response latency hoog**
   - Baseline p95: 253.9ms (target: <50ms)
   - Moderate p95: 266.8ms (target: <100ms)
   - Dit wijst op suboptimale request processing

5. ⚠️ **Worker thread efficiency**
   - 4 workers kunnen slechts ~65-70 requests/sec verwerken
   - Dit is laag voor moderne Python-applicaties

---

## Recommendations

### Kritiek (implementeren voor productie)

1. **Tune TCP TIME_WAIT settings**
   ```bash
   # Voeg toe aan /etc/sysctl.conf
   net.ipv4.tcp_tw_reuse = 1
   net.ipv4.tcp_fin_timeout = 30
   net.ipv4.tcp_max_tw_buckets = 8000
   sysctl -p
   ```
   - Verwachte verbetering: +50-100% concurrent users

2. **Optimize Gunicorn configuration**
   ```bash
   # Increase worker connections
   --worker-connections 1024
   --keepalive 5
   ```
   - Verwachte verbetering: +20-30% throughput

3. **Enable response caching**
   - Verhoog TTL naar 600s (10 minuten) voor /api/v1/generation-mix
   - Implementeer cache busting op data updates
   - Verwachte verbetering: 2-5x throughput

4. **Connection pooling database**
   - Implementeer PgBouncer of Connection Pool in FastAPI
   - Max pool size: 20-30 connections
   - Verwachte verbetering: Stabieler onder load

### Belangrijk (medium-term improvements)

5. **Upgrade server spec**
   - Hetzner CX43 (8 vCPU, 16GB RAM) minimaal
   - Huidige CX33 is undersized voor 100+ concurrent users

6. **Implement request compression**
   - Zorg ervoor gzip is ingeschakeld in Nginx
   - Response size: 668 bytes per request kan 50% worden gereduceerd

7. **API performance profiling**
   - Gebruik Python profiler (py-spy, cProfile)
   - Identificeer slow code paths in handlers

### Nice-to-have (optimisaties)

8. **Implement CDN/edge caching**
   - CloudFlare of Bunny CDN voor statische responses
   - Minder belasting op origin server

9. **Load balancing**
   - Meerdere API servers achter load balancer
   - Distribueer 100+ concurrent users over meerdere servers

10. **Async database access**
    - Zorg ervoor alle database queries zijn async
    - Prevent blocking event loop

---

## Root Cause Analysis

### Waarom faalt het systeem bij 50+ concurrent users?

**Primair:** TIME_WAIT socket accumulation
- Bij elk verbinding sluiten gaat Linux socket in TIME_WAIT voor 60 seconden
- Bij 50 concurrent users worden veel verbindingen gemaakt/gesloten
- Na ~650 TIME_WAIT sockets kunnen geen nieuwe connections meer worden aanvaard
- Dit leidt tot 502 Bad Gateway errors van Nginx

**Secundair:** Suboptimale cache implementatie
- Cache geeft slechts 12.8% verbetering
- De meeste requests gaan naar database
- Database is snel (6ms), maar met 50 concurrent users accumuleert dit

**Tertiair:** Worker saturation
- 4 workers kunnen slechts 65-70 requests/sec verwerken
- Dit is ~16-18 requests per worker per seconde
- Gelijk aan ~55-65ms per request gemiddeld
- Dit klopt met gemeten p50 latency van 135.6ms

---

## Conclusie

**Production ready:** ❌ **NEE**
**Max safe concurrent users:** ~30-40
**Headroom:** Zeer beperkt (0-10 users tot critical errors)

### Samenvatting

Het systeem kan op dit moment **NIET in productie gaan** voor 100+ concurrent users. De server faalt al bij 50 concurrent users vanwege:

1. **TCP TIME_WAIT socket accumulation** (primaire oorzaak)
2. **Onvoldoende cache effectiviteit**
3. **Suboptimale Gunicorn/worker configuratie**

De goede berichten:
- Database is snel en niet het probleem
- Hardware heeft voldoende resources (CPU/Memory niet maxed)
- Problemen zijn **oplypbaar** met configuratie-optimalisaties

### Actions Required Before Production

**Urgent:**
- [ ] TCP TIME_WAIT tuning (kan meteen)
- [ ] Gunicorn worker-connections verhogen
- [ ] Cache TTL optimaliseren
- [ ] Hertest met moderate load

**Voor 100+ concurrent users:**
- [ ] Server upgrade naar CX43
- [ ] Connection pooling implementeren
- [ ] Load balancing setup
- [ ] Performance profiling en optimization

**Timeline:** Met de urgent fixes kan het systeem waarschijnlijk 50-75 concurrent users aan. Voor 100+ concurrent users zijn hardware upgrade + architectural changes nodig.

---

## Test Details

- **Test datum:** 2025-12-30
- **Test tools:** hey (HTTP benchmarking), wrk (Lua scripting)
- **Test locatie:** Local (127.0.0.1)
- **Network latency:** Minimaal (localhost)
- **Database size:** ~86 MB (klein)
- **Cache enabled:** Ja (TTL 300s)
