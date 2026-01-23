# CLAUDE.md - Project Context for Claude Code

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
~/bin/deploy-prod         # Then deploy to PROD
~/bin/prod-status         # Verify deployment
```

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
