# Release Checklist

## Pre-Release Security Gate

Voordat ELKE release naar productie:

### Automated (moet PASS zijn)

- [ ] `bandit -ll -r synctacles_db/` - 0 HIGH/MEDIUM findings
- [ ] `pip-audit -r requirements.txt` - 0 HIGH/CRITICAL vulnerabilities  
- [ ] `safety check --file requirements.txt` - 0 vulnerabilities
- [ ] All tests pass: `pytest tests/ -v`
- [ ] Coverage ≥70%: `pytest --cov=synctacles_db --cov-fail-under=70`

### Manual Review

- [ ] Geen nieuwe SQL queries met string concatenation (f-strings)
- [ ] Geen nieuwe hardcoded secrets
- [ ] Geen nieuwe file operations zonder path validation
- [ ] CHANGELOG.md updated
- [ ] Version bumped in `__init__.py` and `pyproject.toml`
- [ ] All GitHub issues closed or deferred
- [ ] Documentation updated

### Deployment

- [ ] Git tag created: `git tag -a vX.X.X -m "Release X.X.X"`
- [ ] CI/CD pipeline passed (GitHub Actions green)
- [ ] Deployed to staging first (`ssh synct-dev`)
- [ ] Smoke test passed on staging
- [ ] Production deploy: `~/bin/deploy-prod`
- [ ] Health check passed: `curl https://api.synctacles.com/health`
- [ ] Monitoring dashboards checked (Grafana)

### Rollback Ready

- [ ] Previous version noted: vX.X.X
- [ ] Rollback command ready: `git checkout vX.X.X && ~/bin/deploy-prod`
- [ ] Database migrations are reversible (if applicable)
- [ ] Backup verified before deploy

---

## Security Review Criteria

### Code Changes

**SQL queries:**
- ✅ All queries use parameterized statements
- ✅ No f-strings in `execute()` calls
- ✅ Table/column names validated against whitelist if dynamic

**Secrets:**
- ✅ All secrets via environment variables
- ✅ No hardcoded API keys, passwords, tokens
- ✅ `.env` files in `.gitignore`

**Input validation:**
- ✅ All external input validated
- ✅ Path traversal checks on file operations
- ✅ No `eval()`, `exec()`, or `pickle.loads()` on untrusted data

**Dependencies:**
- ✅ All versions pinned in `requirements.txt`
- ✅ No known vulnerabilities (pip-audit, safety)
- ✅ License compatibility checked

---

## Post-Release

### Immediate (within 1 hour)

- [ ] Monitor error logs: `journalctl -u synctacles-api -f`
- [ ] Check health endpoint: `curl https://api.synctacles.com/health`
- [ ] Verify key endpoints work:
  - `/api/v1/prices/today`
  - `/api/v1/prices/tomorrow`
  - `/api/v1/health`
- [ ] Check Grafana dashboards for anomalies

### Within 24 hours

- [ ] Review error rates in logs
- [ ] Check API response times
- [ ] Verify data collection (no gaps in time series)
- [ ] User feedback review (if applicable)

### Within 1 week

- [ ] Performance analysis (any degradation?)
- [ ] Resource usage check (memory leaks?)
- [ ] Backup verification
- [ ] Update runbook if needed

---

## Emergency Rollback Procedure

If critical issue discovered:

1. **STOP** - Assess severity (data loss? security breach?)
2. **COMMUNICATE** - Notify Leo immediately
3. **ROLLBACK**:
   ```bash
   git checkout v<PREVIOUS_VERSION>
   ~/bin/deploy-prod
   ```
4. **VERIFY** - Check health endpoint, logs
5. **POSTMORTEM** - Document what went wrong, how it slipped through

---

## Release Types

### Patch (vX.X.Y)
- Bug fixes only
- Security patches
- No new features
- Minimal review needed

### Minor (vX.Y.0)
- New features
- Non-breaking changes
- Full security review
- Staging deployment required

### Major (vX.0.0)
- Breaking changes
- Architecture changes
- Extended testing period
- Deployment plan required
- Rollback strategy mandatory

---

*This checklist is MANDATORY for all production releases.*  
*Skipping steps = production incident risk.*

**Version:** 1.0  
**Last Updated:** 2026-01-26
