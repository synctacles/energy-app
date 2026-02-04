# BRAINS Server Architecture

**Server:** brains.synctacles.com (173.249.55.109)
**Last Updated:** 2026-02-04
**Status:** ✅ Production

## Overview

BRAINS is a single-purpose production server hosting the Knowledge Base system for Home Assistant community support. It runs a Telegram support bot, automated content harvesters, and local LLM inference using Ollama.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     BRAINS Server                            │
│                  brains.synctacles.com                       │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   ┌────▼────┐          ┌─────▼─────┐        ┌─────▼─────┐
   │ Support │          │ Harvesters│        │  Ollama   │
   │   Bot   │          │  (hourly) │        │    LLM    │
   └────┬────┘          └─────┬─────┘        └─────┬─────┘
        │                     │                     │
        └──────────┬──────────┴──────────┬──────────┘
                   │                     │
              ┌────▼─────────────────────▼────┐
              │    PostgreSQL 16 + pgvector    │
              │      brains_kb database        │
              │    17,297+ active KB entries   │
              └────────────────────────────────┘
```

## Components

### 1. OpenClaw Support Bot (`openclaw-support.service`)

**Purpose:** Telegram bot for responding to Home Assistant support questions

**Technology:**
- Python 3.12
- python-telegram-bot library
- Async/await architecture

**Service Configuration:**
- **Type:** `simple` (daemon)
- **User:** `brains`
- **WorkingDirectory:** `/opt/openclaw/harvesters`
- **ExecStart:** `/opt/openclaw/harvesters/venv/bin/python support_agent/main.py`
- **Environment:** `/opt/openclaw/harvesters/.env`
- **Restart:** `always` (10s delay)
- **Security:** `ProtectSystem=strict`, `NoNewPrivileges=true`, `PrivateTmp=true`
- **Resources:** `MemoryMax=512M`, `TasksMax=100`

**Features:**
- `/start` - Bot introduction
- `/help` - Command overview
- `/status` - System status
- `/faq [query]` - Search Knowledge Base
- `/analyze` - Analyze uploaded logs
- Document handler for log files

**Database Integration:**
- Uses `shared.knowledge_repository.KnowledgeRepository`
- Queries `kb.knowledge_base` table
- Tracks usage in `kb.knowledge_base_usage`
- Records feedback in `kb.knowledge_base_feedback`

**Telegram Configuration:**
- **Username:** @SynctaclesCareBot
- **Group ID:** -1003846489213
- **Topics:** 2 (support), 3 (monitoring)
- **Privacy Mode:** Enabled (sees mentions only)

### 2. KB Harvesters (`openclaw-harvest.service` + timer)

**Purpose:** Automated content harvesting from multiple sources

**Technology:**
- Python 3.12 async scanners
- GROQ API (free tier) for LLM processing
- Anthropic Claude API for advanced processing

**Service Configuration:**
- **Type:** `oneshot` (runs to completion)
- **User:** `brains`
- **ExecStart:** `/opt/openclaw/harvesters/venv/bin/python3 tools/scanners/run_harvest.py --backlog`
- **Restart:** `on-failure` (60s delay)
- **Security:** `ProtectSystem=strict`, `ProtectHome=read-only`
- **WritePaths:** `/opt/openclaw/harvesters`, `/opt/openclaw/logs`

**Timer Configuration:**
- **Schedule:** Hourly (at :00)
- **Persistent:** Yes (catches up missed runs)

**Scanners:**
1. **GitHub Scanner** (`tools/scanners/github_scanner.py`)
   - Scans home-assistant/core issues
   - Filters by labels (bug, integration, automation, etc.)
   - Extracts problem + solution from closed issues

2. **Forum Scanner** (`tools/scanners/forum_scanner.py`)
   - Scans community.home-assistant.io
   - Categories: configuration, installation, automation, etc.
   - Page limit per category to avoid overload

3. **Reddit Scanner** (`tools/scanners/reddit_scanner.py`)
   - Scans r/homeassistant subreddit
   - Extracts Q&A from posts and comments

4. **StackOverflow Scanner** (`tools/scanners/stackoverflow_scanner.py`)
   - Searches home-assistant tag
   - Extracts accepted answers

**State Management:**
- Uses `public.harvest_state` table
- Tracks cursor/last_item_id per scanner
- Prevents duplicate harvesting
- Tracks backlog completion status

**Logging:**
- **File:** `/opt/openclaw/logs/harvest.log`
- **Journal:** `journalctl -u openclaw-harvest`
- **Notifications:** Telegram group topic 3 on completion

### 3. Database (PostgreSQL 16 + pgvector)

**Database:** `brains_kb`

**Schemas:**

#### `kb` Schema (Knowledge Base)

**Tables:**

1. **`knowledge_base`** - Main KB entries
   ```sql
   id                      SERIAL PRIMARY KEY
   problem_title           TEXT NOT NULL
   problem_description     TEXT
   problem_category        VARCHAR(100)
   problem_component       VARCHAR(100)
   problem_keywords        TEXT[]
   solution_text           TEXT NOT NULL
   solution_steps          JSONB
   solution_code_snippets  JSONB
   confidence_score        DECIMAL(3,2) DEFAULT 0.5
   source                  VARCHAR(50)
   source_url              TEXT
   is_active               BOOLEAN DEFAULT true
   view_count              INTEGER DEFAULT 0
   helpful_count           INTEGER DEFAULT 0
   not_helpful_count       INTEGER DEFAULT 0
   last_used_at            TIMESTAMPTZ
   created_at              TIMESTAMPTZ DEFAULT NOW()
   updated_at              TIMESTAMPTZ DEFAULT NOW()
   ```

   **Indexes:**
   - `idx_kb_category` on `problem_category`
   - `idx_kb_component` on `problem_component`
   - `idx_kb_source` on `source`
   - `idx_kb_active` on `is_active`
   - Full-text search on `problem_title || ' ' || problem_description || ' ' || solution_text`

2. **`knowledge_base_categories`** - Category definitions
3. **`knowledge_base_feedback`** - User feedback tracking
4. **`knowledge_base_usage`** - Usage analytics

**Triggers:**
- `update_knowledge_base_timestamp` - Auto-updates `updated_at`
- `update_knowledge_base_feedback_counters` - Updates helpful/not_helpful counts
- `update_knowledge_base_usage_tracking` - Tracks view counts

#### `public` Schema

1. **`harvest_state`** - Scanner progress tracking
   ```sql
   id                    SERIAL PRIMARY KEY
   scanner_name          VARCHAR(50) UNIQUE NOT NULL
   last_cursor           TEXT
   last_item_id          TEXT
   last_item_timestamp   TIMESTAMPTZ
   backlog_start_date    DATE NOT NULL
   backlog_complete      BOOLEAN DEFAULT false
   backlog_completed_at  TIMESTAMPTZ
   total_harvested       INTEGER DEFAULT 0
   total_duplicates      INTEGER DEFAULT 0
   total_errors          INTEGER DEFAULT 0
   created_at            TIMESTAMPTZ DEFAULT NOW()
   updated_at            TIMESTAMPTZ DEFAULT NOW()
   ```

**Users:**
- `brains_admin` - Full access (used by bot and harvesters)
- `postgres` - Superuser

**Configuration:**
- `search_path = kb, public` (set at database level)
- Connection: `postgresql://brains_admin:***@localhost:5432/brains_kb?sslmode=disable`

### 4. Ollama (Local LLM Inference)

**Models:**
- `phi3:mini` - 2 GB - General purpose LLM
- `nomic-embed-text:latest` - 0.3 GB - Text embeddings

**Service:**
- **Port:** 11434
- **API:** REST API on localhost
- **Usage:** KB query processing, semantic search

**API Endpoints:**
- `http://localhost:11434/api/tags` - List models
- `http://localhost:11434/api/generate` - Text generation
- `http://localhost:11434/api/embeddings` - Vector embeddings

### 5. MCP Server (Model Context Protocol)

**Purpose:** KB search interface for OpenClaw agents

**Location:** `/opt/openclaw/mcp/kb-search.js`

**Technology:** Node.js 22.22.0 with @modelcontextprotocol/sdk

**Tools Provided:**
1. `search_kb` - Full-text and category search
2. `get_kb_entry` - Retrieve specific entry by ID
3. `list_categories` - List all KB categories
4. `search_by_component` - Component-specific search

**Database Connection:**
- Uses `DATABASE_URL` from environment
- Direct PostgreSQL connection via `pg` module

## File System Layout

```
/opt/openclaw/
├── harvesters/                  # Python application root
│   ├── venv/                   # Python virtual environment
│   ├── .env                    # Environment variables & secrets
│   ├── support_agent/          # Telegram bot code
│   │   ├── main.py            # Bot entry point
│   │   ├── handlers.py        # Command handlers
│   │   └── faq_handler.py     # KB query handler
│   ├── tools/                  # Harvester tools
│   │   └── scanners/
│   │       ├── run_harvest.py        # Main harvest runner
│   │       ├── base_scanner.py       # Abstract scanner class
│   │       ├── github_scanner.py
│   │       ├── forum_scanner.py
│   │       ├── reddit_scanner.py
│   │       └── stackoverflow_scanner.py
│   └── shared/                 # Shared libraries
│       ├── database.py        # Database connection pool
│       ├── secrets.py         # Secret management
│       ├── config.py          # Configuration
│       ├── logging_filter.py  # Token redaction
│       └── knowledge_repository.py  # KB query interface
├── mcp/                        # Model Context Protocol server
│   └── kb-search.js           # Node.js MCP server
└── logs/                       # Application logs
    └── harvest.log            # Harvest run logs

/etc/systemd/system/
├── openclaw-support.service    # Support bot service
├── openclaw-harvest.service    # Harvest service
└── openclaw-harvest.timer      # Hourly harvest timer
```

## Environment Variables

**File:** `/opt/openclaw/harvesters/.env`

```bash
# Environment
ENVIRONMENT=production

# Database
DATABASE_URL=postgresql://brains_admin:***@localhost:5432/brains_kb?sslmode=disable
DATABASE_POOL_SIZE=5
DATABASE_MAX_OVERFLOW=10

# Telegram
TELEGRAM_BOT_TOKEN_SUPPORT=8574419456:***
TELEGRAM_GROUP_ID=-1003846489213
TELEGRAM_TOPIC_SUPPORT=2
TELEGRAM_TOPIC_MONITORING=3

# GROQ API (free tier for LLM processing)
GROQ_API_KEY=gsk_***

# Anthropic Claude API (for advanced processing)
ANTHROPIC_API_KEY=sk-ant-api03-***

# GitHub (for scanning issues)
GITHUB_REPO_OWNER=home-assistant
GITHUB_REPO_NAME=core

# Logging
LOG_LEVEL=info
```

## Monitoring & Observability

### Systemd Service Status
```bash
systemctl status openclaw-support       # Support bot
systemctl status openclaw-harvest       # Current harvest run
systemctl status openclaw-harvest.timer # Timer status
systemctl status postgresql             # Database
systemctl status ollama                 # LLM inference
```

### Logs
```bash
# Support bot logs
sudo journalctl -u openclaw-support -f

# Harvest logs
sudo journalctl -u openclaw-harvest -f
tail -f /opt/openclaw/logs/harvest.log

# PostgreSQL logs
sudo journalctl -u postgresql -f
```

### Metrics
- **Prometheus Node Exporter:** `http://173.249.55.109:9100/metrics`
- **Scrape Interval:** 15s
- **Alerting:** Slack #critical-alerts via Alertmanager

### Database Monitoring
```sql
-- KB statistics
SELECT
  COUNT(*) as total_entries,
  COUNT(*) FILTER (WHERE is_active = true) as active_entries,
  COUNT(DISTINCT problem_category) as categories,
  ROUND(AVG(confidence_score)::numeric, 2) as avg_confidence
FROM kb.knowledge_base;

-- Harvest progress
SELECT
  scanner_name,
  backlog_complete,
  total_harvested,
  total_duplicates,
  last_item_timestamp
FROM public.harvest_state
ORDER BY scanner_name;

-- Most viewed entries
SELECT
  problem_title,
  view_count,
  helpful_count,
  problem_category
FROM kb.knowledge_base
WHERE is_active = true
ORDER BY view_count DESC
LIMIT 10;
```

## Deployment & Maintenance

### Service Management
```bash
# Restart support bot
sudo systemctl restart openclaw-support

# Manually trigger harvest
sudo systemctl start openclaw-harvest

# Stop/disable harvest timer
sudo systemctl stop openclaw-harvest.timer

# Enable harvest timer
sudo systemctl enable --now openclaw-harvest.timer
```

### Database Maintenance
```bash
# Backup database
sudo -u postgres pg_dump -d brains_kb -F c -f /backups/brains_kb_$(date +%Y%m%d).dump

# Restore database
sudo -u postgres pg_restore -d brains_kb -c /backups/brains_kb_20260204.dump

# Vacuum analyze
sudo -u postgres psql -d brains_kb -c "VACUUM ANALYZE;"

# Reset harvest state (if needed)
sudo -u postgres psql -d brains_kb -c "TRUNCATE public.harvest_state RESTART IDENTITY;"
```

### Code Updates
```bash
# Pull latest code from DEV (after testing)
cd /opt/openclaw/harvesters
git pull origin main  # Or copy files from DEV

# Reinstall dependencies if changed
source venv/bin/activate
pip install -r requirements.txt

# Restart services
sudo systemctl restart openclaw-support
sudo systemctl restart openclaw-harvest
```

## Security

### Hardening
- Services run as non-root `brains` user
- `ProtectSystem=strict` - Read-only /usr, /boot, /efi
- `ProtectHome=true` - No access to /home (except /home/brains)
- `NoNewPrivileges=true` - Cannot gain new privileges
- `PrivateTmp=true` - Private /tmp namespace
- Resource limits: `MemoryMax=512M`, `TasksMax=100`

### Secrets Management
- All secrets in `/opt/openclaw/harvesters/.env`
- File permissions: `chmod 600` (owner read/write only)
- Owner: `brains:brains`
- Loaded via `shared.secrets.py` with audit logging
- Token redaction in logs via `logging_filter.py`

### Network Security
- Database: localhost only (no external access)
- Ollama: localhost only (port 11434)
- Node Exporter: port 9100 (Prometheus only)
- SSH: Key-based authentication only

## Troubleshooting

### Support Bot Not Responding
1. Check service status: `systemctl status openclaw-support`
2. Check logs: `journalctl -u openclaw-support -n 50`
3. Common issues:
   - Database connection failed (check password in .env)
   - Telegram API conflict (another bot instance running)
   - Privacy mode (bot needs @mention in groups)

### Harvest Failing
1. Check service status: `systemctl status openclaw-harvest`
2. Check logs: `journalctl -u openclaw-harvest -n 50`
3. Common issues:
   - Missing `harvest_state` table
   - Database schema path not set
   - Read-only filesystem (check ReadWritePaths in service)
   - API rate limits (GROQ/Anthropic)

### Database Issues
1. Check PostgreSQL status: `systemctl status postgresql`
2. Check connections: `sudo -u postgres psql -d brains_kb -c "SELECT count(*) FROM pg_stat_activity;"`
3. Check disk space: `df -h /var/lib/postgresql`
4. Verify schema: `sudo -u postgres psql -d brains_kb -c "\\dn"`

### Performance Issues
1. Check resource usage: `systemctl status openclaw-support openclaw-harvest`
2. Check database size: `sudo -u postgres psql -d brains_kb -c "SELECT pg_size_pretty(pg_database_size('brains_kb'));"`
3. Check slow queries: Enable `log_min_duration_statement` in PostgreSQL
4. Vacuum/analyze: `sudo -u postgres psql -d brains_kb -c "VACUUM ANALYZE;"`

## Migration History

**Date:** 2026-02-04
**From:** DEV server (synctacles_dev database)
**To:** BRAINS server (brains_kb database)

**Migrated:**
- ✅ 18,413 KB entries from `kb.knowledge_base`
- ✅ KB schema (tables, triggers, functions)
- ✅ Python harvester code (support_agent, tools/scanners, shared)
- ✅ Environment configuration (.env with API keys)
- ✅ Telegram bot token (moltbot-support → openclaw-support)

**Not Migrated:**
- ❌ Platform monitoring bots (stayed on DEV)
- ❌ Dev-specific test data

**Post-Migration Issues Fixed:**
1. Database password placeholder → Generated secure password
2. SSL connection errors → Added `?sslmode=disable`
3. Telegram bot conflict → Stopped DEV bot
4. Harvest logs directory → Created `/opt/openclaw/logs`
5. Read-only filesystem → Added to `ReadWritePaths`
6. Missing `harvest_state` table → Migrated from DEV
7. Schema path issues → Set `search_path = kb, public`

## Performance Metrics

**Current (2026-02-04):**
- KB Entries: 17,297 active
- Categories: 24
- Avg Confidence: 0.78
- Support Bot Uptime: 14+ minutes
- Harvest Frequency: Hourly
- Database Size: ~500 MB
- Memory Usage: ~43 MB per service
- CPU Usage: <2% per service

## Future Enhancements

**Planned:**
- [ ] Automated database backups (daily cron job)
- [ ] Grafana dashboard for KB metrics
- [ ] Rate limiting for Telegram commands
- [ ] Semantic search using pgvector + embeddings
- [ ] Web UI for KB browsing
- [ ] A/B testing for different LLM models
- [ ] Community feedback loop integration
- [ ] Duplicate detection improvements
- [ ] Multi-language support

**Under Consideration:**
- [ ] Separate read-replica for analytics
- [ ] Redis cache for frequent queries
- [ ] Elasticsearch for advanced search
- [ ] Kubernetes deployment
- [ ] Blue-green deployment strategy
