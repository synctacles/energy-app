"""
Balance Endpoint - DEPRECATED (TenneT BYO-Key Only)

Grid balance data is no longer available via the SYNCTACLES API due to
TenneT API license restrictions. Access is now available only via BYO-key
(Bring Your Own) in the Home Assistant component.
"""
from fastapi import APIRouter
from starlette.responses import JSONResponse
from config.settings import GITHUB_ACCOUNT, HA_REPO_NAME

router = APIRouter()

# Build documentation URL from ENV (brand-free)
_DOC_URL = f"https://github.com/{GITHUB_ACCOUNT}/{HA_REPO_NAME}#tennet-byo-key-setup"

@router.get("/balance")
async def get_balance():
    """
    Grid balance data - Available via BYO-key in Home Assistant component.

    TenneT API license prohibits server-side redistribution.
    Configure your TenneT API key in Home Assistant for real-time balance data.
    """
    return JSONResponse(
        status_code=410,
        content={
            "error": "Gone",
            "message": "TenneT balance endpoint is deprecated. Data is now available via BYO-key (Bring Your Own Key) model in the Home Assistant integration.",
            "documentation": _DOC_URL,
            "reason": "TenneT API license prohibits server-side redistribution (ADR-001)",
            "migration": {
                "setup_guide": _DOC_URL,
                "get_api_key": "https://www.tennet.org/",
                "alternative": "Use Home Assistant integration with your personal TenneT API key"
            }
        }
    )