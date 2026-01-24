# Local Development Setup

Complete guide for setting up SYNCTACLES development environment.

## Prerequisites

- Ubuntu 22.04+ or macOS
- Python 3.12
- PostgreSQL 16 with TimescaleDB
- Git

## Step 1: Clone Repository

```bash
git clone git@github.com:synctacles/backend.git
cd backend
```

## Step 2: Python Environment

```bash
# Create virtual environment
python3.12 -m venv venv

# Activate
source venv/bin/activate

# Install dependencies
pip install --upgrade pip
pip install -r requirements.txt

# Verify
python -c "import fastapi; print(f'FastAPI {fastapi.__version__}')"
```

## Step 3: PostgreSQL + TimescaleDB

### Ubuntu

```bash
# Install PostgreSQL 16
sudo apt install postgresql-16 postgresql-client-16

# Install TimescaleDB
sudo apt install timescaledb-2-postgresql-16
sudo timescaledb-tune --quiet --yes

# Start service
sudo systemctl start postgresql
```

### macOS

```bash
brew install postgresql@16 timescaledb
brew services start postgresql@16
```

### Create Database

```bash
# Create user and database
sudo -u postgres psql << 'SQL'
CREATE USER synctacles WITH PASSWORD 'dev_password';
CREATE DATABASE synctacles OWNER synctacles;
\c synctacles
CREATE EXTENSION IF NOT EXISTS timescaledb;
GRANT ALL PRIVILEGES ON DATABASE synctacles TO synctacles;
SQL
```

## Step 4: Environment Configuration

Create `/opt/.env` (or local `.env` for development):

```bash
sudo tee /opt/.env << 'EOF'
BRAND_NAME=SYNCTACLES
BRAND_SLUG=synctacles
BRAND_DOMAIN=localhost

INSTALL_PATH=/opt/synctacles
APP_PATH=/opt/github/synctacles-api
LOG_PATH=/var/log/synctacles

DB_HOST=localhost
DB_PORT=5432
DB_NAME=synctacles
DB_USER=synctacles
DATABASE_URL=postgresql://synctacles:dev_password@localhost:5432/synctacles

API_HOST=0.0.0.0
API_PORT=8000
EOF
```

## Step 5: Database Migrations

```bash
# Activate venv
source venv/bin/activate

# Set environment
export DATABASE_URL="postgresql://synctacles:dev_password@localhost:5432/synctacles"

# Run migrations
alembic upgrade head
```

## Step 6: Run Development Server

```bash
# Activate venv
source venv/bin/activate

# Run with auto-reload
uvicorn synctacles_db.api.main:app --reload --host 0.0.0.0 --port 8000
```

Access:
- API: http://localhost:8000
- Docs: http://localhost:8000/docs
- Health: http://localhost:8000/health

## Step 7: Run Tests

```bash
# All tests
pytest tests/ -v

# With coverage
pytest tests/ --cov=synctacles_db --cov-report=term-missing

# Specific test
pytest tests/test_api.py -v
```

## IDE Setup

### VSCode

Recommended extensions:
- Python
- Pylance
- Python Test Explorer

`.vscode/settings.json`:
```json
{
    "python.defaultInterpreterPath": "${workspaceFolder}/venv/bin/python",
    "python.testing.pytestEnabled": true,
    "python.testing.pytestArgs": ["tests/"],
    "editor.formatOnSave": true
}
```

### PyCharm

1. Open project folder
2. Configure Python interpreter: `venv/bin/python`
3. Mark `synctacles_db` as Sources Root
4. Configure pytest as test runner

## Common Tasks

### Add New Endpoint

1. Create handler in `synctacles_db/api/endpoints/`
2. Add router in `synctacles_db/api/routes/`
3. Register in `synctacles_db/api/main.py`
4. Add tests in `tests/`

### Add Database Model

1. Create model in `synctacles_db/models/`
2. Generate migration: `alembic revision --autogenerate -m "Add model"`
3. Apply: `alembic upgrade head`

### Test Against PROD Data

```bash
# SSH tunnel to PROD database (read-only)
ssh -L 5433:localhost:5432 cc-hub -t "ssh synct-prod"

# In another terminal
export DATABASE_URL="postgresql://synctacles@localhost:5433/synctacles"
python -c "from synctacles_db.api.dependencies import get_db; ..."
```

## Troubleshooting

### "Module not found" errors

```bash
# Ensure venv is activated
source venv/bin/activate

# Add project root to PYTHONPATH
export PYTHONPATH="${PYTHONPATH}:$(pwd)"
```

### Database connection errors

```bash
# Check PostgreSQL is running
sudo systemctl status postgresql

# Test connection
psql -h localhost -U synctacles -d synctacles -c "SELECT 1;"
```

### Port already in use

```bash
# Find process
lsof -i :8000

# Kill it
kill -9 <PID>
```

## Next Steps

- Read [CONTRIBUTING.md](../CONTRIBUTING.md) for workflow
- Review [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- Check [api-reference.md](api-reference.md) for endpoints
