# BRAINS Server Troubleshooting Guide

Quick reference for common issues on the BRAINS server.

## Quick Health Check

```bash
# Check all services at once
ssh cc-hub 'ssh brains "systemctl is-active postgresql ollama openclaw-support openclaw-harvest.timer node_exporter"'

# Expected output: all "active"
```

## Issue: Support Bot Not Starting

### Symptom
```bash
$ systemctl status openclaw-support
● openclaw-support.service - OpenClaw KB Support Bot
   Active: activating (auto-restart)
```

### Diagnosis Steps

1. **Check recent logs:**
```bash
sudo journalctl -u openclaw-support -n 50 --no-pager
```

2. **Look for common errors:**
- `ValueError: invalid literal for int()` → Database password issue
- `PermissionError: [Errno 13]` → SSL certificate access issue
- `Conflict: terminated by other getUpdates` → Multiple bot instances
- `relation "knowledge_base" does not exist` → Schema path issue

### Solutions

#### Database Connection Error
```bash
# Reset database password
ssh cc-hub 'ssh brains "python3 << EOF
import subprocess
import secrets
import string

alphabet = string.ascii_letters + string.digits
password = '.'.join(secrets.choice(alphabet) for i in range(24))

# Reset password
subprocess.run([\"sudo\", \"-u\", \"postgres\", \"psql\", \"-c\",
  f\"ALTER USER brains_admin WITH PASSWORD '{password}';\"])

# Update .env
with open(\"/opt/openclaw/harvesters/.env\", \"r\") as f:
    content = f.read()

import re
content = re.sub(
    r\"DATABASE_URL=.*\",
    f\"DATABASE_URL=postgresql://brains_admin:{password}@localhost:5432/brains_kb?sslmode=disable\",
    content
)

with open(\"/tmp/new_env\", \"w\") as f:
    f.write(content)

subprocess.run([\"sudo\", \"cp\", \"/tmp/new_env\", \"/opt/openclaw/harvesters/.env\"])
print(\"Password updated\")
EOF
"'

sudo systemctl restart openclaw-support
```

#### SSL Connection Error
```bash
# Add sslmode=disable to DATABASE_URL
sudo sed -i 's|@localhost:5432/brains_kb|@localhost:5432/brains_kb?sslmode=disable|' /opt/openclaw/harvesters/.env
sudo systemctl restart openclaw-support
```

#### Telegram Bot Conflict
```bash
# Check if another bot is running on DEV
ssh cc-hub 'ssh synct-dev "systemctl status moltbot-support"'

# If running, stop it
ssh cc-hub 'ssh synct-dev "sudo systemctl stop moltbot-support"'

# Restart BRAINS bot
ssh cc-hub 'ssh brains "sudo systemctl restart openclaw-support"'
```

#### Schema Path Error
```bash
# Set database search path
sudo -u postgres psql -d brains_kb -c "ALTER DATABASE brains_kb SET search_path TO kb, public;"

# Restart bot to reconnect
sudo systemctl restart openclaw-support
```

## Issue: Harvest Service Failing

### Symptom
```bash
$ systemctl status openclaw-harvest
● openclaw-harvest.service - OpenClaw KB Harvest Scanner
   Active: failed (Result: exit-code)
```

### Diagnosis Steps

1. **Check logs:**
```bash
sudo journalctl -u openclaw-harvest -n 50 --no-pager
```

2. **Common errors:**
- `FileNotFoundError: /opt/synctacles/logs/harvest.log` → Wrong log path
- `OSError: [Errno 30] Read-only file system` → Missing WritePaths
- `relation "harvest_state" does not exist` → Missing table
- `relation "knowledge_base" does not exist` → Schema path issue

### Solutions

#### Wrong Log Path
```bash
# Fix hardcoded paths
sudo sed -i 's|/opt/synctacles/logs/|/opt/openclaw/logs/|g' /opt/openclaw/harvesters/tools/scanners/run_harvest.py

# Create logs directory
sudo mkdir -p /opt/openclaw/logs
sudo chown brains:brains /opt/openclaw/logs

# Restart service
sudo systemctl restart openclaw-harvest
```

#### Read-Only Filesystem
```bash
# Add logs directory to writable paths
sudo sed -i 's|ReadWritePaths=/opt/openclaw/harvesters|ReadWritePaths=/opt/openclaw/harvesters /opt/openclaw/logs|' /etc/systemd/system/openclaw-harvest.service

sudo systemctl daemon-reload
sudo systemctl restart openclaw-harvest
```

#### Missing harvest_state Table
```bash
# Create table
sudo -u postgres psql -d brains_kb << 'EOSQL'
CREATE TABLE IF NOT EXISTS public.harvest_state (
    id SERIAL PRIMARY KEY,
    scanner_name VARCHAR(50) NOT NULL UNIQUE,
    last_cursor TEXT,
    last_item_id TEXT,
    last_item_timestamp TIMESTAMP WITH TIME ZONE,
    backlog_start_date DATE NOT NULL,
    backlog_complete BOOLEAN DEFAULT false,
    backlog_completed_at TIMESTAMP WITH TIME ZONE,
    total_harvested INTEGER DEFAULT 0,
    total_duplicates INTEGER DEFAULT 0,
    total_errors INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_harvest_state_scanner ON public.harvest_state (scanner_name);
GRANT SELECT, INSERT, UPDATE ON TABLE public.harvest_state TO brains_admin;
GRANT USAGE, SELECT ON SEQUENCE public.harvest_state_id_seq TO brains_admin;
EOSQL

# Restart service
sudo systemctl restart openclaw-harvest
```

## Issue: Database Connection Problems

### Check Database Status
```bash
systemctl status postgresql

# If not running:
sudo systemctl start postgresql
```

### Verify Database Exists
```bash
sudo -u postgres psql -l | grep brains_kb
```

### Test Connection
```bash
# As postgres user
sudo -u postgres psql -d brains_kb -c "SELECT COUNT(*) FROM kb.knowledge_base;"

# As brains_admin user
sudo -u postgres psql -d brains_kb -c "SELECT COUNT(*) FROM kb.knowledge_base;" -U brains_admin
```

### Check Permissions
```bash
# List user permissions
sudo -u postgres psql -d brains_kb -c "\\du brains_admin"

# Check table permissions
sudo -u postgres psql -d brains_kb -c "\\dp kb.knowledge_base"
```

### Reset User Password
See "Database Connection Error" solution above.

## Issue: Telegram Bot Not Responding

### Check Bot Status
```bash
# Verify bot is running
systemctl status openclaw-support

# Check recent logs
sudo journalctl -u openclaw-support -n 20 --no-pager
```

### Test Bot Token
```bash
# Get bot info
TOKEN=$(grep TELEGRAM_BOT_TOKEN_SUPPORT /opt/openclaw/harvesters/.env | cut -d= -f2)
curl -s "https://api.telegram.org/bot${TOKEN}/getMe" | jq
```

### Check Bot Group Membership
```bash
TOKEN=$(grep TELEGRAM_BOT_TOKEN_SUPPORT /opt/openclaw/harvesters/.env | cut -d= -f2)
GROUP_ID=$(grep TELEGRAM_GROUP_ID /opt/openclaw/harvesters/.env | cut -d= -f2)
BOT_ID=$(echo $TOKEN | cut -d: -f1)

curl -s "https://api.telegram.org/bot${TOKEN}/getChatMember?chat_id=${GROUP_ID}&user_id=${BOT_ID}" | jq
```

### Privacy Mode Issue
If bot doesn't see group messages, privacy mode is likely enabled:
1. Contact @BotFather on Telegram
2. Use `/setprivacy`
3. Select your bot
4. Choose "Disable"

Or, use @mentions when sending commands in groups:
```
/help@SynctaclesSupportBot
```

## Issue: Ollama Not Working

### Check Service
```bash
systemctl status ollama
```

### Test API
```bash
curl http://localhost:11434/api/tags
```

### List Models
```bash
ollama list

# Expected:
# NAME                    ID              SIZE
# phi3:mini               ...             2.2 GB
# nomic-embed-text:latest ...             274 MB
```

### Restart Ollama
```bash
sudo systemctl restart ollama
sleep 5
curl http://localhost:11434/api/tags
```

## Issue: High Resource Usage

### Check Memory Usage
```bash
systemctl status openclaw-support --no-pager | grep Memory
systemctl status openclaw-harvest --no-pager | grep Memory
```

### Check CPU Usage
```bash
top -b -n 1 | grep python
```

### Database Size
```bash
sudo -u postgres psql -d brains_kb -c "SELECT pg_size_pretty(pg_database_size('brains_kb'));"
```

### Vacuum Database
```bash
sudo -u postgres psql -d brains_kb -c "VACUUM ANALYZE;"
```

## Issue: Harvest Not Finding New Content

### Check Harvest State
```bash
sudo -u postgres psql -d brains_kb -c "SELECT scanner_name, backlog_complete, total_harvested, last_item_timestamp FROM public.harvest_state;"
```

### Reset Harvest State (Careful!)
```bash
# This will cause re-harvesting from the beginning
sudo -u postgres psql -d brains_kb -c "TRUNCATE public.harvest_state RESTART IDENTITY;"
```

### Check API Keys
```bash
# Verify GROQ key
grep GROQ_API_KEY /opt/openclaw/harvesters/.env

# Verify Anthropic key
grep ANTHROPIC_API_KEY /opt/openclaw/harvesters/.env

# Test GROQ API (if key is valid)
# Note: This is just a placeholder test
curl -X POST https://api.groq.com/openai/v1/chat/completions \
  -H "Authorization: Bearer $(grep GROQ_API_KEY /opt/openclaw/harvesters/.env | cut -d= -f2)" \
  -H "Content-Type: application/json" \
  -d '{"model":"mixtral-8x7b-32768","messages":[{"role":"user","content":"test"}],"max_tokens":5}'
```

## General Debugging Commands

### View All Service Logs
```bash
# Follow all logs in real-time
sudo journalctl -f -u openclaw-support -u openclaw-harvest -u postgresql -u ollama
```

### Check Systemd Service Files
```bash
systemctl cat openclaw-support
systemctl cat openclaw-harvest
systemctl cat openclaw-harvest.timer
```

### Test Database Query
```bash
sudo -u postgres psql -d brains_kb << 'EOSQL'
SELECT
  problem_title,
  problem_category,
  confidence_score,
  source
FROM kb.knowledge_base
WHERE is_active = true
ORDER BY created_at DESC
LIMIT 5;
EOSQL
```

### Check Disk Space
```bash
df -h
du -sh /opt/openclaw/*
du -sh /var/lib/postgresql
```

### Check Network Connectivity
```bash
# Test Telegram API
ping -c 3 api.telegram.org

# Test GitHub API
ping -c 3 api.github.com

# Test GROQ API
ping -c 3 api.groq.com
```

## Emergency Procedures

### Stop All Services
```bash
sudo systemctl stop openclaw-support
sudo systemctl stop openclaw-harvest.timer
sudo systemctl stop openclaw-harvest
```

### Restart Everything
```bash
sudo systemctl restart postgresql
sudo systemctl restart ollama
sleep 5
sudo systemctl restart openclaw-support
sudo systemctl restart openclaw-harvest.timer
```

### Restore from Backup
```bash
# Stop services
sudo systemctl stop openclaw-support openclaw-harvest.timer openclaw-harvest

# Restore database
sudo -u postgres pg_restore -d brains_kb -c /backups/brains_kb_YYYYMMDD.dump

# Restart services
sudo systemctl start openclaw-support openclaw-harvest.timer
```

## Getting Help

If none of these solutions work:

1. **Collect logs:**
```bash
# Save logs to file
sudo journalctl -u openclaw-support -n 200 > /tmp/support-bot-logs.txt
sudo journalctl -u openclaw-harvest -n 200 > /tmp/harvest-logs.txt
```

2. **Check system status:**
```bash
systemctl status openclaw-support openclaw-harvest.timer postgresql ollama > /tmp/system-status.txt
```

3. **Create GitHub issue:**
```bash
gh issue create --repo synctacles/platform \
  --title "BRAINS: [Brief description of issue]" \
  --body "## Problem
[Describe the issue]

## Steps to Reproduce
1. ...
2. ...

## Logs
[Paste relevant log excerpts]

## System Status
[Paste system status]
" \
  --label "bug,brains-server"
```

## Useful One-Liners

```bash
# Full system health check
ssh cc-hub 'ssh brains "echo \"=== Services ===\" && systemctl is-active postgresql ollama openclaw-support openclaw-harvest.timer && echo \"\" && echo \"=== KB Stats ===\" && sudo -u postgres psql -d brains_kb -c \"SELECT COUNT(*) as entries, COUNT(DISTINCT problem_category) as categories FROM kb.knowledge_base WHERE is_active = true;\" && echo \"\" && echo \"=== Disk Usage ===\" && df -h / && echo \"\" && echo \"=== Memory ===\" && free -h"'

# Quick service restart
ssh cc-hub 'ssh brains "sudo systemctl restart openclaw-support && sleep 3 && systemctl status openclaw-support --no-pager | head -10"'

# Check last harvest run
ssh cc-hub 'ssh brains "sudo journalctl -u openclaw-harvest --since \"1 hour ago\" | grep -E \"(SUMMARY|Total|Error)\""'

# Test bot connectivity
ssh cc-hub 'ssh brains "curl -s https://api.telegram.org/bot\$(grep TELEGRAM_BOT_TOKEN_SUPPORT /opt/openclaw/harvesters/.env | cut -d= -f2)/getMe | jq -r .result.username"'
```
