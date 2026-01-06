#!/bin/bash

# Import Tasks to GitHub Issues
# This script reads the task list and creates GitHub Issues
# Requires: GitHub CLI (gh) to be installed and authenticated

set -e

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║  Importing Tasks to GitHub Issues                             ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) not found."
    echo "Install from: https://cli.github.com/"
    exit 1
fi

# Check authentication
if ! gh auth status &> /dev/null; then
    echo "❌ Not authenticated with GitHub"
    echo "Run: gh auth login"
    exit 1
fi

echo "✅ GitHub CLI authenticated"

# Create CRITICAL tasks
echo ""
echo "📝 Creating CRITICAL tasks..."

# Task 1: Fix CORS
gh issue create \
    --title "[CRITICAL] Fix CORS Security Configuration" \
    --label "security,critical,cors" \
    --body '## Description
Replace hardcoded allow_origins=["*"] with environment-configurable setting.

## Current Code
```python
CORSMiddleware(app, allow_origins=["*"])
```

## Solution
```python
allow_origins = settings.cors_allowed_origins or ["https://example.nl"]
CORSMiddleware(app, allow_origins=allow_origins)
```

## Acceptance Criteria
- [ ] CORS configured from environment variable
- [ ] Only specific domains allowed (not *)
- [ ] Tested with home-assistant.io domain
- [ ] Fallback domains documented
- [ ] Code review approved

## Effort
1 hour

## File
`synctacles_db/api/main.py:51`

## Priority
CRITICAL - BLOCKS LAUNCH' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #1 created"

# Task 2: Dependabot
gh issue create \
    --title "[CRITICAL] Setup Automated Dependency Scanning" \
    --label "security,critical,devops,dependencies" \
    --body '## Description
Setup GitHub Actions + Dependabot for automated dependency updates and vulnerability scanning.

## Files to Create
- `.github/dependabot.yml`
- `.github/workflows/security-scan.yml`

## Acceptance Criteria
- [ ] Dependabot enabled and configured
- [ ] Weekly dependency check runs
- [ ] Automatic PR creation for updates
- [ ] Security scanning workflow active
- [ ] Bandit integration for code security
- [ ] Test run successful

## Subtasks
- [ ] Setup Dependabot config
- [ ] Create security scanning workflow
- [ ] Configure Bandit for Python code scanning
- [ ] Test and verify workflows

## Effort
2 hours

## Priority
CRITICAL - BLOCKS LAUNCH' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #2 created"

# Task 3: Deployment Verification
gh issue create \
    --title "[CRITICAL] Create Post-Deployment Verification Script" \
    --label "deployment,critical,automation,testing" \
    --body '## Description
Comprehensive deployment validation script to catch broken deployments before going live.

## File
Create `scripts/post-deploy-verify.sh`

## Validation Checks
- [ ] All 7 API endpoints responding (200 OK)
- [ ] Database connection verified
- [ ] All collectors running
- [ ] All normalizers running
- [ ] Data freshness within threshold (< 30 min)
- [ ] Health check endpoint working
- [ ] Prometheus metrics available
- [ ] No error logs in first 5 minutes

## Acceptance Criteria
- [ ] Script is executable and idempotent
- [ ] Returns clear pass/fail status
- [ ] Tested in staging environment
- [ ] Integrated into deployment pipeline
- [ ] Rollback triggered on verification failure

## Effort
2 hours

## Priority
CRITICAL - BLOCKS LAUNCH' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #3 created"

# Create HIGH priority tasks
echo ""
echo "📝 Creating HIGH priority tasks..."

# Task 4: Unit Tests
gh issue create \
    --title "[HIGH] Implement Unit Test Suite" \
    --label "testing,quality,backend,high" \
    --body '## Description
Implement comprehensive unit tests for critical paths (data normalization, API response validation).

## Test Targets
- [ ] Data normalization logic (60%+ coverage)
- [ ] API response validation
- [ ] Price data parsing
- [ ] Generation data normalization
- [ ] Load data transformation
- [ ] Error handling paths

## Tool
pytest with fixtures

## Acceptance Criteria
- [ ] 60%+ test coverage on critical paths
- [ ] All critical functions have unit tests
- [ ] Tests run in CI/CD pipeline
- [ ] All tests passing
- [ ] Coverage report generated

## Effort
8 hours

## Files Affected
- synctacles_db/collectors/
- synctacles_db/importers/
- synctacles_db/normalizers/

## When
Next sprint (post-launch)' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #4 created"

# Task 5: Monitoring & Alerting
gh issue create \
    --title "[HIGH] Setup Monitoring & Alerting" \
    --label "monitoring,observability,devops,high" \
    --body '## Description
Setup Prometheus alerts and monitoring dashboard for production observability.

## Alert Rules
- [ ] Data staleness > 30 minutes (CRITICAL)
- [ ] API error rate > 1% (HIGH)
- [ ] Database connection failures (CRITICAL)
- [ ] Collector/normalizer failures (HIGH)
- [ ] Memory usage > 80% (MEDIUM)
- [ ] CPU usage > 75% (MEDIUM)

## Acceptance Criteria
- [ ] Alert rules defined in Prometheus
- [ ] AlertManager configured
- [ ] Slack/email notifications working
- [ ] Dashboard created in Grafana
- [ ] Test alerts firing correctly
- [ ] Runbooks written for each alert

## Effort
6 hours

## When
Next sprint' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #5 created"

# Task 6: Code Quality Tools
gh issue create \
    --title "[HIGH] Setup Code Quality Tools & CI/CD Pipeline" \
    --label "quality,devops,ci-cd,high" \
    --body '## Description
Setup automated code quality checks and CI/CD pipeline.

## Tools to Setup
- [ ] Black (code formatting)
- [ ] Ruff (linting)
- [ ] Mypy (type checking)
- [ ] Pre-commit hooks
- [ ] GitHub Actions for CI
- [ ] Branch protection rules

## Acceptance Criteria
- [ ] Pre-commit hooks installed and working
- [ ] GitHub Actions workflow created and passing
- [ ] Branch protection rules enforced
- [ ] All team members can run checks locally
- [ ] CI pipeline runs on every PR
- [ ] Code style consistent across repo

## Configuration Files
- `.pre-commit-config.yaml`
- `.github/workflows/ci.yml`
- `pyproject.toml`

## Effort
4 hours

## When
Next sprint' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #6 created"

# Task 7: Database Resilience
gh issue create \
    --title "[HIGH] Setup Database Resilience & HA" \
    --label "database,ha,resilience,infrastructure,high" \
    --body '## Description
Improve database resilience and prepare for high-availability setup.

## Implementation
- [ ] Connection pooling (pgBouncer)
- [ ] Read replica setup (optional, for analytics)
- [ ] Automated backups with verification
- [ ] Recovery testing (tested end-to-end)
- [ ] Backup retention policy documented

## Acceptance Criteria
- [ ] pgBouncer installed and configured
- [ ] Connection pooling working
- [ ] Automated backups running daily
- [ ] Backup verification passing
- [ ] Recovery procedure documented
- [ ] Recovery tested successfully
- [ ] HA architecture documented

## Effort
12 hours

## When
Q1 2026 (before scale-up)' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #7 created"

# Task 8: Comprehensive Test Plan
gh issue create \
    --title "[HIGH] Create Comprehensive Test Plan" \
    --label "testing,quality,deployment,high" \
    --body '## Description
Create and execute comprehensive test plan covering all system aspects.

## Test Suites
- [ ] Integration test suite (end-to-end data flow)
- [ ] Performance benchmark (latency/throughput)
- [ ] Failover testing (fallback sources)
- [ ] Database backup/restore drill
- [ ] Load testing (10x expected traffic)

## Acceptance Criteria
- [ ] All integration tests passing
- [ ] Performance baseline established
- [ ] Failover tested successfully
- [ ] Backup/restore tested end-to-end
- [ ] Load testing shows no issues at 10x traffic
- [ ] Test results documented

## Performance Targets
- API response time < 100ms (p95)
- Cache hit rate > 75%
- Error rate < 0.1%

## Effort
10 hours

## When
Before v1.0.0 release' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ Task #8 created"

echo ""
echo "✅ HIGH priority tasks created"

# Create MEDIUM priority tasks
echo ""
echo "📝 Creating MEDIUM priority tasks..."

gh issue create \
    --title "[MEDIUM] Create Developer Documentation" \
    --label "documentation,developer,onboarding,medium" \
    --body '## Description
Create comprehensive developer onboarding documentation.

## Files to Create
- [ ] `CONTRIBUTING.md` - How to submit PRs, code review process
- [ ] `docs/LOCAL_SETUP.md` - Local development environment setup
- [ ] `docs/CODE_STRUCTURE.md` - Where to find features and code organization
- [ ] `docs/ADDING_FEATURES.md` - Step-by-step guide for adding new features

## Acceptance Criteria
- [ ] All 4 documents completed
- [ ] Covers full developer workflow
- [ ] Includes code examples
- [ ] Links to existing documentation
- [ ] Reviewed by team

## Effort
6 hours

## When
Q1 2026 (before open-sourcing)' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

gh issue create \
    --title "[MEDIUM] Document Release Process" \
    --label "documentation,release,devops,medium" \
    --body '## Description
Document and automate release procedures.

## Files to Create
- [ ] `docs/RELEASE_PROCESS.md` - Release checklist and procedures
- [ ] Update `CHANGELOG.md` template for auto-generation
- [ ] Document version tagging strategy
- [ ] Document rollback procedures

## Acceptance Criteria
- [ ] Release process fully documented
- [ ] Version tagging strategy clear
- [ ] Rollback procedures documented
- [ ] CHANGELOG template ready
- [ ] Team trained on release process

## Effort
4 hours

## When
Q1 2026' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

gh issue create \
    --title "[MEDIUM] Implement Feature Flag System" \
    --label "feature-flags,deployment,infrastructure,medium" \
    --body '## Description
Implement feature toggles for gradual rollouts and A/B testing.

## Features
- [ ] Feature toggle system (library: Unleash, LaunchDarkly, or custom)
- [ ] A/B testing capability
- [ ] Kill switches for problematic endpoints
- [ ] Admin dashboard for feature control

## Acceptance Criteria
- [ ] Feature flag system integrated
- [ ] Can enable/disable features without deployment
- [ ] A/B testing working
- [ ] Performance impact minimal
- [ ] Documentation written

## Effort
8 hours

## When
Q2 2026 (if doing complex deployments)' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

gh issue create \
    --title "[MEDIUM] Plan Multi-Region/Multi-Country Support" \
    --label "feature,expansion,architecture,scaling,medium" \
    --body '## Description
Plan and implement support for multiple European countries (DE, FR, BE).

## Implementation Plan
- [ ] Analyze data sources for each country
- [ ] Design multi-country architecture
- [ ] Replicate architecture for Germany (DE)
- [ ] Replicate for France (FR)
- [ ] Replicate for Belgium (BE)
- [ ] Setup unified admin console
- [ ] Country-specific branding support

## Acceptance Criteria
- [ ] Architecture supports 4+ countries
- [ ] Single codebase, different deployments
- [ ] Unified monitoring across regions
- [ ] Easy to add new country
- [ ] Performance not degraded

## Effort
40 hours (phased)

## When
Q2-Q3 2026

## Effort Breakdown
- Planning: 8h
- Germany setup: 12h
- France setup: 10h
- Belgium setup: 10h' \
    --assignee="" \
    2>&1 | grep -o "#[0-9]*" | head -1

echo "✅ MEDIUM priority tasks created"

echo ""
echo "✅ All tasks imported to GitHub!"
echo ""
echo "Next steps:"
echo "1. Visit your GitHub issues page"
echo "2. Add any labels or assignees manually"
echo "3. Link related issues together"
echo "4. Set milestone for CRITICAL items: v1.0.0 Launch"
echo ""
