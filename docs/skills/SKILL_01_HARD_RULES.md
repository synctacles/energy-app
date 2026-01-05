# SKILL 1 — HARD RULES

Fundamental Non-Negotiable Rules for SYNCTACLES
Version: 1.0 (2025-12-30)

---

## PURPOSE

Define the absolute, non-negotiable rules that every line of code, every commit, and every deployment must follow. These are not guidelines or best practices—they are rules with consequences.

---

## CORE HARD RULES

### Rule 1: KISS (Keep It Simple, Stupid)

**Statement:** Every solution must be the simplest possible implementation that solves the problem.

**What This Means:**
- Simplicity beats cleverness
- No premature optimization
- No over-engineering
- No "it might be useful later" features

**Violations:**
- Writing a complex function when a simple loop works
- Adding layers of abstraction for hypothetical future use
- "Smart" solutions that require deep knowledge to understand

**Enforcement:**
- Code review must reject complex solutions if simpler alternatives exist
- Measurable: Can a junior developer understand this in 5 minutes?

**Examples:**

❌ **Bad (Over-engineered):**
```python
# Creating an abstract factory for single use case
class DataSourceFactory:
    _strategies = {}

    @classmethod
    def register(cls, name, strategy):
        cls._strategies[name] = strategy

    @classmethod
    def create(cls, name):
        return cls._strategies[name]()

# Later: factory.create('entso_e')
```

✅ **Good (Simple):**
```python
def get_data_source(name):
    if name == 'entso_e':
        return EntsoEClient()
    elif name == 'tennet':
        return TennetClient()
    else:
        raise ValueError(f"Unknown source: {name}")
```

---

### Rule 2: Fail-Fast, No Silent Failures

**Statement:** When something is wrong, fail immediately with a clear error message. Never hide errors.

**What This Means:**
- Missing configuration → ValueError, not default
- Invalid input → TypeError, not None
- Missing file → FileNotFoundError, not skip it
- No fallback defaults (they hide bugs)

**Violations:**
- `os.getenv("BRAND_NAME", "Default Brand")` - hides misconfiguration
- Try/except that catches and ignores errors
- `return None` when error should be raised

**Enforcement:**
- Every ValueError must have a message explaining how to fix it
- Tests must verify that missing config raises an error
- Code review must check for fallback defaults

**Examples:**

❌ **Bad (Silent failure):**
```python
def load_config():
    brand = os.getenv("BRAND_NAME", "Energy Insights")  # Hides misconfiguration
    return {"brand": brand}

# Later: silently runs with wrong brand
```

✅ **Good (Fail-fast):**
```python
def load_config():
    brand = os.getenv("BRAND_NAME")
    if not brand:
        raise ValueError(
            "BRAND_NAME not set. Run FASE 0 first or set environment variable."
        )
    return {"brand": brand}

# Fails immediately with actionable message
```

---

### Rule 3: Repository is Brand-Free

**Statement:** The git repository contains NO hardcoded branding, domains, or tenant-specific values.

**What This Means:**
- No "Energy Insights NL" strings in code
- No `energy-insights.nl` domains in config
- No tenant-specific paths like `/opt/synctacles`
- All values come from templates or `.env`

**Violations:**
- Hardcoded BRAND_NAME in Python file
- Commit manifest.json (should be .template only)
- Include tenant .env files in git

**Enforcement:**
```bash
# Should return 0 (no matches):
grep -r "Energy Insights" . | grep -v ".template" | grep -v ".example" | wc -l
grep -r "/opt/synctacles" . | grep -v ".template" | grep -v "{{" | wc -l
```

**Examples:**

❌ **Bad (Branded):**
```python
# const.py
BRAND_NAME = "Energy Insights NL"
DOMAIN = "energy-insights.nl"
```

✅ **Good (Brand-free):**
```python
# const.py
import os

BRAND_NAME = os.getenv("BRAND_NAME")
if not BRAND_NAME:
    raise ValueError("BRAND_NAME required")

DOMAIN = os.getenv("BRAND_DOMAIN")
if not DOMAIN:
    raise ValueError("BRAND_DOMAIN required")
```

---

### Rule 4: Template System with `{{PLACEHOLDER}}`

**Statement:** All variable values use `{{PLACEHOLDER}}` format in templates.

**What This Means:**
- Placeholders are `{{VARIABLE_NAME}}` (double braces)
- Not `${VAR}` (bash) or `{% var %}` (jinja)
- Sed-friendly: `sed 's/{{VAR}}/value/g'`
- Every `.template` file must have matching example/documentation

**Violations:**
- Using `${BRAND_NAME}` in templates
- Using single braces `{BRAND_NAME}`
- Template files without documentation

**Enforcement:**
- Grep for template files: `find . -name "*.template"`
- Verify format: `grep "{{[A-Z_]*}}" *.template`
- Document all placeholders in README or .example file

**Examples:**

❌ **Bad (Wrong format):**
```json
{
  "name": "${BRAND_NAME}",
  "domain": "${BRAND_SLUG}"
}
```

✅ **Good (Correct format):**
```json
{
  "name": "{{BRAND_NAME}}",
  "domain": "{{BRAND_SLUG}}"
}
```

---

### Rule 5: No Defaults for Required Configuration

**Statement:** Critical configuration variables MUST NOT have defaults. Force explicit setup.

**What This Means:**
- `BRAND_NAME` has no default
- `DB_HOST` has no default
- `API_KEY` has no default
- Only optional values (timeouts, retries) can have defaults

**Violations:**
- `os.getenv("RETRY_COUNT", 3)` for critical config
- Any env var with hardcoded fallback
- "Convenient" defaults that mask misconfiguration

**Enforcement:**
- Code review must flag every `os.getenv()` with a default
- If it looks like config, it needs explicit validation

**Examples:**

❌ **Bad (Has default):**
```python
db_host = os.getenv("DB_HOST", "localhost")
db_port = os.getenv("DB_PORT", 5432)
# Works silently with localhost if not configured
```

✅ **Good (No default, fail-fast):**
```python
db_host = os.getenv("DB_HOST")
if not db_host:
    raise ValueError("DB_HOST required. Set in .env file.")

db_port = os.getenv("DB_PORT")
if not db_port:
    raise ValueError("DB_PORT required. Set in .env file.")
```

---

### Rule 6: Environment Variables at Runtime, Never Hardcoded

**Statement:** All configuration comes from environment variables (usually via `.env`). Never hardcode configuration.

**What This Means:**
- Use `os.getenv()` to read `.env`
- Systemd services use `EnvironmentFile=/opt/.env`
- No hardcoded paths, domains, API keys, credentials
- Configuration changes don't require code changes

**Violations:**
- API_KEY = "sk_live_abc123" in code
- BASE_URL = "https://api.example.com" in code
- Any value in code that differs per tenant

**Enforcement:**
- Grep for hardcoded values: `grep -r '"https://' . | grep -v ".template"`
- Code review: "This looks like config. Should it be in .env?"

**Examples:**

❌ **Bad (Hardcoded):**
```python
API_KEY = "sk_live_abc123"
BASE_URL = "https://energy-insights.nl"
DB_HOST = "localhost"
```

✅ **Good (From environment):**
```python
API_KEY = os.getenv("API_KEY")
if not API_KEY:
    raise ValueError("API_KEY required in .env")

BASE_URL = os.getenv("BRAND_DOMAIN")
if not BASE_URL:
    raise ValueError("BRAND_DOMAIN required in .env")

DB_HOST = os.getenv("DB_HOST")
if not DB_HOST:
    raise ValueError("DB_HOST required in .env")
```

---

### Rule 7: All Data Includes Quality Metadata

**Statement:** Every data point stored includes source, timestamp, quality, and age information.

**What This Means:**
- Each record tracks: `source`, `timestamp`, `quality`, `age_minutes`
- No bare data without provenance
- Enables fallback and data quality decisions
- API responses include quality indicators

**Violations:**
- Storing generation data without source timestamp
- Normalizers dropping quality information
- API responses without data age

**Enforcement:**
- Schema review: Every normalized table must have quality fields
- Tests: Verify quality metadata always present

**Examples:**

✅ **Good (With metadata):**
```python
{
    "generation_mw": 2500,
    "timestamp": "2025-12-30T10:15:00Z",
    "source": "entso_e_a75",
    "quality": 0.95,
    "age_minutes": 5
}
```

---

### Rule 8: Systemd Units, Not Custom Init

**Statement:** Use systemd for all service management. No custom init scripts.

**What This Means:**
- Each service has a `.service` file in systemd/
- Each scheduled task has a `.timer` file
- Generated from `.template` files with ENV vars
- Logging via journalctl
- No custom bash daemon wrappers

**Violations:**
- Custom start/stop scripts
- Hardcoded service paths
- Manual process management

**Enforcement:**
- Check: All services registered with `systemctl`
- Verify: All run scripts are generated from templates

---

### Rule 9: Tests Are Mandatory for Non-Trivial Logic

**Statement:** Any non-trivial function must have tests. Trivial (3 lines or less) functions don't need tests.

**What This Means:**
- Logic → must have test
- Data transformation → must have test
- Configuration → must have test (especially fail-fast behavior)
- Rendering HTML → must have test

**Violations:**
- Untested data normalization
- No test for missing env var behavior
- Logic in templates without tests

**Enforcement:**
- Code review: "What tests verify this behavior?"
- CI must fail if coverage drops below threshold

---

### Rule 11: Centralized Database Configuration

**Statement:** All database connections use centralized config.settings module. Never hardcode DATABASE_URL or credentials.

**What This Means:**
- Import `from config.settings import DATABASE_URL`
- Never use `os.getenv("DATABASE_URL", "postgresql://...")`
- Never hardcode "synctacles@localhost" or any user/host
- All normalizers, collectors, importers, and scripts use the same config
- Fail-fast validation at startup ensures credentials are correct

**Violations:**
- `DATABASE_URL = "postgresql://synctacles@localhost/synctacles"`
- `os.getenv("DATABASE_URL", "postgresql://synctacles@localhost")`
- Hardcoded credentials in collector or importer modules
- Different modules using different credential sources

**Enforcement:**
- Pre-commit hook blocks commits with pattern: `synctacles@`, `postgresql://[a-z_]+@`
- Every normalizer calls `validate_db_connection()` at startup
- All 4 normalizers, all collectors, all importers must import from config.settings
- Database validation happens before any data processing

**Examples:**

❌ **Bad (Hardcoded credentials):**
```python
DATABASE_URL = "postgresql://synctacles@localhost:5432/synctacles"
engine = create_engine(DATABASE_URL)

# Or with fallback:
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://synctacles@localhost/synctacles")
```

✅ **Good (Centralized config):**
```python
from config.settings import DATABASE_URL
from synctacles_db.normalizers.base import validate_db_connection

# Validate at startup (fail-fast)
validate_db_connection()

engine = create_engine(DATABASE_URL)
```

**Config Module (config/settings.py):**
```python
import os
from sqlalchemy.engine import URL

DATABASE_URL = os.getenv("DATABASE_URL")
if not DATABASE_URL:
    raise ValueError(
        "DATABASE_URL not set. Set in /opt/.env\n"
        "Expected format: postgresql://user@host:port/dbname"
    )
```

**Startup Validation Pattern:**
```python
from synctacles_db.normalizers.base import validate_db_connection

def main():
    # Fail immediately if DB unreachable
    validate_db_connection()

    # ... rest of normalizer logic
```

---

### Rule 10: Documentation Stays Current

**Statement:** Code comments describe WHY, not WHAT. Keep docs in sync with code.

**What This Means:**
- Comments explain decisions, not obvious code
- Every major function has a docstring
- ARCHITECTURE.md stays current
- SKILLS reflect actual system

**Violations:**
- Comment: `x = x + 1  # increment x`
- Outdated architecture diagrams
- SKILL docs describing old behavior

**Enforcement:**
- Code review: Does the comment explain a non-obvious decision?
- Before release: Update ARCHITECTURE.md

---

## ENFORCEMENT MECHANISMS

### Code Review Gates

1. **Static Checks:**
   - `grep -r "os.getenv.*," .` - Find defaults
   - `grep -r "^[A-Z_]* =" . | grep -v "{{" | grep -v "test"` - Find hardcoded config
   - `grep -r "Energy Insights" . | grep -v ".template" | grep -v ".example"` - Find branded strings

2. **Test Requirements:**
   - Fail-fast behavior tested (missing env var raises)
   - Data quality metadata verified
   - Schema validation working

3. **Manual Review:**
   - Is this the simplest solution?
   - Does it fail fast or hide errors?
   - Is config from environment?
   - Are there quality metadata?

### Violation Consequences

| Violation | Consequence |
|-----------|------------|
| KISS violation (over-engineered) | Reject PR, request simplification |
| Silent failure (no fail-fast) | Reject PR, security issue |
| Hardcoded branding | Reject PR, blocks deployment |
| Hardcoded config | Reject PR, security issue |
| Missing quality metadata | Reject PR, data integrity |
| No test for logic | Reject PR (depends on complexity) |
| Outdated docs | Reject PR, maintains knowledge |

---

## QUICK REFERENCE CHECKLIST

Before every commit:

```
Code Quality:
□ Simplest possible solution (KISS)?
□ Fail-fast on errors?
□ Clear error messages?
□ No hardcoded config?
□ All config from environment?

Brand-Free:
□ No hardcoded brand names?
□ No hardcoded domains?
□ Templates use {{PLACEHOLDER}}?
□ .env and manifest.json are .gitignored?

Data Quality:
□ All data includes metadata (source, timestamp, quality)?
□ Quality tracked and accessible?
□ Fallback strategy documented?

Tests & Docs:
□ Non-trivial logic has tests?
□ Tests verify fail-fast behavior?
□ Comments explain WHY, not WHAT?
□ Architecture docs current?
```

---

## RELATED SKILLS

- **SKILL 2**: Architecture - How these rules manifest in design
- **SKILL 3**: Coding Standards - Code-level implementation
- **SKILL 12**: Brand-Free Architecture - Configuration philosophy
- **SKILL 9**: Installer Specs - FASE 0 enforcement of rules

---

## QUESTIONS?

These rules are non-negotiable because they ensure:
1. **Maintainability** - Simple code is easier to understand and fix
2. **Reliability** - Fail-fast prevents silent failures in production
3. **Scalability** - Multi-tenant separation enables growth
4. **Security** - Explicit config and no hardcoded secrets
5. **Data Quality** - Metadata enables intelligent fallback

If a rule seems to conflict with a requirement, discuss with the team. Rules can evolve, but they're not work-arounds.
