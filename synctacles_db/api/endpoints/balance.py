"""
Balance Endpoint - DEPRECATED (TenneT BYO-Key Only)

Grid balance data is no longer available via the SYNCTACLES API due to
TenneT API license restrictions. Access is now available only via BYO-key
(Bring Your Own) in the Home Assistant component.

See: https://github.com/DATADIO/ha-energy-insights-nl#tennet-byo-key
"""
from fastapi import APIRouter
from starlette.responses import JSONResponse

router = APIRouter()

@router.get("/balance")
async def get_balance():
    """
    Grid balance data - Available via BYO-key in Home Assistant component.

    TenneT API license prohibits server-side redistribution.
    Configure your TenneT API key in Home Assistant for real-time balance data.
    """
    return JSONResponse(
        status_code=501,
        content={
            "error": "Not Implemented",
            "message": "Balance data available via BYO-key in HA component",
            "documentation": "https://github.com/DATADIO/ha-energy-insights-nl#tennet-byo-key",
            "reason": "TenneT API license prohibits server-side redistribution"
        }
    )