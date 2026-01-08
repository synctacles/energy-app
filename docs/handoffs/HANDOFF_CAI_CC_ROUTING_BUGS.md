# HANDOFF: CAI → CC

**Datum:** 2026-01-08
**Van:** CAI
**Naar:** CC
**Prioriteit:** HIGH
**Type:** Bug Fixes

---

## CONTEXT

Tijdens endpoint routing audit (#40) zijn 2 bugs ontdekt die gefixed moeten worden.

---

## BUG 1: /api/v1/now Returns 500 Error

**Prioriteit:** HIGH

**Symptoom:**
```bash
curl http://localhost:8000/api/v1/now
# HTTP 500 Internal Server Error
```

**Verwacht:** Endpoint moet huidige timestamp + pipeline status retourneren, of 404 als niet geïmplementeerd.

**Diagnose:**
```bash
# Check logs voor error details
sudo journalctl -u energy-insights-nl-api --since "1 hour ago" | grep -i "now\|error\|500"

# Check route implementatie
grep -rn "now" /opt/energy-insights-nl/app/synctacles_db/api/routes/

# Check of endpoint bestaat
cat /opt/energy-insights-nl/app/synctacles_db/api/main.py | grep -i "now"
```

**Mogelijke oorzaken:**
1. Route bestaat maar handler crasht (missing import, DB error)
2. Route verwijst naar niet-bestaande functie
3. Dependency injection failure

**Fix opties:**
- A) Repareer handler als endpoint bedoeld is
- B) Verwijder route als endpoint niet bedoeld is
- C) Return 501 Not Implemented als placeholder

**Rapporteer:** Root cause + gekozen fix

---

## BUG 2: /api/v1/balance Returns 501 Instead of 410

**Prioriteit:** LOW

**Symptoom:**
```bash
curl http://localhost:8000/api/v1/balance
# HTTP 501 Not Implemented
```

**Verwacht:** HTTP 410 Gone (TenneT is deprecated, niet "not implemented")

**Context:** TenneT balance endpoint is deprecated (zie issue #39). De juiste HTTP status is:
- 410 Gone = "Resource existed but is permanently removed"
- 501 Not Implemented = "Server doesn't support this functionality"

**Fix:**
```python
# In balance route handler
from fastapi import HTTPException

@router.get("/balance")
async def get_balance():
    raise HTTPException(
        status_code=410,
        detail={
            "error": "Gone",
            "message": "TenneT balance endpoint is deprecated. See ADR-001 for BYO-key model via Home Assistant integration.",
            "migration": "https://docs.synctacles.com/migration/tennet-byo-key"
        }
    )
```

**Locatie:** `/opt/energy-insights-nl/app/synctacles_db/api/routes/balance.py` (of waar balance route gedefinieerd is)

---

## DELIVERABLES

1. [ ] /api/v1/now - Root cause identified
2. [ ] /api/v1/now - Fixed of removed
3. [ ] /api/v1/balance - Status code 501→410
4. [ ] /api/v1/balance - Deprecation message met migration info
5. [ ] Beide fixes getest
6. [ ] Git commit + push

---

## VERIFICATIE

```bash
# Na fixes
curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/v1/now
# Verwacht: 200 of 404 (niet 500)

curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/api/v1/balance
# Verwacht: 410 (niet 501)

curl -s http://localhost:8000/api/v1/balance | jq .
# Verwacht: Deprecation message met migration info
```

---

## GIT COMMIT

```bash
git commit -m "fix: resolve /now 500 error + /balance 501→410 deprecation

- /api/v1/now: [beschrijf fix]
- /api/v1/balance: Changed to 410 Gone with deprecation message

Discovered during: Endpoint routing audit (#40)"
```

---

*Template versie: 1.0*
