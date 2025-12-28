"""Config flow for SYNCTACLES integration."""
from __future__ import annotations

import logging
from typing import Any

import voluptuous as vol
from homeassistant import config_entries
from homeassistant.core import HomeAssistant
from homeassistant.data_entry_flow import FlowResult
import homeassistant.helpers.config_validation as cv

from .const import DOMAIN, CONF_API_URL, DEFAULT_API_URL, HA_COMPONENT_NAME

_LOGGER = logging.getLogger(__name__)

STEP_USER_DATA_SCHEMA = vol.Schema(
    {
        vol.Required(CONF_API_URL, default=DEFAULT_API_URL): cv.string,
    }
)


class SynctaclesConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Handle a config flow for SYNCTACLES."""

    VERSION = 1

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> FlowResult:
        """Handle the initial step."""
        errors: dict[str, str] = {}

        if user_input is not None:
            api_url = user_input[CONF_API_URL].rstrip("/")
            
            # Basic validation
            if not api_url.startswith(("http://", "https://")):
                errors["base"] = "invalid_url"
            else:
                # Create entry
                await self.async_set_unique_id(api_url)
                self._abort_if_unique_id_configured()
                
                return self.async_create_entry(
                    title=f"{HA_COMPONENT_NAME} Energy Data",
                    data={CONF_API_URL: api_url},
                )

        return self.async_show_form(
            step_id="user",
            data_schema=STEP_USER_DATA_SCHEMA,
            errors=errors,
        )
