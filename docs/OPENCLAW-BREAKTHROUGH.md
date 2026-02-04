# OpenClaw - Breakthrough Architecture for Personalized HA Support

**Date:** 2026-02-04
**Status:** 🚀 BREAKTHROUGH ARCHITECTURE DEFINED - Ready for Implementation
**Priority:** HIGH - Baanbrekend for Home Assistant Community

---

## 🎯 Executive Summary

**The Breakthrough Discovery:**

OpenClaw represents a unique value proposition that neither ChatGPT nor existing HA tools can provide:

```
OpenClaw = CARE Knowledge Base (18K+ solutions)
          + Home Assistant API (user's actual setup)
          + Claude AI (conversational intelligence)
```

**What makes this breakthrough:**
- **ChatGPT**: Generic HA knowledge, NO access to user's setup, NO verified solutions
- **CARE/Moltbot**: 18K+ verified solutions, NO personalization, NO access to user's HA
- **OpenClaw**: Verified solutions + Direct access to user's setup + AI reasoning = **Personalized troubleshooting**

**Example Use Case:**
```
User: "My bedroom automation isn't working"

ChatGPT: "Check your YAML syntax..."
CARE: "Here are 47 automation-related solutions from our KB..."
OpenClaw: "I inspected your automation.bedroom_lights. The problem is on line 12:
           you reference sensor.bedroom_motion but that entity doesn't exist in your
           system. You have binary_sensor.bedroom_motion_sensor instead.

           Want me to fix it? [Apply Fix]"
```

---

## 📊 Infrastructure Status

### Brains Server (Production)
- **Hostname:** brains.synctacles.com
- **IP:** 173.249.55.109
- **Status:** ✅ Fully configured and monitored
- **Access:** `ssh cc-hub "ssh brains '...'"`
- **User:** `brains` (dedicated non-root user)

**Completed Setup:**
- ✅ SSH access via cc-hub
- ✅ Security hardening (firewall, updates, SSH keys)
- ✅ Prometheus monitoring (node_exporter on port 9100)
- ✅ Alert rules (BrainsMetricsDown, BrainsHighMemory, BrainsHighCPU)
- ✅ Dashboard integration (cc-hub monitoring)
- ✅ Documentation (CLAUDE.md, BRAINS-ARCHITECTURE.md)
- ✅ **KB Support Bot operational** (18K+ entries, @SynctaclesSupportBot)
- ✅ **Harvesters running hourly** (GitHub, Forum, Reddit, StackOverflow)

**Pending Deployment:**
- ⏳ **HA API Integration** (OAuth2 flow)
- ⏳ **Claude API integration** (tiered Haiku/Sonnet)
- ⏳ **One-click fix system** (YAML generation + service calls)
- ⏳ **Community intelligence** (pattern detection)

### DEV Server (Testing)
- **Hostname:** synct-dev
- **Purpose:** OpenClaw TEST instance + Platform bot
- **Resources:** 5GB RAM available, 52GB disk free
- **Status:** ✅ Ready for deployment

---

## 🏗️ Architecture Deep Dive

### Current State (Phase 0 - OPERATIONAL)

```
┌─────────────────────────────────────────────────────────────┐
│                      BRAINS SERVER (PROD)                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │  Telegram Bot    │◄────────┤  Support Agent   │          │
│  │  @Synctacles     │         │  (Python)        │          │
│  │  SupportBot      │         └────────┬─────────┘          │
│  └──────────────────┘                  │                    │
│                                         │                    │
│         │                               │                    │
│         ▼                               ▼                    │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │  Knowledge Base  │         │  Ollama          │          │
│  │  PostgreSQL      │         │  (Local LLM)     │          │
│  │  + pgvector      │         │  phi3:mini       │          │
│  │  18K+ entries    │         └──────────────────┘          │
│  └──────────────────┘                                        │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

**Capabilities:**
- ✅ Search 18K+ verified HA solutions
- ✅ FAQ responses
- ✅ Log analysis (GROQ API)
- ❌ NO access to user's HA setup
- ❌ NO personalized troubleshooting
- ❌ NO automated fixes

---

### Target State (Phase 2+ - BREAKTHROUGH)

```
┌─────────────────────────────────────────────────────────────┐
│                      BRAINS SERVER (PROD)                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │  Telegram Bot    │◄────────┤  OpenClaw API    │          │
│  │  (Public)        │         │  (FastAPI)       │          │
│  │  @OpenClawBot    │         └────────┬─────────┘          │
│  └──────────────────┘                  │                    │
│                                         │                    │
│         │                               │                    │
│         ▼                               ▼                    │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │  Knowledge Base  │         │  Home Assistant  │          │
│  │  PostgreSQL      │         │  API Client      │          │
│  │  + pgvector      │         │  (REST/WS)       │          │
│  │  18K+ entries    │         └──────────────────┘          │
│  └──────────────────┘                  │                    │
│         │                               │                    │
│         ▼                               ▼                    │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │  Ollama          │         │  Claude API      │          │
│  │  (Local LLM)     │         │  (Haiku/Sonnet)  │          │
│  │  Embeddings      │         │  Reasoning       │          │
│  └──────────────────┘         └──────────────────┘          │
│                                                               │
└─────────────────────────────────────────────────────────────┘

                            │
                            │ User: "Fix my automation"
                            ▼

            ┌───────────────────────────────┐
            │    User's Home Assistant      │
            │    (home.example.com)         │
            │                               │
            │  - Entities: 247              │
            │  - Automations: 42            │
            │  - Services: 156              │
            │                               │
            │  OpenClaw reads/writes here   │
            └───────────────────────────────┘
```

**New Capabilities:**
- ✅ Everything from Phase 0
- ✅ **Direct entity inspection** (light.bedroom state)
- ✅ **Automation analysis** (find broken entity references)
- ✅ **Root cause diagnosis** (Claude reasoning over KB + HA state)
- ✅ **One-click fixes** (generate + apply YAML changes)
- ✅ **Fix verification** (check if solution worked)

---

### Data Flow Example: "Why isn't my bedroom light turning on?"

```
1. USER → Telegram → OpenClaw Bot
   "Why isn't my bedroom light turning on?"

2. Bot → OpenClaw API → Home Assistant API
   GET /api/states/light.bedroom
   GET /api/states (find related automations)

3. OpenClaw API → PostgreSQL KB
   SELECT * FROM knowledge_base
   WHERE vector_similarity('light not responding') > 0.8
   LIMIT 10

4. OpenClaw API → Claude API (Haiku)
   Context: User's light.bedroom state: unavailable
           User's automation.bedroom_lights trigger: state change
           KB: 3 similar cases found (Zigbee issues)

   Prompt: "Analyze this light entity state and provide diagnosis"

5. Claude → OpenClaw API
   "The light is unavailable. Common causes:
    1. Zigbee coordinator offline (CHECK coordinator state)
    2. Device battery dead (CHECK battery sensor)
    3. Network interference"

6. OpenClaw API → Home Assistant API
   GET /api/states/sensor.coordinator_bedroom
   Status: unavailable ✓ (FOUND ROOT CAUSE)

7. OpenClaw API → Claude API (Sonnet - complex reasoning)
   "Root cause found: Zigbee coordinator offline.
    Provide step-by-step fix with HA service calls."

8. Claude → User (via Telegram)
   "🔍 Found the problem: Your Zigbee coordinator is offline.

   This explains why light.bedroom shows 'unavailable'.

   Fix steps:
   1. Restart Zigbee integration: [Apply Fix]
   2. If that fails, power cycle the USB coordinator

   I can apply step 1 automatically. Want me to?"

9. USER → "Yes, apply fix"

10. OpenClaw → Home Assistant API
    POST /api/services/homeassistant/reload_config_entry
    { "entry_id": "zigbee_coordinator_id" }

11. OpenClaw → User
    "✅ Restarted Zigbee integration.
    Waiting 30 seconds for devices to reconnect..."

    [After 30s]
    "✅ Confirmed: light.bedroom is now 'on'. Problem solved!"
```

**Key Differentiators:**
- Direct entity inspection (light.bedroom state)
- Related automation discovery
- KB-verified solution patterns
- Root cause analysis via coordinator check
- One-click fix application
- Verification of fix success

---

## 💾 CARE Knowledge Base Analysis

### Repository Overview
**Location:** `/opt/github/synctacles-care/`
**Size:** 22,534 lines of Python
**Tests:** 221 tests, 100% passing

### Knowledge Base Schema

**File:** Already deployed on brains server

```sql
CREATE TABLE knowledge_base (
    id SERIAL PRIMARY KEY,
    problem_title VARCHAR(500) NOT NULL,
    problem_description TEXT,
    problem_category VARCHAR(100),
    problem_component VARCHAR(100),
    problem_keywords TEXT[],
    solution_text TEXT NOT NULL,
    solution_steps JSONB,
    solution_code_snippets JSONB,
    confidence_score DECIMAL(3,2) DEFAULT 0.50,
    source VARCHAR(50),
    source_url TEXT,
    is_active BOOLEAN DEFAULT true,
    view_count INTEGER DEFAULT 0,
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Current Data (on brains server):**
- **17,297 entries** active
- **24 categories** (integrations, automations, troubleshooting, etc.)
- **Average confidence score:** 0.78
- **Sources:** GitHub issues, HA forums, documentation, Reddit

### Smart Classification System

**File:** `shared/knowledge_repository.py` (already deployed)

**Key Innovation:** 70-80% cost reduction via intelligent caching

```python
async def classify_log(self, log_content: str) -> dict:
    """
    Smart classification with caching:
    1. Compute embedding for log content
    2. Check cache for similar classifications (vector similarity > 0.95)
    3. If cache hit: Return cached classification (FREE)
    4. If cache miss: Call GROQ API ($$$), cache result
    """

    # This reduces GROQ API calls by 70-80%
    # Example: 100 "Zigbee coordinator offline" logs
    #          → 1 GROQ call + 99 cache hits
```

**Reusable for OpenClaw:**
- ✅ Already working on brains server
- ✅ Same caching strategy for KB queries
- ✅ Same embedding approach for semantic search
- ✅ Same confidence scoring for answer quality

---

## 🔌 Home Assistant API Integration (NEW - Phase 2)

### What CARE Can't Do (But OpenClaw Will)

**CARE Limitations:**
- ❌ Cannot access user's entities (`light.bedroom`)
- ❌ Cannot inspect automations
- ❌ Cannot read current states
- ❌ Cannot apply fixes via service calls
- ❌ Cannot verify if suggested solution worked

**OpenClaw Capabilities (Phase 2+):**

1. **Entity Inspection**
   ```python
   GET /api/states/light.bedroom
   {
     "entity_id": "light.bedroom",
     "state": "unavailable",
     "attributes": {
       "friendly_name": "Bedroom Light",
       "integration": "zigbee2mqtt",
       "last_seen": "2026-02-04T10:23:45"
     }
   }
   ```

2. **Automation Analysis**
   ```python
   GET /api/config/automation/config
   [
     {
       "id": "bedroom_lights",
       "alias": "Bedroom Motion Lights",
       "trigger": [
         {"platform": "state", "entity_id": "sensor.bedroom_motion"}
       ],
       "action": [
         {"service": "light.turn_on", "target": {"entity_id": "light.bedroom"}}
       ]
     }
   ]
   ```

3. **Service Execution**
   ```python
   POST /api/services/light/turn_on
   {
     "entity_id": "light.bedroom",
     "brightness": 255
   }
   ```

4. **Configuration Validation**
   ```python
   POST /api/config/core/check_config
   {
     "valid": false,
     "errors": ["automation.bedroom_lights: entity sensor.bedroom_motion not found"]
   }
   ```

### Security Model

**User Authorization Flow:**

```
1. User starts conversation with OpenClaw bot
2. Bot: "To help you, I need access to your Home Assistant.
        Click here to authorize: [Authorize]"
3. User clicks → Redirected to HA OAuth flow
4. User: Grants OpenClaw permissions (read states, write automations, call services)
5. HA: Returns access token (JWT, 10-year expiry)
6. OpenClaw: Stores encrypted token in PostgreSQL
7. All future API calls use this token

Permissions requested:
- Read: entities, automations, configuration
- Write: automations (after user confirmation)
- Execute: services (after user confirmation)
```

**Token Security:**
- Encrypted at rest (Fernet encryption)
- Scoped to specific HA instance
- Revocable by user via HA UI
- Rate limited (1000 requests/hour per user)

---

## 💰 Cost Analysis & Optimization

### Without Optimization (Baseline)

**Assumptions:**
- 10,000 active users
- Average 5 queries/user/day = 50,000 queries/day
- All queries use Claude Sonnet
- Average context: 4K input tokens, 1K output tokens

**Monthly Cost:**
```
Input:  50,000 queries × 4K tokens × 30 days × $3/MTok = $18,000
Output: 50,000 queries × 1K tokens × 30 days × $15/MTok = $22,500
Total: $40,500/month 😱
```

### With Optimization (Tiered Strategy)

**Smart Routing:**

```
┌─────────────────────────────────────────────────────┐
│                   Query Analysis                     │
│  "Can this be answered from KB alone?"              │
└────────────┬────────────────────────────────────────┘
             │
    ┌────────┴────────┐
    │                 │
    ▼                 ▼
┌─────────┐      ┌──────────────────────────────┐
│ KB Only │ 30%  │  Requires HA API Access      │ 70%
│ (FREE)  │      └──────────┬───────────────────┘
└─────────┘                 │
                   ┌────────┴────────┐
                   │                 │
            Simple │ 50%      Complex│ 20%
                   ▼                 ▼
            ┌──────────┐      ┌──────────┐
            │  Haiku   │      │  Sonnet  │
            │  $0.25/  │      │  $3/     │
            │  MTok    │      │  MTok    │
            └──────────┘      └──────────┘
```

### With Caching + Optimization

**Cache Hit Rates:**
- KB query cache: 70% hit rate (similar questions)
- HA API cache: 50% hit rate (entity states, 5min TTL)
- Claude response cache: 40% hit rate (similar contexts)

**Effective Queries After Caching:**

| Tier | Original | After Cache | Cost/Day | Cost/Month |
|------|----------|-------------|----------|------------|
| KB Only | 15,000 | 15,000 | $0 | $0 |
| Haiku | 25,000 | 15,000 (40% cached) | $6.25 | $187 |
| Sonnet | 10,000 | 6,000 (40% cached) | $30 | $900 |
| **TOTAL** | 50,000 | 36,000 | **$36.25** | **$1,087** |

**Cost per User:** $0.11/month

**Scalability:**
- 1K users: $109/month ✅
- 10K users: $1,087/month ✅
- 100K users: $10,870/month ⚠️ (optimize further)

---

## 🚀 Implementation Phases

### Phase 1: Foundation ✅ COMPLETE (2026-02-04)
**Goal:** OpenClaw can answer questions using KB only (like CARE, but better UX)

**Completed:**
- ✅ Brains server infrastructure setup
- ✅ Deployed PostgreSQL + pgvector
- ✅ Migrated 18K KB entries from CARE database
- ✅ Deployed Ollama (phi3:mini for embeddings)
- ✅ Built support bot with KB search
- ✅ Deployed to production (@SynctaclesSupportBot)
- ✅ Harvesters running hourly

**Testing Criteria:** ✅ ALL MET
- User can ask HA questions via Telegram
- Bot searches KB and returns relevant solutions
- Response time < 3 seconds
- Services running stable (14+ minutes uptime)

**Example Interaction:**
```
User: "My Zigbee devices keep dropping"
Bot: "I found 3 relevant solutions from our knowledge base:

     1. Check Zigbee channel interference (confidence: 0.92)
     2. Update Zigbee2MQTT to latest version (confidence: 0.87)
     3. Increase coordinator TX power (confidence: 0.81)

     Would you like details on any of these?"
```

---

### Phase 2: HA API Integration ⏳ NEXT (Week 3-4)
**Goal:** OpenClaw can inspect user's actual HA setup

**Tasks:**
1. ⏳ Implement HA OAuth2 flow
2. ⏳ Build HA API client (REST + WebSocket)
3. ⏳ Add token management (encryption, renewal)
4. ⏳ Implement entity inspector
5. ⏳ Add automation analyzer
6. ⏳ Build Claude integration (Haiku tier)
7. ⏳ Add smart routing (KB → Haiku → Sonnet)

**Testing Criteria:**
- User can authorize HA access via OAuth
- Bot can read entity states
- Bot can analyze automations
- Response includes actual entity data
- Smart routing reduces costs by 60%

**Example Interaction:**
```
User: "Why isn't my bedroom automation working?"
Bot: "Let me check your setup..."
     [Inspecting automation.bedroom_lights...]
     [Checking entities: sensor.bedroom_motion, light.bedroom...]

     "Found the issue! Your automation references sensor.bedroom_motion,
     but that entity doesn't exist. You have:
     - binary_sensor.bedroom_motion_sensor ✓

     The entity was renamed in HA 2024.12. Want me to fix it?"
```

---

### Phase 3: One-Click Fixes (Week 5-6)
**Goal:** OpenClaw can apply fixes automatically with user approval

**Tasks:**
1. ⏳ Build YAML generator for automations
2. ⏳ Add diff preview for changes
3. ⏳ Implement service call executor
4. ⏳ Add rollback mechanism
5. ⏳ Build fix verification system
6. ⏳ Add user confirmation flow

**Testing Criteria:**
- Bot generates valid YAML for fixes
- User sees diff before applying
- Service calls execute successfully
- Fixes are verified automatically
- Rollback works if fix fails

**Example Interaction:**
```
User: "Yes, fix it"
Bot: "Preview of changes:

     automation.bedroom_lights:
     - sensor.bedroom_motion
     + binary_sensor.bedroom_motion_sensor

     This will:
     1. Update automation YAML
     2. Reload automations
     3. Test the trigger

     Confirm? [Yes] [No]"

User: "Yes"
Bot: "✅ Applied fix
     ✅ Reloaded automations
     ✅ Tested trigger: Working correctly!

     Your automation is now fixed. Try it out!"
```

---

### Phase 4: Community Intelligence (Week 7-8)
**Goal:** Learn from all users to provide version-specific insights

**Tasks:**
1. ⏳ Add anonymous telemetry (opt-in)
2. ⏳ Build pattern detection system
3. ⏳ Create version-specific KB entries
4. ⏳ Add proactive notifications
5. ⏳ Build community insights dashboard

**Testing Criteria:**
- Users can opt-in to telemetry
- System detects common patterns (e.g., "50% of 2024.12 users have Zigbee issues")
- Proactive notifications sent before user asks
- Privacy preserved (no PII stored)

**Example Interaction:**
```
Bot: "📢 Community Alert

     I've noticed you're running HA 2024.12.3.
     47% of users on this version reported Zigbee coordinator crashes
     after update.

     Recommended actions:
     1. Pin Zigbee2MQTT to version 1.35.1 (stable)
     2. Update to HA 2024.12.4 (fixes the issue)

     Want me to check if you're affected? [Check Status]"
```

---

## 📋 Phase 2 Deployment Checklist (NEXT PRIORITY)

### Code Repositories

**Create New Repos:**
```bash
# 1. OpenClaw API (FastAPI backend)
gh repo create synctacles/openclaw-api --public \
  --description "OpenClaw API - Personalized Home Assistant support with HA API integration"

# 2. OpenClaw Bot (Telegram bot - enhanced version)
gh repo create synctacles/openclaw-bot --public \
  --description "OpenClaw Telegram Bot - Conversational HA troubleshooting"
```

**Repository Structure:**

```
openclaw-api/
├── main.py                     # FastAPI app
├── routers/
│   ├── auth.py                # OAuth2 flow for HA
│   ├── ha_api.py              # HA API client
│   ├── kb.py                  # KB search (reuse from CARE)
│   └── claude.py              # Claude API integration
├── models/
│   ├── user.py                # User + HA token storage
│   ├── kb.py                  # KB models (reuse)
│   └── ha.py                  # HA entity/automation models
├── services/
│   ├── ha_inspector.py        # Entity inspection logic
│   ├── automation_analyzer.py # Automation analysis
│   ├── smart_router.py        # KB/Haiku/Sonnet routing
│   └── token_manager.py       # Token encryption/renewal
├── requirements.txt
├── .env.example
└── README.md

openclaw-bot/
├── main.py                     # Bot entry point
├── handlers/
│   ├── start.py               # /start command
│   ├── authorize.py           # /authorize (HA OAuth)
│   ├── troubleshoot.py        # Conversational troubleshooting
│   └── feedback.py            # User feedback
├── api_client.py              # OpenClaw API client
├── requirements.txt
└── README.md
```

### Database Schema Updates

**Add to brains_kb database:**

```sql
-- User HA tokens (encrypted)
CREATE TABLE public.user_ha_tokens (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(20) DEFAULT 'telegram',
    platform_user_id VARCHAR(100) NOT NULL,
    ha_instance_url VARCHAR(500) NOT NULL,
    encrypted_token TEXT NOT NULL,
    token_expires_at TIMESTAMPTZ,
    refresh_token TEXT,
    permissions JSONB,  -- ['read', 'write', 'execute']
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    UNIQUE(platform, platform_user_id)
);

-- Query logs for caching
CREATE TABLE public.query_cache (
    id SERIAL PRIMARY KEY,
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    query_text TEXT NOT NULL,
    response JSONB NOT NULL,
    model_used VARCHAR(20),  -- 'kb', 'haiku', 'sonnet'
    tokens_used INTEGER,
    cost_usd DECIMAL(10,4),
    hit_count INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_query_cache_expires ON public.query_cache (expires_at);
CREATE INDEX idx_query_cache_hash ON public.query_cache (query_hash);

-- Cost tracking
CREATE TABLE public.api_costs (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    model VARCHAR(20) NOT NULL,
    queries_total INTEGER DEFAULT 0,
    queries_cached INTEGER DEFAULT 0,
    tokens_input BIGINT DEFAULT 0,
    tokens_output BIGINT DEFAULT 0,
    cost_usd DECIMAL(10,2) DEFAULT 0,
    UNIQUE(date, model)
);
```

### Brains Server Deployment

**1. Deploy OpenClaw API:**

```bash
ssh cc-hub 'ssh brains "
cd /opt/openclaw
git clone https://github.com/synctacles/openclaw-api.git api
cd api

# Setup venv
python3.12 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Configure
cp .env.example .env
nano .env
"'

# Add to .env:
DATABASE_URL=postgresql://brains_admin:***@localhost:5432/brains_kb?sslmode=disable
CLAUDE_API_KEY=sk-ant-api03-***
FERNET_ENCRYPTION_KEY=<generate with: python -c 'from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())'>
HA_OAUTH_CLIENT_ID=openclaw
HA_OAUTH_CLIENT_SECRET=<generate random string>
HA_OAUTH_REDIRECT_URI=https://brains.synctacles.com/auth/callback
```

**2. Create systemd service:**

```bash
ssh cc-hub 'ssh brains "sudo tee /etc/systemd/system/openclaw-api.service > /dev/null << 'EOF'
[Unit]
Description=OpenClaw API
After=network.target postgresql.service

[Service]
Type=simple
User=brains
WorkingDirectory=/opt/openclaw/api
Environment=\"PATH=/opt/openclaw/api/venv/bin\"
ExecStart=/opt/openclaw/api/venv/bin/uvicorn main:app --host 0.0.0.0 --port 8000
Restart=always
RestartSec=10s

# Security
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true
PrivateTmp=true
ReadWritePaths=/opt/openclaw/api /opt/openclaw/logs

# Resources
MemoryMax=1G
TasksMax=200

[Install]
WantedBy=multi-user.target
EOF
"'

ssh cc-hub 'ssh brains "
sudo systemctl daemon-reload
sudo systemctl enable --now openclaw-api
sudo systemctl status openclaw-api
"'
```

**3. Update Nginx (add API endpoint):**

```bash
ssh cc-hub 'ssh brains "sudo nano /etc/nginx/sites-available/openclaw"'

# Add after existing config:
location /api/ {
    proxy_pass http://localhost:8000/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}

location /auth/ {
    proxy_pass http://localhost:8000/auth/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}

ssh cc-hub 'ssh brains "
sudo nginx -t
sudo systemctl reload nginx
"'
```

**4. Enable Prometheus metrics:**

```bash
# Update prometheus.yml on monitoring server
ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'nano /opt/monitoring/prometheus/prometheus.yml'"

# Uncomment:
- job_name: "brains-prod-api"
  scheme: https
  metrics_path: /metrics
  static_configs:
    - targets: ["brains.synctacles.com:443"]

# Reload
ssh cc-hub "ssh -i ~/.ssh/id_monitoring monitoring@77.42.41.135 'docker exec prometheus kill -HUP 1'"
```

---

## 🎯 Success Metrics

### Phase 1 ✅ ACHIEVED
- [x] Infrastructure setup complete
- [x] KB bot operational (17,297 entries)
- [x] Harvesters running hourly
- [x] Monitoring active

### Phase 2 (HA API Integration)
- [ ] 500 users authorized HA access
- [ ] 5,000 entity inspections
- [ ] > 80% query accuracy (verified by user feedback)
- [ ] < $0.15 cost per user per month
- [ ] < 5 sec average response time

### Phase 3 (One-Click Fixes)
- [ ] 1,000 fixes applied
- [ ] > 95% fix success rate
- [ ] 0 data loss incidents
- [ ] < 1% rollback rate

### Phase 4 (Community Intelligence)
- [ ] 50% opt-in rate for telemetry
- [ ] 100+ detected patterns
- [ ] 1,000+ proactive notifications sent
- [ ] > 80% proactive fix acceptance rate

---

## 🔒 Security Considerations

### Data Privacy

**What We Store:**
- KB entries (public HA community knowledge)
- User's Telegram ID
- Encrypted HA access tokens
- Anonymous query telemetry (opt-in)

**What We DON'T Store:**
- User's HA entity states (fetched on-demand, not persisted)
- User's automation YAML (processed in-memory only)
- Personal information (names, addresses, etc.)

**GDPR Compliance:**
- [ ] User can request data export
- [ ] User can request data deletion
- [ ] All PII encrypted at rest
- [ ] Clear privacy policy in bot /start message

### Token Security

**Encryption:**
```python
from cryptography.fernet import Fernet

# Generate key (store in environment variable)
key = Fernet.generate_key()
cipher = Fernet(key)

# Encrypt token before storing
encrypted_token = cipher.encrypt(ha_token.encode())

# Decrypt when needed
decrypted_token = cipher.decrypt(encrypted_token).decode()
```

**Token Rotation:**
- Tokens expire after 10 years (HA default)
- Refresh token flow implemented
- User notified if token invalid

**Rate Limiting:**
- 1,000 HA API calls per user per hour
- 100 Claude API calls per user per hour
- Protection against abuse

---

## 📞 Support & Monitoring

### Alert Thresholds

**Critical Alerts → #critical-alerts:**
- OpenClaw API down (> 5 min)
- Database connection lost
- Redis connection lost
- Telegram bot offline (> 5 min)
- Error rate > 10%

**Warning Alerts → #warnings:**
- Response time > 5 sec (95th percentile)
- Cache hit rate < 50%
- API costs > $50/day
- Disk usage > 80%
- Memory usage > 85%

### Dashboards

**Grafana Dashboards (to be created):**
1. **OpenClaw Overview**
   - Queries per second
   - Response time percentiles (p50, p95, p99)
   - Error rate
   - Cost per query

2. **Knowledge Base**
   - KB entries count
   - Search performance
   - Cache hit rate
   - Most queried topics

3. **HA API Integration**
   - OAuth success rate
   - API call latency
   - Token refresh rate
   - Entity inspection count

4. **Cost Analysis**
   - Daily API costs (Claude + HA)
   - Cost per user
   - Cost per query
   - Tier distribution (KB/Haiku/Sonnet)

---

## 🚨 Known Risks & Mitigations

### Risk 1: HA API Rate Limiting
**Impact:** User queries fail if HA instance rate limits us
**Probability:** Medium (HA default: 100 req/min)
**Mitigation:**
- Implement caching (5min TTL for entity states)
- Batch entity requests when possible
- Educate users about HA rate limits
- Provide fallback to KB-only mode

### Risk 2: Claude API Costs Spike
**Impact:** Budget overrun if usage exceeds projections
**Probability:** Medium (viral growth scenario)
**Mitigation:**
- Hard cap: $100/day budget
- Alert at $50/day
- Auto-downgrade to KB-only if cap reached
- Implement waiting queue for non-critical queries

### Risk 3: KB Data Quality
**Impact:** Wrong solutions provided to users
**Probability:** Low (CARE data well-vetted)
**Mitigation:**
- User feedback on every answer
- Flag entries with < 3.0/5.0 rating for review
- Manual review process for new entries
- Community voting system (Phase 4)

### Risk 4: Security Breach (HA Token Leak)
**Impact:** Attacker gains access to user's HA instance
**Probability:** Low (encrypted storage)
**Mitigation:**
- Encrypt tokens at rest (Fernet)
- Rotate encryption keys quarterly
- Audit logs for all token access
- Incident response plan documented
- Auto-revoke tokens on suspicious activity

### Risk 5: Telegram Bot Abuse
**Impact:** Spam queries, resource exhaustion
**Probability:** Medium (public bot)
**Mitigation:**
- Rate limiting per user (10 queries/hour free tier)
- CAPTCHA for new users
- Ban list for abusive users
- Cloudflare protection for API

---

## 📚 References

### Documentation
- **CARE Repository:** `/opt/github/synctacles-care/`
- **Brains Architecture:** [/opt/github/synctacles-api/docs/BRAINS-ARCHITECTURE.md](BRAINS-ARCHITECTURE.md)
- **Setup Guide:** `/tmp/BRAINS_SERVER_SETUP.md`
- **Platform Docs:** [/opt/github/synctacles-api/CLAUDE.md](../CLAUDE.md)
- **HA API Docs:** https://developers.home-assistant.io/docs/api/rest/

### Code Repositories
- **OpenClaw API:** `https://github.com/synctacles/openclaw-api` (TO BE CREATED)
- **OpenClaw Bot:** `https://github.com/synctacles/openclaw-bot` (TO BE CREATED)
- **CARE (Reference):** `https://github.com/synctacles/care`
- **Brains (Harvesters):** `/opt/openclaw/harvesters` on brains server

### Key Technologies
- **FastAPI:** https://fastapi.tiangolo.com/
- **python-telegram-bot:** https://python-telegram-bot.org/
- **pgvector:** https://github.com/pgvector/pgvector
- **Ollama:** https://ollama.ai/
- **Claude API:** https://docs.anthropic.com/
- **HA API:** https://developers.home-assistant.io/

---

## 🎬 Next Steps for CAI

**Immediate Actions:**

1. **Create GitHub Repositories**
   ```bash
   gh repo create synctacles/openclaw-api --public --description "OpenClaw API - Personalized Home Assistant support"
   gh repo create synctacles/openclaw-bot --public --description "OpenClaw Telegram Bot"
   ```

2. **Initialize API Codebase**
   - FastAPI project structure
   - HA OAuth2 implementation
   - HA API client (REST + WebSocket)
   - Claude API integration (tiered routing)
   - Token encryption (Fernet)
   - Caching layer (PostgreSQL-based)

3. **Initialize Bot Codebase**
   - Telegram bot setup (python-telegram-bot)
   - Command handlers (/authorize, /troubleshoot, /fix)
   - API client for OpenClaw API
   - Conversational flow design

4. **Deploy Phase 2 to Brains Server**
   - Follow deployment checklist above
   - Verify all services running
   - Test OAuth flow
   - Test entity inspection

5. **Create GitHub Project Board**
   - Milestones for each phase
   - Issues for all tasks
   - Link to this architecture doc

---

## ✨ Why This Is Breakthrough

**The Competition:**

| Solution | KB Access | User's HA Access | AI Reasoning | Cost | Verdict |
|----------|-----------|------------------|--------------|------|---------|
| **ChatGPT** | Generic | ❌ No | ✅ Yes | Free-$20/mo | Generic answers only |
| **HA Forums** | ✅ Yes | ❌ No | ❌ No | Free | Manual search, no personalization |
| **CARE/Moltbot** | ✅ Yes (18K+) | ❌ No | Limited (GROQ) | Free | No personalization |
| **HA Assist** | ❌ No | ✅ Yes | ❌ No | Free | Voice control only, no troubleshooting |
| **OpenClaw** | ✅ Yes (18K+) | ✅ Yes | ✅ Yes (Claude) | $0.11/user | **🚀 Personalized + Verified + AI** |

**What Makes OpenClaw Unique:**

1. **Verified Solutions**: 18K+ real problems solved by HA community
2. **Your Actual Setup**: Direct access to your entities, automations, logs
3. **AI Reasoning**: Claude connects the dots between KB and your setup
4. **One-Click Fixes**: Not just advice, but executable solutions
5. **Community Intelligence**: Learn from all users to prevent issues before they happen

**User Value Proposition:**

> "Ask OpenClaw: Why isn't my automation working?
>
>  ChatGPT says: 'Check your YAML syntax'
>  CARE says: 'Here are 47 automation solutions'
>  OpenClaw says: 'Your sensor.bedroom_motion was renamed to binary_sensor.bedroom_motion_sensor in HA 2024.12. [Fix it now]'
>
>  Problem solved in 10 seconds. No searching, no trial-and-error, just working."

---

**Document Status:** 🟢 READY FOR IMPLEMENTATION - PHASE 2
**Last Updated:** 2026-02-04
**Next Review:** After Phase 2 deployment

**Questions?** Contact: @synctacles-dev on Telegram

---

*This document represents a breakthrough architecture for the Home Assistant community. Phase 1 (KB foundation) is complete and operational. Phase 2 (HA API integration) is the next priority and will unlock the unique personalized troubleshooting capabilities that make OpenClaw truly revolutionary.*
