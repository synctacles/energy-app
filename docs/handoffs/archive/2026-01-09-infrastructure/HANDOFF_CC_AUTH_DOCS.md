# HANDOFF: API Key Systeem Documentatie

**Van:** Claude (Opus)  
**Naar:** Claude Code  
**Datum:** 2026-01-09  
**GitHub Issues:** #48 (closed), #49 (check)

---

## CONTEXT

API Key systeem is volledig geïmplementeerd en getest, maar **niet gedocumenteerd**. AI assistants en toekomstige developers weten niet dat het bestaat of hoe het werkt.

---

## OPDRACHT

### 1. Documenteer in ARCHITECTURE.md

Onder `## Security Model`, nieuwe sectie:

```markdown
### Authentication System

**Status:** Implemented, disabled by default (MVP free tier)

**Feature Flags:**
| Flag | Default | Effect |
|------|---------|--------|
| AUTH_REQUIRED | false | Endpoints vereisen API key |
| RATE_LIMIT_ENABLED | false | Rate limiting actief |

**Flow:**
```
Request → Middleware → Check AUTH_REQUIRED
                      ↓
              false: pass through
              true: validate X-API-Key header
                      ↓
              invalid: 401 Unauthorized
              valid: attach user to request → continue
```

**Key Storage:**
- API keys SHA-256 gehashed in database
- Plaintext key alleen bij generatie getoond (één keer)

**Models:**
- `User` — id, email, api_key_hash, created_at, is_active
- `APIUsage` — tracking per user per endpoint

**Endpoints:**
| Endpoint | Method | Auth | Beschrijving |
|----------|--------|------|--------------|
| /api/v1/auth/signup | POST | None | Maak user, ontvang API key |
| /api/v1/auth/stats | GET | Key | User info + usage stats |
| /api/v1/auth/regenerate | POST | Key | Genereer nieuwe key |
| /api/v1/auth/deactivate | POST | Key | Deactiveer account |
| /api/v1/admin/users | GET | Admin | Lijst alle users |
```

---

### 2. Documenteer in api-reference.md

Voeg auth endpoints sectie toe:

```markdown
## Authentication Endpoints

### POST /api/v1/auth/signup

Maak nieuw account en ontvang API key.

**Request:**
```json
{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "api_key": "sk_live_xxxxx"  // Alleen bij signup getoond!
}
```

---

### GET /api/v1/auth/stats

Haal account info en usage statistics op.

**Headers:**
```
X-API-Key: sk_live_xxxxx
```

**Response:**
```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "created_at": "2026-01-09T12:00:00Z",
  "rate_limit": {
    "requests_today": 150,
    "limit": 1000
  }
}
```

---

### POST /api/v1/auth/regenerate

Genereer nieuwe API key. Oude key wordt ongeldig.

**Headers:**
```
X-API-Key: sk_live_xxxxx (oude key)
```

**Response:**
```json
{
  "api_key": "sk_live_yyyyy"  // Nieuwe key
}
```

---

### POST /api/v1/auth/deactivate

Deactiveer account permanent.

**Headers:**
```
X-API-Key: sk_live_xxxxx
```

**Response:**
```json
{
  "status": "deactivated"
}
```
```

---

### 3. Update SKILL_02_ARCHITECTURE.md of SKILL_03_CODING_STANDARDS.md

Voeg toe hoe auth toe te passen op nieuwe endpoints:

```markdown
### Adding Auth to New Endpoints

Endpoints achter auth middleware plaatsen:

1. Endpoint NIET in `EXEMPT_PATHS` lijst (middleware.py)
2. User beschikbaar via `request.state.user`

```python
@router.get("/protected")
async def protected_endpoint(request: Request):
    user = request.state.user  # Gevalideerde user
    return {"user_id": user.id}
```

Feature flags in .env:
```
AUTH_REQUIRED=true      # Activeer auth
RATE_LIMIT_ENABLED=true # Activeer rate limiting
```
```

---

### 4. Update user-guide.md

Voeg sectie toe voor eindgebruikers:

```markdown
## Getting Your API Key

1. Ga naar `https://api.example.com/api/v1/auth/signup`
2. POST met je email
3. Bewaar je API key veilig — deze wordt maar één keer getoond
4. Gebruik header `X-API-Key: jouw_key` bij alle requests

### Key kwijt?

Regenereer via `/api/v1/auth/regenerate` met je huidige key.
Als je geen toegang meer hebt, neem contact op met support.
```

---

### 5. Check #49 User Management

Vergelijk #49 scope met wat #48 al implementeert:

**#48 implementeert al:**
- User signup
- API key generatie
- Key regeneratie
- Account deactivatie
- Usage tracking
- Admin user list

**#49 zou kunnen toevoegen:**
- Email verificatie?
- Password login (naast API key)?
- User roles/permissions?
- Account recovery?

**Rapporteer:** Is #49 volledig gedekt door #48, of blijft er scope over?

---

## FILES BETROKKEN

Lees eerst:
- `synctacles_db/auth_models.py`
- `synctacles_db/api/endpoints/auth.py`
- `synctacles_db/api/middleware.py`

Update:
- `/opt/github/synctacles-api/docs/ARCHITECTURE.md`
- `/opt/github/synctacles-api/docs/api-reference.md`
- `/opt/github/synctacles-api/docs/user-guide.md`
- Relevante SKILL file

---

## GIT

```bash
sudo -u energy-insights-nl git -C /opt/github/synctacles-api add -A
sudo -u energy-insights-nl git -C /opt/github/synctacles-api commit -m "docs: API key authentication system documentation #48"
sudo -u energy-insights-nl git -C /opt/github/synctacles-api push
```

---

## EXIT CRITERIA

- [ ] ARCHITECTURE.md: Auth system sectie toegevoegd
- [ ] api-reference.md: Auth endpoints gedocumenteerd
- [ ] user-guide.md: API key instructies toegevoegd
- [ ] SKILL file: Auth integration guide
- [ ] #49 scope geanalyseerd en gerapporteerd
