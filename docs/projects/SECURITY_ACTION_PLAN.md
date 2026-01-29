# SYNCTACLES Security Action Plan - High Priority

**Created:** 2026-01-29
**Status:** 🔴 ACTIVE
**Owner:** Development Team
**Environment:** DEV Server (135.181.255.83)

---

## 📋 Executive Summary

Following security audit of SYNCTACLES CARE API and Moltbot infrastructure, multiple vulnerabilities and risks have been identified. This document outlines all action items with priorities, owners, and estimated effort.

**Risk Level:** 🟡 MEDIUM (DEV environment)
**Target:** 🟢 LOW (Production-ready)

---

## 🔴 P0 - CRITICAL (Execute Immediately)

### P0.1: Block Exposed Next.js Port 3000

**Risk:** Next.js application exposed to internet on port 3000
**Impact:** Unauthorized access, potential data exposure, attack vector
**Effort:** 5 minutes
**Status:** ⏳ PENDING

**Actions:**
```bash
# Block port 3000 with UFW
sudo ufw deny 3000/tcp
sudo ufw reload

# Verify
sudo ufw status | grep 3000

# Test from external
curl -I http://135.181.255.83:3000 --max-time 5
# Should timeout or be refused
```

**Verification:**
- [ ] Port 3000 blocked in UFW
- [ ] External access test fails
- [ ] Application still accessible via nginx proxy

**Owner:** DevOps
**Deadline:** 2026-01-29 (Today)

---

### P0.2: Fix Old Bot Log File Permissions

**Risk:** Old bot log still has world-readable permissions
**Impact:** Token exposure in /tmp/support-bot.log
**Effort:** 2 minutes
**Status:** ⏳ PENDING

**Actions:**
```bash
# Fix old log file
chmod 600 /tmp/support-bot.log

# Verify
ls -la /tmp/support-bot*.log

# Clean up old logs
# After verifying bot works with new log
rm /tmp/support-bot.log
```

**Verification:**
- [ ] /tmp/support-bot.log has 600 permissions
- [ ] Only /tmp/support-bot-new.log in use
- [ ] Old log cleaned up

**Owner:** DevOps
**Deadline:** 2026-01-29 (Today)

---

## 🟠 P1 - HIGH (Within 24 Hours)

### P1.1: Implement Command Rate Limiting

**Risk:** No rate limiting on bot commands (only daily limit)
**Impact:** DDoS, resource exhaustion, API quota abuse
**Effort:** 15 minutes
**Status:** ⏳ PENDING

**Actions:**

1. **Create rate limiter module:**

```python
# /opt/synctacles/moltbot/shared/rate_limiter.py
"""Rate limiter for bot commands."""
from datetime import datetime, timedelta
from collections import defaultdict
from typing import Dict, List

class CommandRateLimiter:
    """Rate limiter for bot commands.

    Limits:
    - Max 10 commands per minute per user
    - Max 3 file uploads per hour per user
    """

    def __init__(self):
        # user_id -> list of timestamps
        self.command_timestamps: Dict[int, List[datetime]] = defaultdict(list)
        self.upload_timestamps: Dict[int, List[datetime]] = defaultdict(list)

    def check_command_limit(self, user_id: int) -> tuple[bool, int]:
        """Check if user can execute command.

        Returns:
            tuple: (allowed: bool, remaining: int)
        """
        now = datetime.now()

        # Clean old timestamps (older than 1 minute)
        self.command_timestamps[user_id] = [
            ts for ts in self.command_timestamps[user_id]
            if now - ts < timedelta(minutes=1)
        ]

        count = len(self.command_timestamps[user_id])
        if count >= 10:
            return False, 0

        self.command_timestamps[user_id].append(now)
        return True, 10 - count - 1

    def check_upload_limit(self, user_id: int) -> tuple[bool, int]:
        """Check if user can upload file.

        Returns:
            tuple: (allowed: bool, remaining: int)
        """
        now = datetime.now()

        # Clean old timestamps (older than 1 hour)
        self.upload_timestamps[user_id] = [
            ts for ts in self.upload_timestamps[user_id]
            if now - ts < timedelta(hours=1)
        ]

        count = len(self.upload_timestamps[user_id])
        if count >= 3:
            return False, 0

        self.upload_timestamps[user_id].append(now)
        return True, 3 - count - 1

# Global instance
rate_limiter = CommandRateLimiter()
```

2. **Update handlers.py:**

```python
# In handle_document function, add at the beginning:
from shared.rate_limiter import rate_limiter

async def handle_document(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user_id = update.effective_user.id

    # Check upload rate limit (3/hour)
    allowed, remaining = rate_limiter.check_upload_limit(user_id)
    if not allowed:
        await update.message.reply_text(
            "⏱️ **Rate Limit**\n\n"
            "You're uploading files too quickly.\n"
            "Please wait a few minutes before trying again.\n\n"
            "Limit: 3 uploads per hour"
        )
        return

    # ... rest of existing code
```

3. **Add to main.py commands:**

```python
# Decorator for command handlers
from shared.rate_limiter import rate_limiter

async def check_rate_limit(update: Update) -> bool:
    user_id = update.effective_user.id
    allowed, remaining = rate_limiter.check_command_limit(user_id)

    if not allowed:
        await update.message.reply_text(
            "⏱️ **Slow Down**\n\n"
            "Too many commands. Please wait a minute.\n\n"
            "Limit: 10 commands per minute"
        )
    return allowed

# Add to each command handler:
async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    if not await check_rate_limit(update):
        return
    # ... existing code
```

**Verification:**
- [ ] Rate limiter module created
- [ ] Integrated in handlers.py
- [ ] Tested: 10+ commands in 1 minute → blocked
- [ ] Tested: 3+ uploads in 1 hour → blocked
- [ ] User sees friendly error messages

**Owner:** Backend Developer
**Deadline:** 2026-01-30

---

### P1.2: Add User Verification Requirement

**Risk:** Public bot, anyone can use it
**Impact:** Abuse, spam, resource exhaustion
**Effort:** 20 minutes
**Status:** ⏳ PENDING

**Actions:**

1. **Add verification check:**

```python
# In handlers.py
async def check_user_linked(user_id: int) -> bool:
    """Check if user has linked their account."""
    user_data = await db.fetchrow(
        "SELECT telegram_user_id FROM telegram_users WHERE telegram_user_id = $1",
        user_id
    )
    return user_data is not None

async def handle_document(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user_id = update.effective_user.id

    # Check if user is linked
    if not await check_user_linked(user_id):
        await update.message.reply_text(
            "🔒 **Account Not Linked**\n\n"
            "To use the Support Bot, you need to link your account first.\n\n"
            "**How to link:**\n"
            "1. Open SYNCTACLES CARE Add-on in Home Assistant\n"
            "2. Go to Settings → Support\n"
            "3. Click 'Link Telegram Account'\n"
            "4. Follow the instructions\n\n"
            "Or use /start to create a free account (5 logs/day)."
        )
        return

    # ... rest of code
```

2. **Update /start command:**

```python
async def start_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user_id = update.effective_user.id

    # Auto-register new users as free tier
    await db.execute(
        """
        INSERT INTO telegram_users (telegram_user_id)
        VALUES ($1)
        ON CONFLICT (telegram_user_id) DO NOTHING
        """,
        user_id
    )

    # Initialize rate limit
    await db.execute(
        """
        INSERT INTO support_rate_limits (telegram_user_id, logs_today)
        VALUES ($1, 0)
        ON CONFLICT (telegram_user_id) DO NOTHING
        """,
        user_id
    )

    await update.message.reply_text(
        "👋 **Welcome to SYNCTACLES CARE Support!**\n\n"
        "✅ Free account created!\n"
        "📊 Limit: 5 log analyses per day\n\n"
        "Use /help to see available commands."
    )
```

**Verification:**
- [ ] New users auto-registered on /start
- [ ] /analyze requires linked account
- [ ] User gets clear instructions on how to link
- [ ] Database entries created properly

**Owner:** Backend Developer
**Deadline:** 2026-01-30

---

### P1.3: Commit Security Updates to GitHub

**Risk:** Security fixes only local, not in version control
**Impact:** Lost on server rebuild, no audit trail
**Effort:** 10 minutes
**Status:** ⏳ PENDING

**Actions:**

```bash
# Moltbot repository
cd /opt/synctacles/moltbot

git add shared/logging_filter.py
git add shared/rate_limiter.py  # After P1.1
git add support_agent/main.py
git add support_agent/handlers.py
git add .claudeignore

git commit -m "$(cat <<'EOF'
security: implement token protection and rate limiting

- Add TokenRedactionFilter to prevent token logging
- Disable httpx/telegram INFO logging to stop token exposure
- Implement command rate limiting (10/min per user)
- Add upload rate limiting (3/hour per user)
- Add user verification requirement for /analyze
- Create .claudeignore to prevent credential leaks
- Fix file permissions (600 for .env and logs)

Security improvements address:
- Token exposure in logs
- DoS via command spam
- Unauthorized bot usage
- Future credential leaks to Claude

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"

git push origin main
```

```bash
# Backend repository
cd /opt/github/synctacles-api

git add docs/projects/CLAUDE_SECURITY_HYGIENE.md
git add docs/projects/SECURITY_ACTION_PLAN.md
git add .claudeignore

git commit -m "$(cat <<'EOF'
docs: add comprehensive security audit and action plan

- Document credential exposure incident
- Create detailed remediation plan
- Add port exposure analysis
- Define priority action items
- Add .claudeignore for future protection

Related to Moltbot security hardening.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"

# Create branch (pre-commit hook blocks direct main)
git checkout -b security/audit-and-hardening
git push -u origin security/audit-and-hardening

# Create PR
gh pr create --title "Security: audit documentation and hardening plan" \
  --body "Comprehensive security audit and action plan following Moltbot development. See CLAUDE_SECURITY_HYGIENE.md and SECURITY_ACTION_PLAN.md for details."
```

**Verification:**
- [ ] All security code committed to Moltbot repo
- [ ] Security docs committed to Backend repo
- [ ] PR created for backend changes
- [ ] CI passes

**Owner:** Developer
**Deadline:** 2026-01-30

---

## 🟡 P2 - MEDIUM (Within 1 Week)

### P2.1: Enable PostgreSQL SSL/TLS

**Risk:** Database connections not encrypted
**Impact:** Credentials sniffable on localhost
**Effort:** 30 minutes
**Status:** ⏳ PENDING

**Actions:**

1. **Generate SSL certificates:**

```bash
# Self-signed cert for localhost
cd /etc/postgresql/16/main

sudo openssl req -new -x509 -days 365 -nodes -text \
  -out server.crt \
  -keyout server.key \
  -subj "/CN=localhost"

sudo chmod 600 server.key
sudo chown postgres:postgres server.key server.crt
```

2. **Enable SSL in PostgreSQL:**

```bash
# /etc/postgresql/16/main/postgresql.conf
sudo nano /etc/postgresql/16/main/postgresql.conf

# Change:
ssl = on
ssl_cert_file = '/etc/postgresql/16/main/server.crt'
ssl_key_file = '/etc/postgresql/16/main/server.key'

# Restart
sudo systemctl restart postgresql
```

3. **Update connection strings:**

```bash
# .env.development
DATABASE_URL=postgresql://care_dev:PASSWORD@localhost/synctacles_dev?sslmode=require
```

4. **Update database.py:**

```python
# app/database.py and shared/database.py
self.pool = await asyncpg.create_pool(
    dsn=settings.database_url,
    min_size=5,
    max_size=20,
    command_timeout=60,
    ssl='require'  # Add this
)
```

**Verification:**
- [ ] SSL enabled in PostgreSQL
- [ ] Certificates generated
- [ ] Connection strings updated
- [ ] All services restart successfully
- [ ] Verify SSL with: `psql "sslmode=require" -h localhost`

**Owner:** DevOps
**Deadline:** 2026-02-05

---

### P2.2: Implement Database Access Segregation

**Risk:** All services use same database user
**Impact:** Lateral movement, data access across services
**Effort:** 45 minutes
**Status:** ⏳ PENDING

**Actions:**

1. **Create separate roles:**

```sql
-- Create roles
CREATE ROLE moltbot_role WITH LOGIN PASSWORD 'NEW_SECURE_PASSWORD';
CREATE ROLE care_api_role WITH LOGIN PASSWORD 'NEW_SECURE_PASSWORD';
CREATE ROLE energy_api_role WITH LOGIN PASSWORD 'NEW_SECURE_PASSWORD';

-- Moltbot: CARE tables only
GRANT CONNECT ON DATABASE synctacles_dev TO moltbot_role;
GRANT USAGE ON SCHEMA public TO moltbot_role;
GRANT SELECT, INSERT, UPDATE ON care_licenses TO moltbot_role;
GRANT SELECT, INSERT, UPDATE ON telegram_users TO moltbot_role;
GRANT SELECT, INSERT, UPDATE ON support_logs TO moltbot_role;
GRANT SELECT, INSERT, UPDATE ON support_rate_limits TO moltbot_role;
GRANT SELECT, INSERT ON llm_usage TO moltbot_role;

-- CARE API: CARE tables + validation
GRANT CONNECT ON DATABASE synctacles_dev TO care_api_role;
GRANT USAGE ON SCHEMA public TO care_api_role;
GRANT SELECT, INSERT, UPDATE ON care_installs TO care_api_role;
GRANT SELECT, INSERT, UPDATE ON care_licenses TO care_api_role;
GRANT SELECT, INSERT ON care_validations TO care_api_role;
GRANT SELECT, INSERT ON care_license_transfers TO care_api_role;

-- Energy API: Energy tables only
GRANT CONNECT ON DATABASE synctacles_dev TO energy_api_role;
GRANT USAGE ON SCHEMA public TO energy_api_role;
-- Grant to Energy API specific tables (not shown)

-- Revoke from care_dev
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM care_dev;
```

2. **Update .env files:**

```bash
# Moltbot
DATABASE_URL=postgresql://moltbot_role:PASSWORD@localhost/synctacles_dev?sslmode=require

# CARE API
DATABASE_URL=postgresql://care_api_role:PASSWORD@localhost/synctacles_dev?sslmode=require

# Energy API (separate)
DATABASE_URL=postgresql://energy_api_role:PASSWORD@localhost/synctacles_dev?sslmode=require
```

**Verification:**
- [ ] Roles created with minimal permissions
- [ ] Each service uses dedicated role
- [ ] Moltbot cannot access Energy tables
- [ ] Energy API cannot access CARE tables
- [ ] All services still functional

**Owner:** Database Admin
**Deadline:** 2026-02-05

---

### P2.3: Add Monitoring & Alerting

**Risk:** No visibility into security events
**Impact:** Attacks undetected, slow incident response
**Effort:** 1 hour
**Status:** ⏳ PENDING

**Actions:**

1. **Add security logging:**

```python
# /opt/synctacles/moltbot/shared/security_logger.py
"""Security event logging."""
import logging
from datetime import datetime

security_logger = logging.getLogger('security')
security_logger.setLevel(logging.INFO)

# File handler for security events
handler = logging.FileHandler('/var/log/moltbot/security.log')
handler.setFormatter(logging.Formatter(
    '%(asctime)s - SECURITY - %(levelname)s - %(message)s'
))
security_logger.addHandler(handler)

async def log_rate_limit_hit(user_id: int, limit_type: str):
    """Log rate limit violations."""
    security_logger.warning(
        f"Rate limit hit: user_id={user_id} type={limit_type}"
    )

async def log_unlinked_access_attempt(user_id: int):
    """Log unlinked user access attempts."""
    security_logger.warning(
        f"Unlinked access attempt: user_id={user_id}"
    )

async def log_suspicious_upload(user_id: int, file_size: int, file_type: str):
    """Log suspicious file uploads."""
    if file_size > 5 * 1024 * 1024:  # > 5MB
        security_logger.info(
            f"Large upload: user_id={user_id} size={file_size} type={file_type}"
        )
```

2. **Add alerts:**

```python
# Alert on repeated violations
VIOLATION_THRESHOLD = 10  # per hour

async def check_violations(user_id: int):
    """Check if user has excessive violations."""
    count = await db.fetchval(
        """
        SELECT COUNT(*) FROM security_events
        WHERE user_id = $1
        AND created_at > NOW() - INTERVAL '1 hour'
        """,
        user_id
    )

    if count > VIOLATION_THRESHOLD:
        # Send alert to Telegram monitoring channel
        await send_admin_alert(
            f"⚠️ Security Alert\n\n"
            f"User {user_id} has {count} violations in last hour.\n"
            f"Possible abuse detected."
        )
```

**Verification:**
- [ ] Security logging implemented
- [ ] Log file created with proper permissions
- [ ] Alerts sent to monitoring channel
- [ ] False positive rate acceptable

**Owner:** Backend Developer
**Deadline:** 2026-02-05

---

### P2.4: Create Systemd Services for Moltbot

**Risk:** Bot runs ad-hoc, no auto-restart
**Impact:** Manual restart needed, no service management
**Effort:** 30 minutes
**Status:** ⏳ PENDING

**Actions:**

1. **Create systemd service:**

```ini
# /etc/systemd/system/moltbot-support.service
[Unit]
Description=SYNCTACLES Moltbot Support Agent
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=synctacles-dev
Group=synctacles-dev
WorkingDirectory=/opt/synctacles/moltbot
Environment="PATH=/opt/synctacles/moltbot/venv/bin:/usr/bin:/bin"
EnvironmentFile=/opt/synctacles/config/.env.development

ExecStart=/opt/synctacles/moltbot/venv/bin/python support_agent/main.py

Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/synctacles/moltbot

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=moltbot-support

[Install]
WantedBy=multi-user.target
```

2. **Enable and start:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable moltbot-support
sudo systemctl start moltbot-support

# Check status
sudo systemctl status moltbot-support

# View logs
sudo journalctl -u moltbot-support -f
```

**Verification:**
- [ ] Service file created
- [ ] Service starts successfully
- [ ] Auto-restarts on failure
- [ ] Logs to journald
- [ ] Starts on boot

**Owner:** DevOps
**Deadline:** 2026-02-05

---

## 🟢 P3 - LOW (Nice to Have)

### P3.1: Implement Docker Containerization

**Risk:** No process isolation
**Impact:** Resource limits, deployment complexity
**Effort:** 4 hours
**Status:** 🔵 FUTURE

**Actions:**
- Create Dockerfile for Moltbot
- Docker Compose for all services
- Network segmentation via Docker networks
- Resource limits (CPU, memory)

**Owner:** DevOps
**Deadline:** TBD

---

### P3.2: HashiCorp Vault for Secrets

**Risk:** Secrets in files
**Impact:** Rotation complexity, audit trail
**Effort:** 8 hours
**Status:** 🔵 FUTURE

**Actions:**
- Install Vault
- Migrate secrets to Vault
- Dynamic secret generation
- Automatic rotation

**Owner:** Security Team
**Deadline:** TBD

---

### P3.3: Geographic Rate Limiting

**Risk:** Global bot abuse
**Impact:** International DDoS
**Effort:** 2 hours
**Status:** 🔵 FUTURE

**Actions:**
- Implement IP geolocation
- Rate limit by country
- Whitelist Netherlands/Belgium
- Alert on unusual patterns

**Owner:** Backend Developer
**Deadline:** TBD

---

## 📊 Progress Tracking

### Overall Status

| Priority | Total | Completed | In Progress | Pending | Blocked |
|----------|-------|-----------|-------------|---------|---------|
| P0 | 2 | 0 | 0 | 2 | 0 |
| P1 | 3 | 0 | 0 | 3 | 0 |
| P2 | 4 | 0 | 0 | 4 | 0 |
| P3 | 3 | 0 | 0 | 3 | 0 |
| **TOTAL** | **12** | **0** | **0** | **12** | **0** |

### Timeline

```
Week 1 (Now):           P0.1, P0.2
Week 1 (Day 2):         P1.1, P1.2, P1.3
Week 2:                 P2.1, P2.2, P2.3, P2.4
Week 3+:                P3.x items
```

### Risk Reduction

```
Current Risk:     🟡 5.2/10
After P0:         🟢 4.0/10 (-23%)
After P1:         🟢 3.0/10 (-42%)
After P2:         🟢 2.4/10 (-54%)
Target:           🟢 2.0/10 or below
```

---

## 🔄 Review Schedule

| Review Type | Frequency | Next Review |
|-------------|-----------|-------------|
| Progress Check | Daily | 2026-01-30 |
| Security Audit | Weekly | 2026-02-05 |
| Full Assessment | Monthly | 2026-03-01 |

---

## 📝 Notes

- All timestamps in UTC
- Development environment - production will need separate hardening
- Token rotation still pending (see CLAUDE_SECURITY_HYGIENE.md Phase 2)
- Monitor bot activity during rollout of rate limiting
- Consider A/B testing for user verification impact

---

## ✅ Sign-Off

**Prepared By:** Security Audit Team
**Reviewed By:** (Pending)
**Approved By:** (Pending)
**Date:** 2026-01-29

**Next Action:** Execute P0 items immediately

---

**Document Version:** 1.0
**Last Updated:** 2026-01-29 10:40 UTC
