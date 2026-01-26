# SKILL CORE 05 — SECURITY STANDARDS

**Status:** MANDATORY
**Version:** 1.0
**Last Updated:** 2026-01-26

---

## PURPOSE

Definieert VERPLICHTE security standaarden voor alle SYNCTACLES ontwikkeling. Dit document borgt dat security STRUCTUREEL is ingebouwd, niet een afterthought.

**GEEN UITZONDERINGEN. GEEN "IK FIX HET LATER".**

---

## SECURITY LAYERS

```
┌─────────────────────────────────────────────────────────────┐
│                    SECURITY BORGING                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LAYER 1: PRE-COMMIT (lokaal, blokkeert commit)             │
│  ├── SQL injection patterns                                  │
│  ├── Hardcoded secrets                                       │
│  ├── Direct main commits                                     │
│  └── Code formatting                                         │
│                                                              │
│  LAYER 2: CI/CD (GitHub Actions, blokkeert merge)           │
│  ├── Bandit static analysis                                  │
│  ├── pip-audit dependencies                                  │
│  ├── Safety vulnerability check                              │
│  └── Test coverage gate                                      │
│                                                              │
│  LAYER 3: RELEASE (handmatig, blokkeert deploy)             │
│  ├── Security rapport review                                 │
│  ├── Manual checklist                                        │
│  └── Leo approval                                            │
│                                                              │
│  LAYER 4: RUNTIME (productie, detecteert issues)            │
│  ├── Structured logging                                      │
│  ├── Error monitoring                                        │
│  └── Health checks                                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## HARD RULES

### Rule S1: No SQL String Concatenation

**VERBODEN:**
```python
# ❌ NOOIT - SQL injection risk
cursor.execute(f"SELECT * FROM users WHERE id = '{user_id}'")
cursor.execute("SELECT * FROM users WHERE id = " + user_id)
cursor.execute("SELECT * FROM users WHERE id = %s" % user_id)
```

**VERPLICHT:**
```python
# ✅ ALTIJD - Parameterized queries
cursor.execute("SELECT * FROM users WHERE id = ?", (user_id,))
cursor.execute("SELECT * FROM users WHERE id = %s", (user_id,))
```

**Enforcement:** Pre-commit hook + CI check

---

### Rule S2: No Hardcoded Secrets

**VERBODEN:**
```python
# ❌ NOOIT
API_KEY = "sk_live_abc123"
DATABASE_URL = "postgresql://user:password@localhost/db"
SECRET = "my-secret-key"
```

**VERPLICHT:**
```python
# ✅ ALTIJD - Environment variables
API_KEY = os.getenv("API_KEY")
if not API_KEY:
    raise ValueError("API_KEY required in environment")

DATABASE_URL = os.getenv("DATABASE_URL")
if not DATABASE_URL:
    raise ValueError("DATABASE_URL required in environment")
```

**Enforcement:** Pre-commit hook + CI check + .gitignore voor .env

---

### Rule S3: Validate All External Input

**VERBODEN:**
```python
# ❌ NOOIT - Untrusted input direct gebruiken
def get_file(filename):
    return open(f"/data/{filename}").read()

def query_entity(entity_id):
    cursor.execute(f"SELECT * FROM entities WHERE id = '{entity_id}'")
```

**VERPLICHT:**
```python
# ✅ ALTIJD - Validate + sanitize
import re
from pathlib import Path

def get_file(filename: str) -> str:
    # Validate: only alphanumeric + limited chars
    if not re.match(r'^[\w\-\.]+$', filename):
        raise ValueError(f"Invalid filename: {filename}")
    
    # Prevent path traversal
    safe_path = Path("/data") / filename
    if not safe_path.resolve().is_relative_to(Path("/data")):
        raise ValueError("Path traversal detected")
    
    return safe_path.read_text()

def query_entity(entity_id: str) -> dict:
    # Validate format
    if not re.match(r'^[\w\.:_-]+$', entity_id):
        raise ValueError(f"Invalid entity_id: {entity_id}")
    
    # Parameterized query
    cursor.execute("SELECT * FROM entities WHERE id = ?", (entity_id,))
```

**Enforcement:** Code review + Bandit

---

### Rule S4: No Dangerous Functions

**VERBODEN:**
```python
# ❌ NOOIT
eval(user_input)
exec(user_code)
os.system(command)
subprocess.run(command, shell=True)
pickle.loads(untrusted_data)
```

**VERPLICHT:**
```python
# ✅ ALTIJD - Safe alternatives
import subprocess
import json

# Subprocess zonder shell
subprocess.run(["ls", "-la", directory], check=True)

# JSON ipv pickle voor serialization
data = json.loads(trusted_json)
```

**Enforcement:** Bandit B102, B307, B602

---

### Rule S5: Dependencies Must Be Pinned & Audited

**VERBODEN:**
```
# requirements.txt
# ❌ NOOIT - Unpinned versions
requests
sqlalchemy
fastapi
```

**VERPLICHT:**
```
# requirements.txt
# ✅ ALTIJD - Pinned versions
requests==2.31.0
sqlalchemy==2.0.25
fastapi==0.109.0
```

**Audit commands:**
```bash
# Run before EVERY release
pip-audit -r requirements.txt
safety check --file requirements.txt
```

**Enforcement:** CI pipeline

---

### Rule S6: Errors Never Expose Internals

**VERBODEN:**
```python
# ❌ NOOIT - Exposes stack trace to user
@app.exception_handler(Exception)
async def handler(request, exc):
    return JSONResponse(
        status_code=500,
        content={"error": str(exc), "trace": traceback.format_exc()}
    )
```

**VERPLICHT:**
```python
# ✅ ALTIJD - Generic message to user, details to log
@app.exception_handler(Exception)
async def handler(request, exc):
    # Log full details internally
    logger.error("Unhandled exception", exc_info=exc, extra={
        "path": request.url.path,
        "method": request.method,
    })
    
    # Generic message to user
    return JSONResponse(
        status_code=500,
        content={"error": "Internal server error"}
    )
```

**Enforcement:** Code review

---

### Rule S7: Authentication on All Sensitive Endpoints

**VERBODEN:**
```python
# ❌ NOOIT - Unprotected sensitive endpoint
@app.post("/api/v1/cleanup")
async def cleanup(entity_ids: list[str]):
    return await cleaner.delete(entity_ids)
```

**VERPLICHT:**
```python
# ✅ ALTIJD - Auth required
@app.post("/api/v1/cleanup")
async def cleanup(
    entity_ids: list[str],
    api_key: str = Header(..., alias="X-API-Key")
):
    # Validate API key
    subscription = await validate_api_key(api_key)
    if not subscription:
        raise HTTPException(401, "Invalid API key")
    
    # Check tier
    if subscription.tier != "premium":
        raise HTTPException(403, "Premium required")
    
    return await cleaner.delete(entity_ids)
```

**Enforcement:** Code review + integration tests

---

## SECURITY TOOLS

### Required Tools

| Tool | Purpose | When |
|------|---------|------|
| `bandit` | Static code analysis | Pre-commit + CI |
| `pip-audit` | Dependency vulnerabilities | CI + release |
| `safety` | Known vulnerabilities DB | CI + release |
| `ruff` | Linting + formatting | Pre-commit |

### Install
```bash
pip install bandit pip-audit safety ruff pre-commit
```

### Run Manually
```bash
# Full security scan
bandit -r synctacles_db/ -f txt
pip-audit -r requirements.txt
safety check --file requirements.txt

# Quick check
bandit -r synctacles_db/ -ll  # Only HIGH severity
```

---

## INCIDENT RESPONSE

### If Vulnerability Found in Production

1. **ASSESS** - Severity? Exploited?
2. **CONTAIN** - Disable affected endpoint if needed
3. **FIX** - Patch on hotfix branch
4. **DEPLOY** - Emergency deploy
5. **NOTIFY** - If user data affected
6. **POSTMORTEM** - How did it slip through?

### Severity Levels

| Level | Response Time | Example |
|-------|---------------|---------|
| CRITICAL | 1 hour | SQL injection, RCE, data breach |
| HIGH | 24 hours | Auth bypass, sensitive data exposure |
| MEDIUM | 1 week | XSS, CSRF, info disclosure |
| LOW | Next release | Best practice violations |

---

## COMPLIANCE CHECKLIST

### Before Every Commit
- [ ] No SQL string concatenation
- [ ] No hardcoded secrets
- [ ] Input validation on external data
- [ ] Parameterized queries only

### Before Every Release
- [ ] `bandit` scan clean (0 HIGH/MEDIUM)
- [ ] `pip-audit` clean (0 HIGH/CRITICAL)
- [ ] `safety check` clean
- [ ] All tests pass
- [ ] Security section in CHANGELOG if relevant

### Quarterly Review
- [ ] Dependency update check
- [ ] Access control review
- [ ] Log retention check
- [ ] Backup verification

---

## EXCEPTIONS

Security rules kunnen ALLEEN worden overruled door Leo met:

```markdown
## Security Exception

**Rule bypassed:** S3 (Input validation)
**Location:** synctacles_db/legacy/importer.py:45
**Reason:** Legacy code, refactor scheduled Q2
**Risk:** Low - internal use only
**Mitigation:** Rate limiting + logging
**Expiry:** 2026-04-01
**Approved by:** Leo

# nosec B608 - Exception approved, see docs/exceptions/SEC-001.md
```

**Zonder goedkeuring = GEEN uitzondering.**

---

## RELATED SKILLS

| Skill | Description |
|-------|-------------|
| [SKILL_01_HARD_RULES.md](SKILL_01_HARD_RULES.md) | Core development rules |
| [SKILL_02_CODING_STANDARDS.md](SKILL_02_CODING_STANDARDS.md) | Code quality |
| [SKILL_04_DEVELOPMENT.md](SKILL_04_DEVELOPMENT.md) | Development workflow |

---

*Dit document is VERPLICHT voor alle SYNCTACLES ontwikkeling.*
*Security is niet optioneel. Security is niet "later".*
*Security is NU.*
