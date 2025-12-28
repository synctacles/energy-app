"""The SYNCTACLES integration."""
from __future__ import annotations

import logging
from datetime import timedelta

from homeassistant.config_entries import ConfigEntry
from homeassistant.const import Platform
from homeassistant.core import HomeAssistant
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed

from .const import DOMAIN, CONF_API_URL, SCAN_INTERVAL_GENERATION

_LOGGER = logging.getLogger(__name__)

PLATFORMS: list[Platform] = [Platform.SENSOR]


async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Set up SYNCTACLES from a config entry."""
    api_url = entry.data[CONF_API_URL]
    
    # Create data coordinator
    coordinator = SynctaclesDataCoordinator(hass, api_url)
    
    # Initial fetch
    await coordinator.async_config_entry_first_refresh()
    
    # Store coordinator
    hass.data.setdefault(DOMAIN, {})
    hass.data[DOMAIN][entry.entry_id] = coordinator
    
    # Forward to platforms
    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)
    
    return True


async def async_unload_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Unload a config entry."""
    if unload_ok := await hass.config_entries.async_unload_platforms(entry, PLATFORMS):
        hass.data[DOMAIN].pop(entry.entry_id)
    
    return unload_ok


class SynctaclesDataCoordinator(DataUpdateCoordinator):
    """Class to manage fetching SYNCTACLES data from API."""

    def __init__(self, hass: HomeAssistant, api_url: str) -> None:
        """Initialize."""
        self.api_url = api_url
        self.session = async_get_clientsession(hass)
        
        super().__init__(
            hass,
            _LOGGER,
            name=DOMAIN,
            update_interval=timedelta(seconds=SCAN_INTERVAL_GENERATION),
        )

    async def _async_update_data(self):
        """Fetch data from API."""
        try:
            data = {}
            
            # Fetch all 3 endpoints
            endpoints = {
                "generation": "/api/v1/generation-mix",
                "load": "/api/v1/load",
                "balance": "/api/v1/balance",
            }
            
            for key, path in endpoints.items():
                url = f"{self.api_url}{path}"
                async with self.session.get(url, timeout=10) as response:
                    if response.status == 200:
                        data[key] = await response.json()
                    else:
                        _LOGGER.warning(f"API error {response.status} for {key}")
                        data[key] = None
            
            return data
        
        except Exception as err:
            raise UpdateFailed(f"Error communicating with API: {err}")
