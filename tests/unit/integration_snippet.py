"""
Integration snippet for main.py
Shows how to add signals router to existing FastAPI app
"""

# In main.py, add:

from synctacles_db.api import signals  # NEW import

app = FastAPI(title="SYNCTACLES API")

# Existing routers
app.include_router(auth.router, prefix="/auth")
app.include_router(generation.router)
app.include_router(load.router)
app.include_router(balance.router)

# NEW: Signals router
app.include_router(signals.router)  # Adds /v1/signals endpoint

# Auth middleware remains unchanged
