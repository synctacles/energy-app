"""Sensor platform for SYNCTACLES integration."""
from __future__ import annotations

import logging
from typing import Any

from homeassistant.components.sensor import (
    SensorDeviceClass,
    SensorEntity,
    SensorStateClass,
)
from homeassistant.config_entries import ConfigEntry
from homeassistant.const import UnitOfPower
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from . import SynctaclesDataCoordinator
from .const import DOMAIN

_LOGGER = logging.getLogger(__name__)


async def async_setup_entry(
    hass: HomeAssistant,
    entry: ConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up SYNCTACLES sensors from a config entry."""
    coordinator: SynctaclesDataCoordinator = hass.data[DOMAIN][entry.entry_id]

    # Create 3 sensors
    entities = [
        SynctaclesGenerationSensor(coordinator),
        SynctaclesLoadSensor(coordinator),
        SynctaclesBalanceSensor(coordinator),
    ]

    async_add_entities(entities)


class SynctaclesGenerationSensor(CoordinatorEntity, SensorEntity):
    """Sensor for generation mix total."""

    _attr_name = "SYNCTACLES Generation Total"
    _attr_unique_id = "synctacles_generation_total"
    _attr_device_class = SensorDeviceClass.POWER
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_native_unit_of_measurement = UnitOfPower.MEGA_WATT

    def __init__(self, coordinator: SynctaclesDataCoordinator) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator)
        self._attr_device_info = {
            "identifiers": {(DOMAIN, "synctacles_api")},
            "name": "SYNCTACLES Energy Data",
            "manufacturer": "DATADIO",
        }

    @property
    def native_value(self) -> float | None:
        """Return the state of the sensor."""
        if not self.coordinator.data or "generation" not in self.coordinator.data:
            return None

        gen_data = self.coordinator.data["generation"]
        if not gen_data or "data" not in gen_data or not gen_data["data"]:
            return None

        # Get most recent record
        latest = gen_data["data"][0]
        return latest.get("total_mw")

    @property
    def icon(self) -> str:
        """Return icon based on quality status."""
        if not self.coordinator.data or "generation" not in self.coordinator.data:
            return "mdi:lightning-bolt"
        
        meta = self.coordinator.data["generation"].get("meta", {})
        quality = meta.get("quality_status", "NO_DATA")
        
        if quality == "OK":
            return "mdi:check-circle"
        elif quality == "STALE":
            return "mdi:alert-circle"
        else:
            return "mdi:close-circle"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return additional attributes."""
        if not self.coordinator.data or "generation" not in self.coordinator.data:
            return {}

        gen_data = self.coordinator.data["generation"]
        if not gen_data or "data" not in gen_data or not gen_data["data"]:
            return {}

        latest = gen_data["data"][0]
        meta = gen_data.get("meta", {})

        attrs = {
            "quality_status": meta.get("quality_status", "UNKNOWN"),
            "source": meta.get("source", "UNKNOWN"),
            "data_age_seconds": meta.get("data_age_seconds"),
            "timestamp": latest.get("timestamp"),
        }

        # Add PSR-type breakdown
        psr_types = ["biomass", "gas", "coal", "nuclear", "solar", "waste",
                     "wind_offshore", "wind_onshore", "other"]
        for psr in psr_types:
            key = f"{psr}_mw"
            if key in latest:
                attrs[key] = latest[key]

        return attrs


class SynctaclesLoadSensor(CoordinatorEntity, SensorEntity):
    """Sensor for load (consumption)."""

    _attr_name = "SYNCTACLES Load Actual"
    _attr_unique_id = "synctacles_load_actual"
    _attr_device_class = SensorDeviceClass.POWER
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_native_unit_of_measurement = UnitOfPower.MEGA_WATT

    def __init__(self, coordinator: SynctaclesDataCoordinator) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator)
        self._attr_device_info = {
            "identifiers": {(DOMAIN, "synctacles_api")},
            "name": "SYNCTACLES Energy Data",
            "manufacturer": "DATADIO",
        }

    @property
    def native_value(self) -> float | None:
        """Return the state of the sensor."""
        if not self.coordinator.data or "load" not in self.coordinator.data:
            return None

        load_data = self.coordinator.data["load"]
        if not load_data or "data" not in load_data or not load_data["data"]:
            return None

        latest = load_data["data"][0]
        return latest.get("actual_mw")

    @property
    def icon(self) -> str:
        """Return icon based on quality status."""
        if not self.coordinator.data or "load" not in self.coordinator.data:
            return "mdi:transmission-tower"
        
        meta = self.coordinator.data["load"].get("meta", {})
        quality = meta.get("quality_status", "NO_DATA")
        
        if quality == "OK":
            return "mdi:check-circle"
        elif quality == "STALE":
            return "mdi:alert-circle"
        else:
            return "mdi:close-circle"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return additional attributes."""
        if not self.coordinator.data or "load" not in self.coordinator.data:
            return {}

        load_data = self.coordinator.data["load"]
        if not load_data or "data" not in load_data or not load_data["data"]:
            return {}

        latest = load_data["data"][0]
        meta = load_data.get("meta", {})

        return {
            "quality_status": meta.get("quality_status", "UNKNOWN"),
            "source": meta.get("source", "UNKNOWN"),
            "data_age_seconds": meta.get("data_age_seconds"),
            "timestamp": latest.get("timestamp"),
            "forecast_mw": latest.get("forecast_mw"),
        }


class SynctaclesBalanceSensor(CoordinatorEntity, SensorEntity):
    """Sensor for balance delta."""

    _attr_name = "SYNCTACLES Balance Delta"
    _attr_unique_id = "synctacles_balance_delta"
    _attr_device_class = SensorDeviceClass.POWER
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_native_unit_of_measurement = UnitOfPower.MEGA_WATT

    def __init__(self, coordinator: SynctaclesDataCoordinator) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator)
        self._attr_device_info = {
            "identifiers": {(DOMAIN, "synctacles_api")},
            "name": "SYNCTACLES Energy Data",
            "manufacturer": "DATADIO",
        }

    @property
    def native_value(self) -> float | None:
        """Return the state of the sensor."""
        if not self.coordinator.data or "balance" not in self.coordinator.data:
            return None

        balance_data = self.coordinator.data["balance"]
        if not balance_data or "data" not in balance_data or not balance_data["data"]:
            return None

        latest = balance_data["data"][0]
        return latest.get("delta_mw")

    @property
    def icon(self) -> str:
        """Return icon based on quality status."""
        if not self.coordinator.data or "balance" not in self.coordinator.data:
            return "mdi:scale-balance"
        
        meta = self.coordinator.data["balance"].get("meta", {})
        quality = meta.get("quality_status", "NO_DATA")
        
        if quality == "OK":
            return "mdi:check-circle"
        elif quality == "STALE":
            return "mdi:alert-circle"
        else:
            return "mdi:close-circle"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return additional attributes."""
        if not self.coordinator.data or "balance" not in self.coordinator.data:
            return {}

        balance_data = self.coordinator.data["balance"]
        if not balance_data or "data" not in balance_data or not balance_data["data"]:
            return {}

        latest = balance_data["data"][0]
        meta = balance_data.get("meta", {})

        return {
            "quality_status": meta.get("quality_status", "UNKNOWN"),
            "source": meta.get("source", "UNKNOWN"),
            "data_age_seconds": meta.get("data_age_seconds"),
            "timestamp": latest.get("timestamp"),
            "price_eur_mwh": latest.get("price_eur_mwh"),
        }
