# Contributing to SYNCTACLES

Thank you for your interest in contributing!

## Quick Start

```bash
# Clone repository
git clone git@github.com:synctacles/backend.git
cd backend

# Setup Python environment
python3.12 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Run tests
pytest tests/ -v
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout main
git pull origin main
git checkout -b feature/your-feature-name
```

Branch naming:
- `feature/` - New functionality
- `fix/` - Bug fixes
- `docs/` - Documentation
- `refactor/` - Code improvements

### 2. Make Changes

- Follow existing code style
- Add tests for new functionality
- Update documentation if needed

### 3. Test Locally

```bash
# Run all tests
pytest tests/ -v

# Run specific test file
pytest tests/test_api.py -v

# Run with coverage
pytest tests/ --cov=synctacles_db
```

### 4. Commit

```bash
git add .
git commit -m "feat: add new feature description"
```

Commit message format:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance

### 5. Push & Create PR

```bash
git push -u origin feature/your-feature-name
gh pr create --title "Feature: Description" --body "Details..."
```

## Code Style

### Python

- Python 3.12+
- Use type hints
- Follow PEP 8
- Docstrings for public functions

```python
def get_prices(hours: int = 24) -> list[dict]:
    """
    Fetch electricity prices.

    Args:
        hours: Number of hours to fetch (default 24)

    Returns:
        List of price dictionaries with timestamp and price_eur_mwh
    """
    ...
```

### Configuration

- No hardcoded credentials
- Use `config.settings` for configuration
- Environment variables via `/opt/.env`

## Project Structure

```
synctacles_db/
├── api/                 # FastAPI endpoints
│   ├── endpoints/       # Route handlers
│   ├── routes/          # Router configuration
│   └── main.py          # App entry point
├── clients/             # External API clients
├── collectors/          # Data collection scripts
├── importers/           # Data import scripts
├── normalizers/         # Data normalization
├── models/              # SQLAlchemy models
└── config/              # Configuration

tests/                   # Test files
docs/                    # Documentation
scripts/                 # Utility scripts
systemd/                 # Service templates
```

## Testing

### Running Tests

```bash
# All tests
pytest tests/ -v

# Specific test
pytest tests/test_api.py::TestHealthEndpoint -v

# With output
pytest tests/ -v -s
```

### Writing Tests

```python
def test_health_endpoint(client):
    """Test health endpoint returns 200."""
    response = client.get("/health")
    assert response.status_code == 200
    assert "status" in response.json()
```

## Database

### Migrations

We use Alembic for database migrations:

```bash
# Create migration
alembic revision --autogenerate -m "Add new table"

# Apply migrations
alembic upgrade head

# Rollback
alembic downgrade -1
```

### Models

SQLAlchemy models in `synctacles_db/models/`:

```python
from sqlalchemy import Column, String, Float, DateTime
from synctacles_db.models.base import Base

class PriceData(Base):
    __tablename__ = "price_data"

    timestamp = Column(DateTime, primary_key=True)
    price_eur_mwh = Column(Float, nullable=False)
```

## Questions?

- Check existing issues on GitHub
- Review documentation in `docs/`
- Ask in PR comments
