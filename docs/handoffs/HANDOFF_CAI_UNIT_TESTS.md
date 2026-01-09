# HANDOFF: Unit Test Suite Uitbreiden

**Van:** Claude Code (Opus)
**Naar:** Claude AI (Sonnet)
**Datum:** 2026-01-09
**GitHub Issue:** #5 [CRITICAL]

---

## CONTEXT

Stabiliteitscan toonde dat test coverage de grootste gap is:
- Slechts 3 testbestanden (~167 LOC)
- Geen pytest.ini configuratie
- Geen coverage rapportage
- Geen test workflow in CI/CD

Dit is een **critical blocker** voor launch (zie issue #5).

---

## HUIDIGE STAAT

```
tests/
├── __init__.py
├── test_setup.py              # 43 lines - basic setup tests
├── test_ha_component.py       # 71 lines - HA component tests
├── unit/
│   ├── __init__.py
│   ├── test_signals_standalone.py  # 53 lines
│   └── integration_snippet.py
├── integration/
│   └── __init__.py
└── scripts/
    ├── test_backfill.py
    ├── backfill_test_raw.py
    └── audit_zero_values.py
```

---

## OPDRACHT

### Fase 1: Configuratie

1. **Maak pytest.ini:**
```ini
[pytest]
testpaths = tests
python_files = test_*.py
python_classes = Test*
python_functions = test_*
addopts = -v --tb=short
filterwarnings = ignore::DeprecationWarning
```

2. **Maak conftest.py met fixtures:**
```python
# tests/conftest.py
import pytest
from unittest.mock import MagicMock

@pytest.fixture
def mock_db_session():
    """Mock database session"""
    session = MagicMock()
    yield session
    session.close()

@pytest.fixture
def sample_generation_data():
    """Sample generation mix data met NULLs"""
    return {
        "timestamp": "2026-01-09T10:00:00+00:00",
        "solar_mw": 500.0,
        "wind_offshore_mw": 1200.0,
        "wind_onshore_mw": None,  # NULL
        "nuclear_mw": None,       # NULL
        "gas_mw": 3000.0,
        "total_mw": 8000.0
    }

@pytest.fixture
def sample_price_data():
    """Sample price data"""
    return [
        {"timestamp": "2026-01-09T10:00:00+00:00", "price_eur_mwh": 85.50},
        {"timestamp": "2026-01-09T11:00:00+00:00", "price_eur_mwh": 92.30},
    ]
```

### Fase 2: Core Unit Tests

Prioriteit op basis van kritieke componenten:

#### 2.1 FallbackManager Tests (HOOGSTE PRIORITEIT)
```python
# tests/unit/test_fallback_manager.py
"""Tests voor FallbackManager - het hart van data resilience."""

import pytest
from datetime import datetime, timezone
from synctacles_db.fallback.fallback_manager import FallbackManager, _ec_circuit_breaker

class TestCircuitBreaker:
    """Circuit breaker functionaliteit"""

    def test_circuit_breaker_initially_closed(self):
        """Circuit breaker start gesloten"""
        _ec_circuit_breaker["last_404_time"] = None
        assert FallbackManager._check_circuit_breaker() == False

    def test_circuit_breaker_opens_after_404(self):
        """Circuit breaker opent na 404"""
        FallbackManager._open_circuit_breaker()
        assert _ec_circuit_breaker["is_open"] == True
        assert FallbackManager._check_circuit_breaker() == True

    def test_circuit_breaker_cooldown_2_hours(self):
        """Cooldown is 2 uur"""
        assert _ec_circuit_breaker["cooldown_minutes"] == 120

class TestRenewableCalculation:
    """Renewable percentage berekeningen"""

    def test_calculate_renewable_percentage_normal(self):
        """Normale berekening"""
        data = {
            "solar_mw": 500,
            "wind_offshore_mw": 300,
            "wind_onshore_mw": 200,
            "biomass_mw": 100,
            "total_mw": 2000
        }
        result = FallbackManager.calculate_renewable_percentage(data)
        assert result == 55.0  # (500+300+200+100)/2000 * 100

    def test_calculate_renewable_percentage_with_nulls(self):
        """Berekening met NULL waarden"""
        data = {
            "solar_mw": None,
            "wind_offshore_mw": 1000,
            "total_mw": 2000
        }
        result = FallbackManager.calculate_renewable_percentage(data)
        assert result == 50.0

    def test_calculate_renewable_percentage_zero_total(self):
        """Zero total geeft None"""
        data = {"solar_mw": 100, "total_mw": 0}
        result = FallbackManager.calculate_renewable_percentage(data)
        assert result is None

class TestNullFieldDetection:
    """NULL veld detectie"""

    def test_find_null_fields_some_nulls(self, sample_generation_data):
        """Vindt NULL velden correct"""
        nulls = FallbackManager._find_null_fields(sample_generation_data)
        assert "wind_onshore_mw" in nulls
        assert "nuclear_mw" in nulls
        assert "solar_mw" not in nulls

    def test_find_null_fields_no_nulls(self):
        """Geen NULLs geeft lege lijst"""
        data = {
            "solar_mw": 100,
            "wind_offshore_mw": 200,
            "gas_mw": 300
        }
        nulls = FallbackManager._find_null_fields(data)
        # Alleen PSR velden worden gecheckt
        assert len([n for n in nulls if n in data]) == 0

class TestKnownCapacity:
    """Known capacity filling"""

    def test_fill_nuclear_with_known_capacity(self):
        """Nuclear wordt gevuld met 485 MW (Borssele)"""
        data = {"nuclear_mw": None, "timestamp": "2026-01-09T12:00:00+00:00"}
        filled, sources = FallbackManager._fill_with_known_capacity(data)
        assert filled["nuclear_mw"] == 485.0
        assert sources["nuclear_mw"] == "Known Capacity"

    def test_solar_estimation_night(self):
        """Solar is 0 's nachts"""
        result = FallbackManager._estimate_solar_nl("2026-01-09T02:00:00+00:00")
        assert result == 0.0

    def test_solar_estimation_midday(self):
        """Solar > 0 overdag"""
        result = FallbackManager._estimate_solar_nl("2026-01-09T13:00:00+00:00")
        assert result > 0
```

#### 2.2 API Endpoint Tests
```python
# tests/unit/test_api_endpoints.py
"""Tests voor API endpoints."""

import pytest
from fastapi.testclient import TestClient
from synctacles_db.api.main import app

client = TestClient(app)

class TestHealthEndpoint:
    """Health endpoint tests"""

    def test_health_returns_ok(self):
        """Health endpoint geeft status ok"""
        response = client.get("/health")
        assert response.status_code == 200
        assert response.json()["status"] == "ok"

    def test_health_includes_version(self):
        """Health bevat versie info"""
        response = client.get("/health")
        assert "version" in response.json()

class TestPricesEndpoint:
    """Prices endpoint tests"""

    def test_prices_now_returns_200(self):
        """Prices/now endpoint bereikbaar"""
        response = client.get("/api/v1/prices/now")
        assert response.status_code in [200, 503]  # 503 als geen data

    def test_prices_today_returns_list(self):
        """Prices/today geeft lijst"""
        response = client.get("/api/v1/prices/today")
        if response.status_code == 200:
            assert isinstance(response.json(), list)
```

#### 2.3 Normalizer Tests
```python
# tests/unit/test_normalizers.py
"""Tests voor data normalizers."""

import pytest
from synctacles_db.normalizers.base import validate_db_connection

class TestDatabaseValidation:
    """Database connectie validatie"""

    def test_validate_db_connection_success(self, monkeypatch):
        """Succesvolle DB connectie"""
        # Mock de engine
        from unittest.mock import MagicMock, patch

        mock_engine = MagicMock()
        mock_conn = MagicMock()
        mock_engine.connect.return_value.__enter__ = lambda s: mock_conn
        mock_engine.connect.return_value.__exit__ = MagicMock(return_value=False)

        with patch('synctacles_db.normalizers.base.create_engine', return_value=mock_engine):
            result = validate_db_connection()
            assert result is not None

    def test_validate_db_connection_failure_exits(self, monkeypatch):
        """DB failure geeft SystemExit(1)"""
        from unittest.mock import patch

        with patch('synctacles_db.normalizers.base.create_engine', side_effect=Exception("Connection refused")):
            with pytest.raises(SystemExit) as exc_info:
                validate_db_connection()
            assert exc_info.value.code == 1
```

### Fase 3: GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: "Run Tests"

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_pass
          POSTGRES_DB: test_db
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.12'
          cache: 'pip'

      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install pytest pytest-cov pytest-asyncio

      - name: Run tests with coverage
        env:
          DATABASE_URL: postgresql://test_user:test_pass@localhost:5432/test_db
        run: |
          pytest --cov=synctacles_db --cov-report=xml --cov-report=term-missing

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.xml
```

---

## VERWACHTE DELIVERABLES

1. `pytest.ini` - pytest configuratie
2. `tests/conftest.py` - shared fixtures
3. `tests/unit/test_fallback_manager.py` - FallbackManager tests
4. `tests/unit/test_api_endpoints.py` - API endpoint tests
5. `tests/unit/test_normalizers.py` - Normalizer tests
6. `.github/workflows/test.yml` - CI workflow

---

## VERIFICATIE

```bash
# Run tests lokaal
cd /opt/github/synctacles-api
source /opt/energy-insights-nl/venv/bin/activate
pip install pytest pytest-cov pytest-asyncio
pytest -v

# Check coverage
pytest --cov=synctacles_db --cov-report=term-missing
```

---

## PRIORITEIT

**CRITICAL** - Dit is een launch blocker (issue #5).

Minimale coverage target: 60% voor core modules:
- `fallback/fallback_manager.py`
- `api/endpoints/*.py`
- `normalizers/base.py`

---

## GIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add -A
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "test: implement unit test suite for core modules

- Add pytest.ini configuration
- Add conftest.py with shared fixtures
- Add FallbackManager tests (circuit breaker, renewable calc)
- Add API endpoint tests (health, prices)
- Add normalizer tests (DB validation)
- Add GitHub Actions test workflow

Closes #5"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```
