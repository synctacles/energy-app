# SKILL 10 — DEPLOYMENT WORKFLOW

Deployment Strategy for SYNCTACLES
Version: 1.0 (2025-12-21)

---

## GOAL

Formalize deployment from DEV → PROD with:
- Traceability (which files, when, by whom)
- Rollback capability (previous version restore)
- Validation (pre/post deploy checks)
- Automation (reproducible process)

---

## DIRECTORY STRUCTURE

Repository (DEV - Git Source of Truth):
```
/opt/github/synctacles-repo/
├── deployment/
│   ├── sync/                       ← Staged changes per feature
│   │   ├── f8.3-fallback/
│   │   ├── f8.4-license/
│   │   └── f8.5-usermngmt/
│   ├── deploy.sh                   ← Master deployment script
│   ├── rollback.sh                 ← Rollback to previous version
│   ├── sync-manifest.txt           ← Declarative file mappings
│   └── pre-deploy-checks.sh        ← Validation before deploy
```

Production (PROD - Runtime):
```
{{INSTALL_PATH}}/app/              ← Deployed code
├── synctacles_db/                 ← Deployed code
├── alembic/
└── VERSION                         ← Current production version

{{INSTALL_PATH}}/backups/deployment/ ← Deployment backups
└── YYYYMMDD-HHMMSS/                ← Timestamped snapshots
```

---

## DEPLOYMENT WORKFLOW

### FASE 1: Pre-Deploy Validation

Run `./deployment/pre-deploy-checks.sh` to verify:
- ✓ Git status clean (no uncommitted changes)
- ✓ All tests passing
- ✓ Database migrations exist
- ✓ VERSION file updated
- ✓ No merge conflicts

### FASE 2: Backup Current Production

```bash
mkdir -p {{INSTALL_PATH}}/backups/deployment/$(date +%Y%m%d-%H%M%S)
rsync -a {{INSTALL_PATH}}/app/ {{INSTALL_PATH}}/backups/deployment/.../
```

### FASE 3: Sync Files (via manifest)

Per sync-manifest.txt:
```
synctacles_db/              → {{INSTALL_PATH}}/app/synctacles_db/
alembic/versions/           → {{INSTALL_PATH}}/app/alembic/versions/
config/                     → {{INSTALL_PATH}}/app/config/
scripts/                    → {{INSTALL_PATH}}/scripts/
systemd/*.service           → /etc/systemd/system/
systemd/*.timer             → /etc/systemd/system/
```

EXCLUDE:
```
.env, venv/, __pycache__/, logs/, .git/
```

### FASE 4: Post-Sync Actions

1. **Database migrations:**
   ```bash
   cd {{INSTALL_PATH}}/app
   alembic upgrade head
   ```

2. **Systemd reload:**
   ```bash
   systemctl daemon-reload
   ```

3. **Service restart:**
   ```bash
   systemctl restart {{SERVICE_USER}}-api
   ```

4. **Permissions:**
   ```bash
   chown -R {{SERVICE_USER}}:{{SERVICE_GROUP}} {{INSTALL_PATH}}/app
   ```

### FASE 5: Validation

```bash
curl http://localhost:8000/health
  ✓ Status: ok
  ✓ Version matches deployed VERSION file

systemctl list-timers {{SERVICE_USER}}-*
  ✓ Expected timer count (≥3)

psql -U {{DB_USER}} -d {{DB_NAME}} -c "SELECT 1"
  ✓ Database connectivity
```

### FASE 6: Rollback (if validation fails)

```bash
./deployment/rollback.sh
  → Restore from backup
  → Alembic downgrade (if needed)
  → Service restart
  → Re-validate
```

---

## SYNC MANIFEST FORMAT

**Format:** SOURCE_PATH → DESTINATION_PATH

```
# Core application
synctacles_db/ → {{INSTALL_PATH}}/app/synctacles_db/

# Database
alembic/versions/ → {{INSTALL_PATH}}/app/alembic/versions/

# Configuration
config/ → {{INSTALL_PATH}}/app/config/

# Scripts (runtime)
scripts/ → {{INSTALL_PATH}}/scripts/

# Systemd units
systemd/*.service → /etc/systemd/system/
systemd/*.timer → /etc/systemd/system/

# Version tracking
VERSION → {{INSTALL_PATH}}/app/VERSION

# EXCLUDE (never sync)
.env
venv/
__pycache__/
*.pyc
logs/
.git/
```

---

## DEPLOY.SH USAGE

**Basic deployment:**
```bash
./deployment/deploy.sh
```

**With version tag:**
```bash
./deployment/deploy.sh v1.0.1
```

**Dry-run (show what would be synced):**
```bash
./deployment/deploy.sh --dry-run
```

**Skip validation (dangerous):**
```bash
./deployment/deploy.sh --skip-checks
```

---

## ROLLBACK.SH USAGE

**Rollback to most recent backup:**
```bash
./deployment/rollback.sh
```

**Rollback to specific backup:**
```bash
./deployment/rollback.sh 20251220-153045
```

**List available backups:**
```bash
./deployment/rollback.sh --list
```

---

## FEATURE DEPLOYMENT WORKFLOW

Example workflow for deploying features:

1. **Development (local changes)**
   ```
   ✓ Code written
   ✓ Tested locally
   ```

2. **Sync to staging directory:**
   ```bash
   mkdir -p deployment/sync/f8.5-usermngmt/
   cp files → deployment/sync/f8.5-usermngmt/
   Create DEPLOY.md (manifest per feature)
   ```

3. **Commit to git:**
   ```bash
   cd /opt/github/synctacles-repo
   git add deployment/sync/f8.5-usermngmt/
   git commit -m "ADD: F8.5 user management"
   git push origin main
   ```

4. **Deploy to production:**
   ```bash
   ./deployment/deploy.sh
   → Reads sync-manifest.txt
   → Syncs files to {{INSTALL_PATH}}/app/
   → Runs migrations
   → Restarts services
   ```

5. **Validate:**
   ```bash
   curl http://localhost:8000/auth/signup
   ✓ Working
   ```

---

## VERSION TRACKING

**VERSION file format (repo root):**
```
1.0.1
```

**Increment rules:**
- MAJOR: Breaking changes (API contract)
- MINOR: New features (backward compatible)
- PATCH: Bugfixes

**Deployment updates VERSION:**
1. Update VERSION file in repo
2. `git tag v1.0.1`
3. `git push --tags`
4. `./deployment/deploy.sh v1.0.1`
5. Verify: `curl http://localhost:8000/health | jq .version`

---

## EMERGENCY PROCEDURES

### API Down After Deploy
1. `./deployment/rollback.sh`
2. Check logs: `journalctl -u {{SERVICE_USER}}-api -n 100`
3. Verify .env not corrupted
4. Check database connectivity

### Database Migration Failed
1. `alembic downgrade -1`
2. Fix migration script
3. `alembic upgrade head`

### Service Won't Start
1. Check systemd unit paths ({{INSTALL_PATH}}/app not /opt/github/)
2. Verify EnvironmentFile=/opt/.env
3. Check permissions ({{SERVICE_USER}}:{{SERVICE_GROUP}})

---

## PRE-DEPLOY CHECKLIST

### Development
- □ All tests pass
- □ Local testing complete
- □ Changes committed to git
- □ VERSION file updated (if needed)

### Staging
- □ Files in deployment/sync/<feature>/
- □ DEPLOY.md created
- □ Git pushed to origin

### Production
- □ Pre-deploy checks pass
- □ Backup created
- □ .env not overwritten
- □ Migrations tested

### Post-Deploy
- □ Health check OK
- □ Services running
- □ Timers active
- □ Database accessible

---

## FUTURE ENHANCEMENTS

### V1.1
- GitHub Actions trigger
- Automated testing
- Slack notifications

### V2 (Multi-server)
- Blue-green deployment
- Canary releases
- Auto-rollback on error

---

## RELATED SKILLS

- SKILL 9: Installer Specs (with brand-aware .env)
- SKILL 12: Brand-Free Template Architecture (ENV-driven design)
