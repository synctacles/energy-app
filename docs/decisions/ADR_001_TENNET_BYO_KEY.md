# ADR-001: TenneT Bring-Your-Own-Key Model

**Date:** 2026-01-07
**Status:** Accepted
**Deciders:** Leo, CAI
**Related:** SKILL_02_ARCHITECTURE.md, SKILL_04_PRODUCT_REQUIREMENTS.md

---

## Context

TenneT provides real-time energy balance data for the Netherlands via their API. Access requires an API key that users must request from TenneT. The question arose: should SYNCTACLES/Energy Insights NL provide TenneT data through our centralized API (requiring us to manage TenneT keys), or should users bring their own keys?

Key considerations:
- TenneT API keys are free but require manual registration
- TenneT has rate limits per key
- Users may want real-time balance data for their personal use
- Our API serves multiple users who might all request TenneT data simultaneously

---

## Decision

**TenneT data is ONLY available via Bring-Your-Own-Key (BYO-key) in the Home Assistant component.**

The SYNCTACLES API does NOT provide TenneT data as a centralized service. Users who want TenneT balance sensors must:
1. Register for their own TenneT API key
2. Configure the key in the Home Assistant integration
3. The HA component fetches data directly from TenneT using the user's key

---

## Rationale

### Why BYO-key?

1. **Rate Limit Management**: Each user's TenneT key has its own rate limits. If we used a shared key, one user's excessive polling could affect all other users.

2. **Legal Compliance**: TenneT's terms of service require each user to register individually. Using a shared key for commercial resale would violate their TOS.

3. **Scalability**: As user base grows, a single centralized key would become a bottleneck and single point of failure.

4. **User Control**: Users control their own polling frequency and data freshness without affecting others.

5. **Zero Server Load**: TenneT requests go directly from user's HA instance to TenneT, not through our servers.

### Why HA-only (not in API)?

The SYNCTACLES API provides historical and aggregated data from ENTSO-E and Energy-Charts. TenneT balance data is:
- Real-time only (no historical database storage needed)
- Personal monitoring (not aggregated insights)
- High-frequency polling (every 30-60 seconds)
- User-specific rate limits

This makes it a perfect fit for client-side implementation in HA, not server-side API.

---

## Alternatives Considered

### Alternative 1: Centralized TenneT Key (Shared)
- **Pros:**
  - Users don't need to register with TenneT
  - Simpler user onboarding
- **Cons:**
  - Rate limit sharing causes contention
  - Violates TenneT TOS
  - Single point of failure
  - Server load for real-time polling
- **Why rejected:** Legal and scalability issues

### Alternative 2: Multi-Key Pool Management
- **Pros:**
  - Could distribute load across multiple keys
  - Users don't need to register
- **Cons:**
  - Complex key rotation logic
  - Still violates TenneT TOS
  - Requires key acquisition and management
  - High maintenance burden
- **Why rejected:** Over-engineered, still illegal

### Alternative 3: No TenneT Support
- **Pros:**
  - Simplest implementation
  - No legal concerns
- **Cons:**
  - Users lose access to real-time balance data
  - Reduced product value for Dutch market
- **Why rejected:** TenneT data is valuable for users

---

## Consequences

### Positive
- Clear legal compliance with TenneT TOS
- Zero server load for TenneT data
- Each user has dedicated rate limits
- Scales infinitely (no central bottleneck)
- Users control their own data freshness

### Negative
- Extra setup step for users who want TenneT data
- User must register with TenneT (free but manual process)
- Support burden: helping users get TenneT keys

### Neutral
- TenneT sensors are optional, not core functionality
- Users who don't want real-time balance data can skip this entirely

---

## Implementation Notes

### Home Assistant Component
- TenneT API key is optional configuration parameter
- If no key provided: TenneT balance sensors are disabled
- If key provided: HA component fetches directly from TenneT API
- Polling frequency: 60 seconds (configurable)
- Error handling: graceful degradation if key is invalid/rate limited

### API Server
- TenneT data is NOT stored in database
- TenneT endpoints are NOT exposed in API
- SKILL_02 documents: "TenneT (BYO-key via HA only, not server)"

### User Documentation
```markdown
## Optional: TenneT Real-Time Balance Data

To enable real-time energy balance sensors:
1. Register for free TenneT API key: https://www.tennet.org/...
2. Add key to HA integration configuration
3. Balance sensors will appear automatically
```

---

## Validation

This decision is correct if:
- ✅ No rate limit complaints from TenneT about shared key abuse
- ✅ Users successfully configure their own keys
- ✅ TenneT sensor adoption rate is healthy (>20% of users)
- ✅ No TenneT TOS violation complaints
- ✅ Server load remains stable (no TenneT polling overhead)

---

## References

- SKILL_02_ARCHITECTURE.md: Documents BYO-key model
- TenneT API Documentation: https://www.tennet.org/
- SKILL_04_PRODUCT_REQUIREMENTS.md: Optional vs required sensors
- Home Assistant Integration: `custom_components/ha_energy_insights_nl/tennet_client.py`
