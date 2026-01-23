"""HTTP clients for external services."""
from synctacles_db.clients.easyenergy_client import EasyEnergyClient
from synctacles_db.clients.frank_energie_client import FrankEnergieClient

# Legacy import for backward compatibility (deprecated)
try:
    from synctacles_db.clients.consumer_price_client import ConsumerPriceClient
except ImportError:
    ConsumerPriceClient = None

__all__ = ["FrankEnergieClient", "EasyEnergyClient", "ConsumerPriceClient"]
