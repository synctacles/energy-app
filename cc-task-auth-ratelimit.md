# CC Task: Auth & Rate Limiting System

## Context

Server: 135.181.255.83 (SSH as root)
Project: /opt/github/ha-energy-insights-nl
Venv: /opt/energy-insights-nl/venv
Database: energy_insights_nl (PostgreSQL)
Service: energy-insights-nl-api

## Doel

Implementeer API key authenticatie en rate limiting met feature flags.

---

## Stap 1: Database Migratie

Maak Alembic migratie voor:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'active',  -- active/suspended/cancelled
    tier VARCHAR(20) DEFAULT 'beta'       -- beta/free/paid/unlimited
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(64) NOT NULL,        -- SHA256 hash
    key_prefix VARCHAR(8) NOT NULL,       -- First 8 chars for identification
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NULL,            -- NULL = never expires
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP NULL
);

CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);
CREATE INDEX idx_users_email ON users(email);
```

---

## Stap 2: Feature Flags in .env

Voeg toe aan `/opt/.env`:

```bash
# Auth & Rate Limiting (disabled by default)
AUTH_REQUIRED=false
RATE_LIMIT_ENABLED=false
DEFAULT_TIER=beta
```

Update `synctacles_db/config/settings.py` om deze te laden met defaults.

---

## Stap 3: Tier Config

Maak `synctacles_db/auth/tiers.py`:

```python
TIER_LIMITS = {
    "beta": 10_000,
    "free": 1_000,
    "paid": 100_000,
    "unlimited": 100_000,
}

def get_rate_limit(tier: str) -> int:
    return TIER_LIMITS.get(tier, TIER_LIMITS["free"])
```

---

## Stap 4: Auth Services

Maak `synctacles_db/auth/services.py`:

Functies:
- `generate_api_key()` → returns plain key (64 hex chars)
- `hash_api_key(key)` → SHA256 hash
- `create_user(email)` → creates user + api_key, returns {user_id, api_key}
- `validate_api_key(key)` → returns user or None
- `regenerate_api_key(user_id)` → deactivates old, creates new
- `get_user_stats(user_id)` → returns {tier, usage_today, limit}
- `increment_usage(user_id)` → track daily usage (in-memory dict, reset daily)

---

## Stap 5: Middleware

Maak `synctacles_db/api/middleware.py`:

### AuthMiddleware
- Check header `X-API-Key`
- If AUTH_REQUIRED=false → skip (allow all)
- If AUTH_REQUIRED=true → validate key, 401 if invalid
- Attach user to request state

### RateLimitMiddleware  
- If RATE_LIMIT_ENABLED=false → skip
- If RATE_LIMIT_ENABLED=true → check usage vs tier limit
- 429 if exceeded with headers:
  - `X-RateLimit-Limit`
  - `X-RateLimit-Remaining`
  - `X-RateLimit-Reset` (UTC timestamp)

---

## Stap 6: Auth Endpoints

Maak `synctacles_db/api/routes/auth.py`:

### POST /auth/signup
```python
Request: {"email": "user@example.com"}
Response: {
    "user_id": "uuid",
    "email": "user@example.com",
    "api_key": "64-char-hex-key",  # Only shown once!
    "tier": "beta",
    "message": "Save your API key - it cannot be retrieved later!"
}
```
- Validate email format
- Check email not exists (409 if duplicate)
- Create user + key
- Return key (plain, only time shown)

### GET /auth/stats
```python
Headers: X-API-Key: <key>
Response: {
    "user_id": "uuid",
    "email": "user@example.com",
    "tier": "beta",
    "rate_limit_daily": 10000,
    "usage_today": 42,
    "remaining_today": 9958
}
```

### POST /auth/regenerate
```python
Headers: X-API-Key: <current-key>
Response: {
    "new_api_key": "64-char-hex-key",
    "message": "Old key is now invalid."
}
```

---

## Stap 7: Register in FastAPI

Update `synctacles_db/api/main.py`:
- Add middleware (order: RateLimit first, then Auth)
- Register auth router
- Exclude `/auth/signup` and `/health` from auth requirement

---

## Stap 8: Documentatie Updates

### api-reference.md
- Add Authentication section
- Add auth endpoints
- Add rate limit info

### SKILL_02_ARCHITECTURE.md
- Add Auth section with diagram

### SKILL_04_PRODUCT_REQUIREMENTS.md
- Add tiers table

### SKILL_09_INSTALLER_SPECS.md
- Add .env variabelen

---

## Stap 9: Deploy & Test

```bash
# Migratie
cd /opt/energy-insights-nl/app
source /opt/energy-insights-nl/venv/bin/activate
alembic upgrade head

# Restart
systemctl restart energy-insights-nl-api

# Test endpoints
curl -X POST https://enin.xteleo.nl/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com"}'

# Test met key
curl -H "X-API-Key: <received-key>" \
  https://enin.xteleo.nl/auth/stats

# Test regenerate
curl -X POST -H "X-API-Key: <key>" \
  https://enin.xteleo.nl/auth/regenerate
```

---

## Stap 10: Validatie Checklist

- [ ] users tabel bestaat
- [ ] api_keys tabel bestaat
- [ ] POST /auth/signup werkt
- [ ] GET /auth/stats werkt (met key)
- [ ] POST /auth/regenerate werkt
- [ ] 401 bij ongeldige key (als AUTH_REQUIRED=true)
- [ ] 429 bij rate limit exceeded (als RATE_LIMIT_ENABLED=true)
- [ ] Feature flags werken (false = open access)
- [ ] Documentatie bijgewerkt
- [ ] Git commit + push

---

## Constraints

- KISS: Geen email versturen, key direct tonen
- Geen Mollie integratie (later)
- Geen admin endpoints (later)
- In-memory rate limit tracking (geen Redis)
- Feature flags default OFF

---

## Exit Criteria

1. Alle 3 auth endpoints werken
2. Feature flags configureerbaar
3. Rate limits per tier correct
4. Documentatie bijgewerkt
5. Tests geslaagd
6. Git gepusht
