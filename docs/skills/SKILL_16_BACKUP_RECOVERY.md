# SKILL 16 — BACKUP & RECOVERY

Disaster Recovery Procedures for SYNCTACLES Infrastructure
Version: 1.1 (2026-01-22)

---

## GOAL

Ensure business continuity through documented backup and recovery procedures for all critical components.

**Recovery Time Objective (RTO):** 1 hour
**Recovery Point Objective (RPO):** 24 hours (daily backups)

---

## INFRASTRUCTURE OVERVIEW

### Servers

| Server | IP | Purpose | Critical Data |
|--------|-----|---------|---------------|
| PROD | 46.62.212.227 | Production API (synctacles.com) | PostgreSQL, .env |
| DEV | 135.181.255.83 | Development + Claude Code (dev.synctacles.com) | PostgreSQL, .env, repos |
| HUB | 135.181.201.253 | CAI Skills Hub (Claude AI context) | Skills, CAI config |
| HA/VPN | 91.99.150.36 | Home Assistant + WireGuard | HA config, VPN config |
| Monitor | 77.42.41.135 | Grafana + Prometheus (monitor.synctacles.com) | Dashboards, metrics |

### Critical Components

1. **Environment Files (.env)** - API keys, secrets, database credentials
2. **PostgreSQL Databases** - All application data
3. **Systemd Services** - Service configurations
4. **SSL Certificates** - If custom certs configured

---

## BACKUP PROCEDURES

### 1. Environment Files (.env)

**Location:** `/opt/github/<repo>/.env`
**Frequency:** After every change, minimum weekly
**Retention:** 30 days

```bash
# Synctacles Server
sudo mkdir -p /opt/backups/env
sudo cp /opt/github/synctacles-api/.env /opt/backups/env/synctacles-api.env.$(date +%Y%m%d)
sudo chmod 600 /opt/backups/env/*
```

### 2. PostgreSQL Database

**Synctacles Database:**
```bash
# Manual backup
pg_dump -U energy_insights_nl energy_insights_nl > /opt/backups/db/synctacles_$(date +%Y%m%d).sql

# Compressed backup
pg_dump -U energy_insights_nl energy_insights_nl | gzip > /opt/backups/db/synctacles_$(date +%Y%m%d).sql.gz
```

### 3. Systemd Service Files

```bash
# Backup all custom services
sudo cp /etc/systemd/system/energy-insights-nl-*.service /opt/backups/systemd/
sudo cp /etc/systemd/system/energy-insights-nl-*.timer /opt/backups/systemd/
```

---

## AUTOMATED BACKUP SCRIPT

Create `/opt/scripts/daily_backup.sh`:

```bash
#!/bin/bash
# Daily backup script for SYNCTACLES

set -e

BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d)
RETENTION_DAYS=30

# Create directories
mkdir -p ${BACKUP_DIR}/{env,db,systemd}

# 1. Backup .env
cp /opt/github/synctacles-api/.env ${BACKUP_DIR}/env/synctacles.env.${DATE}

# 2. Backup database
pg_dump -U energy_insights_nl energy_insights_nl | gzip > ${BACKUP_DIR}/db/synctacles_${DATE}.sql.gz

# 3. Backup systemd services
cp /etc/systemd/system/energy-insights-nl-*.service ${BACKUP_DIR}/systemd/ 2>/dev/null || true
cp /etc/systemd/system/energy-insights-nl-*.timer ${BACKUP_DIR}/systemd/ 2>/dev/null || true

# 4. Set permissions
chmod 600 ${BACKUP_DIR}/env/*
chmod 600 ${BACKUP_DIR}/db/*

# 5. Cleanup old backups
find ${BACKUP_DIR}/env -name "*.env.*" -mtime +${RETENTION_DAYS} -delete
find ${BACKUP_DIR}/db -name "*.sql.gz" -mtime +${RETENTION_DAYS} -delete

echo "Backup completed: ${DATE}"
```

### Cron Setup

```bash
# Add to crontab (daily at 3 AM)
sudo crontab -e
0 3 * * * /opt/scripts/daily_backup.sh >> /var/log/backup.log 2>&1
```

---

## RECOVERY PROCEDURES

### Scenario 1: .env File Lost/Corrupted

```bash
# 1. List available backups
ls -la /opt/backups/env/

# 2. Restore most recent
cp /opt/backups/env/synctacles.env.YYYYMMDD /opt/github/synctacles-api/.env
chmod 600 /opt/github/synctacles-api/.env

# 3. Restart services
sudo systemctl restart energy-insights-nl-api
```

### Scenario 2: Database Corruption

```bash
# 1. Stop services
sudo systemctl stop energy-insights-nl-api

# 2. Drop and recreate database
sudo -u postgres psql -c "DROP DATABASE energy_insights_nl;"
sudo -u postgres psql -c "CREATE DATABASE energy_insights_nl OWNER energy_insights_nl;"

# 3. Restore from backup
gunzip -c /opt/backups/db/synctacles_YYYYMMDD.sql.gz | psql -U energy_insights_nl energy_insights_nl

# 4. Restart services
sudo systemctl start energy-insights-nl-api

# 5. Verify
curl -s http://localhost:8000/health | jq .
```

### Scenario 3: Complete Server Loss

```bash
# On new server:

# 1. Install dependencies
sudo apt update && sudo apt install -y postgresql python3.12 python3.12-venv nginx

# 2. Create user
sudo useradd -m -s /bin/bash energy-insights-nl

# 3. Clone repository
sudo mkdir -p /opt/github
cd /opt/github
sudo git clone git@github.com:synctacles/backend.git synctacles-api
sudo chown -R energy-insights-nl:energy-insights-nl synctacles-api

# 4. Restore .env from backup (copy from secure location)
cp /path/to/backup/synctacles.env /opt/github/synctacles-api/.env

# 5. Setup virtual environment
cd /opt/github/synctacles-api
python3.12 -m venv /opt/energy-insights-nl/venv
source /opt/energy-insights-nl/venv/bin/activate
pip install -r requirements.txt

# 6. Setup database
sudo -u postgres createuser energy_insights_nl
sudo -u postgres createdb -O energy_insights_nl energy_insights_nl

# 7. Restore database from backup
gunzip -c /path/to/backup/synctacles_YYYYMMDD.sql.gz | psql -U energy_insights_nl energy_insights_nl

# 8. Install systemd services
sudo cp /path/to/backup/systemd/*.service /etc/systemd/system/
sudo cp /path/to/backup/systemd/*.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable energy-insights-nl-api
sudo systemctl start energy-insights-nl-api

# 9. Verify
curl -s http://localhost:8000/health | jq .
```

### Scenario 4: Code Rollback

```bash
# Rollback to previous version
cd /opt/github/synctacles-api
git log --oneline -10  # Find target commit
git checkout <commit-hash>
sudo systemctl restart energy-insights-nl-api

# Return to latest
git checkout main
git pull origin main
sudo systemctl restart energy-insights-nl-api
```

---

## BACKUP VERIFICATION CHECKLIST

Weekly verification (every Monday):

- [ ] Check backup directory exists: `ls -la /opt/backups/`
- [ ] Verify recent .env backup: `ls -la /opt/backups/env/ | tail -5`
- [ ] Verify recent DB backup: `ls -la /opt/backups/db/ | tail -5`
- [ ] Test DB backup integrity: `gunzip -t /opt/backups/db/latest.sql.gz`
- [ ] Check backup log: `tail -20 /var/log/backup.log`
- [ ] Verify cron job running: `sudo crontab -l | grep backup`

---

## OFF-SITE BACKUP

For disaster recovery, copy backups to remote location:

```bash
# Sync to remote server (daily)
rsync -avz /opt/backups/ backup-user@remote-server:/backups/synctacles/

# Or to object storage (S3-compatible)
aws s3 sync /opt/backups/ s3://bucket-name/synctacles-backups/
```

---

## EMERGENCY CONTACTS

| Role | Contact |
|------|---------|
| Server Admin | Leo Bultmann |
| Hetzner Support | support@hetzner.com |
| GitHub | github.com/synctacles |

---

## BACKUP LOCATIONS SUMMARY

| Component | Location |
|-----------|----------|
| .env | `/opt/backups/env/synctacles.env.*` |
| Database | `/opt/backups/db/synctacles_*.sql.gz` |
| Systemd | `/opt/backups/systemd/` |

---

## RELATED SKILLS

- SKILL 08: Hardware Profile (server specs)
- SKILL 10: Deployment Workflow (deployment procedures)
- SKILL 11: Repo and Accounts (GitHub access)
