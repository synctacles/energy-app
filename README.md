# SYNCTACLES Development Repository

Development workspace for SYNCTACLES energy data aggregation platform.

## Quick Start

```bash
cd /opt/github/synctacles-repo
source venv/bin/activate
pip install -r requirements.txt
python test_setup.py
```

## Directory Structure

```
sparkcrawler/          # Raw data collection engine
  ├── parsers/         # XML/JSON parsers per data source
  ├── collectors/      # API collectors
  └── models/          # Data models

synctacles/            # Normalized API layer
  ├── api/             # FastAPI endpoints
  ├── models/          # Database models
  └── services/        # Business logic

tests/                 # Unit & integration tests
migrations/            # Database schema versions (Alembic)
config/                # Configuration files
logs/                  # Application logs
```

## Environment Setup

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your API keys

3. Install dependencies:
   ```bash
   source venv/bin/activate
   pip install -r requirements.txt
   ```

4. Test setup:
   ```bash
   python test_setup.py
   ```

## Development Workflow

```bash
# Activate venv
source venv/bin/activate

# Make changes
# ... code here ...

# Test locally
python test_setup.py

# Commit
git add .
git commit -m "Feature: description"
git push
```

## Resources

- Production: `/opt/synctacles/`
- Logs: `/var/log/synctacles-setup/`
