# Claude Code - Load Test Scenario

**Datum:** 2025-12-30
**Server:** 135.181.255.83 (SSH: ha-energy-insights-nl)
**Doel:** Bepaal maximale capaciteit en identificeer bottlenecks

---

## Context

- Server: Hetzner CX33 (4 vCPU, 8GB RAM)
- API: FastAPI met async caching (TTL 60-300s)
- Database: PostgreSQL met indexes
- Proxy: Nginx met gzip

**Verwachte baseline (uit ARCHITECTURE.md):**
- Target: p95 < 100ms
- Target: 100+ concurrent users
- Target: < 1% error rate

---

## TAAK 1: Installeer load test tools

```bash
# Installeer wrk (moderne HTTP benchmarking tool)
apt update && apt install -y wrk

# Installeer hey (Go-based, makkelijke output)
wget -qO /usr/local/bin/hey https://hey-release.s3.us-east-2.amazonaws.com/hey_linux_amd64
chmod +x /usr/local/bin/hey

# Verify
wrk --version
hey -h | head -3
```

---

## TAAK 2: Pre-test health check

```bash
echo "=== Pre-Test Health Check ==="

# Check services
systemctl is-active energy-insights-nl-api && echo "✅ API running" || echo "❌ API down"
systemctl is-active postgresql && echo "✅ PostgreSQL running" || echo "❌ PostgreSQL down"
systemctl is-active nginx && echo "✅ Nginx running" || echo "❌ Nginx down"

# Check endpoints respond
for endpoint in health api/v1/generation-mix api/v1/load api/v1/balance; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8000/$endpoint")
    echo "/$endpoint: HTTP $STATUS"
done

# Baseline single request timing
echo ""
echo "=== Baseline (single request) ==="
curl -w "Time: %{time_total}s\n" -s -o /dev/null http://localhost:8000/api/v1/generation-mix
curl -w "Time: %{time_total}s\n" -s -o /dev/null http://localhost:8000/api/v1/generation-mix
```

---

## TAAK 3: Baseline test (light load)

```bash
echo "=== BASELINE TEST: 10 users, 30 seconds ==="

# Via nginx (port 80)
hey -n 1000 -c 10 -z 30s http://localhost/api/v1/generation-mix

# Record baseline metrics
echo ""
echo "Save these as BASELINE metrics"
```

**Verwachte resultaten:**
- Requests/sec: 50-200
- Latency p95: < 50ms
- Error rate: 0%

---

## TAAK 4: Moderate load test

```bash
echo "=== MODERATE TEST: 50 users, 60 seconds ==="

hey -n 5000 -c 50 -z 60s http://localhost/api/v1/generation-mix

echo ""
echo "=== Also test other endpoints ==="
hey -n 1000 -c 50 -z 30s http://localhost/api/v1/load
hey -n 1000 -c 50 -z 30s http://localhost/api/v1/balance
```

**Verwachte resultaten:**
- Requests/sec: 100-500
- Latency p95: < 100ms
- Error rate: < 1%

---

## TAAK 5: Stress test (find breaking point)

```bash
echo "=== STRESS TEST: 100 users, 60 seconds ==="

hey -n 10000 -c 100 -z 60s http://localhost/api/v1/generation-mix

echo ""
echo "=== EXTREME TEST: 200 users, 30 seconds ==="

hey -n 10000 -c 200 -z 30s http://localhost/api/v1/generation-mix
```

**Let op:**
- Waar begint latency te stijgen?
- Waar komen eerste errors?
- CPU/memory tijdens test

---

## TAAK 6: Monitor resources tijdens test

Open een **tweede terminal** en run:

```bash
# Real-time monitoring (run dit TIJDENS de load test)
echo "=== Resource Monitor ==="
echo "Start load test in andere terminal..."
echo ""

# Watch CPU, memory, connections
watch -n 1 '
echo "=== $(date) ==="
echo ""
echo "--- CPU & Memory ---"
top -bn1 | head -5
echo ""
echo "--- API Process ---"
ps aux | grep uvicorn | grep -v grep
echo ""
echo "--- Active Connections ---"
ss -s | head -5
echo ""
echo "--- PostgreSQL ---"
sudo -u postgres psql -c "SELECT count(*) as active_connections FROM pg_stat_activity WHERE state = '\''active'\'';" 2>/dev/null | tail -3
'
```

---

## TAAK 7: Database performance test

```bash
echo "=== DATABASE PERFORMANCE ==="

# Direct database query timing
sudo -u postgres psql -d energy_insights_nl << 'SQL'
\timing on

-- Most used query (generation endpoint)
SELECT * FROM norm_entso_e_a75 
ORDER BY timestamp DESC 
LIMIT 1;

-- Load query
SELECT * FROM norm_entso_e_a65 
ORDER BY timestamp DESC 
LIMIT 1;

-- Balance query  
SELECT * FROM norm_tennet_balance 
ORDER BY timestamp DESC 
LIMIT 1;

-- Table sizes
SELECT 
    relname as table,
    pg_size_pretty(pg_total_relation_size(relid)) as size,
    n_live_tup as rows
FROM pg_stat_user_tables 
ORDER BY pg_total_relation_size(relid) DESC;
SQL
```

---

## TAAK 8: Wrk advanced test (Lua scripting)

```bash
# Create Lua script for mixed endpoint testing
cat > /tmp/mixed_load.lua << 'LUA'
-- Mixed endpoint load test
local endpoints = {
    "/api/v1/generation-mix",
    "/api/v1/load", 
    "/api/v1/balance",
    "/api/v1/signals",
    "/health"
}

request = function()
    local path = endpoints[math.random(#endpoints)]
    return wrk.format("GET", path)
end

done = function(summary, latency, requests)
    io.write("------------------------------\n")
    io.write(string.format("Total requests: %d\n", summary.requests))
    io.write(string.format("Total errors:   %d\n", summary.errors.status))
    io.write(string.format("Requests/sec:   %.2f\n", summary.requests / summary.duration * 1000000))
    io.write(string.format("Avg latency:    %.2f ms\n", latency.mean / 1000))
    io.write(string.format("P99 latency:    %.2f ms\n", latency:percentile(99) / 1000))
end
LUA

echo "=== MIXED ENDPOINT TEST: 50 users, 60 seconds ==="
wrk -t4 -c50 -d60s -s /tmp/mixed_load.lua http://localhost:8000
```

---

## TAAK 9: Cache effectiveness test

```bash
echo "=== CACHE EFFECTIVENESS TEST ==="

# Clear any existing cache by restarting API
systemctl restart energy-insights-nl-api
sleep 3

echo "--- Cold cache (first requests) ---"
for i in {1..5}; do
    curl -w "Request $i: %{time_total}s\n" -s -o /dev/null http://localhost/api/v1/generation-mix
done

echo ""
echo "--- Warm cache (subsequent requests) ---"
for i in {1..5}; do
    curl -w "Request $i: %{time_total}s\n" -s -o /dev/null http://localhost/api/v1/generation-mix
done

echo ""
echo "Expected: Warm cache requests should be 2-10x faster"
```

---

## TAAK 10: Generate load test report

```bash
cat > /opt/github/ha-energy-insights-nl/LOAD_TEST_REPORT.md << 'REPORT'
# SYNCTACLES Load Test Report

**Datum:** $(date +%Y-%m-%d)
**Server:** Hetzner CX33 (4 vCPU, 8GB RAM)
**Tester:** Claude Code

---

## Test Configuratie

- Tool: hey + wrk
- Target: http://localhost (via nginx)
- Endpoints: /api/v1/generation-mix (primary)
- Caching: TTL 300s (generation)

---

## Resultaten

### Baseline (10 concurrent users)

| Metric | Waarde | Target | Status |
|--------|--------|--------|--------|
| Requests/sec | XXX | 50+ | ✅/❌ |
| Latency p50 | XXX ms | < 20ms | ✅/❌ |
| Latency p95 | XXX ms | < 50ms | ✅/❌ |
| Latency p99 | XXX ms | < 100ms | ✅/❌ |
| Error rate | XXX% | < 0.1% | ✅/❌ |

### Moderate (50 concurrent users)

| Metric | Waarde | Target | Status |
|--------|--------|--------|--------|
| Requests/sec | XXX | 100+ | ✅/❌ |
| Latency p50 | XXX ms | < 30ms | ✅/❌ |
| Latency p95 | XXX ms | < 100ms | ✅/❌ |
| Latency p99 | XXX ms | < 200ms | ✅/❌ |
| Error rate | XXX% | < 1% | ✅/❌ |

### Stress (100 concurrent users)

| Metric | Waarde | Target | Status |
|--------|--------|--------|--------|
| Requests/sec | XXX | 100+ | ✅/❌ |
| Latency p95 | XXX ms | < 200ms | ✅/❌ |
| Error rate | XXX% | < 5% | ✅/❌ |

### Breaking Point

| Metric | Waarde |
|--------|--------|
| Max concurrent users | XXX |
| Max requests/sec | XXX |
| First errors at | XXX users |
| Unacceptable latency at | XXX users |

---

## Cache Effectiveness

| Scenario | Avg Response Time |
|----------|-------------------|
| Cold cache | XXX ms |
| Warm cache | XXX ms |
| Improvement | XXX% |

---

## Resource Usage (peak)

| Resource | Usage |
|----------|-------|
| CPU | XXX% |
| Memory | XXX MB |
| DB connections | XXX |
| Open files | XXX |

---

## Bottlenecks Identified

1. [ ] CPU bound
2. [ ] Memory bound
3. [ ] Database connections
4. [ ] Network I/O
5. [ ] Disk I/O

---

## Recommendations

1. ...
2. ...
3. ...

---

## Conclusie

**Production ready:** ✅/❌
**Max safe concurrent users:** XXX
**Headroom:** XXX%

REPORT

echo "✅ Report template created: /opt/github/ha-energy-insights-nl/LOAD_TEST_REPORT.md"
echo "Update with actual values after tests complete"
```

---

## TAAK 11: Run complete test suite en vul rapport in

```bash
echo "========================================"
echo "  COMPLETE LOAD TEST SUITE"
echo "  $(date)"
echo "========================================"

# Run all tests sequentially and capture output
RESULTS_FILE="/tmp/loadtest_results_$(date +%Y%m%d_%H%M%S).txt"

{
    echo "=== BASELINE (10 users, 30s) ==="
    hey -n 1000 -c 10 -z 30s http://localhost/api/v1/generation-mix
    
    sleep 5
    
    echo ""
    echo "=== MODERATE (50 users, 60s) ==="
    hey -n 5000 -c 50 -z 60s http://localhost/api/v1/generation-mix
    
    sleep 5
    
    echo ""
    echo "=== STRESS (100 users, 60s) ==="
    hey -n 10000 -c 100 -z 60s http://localhost/api/v1/generation-mix
    
    sleep 5
    
    echo ""
    echo "=== EXTREME (200 users, 30s) ==="
    hey -n 10000 -c 200 -z 30s http://localhost/api/v1/generation-mix
    
} | tee "$RESULTS_FILE"

echo ""
echo "========================================"
echo "Results saved to: $RESULTS_FILE"
echo "========================================"

# Parse key metrics and update report
echo ""
echo "Update LOAD_TEST_REPORT.md with the values above"
```

---

## Exit Criteria

Na uitvoering:
- [ ] hey en wrk geïnstalleerd
- [ ] Baseline test voltooid
- [ ] Stress test voltooid
- [ ] Breaking point geïdentificeerd
- [ ] Cache effectiveness gemeten
- [ ] LOAD_TEST_REPORT.md ingevuld met resultaten
- [ ] Bottlenecks geïdentificeerd (indien aanwezig)

**Success criteria:**
- p95 latency < 100ms bij 50 concurrent users
- Error rate < 1% bij 100 concurrent users
- Requests/sec > 100 sustained

---

## Initiële Opdracht voor Claude Code

```
Je bent een performance engineer die een load test uitvoert op SYNCTACLES.

WERKWIJZE:
1. Lees CLAUDE_CODE_LOADTEST.md volledig
2. Voer taken 1-11 sequentieel uit
3. Capture alle output
4. Vul LOAD_TEST_REPORT.md in met echte waarden
5. Identificeer bottlenecks en geef recommendations

BELANGRIJK:
- Wacht 5 seconden tussen zware tests
- Monitor resources tijdens stress tests
- Stop als server instabiel wordt
- Rapporteer alle metrics

START:
Begin met TAAK 1 (install tools).
```
