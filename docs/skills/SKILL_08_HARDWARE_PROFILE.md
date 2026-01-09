# SKILL 8 — HARDWARE PROFILE

System Requirements and Deployment Infrastructure
Version: 1.0 (2025-12-30)

---

## PURPOSE

Define the hardware requirements for running SYNCTACLES: minimum specs, recommended specs, infrastructure requirements, and how to scale the system.

---

## DEPLOYMENT SCENARIOS

### Scenario 1: Home Assistant Integration (Local Deployment)

User runs SYNCTACLES on their Home Assistant server.

**Target Hardware:**
- Home Assistant device (Raspberry Pi, NUC, or equivalent)
- Already running Home Assistant OS

**Hardware Requirements:**
- CPU: Dual-core @ 2 GHz or better
- RAM: 2 GB minimum, 4 GB recommended
- Storage: 500 MB for application + 1 GB for database (initial)
- Network: Ethernet or WiFi

**Installation:** FASE 0-6 setup script

---

### Scenario 2: Dedicated Server (Production)

SYNCTACLES runs on its own server, serving multiple Home Assistant instances.

**Target Hardware:**
- VPS (Virtual Private Server)
- Dedicated Linux server
- Raspberry Pi 4+ (if low traffic)

**Hardware Requirements:**
- CPU: Quad-core @ 2 GHz or better
- RAM: 4 GB minimum, 8 GB recommended
- Storage: 2 GB for application + 10 GB for database (1 year data)
- Network: 1 Gbps connection

**Installation:** FASE 0-6 setup script

---

### Scenario 3: Multi-Tenant Cloud (SaaS)

Multiple tenants on one server (different brands).

**Target Infrastructure:**
- Cloud provider (AWS, Hetzner, DigitalOcean)
- Load balancer
- Database replication

**Hardware Requirements:**
- CPU: 8+ cores (shared across tenants)
- RAM: 16+ GB (shared across tenants)
- Storage: 100+ GB (database per tenant)
- Network: 10 Gbps

**Installation:** Terraform/Ansible automation

---

## MINIMUM SYSTEM REQUIREMENTS

### CPU

**Minimum:** Single-core @ 2 GHz
- OK for personal use
- Tight for multi-tenant
- Frequent data collection tasks (polling)

**Recommended:** Dual-core @ 2 GHz or better
- Comfortable for single tenant
- Adequate for small multi-tenant (2-3 users)
- No performance issues

**Production:** Quad-core @ 2 GHz or better
- Handles 10+ concurrent requests
- Database queries don't block collection
- Comfortable headroom

### RAM

**Minimum:** 1 GB
- Very tight
- Only viable if minimal logging/caching
- Frequent swap usage

**Recommended:** 4 GB
- Comfortable for single tenant
- PostgreSQL buffer pool: 1 GB
- Application memory: 500 MB
- OS and other services: 1.5 GB

**Production:** 8-16 GB
- Generous buffer pool (2-4 GB)
- Application instances (2+)
- Monitoring/logging
- Database replication

### Storage

**Disk Space Requirements:**

```
Application code:        200 MB
Python dependencies:     800 MB
PostgreSQL:
  - Raw data (3 months):  5 GB
  - Normalized data:      2 GB
  - Indexes:              1 GB
Logs:                     2 GB (retention: 30 days)
Backups:                  10 GB (weekly)
─────────────────────────────────
Total:                    ~20 GB
```

**Minimum:** 30 GB SSD
**Recommended:** 100 GB SSD (allows growth)
**Production:** 500+ GB with RAID backup

### Network

**Minimum:** 10 Mbps
- Sufficient for data collection
- API response time: 50-100 ms

**Recommended:** 50 Mbps or better
- No bandwidth bottleneck
- Comfortable for 10+ concurrent users

**Production:** 1 Gbps
- Handles traffic spikes
- Database replication
- Backup traffic

---

## OPERATING SYSTEM

### Supported Operating Systems

| OS | Version | Support | Notes |
|----|---------|---------|-------|
| Ubuntu | 22.04 LTS | ✅ Primary | Recommended, well-tested |
| Ubuntu | 24.04 LTS | ✅ Primary | Latest LTS |
| Debian | 12 (Bookworm) | ✅ Supported | Compatible with Ubuntu scripts |
| Debian | 11 (Bullseye) | ⚠️ Limited | Older Python packages |
| Home Assistant OS | Latest | ✅ Add-on | Via add-on system |
| Raspberry Pi OS | Latest | ✅ Supported | ARM64 support |
| CentOS | 9 Stream | ⚠️ Limited | Requires script adaptation |
| Alpine Linux | Latest | ⚠️ Possible | Minimal, may require tweaks |

### Why Linux?

- SYNCTACLES is Linux-native (systemd, bash scripts)
- Windows/macOS not officially supported
- Container deployment (Docker) available as alternative

---

## SOFTWARE STACK REQUIREMENTS

### Required

| Component | Version | Purpose |
|-----------|---------|---------|
| Python | 3.10+ | Application runtime |
| PostgreSQL | 13+ | Data storage |
| systemd | Latest | Service management |
| OpenSSL | 1.1+ | TLS/HTTPS |
| curl/wget | Latest | HTTP requests |
| git | Latest | Repository management |

### Optional

| Component | Purpose | Use When |
|-----------|---------|----------|
| nginx | Reverse proxy | HTTPS, multiple apps |
| Docker | Containerization | Cloud deployments |
| Ansible | Automation | Multi-server setup |
| Grafana | Monitoring | Visualize metrics |
| Prometheus | Metrics | System monitoring |

---

## PYTHON DEPENDENCIES

### Core Dependencies (in requirements.txt)

```
fastapi==0.104.0           # Web framework
uvicorn==0.24.0            # ASGI server
psycopg2-binary==2.9.0     # PostgreSQL driver
sqlalchemy==2.0.0          # ORM
pydantic==2.4.0            # Data validation
requests==2.31.0           # HTTP client
python-dotenv==1.0.0       # .env loading
alembic==1.12.0            # Database migrations
```

### Development Dependencies

```
pytest==7.4.0              # Testing
black==23.11.0             # Code formatting
flake8==6.1.0              # Linting
isort==5.12.0              # Import sorting
mypy==1.7.0                # Type checking
```

### Installation

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

---

## DATABASE REQUIREMENTS

### PostgreSQL Configuration

**Minimum (single tenant):**
```sql
max_connections = 20
shared_buffers = 256 MB
effective_cache_size = 1 GB
maintenance_work_mem = 64 MB
checkpoint_completion_target = 0.9
wal_buffers = 16 MB
default_statistics_target = 100
```

**Recommended (multi-tenant):**
```sql
max_connections = 100
shared_buffers = 4 GB
effective_cache_size = 12 GB
maintenance_work_mem = 1 GB
checkpoint_completion_target = 0.9
wal_buffers = 16 MB
default_statistics_target = 100
```

### Storage for Data Retention

**1 Year of Data:**
- Generation (A75): 5 GB
- Load (A65): 1 GB
- Prices (A44): 500 MB
- Balance (TenneT): 1 GB
- Indexes: 2 GB
- Backups (weekly): 10 GB
- **Total: ~20 GB**

---

## NETWORK CONFIGURATION

### Ports

| Port | Service | Direction | Security |
|------|---------|-----------|----------|
| 8000 | FastAPI | Localhost only | No auth required (behind nginx) |
| 5432 | PostgreSQL | Localhost only | No external access |
| 80 | HTTP | External | Redirect to HTTPS |
| 443 | HTTPS | External | Public, needs TLS cert |
| 22 | SSH | Restricted | Admin only |

## NETWORK SECURITY

### Hetzner Cloud Firewall (Primary)

Alle netwerkbeveiliging via Hetzner Cloud Firewall, niet OS-level.

**Huidige Ruleset:**

| Rule | Protocol | Port | Source | Action |
|------|----------|------|--------|--------|
| SSH | TCP | 22 | Leo's IP(s) | Allow |
| HTTPS | TCP | 443 | Any | Allow |
| API (internal) | TCP | 8000 | Localhost only | - |
| All other | * | * | * | Deny |

**Beheer:** Hetzner Cloud Console → Firewalls

**Waarom geen UFW:**
- Zie ADR-001 in ARCHITECTURE.md
- Hetzner FW blokkeert traffic vóór server
- Centraal beheer, minder drift

### Firewall Rules (Outgoing)

```bash
# Outgoing (to APIs)
Allow: ENTSO-E (api.entso-e.eu)
Allow: TenneT (api.tennet.nl)
Allow: Energy-Charts (api.energy-charts.info)

# DNS
Allow: Port 53 (DNS queries)
```

---

## PERFORMANCE BENCHMARKS

### Expected Performance (Recommended Hardware)

| Operation | Time | Notes |
|-----------|------|-------|
| Collect ENTSO-E A75 | 2 sec | HTTP + parse + save |
| Import 150 records | 1 sec | Database INSERT |
| Normalize generation | 0.5 sec | Calculate quality |
| API response (generation) | 50 ms | Database query |
| Health check | 10 ms | In-memory |
| Full pipeline cycle | ~5 sec | All layers combined |

### Load Testing Results

Tested on: 4 vCPU, 8 GB RAM server

```
Concurrent Users: 100
Request Rate: 100 req/sec
API Response Time (p95): 50 ms
Database Query Time (p95): 100 ms
No errors during 1 hour test
```

---

## SCALING CONSIDERATIONS

### Single Server Limits

With recommended hardware, SYNCTACLES can handle:
- ~100 concurrent API requests
- ~10 MB/sec of data ingestion
- 5+ tenants (independent instances)
- 1+ year of historical data

### Scaling Beyond Single Server

**Option 1: Database Replication**
- Primary server (write)
- Replica server (read-only backups)
- PostgreSQL streaming replication

**Option 2: Multiple Application Servers**
- Load balancer (nginx)
- Multiple API instances
- Shared PostgreSQL database

**Option 3: Microservices**
- Separate services per layer
- Message queue (RabbitMQ/Redis)
- Distributed data pipeline

---

## DEPLOYMENT ENVIRONMENTS

### Development

```
Hardware: Developer laptop/desktop
Requirements:
  - 4 GB RAM minimum
  - 50 GB disk space
  - Modern CPU (multi-core)
OS: Linux/macOS/Windows (WSL)
Database: PostgreSQL (local Docker)
Services: Run directly (no systemd)
```

### Testing

```
Hardware: Small VPS (1 vCPU, 2 GB RAM)
Requirements: Same as Recommended, but minimal
OS: Ubuntu 22.04 LTS
Database: PostgreSQL (full)
Services: systemd units
Network: Private (not public-facing)
```

### Staging

```
Hardware: Medium VPS (2 vCPU, 4 GB RAM)
Requirements: Recommended specs
OS: Ubuntu 24.04 LTS
Database: PostgreSQL (replicated)
Services: systemd units + monitoring
Network: Public, but behind basic firewall
```

### Production

```
Hardware: Dedicated or large VPS
Requirements: Production specs (see above)
OS: Ubuntu 24.04 LTS (Long-term support)
Database: PostgreSQL + replication
Services: systemd + monitoring + alerting
Network: Public, full security hardening
Backups: Daily, off-site
Monitoring: Prometheus + Grafana
Logging: Centralized (ELK stack optional)
```

---

## MONITORING & OBSERVABILITY

### System Metrics to Monitor

```bash
# CPU usage
top, htop, or systemd per-service

# Memory usage
free -h, /proc/meminfo

# Disk usage
df -h, du -sh /var/lib/postgresql

# Network
netstat -i, ss -i

# Database
pg_stat_statements, pg_stat_activity

# Application
journalctl -u synctacles-api -f
```

### Alerting Rules

```
Alert if:
  - CPU > 80% for 10 minutes
  - RAM > 90% available
  - Disk > 85% full
  - Database connection pool exhausted
  - API error rate > 1%
  - ENTSO-E data > 30 minutes old
  - Database replication lag > 1 minute
```

---

## INFRASTRUCTURE AS CODE

### Terraform (IaC)

Example for cloud deployment:

```hcl
resource "aws_instance" "synctacles" {
  ami           = "ami-0c55b159cbfafe1f0"  # Ubuntu 24.04
  instance_type = "t3.medium"              # 2 vCPU, 4 GB RAM

  root_block_device {
    volume_size = 100
    volume_type = "gp3"
  }

  security_groups = ["synctacles-api"]

  tags = {
    Name = "synctacles-api"
  }
}
```

### Docker (Containerization)

```dockerfile
FROM python:3.11-slim

WORKDIR /opt/synctacles

# Install dependencies
RUN apt-get update && apt-get install -y postgresql-client
RUN pip install -r requirements.txt

# Copy application
COPY . .

# Environment-driven config
ENV BRAND_NAME=${BRAND_NAME}
ENV DB_HOST=${DB_HOST}

# Run application
CMD ["uvicorn", "synctacles_db.api.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

---

## DISASTER RECOVERY

### Backup Strategy

```
Daily backups:
  pg_dump synctacles_nl > /backups/synctacles-nl-YYYY-MM-DD.sql

Weekly offsite:
  aws s3 cp /backups/ s3://synctacles-backups/

Retention:
  - Daily: 30 days
  - Weekly: 1 year
  - Monthly: 5 years
```

### Recovery Testing

```bash
# Test backup restore monthly
pg_restore /backups/synctacles-nl-2025-12-30.sql --clean
# Verify data integrity
psql synctacles_nl -c "SELECT COUNT(*) FROM norm_generation"
```

---

## SECURITY HARDENING

### OS Hardening

```bash
# Network Security: Hetzner Cloud Firewall (zie ADR-001)
# Geen UFW configuratie nodig

# SSH
PasswordAuthentication no  # Keys only
PermitRootLogin no
```

### Database Security

```bash
# PostgreSQL
- Require password for all users
- No default postgres user access from network
- Row-level security (optional per tenant)
- Regular updates and patches
```

### Application Security

```bash
- HTTPS/TLS for all external traffic
- Environment-based secrets (never git)
- Regular dependency updates
- Code scanning (via CI/CD)
```

---

## PERFORMANCE OPTIMIZATION

### PostgreSQL Tuning

```sql
-- Create indexes for frequent queries
CREATE INDEX ON norm_generation(source_timestamp DESC);
CREATE INDEX ON raw_entso_e_a75(import_timestamp DESC);

-- Analyze tables regularly
ANALYZE;

-- Vacuum to reclaim space
VACUUM ANALYZE;
```

### Application Caching

```python
# Cache API responses for 5 minutes
@app.get("/v1/generation/current")
@cache(expire=300)
async def get_generation():
    ...
```

### Database Connection Pooling

```python
# Use pgBouncer for connection pooling
# Allows 100 client connections to share 10 database connections
```

---

## RELATED SKILLS

- **SKILL 2**: Architecture (deployed on this hardware)
- **SKILL 9**: Installer (installs on this hardware)
- **SKILL 10**: Deployment (from development to this hardware)
