# SYNCTACLES V1 — F7-F9 Hardened Roadmap

**Versie:** 1.0  
**Datum:** 2025-12-14  
**Status:** Ready for Implementation  
**Developer:** Leo Blom (DATADIO)

---

## Executive Summary

Drie fases naar V1 launch-ready:
- **F7** (4-5 dagen): Technical hardening → stabiel, meetbaar, reproduceerbaar
- **F8** (2-3 dagen): Pre-productie → end-to-end user experience
- **F9** (2-3 dagen): Production launch → betalingen, soft launch, support

**Totaal:** ~10-11 dagen @ 6-8h/dag = V1 live

---

# F7 — Self-Test / Tech Hardening

**Doel:** V1 energy stack is stabiel, reproduceerbaar, meetbaar, belastbaar. Alles moet "groen" (#E53935 HIGH items moeten 100% slagen).

## F7 Items & Exit-Criteria

| # | Item | Prioriteit | Exit-Criterium | Bewijsregel (Command = Proof) |
|---|------|-----------|---|---|
| 7.1 | DNS + email/website setup | 🔴 HIGH | DNS resolveert, SSL cert werkt | `curl -I https://synctacles.io` → 200, cert valid |
| 7.2 | Logging/alerting (Grafana+Prometheus) | 🟠 MEDIUM | Dashboard zichtbaar, alerts triggeren | `curl http://localhost:3000/api/health` → 200; test alert fires |
| 7.3 | Server install script PROD-READY | 🔴 HIGH | Fresh VPS → running zonder handwerk | `sudo ./setup_v1_9.sh fase1-4` on clean Ubuntu 24.04 → `systemctl status synctacles-api` → active |
| 7.4 | Externe monitoring (UptimeRobot) | 🔴 HIGH | 48h+ uptime logs, alerting configured | UptimeRobot dashboard green; webhook test successful |
| 7.5 | Load testing (wrk + k6) | 🔴 HIGH | **Ceiling numbers**: p95 latency, max RPS, DB impact | `wrk -t12 -c100 -d30s http://localhost:8000/api/v1/balance` → p95 < 40ms; k6 test 5k RPS → 0 errors |
| 7.6 | Backups + restore test | 🔴 HIGH | Restore from backup works < 30 min, data integrity verified | `pg_restore backup.sql.gz; psql -c "SELECT COUNT(*) FROM norm_entso_e_a75"` → ✓ expected count |
| 7.7 | Automated tests (pytest + API contract + migrations) | 🔴 HIGH | All tests pass, coverage > 80% critical paths | `pytest -v tests/; coverage > 80%` → exit code 0 |
| 7.8 | Release discipline (versioning, tags, changelog, rollback) | 🟠 MEDIUM | Git tags exist (v1.0.0-rc1), changelog accurate, rollback plan docs | `git tag v1.0.0-rc1; cat CHANGELOG.md; test rollback script` |
| 7.9 | Security baseline (firewall, SSH, secrets, log retention) | 🟠 MEDIUM | SSH hardened, firewall rules active, no secrets in git, logs rotate | `sudo iptables -L; grep PermitRootLogin /etc/ssh/sshd_config` → no |
| 7.10 | SLO/Health definitie (what = "up", latencies, error budget) | 🟠 MEDIUM | SLO doc exists + SLI metrics in Prometheus | Document: "API up = all 3 endpoints responding < 100ms p95, < 0.1% error rate" |
| 7.11 | Multi-country DB schema (V2 prep) | 🟢 LOW | Schema doc complete, **NIET-BLOKKEREND** | `docs/v2_schema.md` exists, peer-reviewed, GitHub issue tagged v2 |

### F7 Go/No-Go Gate

```
F7 is DONE als:
✓ 7.1-7.5 + 7.6-7.10 (HIGH + MEDIUM) slagen = 100%
✓ Multi-country (7.11) mag later → niet-blokkerend
✓ Minimaal 72h stabiele self-test run zonder silent failures
✓ Alle kritieke commands werken → logboek opgeslagen in /opt/synctacles/logs/f7_validation.log
```

**Geschatte Tijd:** 12-18 uur (4-5 dagen @ 6-8h/dag)

---

# F8 — Pre-Productie (Productvorming zonder verkoop)

**Doel:** Nieuwe HA gebruiker kan in 15 min installeren → key plakken → sensor zien → zonder jouw handmatig babysitten.

## F8 Items & Exit-Criteria

| # | Item | Prioriteit | Exit-Criterium | Bewijsregel |
|---|------|-----------|---|---|
| 8.1 | Spoedcursus Home Assistant | 🔴 HIGH | Jij begrijpt HA architecture (entities, domains, integrations) | Kan binnen 30 min custom sensor entity van scratch bouwen |
| 8.2 | HA Integration + Blueprints | 🔴 HIGH | Custom component installeert via HACS, blueprints werken | `HACS → custom repo → install synctacles → entities created` |
| 8.3 | Fallback inbouwen (Fraunhofer/Energy-Charts) | 🔴 HIGH | Primaire bron uit → API wisselt naar fallback, blijft up | Kill ENTSO-E API; endpoint returns fallback data met `quality_status: FALLBACK` |
| 8.4 | Licentie key system (API + HA) | 🔴 HIGH | License key flow end-to-end werkt in staging (geen handmatige DB edits) | `POST /auth/generate-license` → key → `PUT /activate?license=XXX` → 200 OK |
| 8.5 | Usermanagement (backend MVP) ⚠️ | 🔴 HIGH | Email signup → API key generation → rate limit enforcement (KISS: geen password reset, geen roles yet) | `POST /users/signup?email=test@example.com` → API key returned; `X-API-Key` header blocks unlicensed |
| 8.6 | Documentatie platform | 🟠 MEDIUM | Docs site live, navigeerbaar, 90% coverage (install + config + troubleshoot) | ReadTheDocs mirror live; internal search works; Zendesk FAQ > 10 items |

### F8 Go/No-Go Gate

```
F8 is DONE als:
✓ Nieuwe user: 15 min → install → key → data zien (end-to-end demo recorded)
✓ Fallback aantoonbaar werkt (chaos test: primary down, fallback active)
✓ License flow works end-to-end (staging test, no manual edits)
✓ Zero handmatige DB edits voor user setup
✓ Docs accessible + 90% complete
```

### ⚠️ SCOPE-CREEP WAARSCHUWING

**Usermanagement = "license key → API key → rate limit".**  
NIET: password resets, role-based access, SSO, 2FA. → V1.1

**Geschatte Tijd:** 14-20 uur (2-3 dagen @ 6-8h/dag, overlappend met F9-voorbereiding)

---

# F9 — Productie / Launch (Geld + Merk + Operatie)

**Doel:** Betalingen accepteren, support leveren, gecontroleerd groeien.

## F9 Items & Exit-Criteria

| # | Item | Prioriteit | Exit-Criterium | Bewijsregel |
|---|------|-----------|---|---|
| 9.1 | Payment gateway (Mollie/Stripe) | 🔴 HIGH | Betaling → licentie → toegang = volledig automatisch | Test payment in Mollie → webhook creates license → user can activate |
| 9.2 | Soft launch (5–10 beta users) | 🔴 HIGH | 5 echte users, betaald, monitored, support pad actief | Dashboard: 5 active licenses; support@synctacles.io ontvangt tickets |
| 9.3 | Branding + logo | 🟠 MEDIUM | Logo gereed (extern of AI), website branded | `synctacles.io` toont logo; brand guide exists (can be minimal) |
| 9.4 | Website/landing page (Webflow) | 🟠 MEDIUM | Landing page live, pricing duidelijk, "how it works" visual | `synctacles.io` loads; pricing table visible; 1 demo video/gif |
| 9.5 | Bedrijf starten (Estonia/Portugal?) | 🟠 MEDIUM | Company registered, tax ID verkomen | Company extract + tax ID document (parallel to F9, niet-blokkerend) |
| 9.6 | Zakelijke bankrekening (Wise?) | 🟠 MEDIUM | Wise account setup, Mollie linked | Wise account active; test transfer successful |
| 9.7 | Support/FAQ (Zendesk?) | 🟠 MEDIUM | Support email live, FAQ > 10 items, response plan documented | `support@synctacles.io` forwards to Zendesk; FAQ page ≥ 10 items |

### F9 Go/No-Go Gate

```
F9 is DONE als:
✓ Payment → license → access = volledig automatisch
✓ 5-10 beta users paying, monitored, supported (dashboard shows active subscriptions)
✓ Website + pricing duidelijk (public.synctacles.io live)
✓ Support pad werkt (email forwarding tested, Zendesk active)
✓ Company legal ≥ setup initiated (kan parallel, check Mollie requirements)
```

### Snijlijst (Mag Later zonder Launch-Blocker)

- **Branding/logo:** Tekstlogo voldoende (kan V1.1)
- **Bedrijf/bank:** Kan parallel; maar payment provider (Mollie) kan eisen stellen → check
- **Documentatie:** MVP genoeg; verdere polish = V1.1

**Geschatte Tijd:** 12-18 uur (start mid-F8, 2-3 dagen overlap)

---

# Master Timeline

```
Dag 1-5:   F7 start
           └─ DNS, monitoring, load test, backups, tests, security, SLO

Dag 4-8:   F8 start (overlap met F7 tail)
           └─ HA, license, user mgmt
           
Dag 7-10:  F9 start (parallel met F8)
           └─ Payment, soft launch prep, website
           
Dag 10-11: F9 finish → Go/No-Go → LAUNCH

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: ~10-11 dagen @ 6-8h/dag = V1 Launch-Ready
```

---

# Dagelijkse Afteken-Checklist (Printable)

## DAG 1 — F7 Kickoff: DNS + Monitoring

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **DNS configureren** | `nslookup synctacles.io` | Resolveert naar server IP | ☐ |
| **SSL cert (Let's Encrypt)** | `certbot certonly -d synctacles.io` | Certificate installed in `/etc/letsencrypt/` | ☐ |
| **DNS werkingtest** | `curl -I https://synctacles.io` | `HTTP/2 200` + valid cert | ☐ |
| **Prometheus starten** | `docker run -d -p 9090:9090 prom/prometheus` | Port 9090 accessible | ☐ |
| **Grafana starten** | `docker run -d -p 3000:3000 grafana/grafana` | Port 3000 accessible, login works | ☐ |
| **API monitoring dashboard** | Create dashboard in Grafana for `/health`, response times | Dashboard shows live API metrics | ☐ |
| **EOD Go/No-Go** | All items above working | ✓ Proceed DAG 2 | ☐ |

---

## DAG 2 — F7 Testing: Load + Backups

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **Install wrk** | `apt-get install wrk` | wrk version prints | ☐ |
| **Baseline test (generation)** | `wrk -t12 -c100 -d30s http://localhost:8000/api/v1/generation-mix` | p50, p95, p99 latencies + RPS | ☐ |
| **Baseline test (load)** | `wrk -t12 -c100 -d30s http://localhost:8000/api/v1/load` | p50, p95, p99 latencies + RPS | ☐ |
| **Baseline test (balance)** | `wrk -t12 -c100 -d30s http://localhost:8000/api/v1/balance` | p50, p95, p99 latencies + RPS | ☐ |
| **Record ceiling numbers** | Log results in `/opt/synctacles/logs/f7_load_test.txt` | p95 < 40ms, max 5k RPS ✓ | ☐ |
| **Backup database** | `/opt/synctacles/scripts/backup_database.sh` | Backup file in `/opt/synctacles/backups/` | ☐ |
| **Restore test (staging DB)** | `pg_restore /opt/synctacles/backups/*.sql.gz` | Restore completes, data integrity OK | ☐ |
| **Time restore** | Time from backup start → usable DB | < 30 min ✓ | ☐ |
| **EOD Go/No-Go** | Load test ceiling numbers recorded + restore verified | ✓ Proceed DAG 3 | ☐ |

---

## DAG 3 — F7 Automation: Tests + Security

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **Create pytest suite** | `mkdir -p tests/; touch tests/test_api.py tests/test_db.py` | Test files exist | ☐ |
| **API contract tests** | Write POST/GET tests for 3 endpoints | All tests pass (`pytest -v` → exit 0) | ☐ |
| **Migration tests** | Test Alembic upgrade/downgrade | `alembic upgrade head`, then `alembic downgrade -1`, then `upgrade head` → OK | ☐ |
| **Coverage report** | `coverage run -m pytest; coverage report` | Coverage > 80% on critical paths | ☐ |
| **SSH hardening check** | `grep PermitRootLogin /etc/ssh/sshd_config` | Output: `PermitRootLogin no` | ☐ |
| **Firewall rules active** | `sudo iptables -L \| grep ACCEPT` | Rules for SSH, HTTP, HTTPS visible | ☐ |
| **No secrets in git** | `git log -p \| grep -i password` | No output (clean) | ☐ |
| **Log rotation active** | `cat /etc/logrotate.d/synctacles` | Rotation policy defined | ☐ |
| **EOD Go/No-Go** | All tests pass + security hardened | ✓ Proceed DAG 4 | ☐ |

---

## DAG 4 — F7 Monitoring: SLO + UptimeRobot

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **Define SLO document** | Create `docs/SLO.md`: "API up = all 3 endpoints < 100ms p95, < 0.1% error rate" | Document committed to git | ☐ |
| **Setup SLI metrics (Prometheus)** | Add scrape configs in Prometheus for `/health`, `/metrics` | Metrics visible in Prometheus dashboard | ☐ |
| **Create Grafana alerts** | Alert rules: "p95 latency > 100ms" + "error rate > 0.1%" + "any endpoint down" | Alerts fire on test (kill endpoint → alert triggers) | ☐ |
| **UptimeRobot account** | Signup + add 3 endpoint checks | Dashboard shows green status | ☐ |
| **UptimeRobot webhook** | Configure webhook to Slack/email on down | Test: kill endpoint → webhook fires within 60s | ☐ |
| **72h stability run** | Let system run unattended 72h, monitor logs for errors | `/opt/synctacles/logs/scheduler/*.log` clean (no critical errors) | ☐ |
| **Collect stability report** | `tail -100 /opt/synctacles/logs/scheduler/*.log > f7_stability_report.txt` | Report shows 72h of successful cycles | ☐ |
| **EOD Go/No-Go** | SLO defined, monitoring active, 72h run complete | ✓ F7 COMPLETE → F8 Start | ☐ |

---

## DAG 5 — F7 Validation: Fresh Install Test

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **Spin fresh Ubuntu 24.04 VM** | On Hetzner or local KVM | VM accessible via SSH | ☐ |
| **Run full install script** | `sudo ./setup_synctacles_server_v1_9.sh fase1` | fase1 completes, no errors | ☐ |
| **Run fase2** | `sudo ./setup_synctacles_server_v1_9.sh fase2` | fase2 completes (Docker, PostgreSQL, Redis) | ☐ |
| **Run fase3** | `sudo ./setup_synctacles_server_v1_9.sh fase3` | fase3 completes (security, SSH hardening) | ☐ |
| **Run fase4** | `sudo ./setup_synctacles_server_v1_9.sh fase4` | fase4 completes (Python venv + packages) | ☐ |
| **Verify services** | `systemctl status synctacles-api synctacles-collector synctacles-importer` | All 3 services running (active) | ☐ |
| **API health check** | `curl http://localhost:8000/health` | `{"status": "ok"}` | ☐ |
| **Database connectivity** | `psql -U synctacles -d synctacles -c "SELECT 1"` | `1` returned | ☐ |
| **Run validation script** | `/opt/github/synctacles-repo/validate_synctacles_setup_v2_fixed.sh` | Exit code 0 (all checks pass) | ☐ |
| **EOD Go/No-Go** | Fresh install flawless, all services active, API responding | ✓ F7 COMPLETE | ☐ |

---

## DAG 6-7 — F8 Kickoff: HA Integration

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **HA Learning** | Install local HA dev instance (Docker) + study entity/integration architecture | Can describe: "Entity = sensor with state + attributes; Integration = coordinator polling APIs" | ☐ |
| **Custom component skeleton** | `mkdir -p custom_components/synctacles/` + copy existing from F5 | Component structure matches HA standards | ☐ |
| **Config flow UI** | Review + refine config_flow.py for UX (API URL input) | User can input URL in 2 clicks | ☐ |
| **Test install (HACS)** | Add custom repo to HACS, install synctacles component | Component appears in HACS → Install button works | ☐ |
| **Entity creation** | Verify 3 sensors spawn (generation, load, balance) | `Developer Tools → States` shows 3 sensor entities | ☐ |
| **Blueprints** | Create 1-2 example automations (e.g., "alert when balance > 200MW") | Blueprint YAML valid, imports without error | ☐ |
| **EOD Go/No-Go** | Component installs, 3 sensors created, blueprints work | ✓ Proceed DAG 8 | ☐ |

---

## DAG 8 — F8 Hardening: License + Fallback

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **License key generation** | `POST /auth/generate-license?email=beta@example.com` | Returns UUID license key | ☐ |
| **License activation (API)** | `PUT /activate?license=<key>` | Returns API key + rate limit info | ☐ |
| **License validation (HA)** | HA component reads license_key from config → validates via API | Component enables only if license valid | ☐ |
| **API key rate limiting** | Call endpoint 100x in 1 sec with same API key | Response 429 (Too Many Requests) after limit hit | ☐ |
| **Fallback: Kill ENTSO-E** | Stop ENTSO-E collector, test `/api/v1/generation-mix` | Endpoint returns fallback data (Energy-Charts) + `quality_status: FALLBACK` | ☐ |
| **Fallback: Kill Energy-Charts** | Stop fallback too, test endpoint | Endpoint returns cache (old data) + `quality_status: STALE` | ☐ |
| **Chaos test** | Restart API, ensure in-flight requests don't crash | No 500 errors during restart cycle | ☐ |
| **EOD Go/No-Go** | License flow works end-to-end, fallback active, chaos tested | ✓ Proceed DAG 9 | ☐ |

---

## DAG 9-10 — F9 Payment + Soft Launch

| Taak | Commando | Expected Output | ✓ |
|------|----------|---|---|
| **Mollie sandbox account** | Create account, setup test keys | Mollie dashboard accessible | ☐ |
| **Payment webhook** | Configure webhook (Mollie → `POST /webhook/payment`) | Test payment fires webhook | ☐ |
| **Payment → License flow** | Test payment in Mollie sandbox → license auto-created | Dashboard: new license entry + email sent to user | ☐ |
| **Website landing page (Webflow)** | Setup Webflow template, add pricing table, "how it works" section | `synctacles.io` loads, displays pricing (€4.99/mo) | ☐ |
| **Support email + Zendesk** | Setup `support@synctacles.io`, forward to Zendesk | Test email → appears in Zendesk ticket | ☐ |
| **FAQ page** | Write ≥ 10 FAQ items (install, license, troubleshoot, pricing) | FAQ live on `synctacles.io/faq` | ☐ |
| **Invite 5 beta users** | Send invite email with sign-up link | 5 users receive email, can create account | ☐ |
| **Monitor 5 users** | Dashboard shows 5 active subscriptions | All 5 users show in licensing DB | ☐ |
| **Support test** | Have 1 user submit test support ticket | Ticket appears in Zendesk, you can respond | ☐ |
| **EOD Go/No-Go** | Payment works, 5 users active, support responds | ✓ F9 COMPLETE → V1 LAUNCH READY | ☐ |

---

## DAG 11 — Go/No-Go Final

| Taak | Check | Status | ✓ |
|------|-------|--------|---|
| **F7 all HIGH items pass** | 7.1-7.5 + 7.6-7.10 | 100% ✓ | ☐ |
| **F8 all HIGH items pass** | 8.1-8.5 | 100% ✓ | ☐ |
| **F9 all HIGH items pass** | 9.1-9.2 | 100% ✓ | ☐ |
| **No critical bugs** | Zero known P0 issues | All issues either fixed or documented as known limitation | ☐ |
| **Monitoring active** | UptimeRobot + Grafana + alerts | Green across all channels | ☐ |
| **Support pad operational** | Support email + Zendesk + FAQ live | Can handle incoming tickets | ☐ |
| **Communication ready** | Blog post / Twitter / announcement drafted | Ready to publish at go/no-go decision | ☐ |
| ****LAUNCH DECISION** | Go or No-Go | **GO** → publish announcement | ☐ |

---

# Reference: Commands by Category

## Monitoring & Health

```bash
# Check API status
curl http://localhost:8000/health

# Check all services
systemctl status synctacles-api synctacles-collector synctacles-importer synctacles-normalizer

# View active timers
systemctl list-timers synctacles-*

# Check database
psql -U synctacles -d synctacles -c "SELECT COUNT(*) FROM norm_entso_e_a75;"

# Monitor logs
tail -f /opt/synctacles/logs/scheduler/*.log
```

## Load Testing

```bash
# Install wrk
apt-get install wrk

# Baseline: generation endpoint
wrk -t12 -c100 -d30s http://localhost:8000/api/v1/generation-mix

# Baseline: load endpoint
wrk -t12 -c100 -d30s http://localhost:8000/api/v1/load

# Baseline: balance endpoint
wrk -t12 -c100 -d30s http://localhost:8000/api/v1/balance

# Install k6 (optional, database-aware load testing)
apt-get install k6

# Run k6 script
k6 run load_test.js
```

## Backup & Restore

```bash
# Manual backup
/opt/synctacles/scripts/backup_database.sh

# List backups
ls -lh /opt/synctacles/backups/

# Restore (on staging DB)
pg_restore /opt/synctacles/backups/synctacles_YYYYMMDD_HHMMSS.sql.gz
```

## Security

```bash
# Check SSH hardening
grep PermitRootLogin /etc/ssh/sshd_config

# Check firewall rules
sudo iptables -L

# Verify no secrets in git
git log -p | grep -i password

# Check log rotation
cat /etc/logrotate.d/synctacles
```

## Testing

```bash
# Run pytest
pytest -v tests/

# Coverage report
coverage run -m pytest
coverage report

# API contract tests
pytest tests/test_api.py -v

# Migration tests
alembic upgrade head && alembic downgrade -1 && alembic upgrade head
```

## Fresh Install Validation

```bash
# Run full validation
/opt/github/synctacles-repo/validate_synctacles_setup_v2_fixed.sh

# Check specific items
systemctl status synctacles-api
curl http://localhost:8000/health
psql -U synctacles -d synctacles -c "SELECT 1"
```

---

# Appendix: Key Decision Points

## F7 Exit

**HARD GO/NO-GO:**
- All HIGH items (7.1-7.5 + 7.6-7.10) = 100% pass
- 72h stable run = 0 silent failures
- Load test ceiling numbers recorded
- Fresh install validated on clean VM

**If No-Go:** Fix blocker, retry. No exceptions.

---

## F8 Exit

**HARD GO/NO-GO:**
- New user: 15 min → install → key → data (demo recorded)
- Fallback chaos test: primary down, fallback active, no data loss
- License flow: zero manual DB edits
- Docs: 90% complete, searchable

**If No-Go:** Scope reduction (e.g., defer blueprints to V1.1) or more dev time.

---

## F9 Exit

**HARD GO/NO-GO:**
- Payment flow: sandbox test successful, webhook fires
- 5 beta users active, paying, monitored
- Website + pricing public
- Support pad operational (Zendesk live)

**If No-Go:** Delay soft launch 1 week, complete missing pieces.

---

# Version History

| Versie | Datum | Changes |
|--------|-------|---------|
| 1.0 | 2025-12-14 | Initial hardened roadmap + daily checklists |

---

**Document Owner:** Leo Blom (DATADIO)  
**Last Updated:** 2025-12-14  
**Status:** Ready for Implementation

---

*Afdrukken, aantekenen, commit naar git. Elke dag één checklist voltooid = progress tracking.*
