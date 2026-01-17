# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** GitHub Reorganization

---

## DOEL

GitHub wordt Single Source of Truth voor alle taken. Reorganiseer bestaande issues, maak nieuwe aan, en structureer met labels/milestones.

---

## DEEL 1: GITHUB STRUCTUUR

### Labels Aanmaken

```bash
# Prioriteit (kleur: rood → groen)
critical    #B60205  "Blocker voor launch"
high        #D93F0B  "Moet voor launch"
medium      #FBCA04  "Nice to have voor launch"
low         #0E8A16  "Post-launch"

# Fase
F7          #1D76DB  "Technical Hardening"
F8          #5319E7  "Pre-Production"
F9          #0052CC  "Production Launch"

# Type
bug         #D73A4A  "Something isn't working"
feature     #A2EEEF  "New feature"
docs        #0075CA  "Documentation"
devops      #FEF2C0  "CI/CD, Infrastructure"
security    #D4C5F9  "Security related"

# Status
blocked     #000000  "Waiting on dependency"
in-progress #FFFF00  "Currently being worked on"
```

### Milestones Aanmaken

| Milestone | Due Date | Description |
|-----------|----------|-------------|
| V1 Launch | 2026-02-01 | Eerste betalende klanten |
| V1.1 Stability | 2026-02-15 | Post-launch fixes |
| V2 Expansion | 2026-03-01 | DE markt, extra features |

---

## DEEL 2: BESTAANDE ISSUES UPDATEN

### Sluiten met Comment

| Issue | Reden | Comment |
|-------|-------|---------|
| #15 | Cache compleet | "✅ Closed: Cache migration complete. 91.7% hit rate, 39x performance improvement, 20x capacity increase. Commits: f6f4bbb, e8a6152" |
| #35 | Endpoint bestaat niet | "✅ Closed: /v1/now endpoint does not exist in codebase. Verified via code search and 404 response." |
| #36 | Gedocumenteerd | "✅ Closed: /metrics endpoints documented in ARCHITECTURE.md. Commit: d33db5b" |
| #37 | Gedocumenteerd | "✅ Closed: Cache endpoints documented in api-reference.md. Commit: d33db5b" |
| #38 | Gedocumenteerd | "✅ Closed: /auth/admin/users documented. Commit: e0d356e" |
| #39 | Gedocumenteerd | "✅ Closed: TenneT tables marked as deprecated in ARCHITECTURE.md. Commit: e0d356e" |
| #41 | Gedocumenteerd | "✅ Closed: Systemd timers documented. Commit: d33db5b" |

### Labels Toevoegen aan Bestaande Issues

| Issue | Titel | Nieuwe Labels | Milestone |
|-------|-------|---------------|-----------|
| #5 | Unit Test Suite | `high`, `F7`, `devops` | V1.1 Stability |
| #6 | Monitoring & Alerting | `high`, `F7`, `devops` | V1 Launch |
| #7 | CI/CD Pipeline | `medium`, `F7`, `devops` | V1.1 Stability |
| #8 | Database HA & Backups | `high`, `F7`, `devops`, `security` | V1 Launch |
| #22 | Load Testing Edge Cases | `low`, `F7`, `devops` | V1.1 Stability |
| #23 | Code Quality Audit | `low`, `F7`, `docs` | V1.1 Stability |
| #40 | Endpoint Routing Verification | `low`, `F7`, `docs` | V1 Launch |
| #42 | API Reference Complete | `medium`, `F8`, `docs` | V1 Launch |

### Issues die al Compleet zijn (Sluiten)

| Issue | Status | Comment |
|-------|--------|---------|
| #22 | Load test uitgevoerd | "✅ Closed: Load testing complete. 2,596 req/sec peak, 100K+ user capacity. Report: docs/performance/CAPACITY_TEST_2026-01-08.md" |
| #40 | Routing verified | "✅ Closed: All routes verified working. Audit: docs/ENDPOINT_ROUTING_AUDIT.md. Bugs found → separate issues #XX, #XX" |

---

## DEEL 3: NIEUWE ISSUES AANMAKEN

### F7 - Technical Hardening (afronden)

```markdown
---
### Issue: F7.3 - Install Script Validation
**Labels:** `medium`, `F7`, `devops`
**Milestone:** V1 Launch

**Description:**
Validate installer works on fresh Ubuntu 24.04.

**Acceptance Criteria:**
- [ ] Fresh VM provisioned
- [ ] `./scripts/setup/setup_*.sh fase0` runs without errors
- [ ] All fases complete successfully
- [ ] Services running after install
- [ ] API responds to health check

**Effort:** 2-3 hours
---

### Issue: F7.4 - External Uptime Monitoring
**Labels:** `high`, `F7`, `devops`
**Milestone:** V1 Launch

**Description:**
Setup external monitoring (UptimeRobot/Pingdom) for API availability.

**Acceptance Criteria:**
- [ ] External monitor configured
- [ ] Checks https://api.synctacles.com/health every 1 min
- [ ] Alert via email/Telegram on downtime
- [ ] 48h+ uptime baseline established

**Effort:** 30 minutes
---

### Issue: F7.6 - Backup/Restore Test
**Labels:** `high`, `F7`, `devops`, `security`
**Milestone:** V1 Launch

**Description:**
Test database backup and restore procedure.

**Acceptance Criteria:**
- [ ] Manual backup created: `pg_dump energy_insights_nl | gzip > backup.sql.gz`
- [ ] Restore tested on separate DB or fresh install
- [ ] Data integrity verified (record counts match)
- [ ] Restore time < 30 min documented
- [ ] Automated daily backup script created

**Effort:** 1-2 hours
---

### Issue: F7.9 - Security Audit
**Labels:** `high`, `F7`, `security`
**Milestone:** V1 Launch

**Description:**
Basic security audit before public launch.

**Checklist:**
- [ ] UFW firewall enabled (22, 80, 443 only)
- [ ] SSH password auth disabled
- [ ] No secrets in git history
- [ ] .env permissions 600
- [ ] PostgreSQL not exposed externally
- [ ] API rate limiting active
- [ ] HTTPS enforced (no HTTP)
- [ ] Dependencies scanned for vulnerabilities

**Effort:** 2-3 hours
---
```

### F8 - Pre-Production (CRITICAL)

```markdown
---
### Issue: F8.2 - Home Assistant Custom Component
**Labels:** `critical`, `F8`, `feature`
**Milestone:** V1 Launch

**Description:**
Build HACS-compatible Home Assistant integration.

**Features:**
- Custom component installable via HACS
- Config flow (UI-based setup)
- Sensors for: generation mix, load, prices, signals
- 15-minute polling interval
- Error handling with meaningful messages

**Acceptance Criteria:**
- [ ] HACS installation works
- [ ] Config flow: API URL + API key input
- [ ] Sensors created: `sensor.synctacles_*`
- [ ] Data updates every 15 min
- [ ] Graceful error handling (API down, invalid key)
- [ ] Documentation for users

**Effort:** 2-3 days
**Note:** Consider ChatGPT for HA-specific expertise
---

### Issue: F8.4 - License/API Key System
**Labels:** `critical`, `F8`, `feature`
**Milestone:** V1 Launch

**Description:**
Implement API key generation and validation.

**Features:**
- API key generation on signup
- Key validation middleware
- Key stored hashed in database
- Rate limiting per key
- Key regeneration endpoint

**Database:**
```sql
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    key_hash VARCHAR(64) NOT NULL,
    tier VARCHAR(20) DEFAULT 'free',
    daily_limit INT DEFAULT 10000,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);
```

**Acceptance Criteria:**
- [ ] POST /auth/signup → returns API key (once)
- [ ] All /api/v1/* endpoints require X-API-Key header
- [ ] Invalid key → 401 Unauthorized
- [ ] Rate limit headers in response
- [ ] Key regeneration works

**Effort:** 1-2 days
---

### Issue: F8.5 - User Registration System
**Labels:** `critical`, `F8`, `feature`
**Milestone:** V1 Launch
**Depends on:** #F8.4

**Description:**
User signup and management.

**Features:**
- Email signup (no password initially - API key only)
- Email verification (optional for V1)
- User dashboard (usage stats)
- Account deactivation

**Database:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    tier VARCHAR(20) DEFAULT 'free',
    created_at TIMESTAMP DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);
```

**Endpoints:**
- POST /auth/signup - Create account
- GET /auth/stats - View usage
- POST /auth/regenerate-key - New API key
- POST /auth/deactivate - Disable account

**Acceptance Criteria:**
- [ ] Signup with email works
- [ ] API key returned on signup
- [ ] Usage stats endpoint works
- [ ] Rate limit tracking per user

**Effort:** 1 day
---

### Issue: F8.6 - ReadTheDocs Deployment
**Labels:** `medium`, `F8`, `docs`
**Milestone:** V1 Launch

**Description:**
Deploy documentation to ReadTheDocs.

**Acceptance Criteria:**
- [ ] docs.synctacles.com live
- [ ] Sphinx/MkDocs configured
- [ ] Auto-deploy on git push
- [ ] Search functional
- [ ] Mobile responsive

**Effort:** 4-6 hours
---
```

### F9 - Production Launch

```markdown
---
### Issue: F9.1 - Payment Gateway Integration
**Labels:** `high`, `F9`, `feature`
**Milestone:** V1 Launch

**Description:**
Integrate Mollie for EU payments.

**Features:**
- Mollie checkout integration
- Monthly subscription (€4.99)
- Automatic tier upgrade on payment
- Webhook for payment confirmation
- Invoice generation

**Acceptance Criteria:**
- [ ] Mollie account configured
- [ ] Checkout flow works
- [ ] Payment → tier upgrade automatic
- [ ] Webhook processes correctly
- [ ] Test mode verified

**Effort:** 1 day
---

### Issue: F9.2 - Landing Page
**Labels:** `medium`, `F9`, `feature`
**Milestone:** V1 Launch

**Description:**
Public website for SYNCTACLES.

**Pages:**
- Home (value prop, features)
- Pricing (free vs paid)
- Docs (link to ReadTheDocs)
- Signup

**Acceptance Criteria:**
- [ ] synctacles.com live
- [ ] Responsive design
- [ ] Signup CTA works
- [ ] Links to HA integration docs

**Effort:** 1 day
---

### Issue: F9.3 - Beta User Onboarding
**Labels:** `high`, `F9`, `feature`
**Milestone:** V1 Launch

**Description:**
Onboard 5-10 beta users for soft launch.

**Process:**
1. Recruit from HA community (Reddit, Discord)
2. Provide free premium tier
3. Collect feedback actively
4. Monitor usage and errors
5. Iterate before public launch

**Acceptance Criteria:**
- [ ] 5-10 beta users active
- [ ] Feedback channel established
- [ ] No critical bugs reported
- [ ] 48h stable operation

**Effort:** Ongoing (1 week)
---
```

### Bugs Gevonden Vandaag

```markdown
---
### Issue: /api/v1/now Returns 500 Error
**Labels:** `high`, `bug`, `F7`
**Milestone:** V1 Launch

**Description:**
`GET /api/v1/now` returns HTTP 500 Internal Server Error.

**Expected:** 200 with current timestamp + status, or 404 if not implemented.

**Steps to Reproduce:**
```bash
curl http://localhost:8000/api/v1/now
# Returns 500
```

**Discovered:** Endpoint routing audit (#40)

**Effort:** 1-2 hours
---

### Issue: /api/v1/balance Returns 501 Instead of 410
**Labels:** `low`, `bug`, `F7`
**Milestone:** V1.1 Stability

**Description:**
Deprecated TenneT endpoint returns 501 (Not Implemented) instead of 410 (Gone).

**Expected:** 410 Gone with deprecation message and migration info.

**Fix:**
```python
raise HTTPException(
    status_code=410,
    detail={
        "error": "Gone",
        "message": "TenneT balance endpoint deprecated. See ADR-001.",
        "migration": "Use BYO-key model via Home Assistant"
    }
)
```

**Effort:** 15 minutes
---
```

---

## DEEL 4: PRIORITEIT OVERZICHT

### V1 Launch Critical Path

```
Week 1:
├── F8.4 License System (1-2d) ──┐
├── F8.5 User Management (1d) ───┴── Parallel
├── F7.4 Uptime Monitor (30m)
└── F7.6 Backup Test (2h)

Week 2:
├── F8.2 HA Component (2-3d) ←── ChatGPT
└── F7.9 Security Audit (3h)

Week 3:
├── F9.1 Payment Gateway (1d)
├── F8.6 ReadTheDocs (4h)
└── F9.2 Landing Page (1d)

Week 4:
├── F9.3 Beta Onboarding
└── Bug fixes + polish
```

### Issue Prioriteit Matrix

| Prioriteit | Issues | Totaal Effort |
|------------|--------|---------------|
| **CRITICAL** | F8.2, F8.4, F8.5 | 4-6 dagen |
| **HIGH** | F7.4, F7.6, F7.9, F9.1, F9.3, #6, #8, /now bug | 2-3 dagen |
| **MEDIUM** | F7.3, F8.6, F9.2, #42 | 2-3 dagen |
| **LOW** | #5, #7, #22, #23, /balance bug | Post-launch |

---

## DEEL 5: VERIFICATIE

Na uitvoering:

```bash
# Check alle issues aangemaakt
gh issue list --repo [repo] --state open

# Check labels bestaan
gh label list --repo [repo]

# Check milestones
gh api repos/[owner]/[repo]/milestones
```

**Report terug:**
- Aantal issues aangemaakt
- Aantal issues gesloten
- Screenshot van project board (indien aangemaakt)

---

## GIT COMMIT

Geen code changes, alleen GitHub management.

---

*Template versie: 1.0*
