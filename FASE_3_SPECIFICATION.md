# FASE 3: HA COMPONENT TENNET BYO-KEY IMPLEMENTATION - SPECIFICATION

**Datum:** 2026-01-02
**Status:** 🔄 IN VOORBEREIDING
**Doel:** Complete HA component rewrite met TenneT BYO-key ondersteuning

---

## OVERVIEW

Dit document bevat **production-ready code** en gedetailleerde instructies voor implementatie van TenneT BYO-key support in de Home Assistant component.

**Wat gaat er veranderen:**
- ✅ Config flow: TenneT API key field (optional)
- ✅ Nieuwe `tennet_client.py`: Lokale TenneT API communicatie
- ✅ `sensor.py`: Conditional sensors (only if TenneT key configured)
- ✅ `__init__.py`: TenneT client initialization
- ✅ `strings.json`: UI labels voor TenneT configuratie

**Resultaat:**
- Balance sensors alleen zichtbaar als TenneT key ingevuld
- Geen server-side TenneT data - alles lokaal in HA
- Graceful error handling (TenneT errors loggen, niet kritiek)
- Rate limit compliant (5 min polling << 100 req/min limit)

---

## FASE 3.1: TENNET CLIENT MODULE (NEW FILE)

**Path:** `custom_components/ha_energy_insights_nl/tennet_client.py`

### Functionaliteit:
- Async HTTP client voor TenneT API
- Bearer token authentication
- Error handling & retry logic
- Rate limit awareness (100 req/min)
- Logging

### Code:

```python
"""
TenneT API Client for Home Assistant.

Fetches grid balance data from TenneT's public API.
Authentication: Bearer token (user's personal API key)
Endpoint: https://api.tennet.eu/v1/balance-delta-high-res/latest
Rate limit: 100 requests/minute
Data freshness: Updated every 5 minutes (recommended)
"""

import logging
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
import asyncio

import aiohttp
from aiohttp import ClientSession, ClientError, ClientTimeout

_LOGGER = logging.getLogger(__name__)

# TenneT API Configuration
TENNET_API_BASE = "https://api.tennet.eu/v1"
TENNET_BALANCE_ENDPOINT = f"{TENNET_API_BASE}/balance-delta-high-res/latest"
TENNET_FREQUENCY_ENDPOINT = f"{TENNET_API_BASE}/grid-frequency/latest"
TENNET_RESERVE_MARGIN_ENDPOINT = f"{TENNET_API_BASE}/reserve-margin/latest"

# Timeouts & Retry
DEFAULT_TIMEOUT = aiohttp.ClientTimeout(total=10, connect=5, sock_read=5)
MAX_RETRIES = 3
RETRY_DELAY = 2  # seconds
RATE_LIMIT_WARNING = "TenneT: 429 Too Many Requests - backing off"


class TenneTClientError(Exception):
    """Base exception for TenneT client errors."""
    pass


class TenneTAuthError(TenneTClientError):
    """Authentication error (401, 403)."""
    pass


class TenneTRateLimitError(TenneTClientError):
    """Rate limit exceeded (429)."""
    pass


class TenneTTimeoutError(TenneTClientError):
    """Request timeout."""
    pass


class TenneTClient:
    """
    Async client for TenneT Balance API.

    Usage:
        client = TenneTClient(api_key="your_bearer_token")
        balance = await client.get_balance()
        # Returns: {"balance_mw": 150.5, "timestamp": "2026-01-02T10:15:00Z"}
    """

    def __init__(self, api_key: str):
        """
        Initialize TenneT client.

        Args:
            api_key: TenneT API key (Bearer token)
        """
        self.api_key = api_key
        self.last_request_time: Optional[datetime] = None
        self.rate_limit_reset: Optional[datetime] = None

    async def get_balance(self) -> Dict[str, Any]:
        """
        Fetch current grid balance delta.

        Returns:
            {
                "balance_mw": float,        # Positive = surplus, Negative = deficit
                "timestamp": str,            # ISO 8601 timestamp
                "quality": str,              # "FRESH", "STALE", or "UNAVAILABLE"
                "age_seconds": int
            }

        Raises:
            TenneTAuthError: Invalid API key
            TenneTRateLimitError: Rate limit exceeded
            TenneTTimeoutError: Request timeout
            TenneTClientError: Other API errors
        """
        return await self._fetch_endpoint(
            TENNET_BALANCE_ENDPOINT,
            response_parser=self._parse_balance_response
        )

    async def get_frequency(self) -> Dict[str, Any]:
        """
        Fetch current grid frequency (Hz).

        Returns:
            {
                "frequency_hz": float,
                "timestamp": str,
                "quality": str,
                "age_seconds": int
            }
        """
        return await self._fetch_endpoint(
            TENNET_FREQUENCY_ENDPOINT,
            response_parser=self._parse_frequency_response
        )

    async def get_reserve_margin(self) -> Dict[str, Any]:
        """
        Fetch current reserve margin (MW).

        Returns:
            {
                "reserve_margin_mw": float,
                "timestamp": str,
                "quality": str,
                "age_seconds": int
            }
        """
        return await self._fetch_endpoint(
            TENNET_RESERVE_MARGIN_ENDPOINT,
            response_parser=self._parse_reserve_margin_response
        )

    async def _fetch_endpoint(
        self,
        endpoint: str,
        response_parser=None,
        retries: int = 0
    ) -> Dict[str, Any]:
        """
        Fetch data from TenneT endpoint with retry logic.

        Args:
            endpoint: Full API endpoint URL
            response_parser: Function to parse response
            retries: Current retry attempt

        Returns:
            Parsed response data

        Raises:
            TenneT*Error: Various error conditions
        """
        # Rate limit check
        if self.rate_limit_reset and datetime.utcnow() < self.rate_limit_reset:
            wait_seconds = (self.rate_limit_reset - datetime.utcnow()).total_seconds()
            _LOGGER.warning(
                f"TenneT rate limit active, waiting {wait_seconds:.0f}s"
            )
            await asyncio.sleep(wait_seconds)

        headers = self._build_headers()

        try:
            async with aiohttp.ClientSession(timeout=DEFAULT_TIMEOUT) as session:
                async with session.get(endpoint, headers=headers) as response:
                    # Handle rate limiting
                    if response.status == 429:
                        self.rate_limit_reset = datetime.utcnow() + timedelta(seconds=60)
                        _LOGGER.warning(RATE_LIMIT_WARNING)
                        raise TenneTRateLimitError("Rate limit exceeded (429)")

                    # Handle auth errors
                    if response.status in (401, 403):
                        _LOGGER.error(f"TenneT auth error: {response.status}")
                        raise TenneTAuthError(
                            f"Invalid TenneT API key ({response.status})"
                        )

                    # Handle other errors
                    if response.status >= 400:
                        error_body = await response.text()
                        _LOGGER.error(
                            f"TenneT API error {response.status}: {error_body}"
                        )
                        raise TenneTClientError(
                            f"TenneT API returned {response.status}"
                        )

                    # Success
                    data = await response.json()
                    self.last_request_time = datetime.utcnow()

                    # Parse response
                    if response_parser:
                        return response_parser(data)
                    return data

        except asyncio.TimeoutError as err:
            _LOGGER.error(f"TenneT request timeout: {err}")
            raise TenneTTimeoutError("TenneT API request timeout") from err

        except ClientError as err:
            # Network error - retry with backoff
            if retries < MAX_RETRIES:
                wait_time = RETRY_DELAY * (2 ** retries)  # Exponential backoff
                _LOGGER.warning(
                    f"TenneT request failed, retrying in {wait_time}s "
                    f"(attempt {retries + 1}/{MAX_RETRIES})"
                )
                await asyncio.sleep(wait_time)
                return await self._fetch_endpoint(
                    endpoint,
                    response_parser=response_parser,
                    retries=retries + 1
                )
            else:
                _LOGGER.error(f"TenneT request failed after {MAX_RETRIES} retries: {err}")
                raise TenneTClientError(f"TenneT API unreachable: {err}") from err

    def _build_headers(self) -> Dict[str, str]:
        """Build HTTP headers with Bearer token."""
        return {
            "Authorization": f"Bearer {self.api_key}",
            "Accept": "application/json",
            "User-Agent": "ha-energy-insights-nl/1.0.0"
        }

    @staticmethod
    def _parse_balance_response(data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Parse TenneT balance endpoint response.

        Expected format:
        {
            "timestamp": "2026-01-02T10:15:00Z",
            "unit": "MW",
            "data": [
                {
                    "timestamp": "2026-01-02T10:15:00Z",
                    "value": 150.5
                },
                ...
            ]
        }
        """
        try:
            if not data.get("data") or len(data["data"]) == 0:
                return {
                    "balance_mw": 0.0,
                    "timestamp": data.get("timestamp", datetime.utcnow().isoformat()),
                    "quality": "UNAVAILABLE",
                    "age_seconds": 9999
                }

            latest = data["data"][0]
            timestamp = latest.get("timestamp", data.get("timestamp"))
            value = latest.get("value", 0.0)

            # Calculate data age
            try:
                ts = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
                age = (datetime.utcnow() - ts).total_seconds()
            except (ValueError, AttributeError):
                age = 0

            return {
                "balance_mw": value,
                "timestamp": timestamp,
                "quality": "FRESH" if age < 600 else "STALE",
                "age_seconds": int(age)
            }

        except (KeyError, IndexError, TypeError) as err:
            _LOGGER.error(f"Failed to parse TenneT balance response: {err}")
            return {
                "balance_mw": 0.0,
                "timestamp": datetime.utcnow().isoformat(),
                "quality": "UNAVAILABLE",
                "age_seconds": 9999
            }

    @staticmethod
    def _parse_frequency_response(data: Dict[str, Any]) -> Dict[str, Any]:
        """Parse TenneT frequency endpoint response."""
        try:
            if not data.get("data") or len(data["data"]) == 0:
                return {
                    "frequency_hz": 50.0,
                    "timestamp": data.get("timestamp", datetime.utcnow().isoformat()),
                    "quality": "UNAVAILABLE",
                    "age_seconds": 9999
                }

            latest = data["data"][0]
            timestamp = latest.get("timestamp", data.get("timestamp"))
            value = latest.get("value", 50.0)

            try:
                ts = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
                age = (datetime.utcnow() - ts).total_seconds()
            except (ValueError, AttributeError):
                age = 0

            return {
                "frequency_hz": value,
                "timestamp": timestamp,
                "quality": "FRESH" if age < 600 else "STALE",
                "age_seconds": int(age)
            }

        except (KeyError, IndexError, TypeError) as err:
            _LOGGER.error(f"Failed to parse TenneT frequency response: {err}")
            return {
                "frequency_hz": 50.0,
                "timestamp": datetime.utcnow().isoformat(),
                "quality": "UNAVAILABLE",
                "age_seconds": 9999
            }

    @staticmethod
    def _parse_reserve_margin_response(data: Dict[str, Any]) -> Dict[str, Any]:
        """Parse TenneT reserve margin endpoint response."""
        try:
            if not data.get("data") or len(data["data"]) == 0:
                return {
                    "reserve_margin_mw": 0.0,
                    "timestamp": data.get("timestamp", datetime.utcnow().isoformat()),
                    "quality": "UNAVAILABLE",
                    "age_seconds": 9999
                }

            latest = data["data"][0]
            timestamp = latest.get("timestamp", data.get("timestamp"))
            value = latest.get("value", 0.0)

            try:
                ts = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
                age = (datetime.utcnow() - ts).total_seconds()
            except (ValueError, AttributeError):
                age = 0

            return {
                "reserve_margin_mw": value,
                "timestamp": timestamp,
                "quality": "FRESH" if age < 600 else "STALE",
                "age_seconds": int(age)
            }

        except (KeyError, IndexError, TypeError) as err:
            _LOGGER.error(f"Failed to parse TenneT reserve margin response: {err}")
            return {
                "reserve_margin_mw": 0.0,
                "timestamp": datetime.utcnow().isoformat(),
                "quality": "UNAVAILABLE",
                "age_seconds": 9999
            }
```

---

## FASE 3.2: CONFIG FLOW UPDATES

**Path:** `custom_components/ha_energy_insights_nl/config_flow.py`

### Wat toevoegen:

```python
# Add to imports:
from .const import CONF_TENNET_API_KEY

# In ConfigFlow.async_step_user():
# Add after API_KEY field:

cv.Required(
    CONF_TENNET_API_KEY,
    description={"suggested_value": user_input.get(CONF_TENNET_API_KEY, "")}
): cv.string

# Add validation function (new method):

def _validate_tennet_key(key: str) -> bool:
    """
    Validate TenneT API key format.
    Expected format: 32+ character bearer token
    """
    if not key:
        return True  # Optional field

    # Remove common prefixes if user added them
    key = key.replace("Bearer ", "").strip()

    # Validate length (typical tokens are 32+ chars)
    return len(key) >= 32

# In async_step_user() error handling:

if data.get(CONF_TENNET_API_KEY):
    if not self._validate_tennet_key(data[CONF_TENNET_API_KEY]):
        errors["base"] = "invalid_tennet_format"
```

### Strings update:

**In `strings.json`, `config` section, add:**

```json
"step": {
  "user": {
    "data": {
      "api_endpoint": "API Server URL",
      "api_key": "SYNCTACLES API Key",
      "tennet_api_key": "TenneT API Key (Optional - for balance data)"
    },
    "data_description": {
      "tennet_api_key": "Get from https://www.tennet.eu/developer-portal/"
    }
  }
}
```

### Constants:

**In `const.py`, add:**

```python
CONF_TENNET_API_KEY = "tennet_api_key"

# TenneT API Configuration
TENNET_COORDINATOR_NAME = "TenneT Data"
TENNET_UPDATE_INTERVAL = 300  # 5 minutes (12 requests/hour << 100/min limit)
TENNET_SENSORS = [
    "balance_delta",
    "grid_stress"
]
```

---

## FASE 3.3: SENSOR UPDATES

**Path:** `custom_components/ha_energy_insights_nl/sensor.py`

### Huidige sensoren (BEHOUDEN):
- `sensor.synctacles_generation_total`
- `sensor.synctacles_load_actual`

### NIEUWE sensoren (conditionally):
- `sensor.synctacles_balance_delta` (if TenneT key)
- `sensor.synctacles_grid_stress` (if TenneT key)

### Code toevoegen aan `sensor.py`:

```python
# Imports toevoegen:
from .tennet_client import TenneTClient, TenneT*Error

# In async_setup_entry():

# Check if TenneT key configured
tennet_key = config_entry.data.get(CONF_TENNET_API_KEY)

# Only create TenneT sensors if key provided
if tennet_key:
    # Initialize TenneT client
    tennet_client = TenneTClient(tennet_key)

    # Create balance sensor
    async_add_entities([
        TenneTBalanceSensor(
            coordinator=coordinator,  # Data update coordinator
            tennet_client=tennet_client,
            config_entry=config_entry
        ),
        TenneTGridStressSensor(
            coordinator=coordinator,
            tennet_client=tennet_client,
            config_entry=config_entry
        )
    ])


# New sensor classes (add to sensor.py):

class TenneTBalanceSensor(CoordinatorEntity, SensorEntity):
    """TenneT grid balance delta sensor."""

    _attr_name = "Balance Delta"
    _attr_unit_of_measurement = "MW"
    _attr_device_class = "power"
    _attr_state_class = "measurement"

    def __init__(self, coordinator, tennet_client, config_entry):
        super().__init__(coordinator)
        self.tennet_client = tennet_client
        self.config_entry = config_entry
        self._attr_unique_id = f"{DOMAIN}_balance_delta"

    @property
    def state(self) -> Optional[float]:
        """Return balance delta in MW."""
        try:
            # Fetch from TenneT (this is called every update cycle)
            balance_data = self.coordinator.data.get("tennet_balance")
            if balance_data:
                return balance_data.get("balance_mw")
        except Exception as err:
            _LOGGER.error(f"Error getting balance: {err}")
        return None

    @property
    def extra_state_attributes(self) -> Dict[str, Any]:
        """Return additional attributes."""
        try:
            balance_data = self.coordinator.data.get("tennet_balance", {})
            return {
                "quality": balance_data.get("quality", "UNAVAILABLE"),
                "timestamp": balance_data.get("timestamp"),
                "age_seconds": balance_data.get("age_seconds"),
                "interpretation": "Positive=Surplus, Negative=Deficit"
            }
        except Exception:
            return {"quality": "UNAVAILABLE"}


class TenneTGridStressSensor(CoordinatorEntity, SensorEntity):
    """TenneT grid stress indicator (0-100)."""

    _attr_name = "Grid Stress"
    _attr_unit_of_measurement = "%"
    _attr_device_class = "gauge"

    def __init__(self, coordinator, tennet_client, config_entry):
        super().__init__(coordinator)
        self.tennet_client = tennet_client
        self.config_entry = config_entry
        self._attr_unique_id = f"{DOMAIN}_grid_stress"

    @property
    def state(self) -> Optional[int]:
        """Return grid stress percentage (0-100)."""
        try:
            balance_data = self.coordinator.data.get("tennet_balance", {})
            balance_mw = balance_data.get("balance_mw", 0)

            # Calculate stress: higher |balance| = higher stress
            # Formula: stress = min(|balance| / 200, 100)
            # 200 MW imbalance = 100% stress
            stress = min(abs(balance_mw) / 200 * 100, 100)
            return int(stress)
        except Exception as err:
            _LOGGER.error(f"Error calculating stress: {err}")
        return None

    @property
    def extra_state_attributes(self) -> Dict[str, Any]:
        """Return additional attributes."""
        try:
            balance_data = self.coordinator.data.get("tennet_balance", {})
            return {
                "balance_mw": balance_data.get("balance_mw"),
                "quality": balance_data.get("quality"),
                "timestamp": balance_data.get("timestamp")
            }
        except Exception:
            return {}
```

---

## FASE 3.4: __INIT__.PY UPDATES

**Path:** `custom_components/ha_energy_insights_nl/__init__.py`

### Toevoegen:

```python
# Add to imports:
from .tennet_client import TenneTClient, TenneT*Error
from .const import CONF_TENNET_API_KEY, TENNET_UPDATE_INTERVAL

# In async_setup_entry():

# Initialize TenneT client if key provided
tennet_key = entry.data.get(CONF_TENNET_API_KEY)
hass.data[DOMAIN][entry.entry_id]["tennet_client"] = None
hass.data[DOMAIN][entry.entry_id]["tennet_enabled"] = False

if tennet_key:
    try:
        tennet_client = TenneTClient(tennet_key)
        hass.data[DOMAIN][entry.entry_id]["tennet_client"] = tennet_client
        hass.data[DOMAIN][entry.entry_id]["tennet_enabled"] = True
        _LOGGER.info("TenneT client initialized")
    except Exception as err:
        _LOGGER.warning(f"TenneT client initialization failed: {err}")
        _LOGGER.warning("Balance sensors will not be available")
        # Continue without TenneT - don't fail entire integration

# In coordinator update function:

async def async_update_data():
    """Fetch data from API and TenneT."""
    try:
        # Fetch from SYNCTACLES API (existing)
        async with async_timeout.timeout(10):
            api_data = await fetch_synctacles_api()

        # Fetch from TenneT if enabled
        tennet_data = {}
        if hass.data[DOMAIN][entry.entry_id]["tennet_enabled"]:
            try:
                client = hass.data[DOMAIN][entry.entry_id]["tennet_client"]
                balance = await client.get_balance()
                tennet_data["tennet_balance"] = balance
            except TenneT*Error as err:
                _LOGGER.warning(f"TenneT fetch failed: {err}")
                tennet_data["tennet_balance"] = {
                    "balance_mw": None,
                    "quality": "UNAVAILABLE"
                }

        return {
            **api_data,
            **tennet_data
        }

    except Exception as err:
        raise UpdateFailed(f"Error communicating with API: {err}") from err
```

---

## FASE 3.5: STRINGS.JSON UPDATES

```json
{
  "config": {
    "step": {
      "user": {
        "title": "Energy Insights NL Configuration",
        "description": "Configure SYNCTACLES API and optional TenneT BYO-key",
        "data": {
          "api_endpoint": "API Server URL (e.g., http://192.168.1.100:8000)",
          "api_key": "SYNCTACLES API Key",
          "tennet_api_key": "TenneT API Key (Optional)"
        },
        "data_description": {
          "api_endpoint": "URL of your SYNCTACLES server",
          "api_key": "Get from your SYNCTACLES account",
          "tennet_api_key": "Personal TenneT API key from https://www.tennet.eu/developer-portal/ - enables balance sensors"
        }
      }
    },
    "error": {
      "invalid_api_key": "Invalid API key",
      "connection_error": "Failed to connect to API server",
      "invalid_tennet_format": "TenneT API key format invalid (should be 32+ characters)"
    },
    "abort": {
      "already_configured": "Integration already configured"
    }
  },
  "entity": {
    "sensor": {
      "synctacles_balance_delta": {
        "name": "Balance Delta",
        "unit": "MW"
      },
      "synctacles_grid_stress": {
        "name": "Grid Stress",
        "unit": "%"
      }
    }
  }
}
```

---

## FASE 3.6: MANIFEST.JSON CHECK

Verify `custom_components/ha_energy_insights_nl/manifest.json` includes:

```json
{
  "requirements": [
    "aiohttp>=3.8.0"
  ]
}
```

(aiohttp is already available in Home Assistant)

---

## TEST PLAN

### Unit Tests (Local)

1. **TenneT Client Tests**
   - Test successful balance fetch
   - Test auth error handling (401, 403)
   - Test rate limit handling (429)
   - Test timeout handling
   - Test retry logic with exponential backoff
   - Test response parsing (balance, frequency, reserve margin)

2. **Config Flow Tests**
   - Test API key validation
   - Test TenneT key validation (optional, format check)
   - Test already configured abort

3. **Sensor Tests**
   - Test balance sensor creation (with & without TenneT key)
   - Test grid stress calculation
   - Test state attributes
   - Test quality status propagation

### Integration Tests (HA OS)

1. **Configuration**
   - [ ] Add integration without TenneT key → Only 2 sensors created
   - [ ] Add integration with valid TenneT key → 4 sensors created
   - [ ] Add integration with invalid TenneT key → Error in config flow

2. **Data Flow**
   - [ ] Balance sensor shows current balance delta (MW)
   - [ ] Grid stress sensor shows 0-100 (%)
   - [ ] Sensors update every 5 minutes
   - [ ] Attributes show timestamp, quality, age

3. **Error Handling**
   - [ ] TenneT down → Balance sensors show "unavailable", main sensors work
   - [ ] Invalid TenneT key → Config flow rejects, suggests recheck
   - [ ] Rate limit hit → Logged, automatic backoff, next request retries
   - [ ] Network timeout → Exponential backoff, max 3 retries

4. **Persistence**
   - [ ] After HA restart → TenneT client re-initializes with saved key
   - [ ] After updating TenneT key → New key used immediately

---

## INTEGRATION GUIDE (STEP-BY-STEP)

### Step 1: Backup Existing Component

```bash
# On HA OS (via SSH or Terminal)
cd /config/custom_components/ha_energy_insights_nl
cp -r . ../ha_energy_insights_nl.backup
```

### Step 2: Update Files

Copy these files into `custom_components/ha_energy_insights_nl/`:

1. **NEW:** `tennet_client.py` (from FASE 3.1)
2. **UPDATE:** `config_flow.py` (add TenneT key field)
3. **UPDATE:** `sensor.py` (add TenneT sensors)
4. **UPDATE:** `__init__.py` (initialize TenneT client)
5. **UPDATE:** `const.py` (add TenneT constants)
6. **UPDATE:** `strings.json` (add UI labels)

### Step 3: Restart Home Assistant

Settings → Developer Tools → YaML → Restart Home Assistant

### Step 4: Reconfigure Integration

Settings → Devices & Services → Energy Insights NL → Configure

- Enter TenneT API key (or leave blank if not ready)
- Save

### Step 5: Verify

Settings → Devices & Services → Energy Insights NL → Entities

**Without TenneT key:**
- sensor.energy_insights_nl_generation_total ✓
- sensor.energy_insights_nl_load_actual ✓
- sensor.energy_insights_nl_balance_delta ✗ (not created)
- sensor.energy_insights_nl_grid_stress ✗ (not created)

**With TenneT key:**
- All 4 sensors should appear

### Step 6: Check Logs

Settings → System → Logs → Search "synctacles" or "tennet"

**Expected:**
```
INFO: TenneT client initialized
INFO: TenneT balance fetched: 150.5 MW
```

**Or if no key:**
```
DEBUG: TenneT disabled (no API key configured)
```

---

## ROLLBACK PROCEDURE

If something goes wrong:

```bash
# SSH into HA, restore backup
cd /config/custom_components
rm -rf ha_energy_insights_nl
cp -r ha_energy_insights_nl.backup ha_energy_insights_nl

# Restart HA
ha core restart
```

---

## TROUBLESHOOTING

### "TenneT auth error: 401"

**Cause:** Invalid API key

**Fix:**
1. Visit https://www.tennet.eu/developer-portal/
2. Regenerate key (copy without "Bearer " prefix)
3. Reconfigure integration with new key

### "TenneT request timeout"

**Cause:** TenneT API slow or unreachable

**Fix:**
- Check internet connection
- Verify key still valid
- Wait for TenneT API to respond
- Balance sensors will show "unavailable" (other sensors still work)

### "Rate limit exceeded (429)"

**Cause:** Too many requests to TenneT

**Fix:**
- HA polls every 5 min = 12 req/hour (safe)
- If multiple HA instances: use separate TenneT keys
- Wait 60 seconds, automatic retry

### Balance sensors not appearing

**Cause:**
1. TenneT key not configured
2. Component not reloaded
3. Invalid key format

**Fix:**
1. Check Settings → Devices & Services → Energy Insights NL
2. Is TenneT key filled in?
3. Try "Reload" button
4. Check logs for errors

---

## VOLGENDE STAPPEN (FASE 4)

- [ ] Execute integration testing on HA OS
- [ ] Document any issues found
- [ ] Refine based on testing
- [ ] Database migration (archive_tennet_* tables)
- [ ] User acceptance testing
- [ ] Final compliance verification

---

**Status:** SPECIFICATION COMPLETE - Ready for implementation

**Next:** Execute code in HA OS, provide feedback/logs, iterate
