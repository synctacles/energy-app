# CLAUDE.md - Project Context for Claude Code

## ⚠️ SCOPE WARNING

**Dit is de ENERGY API repo. CARE/Moltbot werk hoort hier NIET.**

| Project | Juiste locatie |
|---------|----------------|
| CARE code | `/opt/synctacles/moltbot/` |
| KB/Support | `/opt/synctacles/moltbot/` |
| Deming Cycle | `/opt/synctacles/moltbot/` |
| Claude SDK | `/opt/synctacles/moltbot/shared/` |
| Moltbot issues | `synctacles/moltbot` repo |

**Deze repo is ALLEEN voor:**
- Energy API endpoints
- Price collectors (TenneT, Frank, ENTSO-E)
- HA integration backend

---

## Error Handling Protocol

**CRITICAL: When errors are detected, follow this protocol:**

1. **Immediate Investigation**
   - Stop current work and investigate the error immediately
   - Identify root cause and impact
   - Document findings

2. **Priority Assessment**
   - **High Priority** (blocking, security, data loss): Fix immediately
   - **Medium Priority** (bugs, broken features): Create GitHub issue in this repo
   - **Low Priority** (nice-to-have, optimization): Create GitHub issue with "enhancement" label

3. **Issue Creation Template**
   ```bash
   gh issue create --repo synctacles/platform \
     --title "Error: [Brief description]" \
     --body "## Problem
   [Error description]

   ## Context
   [When/where detected]

   ## Investigation
   [Root cause analysis]

   ## Impact
   [What's affected]

   ## Suggested Fix
   [Potential solution]" \
     --label "bug"
   ```

4. **Never Ignore Errors**
   - All errors must be either fixed immediately or tracked in GitHub
   - Silent failures are unacceptable
   - Always inform user of error and tracking status

---

## MANDATORY: Read SKILLs First

**Before ANY action, read these SKILLs:**
| SKILL | File | Purpose |
|-------|------|---------|
| **SKILL 00** | `docs/skills/SKILL_00_AI_OPERATING_PROTOCOL.md` | Operating protocol (VERPLICHT) |
| **SKILL 01** | `docs/skills/SKILL_01_HARD_RULES.md` | Non-negotiable rules |
| **SKILL 02** | `docs/skills/SKILL_02_ARCHITECTURE.md` | System architecture |
| **SKILL 11** | `docs/skills/SKILL_11_REPO_AND_ACCOUNTS.md` | Git workflow, accounts |

**For infrastructure work:**
- `docs/CREDENTIALS.md` - Server access, SSH keys, deployment commands

## Project Overview
SYNCTACLES - Energy price API serving real-time and day-ahead electricity prices for the Netherlands.

## Infrastructure

### Servers
| Server | Purpose | Access |
|--------|---------|--------|
| DEV | Development (this machine) | Direct |
| PROD | Production (46.62.212.227) | Via `ssh cc-hub "ssh synct-prod '...'"` |

### GitHub Account
- **Bot account**: `synctacles-bot`
- **Repository**: `synctacles/backend`
- **Authentication**: PAT token (configured in gh CLI)

### Quick Commands

**Deploy to PROD (from DEV):**
```bash
git push origin main      # Push code first
~/bin/deploy-prod         # Then deploy to PROD (checks CI first!)
~/bin/prod-status         # Verify deployment
```

## Development Workflow

### Pre-commit Hooks
Located in `.git/hooks/pre-commit`. Runs automatically on every commit:

1. **Credentials check** - Blocks hardcoded passwords/secrets
2. **Ruff format** - Auto-fixes Python formatting
3. **Ruff check** - Auto-fixes linting errors (blocks unfixable ones)

Files are automatically reformatted and re-staged before commit.

### CI Pipeline (GitHub Actions)
- Runs on every push to `main`
- Checks: ruff format, ruff check, pytest, build validation
- Must pass before deployment

### Deploy Safety
`~/bin/deploy-prod` automatically:
1. Checks if CI passed for current commit
2. Waits if CI is still running (max 5 min)
3. Blocks deployment if CI failed
4. Only deploys after CI success

**Check PROD status:**
```bash
ssh cc-hub "ssh synct-prod 'systemctl status synctacles-api'"
```

**Check PROD logs:**
```bash
ssh cc-hub "ssh synct-prod 'journalctl -u synctacles-api -n 50'"
```

## Brand-Free Architecture
All scripts and templates use environment variables from `/opt/.env`:
- `BRAND_NAME`, `BRAND_SLUG` - Brand identity
- `DB_NAME`, `DB_USER` - Database credentials
- `APP_PATH`, `INSTALL_PATH` - Paths
- Templates use `{{PLACEHOLDER}}` syntax

## Key Directories
```
/opt/github/synctacles-api/     # Application code
/opt/synctacles/                 # Runtime (venv, logs, backups)
/opt/.env                        # Environment configuration
```

## Services (PROD)
- `synctacles-api` - Main FastAPI server (8 Gunicorn workers)
- `synctacles-collector` - Data collection (timer)
- `synctacles-importer` - Data import (timer)
- `synctacles-normalizer` - Data normalization (timer)
- `synctacles-health` - Health checks (timer)
- `synctacles-frank-collector` - Frank Energie data (timer)
