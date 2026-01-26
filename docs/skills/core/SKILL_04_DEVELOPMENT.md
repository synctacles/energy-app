# SKILL CORE 04 — DEVELOPMENT STANDARDS

**Status:** MANDATORY for all development sessions
**Version:** 1.0
**Last Updated:** 2026-01-26  

---

## PURPOSE

This document defines MANDATORY development standards for all SYNCTACLES projects. Claude Code (CC) MUST read and follow these standards for EVERY development session.

**NON-COMPLIANCE = SESSION INVALID**

---

## SECTION 1: SESSION START CHECKLIST

Before writing ANY code, CC MUST:

```markdown
## Session Start
- [ ] Read all relevant SKILL files
- [ ] Confirm current branch (NEVER commit directly to main)
- [ ] Review open issues/TODOs
- [ ] State what will be built this session
```

CC must output this checklist at session start.

---

## SECTION 2: BRANCHING STRATEGY

### Branch Structure

```
main (protected)     ← Only releases, tagged
  └── develop        ← Integration branch
       └── feature/* ← New features
       └── fix/*     ← Bug fixes
       └── hotfix/*  ← Emergency production fixes
```

### Rules

| Rule | Enforcement |
|------|-------------|
| NEVER commit directly to main | CC must refuse |
| All work on feature branches | CC must create branch first |
| Branch naming: `type/short-description` | e.g., `feature/self-healing` |
| Delete branch after merge | Keep repo clean |

### Commands

```bash
# Start new feature
git checkout develop
git pull origin develop
git checkout -b feature/my-feature

# Ready to merge
git checkout develop
git merge feature/my-feature
git branch -d feature/my-feature
```

---

## SECTION 3: CODE QUALITY STANDARDS

### 3.1 Error Handling

**FORBIDDEN:**
```python
# ❌ NEVER do this
try:
    risky_operation()
except:
    pass

# ❌ NEVER do this
except Exception as e:
    print(e)
```

**REQUIRED:**
```python
# ✅ Always specific exceptions
try:
    risky_operation()
except sqlite3.OperationalError as e:
    logger.error(f"Database locked: {e}", exc_info=True)
    raise DatabaseLockError(f"Could not access database: {e}") from e
except ValueError as e:
    logger.warning(f"Invalid input: {e}")
    return Result(success=False, error=str(e))
```

### 3.2 Logging Standards

```python
import logging

logger = logging.getLogger(__name__)

# Levels:
logger.debug("Detailed diagnostic info")      # Development only
logger.info("Normal operation events")         # User-visible operations
logger.warning("Unexpected but handled")       # Degraded but working
logger.error("Operation failed", exc_info=True) # Include stack trace
logger.critical("System cannot continue")      # Fatal errors
```

### 3.3 Input Validation

**ALL external input MUST be validated:**

```python
# ✅ Required pattern
def delete_orphaned_statistics(entity_ids: list[str], dry_run: bool = True) -> Result:
    # Validate input
    if not isinstance(entity_ids, list):
        raise TypeError(f"entity_ids must be list, got {type(entity_ids)}")
    
    if not all(isinstance(e, str) for e in entity_ids):
        raise ValueError("All entity_ids must be strings")
    
    if len(entity_ids) > 10000:
        raise ValueError(f"Too many entities: {len(entity_ids)}, max 10000")
    
    # Sanitize - prevent SQL injection
    sanitized_ids = [e for e in entity_ids if re.match(r'^[\w\.:_-]+$', e)]
    
    # Now safe to proceed
    ...
```

### 3.4 Security Requirements

| Requirement | Implementation |
|-------------|----------------|
| No secrets in code | Use environment variables |
| No SQL string concatenation | Use parameterized queries |
| No eval/exec | Never execute dynamic code |
| No shell=True | Use subprocess with list args |
| Validate file paths | Prevent path traversal |

**SQL Example:**
```python
# ❌ FORBIDDEN - SQL injection risk
cursor.execute(f"DELETE FROM states WHERE entity_id = '{entity_id}'")

# ✅ REQUIRED - Parameterized query
cursor.execute("DELETE FROM states WHERE entity_id = ?", (entity_id,))
```

### 3.5 Dependency Management

**requirements.txt MUST have pinned versions:**

```
# ❌ FORBIDDEN
requests
sqlalchemy

# ✅ REQUIRED
requests==2.31.0
sqlalchemy==2.0.25
```

**After adding dependencies:**
```bash
pip freeze > requirements.txt
```

---

## SECTION 4: TESTING REQUIREMENTS

### 4.1 Minimum Coverage

| Component | Unit Tests | Integration Tests |
|-----------|------------|-------------------|
| Core logic | REQUIRED | REQUIRED |
| API endpoints | REQUIRED | REQUIRED |
| Database operations | REQUIRED | REQUIRED |
| Utilities | REQUIRED | Optional |

### 4.2 Test Naming

```python
def test_<function>_<scenario>_<expected_result>():
    """Test that <function> does <expected> when <scenario>."""
    
# Examples:
def test_delete_orphans_with_empty_list_returns_zero():
def test_delete_orphans_with_locked_db_raises_error():
def test_health_score_with_fragmented_db_returns_grade_c():
```

### 4.3 Test Structure

```python
def test_example():
    # Arrange - Set up test data
    db = create_test_db(orphans=100)
    
    # Act - Execute the function
    result = delete_orphaned_statistics(db)
    
    # Assert - Verify results
    assert result.deleted_count == 100
    assert result.success is True
```

### 4.4 Before Merge Requirements

```bash
# ALL must pass before merge
pytest tests/ -v
pytest tests/ --cov=src --cov-fail-under=70
```

---

## SECTION 5: RELEASE PROCESS

### 5.1 Semantic Versioning

```
MAJOR.MINOR.PATCH

MAJOR: Breaking changes (API incompatible)
MINOR: New features (backward compatible)
PATCH: Bug fixes (backward compatible)
```

### 5.2 Release Checklist

CC MUST complete before ANY release:

```markdown
## Pre-Release Checklist v[X.X.X]

### Code Quality
- [ ] All tests pass: `pytest tests/ -v`
- [ ] No hardcoded secrets: `grep -r "api_key\|password\|secret" src/`
- [ ] Error handling audit complete
- [ ] Input validation on all external inputs
- [ ] SQL injection check (no string concatenation)

### Security
- [ ] Dependencies scanned: `pip-audit`
- [ ] No sensitive data in logs
- [ ] File permissions correct

### Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual smoke test completed
- [ ] Tested on clean HA installation

### Documentation
- [ ] CHANGELOG.md updated
- [ ] README.md current
- [ ] Version bumped in all files

### Release
- [ ] Git tag created: `git tag -a vX.X.X -m "Release X.X.X"`
- [ ] Tag pushed: `git push origin vX.X.X`
- [ ] Release notes written
```

### 5.3 CHANGELOG Format

```markdown
# Changelog

## [1.1.0] - 2026-02-15

### Added
- Self-healing bug reporting system
- Statistics editor

### Fixed
- Database lock handling (#142)

### Changed
- Improved error messages

### Security
- Fixed SQL injection in entity filter
```

---

## SECTION 6: DOCUMENTATION REQUIREMENTS

### 6.1 Code Comments

```python
def complex_function(data: dict) -> Result:
    """
    Short description of what function does.
    
    Args:
        data: Description of parameter
        
    Returns:
        Result object with success status and data
        
    Raises:
        ValueError: If data is invalid
        DatabaseError: If database operation fails
        
    Example:
        >>> result = complex_function({"key": "value"})
        >>> result.success
        True
    """
```

### 6.2 TODO Format

```python
# TODO(leo): Refactor this to use async - Issue #45
# FIXME(cc): Race condition when concurrent access - Critical
# HACK: Temporary workaround for HA 2025.1 bug - Remove after 2025.2
```

---

## SECTION 7: SESSION END CHECKLIST

Before ending ANY session, CC MUST:

```markdown
## Session End Checklist

### Code Status
- [ ] All changes committed with descriptive messages
- [ ] Branch pushed to remote
- [ ] No uncommitted work left

### Quality
- [ ] Tests written for new code
- [ ] Tests pass locally
- [ ] No new linting errors

### Documentation
- [ ] Code comments added
- [ ] SKILL files updated if architecture changed
- [ ] Handoff notes written if session incomplete

### Handoff
- [ ] Summary of what was done
- [ ] Summary of what remains
- [ ] Any blockers or decisions needed
```

---

## SECTION 8: CC-SPECIFIC INSTRUCTIONS

### 8.1 At Session Start

CC MUST output:

```
## Session Start Confirmation

✅ Read SKILL files: [list which ones]
✅ Current branch: [branch name]
✅ Session goal: [what will be built]
✅ Estimated scope: [small/medium/large]
```

### 8.2 At Session End

CC MUST output:

```
## Session End Report

### Completed
- [list of completed items]

### Tests
- Added: [number]
- Passing: [number]
- Coverage: [percentage if known]

### Commits
- [list commit messages]

### Remaining Work
- [list if any]

### Blockers/Decisions Needed
- [list if any]
```

### 8.3 Before Any Destructive Operation

CC MUST confirm:

```
⚠️ DESTRUCTIVE OPERATION DETECTED

Action: [describe action]
Affected: [what will be changed/deleted]
Reversible: [yes/no]
Backup required: [yes/no]

Proceeding requires explicit confirmation from Leo.
```

---

## SECTION 9: ENFORCEMENT

### How This Is Enforced

1. **Leo** references this SKILL in CC session prompts
2. **CC** outputs checklists at session start/end
3. **Pre-commit hooks** (future) block non-compliant commits
4. **CI/CD** (future) runs automated checks

### Non-Compliance Response

If CC detects it has violated standards:

```
⚠️ STANDARDS VIOLATION DETECTED

Violation: [describe]
Section: [reference SKILL section]
Remediation: [what CC will do to fix]
```

---

## SECTION 10: EXCEPTIONS

Standards may be bypassed ONLY with explicit approval:

```
## Standards Exception Request

Requested by: [Leo/CC]
Standard bypassed: [reference section]
Reason: [why exception needed]
Risk assessment: [what could go wrong]
Mitigation: [how we handle the risk]

Approved: [ ] Yes [ ] No
```

---

## APPENDIX A: Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│                  DEVELOPMENT QUICK REFERENCE                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  BRANCHING          NEVER commit to main directly          │
│                     Always: feature/* or fix/*             │
│                                                             │
│  ERRORS             NEVER except: pass                     │
│                     ALWAYS specific exceptions + logging   │
│                                                             │
│  SQL                NEVER string concatenation             │
│                     ALWAYS parameterized queries           │
│                                                             │
│  SECRETS            NEVER in code                          │
│                     ALWAYS environment variables           │
│                                                             │
│  TESTING            ALWAYS tests for new code              │
│                     ALWAYS run tests before merge          │
│                                                             │
│  RELEASES           ALWAYS semantic versioning             │
│                     ALWAYS changelog entry                 │
│                     ALWAYS git tag                         │
│                                                             │
│  DEPENDENCIES       ALWAYS pinned versions                 │
│                     ALWAYS pip-audit before release        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## APPENDIX B: CC Session Prompt Template

Leo should start CC sessions with:

```
Read the following SKILL files before proceeding:
- core/SKILL_04_DEVELOPMENT.md (MANDATORY)
- [other relevant SKILLs]

Confirm you have read them by outputting the Session Start Checklist.

Today's task: [description]
```

---

*This document is the authoritative source for development standards.*
*All SYNCTACLES development MUST comply with these standards.*
*Last review: 2026-01-26*
