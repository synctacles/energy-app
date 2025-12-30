# SKILL 3 — CODING STANDARDS

Code Quality, Style, and Best Practices
Version: 1.0 (2025-12-30)

---

## PURPOSE

Define the specific code-level standards that implement SKILL 1 (Hard Rules) and SKILL 2 (Architecture). These standards ensure consistency, readability, maintainability, and adherence to fail-fast principles.

---

## GENERAL PRINCIPLES

### 1. Readability Over Cleverness

Code should be readable by a junior developer without explanation.

**Good:**
```python
def calculate_generation_quality(raw_data):
    """Calculate quality score for generation data (0.0-1.0)."""
    if not raw_data:
        return 0.0

    total_values = len(raw_data)
    complete_values = sum(1 for r in raw_data if r.value is not None)
    completeness = complete_values / total_values

    if completeness < 0.5:
        return 0.0  # Too incomplete to be useful

    # Age penalty: older data is less valuable
    age_minutes = (datetime.now() - raw_data[0].timestamp).total_seconds() / 60
    age_penalty = min(age_minutes / 60, 1.0)  # 0 min = 0 penalty, 60+ min = full penalty

    return completeness * (1 - age_penalty)
```

**Bad (too clever):**
```python
def cq(d):
    return 0 if not d else (((l := len(d)) and (sum(1 for r in d if r.v) / l)) or 0) * max(0, 1 - (((datetime.now() - d[0].t).total_seconds() / 60) / 60))
```

### 2. Explicit is Better Than Implicit

Code should show intent clearly.

**Good:**
```python
def validate_config():
    """Validate all required environment variables at startup."""
    required = {
        'BRAND_NAME': os.getenv('BRAND_NAME'),
        'DB_HOST': os.getenv('DB_HOST'),
        'API_KEY': os.getenv('API_KEY'),
    }

    missing = [k for k, v in required.items() if not v]
    if missing:
        raise ValueError(
            f"Missing required variables: {', '.join(missing)}\n"
            f"Set them in /opt/.env or environment"
        )
    return required
```

**Bad (implicit):**
```python
def validate_config():
    return {k: os.getenv(k) for k in ['BRAND_NAME', 'DB_HOST', 'API_KEY']}
    # Silently returns None for missing values!
```

### 3. Fail-Fast Error Handling

Errors should be caught early with clear messages.

**Good:**
```python
def import_generation_data(xml_file):
    """Import generation data from XML file."""
    if not os.path.exists(xml_file):
        raise FileNotFoundError(
            f"XML file not found: {xml_file}\n"
            f"Collector should save raw data to /var/log/*/collectors/raw/"
        )

    try:
        tree = ET.parse(xml_file)
    except ET.ParseError as e:
        raise ValueError(
            f"Invalid XML in {xml_file}: {e}\n"
            f"Check collector logs for fetch errors"
        )

    # Process successfully parsed tree
    return process_tree(tree)
```

**Bad (silent failure):**
```python
def import_generation_data(xml_file):
    try:
        tree = ET.parse(xml_file)
        return process_tree(tree)
    except:
        return None  # Silently fails, upstream doesn't know
```

---

## PYTHON STYLE

### Style Guide

Follow PEP 8 with these clarifications:

- **Line length:** 100 characters (not 79, realistic for modern screens)
- **Indentation:** 4 spaces (not tabs)
- **Imports:** Alphabetical within groups (stdlib, third-party, local)
- **Naming:** snake_case for functions/variables, PascalCase for classes

### Imports Organization

```python
# stdlib imports
import os
import sys
from datetime import datetime, timedelta
from pathlib import Path

# third-party imports
import psycopg2
import requests
from pydantic import BaseModel, ValidationError

# local imports
from synctacles_db.config import settings
from synctacles_db.models import RawData
from synctacles_db.utils import parse_xml
```

### Function Docstrings

Every function (except trivial 1-liners) must have a docstring.

**Good:**
```python
def normalize_generation(raw_table: str) -> Dict[str, float]:
    """
    Normalize raw generation data with quality metadata.

    Args:
        raw_table: Name of raw table (e.g., 'raw_entso_e_a75')

    Returns:
        Dictionary with normalized generation by PSR type

    Raises:
        ValueError: If raw_table is empty
        DatabaseError: If database query fails

    Example:
        >>> norm = normalize_generation('raw_entso_e_a75')
        >>> norm['solar']
        450.2
    """
```

**Bad:**
```python
def normalize_generation(raw_table: str):
    # normalizaes gen data
    ...

def normalize_generation(raw_table):
    """Normalize generation."""  # Too vague
    ...
```

### Comments

Comments should explain **WHY**, not **WHAT**.

**Good:**
```python
# Reject data older than 4 hours: ENTSO-E publishes every 15 min,
# so > 4 hours indicates a collection failure. Better to use fallback.
if age_minutes > 240:
    return fallback_data()
```

**Bad:**
```python
age_minutes = (datetime.now() - timestamp).total_seconds() / 60  # Calculate age in minutes
if age_minutes > 240:  # If older than 240 minutes
    return fallback_data()  # Return fallback
```

---

## CONFIGURATION MANAGEMENT

### Environment Variables (Fail-Fast Pattern)

**Structure:**
```python
# config/settings.py
import os
from pathlib import Path

class Settings:
    """Load and validate configuration from environment."""

    def __init__(self):
        """Initialize settings, fail-fast if missing critical config."""
        # Required: Brand configuration
        self.brand_name = self._get_required('BRAND_NAME')
        self.brand_slug = self._get_required('BRAND_SLUG')
        self.brand_domain = self._get_required('BRAND_DOMAIN')

        # Required: Database
        self.db_host = self._get_required('DB_HOST')
        self.db_port = self._get_required('DB_PORT')
        self.db_name = self._get_required('DB_NAME')
        self.db_user = self._get_required('DB_USER')
        self.db_password = self._get_required('DB_PASSWORD')

        # Optional: API (with reasonable defaults)
        self.api_host = os.getenv('API_HOST', '0.0.0.0')
        self.api_port = int(os.getenv('API_PORT', '8000'))
        self.log_level = os.getenv('LOG_LEVEL', 'INFO')

    def _get_required(self, name: str) -> str:
        """Get required environment variable, raise if missing."""
        value = os.getenv(name)
        if not value:
            raise ValueError(
                f"Required environment variable '{name}' not set.\n"
                f"Run setup script FASE 0 or set in /opt/.env"
            )
        return value

    def db_url(self) -> str:
        """Generate PostgreSQL connection URL."""
        return (
            f"postgresql://{self.db_user}:{self.db_password}"
            f"@{self.db_host}:{self.db_port}/{self.db_name}"
        )

# Singleton: imported modules get validated config
settings = Settings()
```

**Usage:**
```python
# In any module
from synctacles_db.config import settings

# Guaranteed to exist and be valid
db_url = settings.db_url()
api_port = settings.api_port
```

### Constants Defined at Module Level

```python
# const.py
import os
from synctacles_db.config import settings

# Derived from settings, computed once at import time
DB_URL = settings.db_url()
SERVICE_NAME = f"{settings.brand_slug}-api"
HA_COMPONENT = settings.brand_name  # For Home Assistant manifest

# Constants
MAX_AGE_MINUTES = 240  # Reject data older than 4 hours
QUALITY_THRESHOLD = 0.6  # Use fallback if quality < 0.6
ENTSO_E_TIMEOUT = 30  # Request timeout in seconds
```

---

## ERROR HANDLING

### Exception Types

Use specific exceptions:

```python
# Good: specific, informative
raise ValueError("BRAND_NAME required in .env")
raise FileNotFoundError(f"Collector output: {expected_path}")
raise ConnectionError("Database unreachable at localhost:5432")

# Bad: too generic
raise Exception("Something went wrong")
raise RuntimeError("Error in import")
```

### Error Messages

Every error message must guide to solution:

```python
# Good: explains problem + solution
if not manifest_path.exists():
    raise FileNotFoundError(
        f"manifest.json not found at {manifest_path}\n"
        f"This file is generated from manifest.json.template.\n"
        f"Run FASE 0 setup script to generate it:\n"
        f"  sudo ./scripts/setup/setup.sh fase0"
    )

# Bad: vague
raise FileNotFoundError("manifest.json missing")
```

### Try/Except Guidance

Use try/except sparingly, at boundaries:

```python
# Good: Catch at API boundary, transform to user-friendly error
@app.get("/v1/generation/current")
async def get_generation():
    try:
        data = normalize_generation()
        return JSONResponse(data)
    except DatabaseError as e:
        logger.error(f"Database error: {e}")
        return JSONResponse(
            {"error": "Database unavailable", "status": 503},
            status_code=503
        )

# Bad: Catch everywhere, hiding errors
def get_data():
    try:
        data = normalize_generation()
    except:
        data = None  # Silent failure!
    return data
```

---

## DATABASE CODE

### Query Building

Use parameterized queries (prevent SQL injection):

**Good:**
```python
import psycopg2

conn = psycopg2.connect(settings.db_url())
cur = conn.cursor()

# Parameterized query (safe)
cur.execute(
    "SELECT * FROM raw_entso_e_a75 WHERE psr_type = %s AND import_timestamp > %s",
    (psr_type, cutoff_time)
)
results = cur.fetchall()
```

**Bad:**
```python
# String interpolation (SQL injection risk!)
cur.execute(f"SELECT * FROM raw_entso_e_a75 WHERE psr_type = '{psr_type}'")
```

### ORM vs Raw SQL

Use ORM (SQLAlchemy) for application code, raw SQL only for migrations:

```python
# Application: use ORM
from sqlalchemy import create_engine, Column, Integer, String
from sqlalchemy.orm import sessionmaker

engine = create_engine(settings.db_url())
Session = sessionmaker(bind=engine)

class RawEntsoEA75(Base):
    __tablename__ = 'raw_entso_e_a75'
    id = Column(Integer, primary_key=True)
    psr_type = Column(String(50))
    value_mw = Column(Float)
    source_timestamp = Column(DateTime)

# Query is safe and readable
session = Session()
data = session.query(RawEntsoEA75).filter(
    RawEntsoEA75.psr_type == 'Solar'
).all()
```

### Schema Documentation

Every table comment explains purpose:

```python
# migrations/versions/001_create_raw_tables.py
def upgrade():
    op.create_table(
        'raw_entso_e_a75',
        sa.Column('id', sa.Integer, primary_key=True),
        sa.Column('psr_type', sa.String(50)),
        sa.Column('value_mw', sa.Float),
        sa.Column('source_timestamp', sa.DateTime),
        sa.Column('import_timestamp', sa.DateTime),
        sa.Column('file_source', sa.String(255)),
        comment="Raw ENTSO-E A75 data (generation by fuel type). "
                "Imported as-is from XML, quality checking in normalizers."
    )
```

---

## TESTING STANDARDS

### Test Organization

```
tests/
├── unit/
│   ├── test_config.py          # Config loading
│   ├── test_normalizers.py     # Data transformation
│   └── test_api.py             # API endpoints
├── integration/
│   ├── test_pipeline.py        # Full pipeline
│   └── test_database.py        # Database operations
└── fixtures/
    ├── sample_entso_e.xml      # Sample raw data
    └── sample_config.env       # Test configuration
```

### Test Naming

```python
# Good: describes what's being tested
def test_config_fails_without_brand_name():
    with pytest.raises(ValueError, match="BRAND_NAME"):
        Settings()

def test_normalize_generation_quality_score():
    norm = normalize_generation(raw_data)
    assert 0.0 <= norm['quality'] <= 1.0

def test_api_returns_404_for_missing_endpoint():
    response = client.get('/v1/nonexistent')
    assert response.status_code == 404

# Bad: vague
def test_config():
    ...

def test_generation():
    ...

def test_api():
    ...
```

### Coverage Requirements

- Overall: >= 80%
- Core logic (normalizers, API): >= 90%
- Configuration/constants: >= 70%
- Tests must verify fail-fast behavior

**Example: Testing fail-fast**
```python
def test_missing_env_var_raises_error():
    """Verify fail-fast when BRAND_NAME missing."""
    import os
    os.environ.pop('BRAND_NAME', None)

    with pytest.raises(ValueError) as excinfo:
        from synctacles_db.config import Settings
        Settings()

    assert "BRAND_NAME" in str(excinfo.value)
    assert "FASE 0" in str(excinfo.value)  # Helpful message
```

---

## LOGGING

### Logging Configuration

```python
# config/logging.py
import logging
import os

LOG_LEVEL = os.getenv('LOG_LEVEL', 'INFO')

logging.basicConfig(
    level=LOG_LEVEL,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

def get_logger(name):
    return logging.getLogger(name)
```

### Log Usage

```python
# Good: informative at appropriate levels
logger = get_logger(__name__)

def import_entso_e_data(xml_file):
    logger.debug(f"Starting import of {xml_file}")

    try:
        tree = ET.parse(xml_file)
        logger.info(f"Parsed {xml_file}, found {len(data)} records")
    except ET.ParseError as e:
        logger.error(f"Failed to parse {xml_file}: {e}")
        raise

    # Don't log secrets!
    logger.debug(f"Inserting {len(data)} records to database")

    return imported_count
```

**Log levels:**
- **DEBUG:** Detailed info for debugging (file paths, record counts)
- **INFO:** Normal operations (started, completed)
- **WARNING:** Something unexpected (fallback activated, old data used)
- **ERROR:** Something failed (import failed, database error)
- **CRITICAL:** System can't continue (misconfiguration, DB down)

---

## SECURITY

### Secrets Management

**Good:**
```python
# secrets stored in .env (never committed)
API_KEY = os.getenv('API_KEY')
DB_PASSWORD = os.getenv('DB_PASSWORD')
ENTSO_E_TOKEN = os.getenv('ENTSO_E_TOKEN')

# Use them
headers = {'Authorization': f'Bearer {API_KEY}'}
```

**Bad:**
```python
# Hardcoded secrets (CRITICAL SECURITY ISSUE)
API_KEY = "sk_live_abc123def456"
DB_PASSWORD = "postgres"
ENTSO_E_TOKEN = "secret_token_here"
```

### Never Log Secrets

```python
# Good: redact sensitive info
logger.debug(f"API key: {api_key[:8]}...")
logger.debug(f"DB password: ***")

# Bad: logs full secret
logger.debug(f"Using API key: {api_key}")
logger.debug(f"Database password: {db_password}")
```

---

## FILE OPERATIONS

### Path Handling

Use `pathlib.Path` not string paths:

```python
# Good: cross-platform, safe
from pathlib import Path

log_dir = Path(settings.log_path)
log_dir.mkdir(parents=True, exist_ok=True)
log_file = log_dir / "collectors.log"

# Good: relative imports
config_path = Path(__file__).parent / "config.json"

# Bad: string paths are fragile
log_file = f"{settings.log_path}/collectors.log"
config_path = "config/config.json"  # Breaks if run from wrong directory
```

### File Validation

```python
def load_raw_xml(xml_file):
    """Load and validate raw XML file."""
    xml_path = Path(xml_file)

    # Validate existence
    if not xml_path.exists():
        raise FileNotFoundError(f"XML file not found: {xml_path}")

    # Validate is file, not directory
    if not xml_path.is_file():
        raise ValueError(f"Expected file, got directory: {xml_path}")

    # Validate readable
    if not os.access(xml_path, os.R_OK):
        raise PermissionError(f"Cannot read file: {xml_path}")

    # Read and parse
    return ET.parse(xml_path)
```

---

## CODE REVIEW CHECKLIST

Before submitting PR:

```
Readability & Style:
□ PEP 8 compliant (use black/flake8)
□ Functions have docstrings
□ Comments explain WHY, not WHAT
□ Naming is clear (no abbreviations)

Configuration:
□ No hardcoded config values
□ All config from environment
□ Fail-fast validation at startup
□ Error messages guide to solution

Error Handling:
□ Specific exceptions (not generic)
□ All errors have actionable messages
□ No silent failures (no empty except:)
□ Errors logged before re-raising

Database:
□ Parameterized queries (SQL injection safe)
□ ORM used for application code
□ Schema documented
□ Migrations tested

Security:
□ No secrets logged
□ No hardcoded API keys
□ No secrets in git
□ Permissions validated

Testing:
□ Tests for non-trivial logic
□ Fail-fast behavior tested
□ Clear test names
□ Coverage >= threshold

Documentation:
□ README updated if needed
□ Complex logic documented
□ ADRs created for major changes
```

---

## TOOLS & AUTOMATION

### Code Formatting

```bash
# Auto-format with black
pip install black
black synctacles_db/

# Check style with flake8
pip install flake8
flake8 synctacles_db/

# Sort imports with isort
pip install isort
isort synctacles_db/
```

### Testing

```bash
# Run all tests with coverage
pytest --cov=synctacles_db tests/

# Run specific test
pytest tests/unit/test_config.py::test_config_fails_without_brand_name
```

### Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/psf/black
    rev: 23.1.0
    hooks:
      - id: black

  - repo: https://github.com/PyCQA/flake8
    rev: 5.0.4
    hooks:
      - id: flake8
```

---

## RELATED SKILLS

- **SKILL 1**: Hard Rules (standards enforce these)
- **SKILL 2**: Architecture (how code is organized)
- **SKILL 9**: Installer (templates use these patterns)
- **SKILL 12**: Brand-Free (env configuration patterns)
