# CC TASK 10: Legacy Paths & Template Mismatch Issue

**Date:** 2026-01-06
**Status:** INVESTIGATION COMPLETE - SOLUTION READY
**Priority:** 🔴 CRITICAL (6 systemd services failing)
**Author:** Claude Code + Claude AI Consultation

---

## EXECUTIVE SUMMARY

The Energy Insights NL system has a critical infrastructure gap between repository design (templates) and actual deployments (missing scripts). This causes 6 systemd services to fail with `203/EXEC` errors (script not found).

**Impact:**
- ❌ Importers not running → data collection stalled
- ❌ Normalizers not running → data processing stalled
- ❌ Health checks failing → monitoring blind spots
- ✅ API still running (only critical service working)

**Root Cause:** Templates in `/systemd/scripts/` are never expanded to `/scripts/` during deployment

**Solution:** Implement template generation in deployment pipeline + update documentation

**Estimated Fix Time:** 30 minutes (immediate manual fix) → 2-4 hours (complete implementation)

---

## PROBLEM DETAILS

### The Legacy Path Problem

**Before (deprecated):**
```
/opt/synctacles/
├── app/
├── venv/
└── logs/
```

**After (current):**
```
/opt/energy-insights-nl/
├── app/
├── venv/
└── (logs at /var/log/energy-insights-nl/)
```

**Issue:** Scripts contain hardcoded legacy paths that no longer exist

**Example:**
```bash
# OLD (broken):
LOG_DIR="/opt/synctacles/logs/scheduler"
psql -U synctacles -d synctacles

# NEW (correct):
LOG_DIR="${LOG_PATH:-/var/log/energy-insights-nl}/scheduler"
psql -U energy_insights_nl -d energy_insights_nl
```

**Mitigation Added:** Pre-commit hook now blocks new commits with legacy paths

---

### The Template vs Script Mismatch

#### Repository Structure

```
/opt/github/synctacles-api/
│
├── scripts/                                    ← Direct scripts
│   ├── run_collectors.sh                       ✅ Direct (no template)
│   ├── run_normalizers.sh                      ✅ Direct (no template)
│   ├── health_check.sh                         ✅ Direct (FIXED)
│   └── [others]
│
└── systemd/scripts/                            ← Templates (NEVER EXPANDED)
    ├── run_collectors.sh.template              ⚠️ Has {{PLACEHOLDERS}}
    ├── run_importers.sh.template               ❌ Never generated
    ├── run_normalizers.sh.template             ❌ Never generated
    └── health_check.sh.template                ⚠️ Has {{PLACEHOLDERS}}
```

#### Current Deployment Flow

```
1. git pull origin main
   ↓
2. rsync /opt/github/synctacles-api/scripts/ → /opt/energy-insights-nl/app/scripts/
   ├── ✅ run_collectors.sh (exists, copied)
   ├── ✅ run_normalizers.sh (exists, copied)
   ├── ✅ health_check.sh (exists, copied)
   └── ❌ run_importers.sh (MISSING - only template exists!)
   ↓
3. systemctl restart services
   ├── ✅ health service (now works after rsync)
   ├── ❌ importer service (203/EXEC - script not found)
   ├── ❌ normalizer service (partially - depends on importer)
   └── ❌ tennet service (401 auth error - separate issue)
```

#### The Missing Link

**File:** `/opt/github/synctacles-api/scripts/setup/setup_synctacles_server_v2.3.4.sh` (lines 1851-1906)

This script HAS template expansion logic:

```bash
generate_script() {
    sed -e "s|{{INSTALL_PATH}}|${INSTALL_PATH}|g" \
        -e "s|{{LOG_PATH}}|${LOG_PATH}|g" \
        -e "s|{{ENV_FILE}}|/opt/.env|g" \
        "$template" > "$output"
    chmod +x "$output"
}
```

**But:** This is in an old installer script (FASE 2.3.4) that's DEPRECATED and never called during regular deployments

**Result:** Template expansion only works for NEW clean installations, not for production deployments

---

## FAILED SERVICES ANALYSIS

### Services Status

```
systemctl list-units --type=service --state=failed

❌ energy-insights-nl-importer.service
   └─ Error: 203/EXEC (script not found: run_importers.sh)

❌ energy-insights-nl-normalizer.service
   └─ Error: 203/EXEC (script not found: run_normalizers.sh)

❌ synctacles-importer.service
   └─ Error: 203/EXEC (script not found: run_importers.sh)

❌ synctacles-normalizer.service
   └─ Error: 203/EXEC (script not found: run_normalizers.sh)

❌ energy-insights-nl-tennet.service
   └─ Error: 401 Authentication Failed (separate issue - BYO-KEY only)

❌ energy-insights-nl-health.service
   └─ Error: [FIXED after rsync] scripts copied, now working

✅ synctacles-health.service
   └─ Status: WORKING (health_check.sh now copied)

✅ energy-insights-nl-api.service
   └─ Status: RUNNING (Gunicorn, port 8000)
```

### Why 203/EXEC Happens

```bash
# Systemd service definition expects:
ExecStart=/opt/energy-insights-nl/app/scripts/run_importers.sh

# But:
ls /opt/energy-insights-nl/app/scripts/run_importers.sh
# → File not found

# Because:
ls /opt/github/synctacles-api/systemd/scripts/run_importers.sh.template
# → Only template exists, never expanded
```

---

## ROOT CAUSE ANALYSIS

### Why This Happened

#### 1. **Two Different Script Locations (Design Debt)**

- **`/scripts/`**: Pre-written scripts with hardcoded defaults
  - Used to work before migration from `/opt/synctacles` to `/opt/energy-insights-nl`
  - Now outdated but partially copied by rsync

- **`/systemd/scripts/`**: Templates for flexible deployment
  - Designed for FASE 5 (new installations)
  - Contains placeholders for environment-specific values
  - Never processed during regular deployments

#### 2. **Incomplete Deployment Procedure**

Documentation (DEPLOYMENT_SCRIPTS_GUIDE.md) says:

```bash
sudo rsync -av \
    /opt/github/synctacles-api/scripts/ \
    /opt/energy-insights-nl/app/scripts/
```

But missing step:

```bash
# THIS STEP IS MISSING:
bash /opt/github/synctacles-api/scripts/generate-templates.sh /opt/energy-insights-nl
```

#### 3. **Setup Script Isolation**

- Setup script has template expansion logic
- **But:** Only called during FASE 2-5 fresh installations
- **Not called:** During regular deployments (git pull → rsync → restart)
- No bridge between new-install logic and regular-deploy procedure

#### 4. **Pre-commit Hook Success**

New pre-commit hook successfully blocks legacy paths from being committed:

```bash
✗ BLOCKED: Legacy /opt/synctacles path in FILE
```

**Good Side Effect:** Prevents future legacy path commits

**Side Effect:** Old test/deprecated scripts still reference legacy paths but can't be re-committed

---

## SOLUTIONS

### IMMEDIATE FIX (5 minutes)

Manually expand templates for current system:

```bash
#!/bin/bash
# Immediate fix for missing run_importers.sh and run_normalizers.sh

REPO="/opt/github/synctacles-api"
INSTALL_PATH="/opt/energy-insights-nl"
LOG_PATH="/var/log/energy-insights-nl"
ENV_FILE="/opt/.env"

# Generate run_importers.sh
sed -e "s|{{INSTALL_PATH}}|${INSTALL_PATH}|g" \
    -e "s|{{LOG_PATH}}|${LOG_PATH}|g" \
    -e "s|{{ENV_FILE}}|${ENV_FILE}|g" \
    "${REPO}/systemd/scripts/run_importers.sh.template" \
    > "${INSTALL_PATH}/app/scripts/run_importers.sh"

# Generate run_normalizers.sh
sed -e "s|{{INSTALL_PATH}}|${INSTALL_PATH}|g" \
    -e "s|{{LOG_PATH}}|${LOG_PATH}|g" \
    -e "s|{{ENV_FILE}}|${ENV_FILE}|g" \
    "${REPO}/systemd/scripts/run_normalizers.sh.template" \
    > "${INSTALL_PATH}/app/scripts/run_normalizers.sh"

# Fix permissions
chmod +x "${INSTALL_PATH}/app/scripts/run_importers.sh"
chmod +x "${INSTALL_PATH}/app/scripts/run_normalizers.sh"
chown energy-insights-nl:energy-insights-nl \
    "${INSTALL_PATH}/app/scripts/run_importers.sh" \
    "${INSTALL_PATH}/app/scripts/run_normalizers.sh"

# Verify
echo "✓ run_importers.sh generated"
echo "✓ run_normalizers.sh generated"
```

**Result:** Fixes 4 services, unblocks data collection/processing

---

### SHORT-TERM FIX (1-2 hours)

#### Step 1: Create Template Generation Script

**File:** `scripts/generate-templates.sh`

```bash
#!/usr/bin/env bash
# Generate all scripts from templates
# Usage: ./generate-templates.sh [INSTALL_PATH]

set -euo pipefail

INSTALL_PATH="${1:-/opt/energy-insights-nl}"
LOG_PATH="/var/log/energy-insights-nl"
ENV_FILE="/opt/.env"
SERVICE_USER="energy-insights-nl"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

generate_script() {
    local template="$1"
    local output="$2"
    local name=$(basename "$output")

    echo "[*] Generating: $name"

    sed -e "s|{{INSTALL_PATH}}|${INSTALL_PATH}|g" \
        -e "s|{{LOG_PATH}}|${LOG_PATH}|g" \
        -e "s|{{ENV_FILE}}|${ENV_FILE}|g" \
        -e "s|{{SERVICE_USER}}|${SERVICE_USER}|g" \
        "$template" > "$output"

    chmod +x "$output"

    # Verify no placeholders
    if grep -q "{{" "$output"; then
        echo "[✗] FAILED: Unresolved placeholders in $name"
        grep "{{" "$output" | head -3
        exit 1
    fi

    echo "[✓] $name generated successfully"
}

# Generate all scripts
generate_script "${REPO_ROOT}/systemd/scripts/run_importers.sh.template" \
                "${REPO_ROOT}/scripts/run_importers.sh"
generate_script "${REPO_ROOT}/systemd/scripts/run_normalizers.sh.template" \
                "${REPO_ROOT}/scripts/run_normalizers.sh"
generate_script "${REPO_ROOT}/systemd/scripts/health_check.sh.template" \
                "${REPO_ROOT}/scripts/health_check.sh"
generate_script "${REPO_ROOT}/systemd/scripts/run_collectors.sh.template" \
                "${REPO_ROOT}/scripts/run_collectors.sh"

echo ""
echo "[✓] All scripts generated successfully"
```

#### Step 2: Update Deployment Script

Add to `/opt/github/synctacles-api/scripts/deploy/deploy.sh`:

```bash
# After git pull, before rsync:
echo "[*] Generating scripts from templates..."
cd "$REPO_DIR"
bash ./scripts/generate-templates.sh "$INSTALL_PATH"

# Then sync as normal
echo "[*] Syncing to production..."
rsync -av \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    /opt/github/synctacles-api/scripts/ \
    /opt/energy-insights-nl/app/scripts/
```

#### Step 3: Add Deployment Validation

**File:** `scripts/deploy/validate-deployment.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${1:-/opt/energy-insights-nl/app}"

REQUIRED_SCRIPTS=(
    "scripts/run_importers.sh"
    "scripts/run_normalizers.sh"
    "scripts/run_collectors.sh"
    "scripts/health_check.sh"
)

echo "[*] Validating deployment at $APP_DIR..."
FAILED=0

for script in "${REQUIRED_SCRIPTS[@]}"; do
    path="$APP_DIR/$script"
    name=$(basename "$script")

    if [[ ! -f "$path" ]]; then
        echo "[✗] MISSING: $name"
        FAILED=$((FAILED + 1))
    elif [[ ! -x "$path" ]]; then
        echo "[✗] NOT EXECUTABLE: $name"
        FAILED=$((FAILED + 1))
    elif grep -q "{{" "$path"; then
        echo "[✗] UNRESOLVED PLACEHOLDERS: $name"
        grep "{{" "$path" | head -3
        FAILED=$((FAILED + 1))
    else
        echo "[✓] $name"
    fi
done

if [[ $FAILED -gt 0 ]]; then
    echo ""
    echo "[✗] Validation FAILED: $FAILED script(s) have issues"
    exit 1
fi

echo "[✓] All scripts validated successfully"
```

---

### LONG-TERM FIX (2-4 hours + ongoing)

#### Architecture Decision

**Single Source of Truth: `/systemd/scripts/` templates**

```
/systemd/scripts/           ← Authoritative templates
├── run_importers.sh.template
├── run_normalizers.sh.template
├── health_check.sh.template
└── run_collectors.sh.template
    │
    ├─→ [CI/CD: Generate + Test]
    │
    └─→ /scripts/            ← Pre-generated, version-controlled
        ├── run_importers.sh
        ├── run_normalizers.sh
        ├── health_check.sh
        └── run_collectors.sh
            │
            └─→ [Deploy: Simple rsync, no processing]
                │
                └─→ /opt/energy-insights-nl/app/scripts/
                    (Production runtime)
```

#### Implementation

1. **Git**: Store templates in `/systemd/scripts/`
2. **Build**: Generate scripts with CI/CD pipeline
3. **Repository**: Commit generated scripts to `/scripts/` (version-controlled)
4. **Deploy**: Simple rsync (no processing needed)
5. **Validate**: Pre-deployment script checks all required files exist

#### CI/CD Integration

**GitHub Actions workflow:**

```yaml
name: Generate Scripts from Templates

on:
  pull_request:
  push:
    branches: [main]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Generate scripts from templates
        run: bash scripts/generate-templates.sh /opt/energy-insights-nl

      - name: Validate generated scripts
        run: bash scripts/deploy/validate-deployment.sh /opt/energy-insights-nl

      - name: Commit generated scripts (if changed)
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add scripts/run_*.sh
          git commit -m "ci: regenerate scripts from templates" || true
          git push
```

---

## PREVENTION MEASURES

### 1. Pre-commit Hook Enhancement

Add to `.git/hooks/pre-commit`:

```bash
# Verify templates have corresponding generated scripts
echo "[*] Checking template completeness..."
for template in systemd/scripts/*.template; do
    script_name=$(basename "$template" .template)
    script_path="scripts/${script_name}"

    if [[ ! -f "$script_path" ]]; then
        echo "[✗] Missing generated script: $script_path"
        echo "    Template: $template"
        echo "    Run: bash scripts/generate-templates.sh"
        BLOCKED=1
    fi
done

if [[ $BLOCKED -eq 1 ]]; then
    exit 1
fi
```

### 2. Service Template Validation

Update systemd service templates with pre-checks:

```ini
[Unit]
Description=Energy Insights NL Data Importer
After=synctacles-collector.service

[Service]
Type=oneshot
User=energy-insights-nl
WorkingDirectory={{INSTALL_PATH}}/app
EnvironmentFile={{ENV_FILE}}

# Validate script exists before running
ExecStartPre=-/bin/bash -c 'test -x {{INSTALL_PATH}}/app/scripts/run_importers.sh || \
    (echo "ERROR: run_importers.sh not found or not executable"; exit 1)'

ExecStart={{INSTALL_PATH}}/app/scripts/run_importers.sh

StandardOutput=journal
StandardError=journal
```

### 3. Documentation Updates

**Single deployment procedure document** (not scattered across multiple files):

Create: `docs/operations/DEPLOYMENT_GUIDE.md`

Contains:
- Pre-deployment checklist
- Step-by-step procedure
- Validation commands
- Troubleshooting section
- Rollback procedures

### 4. Test Coverage

```bash
# Unit test for template expansion
test_template_generation() {
    local template="systemd/scripts/run_importers.sh.template"
    local output=$(mktemp)

    sed -e 's|{{INSTALL_PATH}}|/test/path|g' \
        -e 's|{{LOG_PATH}}|/test/logs|g' \
        -e 's|{{ENV_FILE}}|/test/.env|g' \
        "$template" > "$output"

    # Check no unresolved placeholders
    if grep -q "{{" "$output"; then
        echo "FAIL: Unresolved placeholders in expanded script"
        return 1
    fi

    echo "PASS: Template expansion works correctly"
    rm "$output"
}
```

---

## TIMELINE & PRIORITY

| Task | Time | Priority | Status |
|------|------|----------|--------|
| **Immediate:** Manual template expansion | 5 min | 🔴 CRITICAL | Ready to execute |
| **Quick:** Create generate-templates.sh | 20 min | 🔴 CRITICAL | Code ready |
| **Quick:** Update deployment script | 15 min | 🔴 CRITICAL | Code ready |
| **Quick:** Add validation script | 15 min | 🟠 HIGH | Code ready |
| **Short-term:** CI/CD automation | 1-2 hours | 🟠 HIGH | Design ready |
| **Medium-term:** Document update | 30 min | 🟠 HIGH | Content ready |
| **Ongoing:** Test coverage | 1-2 hours | 🟡 MEDIUM | Framework ready |

---

## NEXT STEPS

1. **User Decision Required:**
   - Execute immediate fix? (5 minutes, unblocks services)
   - Proceed with short-term implementation? (2-4 hours, permanent solution)
   - Or wait for long-term CI/CD integration? (1-2 days, enterprise-grade)

2. **If Proceeding:**
   - Run immediate fix script
   - Verify services start successfully
   - Create/test generate-templates.sh
   - Update deployment procedure
   - Add CI/CD automation

3. **After Deployment:**
   - Update GitHub issue #25 (Monitoring Phase 1)
   - Document lessons learned
   - Archive this analysis as reference

---

## REFERENCES

- DEPLOYMENT_SCRIPTS_GUIDE.md
- setup_synctacles_server_v2.3.4.sh (lines 1851-1906)
- SKILL_09 (deployment standards)
- Pre-commit hook enhancements (legacy path blocking)

