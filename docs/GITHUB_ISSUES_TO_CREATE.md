# GITHUB ISSUES TO CREATE

**Generated:** 2026-01-08
**Context:** Product Reality Check audit found 11 gaps
**Repo:** ldraism/synctacles-api

---

## PREREQUISITES

```bash
# Authenticate gh CLI first
gh auth login

# Create labels if they don't exist
gh label create "gap-audit" --description "Found during gap audit" --color "FBCA04"
gh label create "docs-code-mismatch" --description "Documentation doesn't match code" --color "D93F0B"
```

---

## HIGH PRIORITY ISSUES (3)

### Issue 1: Database not updating - stale data since 2026-01-05

```bash
gh issue create \
  --title "[GAP] Database not updating - stale data since 2026-01-05" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
Database bevat geen data nieuwer dan 2026-01-05 13:45:00 UTC (2.5 dagen oud).
Collectors draaien volgens systemd timers, maar schrijven niet naar database.

## Evidence
- Database query: Latest data \`2026-01-05 13:45:00+00\`
- Timers active: collector (15 min), importer (15 min), normalizer (15 min)
- API health: Server running, maar serveert stale data
- Record counts: norm_a75=1502, norm_a65=1772, norm_a44=856

## Source
- SKILL: SKILL_02, SKILL_06 (data should be < 90 min old)
- Code: Database tables \`norm_entso_e_a75/a65/a44\` have stale data
- Production: Verified via \`sudo -u postgres psql -d energy_insights_nl\`

## Action Required
- [ ] Check collector logs: \`journalctl -u energy-insights-nl-collector -n 100\`
- [ ] Check importer logs: \`journalctl -u energy-insights-nl-importer -n 100\`
- [ ] Verify database connection credentials
- [ ] Verify ENTSO-E API token validity
- [ ] Add monitoring/alerts for data staleness

## Priority
**HIGH** - Production system serving 2.5-day-old data to users

## Impact
All API endpoints (\`/v1/generation-mix\`, \`/v1/load\`, \`/v1/prices\`, \`/v1/signals\`) returning stale data." \
  --label "gap-audit,bug,priority:high"
```

---

### Issue 2: TenneT service failing continuously for 24+ hours

```bash
gh issue create \
  --title "[GAP] TenneT service failing continuously - should be disabled" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
\`energy-insights-nl-tennet.service\` fails every 5 minutes (30 failures in 24h).
SKILL_02 and SKILL_06 document TenneT as **BYO-key only** (HA component), but server-side collector still active.

## Evidence
- Service status: \`systemd[1]: Failed to start energy-insights-nl-tennet.service - Energy Insights NL TenneT Collector (Rate Limited)\`
- Error frequency: Every 5 minutes for 24+ hours
- Journal errors: 30 identical failures
- SKILL_02 line 102: \"TenneT (BYO-key via HA only, not server)\"
- SKILL_06 line 116-119: \"❌ NOT available via SYNCTACLES API\"

## Source
- SKILL: SKILL_02, SKILL_06 (TenneT BYO-only policy)
- Code: Service \`energy-insights-nl-tennet.service\` still exists and is triggered by timer
- Production: \`systemctl status energy-insights-nl-tennet.service\` shows failed

## Action Required
- [ ] Disable TenneT service: \`sudo systemctl disable energy-insights-nl-tennet.service\`
- [ ] Disable TenneT timer: \`sudo systemctl disable energy-insights-nl-tennet.timer\`
- [ ] Stop both: \`sudo systemctl stop energy-insights-nl-tennet.{service,timer}\`
- [ ] Remove TenneT collector code or mark as deprecated
- [ ] Document TenneT tables as deprecated (\`raw_tennet_balance\`, \`norm_tennet_balance\`)

## Priority
**HIGH** - Service failures polluting logs, wasting resources

## Impact
Continuous error logging, potential rate limit issues if API key is valid." \
  --label "gap-audit,docs-code-mismatch,priority:high"
```

---

### Issue 3: API serving stale data without quality warnings

```bash
gh issue create \
  --title "[GAP] API serving 2.5-day-old data - quality metadata not reflecting staleness" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
API endpoints return 2.5-day-old data but may not reflect staleness in quality metadata.
SKILL_02 defines FRESH < 90min, STALE 90-180min, FALLBACK > 180min.
Current data age: ~62 hours (3720 minutes) - far beyond fallback threshold.

## Evidence
- Latest database data: 2026-01-05 13:45:00 UTC
- Current time: 2026-01-08 03:00 UTC
- Data age: 2.5 days (150x FRESH threshold)
- SKILL_02 freshness thresholds: ENTSO-E FRESH < 90min, fallback trigger > 180min

## Source
- SKILL: SKILL_02 (Fallback Strategy section, lines 507-543)
- Code: API endpoints \`/v1/generation-mix\`, \`/v1/load\`, \`/v1/prices\`
- Production: Verified data age via database query

## Action Required
- [ ] Add staleness detection to API responses
- [ ] Return \`quality_status: UNAVAILABLE\` for data > 3 hours old
- [ ] Add \`data_age_hours\` to metadata
- [ ] Implement alerts when data > 1 hour old
- [ ] Consider fallback to Energy-Charts if ENTSO-E data > 3h old

## Priority
**HIGH** - Users receiving ancient data without warnings

## Impact
Automation based on stale data, incorrect signals, user trust erosion." \
  --label "gap-audit,bug,priority:high"
```

---

## MEDIUM PRIORITY ISSUES (5)

### Issue 4: Undocumented API endpoint `/v1/now`

```bash
gh issue create \
  --title "[GAP] Undocumented endpoint: /v1/now" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
Endpoint \`/v1/now\` exists in code but not documented in any SKILL.

## Source
- SKILL: Not mentioned in SKILL_02, SKILL_04
- Code: Found via \`grep -r \"@app\\.\\|@router\\.\" --include=\"*.py\"\`

## Action Required
- [ ] Document endpoint in SKILL_04 (Product Requirements)
- [ ] Add to API reference documentation (when created)
- [ ] Describe what data it returns (combined current data?)

## Priority
MEDIUM" \
  --label "gap-audit,documentation"
```

---

### Issue 5: Undocumented endpoint `/metrics`

```bash
gh issue create \
  --title "[GAP] Undocumented endpoint: /metrics (Prometheus)" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
Endpoint \`/metrics\` exists (Prometheus metrics) but not documented in SKILLs.

## Source
- SKILL: Not mentioned in SKILL_02 Observability section
- Code: Found via grep

## Action Required
- [ ] Document endpoint in SKILL_02 (Observability section)
- [ ] Add to API reference documentation
- [ ] Describe metrics exposed

## Priority
MEDIUM" \
  --label "gap-audit,documentation"
```

---

### Issue 6: Undocumented cache management endpoints (3 endpoints)

```bash
gh issue create \
  --title "[GAP] Undocumented cache management endpoints" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
3 cache endpoints exist but not documented:
- \`GET /cache/stats\`
- \`POST /cache/clear\`
- \`POST /cache/invalidate/{pattern}\`

## Source
- SKILL: Not mentioned in SKILL_02 or any other SKILL
- Code: Found via grep

## Action Required
- [ ] Document in SKILL_02 or create operations guide
- [ ] Add to API reference documentation
- [ ] Describe use cases (ops/debugging)
- [ ] Document authentication requirements

## Priority
MEDIUM" \
  --label "gap-audit,documentation"
```

---

### Issue 7: Undocumented admin endpoint `/auth/admin/users`

```bash
gh issue create \
  --title "[GAP] Undocumented endpoint: /auth/admin/users" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
Endpoint \`/auth/admin/users\` exists but not documented.

## Source
- SKILL: Not mentioned in SKILL_04 (auth endpoints section)
- Code: Found via grep

## Action Required
- [ ] Document in SKILL_04 or mark as internal-only
- [ ] If public: add to API reference
- [ ] If internal: add to ops guide or remove from public routes
- [ ] Describe authentication/authorization requirements

## Priority
MEDIUM" \
  --label "gap-audit,documentation"
```

---

### Issue 8: Orphaned TenneT database tables

```bash
gh issue create \
  --title "[GAP] Orphaned TenneT tables in database" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
Database contains TenneT tables but TenneT is BYO-only (no server-side collection):
- \`raw_tennet_balance\`
- \`norm_tennet_balance\`

## Source
- SKILL: SKILL_02, SKILL_06 (TenneT BYO-only)
- Code: Database schema shows tables exist
- Production: Verified via \`\dt\` in psql

## Action Required
- [ ] Mark tables as deprecated in schema documentation
- [ ] Add migration guide if data needs archiving
- [ ] Consider dropping tables in future migration
- [ ] Update SKILL_02 database schema section to note deprecation

## Priority
MEDIUM" \
  --label "gap-audit,docs-code-mismatch"
```

---

## LOW PRIORITY ISSUES (3)

### Issue 9: Verify endpoint aliases (generation/current vs generation-mix)

```bash
gh issue create \
  --title "[GAP] Verify endpoint routing aliases" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
SKILLs use \`/v1/generation/current\`, grep found \`/v1/generation-mix\`.
May be aliases in routing, but not verified.

Similar patterns for load, prices, balance, signals.

## Source
- SKILL: SKILL_02 documents \`/v1/generation/current\`
- Code: Grep found \`/v1/generation-mix\`

## Action Required
- [ ] Check FastAPI routing for aliases
- [ ] Document all endpoint variants
- [ ] Update SKILLs with canonical + alias list
- [ ] Ensure both forms work (if aliases exist)

## Priority
LOW" \
  --label "gap-audit,documentation"
```

---

### Issue 10: Incomplete systemd timer documentation

```bash
gh issue create \
  --title "[GAP] Incomplete systemd timer documentation" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
SKILL_02 mentions systemd timers generically, but doesn't list all 5 active timers:
- energy-insights-nl-importer.timer (15 min)
- energy-insights-nl-health.timer (5 min)
- energy-insights-nl-tennet.timer (5 min) - should be disabled
- energy-insights-nl-collector.timer (15 min)
- energy-insights-nl-normalizer.timer (15 min)

## Source
- SKILL: SKILL_02 lines 791-803 (generic timer mention)
- Production: \`systemctl list-timers | grep energy\`

## Action Required
- [ ] Add complete timer reference to SKILL_02
- [ ] Document each timer's interval and purpose
- [ ] Update after disabling TenneT timer

## Priority
LOW" \
  --label "gap-audit,documentation"
```

---

### Issue 11: Missing comprehensive API reference documentation

```bash
gh issue create \
  --title "[GAP] Missing comprehensive API reference documentation" \
  --body "## Context
Gevonden tijdens Product Reality Check 2026-01-08

## Gap
No \`api-reference.md\` file exists. Endpoints scattered across SKILLs.
Audit found 16 endpoints, only 11 documented.

## Source
- SKILL: Endpoints mentioned in SKILL_02, SKILL_04, but no central reference
- Code: 16 endpoints found via grep
- Expected: \`docs/api-reference.md\` does not exist

## Action Required
- [ ] Create \`docs/api-reference.md\`
- [ ] Document all 16 endpoints with:
  - Method (GET/POST)
  - URL
  - Query/body parameters
  - Response format
  - Example request/response
  - Authentication requirements
- [ ] Group by category (data, cache, auth, admin, health)
- [ ] Include rate limits per endpoint
- [ ] Link from README.md

## Priority
LOW (but improves discoverability)" \
  --label "gap-audit,documentation,enhancement"
```

---

## BULK CREATION SCRIPT

To create all issues at once:

```bash
#!/bin/bash
# save as create_gap_issues.sh

set -e

echo "Creating 11 gap audit issues..."

# Issue 1 - Database not updating
gh issue create --title "[GAP] Database not updating - stale data since 2026-01-05" --body "..." --label "gap-audit,bug,priority:high"

# Issue 2 - TenneT service failing
gh issue create --title "[GAP] TenneT service failing continuously - should be disabled" --body "..." --label "gap-audit,docs-code-mismatch,priority:high"

# ... (copy all commands above)

echo "All 11 issues created successfully!"
gh issue list --label "gap-audit"
```

---

## VERIFICATION

After creating issues:

```bash
# List all gap audit issues
gh issue list --label "gap-audit"

# Count by priority
gh issue list --label "gap-audit,priority:high" --json number | jq 'length'
gh issue list --label "gap-audit" --label "documentation" --json number | jq 'length'
```

---

## NOTES

- All issues include context from PRODUCT_REALITY_CHECK.md
- All reference specific SKILL locations (file + line numbers)
- All have actionable checklist items
- Labels: `gap-audit` (all), `docs-code-mismatch` (mismatch), `bug` (broken), `documentation` (docs), `priority:high` (critical)
- Total: 11 issues (3 high, 5 medium, 3 low)

---

*Generated: 2026-01-08*
*Source: Product Reality Check Audit*
