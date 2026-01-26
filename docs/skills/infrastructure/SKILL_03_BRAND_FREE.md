# SKILL INFRASTRUCTURE 03 — BRAND-FREE TEMPLATE ARCHITECTURE

Multi-Tenant Deployment Architecture via Template System
Version: 1.1 (2026-01-22)

---

## PURPOSE

Define architectural principles for brand-agnostic repository design that
enables multi-tenant deployments without git conflicts or code duplication.

**Problem Solved:**
Single codebase deployed with different branding across multiple servers
without modifying repository code.

**Use Cases:**
- Multi-region deployment (NL, DE, FR, BE - different brands)
- White-label offerings (same product, different customer brands)
- Testing environments (prod vs staging vs dev with different configs)
- Partner deployments (resellers with own branding)

---

## CORE PRINCIPLES

### 1. Repository = Brand-Free Templates
```
Git Repository:
  ✅ Contains: Generic code + templates
  ❌ Never: Specific brand names, URLs, domains
  ❌ Never: Configuration values
  ❌ Never: Generated files
```

### 2. Runtime = Generated from Templates + .env
```
Server Installation:
  1. Clone brand-free repo
  2. Interactive configuration (FASE 0)
  3. Generate .env + manifest.json
  4. Deploy with generated config
  5. .gitignore prevents accidental commits
```

### 3. Fail-Fast Validation
```
Code Behavior:
  ✅ No defaults (force explicit configuration)
  ✅ Clear errors (missing ENV vars = ValueError)
  ✅ Early detection (fail at import, not at runtime)
```

---

## TEMPLATE SYSTEM ARCHITECTURE

### Placeholder Format

**Standard:** `{{VARIABLE_NAME}}`

**Why double braces:**
- Distinct from bash variables `${VAR}`
- Sed-friendly: `sed 's/{{VAR}}/value/g'`
- Human-readable in templates
- Compatible with jinja2 (future option)

**Common Placeholders:**
```
{{BRAND_NAME}}        - Display name
{{BRAND_SLUG}}        - Technical identifier
{{BRAND_DOMAIN}}      - Production domain
{{GITHUB_ACCOUNT}}    - Repository owner
{{REPO_NAME}}         - Repository name
{{HA_DOMAIN}}         - Home Assistant domain
```

### Template Files

**manifest.json.template:**
```json
{
  "domain": "{{BRAND_SLUG}}",
  "name": "{{BRAND_NAME}}",
  "codeowners": ["@{{GITHUB_ACCOUNT}}"],
  "documentation": "https://github.com/{{GITHUB_ACCOUNT}}/{{REPO_NAME}}"
}
```

**Generation:**
```bash
sed -e "s/{{BRAND_NAME}}/$BRAND_NAME/g" \
    -e "s/{{BRAND_SLUG}}/$BRAND_SLUG/g" \
    manifest.json.template > manifest.json
```

### .env.example Pattern

**Purpose:** Documentation + manual configuration guide

**Format:**
```bash
# Placeholders use YOUR_* or your-* prefix
BRAND_NAME="YOUR_BRAND_NAME"
BRAND_SLUG="your-brand-slug"

# Comments explain expected format
# Example values show typical patterns
```

**Benefits:**
- Clear what needs customization
- Self-documenting
- Copy-paste friendly for manual setup

---

## PYTHON FAIL-FAST PATTERNS

### No Defaults Strategy

**Anti-Pattern (Dangerous):**
```python
# BAD: Silent fallback to branded default
brand_name = os.getenv("BRAND_NAME", "Energy Insights NL")
```

**Correct Pattern:**
```python
# GOOD: Fail fast with clear message
brand_name = os.getenv("BRAND_NAME")
if not brand_name:
    raise ValueError(
        "BRAND_NAME environment variable required.\n"
        "Run setup script FASE 0 or create .env from .env.example"
    )
```

**Benefits:**
- Prevents silent misconfiguration
- Forces proper setup
- Clear error messages guide user
- No "UNCONFIGURED" leaking to production

### Validation at Import Time

```python
# config/settings.py
class Settings:
    def __init__(self):
        # Validate ALL required vars at init
        required = {
            "BRAND_NAME": os.getenv("BRAND_NAME"),
            "BRAND_SLUG": os.getenv("BRAND_SLUG"),
            "BRAND_DOMAIN": os.getenv("BRAND_DOMAIN"),
        }

        missing = [k for k, v in required.items() if not v]
        if missing:
            raise ValueError(
                f"Missing required environment variables: {', '.join(missing)}"
            )

        # Only assign after validation
        self.brand_name = required["BRAND_NAME"]
        # etc.

# Singleton pattern
settings = Settings()  # Fails at import if .env missing
```

**Why Import-Time:**
- Fail before server starts
- Clear systemd error logs
- Developer sees issue immediately
- Production never runs misconfigured

### Manifest Loading Pattern

```python
# custom_components/synctacles/const.py
_manifest_path = Path(__file__).parent / "manifest.json"

if not _manifest_path.exists():
    raise FileNotFoundError(
        f"manifest.json not found at {_manifest_path}\n"
        f"This file is generated from manifest.json.template.\n"
        f"Run setup script FASE 0 to generate it."
    )

_manifest = json.loads(_manifest_path.read_text())
HA_COMPONENT_NAME = _manifest["name"]  # No .get() fallback
```

**Key Points:**
- No try/except silence
- No fallback values
- Detailed error message
- Guides user to solution

---

## .GITIGNORE STRATEGY

### Critical Exclusions

```gitignore
# Generated configuration (NEVER commit)
.env
.env.local
.env.*.local

# Generated from templates (NEVER commit)
custom_components/synctacles/manifest.json

# Brand-specific runtime paths
/opt/*/
/var/log/*/
/var/lib/*/

# Python runtime
__pycache__/
*.pyc
venv/
.venv/

# Development
.vscode/
.idea/
*.swp
```

### Why Strict Exclusions

**Problem Prevented:**
```
Developer 1: Commits .env with "Brand A"
Developer 2: Pulls, overwrites with "Brand B"
Result: Git conflicts, confusion, security leak
```

**Solution:**
- .env never tracked
- manifest.json generated, never committed
- Clear .gitignore.template in repo

### Template Commit Strategy

**DO Commit:**
- ✅ manifest.json.template
- ✅ .env.example
- ✅ .gitignore

**NEVER Commit:**
- ❌ manifest.json (generated)
- ❌ .env (contains secrets)
- ❌ Brand-specific configs

---

## MULTI-TENANT DEPLOYMENT PATTERNS

### Pattern 1: Regional Instances

```
Repository: synctacles-api (brand-free)

Server NL (Production):
  .env → BRAND_NAME="SYNCTACLES"
  Domain: synctacles.com

Server NL (Development):
  .env → BRAND_NAME="SYNCTACLES [DEV]"
  Domain: dev.synctacles.com

Server DE (Future):
  .env → BRAND_NAME="SYNCTACLES DE"
  Domain: synctacles.de
```

**Benefits:**
- Single repo maintenance
- Consistent core logic
- Localized branding
- Independent deployments

### Pattern 2: White-Label SaaS

```
Repository: white-label-energy (brand-free)

Customer A:
  .env → BRAND_NAME="Client A Energy"
  GitHub: clientA/energy-platform

Customer B:
  .env → BRAND_NAME="Client B Power"
  GitHub: clientB/power-insights
```

**Benefits:**
- Each customer thinks it's their product
- Centralized code updates
- No code duplication
- Easy customer onboarding

### Pattern 3: Environment Segregation

```
Repository: synctacles-api (brand-free)

Production:
  .env → BRAND_NAME="SYNCTACLES"
  BRAND_DOMAIN="synctacles.com"

Staging:
  .env → BRAND_NAME="SYNCTACLES [STAGING]"
  BRAND_DOMAIN="staging.synctacles.com"

Development:
  .env → BRAND_NAME="SYNCTACLES [DEV]"
  BRAND_DOMAIN="dev.synctacles.com"
```

**Benefits:**
- Clear environment identification
- Prevents production accidents
- Same codebase, different configs

---

## TEMPLATE GENERATION STRATEGIES

### Strategy 1: Interactive (Recommended)

**When:** Initial server setup

**How:** FASE 0 prompts user for values

```bash
read -p "Brand Name: " BRAND_NAME
read -p "Brand Slug: " BRAND_SLUG
# Generate .env
# Generate manifest.json
```

**Pros:**
- User-friendly
- Validates input
- Immediate feedback
- Guided process

### Strategy 2: Pre-Configured .env

**When:** Automated deployments (Ansible, Terraform)

**How:** Deploy .env before running installer

```bash
# Deployment script
scp brand-a.env server:/opt/.env
ssh server "./setup.sh fase1-6"
```

**Pros:**
- Automation-friendly
- Infrastructure as Code
- No manual input
- Repeatable

### Strategy 3: Environment Variables

**When:** Container deployments (Docker, Kubernetes)

**How:** Pass ENV vars to container

```yaml
# docker-compose.yml
environment:
  - BRAND_NAME=SYNCTACLES
  - BRAND_SLUG=synctacles
```

**Pros:**
- Cloud-native
- 12-factor app compliant
- Secret management integration
- No file dependencies

---

## TESTING BRAND-FREE CODE

### Unit Testing Pattern

```python
# tests/test_branding.py
import os
import pytest

def test_settings_requires_env(monkeypatch):
    # Remove brand env vars
    monkeypatch.delenv("BRAND_NAME", raising=False)

    # Should raise ValueError
    with pytest.raises(ValueError, match="BRAND_NAME"):
        from config.settings import settings

def test_settings_loads_correctly(monkeypatch):
    # Set brand env vars
    monkeypatch.setenv("BRAND_NAME", "Test Brand")
    monkeypatch.setenv("BRAND_SLUG", "test-brand")

    from config.settings import settings
    assert settings.brand_name == "Test Brand"
```

### Integration Testing

```bash
# Test fresh install with different brands
./test-install.sh "Brand A" "brand-a"
./test-install.sh "Brand B" "brand-b"

# Verify no cross-contamination
grep -r "Brand A" /opt/brand-b/  # Should return nothing
```

### CI/CD Testing

```yaml
# .github/workflows/test-multi-brand.yml
jobs:
  test-brands:
    strategy:
      matrix:
        brand:
          - {name: "Brand A", slug: "brand-a"}
          - {name: "Brand B", slug: "brand-b"}

    steps:
      - name: Configure brand
        run: |
          echo "BRAND_NAME=${{ matrix.brand.name }}" > .env
          echo "BRAND_SLUG=${{ matrix.brand.slug }}" >> .env

      - name: Run tests
        run: pytest
```

---

## MIGRATION STRATEGIES

### From Branded to Brand-Free

**Phase 1: Template Creation**
```bash
# 1. Identify all branded strings
grep -r "Your Specific Brand" .

# 2. Create templates
cp manifest.json manifest.json.template
sed -i 's/Your Specific Brand/{{BRAND_NAME}}/g' manifest.json.template

# 3. Update .gitignore
echo "manifest.json" >> .gitignore
```

**Phase 2: Code Refactoring**
```python
# Before:
brand_name = "Your Specific Brand"

# After:
brand_name = os.getenv("BRAND_NAME")
if not brand_name:
    raise ValueError("BRAND_NAME required")
```

**Phase 3: Installation Flow**
```bash
# Add FASE 0 to setup script
# Test fresh install
# Document migration for existing servers
```

### Existing Servers Migration

```bash
# 1. Create .env from current config
cat > /opt/.env << EOF
BRAND_NAME="Current Brand"
BRAND_SLUG="current-brand"
# ... existing values
EOF

# 2. Generate manifest from template
./scripts/generate-manifest.sh

# 3. Test services still work
systemctl restart all-services
curl /health
```

---

## SECURITY CONSIDERATIONS

### .env File Protection

```bash
# Permissions
chmod 600 /opt/.env
chown root:root /opt/.env

# Systemd EnvironmentFile
[Service]
EnvironmentFile=/opt/.env
# Automatically protected by systemd
```

### Secrets Management

**DO:**
- ✅ Generate secrets at install time
- ✅ Store in .env (never git)
- ✅ Use systemd EnvironmentFile
- ✅ Rotate regularly

**DON'T:**
- ❌ Commit .env to git
- ❌ Share .env between servers
- ❌ Hardcode secrets in code
- ❌ Log secret values

### Template Injection Prevention

```bash
# Validate input before templating
if [[ "$BRAND_NAME" =~ [^a-zA-Z0-9\ ] ]]; then
    echo "Error: BRAND_NAME contains invalid characters"
    exit 1
fi
```

---

## BEST PRACTICES

### Repository Structure

```
repo/
├── .env.example              # Template with YOUR_* placeholders
├── .gitignore               # Excludes .env, manifest.json
├── custom_components/
│   └── synctacles/
│       ├── manifest.json.template  # {{PLACEHOLDERS}}
│       └── *.py             # No branded strings
├── config/
│   └── settings.py          # Fail-fast ENV loading
└── scripts/
    └── setup/
        └── setup.sh         # FASE 0 interactive config
```

### Naming Conventions

**Templates:** `filename.template`
**Examples:** `filename.example`
**Generated:** `filename` (gitignored)

### Documentation Requirements

**Every template needs:**
- README section explaining placeholders
- Example values
- Generation instructions
- .gitignore reference

### Code Review Checklist

```
□ No hardcoded brand strings
□ No fallback defaults (fail-fast)
□ Templates use {{PLACEHOLDER}} format
□ .gitignore includes generated files
□ .env.example documents all variables
□ Error messages guide to solution
□ Tests cover missing ENV vars
```

---

## FUTURE ENHANCEMENTS

### V1.1: Jinja2 Templates
- More complex logic in templates
- Conditional sections
- Loops for repeated config

### V1.2: Config Validation
- JSON schema for .env
- Pre-flight checks
- Automated testing

### V2: Multi-Tenant SaaS
- Database per tenant
- Shared infrastructure
- Tenant isolation

---

## RELATED SKILLS

- SKILL 9: Installer Specs (practical implementation)
- SKILL 10: Deployment Workflow (with brand-aware .env)
- SKILL 3: Coding Standards (fail-fast patterns)
- SKILL 2: Architecture (ENV-driven design)

---

## CRITICAL SUCCESS FACTORS

1. **Zero Brand Strings in Repository**
   - Verify: `grep -r "Specific Brand" . | grep -v ".template\|.example"`
   - Result: Should be empty

2. **Fail-Fast Without .env**
   - Test: Import code without .env
   - Result: Clear ValueError with guidance

3. **Idempotent FASE 0**
   - Test: Run FASE 0 twice
   - Result: No errors, same config

4. **Multi-Brand Deployments Work**
   - Test: Same repo, different .env files
   - Result: Independent branded instances
