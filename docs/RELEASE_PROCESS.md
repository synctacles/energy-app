# Release Process

Last updated: 2026-01-24
Owner: @synctacles-bot
Review cycle: Bij elke major release of kwartaal

## Current Phase: Beta

**Simplified process voor beta fase:**
- Push to `main` → CI runs → `deploy-prod` deployt
- Geen formele version tags (nog)
- Geen CHANGELOG updates (nog)

**Na beta launch:** Formeel versioning activeren (zie sectie hieronder).

## Overview

SYNCTACLES uses a simple release process:
- `main` branch = production-ready code
- CI/CD via GitHub Actions
- Manual deployment to PROD via `deploy-prod` script

## Version Numbering

We use [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH
  │     │     └── Bug fixes, no API changes
  │     └──────── New features, backwards compatible
  └────────────── Breaking changes
```

Current version: see `VERSION` file in repo root.

## Release Checklist

### 1. Pre-Release

```bash
# Ensure on main branch
git checkout main
git pull origin main

# Run tests
pytest tests/ -v

# Check for uncommitted changes
git status
```

### 2. Update Version

```bash
# Update VERSION file
echo "1.1.0" > VERSION

# Update CHANGELOG.md
# Move [Unreleased] items to new version section
```

### 3. Commit & Tag

```bash
git add VERSION CHANGELOG.md
git commit -m "release: v1.1.0"
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin main --tags
```

### 4. Wait for CI

GitHub Actions runs automatically on push:
- Linting
- Tests
- Build validation

Check status: `gh run list --limit 5`

### 5. Deploy to PROD

```bash
# From DEV server
~/bin/deploy-prod
```

The script:
1. Checks CI status (blocks if failed)
2. SSH to PROD via cc-hub
3. Runs `git pull`
4. Restarts services
5. Verifies health endpoint

### 6. Verify Deployment

```bash
# Check PROD status
~/bin/prod-status

# Check API health
curl -s https://api.synctacles.com/health | jq

# Check logs for errors
ssh cc-hub "ssh synct-prod 'journalctl -u synctacles-api -n 20'"
```

## Rollback Procedure

### Quick Rollback (< 5 min)

```bash
# SSH to PROD
ssh cc-hub "ssh synct-prod 'bash'"

# Rollback to previous commit
sudo -u synctacles git -C /opt/github/synctacles-api reset --hard HEAD~1
sudo systemctl restart synctacles-api

# Verify
curl -s https://api.synctacles.com/health
```

### Rollback to Specific Version

```bash
# SSH to PROD
ssh cc-hub "ssh synct-prod 'bash'"

# List recent tags
sudo -u synctacles git -C /opt/github/synctacles-api tag --sort=-creatordate | head -5

# Checkout specific version
sudo -u synctacles git -C /opt/github/synctacles-api checkout v1.0.0
sudo systemctl restart synctacles-api

# Verify
curl -s https://api.synctacles.com/health
```

### Database Rollback

If migrations need reverting:

```bash
# SSH to PROD
ssh cc-hub "ssh synct-prod 'bash'"

# Check current migration
cd /opt/github/synctacles-api
source /opt/synctacles/venv/bin/activate
alembic current

# Rollback one migration
alembic downgrade -1

# Or to specific revision
alembic downgrade abc123
```

## Hotfix Process

For urgent production fixes:

```bash
# Create hotfix branch
git checkout main
git pull
git checkout -b hotfix/fix-critical-bug

# Make fix
# ... edit files ...

# Commit
git add .
git commit -m "fix: critical bug description"

# Push and create PR
git push -u origin hotfix/fix-critical-bug
gh pr create --title "Hotfix: Critical bug" --body "Description of fix"

# After PR approved and merged
git checkout main
git pull
~/bin/deploy-prod
```

## Monitoring After Release

After any release, monitor for 15-30 minutes:

1. **Grafana Dashboard**: https://monitor.synctacles.com/grafana
2. **Error rate**: Should be < 1%
3. **Response time**: p95 < 200ms
4. **Memory usage**: Stable, no growth

## CHANGELOG Format

```markdown
## [1.1.0] - 2026-01-24

### Added
- New feature X

### Changed
- Updated behavior Y

### Fixed
- Bug fix Z

### Removed
- Deprecated feature W
```

## Scripts Reference

| Script | Location | Purpose |
|--------|----------|---------|
| `deploy-prod` | `~/bin/deploy-prod` | Deploy to PROD (checks CI first) |
| `prod-status` | `~/bin/prod-status` | Check PROD service status |

## Emergency Contacts

- **On-call**: Check Slack #synctacles-alerts
- **Escalation**: See CREDENTIALS.md for contacts
